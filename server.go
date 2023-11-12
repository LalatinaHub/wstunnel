package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

const (
	BufLen   = 4096 * 4
	Timeout  = 60
	Response = "HTTP/1.1 101 Switching Protocols\r\n\r\n"
)

// Server represents the server
type Server struct {
	running     bool
	host        string
	port        int
	connections []*ConnectionHandler
	connMutex   sync.Mutex
}

// Run starts the server
func (s *Server) Run() {
	addr := net.JoinHostPort(s.host, strconv.Itoa(s.port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("Error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()

	s.running = true

	for s.running {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		connHandler := NewConnectionHandler(conn, s)
		go connHandler.Run()
		s.AddConnection(connHandler)
	}

	s.running = false
}

// PrintLog prints the log
func (s *Server) PrintLog(text string) {
	log.Println(text)
}

// AddConnection adds a connection to the server
func (s *Server) AddConnection(conn *ConnectionHandler) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	if s.running {
		s.connections = append(s.connections, conn)
	}
}

// RemoveConnection removes a connection from the server
func (s *Server) RemoveConnection(conn *ConnectionHandler) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	for i, c := range s.connections {
		if c == conn {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
			break
		}
	}
}

// Close stops the server
func (s *Server) Close() {
	s.running = false
	s.connMutex.Lock()

	connections := make([]*ConnectionHandler, len(s.connections))
	copy(connections, s.connections)

	s.connMutex.Unlock()

	for _, c := range connections {
		c.Close()
	}
}
