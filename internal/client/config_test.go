package client

import (
	"os"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

// TestClientConfigEmpty tests empty of ClientConfig
func TestClientConfigEmpty(t *testing.T) {
	// test empty
	c := newClientConfig()
	want := true
	got := c.empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test not empty
	c.User = "User"
	want = false
	got = c.empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test nil
	c = nil
	want = true
	got = c.empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestLoacClientConfig tests save of ClientConfig and loadClientConfig
func TestLoadClientConfig(t *testing.T) {
	// create test config
	want := &ClientConfig{
		ClientCertificate: "/some/cert",
		ClientKey:         "/some/key",
		CACertificate:     "/some/ca",
		VPNServer:         "server.example.com",
		User:              "user1",
		Password:          "passwd1",
	}

	// create temporary file
	f, err := os.CreateTemp("", "oc-client-config")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()

	// save config to temporary file
	want.save(f.Name())

	// load config from temporary file
	got := loadClientConfig(f.Name())

	// make sure configs are equal
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewClientConfig tests newClientConfig
func TestNewClientConfig(t *testing.T) {
	c := newClientConfig()
	if c == nil {
		t.Errorf("got nil, want != nil")
	}
}
