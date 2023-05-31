package splitrt

import (
	"net"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/addrmon"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/vishvananda/netlink"
)

// TestSplitRoutingHandleDeviceUpdate tests handleDeviceUpdate of SplitRouting
func TestSplitRoutingHandleDeviceUpdate(t *testing.T) {
	config := vpnconfig.New()
	s := NewSplitRouting(config)

	want := []string{"nothing else"}
	got := []string{"nothing else"}
	runNft = func(s string) {
		got = append(got, s)
	}

	// test adding
	update := getTestDevMonUpdate()
	s.handleDeviceUpdate(update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test removing
	update.Add = false
	s.handleDeviceUpdate(update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitRoutingHandleAddressUpdate tests handleAddressUpdate of SplitRouting
func TestSplitRoutingHandleAddressUpdate(t *testing.T) {
	config := vpnconfig.New()
	config.Split.ExcludeIPv4 = []*net.IPNet{
		{
			IP:   net.IPv4(0, 0, 0, 0),
			Mask: net.CIDRMask(32, 32),
		},
	}
	s := NewSplitRouting(config)
	s.devices.Add(getTestDevMonUpdate())

	got := []string{}
	runNft = func(s string) {
		got = append(got, s)
	}

	// test adding
	want := []string{
		"add element inet oc-daemon-routing excludes4 { 192.168.1.1/32 }",
	}
	update := getTestAddrMonUpdate("192.168.1.1/32")
	s.handleAddressUpdate(update)
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
	s.handleAddressUpdate(update)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitRoutingHandleDNSReport tests handleDNSReport of SplitRouting
func TestSplitRoutingHandleDNSReport(t *testing.T) {
	config := vpnconfig.New()
	s := NewSplitRouting(config)

	got := []string{}
	runNft = func(s string) {
		got = append(got, s)
	}

	// test ipv4
	report := dnsproxy.NewReport("example.com", net.ParseIP("192.168.1.1"), 300)
	go s.handleDNSReport(report)
	report.Wait()

	// test ipv6
	report = dnsproxy.NewReport("example.com", net.ParseIP("2001::1"), 300)
	go s.handleDNSReport(report)
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
	config := vpnconfig.New()
	s := NewSplitRouting(config)

	// set dummy low level functions for testing
	cmd := func(s string) {}
	runNft = cmd
	runCmd = cmd
	addrmon.RegisterAddrUpdates = func(*addrmon.AddrMon) chan netlink.AddrUpdate {
		return nil
	}
	devmon.RegisterLinkUpdates = func(*devmon.DevMon) chan netlink.LinkUpdate {
		return nil
	}

	s.Start()
	s.Stop()
}

// TestSplitRoutingDNSReports tests DNSReports of SplitRouting
func TestSplitRoutingDNSReports(t *testing.T) {
	config := vpnconfig.New()
	s := NewSplitRouting(config)
	want := s.dnsreps
	got := s.DNSReports()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewSplitRouting tests NewSplitRouting
func TestNewSplitRouting(t *testing.T) {
	config := vpnconfig.New()
	s := NewSplitRouting(config)
	if s.config != config {
		t.Errorf("got %p, want %p", s.config, config)
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
	cmd := func(s string) {
		got = append(got, s)
	}
	runCleanupCmd = cmd
	runCleanupNft = cmd
	Cleanup()
	want := []string{
		"ip -4 rule delete pref 2111",
		"ip -4 rule delete pref 2112",
		"ip -6 rule delete pref 2111",
		"ip -6 rule delete pref 2112",
		"ip -4 route flush table 42111",
		"ip -6 route flush table 42111",
		"delete table inet oc-daemon-routing",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
