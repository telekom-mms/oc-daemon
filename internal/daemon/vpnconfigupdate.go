package daemon

import (
	"encoding/json"

	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnconfig"
)

// VPNConfigUpdate is a VPN configuration update
type VPNConfigUpdate struct {
	Reason string
	Token  string
	Config *vpnconfig.Config
}

// Valid returns if the config update is valid
func (c *VPNConfigUpdate) Valid() bool {
	switch c.Reason {
	case "disconnect":
		// token must be valid and config nil
		if c.Token == "" || c.Config != nil {
			return false
		}
	case "connect":
		// token and config must be valid
		if c.Token == "" || c.Config == nil {
			return false
		}
		if !c.Config.Valid() {
			return false
		}
	default:
		return false
	}

	return true
}

// JSON returns the Config as JSON
func (c *VPNConfigUpdate) JSON() ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// VPNConfigUpdateFromJSON parses and returns the VPNConfigUpdate in b
func VPNConfigUpdateFromJSON(b []byte) (*VPNConfigUpdate, error) {
	c := NewVPNConfigUpdate()
	err := json.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewVPNConfigUpdate returns a new VPNConfigUpdate
func NewVPNConfigUpdate() *VPNConfigUpdate {
	return &VPNConfigUpdate{}
}
