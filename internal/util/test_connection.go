package util

import (
	"bytes"
	"io"
)

type TestConnection struct {
	reader  io.Reader
	closed  bool
	results [][]byte
}

func (conn *TestConnection) Read(p []byte) (n int, err error) {
	return conn.reader.Read(p)
}

func (conn *TestConnection) Write(value []byte) (n int, err error) {
	// util.Log("Write: Saw bytes: %x => %d", value, value)
	conn.results = append(conn.results, value)
	return len(value), nil
}

func (conn *TestConnection) Close() error {
	conn.closed = true
	return nil
}

func (conn *TestConnection) IsClosed() bool {
	return conn.closed
}

func (conn *TestConnection) Results() [][]byte {
	return conn.results
}

func NewTestConnectionFromBytes(source_bytes []byte) *TestConnection {
	return &TestConnection{
		closed:  false,
		results: [][]byte{},
		reader:  bytes.NewReader(source_bytes),
	}
}

func NewTestConnectionFromString(source_string string) *TestConnection {
	return NewTestConnectionFromBytes([]byte(source_string))
}
