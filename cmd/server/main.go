package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	handlers "proto-challenge/internal/handlers"
	"proto-challenge/internal/util"
	"time"
)

type Protocol byte

const (
	PROTOCOL_TCP Protocol = 'T'
	PROTOCOL_UDP Protocol = 'U'
)

type ServiceHandler func(conn io.ReadWriteCloser) (int64, error)

type Service struct {
	protocol    Protocol
	name        string
	handlerFunc ServiceHandler
}

func main() {
	services := map[string]Service{
		"echo": {
			name:        "echo",
			protocol:    PROTOCOL_TCP,
			handlerFunc: handlers.EchoHandler,
		},
		"prime": {
			name:        "prime",
			protocol:    PROTOCOL_TCP,
			handlerFunc: handlers.PrimeHandler,
		},
		"means": {
			name:        "means",
			protocol:    PROTOCOL_TCP,
			handlerFunc: handlers.MeansHandler,
		},
		"chat": {
			name:        "chat",
			protocol:    PROTOCOL_TCP,
			handlerFunc: handlers.ChatHandler,
		},
		"udp_kv": {
			name:        "udp_kv",
			protocol:    PROTOCOL_UDP,
			handlerFunc: handlers.UdpKeyValueHandler,
		},
	}

	// Parse flags
	listen_host := flag.String("host", "localhost", "Network bind host")
	listen_port := flag.Int("port", 9999, "Network listen port")
	service_name := flag.String("service", "echo", "Service to run")

	flag.Parse()

	service, exists := services[*service_name]
	if !exists {
		util.Log("server: Service not supported: %v", *service_name)
		os.Exit(1)
	}

	// Listen for incoming connections.
	conn_spec := fmt.Sprintf("%v:%v", *listen_host, *listen_port)
	if service.protocol == PROTOCOL_TCP {
		startTCP(conn_spec, &service)
	} else if service.protocol == PROTOCOL_UDP {
		startUDP(conn_spec, &service)
	} else {
		util.Log("server: Protocal not supported: %v", service.protocol)
		os.Exit(1)
	}

}

func startTCP(conn_spec string, service *Service) {
	listener, err := net.Listen("tcp", conn_spec)
	if err != nil {
		panic(fmt.Errorf("server: Failed to start TCP listener on [%v]: %w", conn_spec, err))
	}
	// Close the listener when the application closes.
	defer listener.Close()

	util.Log("server: Started service: [%v]", service.name)
	util.Log("server: TCP Listening on [%v]", conn_spec)
	for {
		// Listen for an incoming connection.
		connection, err := listener.Accept()
		if err != nil {
			panic(fmt.Errorf("server: Error accepting connection: %w", err))
		}
		// Handle connections in a new goroutine.
		util.Log("server: saw connection from: [%s]", connection.RemoteAddr())
		go executeHandler(connection, service)
	}
}

func startUDP(conn_spec string, service *Service) {
	udp_addr, err := net.ResolveUDPAddr("udp", conn_spec)
	if err != nil {
		panic(fmt.Errorf("server: Failed to resolve UDP address from %v => %w", conn_spec, err))
	}

	connection, err := net.ListenUDP("udp", udp_addr)
	if err != nil {
		panic(fmt.Errorf("server: Failed to start UDP listen on %v => %w", conn_spec, err))
	}

	// Close the UDP connection when the application exits.
	defer connection.Close()

	util.Log("server: Started service: [%v]", service.name)
	util.Log("server: UDP Listening on [%v]", conn_spec)
	for {
		// Handle connections in a go routine
		buffer := make([]byte, 1024)
		bytesRead, sendAddr, err := connection.ReadFromUDP(buffer)
		if err != nil {
			panic(fmt.Errorf("server: failed to read UDP => %w", err))
		}
		util.Log("server: Received request [%v bytes] from sender: [%v]", bytesRead, sendAddr.String())
		requestReader := bytes.NewReader(buffer[:bytesRead])
		udpWrapper := util.NewUdpRequestWrapper(connection, sendAddr, requestReader)
		go executeHandler(udpWrapper, service)
	}
}

func executeHandler(connection io.ReadWriteCloser, service *Service) {
	start := time.Now()
	bytesWritten, err := service.handlerFunc(connection)
	elapsed := time.Since(start)
	if err != nil {
		util.Log("%v: Handler execution failed => %v (in %v)", service.name, err.Error(), elapsed)
	} else {
		util.Log("%v: Handler completed; Wrote %v bytes (in %v)", service.name, bytesWritten, elapsed)
	}
}
