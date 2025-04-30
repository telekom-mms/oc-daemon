// Package trafpol contains the traffic policing.
package trafpol

import (
	"context"
	"fmt"
	"net/netip"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cpd"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsmon"
)

// TrafPol command types.
const (
	trafPolCmdAddAddress uint8 = iota + 1
	trafPolCmdRemoveAddress
	trafPolCmdGetState
)

// State is the internal TrafPol state.
type State struct {
	CaptivePortal    bool
	AllowedDevices   []string
	AllowedAddresses []netip.Prefix
	AllowedNames     map[string][]netip.Addr
}

// trafPolCmd is a TrafPol command.
type trafPolCmd struct {
	typ   uint8
	ip    netip.Addr
	ok    bool
	state *State
	done  chan struct{}
}

// TrafPol is a traffic policing component.
type TrafPol struct {
	config *daemoncfg.Config
	devmon *devmon.DevMon
	dnsmon *dnsmon.DNSMon
	cpd    *cpd.CPD

	// capPortal indicates if a captive portal is detected
	capPortal bool

	// cpdStatus is a channel for CPD status updates
	cpdStatus chan bool

	// allowed devices, addresses, names
	allowDevs  *AllowDevs
	allowAddrs *AllowAddrs
	allowNames *AllowNames

	// resolver for allowed names, channel for resolver updates
	resolver *Resolver
	resolvUp chan *ResolvedName

	// address commands channel
	cmds chan *trafPolCmd

	loopDone chan struct{}
	done     chan struct{}
}

// handleDeviceUpdate handles a device update.
func (t *TrafPol) handleDeviceUpdate(ctx context.Context, u *devmon.Update) {
	// add or remove virtual device to/from allowed devices
	// skip adding physical devices and only allow adding virtual devices.
	// we cannot be sure about the type when removing devices, so do not
	// skip when removing devices.
	if u.Add && u.Type != "device" {
		if t.allowDevs.Add(u.Device) {
			setAllowedDevices(ctx, t.config, t.allowDevs.List())
		}
		return
	}
	if t.allowDevs.Remove(u.Device) {
		setAllowedDevices(ctx, t.config, t.allowDevs.List())
	}
}

// handleDNSUpdate handles a dns config update.
func (t *TrafPol) handleDNSUpdate() {
	// update allowed names
	t.resolver.Resolve()

	// trigger captive portal detection
	t.cpd.Probe()
}

// handleCPDReport handles a CPD report, returns whether status changed.
func (t *TrafPol) handleCPDReport(ctx context.Context, report *cpd.Report) bool {
	if !report.Detected {
		// no captive portal detected
		// check if there was a portal before
		if t.capPortal {
			// refresh all IPs, maybe they pointed to a
			// portal host in case of dns-based portals
			t.resolver.Resolve()

			// remove ports from allowed ports
			setAllowedPorts(ctx, t.config, []uint16{})
			t.capPortal = false
			log.WithField("capPortal", t.capPortal).Info("TrafPol changed CPD status")
			return true
		}
		return false
	}

	// add ports to allowed ports
	if !t.capPortal {
		setAllowedPorts(ctx, t.config, t.config.TrafficPolicing.PortalPorts)
		t.capPortal = true
		log.WithField("capPortal", t.capPortal).Info("TrafPol changed CPD status")
		return true
	}
	return false
}

// getAllowedHostsIPs returns the IPs of the allowed hosts,
// used for filter rules
func (t *TrafPol) getAllowedHostsIPs() []netip.Prefix {
	// get a list of all unique ip addresses from
	// - allowed names
	// - allowed addrs
	ipset := make(map[string]netip.Prefix)
	for _, n := range t.allowNames.GetAll() {
		for _, ip := range n {
			prefix := netip.PrefixFrom(ip, ip.BitLen())
			ipset[prefix.String()] = prefix
		}
	}
	for _, a := range t.allowAddrs.List() {
		ipset[a.String()] = a
	}

	// get resulting list of IPs
	ips := []netip.Prefix{}
	for _, ip := range ipset {
		ips = append(ips, ip)
	}

	return ips
}

