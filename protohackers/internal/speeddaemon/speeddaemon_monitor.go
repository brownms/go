package speeddaemon

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"
)

type Client struct {
	id            uint16
	conn          io.ReadWriteCloser
	bytesSent     atomic.Int64
	dispatcher    *Dispatcher
	camera        *Camera
	heartbeatStop chan struct{}
	shutdownOnce  sync.Once
}

func (c *Client) isCamera() bool {
	return c.camera != nil
}

func (c *Client) isDispatcher() bool {
	return c.dispatcher != nil
}

func (c *Client) isNew() bool {
	return !c.isCamera() && !c.isDispatcher()
}

// startHeartbeat sends heartbeats until shutdown, or until a send fails, which disconnects the client.
func (c *Client) startHeartbeat(intervalDeciSeconds uint32) error {
	if c.heartbeatStop != nil {
		return fmt.Errorf("[client:%v] received duplicate heartbeat request", c.id)
	}

	interval := time.Duration(intervalDeciSeconds) * 100 * time.Millisecond
	stop := make(chan struct{})
	c.heartbeatStop = stop
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for sendCount := 0; ; sendCount++ {
			select {
			case <-stop:
				return
			case now := <-ticker.C:
				if sendCount%10 == 0 {
					slog.Debug("sending heartbeat", "client", c.id, "count", sendCount, "at", now)
				}
				if err := c.send(msgOutHeartbeat{mType: msgType_oHeartbeat}); err != nil {
					slog.Error("failed to send heartbeat", "client", c.id, "at", now, "err", err)
					c.disconnect()
					return
				}
			}
		}
	}()

	slog.Info("started heartbeat", "client", c.id, "interval", interval)
	return nil
}

func (c *Client) send(message any) error {
	slog.Debug("sending message", "client", c.id, "type", fmt.Sprintf("%T", message), "message", message)
	mType := msgType_None
	switch msg := message.(type) {
	case msgOutTicket:
		mType = msgType_oTicket
	case msgOutError:
		mType = msgType_oError
	case msgOutHeartbeat:
		mType = msgType_oHeartbeat
	case msgInAmCamera:
		return fmt.Errorf("[client:%v] non-writeable msgType: %v", c.id, msg.mType)
	case msgInAmDispatcher:
		return fmt.Errorf("[client:%v] non-writeable msgType: %v", c.id, msg.mType)
	case msgInWantHeartbeat:
		return fmt.Errorf("[client:%v] non-writeable msgType: %v", c.id, msg.mType)
	case msgInPlate:
		return fmt.Errorf("[client:%v] non-writeable msgType: %v", c.id, msg.mType)
	}

	if mType == msgType_None {
		return fmt.Errorf("[client:%v] unknown message type => message: [%x]", c.id, message)
	}

	strategy, ok := strategies[mType]
	if !ok {
		return fmt.Errorf("[client:%v] no strategy found for msgType: %v", c.id, mType)
	}
	msgBytes, err := strategy.serialize(message)
	if err != nil {
		return fmt.Errorf("[client:%v] failed to serialize msgType: %v => %w", c.id, mType, err)
	}
	n, err := c.conn.Write(msgBytes)
	c.bytesSent.Add(int64(n))
	if err != nil {
		return fmt.Errorf("[client:%v] failed to write bytes for msgType: %v => %w", c.id, mType, err)
	}
	return nil
}

// disconnect closes the connection, which ends the client's read loop; the
// monitor then stops the client.
func (c *Client) disconnect() {
	slog.Debug("disconnecting", "client", c.id)
	c.conn.Close()
}

// shutdown stops the client's heartbeat. It is safe to call more than once.
func (c *Client) shutdown() {
	c.shutdownOnce.Do(func() {
		if c.heartbeatStop != nil {
			close(c.heartbeatStop)
			slog.Info("stopped heartbeat", "client", c.id)
		}
	})
}

type Dispatcher struct {
	// track all roads managed by this dispatcher with an O(1) lookup
	roads map[uint16]bool
}

type Camera struct {
	road  uint16
	mile  uint16
	limit uint16
}

type Observation struct {
	mile      uint16
	timestamp uint32
}

type Car struct {
	plate              string
	observationsByRoad map[uint16][]Observation
	ticketsByDay       map[uint32]*msgOutTicket
}

func (c *Car) observe(road uint16, mile uint16, timestamp uint32) {
	observations, ok := c.observationsByRoad[road]
	if !ok {
		c.observationsByRoad[road] = []Observation{}
	}
	observations = append(observations, Observation{mile: mile, timestamp: timestamp})
	// sort by mile (position)
	sort.Slice(observations, func(i, j int) bool { return observations[i].timestamp < observations[j].timestamp })
	c.observationsByRoad[road] = observations // necessary ?
}

