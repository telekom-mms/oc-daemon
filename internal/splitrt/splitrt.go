// Package splitrt contains the split routing.
package splitrt

import (
	"context"
	"fmt"
	"net/netip"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/addrmon"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// SplitRouting is a split routing configuration.
type SplitRouting struct {
	config    *Config
	vpnconfig *vpnconfig.Config
	devmon    *devmon.DevMon
	addrmon   *addrmon.AddrMon
	devices   *Devices
	addrs     *Addresses
	locals    []netip.Prefix
	excludes  *Excludes
	dnsreps   chan *dnsproxy.Report
	done      chan struct{}
	closed    chan struct{}
}

// setupRouting sets up routing using config.
func (s *SplitRouting) setupRouting(ctx context.Context) {
	// prepare netfilter and excludes
	setRoutingRules(ctx, s.config.FirewallMark)

	// convert to netip
	pre4 := netip.Prefix{}
	if ipv4, ok := netip.AddrFromSlice(s.vpnconfig.IPv4.Address.To4()); ok {
		one4, _ := s.vpnconfig.IPv4.Netmask.Size()
		pre4 = netip.PrefixFrom(ipv4, one4)
	}
	pre6 := netip.Prefix{}
	if ipv6, ok := netip.AddrFromSlice(s.vpnconfig.IPv6.Address); ok {
		one6, _ := s.vpnconfig.IPv6.Netmask.Size()
		pre6 = netip.PrefixFrom(ipv6, one6)
	}

	// filter non-local traffic to vpn addresses
	addLocalAddressesIPv4(ctx, s.vpnconfig.Device.Name, []netip.Prefix{pre4})
	addLocalAddressesIPv6(ctx, s.vpnconfig.Device.Name, []netip.Prefix{pre6})

	// reject unsupported ip versions on vpn
	if !pre6.IsValid() {
		rejectIPv6(ctx, s.vpnconfig.Device.Name)
	}
	if !pre4.IsValid() {
		rejectIPv4(ctx, s.vpnconfig.Device.Name)
	}

	// add excludes
	s.excludes.Start()

	// add gateway to static excludes
	if s.vpnconfig.Gateway != nil {
		g := netip.MustParseAddr(s.vpnconfig.Gateway.String())
		gateway := netip.PrefixFrom(g, g.BitLen())
		s.excludes.AddStatic(ctx, gateway)
	}

	// add static IPv4 excludes
	for _, e := range s.vpnconfig.Split.ExcludeIPv4 {
		if e.String() == "0.0.0.0/32" {
			continue
		}
		p := netip.MustParsePrefix(e.String())
		s.excludes.AddStatic(ctx, p)
	}

	// add static IPv6 excludes
	for _, e := range s.vpnconfig.Split.ExcludeIPv6 {
		// TODO: does ::/128 exist?
		if e.String() == "::/128" {
			continue
		}
		p := netip.MustParsePrefix(e.String())
		s.excludes.AddStatic(ctx, p)
	}

	// setup routing
	addDefaultRouteIPv4(ctx, s.vpnconfig.Device.Name, s.config.RoutingTable,
		s.config.RulePriority1, s.config.FirewallMark, s.config.RulePriority2)
	addDefaultRouteIPv6(ctx, s.vpnconfig.Device.Name, s.config.RoutingTable,
		s.config.RulePriority1, s.config.FirewallMark, s.config.RulePriority2)

}

// teardownRouting tears down the routing configuration.
func (s *SplitRouting) teardownRouting(ctx context.Context) {
	deleteDefaultRouteIPv4(ctx, s.vpnconfig.Device.Name, s.config.RoutingTable)
	deleteDefaultRouteIPv6(ctx, s.vpnconfig.Device.Name, s.config.RoutingTable)
	unsetRoutingRules(ctx)

	// remove excludes
	s.excludes.Stop()
}

// excludeSettings returns whether local (virtual) networks should be excluded.
func (s *SplitRouting) excludeLocalNetworks() (exclude bool, virtual bool) {
	for _, e := range s.vpnconfig.Split.ExcludeIPv4 {
		if e.String() == "0.0.0.0/32" {
			exclude = true
		}
	}
	if s.vpnconfig.Split.ExcludeVirtualSubnetsOnlyIPv4 {
		virtual = true
	}
	return
}

// updateLocalNetworkExcludes updates the local network split excludes.
func (s *SplitRouting) updateLocalNetworkExcludes(ctx context.Context) {
	exclude, virtual := s.excludeLocalNetworks()

	// stop if exclude local networks is disabled
	if !exclude {
		return
	}

	// get local (virtual) network devices
	devs := s.devices.GetVirtual()
	if !virtual {
		devs = s.devices.GetAll()
	}

	// get addresses of these devices
	excludes := []netip.Prefix{}
	for _, d := range devs {
		excludes = append(excludes, s.addrs.Get(d)...)
	}

	// determine changes
	// TODO: move s.locals into excludes?
	isIn := func(n netip.Prefix, nets []netip.Prefix) bool {
		for _, net := range nets {
			if n.String() == net.String() {
				return true
			}
		}
		return false
	}

	// add new excludes
	for _, e := range excludes {
		if !isIn(e, s.locals) {
			s.excludes.AddStatic(ctx, e)
		}
	}

	// remove old excludes
	for _, l := range s.locals {
		if !isIn(l, excludes) {
			s.excludes.RemoveStatic(ctx, l)
		}
	}

	// save local excludes
	s.locals = excludes
	log.WithField("locals", s.locals).Debug("SplitRouting updated exclude local networks")
}

// handleDeviceUpdate handles a device update from the device monitor.
func (s *SplitRouting) handleDeviceUpdate(ctx context.Context, u *devmon.Update) {
	log.WithField("update", u).Debug("SplitRouting got device update")

	if u.Add {
		if u.Type == "loopback" {
			// skip loopback devices
			return
		}
		if u.Device == s.vpnconfig.Device.Name {
			// skip vpn tunnel device, so we do not use it for
			// split excludes
			return
		}
		s.devices.Add(u)
	} else {
		s.devices.Remove(u)
	}
	s.updateLocalNetworkExcludes(ctx)
}

// handleAddressUpdate handles an address update from the address monitor.
func (s *SplitRouting) handleAddressUpdate(ctx context.Context, u *addrmon.Update) {
	log.WithField("update", u).Debug("SplitRouting got address update")

	if u.Add {
		s.addrs.Add(u)
	} else {
		s.addrs.Remove(u)
	}
	s.updateLocalNetworkExcludes(ctx)
}

// handleDNSReport handles a DNS report.
func (s *SplitRouting) handleDNSReport(ctx context.Context, r *dnsproxy.Report) {
	defer r.Close()
	log.WithField("report", r).Debug("SplitRouting handling DNS report")

	s.excludes.AddDynamic(ctx, netip.PrefixFrom(r.IP, r.IP.BitLen()), r.TTL)
}

// start starts split routing.
func (s *SplitRouting) start(ctx context.Context) {
	defer close(s.closed)
	defer s.teardownRouting(ctx)
	defer s.devmon.Stop()
	defer s.addrmon.Stop()

	// main loop
	for {
		select {
		case u := <-s.devmon.Updates():
			s.handleDeviceUpdate(ctx, u)
		case u := <-s.addrmon.Updates():
			s.handleAddressUpdate(ctx, u)
		case r := <-s.dnsreps:
			s.handleDNSReport(ctx, r)
		case <-s.done:
			return
		}
	}
}

// Start starts split routing.
func (s *SplitRouting) Start() error {
	log.Debug("SplitRouting starting")

	// create context
	ctx := context.Background()

	// configure routing
	s.setupRouting(ctx)

	// start device monitor
	if err := s.devmon.Start(); err != nil {
		s.teardownRouting(ctx)
		return fmt.Errorf("SplitRouting could not start DevMon: %w", err)
	}

	// start address monitor
	if err := s.addrmon.Start(); err != nil {
		s.devmon.Stop()
		s.teardownRouting(ctx)
		return fmt.Errorf("SplitRouting could not start AddrMon: %w", err)
	}

	go s.start(ctx)
	return nil
}

// Stop stops split routing.
func (s *SplitRouting) Stop() {
	close(s.done)
	<-s.closed
	log.Debug("SplitRouting stopped")
}

// DNSReports returns the channel for dns reports.
func (s *SplitRouting) DNSReports() chan *dnsproxy.Report {
	return s.dnsreps
}

// NewSplitRouting returns a new SplitRouting.
func NewSplitRouting(config *Config, vpnconfig *vpnconfig.Config) *SplitRouting {
	return &SplitRouting{
		config:    config,
		vpnconfig: vpnconfig,
		devmon:    devmon.NewDevMon(),
		addrmon:   addrmon.NewAddrMon(),
		devices:   NewDevices(),
		addrs:     NewAddresses(),
		excludes:  NewExcludes(),
		dnsreps:   make(chan *dnsproxy.Report),
		done:      make(chan struct{}),
		closed:    make(chan struct{}),
	}
}

// Cleanup cleans up old configuration after a failed shutdown.
func Cleanup(ctx context.Context, config *Config) {
	cleanupRouting(ctx, config.RoutingTable, config.RulePriority1,
		config.RulePriority2)
	cleanupRoutingRules(ctx)
}
