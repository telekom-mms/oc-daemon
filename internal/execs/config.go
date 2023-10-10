package execs

import "os/exec"

// default values
var (
	IP         = "ip"
	Nft        = "nft"
	Resolvectl = "resolvectl"
	Sysctl     = "sysctl"
)

// Config is executables configuration
type Config struct {
	IP         string
	Nft        string
	Resolvectl string
	Sysctl     string
}

// Valid returns whether config is valid
func (c *Config) Valid() bool {
	if c == nil ||
		c.IP == "" ||
		c.Nft == "" ||
		c.Resolvectl == "" ||
		c.Sysctl == "" {
		// invalid
		return false
	}
	return true
}

// CheckExecutables checks whether executables in config exist in the
// file system and are executable
func (c *Config) CheckExecutables() error {
	for _, f := range []string{
		c.IP, c.Nft, c.Resolvectl, c.Sysctl,
	} {
		if _, err := exec.LookPath(f); err != nil {
			return err
		}
	}
	return nil
}

// NewConfig returns a new Config
func NewConfig() *Config {
	return &Config{
		IP:         IP,
		Nft:        Nft,
		Resolvectl: Resolvectl,
		Sysctl:     Sysctl,
	}
}
