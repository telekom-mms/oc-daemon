package trafpol

import (
	"net/netip"
	"reflect"
	"testing"
	"time"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// TestResolvedNameCopy tests copy of ResolvedName.
func TestResolvedNameCopy(t *testing.T) {
	// test empty, filled
	for _, r := range []*ResolvedName{
		{},
		{
			Name: "test",
			IPs:  []netip.Addr{netip.MustParseAddr("192.168.1.1")},
			TTL:  10 * time.Second,
		},
	} {
		want := r
		got := want.copy()

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test modification after copy
	r1 := &ResolvedName{}
	r2 := r1.copy()
	r1.Name = "test"
	r1.IPs = []netip.Addr{netip.MustParseAddr("192.168.1.1")}
	r1.TTL = 10 * time.Second

	if reflect.DeepEqual(r1, r2) {
		t.Error("copies should not be equal after modification")
	}
}

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
