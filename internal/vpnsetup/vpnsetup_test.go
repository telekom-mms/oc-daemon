package vpnsetup

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/internal/execs"
	"github.com/telekom-mms/oc-daemon/internal/splitrt"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// TestSetupVPNDevice tests setupVPNDevice
func TestSetupVPNDevice(t *testing.T) {
	c := vpnconfig.New()
	c.Device.Name = "tun0"
	c.Device.MTU = 1300
	c.IPv4.Address = net.IPv4(192, 168, 0, 123)
	c.IPv4.Netmask = net.IPv4Mask(255, 255, 255, 0)
	c.IPv6.Address = net.ParseIP("2001::1")
	c.IPv6.Netmask = net.CIDRMask(64, 128)

	// overwrite RunCmd
	want := []string{
		"link set tun0 mtu 1300",
		"link set tun0 up",
		"address add 192.168.0.123/24 dev tun0",
		"address add 2001::1/64 dev tun0",
	}
	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, strings.Join(arg, " "))
		return nil
	}

	// test
	setupVPNDevice(context.Background(), c)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestTeardownVPNDevice tests teardownVPNDevice
func TestTeardownVPNDevice(t *testing.T) {
	c := vpnconfig.New()
	c.Device.Name = "tun0"

	// overwrite RunCmd
	want := []string{
		"link set tun0 down",
	}
	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, strings.Join(arg, " "))
		return nil
	}

	// test
	teardownVPNDevice(context.Background(), c)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestVPNSetupSetupDNS tests setupDNS of VPNSetup
func TestVPNSetupSetupDNS(t *testing.T) {
	c := vpnconfig.New()
	c.Device.Name = "tun0"
	c.DNS.DefaultDomain = "mycompany.com"

	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, strings.Join(arg, " "))
		return nil
	}
	v := NewVPNSetup(dnsproxy.NewConfig(), splitrt.NewConfig())
	v.setupDNS(context.Background(), c)

	want := []string{
		"dns tun0 127.0.0.1:4253",
		"domain tun0 mycompany.com ~.",
		"default-route tun0 yes",
		"flush-caches",
		"reset-server-features",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestVPNSetupTeardownDNS tests teardownDNS of VPNSetup
func TestVPNSetupTeardownDNS(t *testing.T) {
	c := vpnconfig.New()
	c.Device.Name = "tun0"

	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, strings.Join(arg, " "))
		return nil
	}
	v := NewVPNSetup(dnsproxy.NewConfig(), splitrt.NewConfig())
	v.teardownDNS(context.Background(), c)

	want := []string{
		"revert tun0",
		"flush-caches",
		"reset-server-features",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestVPNSetupCheckDNSProtocols tests checkDNSProtocols of VPNSetup
func TestVPNSetupCheckDNSProtocols(t *testing.T) {
	v := NewVPNSetup(dnsproxy.NewConfig(), splitrt.NewConfig())

	// test invalid
	for _, invalid := range [][]string{
		{},
		{"x", "y", "z"},
	} {
		if ok := v.checkDNSProtocols(invalid); ok {
			t.Errorf("dns check should fail with %v", invalid)
		}
	}

	// test valid
	for _, valid := range [][]string{
		{"+DefaultRoute"},
		{"x", "y", "z", "+DefaultRoute"},
	} {
		if ok := v.checkDNSProtocols(valid); !ok {
			t.Errorf("dns check should not fail with %v", valid)
		}
	}
}

// TestVPNSetupCheckDNSServers tests checkDNSServers of VPNSetup
func TestVPNSetupCheckDNSServers(t *testing.T) {
	v := NewVPNSetup(dnsproxy.NewConfig(), splitrt.NewConfig())

	// test invalid
	for _, invalid := range [][]string{
		{},
		{"x", "y", "z"},
	} {
		if ok := v.checkDNSServers(invalid); ok {
			t.Errorf("dns check should fail with %v", invalid)
		}
	}

	// test valid
	proxy := []string{v.dnsProxyConf.Address}
	if ok := v.checkDNSServers(proxy); !ok {
		t.Errorf("dns check should not fail with %v", proxy)
	}
}

// TestVPNSetupCheckDNSDomain tests checkDNSDomain of VPNSetup
func TestVPNSetupCheckDNSDomain(t *testing.T) {
	v := NewVPNSetup(dnsproxy.NewConfig(), splitrt.NewConfig())
	vpnconf := vpnconfig.New()
	vpnconf.DNS.DefaultDomain = "test.example.com"

	// test invalid
	for _, invalid := range [][]string{
		{},
		{"x", "y", "z"},
		{vpnconf.DNS.DefaultDomain},
	} {
		if ok := v.checkDNSDomain(vpnconf, invalid); ok {
			t.Errorf("dns check should fail with %v", invalid)
		}
	}

	// test valid
	for _, valid := range [][]string{
		{"~.", vpnconf.DNS.DefaultDomain},
		{vpnconf.DNS.DefaultDomain, "~."},
	} {
		if ok := v.checkDNSDomain(vpnconf, valid); !ok {
			t.Errorf("dns check should not fail with %v", valid)
		}
	}
}

func TestVPNSetupEnsureDNS(t *testing.T) {
	v := NewVPNSetup(dnsproxy.NewConfig(), splitrt.NewConfig())
	ctx := context.Background()
	vpnconf := vpnconfig.New()

	// test settings
	v.dnsProxyConf.Address = "127.0.0.1:4253"
	vpnconf.DNS.DefaultDomain = "test.example.com"

	// override RunCmd and clean up after tests
	oldCmd := execs.RunCmd
	defer func() { execs.RunCmd = oldCmd }()

	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		return nil
	}

	// clean up RunCmdOutput after tests
	oldCmdOutput := execs.RunCmdOutput
	defer func() { execs.RunCmdOutput = oldCmdOutput }()

	// test resolvectl error
	execs.RunCmdOutput = func(ctx context.Context, cmd string, s string, arg ...string) ([]byte, error) {
		return nil, errors.New("test error")
	}

	if ok := v.ensureDNS(ctx, vpnconf); ok {
		t.Errorf("ensure dns should fail with resolvectl error")
	}

	// test wrong settings
	for _, invalid := range [][]byte{
		[]byte(""),
		[]byte("header\n Protocols:\n DNS Servers:\n DNS Domain:\n"),
		[]byte("header\n Protocols: x y z \n DNS Servers: x y z \n DNS Domain: x y z\n"),
		[]byte("header\nProtocols: other\nDNS Servers: 127.0.0.1:4253\nDNS Domain: test.example.com ~.\n"),
		[]byte("header\nProtocols: +DefaultRoute\nDNS Servers: other\nDNS Domain: test.example.com ~.\n"),
		[]byte("header\nProtocols: +DefaultRoute\nDNS Servers: 127.0.0.1:4253\nDNS Domain: other\n"),
	} {
		execs.RunCmdOutput = func(ctx context.Context, cmd string, s string, arg ...string) ([]byte, error) {
			return invalid, nil
		}

		if ok := v.ensureDNS(ctx, vpnconf); ok {
			t.Errorf("ensure dns should fail with %v", invalid)
		}
	}

	// test correct settings
	for _, valid := range [][]byte{
		[]byte("header\nProtocols: +DefaultRoute\nDNS Servers: 127.0.0.1:4253\nDNS Domain: test.example.com ~.\n"),
		[]byte("header\n Protocols:  +DefaultRoute  \nother\n  " +
			"DNS Servers: 127.0.0.1:4253  \n  DNS Domain: test.example.com ~.\nother\n"),
	} {
		execs.RunCmdOutput = func(ctx context.Context, cmd string, s string, arg ...string) ([]byte, error) {
			return valid, nil
		}

		if ok := v.ensureDNS(ctx, vpnconf); !ok {
			t.Errorf("ensure dns should not fail with %v", valid)
		}
	}
}

