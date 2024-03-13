// Package vpnsetup contains the VPN setup component.
package vpnsetup

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/internal/execs"
	"github.com/telekom-mms/oc-daemon/internal/splitrt"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// command types.
const (
	commandSetup uint8 = iota
	commandTeardown
)

// command is a VPNSetup command.
type command struct {
	cmd     uint8
	vpnconf *vpnconfig.Config
}

// Event types.
const (
	EventSetupOK uint8 = iota
	EventTeardownOK
)

// Event is a VPNSetup event.
type Event struct {
	Type uint8
}

// VPNSetup sets up the configuration of the vpn tunnel that belongs to the
// current VPN connection.
type VPNSetup struct {
	splitrt     *splitrt.SplitRouting
	splitrtConf *splitrt.Config

	dnsProxy     *dnsproxy.Proxy
	dnsProxyConf *dnsproxy.Config

	ensureDone   chan struct{}
	ensureClosed chan struct{}

	cmds   chan *command
	events chan *Event
	done   chan struct{}
	closed chan struct{}
}

// sendEvents sends the event.
func (v *VPNSetup) sendEvent(event *Event) {
	select {
	case v.events <- event:
	case <-v.done:
	}
}

// setupVPNDevice sets up the vpn device with config.
func setupVPNDevice(ctx context.Context, c *vpnconfig.Config) {
	// set mtu on device
	mtu := strconv.Itoa(c.Device.MTU)
	if err := execs.RunIPLink(ctx, "set", c.Device.Name, "mtu", mtu); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"device": c.Device.Name,
			"mtu":    mtu,
		}).Error("Daemon could not set mtu on device")
		return
	}

	// set device up
	if err := execs.RunIPLink(ctx, "set", c.Device.Name, "up"); err != nil {
		log.WithError(err).WithField("device", c.Device.Name).
			Error("Daemon could not set device up")
		return
	}

	// set ipv4 and ipv6 addresses on device
	setupIP := func(ip net.IP, mask net.IPMask) {
		ipnet := &net.IPNet{
			IP:   ip,
			Mask: mask,
		}
		dev := c.Device.Name
		addr := ipnet.String()
		if err := execs.RunIPAddress(ctx, "add", addr, "dev", dev); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"device": dev,
				"ip":     addr,
			}).Error("Daemon could not set ip on device")
			return
		}

	}
	if len(c.IPv4.Address) > 0 {
		setupIP(c.IPv4.Address, c.IPv4.Netmask)
	}
	if len(c.IPv6.Address) > 0 {
		setupIP(c.IPv6.Address, c.IPv6.Netmask)
	}
}

// teardownVPNDevice tears down the configured vpn device.
func teardownVPNDevice(ctx context.Context, c *vpnconfig.Config) {
	// set device down
	if err := execs.RunIPLink(ctx, "set", c.Device.Name, "down"); err != nil {
		log.WithError(err).WithField("device", c.Device.Name).
			Error("Daemon could not set device down")
		return
	}

}

// setupRouting sets up routing using config.
func (v *VPNSetup) setupRouting(vpnconf *vpnconfig.Config) {
	if v.splitrt != nil {
		return
	}
	v.splitrt = splitrt.NewSplitRouting(v.splitrtConf, vpnconf)
	if err := v.splitrt.Start(); err != nil {
		log.WithError(err).Error("VPNSetup error setting split routing")
	}
}

// teardownRouting tears down the routing configuration.
func (v *VPNSetup) teardownRouting() {
	if v.splitrt == nil {
		return
	}
	v.splitrt.Stop()
	v.splitrt = nil
}

// setupDNSServer sets the DNS server.
func (v *VPNSetup) setupDNSServer(ctx context.Context, config *vpnconfig.Config) {
	device := config.Device.Name
	if err := execs.RunResolvectl(ctx, "dns", device, v.dnsProxyConf.Address); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"device": device,
			"server": v.dnsProxyConf.Address,
		}).Error("VPNSetup error setting dns server")
	}
}

// setupDNSDomains sets the DNS domains.
func (v *VPNSetup) setupDNSDomains(ctx context.Context, config *vpnconfig.Config) {
	device := config.Device.Name
	if err := execs.RunResolvectl(ctx, "domain", device, config.DNS.DefaultDomain, "~."); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"device": device,
			"domain": config.DNS.DefaultDomain,
		}).Error("VPNSetup error setting dns domains")
	}
}

// setupDNSDefaultRoute sets the DNS default route.
func (v *VPNSetup) setupDNSDefaultRoute(ctx context.Context, config *vpnconfig.Config) {
	device := config.Device.Name
	if err := execs.RunResolvectl(ctx, "default-route", device, "yes"); err != nil {
		log.WithError(err).WithField("device", device).
			Error("VPNSetup error setting dns default route")
	}
}

