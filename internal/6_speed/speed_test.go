package handlers

import (
	"bytes"
	"fmt"
	"math/rand"
	"proto-challenge/internal/util"
	"reflect"
	"testing"
)

const (
	TestScale = 100000
)

func randMsg() (speedMsgType, any) {
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
	message any
}

type testMonitor struct {
	messages []any
	stopped  bool
}

func (m *testMonitor) accept(client *Client, rawMessage any) error {
	m.messages = append(m.messages, rawMessage)
	return nil
}

func (m *testMonitor) stop(client *Client) {
	m.stopped = true
}

func (m *testMonitor) size() int {
	return len(m.messages)
}

func buildTestRequestBytes(t *testing.T, requestMessages []testData) []byte {
	request := new(bytes.Buffer)
	for i, msg := range requestMessages {
		strat := strategies[msg.mType]
		msgBytes, err := strat.serialize(msg.message)
		if err != nil {
			t.Errorf("failed to serialize message [%v of %v]: %+v => %v", i+1, len(requestMessages), msg.message, err.Error())
			t.FailNow()
		}
		request.Write(msgBytes)
		// fmt.Printf("wrote message [%v of %v] [%+v] => bytes[%+x]\n", i+1, len(requestMessages), msg, msgBytes)
	}
	b := request.Bytes()
	if len(b) == 0 {
		t.FailNow()
	}
	return b
}

func TestSpeedMessageParsing(t *testing.T) {
	// util.Log.SetLevel("trace")

	requestMessages := []testData{}
	for i := 1; i <= TestScale; i++ {
		mType, message := randMsg()
		requestMessages = append(requestMessages, testData{mType: mType, message: message})
	}

	requestBytes := buildTestRequestBytes(t, requestMessages)
	// fmt.Printf("wrote %v messages to request with %v total bytes =>[%x]\n", len(requestMessages), len(requestBytes), requestBytes)

	monitor := &testMonitor{}
	conn := util.NewTestConnectionFromBytes(requestBytes)
	_, err := monitorConnection(conn, monitor)
	if err != nil {
		t.Errorf("handler error (msg %v of %v): => %v", monitor.size()+1, len(requestMessages), err.Error())
		t.FailNow()
	}

	if monitor.size() != len(requestMessages) {
		t.Errorf("message parsing failed; saw %v messages but expected %v", monitor.size(), len(requestMessages))
		t.FailNow()
	}

	for i, testData := range requestMessages {
		actual := monitor.messages[i]
		expected := testData.message
		// fmt.Printf("comparing actual: [%+v] to expected: [%+v]\n", actual, expected)
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("message [%v] mismatch. expected: [%+v]; actual: [%+v]", i, expected, actual)
			t.FailNow()
		}
		// fmt.Printf("correctly read message => [%+v]\n", actual)
	}

	if !monitor.stopped {
		t.Errorf("failed to stop monitor for connection")
		t.FailNow()
	}

	fmt.Printf("all %v messages correctly parsed\n", len(requestMessages))
}

func TestSpeedComputation(t *testing.T) {
	tests := []struct {
		o1     Observation
		o2     Observation
		expect float64
	}{
		{
			o1:     Observation{mile: 20, timestamp: 10000},
			o2:     Observation{mile: 21, timestamp: 10002},
			expect: 1800,
		},
		{
			o1:     Observation{mile: 20, timestamp: 10000},
			o2:     Observation{mile: 35, timestamp: 10400},
			expect: 135,
		},
		{
			o1:     Observation{mile: 20, timestamp: 10000},
			o2:     Observation{mile: 35, timestamp: 10477},
			expect: 113,
		},
		{
			o1:     Observation{mile: 20, timestamp: 10000},
			o2:     Observation{mile: 5, timestamp: 10477},
			expect: 113,
		},
	}

	for _, test := range tests {
		actual := computeSpeedMph(test.o1, test.o2)
		if actual != test.expect {
			t.Errorf("incorrect speed calculated: [%v]; expected => [%+v]", actual, test)
			t.FailNow()
		}
	}
}

func TestDayNumber(t *testing.T) {
	tests := []struct {
		timestamp uint32
		day       uint32
	}{
		{timestamp: 4106615, day: 47},
	}
	for _, test := range tests {
		ticket := &msgOutTicket{timestamp1: test.timestamp, timestamp2: test.timestamp}
		_, actual := days(ticket)
		if actual != test.day {
			t.Errorf("incorrect day calculated: [%v]; expected => [%v]", actual, test.day)
			t.FailNow()
		}
	}
}
