package cpd

import "time"

var (
	// Host is the host address used for probing
	Host = "connectivity-check.ubuntu.com"

	// HTTPTimeout is the timeout for http requests in seconds
	HTTPTimeout = 5 * time.Second

	// ProbeCount is the number of probes to run
	ProbeCount = 3

	// ProbeWait is the time between probes
	ProbeWait = time.Second

	// ProbeTimer is the probe timer in case of no detected portal
	// in seconds
	ProbeTimer = 300 * time.Second

	// ProbeTimerDetected is the probe timer in case of a detected portal
	// in seconds
	ProbeTimerDetected = 15 * time.Second
)

// Config is the configuration of the captive portal detection
type Config struct {
	Host               string
	HTTPTimeout        time.Duration
	ProbeCount         int
	ProbeWait          time.Duration
	ProbeTimer         time.Duration
	ProbeTimerDetected time.Duration
}

// Valid returns whether the captive portal detection configuration is valid
func (c *Config) Valid() bool {
	if c == nil ||
		c.Host == "" ||
		c.HTTPTimeout <= 0 ||
		c.ProbeCount <= 0 ||
		c.ProbeWait <= 0 ||
		c.ProbeTimer <= 0 ||
		c.ProbeTimerDetected <= 0 {

		return false
	}
	return true
}

// NewConfig returns a new default configuration for captive portal detection
func NewConfig() *Config {
	return &Config{
		Host:               Host,
		HTTPTimeout:        HTTPTimeout,
		ProbeCount:         ProbeCount,
		ProbeWait:          ProbeWait,
		ProbeTimer:         ProbeTimer,
		ProbeTimerDetected: ProbeTimerDetected,
	}
}
