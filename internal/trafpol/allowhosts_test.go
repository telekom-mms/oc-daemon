package trafpol

import "testing"

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
func TestAllowHostsStartStop(t *testing.T) {
	config := NewConfig()
	a := NewAllowHosts(config)
	a.Start()
	a.Stop()
}

// TestAllowHostsUpdate tests Update of AllowHosts
func TestAllowHostsUpdate(t *testing.T) {
	config := NewConfig()
	a := NewAllowHosts(config)
	host := "example.com"
	a.Add(host)
	a.Start()
	a.Update()
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
