package splitrt

import (
	"net"

	"github.com/T-Systems-MMS/oc-daemon/internal/addrmon"
	"github.com/T-Systems-MMS/oc-daemon/internal/devmon"
	"github.com/T-Systems-MMS/oc-daemon/internal/dnsproxy"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnconfig"
	log "github.com/sirupsen/logrus"
)

// SplitRouting is a split routing configuration
type SplitRouting struct {
	config   *vpnconfig.Config
	devmon   *devmon.DevMon
	addrmon  *addrmon.AddrMon
	devices  *Devices
	addrs    *Addresses
	locals   []*net.IPNet
	excludes *Excludes
	dnsreps  chan *dnsproxy.Report
	done     chan struct{}
	closed   chan struct{}
}

// setupRouting sets up routing using config
func (s *SplitRouting) setupRouting() {
	// get vpn network addresses
	ipnet4 := &net.IPNet{
		IP:   s.config.IPv4.Address,
		Mask: s.config.IPv4.Netmask,
	}
	ipnet6 := &net.IPNet{
		IP:   s.config.IPv6.Address,
		Mask: s.config.IPv6.Netmask,
	}

	// prepare netfilter and excludes
	setRoutingRules()

	// filter non-local traffic to vpn addresses
	addLocalAddressesIPv4(s.config.Device.Name, []*net.IPNet{ipnet4})
	addLocalAddressesIPv6(s.config.Device.Name, []*net.IPNet{ipnet6})

	// reject unsupported ip versions on vpn
	if len(s.config.IPv6.Address) == 0 {
		rejectIPv6(s.config.Device.Name)
	}
	if len(s.config.IPv4.Address) == 0 {
		rejectIPv4(s.config.Device.Name)
	}

	// add excludes
	s.excludes.Start()

	// add gateway to static excludes
	gateway := &net.IPNet{
		IP:   s.config.Gateway,
		Mask: net.CIDRMask(32, 32),
	}
	if gateway.IP.To4() == nil {
		gateway.Mask = net.CIDRMask(128, 128)
	}
	s.excludes.AddStatic(gateway)

	// add static IPv4 excludes
	for _, e := range s.config.Split.ExcludeIPv4 {
		if e.String() == "0.0.0.0/32" {
			continue
		}
		s.excludes.AddStatic(e)
	}

	// add static IPv6 excludes
	for _, e := range s.config.Split.ExcludeIPv6 {
		// TODO: does ::/128 exist?
		if e.String() == "::/128" {
			continue
		}
		s.excludes.AddStatic(e)
	}

	// setup routing
	// TODO: add netlink variant?
	addDefaultRouteIPv4(s.config.Device.Name)
	addDefaultRouteIPv6(s.config.Device.Name)

}

// teardownRouting tears down the routing configuration
func (s *SplitRouting) teardownRouting() {
	deleteDefaultRouteIPv4(s.config.Device.Name)
	deleteDefaultRouteIPv6(s.config.Device.Name)
	unsetRoutingRules()

	// remove excludes
	s.excludes.Stop()
}

// excludeSettings returns if local (virtual) networks should be excluded
func (s *SplitRouting) excludeLocalNetworks() (exclude bool, virtual bool) {
	for _, e := range s.config.Split.ExcludeIPv4 {
		if e.String() == "0.0.0.0/32" {
			exclude = true
		}
	}
	if s.config.Split.ExcludeVirtualSubnetsOnlyIPv4 {
		virtual = true
	}
	return
}

// updateLocalNetworkExcludes updates the local network split excludes
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
	excludes := []*net.IPNet{}
	for _, d := range devs {
		excludes = append(excludes, s.addrs.Get(d)...)
	}

	// determine changes
	// TODO: move s.locals into excludes?
	isIn := func(n *net.IPNet, nets []*net.IPNet) bool {
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
			s.excludes.AddStatic(e)
		}
	}

	// remove old excludes
	for _, l := range s.locals {
		if !isIn(l, excludes) {
			s.excludes.Remove(l)
		}
	}

	// save local excludes
	s.locals = excludes
	log.WithField("locals", s.locals).Debug("SplitRouting updated exclude local networks")
}

// handleDeviceUpdate handles a device update from the device monitor
func (s *SplitRouting) handleDeviceUpdate(u *devmon.Update) {
	log.WithField("update", u).Debug("SplitRouting got device update")

	if u.Add {
		if u.Type == "loopback" {
			// skip loopback devices
			return
		}
		if u.Device == s.config.Device.Name {
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

// handleAddressUpdate handles an address update from the address monitor
func (s *SplitRouting) handleAddressUpdate(u *addrmon.Update) {
	log.WithField("update", u).Debug("SplitRouting got address update")

	if u.Add {
		s.addrs.Add(u)
	} else {
		s.addrs.Remove(u)
	}
	s.updateLocalNetworkExcludes()
}

// handleDNSReport handles a DNS report
func (s *SplitRouting) handleDNSReport(r *dnsproxy.Report) {
	defer r.Done()
	log.WithField("report", r).Debug("SplitRouting handling DNS report")

	if r.IP.To4() != nil {
		s.excludes.AddDynamic(&net.IPNet{
			IP:   r.IP,
			Mask: net.CIDRMask(32, 32),
		}, r.TTL)
		return
	}
	s.excludes.AddDynamic(&net.IPNet{
		IP:   r.IP,
		Mask: net.CIDRMask(128, 128),
	}, r.TTL)
}

// start starts split routing
func (s *SplitRouting) start() {
	log.Debug("SplitRouting starting")
	defer close(s.closed)

	// configure routing
	s.setupRouting()
	defer s.teardownRouting()

	// start device monitor
	s.devmon.Start()
	defer s.devmon.Stop()

	// start address monitor
	s.addrmon.Start()
	defer s.addrmon.Stop()

	// main loop
	for {
		select {
		case u := <-s.devmon.Updates():
			s.handleDeviceUpdate(u)
		case u := <-s.addrmon.Updates():
			s.handleAddressUpdate(u)
		case r := <-s.dnsreps:
			s.handleDNSReport(r)
		case <-s.done:
			return
		}
	}
}

// Start starts split routing
func (s *SplitRouting) Start() {
	go s.start()
}

// Stop stops split routing
func (s *SplitRouting) Stop() {
	close(s.done)
	<-s.closed
	log.Debug("SplitRouting stopped")
}

// DNSReports returns the channel for dns reports
func (s *SplitRouting) DNSReports() chan *dnsproxy.Report {
	return s.dnsreps
}

// NewSplitRouting returns a new SplitRouting
func NewSplitRouting(config *vpnconfig.Config) *SplitRouting {
	return &SplitRouting{
		config:   config,
		devmon:   devmon.NewDevMon(),
		addrmon:  addrmon.NewAddrMon(),
		devices:  NewDevices(),
		addrs:    NewAddresses(),
		excludes: NewExcludes(),
		dnsreps:  make(chan *dnsproxy.Report),
		done:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
}

// Cleanup cleans up old configuration after a failed shutdown
func Cleanup() {
	cleanupRouting()
	cleanupRoutingRules()
}
