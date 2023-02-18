package handlers

import (
	"bytes"
	"io"
	"testing"
)

type TestConnection struct {
	reader  io.Reader
	closed  bool
	results [][]byte
}

func (conn *TestConnection) Read(p []byte) (n int, err error) {
	return conn.reader.Read(p)
}

func (conn *TestConnection) Write(value []byte) (n int, err error) {
	// util.Log("Write: Saw bytes: %x => %d", value, value)
	conn.results = append(conn.results, value)
	return len(value), nil
}

func (conn *TestConnection) Close() error {
	conn.closed = true
	return nil
}

func (conn *TestConnection) IsClosed() bool {
	return conn.closed
}

func NewTestConnectionFromBytes(source_bytes []byte) *TestConnection {
	return &TestConnection{
		closed:  false,
		results: [][]byte{},
		reader:  bytes.NewReader(source_bytes),
	}
}

func NewTestConnectionFromString(source_string string) *TestConnection {
	return NewTestConnectionFromBytes([]byte(source_string))
}

type echoSpec struct {
	input           string
	expectedResults int
	expectedOutput  string
}

func echoSpecFrom(in string, out string) echoSpec {
	expectedResults := 1
	if len(out) == 0 {
		// expect no results if the output is empty
		expectedResults = 0
	}
	return echoSpec{
		input:           in,
		expectedOutput:  out,
		expectedResults: expectedResults,
	}
}

func TestEcho(t *testing.T) {
	specs := []echoSpec{
		echoSpecFrom("foo", "foo"),
		echoSpecFrom("foo bar baz\nbat buzz\t", "foo bar baz\nbat buzz\t"),
		echoSpecFrom("", ""),
	}

	for _, spec := range specs {
		conn := NewTestConnectionFromString(spec.input)
		_, err := EchoHandler(conn)

		if err != nil {
			t.Errorf("Handler returned error %v", err.Error())
			t.FailNow()
		}

		if !conn.IsClosed() {
			t.Errorf("handler: Connection was not properly closed")
			t.Fail()
		}

		if len(conn.results) != spec.expectedResults {
			t.Errorf("handler: expected %d results, actual: %d", 1, len(conn.results))
			t.FailNow()
		}

		if spec.expectedResults > 0 {
			actual := string(conn.results[0])
			if actual != spec.expectedOutput {
				t.Errorf("handler: actual: %v; expected %v", actual, spec.expectedOutput)
				t.FailNow()
			}
		}
	}
}
