// Package daemoncfg contains the internal daemon configuration.
package daemoncfg

import (
	"encoding/json"
	"net/netip"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"time"

	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/tnd/pkg/tnd"
)

// Socket Server default values.
var (
	// SocketServerSocketFile is the unix socket file.
	SocketServerSocketFile = "/run/oc-daemon/daemon.sock"

	// SocketServerSocketOwner is the owner of the socket file.
	SocketServerSocketOwner = ""

	// SocketServerSocketGroup is the group of the socket file.
	SocketServerSocketGroup = ""

	// SocketServerSocketPermissions are the file permissions of the socket file.
	SocketServerSocketPermissions = "0700"

	// SocketServerRequestTimeout is the timeout for an entire request/response
	// exchange initiated by a client.
	SocketServerRequestTimeout = 30 * time.Second
)

// SocketServer the socket server configuration.
type SocketServer struct {
	SocketFile        string
	SocketOwner       string
	SocketGroup       string
	SocketPermissions string
	RequestTimeout    time.Duration
}

// Copy returns a copy of the SocketServer configuration.
func (c *SocketServer) Copy() *SocketServer {
	s := *c
	return &s
}

// Valid returns whether server config is valid.
func (c *SocketServer) Valid() bool {
	if c == nil ||
		c.SocketFile == "" ||
		c.RequestTimeout < 0 {
		return false
	}
	if c.SocketPermissions != "" {
		perm, err := strconv.ParseUint(c.SocketPermissions, 8, 32)
		if err != nil {
			return false
		}
		if perm > 0777 {
			return false
		}
	}
	return true
}

// NewSocketServer returns a new server configuration.
func NewSocketServer() *SocketServer {
	return &SocketServer{
		SocketFile:        SocketServerSocketFile,
		SocketOwner:       SocketServerSocketOwner,
		SocketGroup:       SocketServerSocketGroup,
		SocketPermissions: SocketServerSocketPermissions,
		RequestTimeout:    SocketServerRequestTimeout,
	}
}

// CPD default values.
var (
	// CPDHost is the host address used for probing.
	CPDHost = "connectivity-check.ubuntu.com"

	// CPDHTTPTimeout is the timeout for http requests in seconds.
	CPDHTTPTimeout = 5 * time.Second

	// CPDProbeCount is the number of probes to run.
	CPDProbeCount = 3

	// CPDProbeWait is the time between probes.
	CPDProbeWait = time.Second

	// CPDProbeTimer is the probe timer in case of no detected portal
	// in seconds.
	CPDProbeTimer = 300 * time.Second

	// CPDProbeTimerDetected is the probe timer in case of a detected portal
	// in seconds.
	CPDProbeTimerDetected = 15 * time.Second
)

// CPD is the configuration of the captive portal detection.
type CPD struct {
	Host               string
	HTTPTimeout        time.Duration
	ProbeCount         int
	ProbeWait          time.Duration
	ProbeTimer         time.Duration
	ProbeTimerDetected time.Duration
}

// Copy returns a copy of the CPD configuration.
func (c *CPD) Copy() *CPD {
	n := *c
	return &n
}

