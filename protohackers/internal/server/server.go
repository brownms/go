// Package server runs a protohackers handler on a TCP or UDP listener.
package server

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"time"
)

// TCPHandler serves one TCP connection and returns the number of bytes written.
type TCPHandler func(conn io.ReadWriteCloser) (int64, error)

// UDPHandler serves one UDP datagram; seq numbers datagrams in arrival order.
type UDPHandler func(seq int64, request string, conn io.Writer) error

type udpWriter struct {
	conn *net.UDPConn
	addr *net.UDPAddr
}

func (w *udpWriter) Write(p []byte) (int, error) {
	return w.conn.WriteToUDP(p, w.addr)
}

// ParseFlags parses the -host, -port and -log flags, installs a default slog
// logger at the requested level, and returns the listen address.
func ParseFlags() string {
	host := flag.String("host", "localhost", "network bind host")
	port := flag.Int("port", 9999, "network listen port")
	var level slog.Level
	flag.TextVar(&level, "log", slog.LevelInfo, "log level: debug, info, warn, error")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
	return net.JoinHostPort(*host, strconv.Itoa(*port))
}

// ServeTCP accepts connections on addr and serves each with handler in its own goroutine.
func ServeTCP(addr string, name string, handler TCPHandler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on TCP %v: %w", addr, err)
	}
	defer listener.Close()

	slog.Info("TCP listening", "service", name, "addr", addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accepting connection: %w", err)
		}
		slog.Info("connection", "service", name, "remote", conn.RemoteAddr())
		go serveTCPConn(conn, name, handler)
	}
}

func serveTCPConn(conn io.ReadWriteCloser, name string, handler TCPHandler) {
	start := time.Now()
	bytesWritten, err := handler(conn)
	elapsed := time.Since(start)
	if err != nil {
		slog.Warn("handler failed", "service", name, "elapsed", elapsed, "err", err)
		return
	}
	slog.Info("handler completed", "service", name, "bytes", bytesWritten, "elapsed", elapsed)
}

// ServeUDP reads datagrams on addr and passes each to handler, one at a time.
func ServeUDP(addr string, name string, handler UDPHandler) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolving UDP address %v: %w", addr, err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listening on UDP %v: %w", udpAddr, err)
	}
	defer conn.Close()

	slog.Info("UDP listening", "service", name, "addr", addr)
	var seq int64
	buf := make([]byte, 1024)
	for {
		n, sender, err := conn.ReadFromUDP(buf)
		if err != nil {
			slog.Error("reading UDP", "service", name, "err", err)
			continue
		}
		seq++
		slog.Info("datagram received", "service", name, "seq", seq, "bytes", n)
		if err := handler(seq, string(buf[:n]), &udpWriter{conn: conn, addr: sender}); err != nil {
			slog.Error("handler failed", "service", name, "seq", seq, "err", err)
		}
	}
}
