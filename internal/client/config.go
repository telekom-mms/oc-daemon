package client

import (
	"encoding/json"
	"os"
)

// ClientConfig is a configuration for the OC client
type ClientConfig struct {
	ClientCertificate string
	ClientKey         string
	CACertificate     string
	VPNServer         string
	User              string
	Password          string
}

// empty returns if the config is empty
func (c *ClientConfig) empty() bool {
	if c == nil {
		return true
	}

	if c.ClientCertificate == "" &&
		c.ClientKey == "" &&
		c.CACertificate == "" &&
		c.VPNServer == "" &&
		c.User == "" &&
		c.Password == "" {
		// empty
		return true
	}

	return false
}

// save saves the config to file
func (c *ClientConfig) save(file string) {
	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return
	}
	if err := os.WriteFile(file, b, 0600); err != nil {
		return
	}
}

// loadClientConfig loads a ClientConfig from file
func loadClientConfig(file string) *ClientConfig {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	conf := newClientConfig()
	if err := json.Unmarshal(b, conf); err != nil {
		return nil
	}

	return conf
}

// newClientConfig returns a new ClientConfig
func newClientConfig() *ClientConfig {
	return &ClientConfig{}
}
