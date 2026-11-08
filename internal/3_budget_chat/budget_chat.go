package handlers

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"proto-challenge/internal/util"
	"regexp"
	"strings"
)

type ChatState byte
type ChatMessageType byte

const (
	StateNew          ChatState = 'N'
	StateJoined       ChatState = 'J'
	StateDisconnected ChatState = 'D'

	MessagePresence ChatMessageType = 'P'
	MessageEntry    ChatMessageType = 'E'
	MessageExit     ChatMessageType = 'X'
	MessageSend     ChatMessageType = 'S'
	NameRegexp      string          = `^[A-Za-z0-9]*$`
)

type ChatSession struct {
	bytes_written int64
	state         ChatState
	name          string
	conn          io.ReadWriteCloser
}

var sessions = map[string]*ChatSession{}

func (sender *ChatSession) Send(message string) error {
	bytes_written, err := sender.conn.Write([]byte(message + "\n"))
	if err != nil {
		return fmt.Errorf("failed to send message: [%v] => %w", message, err)
	}
	sender.bytes_written += int64(bytes_written)
	return nil
}

func (sender *ChatSession) Broadcast(msgType ChatMessageType, message *string) {
	for name, receiver := range sessions {
		if name != sender.name && receiver.state == StateJoined {
			if err := receiver.Send(sender.formatMessage(msgType, message)); err != nil {
				util.Log.Info("chat: failed to send broadcast message: [%v]", err.Error())
			}
		}
	}
}

func (sender *ChatSession) FindJoinedPeers() []string {
	peer_names := []string{}
	for name, receiver := range sessions {
		if name != sender.name && receiver.state == StateJoined {
			peer_names = append(peer_names, receiver.name)
		}
	}
	return peer_names
}

func (sender *ChatSession) formatMessage(msgType ChatMessageType, message *string) string {
	switch msgType {
	case MessagePresence:
		peer_names := strings.Join(sender.FindJoinedPeers(), ",")
		if peer_names == "" {
			peer_names = "<none present>"
		}
		return fmt.Sprintf("* The room contains: %v", peer_names)
	case MessageEntry:
		return fmt.Sprintf("* %v has entered the room", sender.name)
	case MessageExit:
		return fmt.Sprintf("* %v has left the room", sender.name)
	case MessageSend:
		return fmt.Sprintf("[%v] %s", sender.name, *message)
	}
	return fmt.Sprintf("unknown message type => %v", msgType)
}

func ChatHandler(conn io.ReadWriteCloser) (int64, error) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	session := ChatSession{
		state: StateNew,
		conn:  conn,
	}

	err := session.Send("Welcome to budgetchat! What shall I call you?")
	if err != nil {
		return session.bytes_written, fmt.Errorf("chat: sending welcome message %w", err)
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if session.state == StateNew {
			// accept new name and transition to Joined state
			name, err := validateName(line)
			if err != nil {
				session.state = StateDisconnected
				if err := session.Send(err.Error()); err != nil {
					util.Log.Info("chat: failed to send error message: [%v]", err.Error())
				}
				break
			}
			session.name = name
			session.state = StateJoined
			if err := session.Send(session.formatMessage(MessagePresence, nil)); err != nil {
				util.Log.Info("chat: failed to send presence message: [%v]", err.Error())
			}
			session.Broadcast(MessageEntry, nil)
			sessions[name] = &session

		} else if session.state == StateJoined {
			// broadcast message to others
			session.Broadcast(MessageSend, &line)
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("chat: Error reading connection %w", err)
	}
	if session.state == StateJoined {
		util.Log.Info("chat: Session ended for Joined user: [%v]; broadcasting Exit", session.name)
		session.Broadcast(MessageExit, nil)
		delete(sessions, session.name)
	} else {
		util.Log.Info("chat: Session ended for non-Joined user: [%v]", session.name)
	}
	return session.bytes_written, nil
}

func validateName(name string) (string, error) {
	if len(name) == 0 {
		return name, errors.New("name cannot be empty")
	}
	if len(name) > 20 {
		return name, errors.New("name cannot have more than 20 characters")
	}
	match, err := regexp.Match(NameRegexp, []byte(name))
	if err != nil {
		return name, fmt.Errorf("failed to check name: %w", err)
	}
	if !match {
		return name, fmt.Errorf("invalid name: %v", name)
	}
	return name, nil
}