// setupDNS sets up DNS using config.
func (v *VPNSetup) setupDNS(ctx context.Context, config *vpnconfig.Config) {
	// configure dns proxy

	// set remotes
	remotes := config.DNS.Remotes()
	v.dnsProxy.SetRemotes(remotes)

	// set watches
	excludes := config.Split.DNSExcludes()
	log.WithField("excludes", excludes).Debug("Daemon setting DNS Split Excludes")
	v.dnsProxy.SetWatches(excludes)

	// update dns configuration of host

	// set dns server for device
	v.setupDNSServer(ctx, config)

	// set domains for device
	// this includes "~." to use this device for all domains
	v.setupDNSDomains(ctx, config)

	// set default route for device
	v.setupDNSDefaultRoute(ctx, config)

	// flush dns caches
	if err := execs.RunResolvectl(ctx, "flush-caches"); err != nil {
		log.WithError(err).Error("VPNSetup error flushing dns caches during setup")
	}

	// reset learnt server features
	if err := execs.RunResolvectl(ctx, "reset-server-features"); err != nil {
		log.WithError(err).Error("VPNSetup error resetting server features during setup")
	}
}

// teardownDNS tears down the DNS configuration.
func (v *VPNSetup) teardownDNS(ctx context.Context, vpnconf *vpnconfig.Config) {
	// update dns proxy configuration

	// reset remotes
	remotes := map[string][]string{}
	v.dnsProxy.SetRemotes(remotes)

	// reset watches
	v.dnsProxy.SetWatches([]string{})

	// update dns configuration of host

	// undo device dns configuration
	if err := execs.RunResolvectl(ctx, "revert", vpnconf.Device.Name); err != nil {
		log.WithError(err).WithField("device", vpnconf.Device.Name).
			Error("VPNSetup error reverting dns configuration")
	}

	// flush dns caches
	if err := execs.RunResolvectl(ctx, "flush-caches"); err != nil {
		log.WithError(err).Error("VPNSetup error flushing dns caches during teardown")
	}

	// reset learnt server features
	if err := execs.RunResolvectl(ctx, "reset-server-features"); err != nil {
		log.WithError(err).Error("VPNSetup error resetting server features during teardown")
	}
}

// checkDNSProtocols checks the configured DNS protocols, only checks default-route.
func (v *VPNSetup) checkDNSProtocols(protocols []string) bool {
	// check if default route is set
	ok := false
	for _, protocol := range protocols {
		if protocol == "+DefaultRoute" {
			ok = true
		}
	}

	return ok
}

// checkDNSServers checks the configured DNS servers.
func (v *VPNSetup) checkDNSServers(servers []string) bool {
	// check dns server ip
	if len(servers) != 1 || servers[0] != v.dnsProxyConf.Address {
		// server not correct
		return false
	}

	return true
}

// checkDNSDomain checks the configured DNS domains.
func (v *VPNSetup) checkDNSDomain(config *vpnconfig.Config, domains []string) bool {
	// get domains in config
	inConfig := strings.Fields(config.DNS.DefaultDomain)
	inConfig = append(inConfig, "~.")

	// check domains in config
	for _, c := range inConfig {
		found := false
		for _, d := range domains {
			if c == d {
				found = true
			}
		}

		if !found {
			// domains not correct
			return false
		}
	}

	return true
}

// ensureDNS ensures the DNS config.
func (v *VPNSetup) ensureDNS(ctx context.Context, config *vpnconfig.Config) bool {
	log.Debug("VPNSetup checking DNS settings")

	// get dns settings
	device := config.Device.Name
	stdout, err := execs.RunResolvectlOutput(ctx, "status", device, "--no-pager")
	if err != nil {
		log.WithError(err).WithField("device", device).Error("VPNSetup error getting DNS settings")
		return false
	}

	// parse and check dns settings line by line
	var protOK, srvOK, domOK bool
	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		// try to find separator ":"
		before, after, found := strings.Cut(strings.TrimSpace(line), ":")
		if !found {
			continue
		}

		// get fields after separator
		f := strings.Fields(after)

		// check settings if present
		switch before {
		case "Protocols":
			protOK = v.checkDNSProtocols(f)

		case "DNS Servers":
			srvOK = v.checkDNSServers(f)

		case "DNS Domain":
			domOK = v.checkDNSDomain(config, f)
		}
	}

	// reset settings if incorrect/not present

	if !protOK {
		// protocols are not correct
		log.Error("VPNSetup found invalid DNS protocols, trying to fix")

		// reset default route for device
		v.setupDNSDefaultRoute(ctx, config)
	}

	if !srvOK {
		// servers are not correct
		log.Error("VPNSetup found invalid DNS servers, trying to fix")

		// reset dns server
		v.setupDNSServer(ctx, config)
	}

	if !domOK {
		// domains are not correct
		log.Error("VPNSetup found invalid DNS domains, trying to fix")

		// reset domains for device
		v.setupDNSDomains(ctx, config)
	}

	// combine results
	return protOK && srvOK && domOK
}

