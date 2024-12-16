package api

import (
	"net"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// TestServerHandleRequest tests handleRequest of Server.
func TestServerHandleRequest(t *testing.T) {
	config := daemoncfg.NewSocketServer()
	server := NewServer(config)

	// connection closed
	c1, c2 := net.Pipe()
	if err := c2.Close(); err != nil {
		t.Fatal(err)
	}

	server.handleRequest(c1)

	// invalid messages
	for _, msg := range []*Message{
		// invalid message type
		{Header: Header{Type: TypeUndefined}},

		// unhandled message type
		{Header: Header{Type: TypeError}},
	} {
		c1, c2 := net.Pipe()
		go server.handleRequest(c1)

		if err := WriteMessage(c2, msg); err != nil {
			t.Fatal(err)
		}
		if err := c2.Close(); err != nil {
			t.Fatal(err)
		}
	}

	// valid request
	c1, c2 = net.Pipe()
	go server.handleRequest(c1)

	msg := &Message{Header: Header{Type: TypeVPNConfigUpdate}}
	if err := WriteMessage(c2, msg); err != nil {
		t.Fatal(err)
	}

	req := <-server.requests
	if !reflect.DeepEqual(req.msg.Header, msg.Header) {
		t.Errorf("got %v, want %v", req.msg.Header, msg.Header)
	}
	if req.conn != c1 {
		t.Errorf("got %p, want %p", req.conn, c1)
	}

	// valid request during shutdown
	server.Shutdown()
	c1, c2 = net.Pipe()
	go server.handleRequest(c1)

	msg = &Message{Header: Header{Type: TypeVPNConfigUpdate}}
	if err := WriteMessage(c2, msg); err != nil {
		t.Fatal(err)
	}

	msg, err := ReadMessage(c2)
	if err != nil {
		t.Fatal(err)
	}

	if msg.Header.Type != TypeError || string(msg.Value) != ServerShuttingDown {
		t.Error("unexpected reply")
	}
}

// TestServerSetSocketOwner tests setSocketOwner of Server.
func TestServerSetSocketOwner(_ *testing.T) {
	config := daemoncfg.NewSocketServer()
	server := NewServer(config)

	// no changes
	server.setSocketOwner()

	// user does not exist
	server.config.SocketOwner = "does-not-exist"
	server.setSocketOwner()

	// socket file does not exist
	server.config.SocketOwner = "root"
	server.config.SocketFile = "does-not-exists"
	server.setSocketOwner()
}

// TestServerSetSocketGroup tests setSocketGroup of Server.
func TestServerSetSocketGroup(_ *testing.T) {
	config := daemoncfg.NewSocketServer()
	server := NewServer(config)

	// no changes
	server.setSocketGroup()

	// group does not exist
	server.config.SocketGroup = "does-not-exist"
	server.setSocketGroup()

	// socket file does not exist
	server.config.SocketGroup = "root"
	server.config.SocketFile = "does-not-exists"
	server.setSocketGroup()
}

// TestServerSetSocketPermissions tests setSocketPermissions of Server.
func TestServerSetSocketPermissions(_ *testing.T) {
	config := daemoncfg.NewSocketServer()
	server := NewServer(config)

	// socket file does not exist
	server.config.SocketFile = "does-not-exists"
	server.setSocketPermissions()

	// invalid permissions
	server.config.SocketPermissions = "invalid"
	server.setSocketPermissions()

	// no changes
	server.config.SocketPermissions = ""
	server.setSocketPermissions()
}

// TestServerStartStop tests Start and Stop of Server.
func TestServerStartStop(t *testing.T) {
	config := daemoncfg.NewSocketServer()
	config.SocketFile = "test.sock"
	server := NewServer(config)
	if err := server.Start(); err != nil {
		t.Error(err)
	}
	server.Shutdown()
	server.Stop()
}

// TestServerRequests tests Requests of Server.
func TestServerRequests(t *testing.T) {
	config := daemoncfg.NewSocketServer()
	config.SocketFile = "test.sock"
	server := NewServer(config)
	if server.Requests() != server.requests {
		t.Errorf("got %p, want %p", server.Requests(), server.requests)
	}
}

// TestNewServer tests NewServer.
func TestNewServer(t *testing.T) {
	config := daemoncfg.NewSocketServer()
	server := NewServer(config)

	if server == nil ||
		server.requests == nil ||
		server.shutdown == nil ||
		server.done == nil ||
		server.closed == nil {
		t.Errorf("got nil, want != nil")
	}
	if server.config != config {
		t.Errorf("got %p, want %p", server.config, config)
	}
}
