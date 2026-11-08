package mobproxy

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"sync/atomic"
)

const (
	// DefaultUpstream is the reference Budget Chat server the challenge proxies to.
	DefaultUpstream = "chat.protohackers.com:16963"
	tonyAddress     = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"
)

var boguscoinAddress = regexp.MustCompile("^7[a-zA-Z0-9]{25,34}$")

// Proxy relays Budget Chat sessions to an upstream server, rewriting
// Boguscoin addresses in both directions.
type Proxy struct {
	upstream   string
	lastConnID atomic.Int64
}

type endpoint struct {
	conn io.ReadWriteCloser
	name string
}

// New returns a Proxy that dials upstream for each client.
func New(upstream string) *Proxy {
	return &Proxy{upstream: upstream}
}

// Handle proxies one client connection until either side disconnects, and
// returns the number of bytes written to the client.
func (p *Proxy) Handle(conn io.ReadWriteCloser) (int64, error) {
	id := p.lastConnID.Add(1)
	upstreamConn, err := net.Dial("tcp", p.upstream)
	if err != nil {
		conn.Close()
		return 0, fmt.Errorf("[conn:%v] connecting to upstream: %w", id, err)
	}
	slog.Info("connected downstream", "conn", id)

	downstream := endpoint{conn: conn, name: "downstream"}
	upstream := endpoint{conn: upstreamConn, name: "upstream"}
	go relay(id, downstream, upstream)
	written := relay(id, upstream, downstream)

	slog.Info("proxy session completed", "conn", id, "bytes", written)
	return written, nil
}

// relay copies lines from source to target, rewriting each one, until either
// side fails. It closes target when done, which also ends the opposite relay.
func relay(id int64, source endpoint, target endpoint) int64 {
	defer target.conn.Close()
	reader := bufio.NewReader(source.conn)
	var written int64
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			return written
		}
		message = strings.TrimSuffix(message, "\n")
		slog.Debug("proxying message", "conn", id, "from", source.name, "to", target.name, "message", message)
		n, err := io.WriteString(target.conn, rewrite(message)+"\n")
		written += int64(n)
		if err != nil {
			slog.Info("write failed; ending session", "conn", id, "to", target.name, "err", err)
			return written
		}
	}
}

func rewrite(message string) string {
	words := strings.Split(message, " ")
	for i, word := range words {
		if boguscoinAddress.MatchString(word) {
			words[i] = tonyAddress
		}
	}
	return strings.Join(words, " ")
}
