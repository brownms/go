package handlers

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"proto-challenge/internal/util"
	"regexp"
	"strings"
)

const (
	UpstreamAddr   = "chat.protohackers.com:16963"
	MaliciousToken = "7YWHMfk9JZe0LM0g1ZauHuiSxhI"
)

type ProxyConn struct {
	conn io.ReadWriteCloser
	name string
}

type ProxySession struct {
	id     int64
	source *ProxyConn
	target *ProxyConn
}

var bogusCoinRegexp = regexp.MustCompile("^7([a-zA-Z0-9]{24,34})")

// todo: not the best replacement. could be done with a more complete regex but i'm lazy
func rewrite(message string) string {
	result := message
	tokens := strings.Split(message, " ")
	for _, token := range tokens {
		s := len(token)
		if s >= 26 && s <= 35 && bogusCoinRegexp.Match([]byte(token)) {
			result = strings.Replace(result, token, MaliciousToken, 1)
		}
	}
	return result
}

// watch source connection and send everything to the target
// returns the number of bytes written to the upstream
func proxy(id int64, source *ProxyConn, target *ProxyConn) int64 {
	defer target.conn.Close()
	reader := bufio.NewReader(source.conn)
	totalBytesWritten := 0
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		message = strings.TrimSuffix(message, "\n")
		util.Log.Debug("mob_proxy: [conn:%v] [%v=>%v]; msg: [%v]", id, source.name, target.name, message)
		bytesWritten, err := (target.conn).Write([]byte(rewrite(message) + "\n"))
		if err != nil {
			panic(fmt.Errorf("writing to target [%v] %w", target.name, err))
		}
		totalBytesWritten += bytesWritten
	}

	return int64(totalBytesWritten)
}

var connId int64 = 0

func ChatMobProxyHandler(conn io.ReadWriteCloser) (int64, error) {
	connId += 1

	upstreamConn, err := net.Dial("tcp", UpstreamAddr)
	if err != nil {
		return 0, fmt.Errorf("mob_proxy: [conn:%v] failed to connect to upstream server: %w", connId, err)
	}

	util.Log.Info("mob_proxy: [conn:%v] connected downstream ", connId)
	defer upstreamConn.Close()
	session := &ProxySession{
		id:     connId,
		source: &ProxyConn{conn: conn, name: "downstream"},
		target: &ProxyConn{conn: upstreamConn, name: "upstream"},
	}

	// proxy downstream => upstream
	go proxy(session.id, session.source, session.target)

	// proxy upstream => downstream
	bytesProxied := proxy(session.id, session.target, session.source)
	util.Log.Info("mob_proxy: [conn:%v] completed proxy session; %v bytes written", connId, bytesProxied)
	return bytesProxied, nil

}
