package trafpol

import (
	"context"
	"net/netip"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/cpd"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/execs"
	"github.com/vishvananda/netlink"
)

// TestTrafPolHandleDeviceUpdate tests handleDeviceUpdate of TrafPol.
func TestTrafPolHandleDeviceUpdate(_ *testing.T) {
	tp := NewTrafPol(daemoncfg.NewConfig())
	ctx := context.Background()

	// test adding
	update := &devmon.Update{
		Device: "test0",
		Add:    true,
	}
	tp.handleDeviceUpdate(ctx, update)

	// test removing
	update.Add = false
	tp.handleDeviceUpdate(ctx, update)
}

// TestTrafPolHandleDNSUpdate tests handleDNSUpdate of TrafPol.
func TestTrafPolHandleDNSUpdate(_ *testing.T) {
	tp := NewTrafPol(daemoncfg.NewConfig())

	tp.resolver.Start()
	defer tp.resolver.Stop()
	tp.cpd.Start()
	defer tp.cpd.Stop()

	tp.handleDNSUpdate()
}

// TestTrafPolHandleCPDReport tests handleCPDReport of TrafPol.
func TestTrafPolHandleCPDReport(t *testing.T) {
	tp := NewTrafPol(daemoncfg.NewConfig())
	ctx := context.Background()

	tp.resolver.Start()
	defer tp.resolver.Stop()

	var nftMutex sync.Mutex
	nftCmds := []string{}
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(_ context.Context, cmd string, stdin string, args ...string) ([]byte, []byte, error) {
		nftMutex.Lock()
		defer nftMutex.Unlock()
		nftCmds = append(nftCmds, cmd+" "+strings.Join(args, " ")+" "+stdin)
		return nil, nil, nil
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	getNftCmds := func() []string {
		nftMutex.Lock()
		defer nftMutex.Unlock()
		return append(nftCmds[:0:0], nftCmds...)
	}

	// test not detected
	report := &cpd.Report{}
	tp.handleCPDReport(ctx, report)

	want := []string{}
	got := getNftCmds()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test detected
	report.Detected = true
	tp.handleCPDReport(ctx, report)

	want = []string{
		"nft -f - flush set inet oc-daemon-filter allowports\n" +
			"add element inet oc-daemon-filter allowports { 80 }\n" +
			"add element inet oc-daemon-filter allowports { 443 }\n",
	}
	got = getNftCmds()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test not detected any more
	report.Detected = false
	tp.handleCPDReport(ctx, report)

	want = []string{
		"nft -f - flush set inet oc-daemon-filter allowports\n" +
			"add element inet oc-daemon-filter allowports { 80 }\n" +
			"add element inet oc-daemon-filter allowports { 443 }\n",
		"nft -f - flush set inet oc-daemon-filter allowports\n",
	}
	got = getNftCmds()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestTrafPolStartEvents tests start of TrafPol, events.
func TestTrafPolStartEvents(t *testing.T) {
	// set dummy low level function for devmon
	oldRegisterLinkUpdates := devmon.RegisterLinkUpdates
	devmon.RegisterLinkUpdates = func(*devmon.DevMon) (chan netlink.LinkUpdate, error) {
		return nil, nil
	}
	defer func() { devmon.RegisterLinkUpdates = oldRegisterLinkUpdates }()

	tp := NewTrafPol(daemoncfg.NewConfig())
	if err := tp.Start(); err != nil {
		t.Fatal(err)
	}
	tp.devmon.Updates() <- &devmon.Update{Type: "device"}
	tp.dnsmon.Updates() <- struct{}{}
	tp.cpd.Results() <- &cpd.Report{}
	tp.resolvUp <- &ResolvedName{}
	tp.Stop()
}

// TestTrafPolGetAllowedHostsIPs tests getAllowedHostsIPs of TrafPol.
func TestTrafPolGetAllowedHostsIPs(t *testing.T) {
	// create trafpol with allowed addresses
	c := daemoncfg.NewConfig()
	c.TrafficPolicing.AllowedHosts = append(c.TrafficPolicing.AllowedHosts,
		"192.168.2.0/24", "2001:DB8:2::/64")
	tp := NewTrafPol(c)

	// add allowed names
	tp.allowNames.Add("example.com", []netip.Addr{
		netip.MustParseAddr("192.168.1.1"),
		netip.MustParseAddr("2001:DB8:1::1"),
	})

	// wanted IPs
	want := []netip.Prefix{}
	for _, addr := range []string{
		"192.168.1.1/32",
		"192.168.2.0/24",
		"2001:db8:1::1/128",
		"2001:db8:2::/64",
	} {
		prefix := netip.MustParsePrefix(addr)
		want = append(want, prefix)
	}

	// get IPs
	got := tp.getAllowedHostsIPs()
	sort.Slice(got, func(i, j int) bool {
		return got[i].String() < got[j].String()
	})

	// check
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i].String() != want[i].String() {
			t.Errorf("got %v, want %v", got[i], want[i])
		}
	}
}

