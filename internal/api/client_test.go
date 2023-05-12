package api

import (
	"log"
	"reflect"
	"testing"

	"github.com/T-Systems-MMS/oc-daemon/internal/ocrunner"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnstatus"
)

// initTestClientServer returns a client an server for testing;
// the server simply closes client requests
func initTestClientServer() (*Client, *Server) {
	server := NewServer("test.sock")
	client := NewClient(server.sockFile)
	go func() {
		for r := range server.requests {
			log.Println(r)
			r.Close()
		}
	}()
	return client, server
}

// TestClientRequest tests Request of Client
func TestClientRequest(t *testing.T) {
	client, server := initTestClientServer()
	server.Start()
	reply, _ := client.Request(NewMessage(TypeVPNQuery, nil))
	server.Stop()

	log.Println(reply)
}

// TestClientQuery tests Query of Client
func TestClientQuery(t *testing.T) {
	server := NewServer("test.sock")
	client := NewClient(server.sockFile)
	status := vpnstatus.New()
	go func() {
		for r := range server.requests {
			// handle query requests only,
			// reply with status
			log.Println(r)
			b, err := status.JSON()
			if err != nil {
				log.Fatal(err)
			}
			r.Reply(b)
			r.Close()
		}
	}()
	server.Start()
	want := status
	got, _ := client.Query()
	server.Stop()

	log.Println(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestClientConnect tests Connect of Client
func TestClientConnect(t *testing.T) {
	client, server := initTestClientServer()
	server.Start()
	client.Connect(&ocrunner.LoginInfo{})
	server.Stop()
}

// TestClientDisconnect tests Disconnect of Client
func TestClientDisconnect(t *testing.T) {
	client, server := initTestClientServer()
	server.Start()
	client.Disconnect()
	server.Stop()
}

// TestNewClient tests NewClient
func TestNewClient(t *testing.T) {
	sockFile := "test.sock"
	client := NewClient(sockFile)
	got := client.sockFile
	want := sockFile
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
