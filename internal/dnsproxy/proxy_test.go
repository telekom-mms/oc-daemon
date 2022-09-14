package dnsproxy

import (
	"testing"
)

// TestProxyStartStop tests Start and Stop of Proxy
func TestProxyStartStop(t *testing.T) {
	p := NewProxy("127.0.0.1:4254")
	p.Start()
	p.Stop()
	<-p.Reports()
}

// TestProxyReports tests Reports of Proxy
func TestProxyReports(t *testing.T) {
	p := NewProxy("127.0.0.1:4254")
	want := p.reports
	got := p.Reports()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestProxySetRemotes tests SetRemotes of Proxy
func TestProxySetRemotes(t *testing.T) {
	p := NewProxy("127.0.0.1:4254")
	remotes := getTestRemotes()
	p.SetRemotes(remotes)
}

// TestProxySetWatches tests SetWatches of Proxy
func TestProxySetWatches(t *testing.T) {
	p := NewProxy("127.0.0.1:4254")
	watches := []string{"example.com."}
	p.SetWatches(watches)
}

// TestNewProxy tests NewProxy
func TestNewProxy(t *testing.T) {
	p := NewProxy("127.0.0.1:4254")
	if p.server == nil ||
		p.remotes == nil ||
		p.watches == nil ||
		p.reports == nil ||
		p.stopClean == nil ||
		p.doneClean == nil {

		t.Errorf("got nil, want != nil")
	}
}
