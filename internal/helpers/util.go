package helpers

import (
	"fmt"
	"net"
	"time"
)

type ServiceHandler func(net.Conn) (int64, error)

func Log(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func ExecuteHandler(name string, conn net.Conn, handler ServiceHandler) {
	start := time.Now()
	bytesWritten, err := handler(conn)
	elapsed := time.Since(start)
	if err != nil {
		Log("%v: Handler execution failed => %v (in %v)", name, err.Error(), elapsed)
	} else {
		Log("%v: Handler completed; Wrote %v bytes (in %v)", name, bytesWritten, elapsed)
	}
}
