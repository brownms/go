package handlers

import (
	"fmt"
	"net"
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
	conn   *net.UDPConn
	values map[string]string
	mutex  *sync.Mutex
}

func (kv *KeyValueStore) Store(key string, value string) {
	kv.mutex.Lock()
	defer kv.mutex.Unlock()

	// ignore "version" key
	if key != VERSION_KEY {
		kv.values[key] = value
	}
}

func (kv *KeyValueStore) Retrieve(key string) string {
	return kv.values[key]
}

func (kv *KeyValueStore) accept(id int64, request string, sender *net.UDPAddr) {
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
		bytes_sent, err := kv.conn.WriteToUDP([]byte(fmt.Sprintf("%v=%v", key, kv.Retrieve(key))), sender)
		if err != nil {
			panic(fmt.Errorf("udp_kv: [%v] Write failed => %w", id, err))
		}
		util.Log.Info("udp_kv: [%v] Completed RETRIEVE=>key:[%v] Sent %v bytes (in %v)", id, key, bytes_sent, time.Since(started))
		// util.Log.Trace("udp_kv: [%v] UDP handler completed: Sent response: [%v]", id, response)
	}
}

func KeyValueUdpServer(listenAddress string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddress)
	if err != nil {
		return fmt.Errorf("udp_server: Failed to resolve UDP address from %v => %w", listenAddress, err)
	}

	connection, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("udp_server: Failed to start UDP listen on %v => %w", udpAddr, err)
	}

	util.Log.Info("udp_server: UDP Listening on [%v]", listenAddress)

	kv_store := &KeyValueStore{
		conn: connection,
		values: map[string]string{
			VERSION_KEY: VERSION_VALUE,
		},
		mutex: new(sync.Mutex),
	}

	reqNum := int64(0)
	for {
		// Handle connections in a go routine
		buffer := make([]byte, 1024)
		bytesRead, sendAddr, err := connection.ReadFromUDP(buffer)
		if err != nil {
			util.Log.Error("udp_server: failed to read UDP => %w", err)
			continue
		}
		reqNum += 1
		util.Log.Info("udp_server: [%v] Received [%v] bytes", reqNum, bytesRead)
		request := string(buffer[:bytesRead])
		go kv_store.accept(reqNum, request, sendAddr)
	}
}
