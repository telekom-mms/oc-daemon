package splitrt

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/execs"
)

const routingScript = `
{{define "RoutingRules"}}
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
		{{with IPv4Address}}
		iifname != {{Device}} ip daddr {{.}} fib saddr type != local counter drop
		{{end}}
		{{with IPv6Address}}
		iifname != {{Device}} ip6 daddr {{.}} fib saddr type != local counter drop
		{{end}}
	}

	chain splitrouting {
		# restore mark from conntracking
		ct mark != 0 meta mark set ct mark counter
		meta mark != 0 counter accept

		# mark packets in exclude sets
		ip daddr @excludes4 counter meta mark set {{FWMark}}
		ip6 daddr @excludes6 counter meta mark set {{FWMark}}

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
		meta mark {{WMark}} ct mark set meta mark counter
	}

	chain postrouting {
		type nat hook postrouting priority srcnat; policy accept;

		# masquerare tunnel/exclude traffic to make sure the source IP
		# matches the outgoing interface
		ct mark {{FWMark}} counter masquerade
	}

	chain rejectipversion {
		# used to reject unsupported ip version on the tunnel device

		# make sure exclude traffic is not filtered
		ct mark {{FWMark}} counter accept

		# use tcp reset and icmp admin prohibited
		meta l4proto tcp counter reject with tcp reset
		counter reject with icmpx admin-prohibited
	}

	chain rejectforward {
		type filter hook forward priority filter; policy accept;

		# reject unsupported ip versions when forwarding packets,
		# add matching jump rule to rejectipversion if necessary
		{{if IPv4Address}}
		meta oifname {{Device}} meta nfproto ipv6 counter jump rejectipversion
		{{end}}
		{{if IPv6Address}}
		meta oifname {{Device}} meta nfproto ipv4 counter jump rejectipversion
		{{end}}
	}

	chain rejectoutput {
		type filter hook output priority filter; policy accept;

		# reject unsupported ip versions when sending packets,
		# add matching jump rule to rejectipversion if necessary
		{{if IPv4Address}}
		meta oifname {{Device}} meta nfproto ipv6 counter jump rejectipversion
		{{end}}
		{{if IPv6Address}}
		meta oifname {{Device}} meta nfproto ipv4 counter jump rejectipversion
		{{end}}
	}
}
{{end}}

{{/* TODO: change to SetupRouting and add nft -f - command and use RoutingRules as input? */}}
{{/* TODO: use variables for tools, e.g., IP, NFT, Sysctl? */}}
{{define "SetupRouting"}}

{{/* setup nftables routing rules */}}
{{/* TODO use nft -f - <<EOF {{template "RoutingRules"}} EOF? */}}
nft -f - {{RoutingRules}}

{{/* TODO: can we add a line for the static excludes here? probably not */}}

{{/* setup IPv4 routing */}}
ip -4 route add 0.0.0.0/0 dev {{Device}} table {{RTTable}}
ip -4 rule add iif {{Device}} table main pref {{RulePrio1}}
ip -4 rule add not fwmark {{FWMark}} table {{RTTable}} pref {{RulePrio2}}
sysctl -q net.ipv4.conf.all.src_valid_mark=1

{{/* setup IPv6 routing */}}
ip -6 route add ::/0 dev {{Device}} table {{RTTable}}
ip -6 rule add iif {{Device}} table main pref {{RulePrio1}}
ip -6 rule add not fwmark {{FWMark}} table {{RTTable}} pref {{RulePrio2}}

{{end}}

{{define "TeardownRouting"}}
{{/* TODO:  rename to StartRouting and StopRouting? */}}

{{/* teardown IPv4 routing */}}
ip -4 rule delete table {{RTTable}}
ip -4 rule delete iif {{Device}} table main

{{/* teardown IPv6 routing */}}
ip -6 rule delete table {{RTTable}}
ip -6 rule delete iif {{Device}} table main

{{/* teardown nftables routing rules */}}
nft -f - delete table inet oc-daemon-routing

{{end}}

{{define "CleanupRouting"}}
{{/* TODO: just use template TeardownRouting? */}}

{{/* cleanup routing */}}
ip -4 rule delete pref {{RulePrio1}}
ip -4 rule delete pref {{RulePrio2}}
ip -6 rule delete pref {{RulePrio1}}
ip -6 rule delete pref {{RulePrio2}}
ip -4 route flush table {{RTTable}}
ip -6 route flush table {{RTTable}}

{{/* cleanup nftables routing rules */}}
nft -f - delete table inet oc-daemon-routing

{{end}}

{{define "AddExclude"}}

{{if {{.Is6}}}}
nft -f - add element inet oc-daemon-routing excludes6 { {{.}} }
{{else}}
nft -f - add element inet oc-daemon-routing excludes4 { {{.}} }
{{end}}

{{end}}

{{define "SetExcludes"}}

nft -f - <<EOF
flush set inet oc-daemon-routing excludes4
flush set inet oc-daemon-routing excludes6
{{range {{.}}}}
{{template "AddExclude"}}
{{end}}
EOF

{{end}}
`

