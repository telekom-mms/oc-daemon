package splitrt

import (
	"net/netip"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
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
	conf   *daemoncfg.Config
	s      map[string]netip.Prefix
	d      map[netip.Addr]*dynExclude
	done   chan struct{}
	closed chan struct{}
}

// GetPrefixes returns static and dynamic split excludes as Prefixes.
func (e *Excludes) GetPrefixes() []netip.Prefix {
	e.Lock()
	defer e.Unlock()

	addresses := []netip.Prefix{}
	for _, v := range e.s {
		addresses = append(addresses, v)
	}
	for k := range e.d {
		prefix := netip.PrefixFrom(k, k.BitLen())
		addresses = append(addresses, prefix)
	}

	return addresses
}

// AddStatic adds a static entry to the split excludes.
func (e *Excludes) AddStatic(address netip.Prefix) bool {
	log.WithField("address", address).Debug("SplitRouting adding static exclude")
	e.Lock()
	defer e.Unlock()

	// make sure new prefix in address does not overlap with existing
	// prefixes in static excludes
	for k, v := range e.s {
		if !v.Overlaps(address) {
			// no overlap
			continue
		}
		if v.Bits() <= address.Bits() {
			// new prefix is already in existing prefix,
			// do not add it
			return false
		}

		// new prefix contains old prefix, remove old prefix
		delete(e.s, k)
	}

	// add new prefix to static excludes
	key := address.String()
	e.s[key] = address

	// update netfilter
	return true
}

// AddDynamic adds a dynamic entry to the split excludes.
func (e *Excludes) AddDynamic(address netip.Prefix, ttl uint32) bool {
	log.WithFields(log.Fields{
		"address": address,
		"ttl":     ttl,
	}).Debug("SplitRouting adding dynamic exclude")

	if !address.IsSingleIP() {
		log.Error("SplitRouting error adding dynamic exclude with multiple IPs")
		return false
	}
	a := address.Addr()

	e.Lock()
	defer e.Unlock()

	// make sure new ip address is not in existing static excludes
	for _, v := range e.s {
		if v.Contains(a) {
			return false
		}
	}

	// update existing entry in dynamic excludes
	old := e.d[a]
	if old != nil {
		old.ttl = ttl
		old.updated = true
		return false
	}

	// create new entry in dynamic excludes
	e.d[a] = &dynExclude{
		ttl:     ttl,
		updated: true,
	}

	// update netfilter
	return true
}

// RemoveStatic removes a static entry from the split excludes.
func (e *Excludes) RemoveStatic(address netip.Prefix) bool {
	e.Lock()
	defer e.Unlock()

	addr := address.String()
	if _, ok := e.s[addr]; !ok {
		return false
	}
	delete(e.s, addr)
	return true
}

// cleanup cleans up the dynamic split excludes.
func (e *Excludes) cleanup() bool {
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
	return changed
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
func NewExcludes(conf *daemoncfg.Config) *Excludes {
	return &Excludes{
		conf:   conf,
		s:      make(map[string]netip.Prefix),
		d:      make(map[netip.Addr]*dynExclude),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}