// TestTrafPolStartStop tests Start and Stop of TrafPol.
func TestTrafPolStartStop(t *testing.T) {
	// set dummy low level function for devmon
	oldRegisterLinkUpdates := devmon.RegisterLinkUpdates
	devmon.RegisterLinkUpdates = func(*devmon.DevMon) (chan netlink.LinkUpdate, error) {
		return nil, nil
	}
	defer func() { devmon.RegisterLinkUpdates = oldRegisterLinkUpdates }()

	tp := NewTrafPol(daemoncfg.NewConfig())
	if err := tp.Start(); err != nil {
		t.Fatal(err)
	}
	tp.Stop()
}

// TestTrafPolAddRemoveAllowedAddr tests AddAllowedAddr and RemoveAllowedAddr of Trafpol.
func TestTrafPolAddRemoveAllowedAddr(t *testing.T) {
	// set dummy low level function for devmon
	oldRegisterLinkUpdates := devmon.RegisterLinkUpdates
	devmon.RegisterLinkUpdates = func(*devmon.DevMon) (chan netlink.LinkUpdate, error) {
		return nil, nil
	}
	defer func() { devmon.RegisterLinkUpdates = oldRegisterLinkUpdates }()

	tp := NewTrafPol(daemoncfg.NewConfig())
	if err := tp.Start(); err != nil {
		t.Fatal(err)
	}

	// add ipv4 address
	prefix := netip.MustParsePrefix("192.168.1.1/32")
	if ok := tp.AddAllowedAddr(prefix.Addr()); !ok {
		t.Errorf("address not added")
	}

	want := []netip.Prefix{prefix}
	got := tp.allowAddrs.List()
	if !slices.Equal(got, want) {
		t.Errorf("got %s, want %s", got, want)
	}

	// add ipv4 address again
	if ok := tp.AddAllowedAddr(prefix.Addr()); ok {
		t.Errorf("existing address should not be added again")
	}

	// remove ipv4 address
	if ok := tp.RemoveAllowedAddr(prefix.Addr()); !ok {
		t.Errorf("address not removed")
	}

	want = []netip.Prefix{}
	got = tp.allowAddrs.List()
	if !slices.Equal(got, want) {
		t.Errorf("got %s, want %s", got, want)
	}

	// remove ipv4 address again
	if ok := tp.RemoveAllowedAddr(prefix.Addr()); ok {
		t.Errorf("not existing address should not be removed")
	}

	// add/remove ipv6 address
	ip := netip.MustParseAddr("2001:DB8:1::1")
	if ok := tp.AddAllowedAddr(ip); !ok {
		t.Errorf("address not added")
	}
	if ok := tp.RemoveAllowedAddr(ip); !ok {
		t.Errorf("address not removed")
	}

	tp.Stop()
}

// TestTrafPolGetState tests GetState of Trafpol.
func TestTrafPolGetState(t *testing.T) {
	// set dummy low level function for devmon
	oldRegisterLinkUpdates := devmon.RegisterLinkUpdates
	devmon.RegisterLinkUpdates = func(*devmon.DevMon) (chan netlink.LinkUpdate, error) {
		return nil, nil
	}
	defer func() { devmon.RegisterLinkUpdates = oldRegisterLinkUpdates }()

	// start trafpol
	tp := NewTrafPol(daemoncfg.NewConfig())
	if err := tp.Start(); err != nil {
		t.Fatal(err)
	}

	// check state
	if tp.GetState() == nil {
		t.Errorf("got invalid state")
	}

	// stop trafpol
	tp.Stop()
}

// TestNewTrafPol tests NewTrafPol.
func TestNewTrafPol(t *testing.T) {
	c := daemoncfg.NewConfig()
	c.TrafficPolicing.AllowedHosts = append(c.TrafficPolicing.AllowedHosts,
		"192.168.1.1", "192.168.2.0/24",
		"2001:DB8:1::1", "2001:DB8:2::/64")
	tp := NewTrafPol(c)
	if tp == nil ||
		tp.devmon == nil ||
		tp.dnsmon == nil ||
		tp.cpd == nil ||
		tp.allowDevs == nil ||
		tp.allowAddrs == nil ||
		tp.allowNames == nil ||
		tp.resolver == nil ||
		tp.resolvUp == nil ||
		tp.cmds == nil ||
		tp.loopDone == nil ||
		tp.done == nil {

		t.Errorf("got nil, want != nil")
	}
}

// TestCleanup tests Cleanup.
func TestCleanup(t *testing.T) {
	want := []string{
		"nft -f - delete table inet oc-daemon-filter",
	}
	got := []string{}
	execs.RunCmd = func(_ context.Context, cmd string, _ string, args ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(args, " "))
		return nil, nil, nil
	}
	Cleanup(context.Background(), daemoncfg.NewConfig())
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
