package trafpol

import (
	"testing"
	"time"
)

// TestAllowHostsAdd tests Add of AllowHosts
func TestAllowHostsAdd(t *testing.T) {
	config := NewConfig()
	a := NewAllowHosts(config)
	want := "example.com"
	a.Add(want)
	got := a.m[want].host
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestAllowHostsRemove tests Add of AllowHosts
func TestAllowHostsRemove(t *testing.T) {
	config := NewConfig()
	a := NewAllowHosts(config)
	host := "example.com"
	a.Add(host)
	a.Remove(host)
	if a.m[host] != nil {
		t.Errorf("got %p, want nil", a.m[host])
	}
}

// TestAllowHostsStartStop tests Start and Stop of AllowHosts
func TestAllowHostsStartStop(_ *testing.T) {
	config := NewConfig()
	a := NewAllowHosts(config)
	a.Start()
	a.Stop()
}

// TestAllowHostsUpdate tests Update of AllowHosts
func TestAllowHostsUpdate(_ *testing.T) {
	config := NewConfig()
	config.ResolveTriesSleep = 0
	config.ResolveTimer = 0
	a := NewAllowHosts(config)

	// test with domain name and ip net
	a.Add("example.com")
	a.Add("192.168.1.1/32")

	a.Start()

	// test double update
	a.Update()
	a.Update()

	time.Sleep(time.Second)
	a.Stop()
}

// TestNewAllowHosts tests NewAllowHosts
func TestNewAllowHosts(t *testing.T) {
	config := NewConfig()
	a := NewAllowHosts(config)
	if a.m == nil ||
		a.updates == nil ||
		a.done == nil ||
		a.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
