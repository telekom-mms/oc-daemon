package api

import (
	"net"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// serverTimeout is the timeout for an entire request/response exchange
	// initiated by a client
	serverTimeout = 30 * time.Second
)

// Server is a Daemon API server
type Server struct {
	sockFile string
	listen   net.Listener
	requests chan *Request

	mutex sync.Mutex
	stop  bool
}

// setStopping marks the server as stopping
func (s *Server) setStopping() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.stop = true
}

// isStopping returns whether the server is stopping
func (s *Server) isStopping() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.stop

}

// handleRequest handles a request from the client
func (s *Server) handleRequest(conn net.Conn) {
	// set timeout for entire request/response exchange
	deadline := time.Now().Add(serverTimeout)
	if err := conn.SetDeadline(deadline); err != nil {
		log.WithError(err).Error("Daemon error setting deadline")
		_ = conn.Close()
		return
	}

	// read message from client
	msg, err := ReadMessage(conn)
	if err != nil {
		log.WithError(err).Error("Daemon receive message error")
		_ = conn.Close()
		return
	}

	// check if its a known message type
	switch msg.Type {
	case TypeVPNConnect:
	case TypeVPNDisconnect:
	case TypeVPNQuery:
	case TypeVPNConfigUpdate:
	default:
		// send Error and disconnect
		e := NewError([]byte("invalid message"))
		if err := WriteMessage(conn, e); err != nil {
			log.WithError(err).Error("Daemon message send error")
		}
		_ = conn.Close()
	}

	// forward client's request to daemon
	s.requests <- &Request{
		msg:  msg,
		conn: conn,
	}
}

// handleClients handles client connections
func (s *Server) handleClients() {
	defer func() {
		_ = s.listen.Close()
		close(s.requests)
	}()
	for {
		// wait for new client connection
		conn, err := s.listen.Accept()
		if err != nil {
			if s.isStopping() {
				// ignore error when shutting down
				return
			}

			log.WithError(err).Error("Daemon listener error")
			return
		}

		// read request from client connection and handle it
		s.handleRequest(conn)
	}
}

// Start starts the API server
func (s *Server) Start() {
	// cleanup existing sock file, this should normally fail
	if err := os.Remove(s.sockFile); err == nil {
		log.Warn("Removed existing unix socket file")
	}

	// start listener
	listen, err := net.Listen("unix", s.sockFile)
	if err != nil {
		log.WithError(err).Fatal("Daemon could not start unix listener")
	}
	s.listen = listen

	// make sure everyone can access the sock file
	if err := os.Chmod(s.sockFile, 0777); err != nil {
		log.WithError(err).Error("Daemon could not set permissions of sock file")
	}

	// handle client connections
	go s.handleClients()
}

// Stop stops the API server
func (s *Server) Stop() {
	// stop listener
	s.setStopping()
	err := s.listen.Close()
	if err != nil {
		log.WithError(err).Fatal("Daemon could not close unix listener")
	}
	for range s.requests {
		// wait for clients channel close
	}
}

// Requests returns the clients channel
func (s *Server) Requests() chan *Request {
	return s.requests
}

// NewServer returns a new API server
func NewServer(sockFile string) *Server {
	return &Server{
		sockFile: sockFile,
		requests: make(chan *Request),
	}
}
