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
	id         uint16
	conn       *io.ReadWriteCloser
	bytesSent  int64
	dispatcher *Dispatcher
	camera     *Camera
	heartbeat  *time.Ticker
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
	if c.heartbeat != nil {
		return fmt.Errorf("[client:%v] received duplicate heartbeat request", c.id)
	}

	// deciseconds (10 per second; e.g. 25 = 2.5 seconds)
	intervalDuration := time.Duration(intervalDeciSeconds*100) * time.Millisecond
	ticker := time.NewTicker(intervalDuration)
	go func() {
		for {
			select {
			case currentTime := <-ticker.C:
				util.Log.Info("[client:%v] sending heartbeat message [at %v]", c.id, currentTime)
				if err := c.send(msgOutHeartbeat{mType: msgType_oHeartbeat}); err != nil {
					util.Log.Error("[client:%v] failed to send heartbeat message [at %v] => %v", c.id, err.Error())
					ticker.Stop()
					c.disconnect()
				}
			}
		}
	}()

	util.Log.Info("[client:%v] started heartbeat beckground task (interval: %v)", c.id, intervalDuration)
	c.heartbeat = ticker
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
	if c.heartbeat != nil {
		util.Log.Info("[client:%v] stopped heartbeat ticker", c.id)
		c.heartbeat.Stop()
	}
	util.Log.Info("[client:%v] shutdown completed", c.id)
}

type Dispatcher struct {
	// track all roads with an O(1) lookup
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
}

func (c *Car) observe(road uint16, mile uint16, timestamp uint32) {
	observations, ok := c.observationsByRoad[road]
	if !ok {
		c.observationsByRoad[road] = []Observation{}
	}
	observations = append(observations, Observation{mile: mile, timestamp: timestamp})
	// sort by mile (position)
	sort.Slice(observations, func(i, j int) bool { return observations[i].mile < observations[j].mile })
	c.observationsByRoad[road] = observations // necessary ?
}

type SpeedMonitor struct {
	cars              map[string]Car
	limits            map[uint16]uint16
	clients           map[uint16]*Client
	dispatchersByRoad map[uint16][]*Client

	ticketsState map[msgOutTicket]bool
	ticketMutex  sync.Mutex
}

const (
	hourSeconds = float64(60 * 60)
)

func computeSpeedMph(obs1 Observation, obs2 Observation) float64 {
	distance := math.Abs(float64(obs2.mile) - float64(obs1.mile))
	timeDiffSeconds := math.Abs(float64(obs2.timestamp - obs1.timestamp))
	timeDiffHours := timeDiffSeconds / hourSeconds
	speedMph := distance / timeDiffHours
	util.Log.Info("computeSpeedMph: saw car cover %vmi in %vs (%.4fh) => speed: %v", distance, timeDiffSeconds, timeDiffHours, speedMph)
	return math.Round(speedMph)
}

func (m *SpeedMonitor) findDispatcher(road uint16) *Client {
	dispatchers, ok := m.dispatchersByRoad[road]
	if !ok {
		return nil
	}
	// pick a random dispatcher
	return dispatchers[rand.Intn(len(dispatchers))]
}

func (m *SpeedMonitor) sendPendingTickets() error {
	for ticket, sent := range m.ticketsState {
		if sent {
			continue
		}
		// find dispatcher
		dispatcher := m.findDispatcher(ticket.road)
		if dispatcher != nil {
			err := dispatcher.send(ticket)
			if err != nil {
				return fmt.Errorf("[client:%v] sending ticket: [%+v] => %w", dispatcher.id, ticket, err)
			}
			m.ticketsState[ticket] = true
		}
	}
	return nil
}

func (m *SpeedMonitor) checkForTickets(car Car, road uint16) error {
	observations := car.observationsByRoad[road]
	if len(observations) < 2 {
		return nil
	}
	m.ticketMutex.Lock()
	defer m.ticketMutex.Unlock()
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
		util.Log.Debug("plate [%v] was going [%vmph] on road: [%v]", car.plate, speedMph, road)
		if speedMph > float64(limit) {
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
				m.ticketsState[ticket] = false
			}
		}
	}
	return m.sendPendingTickets()
}

func (m *SpeedMonitor) stop(client *Client) {
	util.Log.Trace("[client:%v] stopping", client.id)
	client.shutdown()
	delete(m.clients, client.id)
	util.Log.Trace("[client:%v] stopped", client.id)
}

func (m *SpeedMonitor) accept(client *Client, rawMessage interface{}) error {
	util.Log.Debug("[client:%v] accepting message [%T]: %+v", client.id, rawMessage, rawMessage)

	switch msg := rawMessage.(type) {
	case msgInAmCamera:
		if !client.isNew() {
			// this client has already declared its type
			util.Log.Error("[client:%v] received %T from not-new client", client.id, msg)
			break
		}
		// this client is a camera
		client.camera = &Camera{road: msg.road, mile: msg.mile, limit: msg.limit}
		m.limits[msg.road] = msg.limit

	case msgInAmDispatcher:
		if !client.isNew() {
			// this client has already declared its type
			util.Log.Error("[client:%v] received %T from not-new client", client.id, msg)
			break
		}
		// this client is a dispatcher
		client.dispatcher = &Dispatcher{roads: map[uint16]bool{}}
		for _, road := range msg.roads {
			// add all roads
			client.dispatcher.roads[road] = true
			// track dispatchers for road
			if _, ok := m.dispatchersByRoad[road]; !ok {
				m.dispatchersByRoad[road] = []*Client{}
			}
			m.dispatchersByRoad[road] = append(m.dispatchersByRoad[road], client)
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
			car = Car{plate: msg.plate, observationsByRoad: map[uint16][]Observation{}}
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
