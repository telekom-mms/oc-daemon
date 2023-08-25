package api

import "testing"

// TestServerStartStop tests Start and Stop of Server
func TestServerStartStop(t *testing.T) {
	config := NewConfig()
	config.SocketFile = "test.sock"
	server := NewServer(config)
	server.Start()
	server.Stop()
}

// TestServerRequests tests Requests of Server
func TestServerRequests(t *testing.T) {
	config := NewConfig()
	config.SocketFile = "test.sock"
	server := NewServer(config)
	if server.Requests() != server.requests {
		t.Errorf("got %p, want %p", server.Requests(), server.requests)
	}
}

// TestNewServer tests NewServer
func TestNewServer(t *testing.T) {
	config := NewConfig()
	server := NewServer(config)
	if server.config != config {
		t.Errorf("got %p, want %p", server.config, config)
	}
	if server.requests == nil {
		t.Errorf("got nil, want != nil")
	}
}
