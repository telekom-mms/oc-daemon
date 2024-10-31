// Package splitrt contains the split routing.
package splitrt

import (
	"context"
	"fmt"
	"net/netip"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/addrmon"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// State is the internal state.
type State struct {
	Config          *Config
	VPNConfig       *vpnconfig.Config
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
	config    *Config
	vpnconfig *vpnconfig.Config
	devmon    *devmon.DevMon
	addrmon   *addrmon.AddrMon
	devices   *Devices
	addrs     *Addresses
	locals    locals
	excludes  *Excludes
	dnsreps   chan *dnsproxy.Report
	done      chan struct{}
	closed    chan struct{}
}

const DefaultTemplates = `
{{- define "RoutingRules"}}
table inet oc-daemon-routing {
	# set for ipv4 excludes
	set excludes4 {
		type ipv4_addr
		flags interval
	}

	# set for ipv6 excludes
	set excludes6 {
		type ipv6_addr
		flags interval
	}

	chain preraw {
		type filter hook prerouting priority raw; policy accept;

		# add drop rules for non-local traffic from other devices to
		# tunnel network addresses here
		{{if .IPv4Address}}
		iifname != {{.Device}} ip daddr {{.IPv4Address}} fib saddr type != local counter drop
		{{end}}
		{{if .IPv6Address}}
		iifname != {{.Device}} ip6 daddr {{.IPv6Address}} fib saddr type != local counter drop
		{{end}}
	}

	chain splitrouting {
		# restore mark from conntracking
		ct mark != 0 meta mark set ct mark counter
		meta mark != 0 counter accept

		# mark packets in exclude sets
		ip daddr @excludes4 counter meta mark set {{.FWMark}}
		ip6 daddr @excludes6 counter meta mark set {{.FWMark}}

		# save mark in conntraction
		ct mark set meta mark counter
	}

	chain premangle {
		type filter hook prerouting priority mangle; policy accept;

		# handle split routing
		counter jump splitrouting
	}

	chain output {
		type route hook output priority mangle; policy accept;

		# handle split routing
		counter jump splitrouting
	}

	chain postmangle {
		type filter hook postrouting priority mangle; policy accept;

		# save mark in conntracking
		meta mark {{.FWMark}} ct mark set meta mark counter
	}

	chain postrouting {
		type nat hook postrouting priority srcnat; policy accept;

		# masquerare tunnel/exclude traffic to make sure the source IP
		# matches the outgoing interface
		ct mark {{.FWMark}} counter masquerade
	}

	chain rejectipversion {
		# used to reject unsupported ip version on the tunnel device

		# make sure exclude traffic is not filtered
		ct mark {{.FWMark}} counter accept

		# use tcp reset and icmp admin prohibited
		meta l4proto tcp counter reject with tcp reset
		counter reject with icmpx admin-prohibited
	}

	chain rejectforward {
		type filter hook forward priority filter; policy accept;

		# reject unsupported ip versions when forwarding packets,
		# add matching jump rule to rejectipversion if necessary
		{{if .IPv4Address}}
		meta oifname {{.Device}} meta nfproto ipv6 counter jump rejectipversion
		{{end}}
		{{if .IPv6Address}}
		meta oifname {{.Device}} meta nfproto ipv4 counter jump rejectipversion
		{{end}}
	}

	chain rejectoutput {
		type filter hook output priority filter; policy accept;

		# reject unsupported ip versions when sending packets,
		# add matching jump rule to rejectipversion if necessary
		{{if .IPv4Address}}
		meta oifname {{.Device}} meta nfproto ipv6 counter jump rejectipversion
		{{end}}
		{{if .IPv6Address}}
		meta oifname {{.Device}} meta nfproto ipv4 counter jump rejectipversion
		{{end}}
	}
}
{{end -}}
`

func (s *SplitRouting) getTemplateData() map[string]string {
	ipv4 := ""
	if len(s.vpnconfig.IPv4.Address) > 0 {
		ipv4 = s.vpnconfig.IPv4.Address.String()
	}
	ipv6 := ""
	if len(s.vpnconfig.IPv6.Address) > 0 {
		ipv4 = s.vpnconfig.IPv6.Address.String()
	}
	// TODO: change names to full names like FirewallMark?
	return map[string]string{
		"Device":      s.vpnconfig.Device.Name,
		"IPv4Address": ipv4,
		"IPv6Address": ipv6,
		"FWMark":      s.config.FirewallMark,
		"RTTable":     s.config.RoutingTable,
		"RulePrio1":   s.config.RulePriority1,
		"RulePrio2":   s.config.RulePriority2,
	}
}

// setupRouting sets up routing using config.
func (s *SplitRouting) setupRouting(ctx context.Context) {

	// TODO: set and load default template in constructor?
	// TODO: rename to NewRunner?
	// TODO: can we use one for the whole splitrouting instance?
	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	// TODO: get commands from config?
	data := s.getTemplateData()
	commands := []*cmdtmpl.Command{
		{Line: "nft -f -", Stdin: `{{template "RoutingRules" .}}`},
		{Line: "ip -4 route add 0.0.0.0/0 dev {{.Device}} table {{.RTTable}}"},
		{Line: "ip -4 rule add iif {{.Device}} table main pref {{.RulePrio1}}"},
		{Line: "ip -4 rule add not fwmark {{.FWMark}} table {{.RTTable}} pref {{.RulePrio2}}"},
		{Line: "sysctl -q net.ipv4.conf.all.src_valid_mark=1"},
		{Line: "ip -6 route add ::/0 dev {{.Device}} table {{.RTTable}}"},
		{Line: "ip -6 rule add iif {{.Device}} table main pref {{.RulePrio1}}"},
		{Line: "ip -6 rule add not fwmark {{.FWMark}} table {{.RTTable}} pref {{.RulePrio2}}"},
	}
	for _, c := range commands {
		// TODO: get final command and stdin
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
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
}

// teardownRouting tears down the routing configuration.
func (s *SplitRouting) teardownRouting(ctx context.Context) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := s.getTemplateData()
	commands := []*cmdtmpl.Command{
		{Line: "ip -4 rule delete table {{.RTTable}}"},
		{Line: "ip -4 rule delete iif {{.Device}} table main"},
		{Line: "ip -6 rule delete table {{.RTTable}}"},
		{Line: "ip -6 rule delete iif {{.Device}} table main"},
		{Line: "nft -f - delete table inet oc-daemon-routing"},
	}
	for _, c := range commands {
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}

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
		if !isIn(e, s.locals.get()) {
			s.excludes.AddStatic(ctx, e)
		}
	}

	// remove old excludes
	for _, l := range s.locals.get() {
		if !isIn(l, excludes) {
			s.excludes.RemoveStatic(ctx, l)
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

// GetState returns the internal state.
func (s *SplitRouting) GetState() *State {
	var locals []string
	for _, l := range s.locals.get() {
		locals = append(locals, l.String())
	}
	static, dynamic := s.excludes.List()
	return &State{
		Config:          s.config,
		VPNConfig:       s.vpnconfig,
		Devices:         s.devices.List(),
		Addresses:       s.addrs.List(),
		LocalExcludes:   locals,
		StaticExcludes:  static,
		DynamicExcludes: dynamic,
	}
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

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := map[string]string{
		"RTTable":   config.RoutingTable,
		"RulePrio1": config.RulePriority1,
		"RulePrio2": config.RulePriority2,
	}
	commands := []*cmdtmpl.Command{
		{Line: "ip -4 rule delete pref {{.RulePrio1}}"},
		{Line: "ip -4 rule delete pref {{.RulePrio2}}"},
		{Line: "ip -6 rule delete pref {{.RulePrio1}}"},
		{Line: "ip -6 rule delete pref {{.RulePrio2}}"},
		{Line: "ip -4 route flush table {{.RTTable}}"},
		{Line: "ip -6 route flush table {{.RTTable}}"},
		{Line: "nft -f - delete table inet oc-daemon-routing"},
	}
	for _, c := range commands {
		// TODO: separate template errors from execution errors? here
		// we want execution to fail but template execution should be
		// not fail
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}
}
