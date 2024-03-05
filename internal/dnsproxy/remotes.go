package dnsproxy

import (
	"sync"

	"github.com/miekg/dns"
)

// Remotes contains a mapping from domain names to remote DNS servers.
type Remotes struct {
	sync.RWMutex
	m map[string][]string
}

// Add adds a mapping from domain to servers.
func (r *Remotes) Add(domain string, servers []string) {
	r.Lock()
	defer r.Unlock()

	r.m[domain] = servers
}

// Remove removes a mapping from domain to servers.
func (r *Remotes) Remove(domain string) {
	r.Lock()
	defer r.Unlock()

	delete(r.m, domain)
}

// Flush removes all mappings.
func (r *Remotes) Flush() {
	r.Lock()
	defer r.Unlock()

	r.m = make(map[string][]string)
}

// Get returns the servers for domain.
func (r *Remotes) Get(domain string) []string {
	r.RLock()
	defer r.RUnlock()

	// only handle domain names
	if _, ok := dns.IsDomainName(domain); !ok {
		// no domain name, return default remote
		return r.m["."]
	}

	// get label indexes and find matching remotes
	labels := dns.Split(domain)
	if labels == nil {
		// root domain, return default remote
		return r.m["."]
	}

	// try finding longest matching domain name
	for _, i := range labels {
		d := domain[i:]
		s := r.m[d]
		if len(s) != 0 {
			return s
		}
	}

	// did not find anything, return default remote
	return r.m["."]
}

// NewRemotes returns a new Remotes.
func NewRemotes() *Remotes {
	return &Remotes{
		m: make(map[string][]string),
	}
}
