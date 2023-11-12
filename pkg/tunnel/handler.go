package tunnel

import (
	"log"
	"net"
	"strings"
)

const (
	DefaultHost = "127.0.0.1:22"
)

// ConnectionHandler represents a connection handler
type ConnectionHandler struct {
	client       net.Conn
	clientClosed bool
	target       net.Conn
	targetClosed bool
	clientBuffer []byte
	server       *Server
	log          string
}

// NewConnectionHandler creates a new connection handler
func NewConnectionHandler(client net.Conn, server *Server) *ConnectionHandler {
	return &ConnectionHandler{
		client: client,
		server: server,
	}
}

// Close closes the connection
func (c *ConnectionHandler) Close() {
	if !c.clientClosed {
		c.client.Close()
		c.clientClosed = true
	}

	if !c.targetClosed {
		c.target.Close()
		c.targetClosed = true
	}
}

// Run runs the connection handler
func (c *ConnectionHandler) Run() {
	defer func() {
		c.Close()
		c.server.RemoveConnection(c)
	}()

	c.clientBuffer = make([]byte, BufLen)
	_, err := c.client.Read(c.clientBuffer)
	if err != nil {
		c.log += " - error: " + err.Error()
		c.server.PrintLog(c.log)
		return
	}

	hostPort := c.findHeader(c.clientBuffer, []byte("X-Real-Host"))

	if len(hostPort) == 0 {
		hostPort = []byte(DefaultHost)
	}

	c.connect(string(hostPort))
}

// FindHeader finds a header in the data
func (c *ConnectionHandler) findHeader(head []byte, header []byte) []byte {
	aux := strings.Index(string(head), string(header)+": ")

	if aux == -1 {
		return []byte{}
	}

	aux = strings.Index(string(head[aux:]), ":")
	head = []byte(string(head[aux+2:]))
	aux = strings.Index(string(head), "\r\n")

	if aux == -1 {
		return []byte{}
	}

	return []byte(string(head[:aux]))
}

// ConnectTarget connects to the target
func (c *ConnectionHandler) connectTarget(host string) {
	var (
		hostPort []string = strings.Split(host, ":")
		port     string
	)

	if len(hostPort) == 2 {
		host = hostPort[0]
		port = hostPort[1]
	}

	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port))
	if err != nil {
		log.Println("Error resolving address:", err.Error())
		return
	}

	c.target, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Println("Error connecting to target:", err.Error())
		return
	}

	c.targetClosed = false
}

// MethodCONNECT handles the CONNECT method
func (c *ConnectionHandler) connect(host string) {
	c.log += " - CONNECT " + host

	c.connectTarget(host)
	c.client.Write([]byte(Response))
	c.clientBuffer = nil

	c.server.PrintLog(c.log)
	c.startTunnel()
}

// DoCONNECT handles the data transfer between client and target
func (c *ConnectionHandler) startTunnel() {
	clientToTarget := make(chan []byte)
	targetToClient := make(chan []byte)

	// Goroutine to read from client connection and send to channel
	go func() {
		defer close(clientToTarget)
		for {
			data := make([]byte, BufLen)
			n, err := c.client.Read(data)
			if err != nil {
				return
			}
			clientToTarget <- data[:n]
		}
	}()

	// Goroutine to read from target connection and send to channel
	go func() {
		defer close(targetToClient)
		for {
			data := make([]byte, BufLen)
			n, err := c.target.Read(data)
			if err != nil {
				return
			}
			targetToClient <- data[:n]
		}
	}()

	// Main loop to write to client and target connections
	for {
		select {
		case data, ok := <-clientToTarget:
			if !ok {
				return
			}
			_, err := c.target.Write(data)
			if err != nil {
				return
			}

		case data, ok := <-targetToClient:
			if !ok {
				return
			}
			_, err := c.client.Write(data)
			if err != nil {
				return
			}
		}
	}
}
