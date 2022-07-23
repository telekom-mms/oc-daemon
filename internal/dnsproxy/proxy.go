package dnsproxy

import (
	"math/rand"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

const (
	// tempWatchCleanInterval clean interval for temporary watches
	// in seconds
	tempWatchCleanInterval = 10
)

// Proxy is a DNS proxy
type Proxy struct {
	server  *dns.Server
	remotes *Remotes
	watches *Watches
	reports chan *Report

	// channels for temp watch cleaning goroutine
	stopClean chan struct{}
	doneClean chan struct{}
}

// handleRequest handles a dns client request
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

		// get TTL and enforce minimum TTL
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
			// CNAME recort, store temporary watch
			rr, ok := a.(*dns.CNAME)
			if !ok {
				log.Error("DNS-Proxy received invalid CNAME record in reply")
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

// cleanTempWatches cleans temporary watches
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

// start starts running the proxy
func (p *Proxy) start() {
	// start cleaning goroutine
	go p.cleanTempWatches()

	// start dns server
	log.Debug("DNS-Proxy registering handler")
	dns.HandleFunc(".", p.handleRequest)
	log.WithFields(log.Fields{
		"addr": p.server.Addr,
		"net":  p.server.Net,
	}).Debug("DNS-Proxy starting server")
	err := p.server.ListenAndServe()
	if err != nil {
		log.WithError(err).Error("DNS-Proxy could not start DNS server")
	}
}

// Start starts running the proxy
func (p *Proxy) Start() {
	go p.start()
}

// Stop stops running the proxy
func (p *Proxy) Stop() {
	// stop cleaning goroutine
	close(p.stopClean)
	<-p.doneClean

	// stop server
	err := p.server.Shutdown()
	if err != nil {
		log.WithError(err).Fatal("DNS-Proxy could not stop DNS server")
	}
	close(p.reports)
}

// Reports returns the Report channel for watched domains
func (p *Proxy) Reports() chan *Report {
	return p.reports
}

// SetRemotes sets the mapping from domain names to remote server addresses
func (p *Proxy) SetRemotes(remotes map[string][]string) {
	p.remotes.Flush()
	for d, s := range remotes {
		p.remotes.Add(d, s)
	}
}

// SetWatches sets the domains watched for A and AAAA record updates
func (p *Proxy) SetWatches(watches []string) {
	p.watches.Flush()
	for _, d := range watches {
		p.watches.Add(d)
	}
}

// NewProxy returns a new Proxy that listens on address
func NewProxy(address string) *Proxy {
	return &Proxy{
		server: &dns.Server{
			Addr: address,
			Net:  "udp",
		},
		remotes: NewRemotes(),
		watches: NewWatches(),
		reports: make(chan *Report),

		stopClean: make(chan struct{}),
		doneClean: make(chan struct{}),
	}
}
