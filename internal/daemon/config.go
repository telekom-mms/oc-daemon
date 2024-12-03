package daemon

import (
	"encoding/json"
	"net/netip"
	"os"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/tnd/pkg/tnd"
)

var (
	// configDir is the directory for the configuration.
	configDir = "/var/lib/oc-daemon"

	// ConfigFile is the default config file.
	ConfigFile = configDir + "/oc-daemon.json"

	// DefaultDNSServer is the default DNS server address, i.e., listen
	// address of systemd-resolved.
	DefaultDNSServer = "127.0.0.53:53"
)

// Config is an OC-Daemon configuration.
// TODO: move this into separate package
// TODO: use it in cmdtmpl?
// TODO: add runtime config login with json:"-"?
// TODO: add runtime config VPNConfig with json:"-"?
type Config struct {
	Config  string `json:"-"`
	Verbose bool

	SocketServer    *daemoncfg.SocketServer
	CPD             *daemoncfg.CPD
	DNSProxy        *daemoncfg.DNSProxy
	OpenConnect     *daemoncfg.OpenConnect
	Executables     *daemoncfg.Executables
	SplitRouting    *daemoncfg.SplitRouting
	TrafficPolicing *daemoncfg.TrafficPolicing
	TND             *tnd.Config
}

// String returns the configuration as string.
func (c *Config) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// Valid returns whether config is valid.
func (c *Config) Valid() bool {
	if c == nil ||
		!c.SocketServer.Valid() ||
		!c.CPD.Valid() ||
		!c.DNSProxy.Valid() ||
		!c.OpenConnect.Valid() ||
		!c.Executables.Valid() ||
		!c.SplitRouting.Valid() ||
		!c.TrafficPolicing.Valid() ||
		!c.TND.Valid() {
		// invalid
		return false
	}
	return true
}

// Load loads the configuration from the config file.
func (c *Config) Load() error {
	// read file contents
	file, err := os.ReadFile(c.Config)
	if err != nil {
		return err
	}

	// parse config
	if err := json.Unmarshal(file, c); err != nil {
		return err
	}

	return nil
}

// GetConfig converts Config to daemoncfg.Config
func (c *Config) GetConfig() *daemoncfg.Config {
	conf := &daemoncfg.Config{
		Verbose: c.Verbose,

		// Socket Server
		SocketServer: c.SocketServer,

		// CPD
		CPD: c.CPD,

		// DNS Proxy
		DNSProxy: c.DNSProxy,

		// OpenConnect
		OpenConnect: c.OpenConnect,

		// Executables
		Executables: c.Executables,

		// SplitRouting
		SplitRouting: c.SplitRouting,

		// TrafficPolicing
		TrafficPolicing: &daemoncfg.TrafficPolicing{
			AllowedHosts:      c.TrafficPolicing.AllowedHosts,
			PortalPorts:       c.TrafficPolicing.PortalPorts,
			FirewallMark:      c.TrafficPolicing.FirewallMark,
			ResolveTimeout:    c.TrafficPolicing.ResolveTimeout,
			ResolveTries:      c.TrafficPolicing.ResolveTries,
			ResolveTriesSleep: c.TrafficPolicing.ResolveTriesSleep,
			ResolveTimer:      c.TrafficPolicing.ResolveTimer,
			ResolveTTL:        c.TrafficPolicing.ResolveTTL,
		},

		// TND
		TND: c.TND,
	}
	return conf
}

func getVPNConfig(vpnconf *vpnconfig.Config) *daemoncfg.VPNConfig {
	// convert gateway
	gateway := netip.Addr{}
	if g, ok := netip.AddrFromSlice(vpnconf.Gateway); ok {
		gateway = g
	}

	// convert ipv4 address
	pre4 := netip.Prefix{}
	if ipv4, ok := netip.AddrFromSlice(vpnconf.IPv4.Address.To4()); ok {
		pre4len, _ := vpnconf.IPv4.Netmask.Size()
		pre4 = netip.PrefixFrom(ipv4, pre4len)
	}

	// convert ipv6 address
	pre6 := netip.Prefix{}
	if ipv6, ok := netip.AddrFromSlice(vpnconf.IPv6.Address); ok {
		pre6len, _ := vpnconf.IPv6.Netmask.Size()
		pre6 = netip.PrefixFrom(ipv6, pre6len)
	}

	// convert ipv4 dns servers
	var dns4 []netip.Addr
	for _, a := range vpnconf.DNS.ServersIPv4 {
		if d, ok := netip.AddrFromSlice(a.To4()); ok {
			dns4 = append(dns4, d)
		}
	}

	// convert ipv6 dns servers
	var dns6 []netip.Addr
	for _, a := range vpnconf.DNS.ServersIPv6 {
		if d, ok := netip.AddrFromSlice(a); ok {
			dns6 = append(dns6, d)
		}
	}

	// convert ipv4 excludes
	var excludes4 []netip.Prefix
	for _, a := range vpnconf.Split.ExcludeIPv4 {
		if ipv4, ok := netip.AddrFromSlice(a.IP.To4()); ok {
			pre4len, _ := a.Mask.Size()
			pre4 := netip.PrefixFrom(ipv4, pre4len)
			excludes4 = append(excludes4, pre4)
		}
	}

	// convert ipv6 excludes
	var excludes6 []netip.Prefix
	for _, a := range vpnconf.Split.ExcludeIPv6 {
		if ipv6, ok := netip.AddrFromSlice(a.IP); ok {
			pre6len, _ := a.Mask.Size()
			pre6 = netip.PrefixFrom(ipv6, pre6len)
			excludes6 = append(excludes6, pre6)
		}
	}

	return &daemoncfg.VPNConfig{
		Gateway: gateway,
		PID:     vpnconf.PID,
		Timeout: vpnconf.Timeout,
		Device: daemoncfg.VPNDevice{
			Name: vpnconf.Device.Name,
			MTU:  vpnconf.Device.MTU,
		},
		IPv4: pre4,
		IPv6: pre6,
		DNS: daemoncfg.VPNDNS{
			DefaultDomain: vpnconf.DNS.DefaultDomain,
			ServersIPv4:   dns4,
			ServersIPv6:   dns6,
		},
		Split: daemoncfg.VPNSplit{
			ExcludeIPv4: excludes4,
			ExcludeIPv6: excludes6,
			ExcludeDNS:  vpnconf.Split.ExcludeDNS,

			ExcludeVirtualSubnetsOnlyIPv4: vpnconf.Split.ExcludeVirtualSubnetsOnlyIPv4,
		},
		Flags: daemoncfg.VPNFlags{
			DisableAlwaysOnVPN: vpnconf.Flags.DisableAlwaysOnVPN,
		},
	}
}

// NewConfig returns a new Config.
func NewConfig() *Config {
	return &Config{
		Config:  ConfigFile,
		Verbose: false,

		SocketServer:    daemoncfg.NewSocketServer(),
		CPD:             daemoncfg.NewCPD(),
		DNSProxy:        daemoncfg.NewDNSProxy(),
		OpenConnect:     daemoncfg.NewOpenConnect(),
		Executables:     daemoncfg.NewExecutables(),
		SplitRouting:    daemoncfg.NewSplitRouting(),
		TrafficPolicing: daemoncfg.NewTrafficPolicing(),
		TND:             tnd.NewConfig(),
	}
}
