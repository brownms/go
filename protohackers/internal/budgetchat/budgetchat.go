package budgetchat

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
)

const maxNameLength = 20

var validName = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// Room is a chat room shared by every connection served by Handle.
type Room struct {
	mu      sync.Mutex
	members map[string]*member
}

type member struct {
	name         string
	conn         io.Writer
	bytesWritten int64
}

// NewRoom returns an empty Room.
func NewRoom() *Room {
	return &Room{members: map[string]*member{}}
}

// Handle serves one chat client: it asks for a name, joins the room, and
// relays each line the client sends to every other member until the client disconnects.
func (r *Room) Handle(conn io.ReadWriteCloser) (int64, error) {
	defer conn.Close()

	self := &member{conn: conn}
	if err := self.send("Welcome to budgetchat! What shall I call you?"); err != nil {
		return self.bytesWritten, fmt.Errorf("sending welcome message: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return self.bytesWritten, scanner.Err()
	}
	self.name = strings.TrimSpace(scanner.Text())
	if err := r.join(self); err != nil {
		if sendErr := self.send(err.Error()); sendErr != nil {
			slog.Info("failed to send error message", "err", sendErr)
		}
		slog.Info("session ended before join", "user", self.name, "reason", err)
		return self.bytesWritten, nil
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		r.broadcast(self, fmt.Sprintf("[%v] %s", self.name, line))
	}
	written := r.leave(self)

	if err := scanner.Err(); err != nil {
		return written, fmt.Errorf("reading connection: %w", err)
	}
	return written, nil
}

// join adds m to the room, or returns an error to show the client if m.name is invalid or taken.
func (r *Room) join(m *member) error {
	if err := validateName(m.name); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, taken := r.members[m.name]; taken {
		return fmt.Errorf("name already in use: %v", m.name)
	}
	list := strings.Join(slices.Sorted(maps.Keys(r.members)), ",")
	if list == "" {
		list = "<none present>"
	}
	if err := m.send("* The room contains: " + list); err != nil {
		slog.Info("failed to send presence message", "err", err)
	}
	r.broadcastLocked(m, fmt.Sprintf("* %v has entered the room", m.name))
	r.members[m.name] = m
	return nil
}

// leave removes m from the room, tells the others, and returns the bytes written to m.
func (r *Room) leave(m *member) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.members, m.name)
	slog.Info("session ended; broadcasting exit", "user", m.name)
	r.broadcastLocked(m, fmt.Sprintf("* %v has left the room", m.name))
	return m.bytesWritten
}

func (r *Room) broadcast(sender *member, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.broadcastLocked(sender, message)
}

func (r *Room) broadcastLocked(sender *member, message string) {
	for name, receiver := range r.members {
		if name == sender.name {
			continue
		}
		if err := receiver.send(message); err != nil {
			slog.Info("failed to send broadcast", "to", name, "err", err)
		}
	}
}

func (m *member) send(message string) error {
	n, err := io.WriteString(m.conn, message+"\n")
	m.bytesWritten += int64(n)
	if err != nil {
		return fmt.Errorf("sending message %q: %w", message, err)
	}
	return nil
}

func validateName(name string) error {
	switch {
	case name == "":
		return errors.New("name cannot be empty")
	case len(name) > maxNameLength:
		return fmt.Errorf("name cannot have more than %d characters", maxNameLength)
	case !validName.MatchString(name):
		return fmt.Errorf("invalid name: %v", name)
	}
	return nil
}
