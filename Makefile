.PHONY: all clean server publish run test test_quiet lint scratch

SERVICE?=smoketest

all: server
clean:
	rm -rf out

server: test_quiet
	go build -o out/ ./protohackers/cmd/...
	GOOS=linux GOARCH=amd64 go build -o out/linux_amd64/ ./protohackers/cmd/...

publish: server
	scp out/linux_amd64/$(SERVICE) yoda:

run:
	go run ./protohackers/cmd/$(SERVICE)

test_quiet:
	go test ./...

test:
	go test -v ./...

lint:
	golangci-lint run

scratch:
	go run ./scratch
