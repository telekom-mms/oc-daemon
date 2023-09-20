package trafpol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// runNft runs nft and passes s to it via stdin
var runNft = func(ctx context.Context, s string) {
	cmd := "nft -f -"
	c := exec.CommandContext(ctx, "bash", "-c", cmd)
	c.Stdin = bytes.NewBufferString(s)
	if err := c.Run(); err != nil {
		if errors.Is(err, context.Canceled) {
			log.WithError(err).Debug("TrafPol nft execution canceled")
			return
		}
		log.WithError(err).Error("TrafPol nft execution error")
	}
}

// setFilterRules sets the filter rules
func setFilterRules(ctx context.Context, fwMark string) {
	const filterRules = `
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
                iifname @allowdevs ct mark $FWMARK counter accept

                # accept traffic on allowed devices
                iifname @allowdevs oifname @allowdevs counter accept
        }
}
`
	r := strings.NewReplacer("$FWMARK", fwMark)
	rules := r.Replace(filterRules)
	runNft(ctx, rules)
}

// unsetFilterRules unsets the filter rules
func unsetFilterRules(ctx context.Context) {
	runNft(ctx, "delete table inet oc-daemon-filter")
}

// addAllowedDevice adds device to the allowed devices
func addAllowedDevice(ctx context.Context, device string) {
	nftconf := fmt.Sprintf("add element inet oc-daemon-filter allowdevs { %s }", device)
	runNft(ctx, nftconf)
}

// removeAllowedDevice removes device from the allowed devices
func removeAllowedDevice(ctx context.Context, device string) {
	nftconf := fmt.Sprintf("delete element inet oc-daemon-filter allowdevs { %s }", device)
	runNft(ctx, nftconf)
}

// setAllowedIPs set the allowed hosts
func setAllowedIPs(ctx context.Context, ips []*net.IPNet) {
	// we perform all nft commands separately here and not as one atomic
	// operation to avoid issues where the whole update fails because nft
	// runs into "file exists" errors even though we remove duplicates from
	// ips before calling this function and we flush the existing entries

	runNft(ctx, "flush set inet oc-daemon-filter allowhosts4")
	runNft(ctx, "flush set inet oc-daemon-filter allowhosts6")

	fmt4 := "add element inet oc-daemon-filter allowhosts4 { %s }"
	fmt6 := "add element inet oc-daemon-filter allowhosts6 { %s }"
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			// ipv4 address
			runNft(ctx, fmt.Sprintf(fmt4, ip))
		} else {
			// ipv6 address
			runNft(ctx, fmt.Sprintf(fmt6, ip))
		}
	}
}

// addPortalPorts adds ports for a captive portal to the allowed ports
func addPortalPorts(ctx context.Context) {
	nftconf := "add element inet oc-daemon-filter allowports { 80, 443 }"
	runNft(ctx, nftconf)
}

// removePortalPorts removes ports for a captive portal from the allowed ports
func removePortalPorts(ctx context.Context) {
	nftconf := "delete element inet oc-daemon-filter allowports { 80, 443 }"
	runNft(ctx, nftconf)
}

// runCleanupNft runs nft for cleanups
var runCleanupNft = func(s string) {
	log.WithField("stdin", s).Debug("TrafPol executing nft cleanup command")
	cmd := "nft -f -"
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = bytes.NewBufferString(s)
	if err := c.Run(); err == nil {
		log.WithField("stdin", s).Debug("TrafPol cleaned up nft")
	}
}

// cleanupFilterRules cleans up the filter rules after a failed shutdown
func cleanupFilterRules() {
	runCleanupNft("delete table inet oc-daemon-filter")
}
