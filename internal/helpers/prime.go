package helpers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

type PrimeRequest struct {
	Method string         `json:"method"`
	Number *BigNumberJSON `json:"number"`
}

type PrimeResponse struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func PrimeHandler(conn io.ReadWriteCloser) (int64, error) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	total_bytes_written := 0

	for scanner.Scan() {
		response_str := handleRequest(scanner.Text())
		bytes_written, err := conn.Write([]byte(response_str + "\n"))
		total_bytes_written += bytes_written
		if err != nil {
			return int64(total_bytes_written), err
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("prime: Error reading connection %w", err)
	}
	return int64(total_bytes_written), nil
}

func handleRequest(request_line string) string {
	Log("prime: raw input => [%v]", string(request_line))
	request := PrimeRequest{}
	err := json.Unmarshal([]byte(request_line), &request)
	if err != nil {
		// failed to parse JSON
		Log("prime: failed to parse JSON => %v", err.Error())
		return "Invalid JSON"
	}

	if request.Method != "isPrime" {
		// unsupported method
		return fmt.Sprintf("Invalid Method: [%v]", request.Method)
	}

	if request.Number == nil {
		return fmt.Sprintf("Missing 'number' property")
	}

	var is_prime = false
	if request.Number.int_val != nil && request.Number.int_val.ProbablyPrime(0) {
		is_prime = true
	}

	response := PrimeResponse{
		Method: request.Method,
		Prime:  is_prime,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Sprintf("Failed to Marshal response [%v]\n", err.Error())
	}
	return string(jsonBytes)
}
