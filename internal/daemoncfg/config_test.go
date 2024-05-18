package daemoncfg

import (
	"log"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/tnd/pkg/tnd"
)

// TestSocketServerValid tests Valid of SocketServer.
func TestSocketServerValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*SocketServer{
		nil,
		{},
		{SocketFile: "test.sock", SocketPermissions: "invalid"},
		{SocketFile: "test.sock", SocketPermissions: "1234"},
	} {
		want := false
		got := invalid.Valid()
		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	for _, valid := range []*SocketServer{
		NewSocketServer(),
		{SocketFile: "test.sock", SocketPermissions: "777"},
	} {
		want := true
		got := valid.Valid()
		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, valid)
		}
	}
}

// TestNewSocketServer tests NewSocketServer.
func TestNewSocketServer(t *testing.T) {
	sc := NewSocketServer()
	if !sc.Valid() {
		t.Errorf("config is not valid")
	}
}

// TestCPDValid tests Valid of CPD.
func TestCPDValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*CPD{
		nil,
		{},
	} {
		if invalid.Valid() {
			t.Errorf("config should be invalid: %v", invalid)
		}
	}

	// test valid
	for _, valid := range []*CPD{
		NewCPD(),
		{
			Host:               "some.host.example.com",
			HTTPTimeout:        3 * time.Second,
			ProbeCount:         5,
			ProbeWait:          2 * time.Second,
			ProbeTimer:         150 * time.Second,
			ProbeTimerDetected: 10 * time.Second,
			SetFirewallMark:    true,
		},
	} {
		if !valid.Valid() {
			t.Errorf("config should be valid: %v", valid)
		}
	}
}

// TestNewCPD tests NewCPD.
func TestNewCPD(t *testing.T) {
	c := NewCPD()
	if !c.Valid() {
		t.Errorf("new config should be valid")
	}
}

// TestDNSProxyValid tests Valid of DNSProxy.
func TestDNSProxyValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*DNSProxy{
		nil,
		{},
	} {
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	for _, valid := range []*DNSProxy{
		NewDNSProxy(),
		{Address: "127.0.0.1:4253", ListenUDP: true},
		{Address: "127.0.0.1:4253", ListenTCP: true},
	} {
		want := true
		got := valid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, valid)
		}
	}
}

// TestNewDNSProxy tests NewDNSProxy.
func TestNewDNSProxy(t *testing.T) {
	want := &DNSProxy{
		Address:   DNSProxyAddress,
		ListenUDP: true,
		ListenTCP: true,
	}
	got := NewDNSProxy()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestOpenConnectValid tests Valid of OpenConnect.
func TestOpenConnectValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*OpenConnect{
		nil,
		{},
		{
			OpenConnect:    "openconnect",
			XMLProfile:     "/test/profile",
			VPNCScript:     "/test/vpncscript",
			VPNDevice:      "test-device",
			PIDFile:        "/test/pid",
			PIDPermissions: "invalid",
		},
		{
			OpenConnect:    "openconnect",
			XMLProfile:     "/test/profile",
			VPNCScript:     "/test/vpncscript",
			VPNDevice:      "test-device",
			PIDFile:        "/test/pid",
			PIDPermissions: "1234",
		},
	} {
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	for _, valid := range []*OpenConnect{
		NewOpenConnect(),
		{
			OpenConnect:    "openconnect",
			XMLProfile:     "/test/profile",
			VPNCScript:     "/test/vpncscript",
			VPNDevice:      "test-device",
			PIDFile:        "/test/pid",
			PIDPermissions: "777",
		},
	} {
		want := true
		got := valid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, valid)
		}
	}
}

