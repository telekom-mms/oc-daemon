package daemon

import (
	"encoding/json"

	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// VPNConfigUpdate is a VPN configuration update.
type VPNConfigUpdate struct {
	Reason string
	Config *vpnconfig.Config
}

// Valid returns whether the config update is valid.
func (c *VPNConfigUpdate) Valid() bool {
	switch c.Reason {
	case "disconnect":
		// config must be nil
		if c.Config != nil {
			return false
		}
	case "connect":
		// config must be valid
		if c.Config == nil || !c.Config.Valid() {
			return false
		}
	default:
		return false
	}

	return true
}

// JSON returns the Config as JSON.
func (c *VPNConfigUpdate) JSON() ([]byte, error) {
	return json.Marshal(c)
}

// VPNConfigUpdateFromJSON parses and returns the VPNConfigUpdate in b.
func VPNConfigUpdateFromJSON(b []byte) (*VPNConfigUpdate, error) {
	c := NewVPNConfigUpdate()
	err := json.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewVPNConfigUpdate returns a new VPNConfigUpdate.
func NewVPNConfigUpdate() *VPNConfigUpdate {
	return &VPNConfigUpdate{}
}
