package vpnsetup

import (
	"net"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/internal/splitrt"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/vishvananda/netlink"
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

	// overwrite netlink functions
	device := ""
	mtu := 0
	up := false
	addrs := []*netlink.Addr{}
	runLinkByName = func(name string) (netlink.Link, error) {
		device = name
		return &netlink.Device{}, nil
	}
	runLinkSetMTU = func(link netlink.Link, m int) error {
		mtu = m
		return nil
	}
	runLinkSetUp = func(netlink.Link) error {
		up = true
		return nil
	}
	runAddrAdd = func(link netlink.Link, addr *netlink.Addr) error {
		addrs = append(addrs, addr)
		return nil
	}

	// test
	setupVPNDevice(c)
	if device != c.Device.Name {
		t.Errorf("got %s, want %s", device, c.Device.Name)
	}
	if mtu != c.Device.MTU {
		t.Errorf("got %d, want %d", mtu, c.Device.MTU)
	}
	if !up {
		t.Errorf("got %t, want true", up)
	}
	a := addrs[0].IPNet
	if !a.IP.Equal(c.IPv4.Address) ||
		a.Mask.String() != c.IPv4.Netmask.String() {
		t.Errorf("got %v, want %v", a, c.IPv4)
	}
	a = addrs[1].IPNet
	if !a.IP.Equal(c.IPv6.Address) ||
		a.Mask.String() != c.IPv6.Netmask.String() {
		t.Errorf("got %v, want %v", a, c.IPv6)
	}
}

// TestTeardownVPNDevice tests teardownVPNDevice
func TestTeardownVPNDevice(t *testing.T) {
	c := vpnconfig.New()
	c.Device.Name = "tun0"

	// overwrite netlink functions
	device := ""
	down := false
	runLinkByName = func(name string) (netlink.Link, error) {
		device = name
		return &netlink.Device{}, nil
	}
	runLinkSetDown = func(netlink.Link) error {
		down = true
		return nil
	}

	// test
	teardownVPNDevice(c)
	if device != c.Device.Name {
		t.Errorf("got %s, want %s", device, c.Device.Name)
	}
	if !down {
		t.Errorf("got %t, want true", down)
	}
}

// TestVPNSetupSetupDNS tests setupDNS of VPNSetup
func TestVPNSetupSetupDNS(t *testing.T) {
	c := vpnconfig.New()
	c.Device.Name = "tun0"
	c.DNS.DefaultDomain = "mycompany.com"

	got := []string{}
	runResolvectl = func(cmd string) {
		got = append(got, cmd)
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
	runResolvectl = func(cmd string) {
		got = append(got, cmd)
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
	runCleanupCmd = func(cmd string) {
		got = append(got, cmd)
	}
	Cleanup("tun0", splitrt.NewConfig())
	want := []string{
		"resolvectl revert tun0",
		"ip link delete tun0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
