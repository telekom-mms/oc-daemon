package vpnsetup

import (
	"fmt"
	"net"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/internal/splitrt"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/vishvananda/netlink"
)

// command types
const (
	commandSetup uint8 = iota
	commandTeardown
)

// command is a VPNSetup command
type command struct {
	cmd     uint8
	vpnconf *vpnconfig.Config
}

// Event types
const (
	EventSetupOK uint8 = iota
	EventTeardownOK
)

// Event is a VPNSetup event
type Event struct {
	Type uint8
}

// VPNSetup sets up the configuration of the vpn tunnel that belongs to the
// current VPN connection
type VPNSetup struct {
	splitrt     *splitrt.SplitRouting
	splitrtConf *splitrt.Config

	dnsProxy     *dnsproxy.Proxy
	dnsProxyConf *dnsproxy.Config

	cmds   chan *command
	events chan *Event
	done   chan struct{}
}

// sendEvents sends the event
func (v *VPNSetup) sendEvent(event *Event) {
	select {
	case v.events <- event:
	case <-v.done:
	}
}

// runLinkByname is a helper for getting a link by name
var runLinkByName = func(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

// runLinkSetMTU is a helper for setting the mtu of link
var runLinkSetMTU = func(link netlink.Link, mtu int) error {
	return netlink.LinkSetMTU(link, mtu)
}

// runLinkSetUp is a helper for setting link up
var runLinkSetUp = func(link netlink.Link) error {
	return netlink.LinkSetUp(link)
}

// runLinkSetDown is a helper for setting link down
var runLinkSetDown = func(link netlink.Link) error {
	return netlink.LinkSetDown(link)
}

// runAddrAdd is a helper for adding address to link
var runAddrAdd = func(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}

// setupVPNDevice sets up the vpn device with config
func setupVPNDevice(c *vpnconfig.Config) {
	// get link for device
	link, err := runLinkByName(c.Device.Name)
	if err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not find device")
		return
	}

	// set mtu on device
	if err := runLinkSetMTU(link, c.Device.MTU); err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not set mtu on device")
		return
	}

	// set device up
	if err := runLinkSetUp(link); err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not set device up")
		return
	}

	// set ipv4 and ipv6 addresses on device
	setupIP := func(ip net.IP, mask net.IPMask) {
		ipnet := &net.IPNet{
			IP:   ip,
			Mask: mask,
		}
		addr := &netlink.Addr{
			IPNet: ipnet,
		}
		if err := runAddrAdd(link, addr); err != nil {
			log.WithFields(log.Fields{
				"device": c.Device.Name,
				"ip":     ip,
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

// teardownVPNDevice tears down the configured vpn device
func teardownVPNDevice(c *vpnconfig.Config) {
	// get link for device
	link, err := runLinkByName(c.Device.Name)
	if err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not find device ")
		return
	}

	// set device down
	if err := runLinkSetDown(link); err != nil {
		log.WithField("device", c.Device.Name).
			Error("Daemon could not set device down")
		return
	}

}

// setupRouting sets up routing using config
func (v *VPNSetup) setupRouting(vpnconf *vpnconfig.Config) {
	if v.splitrt != nil {
		return
	}
	v.splitrt = splitrt.NewSplitRouting(v.splitrtConf, vpnconf)
	v.splitrt.Start()
}

// teardownRouting tears down the routing configuration
func (v *VPNSetup) teardownRouting() {
	if v.splitrt == nil {
		return
	}
	v.splitrt.Stop()
	v.splitrt = nil
}

// runResolvctl runs the resolvectl cmd
var runResolvectl = func(cmd string) {
	log.WithField("command", cmd).Debug("Daemon executing resolvectl command")
	c := exec.Command("bash", "-c", "resolvectl "+cmd)
	if err := c.Run(); err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
		}).Error("Daemon resolvectl command execution error")
	}
}