// TestNewOpenConnect tests NewOpenConnect.
func TestNewOpenConnect(t *testing.T) {
	want := &OpenConnect{
		OpenConnect: OpenConnectOpenConnect,

		XMLProfile: OpenConnectXMLProfile,
		VPNCScript: OpenConnectVPNCScript,
		VPNDevice:  OpenConnectVPNDevice,

		PIDFile:        OpenConnectPIDFile,
		PIDOwner:       OpenConnectPIDOwner,
		PIDGroup:       OpenConnectPIDGroup,
		PIDPermissions: OpenConnectPIDPermissions,

		NoProxy:   OpenConnectNoProxy,
		ExtraEnv:  OpenConnectExtraEnv,
		ExtraArgs: OpenConnectExtraArgs,
	}
	got := NewOpenConnect()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestExecutablesValid tests Valid of Executables.
func TestExecutablesValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Executables{
		nil,
		{},
	} {
		if invalid.Valid() {
			t.Errorf("config should be invalid: %v", invalid)
		}
	}

	// test valid
	for _, valid := range []*Executables{
		NewExecutables(),
		{"/test/ip", "/test/nft", "/test/resolvectl", "/test/sysctl", "/test/openconnect"},
	} {
		if !valid.Valid() {
			t.Errorf("config should be valid: %v", valid)
		}
	}
}

// TestExecutablesCheckExecutables tests CheckExecutables of Executables.
func TestExecutablesCheckExecutables(t *testing.T) {
	// create temporary dir for executables
	dir, err := os.MkdirTemp("", "execs-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// create executable file paths
	ip := filepath.Join(dir, "ip")
	nft := filepath.Join(dir, "nft")
	resolvectl := filepath.Join(dir, "resolvectl")
	sysctl := filepath.Join(dir, "sysctl")
	openconnect := filepath.Join(dir, "openconnect")

	// create config with executables
	c := &Executables{
		IP:          ip,
		Nft:         nft,
		Resolvectl:  resolvectl,
		Sysctl:      sysctl,
		Openconnect: openconnect,
	}

	// test with not all files existing, create files in the process
	for _, f := range []string{
		ip, nft, resolvectl, sysctl, openconnect,
	} {
		// test
		if got := c.CheckExecutables(); got == nil {
			t.Errorf("got nil, want != nil")
		}

		// create executable file
		if err := os.WriteFile(f, []byte{}, 0777); err != nil {
			t.Fatal(err)
		}
	}

	// test with all files existing
	if got := c.CheckExecutables(); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// TestNewExecutables tests NewExecutables.
func TestNewExecutables(t *testing.T) {
	want := &Executables{ExecutablesIP, ExecutablesNft,
		ExecutablesResolvectl, ExecutablesSysctl,
		ExecutablesOpenconnect}
	got := NewExecutables()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitRoutingValid tests Valid of SplitRouting.
func TestSplitRoutingValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*SplitRouting{
		nil,
		{},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "0",
			RulePriority2: "1",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "32766",
			RulePriority2: "32767",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "2111",
			RulePriority2: "2111",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "2112",
			RulePriority2: "2111",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "65537",
			RulePriority2: "2111",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "2111",
			RulePriority2: "65537",
		},
		{
			RoutingTable:  "0",
			FirewallMark:  "42112",
			RulePriority1: "2222",
			RulePriority2: "2223",
		},
		{
			RoutingTable:  "4294967295",
			FirewallMark:  "42112",
			RulePriority1: "2222",
			RulePriority2: "2223",
		},
		{
			RoutingTable:  "42112",
			FirewallMark:  "4294967296",
			RulePriority1: "2222",
			RulePriority2: "2223",
		},
	} {
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	for _, valid := range []*SplitRouting{
		NewSplitRouting(),
		{
			RoutingTable:  "42112",
			FirewallMark:  "42112",
			RulePriority1: "2222",
			RulePriority2: "2223",
		},
	} {
		want := true
		got := valid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, valid)
		}
	}
}

// TestNewSplitRouting tests NewSplitRouting.
func TestNewSplitRouting(t *testing.T) {
	c := NewSplitRouting()
	if !c.Valid() {
		t.Errorf("new config should be valid")
	}
}

// TestTrafficPolicingValid tests Valid of TrafficPolicing.
func TestTrafficPolicingValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*TrafficPolicing{
		nil,
		{},
	} {
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	valid := NewTrafficPolicing()
	want := true
	got := valid.Valid()

	if got != want {
		t.Errorf("got %t, want %t for %v", got, want, valid)
	}
}

// TestNewTrafficPolicing tests NewTrafficPolicing.
func TestNewTrafficPolicing(t *testing.T) {
	c := NewTrafficPolicing()
	if !c.Valid() {
		t.Errorf("new config should be valid")
	}
}

