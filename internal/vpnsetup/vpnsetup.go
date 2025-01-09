// Package vpnsetup contains the VPN setup component.
package vpnsetup

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/internal/splitrt"
)

// command types.
const (
	commandSetup uint8 = iota
	commandTeardown
	commandGetState
)

// State is the internal state of the VPN Setup.
type State struct {
	SplitRouting *splitrt.State
	DNSProxy     *dnsproxy.State
}

// command is a VPNSetup command.
type command struct {
	cmd   uint8
	conf  *daemoncfg.Config
	state *State
	done  chan struct{}
}

// VPNSetup sets up the configuration of the vpn tunnel that belongs to the
// current VPN connection.
type VPNSetup struct {
	splitrt  *splitrt.SplitRouting
	dnsProxy *dnsproxy.Proxy

	ensureDone   chan struct{}
	ensureClosed chan struct{}

	cmds   chan *command
	done   chan struct{}
	closed chan struct{}
}

// setupVPNDevice sets up the vpn device with config.
func setupVPNDevice(ctx context.Context, config *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("VPNSetupSetupVPNDevice", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get setup VPN device commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("VPNSetup could not run setup VPN device command")
		}
	}
}

// teardownVPNDevice tears down the configured vpn device.
func teardownVPNDevice(ctx context.Context, config *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("VPNSetupTeardownVPNDevice", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get teardown VPN device commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("VPNSetup could not run teardown VPN device command")
		}
	}
}

// setupRouting sets up routing using config.
func (v *VPNSetup) setupRouting(config *daemoncfg.Config) {
	if v.splitrt != nil {
		return
	}
	v.splitrt = splitrt.NewSplitRouting(config)
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
func (v *VPNSetup) setupDNSServer(ctx context.Context, config *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("VPNSetupSetupDNSServer", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get setup DNS server commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("VPNSetup could not run setup DNS server command")
		}
	}
}

// setupDNSDomains sets the DNS domains.
func (v *VPNSetup) setupDNSDomains(ctx context.Context, config *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("VPNSetupSetupDNSDomains", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get setup DNS domains commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("VPNSetup could not run setup DNS domains command")
		}
	}
}

// setupDNSDefaultRoute sets the DNS default route.
func (v *VPNSetup) setupDNSDefaultRoute(ctx context.Context, config *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("VPNSetupSetupDNSDefaultRoute", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get setup DNS default route commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("VPNSetup could not run setup DNS default route command")
		}
	}
}

// setupDNS sets up DNS using config.
func (v *VPNSetup) setupDNS(ctx context.Context, config *daemoncfg.Config) {
	// configure dns proxy

	// set remotes
	remotes := config.VPNConfig.DNS.Remotes()
	v.dnsProxy.SetRemotes(remotes)

	// set watches
	excludes := config.VPNConfig.Split.DNSExcludes()
	log.WithField("excludes", excludes).Debug("Daemon setting DNS Split Excludes")
	v.dnsProxy.SetWatches(excludes)

	// update dns configuration of host
	cmds, err := cmdtmpl.GetCmds("VPNSetupSetupDNS", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get setup DNS commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("VPNSetup could not run setup DNS command")
		}
	}
}