// TestVPNSetupStartStop tests Start and Stop of VPNSetup
func TestVPNSetupStartStop(t *testing.T) {
	v := NewVPNSetup(dnsproxy.NewConfig(), splitrt.NewConfig())
	v.Start()
	v.Stop()
}

// TestVPNSetupEvents tests Events of VPNSetup
func TestVPNSetupEvents(t *testing.T) {
	v := NewVPNSetup(dnsproxy.NewConfig(), splitrt.NewConfig())
	want := v.events
	got := v.Events()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewVPNSetup tests NewVPNSetup
func TestNewVPNSetup(t *testing.T) {
	dnsConfig := dnsproxy.NewConfig()
	splitrtConfig := splitrt.NewConfig()

	v := NewVPNSetup(dnsConfig, splitrtConfig)
	if v == nil ||
		v.dnsProxyConf != dnsConfig ||
		v.splitrtConf != splitrtConfig ||
		v.cmds == nil ||
		v.events == nil ||
		v.done == nil {
		t.Errorf("invalid vpn setup")
	}
}

// TestCleanup tests Cleanup
func TestCleanup(t *testing.T) {
	got := []string{}
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		if s == "" {
			got = append(got, cmd+" "+strings.Join(arg, " "))
			return nil
		}
		got = append(got, cmd+" "+strings.Join(arg, " ")+" "+s)
		return nil
	}
	Cleanup(context.Background(), "tun0", splitrt.NewConfig())
	want := []string{
		"resolvectl revert tun0",
		"ip link delete tun0",
		"ip -4 rule delete pref 2111",
		"ip -4 rule delete pref 2112",
		"ip -6 rule delete pref 2111",
		"ip -6 rule delete pref 2112",
		"ip -4 route flush table 42111",
		"ip -6 route flush table 42111",
		"nft -f - delete table inet oc-daemon-routing",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
