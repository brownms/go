package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
)

type PrimeRequest struct {
	method string
	number int
}

type PrimeResponse struct {
	method string
	prime  bool
}

func PrimeHandler(conn net.Conn) (int64, error) {
	defer conn.Close()
	reqBytes, err := io.ReadAll(conn)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("prime: Error reading from connection %s => %v", conn.RemoteAddr(), err.Error()))
	}

	var request PrimeRequest
	err = json.Unmarshal(reqBytes, &request)
	if err != nil {
		// failed to parse JSON
		Log("prime: failed to parse JSON => %v", err.Error())
		bytesWritten, err := conn.Write([]byte("Invalid JSON\n"))
		if err != nil {
			return int64(bytesWritten), err
		}
		return int64(bytesWritten), nil
	}

	if request.method != "isPrime" {
		// unsupported method
		bytesWritten, err := conn.Write([]byte(fmt.Sprintf("Invalid Method: [%v]\n", request.method)))
		if err != nil {
			return int64(bytesWritten), err
		}
		return int64(bytesWritten), nil
	}

	response := PrimeResponse{
		method: request.method,
		prime:  true,
	}
	responseBytes, err := json.Marshal(response)
	bytesWritten, err := conn.Write(responseBytes)
	if err != nil {
		return int64(bytesWritten), err
	}
	return int64(bytesWritten), nil
}
