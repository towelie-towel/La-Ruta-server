package game

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Socket interface {
	GetOutBound() chan<- MessageEnvelope
	GetInBound() <-chan MessageEnvelope
	Close() error
	IsClosed() bool
	WGOutbound() *sync.WaitGroup
	GetClosed() <-chan bool
}

type SocketImpl struct {
	outBound   chan<- MessageEnvelope
	inBound    <-chan MessageEnvelope
	conn       *websocket.Conn
	closed     bool
	outboundWG sync.WaitGroup
	closedCh   chan bool // Channel to notify when socket is closed
}

func (s *SocketImpl) IsClosed() bool {
	return s.closed
}

func (s *SocketImpl) GetOutBound() chan<- MessageEnvelope {
	// THIS IS GROSS... I don't like the fact I am doing this...
	s.outboundWG.Add(1)
	return s.outBound
}

func (s *SocketImpl) GetInBound() <-chan MessageEnvelope {
	return s.inBound
}

func (s *SocketImpl) WGOutbound() *sync.WaitGroup {
	return &s.outboundWG
}

func (s *SocketImpl) GetClosed() <-chan bool {
	return s.closedCh
}

func (s *SocketImpl) Close() error {
	s.outboundWG.Wait()
	err := s.conn.Close()
	s.closed = true
	s.closedCh <- true // Notify that the socket is closed
	close(s.closedCh)  // Close the closed channel
	return err
}

func NewSocket(w http.ResponseWriter, r *http.Request) (Socket, error) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	// from me to network
	out := make(chan MessageEnvelope, 1)

	// from network to me
	in := make(chan MessageEnvelope, 1) // other type

	socket := SocketImpl{
		out,
		in,
		c,
		false,
		sync.WaitGroup{},
		make(chan bool),
	}

	/* socket := SocketImpl{
		out:       out,
		in:        in,
		conn:      c,
		closed:    false,
		outboundWG: sync.WaitGroup{},
		closedCh:   make(chan bool),
	} */

	go func() {
		defer func() {
			c.Close()
			socket.closed = true
			socket.closedCh <- true // Notify that the socket is closed
			close(socket.closedCh)  // Close the closed channel
		}()

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				break
			}

			if mt != websocket.TextMessage {
				continue
			}

			in <- FromSocket(message)
		}
	}()

	go func() {
		for msg := range out {
			msg, err := json.Marshal(msg.Message)
			if err != nil {
				log.Fatalf("%+v\n", err)
			}

			c.WriteMessage(websocket.TextMessage, msg)
			socket.outboundWG.Done()
		}
	}()

	return &socket, nil
}
