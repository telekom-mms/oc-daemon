package trafpol

import "context"

// AllowDevs contains allowed devices.
type AllowDevs struct {
	m map[string]string
}

// Add adds device to the allowed devices.
func (a *AllowDevs) Add(ctx context.Context, device string) {
	if a.m[device] != device {
		a.m[device] = device
		addAllowedDevice(ctx, device)
	}
}

// Remove removes device from the allowed devices.
func (a *AllowDevs) Remove(ctx context.Context, device string) {
	if a.m[device] == device {
		delete(a.m, device)
		removeAllowedDevice(ctx, device)
	}
}

// NewAllowDevs returns new allowDevs.
func NewAllowDevs() *AllowDevs {
	return &AllowDevs{
		m: make(map[string]string),
	}
}
