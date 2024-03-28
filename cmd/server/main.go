package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	echo "proto-challenge/internal/0_echo"
	prime "proto-challenge/internal/1_prime"
	means "proto-challenge/internal/2_means"
	chat "proto-challenge/internal/3_budget_chat"
	udp "proto-challenge/internal/4_udp"
	speed "proto-challenge/internal/6_speed"
	"proto-challenge/internal/util"
	"time"
)

type Protocol byte

const (
	PROTOCOL_TCP Protocol = 'T'
	PROTOCOL_UDP Protocol = 'U'
)

type TcpServiceHandler func(conn io.ReadWriteCloser) (int64, error)
type UdpServiceHandler func(seq int64, request string, conn io.Writer) error

type Service struct {
	protocol  Protocol
	name      string
	handleTcp TcpServiceHandler
	handleUdp UdpServiceHandler
}

func main() {
	services := map[string]Service{
		"echo": {
			name:      "echo",
			protocol:  PROTOCOL_TCP,
			handleTcp: echo.EchoHandler,
		},
		"prime": {
			name:      "prime",
			protocol:  PROTOCOL_TCP,
			handleTcp: prime.PrimeHandler,
		},
		"means": {
			name:      "means",
			protocol:  PROTOCOL_TCP,
			handleTcp: means.MeansHandler,
		},
		"chat": {
			name:      "chat",
			protocol:  PROTOCOL_TCP,
			handleTcp: chat.ChatHandler,
		},
		"udp_kv": {
			name:      "udp_kv",
			protocol:  PROTOCOL_UDP,
			handleUdp: udp.KeyValueUdpHandler,
		},
		"mob": {
			name:      "mob",
			protocol:  PROTOCOL_TCP,
			handleTcp: chat.ChatMobProxyHandler,
		},
		"speed": {
			name:      "speed",
			protocol:  PROTOCOL_TCP,
			handleTcp: speed.SpeedDaemonHandler,
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
	bytesWritten, err := service.handleTcp(connection)
	elapsed := time.Since(start)
	if err != nil {
		util.Log.Warn("%v: Handler execution failed => %v (in %v)", service.name, err.Error(), elapsed)
	} else {
		util.Log.Info("%v: Handler completed; Wrote %v bytes (in %v)", service.name, bytesWritten, elapsed)
	}
}

type UdpWriter struct {
	connection  *net.UDPConn
	sendAddress *net.UDPAddr
}

func (w *UdpWriter) Write(value []byte) (n int, err error) {
	return w.connection.WriteToUDP(value, w.sendAddress)
}

func startUDP(listenAddress string, service *Service) {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddress)
	if err != nil {
		panic(fmt.Errorf("server: Failed to resolve UDP address from %v => %w", listenAddress, err))
	}

	connection, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(fmt.Errorf("server: Failed to start UDP listen on %v => %w", udpAddr, err))
	}

	util.Log.Info("server: Starting service: [%v]", service.name)
	util.Log.Info("server: UDP Listening on [%v]", listenAddress)

	reqNum := int64(0)
	for {
		// Handle connections in a go routine
		buffer := make([]byte, 1024)
		bytesRead, sendAddress, err := connection.ReadFromUDP(buffer)
		if err != nil {
			util.Log.Error("udp_server: failed to read UDP => %w", err)
			continue
		}
		reqNum += 1
		util.Log.Info("udp_server: [%v] Received [%v] bytes", reqNum, bytesRead)
		request := string(buffer[:bytesRead])
		err = service.handleUdp(reqNum, request, &UdpWriter{
			connection:  connection,
			sendAddress: sendAddress,
		})
		if err != nil {
			util.Log.Error("udp_server: [%v] Handler returned Error %w", reqNum, err)
			continue
		}
	}
}
