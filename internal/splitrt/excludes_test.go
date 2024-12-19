package splitrt

import (
	"net/netip"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
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
	statics := getTestStaticExcludes(t)
	e = NewExcludes(daemoncfg.NewConfig())
	for _, exclude := range statics {
		if !e.AddStatic(exclude) {
			t.Errorf("should add exclude %s", exclude)
		}
	}
	for _, exclude := range excludes {
		add := true
		for _, static := range statics {
			if static.Overlaps(exclude) {
				add = false
			}
		}
		if add && !e.AddDynamic(exclude, 300) {
			t.Errorf("should add exclude %s", exclude)
		}
		if !add && e.AddDynamic(exclude, 300) {
			t.Errorf("should not add exclude %s", exclude)
		}
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
		if !e.AddStatic(exclude) {
			t.Fatalf("should add exclude %s", exclude)
		}
	}
	for _, exclude := range excludes {
		if !e.RemoveStatic(exclude) {
			t.Errorf("should remove exclude %s", exclude)
		}
	}

	// test removing with dynamic excludes
	for _, exclude := range excludes {
		if !e.AddStatic(exclude) {
			t.Fatalf("should add exclude %s", exclude)
		}
	}
	for _, exclude := range getTestDynamicExcludes(t) {
		e.AddDynamic(exclude, 300)
	}
	for _, exclude := range excludes {
		if !e.RemoveStatic(exclude) {
			t.Errorf("should remove exclude %s", exclude)
		}
	}
}

// TestExcludesCleanup tests cleanup of Excludes.
func TestExcludesCleanup(t *testing.T) {
	e := NewExcludes(daemoncfg.NewConfig())

	// test without excludes
	if e.cleanup() {
		t.Error("should not remove excludes")
	}

	// test with dynamic excludes
	for _, exclude := range getTestDynamicExcludes(t) {
		if !e.AddDynamic(exclude, excludesTimer) {
			t.Fatalf("should add exclude %s", exclude)
		}
	}

	for i := 0; i <= excludesTimer; i += excludesTimer {
		if e.cleanup() {
			t.Error("should not remove excludes")
		}
	}
	if !e.cleanup() {
		t.Error("should remove excludes")
	}

	// test with static excludes
	for _, exclude := range getTestStaticExcludes(t) {
		if !e.AddStatic(exclude) {
			t.Fatalf("should add exclude %s", exclude)
		}
	}
	if e.cleanup() {
		t.Error("should not remove excludes")
	}

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
