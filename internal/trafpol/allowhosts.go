package trafpol

import (
	"context"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	// resolveTimeout is the timeout for dns lookups
	resolveTimeout = 2 * time.Second

	// resolveTries is the number of tries for dns lookups
	resolveTries = 3

	// resolveTriesSleep is the sleep time between retries
	resolveTriesSleep = time.Second

	// resolveTimer is the time for periodic resolve update checks,
	// should be higher than tries * (timeout + sleep)
	resolveTimer = 30 * time.Second

	// resolveTTL is the lifetime of resolved entries
	resolveTTL = 300 * time.Second
)

// allowHost is an allowed hosts entry
type allowHost struct {
	host string
	ips  []*net.IPNet

	updated    bool
	lastUpdate time.Time
}

// resolve resolves the allowed host to its IP addresses
func (a *allowHost) resolve() {
	// check if host is a network address
	if _, ipnet, err := net.ParseCIDR(a.host); err == nil {
		a.lastUpdate = time.Now()

		// check if address already exists
		if len(a.ips) == 1 && ipnet.String() == a.ips[0].String() {
			return
		}

		// add new address
		a.ips = []*net.IPNet{ipnet}
		a.updated = true
		return
	}

	// try to resolve ip addresses of host
	tries := 0
	for tries < resolveTries {
		tries++

		// sleep before (re)tries
		time.Sleep(resolveTriesSleep)

		// resolve ips
		ctx, cancel := context.WithTimeout(context.TODO(), resolveTimeout)
		defer cancel()
		resolver := &net.Resolver{}
		ips, err := resolver.LookupIP(ctx, "ip", a.host)
		if err != nil {
			// if we cannot resolve the host, retry or
			// keep existing IPs
			continue
		}
		a.lastUpdate = time.Now()

		// convert ips
		ipnets := []*net.IPNet{}
		for _, ip := range ips {
			ipnet := &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(32, 32),
			}
			if ip.To4() == nil {
				ipnet.Mask = net.CIDRMask(128, 128)
			}
			ipnets = append(ipnets, ipnet)
		}

		// sort ips
		sort.Slice(ipnets, func(i, j int) bool {
			return ipnets[i].String() < ipnets[j].String()
		})

		// check if there was an update
		equal := func() bool {
			if len(a.ips) != len(ipnets) {
				return false
			}
			for i := range a.ips {
				if a.ips[i].String() != ipnets[i].String() {
					return false
				}
			}
			return true
		}
		if equal() {
			return
		}

		// update ips
		a.ips = ipnets
		a.updated = true
		return
	}
}

// AllowHosts contains allowed hosts
type AllowHosts struct {
	sync.Mutex
	m map[string]*allowHost

	updates chan struct{}
	done    chan struct{}
	closed  chan struct{}
}

// Add adds host to the allowed hosts
func (a *AllowHosts) Add(host string) {
	a.Lock()
	defer a.Unlock()

	if a.m[host] == nil {
		a.m[host] = &allowHost{
			host: host,
		}
	}
}

// Remove removes host from the allowed hosts
func (a *AllowHosts) Remove(host string) {
	a.Lock()
	defer a.Unlock()

	if a.m[host] != nil {
		delete(a.m, host)
	}
}

// resolveAll resolves the IP addresses of all allowed hosts
func (a *AllowHosts) resolveAll() {
	a.Lock()
	defer a.Unlock()

	var wg sync.WaitGroup
	for _, h := range a.m {
		wg.Add(1)
		go func(host *allowHost) {
			defer wg.Done()
			host.resolve()
		}(h)
	}
	wg.Wait()
}

// getAndClearUpdates checks if the allowed hosts contain updates and resets
// all the update flags
func (a *AllowHosts) getAndClearUpdates() bool {
	a.Lock()
	defer a.Unlock()

	updates := false
	for _, h := range a.m {
		updates = updates || h.updated
		h.updated = false
	}

	return updates
}

// setFilter sets the allowed hosts in the traffic filter
func (a *AllowHosts) setFilter() {
	a.Lock()
	defer a.Unlock()

	// get a list of all unique ip addresses
	ipset := make(map[string]*net.IPNet)
	for _, h := range a.m {
		for _, ip := range h.ips {
			ipset[ip.String()] = ip
		}
	}
	ips := []*net.IPNet{}
	for _, ip := range ipset {
		ips = append(ips, ip)
	}

	// set ips in traffic filter
	setAllowedIPs(ips)
}

// update updates all allowed hosts
func (a *AllowHosts) update() {
	a.resolveAll()
	if a.getAndClearUpdates() {
		a.setFilter()
	}
}

// resolvePeriodic resolves the IP addresses of all allowed hosts with old or
// no ip addresses, called periodically
func (a *AllowHosts) resolvePeriodic() {
	a.Lock()
	defer a.Unlock()

	now := time.Now()
	var wg sync.WaitGroup
	for _, h := range a.m {
		if now.Sub(h.lastUpdate) < resolveTTL && len(h.ips) != 0 {
			continue
		}
		wg.Add(1)
		go func(host *allowHost) {
			defer wg.Done()
			host.resolve()
		}(h)
	}
	wg.Wait()
}

// updatePeriodic updates old entries and entries without ip addresses,
// called periodically
func (a *AllowHosts) updatePeriodic() {
	a.resolvePeriodic()
	if a.getAndClearUpdates() {
		a.setFilter()
	}
}

// start starts the allowed hosts
func (a *AllowHosts) start() {
	defer close(a.closed)

	timer := time.NewTimer(resolveTimer)
	for {
		select {
		case <-a.updates:
			// update all and reset timer
			a.update()
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(resolveTimer)
		case <-timer.C:
			// periodic update and reset timer
			a.updatePeriodic()
			timer.Reset(resolveTimer)
		case <-a.done:
			// stop timer
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

// Start starts the allowed hosts
func (a *AllowHosts) Start() {
	go a.start()
}

// Stop stops the allowed hosts
func (a *AllowHosts) Stop() {
	close(a.done)

	// wait for shutdown
	<-a.closed
}

// Update updates the allowed hosts entry
func (a *AllowHosts) Update() {
	a.updates <- struct{}{}
}

// NewAllowHosts returns new allowHosts
func NewAllowHosts() *AllowHosts {
	return &AllowHosts{
		m: make(map[string]*allowHost),

		updates: make(chan struct{}),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
}
