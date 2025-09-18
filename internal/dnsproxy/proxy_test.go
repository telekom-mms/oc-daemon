package dnsproxy

import (
	"errors"
	"net"
	"net/netip"
	"reflect"
	"testing"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// getTestConfig returns a config for testing.
func getTestConfig() *daemoncfg.DNSProxy {
	return &daemoncfg.DNSProxy{
		Address:   "127.0.0.1:4254",
		ListenUDP: true,
		ListenTCP: true,
	}
}

// getTestDNSServer returns a dns server for testing.
func getTestDNSServer(t *testing.T, handler dns.Handler) *dns.Server {
	p, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	s := &dns.Server{
		Addr:       p.LocalAddr().String(),
		Net:        "udp",
		PacketConn: p,
		Handler:    handler,
	}
	go func() {
		if err := s.ActivateAndServe(); err != nil {
			panic(err)
		}
	}()
	return s
}

// responseWriter is a dns.ResponseWriter for testing.
type responseWriter struct{ err error }

func (r *responseWriter) LocalAddr() net.Addr       { return nil }
func (r *responseWriter) RemoteAddr() net.Addr      { return nil }
func (r *responseWriter) WriteMsg(*dns.Msg) error   { return r.err }
func (r *responseWriter) Write([]byte) (int, error) { return 0, nil }
func (r *responseWriter) Close() error              { return nil }
func (r *responseWriter) TsigStatus() error         { return nil }
func (r *responseWriter) TsigTimersOnly(bool)       {}
func (r *responseWriter) Hijack()                   {}

// TestProxyHandleRequest tests handleRequest of Proxy.
func TestProxyHandleRequest(t *testing.T) {
	p := NewProxy(getTestConfig())

	// without question in request
	p.handleRequest(nil, &dns.Msg{})

	// without remotes in proxy
	p.handleRequest(nil, &dns.Msg{Question: []dns.Question{{Name: "test"}}})

	// with invalid remote in proxy
	p.SetRemotes(map[string][]string{".": {""}})
	p.handleRequest(nil, &dns.Msg{Question: []dns.Question{{Name: "test"}}})

	// start remote
	s := getTestDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		reply := &dns.Msg{}
		reply.SetReply(r)

		dname, _ := dns.NewRR("test.example.com 3600 IN DNAME example.com.")
		cname, _ := dns.NewRR("test.example.com 3600 IN CNAME example.com.")
		a, _ := dns.NewRR("example.com. 3600 IN A 127.0.0.1")
		aaaa, _ := dns.NewRR("example.com. 3600 IN AAAA ::1")

		reply.Answer = []dns.RR{dname, cname, aaaa, a}
		if err := w.WriteMsg(reply); err != nil {
			log.WithError(err).Error("error sending reply")
		}
	}))
	defer func() { _ = s.Shutdown() }()

	// with remotes in proxy
	p.SetRemotes(map[string][]string{".": {s.Addr}})
	p.handleRequest(&responseWriter{}, &dns.Msg{Question: []dns.Question{{Name: "test.example.com."}}})

	// error sending reply
	p.handleRequest(&responseWriter{err: errors.New("test error")},
		&dns.Msg{Question: []dns.Question{{Name: "test.example.com."}}})

	// with watches in proxy
	p.SetWatches([]string{"test.example.com."})

	// watches should contain test.example.com but not example.com
	if !p.watches.Contains("test.example.com.") {
		t.Error("watches should contain test.example.com")
	}
	if p.watches.Contains("example.com.") {
		t.Error("watches should not contain example")
	}

	// handle request and save reports in separate goroutine
	reports := []*Report{}
	reportsDone := make(chan struct{})
	go func() {
		defer close(reportsDone)
		for r := range p.Reports() {
			reports = append(reports, r)
			r.Close()
		}
	}()
	p.handleRequest(&responseWriter{}, &dns.Msg{Question: []dns.Question{{Name: "test.example.com."}}})
	close(p.reports)
	<-reportsDone

	// watches should now contain both test.example.com and example.com
	if !p.watches.Contains("test.example.com.") {
		t.Error("watches should contain test.example.com")
	}
	if !p.watches.Contains("example.com.") {
		t.Error("watches should contain example")
	}

	// reports should contain the IPv4 and the IPv6 address of example.com
	wantIPv4 := netip.MustParseAddr("127.0.0.1")
	wantIPv6 := netip.MustParseAddr("::1")
	for _, r := range reports {
		if r.Name != "example.com." {
			t.Errorf("invalid domain name: %s", r.Name)
		}
		if r.IP != wantIPv4 && r.IP != wantIPv6 {
			t.Errorf("invalid IP: %s", r.IP)
		}

		// make sure IPv4 address is not IPv4-mapped IPv6 address
		if r.IP == wantIPv4 && r.IP.Is4In6() {
			t.Errorf("address is IPv4-mapped IPv6 address: %s", r.IP)
		}
	}
}

