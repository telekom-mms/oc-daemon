package dnsmon

import (
	"testing"
)

// TestDNSMonStartStop tests Start and Stop of DNSMon
func TestDNSMonStartStop(t *testing.T) {
	dnsMon := NewDNSMon(NewConfig())
	dnsMon.Start()
	dnsMon.Stop()
}

// TestDNSMonUpdates tests Updates of DNSMon
func TestDNSMonUpdates(t *testing.T) {
	dnsMon := NewDNSMon(NewConfig())
	got := dnsMon.Updates()
	want := dnsMon.updates
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewDNSMon tests NewDNSMon
func TestNewDNSMon(t *testing.T) {
	config := NewConfig()
	dnsMon := NewDNSMon(config)
	if dnsMon.config != config {
		t.Errorf("got %v, want %v", dnsMon.config, config)
	}
	if dnsMon.updates == nil ||
		dnsMon.done == nil {

		t.Errorf("got nil, want != nil")
	}
}
