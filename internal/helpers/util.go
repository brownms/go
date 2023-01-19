package helpers

import (
	"fmt"
	"io"
	"net"
	"time"
)

type ServiceHandler func(conn io.ReadWriteCloser) (int64, error)

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

// type Recorder struct {
// 	file os.File
// 	Record func(request string) error
// 	Stop func() error``
// }

// const (
// 	RECORDING_FILE = "requests.txt"
// )

// func StartRecording(recorder) (Recorder, error) {
// 	f, err := os.OpenFile(RECORDING_FILE, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		Log("StartRecording failed to open file: %v", err.Error())
// 	}

// 	logger := log.New(f, "prefix", log.LstdFlags)
// 	logger.Println("text to append")
// 	logger.Println("more text to append")
// }
