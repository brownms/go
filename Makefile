.PHONY: all echo clean

SERVICE?=echo

all: server
clean: out
	rm -rf out

server: cmd/server/*.go internal/*/*.go test_quiet
	[ -d out ] || mkdir out
	go build -o out/server_darwin ./cmd/server/*.go
	# GOOS=linux GOARCH=arm64 go build -o out/server_linux_arm64 ./cmd/server/*.go
	GOOS=linux GOARCH=amd64 go build -o out/server_linux_x64 ./cmd/server/*.go

publish: server
	# ssh aws-proto "pkill -f server_linux_arm64"
	# scp out/server_linux_arm64 aws-proto:
	scp out/server_linux_x64 yoda:

run:
	go run -v ./cmd/server/main.go -service=${SERVICE}

test_quiet:
	go test ./...

test:
	go test -v ./...

lint:
	golangci-lint run

scratch:
	go run scratch.go