// ensureConfig ensured that the VPN config is and stays active.
func (v *VPNSetup) ensureConfig(ctx context.Context, vpnconf *vpnconfig.Config) {
	defer close(v.ensureClosed)

	timerInvalid := time.Second
	timerValid := 15 * time.Second
	timer := timerInvalid
	for {
		select {
		case <-time.After(timer):
			log.Debug("VPNSetup checking VPN configuration")

			// ensure DNS settings
			if ok := v.ensureDNS(ctx, vpnconf); !ok {
				timer = timerInvalid
				break
			}

			// vpn config is OK
			timer = timerValid

		case <-v.ensureDone:
			return
		}
	}
}

// startEnsure starts ensuring the VPN config.
func (v *VPNSetup) startEnsure(ctx context.Context, vpnconf *vpnconfig.Config) {
	v.ensureDone = make(chan struct{})
	v.ensureClosed = make(chan struct{})
	go v.ensureConfig(ctx, vpnconf)
}

// stopEnsure stops ensuring the VPN config.
func (v *VPNSetup) stopEnsure() {
	close(v.ensureDone)
	<-v.ensureClosed
}

// setup sets up the vpn configuration.
func (v *VPNSetup) setup(ctx context.Context, vpnconf *vpnconfig.Config) {
	// setup device, routing, dns
	setupVPNDevice(ctx, vpnconf)
	v.setupRouting(vpnconf)
	v.setupDNS(ctx, vpnconf)

	// ensure VPN config
	v.startEnsure(ctx, vpnconf)

	// signal setup complete
	v.sendEvent(&Event{EventSetupOK})
}

// teardown tears down the vpn configuration.
func (v *VPNSetup) teardown(ctx context.Context, vpnconf *vpnconfig.Config) {
	// stop ensuring VPN config
	v.stopEnsure()

	// tear down device, routing, dns
	teardownVPNDevice(ctx, vpnconf)
	v.teardownRouting()
	v.teardownDNS(ctx, vpnconf)

	// signal teardown complete
	v.sendEvent(&Event{EventTeardownOK})
}

// handleCommand handles a command.
func (v *VPNSetup) handleCommand(ctx context.Context, c *command) {
	switch c.cmd {
	case commandSetup:
		v.setup(ctx, c.vpnconf)
	case commandTeardown:
		v.teardown(ctx, c.vpnconf)
	}
}

// handleDNSReport handles a DNS report.
func (v *VPNSetup) handleDNSReport(r *dnsproxy.Report) {
	log.WithField("report", r).Debug("Daemon handling DNS report")

	if v.splitrt == nil {
		return
	}

	// forward report to split routing
	select {
	case v.splitrt.DNSReports() <- r:
	case <-v.done:
	}
}

// start starts the VPN setup.
func (v *VPNSetup) start() {
	defer close(v.closed)
	defer close(v.events)

	// create context
	ctx := context.Background()

	// start DNS-Proxy
	v.dnsProxy.Start()
	defer v.dnsProxy.Stop()

	for {
		select {
		case c := <-v.cmds:
			v.handleCommand(ctx, c)
		case r := <-v.dnsProxy.Reports():
			v.handleDNSReport(r)
		case <-v.done:
			return
		}
	}
}

// Start starts the VPN setup.
func (v *VPNSetup) Start() {
	go v.start()
}

// Stop stops the VPN setup.
func (v *VPNSetup) Stop() {
	close(v.done)
	<-v.closed
}

// Setup sets the VPN config up.
func (v *VPNSetup) Setup(vpnconfig *vpnconfig.Config) {
	v.cmds <- &command{
		cmd:     commandSetup,
		vpnconf: vpnconfig,
	}
}

// Teardown tears the VPN config down.
func (v *VPNSetup) Teardown(vpnconfig *vpnconfig.Config) {
	v.cmds <- &command{
		cmd:     commandTeardown,
		vpnconf: vpnconfig,
	}
}

// Events returns the events channel.
func (v *VPNSetup) Events() chan *Event {
	return v.events
}

// NewVPNSetup returns a new VPNSetup.
func NewVPNSetup(
	dnsProxyConfig *dnsproxy.Config,
	splitrtConfig *splitrt.Config,
) *VPNSetup {
	return &VPNSetup{
		dnsProxy:     dnsproxy.NewProxy(dnsProxyConfig),
		dnsProxyConf: dnsProxyConfig,
		splitrtConf:  splitrtConfig,

		cmds:   make(chan *command),
		events: make(chan *Event),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}

// Cleanup cleans up the configuration after a failed shutdown.
func Cleanup(ctx context.Context, vpnDevice string, splitrtConfig *splitrt.Config) {
	// dns, device, split routing
	if err := execs.RunResolvectl(ctx, "revert", vpnDevice); err == nil {
		log.WithField("device", vpnDevice).
			Warn("VPNSetup cleaned up dns config")
	}
	if err := execs.RunIPLink(ctx, "delete", vpnDevice); err == nil {
		log.WithField("device", vpnDevice).
			Warn("VPNSetup cleaned up vpn device")
	}
	splitrt.Cleanup(ctx, splitrtConfig)
}
