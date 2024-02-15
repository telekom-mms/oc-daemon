package splitrt

import (
	"context"
	"errors"
	"net"
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

// TestSplitRoutingHandleDeviceUpdate tests handleDeviceUpdate of SplitRouting
func TestSplitRoutingHandleDeviceUpdate(t *testing.T) {
	ctx := context.Background()
	s := NewSplitRouting(NewConfig(), vpnconfig.New())

	want := []string{"nothing else"}
	got := []string{"nothing else"}

	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, s)
		return nil
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

// TestSplitRoutingHandleAddressUpdate tests handleAddressUpdate of SplitRouting
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
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, s)
		return nil
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

	// test with exculde and virtual
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

// TestSplitRoutingHandleDNSReport tests handleDNSReport of SplitRouting
func TestSplitRoutingHandleDNSReport(t *testing.T) {
	ctx := context.Background()
	s := NewSplitRouting(NewConfig(), vpnconfig.New())

	got := []string{}
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		got = append(got, s)
		return nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	// test ipv4
	report := dnsproxy.NewReport("example.com", net.ParseIP("192.168.1.1"), 300)
	go s.handleDNSReport(ctx, report)
	report.Wait()

	// test ipv6
	report = dnsproxy.NewReport("example.com", net.ParseIP("2001::1"), 300)
	go s.handleDNSReport(ctx, report)
	report.Wait()

	want := []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.1/32 }",
		"add element inet oc-daemon-routing excludes6 { 2001::1/128 }",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitRoutingStartStop tests Start and Stop of SplitRouting
func TestSplitRoutingStartStop(t *testing.T) {
	// set dummy low level functions for testing
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		return nil
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
			Mask: net.CIDRMask(32, 32),
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
	report := dnsproxy.NewReport("example.com", net.ParseIP("192.168.1.1"), 300)
	s.dnsreps <- report
	report.Wait()
	s.Stop()

	// test with nft errors
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		return errors.New("test error")
	}
	s = NewSplitRouting(NewConfig(), vpnconfig.New())
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.Stop()
}

// TestSplitRoutingDNSReports tests DNSReports of SplitRouting
func TestSplitRoutingDNSReports(t *testing.T) {
	s := NewSplitRouting(NewConfig(), vpnconfig.New())
	want := s.dnsreps
	got := s.DNSReports()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewSplitRouting tests NewSplitRouting
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

// TestCleanup tests Cleanup
func TestCleanup(t *testing.T) {
	got := []string{}

	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
		if s == "" {
			got = append(got, cmd+" "+strings.Join(arg, " "))
			return nil
		}
		got = append(got, cmd+" "+strings.Join(arg, " ")+" "+s)
		return nil
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
