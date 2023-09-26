package vpnsetup

import (
	"context"
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
	setupVPNDevice(c)
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
	teardownVPNDevice(c)
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
	v.setupDNS(c)

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
	v.teardownDNS(c)

	want := []string{
		"revert tun0",
		"flush-caches",
		"reset-server-features",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
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
	Cleanup("tun0", splitrt.NewConfig())
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
