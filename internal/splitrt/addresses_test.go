package splitrt

import (
	"log"
	"net"
	"reflect"
	"testing"

	"github.com/T-Systems-MMS/oc-daemon/internal/addrmon"
)

// getTestAddrMonUpdate returns an AddrMon update for testing
func getTestAddrMonUpdate(addr string) *addrmon.Update {
	_, ipnet, err := net.ParseCIDR(addr)
	if err != nil {
		log.Fatal(err)
	}

	return &addrmon.Update{
		Add:     true,
		Address: *ipnet,
		Index:   1,
	}
}

// TestAddressesAdd tests Add of Addresses
func TestAddressesAdd(t *testing.T) {
	a := NewAddresses()
	update := getTestAddrMonUpdate("192.168.1.0/24")

	// test adding
	a.Add(update)
	if !a.contains(update) {
		t.Errorf("got false, want true")
	}
}

// TestAddressesRemove tests Remove of Addresses
func TestAddressesRemove(t *testing.T) {
	a := NewAddresses()
	update := getTestAddrMonUpdate("192.168.1.0/24")

	// add element
	a.Add(update)
	if !a.contains(update) {
		t.Errorf("got false, want true")
	}

	// test removing element
	a.Remove(update)
	if a.contains(update) {
		t.Errorf("got true, want false")
	}
}

// TestAddressesGet tests Get of Addresses
func TestAddressesGet(t *testing.T) {
	a := NewAddresses()
	update1 := getTestAddrMonUpdate("192.168.1.0/24")
	update2 := getTestAddrMonUpdate("192.168.2.0/24")

	// get empty
	var want []*net.IPNet
	got := a.Get(1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// get with one address
	a.Add(update1)
	want = []*net.IPNet{
		&update1.Address,
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
	want = []*net.IPNet{
		&update1.Address,
		&update2.Address,
	}
	got = a.Get(1)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewAddresses tests NewAddresses
func TestNewAddresses(t *testing.T) {
	a := NewAddresses()
	if a.m == nil {
		t.Errorf("got nil, want != nil")
	}
}
