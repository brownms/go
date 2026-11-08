package means

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"log/slog"
)

type MeansRequestType byte

const (
	typeInsert MeansRequestType = 'I' // 0x49
	typeQuery  MeansRequestType = 'Q' // 0x51
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
	Type MeansRequestType
	Num1 int32
	Num2 int32
}

func Handler(conn io.ReadWriteCloser) (int64, error) {
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
			slog.Debug("sending query result", "average", average)
			err := binary.Write(conn, binary.BigEndian, average)
			if err != nil {
				return 0, fmt.Errorf("writing data %w", err)
			}
			bytes_written += 4
		}
	}
}

func handleQuery(values []Insert, query Query) int32 {
	if query.End < query.Start {
		slog.Info("invalid query: start after end; returning 0", "start", query.Start, "end", query.End)
		return 0
	}

	slog.Debug("querying", "values", len(values), "start", query.Start, "end", query.End)
	count := 0
	total := 0
	for _, v := range values {
		if v.Time >= query.Start && v.Time <= query.End {
			count += 1
			total += int(v.Value)
		}
	}
	slog.Debug("query total", "total", total, "count", count)
	if count == 0 {
		return 0
	}
	return int32(total / count)
}
