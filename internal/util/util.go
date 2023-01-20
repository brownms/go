package util

import (
	"bytes"
	"fmt"
	"net"
)

func Log(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

type UdpRequestReadWriteCloser struct {
	conn   *net.UDPConn
	sender *net.UDPAddr
	reader bytes.Reader
}

func (rwc *UdpRequestReadWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.reader.Read(p)
}

func (rwc *UdpRequestReadWriteCloser) Write(value []byte) (n int, err error) {
	return rwc.conn.WriteToUDP(value, rwc.sender)
}

func (rwc *UdpRequestReadWriteCloser) Close() error {
	return fmt.Errorf("refusing to close single UDP connection %v", rwc.conn)
}

func NewUdpRequestWrapper(conn *net.UDPConn, senderAddr *net.UDPAddr, reader *bytes.Reader) *UdpRequestReadWriteCloser {
	return &UdpRequestReadWriteCloser{
		conn:   conn,
		sender: senderAddr,
		reader: *reader,
	}
}
