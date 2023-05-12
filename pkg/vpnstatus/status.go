package vpnstatus

import (
	"encoding/json"

	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnconfig"
)

// Status is a VPN status
type Status struct {
	TrustedNetwork bool
	Running        bool
	Connected      bool
	Config         *vpnconfig.Config
}

// JSON returns the Status as JSON
func (s *Status) JSON() ([]byte, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// NewFromJSON parses and returns the Status in b
func NewFromJSON(b []byte) (*Status, error) {
	s := New()
	err := json.Unmarshal(b, s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// New returns a new Status
func New() *Status {
	return &Status{}
}
