package ocrunner

import "strconv"

var (
	// OpenConnect is the default openconnect executable
	OpenConnect = "openconnect"

	// XMLProfile is the default AnyConnect Profile
	XMLProfile = "/var/lib/oc-daemon/profile.xml"

	// VPNCScript is the default vpnc-script
	VPNCScript = "/usr/bin/oc-daemon-vpncscript"

	// VPNDevice is the default vpn network device name
	VPNDevice = "oc-daemon-tun0"

	// PIDFile is the default file path of the PID file for openconnect
	PIDFile = "/run/oc-daemon/openconnect.pid"

	// PIDOwner is the default owner of the PID file
	PIDOwner = ""

	// PIDGroup is the default group of the PID file
	PIDGroup = ""

	// PIDPermissions are the default file permissions of the PID file
	PIDPermissions = "0600"

	// NoProxy specifies whether the no proxy flag is set in openconnect
	NoProxy = true

	// ExtraEnv are extra environment variables used by openconnect
	ExtraEnv = []string{}

	// ExtraArgs are extra command line arguments used by openconnect
	ExtraArgs = []string{}
)

// Config is the configuration for an openconnect connection runner
type Config struct {
	OpenConnect string

	XMLProfile string
	VPNCScript string
	VPNDevice  string

	PIDFile        string
	PIDOwner       string
	PIDGroup       string
	PIDPermissions string

	NoProxy   bool
	ExtraEnv  []string
	ExtraArgs []string
}

// Valid returns whether the openconnect configuration is valid
func (c *Config) Valid() bool {
	if c == nil ||
		c.OpenConnect == "" ||
		c.XMLProfile == "" ||
		c.VPNCScript == "" ||
		c.VPNDevice == "" ||
		c.PIDFile == "" ||
		c.PIDPermissions == "" {

		return false
	}
	if c.PIDPermissions != "" {
		perm, err := strconv.ParseUint(c.PIDPermissions, 8, 32)
		if err != nil {
			return false
		}
		if perm > 0777 {
			return false
		}
	}
	return true
}

// NewConfig returns a new configuration for an openconnect connection runner
func NewConfig() *Config {
	return &Config{
		OpenConnect: OpenConnect,

		XMLProfile: XMLProfile,
		VPNCScript: VPNCScript,
		VPNDevice:  VPNDevice,

		PIDFile:        PIDFile,
		PIDOwner:       PIDOwner,
		PIDGroup:       PIDGroup,
		PIDPermissions: PIDPermissions,

		NoProxy:   NoProxy,
		ExtraEnv:  ExtraEnv,
		ExtraArgs: ExtraArgs,
	}
}