// TestVPNDNSRemotes tests Remotes of VPNDNS.
func TestVPNDNSRemotes(t *testing.T) {
	// test empty
	c := &VPNConfig{}
	if len(c.DNS.Remotes()) != 0 {
		t.Errorf("got %d, want 0", len(c.DNS.Remotes()))
	}

	// test ipv4
	for _, want := range [][]string{
		{"127.0.0.1:53"},
		{"127.0.0.1:53", "192.168.1.1:53"},
		{"127.0.0.1:53", "192.168.1.1:53", "10.0.0.1:53"},
	} {
		c := &VPNConfig{}
		for _, ip := range want {
			ip = ip[:len(ip)-3] // remove port
			c.DNS.ServersIPv4 = append(c.DNS.ServersIPv4, netip.MustParseAddr(ip))
		}
		got := c.DNS.Remotes()["."]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test ipv6
	for _, want := range [][]string{
		{"[::1]:53"},
		{"[::1]:53", "[2000::1]:53"},
		{"[::1]:53", "[2000::1]:53", "[2002::1]:53"},
	} {
		c := &VPNConfig{}
		for _, ip := range want {
			ip = ip[1 : len(ip)-4] // remove port and brackets
			c.DNS.ServersIPv6 = append(c.DNS.ServersIPv6, netip.MustParseAddr(ip))
		}
		got := c.DNS.Remotes()["."]
		log.Println(got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test both ipv4 and ipv6
	c = &VPNConfig{}
	dns4 := "127.0.0.1"
	dns6 := "::1"
	c.DNS.ServersIPv4 = append(c.DNS.ServersIPv4, netip.MustParseAddr(dns4))
	c.DNS.ServersIPv6 = append(c.DNS.ServersIPv6, netip.MustParseAddr(dns6))

	want := map[string][]string{
		".": {dns4 + ":53", "[" + dns6 + "]:53"},
	}
	got := c.DNS.Remotes()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestVPNSplitDNSExcludes tests DNSExcludes of VPNSplit.
func TestVPNSplitDNSExcludes(t *testing.T) {
	// test empty
	c := &VPNConfig{}
	if len(c.Split.DNSExcludes()) != 0 {
		t.Errorf("got %d, want 0", len(c.Split.DNSExcludes()))
	}

	// test filled
	c = &VPNConfig{}
	want := []string{"example.com", "test.com"}
	c.Split.ExcludeDNS = want
	for i, got := range c.Split.DNSExcludes() {
		want := want[i] + "."
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

// TestVPNConfigCopy tests Copy of VPNConfig.
func TestVPNConfigCopy(t *testing.T) {
	// test nil
	if (*VPNConfig)(nil).Copy() != nil {
		t.Error("copy of nil should be nil")
	}

	// test with new config
	want := &VPNConfig{}
	got := want.Copy()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test modification after copy
	c1 := &VPNConfig{}
	c2 := c1.Copy()
	c1.PID = 12345
	c1.Split.ExcludeIPv4 = append(c1.Split.ExcludeIPv4, netip.MustParsePrefix("192.168.1.0/24"))

	if reflect.DeepEqual(c1, c2) {
		t.Error("copies should not be equal after modification")
	}
}

// getValidTestVPNConfig returns a valid VPNConfig for testing.
func getValidTestVPNConfig() *VPNConfig {
	c := &VPNConfig{}

	c.Gateway = netip.MustParseAddr("192.168.0.1")
	c.PID = 123456
	c.Timeout = 300
	c.Device.Name = "tun0"
	c.Device.MTU = 1300
	c.IPv4 = netip.MustParsePrefix("192.168.0.123/24")
	c.IPv6 = netip.MustParsePrefix("2001:42:42:42::1/64")
	c.DNS.DefaultDomain = "mycompany.com"
	c.DNS.ServersIPv4 = []netip.Addr{netip.MustParseAddr("192.168.0.53")}
	c.DNS.ServersIPv6 = []netip.Addr{netip.MustParseAddr("2001:53:53:53::53")}
	c.Split.ExcludeIPv4 = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/32"),
		netip.MustParsePrefix("10.0.0.0/24"),
	}
	c.Split.ExcludeIPv6 = []netip.Prefix{
		netip.MustParsePrefix("2001:2:3:4::1/128"),
		netip.MustParsePrefix("2001:2:3:5::1/64"),
	}
	c.Split.ExcludeDNS = []string{"this.other.com", "that.other.com"}
	c.Split.ExcludeVirtualSubnetsOnlyIPv4 = true
	c.Flags.DisableAlwaysOnVPN = true

	return c
}

// TestVPNConfigValid tests Valid of VPNConfig.
func TestVPNConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*VPNConfig{
		// only device name set, invalid device name
		{Device: VPNDevice{Name: "this is too long for a device name"}},
		// only PID set
		{PID: 123},
	} {
		if invalid.Valid() {
			t.Errorf("%v should not be valid", invalid)
		}
	}

	// test valid
	for _, valid := range []*VPNConfig{
		// empty
		{},
		// full valid config
		getValidTestVPNConfig(),
	} {
		if !valid.Valid() {
			t.Errorf("%v should be valid", valid)
		}
	}
}

// TestGetVPNConfig tests GetVPNConfig.
func TestGetVPNConfig(t *testing.T) {
	// create vpnconfig.Config
	c := vpnconfig.New()

	c.Gateway = net.IPv4(192, 168, 0, 1)
	c.PID = 123456
	c.Timeout = 300
	c.Device.Name = "tun0"
	c.Device.MTU = 1300
	c.IPv4.Address = net.IPv4(192, 168, 0, 123)
	c.IPv4.Netmask = net.IPv4Mask(255, 255, 255, 0)
	c.IPv6.Address = net.ParseIP("2001:42:42:42::1")
	c.IPv6.Netmask = net.CIDRMask(64, 128)
	c.DNS.DefaultDomain = "mycompany.com"
	c.DNS.ServersIPv4 = []net.IP{net.IPv4(192, 168, 0, 53)}
	c.DNS.ServersIPv6 = []net.IP{net.ParseIP("2001:53:53:53::53")}
	c.Split.ExcludeIPv4 = []*net.IPNet{
		{
			IP:   net.IPv4(0, 0, 0, 0),
			Mask: net.IPv4Mask(255, 255, 255, 255),
		},
		{
			IP:   net.IPv4(10, 0, 0, 0),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
	}
	c.Split.ExcludeIPv6 = []*net.IPNet{
		{
			IP:   net.ParseIP("2001:2:3:4::1"),
			Mask: net.CIDRMask(128, 128),
		},
		{
			IP:   net.ParseIP("2001:2:3:5::1"),
			Mask: net.CIDRMask(64, 128),
		},
	}
	c.Split.ExcludeDNS = []string{"this.other.com", "that.other.com"}
	c.Split.ExcludeVirtualSubnetsOnlyIPv4 = true
	c.Flags.DisableAlwaysOnVPN = true

	// convert and check
	got := GetVPNConfig(c)
	if got.Gateway.String() != "192.168.0.1" ||
		got.PID != c.PID ||
		got.Timeout != c.Timeout ||
		got.Device.Name != c.Device.Name ||
		got.Device.MTU != c.Device.MTU ||
		got.IPv4.String() != "192.168.0.123/24" ||
		got.IPv6.String() != "2001:42:42:42::1/64" ||
		got.DNS.DefaultDomain != c.DNS.DefaultDomain ||
		got.DNS.ServersIPv4[0].String() != "192.168.0.53" ||
		got.DNS.ServersIPv6[0].String() != "2001:53:53:53::53" ||
		got.Split.ExcludeIPv4[0].String() != "0.0.0.0/32" ||
		got.Split.ExcludeIPv4[1].String() != "10.0.0.0/24" ||
		got.Split.ExcludeIPv6[0].String() != "2001:2:3:4::1/128" ||
		got.Split.ExcludeIPv6[1].String() != "2001:2:3:5::1/64" ||
		got.Split.ExcludeDNS[0] != "this.other.com" ||
		got.Split.ExcludeDNS[1] != "that.other.com" ||
		got.Split.ExcludeVirtualSubnetsOnlyIPv4 != c.Split.ExcludeVirtualSubnetsOnlyIPv4 ||
		got.Flags.DisableAlwaysOnVPN != c.Flags.DisableAlwaysOnVPN {
		t.Errorf("invalid conversion: %+v", got)
	}
}

// TestConfigCopy tests Copy of Config.
func TestConfigCopy(t *testing.T) {
	// test with new config
	want := NewConfig()
	got := want.Copy()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test modification after copy
	c1 := NewConfig()
	c2 := c1.Copy()
	c1.Verbose = !c2.Verbose
	c1.LoginInfo.Cookie = "something else"
	c1.VPNConfig.PID = 123456

	if reflect.DeepEqual(c1, c2) {
		t.Error("copies should not be equal after modification")
	}
}

// TestConfigString tests String of Config.
func TestConfigString(t *testing.T) {
	// test new config
	c := NewConfig()
	if c.String() == "" {
		t.Errorf("string should not be empty: %s", c.String())
	}

	// test nil
	c = nil
	if c.String() != "null" {
		t.Errorf("string should be null: %s", c.String())
	}
}

// TestConfigValid tests Valid of Config.
func TestConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Config{
		nil,
		{},
	} {
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	valid := NewConfig()
	want := true
	got := valid.Valid()

	if got != want {
		t.Errorf("got %t, want %t for %v", got, want, valid)
	}
}

// TestConfigLoad tests Load of Config.
func TestConfigLoad(t *testing.T) {
	conf := NewConfig()
	conf.Config = "does not exist"

	// test invalid path
	err := conf.Load()
	if err == nil {
		t.Errorf("got != nil, want nil")
	}

	// test empty config file
	empty, err := os.CreateTemp("", "oc-daemon-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(empty.Name())
	}()

	conf = NewConfig()
	conf.Config = empty.Name()
	err = conf.Load()
	if err == nil {
		t.Errorf("got != nil, want nil")
	}

	// test invalid config file entries
	// - Config
	// - LoginInfo
	// - VPNConfig
	for _, content := range []string{
		`{
	"Config": "should not be here",
	"Verbose": true
}`,
		`{
	"Verbose": true,
	"LoginInfo": {
		"Server": "192.168.1.1"
	}
}`,
		`{
	"Verbose": true,
	"VPNConfig": {
		"Gateway": "192.168.1.1"
	}
}`,
	} {
		invalid, err := os.CreateTemp("", "oc-daemon-config-test")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Remove(invalid.Name())
		}()

		if _, err := invalid.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}

		conf := NewConfig()
		conf.Config = invalid.Name()
		if err := conf.Load(); err != nil {
			t.Error(err)
		}
		// make sure invalid file entries did not change config
		if conf.Config != invalid.Name() ||
			conf.LoginInfo.Server != "" ||
			conf.VPNConfig.Gateway.IsValid() {
			t.Errorf("loaded config with invalid entry: %s", content)
		}
	}

	// test valid config file
	// - complete config
	// - partial config with defaults
	for _, content := range []string{
		`{
	"Verbose": true,
	"SocketServer": {
		"SocketFile": "/run/oc-daemon/daemon.sock",
		"SocketOwner": "",
		"SocketGroup": "",
		"SocketPermissions":  "0700",
		"RequestTimeout": 30000000000
	},
	"CPD": {
		"Host": "connectivity-check.ubuntu.com",
		"HTTPTimeout": 5000000000,
		"ProbeCount": 3,
		"ProbeTimer": 300000000000,
		"ProbeTimerDetected": 15000000000,
		"SetFirewallMark": true
	},
	"DNSProxy": {
		"Address": "127.0.0.1:4253",
		"ListenUDP": true,
		"ListenTCP": true
	},
	"OpenConnect": {
		"OpenConnect": "openconnect",
		"XMLProfile": "/var/lib/oc-daemon/profile.xml",
		"VPNCScript": "/usr/bin/oc-daemon-vpncscript",
		"VPNDevice": "oc-daemon-tun0",
		"PIDFile": "/run/oc-daemon/openconnect.pid",
		"PIDOwner": "",
		"PIDGroup": "",
		"PIDPermissions": "0600"
	},
	"Executables": {
		"IP": "ip",
		"Nft": "nft",
		"Resolvectl": "resolvectl",
		"Sysctl": "sysctl"
	},
	"SplitRouting": {
		"RoutingTable": "42111",
		"RulePriority1": "2111",
		"RulePriority2": "2112",
		"FirewallMark": "42111"
	},
	"TrafficPolicing": {
		"AllowedHosts": ["connectivity-check.ubuntu.com", "detectportal.firefox.com", "www.gstatic.com", "clients3.google.com", "nmcheck.gnome.org", "networkcheck.kde.org"],
		"PortalPorts": [80, 443],
		"ResolveTimeout": 2000000000,
		"ResolveTries": 3,
		"ResolveTriesSleep": 1000000000,
		"ResolveTTL": 300000000000
	},
	"TND": {
		"WatchFiles": ["/etc/resolv.conf", "/run/systemd/resolve/resolv.conf", "/run/systemd/resolve/stub-resolv.conf"],
		"WaitCheck": 1000000000,
		"HTTPSTimeout": 5000000000,
		"UntrustedTimer": 30000000000,
		"TrustedTimer": 60000000000
	},
	"CommandLists": {
		"ListsFile": "/var/lib/oc-daemon/command-lists.json"
	}
}`,
		`{
	"Verbose": true
}`,
	} {

		valid, err := os.CreateTemp("", "oc-daemon-config-test")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Remove(valid.Name())
		}()

		if _, err := valid.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}

		conf := NewConfig()
		conf.Config = valid.Name()
		if err := conf.Load(); err != nil {
			t.Errorf("could not load valid config: %s", err)
		}

		if !conf.Valid() {
			t.Errorf("config is not valid")
		}

		want := &Config{
			Config:          valid.Name(),
			Verbose:         true,
			SocketServer:    NewSocketServer(),
			CPD:             NewCPD(),
			DNSProxy:        NewDNSProxy(),
			OpenConnect:     NewOpenConnect(),
			Executables:     NewExecutables(),
			SplitRouting:    NewSplitRouting(),
			TrafficPolicing: NewTrafficPolicing(),
			TND:             tnd.NewConfig(),
			CommandLists:    NewCommandLists(),
			LoginInfo:       &logininfo.LoginInfo{},
			VPNConfig:       &VPNConfig{},
		}
		if !reflect.DeepEqual(want.DNSProxy, conf.DNSProxy) {
			t.Errorf("got %v, want %v", conf.DNSProxy, want.DNSProxy)
		}
		if !reflect.DeepEqual(want.OpenConnect, conf.OpenConnect) {
			t.Errorf("got %v, want %v", conf.OpenConnect, want.OpenConnect)
		}
		if !reflect.DeepEqual(want.Executables, conf.Executables) {
			t.Errorf("got %v, want %v", conf.Executables, want.Executables)
		}
		if !reflect.DeepEqual(want.SplitRouting, conf.SplitRouting) {
			t.Errorf("got %v, want %v", conf.SplitRouting, want.SplitRouting)
		}
		if !reflect.DeepEqual(want.TrafficPolicing, conf.TrafficPolicing) {
			t.Errorf("got %v, want %v", conf.TrafficPolicing, want.TrafficPolicing)
		}
		if !reflect.DeepEqual(want.TND, conf.TND) {
			t.Errorf("got %v, want %v", conf.TND, want.TND)
		}
		if !reflect.DeepEqual(want, conf) {
			t.Errorf("got %v, want %v", conf, want)
		}
	}
}

// TestNewConfig tests NewConfig.
func TestNewConfig(t *testing.T) {
	want := &Config{
		Config:          "/var/lib/oc-daemon/oc-daemon.json",
		Verbose:         false,
		SocketServer:    NewSocketServer(),
		CPD:             NewCPD(),
		DNSProxy:        NewDNSProxy(),
		OpenConnect:     NewOpenConnect(),
		Executables:     NewExecutables(),
		SplitRouting:    NewSplitRouting(),
		TrafficPolicing: NewTrafficPolicing(),
		TND:             tnd.NewConfig(),
		CommandLists:    NewCommandLists(),
		LoginInfo:       &logininfo.LoginInfo{},
		VPNConfig:       &VPNConfig{},
	}
	got := NewConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
