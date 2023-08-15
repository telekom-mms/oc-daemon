package client

import (
	"os"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

// TestConfigCopy tests Copy of Config
func TestConfigCopy(t *testing.T) {
	want := NewConfig()
	got := want.Copy()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

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

// TestConfigValid tests Valid of Config
func TestConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Config{
		nil,
		{},
	} {
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	valid := NewConfig()
	valid.ClientCertificate = "test-cert"
	valid.ClientKey = "test-key"
	valid.VPNServer = "test-server"
	want := true
	got := valid.Valid()

	if got != want {
		t.Errorf("got %t, want %t for %v", got, want, valid)
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
		XMLProfile:        "/some/profile",
		VPNServer:         "server.example.com",
		User:              "user1",

		Protocol:  "test",
		UserAgent: "agent",
		Quiet:     true,
		NoProxy:   true,
		ExtraEnv:  []string{"oc_daemon_var_is_not=used"},
		ExtraArgs: []string{"--arg-does-not=exist"},
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
