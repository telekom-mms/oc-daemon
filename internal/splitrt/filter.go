package splitrt

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	// FWMark is the firewall mark
	FWMark = rtTable
)

// runNft runs nft and passes s to it via stdin
var runNft = func(s string) {
	cmd := "nft -f -"
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = bytes.NewBufferString(s)
	if err := c.Run(); err != nil {
		log.WithError(err).Error("SplitRouting nft execution error")
	}
}

// setRoutingRules sets the basic nftables rules for routing
func setRoutingRules() {
	const routeRules = `
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
	}

	chain splitrouting {
		# restore mark from conntracking
		ct mark != 0 meta mark set ct mark counter
		meta mark != 0 counter accept

		# mark packets in exclude sets
		ip daddr @excludes4 counter meta mark set $FWMARK
		ip6 daddr @excludes6 counter meta mark set $FWMARK

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
		meta mark $FWMARK ct mark set meta mark counter
	}

	chain postrouting {
		type nat hook postrouting priority srcnat; policy accept;

		# masquerare tunnel/exclude traffic to make sure the source IP
		# matches the outgoing interface
		ct mark $FWMARK counter masquerade
	}
}
`
	r := strings.NewReplacer("$FWMARK", FWMark)
	rules := r.Replace(routeRules)
	runNft(rules)
}

// unsetRoutingRules removes the nftables rules for routing
func unsetRoutingRules() {
	runNft("delete table inet oc-daemon-routing")
}

// addLocalAddresses adds rules for device and its family (ip, ip6) addresses,
// that drop non-local traffic from other devices to device's network
// addresses; used to filter non-local traffic to vpn addresses
func addLocalAddresses(device, family string, addresses []*net.IPNet) {
	nftconf := ""
	for _, addr := range addresses {
		if addr == nil || len(addr.IP) == 0 || len(addr.Mask) == 0 {
			continue
		}
		nftconf += "add rule inet oc-daemon-routing preraw iifname != "
		nftconf += fmt.Sprintf("%s %s daddr %s ", device, family, addr)
		nftconf += "fib saddr type != local counter drop\n"
	}

	runNft(nftconf)
}

// addLocalAddressesIPv4 adds rules for device and its addresses, that drop
// non-local traffic from other devices to device's network addresses; used to
// filter non-local traffic to vpn addresses
func addLocalAddressesIPv4(device string, addresses []*net.IPNet) {
	addLocalAddresses(device, "ip", addresses)
}

// addLocalAddressesIPv6 adds rules for device and its addresses, that drop
// non-local traffic from other devices to device's network addresses; used to
// filter non-local traffic to vpn addresses
func addLocalAddressesIPv6(device string, addresses []*net.IPNet) {
	addLocalAddresses(device, "ip6", addresses)
}

// addExclude adds exclude address to netfilter
func addExclude(address *net.IPNet) {
	set := "excludes4"
	if address.IP.To4() == nil {
		set = "excludes6"
	}

	nftconf := fmt.Sprintf("add element inet oc-daemon-routing %s { %s }",
		set, address)
	runNft(nftconf)
}

// setExcludes resets the excludes to addresses in netfilter
func setExcludes(addresses []*net.IPNet) {
	// flush existing entries
	nftconf := ""
	nftconf += "flush set inet oc-daemon-routing excludes4\n"
	nftconf += "flush set inet oc-daemon-routing excludes6\n"

	// add entries
	for _, a := range addresses {
		set := "excludes4"
		if a.IP.To4() == nil {
			set = "excludes6"
		}
		nftconf += fmt.Sprintf(
			"add element inet oc-daemon-routing %s { %s }\n",
			set, a)
	}

	// run command
	runNft(nftconf)
}

// runCleanupNft runs nft for cleanups
var runCleanupNft = func(s string) {
	log.WithField("stdin", s).Debug("SplitRouting executing nft cleanup command")
	cmd := "nft -f -"
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = bytes.NewBufferString(s)
	if err := c.Run(); err == nil {
		log.WithField("stdin", s).Debug("SplitRouting cleaned up nft")
	}
}

// cleanupRoutingRules cleans up the nftables rules for routing after a
// failed shutdown
func cleanupRoutingRules() {
	runCleanupNft("delete table inet oc-daemon-routing")
}
