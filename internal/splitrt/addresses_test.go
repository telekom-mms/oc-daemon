package splitrt

import (
	"net/netip"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/addrmon"
)

// getTestAddrMonUpdate returns an AddrMon update for testing.
func getTestAddrMonUpdate(t *testing.T, addr string) *addrmon.Update {
	prefix, err := netip.ParsePrefix(addr)
	if err != nil {
		t.Fatal(err)
	}

	return &addrmon.Update{
		Add:     true,
		Address: prefix,
		Index:   1,
	}
}

// TestAddressesAdd tests Add of Addresses.
func TestAddressesAdd(t *testing.T) {
	a := NewAddresses()
	update := getTestAddrMonUpdate(t, "192.168.1.0/24")

	// test adding
	a.Add(update)
	if !a.contains(update) {
		t.Errorf("got false, want true")
	}
}

// TestAddressesRemove tests Remove of Addresses.
func TestAddressesRemove(t *testing.T) {
	a := NewAddresses()
	updates := []*addrmon.Update{
		getTestAddrMonUpdate(t, "192.168.1.0/24"),
		getTestAddrMonUpdate(t, "192.168.2.0/24"),
		getTestAddrMonUpdate(t, "192.168.3.0/24"),
	}

	// add elements
	for _, update := range updates {
		a.Add(update)
		if !a.contains(update) {
			t.Errorf("got false, want true")
		}
	}

	// test removing element
	for _, update := range updates {
		a.Remove(update)
		if a.contains(update) {
			t.Errorf("got true, want false")
		}
	}

	// test removing again/not existing entries
	for _, update := range updates {
		a.Remove(update)
	}
}

// TestAddressesGet tests Get of Addresses.
func TestAddressesGet(t *testing.T) {
	a := NewAddresses()
	update1 := getTestAddrMonUpdate(t, "192.168.1.0/24")
	update2 := getTestAddrMonUpdate(t, "192.168.2.0/24")

	// get empty
	var want []netip.Prefix
	got := a.Get(1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// get with one address
	a.Add(update1)
	want = []netip.Prefix{
		update1.Address,
	}
	got = a.Get(1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// get with double add
	a.Add(update1)
	got = a.Get(1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// get with multiple addresses
	a.Add(update2)
	want = []netip.Prefix{
		update1.Address,
		update2.Address,
	}
	got = a.Get(1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewAddresses tests NewAddresses.
func TestNewAddresses(t *testing.T) {
	a := NewAddresses()
	if a.m == nil {
		t.Errorf("got nil, want != nil")
	}
}
