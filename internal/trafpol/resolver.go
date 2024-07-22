package trafpol

import (
	"context"
	"errors"
	"net"
	"sort"
	"sync"
	"time"
)

// ResolvedName is a resolved DNS name.
type ResolvedName struct {
	Name string
	IPs  []net.IP
	TTL  time.Duration
}

// sleepResolveTry is used to sleep before resolve (re)tries, can be canceled.
func (r *ResolvedName) sleepResolveTry(ctx context.Context, config *Config) {
	timer := time.NewTimer(config.ResolveTriesSleep)
	select {
	case <-timer.C:
	case <-ctx.Done():
		// stop timer
		if !timer.Stop() {
			<-timer.C
		}
	}
}

// resolve resolves the DNS name to its IP addresses.
func (r *ResolvedName) resolve(ctx context.Context, config *Config, updates chan *ResolvedName) {
	// try to resolve ip addresses of host
	resolver := &net.Resolver{}
	tries := 0
	for tries < config.ResolveTries {
		tries++

		// sleep before (re)tries
		r.sleepResolveTry(ctx, config)

		// set timeout
		ctxTO, cancel := context.WithTimeout(ctx, config.ResolveTimeout)
		defer cancel()

		// the resolver seems to struggle with some domain names if we
		// lookup IPv4 and IPv6 addresses in one call (argument
		// network == "ip"). So, resolve IPv4 and IPv6 addresses in
		// separate calls
		ipv4s, err4 := resolver.LookupIP(ctxTO, "ip4", r.Name)
		ipv6s, err6 := resolver.LookupIP(ctxTO, "ip6", r.Name)
		if err4 != nil && err6 != nil {
			// do not retry hostnames that are not found
			var dnsErr4 *net.DNSError
			var dnsErr6 *net.DNSError
			if errors.As(err4, &dnsErr4) && errors.As(err6, &dnsErr6) {
				if dnsErr4.IsNotFound && dnsErr6.IsNotFound {
					r.TTL = config.ResolveTTL
					return
				}
			}

			// if we cannot resolve the host, retry or
			// keep existing IPs
			continue
		}
		r.TTL = config.ResolveTTL

		// sort ips
		ips := append(ipv4s, ipv6s...)
		sort.Slice(ips, func(i, j int) bool {
			return ips[i].String() < ips[j].String()
		})

		// check if there was an update
		equal := func() bool {
			if len(r.IPs) != len(ips) {
				return false
			}
			for i := range r.IPs {
				if !r.IPs[i].Equal(ips[i]) {
					return false
				}
			}
			return true
		}
		if equal() {
			return
		}

		// update ips
		r.IPs = ips

		// send update over updates channel
		select {
		case updates <- r:
		case <-ctx.Done():
		}
		return
	}
}

// Resolver is a DNS resolver that resolves names to their IP addresses.
type Resolver struct {
	config  *Config
	names   map[string]*ResolvedName
	updates chan *ResolvedName
	cmds    chan struct{}
	done    chan struct{}
	closed  chan struct{}
}

// update resolves the DNS names. If force is set, it updates all names.
// Otherwise, it updates only names that have been resolved more than
// config.ResolveTTL ago.
func (r *Resolver) update(ctx context.Context, upDone chan<- struct{}, force bool) {
	// get names to resolve
	// if force is set, update all names
	// otherwise, only update old names
	names := []*ResolvedName{}
	for _, n := range r.names {
		if !force && n.TTL > r.config.ResolveTimer {
			n.TTL -= r.config.ResolveTimer
			continue
		}
		names = append(names, n)
	}

	// create workers that resolve names concurrently
	// put all names in a queue and read queue from workers
	// use 1 worker per 10 names
	var wg sync.WaitGroup
	queue := make(chan *ResolvedName, len(names))
	for _, n := range names {
		queue <- n
	}
	close(queue)
	workers := (len(names) / 10) + 1
	wg.Add(workers)
	for range workers {
		// start worker
		go func() {
			defer wg.Done()
			for name := range queue {
				name.resolve(ctx, r.config, r.updates)
			}
		}()
	}

	// wait for workers and signal update is done
	wg.Wait()
	upDone <- struct{}{}
}

// start starts the Resolver.
func (r *Resolver) start() {
	defer close(r.closed)

	timer := time.NewTimer(r.config.ResolveTimer)
	updating := false
	upAgain := false
	upDone := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	for {
		select {
		case <-r.cmds:
			if updating {
				// update already in progress, queue another
				upAgain = true
				break
			}

			// update all
			updating = true
			go r.update(ctx, upDone, true)

		case <-upDone:
			if upAgain {
				// trigger another update
				upAgain = false
				go r.update(ctx, upDone, true)
				break
			}
			updating = false

		case <-timer.C:
			// reset periodic timer
			timer.Reset(r.config.ResolveTimer)

			if updating {
				// update already in progress, skip periodic
				break
			}

			// periodic update
			updating = true
			go r.update(ctx, upDone, false)

		case <-r.done:
			// cancel and wait for ongoing update
			cancel()
			if updating {
				<-upDone
			}

			// stop timer
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

// Start starts the Resolver.
func (r *Resolver) Start() {
	go r.start()
}

// Stop stops the Resolver.
func (r *Resolver) Stop() {
	close(r.done)

	// wait for shutdown
	<-r.closed
}

// Resolve resolves all names.
func (r *Resolver) Resolve() {
	r.cmds <- struct{}{}
}

// NewResolver returns a new Resolver.
func NewResolver(config *Config, names []string, updates chan *ResolvedName) *Resolver {
	n := make(map[string]*ResolvedName)
	for _, name := range names {
		n[name] = &ResolvedName{Name: name}
	}
	return &Resolver{
		config:  config,
		names:   n,
		updates: updates,
		cmds:    make(chan struct{}),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
}
