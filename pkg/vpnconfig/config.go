package vpnconfig

import (
	"encoding/json"
	"net"
	"reflect"

	log "github.com/sirupsen/logrus"
)

// Device is a VPN device configuration in Config
type Device struct {
	Name string
	MTU  int
}

// Address is a IPv4/IPv6 address configuration in Config
type Address struct {
	Address net.IP
	Netmask net.IPMask
}

// DNS is a DNS configuration in Config
type DNS struct {
	DefaultDomain string
	ServersIPv4   []net.IP
	ServersIPv6   []net.IP
}

// Remotes returns a map of DNS remotes from the DNS configuration that maps
// domain "." to the IPv4 and IPv6 DNS servers in the configuration including
// port number 53
func (d *DNS) Remotes() map[string][]string {
	remotes := map[string][]string{}
	for _, s := range d.ServersIPv4 {
		server := s.String() + ":53"
		remotes["."] = append(remotes["."], server)
	}
	for _, s := range d.ServersIPv6 {
		server := "[" + s.String() + "]:53"
		remotes["."] = append(remotes["."], server)
	}

	return remotes
}

// Split is a split routing configuration in Config
type Split struct {
	ExcludeIPv4 []*net.IPNet
	ExcludeIPv6 []*net.IPNet
	ExcludeDNS  []string

	ExcludeVirtualSubnetsOnlyIPv4 bool
}

// DNSExcludes returns a list of DNS-based split excludes from the
// split routing configuration. The list contains domain names including the
// trailing "."
func (s *Split) DNSExcludes() []string {
	excludes := make([]string, len(s.ExcludeDNS))
	for i, e := range s.ExcludeDNS {
		excludes[i] = e + "."
	}

	return excludes
}

// Flags are other configuration settings in Config
type Flags struct {
	DisableAlwaysOnVPN bool
}

// Config is a VPN configuration
type Config struct {
	Gateway net.IP
	PID     int
	Timeout int
	Device  Device
	IPv4    Address
	IPv6    Address
	DNS     DNS
	Split   Split
	Flags   Flags
}

// Empty returns if the config is empty
func (c *Config) Empty() bool {
	empty := New()
	return c.Equal(empty)
}

// Equal returns if the config and other are equal
func (c *Config) Equal(other *Config) bool {
	return reflect.DeepEqual(c, other)
}

// Valid returns if the config is valid
func (c *Config) Valid() bool {
	// an empty config is valid
	if c.Empty() {
		return true
	}

	// check config entries
	for i, invalid := range []bool{
		c.Gateway == nil,
		c.Device.Name == "",
		len(c.Device.Name) > 15,
		c.Device.MTU < 68,
		c.Device.MTU > 16384,
		len(c.IPv4.Address) == 0 && len(c.IPv6.Address) == 0,
		len(c.IPv4.Netmask) == 0 && len(c.IPv6.Netmask) == 0,
		len(c.DNS.ServersIPv4) == 0 && len(c.DNS.ServersIPv6) == 0,
	} {
		if invalid {
			log.WithField("check", i).Error("VPNConfig is invalid config")
			return false
		}
	}
	// TODO: check more?

	return true
}

// JSON returns the configuration as JSON
func (c *Config) JSON() ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// New returns a new Config
func New() *Config {
	return &Config{}
}
