package vpnconfig

import (
	"fmt"
	"net"
	"os/exec"
	"reflect"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
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

// runLinkByname is a helper for getting a link by name
var runLinkByName = func(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

// runLinkSetMTU is a helper for setting the mtu of link
var runLinkSetMTU = func(link netlink.Link, mtu int) error {
	return netlink.LinkSetMTU(link, mtu)
}

// runLinkSetUp is a helper for setting link up
var runLinkSetUp = func(link netlink.Link) error {
	return netlink.LinkSetUp(link)
}

// runLinkSetDown is a helper for setting link down
var runLinkSetDown = func(link netlink.Link) error {
	return netlink.LinkSetDown(link)
}

// runAddrAdd is a helper for adding address to link
var runAddrAdd = func(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}

// SetupDevice sets up the vpn device with config
func (c *Config) SetupDevice() {
	// get link for device
	link, err := runLinkByName(c.Device.Name)
	if err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not find device")
		return
	}

	// set mtu on device
	if err := runLinkSetMTU(link, c.Device.MTU); err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not set mtu on device")
		return
	}

	// set device up
	if err := runLinkSetUp(link); err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not set device up")
		return
	}

	// set ipv4 and ipv6 addresses on device
	setupIP := func(ip net.IP, mask net.IPMask) {
		ipnet := &net.IPNet{
			IP:   ip,
			Mask: mask,
		}
		addr := &netlink.Addr{
			IPNet: ipnet,
		}
		if err := runAddrAdd(link, addr); err != nil {
			log.WithFields(log.Fields{
				"device": c.Device.Name,
				"ip":     ip,
			}).Error("Daemon could net set ip on device")
			return
		}

	}
	if len(c.IPv4.Address) > 0 {
		setupIP(c.IPv4.Address, c.IPv4.Netmask)
	}
	if len(c.IPv6.Address) > 0 {
		setupIP(c.IPv6.Address, c.IPv6.Netmask)
	}
}

// TeardownDevice tears down the configured vpn device
func (c *Config) TeardownDevice() {
	// get link for device
	link, err := runLinkByName(c.Device.Name)
	if err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not find device ")
		return
	}

	// set device down
	if err := runLinkSetDown(link); err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not set device down")
		return
	}

}

// runResolvctl runs the resolvectl cmd
var runResolvectl = func(cmd string) {
	log.WithField("command", cmd).Debug("Daemon executing resolvectl command")
	c := exec.Command("bash", "-c", "resolvectl "+cmd)
	if err := c.Run(); err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
		}).Error("Daemon resolvectl command execution error")
	}
}

// SetDNS applies the DNS configuration and sets dns server address;
// the server address should be the local DNS-Proxy
func (c *Config) SetDNS(server string) {
	device := c.Device.Name
	search := c.DNS.DefaultDomain

	// set dns server for device
	runResolvectl(fmt.Sprintf("dns %s %s", device, server))

	// set domains for device
	// this includes "~." to use this device for all domains
	runResolvectl(fmt.Sprintf("domain %s %s ~.", device, search))

	// set default route for device
	runResolvectl(fmt.Sprintf("default-route %s yes", device))

	// flush dns caches
	runResolvectl("flush-caches")

	// reset learnt server features
	runResolvectl("reset-server-features")
}

// UnsetDNS unsets the DNS configuration
func (c *Config) UnsetDNS() {
	device := c.Device.Name

	// undo device dns configuration
	runResolvectl(fmt.Sprintf("revert %s", device))

	// flush dns caches
	runResolvectl("flush-caches")

	// reset learnt server features
	runResolvectl("reset-server-features")
}

// New returns a new Config
func New() *Config {
	return &Config{}
}

// runCleanupCmd runs cmd for cleanups
var runCleanupCmd = func(cmd string) {
	log.WithField("command", cmd).Debug("Daemon executing vpn config cleanup command")
	c := exec.Command("bash", "-c", cmd)
	if err := c.Run(); err == nil {
		log.WithField("command", cmd).Warn("Daemon cleaned up vpn config")
	}
}

// Cleanup cleans up the configuration after a failed shutdown
func Cleanup(device string) {
	runCleanupCmd(fmt.Sprintf("resolvectl revert %s", device))
	runCleanupCmd(fmt.Sprintf("ip link delete %s", device))
}
