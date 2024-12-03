// Package dnsproxy contains the DNS proxy.
package dnsproxy

import (
	"math/rand"
	"net/netip"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// State is the internal state of the DNS Proxy.
type State struct {
	Remotes     map[string][]string
	Watches     []string
	TempWatches []string
}

// Proxy is a DNS proxy.
type Proxy struct {
	config  *daemoncfg.DNSProxy
	udp     *dns.Server
	tcp     *dns.Server
	remotes *Remotes
	watches *Watches
	reports chan *Report
	done    chan struct{}
	closed  chan struct{}
}

// sendReport sends report over the reports channel.
func (p *Proxy) sendReport(report *Report) {
	select {
	case p.reports <- report:
	case <-p.done:
	}
}

// waitReport waits for report to finish.
func (p *Proxy) waitReport(report *Report) {
	select {
	case <-report.Done():
	case <-p.done:
	}
}

// handleRequest handles a dns client request.
func (p *Proxy) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	// make sure the client request is valid
	if len(r.Question) != 1 {
		// TODO: be less strict? send error reply to client?
		log.WithField("request", r).Error("DNS-Proxy received invalid client request")
		return
	}

	// forward request to remote server and get reply
	remotes := p.remotes.Get(r.Question[0].Name)
	if len(remotes) == 0 {
		log.WithField("name", r.Question[0].Name).
			Error("DNS-Proxy has no remotes for question name")
		// TODO: send error reply to client?
		return
	}
	// pick random remote server
	// TODO: query all servers and take fastest reply?
	remote := remotes[rand.Intn(len(remotes))]
	reply, err := dns.Exchange(r, remote)
	if err != nil {
		log.WithError(err).Debug("DNS-Proxy DNS exchange error")
		return
	}

	// parse answers in reply from remote server
	// handler for DNAME answers
	handleDNAME := func(a dns.RR) {
		// DNAME record, store temporary watch
		rr, ok := a.(*dns.DNAME)
		if !ok {
			log.Error("DNS-Proxy received invalid DNAME record in reply")
			return
		}
		log.WithFields(log.Fields{
			"target": rr.Target,
			"ttl":    rr.Hdr.Ttl,
		}).Debug("DNS-Proxy received DNAME in reply")
		p.watches.AddTempDNAME(rr.Target, rr.Hdr.Ttl)
	}

	// handler for CNAME answers
	handleCNAME := func(a dns.RR) {
		// CNAME record, store temporary watch
		rr, ok := a.(*dns.CNAME)
		if !ok {
			log.Error("DNS-Proxy received invalid CNAME record in reply")
			return
		}
		log.WithFields(log.Fields{
			"target": rr.Target,
			"ttl":    rr.Hdr.Ttl,
		}).Debug("DNS-Proxy received CNAME in reply")
		p.watches.AddTempCNAME(rr.Target, rr.Hdr.Ttl)
	}

	// handler for A answers
	handleA := func(a dns.RR) {
		// A Record, get IPv4 address
		rr, ok := a.(*dns.A)
		if !ok {
			log.Error("DNS-Proxy received invalid A record in reply")
			return
		}
		addr, ok := netip.AddrFromSlice(rr.A)
		if !ok {
			log.WithField("A", rr.A).
				Error("DNS-Proxy received invalid IP in A record in reply")
			return
		}
		report := NewReport(rr.Hdr.Name, addr, rr.Hdr.Ttl)
		p.sendReport(report)
		p.waitReport(report)
	}

	// handler for AAAA answers
	handleAAAA := func(a dns.RR) {
		// AAAA Record, get IPv6 address
		rr, ok := a.(*dns.AAAA)
		if !ok {
			log.Error("DNS-Proxy received invalid AAAA record in reply")
			return
		}
		addr, ok := netip.AddrFromSlice(rr.AAAA)
		if !ok {
			log.WithField("AAAA", rr.AAAA).
				Error("DNS-Proxy received invalid IP in AAAA record in reply")
			return
		}
		report := NewReport(rr.Hdr.Name, addr, rr.Hdr.Ttl)
		p.sendReport(report)
		p.waitReport(report)
	}

	// handle DNAME and CNAME records before A and AAAA records to make
	// sure temporary watches are set before checking address records
	for _, m := range []map[uint16]func(dns.RR){
		{dns.TypeDNAME: handleDNAME},
		{dns.TypeCNAME: handleCNAME},
		{dns.TypeA: handleA, dns.TypeAAAA: handleAAAA},
	} {
		for _, a := range reply.Answer {
			// ignore domain names we do not watch
			name := a.Header().Name
			if !p.watches.Contains(name) {
				// not on watch list, ignore answer
				continue
			}

			// handle record types
			typ := a.Header().Rrtype
			if m[typ] != nil {
				m[typ](a)
			}
		}
	}

	// send reply to client
	if err := w.WriteMsg(reply); err != nil {
		log.WithError(err).Error("DNS-Proxy could not forward reply")
	}
}

