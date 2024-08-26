package splitrt

import (
	"context"
	"net/netip"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// excludesTimer is the timer for periodic exclude cleanup in seconds.
	excludesTimer = 300
)

// dynExclude is a dynamic split excludes entry.
type dynExclude struct {
	ttl     uint32
	updated bool
}

// Excludes contains split Excludes.
type Excludes struct {
	sync.Mutex
	s      map[string]netip.Prefix
	d      map[netip.Addr]*dynExclude
	done   chan struct{}
	closed chan struct{}
}

// setFilter resets the excludes in netfilter.
func (e *Excludes) setFilter(ctx context.Context) {
	log.Debug("SplitRouting resetting excludes in netfilter")

	addresses := []netip.Prefix{}
	for _, v := range e.s {
		addresses = append(addresses, v)
	}
	for k := range e.d {
		prefix := netip.PrefixFrom(k, k.BitLen())
		addresses = append(addresses, prefix)
	}
	setExcludes(ctx, addresses)
}

// AddStatic adds a static entry to the split excludes.
func (e *Excludes) AddStatic(ctx context.Context, address netip.Prefix) {
	log.WithField("address", address).Debug("SplitRouting adding static exclude")
	e.Lock()
	defer e.Unlock()

	// make sure new prefix in address does not overlap with existing
	// prefixes in static excludes
	removed := false
	for k, v := range e.s {
		if !v.Overlaps(address) {
			// no overlap
			continue
		}
		if v.Bits() <= address.Bits() {
			// new prefix is already in existing prefix,
			// do not add it
			return
		}

		// new prefix contains old prefix, remove old prefix
		delete(e.s, k)
		removed = true
	}

	// add new prefix to static excludes
	key := address.String()
	e.s[key] = address

	// add to netfilter
	if removed {
		// existing entries removed, we need to reset all excludes
		e.setFilter(ctx)
		return
	}
	// single new entry, add it
	addExclude(ctx, address)
}

// AddDynamic adds a dynamic entry to the split excludes.
func (e *Excludes) AddDynamic(ctx context.Context, address netip.Prefix, ttl uint32) {
	log.WithFields(log.Fields{
		"address": address,
		"ttl":     ttl,
	}).Debug("SplitRouting adding dynamic exclude")

	if !address.IsSingleIP() {
		log.Error("SplitRouting error adding dynamic exclude with multiple IPs")
		return
	}
	a := address.Addr()

	e.Lock()
	defer e.Unlock()

	// make sure new ip address is not in existing static excludes
	for _, v := range e.s {
		if v.Contains(a) {
			return
		}
	}

	// update existing entry in dynamic excludes
	old := e.d[a]
	if old != nil {
		old.ttl = ttl
		old.updated = true
		return
	}

	// create new entry in dynamic excludes
	e.d[a] = &dynExclude{
		ttl:     ttl,
		updated: true,
	}

	// add to netfilter
	addExclude(ctx, address)
}

// RemoveStatic removes a static entry from the split excludes.
func (e *Excludes) RemoveStatic(ctx context.Context, address netip.Prefix) {
	e.Lock()
	defer e.Unlock()

	delete(e.s, address.String())
	e.setFilter(ctx)
}

// cleanup cleans up the dynamic split excludes.
func (e *Excludes) cleanup(ctx context.Context) {
	e.Lock()
	defer e.Unlock()

	changed := false
	for k, v := range e.d {
		// skip recently updated entries
		if v.updated {
			v.updated = false
			continue
		}

		// exclude expired entries
		if v.ttl < excludesTimer {
			delete(e.d, k)
			changed = true
			continue
		}

		// reduce ttl
		v.ttl -= excludesTimer
	}

	// if entries were changed, reset netfilter
	if changed {
		e.setFilter(ctx)
	}
}

// start starts periodic cleanup of the split excludes.
func (e *Excludes) start() {
	defer close(e.closed)

	ctx := context.Background()
	timer := time.NewTimer(excludesTimer * time.Second)
	for {
		select {
		case <-timer.C:
			e.cleanup(ctx)
			timer.Reset(excludesTimer * time.Second)

		case <-e.done:
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

// Start starts periodic cleanup of the split excludes.
func (e *Excludes) Start() {
	log.Debug("SplitRouting starting periodic cleanup of excludes")
	go e.start()
}

// Stop stops periodic cleanup of the split excludes.
func (e *Excludes) Stop() {
	close(e.done)
	<-e.closed
	log.Debug("SplitRouting stopped periodic cleanup of excludes")
}

// List returns the list of static and dynamic excludes.
func (e *Excludes) List() (static, dynamic []string) {
	e.Lock()
	defer e.Unlock()

	for k := range e.s {
		static = append(static, k)
	}
	for k := range e.d {
		dynamic = append(dynamic, k.String())
	}

	return
}

// NewExcludes returns new split excludes.
func NewExcludes() *Excludes {
	return &Excludes{
		s:      make(map[string]netip.Prefix),
		d:      make(map[netip.Addr]*dynExclude),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}
