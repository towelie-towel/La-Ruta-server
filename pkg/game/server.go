package game

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Server struct {
	Out <-chan [2]Socket

	out          chan [2]Socket
	other_socket Socket
	mutex        sync.Mutex
}

var upgrader = websocket.Upgrader{} // use default options

func NewServer() (*Server, error) {
	out := make(chan [2]Socket, 9)
	server := Server{
		Out:          out,
		out:          out,
		other_socket: nil,
	}

	return &server, nil
}

func (tokio *Server) HandleNewConnection(w http.ResponseWriter, r *http.Request) {

	socket, err := NewSocket(w, r)

	if err != nil {
		log.Fatalln("couldn't upgrade socket.", err)
		return
	}

	tokio.mutex.Lock()
	defer tokio.mutex.Unlock()

	if tokio.other_socket != nil {
		tokio.out <- [2]Socket{tokio.other_socket, socket}
		tokio.other_socket = nil
	} else {
		tokio.other_socket = socket
	}
}

func (tokio *Server) HandleNewReconnection(w http.ResponseWriter, r *http.Request) {
	socket, err := NewSocket(w, r)
	if err != nil {
		log.Fatalln("couldn't upgrade socket.", err)
		return
	}

	tokio.mutex.Lock()
	defer tokio.mutex.Unlock()

	if tokio.other_socket != nil {
		// If there is already another socket, close it before establishing a new connection
		tokio.other_socket.Close()
	}

	tokio.other_socket = socket

	// Create a goroutine to handle the connection
	go tokio.handleConnection(socket)
}

func (s *Server) handleConnection(socket Socket) {
	for {
		select {
		case <-socket.GetInBound():
			// Handle incoming messages from the client
			// ...
		case <-socket.GetClosed():
			// Handle disconnection event
			// ...
			return
		}
	}
}