// startDNSServer starts the dns server.
func (p *Proxy) startDNSServer(server *dns.Server) {
	if server == nil {
		return
	}

	log.WithFields(log.Fields{
		"addr": server.Addr,
		"net":  server.Net,
	}).Debug("DNS-Proxy starting server")
	err := server.ListenAndServe()
	if err != nil {
		log.WithError(err).Error("DNS-Proxy DNS server stopped")
	}
}

// stopDNSServer stops the dns server.
func (p *Proxy) stopDNSServer(server *dns.Server) {
	if server == nil {
		return
	}

	err := server.Shutdown()
	if err != nil {
		log.WithFields(log.Fields{
			"addr":  server.Addr,
			"net":   server.Net,
			"error": err,
		}).Error("DNS-Proxy could not stop DNS server")
	}
}

// start starts running the proxy.
func (p *Proxy) start() {
	defer close(p.closed)
	defer close(p.reports)
	defer p.watches.Close()

	// start dns servers
	log.Debug("DNS-Proxy registering handler")
	dns.HandleFunc(".", p.handleRequest)
	for _, srv := range []*dns.Server{p.udp, p.tcp} {
		go p.startDNSServer(srv)
	}

	// wait for proxy termination
	<-p.done

	// stop dns servers
	for _, srv := range []*dns.Server{p.udp, p.tcp} {
		p.stopDNSServer(srv)
	}
}

// Start starts running the proxy.
func (p *Proxy) Start() {
	go p.start()
}

// Stop stops running the proxy.
func (p *Proxy) Stop() {
	close(p.done)
	<-p.closed
}

// Reports returns the Report channel for watched domains.
func (p *Proxy) Reports() chan *Report {
	return p.reports
}

// SetRemotes sets the mapping from domain names to remote server addresses.
func (p *Proxy) SetRemotes(remotes map[string][]string) {
	p.remotes.Flush()
	for d, s := range remotes {
		p.remotes.Add(d, s)
	}
}

// SetWatches sets the domains watched for A and AAAA record updates.
func (p *Proxy) SetWatches(watches []string) {
	p.watches.Flush()
	for _, d := range watches {
		p.watches.Add(d)
	}
}

// GetState returns the internal state of the DNS Proxy.
func (p *Proxy) GetState() *State {
	watches, tempWatches := p.watches.List()
	return &State{
		Remotes:     p.remotes.List(),
		Watches:     watches,
		TempWatches: tempWatches,
	}
}

// NewProxy returns a new Proxy that listens on address.
func NewProxy(config *daemoncfg.DNSProxy) *Proxy {
	var udp *dns.Server
	if config.ListenUDP {
		udp = &dns.Server{
			Addr: config.Address,
			Net:  "udp",
		}
	}
	var tcp *dns.Server
	if config.ListenTCP {
		tcp = &dns.Server{
			Addr: config.Address,
			Net:  "tcp",
		}
	}
	return &Proxy{
		config:  config,
		udp:     udp,
		tcp:     tcp,
		remotes: NewRemotes(),
		watches: NewWatches(),
		reports: make(chan *Report),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
}
