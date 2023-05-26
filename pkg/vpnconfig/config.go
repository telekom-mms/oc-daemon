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

// Copy returns a copy of device
func (d *Device) Copy() Device {
	return Device{
		Name: d.Name,
		MTU:  d.MTU,
	}
}

// Address is a IPv4/IPv6 address configuration in Config
type Address struct {
	Address net.IP
	Netmask net.IPMask
}

// Copy returns a copy of address
func (a *Address) Copy() Address {
	return Address{
		Address: append(a.Address[:0:0], a.Address...),
		Netmask: append(a.Netmask[:0:0], a.Netmask...),
	}
}

// DNS is a DNS configuration in Config
type DNS struct {
	DefaultDomain string
	ServersIPv4   []net.IP
	ServersIPv6   []net.IP
}

// Copy returns a copy of DNS
func (d *DNS) Copy() DNS {
	serversIPv4 := []net.IP{}
	if d.ServersIPv4 == nil {
		serversIPv4 = nil
	}
	for _, s := range d.ServersIPv4 {
		ip := append(s[:0:0], s...)
		serversIPv4 = append(serversIPv4, ip)
	}

	serversIPv6 := []net.IP{}
	if d.ServersIPv6 == nil {
		serversIPv6 = nil
	}
	for _, s := range d.ServersIPv6 {
		ip := append(s[:0:0], s...)
		serversIPv6 = append(serversIPv6, ip)
	}

	return DNS{
		DefaultDomain: d.DefaultDomain,
		ServersIPv4:   serversIPv4,
		ServersIPv6:   serversIPv6,
	}
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

// Copy returns a copy of split
func (s *Split) Copy() Split {
	excludeIPv4 := []*net.IPNet{}
	if s.ExcludeIPv4 == nil {
		excludeIPv4 = nil
	}
	for _, e := range s.ExcludeIPv4 {
		ipnet := &net.IPNet{
			IP:   append(e.IP[:0:0], e.IP...),
			Mask: append(e.Mask[:0:0], e.Mask...),
		}
		excludeIPv4 = append(excludeIPv4, ipnet)
	}

	excludeIPv6 := []*net.IPNet{}
	if s.ExcludeIPv6 == nil {
		excludeIPv6 = nil
	}
	for _, e := range s.ExcludeIPv6 {
		ipnet := &net.IPNet{
			IP:   append(e.IP[:0:0], e.IP...),
			Mask: append(e.Mask[:0:0], e.Mask...),
		}
		excludeIPv6 = append(excludeIPv6, ipnet)
	}

	return Split{
		ExcludeIPv4: excludeIPv4,
		ExcludeIPv6: excludeIPv6,
		ExcludeDNS:  append(s.ExcludeDNS[:0:0], s.ExcludeDNS...),

		ExcludeVirtualSubnetsOnlyIPv4: s.ExcludeVirtualSubnetsOnlyIPv4,
	}
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

// Copy returns a copy of flags
func (f *Flags) Copy() Flags {
	return Flags{
		DisableAlwaysOnVPN: f.DisableAlwaysOnVPN,
	}
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

// Copy returns a new copy of config
func (c *Config) Copy() *Config {
	if c == nil {
		return nil
	}
	return &Config{
		Gateway: append([]byte{}, c.Gateway...),
		PID:     c.PID,
		Timeout: c.Timeout,
		Device:  c.Device.Copy(),
		IPv4:    c.IPv4.Copy(),
		IPv6:    c.IPv6.Copy(),
		DNS:     c.DNS.Copy(),
		Split:   c.Split.Copy(),
		Flags:   c.Flags.Copy(),
	}
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

// NewFromJSON returns a new config parsed from the json in b
func NewFromJSON(b []byte) (*Config, error) {
	c := New()
	if err := json.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}
