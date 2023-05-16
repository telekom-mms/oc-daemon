package client

import (
	"log"
	"reflect"
	"testing"

	"github.com/T-Systems-MMS/oc-daemon/internal/api"
	"github.com/T-Systems-MMS/oc-daemon/pkg/logininfo"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnstatus"
)

const (
	// testSockFile is a socket file for testing
	testSockFile = "test.sock"
)

// initTestClientServer returns a client an server for testing;
// the server simply closes client requests
func initTestClientServer() (*Client, *api.Server) {
	server := api.NewServer(testSockFile)
	client := NewClient(NewConfig())
	client.Config.SocketFile = testSockFile
	go func() {
		for r := range server.Requests() {
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
	reply, _ := client.Request(api.NewMessage(api.TypeVPNQuery, nil))
	server.Stop()

	log.Println(reply)
}

// TestClientQuery tests Query of Client
func TestClientQuery(t *testing.T) {
	server := api.NewServer(testSockFile)
	client := NewClient(NewConfig())
	client.Config.SocketFile = testSockFile
	status := vpnstatus.New()
	go func() {
		for r := range server.Requests() {
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

// TestClientConnect tests connect of Client
func TestClientConnect(t *testing.T) {
	client, server := initTestClientServer()
	server.Start()
	client.Login = &logininfo.LoginInfo{}
	err := client.connect()
	if err != nil {
		t.Error(err)
	}
	server.Stop()
}

// TestClientDisconnect tests disconnect of Client
func TestClientDisconnect(t *testing.T) {
	client, server := initTestClientServer()
	server.Start()
	err := client.disconnect()
	if err != nil {
		t.Error(err)
	}
	server.Stop()
}

// TestNewClient tests NewClient
func TestNewClient(t *testing.T) {
	config := NewConfig()
	client := NewClient(config)
	want := config
	got := client.Config
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}
