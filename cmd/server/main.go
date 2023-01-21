package main

import (
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

type TcpServiceHandler func(conn io.ReadWriteCloser) (int64, error)
type UdpServiceHandler func(listenAddress string) error

type Service struct {
	protocol       Protocol
	name           string
	tcpHandlerFunc TcpServiceHandler
	udpServerFunc  UdpServiceHandler
}

func main() {
	services := map[string]Service{
		"echo": {
			name:           "echo",
			protocol:       PROTOCOL_TCP,
			tcpHandlerFunc: handlers.EchoHandler,
		},
		"prime": {
			name:           "prime",
			protocol:       PROTOCOL_TCP,
			tcpHandlerFunc: handlers.PrimeHandler,
		},
		"means": {
			name:           "means",
			protocol:       PROTOCOL_TCP,
			tcpHandlerFunc: handlers.MeansHandler,
		},
		"chat": {
			name:           "chat",
			protocol:       PROTOCOL_TCP,
			tcpHandlerFunc: handlers.ChatHandler,
		},
		"udp_kv": {
			name:          "udp_kv",
			protocol:      PROTOCOL_UDP,
			udpServerFunc: handlers.KeyValueUdpServer,
		},
	}

	// Parse flags
	listen_host := flag.String("host", "localhost", "Network bind host")
	listen_port := flag.Int("port", 9999, "Network listen port")
	service_name := flag.String("service", "echo", "Service to run")
	log_level := flag.String("log", "info", "Enable debug logging")

	flag.Parse()

	util.Log.SetLevel(*log_level)
	service, exists := services[*service_name]
	if !exists {
		util.Log.Error("server: Service not supported: %v", *service_name)
		os.Exit(1)
	}

	// Listen for incoming connections.
	listenAddress := fmt.Sprintf("%v:%v", *listen_host, *listen_port)
	if service.protocol == PROTOCOL_TCP {
		startTCP(listenAddress, &service)
	} else if service.protocol == PROTOCOL_UDP {
		startUDP(listenAddress, &service)
	} else {
		util.Log.Error("server: Protocal not supported: %v", service.protocol)
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

	util.Log.Info("server: Started service: [%v]", service.name)
	util.Log.Info("server: TCP Listening on [%v]", conn_spec)
	for {
		// Listen for an incoming connection.
		connection, err := listener.Accept()
		if err != nil {
			panic(fmt.Errorf("server: Error accepting connection: %w", err))
		}
		// Handle connections in a new goroutine.
		util.Log.Info("server: saw connection from: [%s]", connection.RemoteAddr())
		go executeTcpHandler(connection, service)
	}
}

func executeTcpHandler(connection io.ReadWriteCloser, service *Service) {
	start := time.Now()
	bytesWritten, err := service.tcpHandlerFunc(connection)
	elapsed := time.Since(start)
	if err != nil {
		util.Log.Warn("%v: Handler execution failed => %v (in %v)", service.name, err.Error(), elapsed)
	} else {
		util.Log.Info("%v: Handler completed; Wrote %v bytes (in %v)", service.name, bytesWritten, elapsed)
	}
}

func startUDP(listenAddress string, service *Service) {
	util.Log.Info("server: Starting service: [%v]", service.name)
	err := service.udpServerFunc(listenAddress)
	if err != nil {
		util.Log.Error("server: Service [%v] exited with error %w", service.name, err)
		os.Exit(1)
	}
}
