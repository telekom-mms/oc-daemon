package dnsproxy

import (
	"net"
	"testing"
)

// TestReportString tests String of Report
func TestReportString(t *testing.T) {
	name := "example.com."
	ip := net.IPv4(192, 168, 1, 1)
	ttl := uint32(300)
	r := NewReport(name, ip, ttl)

	want := "example.com. -> 192.168.1.1 (ttl: 300)"
	got := r.String()
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestReportDone tests Wait and Done of Report
func TestReportWaitDone(t *testing.T) {
	name := "example.com."
	ip := net.IPv4(192, 168, 1, 1)
	ttl := uint32(300)
	r := NewReport(name, ip, ttl)

	go r.Done()
	r.Wait()
}

// TestNewReport tests NewReport
func TestNewReport(t *testing.T) {
	name := "example.com."
	ip := net.IPv4(192, 168, 1, 1)
	ttl := uint32(300)
	r := NewReport(name, ip, ttl)

	if r.done == nil {
		t.Errorf("got nil, want != nil")
	}
	if r.Name != name {
		t.Errorf("got %s, want %s", r.Name, name)
	}
	if !r.IP.Equal(ip) {
		t.Errorf("got %s, want %s", r.IP, ip)
	}
	if r.TTL != ttl {
		t.Errorf("got %d, want %d", r.TTL, ttl)
	}
}
