package splitrt

import (
	"cmp"
	"net/netip"
	"reflect"
	"slices"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/addrmon"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/vishvananda/netlink"
)

// TestSplitRoutingHandleDeviceUpdate tests handleDeviceUpdate of SplitRouting.
func TestSplitRoutingHandleDeviceUpdate(t *testing.T) {
	s := NewSplitRouting(daemoncfg.NewConfig(), make(chan *dnsproxy.Report))

	// test helper
	want := []netip.Prefix{}
	got := []netip.Prefix{}
	test := func(t *testing.T, update *devmon.Update) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.handleDeviceUpdate(update)
		}()
		select {
		case got = <-s.Prefixes():
		case <-done:
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test adding
	update := getTestDevMonUpdate()
	t.Run("adding", func(t *testing.T) { test(t, update) })

	// test removing
	update = getTestDevMonUpdate()
	update.Add = false
	t.Run("removing", func(t *testing.T) { test(t, update) })

	// test adding loopback device
	update = getTestDevMonUpdate()
	update.Type = "loopback"
	t.Run("adding loopback", func(t *testing.T) { test(t, update) })

	// test adding vpn device
	update = getTestDevMonUpdate()
	update.Device = s.config.VPNConfig.Device.Name
	t.Run("adding vpn device", func(t *testing.T) { test(t, update) })
}

// TestSplitRoutingHandleAddressUpdate tests handleAddressUpdate of SplitRouting.
func TestSplitRoutingHandleAddressUpdate(t *testing.T) {

	// test with exclude
	conf := daemoncfg.NewConfig()
	conf.VPNConfig.Split.ExcludeIPv4 = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/32"),
	}
	s := NewSplitRouting(conf, make(chan *dnsproxy.Report))
	s.devices.Add(getTestDevMonUpdate())

	// test helper
	want := []netip.Prefix{}
	got := []netip.Prefix{}
	test := func(t *testing.T, update *addrmon.Update) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.handleAddressUpdate(update)
		}()
		select {
		case got = <-s.Prefixes():
		case <-done:
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test adding
	update := getTestAddrMonUpdate(t, "192.168.1.1/32")
	want = []netip.Prefix{
		netip.MustParsePrefix("192.168.1.1/32"),
	}
	got = []netip.Prefix{}
	t.Run("adding with exclude", func(t *testing.T) { test(t, update) })

	// test removing
	update.Add = false
	want = []netip.Prefix{}
	got = []netip.Prefix{}
	t.Run("removing with exclude", func(t *testing.T) { test(t, update) })

	// test with exclude and virtual
	conf = daemoncfg.NewConfig()
	conf.VPNConfig.Split.ExcludeIPv4 = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/32"),
	}
	conf.VPNConfig.Split.ExcludeVirtualSubnetsOnlyIPv4 = true
	s = NewSplitRouting(conf, make(chan *dnsproxy.Report))
	devUp := getTestDevMonUpdate()
	devUp.Type = "virtual"
	s.devices.Add(devUp)

	// test adding
	update = getTestAddrMonUpdate(t, "192.168.1.1/32")
	want = []netip.Prefix{
		netip.MustParsePrefix("192.168.1.1/32"),
	}
	got = []netip.Prefix{}
	t.Run("adding with exclude and virtual", func(t *testing.T) { test(t, update) })

	// test double adding
	want = []netip.Prefix{}
	got = []netip.Prefix{}
	t.Run("double adding with exclude and virtual", func(t *testing.T) { test(t, update) })

	// test removing
	update.Add = false
	want = []netip.Prefix{}
	got = []netip.Prefix{}
	t.Run("removing with exclude and virtual", func(t *testing.T) { test(t, update) })
}

