// Package trafpol contains the traffic policing.
package trafpol

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cpd"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsmon"
)

// TrafPol is a traffic policing component.
type TrafPol struct {
	config *Config
	devmon *devmon.DevMon
	dnsmon *dnsmon.DNSMon
	cpd    *cpd.CPD

	// capPortal indicates if a captive portal is detected
	capPortal bool

	// allowed devices, addresses, names
	allowDevs  *AllowDevs
	allowAddrs map[string]*net.IPNet
	allowNames map[string][]net.IP

	// resolver for allowed names, channel for resolver updates
	resolver *Resolver
	resolvUp chan *ResolvedName

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
		t.allowDevs.Add(ctx, u.Device)
		return
	}
	t.allowDevs.Remove(ctx, u.Device)
}

// handleDNSUpdate handles a dns config update.
func (t *TrafPol) handleDNSUpdate() {
	// update allowed names
	t.resolver.Resolve()

	// triger captive portal detection
	t.cpd.Probe()
}

// handleCPDReport handles a CPD report.
func (t *TrafPol) handleCPDReport(ctx context.Context, report *cpd.Report) {
	if !report.Detected {
		// no captive portal detected
		// check if there was a portal before
		if t.capPortal {
			// refresh all IPs, maybe they pointed to a
			// portal host in case of dns-based portals
			t.resolver.Resolve()

			// remove ports from allowed ports
			removePortalPorts(ctx, t.config.PortalPorts)
			t.capPortal = false
			log.WithField("capPortal", t.capPortal).Info("TrafPol changed CPD status")
		}
		return
	}

	// add ports to allowed ports
	if !t.capPortal {
		addPortalPorts(ctx, t.config.PortalPorts)
		t.capPortal = true
		log.WithField("capPortal", t.capPortal).Info("TrafPol changed CPD status")
	}
}

// getAllowedHostsIPs returns the IPs of the allowed hosts,
// used for filter rules
func (t *TrafPol) getAllowedHostsIPs() []*net.IPNet {
	// get a list of all unique ip addresses from
	// - allowed names
	// - allowed addrs
	ipset := make(map[string]*net.IPNet)
	for _, n := range t.allowNames {
		for _, ip := range n {
			ipnet := &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(32, 32),
			}
			if ip.To4() == nil {
				ipnet.Mask = net.CIDRMask(128, 128)
			}
			ipset[ipnet.String()] = ipnet
		}
	}
	for _, a := range t.allowAddrs {
		ipset[a.String()] = a
	}

	// get resulting list of IPs
	ips := []*net.IPNet{}
	for _, ip := range ipset {
		ips = append(ips, ip)
	}

	return ips
}

// handleResolverUpdate handles a resolver update.
func (t *TrafPol) handleResolverUpdate(ctx context.Context, update *ResolvedName) {
	// update allowed names
	t.allowNames[update.Name] = update.IPs

	// set new filter rules
	setAllowedIPs(ctx, t.getAllowedHostsIPs())
}

// start starts the traffic policing component.
func (t *TrafPol) start(ctx context.Context) {
	defer close(t.loopDone)
	defer unsetFilterRules(ctx)
	defer t.resolver.Stop()
	defer t.cpd.Stop()
	defer t.devmon.Stop()
	defer t.dnsmon.Stop()

	// enter main loop
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

		case r := <-t.cpd.Results():
			// CPD Result
			log.WithField("result", r).Debug("TrafPol got CPD result")
			t.handleCPDReport(ctx, r)

		case u := <-t.resolvUp:
			// Resolver Update
			log.WithField("update", u).Debug("TrafPol got Resolver update")
			t.handleResolverUpdate(ctx, u)

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
	setFilterRules(ctx, t.config.FirewallMark)

	// set filter rules
	setAllowedIPs(ctx, t.getAllowedHostsIPs())

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
	unsetFilterRules(ctx)

	return err
}

// Stop stops the traffic policing component.
func (t *TrafPol) Stop() {
	close(t.done)

	// wait for everything
	<-t.loopDone
	log.Debug("TrafPol stopped")
}

// parseAllowedHosts parses the allowed hosts and returns IP addresses and DNS names
func parseAllowedHosts(hosts []string) (addrs []*net.IPNet, names []string) {
	for _, h := range hosts {
		// check if it's an IP address
		if ip := net.ParseIP(h); ip != nil {
			ipnet := &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(32, 32),
			}
			if ip.To4() == nil {
				ipnet.Mask = net.CIDRMask(128, 128)
			}
			addrs = append(addrs, ipnet)
			continue
		}
		// check if it's an IP network
		if _, ipnet, err := net.ParseCIDR(h); err == nil {
			addrs = append(addrs, ipnet)
			continue
		}

		// assume dns name
		names = append(names, h)
	}
	return
}

// NewTrafPol returns a new traffic policing component.
func NewTrafPol(config *Config) *TrafPol {
	// create cpd
	c := cpd.NewCPD(cpd.NewConfig())

	// get allowed addrs and names
	hosts := append(config.AllowedHosts, c.Hosts()...)
	a, n := parseAllowedHosts(hosts)

	// create allowed addrs and names
	addrs := make(map[string]*net.IPNet)
	names := make(map[string][]net.IP)
	for _, addr := range a {
		addrs[addr.String()] = addr
	}
	for _, name := range n {
		names[name] = []net.IP{}
	}

	// create channel for resolver updates
	resolvUp := make(chan *ResolvedName)

	// return trafpol
	return &TrafPol{
		config: config,
		devmon: devmon.NewDevMon(),
		dnsmon: dnsmon.NewDNSMon(dnsmon.NewConfig()),
		cpd:    c,

		allowDevs: NewAllowDevs(),

		allowAddrs: addrs,
		allowNames: names,
		resolver:   NewResolver(config, n, resolvUp),
		resolvUp:   resolvUp,

		loopDone: make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Cleanup cleans up old configuration after a failed shutdown.
func Cleanup(ctx context.Context) {
	cleanupFilterRules(ctx)
}
