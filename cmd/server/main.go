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

type ServiceHandler func(conn io.ReadWriteCloser) (int64, error)

type Service struct {
	name         string
	handler_func ServiceHandler
}

const (
	CONN_TYPE = "tcp"
)

func main() {
	services := map[string]Service{
		"echo": {
			name:         "echo",
			handler_func: handlers.EchoHandler,
		},
		"prime": {
			name:         "prime",
			handler_func: handlers.PrimeHandler,
		},
		"means": {
			name:         "means",
			handler_func: handlers.MeansHandler,
		},
	}

	// Parse flags
	listen_host := flag.String("host", "localhost", "TCP bind host")
	listen_port := flag.Int("port", 9999, "TCP listen port")
	service_name := flag.String("service", "echo", "Service to run")

	flag.Parse()

	service, exists := services[*service_name]
	if !exists {
		util.Log("proto: Service not supported: %v", *service_name)
		os.Exit(1)
	}

	// Listen for incoming connections.
	conn_spec := fmt.Sprintf("%v:%v", *listen_host, *listen_port)
	listener, err := net.Listen(CONN_TYPE, conn_spec)
	if err != nil {
		util.Log("proto: Failed to listen %e", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer listener.Close()

	util.Log("proto: Started service: [%v]", service.name)
	util.Log("proto: Listening on [%v]", conn_spec)
	for {
		// Listen for an incoming connection.
		conn, err := listener.Accept()
		if err != nil {
			util.Log("proto: Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		util.Log("proto: saw connection from: [%s]", conn.RemoteAddr())
		go executeHandler(service.name, conn, service.handler_func)
	}
}

func executeHandler(name string, conn net.Conn, handler ServiceHandler) {
	start := time.Now()
	bytesWritten, err := handler(conn)
	elapsed := time.Since(start)
	if err != nil {
		util.Log("%v: Handler execution failed => %v (in %v)", name, err.Error(), elapsed)
	} else {
		util.Log("%v: Handler completed; Wrote %v bytes (in %v)", name, bytesWritten, elapsed)
	}
}
