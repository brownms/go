package unusualdb

import (
	"fmt"
	"io"
	"strings"
	"time"

	"log/slog"
)

const (
	EMPTY_RESPONSE string = ""
	VERSION_KEY    string = "version"
	VERSION_VALUE  string = "brownms udp_kv store 0.0.1"
)

type KeyValueStore struct {
	values map[string]string
}

func (kv *KeyValueStore) Store(key string, value string) {
	// ignore "version" key
	if key != VERSION_KEY {
		kv.values[key] = value
	}
}

func (kv *KeyValueStore) Retrieve(key string) string {
	return kv.values[key]
}

func (kv *KeyValueStore) accept(id int64, request string, writer io.Writer) error {
	started := time.Now()
	key, value, hasEquals := strings.Cut(request, "=")
	if hasEquals {
		// INSERT request (contains key=value)
		kv.Store(key, value)
		slog.Info("insert", "seq", id, "key", key)
	} else {
		// RETRIEVE request (contains key only)
		bytesSent, err := fmt.Fprintf(writer, "%v=%v", key, kv.Retrieve(key))
		if err != nil {
			return fmt.Errorf("udp_kv: [%v] Write failed => %w", id, err)
		}
		slog.Info("retrieve", "seq", id, "key", key, "bytes_sent", bytesSent, "elapsed", time.Since(started))
	}
	return nil
}

var kvStore = &KeyValueStore{
	values: map[string]string{
		VERSION_KEY: VERSION_VALUE,
	},
}

func Handler(seq int64, request string, conn io.Writer) error {
	return kvStore.accept(seq, request, conn)
}
