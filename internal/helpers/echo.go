package helpers

import (
	"io"
	"net"
)

func EchoHandler(conn net.Conn) {
	defer conn.Close()
	bytesWritten, err := io.Copy(conn, conn)
	if err != nil {
		Log("echo: Error handling connection from %s => %v", conn.RemoteAddr(), err.Error())
	} else {
		Log("echo: Wrote %v bytes to: %s", bytesWritten, conn.RemoteAddr())
	}
}
