package daemoncfg

import (
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
		{"/test/ip", "/test/nft", "/test/resolvectl", "/test/sysctl"},
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

	// create config with executables
	c := &Executables{
		IP:         ip,
		Nft:        nft,
		Resolvectl: resolvectl,
		Sysctl:     sysctl,
	}

	// test with not all files existing, create files in the process
	for _, f := range []string{
		ip, nft, resolvectl, sysctl,
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
		ExecutablesResolvectl, ExecutablesSysctl}
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
			VPNConfig:       &vpnconfig.Config{},
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
		VPNConfig:       &vpnconfig.Config{},
	}
	got := NewConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