// TODO: check if we really want it in init
func init() {
	startRouting := execs.CommandList{
		Name: "StartRouting",
		Commands: []execs.Command{
			{Line: "nft -f -", Stdin: "{{RoutingRules}}"},
			//{Line: "nft", Args: []string{"-f", "-"}, Stdin: "{{RoutingRules}}"},
			{Line: "ip -4 route add 0.0.0.0/0 dev {{Device}} table {{RTTable}}"},
			{Line: "ip -4 rule add iif {{Device}} table main pref {{RulePrio1}}"},
			{Line: "ip -4 rule add not fwmark {{FWMark}} table {{RTTable}} pref {{RulePrio2}}"},
			{Line: "sysctl -q net.ipv4.conf.all.src_valid_mark=1"},
			{Line: "ip -6 route add ::/0 dev {{Device}} table {{RTTable}}"},
			{Line: "ip -6 rule add iif {{Device}} table main pref {{RulePrio1}}"},
			{Line: "ip -6 rule add not fwmark {{FWMark}} table {{RTTable}} pref {{RulePrio2}}"},
		},
	}
	log.Println(startRouting)

	stopRouting := execs.CommandList{
		Name: "StopRouting",
		Commands: []execs.Command{
			{Line: "ip -4 rule delete table {{RTTable}}"},
			{Line: "ip -4 rule delete iif {{Device}} table main"},
			{Line: "ip -6 rule delete table {{RTTable}}"},
			{Line: "ip -6 rule delete iif {{Device}} table main"},
			{Line: "nft -f - delete table inet oc-daemon-routing"},
		},
	}
	log.Println(stopRouting)

	cleanupRouting := execs.CommandList{
		Name: "CleanupRouting",
		Commands: []execs.Command{
			{Line: "ip -4 rule delete pref {{RulePrio1}}"},
			{Line: "ip -4 rule delete pref {{RulePrio2}}"},
			{Line: "ip -6 rule delete pref {{RulePrio1}}"},
			{Line: "ip -6 rule delete pref {{RulePrio2}}"},
			{Line: "ip -4 route flush table {{RTTable}}"},
			{Line: "ip -6 route flush table {{RTTable}}"},
			{Line: "nft -f - delete table inet oc-daemon-routing"},
		},
	}
	log.Println(cleanupRouting)

	addExclude := execs.CommandList{
		Name: "AddExclude",
		Commands: []execs.Command{
			{Line: "{{if {{.Is6}}}}nft -f - add element inet oc-daemon-routing excludes6 { {{.}} }{{else}}nft -f - add element inet oc-daemon-routing excludes4 { {{.}} }{{end}}"},
			{Line: "nft -f -",
				Stdin: `
				{{if {{.Is6}}
				add element inet oc-daemon-routing excludes6 { {{.}} }
				{{else}}
				add element inet oc-daemon-routing excludes4 { {{.}} }
				{{end}}`},
		},
	}
	log.Println(addExclude)

	setExcludes := execs.CommandList{
		Name: "SetExcludes",
		Commands: []execs.Command{
			{Line: `nft -f - <<EOF
flush set inet oc-daemon-routing excludes4
flush set inet oc-daemon-routing excludes6
{{range {{.}}}}
{{template "AddExclude"}}
{{end}}
EOF"`},
			{Line: "nft -f -",
				Stdin: `flush set inet oc-daemon-routing excludes4
flush set inet oc-daemon-routing excludes6
{{range {{.}}}}
{{template "AddExclude"}}
{{end}}`},
		},
	}
	log.Println(setExcludes)
}

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

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := address
	commands := []*cmdtmpl.Command{
		//{Line: "{{if {{.Is6}}}}nft -f - add element inet oc-daemon-routing excludes6 { {{.}} }{{else}}nft -f - add element inet oc-daemon-routing excludes4 { {{.}} }{{end}}"},
		{Line: "nft -f -",
			Stdin: `
				{{- if .Addr.Is6 -}}
				add element inet oc-daemon-routing excludes6 { {{.}} }
				{{- else -}}
				add element inet oc-daemon-routing excludes4 { {{.}} }
				{{- end}}`},
	}
	for _, c := range commands {
		// TODO: get final command and stdin
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"address": address,
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error adding exclude")
		}
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