// handleResolverUpdate handles a resolver update.
func (t *TrafPol) handleResolverUpdate(ctx context.Context, update *ResolvedName) {
	// update allowed names
	t.allowNames.Add(update.Name, update.IPs)

	// set new filter rules
	setAllowedIPs(ctx, t.config, t.getAllowedHostsIPs())
}

// handleAddressCommand handles an address command.
func (t *TrafPol) handleAddressCommand(ctx context.Context, cmd *trafPolCmd) {
	// convert to prefix
	prefix := netip.PrefixFrom(cmd.ip, cmd.ip.BitLen())

	// update allowed addrs
	if cmd.typ == trafPolCmdAddAddress {
		if ok := t.allowAddrs.Add(prefix); !ok {
			// ip already in allowed addrs
			return
		}
	} else {
		if ok := t.allowAddrs.Remove(prefix); !ok {
			// ip not in allowed addrs
			return
		}
	}

	// set new filter rules
	setAllowedIPs(ctx, t.config, t.getAllowedHostsIPs())

	// added/removed successfully
	cmd.ok = true
}

// handleGetStateCommand handles a get state command.
func (t *TrafPol) handleGetStateCommand(cmd *trafPolCmd) {
	// set state
	cmd.state = &State{
		CaptivePortal:    t.capPortal,
		AllowedDevices:   t.allowDevs.List(),
		AllowedAddresses: t.allowAddrs.List(),
		AllowedNames:     t.allowNames.GetAll(),
	}
}

// handleCommand handles a command.
func (t *TrafPol) handleCommand(ctx context.Context, cmd *trafPolCmd) {
	defer close(cmd.done)

	switch cmd.typ {
	case trafPolCmdAddAddress, trafPolCmdRemoveAddress:
		t.handleAddressCommand(ctx, cmd)
	case trafPolCmdGetState:
		t.handleGetStateCommand(cmd)
	}
}

// start starts the traffic policing component.
func (t *TrafPol) start(ctx context.Context) {
	defer close(t.loopDone)
	defer close(t.cpdStatus)
	defer unsetFilterRules(ctx, t.config)
	defer t.resolver.Stop()
	defer t.cpd.Stop()
	defer t.devmon.Stop()
	defer t.dnsmon.Stop()

	// enter main loop
	cpdResults := t.cpd.Results()
	var cpdStatus chan bool
	for {
		select {
		case u := <-t.devmon.Updates():
			// Device Update
			log.WithField("update", u).Debug("TrafPol got DevMon update")
			t.handleDeviceUpdate(ctx, u)

		case <-t.dnsmon.Updates():
			// DNS Update
			log.Debug("TrafPol got DNSMon update")
			t.handleDNSUpdate()

		case r := <-cpdResults:
			// CPD Result
			log.WithField("result", r).Debug("TrafPol got CPD result")
			if t.handleCPDReport(ctx, r) {
				// CPD status changed, pause reading CPD
				// results, send status
				cpdResults = nil
				cpdStatus = t.cpdStatus
			}

		case cpdStatus <- t.capPortal:
			// CPD status sent, resume reading CPD results, stop
			// sending status
			cpdResults = t.cpd.Results()
			cpdStatus = nil

		case u := <-t.resolvUp:
			// Resolver Update
			log.WithField("update", u).Debug("TrafPol got Resolver update")
			t.handleResolverUpdate(ctx, u)

		case c := <-t.cmds:
			// Command
			log.WithField("command", c).Debug("TrafPol got command")
			t.handleCommand(ctx, c)

		case <-t.done:
			// shutdown
			return
		}
	}
}

// Start starts the traffic policing component.
func (t *TrafPol) Start() error {
	log.Debug("TrafPol starting")

	// create context
	ctx := context.Background()

	// set firewall config
	setFilterRules(ctx, t.config)

	// set filter rules
	setAllowedIPs(ctx, t.config, t.getAllowedHostsIPs())

	// start resolver for allowed names
	t.resolver.Start()

	// start captive portal detection
	t.cpd.Start()

	// start device monitor
	err := t.devmon.Start()
	if err != nil {
		err = fmt.Errorf("TrafPol could not start DevMon: %w", err)
		goto cleanup_devmon
	}

	// start dns monitor
	err = t.dnsmon.Start()
	if err != nil {
		err = fmt.Errorf("TrafPol could not start DNSMon: %w", err)
		goto cleanup_dnsmon
	}

	go t.start(ctx)
	return nil

	// clean up after error
cleanup_dnsmon:
	t.devmon.Stop()
cleanup_devmon:
	t.cpd.Stop()
	t.resolver.Stop()
	unsetFilterRules(ctx, t.config)

	return err
}