func (c *Car) trackTicket(ticket *msgOutTicket) {
	day1, day2 := days(ticket)
	c.ticketsByDay[day1] = ticket
	c.ticketsByDay[day2] = ticket
}

func (c *Car) wasTicketedOn(day uint32) bool {
	if ticket, ok := c.ticketsByDay[day]; ok {
		slog.Debug("already ticketed", "plate", ticket.plate, "day", day, "ticket", ticket)
		return true
	}
	return false
}

// SpeedMonitor holds the state shared by every camera and dispatcher connection.
type SpeedMonitor struct {
	cars         map[string]Car
	limits       map[uint16]uint16
	clients      map[uint16]*Client
	ticketsState map[msgOutTicket]bool
	stateMutex   sync.Mutex
	lastClientID atomic.Uint32
}

// NewMonitor returns an empty SpeedMonitor.
func NewMonitor() *SpeedMonitor {
	return &SpeedMonitor{
		cars:         make(map[string]Car),
		limits:       make(map[uint16]uint16),
		clients:      make(map[uint16]*Client),
		ticketsState: make(map[msgOutTicket]bool),
	}
}

const (
	hourSeconds = 3600
	daySeconds  = 86400
)

func days(ticket *msgOutTicket) (uint32, uint32) {
	return ticket.timestamp1 / 86400, ticket.timestamp2 / 86400
}

func computeSpeedMph(obs1 Observation, obs2 Observation) float64 {
	distance := math.Abs(float64(obs2.mile) - float64(obs1.mile))
	timeDiffSeconds := math.Abs(float64(obs2.timestamp - obs1.timestamp))
	timeDiffHours := timeDiffSeconds / hourSeconds
	speedMph := distance / timeDiffHours
	slog.Debug("computed speed", "miles", distance, "seconds", timeDiffSeconds, "hours", timeDiffHours, "mph", speedMph)
	return math.Round(speedMph)
}

func (m *SpeedMonitor) findDispatcher(road uint16) *Client {
	dispatchers := []uint16{}
	for _, client := range m.clients {
		if client.isDispatcher() && client.dispatcher.roads[road] {
			dispatchers = append(dispatchers, client.id)
		}
	}

	if len(dispatchers) == 0 {
		slog.Debug("no dispatchers for road", "road", road)
		return nil
	}
	// pick a random dispatcher
	dispIndex := rand.Intn(len(dispatchers))
	dispatcherClientId := dispatchers[dispIndex]
	dispatcher := m.clients[dispatcherClientId]
	slog.Debug("dispatcher found", "road", road, "index", dispIndex, "dispatchers", len(dispatchers), "client", dispatcher.id)
	return dispatcher
}

func (m *SpeedMonitor) sendPendingTickets() error {
	for ticket, sent := range m.ticketsState {
		if sent {
			continue
		}
		// find dispatcher
		dispatcher := m.findDispatcher(ticket.road)
		if dispatcher != nil {
			car, ok := m.cars[ticket.plate]
			if !ok {
				return fmt.Errorf("[client:%v] refusing to send ticket to plate [%v] with untracked car => [%+v]", dispatcher.id, ticket.plate, ticket)
			}
			day1, day2 := days(&ticket)
			slog.Debug("unsent ticket", "client", dispatcher.id, "plate", car.plate, "day1", day1, "day2", day2, "ticket", ticket)
			if car.wasTicketedOn(day1) || car.wasTicketedOn(day2) {
				slog.Info("already ticketed", "client", dispatcher.id, "plate", ticket.plate, "day1", day1, "day2", day2, "ticket", ticket)
				return nil
			}
			slog.Info("sending ticket", "client", dispatcher.id, "plate", car.plate, "day1", day1, "day2", day2, "ticket", ticket)
			err := dispatcher.send(ticket)
			if err != nil {
				return fmt.Errorf("[client:%v] sending ticket: [%+v] => %w", dispatcher.id, ticket, err)
			}
			car.trackTicket(&ticket)
			m.ticketsState[ticket] = true
		} else {
			slog.Warn("no dispatcher for road; ticket not sent", "road", ticket.road, "ticket", ticket)
		}
	}
	return nil
}

