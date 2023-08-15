package dnsmon

import "path/filepath"

var (
	// ETCResolvConf is the default resolv.conf in /etc
	ETCResolvConf = "/etc/resolv.conf"

	// StubResolvConf is the stub resolv.conf of systemd-resolved
	StubResolvConf = "/run/systemd/resolve/stub-resolv.conf"

	// SystemdResolvConf is the systemd resolv.conf
	SystemdResolvConf = "/run/systemd/resolve/resolv.conf"
)

// Config is a DNSMon configuration
type Config struct {
	ETCResolvConf     string
	StubResolvConf    string
	SystemdResolvConf string
}

// resolvConfDirs returns a list of resolv.conf directories watched by DNSMon
func (c *Config) resolvConfDirs() []string {
	// use map to remove duplicate directories
	m := make(map[string]struct{})
	for _, conf := range []string{
		c.ETCResolvConf,
		c.StubResolvConf,
		c.SystemdResolvConf,
	} {
		dir := filepath.Dir(conf)
		m[dir] = struct{}{}
	}

	// convert to list
	dirs := []string{}
	for k := range m {
		dirs = append(dirs, k)

	}

	return dirs
}

// NewConfig returns a new DNSMon config
func NewConfig() *Config {
	return &Config{
		ETCResolvConf:     ETCResolvConf,
		StubResolvConf:    StubResolvConf,
		SystemdResolvConf: SystemdResolvConf,
	}
}
