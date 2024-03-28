package handlers

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"proto-challenge/internal/util"
	"sort"
	"sync"
	"time"
)

type Client struct {
	id            uint16
	conn          *io.ReadWriteCloser
	bytesSent     int64
	dispatcher    *Dispatcher
	camera        *Camera
	heartbeatChan chan struct{}
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

func (c *Client) startHeartbeat(intervalDeciSeconds uint32) error {
	// check for existing heartbeat channel
	if c.heartbeatChan != nil {
		return fmt.Errorf("[client:%v] received duplicate heartbeat request", c.id)
	}

	// deciseconds (10 per second; e.g. 25 = 2.5 seconds)
	intervalDuration := time.Duration(intervalDeciSeconds*100) * time.Millisecond
	ticker := time.NewTicker(intervalDuration)
	stopChan := make(chan struct{})
	sendCount := 0
	go func() {
		for {
			select {
			case <-stopChan:
				ticker.Stop()
				return
			case currentTime := <-ticker.C:
				if sendCount == 0 || sendCount%10 == 0 {
					util.Log.Debug("[client:%v] sending heartbeat message [count: %v; sent: %v]", c.id, sendCount, currentTime)
				}
				if err := c.send(msgOutHeartbeat{mType: msgType_oHeartbeat}); err != nil {
					util.Log.Error("[client:%v] failed to send heartbeat message [at %v] => %v", c.id, err.Error())
					ticker.Stop()
					c.disconnect()
				}
				sendCount += 1
			}
		}
	}()

	util.Log.Info("[client:%v] started heartbeat beckground task (interval: %v)", c.id, intervalDuration)
	c.heartbeatChan = stopChan
	return nil
}

func (c *Client) send(message interface{}) error {
	util.Log.Trace("[client:%v] send raw message => [%T] => [%+v]", c.id, message, message)
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
	bytesSent, err := (*c.conn).Write(msgBytes)
	if err != nil {
		return fmt.Errorf("[client:%v] failed to write bytes for msgType: %v => %w", c.id, mType, err)
	}
	c.bytesSent += int64(bytesSent)
	return nil
}

func (c *Client) disconnect() {
	util.Log.Trace("[client:%v] disconnecting", c.id)
	defer c.shutdown()
	err := (*c.conn).Close()
	if err != nil {
		panic(fmt.Errorf("[client:%v] failed to close connection => %w", c.id, err))
	}

}

func (c *Client) shutdown() {
	util.Log.Trace("[client:%v] shutting down", c.id)
	if c.heartbeatChan != nil {
		util.Log.Info("[client:%v] stopped heartbeat ticker", c.id)
		c.heartbeatChan <- struct{}{}
		close(c.heartbeatChan)
	}
	util.Log.Trace("[client:%v] shutdown completed; sent %v bytes", c.id, c.bytesSent)
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
	// util.Log.Debug("plate: [%v] => ticketed on days: [%v] => [%+v]", ticket.plate, day, ticket)
	c.ticketsByDay[day1] = ticket
	c.ticketsByDay[day2] = ticket
}

func (c *Car) wasTicketedOn(day uint32) bool {
	if ticket, ok := c.ticketsByDay[day]; ok {
		util.Log.Debug("plate: [%v] => already ticketed on day: [%v] => [%+v]", ticket.plate, day, ticket)
		return true
	}
	return false
}

type SpeedMonitor struct {
	cars         map[string]Car
	limits       map[uint16]uint16
	clients      map[uint16]*Client
	ticketsState map[msgOutTicket]bool
	stateMutex   sync.Mutex
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
	util.Log.Trace("computeSpeedMph: saw car cover %vmi in %vs (%.4fh) => speed: %v", distance, timeDiffSeconds, timeDiffHours, speedMph)
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
		util.Log.Debug("no dispatchers available for road: [%v]", road)
		return nil
	}
	// pick a random dispatcher
	dispIndex := rand.Intn(len(dispatchers))
	dispatcherClientId := dispatchers[dispIndex]
	dispatcher := m.clients[dispatcherClientId]
	util.Log.Debug("dispatcher found for road: [%v]; using index [%v] of %v (dispatcher client: %v)", road, dispIndex, len(dispatchers), dispatcher.id)
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
			util.Log.Debug("[client:%v] saw unsent ticket for plate: [%v] on day: [%v:%v] => [%+v]", dispatcher.id, car.plate, day1, day2, ticket)
			if car.wasTicketedOn(day1) || car.wasTicketedOn(day2) {
				util.Log.Info("[client:%v] plate [%v]; already ticketed for day [%v:%v] => [%+v]", dispatcher.id, ticket.plate, day1, day2, ticket)
				return nil
			}
			util.Log.Info("[client:%v] sending ticket for plate: [%v] on day: [%v:%v] => [%+v]", dispatcher.id, car.plate, day1, day2, ticket)
			err := dispatcher.send(ticket)
			if err != nil {
				return fmt.Errorf("[client:%v] sending ticket: [%+v] => %w", dispatcher.id, ticket, err)
			}
			car.trackTicket(&ticket)
			m.ticketsState[ticket] = true
		} else {
			util.Log.Warn("unable to find dispatcher for road: [%v]; not sending ticket: [%+v]", ticket.road, ticket)
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
		util.Log.Debug("plate [%v] was going [%vmph] on: [%v:%v=>%v] with limit [%v]; exceeds: [%v]", car.plate, speedMph, road, obs1.mile, obs2.mile, limit, exceedsLimit)
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
				util.Log.Debug("ticket added to send queue: %+v", ticket)
				m.ticketsState[ticket] = false
			} else {
				util.Log.Debug("ticket already in send queue: %+v", ticket)
			}
		}
	}
	return m.sendPendingTickets()
}

