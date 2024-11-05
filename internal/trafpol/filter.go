package trafpol

import (
	"context"
	"errors"
	"net/netip"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
)

const DefaultTemplates = `
{{- define "FilterRules"}}
table inet oc-daemon-filter {
        # set for allowed devices
        set allowdevs {
                type ifname
                elements = { lo }
        }

        # set for allowed ipv4 hosts
        set allowhosts4 {
                type ipv4_addr
                flags interval
        }

        # set for allowed ipv6 hosts
        set allowhosts6 {
                type ipv6_addr
                flags interval
        }

        # set for allowed ports
        set allowports {
                type inet_service
                elements = { 53 }
        }

        chain input {
                type filter hook input priority 0; policy drop;

                # accept related traffic
                ct state established,related counter accept

                # accept traffic on allowed devices, e.g., lo
                iifname @allowdevs counter accept

                # accept ICMP traffic
                icmp type {
                        echo-reply,
                        destination-unreachable,
                        source-quench,
                        redirect,
                        time-exceeded,
                        parameter-problem,
                        timestamp-reply,
                        info-reply,
                        address-mask-reply,
                        router-advertisement,
                } counter accept

                # accept ICMPv6 traffic otherwise IPv6 connectivity breaks
                icmpv6 type {
                        destination-unreachable,
                        packet-too-big,
                        time-exceeded,
                        echo-reply,
                        mld-listener-query,
                        mld-listener-report,
                        mld2-listener-report,
                        mld-listener-done,
                        nd-router-advert,
                        nd-neighbor-solicit,
                        nd-neighbor-advert,
                        ind-neighbor-solicit,
                        ind-neighbor-advert,
                        nd-redirect,
                        parameter-problem,
                        router-renumbering
                } counter accept

                # accept DHCPv4 traffic
                udp dport 68 udp sport 67 counter accept

                # accept DHCPv6 traffic
                udp dport 546 udp sport 547 counter accept
        }

        chain output {
                type filter hook output priority 0; policy drop;

                # accept related traffic
                ct state established,related counter accept

                # accept traffic on allowed devices, e.g., lo
                oifname @allowdevs counter accept

                # accept traffic to allowed hosts
                ip daddr @allowhosts4 counter accept
                ip6 daddr @allowhosts6 counter accept

                # accept ICMP traffic
                icmp type {
                        source-quench,
                        echo-request,
                        timestamp-request,
                        info-request,
                        address-mask-request,
                        router-solicitation
                } counter accept

                # accept ICMPv6 traffic otherwise IPv6 connectivity breaks
                icmpv6 type {
                        echo-request,
                        mld-listener-report,
                        mld2-listener-report,
                        mld-listener-done,
                        nd-router-solicit,
                        nd-neighbor-solicit,
                        nd-neighbor-advert,
                        ind-neighbor-solicit,
                        ind-neighbor-advert,
                } counter accept

                # accept traffic to allowed ports, e.g., DNS
                udp dport @allowports counter accept
                tcp dport @allowports counter accept

                # accept DHCPv4 traffic
                udp dport 67 udp sport 68 counter accept

                # accept DHCPv6 traffic
                udp dport 547 udp sport 546 counter accept

                # reject everything else
                counter reject
        }

        chain forward {
                type filter hook forward priority 0; policy drop;

                # accept related traffic
                ct state established,related counter accept

                # accept split exclude traffic
                iifname @allowdevs ct mark {{.FirewallMark}} counter accept

                # accept traffic on allowed devices
                iifname @allowdevs oifname @allowdevs counter accept
        }
}
{{end}}`

// setFilterRules sets the filter rules.
func setFilterRules(ctx context.Context, config *Config) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := config // TODO: change?
	commands := []*cmdtmpl.Command{
		{Line: "nft -f -", Stdin: `{{template "FilterRules" .}}`},
	}
	for _, c := range commands {
		// TODO: get final command and stdin
		// TODO: add LogError() helper?
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}
}

// unsetFilterRules unsets the filter rules.
func unsetFilterRules(ctx context.Context) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := "" // TODO: change?
	commands := []*cmdtmpl.Command{
		{Line: "nft -f - delete table inet oc-daemon-filter"},
	}

	for _, c := range commands {
		// TODO: get final command and stdin
		// TODO: add LogError() helper?
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}
}

