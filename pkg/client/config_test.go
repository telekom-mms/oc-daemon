package client

import (
	"os"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

// TestConfigEmpty tests Empty of Config
func TestConfigEmpty(t *testing.T) {
	// test empty
	c := &Config{}
	want := true
	got := c.Empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test not empty
	c.User = "User"
	want = false
	got = c.Empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test nil
	c = nil
	want = true
	got = c.Empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestNewConfig tests NewConfig
func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if c.Empty() {
		t.Errorf("got empty, want not empty")
	}
}

// TestLoadConfig tests Save of Config and LoadConfig
func TestLoadConfig(t *testing.T) {
	// create test config
	want := &Config{
		ClientCertificate: "/some/cert",
		ClientKey:         "/some/key",
		CACertificate:     "/some/ca",
		VPNServer:         "server.example.com",
		User:              "user1",
		Password:          "passwd1",

		SocketFile:        SocketFile,
		ConnectionTimeout: ConnectionTimeout,
		RequestTimeout:    RequestTimeout,
	}

	// create temporary file
	f, err := os.CreateTemp("", "oc-client-config")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()

	// save config to temporary file
	if err := want.Save(f.Name()); err != nil {
		t.Error(err)
	}

	// load config from temporary file
	got, err := LoadConfig(f.Name())
	if err != nil {
		t.Error(err)
	}

	// make sure configs are equal
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
