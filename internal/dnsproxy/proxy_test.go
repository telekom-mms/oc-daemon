package dnsproxy

import (
	"testing"
)

// getTestConfig returns a config for testing
func getTestConfig() *Config {
	return &Config{
		Address:   "127.0.0.1:4254",
		ListenUDP: true,
		ListenTCP: true,
	}
}

// TestProxyStartStop tests Start and Stop of Proxy
func TestProxyStartStop(t *testing.T) {
	p := NewProxy(getTestConfig())
	p.Start()
	p.Stop()
	<-p.Reports()
}

// TestProxyReports tests Reports of Proxy
func TestProxyReports(t *testing.T) {
	p := NewProxy(getTestConfig())
	want := p.reports
	got := p.Reports()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestProxySetRemotes tests SetRemotes of Proxy
func TestProxySetRemotes(t *testing.T) {
	p := NewProxy(getTestConfig())
	remotes := getTestRemotes()
	p.SetRemotes(remotes)
}

// TestProxySetWatches tests SetWatches of Proxy
func TestProxySetWatches(t *testing.T) {
	config := &Config{
		Address:   "127.0.0.1:4254",
		ListenUDP: true,
		ListenTCP: true,
	}
	p := NewProxy(config)
	watches := []string{"example.com."}
	p.SetWatches(watches)
}

// TestNewProxy tests NewProxy
func TestNewProxy(t *testing.T) {
	p := NewProxy(getTestConfig())
	if p.config == nil ||
		p.udp == nil ||
		p.tcp == nil ||
		p.remotes == nil ||
		p.watches == nil ||
		p.reports == nil ||
		p.done == nil ||
		p.stopClean == nil ||
		p.doneClean == nil {

		t.Errorf("got nil, want != nil")
	}
}
