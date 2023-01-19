package helpers

import (
	"fmt"
	"io"
)

func EchoHandler(conn io.ReadWriteCloser) (int64, error) {
	defer conn.Close()
	bytesWritten, err := io.Copy(conn, conn)
	if err != nil {
		return 0, fmt.Errorf("echo: Error handling connection %w", err)
	}
	return bytesWritten, nil
}
