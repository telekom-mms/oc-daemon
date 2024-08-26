package splitrt

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/addrmon"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/internal/execs"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/vishvananda/netlink"
)

// TestSplitRoutingHandleDeviceUpdate tests handleDeviceUpdate of SplitRouting.
func TestSplitRoutingHandleDeviceUpdate(t *testing.T) {
	ctx := context.Background()
	s := NewSplitRouting(NewConfig(), vpnconfig.New())

	want := []string{"nothing else"}
	got := []string{"nothing else"}

	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	// test adding
	update := getTestDevMonUpdate()
	s.handleDeviceUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing
	update.Add = false
	s.handleDeviceUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding loopback device
	update = getTestDevMonUpdate()
	update.Type = "loopback"
	s.handleDeviceUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test adding vpn device
	update = getTestDevMonUpdate()
	update.Device = s.vpnconfig.Device.Name
	s.handleDeviceUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitRoutingHandleAddressUpdate tests handleAddressUpdate of SplitRouting.
func TestSplitRoutingHandleAddressUpdate(t *testing.T) {
	ctx := context.Background()

	// test with exclude
	vpnconf := vpnconfig.New()
	vpnconf.Split.ExcludeIPv4 = []*net.IPNet{
		{
			IP:   net.IPv4(0, 0, 0, 0),
			Mask: net.CIDRMask(32, 32),
		},
	}
	s := NewSplitRouting(NewConfig(), vpnconf)
	s.devices.Add(getTestDevMonUpdate())

	got := []string{}
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	// test adding
	want := []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.1/32 }",
	}
	update := getTestAddrMonUpdate(t, "192.168.1.1/32")
	s.handleAddressUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing
	got = []string{}
	want = []string{
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
	}
	update.Add = false
	s.handleAddressUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with exclude and virtual
	vpnconf = vpnconfig.New()
	vpnconf.Split.ExcludeIPv4 = []*net.IPNet{
		{
			IP:   net.IPv4(0, 0, 0, 0),
			Mask: net.CIDRMask(32, 32),
		},
	}
	vpnconf.Split.ExcludeVirtualSubnetsOnlyIPv4 = true
	s = NewSplitRouting(NewConfig(), vpnconf)
	devUp := getTestDevMonUpdate()
	devUp.Type = "virtual"
	s.devices.Add(devUp)

	got = []string{}

	// test adding
	want = []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.1/32 }",
	}
	update = getTestAddrMonUpdate(t, "192.168.1.1/32")
	s.handleAddressUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test double adding
	s.handleAddressUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing
	got = []string{}
	want = []string{
		"flush set inet oc-daemon-routing excludes4\n" +
			"flush set inet oc-daemon-routing excludes6\n",
	}
	update.Add = false
	s.handleAddressUpdate(ctx, update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitRoutingHandleDNSReport tests handleDNSReport of SplitRouting.
func TestSplitRoutingHandleDNSReport(t *testing.T) {
	ctx := context.Background()
	s := NewSplitRouting(NewConfig(), vpnconfig.New())

	got := []string{}
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, _ string, s string, _ ...string) ([]byte, []byte, error) {
		got = append(got, s)
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	// test ipv4
	report := dnsproxy.NewReport("example.com", netip.MustParseAddr("192.168.1.1"), 300)
	go s.handleDNSReport(ctx, report)
	<-report.Done()

	// test ipv6
	report = dnsproxy.NewReport("example.com", netip.MustParseAddr("2001::1"), 300)
	go s.handleDNSReport(ctx, report)
	<-report.Done()

	want := []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.1/32 }",
		"add element inet oc-daemon-routing excludes6 { 2001::1/128 }",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitRoutingStartStop tests Start and Stop of SplitRouting.
func TestSplitRoutingStartStop(t *testing.T) {
	// set dummy low level functions for testing
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(context.Context, string, string, ...string) ([]byte, []byte, error) {
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	oldRegisterAddrUpdates := addrmon.RegisterAddrUpdates
	addrmon.RegisterAddrUpdates = func(*addrmon.AddrMon) (chan netlink.AddrUpdate, error) {
		return nil, nil
	}
	defer func() { addrmon.RegisterAddrUpdates = oldRegisterAddrUpdates }()

	oldRegisterLinkUpdates := devmon.RegisterLinkUpdates
	devmon.RegisterLinkUpdates = func(*devmon.DevMon) (chan netlink.LinkUpdate, error) {
		return nil, nil
	}
	defer func() { devmon.RegisterLinkUpdates = oldRegisterLinkUpdates }()

	// test with new configs
	s := NewSplitRouting(NewConfig(), vpnconfig.New())
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.Stop()

	// test with excludes
	vpnconf := vpnconfig.New()
	vpnconf.Split.ExcludeIPv4 = []*net.IPNet{
		{
			IP:   net.IPv4(0, 0, 0, 0),
			Mask: net.CIDRMask(32, 32),
		},
		{
			IP:   net.IPv4(192, 168, 1, 1),
			Mask: net.CIDRMask(32, 32),
		},
	}
	vpnconf.Split.ExcludeIPv6 = []*net.IPNet{
		{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(128, 128),
		},
		{
			IP:   net.ParseIP("2000::1"),
			Mask: net.CIDRMask(128, 128),
		},
	}
	s = NewSplitRouting(NewConfig(), vpnconf)
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.Stop()

	// test with vpn address
	vpnconf = vpnconfig.New()
	vpnconf.IPv4.Address = net.IPv4(192, 168, 1, 1)
	vpnconf.IPv4.Netmask = net.CIDRMask(24, 32)
	s = NewSplitRouting(NewConfig(), vpnconf)
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.Stop()

	// test with events
	s = NewSplitRouting(NewConfig(), vpnconfig.New())
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.devmon.Updates() <- getTestDevMonUpdate()
	s.addrmon.Updates() <- getTestAddrMonUpdate(t, "192.168.1.1/32")
	report := dnsproxy.NewReport("example.com", netip.MustParseAddr("192.168.1.1"), 300)
	s.dnsreps <- report
	<-report.Done()
	s.Stop()

	// test with nft errors
	execs.RunCmd = func(context.Context, string, string, ...string) ([]byte, []byte, error) {
		return nil, nil, errors.New("test error")
	}
	s = NewSplitRouting(NewConfig(), vpnconfig.New())
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.Stop()
}

// TestSplitRoutingDNSReports tests DNSReports of SplitRouting.
func TestSplitRoutingDNSReports(t *testing.T) {
	s := NewSplitRouting(NewConfig(), vpnconfig.New())
	want := s.dnsreps
	got := s.DNSReports()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestSplitRoutingGetState tests GetState of SplitRouting.
func TestSplitRoutingGetState(t *testing.T) {
	s := NewSplitRouting(NewConfig(), vpnconfig.New())

	// set devices
	dev := &devmon.Update{
		Add:    true,
		Device: "test",
		Type:   "test",
		Index:  1,
	}
	s.devices.Add(dev)

	// set addresses
	addr := &addrmon.Update{
		Add:     true,
		Address: netip.MustParsePrefix("192.168.1.0/24"),
		Index:   1,
	}
	s.addrs.Add(addr)

	// set local excludes
	locals := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}
	s.locals.set(locals)

	// set static excludes
	static := netip.MustParsePrefix("10.1.0.0/24")
	s.excludes.s[static.String()] = static

	// set dynamic excludes
	dynamic := netip.MustParseAddr("10.2.0.1")
	s.excludes.d[dynamic] = &dynExclude{}

	// get and check state
	want := &State{
		Config:          NewConfig(),
		VPNConfig:       vpnconfig.New(),
		Devices:         []*devmon.Update{dev},
		Addresses:       []*addrmon.Update{addr},
		LocalExcludes:   []string{"10.0.0.0/24"},
		StaticExcludes:  []string{"10.1.0.0/24"},
		DynamicExcludes: []string{"10.2.0.1"},
	}
	got := s.GetState()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewSplitRouting tests NewSplitRouting.
func TestNewSplitRouting(t *testing.T) {
	config := NewConfig()
	vpnconf := vpnconfig.New()
	s := NewSplitRouting(config, vpnconf)
	if s.config != config {
		t.Errorf("got %p, want %p", s.config, config)
	}
	if s.vpnconfig != vpnconf {
		t.Errorf("got %p, want %p", s.vpnconfig, vpnconf)
	}
	if s.devmon == nil ||
		s.addrmon == nil ||
		s.devices == nil ||
		s.addrs == nil ||
		s.excludes == nil ||
		s.dnsreps == nil ||
		s.done == nil ||
		s.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}

// TestCleanup tests Cleanup.
func TestCleanup(t *testing.T) {
	got := []string{}

	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, cmd string, s string, arg ...string) ([]byte, []byte, error) {
		if s == "" {
			got = append(got, cmd+" "+strings.Join(arg, " "))
			return nil, nil, nil
		}
		got = append(got, cmd+" "+strings.Join(arg, " ")+" "+s)
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	Cleanup(context.Background(), NewConfig())
	want := []string{
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
