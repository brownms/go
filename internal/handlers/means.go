package handlers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"proto-challenge/internal/util"
)

type RequestType byte

const (
	typeInsert RequestType = 'I' // 0x49
	typeQuery  RequestType = 'Q' // 0x51
)

type Insert struct {
	Time  int32
	Value int32
}

type Query struct {
	Start int32
	End   int32
}

type RawRequest struct {
	Type RequestType
	Num1 int32
	Num2 int32
}

func MeansHandler(conn io.ReadWriteCloser) (int64, error) {
	defer conn.Close()

	bytes_written := 0
	var values []Insert

	for {
		var req RawRequest
		err := binary.Read(conn, binary.BigEndian, &req)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return int64(bytes_written), nil
			} else {
				return 0, fmt.Errorf("reading binary data %w", err)
			}
		}

		if req.Type == typeInsert {
			values = append(values, Insert{
				Time:  req.Num1,
				Value: req.Num2,
			})
		}
		if req.Type == typeQuery {
			average := handleQuery(values, Query{
				Start: req.Num1,
				End:   req.Num2,
			})
			util.Log("Query: Sending query result: %d", average)
			binary.Write(conn, binary.BigEndian, average)
			bytes_written += 4
		}
	}
}

func handleQuery(values []Insert, query Query) int32 {
	if query.End < query.Start {
		util.Log("Query => Invalid; Start is after End; returning 0")
		return 0
	}

	util.Log("Query: querying %v values between %v => %v", len(values), query.Start, query.End)
	count := 0
	total := 0
	for _, v := range values {
		if v.Time >= query.Start && v.Time <= query.End {
			count += 1
			total += int(v.Value)
		}
	}
	util.Log("Query: Saw total of %d from %d matching values", total, count)
	if count == 0 {
		return 0
	}
	return int32(total / count)
}
