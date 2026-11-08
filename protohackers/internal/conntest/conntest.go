// Package conntest provides an in-memory io.ReadWriteCloser for testing connection handlers.
package conntest

import (
	"bytes"
	"io"
)

// Conn reads from fixed input and records each Write separately.
type Conn struct {
	reader  io.Reader
	closed  bool
	results [][]byte
}

// NewFromBytes returns a Conn whose reads return input.
func NewFromBytes(input []byte) *Conn {
	return &Conn{reader: bytes.NewReader(input)}
}

// NewFromString returns a Conn whose reads return input.
func NewFromString(input string) *Conn {
	return NewFromBytes([]byte(input))
}

func (c *Conn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *Conn) Write(p []byte) (int, error) {
	c.results = append(c.results, bytes.Clone(p))
	return len(p), nil
}

func (c *Conn) Close() error {
	c.closed = true
	return nil
}

// IsClosed reports whether Close has been called.
func (c *Conn) IsClosed() bool {
	return c.closed
}

// Results returns the bytes from each Write call, in order.
func (c *Conn) Results() [][]byte {
	return c.results
}