// TestSplitRoutingHandleDNSReport tests handleDNSReport of SplitRouting.
func TestSplitRoutingHandleDNSReport(t *testing.T) {
	s := NewSplitRouting(daemoncfg.NewConfig(), make(chan *dnsproxy.Report))

	// test helper
	want := []netip.Prefix{}
	got := []netip.Prefix{}
	test := func(t *testing.T, report *dnsproxy.Report) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.handleDNSReport(report)
			<-report.Done()
		}()
		select {
		case got = <-s.Prefixes():
		case <-done:
		}
		cmpPrefixes := func(a, b netip.Prefix) int {
			c := a.Addr().Compare(b.Addr())
			if c == 0 {
				return cmp.Compare(a.Bits(), b.Bits())
			}
			return c
		}
		slices.SortFunc(want, cmpPrefixes)
		slices.SortFunc(got, cmpPrefixes)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test ipv4
	report := dnsproxy.NewReport("example.com", netip.MustParseAddr("192.168.1.1"), 300)
	want = []netip.Prefix{
		netip.MustParsePrefix("192.168.1.1/32"),
	}
	got = []netip.Prefix{}
	t.Run("ipv4", func(t *testing.T) { test(t, report) })

	// test ipv6
	report = dnsproxy.NewReport("example.com", netip.MustParseAddr("2001::1"), 300)
	want = []netip.Prefix{
		netip.MustParsePrefix("192.168.1.1/32"),
		netip.MustParsePrefix("2001::1/128"),
	}
	got = []netip.Prefix{}
	t.Run("ipv6", func(t *testing.T) { test(t, report) })
}

// TestSplitRoutingStartStop tests Start and Stop of SplitRouting.
func TestSplitRoutingStartStop(t *testing.T) {
	// set dummy low level functions for testing
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

	// test with new config
	s := NewSplitRouting(daemoncfg.NewConfig(), make(chan *dnsproxy.Report))
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.Stop()

	// test with excludes
	conf := daemoncfg.NewConfig()
	conf.VPNConfig.Gateway = netip.MustParseAddr("10.0.0.1")
	conf.VPNConfig.IPv4 = netip.MustParsePrefix("192.168.0.1/24")
	conf.VPNConfig.Split.ExcludeIPv4 = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/32"),
		netip.MustParsePrefix("192.168.1.1/32"),
	}
	conf.VPNConfig.Split.ExcludeIPv6 = []netip.Prefix{
		netip.MustParsePrefix("::/128"),
		netip.MustParsePrefix("2000::1/128"),
	}
	s = NewSplitRouting(conf, make(chan *dnsproxy.Report))

	want := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.1/32"),
		netip.MustParsePrefix("192.168.1.1/32"),
		netip.MustParsePrefix("2000::1/128"),
	}
	got := []netip.Prefix{}

	done := make(chan struct{})
	go func(prefixes <-chan []netip.Prefix) {
		defer close(done)
		got = <-prefixes

	}(s.Prefixes())
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	<-done
	s.Stop()

	cmpPrefixes := func(a, b netip.Prefix) int {
		c := a.Addr().Compare(b.Addr())
		if c == 0 {
			return cmp.Compare(a.Bits(), b.Bits())
		}
		return c
	}
	slices.SortFunc(want, cmpPrefixes)
	slices.SortFunc(got, cmpPrefixes)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with events
	dnsReports := make(chan *dnsproxy.Report)
	s = NewSplitRouting(daemoncfg.NewConfig(), dnsReports)

	want = []netip.Prefix{
		netip.MustParsePrefix("192.168.1.1/32"),
	}
	got = []netip.Prefix{}

	done = make(chan struct{})
	go func(prefixes <-chan []netip.Prefix) {
		defer close(done)
		for p := range prefixes {
			got = p
		}
	}(s.Prefixes())
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.devmon.Updates() <- getTestDevMonUpdate()
	s.addrmon.Updates() <- getTestAddrMonUpdate(t, "192.168.1.1/32")
	report := dnsproxy.NewReport("example.com", netip.MustParseAddr("192.168.1.1"), 300)
	dnsReports <- report
	<-report.Done()
	s.Stop()

	<-done
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitRoutingPrefixes tests Prefixes of SplitRouting.
func TestSplitRoutingPrefixes(t *testing.T) {
	s := NewSplitRouting(daemoncfg.NewConfig(), make(chan *dnsproxy.Report))
	want := s.prefixes
	got := s.Prefixes()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestSplitRoutingGetState tests GetState of SplitRouting.
func TestSplitRoutingGetState(t *testing.T) {
	s := NewSplitRouting(daemoncfg.NewConfig(), make(chan *dnsproxy.Report))

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
	config := daemoncfg.NewConfig()
	dnsReports := make(chan *dnsproxy.Report)
	s := NewSplitRouting(config, dnsReports)
	if s.config != config {
		t.Errorf("got %p, want %p", s.config, config)
	}
	if s.dnsreps != dnsReports {
		t.Errorf("got %p, want %p", s.dnsreps, dnsReports)
	}
	if s.devmon == nil ||
		s.addrmon == nil ||
		s.devices == nil ||
		s.addrs == nil ||
		s.excludes == nil ||
		s.prefixes == nil ||
		s.done == nil ||
		s.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
