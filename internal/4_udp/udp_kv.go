package handlers

import (
	"fmt"
	"io"
	"proto-challenge/internal/util"
	"strings"
	"time"
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
	util.Log.Info("udp_kv: [%v] Received request [%v bytes]", id, len(request))
	key, value, hasEquals := strings.Cut(request, "=")
	if hasEquals {
		// INSERT request (contains key=value)
		kv.Store(key, value)
		util.Log.Info("udp_kv: [%v] Completed INSERT=>key:[%v]", id, key)
	} else {
		// RETRIEVE request (contains key only)
		// response =
		// util.Log.Debug("udp_kv: [%v] Writing [%v] to UDP connection", id, response)
		bytes_sent, err := writer.Write([]byte(fmt.Sprintf("%v=%v", key, kv.Retrieve(key))))
		if err != nil {
			return fmt.Errorf("udp_kv: [%v] Write failed => %w", id, err)
		}
		util.Log.Info("udp_kv: [%v] Completed RETRIEVE=>key:[%v] Sent %v bytes (in %v)", id, key, bytes_sent, time.Since(started))
		// util.Log.Trace("udp_kv: [%v] UDP handler completed: Sent response: [%v]", id, response)
	}
	return nil
}

var kvStore = &KeyValueStore{
	values: map[string]string{
		VERSION_KEY: VERSION_VALUE,
	},
}

func KeyValueUdpHandler(seq int64, request string, conn io.Writer) error {
	return kvStore.accept(seq, request, conn)
}
