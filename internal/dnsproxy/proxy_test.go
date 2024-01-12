package dnsproxy

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

// getTestConfig returns a config for testing
func getTestConfig() *Config {
	return &Config{
		Address:   "127.0.0.1:4254",
		ListenUDP: true,
		ListenTCP: true,
	}
}

// TODO: use dns client to query proxy and test handleRequest instead?
type responseWriter struct{}

func (r *responseWriter) LocalAddr() net.Addr       { return nil }
func (r *responseWriter) RemoteAddr() net.Addr      { return nil }
func (r *responseWriter) WriteMsg(*dns.Msg) error   { return nil }
func (r *responseWriter) Write([]byte) (int, error) { return 0, nil }
func (r *responseWriter) Close() error              { return nil }
func (r *responseWriter) TsigStatus() error         { return nil }
func (r *responseWriter) TsigTimersOnly(bool)       {}
func (r *responseWriter) Hijack()                   {}

func getTestDNSServer(handler dns.Handler) *dns.Server {
	s := &dns.Server{
		Addr: "127.0.0.1:4255",
		Net:  "udp",
	}
	s.Handler = handler
	go s.ListenAndServe()
	return s
}

func TestHandleRequest(t *testing.T) {
	p := NewProxy(getTestConfig())

	// without question in request
	p.handleRequest(nil, &dns.Msg{})

	// without remotes in proxy
	p.handleRequest(nil, &dns.Msg{Question: []dns.Question{{Name: "test"}}})

	// start remote
	s := getTestDNSServer(dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		reply := &dns.Msg{}
		reply.SetReply(r)
		cname, _ := dns.NewRR("test.example.com 3600 in CNAME example.com.")
		a, _ := dns.NewRR("example.com. 3600 IN A 127.0.0.1")
		aaaa, _ := dns.NewRR("example.com. 3600 IN AAAA ::1")
		reply.Answer = []dns.RR{cname, aaaa, a}
		if err := w.WriteMsg(reply); err != nil {
			log.WithError(err).Error("error sending reply")
		}
	}))
	defer s.Shutdown()
	time.Sleep(time.Second)

	// with remotes in proxy
	p.SetRemotes(map[string][]string{".": {s.Addr}})
	p.handleRequest(&responseWriter{}, &dns.Msg{Question: []dns.Question{{Name: "test.example.com."}}})

	// with watches in proxy
	p.SetWatches([]string{"test.example.com."})
	go func() {
		for r := range p.Reports() {
			r.Done()
		}
	}()
	p.handleRequest(&responseWriter{}, &dns.Msg{Question: []dns.Question{{Name: "test.example.com."}}})
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
		p.closed == nil ||
		p.stopClean == nil ||
		p.doneClean == nil {

		t.Errorf("got nil, want != nil")
	}
}
