// Package splitrt contains the split routing.
package splitrt

import (
	"fmt"
	"net/netip"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/addrmon"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
)

// State is the internal state.
type State struct {
	Devices         []*devmon.Update
	Addresses       []*addrmon.Update
	LocalExcludes   []string
	StaticExcludes  []string
	DynamicExcludes []string
}

// locals are local excludes.
type locals struct {
	sync.Mutex
	l []netip.Prefix
}

// get returns the local excludes, returned slice should not be modified.
func (l *locals) get() []netip.Prefix {
	l.Lock()
	defer l.Unlock()

	return l.l
}

// set sets the local excludes.
func (l *locals) set(locals []netip.Prefix) {
	l.Lock()
	defer l.Unlock()

	l.l = locals
}

// SplitRouting is a split routing configuration.
type SplitRouting struct {
	config   *daemoncfg.Config
	devmon   *devmon.DevMon
	addrmon  *addrmon.AddrMon
	devices  *Devices
	addrs    *Addresses
	locals   locals
	excludes *Excludes
	prefixes chan []netip.Prefix
	dnsreps  <-chan *dnsproxy.Report
	done     chan struct{}
	closed   chan struct{}
}

// excludeLocalNetworks returns whether local (virtual) networks should be excluded.
func (s *SplitRouting) excludeLocalNetworks() (exclude bool, virtual bool) {
	for _, e := range s.config.VPNConfig.Split.ExcludeIPv4 {
		if e.String() == "0.0.0.0/32" {
			exclude = true
		}
	}
	if s.config.VPNConfig.Split.ExcludeVirtualSubnetsOnlyIPv4 {
		virtual = true
	}
	return
}

// sendPrefixes sends the current prefixes over the prefixes channel.
func (s *SplitRouting) sendPrefixes(p []netip.Prefix) {
	select {
	case s.prefixes <- p:
	case <-s.done:
	}
}

// updateLocalNetworkExcludes updates the local network split excludes.
func (s *SplitRouting) updateLocalNetworkExcludes() {
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
	updated := false
	for _, e := range excludes {
		if !isIn(e, s.locals.get()) {
			updated = s.excludes.AddStatic(e) || updated
		}
	}

	// remove old excludes
	for _, l := range s.locals.get() {
		if !isIn(l, excludes) {
			updated = s.excludes.RemoveStatic(l) || updated
		}
	}

	if updated {
		// signal update
		s.sendPrefixes(s.excludes.GetPrefixes())
	}

	// save local excludes
	s.locals.set(excludes)
	log.WithField("locals", s.locals.get()).Debug("SplitRouting updated exclude local networks")
}

// handleDeviceUpdate handles a device update from the device monitor.
func (s *SplitRouting) handleDeviceUpdate(u *devmon.Update) {
	log.WithField("update", u).Debug("SplitRouting got device update")

	if u.Add {
		if u.Type == "loopback" {
			// skip loopback devices
			return
		}
		if u.Device == s.config.VPNConfig.Device.Name {
			// skip vpn tunnel device, so we do not use it for
			// split excludes
			return
		}
		s.devices.Add(u)
	} else {
		s.devices.Remove(u)
	}
	s.updateLocalNetworkExcludes()
}

// handleAddressUpdate handles an address update from the address monitor.
func (s *SplitRouting) handleAddressUpdate(u *addrmon.Update) {
	log.WithField("update", u).Debug("SplitRouting got address update")

	if u.Add {
		s.addrs.Add(u)
	} else {
		s.addrs.Remove(u)
	}
	s.updateLocalNetworkExcludes()
}

// handleDNSReport handles a DNS report.
func (s *SplitRouting) handleDNSReport(r *dnsproxy.Report) {
	defer r.Close()
	log.WithField("report", r).Debug("SplitRouting handling DNS report")

	exclude := netip.PrefixFrom(r.IP, r.IP.BitLen())
	if s.excludes.AddDynamic(exclude, r.TTL) {
		// signal update
		s.sendPrefixes(s.excludes.GetPrefixes())
	}
}

// start starts split routing.
func (s *SplitRouting) start() {
	defer close(s.closed)
	defer close(s.prefixes)
	defer s.devmon.Stop()
	defer s.addrmon.Stop()

	// send initial prefixes
	if prefixes := s.excludes.GetPrefixes(); len(prefixes) > 0 {
		s.sendPrefixes(prefixes)
	}

	// main loop
	timer := time.NewTimer(excludesTimer * time.Second)
	for {
		select {
		case u := <-s.devmon.Updates():
			s.handleDeviceUpdate(u)
		case u := <-s.addrmon.Updates():
			s.handleAddressUpdate(u)
		case r := <-s.dnsreps:
			s.handleDNSReport(r)
		case <-timer.C:
			s.excludes.cleanup()
			timer.Reset(excludesTimer * time.Second)
		case <-s.done:
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

// Start starts split routing.
func (s *SplitRouting) Start() error {
	log.Debug("SplitRouting starting")

	// start device monitor
	if err := s.devmon.Start(); err != nil {
		return fmt.Errorf("SplitRouting could not start DevMon: %w", err)
	}

	// start address monitor
	if err := s.addrmon.Start(); err != nil {
		s.devmon.Stop()
		return fmt.Errorf("SplitRouting could not start AddrMon: %w", err)
	}

	// add gateway to static excludes
	if s.config.VPNConfig.Gateway.IsValid() {
		gateway := netip.PrefixFrom(s.config.VPNConfig.Gateway,
			s.config.VPNConfig.Gateway.BitLen())
		s.excludes.AddStatic(gateway)
	}

	// add static IPv4 excludes
	for _, e := range s.config.VPNConfig.Split.ExcludeIPv4 {
		if e.String() == "0.0.0.0/32" {
			continue
		}
		s.excludes.AddStatic(e)
	}

	// add static IPv6 excludes
	for _, e := range s.config.VPNConfig.Split.ExcludeIPv6 {
		// TODO: does ::/128 exist?
		if e.String() == "::/128" {
			continue
		}
		s.excludes.AddStatic(e)
	}

	go s.start()
	return nil
}

// Stop stops split routing.
func (s *SplitRouting) Stop() {
	close(s.done)
	<-s.closed
	log.Debug("SplitRouting stopped")
}

// Prefixes returns the channel for the exclude prefixes.
func (s *SplitRouting) Prefixes() <-chan []netip.Prefix {
	return s.prefixes
}

// GetState returns the internal state.
func (s *SplitRouting) GetState() *State {
	var locals []string
	for _, l := range s.locals.get() {
		locals = append(locals, l.String())
	}
	static, dynamic := s.excludes.List()
	return &State{
		Devices:         s.devices.List(),
		Addresses:       s.addrs.List(),
		LocalExcludes:   locals,
		StaticExcludes:  static,
		DynamicExcludes: dynamic,
	}
}

// NewSplitRouting returns a new SplitRouting.
func NewSplitRouting(config *daemoncfg.Config, dnsReports <-chan *dnsproxy.Report) *SplitRouting {
	return &SplitRouting{
		config:   config,
		devmon:   devmon.NewDevMon(),
		addrmon:  addrmon.NewAddrMon(),
		devices:  NewDevices(),
		addrs:    NewAddresses(),
		excludes: NewExcludes(),
		prefixes: make(chan []netip.Prefix),
		dnsreps:  dnsReports,
		done:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
}
