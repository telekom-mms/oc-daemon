// Package splitrt contains the split routing.
package splitrt

import (
	"context"
	"fmt"
	"net/netip"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/addrmon"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
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
	dnsreps  chan *dnsproxy.Report
	done     chan struct{}
	closed   chan struct{}
}

// setupRouting sets up routing using config.
func (s *SplitRouting) setupRouting(ctx context.Context) {
	// set up routing
	data := s.config
	cmds, err := cmdtmpl.GetCmds("SplitRoutingSetupRouting", data)
	if err != nil {
		log.WithError(err).Error("SplitRouting could not get setup routing commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("SplitRouting could not run setup routing command")
		}
	}

	// add gateway to static excludes
	if s.config.VPNConfig.Gateway.IsValid() {
		gateway := netip.PrefixFrom(s.config.VPNConfig.Gateway,
			s.config.VPNConfig.Gateway.BitLen())
		if s.excludes.AddStatic(gateway) {
			setExcludes(ctx, s.config, s.excludes.GetPrefixes())
		}
	}

	// add static IPv4 excludes
	for _, e := range s.config.VPNConfig.Split.ExcludeIPv4 {
		if e.String() == "0.0.0.0/32" {
			continue
		}
		if s.excludes.AddStatic(e) {
			setExcludes(ctx, s.config, s.excludes.GetPrefixes())
		}
	}

	// add static IPv6 excludes
	for _, e := range s.config.VPNConfig.Split.ExcludeIPv6 {
		// TODO: does ::/128 exist?
		if e.String() == "::/128" {
			continue
		}
		if s.excludes.AddStatic(e) {
			setExcludes(ctx, s.config, s.excludes.GetPrefixes())
		}
	}
}

// teardownRouting tears down the routing configuration.
func (s *SplitRouting) teardownRouting(ctx context.Context) {
	// tear down routing
	data := s.config
	cmds, err := cmdtmpl.GetCmds("SplitRoutingTeardownRouting", data)
	if err != nil {
		log.WithError(err).Error("SplitRouting could not get teardown routing commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("SplitRouting could not run teardown routing command")
		}
	}
}

// excludeSettings returns whether local (virtual) networks should be excluded.
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
		if !isIn(e, s.locals.get()) {
			if s.excludes.AddStatic(e) {
				setExcludes(ctx, s.config, s.excludes.GetPrefixes())
			}
		}
	}

	// remove old excludes
	for _, l := range s.locals.get() {
		if !isIn(l, excludes) {
			if s.excludes.RemoveStatic(l) {
				setExcludes(ctx, s.config, s.excludes.GetPrefixes())
			}
		}
	}

	// save local excludes
	s.locals.set(excludes)
	log.WithField("locals", s.locals.get()).Debug("SplitRouting updated exclude local networks")
}

// handleDeviceUpdate handles a device update from the device monitor.
func (s *SplitRouting) handleDeviceUpdate(ctx context.Context, u *devmon.Update) {
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

	exclude := netip.PrefixFrom(r.IP, r.IP.BitLen())
	if s.excludes.AddDynamic(exclude, r.TTL) {
		setExcludes(ctx, s.config, s.excludes.GetPrefixes())
	}
}

// start starts split routing.
func (s *SplitRouting) start(ctx context.Context) {
	defer close(s.closed)
	defer s.teardownRouting(ctx)
	defer s.devmon.Stop()
	defer s.addrmon.Stop()

	// main loop
	timer := time.NewTimer(excludesTimer * time.Second)
	for {
		select {
		case u := <-s.devmon.Updates():
			s.handleDeviceUpdate(ctx, u)
		case u := <-s.addrmon.Updates():
			s.handleAddressUpdate(ctx, u)
		case r := <-s.dnsreps:
			s.handleDNSReport(ctx, r)
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
func NewSplitRouting(config *daemoncfg.Config) *SplitRouting {
	return &SplitRouting{
		config:   config,
		devmon:   devmon.NewDevMon(),
		addrmon:  addrmon.NewAddrMon(),
		devices:  NewDevices(),
		addrs:    NewAddresses(),
		excludes: NewExcludes(config),
		dnsreps:  make(chan *dnsproxy.Report),
		done:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
}

// Cleanup cleans up old configuration after a failed shutdown.
func Cleanup(ctx context.Context, config *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("SplitRoutingCleanup", config)
	if err != nil {
		log.WithError(err).Error("SplitRouting could not get cleanup commands")
	}
	for _, c := range cmds {
		if _, _, err := c.Run(ctx); err == nil {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
			}).Debug("SplitRouting cleaned up configuration")
		}
	}
}
