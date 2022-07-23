package vpnconfig

import "encoding/json"

// ConfigUpdate is a VPN configuration update
type ConfigUpdate struct {
	Reason string
	Token  string
	Config *Config
}

// Valid returns if the config update is valid
func (c *ConfigUpdate) Valid() bool {
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
func (c *ConfigUpdate) JSON() ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// UpdateFromJSON parses and returns the ConfigUpdate in b
func UpdateFromJSON(b []byte) (*ConfigUpdate, error) {
	c := NewUpdate()
	err := json.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewUpdate returns a new ConfigUpdate
func NewUpdate() *ConfigUpdate {
	return &ConfigUpdate{}
}
