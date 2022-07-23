package splitrt

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// excludesTimer is the timer for periodic exclude cleanup in seconds
	excludesTimer = 300
)

// exclude is a split excludes entry
type exclude struct {
	net     *net.IPNet
	static  bool
	ttl     uint32
	updated bool
}

// Excludes contains split Excludes
type Excludes struct {
	sync.Mutex
	m      map[string]*exclude
	done   chan struct{}
	closed chan struct{}
}

// addFilter adds the exclude to netfilter
func (e *Excludes) addFilter(exclude *exclude) {
	log.WithField("address", exclude.net).Debug("SplitRouting adding exclude to netfilter")
	addExclude(exclude.net)
}

// setFilter resets the excludes in netfilter
func (e *Excludes) setFilter() {
	log.Debug("SplitRouting resetting excludes in netfilter")

	addresses := []*net.IPNet{}
	for _, v := range e.m {
		addresses = append(addresses, v.net)
	}
	setExcludes(addresses)
}

// add adds the exclude entry for ip to the split excludes
func (e *Excludes) add(ip *net.IPNet, exclude *exclude) {
	e.Lock()
	defer e.Unlock()

	key := ip.String()
	old := e.m[key]

	// new entry, just add it
	if old == nil {
		e.m[key] = exclude
		e.addFilter(exclude)
		return
	}

	// old entry exists, update values
	if old.static {
		// static entry is not updated
		return
	}
	// update entry
	old.static = exclude.static
	old.ttl = exclude.ttl
	old.updated = true
}

// AddStatic adds a static entry to the split excludes
func (e *Excludes) AddStatic(address *net.IPNet) {
	log.WithField("address", address).Debug("SplitRouting adding static exclude")
	e.add(address, &exclude{
		net:    address,
		static: true,
	})
}

// AddDynamic adds a dynamic entry to the split excludes
func (e *Excludes) AddDynamic(address *net.IPNet, ttl uint32) {
	log.WithFields(log.Fields{
		"address": address,
		"ttl":     ttl,
	}).Debug("SplitRouting adding dynamic exclude")
	e.add(address, &exclude{
		net:     address,
		ttl:     ttl,
		updated: true,
	})
}

// Remove removes an entry from the split excludes
func (e *Excludes) Remove(address *net.IPNet) {
	e.Lock()
	defer e.Unlock()

	delete(e.m, address.String())
	e.setFilter()
}

// cleanup cleans up the split excludes
func (e *Excludes) cleanup() {
	e.Lock()
	defer e.Unlock()

	changed := false
	for k, v := range e.m {
		// skip static entries
		if v.static {
			continue
		}

		// skip recently updated entries
		if v.updated {
			v.updated = false
			continue
		}

		// exclude expired entries
		if v.ttl < excludesTimer {
			delete(e.m, k)
			changed = true
			continue
		}

		// reduce ttl
		v.ttl -= excludesTimer
	}

	// if entries were changed, reset netfilter
	if changed {
		e.setFilter()
	}
}

// start starts pertiodic cleanup of the split excludes
func (e *Excludes) start() {
	defer close(e.closed)

	timer := time.NewTimer(excludesTimer * time.Second)
	for {
		select {
		case <-timer.C:
			e.cleanup()
			timer.Reset(excludesTimer * time.Second)

		case <-e.done:
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

// Start starts pertiodic cleanup of the split excludes
func (e *Excludes) Start() {
	log.Debug("SplitRouting starting periodic cleanup of excludes")
	go e.start()
}

// Stop stops pertiodic cleanup of the split excludes
func (e *Excludes) Stop() {
	close(e.done)
	<-e.closed
	log.Debug("SplitRouting stopped periodic cleanup of excludes")
}

// NewExcludes returns new split excludes
func NewExcludes() *Excludes {
	return &Excludes{
		m:      make(map[string]*exclude),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}
