package handlers

import (
	"bufio"
	"fmt"
	"io"
	"proto-challenge/internal/util"
)

var allMessages []interface{}

func Dispatch(msg interface{}) {
	util.Log.Debug("dispatching message: %+v\n", msg)
	allMessages = append(allMessages, msg)
}

func SpeedDaemonHandler(conn io.ReadWriteCloser) (int64, error) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Split(bufio.ScanBytes)

	bytesWritten := int64(0)

	fieldLen := 0
	fieldBytes := []byte{}
	msgFields := [][]byte{}
	var strat strategy
	for scanner.Scan() {
		nextByte := scanner.Bytes()[0]
		util.Log.Trace("visiting nextByte => [%x]", nextByte)
		if strat.typeIndicator == msgType_None {
			// start of new message
			fieldLen = 0
			fieldBytes = []byte{}
			msgFields = [][]byte{}
			s, ok := strategies[speedMsgType(nextByte)]
			if !ok {
				return bytesWritten, fmt.Errorf("invalid message type indicator %x", nextByte)
			}
			strat = s
			util.Log.Debug("detected new message [%v] (type: %v, num fields: %v)", len(allMessages)+1, strat.name, len(strat.fieldTypes))
			if len(strat.fieldTypes) == 0 {
				msg, err := strat.parse(msgFields)
				if err != nil {
					return bytesWritten, err
				}
				// reset strategy to start new message
				strat = strategy{}
				Dispatch(msg)
			}
			continue
		}

		fieldNum := len(msgFields)
		fieldType := strat.fieldTypes[fieldNum]
		// append byte if not at end of current field
		if fieldType == u8 {
			util.Log.Trace("tracking u8 field (field %v of %v) current size: (%v)", fieldNum, len(strat.fieldTypes), len(fieldBytes))
			if len(fieldBytes) == 0 {
				fieldBytes = []byte{nextByte}
			}
			if len(fieldBytes) == 1 {
				msgFields = append(msgFields, fieldBytes)
			}
		}

		if fieldType == u16 {
			util.Log.Trace("tracking u16 field (field %v of %v) current size: (%v)", fieldNum, len(strat.fieldTypes), len(fieldBytes))
			if len(fieldBytes) < 2 {
				fieldBytes = append(fieldBytes, nextByte)
			}
			if len(fieldBytes) == 2 {
				msgFields = append(msgFields, fieldBytes)
			}
		}

		if fieldType == u32 {
			util.Log.Trace("tracking u32 field (field %v of %v) current size: (%v)", fieldNum, len(strat.fieldTypes), len(fieldBytes))
			if len(fieldBytes) < 4 {
				fieldBytes = append(fieldBytes, nextByte)
			}
			if len(fieldBytes) == 4 {
				msgFields = append(msgFields, fieldBytes)
			}
		}

		if fieldType == str || fieldType == arr_u16 {
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
				util.Log.Trace("message %v complete (type: %v) => [%x]", len(allMessages)+1, strat.name, msgFields)
			}
			// reset strategy to start new message
			strat = strategy{}
			Dispatch(msg)
		}
	}

	if scanner.Err() != nil {
		return bytesWritten, fmt.Errorf("scanning connection %w", scanner.Err())
	}

	return bytesWritten, nil
}
