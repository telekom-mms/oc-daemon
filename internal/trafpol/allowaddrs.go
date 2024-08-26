package trafpol

import "net/netip"

// AllowAddrs are allowed addresses.
type AllowAddrs struct {
	m map[string]netip.Prefix
}

// Add adds prefix to the allowed addresses.
func (a *AllowAddrs) Add(prefix netip.Prefix) bool {
	s := prefix.String()
	if _, ok := a.m[s]; ok {
		// ip already in allowed addrs
		return false
	}
	a.m[s] = prefix
	return true
}

// Remove removes prefix from the allowed addresses.
func (a *AllowAddrs) Remove(prefix netip.Prefix) bool {
	s := prefix.String()
	if _, ok := a.m[s]; !ok {
		// ip not in allowed addrs
		return false
	}
	delete(a.m, s)
	return true
}

// List returns a list of all allowed addresses.
func (a *AllowAddrs) List() []netip.Prefix {
	var prefixes []netip.Prefix
	for _, p := range a.m {
		prefixes = append(prefixes, p)
	}
	return prefixes
}

// NewAllowAddrs returns new AllowAddrs.
func NewAllowAddrs() *AllowAddrs {
	return &AllowAddrs{
		m: make(map[string]netip.Prefix),
	}
}
