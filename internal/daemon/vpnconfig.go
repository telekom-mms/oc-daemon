package daemon

import (
	"fmt"
	"net"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/vishvananda/netlink"
)

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
			}).Error("Daemon could net set ip on device")
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

// setVPNDNS applies the DNS configuration and sets dns server address;
// the server address should be the local DNS-Proxy
func setVPNDNS(c *vpnconfig.Config, server string) {
	device := c.Device.Name
	search := c.DNS.DefaultDomain

	// set dns server for device
	runResolvectl(fmt.Sprintf("dns %s %s", device, server))

	// set domains for device
	// this includes "~." to use this device for all domains
	runResolvectl(fmt.Sprintf("domain %s %s ~.", device, search))

	// set default route for device
	runResolvectl(fmt.Sprintf("default-route %s yes", device))

	// flush dns caches
	runResolvectl("flush-caches")

	// reset learnt server features
	runResolvectl("reset-server-features")
}

// unsetVPNDNS unsets the DNS configuration
func unsetVPNDNS(c *vpnconfig.Config) {
	device := c.Device.Name

	// undo device dns configuration
	runResolvectl(fmt.Sprintf("revert %s", device))

	// flush dns caches
	runResolvectl("flush-caches")

	// reset learnt server features
	runResolvectl("reset-server-features")
}

// runCleanupCmd runs cmd for cleanups
var runCleanupCmd = func(cmd string) {
	log.WithField("command", cmd).Debug("Daemon executing vpn config cleanup command")
	c := exec.Command("bash", "-c", cmd)
	if err := c.Run(); err == nil {
		log.WithField("command", cmd).Warn("Daemon cleaned up vpn config")
	}
}

// cleanupVPNConfig cleans up the configuration after a failed shutdown
func cleanupVPNConfig(device string) {
	runCleanupCmd(fmt.Sprintf("resolvectl revert %s", device))
	runCleanupCmd(fmt.Sprintf("ip link delete %s", device))
}
