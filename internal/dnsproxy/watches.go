package dnsproxy

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

const (
	// tempWatchCleanInterval clean interval for temporary watches
	// in seconds.
	tempWatchCleanInterval = 10
)

// tempWatch is information about a temporary watch domain.
type tempWatch struct {
	ttl     uint32
	updated bool
}

// Watches contains a list of domains to watch for A and AAAA updates.
type Watches struct {
	sync.RWMutex
	m map[string]bool
	t map[string]*tempWatch

	done   chan struct{}
	closed chan struct{}
}

// Add adds domain to the watch list.
func (w *Watches) Add(domain string) {
	w.Lock()
	defer w.Unlock()

	w.m[domain] = true
}

// AddTemp adds a temporary domain to the watch list with a ttl.
func (w *Watches) AddTemp(domain string, ttl uint32) {
	w.Lock()
	defer w.Unlock()

	w.t[domain] = &tempWatch{
		ttl:     ttl,
		updated: true,
	}
}

// Remove removes domain from the watch list.
func (w *Watches) Remove(domain string) {
	w.Lock()
	defer w.Unlock()

	delete(w.m, domain)
	delete(w.t, domain)
}

// cleanTemp removes expired temporary entries from the watch list and reduces
// the ttl of all other entries by interval seconds; this is meant to be called
// once every interval seconds to actually implement cleaning of the cache.
func (w *Watches) cleanTemp(interval uint32) {
	w.Lock()
	defer w.Unlock()

	for d, t := range w.t {
		if t.updated {
			// mark new entries as old
			t.updated = false
			continue
		}

		if t.ttl < interval {
			// delete expired entry
			delete(w.t, d)
			continue
		}

		// reduce ttl
		t.ttl -= interval
	}
}

// cleanTempWatches cleans temporary watches.
func (w *Watches) cleanTempWatches() {
	defer close(w.closed)

	timer := time.NewTimer(tempWatchCleanInterval * time.Second)
	for {
		select {
		case <-timer.C:
			// reset timer
			timer.Reset(tempWatchCleanInterval * time.Second)

			// clean temporary watches
			w.cleanTemp(tempWatchCleanInterval)

		case <-w.done:
			// stop timer
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

// Flush removes all entries from the watch list.
func (w *Watches) Flush() {
	w.Lock()
	defer w.Unlock()

	w.m = make(map[string]bool)
	w.t = make(map[string]*tempWatch)
}

// Contains returns whether the domain is in the watch list.
func (w *Watches) Contains(domain string) bool {
	w.RLock()
	defer w.RUnlock()

	// only handle domain names
	if _, ok := dns.IsDomainName(domain); !ok {
		return false
	}

	// get label indexes and find matching domains
	labels := dns.Split(domain)
	if labels == nil {
		// root domain
		// TODO: remove temp domain check here?
		return w.m["."] || w.t["."] != nil
	}

	// try finding longest matching domain name
	for _, i := range labels {
		d := domain[i:]
		if w.m[d] || w.t[d] != nil {
			return true
		}
	}

	// did not find anything
	return false
}

// Close closes the watches
func (w *Watches) Close() {
	close(w.done)
	<-w.closed
}

// NewWatches returns a new Watches.
func NewWatches() *Watches {
	w := &Watches{
		m: make(map[string]bool),
		t: make(map[string]*tempWatch),

		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}

	// start cleaning goroutine
	go w.cleanTempWatches()

	return w
}
