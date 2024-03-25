// Package dnsproxy contains the DNS proxy.
package dnsproxy

import (
	"math/rand"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

const (
	// tempWatchCleanInterval clean interval for temporary watches
	// in seconds.
	tempWatchCleanInterval = 10
)

// Proxy is a DNS proxy.
type Proxy struct {
	config  *Config
	udp     *dns.Server
	tcp     *dns.Server
	remotes *Remotes
	watches *Watches
	reports chan *Report
	done    chan struct{}
	closed  chan struct{}

	// channels for temp watch cleaning goroutine
	stopClean chan struct{}
	doneClean chan struct{}
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
	for _, a := range reply.Answer {
		name := a.Header().Name
		if !p.watches.Contains(r.Question[0].Name) &&
			!p.watches.Contains(name) {
			// not on watch list, ignore answer
			continue
		}

		// get TTL
		ttl := a.Header().Ttl

		switch a.Header().Rrtype {
		case dns.TypeA:
			// A Record, get IPv4 address
			rr, ok := a.(*dns.A)
			if !ok {
				log.Error("DNS-Proxy received invalid A record in reply")
				continue
			}
			report := NewReport(name, rr.A, ttl)
			p.reports <- report
			report.Wait()

		case dns.TypeAAAA:
			// AAAA Record, get IPv6 address
			rr, ok := a.(*dns.AAAA)
			if !ok {
				log.Error("DNS-Proxy received invalid AAAA record in reply")
				continue
			}
			report := NewReport(name, rr.AAAA, ttl)
			p.reports <- report
			report.Wait()

		case dns.TypeCNAME:
			// CNAME record, store temporary watch
			rr, ok := a.(*dns.CNAME)
			if !ok {
				log.Error("DNS-Proxy received invalid CNAME record in reply")
				continue
			}
			log.WithFields(log.Fields{
				"target": rr.Target,
				"ttl":    ttl,
			}).Debug("DNS-Proxy received CNAME in reply")
			p.watches.AddTemp(rr.Target, ttl)
		}
	}

	// send reply to client
	if err := w.WriteMsg(reply); err != nil {
		log.WithError(err).Error("DNS-Proxy could not forward reply")
	}
}

// cleanTempWatches cleans temporary watches.
func (p *Proxy) cleanTempWatches() {
	defer close(p.doneClean)

	timer := time.NewTimer(tempWatchCleanInterval * time.Second)
	for {
		select {
		case <-timer.C:
			// reset timer
			timer.Reset(tempWatchCleanInterval * time.Second)

			// clean temporary watches
			p.watches.CleanTemp(tempWatchCleanInterval)

		case <-p.stopClean:
			// stop timer
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
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

	// start cleaning goroutine
	go p.cleanTempWatches()

	// start dns servers
	log.Debug("DNS-Proxy registering handler")
	dns.HandleFunc(".", p.handleRequest)
	for _, srv := range []*dns.Server{p.udp, p.tcp} {
		go p.startDNSServer(srv)
	}

	// wait for proxy termination
	<-p.done

	// stop cleaning goroutine
	close(p.stopClean)
	<-p.doneClean

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

// NewProxy returns a new Proxy that listens on address.
func NewProxy(config *Config) *Proxy {
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

		stopClean: make(chan struct{}),
		doneClean: make(chan struct{}),
	}
}
