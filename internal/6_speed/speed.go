package handlers

import (
	"bufio"
	"fmt"
	"io"
	"proto-challenge/internal/util"
	"sync"
)

type MessageHandler interface {
	accept(*Client, any) error
	stop(*Client)
}

var monitorState = &SpeedMonitor{
	cars:         make(map[string]Car),
	limits:       make(map[uint16]uint16),
	clients:      make(map[uint16]*Client),
	ticketsState: make(map[msgOutTicket]bool),
	stateMutex:   sync.Mutex{},
}

func SpeedDaemonHandler(conn io.ReadWriteCloser) (int64, error) {
	return monitorConnection(conn, monitorState)
}

var clientId = 0
var clientMutex = sync.Mutex{}

func nextClientId() uint16 {
	clientMutex.Lock()
	defer clientMutex.Unlock()
	clientId += 1
	return uint16(clientId)
}

func monitorConnection(conn io.ReadWriteCloser, monitor MessageHandler) (int64, error) {

	client := &Client{
		id:   nextClientId(),
		conn: &conn,
	}
	defer monitor.stop(client)

	scanner := bufio.NewScanner(conn)
	scanner.Split(bufio.ScanBytes)

	fieldLen := 0
	fieldBytes := []byte{}
	msgFields := [][]byte{}
	messageCount := 0
	var strat strategy
	for scanner.Scan() {
		nextByte := scanner.Bytes()[0]
		util.Log.Trace("visiting nextByte => [%x]", nextByte)
		if strat.typeIndicator == msgType_None {
			// start of new message
			messageCount += 1
			fieldLen = 0
			fieldBytes = []byte{}
			msgFields = [][]byte{}
			s, ok := strategies[speedMsgType(nextByte)]
			if !ok {
				return client.bytesSent, fmt.Errorf("invalid message type indicator %x", nextByte)
			}
			strat = s
			util.Log.Trace("detected new message [%v] (type: %v, num fields: %v)", messageCount+1, strat.name, len(strat.fieldTypes))
			if len(strat.fieldTypes) == 0 {
				msg, err := strat.parse(msgFields)
				if err != nil {
					return client.bytesSent, err
				}
				// reset strategy to start new message
				strat = strategy{}
				err = monitor.accept(client, msg)
				if err != nil {
					return client.bytesSent, err
				}
			}
			continue
		}

		fieldNum := len(msgFields)
		fieldType := strat.fieldTypes[fieldNum]
		fieldTypeByteLength, ok := fieldTypeByteLengths[fieldType]
		if ok {
			// fixed-length field
			// append byte if not at end of current field
			util.Log.Trace("tracking %v field (field %v of %v) current size: (%v)", fieldType, fieldNum, len(strat.fieldTypes), len(fieldBytes))
			if len(fieldBytes) < fieldTypeByteLength {
				fieldBytes = append(fieldBytes, nextByte)
			}
			if len(fieldBytes) == fieldTypeByteLength {
				msgFields = append(msgFields, fieldBytes)
			}
		} else {
			// variable-length (length-prefixed) field
			if fieldLen == 0 {
				fieldLen = int(uint8(nextByte))
				if fieldType == arr_u16 {
					// 2 bytes per u16 array element
					fieldLen = fieldLen * 2
				}
				util.Log.Trace("start tracking %v field (field %v of %v) expected length: %v", fieldType, fieldNum+1, len(strat.fieldTypes), fieldLen)
				continue
			}
			if len(fieldBytes) < fieldLen {
				fieldBytes = append(fieldBytes, nextByte)
			}
			util.Log.Trace("continue tracking str field (field %v of %v) expected length: %v current size: (%v)", fieldNum+1, len(strat.fieldTypes), fieldLen, len(fieldBytes))
			if len(fieldBytes) == fieldLen {
				msgFields = append(msgFields, fieldBytes)
			}
		}

		// was field was appended to message? if so, field is complete
		if len(msgFields) > fieldNum {
			// reset field tracking
			util.Log.Trace("field %v complete (type: %v)", fieldNum, fieldType)
			fieldLen = 0
			fieldBytes = []byte{}
		}

		// message is complete if number of fields read >= number expected for the message
		if len(msgFields) >= len(strat.fieldTypes) {
			msg, err := strat.parse(msgFields)
			if err != nil {
				util.Log.Warn("failed to parse message (type: %v): %x => %v", strat.name, msgFields, err.Error())
			} else {
				util.Log.Trace("message complete (type: %v) => [%x]", messageCount, strat.name, msgFields)
			}
			// reset strategy to start new message
			strat = strategy{}
			err = monitor.accept(client, msg)
			if err != nil {
				util.Log.Error("[client:%v] error => %v", err.Error())
				// intentionally ignoring send error here; report the outer error
				// could use a multierror here and report both
				_ = client.send(err.Error())
				return client.bytesSent, err
			}
		}
	}

	if scanner.Err() != nil {
		return client.bytesSent, fmt.Errorf("scanning connection %w", scanner.Err())
	}

	return client.bytesSent, nil
}
