package trafpol

import (
	"context"
	"net"
	"sort"
	"sync"
	"time"
)

// allowHost is an allowed hosts entry.
type allowHost struct {
	host string
	ips  []*net.IPNet

	updated    bool
	lastUpdate time.Time
}

// sleepResolveTry is used to sleep before resolve (re)tries, can be canceled.
func (a *allowHost) sleepResolveTry(ctx context.Context, config *Config) {
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

// resolve resolves the allowed host to its IP addresses.
func (a *allowHost) resolve(ctx context.Context, config *Config) {
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
	for tries < config.ResolveTries {
		tries++

		// sleep before (re)tries
		a.sleepResolveTry(ctx, config)

		// resolve ips
		ctx, cancel := context.WithTimeout(ctx, config.ResolveTimeout)
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

// AllowHosts contains allowed hosts.
type AllowHosts struct {
	sync.Mutex
	config *Config
	m      map[string]*allowHost

	updates chan struct{}
	done    chan struct{}
	closed  chan struct{}
}

// Add adds host to the allowed hosts.
func (a *AllowHosts) Add(host string) {
	a.Lock()
	defer a.Unlock()

	if a.m[host] == nil {
		a.m[host] = &allowHost{
			host: host,
		}
	}
}

// Remove removes host from the allowed hosts.
func (a *AllowHosts) Remove(host string) {
	a.Lock()
	defer a.Unlock()

	if a.m[host] != nil {
		delete(a.m, host)
	}
}

// resolveAll resolves the IP addresses of all allowed hosts.
func (a *AllowHosts) resolveAll(ctx context.Context) {
	a.Lock()
	defer a.Unlock()

	var wg sync.WaitGroup
	for _, h := range a.m {
		wg.Add(1)
		go func(host *allowHost) {
			defer wg.Done()
			host.resolve(ctx, a.config)
		}(h)
	}
	wg.Wait()
}

// getAndClearUpdates checks if the allowed hosts contain updates and resets
// all the update flags.
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

// setFilter sets the allowed hosts in the traffic filter.
func (a *AllowHosts) setFilter(ctx context.Context) {
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
	setAllowedIPs(ctx, ips)
}

// update updates all allowed hosts.
func (a *AllowHosts) update(ctx context.Context, upDone chan<- struct{}) {
	a.resolveAll(ctx)
	if a.getAndClearUpdates() {
		a.setFilter(ctx)
	}
	upDone <- struct{}{}
}

// resolvePeriodic resolves the IP addresses of all allowed hosts with old or
// no ip addresses, called periodically.
func (a *AllowHosts) resolvePeriodic(ctx context.Context) {
	a.Lock()
	defer a.Unlock()

	now := time.Now()
	var wg sync.WaitGroup
	for _, h := range a.m {
		if now.Sub(h.lastUpdate) < a.config.ResolveTTL && len(h.ips) != 0 {
			continue
		}
		wg.Add(1)
		go func(host *allowHost) {
			defer wg.Done()
			host.resolve(ctx, a.config)
		}(h)
	}
	wg.Wait()
}

// updatePeriodic updates old entries and entries without ip addresses,
// called periodically.
func (a *AllowHosts) updatePeriodic(ctx context.Context, upDone chan<- struct{}) {
	a.resolvePeriodic(ctx)
	if a.getAndClearUpdates() {
		a.setFilter(ctx)
	}
	upDone <- struct{}{}
}

// start starts the allowed hosts.
func (a *AllowHosts) start() {
	defer close(a.closed)

	timer := time.NewTimer(a.config.ResolveTimer)
	updating := false
	upAgain := false
	upDone := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	for {
		select {
		case <-a.updates:
			if updating {
				// update already in progress, queue another
				upAgain = true
				break
			}

			// update all
			updating = true
			go a.update(ctx, upDone)

		case <-upDone:
			if upAgain {
				// trigger another update
				upAgain = false
				go a.update(ctx, upDone)
				break
			}
			updating = false

		case <-timer.C:
			// reset periodic timer
			timer.Reset(a.config.ResolveTimer)

			if updating {
				// update already in progress, skip periodic
				break
			}

			// periodic update
			updating = true
			go a.updatePeriodic(ctx, upDone)

		case <-a.done:
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

// Start starts the allowed hosts.
func (a *AllowHosts) Start() {
	go a.start()
}

// Stop stops the allowed hosts.
func (a *AllowHosts) Stop() {
	close(a.done)

	// wait for shutdown
	<-a.closed
}

// Update updates the allowed hosts entry.
func (a *AllowHosts) Update() {
	a.updates <- struct{}{}
}

// NewAllowHosts returns new AllowHosts.
func NewAllowHosts(config *Config) *AllowHosts {
	return &AllowHosts{
		config: config,
		m:      make(map[string]*allowHost),

		updates: make(chan struct{}),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
}
