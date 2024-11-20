package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
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