// setupDNS sets up DNS using config
func (v *VPNSetup) setupDNS(config *vpnconfig.Config) {
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
	device := config.Device.Name
	runResolvectl(fmt.Sprintf("dns %s %s", device, v.dnsProxyConf.Address))

	// set domains for device
	// this includes "~." to use this device for all domains
	runResolvectl(fmt.Sprintf("domain %s %s ~.", device, config.DNS.DefaultDomain))

	// set default route for device
	runResolvectl(fmt.Sprintf("default-route %s yes", device))

	// flush dns caches
	runResolvectl("flush-caches")

	// reset learnt server features
	runResolvectl("reset-server-features")
}

// teardownDNS tears down the DNS configuration
func (v *VPNSetup) teardownDNS(vpnconf *vpnconfig.Config) {
	// update dns proxy configuration

	// reset remotes
	remotes := map[string][]string{}
	v.dnsProxy.SetRemotes(remotes)

	// reset watches
	v.dnsProxy.SetWatches([]string{})

	// update dns configuration of host

	// undo device dns configuration
	runResolvectl(fmt.Sprintf("revert %s", vpnconf.Device.Name))

	// flush dns caches
	runResolvectl("flush-caches")

	// reset learnt server features
	runResolvectl("reset-server-features")
}

// setup sets up the vpn configuration
func (v *VPNSetup) setup(vpnconf *vpnconfig.Config) {
	setupVPNDevice(vpnconf)
	v.setupRouting(vpnconf)
	v.setupDNS(vpnconf)

	v.sendEvent(&Event{EventSetupOK})
}

// teardown tears down the vpn configuration
func (v *VPNSetup) teardown(vpnconf *vpnconfig.Config) {
	teardownVPNDevice(vpnconf)
	v.teardownRouting()
	v.teardownDNS(vpnconf)

	v.sendEvent(&Event{EventTeardownOK})
}

// handleCommand handles a command
func (v *VPNSetup) handleCommand(c *command) {
	switch c.cmd {
	case commandSetup:
		v.setup(c.vpnconf)
	case commandTeardown:
		v.teardown(c.vpnconf)
	}
}

// handleDNSReport handles a DNS report
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

// start starts the VPN setup
func (v *VPNSetup) start() {
	defer close(v.events)

	// start DNS-Proxy
	v.dnsProxy.Start()
	defer v.dnsProxy.Stop()

	for {
		select {
		case c := <-v.cmds:
			v.handleCommand(c)
		case r := <-v.dnsProxy.Reports():
			v.handleDNSReport(r)
		case <-v.done:
			return
		}
	}
}

// Start starts the VPN setup
func (v *VPNSetup) Start() {
	go v.start()
}

// Stop stops the VPN setup
func (v *VPNSetup) Stop() {
	close(v.done)
	for range v.events {
		// wait until channel closed
	}
}

// Setup sets the VPN config up
func (v *VPNSetup) Setup(vpnconfig *vpnconfig.Config) {
	v.cmds <- &command{
		cmd:     commandSetup,
		vpnconf: vpnconfig,
	}
}

// Teardown tears the VPN config down
func (v *VPNSetup) Teardown(vpnconfig *vpnconfig.Config) {
	v.cmds <- &command{
		cmd:     commandTeardown,
		vpnconf: vpnconfig,
	}
}

// Events returns the events channel
func (v *VPNSetup) Events() chan *Event {
	return v.events
}

// NewVPNSetup returns a new VPNSetup
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
	}
}

// runCleanupCmd runs cmd for cleanups
var runCleanupCmd = func(cmd string) {
	log.WithField("command", cmd).Debug("Daemon executing vpn config cleanup command")
	c := exec.Command("bash", "-c", cmd)
	if err := c.Run(); err == nil {
		log.WithField("command", cmd).Warn("Daemon cleaned up vpn config")
	}
}

// Cleanup cleans up the configuration after a failed shutdown
func Cleanup(vpnDevice string, splitrtConfig *splitrt.Config) {
	// dns, device, split routing
	runCleanupCmd(fmt.Sprintf("resolvectl revert %s", vpnDevice))
	runCleanupCmd(fmt.Sprintf("ip link delete %s", vpnDevice))
	splitrt.Cleanup(splitrtConfig)
}
