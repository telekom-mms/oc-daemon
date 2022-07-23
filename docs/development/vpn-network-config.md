# VPN Network Configuration

The VPN network configuration is retrieved over the VPN tunnel by openconnect,
passed to oc-dameon-vpncscript, and then parsed by the oc-daemon-vpncscript and
sent to the oc-daemon. The oc-daemon finally configures the VPN network.

Go-representation of the VPN network configuration:

```go
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

// Split is a split routing configuration in Config
type Split struct {
	ExcludeIPv4 []*net.IPNet
	ExcludeIPv6 []*net.IPNet
	ExcludeDNS  []string

	ExcludeVirtualSubnetsOnlyIPv4 bool
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
```

The oc-daemon sets up the basic network interface configuration using the tool
`ip`. The basic DNS resolver configuration is set up using `resolvectl`. For
more information, see the [Split Routing](split-routing.md) and [DNS
Configuration](dns-config.md).
