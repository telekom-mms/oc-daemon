package trafpol

import (
	"time"
)

var (
	// AllowedHosts is the default list of allowed hosts, this is
	// initialized with hosts for captive portal detection, e.g.,
	// used by browsers.
	AllowedHosts = []string{
		"connectivity-check.ubuntu.com", // ubuntu
		"detectportal.firefox.com",      // firefox
		"www.gstatic.com",               // chrome
		"clients3.google.com",           // chromium
		"nmcheck.gnome.org",             // gnome
	}

	// PortalPorts are the default ports that are allowed to register on a
	// captive portal.
	PortalPorts = []uint16{
		80,
		443,
	}

	// ResolveTimeout is the timeout for dns lookups.
	ResolveTimeout = 2 * time.Second

	// ResolveTries is the number of tries for dns lookups.
	ResolveTries = 3

	// ResolveTriesSleep is the sleep time between retries.
	ResolveTriesSleep = time.Second

	// ResolveTimer is the time for periodic resolve update checks,
	// should be higher than tries * (timeout + sleep).
	ResolveTimer = 30 * time.Second

	// ResolveTTL is the lifetime of resolved entries.
	ResolveTTL = 300 * time.Second
)

// Config is a TrafPol configuration.
type Config struct {
	AllowedHosts []string
	PortalPorts  []uint16
	FirewallMark string `json:"-"`

	ResolveTimeout    time.Duration
	ResolveTries      int
	ResolveTriesSleep time.Duration
	ResolveTimer      time.Duration
	ResolveTTL        time.Duration
}

// Valid returns whether the TrafPol configuration is valid.
func (c *Config) Valid() bool {
	if c == nil ||
		len(c.PortalPorts) == 0 ||
		c.ResolveTimeout < 0 ||
		c.ResolveTries < 1 ||
		c.ResolveTriesSleep < 0 ||
		c.ResolveTimer < 0 ||
		c.ResolveTTL < 0 {

		return false
	}
	return true
}

// NewConfig returns a new TrafPol configuration.
func NewConfig() *Config {
	return &Config{
		AllowedHosts: append(AllowedHosts[:0:0], AllowedHosts...),
		PortalPorts:  append(PortalPorts[:0:0], PortalPorts...),

		ResolveTimeout:    ResolveTimeout,
		ResolveTries:      ResolveTries,
		ResolveTriesSleep: ResolveTriesSleep,
		ResolveTimer:      ResolveTimer,
		ResolveTTL:        ResolveTTL,
	}
}
