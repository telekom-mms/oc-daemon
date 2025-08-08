// Package api contains the Unix Domain Socket API.
package api

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

const (
	// ServerShuttingDown is the error reply to client requests when the
	// API server is shutting down.
	ServerShuttingDown = "server is shutting down"
)

// SocketAPI is the Socket API interface.
type SocketAPI interface {
	Requests() chan *Request
	Shutdown()
	Start() error
	Stop()
}

var _ SocketAPI = &Server{}

// Server is a Daemon API server.
type Server struct {
	config   *daemoncfg.SocketServer
	listen   net.Listener
	requests chan *Request
	shutdown chan struct{}
	done     chan struct{}
	closed   chan struct{}
}

// isStopping returns whether the server is stopping.
func (s *Server) isStopping() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// sendShuttingDownError sends the "shutting down" error to the client and
// closes the connection.
func (s *Server) sendShuttingDownError(conn net.Conn) {
	// tell client server is not accepting connections anymore
	e := NewError([]byte(ServerShuttingDown))
	if err := WriteMessage(conn, e); err != nil {
		log.WithError(err).Error("Daemon message send error")
	}
	_ = conn.Close()
}

// handleRequest handles a request from the client.
func (s *Server) handleRequest(conn net.Conn) {
	// set timeout for entire request/response exchange
	deadline := time.Now().Add(s.config.RequestTimeout)
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
	case TypeVPNConfigUpdate:
	default:
		// send Error and disconnect
		e := NewError([]byte("invalid message"))
		if err := WriteMessage(conn, e); err != nil {
			log.WithError(err).Error("Daemon message send error")
		}
		_ = conn.Close()
		return
	}

	// forward client's request to daemon
	select {
	case s.requests <- &Request{
		msg:  msg,
		conn: conn,
	}:
	case <-s.shutdown:
		s.sendShuttingDownError(conn)
	case <-s.done:
		s.sendShuttingDownError(conn)
	}
}

// handleClients handles client connections.
func (s *Server) handleClients() {
	defer func() {
		_ = s.listen.Close()
		close(s.requests)
		close(s.closed)
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

// setSocketOwner sets the owner of the socket file.
func (s *Server) setSocketOwner() {
	if s.config.SocketOwner == "" {
		// do not change owner
		return
	}

	user, err := user.Lookup(s.config.SocketOwner)
	if err != nil {
		log.WithError(err).Error("Daemon could not get UID of socket file owner")
		return
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		log.WithError(err).Error("Daemon could not convert UID of socket file owner to int")
		return
	}

	if err := os.Chown(s.config.SocketFile, uid, -1); err != nil {
		log.WithError(err).Error("Daemon could not change owner of socket file")
	}
}

// setSocketGroup sets the group of the socket file.
func (s *Server) setSocketGroup() {
	if s.config.SocketGroup == "" {
		// do not change group
		return
	}

	group, err := user.LookupGroup(s.config.SocketGroup)
	if err != nil {
		log.WithError(err).Error("Daemon could not get GID of socket file group")
		return
	}

	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		log.WithError(err).Error("Daemon could not convert GID of socket file group to int")
		return
	}

	if err := os.Chown(s.config.SocketFile, -1, gid); err != nil {
		log.WithError(err).Error("Daemon could not change group of socket file")
	}
}

// setSocketPermissions sets the file permissions of the socket file.
func (s *Server) setSocketPermissions() {
	if s.config.SocketPermissions == "" {
		// do not change permissions
		return
	}

	perm, err := strconv.ParseUint(s.config.SocketPermissions, 8, 32)
	if err != nil {
		log.WithError(err).Error("Daemon could not convert permissions of sock file to uint")
		return
	}

	if err := os.Chmod(s.config.SocketFile, os.FileMode(perm)); err != nil {
		log.WithError(err).Error("Daemon could not set permissions of sock file")
	}

}

// Start starts the API server.
func (s *Server) Start() error {
	// cleanup existing sock file, this should normally fail
	if err := os.Remove(s.config.SocketFile); err == nil {
		log.Warn("Removed existing unix socket file")
	}

	// start listener
	listen, err := net.Listen("unix", s.config.SocketFile)
	if err != nil {
		return fmt.Errorf("could not start unix listener: %w", err)
	}
	s.listen = listen

	// set owner of sock file
	s.setSocketOwner()

	// set group of sock file
	s.setSocketGroup()

	// set file permissions of sock file
	s.setSocketPermissions()

	// handle client connections
	go s.handleClients()
	return nil
}

// Stop stops the API server.
func (s *Server) Stop() {
	close(s.done)

	// stop listener
	err := s.listen.Close()
	if err != nil {
		log.WithError(err).Error("Daemon could not close unix listener")
	}

	<-s.closed
}

// Shutdown stops accepting client requests in the API server. This is called
// before Stop to tell clients the server is shutting down. Client requests
// should not be read from the requests channel after calling this.
func (s *Server) Shutdown() {
	close(s.shutdown)
}

// Requests returns the clients channel.
func (s *Server) Requests() chan *Request {
	return s.requests
}

// NewServer returns a new API server.
func NewServer(config *daemoncfg.SocketServer) *Server {
	return &Server{
		config:   config,
		requests: make(chan *Request),
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
}