func (m *SpeedMonitor) stop(client *Client) {
	util.Log.Trace("[client:%v] stopping", client.id)
	client.shutdown()
	delete(m.clients, client.id)
	util.Log.Info("[client:%v] stopped; sent %v bytes", client.id, client.bytesSent)

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
	util.Log.Info("all clients disconnected; tickets counts: [sent: %v; unsent: %v]", sentCount, len(unsent))
	if len(unsent) > 0 {
		for _, ticket := range unsent {
			util.Log.Info("unsent ticket ==> %+v", ticket)
		}
	}
}

func (m *SpeedMonitor) accept(client *Client, rawMessage interface{}) error {
	// obtain lock to ensure only one concurrent writer to state
	// TODO => could make this more granular to speed up handling, but messages
	// are small and inexpensive; forcing serial handling is probably fine
	m.stateMutex.Lock()
	defer m.stateMutex.Unlock()

	// track the client
	m.clients[client.id] = client

	util.Log.Debug("[client:%v] accepting message [%T]: %+v", client.id, rawMessage, rawMessage)
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
			util.Log.Debug("dispatcher connected for road: [%v]", road)
			client.dispatcher.roads[road] = true
			// check cars observed on roads; handles scenario where dispatcher
			// arrives and pending tickets exist for one their roads
			for _, car := range m.cars {
				_, ok := car.observationsByRoad[road]
				if ok {
					util.Log.Debug("[client:%v] checking plate [%v] for tickets on road: [%v]", client.id, car.plate, road)
					return m.checkForTickets(car, road)
				}
			}
		}

	case msgInPlate:
		if !client.isCamera() {
			util.Log.Error("[client:%v] received %T from non-Camera client", client.id, msg)
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
		util.Log.Debug("[client:%v] checking plate [%v] for tickets on road: [%v]", client.id, msg.plate, road)
		err := m.checkForTickets(car, road)
		if err != nil {
			return fmt.Errorf("[client:%v] checking plate [%v] for tickets on road: [%v] => %w", client.id, msg.plate, road, err)
		}
		return nil

	case msgInWantHeartbeat:
		if msg.interval == 0 {
			util.Log.Error("[client:%v] does not want heartbeat (interval = 0)", client.id)
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
		fmt.Printf("unknown message %v ", msg)
	}
	return nil
}
