package splitrt

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// setRoutingRules sets the basic nftables rules for routing.
func setRoutingRules(ctx context.Context, fwMark string) {
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

	chain rejectipversion {
		# used to reject unsupported ip version on the tunnel device

		# make sure exclude traffic is not filtered
		ct mark $FWMARK counter accept

		# use tcp reset and icmp admin prohibited
		meta l4proto tcp counter reject with tcp reset
		counter reject with icmpx admin-prohibited
	}

	chain rejectforward {
		type filter hook forward priority filter; policy accept;

		# reject unsupported ip versions when forwarding packets,
		# add matching jump rule to rejectipversion if necessary
	}

	chain rejectoutput {
		type filter hook output priority filter; policy accept;

		# reject unsupported ip versions when sending packets,
		# add matching jump rule to rejectipversion if necessary
	}
}
`
	r := strings.NewReplacer("$FWMARK", fwMark)
	rules := r.Replace(routeRules)
	if stdout, stderr, err := execs.RunNft(ctx, rules); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"fwMark": fwMark,
			"stdout": string(stdout),
			"stderr": string(stderr),
		}).Error("SplitRouting error setting routing rules")
	}
}

// unsetRoutingRules removes the nftables rules for routing.
func unsetRoutingRules(ctx context.Context) {
	if stdout, stderr, err := execs.RunNft(ctx, "delete table inet oc-daemon-routing"); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"stdout": string(stdout),
			"stderr": string(stderr),
		}).Error("SplitRouting error unsetting routing rules")
	}
}

// addLocalAddresses adds rules for device and its family (ip, ip6) addresses,
// that drop non-local traffic from other devices to device's network
// addresses; used to filter non-local traffic to vpn addresses.
func addLocalAddresses(ctx context.Context, device, family string, addresses []netip.Prefix) {
	nftconf := ""
	for _, addr := range addresses {
		if !addr.IsValid() {
			continue
		}
		nftconf += "add rule inet oc-daemon-routing preraw iifname != "
		nftconf += fmt.Sprintf("%s %s daddr %s ", device, family, addr)
		nftconf += "fib saddr type != local counter drop\n"
	}

	if stdout, stderr, err := execs.RunNft(ctx, nftconf); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"device":    device,
			"family":    family,
			"addresses": addresses,
			"stdout":    string(stdout),
			"stderr":    string(stderr),
		}).Error("SplitRouting error adding local addresses")
	}
}

// addLocalAddressesIPv4 adds rules for device and its addresses, that drop
// non-local traffic from other devices to device's network addresses; used to
// filter non-local traffic to vpn addresses.
func addLocalAddressesIPv4(ctx context.Context, device string, addresses []netip.Prefix) {
	addLocalAddresses(ctx, device, "ip", addresses)
}

// addLocalAddressesIPv6 adds rules for device and its addresses, that drop
// non-local traffic from other devices to device's network addresses; used to
// filter non-local traffic to vpn addresses.
func addLocalAddressesIPv6(ctx context.Context, device string, addresses []netip.Prefix) {
	addLocalAddresses(ctx, device, "ip6", addresses)
}

// rejectIPVersion adds rules for the tunnel device to reject an unsupported ip
// version ("ipv6" or "ipv4").
func rejectIPVersion(ctx context.Context, device, version string) {
	nftconf := ""
	for _, chain := range []string{"rejectforward", "rejectoutput"} {
		nftconf += fmt.Sprintf("add rule inet oc-daemon-routing %s ",
			chain)
		nftconf += fmt.Sprintf("meta oifname %s meta nfproto %s ",
			device, version)
		nftconf += "counter jump rejectipversion\n"
	}

	if stdout, stderr, err := execs.RunNft(ctx, nftconf); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"device":  device,
			"version": version,
			"stdout":  string(stdout),
			"stderr":  string(stderr),
		}).Error("SplitRouting error setting ip version reject rules")
	}
}

// rejectIPv6 adds rules for the tunnel device that reject IPv6 traffic on it;
// used to avoid sending IPv6 packets over a tunnel that only supports IPv4.
func rejectIPv6(ctx context.Context, device string) {
	rejectIPVersion(ctx, device, "ipv6")
}

// rejectIPv4 adds rules for the tunnel device that reject IPv4 traffic on it;
// used to avoid sending IPv4 packets over a tunnel that only supports IPv6.
func rejectIPv4(ctx context.Context, device string) {
	rejectIPVersion(ctx, device, "ipv4")
}

// addExclude adds exclude address to netfilter.
func addExclude(ctx context.Context, address netip.Prefix) {
	log.WithField("address", address).Debug("SplitRouting adding exclude to netfilter")

	set := "excludes4"
	if address.Addr().Is6() {
		set = "excludes6"
	}

	nftconf := fmt.Sprintf("add element inet oc-daemon-routing %s { %s }",
		set, address)
	if stdout, stderr, err := execs.RunNft(ctx, nftconf); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"address": address,
			"stdout":  string(stdout),
			"stderr":  string(stderr),
		}).Error("SplitRouting error adding exclude")
	}
}

// setExcludes resets the excludes to addresses in netfilter.
func setExcludes(ctx context.Context, addresses []netip.Prefix) {
	// flush existing entries
	nftconf := ""
	nftconf += "flush set inet oc-daemon-routing excludes4\n"
	nftconf += "flush set inet oc-daemon-routing excludes6\n"

	// add entries
	for _, a := range addresses {
		set := "excludes4"
		if a.Addr().Is6() {
			set = "excludes6"
		}
		nftconf += fmt.Sprintf(
			"add element inet oc-daemon-routing %s { %s }\n",
			set, a)
	}

	// run command
	if stdout, stderr, err := execs.RunNft(ctx, nftconf); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"addresses": addresses,
			"stdout":    string(stdout),
			"stderr":    string(stderr),
		}).Error("SplitRouting error setting excludes")
	}
}

// cleanupRoutingRules cleans up the nftables rules for routing after a
// failed shutdown.
func cleanupRoutingRules(ctx context.Context) {
	if _, _, err := execs.RunNft(ctx, "delete table inet oc-daemon-routing"); err == nil {
		log.Debug("SplitRouting cleaned up nft")
	}
}
