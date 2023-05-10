package vpnstatus

import (
	"encoding/json"

	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnconfig"
)

// TrustedNetwork is the current trusted network state
type TrustedNetwork uint32

// TrustedNetwork states
const (
	TrustedNetworkUnknown TrustedNetwork = iota
	TrustedNetworkNotTrusted
	TrustedNetworkTrusted
)

// Trusted returns whether TrustedNetwork is state "trusted"
func (t TrustedNetwork) Trusted() bool {
	return t == TrustedNetworkTrusted
}

// String returns t as string
func (t TrustedNetwork) String() string {
	switch t {
	case TrustedNetworkUnknown:
		return "unknown"
	case TrustedNetworkNotTrusted:
		return "not trusted"
	case TrustedNetworkTrusted:
		return "trusted"
	}
	return ""
}

// Status is a VPN status
type Status struct {
	TrustedNetwork TrustedNetwork
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
