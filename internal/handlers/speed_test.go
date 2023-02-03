package handlers

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

const (
	TestScale = 100000
)

func randMsg() (speedMsgType, interface{}) {
	types := []speedMsgType{msgType_iAmCamera, msgType_iAmDispatcher, msgType_iPlate, msgType_iWantHeartbeat, msgType_oHeartbeat, msgType_oTicket}
	mType := types[rand.Intn(len(types))]
	switch mType {
	case msgType_iAmCamera:
		return mType, msgInAmCamera{
			mType: mType,
			road:  uint16(rand.Intn(10000)),
			mile:  uint16(rand.Intn(500)),
			limit: uint16(rand.Intn(90)),
		}
	case msgType_iPlate:
		return mType, msgInPlate{
			mType:     mType,
			plate:     fmt.Sprintf("%x", rand.Intn(1000000)),
			timestamp: uint32(rand.Intn(10000000)),
		}
	case msgType_iAmDispatcher:
		roadCount := uint8(rand.Intn(20) + 1)
		roads := []uint16{}
		for i := 1; i <= int(roadCount); i++ {
			roads = append(roads, uint16(rand.Intn(500)))
		}
		return mType, msgInAmDispatcher{
			mType:    mType,
			numRoads: roadCount,
			roads:    roads,
		}
	case msgType_iWantHeartbeat:
		return mType, msgInWantHeartbeat{
			mType:    mType,
			interval: uint32(rand.Intn(1000)),
		}
	case msgType_oError:
		return mType, msgOutError{
			mType:        mType,
			errorMessage: fmt.Sprintf("a big bad error with code %v", rand.Intn(1000)),
		}
	case msgType_oHeartbeat:
		return mType, msgOutHeartbeat{mType: mType}
	case msgType_oTicket:
		return mType, msgOutTicket{
			mType:      mType,
			plate:      fmt.Sprintf("%x", rand.Intn(1000000)),
			road:       uint16(rand.Intn(500)),
			mile1:      uint16(rand.Intn(500)),
			timestamp1: uint32(rand.Intn(1000000000)),
			mile2:      uint16(rand.Intn(500)),
			timestamp2: uint32(rand.Intn(1000000000)),
			speed:      uint16(rand.Intn(100) * 100),
		}
	default:
		panic(fmt.Errorf("unknown message type: %v", mType))
	}
}

type testData struct {
	mType   speedMsgType
	message interface{}
}

func TestSpeedMessageParsing(t *testing.T) {
	// util.Log.SetLevel("trace")
	rand.Seed(time.Now().UnixMilli())
	requestMessages := []testData{}
	for i := 1; i <= TestScale; i++ {
		mType, message := randMsg()
		requestMessages = append(requestMessages, testData{mType: mType, message: message})
	}

	request := new(bytes.Buffer)

	for i, msg := range requestMessages {
		strat := strategies[msg.mType]
		msgBytes, err := strat.serialize(msg.message)
		if err != nil {
			t.Errorf("failed to serialize message [%v of %v]: %+v => %v", i+1, len(requestMessages), msg.message, err.Error())
			t.FailNow()
		}
		request.Write(msgBytes)
		fmt.Printf("wrote message [%v of %v] [%+v] => bytes[%+x]\n", i+1, len(requestMessages), msg, msgBytes)
	}

	requestBytes := request.Bytes()
	if len(requestBytes) == 0 {
		t.FailNow()
	}

	fmt.Printf("wrote %v messages to request with %v total bytes =>[%x]\n", len(requestMessages), len(requestBytes), requestBytes)

	conn := NewTestConnectionFromBytes(requestBytes)
	_, err := SpeedDaemonHandler(conn)
	if err != nil {
		t.Errorf("handler error (msg %v of %v): => %v", len(allMessages)+1, len(requestMessages), err.Error())
		t.FailNow()
	}

	if len(allMessages) != len(requestMessages) {
		t.Errorf("message parsing failed; saw %v messages but expected %v", len(allMessages), len(requestMessages))
		t.FailNow()
	}

	for i, testData := range requestMessages {
		actual := allMessages[i]
		expected := testData.message
		fmt.Printf("comparing actual: [%+v] to expected: [%+v]\n", actual, expected)
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("message [%v] mismatch. expected: [%+v]; actual: [%+v]", i, expected, actual)
			t.FailNow()
		}
		fmt.Printf("correctly read message => [%+v]\n", actual)
	}

	fmt.Printf("all %v messages correctly parsed\n", len(requestMessages))
}
