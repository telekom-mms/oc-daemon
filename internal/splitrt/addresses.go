package splitrt

import (
	"net"

	"github.com/telekom-mms/oc-daemon/internal/addrmon"
)

// Addresses is a set of addresses.
type Addresses struct {
	m map[int][]*addrmon.Update
}

// contains checks if address info in addr is in addresses.
func (a *Addresses) contains(addr *addrmon.Update) bool {
	if a.m[addr.Index] == nil {
		return false
	}
	for _, x := range a.m[addr.Index] {
		if x.Address.String() == addr.Address.String() {
			return true
		}
	}
	return false
}

// Add adds address info in addr to addresses.
func (a *Addresses) Add(addr *addrmon.Update) {
	if a.contains(addr) {
		return
	}
	a.m[addr.Index] = append(a.m[addr.Index], addr)
}

// Remove removes address info in addr from addresses.
func (a *Addresses) Remove(addr *addrmon.Update) {
	if !a.contains(addr) {
		return
	}

	old := a.m[addr.Index]
	removed := []*addrmon.Update{}
	for _, x := range old {
		if x.Address.String() == addr.Address.String() {
			// skip/remove element
			continue
		}
		removed = append(removed, x)
	}
	a.m[addr.Index] = removed
}

// Get returns the addresses of the device identified by index.
func (a *Addresses) Get(index int) (addrs []*net.IPNet) {
	for _, x := range a.m[index] {
		addrs = append(addrs, &x.Address)
	}
	return
}

// NewAddresses returns new Addresses.
func NewAddresses() *Addresses {
	return &Addresses{
		m: make(map[int][]*addrmon.Update),
	}
}