// Valid returns whether the captive portal detection configuration is valid.
func (c *CPD) Valid() bool {
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

// NewCPD returns a new default configuration for captive portal detection.
func NewCPD() *CPD {
	return &CPD{
		Host:               CPDHost,
		HTTPTimeout:        CPDHTTPTimeout,
		ProbeCount:         CPDProbeCount,
		ProbeWait:          CPDProbeWait,
		ProbeTimer:         CPDProbeTimer,
		ProbeTimerDetected: CPDProbeTimerDetected,
	}
}

// DNSProxy default values.
var (
	// DNSProxyAddress is the default listen address of the DNS proxy.
	DNSProxyAddress = "127.0.0.1:4253"

	// DNSProxyListenUDP specifies whether the DNS proxy listens on UDP.
	DNSProxyListenUDP = true

	// DNSProxyListenTCP specifies whether the DNS proxy listens on TCP.
	DNSProxyListenTCP = true
)

// DNSProxy is the DNS proxy configuration.
type DNSProxy struct {
	Address   string
	ListenUDP bool
	ListenTCP bool
}

// Copy returns a copy of the DNSProxy configuration.
func (c *DNSProxy) Copy() *DNSProxy {
	d := *c
	return &d
}

// Valid returns whether the DNS proxy configuration is valid.
func (c *DNSProxy) Valid() bool {
	if c == nil ||
		c.Address == "" ||
		(!c.ListenUDP && !c.ListenTCP) {

		return false
	}
	return true
}

// NewDNSProxy returns a new DNS proxy configuration.
func NewDNSProxy() *DNSProxy {
	return &DNSProxy{
		Address:   DNSProxyAddress,
		ListenUDP: DNSProxyListenUDP,
		ListenTCP: DNSProxyListenTCP,
	}
}

// OpenConnect default values.
var (
	// OpenConnectOpenConnect is the default openconnect executable.
	OpenConnectOpenConnect = "openconnect"

	// OpenConnectXMLProfile is the default AnyConnect Profile.
	OpenConnectXMLProfile = "/var/lib/oc-daemon/profile.xml"

	// OpenConnectVPNCScript is the default vpnc-script.
	OpenConnectVPNCScript = "/usr/bin/oc-daemon-vpncscript"

	// OpenConnectVPNDevice is the default vpn network device name.
	OpenConnectVPNDevice = "oc-daemon-tun0"

	// OpenConnectPIDFile is the default file path of the PID file for openconnect.
	OpenConnectPIDFile = "/run/oc-daemon/openconnect.pid"

	// OpenConnectPIDOwner is the default owner of the PID file.
	OpenConnectPIDOwner = ""

	// OpenConnectPIDGroup is the default group of the PID file.
	OpenConnectPIDGroup = ""

	// OpenConnectPIDPermissions are the default file permissions of the PID file.
	OpenConnectPIDPermissions = "0600"

	// OpenConnectNoProxy specifies whether the no proxy flag is set in openconnect.
	OpenConnectNoProxy = true

	// OpenConnectExtraEnv are extra environment variables used by openconnect.
	OpenConnectExtraEnv = []string{}

	// OpenConnectExtraArgs are extra command line arguments used by openconnect.
	OpenConnectExtraArgs = []string{}
)

// OpenConnect is the configuration for the openconnect connection runner.
type OpenConnect struct {
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

// Copy returns a copy of the OpenConnect configuration.
func (c *OpenConnect) Copy() *OpenConnect {
	openConnect := *c
	openConnect.ExtraEnv = append(c.ExtraEnv[:0:0], c.ExtraEnv...)
	openConnect.ExtraArgs = append(c.ExtraArgs[:0:0], c.ExtraArgs...)

	return &openConnect
}

// Valid returns whether the openconnect configuration is valid.
func (c *OpenConnect) Valid() bool {
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

// NewOpenConnect returns a new configuration for an openconnect connection runner.
func NewOpenConnect() *OpenConnect {
	return &OpenConnect{
		OpenConnect: OpenConnectOpenConnect,

		XMLProfile: OpenConnectXMLProfile,
		VPNCScript: OpenConnectVPNCScript,
		VPNDevice:  OpenConnectVPNDevice,

		PIDFile:        OpenConnectPIDFile,
		PIDOwner:       OpenConnectPIDOwner,
		PIDGroup:       OpenConnectPIDGroup,
		PIDPermissions: OpenConnectPIDPermissions,

		NoProxy:   OpenConnectNoProxy,
		ExtraEnv:  append(OpenConnectExtraEnv[:0:0], OpenConnectExtraEnv...),
		ExtraArgs: append(OpenConnectExtraArgs[:0:0], OpenConnectExtraArgs...),
	}
}

// Executables default values.
var (
	ExecutablesIP         = "ip"
	ExecutablesNft        = "nft"
	ExecutablesResolvectl = "resolvectl"
	ExecutablesSysctl     = "sysctl"
)

// Executables is the executables configuration.
type Executables struct {
	IP         string
	Nft        string
	Resolvectl string
	Sysctl     string
}

// Copy returns a copy of the executables configuration.
func (c *Executables) Copy() *Executables {
	e := *c
	return &e
}

// Valid returns whether config is valid.
func (c *Executables) Valid() bool {
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
// file system and are executable.
func (c *Executables) CheckExecutables() error {
	for _, f := range []string{
		c.IP, c.Nft, c.Resolvectl, c.Sysctl,
	} {
		if _, err := exec.LookPath(f); err != nil {
			return err
		}
	}
	return nil
}

// NewExecutables returns a new Executables configuration.
func NewExecutables() *Executables {
	return &Executables{
		IP:         ExecutablesIP,
		Nft:        ExecutablesNft,
		Resolvectl: ExecutablesResolvectl,
		Sysctl:     ExecutablesSysctl,
	}
}

// SplitRouting default values.
var (
	// SplitRoutingRoutingTable is the routing table.
	SplitRoutingRoutingTable = "42111"

	// SplitRoutingRulePriority1 is the first routing rule priority. It must be unique,
	// higher than the local rule, lower than the main and default rules,
	// lower than the second routing rule priority.
	SplitRoutingRulePriority1 = "2111"

	// SplitRoutingRulePriority2 is the second routing rule priority. It must be unique,
	// higher than the local rule, lower than the main and default rules,
	// higher than the first routing rule priority.
	SplitRoutingRulePriority2 = "2112"

	// SplitRoutingFirewallMark is the firewall mark used for split routing.
	SplitRoutingFirewallMark = SplitRoutingRoutingTable
)

// SplitRouting is the split routing configuration.
type SplitRouting struct {
	RoutingTable  string
	RulePriority1 string
	RulePriority2 string
	FirewallMark  string
}

// Copy returns a copy of the SplitRouting configuration.
func (c *SplitRouting) Copy() *SplitRouting {
	s := *c
	return &s
}

// Valid returns whether the split routing configuration is valid.
func (c *SplitRouting) Valid() bool {
	if c == nil ||
		c.RoutingTable == "" ||
		c.RulePriority1 == "" ||
		c.RulePriority2 == "" ||
		c.FirewallMark == "" {

		return false
	}

	// check routing table value: must be > 0, < 0xFFFFFFFF
	rtTable, err := strconv.ParseUint(c.RoutingTable, 10, 32)
	if err != nil || rtTable == 0 || rtTable >= 0xFFFFFFFF {
		return false
	}

	// check rule priority values: must be > 0, < 32766, prio1 < prio2
	prio1, err := strconv.ParseUint(c.RulePriority1, 10, 16)
	if err != nil {
		return false
	}
	prio2, err := strconv.ParseUint(c.RulePriority2, 10, 16)
	if err != nil {
		return false
	}
	if prio1 == 0 || prio2 == 0 ||
		prio1 >= 32766 || prio2 >= 32766 ||
		prio1 >= prio2 {

		return false
	}

	// check fwmark value: must be 32 bit unsigned int
	if _, err := strconv.ParseUint(c.FirewallMark, 10, 32); err != nil {
		return false
	}

	return true
}

// NewSplitRouting returns a new split routing configuration.
func NewSplitRouting() *SplitRouting {
	return &SplitRouting{
		RoutingTable:  SplitRoutingRoutingTable,
		RulePriority1: SplitRoutingRulePriority1,
		RulePriority2: SplitRoutingRulePriority2,
		FirewallMark:  SplitRoutingFirewallMark,
	}
}

// Traffic Policing default values.
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
		"networkcheck.kde.org",          // kde
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

// TrafficPolicing is a TrafPol configuration.
type TrafficPolicing struct {
	AllowedHosts []string
	PortalPorts  []uint16

	ResolveTimeout    time.Duration
	ResolveTries      int
	ResolveTriesSleep time.Duration
	ResolveTimer      time.Duration
	ResolveTTL        time.Duration
}

// Copy returns a copy of the TrafficPolicing configuration.
func (c *TrafficPolicing) Copy() *TrafficPolicing {
	trafpol := *c
	trafpol.AllowedHosts = append(c.AllowedHosts[:0:0], c.AllowedHosts...)
	trafpol.PortalPorts = append(c.PortalPorts[:0:0], c.PortalPorts...)

	return &trafpol
}

// Valid returns whether the TrafPol configuration is valid.
func (c *TrafficPolicing) Valid() bool {
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

// NewTrafficPolicing returns a new TrafPol configuration.
func NewTrafficPolicing() *TrafficPolicing {
	return &TrafficPolicing{
		AllowedHosts: append(AllowedHosts[:0:0], AllowedHosts...),
		PortalPorts:  append(PortalPorts[:0:0], PortalPorts...),

		ResolveTimeout:    ResolveTimeout,
		ResolveTries:      ResolveTries,
		ResolveTriesSleep: ResolveTriesSleep,
		ResolveTimer:      ResolveTimer,
		ResolveTTL:        ResolveTTL,
	}
}

// Command lists default values
var (
	CommandListsListsFile     = configDir + "/command-lists.json"
	CommandListsTemplatesFile = configDir + "/command-lists.tmpl"
)

// CommandLists is the command lists configuration.
type CommandLists struct {
	ListsFile     string
	TemplatesFile string
}

// Copy returns a copy of the command lists configuration.
func (c *CommandLists) Copy() *CommandLists {
	n := *c
	return &n
}

// Valid returns whether the command lists configuration is valid.
func (c *CommandLists) Valid() bool {
	if c == nil ||
		c.ListsFile == "" ||
		c.TemplatesFile == "" {

		return false
	}
	return true
}

// NewCommandLists returns a new command lists configuration.
func NewCommandLists() *CommandLists {
	return &CommandLists{
		ListsFile:     CommandListsListsFile,
		TemplatesFile: CommandListsTemplatesFile,
	}
}

// VPNDevice is a VPN device configuration in VPNConfig.
type VPNDevice struct {
	Name string
	MTU  int
}

// Copy returns a copy of the VPN device.
func (d *VPNDevice) Copy() VPNDevice {
	return VPNDevice{
		Name: d.Name,
		MTU:  d.MTU,
	}
}

// VPNDNS is a DNS configuration in VPNConfig.
type VPNDNS struct {
	DefaultDomain string
	ServersIPv4   []netip.Addr
	ServersIPv6   []netip.Addr
}

// Copy returns a copy of VPNDNS.
func (d *VPNDNS) Copy() VPNDNS {
	return VPNDNS{
		DefaultDomain: d.DefaultDomain,
		ServersIPv4:   append(d.ServersIPv4[:0:0], d.ServersIPv4...),
		ServersIPv6:   append(d.ServersIPv6[:0:0], d.ServersIPv6...),
	}
}

// Remotes returns a map of DNS remotes from the DNS configuration that maps
// domain "." to the IPv4 and IPv6 DNS servers in the configuration including
// port number 53.
func (d *VPNDNS) Remotes() map[string][]string {
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

// VPNSplit is a split routing configuration in VPNConfig.
type VPNSplit struct {
	ExcludeIPv4 []netip.Prefix
	ExcludeIPv6 []netip.Prefix
	ExcludeDNS  []string

	ExcludeVirtualSubnetsOnlyIPv4 bool
}

// Copy returns a copy of VPN split.
func (s *VPNSplit) Copy() VPNSplit {
	return VPNSplit{
		ExcludeIPv4: append(s.ExcludeIPv4[:0:0], s.ExcludeIPv4...),
		ExcludeIPv6: append(s.ExcludeIPv6[:0:0], s.ExcludeIPv6...),
		ExcludeDNS:  append(s.ExcludeDNS[:0:0], s.ExcludeDNS...),

		ExcludeVirtualSubnetsOnlyIPv4: s.ExcludeVirtualSubnetsOnlyIPv4,
	}
}

// DNSExcludes returns a list of DNS-based split excludes from the
// split routing configuration. The list contains domain names including the
// trailing ".".
func (s *VPNSplit) DNSExcludes() []string {
	excludes := make([]string, len(s.ExcludeDNS))
	for i, e := range s.ExcludeDNS {
		excludes[i] = e + "."
	}

	return excludes
}

// VPNFlags are other configuration settings in VPNConfig.
type VPNFlags struct {
	DisableAlwaysOnVPN bool
}

// Copy returns a copy of VPN flags.
func (f *VPNFlags) Copy() VPNFlags {
	return VPNFlags{
		DisableAlwaysOnVPN: f.DisableAlwaysOnVPN,
	}
}

// VPNConfig is a VPN configuration.
type VPNConfig struct {
	Gateway netip.Addr
	PID     int
	Timeout int
	Device  VPNDevice
	IPv4    netip.Prefix
	IPv6    netip.Prefix
	DNS     VPNDNS
	Split   VPNSplit
	Flags   VPNFlags
}

// Copy returns a copy of the VPN configuration.
func (c *VPNConfig) Copy() *VPNConfig {
	if c == nil {
		return nil
	}
	return &VPNConfig{
		Gateway: c.Gateway,
		PID:     c.PID,
		Timeout: c.Timeout,
		Device:  c.Device.Copy(),
		IPv4:    c.IPv4,
		IPv6:    c.IPv6,
		DNS:     c.DNS.Copy(),
		Split:   c.Split.Copy(),
		Flags:   c.Flags.Copy(),
	}
}

// Empty returns whether the VPN configuration is empty.
func (c *VPNConfig) Empty() bool {
	empty := &VPNConfig{}
	return reflect.DeepEqual(c, empty)
}

// Valid returns whether the VPN configuration is valid.
func (c *VPNConfig) Valid() bool {
	// an empty config is valid
	if c.Empty() {
		return true
	}

	// check config entries
	for _, invalid := range []bool{
		!c.Gateway.IsValid(),
		c.Device.Name == "",
		len(c.Device.Name) > 15,
		c.Device.MTU < 68,
		c.Device.MTU > 16384,
		!c.IPv4.IsValid() && !c.IPv6.IsValid(),
		len(c.DNS.ServersIPv4) == 0 && len(c.DNS.ServersIPv6) == 0,
	} {
		if invalid {
			return false
		}
	}

	return true
}

// GetVPNConfig converts vpnconf to VPNConfig.
func GetVPNConfig(vpnconf *vpnconfig.Config) *VPNConfig {
	// convert gateway
	gateway := netip.Addr{}
	if g, ok := netip.AddrFromSlice(vpnconf.Gateway); ok {
		// unmap to make sure we don't get an IPv4-mapped IPv6 address
		gateway = g.Unmap()
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
			pre6 := netip.PrefixFrom(ipv6, pre6len)
			excludes6 = append(excludes6, pre6)
		}
	}

	return &VPNConfig{
		Gateway: gateway,
		PID:     vpnconf.PID,
		Timeout: vpnconf.Timeout,
		Device: VPNDevice{
			Name: vpnconf.Device.Name,
			MTU:  vpnconf.Device.MTU,
		},
		IPv4: pre4,
		IPv6: pre6,
		DNS: VPNDNS{
			DefaultDomain: vpnconf.DNS.DefaultDomain,
			ServersIPv4:   dns4,
			ServersIPv6:   dns6,
		},
		Split: VPNSplit{
			ExcludeIPv4: excludes4,
			ExcludeIPv6: excludes6,
			ExcludeDNS:  vpnconf.Split.ExcludeDNS,

			ExcludeVirtualSubnetsOnlyIPv4: vpnconf.Split.ExcludeVirtualSubnetsOnlyIPv4,
		},
		Flags: VPNFlags{
			DisableAlwaysOnVPN: vpnconf.Flags.DisableAlwaysOnVPN,
		},
	}
}

// Config default values.
var (
	// configDir is the directory for the configuration.
	configDir = "/var/lib/oc-daemon"

	// ConfigFile is the default config file.
	ConfigFile = configDir + "/oc-daemon.json"
)

// Config is an OC-Daemon configuration.
type Config struct {
	Config  string `json:"-"`
	Verbose bool

	SocketServer    *SocketServer
	CPD             *CPD
	DNSProxy        *DNSProxy
	OpenConnect     *OpenConnect
	Executables     *Executables
	SplitRouting    *SplitRouting
	TrafficPolicing *TrafficPolicing
	TND             *tnd.Config

	CommandLists *CommandLists

	LoginInfo *logininfo.LoginInfo `json:"-"`
	VPNConfig *VPNConfig           `json:"-"`
}

// Copy returns a copy of the configuration.
func (c *Config) Copy() *Config {
	return &Config{
		Config:  c.Config,
		Verbose: c.Verbose,

		SocketServer:    c.SocketServer.Copy(),
		CPD:             c.CPD.Copy(),
		DNSProxy:        c.DNSProxy.Copy(),
		OpenConnect:     c.OpenConnect.Copy(),
		Executables:     c.Executables.Copy(),
		SplitRouting:    c.SplitRouting.Copy(),
		TrafficPolicing: c.TrafficPolicing.Copy(),
		TND:             c.TND.Copy(),

		CommandLists: c.CommandLists.Copy(),

		LoginInfo: c.LoginInfo.Copy(),
		VPNConfig: c.VPNConfig.Copy(),
	}
}

// String returns the configuration as string.
func (c *Config) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// loginInfoEmpty returns whether l is empty.
func loginInfoEmpty(l *logininfo.LoginInfo) bool {
	empty := &logininfo.LoginInfo{}
	return reflect.DeepEqual(l, empty)
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
		!c.TND.Valid() ||
		!c.CommandLists.Valid() ||
		!loginInfoEmpty(c.LoginInfo) && !c.LoginInfo.Valid() ||
		!c.VPNConfig.Valid() {
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
	return json.Unmarshal(file, c)
}

// NewConfig returns a new Config.
func NewConfig() *Config {
	return &Config{
		Config:  ConfigFile,
		Verbose: false,

		SocketServer:    NewSocketServer(),
		CPD:             NewCPD(),
		DNSProxy:        NewDNSProxy(),
		OpenConnect:     NewOpenConnect(),
		Executables:     NewExecutables(),
		SplitRouting:    NewSplitRouting(),
		TrafficPolicing: NewTrafficPolicing(),
		TND:             tnd.NewConfig(),

		CommandLists: NewCommandLists(),

		LoginInfo: &logininfo.LoginInfo{},
		VPNConfig: &VPNConfig{},
	}
}
