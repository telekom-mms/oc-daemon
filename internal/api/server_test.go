package api

import "testing"

// TestServerStartStop tests Start and Stop of Server
func TestServerStartStop(t *testing.T) {
	server := NewServer("test.sock")
	server.Start()
	server.Stop()
}

// TestServerRequests tests Requests of Server
func TestServerRequests(t *testing.T) {
	server := NewServer("test.sock")
	if server.Requests() != server.requests {
		t.Errorf("got %p, want %p", server.Requests(), server.requests)
	}
}

// TestNewServer tests NewServer
func TestNewServer(t *testing.T) {
	sockFile := "test.sock"
	server := NewServer(sockFile)
	if server.sockFile != sockFile {
		t.Errorf("got %s, want %s", server.sockFile, sockFile)
	}
	if server.requests == nil {
		t.Errorf("got nil, want != nil")
	}
}
