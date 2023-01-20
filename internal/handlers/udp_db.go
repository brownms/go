package handlers

import (
	"fmt"
	"io"
	"proto-challenge/internal/util"
	"strings"
	"sync"
	"time"
)

const (
	EMPTY_RESPONSE string = ""
	VERSION_KEY    string = "version"
	VERSION_VALUE  string = "brownms udp_kv store 0.0.1"
)

type KeyValueStore struct {
	values map[string]string
	mutex  *sync.Mutex
}

func (kv *KeyValueStore) Store(key string, value string) {
	kv.mutex.Lock()
	defer kv.mutex.Unlock()
	kv.values[key] = value
}

func (kv *KeyValueStore) Retrieve(key string) string {
	return kv.values[key]
}

var kv_store = KeyValueStore{
	values: map[string]string{
		VERSION_KEY: VERSION_VALUE,
	},
	mutex: new(sync.Mutex),
}

func UdpKeyValueHandler(conn io.ReadWriteCloser) (int64, error) {
	started := time.Now()
	buffer := new(strings.Builder)
	_, err := io.Copy(buffer, conn)
	if err != nil {
		return 0, fmt.Errorf("udp_kv: Failed to read request => %w", err)
	}
	request := strings.TrimSpace(buffer.String())
	util.Log.Debug("udp_kv: Got request: [%v]", request)
	response := handleKvRequest(request)
	util.Log.Trace("udp_kv: UDP handler completed: KV => [%v]", kv_store.values)
	if len(response) > 0 {
		util.Log.Debug("udp_kv: Writing [%v] to UDP connection", response)
		bytes_sent, err := conn.Write([]byte(response))
		if err != nil {
			return 0, fmt.Errorf("udp_kv: UDP write failed => %w", err)
		}
		util.Log.Info("udp_kv: UDP handler completed: Sent %v bytes (in %v)", bytes_sent, time.Since(started))
		util.Log.Trace("udp_kv: UDP handler completed: Sent response: [%v]", response)
		return int64(bytes_sent), nil
	}
	return 0, nil
}

func handleKvRequest(requestText string) string {
	if len(requestText) > 1000 {
		return EMPTY_RESPONSE
	}
	key, value, hasEquals := strings.Cut(requestText, "=")
	if hasEquals {
		// INSERT request (contains key=value)
		// ignore "version" key
		if key != VERSION_KEY {
			kv_store.values[key] = strings.TrimSuffix(value, "\n")
		}
		return EMPTY_RESPONSE
	} else {
		// RETRIEVE request (contains key only)
		value = kv_store.values[key]
		return fmt.Sprintf("%v=%v", key, value)
	}
}
