.PHONY: all echo clean

all: echo
clean: out
	rm -rf out

echo: cmd/echo/*.go internal/*/*.go
	[ -d out ] || mkdir out
	go build -o out/echo_darwin ./cmd/echo/*.go
	GOOS=linux GOARCH=arm64 go build -o out/echo_linux_arm64 ./cmd/echo/*.go

publish: echo
	# ssh aws-proto "pkill -f echo_linux_arm64"
	scp out/echo_linux_arm64 aws-proto: