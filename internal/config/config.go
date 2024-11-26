// Package config contains the internal daemon configuration.
package config

import (
	"net/netip"
	"os/exec"
	"strconv"
	"time"

	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
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

// NewConfig returns a new server configuration.
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
	FirewallMark string `json:"-"`

	ResolveTimeout    time.Duration
	ResolveTries      int
	ResolveTriesSleep time.Duration
	ResolveTimer      time.Duration
	ResolveTTL        time.Duration
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

//// TND is the TND configuration.
//type TND struct {
//	WaitCheck      time.Duration
//	HTTPSTimeout   time.Duration
//	UntrustedTimer time.Duration
//	TrustedTimer   time.Duration
//}

//// LoginInfo is login information for OpenConnect.
//type LoginInfo struct {
//	Server      string
//	Cookie      string
//	Host        string
//	ConnectURL  string
//	Fingerprint string
//	Resolve     string
//}

// VPNDevice is a VPN device configuration in VPNConfig.
type VPNDevice struct {
	Name string
	MTU  int
}

// VPNDNS is a DNS configuration in VPNConfig.
type VPNDNS struct {
	DefaultDomain string
	ServersIPv4   []netip.Addr
	ServersIPv6   []netip.Addr
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

// Config is an OC-Daemon configuration.
// TODO: use it in cmdtmpl?
// TODO: add runtime config login with json:"-"?
// TODO: add runtime config VPNConfig with json:"-"?
type Config struct {
	//Config  string `json:"-"`
	Verbose bool

	SocketServer    *SocketServer
	CPD             *CPD
	DNSProxy        *DNSProxy
	OpenConnect     *OpenConnect
	Executables     *Executables
	SplitRouting    *SplitRouting
	TrafficPolicing *TrafficPolicing
	TND             *tnd.Config

	LoginInfo *logininfo.LoginInfo //`json:"-"`
	VPNConfig *VPNConfig           //`json:"-"`
}

//// Load loads the configuration from the config file.
//func (c *Config) Load(file string) error {
//	// read file contents
//	b, err := os.ReadFile(file)
//	if err != nil {
//		return err
//	}
//
//	// parse config
//	if err := json.Unmarshal(b, c); err != nil {
//		return err
//	}
//
//	return nil
//}

// NewConfig returns a new Config.
func NewConfig() *Config {
	return &Config{
		Verbose: false,

		SocketServer:    NewSocketServer(),
		CPD:             NewCPD(),
		DNSProxy:        NewDNSProxy(),
		OpenConnect:     NewOpenConnect(),
		Executables:     NewExecutables(),
		SplitRouting:    NewSplitRouting(),
		TrafficPolicing: NewTrafficPolicing(),
		TND:             tnd.NewConfig(),

		LoginInfo: &logininfo.LoginInfo{},
		VPNConfig: &VPNConfig{},
	}
}
