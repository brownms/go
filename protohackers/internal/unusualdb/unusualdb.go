package unusualdb

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

const (
	versionKey   = "version"
	versionValue = "brownms udp_kv store 0.0.1"
)

// Store is a key-value store served over UDP. It is not safe for concurrent
// use; server.ServeUDP calls Handle for one datagram at a time.
type Store struct {
	values map[string]string
}

// NewStore returns a Store holding only the read-only version key.
func NewStore() *Store {
	return &Store{values: map[string]string{versionKey: versionValue}}
}

// Handle serves one request: "key=value" inserts, and "key" replies with "key=value".
func (s *Store) Handle(seq int64, request string, conn io.Writer) error {
	started := time.Now()
	key, value, isInsert := strings.Cut(request, "=")
	if isInsert {
		if key != versionKey {
			s.values[key] = value
		}
		slog.Info("insert", "seq", seq, "key", key)
		return nil
	}

	n, err := fmt.Fprintf(conn, "%v=%v", key, s.values[key])
	if err != nil {
		return fmt.Errorf("[%v] writing response: %w", seq, err)
	}
	slog.Info("retrieve", "seq", seq, "key", key, "bytes_sent", n, "elapsed", time.Since(started))
	return nil
}
