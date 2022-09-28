package ocrunner

import "testing"

// TestConnectStartStop tests Start and Stop of Connect
func TestConnectStartStop(t *testing.T) {
	c := NewConnect("", "", "")
	c.Start()
	c.Stop()
}

// TestConnectDisconnect tests Disconnect of Connect
func TestConnectDisconnect(t *testing.T) {
	c := NewConnect("", "", "")
	c.Start()
	c.Disconnect()
	c.Stop()
}

// TestConnectEvents tests Events of Connect
func TestConnectEvents(t *testing.T) {
	c := NewConnect("", "", "")

	want := c.events
	got := c.Events()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewConnect tests NewConnect
func TestNewConnect(t *testing.T) {
	profile := "/some/profile/file"
	script := "/some/vpnc/script"
	device := "tun999"
	c := NewConnect(profile, script, device)
	if c.profile != profile {
		t.Errorf("got %s, want %s", c.profile, profile)
	}
	if c.script != script {
		t.Errorf("got %s, want %s", c.script, script)
	}
	if c.device != device {
		t.Errorf("got %s, want %s", c.script, script)
	}
	if c.exits == nil ||
		c.commands == nil ||
		c.done == nil ||
		c.events == nil {

		t.Errorf("got nil, want != nil")
	}
}
