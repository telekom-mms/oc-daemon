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
}

// empty returns if the config is empty
func (o *ClientConfig) empty() bool {
	if o == nil {
		return true
	}

	if o.ClientCertificate == "" &&
		o.ClientKey == "" &&
		o.CACertificate == "" &&
		o.VPNServer == "" &&
		o.User == "" {
		// empty
		return true
	}

	return false
}

// save saves the config to file
func (o *ClientConfig) save(file string) {
	b, err := json.MarshalIndent(o, "", "    ")
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
