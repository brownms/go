package handlers

import (
	"strings"
	"testing"
)

type primeSpec struct {
	requests []string
	results  []string
}

func primeSpecFrom(in string, out string) primeSpec {
	return primeSpec{
		requests: []string{in},
		results:  []string{out},
	}
}

func TestPrime(t *testing.T) {
	specs := []primeSpec{
		primeSpecFrom(`{ "method": "isPrime", "number": 7 }`, `{"method":"isPrime","prime":true}`),
		primeSpecFrom(`{ "method": "isPrime", "number": 32 }`, `{"method":"isPrime","prime":false}`),
		primeSpecFrom("not json", "Invalid JSON"),
		primeSpecFrom("   ", "Invalid Request"),
		{
			requests: []string{
				`{ "method": "isPrime", "number": 7 }`, `{ "method": "isPrime", "number": 14 }`, `{ "method": "foo", "number": 14 }`,
			},
			results: []string{
				`{"method":"isPrime","prime":true}`, `{"method":"isPrime","prime":false}`, "Invalid Method: [foo]",
			},
		},
	}

	for _, spec := range specs {
		request_all_text := strings.Join(spec.requests, "\n")
		conn := NewTestConnectionFromString(request_all_text)
		_, err := PrimeHandler(conn)

		if err != nil {
			t.Errorf("Handler returned error %v", err.Error())
			t.FailNow()
		}

		if !conn.IsClosed() {
			t.Errorf("handler: Connection was not properly closed")
			t.Fail()
		}

		if len(conn.results) != len(spec.requests) {
			t.Errorf("handler: expected %d results, actual: %d", len(spec.requests), len(conn.results))
			t.FailNow()
		}

		for i, expected := range spec.results {
			actual := strings.Trim(string(conn.results[i]), "\n")
			if !strings.EqualFold(actual, expected) {
				t.Errorf("handler: result mismatch:\nactual:   [%v]\nexpected: [%v]", actual, expected)
				t.FailNow()
			}
		}
	}
}
