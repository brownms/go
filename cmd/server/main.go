package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

const (
	CONN_TYPE = "tcp"
)

func log(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func main() {
	// Parse flags
	listen_host := flag.String("host", "localhost", "TCP bind host")
	listen_port := flag.Int("port", 9999, "TCP listen port")

	flag.Parse()

	// Listen for incoming connections.
	conn_spec := fmt.Sprintf("%v:%v", *listen_host, *listen_port)
	listener, err := net.Listen(CONN_TYPE, conn_spec)
	if err != nil {
		log("Failed to listen %e", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer listener.Close()

	log("Echo service started; Listening on %v", conn_spec)
	for {
		// Listen for an incoming connection.
		conn, err := listener.Accept()
		if err != nil {
			log("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		log("Saw connection from: %s", conn.RemoteAddr())
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	bytesWritten, err := io.Copy(conn, conn)
	if err != nil {
		log("Error handling connection from %s => %v", conn.RemoteAddr(), err.Error())
	} else {
		log("Wrote %v bytes to: %s", bytesWritten, conn.RemoteAddr())
	}
}
