package speeddaemon

import (
	"bufio"
	"fmt"
	"io"
	"sync"

	"log/slog"
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

func Handler(conn io.ReadWriteCloser) (int64, error) {
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
			slog.Debug("new message", "n", messageCount, "type", strat.name, "fields", len(strat.fieldTypes))
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
				continue
			}
			if len(fieldBytes) < fieldLen {
				fieldBytes = append(fieldBytes, nextByte)
			}
			if len(fieldBytes) == fieldLen {
				msgFields = append(msgFields, fieldBytes)
			}
		}

		// was field was appended to message? if so, field is complete
		if len(msgFields) > fieldNum {
			// reset field tracking
			fieldLen = 0
			fieldBytes = []byte{}
		}

		// message is complete if number of fields read >= number expected for the message
		if len(msgFields) >= len(strat.fieldTypes) {
			msg, err := strat.parse(msgFields)
			if err != nil {
				slog.Warn("failed to parse message", "type", strat.name, "fields", fmt.Sprintf("%x", msgFields), "err", err)
			} else {
				slog.Debug("message complete", "n", messageCount, "type", strat.name)
			}
			// reset strategy to start new message
			strat = strategy{}
			err = monitor.accept(client, msg)
			if err != nil {
				slog.Error("message rejected", "client", client.id, "err", err)
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