// addAllowedDevice adds device to the allowed devices.
func addAllowedDevice(ctx context.Context, device string) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := device // TODO: change?
	commands := []*cmdtmpl.Command{
		{Line: "nft -f - add element inet oc-daemon-filter allowdevs { {{.}} }"},
	}

	for _, c := range commands {
		// TODO: get final command and stdin
		// TODO: add LogError() helper?
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}
}

// removeAllowedDevice removes device from the allowed devices.
func removeAllowedDevice(ctx context.Context, device string) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := device // TODO: change?
	commands := []*cmdtmpl.Command{
		{Line: "nft -f - delete element inet oc-daemon-filter allowdevs { {{.}} }"},
	}

	for _, c := range commands {
		// TODO: get final command and stdin
		// TODO: add LogError() helper?
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}
}

// setAllowedIPs set the allowed hosts.
func setAllowedIPs(ctx context.Context, ips []netip.Prefix) {
	// we perform all nft commands separately here and not as one atomic
	// operation to avoid issues where the whole update fails because nft
	// runs into "file exists" errors even though we remove duplicates from
	// ips before calling this function and we flush the existing entries

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := "" // TODO: change?

	// flush allowed hosts
	commands := []*cmdtmpl.Command{
		{Line: "nft -f - flush set inet oc-daemon-filter allowhosts4"},
		{Line: "nft -f - flush set inet oc-daemon-filter allowhosts6"},
	}

	for _, c := range commands {
		// TODO: get final command and stdin
		// TODO: add LogError() helper?
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}

	for _, ip := range ips {
		data := ip // TODO: change?
		commands := []*cmdtmpl.Command{
			{Line: "nft -f -",
				Stdin: `
				{{if .Addr.Is4}}
				add element inet oc-daemon-filter allowhosts4 { {{.}} }
				{{else}}
				add element inet oc-daemon-filter allowhosts6 { {{.}} }
				{{end}}
				`,
			},
		}
		for _, c := range commands {
			// TODO: get final command and stdin
			// TODO: add LogError() helper?
			stdout, stderr, err := ct.RunCommand(ctx, c, data)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.WithError(err).WithFields(log.Fields{
					"command": c.Line,
					"stdin":   c.Stdin,
					"stdout":  string(stdout),
					"stderr":  string(stderr),
				}).Error("Error executing command")
			}
		}
	}
}

// portsToString returns ports as string.
func portsToString(ports []uint16) string {
	s := []string{}
	for _, port := range ports {
		s = append(s, strconv.FormatUint(uint64(port), 10))
	}
	return strings.Join(s, ", ")
}

// addPortalPorts adds ports for a captive portal to the allowed ports.
func addPortalPorts(ctx context.Context, ports []uint16) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := portsToString(ports) // TODO: change?
	commands := []*cmdtmpl.Command{
		{Line: "nft -f - add element inet oc-daemon-filter allowports { {{.}} }"},
	}

	for _, c := range commands {
		// TODO: get final command and stdin
		// TODO: add LogError() helper?
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}
}

// removePortalPorts removes ports for a captive portal from the allowed ports.
func removePortalPorts(ctx context.Context, ports []uint16) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := portsToString(ports) // TODO: change?
	commands := []*cmdtmpl.Command{
		{Line: "nft -f - delete element inet oc-daemon-filter allowports { {{.}} }"},
	}

	for _, c := range commands {
		// TODO: get final command and stdin
		// TODO: add LogError() helper?
		stdout, stderr, err := ct.RunCommand(ctx, c, data)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).WithFields(log.Fields{
				"command": c.Line,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command")
		}
	}

}

// cleanupFilterRules cleans up the filter rules after a failed shutdown.
func cleanupFilterRules(ctx context.Context) {

	ct := cmdtmpl.NewCommandTemplates(DefaultTemplates)
	data := "" // TODO: change?
	commands := []*cmdtmpl.Command{
		{Line: "nft -f - delete table inet oc-daemon-filter"},
	}

	for _, c := range commands {
		// TODO: get final command and stdin
		// TODO: add LogError() helper?
		// TODO: check template vs execution error
		// stdout, stderr, err := ct.RunCommand(ctx, c, data)
		// if err != nil && !errors.Is(err, context.Canceled) {
		//	log.WithError(err).WithFields(log.Fields{
		//		"command": c.Line,
		//		"stdin":   c.Stdin,
		//		"stdout":  string(stdout),
		//		"stderr":  string(stderr),
		//	}).Error("Error executing command")
		// }
		if _, _, err := ct.RunCommand(ctx, c, data); err == nil {
			log.Debug("TrafPol cleaned up nft")
		}
	}
}
