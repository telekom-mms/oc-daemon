package trafpol

import (
	"net/netip"
	"testing"
	"time"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// TestResolverStartStop tests Start and Stop of Resolver.
func TestResolverStartStop(_ *testing.T) {
	config := daemoncfg.NewTrafficPolicing()
	names := []string{}
	updates := make(chan *ResolvedName)
	r := NewResolver(config, names, updates)
	r.Start()
	r.Stop()
}

// TestResolverResolve tests Resolve of Resolver.
func TestResolverResolve(_ *testing.T) {
	config := daemoncfg.NewTrafficPolicing()
	config.ResolveTriesSleep = 0
	config.ResolveTimer = 0
	names := []string{"does not exist...", "example.com"}
	updates := make(chan *ResolvedName)
	r := NewResolver(config, names, updates)

	r.Start()

	// test resolve
	r.Resolve()
	u := <-updates // use it for test with changes

	// test double update, should not resolve entries because of TTL
	r.Resolve()
	r.Resolve()

	// wait for timer
	time.Sleep(time.Second)

	r.Stop()

	// try to test resolve with changes, using previous update
	if len(u.IPs) > 0 {
		names = []string{"example.com"}
		r = NewResolver(config, names, updates)
		r.names["example.com"] = u
		r.names["example.com"].IPs[0] = netip.MustParseAddr("127.0.0.1")

		r.Start()

		r.Resolve()
		<-updates

		r.Stop()
	}
}

// TestNewResolver tests NewResolver.
func TestNewResolver(t *testing.T) {
	config := daemoncfg.NewTrafficPolicing()
	names := []string{"test.example.com"}
	updates := make(chan *ResolvedName)
	r := NewResolver(config, names, updates)

	// check resolver
	if r == nil ||
		r.config != config ||
		r.updates != updates ||
		r.cmds == nil ||
		r.done == nil ||
		r.closed == nil {

		t.Fatalf("invalid resolver")
	}

	// check names
	if len(r.names) != len(names) {
		t.Fatalf("names do not match")
	}
	for rn := range r.names {
		found := false
		for _, n := range names {
			if n == rn {
				found = true
			}
		}
		if !found {
			t.Fatalf("names do not match")
		}
	}
}