// Stop stops the traffic policing component.
func (t *TrafPol) Stop() {
	close(t.done)

	// wait for everything
	<-t.loopDone
	log.Debug("TrafPol stopped")
}

// AddAllowedAddr adds addr to the allowed addresses.
func (t *TrafPol) AddAllowedAddr(addr netip.Addr) (ok bool) {
	log.WithField("addr", addr).
		Debug("TrafPol adding IP to allowed addresses")

	c := &trafPolCmd{
		typ:  trafPolCmdAddAddress,
		ip:   addr,
		done: make(chan struct{}),
	}
	t.cmds <- c
	<-c.done

	return c.ok
}

// RemoveAllowedAddr removes addr from the allowed addresses.
func (t *TrafPol) RemoveAllowedAddr(addr netip.Addr) (ok bool) {
	log.WithField("addr", addr).
		Debug("TrafPol removing IP from allowed addresses")

	c := &trafPolCmd{
		typ:  trafPolCmdRemoveAddress,
		ip:   addr,
		done: make(chan struct{}),
	}
	t.cmds <- c
	<-c.done

	return c.ok
}

// GetState returns the internal TrafPol state.
func (t *TrafPol) GetState() *State {
	log.Debug("TrafPol getting internal state")

	c := &trafPolCmd{
		typ:  trafPolCmdGetState,
		done: make(chan struct{}),
	}
	t.cmds <- c
	<-c.done

	return c.state
}

// CPDStatus returns the channel for CPD status updates.
func (t *TrafPol) CPDStatus() <-chan bool {
	return t.cpdStatus
}

// parseAllowedHosts parses the allowed hosts and returns IP addresses and DNS names
func parseAllowedHosts(hosts []string) (addrs []netip.Prefix, names []string) {
	for _, h := range hosts {
		// check if it's an IP address
		if ip, err := netip.ParseAddr(h); err == nil {
			prefix := netip.PrefixFrom(ip, ip.BitLen())
			addrs = append(addrs, prefix)
			continue
		}
		// check if it's an IP network
		if prefix, err := netip.ParsePrefix(h); err == nil {
			addrs = append(addrs, prefix)
			continue
		}

		// assume dns name
		names = append(names, h)
	}
	return
}

// NewTrafPol returns a new traffic policing component.
func NewTrafPol(config *daemoncfg.Config) *TrafPol {
	// create cpd
	c := cpd.NewCPD(daemoncfg.NewCPD())

	// get allowed addrs and names
	hosts := append(config.TrafficPolicing.AllowedHosts, c.Hosts()...)
	a, n := parseAllowedHosts(hosts)

	// create allowed addrs and names
	addrs := NewAllowAddrs()
	names := NewAllowNames()
	for _, addr := range a {
		addrs.Add(addr)
	}
	for _, name := range n {
		names.Add(name, []netip.Addr{})
	}

	// create channel for resolver updates
	resolvUp := make(chan *ResolvedName)

	// return trafpol
	return &TrafPol{
		config: config,
		devmon: devmon.NewDevMon(),
		dnsmon: dnsmon.NewDNSMon(dnsmon.NewConfig()),
		cpd:    c,

		cpdStatus: make(chan bool),

		allowDevs: NewAllowDevs(),

		allowAddrs: addrs,
		allowNames: names,
		resolver:   NewResolver(config.TrafficPolicing, n, resolvUp),
		resolvUp:   resolvUp,

		cmds: make(chan *trafPolCmd),

		loopDone: make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Cleanup cleans up old configuration after a failed shutdown.
func Cleanup(ctx context.Context, conf *daemoncfg.Config) {
	cleanupFilterRules(ctx, conf)
}
