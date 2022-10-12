package dnsmon

import "testing"

// TestDNSMonStartStop tests Start and Stop of DNSMon
func TestDNSMonStartStop(t *testing.T) {
	dnsMon := NewDNSMon()
	dnsMon.Start()
	dnsMon.Stop()
}

// TestDNSMonUpdates tests Updates of DNSMon
func TestDNSMonUpdates(t *testing.T) {
	dnsMon := NewDNSMon()
	got := dnsMon.Updates()
	want := dnsMon.updates
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewDNSMon tests NewDNSMon
func TestNewDNSMon(t *testing.T) {
	dnsMon := NewDNSMon()
	if dnsMon.updates == nil ||
		dnsMon.done == nil {

		t.Errorf("got nil, want != nil")
	}
}
