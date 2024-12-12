package trafpol

// AllowDevs contains allowed devices.
type AllowDevs struct {
	m map[string]string
}

// Add adds device to the allowed devices.
func (a *AllowDevs) Add(device string) bool {
	if a.m[device] != device {
		a.m[device] = device
		return true
	}
	return false
}

// Remove removes device from the allowed devices.
func (a *AllowDevs) Remove(device string) bool {
	if a.m[device] == device {
		delete(a.m, device)
		return true
	}
	return false
}

// List returns a slice of all allowed devices.
func (a *AllowDevs) List() []string {
	var l []string
	for _, v := range a.m {
		l = append(l, v)
	}
	return l
}

// NewAllowDevs returns new allowDevs.
func NewAllowDevs() *AllowDevs {
	return &AllowDevs{
		m: make(map[string]string),
	}
}
