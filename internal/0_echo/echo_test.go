package handlers

import (
	"proto-challenge/internal/util"
	"testing"
)

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
		conn := util.NewTestConnectionFromString(spec.input)
		_, err := EchoHandler(conn)

		if err != nil {
			t.Errorf("Handler returned error %v", err.Error())
			t.FailNow()
		}

		if !conn.IsClosed() {
			t.Errorf("handler: Connection was not properly closed")
			t.Fail()
		}

		results := conn.Results()
		if len(results) != spec.expectedResults {
			t.Errorf("handler: expected %d results, actual: %d", 1, len(results))
			t.FailNow()
		}

		if spec.expectedResults > 0 {
			actual := string(results[0])
			if actual != spec.expectedOutput {
				t.Errorf("handler: actual: %v; expected %v", actual, spec.expectedOutput)
				t.FailNow()
			}
		}
	}
}