// teardownDNS tears down the DNS configuration.
func (v *VPNSetup) teardownDNS(ctx context.Context, config *daemoncfg.Config) {
	// update dns proxy configuration

	// reset remotes
	remotes := map[string][]string{}
	v.dnsProxy.SetRemotes(remotes)

	// reset watches
	v.dnsProxy.SetWatches([]string{})

	// update dns configuration of host
	cmds, err := cmdtmpl.GetCmds("VPNSetupTeardownDNS", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get teardown DNS commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("VPNSetup could not run teardown DNS command")
		}
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
func (v *VPNSetup) checkDNSServers(conf *daemoncfg.Config, servers []string) bool {
	// check dns server ip
	if len(servers) != 1 || servers[0] != conf.DNSProxy.Address {
		// server not correct
		return false
	}

	return true
}

// checkDNSDomain checks the configured DNS domains.
func (v *VPNSetup) checkDNSDomain(config *daemoncfg.Config, domains []string) bool {
	// get domains in config
	inConfig := strings.Fields(config.VPNConfig.DNS.DefaultDomain)
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
func (v *VPNSetup) ensureDNS(ctx context.Context, config *daemoncfg.Config) bool {
	log.Debug("VPNSetup checking DNS settings")

	// get dns settings
	cmds, err := cmdtmpl.GetCmds("VPNSetupEnsureDNS", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get ensure DNS commands")
	}
	var stdout []byte
	for _, c := range cmds {
		sout, serr, err := c.Run(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(sout),
				"stderr":  string(serr),
			}).Error("VPNSetup could not run ensure DNS command")
			return false
		}
		// collect output
		stdout = slices.Concat(stdout, sout)
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
			srvOK = v.checkDNSServers(config, f)

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
func (v *VPNSetup) ensureConfig(ctx context.Context, conf *daemoncfg.Config) {
	defer close(v.ensureClosed)

	timerInvalid := time.Second
	timerValid := 15 * time.Second
	timer := timerInvalid
	for {
		select {
		case <-time.After(timer):
			log.Debug("VPNSetup checking VPN configuration")

			// ensure DNS settings
			if ok := v.ensureDNS(ctx, conf); !ok {
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
func (v *VPNSetup) startEnsure(ctx context.Context, conf *daemoncfg.Config) {
	v.ensureDone = make(chan struct{})
	v.ensureClosed = make(chan struct{})
	go v.ensureConfig(ctx, conf)
}

// stopEnsure stops ensuring the VPN config.
func (v *VPNSetup) stopEnsure() {
	close(v.ensureDone)
	<-v.ensureClosed
}

// setup sets up the vpn configuration.
func (v *VPNSetup) setup(ctx context.Context, conf *daemoncfg.Config) {
	// setup device, routing, dns
	setupVPNDevice(ctx, conf)
	v.setupRouting(conf)
	v.setupDNS(ctx, conf)

	// ensure VPN config
	v.startEnsure(ctx, conf)
}

// teardown tears down the vpn configuration.
func (v *VPNSetup) teardown(ctx context.Context, conf *daemoncfg.Config) {
	// stop ensuring VPN config
	v.stopEnsure()

	// tear down device, routing, dns
	teardownVPNDevice(ctx, conf)
	v.teardownRouting()
	v.teardownDNS(ctx, conf)
}

// getState gets the internal state.
func (v *VPNSetup) getState(c *command) {
	state := &State{}
	if v.splitrt != nil {
		state.SplitRouting = v.splitrt.GetState()
	}
	if v.dnsProxy != nil {
		state.DNSProxy = v.dnsProxy.GetState()
	}
	c.state = state
}

// handleCommand handles a command.
func (v *VPNSetup) handleCommand(ctx context.Context, c *command) {
	defer close(c.done)

	switch c.cmd {
	case commandSetup:
		v.setup(ctx, c.conf)
	case commandTeardown:
		v.teardown(ctx, c.conf)
	case commandGetState:
		v.getState(c)
	}
}

// handleDNSReport handles a DNS report.
func (v *VPNSetup) handleDNSReport(r *dnsproxy.Report) {
	log.WithField("report", r).Debug("Daemon handling DNS report")

	if v.splitrt == nil {
		// split routing not active, close report and do not forward
		r.Close()
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
func (v *VPNSetup) Setup(conf *daemoncfg.Config) {
	c := &command{
		cmd:  commandSetup,
		conf: conf,
		done: make(chan struct{}),
	}
	v.cmds <- c
	<-c.done
}

// Teardown tears the VPN config down.
func (v *VPNSetup) Teardown(conf *daemoncfg.Config) {
	c := &command{
		cmd:  commandTeardown,
		conf: conf,
		done: make(chan struct{}),
	}
	v.cmds <- c
	<-c.done
}

// GetState returns the internal state of the VPN config.
func (v *VPNSetup) GetState() *State {
	c := &command{
		cmd:  commandGetState,
		done: make(chan struct{}),
	}
	v.cmds <- c
	<-c.done
	return c.state
}

// NewVPNSetup returns a new VPNSetup.
func NewVPNSetup(dnsProxy *dnsproxy.Proxy) *VPNSetup {
	return &VPNSetup{
		dnsProxy: dnsProxy,

		cmds:   make(chan *command),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}

// Cleanup cleans up the configuration after a failed shutdown.
func Cleanup(ctx context.Context, config *daemoncfg.Config) {
	// dns, device, split routing
	vpnDevice := config.OpenConnect.VPNDevice
	cmds, err := cmdtmpl.GetCmds("VPNSetupCleanup", config)
	if err != nil {
		log.WithError(err).Error("VPNSetup could not get cleanup commands")
	}
	for _, c := range cmds {
		if _, _, err := c.Run(ctx); err == nil {
			log.WithFields(log.Fields{
				"device":  vpnDevice,
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
			}).Warn("VPNSetup cleaned up configuration")
		}
	}
	splitrt.Cleanup(ctx, config)
}
