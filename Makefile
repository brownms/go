.PHONY: all echo clean

all: server
clean: out
	rm -rf out

server: cmd/server/*.go internal/*/*.go
	[ -d out ] || mkdir out
	go build -o out/server_darwin ./cmd/server/*.go
	GOOS=linux GOARCH=arm64 go build -o out/server_linux_arm64 ./cmd/server/*.go

publish: server
	# ssh aws-proto "pkill -f server_linux_arm64"
	scp out/server_linux_arm64 aws-proto:

test:
	go test -v ./internal/handlers

scratch:
	go run scratch.go