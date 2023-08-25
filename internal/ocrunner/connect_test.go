package ocrunner

import (
	"reflect"
	"testing"
)

// TestConnectStartStop tests Start and Stop of Connect
func TestConnectStartStop(t *testing.T) {
	c := NewConnect(NewConfig())
	c.Start()
	c.Stop()
}

// TestConnectDisconnect tests Disconnect of Connect
func TestConnectDisconnect(t *testing.T) {
	c := NewConnect(NewConfig())
	c.Start()
	c.Disconnect()
	c.Stop()
}

// TestConnectEvents tests Events of Connect
func TestConnectEvents(t *testing.T) {
	c := NewConnect(NewConfig())

	want := c.events
	got := c.Events()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewConnect tests NewConnect
func TestNewConnect(t *testing.T) {
	config := NewConfig()
	config.XMLProfile = "/some/profile/file"
	config.VPNCScript = "/some/vpnc/script"
	config.VPNDevice = "tun999"
	c := NewConnect(config)
	if !reflect.DeepEqual(c.config, config) {
		t.Errorf("got %v, want %v", c.config, config)
	}
	if c.exits == nil ||
		c.commands == nil ||
		c.done == nil ||
		c.events == nil {

		t.Errorf("got nil, want != nil")
	}
}