// TestProxyHandleRequest tests handleRequest of Proxy, DNS records.
// This tests different CNAME, DNAME, A, AAAA combinations.
func TestProxyHandleRequestRecords(t *testing.T) {
	// dns records
	dname, _ := dns.NewRR("test.example.com 3600 IN DNAME example.com.")
	cname, _ := dns.NewRR("test.example.com 3600 IN CNAME example.com.")
	a, _ := dns.NewRR("example.com. 3600 IN A 127.0.0.1")
	aaaa, _ := dns.NewRR("example.com. 3600 IN AAAA ::1")

	// answers to test with CNAME, DNAME, A, AAAA combinations
	answers := [][]dns.RR{
		{cname, a, aaaa},
		{aaaa, a, cname},
		{dname, a, aaaa},
		{aaaa, a, dname},
		{dname, cname, aaaa, a},
		{cname, aaaa, a, dname},
		{aaaa, a, dname, cname},
		{aaaa, a, cname, dname},
	}

	// start test server that returns answers
	answersChan := make(chan []dns.RR, len(answers))
	for _, a := range answers {
		answersChan <- a
	}
	s := getTestDNSServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		reply := &dns.Msg{}
		reply.SetReply(r)

		reply.Answer = <-answersChan
		if err := w.WriteMsg(reply); err != nil {
			log.WithError(err).Error("error sending reply")
		}
	}))
	defer func() { _ = s.Shutdown() }()

	// test helper function
	handle := func() []*Report {
		// start proxy with remotes and watches
		p := NewProxy(getTestConfig())
		p.SetRemotes(map[string][]string{".": {s.Addr}})
		p.SetWatches([]string{"test.example.com."})

		// collect reports
		reports := []*Report{}
		reportsDone := make(chan struct{})
		go func() {
			defer close(reportsDone)
			for r := range p.Reports() {
				reports = append(reports, r)
				r.Close()
			}
		}()

		// handle request and return reports
		p.handleRequest(&responseWriter{}, &dns.Msg{Question: []dns.Question{
			{Name: "test.example.com."}}})
		close(p.reports)
		<-reportsDone

		return reports
	}

	// test CNAME, DNAME, A, AAAA combinations in answers
	for i := range answers {
		reports := handle()
		if len(reports) != 2 {
			t.Fatalf("invalid reports for run %d: %v", i, reports)
		}
		for _, r := range reports {
			if r.IP != netip.MustParseAddr("127.0.0.1") &&
				r.IP != netip.MustParseAddr("::1") {

				t.Errorf("invalid report for run %d: %v", i, r)
			}
		}

	}
}

// TestProxyStartStop tests Start and Stop of Proxy.
func TestProxyStartStop(_ *testing.T) {
	// tcp and udp listeners
	p := NewProxy(getTestConfig())
	p.Start()
	p.Stop()
	<-p.Reports()

	// no listeners
	c := getTestConfig()
	c.ListenUDP = false
	c.ListenTCP = false
	p = NewProxy(c)
	p.Start()
	p.Stop()
	<-p.Reports()

	// invalid listener address
	c = getTestConfig()
	c.Address = "invalid address"
	p = NewProxy(c)
	p.Start()
	p.Stop()
	<-p.Reports()

}

// TestProxyReports tests Reports of Proxy.
func TestProxyReports(t *testing.T) {
	p := NewProxy(getTestConfig())
	want := p.reports
	got := p.Reports()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestProxySetRemotes tests SetRemotes of Proxy.
func TestProxySetRemotes(_ *testing.T) {
	p := NewProxy(getTestConfig())
	remotes := getTestRemotes()
	p.SetRemotes(remotes)
}

// TestProxySetWatches tests SetWatches of Proxy.
func TestProxySetWatches(_ *testing.T) {
	config := &daemoncfg.DNSProxy{
		Address:   "127.0.0.1:4254",
		ListenUDP: true,
		ListenTCP: true,
	}
	p := NewProxy(config)
	watches := []string{"example.com."}
	p.SetWatches(watches)
}

// TestProxyGetState tests GetState of Proxy.
func TestProxyGetState(t *testing.T) {
	p := NewProxy(getTestConfig())

	// set remotes
	getRemotes := func() map[string][]string {
		return map[string][]string{".": {"192.168.1.1"}}
	}
	p.SetRemotes(getRemotes())

	// set watches
	p.watches.Add("example.com.")
	p.watches.AddTempCNAME("cname.example.com.", 300)
	p.watches.AddTempDNAME("dname.example.com.", 300)

	// check state
	want := &State{
		Remotes:     getRemotes(),
		Watches:     []string{"example.com."},
		TempWatches: []string{"cname.example.com.", "dname.example.com."},
	}
	got := p.GetState()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewProxy tests NewProxy.
func TestNewProxy(t *testing.T) {
	p := NewProxy(getTestConfig())
	if p.config == nil ||
		p.udp == nil ||
		p.tcp == nil ||
		p.remotes == nil ||
		p.watches == nil ||
		p.reports == nil ||
		p.done == nil ||
		p.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
