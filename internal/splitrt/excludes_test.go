package splitrt

import (
	"context"
	"net/netip"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// getTestExcludes returns excludes for testing.
func getTestExcludes(t *testing.T, es []string) []netip.Prefix {
	excludes := []netip.Prefix{}
	for _, s := range es {
		exclude, err := netip.ParsePrefix(s)
		if err != nil {
			t.Fatal(err)
		}
		excludes = append(excludes, exclude)
	}
	return excludes
}

// getTestStaticExcludes returns static excludes for testing.
func getTestStaticExcludes(t *testing.T) []netip.Prefix {
	return getTestExcludes(t, []string{
		"192.168.1.0/24",
		"2001::/64",
	})
}

// getTestStaticExcludesOverlap returns static excludes that overlap for testing.
func getTestStaticExcludesOverlap(t *testing.T) []netip.Prefix {
	return getTestExcludes(t, []string{
		"192.168.1.0/26",
		"192.168.1.64/26",
		"192.168.1.128/26",
		"192.168.1.192/26",
		"192.168.1.0/25",
		"192.168.1.128/25",
		"192.168.1.0/24",
		"2001:2001:2001:2000::/64",
		"2001:2001:2001:2001::/64",
		"2001:2001:2001:2002::/64",
		"2001:2001:2001:2003::/64",
		"2001:2001:2001:2000::/63",
		"2001:2001:2001:2002::/63",
		"2001:2001:2001:2000::/56",
	})
}

// getTestDynamicExcludes returns dynamic excludes for testing.
func getTestDynamicExcludes(t *testing.T) []netip.Prefix {
	return getTestExcludes(t, []string{
		"192.168.1.1/32",
		"2001::1/128",
		"172.16.1.1/32",
	})
}

// TestExcludesAddStatic tests AddStatic of Excludes.
func TestExcludesAddStatic(t *testing.T) {
	e := NewExcludes(daemoncfg.NewConfig())
	excludes := getTestStaticExcludes(t)

	// test adding excludes
	for _, exclude := range excludes {
		if !e.AddStatic(exclude) {
			t.Errorf("should add exclude %s", exclude)
		}
	}

	// test adding excludes again, should not change nft commands
	for _, exclude := range excludes {
		if e.AddStatic(exclude) {
			t.Errorf("should not add exclude %s", exclude)
		}
	}

	// test adding overlapping excludes
	e = NewExcludes(daemoncfg.NewConfig())
	for _, exclude := range getTestStaticExcludesOverlap(t) {
		e.AddStatic(exclude)
	}
	for k := range e.s {
		if k != "192.168.1.0/24" && k != "2001:2001:2001:2000::/56" {
			t.Errorf("unexpected key: %s", k)
		}
	}
}

// TestExcludesAddDynamic tests AddDynamic of Excludes.
func TestExcludesAddDynamic(t *testing.T) {
	e := NewExcludes(daemoncfg.NewConfig())
	excludes := getTestDynamicExcludes(t)

	// test adding excludes
	for _, exclude := range excludes {
		if !e.AddDynamic(exclude, 300) {
			t.Errorf("should add exclude %s", exclude)
		}
	}

	// test adding excludes again, should not change nft commands
	for _, exclude := range excludes {
		if e.AddDynamic(exclude, 300) {
			t.Errorf("should not add exclude %s", exclude)
		}
	}

	// test adding excludes with existing static excludes,
	// should only add new excludes
	// TODO: fix
	e = NewExcludes(daemoncfg.NewConfig())
	for _, exclude := range getTestStaticExcludes(t) {
		e.AddStatic(exclude)
	}
	for _, exclude := range excludes {
		e.AddDynamic(exclude, 300)
	}

	// test adding invalid excludes (static as dynamic)
	e = NewExcludes(daemoncfg.NewConfig())
	for _, exclude := range getTestStaticExcludes(t) {
		if e.AddDynamic(exclude, 300) {
			t.Errorf("should not add exclude %s", exclude)
		}
	}
}

// TestExcludesRemoveStatic tests RemoveStatic of Excludes.
func TestExcludesRemove(t *testing.T) {
	e := NewExcludes(daemoncfg.NewConfig())
	excludes := getTestStaticExcludes(t)

	// test removing not existing excludes
	for _, exclude := range excludes {
		if e.RemoveStatic(exclude) {
			t.Errorf("should not remove exclude %s", exclude)
		}
	}

	// test removing static excludes
	for _, exclude := range excludes {
		e.AddStatic(exclude)
	}
	for _, exclude := range excludes {
		e.RemoveStatic(exclude)
	}

	// test removing with dynamic excludes
	for _, exclude := range excludes {
		e.AddStatic(exclude)
	}
	for _, exclude := range getTestDynamicExcludes(t) {
		e.AddDynamic(exclude, 300)
	}
	for _, exclude := range excludes {
		e.RemoveStatic(exclude)
	}
}

// TestExcludesCleanup tests cleanup of Excludes.
func TestExcludesCleanup(t *testing.T) {
	e := NewExcludes(daemoncfg.NewConfig())

	// set testing runNft function
	got := []string{}
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}

	// test without excludes
	want := []string{}
	e.cleanup()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with dynamic excludes
	for _, exclude := range getTestDynamicExcludes(t) {
		e.AddDynamic(exclude, excludesTimer)
	}

	got = []string{}
	for i := 0; i <= excludesTimer; i += excludesTimer {
		want := []string{}
		e.cleanup()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	want = []string{
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
	}
	e.cleanup()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with static excludes
	for _, exclude := range getTestStaticExcludes(t) {
		e.AddStatic(exclude)
	}
	got = []string{}
	want = []string{}
	e.cleanup()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestExcludesStartStop tests Start and Stop of Excludes.
func TestExcludesStartStop(_ *testing.T) {
	e := NewExcludes(daemoncfg.NewConfig())
	e.Start()
	e.Stop()
}

// TestNewExcludes tests NewExcludes.
func TestNewExcludes(t *testing.T) {
	conf := daemoncfg.NewConfig()
	e := NewExcludes(conf)
	if e == nil ||
		e.conf != conf ||
		e.s == nil ||
		e.d == nil ||
		e.done == nil ||
		e.closed == nil {

		t.Errorf("invalid excludes")
	}
}
