package speeddaemon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type fieldType string
type speedMsgType byte

const (
	u8      fieldType = "u8"
	u16     fieldType = "u16"
	u32     fieldType = "u32"
	str     fieldType = "string"
	arr_u16 fieldType = "arr_u16"

	msgType_None           speedMsgType = 0x00 // 0
	msgType_iAmCamera      speedMsgType = 0x80 // 128
	msgType_iAmDispatcher  speedMsgType = 0x81 // 129
	msgType_iWantHeartbeat speedMsgType = 0x40 // 64
	msgType_oHeartbeat     speedMsgType = 0x41 // 65
	msgType_iPlate         speedMsgType = 0x20 // 32
	msgType_oTicket        speedMsgType = 0x21 // 33
	msgType_oError         speedMsgType = 0x10 // 16
)

var fieldTypeByteLengths = map[fieldType]int{
	u8:  1,
	u16: 2,
	u32: 4,
}

func toUint16(bytes []byte) uint16 {
	return binary.BigEndian.Uint16(bytes)
}

func toUint32(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}

func writeString(w io.Writer, strValue string) error {
	strLen := uint8(len(strValue))
	if err := binary.Write(w, binary.BigEndian, strLen); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, []byte(strValue)); err != nil {
		return err
	}
	return nil
}

func errorMessage(message string) msgOutError {
	return msgOutError{mType: msgType_oError, errorMessage: message}
}

// Incoming messages (Client => Server)
// ==> Messages that can be received from the Client
type msgInWantHeartbeat struct {
	mType    speedMsgType
	interval uint32 // deciseconds (10 per second; e.g. 25 = 2.5 seconds)
}

type msgInAmCamera struct {
	mType speedMsgType
	road  uint16
	mile  uint16
	limit uint16 // (miles per hour)
}

type msgInAmDispatcher struct {
	mType    speedMsgType
	numRoads uint8
	roads    []uint16
}

type msgInPlate struct {
	mType     speedMsgType
	plate     string
	timestamp uint32
}

// Outgoing messages (Server => Client)
// ==> Messages that are sent to the Client
type msgOutError struct {
	mType        speedMsgType
	errorMessage string
}

type msgOutTicket struct {
	mType      speedMsgType
	plate      string
	road       uint16
	mile1      uint16
	timestamp1 uint32
	mile2      uint16
	timestamp2 uint32
	speed      uint16 // (100x miles per hour)
}

type msgOutHeartbeat struct {
	mType speedMsgType
}

type strategy struct {
	typeIndicator speedMsgType
	fieldTypes    []fieldType
	name          string
	parse         func([][]byte) (any, error)
	serialize     func(any) ([]byte, error)
}

