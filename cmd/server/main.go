package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	pc "proto-challenge/internal/helpers"
)

type Handler func(net.Conn)

type Service struct {
	name         string
	handler_func Handler
}

const (
	CONN_TYPE = "tcp"
)

func main() {
	services := map[string]Service{
		"echo": {
			name:         "echo",
			handler_func: pc.EchoHandler,
		},
	}

	// Parse flags
	listen_host := flag.String("host", "localhost", "TCP bind host")
	listen_port := flag.Int("port", 9999, "TCP listen port")
	service_name := flag.String("service", "echo", "Service to run")

	flag.Parse()

	service, exists := services[*service_name]
	if !exists {
		pc.Log("proto: Service not supported: %v", *service_name)
		os.Exit(1)
	}

	// Listen for incoming connections.
	conn_spec := fmt.Sprintf("%v:%v", *listen_host, *listen_port)
	listener, err := net.Listen(CONN_TYPE, conn_spec)
	if err != nil {
		pc.Log("proto: Failed to listen %e", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer listener.Close()

	pc.Log("proto: Started service: [%v]", service.name)
	pc.Log("proto: Listening on [%v]", conn_spec)
	for {
		// Listen for an incoming connection.
		conn, err := listener.Accept()
		if err != nil {
			pc.Log("proto: Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		pc.Log("proto: saw connection from: [%s]", conn.RemoteAddr())
		go service.handler_func(conn)
	}
}
