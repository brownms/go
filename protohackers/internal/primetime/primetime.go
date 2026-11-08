package primetime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"log/slog"
)

type PrimeRequest struct {
	Method string         `json:"method"`
	Number *BigNumberJSON `json:"number"`
}

type PrimeResponse struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func Handler(conn io.ReadWriteCloser) (int64, error) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	total_bytes_written := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		var response_str string
		if len(line) > 0 {
			response_str = handleRequest(line)
		} else {
			response_str = "Invalid Request"
		}

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

func handleRequest(requestLine string) string {
	slog.Debug("raw input", "line", requestLine)
	request := PrimeRequest{}
	err := json.Unmarshal([]byte(requestLine), &request)
	if err != nil {
		// failed to parse JSON
		slog.Info("failed to parse JSON", "err", err)
		return "Invalid JSON"
	}

	if request.Method != "isPrime" {
		// unsupported method
		return fmt.Sprintf("Invalid Method: [%v]", request.Method)
	}

	if request.Number == nil {
		return "Missing 'number' property"
	}

	response := PrimeResponse{
		Method: request.Method,
		Prime:  request.Number.IntVal != nil && request.Number.IntVal.ProbablyPrime(0),
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Sprintf("Failed to Marshal response [%v]\n", err.Error())
	}
	return string(jsonBytes)
}
