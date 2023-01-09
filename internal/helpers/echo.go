package helpers

import (
	"errors"
	"fmt"
	"io"
	"net"
)

func EchoHandler(conn net.Conn) (int64, error) {
	defer conn.Close()
	bytesWritten, err := io.Copy(conn, conn)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("echo: Error handling connection from %s => %v", conn.RemoteAddr(), err.Error()))
	}
	return bytesWritten, nil
}
