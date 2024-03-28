package handlers

import (
	"bytes"
	"encoding/binary"
	"proto-challenge/internal/util"
	"testing"
)

type MeansTestResponse struct {
	Value int32
}

func TestMeans(t *testing.T) {
	// <-- 49 00 00 30 39 00 00 00 65   I 12345 101
	// <-- 49 00 00 30 3a 00 00 00 66   I 12346 102
	// <-- 49 00 00 30 3b 00 00 00 64   I 12347 100
	// <-- 49 00 00 a0 00 00 00 00 05   I 40960 5
	// <-- 51 00 00 30 00 00 00 40 00   Q 12288 16384
	// --> 00 00 00 65                  101
	// <-- 49 00 00 a0 00 00 00 00 05   I 40960 5
	// <-- 49 00 00 a0 28 00 00 13 88   I 41000 5000
	// <-- 49 00 00 a4 10 00 00 03 dc   I 42000 988
	// <-- 51 00 00 9c 40 00 00 a4 10   Q 40000 42000
	// --> 00 00 05 db					1499
	// <-- 49 00 00 c3 51 77 35 94 00   I 50001 2000000000
	// <-- 49 00 00 c3 b4 77 35 94 00   I 50100 2000000000
	// <-- 49 00 00 c4 18 77 35 94 00   I 50200 2000000000
	// <-- 51 00 00 c3 50 00 00 c4 18   Q 50000 52000
	// --> 77 35 94 00					2000000000
	// <-- 51 00 00 00 00 00 00 c4 18   Q 0 52000
	// 6000006301 / 10 => 600000630.1 => 600000630
	// --> 23 c3 48 76

	requests := [][]byte{
		{0x49, 0x00, 0x00, 0x30, 0x39, 0x00, 0x00, 0x00, 0x65},
		{0x49, 0x00, 0x00, 0x30, 0x3a, 0x00, 0x00, 0x00, 0x66},
		{0x49, 0x00, 0x00, 0x30, 0x3b, 0x00, 0x00, 0x00, 0x64},
		{0x49, 0x00, 0x00, 0xa0, 0x00, 0x00, 0x00, 0x00, 0x05},
		{0x51, 0x00, 0x00, 0x30, 0x00, 0x00, 0x00, 0x40, 0x00},
		{0x49, 0x00, 0x00, 0xa0, 0x00, 0x00, 0x00, 0x00, 0x05},
		{0x49, 0x00, 0x00, 0xa0, 0x28, 0x00, 0x00, 0x13, 0x88},
		{0x49, 0x00, 0x00, 0xa4, 0x10, 0x00, 0x00, 0x03, 0xdc},
		{0x51, 0x00, 0x00, 0x9c, 0x40, 0x00, 0x00, 0xa4, 0x10},
		{0x49, 0x00, 0x00, 0xc3, 0x51, 0x77, 0x35, 0x94, 0x00},
		{0x49, 0x00, 0x00, 0xc3, 0xb4, 0x77, 0x35, 0x94, 0x00},
		{0x49, 0x00, 0x00, 0xc4, 0x18, 0x77, 0x35, 0x94, 0x00},
		{0x51, 0x00, 0x00, 0xc3, 0x50, 0x00, 0x00, 0xc4, 0x18},
		{0x51, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xc4, 0x18},
	}

	request_all_bytes := []byte{}
	for _, b := range requests {
		request_all_bytes = append(request_all_bytes, b...)
	}
	conn := util.NewTestConnectionFromBytes(request_all_bytes)
	_, err := MeansHandler(conn)

	if err != nil {
		t.Errorf("Handler returned error %v", err.Error())
		t.FailNow()
	}
	if !conn.IsClosed() {
		t.Errorf("handler: Connection was not properly closed")
		t.Fail()
	}

	expected_results := []int32{
		101, 1499, 2000000000, 600000630,
	}

	results := conn.Results()
	if len(results) != len(expected_results) {
		t.Errorf("handler: expected %d results, saw: %d", len(expected_results), len(results))
		t.FailNow()
	}

	for i, expected := range expected_results {
		var response MeansTestResponse
		err := binary.Read(bytes.NewReader(results[i]), binary.BigEndian, &response)
		if err != nil {
			t.Errorf("handler: caugh error %v", err.Error())
			t.FailNow()
		}
		if response.Value != expected {
			t.Errorf("handler: actual: %d; expected %d", response.Value, expected)
			t.FailNow()
		}
	}
}
