package trafpol

import "net/netip"

// AllowNames are allowed DNS names.
type AllowNames struct {
	m map[string][]netip.Addr
}

// Add adds and updates the allowed name and its IP addresses.
func (a *AllowNames) Add(name string, addrs []netip.Addr) {
	a.m[name] = addrs
}

// GetAll returns all allowed names with their IP addresses.
func (a *AllowNames) GetAll() map[string][]netip.Addr {
	names := make(map[string][]netip.Addr)
	for k, v := range a.m {
		names[k] = append(names[k], v...)

	}
	return names
}

// NewAllowNames returns new AllowNames.
func NewAllowNames() *AllowNames {
	return &AllowNames{
		m: make(map[string][]netip.Addr),
	}
}