var strategies = map[speedMsgType]strategy{
	msgType_iAmCamera: {
		typeIndicator: msgType_iAmCamera,
		fieldTypes:    []fieldType{u16, u16, u16},
		name:          "IAmCamera",
		parse: func(msgBytes [][]byte) (any, error) {
			if len(msgBytes) != 3 {
				return nil, fmt.Errorf("invalid number of fields: %v (expected %v)", len(msgBytes), 3)
			}
			return msgInAmCamera{
				mType: msgType_iAmCamera,
				road:  toUint16(msgBytes[0]),
				mile:  toUint16(msgBytes[1]),
				limit: toUint16(msgBytes[2]),
			}, nil
		},
		serialize: func(s any) ([]byte, error) {
			t := s.(msgInAmCamera)
			if t.mType != msgType_iAmCamera {
				return nil, fmt.Errorf("type mismatch: %v (expected %v)", t.mType, msgType_iAmCamera)
			}
			w := bytes.NewBuffer([]byte{})
			if err := binary.Write(w, binary.BigEndian, t.mType); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.road); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.mile); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.limit); err != nil {
				return nil, err
			}
			return w.Bytes(), nil
		},
	},
	msgType_iAmDispatcher: {
		typeIndicator: msgType_iAmDispatcher,
		fieldTypes:    []fieldType{arr_u16},
		name:          "IAmDispatcher",
		parse: func(msgBytes [][]byte) (any, error) {
			if len(msgBytes) != 1 {
				return nil, fmt.Errorf("invalid number of fields: %v (expected %v)", len(msgBytes), 1)
			}
			msg := msgInAmDispatcher{
				mType:    msgType_iAmDispatcher,
				numRoads: 0,
				roads:    []uint16{},
			}
			arrBytes := msgBytes[0]
			for i := 0; i < len(arrBytes); i += 2 {
				nextRoad := toUint16(arrBytes[i : i+2])
				msg.roads = append(msg.roads, nextRoad)
				msg.numRoads = uint8(len(msg.roads))
			}
			return msg, nil
		},
		serialize: func(s any) ([]byte, error) {
			t := s.(msgInAmDispatcher)
			if t.mType != msgType_iAmDispatcher {
				return nil, fmt.Errorf("type mismatch: %v (expected %v)", t.mType, msgType_iAmDispatcher)
			}
			w := bytes.NewBuffer([]byte{})
			if err := binary.Write(w, binary.BigEndian, t.mType); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.numRoads); err != nil {
				return nil, err
			}
			for _, road := range t.roads {
				if err := binary.Write(w, binary.BigEndian, road); err != nil {
					return nil, err
				}
			}
			return w.Bytes(), nil
		},
	},
	msgType_iPlate: {
		typeIndicator: msgType_iPlate,
		fieldTypes:    []fieldType{str, u32},
		name:          "Plate",
		parse: func(msgBytes [][]byte) (any, error) {
			if len(msgBytes) != 2 {
				return nil, fmt.Errorf("invalid number of fields: %v (expected %v)", len(msgBytes), 2)
			}
			return msgInPlate{
				mType:     msgType_iPlate,
				plate:     string(msgBytes[0]),
				timestamp: toUint32(msgBytes[1]),
			}, nil
		},
		serialize: func(s any) ([]byte, error) {
			t := s.(msgInPlate)
			if t.mType != msgType_iPlate {
				return nil, fmt.Errorf("type mismatch: %v (expected %v)", t.mType, msgType_iPlate)
			}
			w := bytes.NewBuffer([]byte{})
			if err := binary.Write(w, binary.BigEndian, t.mType); err != nil {
				return nil, err
			}
			if err := writeString(w, t.plate); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.timestamp); err != nil {
				return nil, err
			}
			return w.Bytes(), nil
		},
	},
	msgType_iWantHeartbeat: {
		typeIndicator: msgType_iWantHeartbeat,
		fieldTypes:    []fieldType{u32},
		name:          "WantHeartBeat",
		parse: func(msgBytes [][]byte) (any, error) {
			if len(msgBytes) != 1 {
				return nil, fmt.Errorf("invalid number of fields: %v (expected %v)", len(msgBytes), 1)
			}
			return msgInWantHeartbeat{
				mType:    msgType_iWantHeartbeat,
				interval: toUint32(msgBytes[0]),
			}, nil
		},
		serialize: func(s any) ([]byte, error) {
			t := s.(msgInWantHeartbeat)
			if t.mType != msgType_iWantHeartbeat {
				return nil, fmt.Errorf("type mismatch: %v (expected %v)", t.mType, msgType_iWantHeartbeat)
			}
			w := bytes.NewBuffer([]byte{})
			if err := binary.Write(w, binary.BigEndian, t.mType); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.interval); err != nil {
				return nil, err
			}
			return w.Bytes(), nil
		},
	},

	msgType_oError: {
		typeIndicator: msgType_oError,
		fieldTypes:    []fieldType{str},
		name:          "Error",
		parse: func(msgBytes [][]byte) (any, error) {
			if len(msgBytes) != 1 {
				return nil, fmt.Errorf("invalid number of fields: %v (expected %v)", len(msgBytes), 1)
			}
			return msgOutError{
				mType:        msgType_oError,
				errorMessage: string(msgBytes[0]),
			}, nil
		},
		serialize: func(s any) ([]byte, error) {
			t := s.(msgOutError)
			if t.mType != msgType_oError {
				return nil, fmt.Errorf("type mismatch: %v (expected %v)", t.mType, msgType_oError)
			}
			w := bytes.NewBuffer([]byte{})
			if err := binary.Write(w, binary.BigEndian, t.mType); err != nil {
				return nil, err
			}
			if err := writeString(w, t.errorMessage); err != nil {
				return nil, err
			}
			return w.Bytes(), nil
		},
	},

	msgType_oHeartbeat: {
		typeIndicator: msgType_oHeartbeat,
		fieldTypes:    []fieldType{},
		name:          "HeartBeat",
		parse: func(msgBytes [][]byte) (any, error) {
			if len(msgBytes) != 0 {
				return nil, fmt.Errorf("invalid number of fields: %v (expected %v)", len(msgBytes), 0)
			}
			return msgOutHeartbeat{mType: msgType_oHeartbeat}, nil
		},
		serialize: func(s any) ([]byte, error) {
			t, ok := s.(msgOutHeartbeat)
			if !ok || t.mType != msgType_oHeartbeat {
				return nil, fmt.Errorf("type mismatch: %v (expected %v)", t.mType, msgType_oHeartbeat)
			}
			w := bytes.NewBuffer([]byte{})
			if err := binary.Write(w, binary.BigEndian, t.mType); err != nil {
				return nil, err
			}
			return w.Bytes(), nil
		},
	},

	msgType_oTicket: {
		typeIndicator: msgType_oTicket,
		fieldTypes:    []fieldType{str, u16, u16, u32, u16, u32, u16},
		name:          "Ticket",
		parse: func(msgBytes [][]byte) (any, error) {
			if len(msgBytes) != 7 {
				return nil, fmt.Errorf("invalid number of fields: %v (expected %v)", len(msgBytes), 7)
			}
			return msgOutTicket{
				mType:      msgType_oTicket,
				plate:      string(msgBytes[0]),
				road:       toUint16(msgBytes[1]),
				mile1:      toUint16(msgBytes[2]),
				timestamp1: toUint32(msgBytes[3]),
				mile2:      toUint16(msgBytes[4]),
				timestamp2: toUint32(msgBytes[5]),
				speed:      toUint16(msgBytes[6]),
			}, nil
		},
		serialize: func(s any) ([]byte, error) {
			t, ok := s.(msgOutTicket)
			if !ok || t.mType != msgType_oTicket {
				return nil, fmt.Errorf("type mismatch: %v (expected %v)", t.mType, msgType_oTicket)
			}
			w := bytes.NewBuffer([]byte{})
			if err := binary.Write(w, binary.BigEndian, t.mType); err != nil {
				return nil, err
			}
			if err := writeString(w, t.plate); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.road); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.mile1); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.timestamp1); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.mile2); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.timestamp2); err != nil {
				return nil, err
			}
			if err := binary.Write(w, binary.BigEndian, t.speed); err != nil {
				return nil, err
			}
			return w.Bytes(), nil
		},
	},
}