func (m *SpeedMonitor) checkForTickets(car Car, road uint16) error {
	observations := car.observationsByRoad[road]
	if len(observations) < 2 {
		return nil
	}
	limit, ok := m.limits[road]
	if !ok {
		return fmt.Errorf("observation for road [%v] without camera", road)
	}
	for i := 0; i < len(observations)-1; i += 2 {
		obs1 := observations[i]
		obs2 := observations[i+1]
		if obs2.timestamp < obs1.timestamp {
			// order by occurrence
			obs1 = observations[i+1]
			obs2 = observations[i]
		}
		speedMph := computeSpeedMph(obs1, obs2)
		exceedsLimit := speedMph > float64(limit)
		slog.Debug("speed check", "plate", car.plate, "mph", speedMph, "road", road, "mile1", obs1.mile, "mile2", obs2.mile, "limit", limit, "exceeds", exceedsLimit)
		if exceedsLimit {
			// generate ticket
			ticket := msgOutTicket{
				mType:      msgType_oTicket,
				plate:      car.plate,
				road:       road,
				mile1:      obs1.mile,
				timestamp1: obs1.timestamp,
				mile2:      obs2.mile,
				timestamp2: obs2.timestamp,
				speed:      uint16(speedMph * 100),
			}
			_, ok := m.ticketsState[ticket]
			if !ok {
				slog.Debug("ticket queued", "ticket", ticket)
				m.ticketsState[ticket] = false
			} else {
				slog.Debug("ticket already queued", "ticket", ticket)
			}
		}
	}
	return m.sendPendingTickets()
}

func (m *SpeedMonitor) stop(client *Client) {
	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()

	slog.Debug("stopping", "client", client.id)
	client.shutdown()
	delete(m.clients, client.id)
	slog.Info("client stopped", "client", client.id, "bytes_sent", client.bytesSent.Load())

	if len(m.clients) > 0 {
		return
	}

	// if no clients are left connected, dump unsent tickets
	sentCount := 0
	unsent := []msgOutTicket{}
	for ticket, sent := range m.ticketsState {
		if !sent {
			unsent = append(unsent, ticket)
		} else {
			sentCount++
		}
	}
	slog.Info("all clients disconnected", "sent", sentCount, "unsent", len(unsent))
	if len(unsent) > 0 {
		for _, ticket := range unsent {
			slog.Info("unsent ticket", "ticket", ticket)
		}
	}
}

func (m *SpeedMonitor) accept(client *Client, rawMessage any) error {
	// obtain lock to ensure only one concurrent writer to state
	// TODO => could make this more granular to speed up handling, but messages
	// are small and inexpensive; forcing serial handling is probably fine
	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()

	// track the client
	m.clients[client.id] = client

	slog.Debug("accepting message", "client", client.id, "type", fmt.Sprintf("%T", rawMessage), "message", rawMessage)
	switch msg := rawMessage.(type) {
	case msgInAmCamera:
		if !client.isNew() {
			// this client has already declared its type
			return fmt.Errorf("received IAmCamera from client with declared type")
		}
		// this client is a camera
		client.camera = &Camera{road: msg.road, mile: msg.mile, limit: msg.limit}
		m.limits[msg.road] = msg.limit

	case msgInAmDispatcher:
		if !client.isNew() {
			// this client has already declared its type
			return fmt.Errorf("received IAmDispatcher from client with declared type")
		}
		// this client is a dispatcher
		client.dispatcher = &Dispatcher{roads: map[uint16]bool{}}
		for _, road := range msg.roads {
			// add all roads
			slog.Debug("dispatcher connected", "road", road)
			client.dispatcher.roads[road] = true
			// check cars observed on roads; handles scenario where dispatcher
			// arrives and pending tickets exist for one their roads
			for _, car := range m.cars {
				_, ok := car.observationsByRoad[road]
				if ok {
					slog.Debug("checking plate for tickets", "client", client.id, "plate", car.plate, "road", road)
					return m.checkForTickets(car, road)
				}
			}
		}

	case msgInPlate:
		if !client.isCamera() {
			slog.Error("received camera message from non-camera client", "client", client.id, "type", fmt.Sprintf("%T", msg))
			break
		}
		// track plate position & time
		car, ok := m.cars[msg.plate]
		if !ok {
			car = Car{
				plate:              msg.plate,
				observationsByRoad: map[uint16][]Observation{},
				ticketsByDay:       map[uint32]*msgOutTicket{},
			}
			m.cars[msg.plate] = car
		}
		// observe the car's position and check for tickets
		road := client.camera.road
		car.observe(road, client.camera.mile, msg.timestamp)
		slog.Debug("checking plate for tickets", "client", client.id, "plate", msg.plate, "road", road)
		err := m.checkForTickets(car, road)
		if err != nil {
			return fmt.Errorf("[client:%v] checking plate [%v] for tickets on road: [%v] => %w", client.id, msg.plate, road, err)
		}
		return nil

	case msgInWantHeartbeat:
		if msg.interval == 0 {
			slog.Debug("heartbeat disabled (interval 0)", "client", client.id)
			return nil
		}
		err := client.startHeartbeat(msg.interval)
		if err != nil {
			errMsg := errorMessage(fmt.Sprintf("failed to start heartbeat task: %v", err.Error()))
			err := client.send(errMsg)
			if err != nil {
				return err
			}
			client.disconnect()
		}

	default:
		slog.Warn("unknown message", "client", client.id, "type", fmt.Sprintf("%T", msg))
	}
	return nil
}
