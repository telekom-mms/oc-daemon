package trafpol

// AllowDevs contains allowed devices
type AllowDevs struct {
	m map[string]string
}

// Add adds device to the allowed devices
func (a *AllowDevs) Add(device string) {
	if a.m[device] != device {
		a.m[device] = device
		addAllowedDevice(device)
	}
}

// Remove removes device from the allowed devices
func (a *AllowDevs) Remove(device string) {
	if a.m[device] == device {
		delete(a.m, device)
		removeAllowedDevice(device)
	}
}

// NewAllowDevs returns new allowDevs
func NewAllowDevs() *AllowDevs {
	return &AllowDevs{
		m: make(map[string]string),
	}
}
