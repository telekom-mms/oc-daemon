// Package cmdtmpl contains command lists for external commands with templates.
package cmdtmpl

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// Command consists of a command line to be executed and an optional Stdin to
// be passed to the command on execution.
type Command struct {
	Line  string
	Stdin string
}

// CommandList is a list of Commands.
type CommandList struct {
	Name     string
	Commands []*Command

	defaultTemplate string
	template        *template.Template
}

// executeTemplate executes the template on data and returns the resulting
// output as string.
func (cl *CommandList) executeTemplate(tmpl string, data any) (string, error) {
	t, err := cl.template.Clone()
	if err != nil {
		return "", err
	}
	t, err = t.Parse(tmpl)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, data); err != nil {
		return "", err
	}

	s := buf.String()
	return s, nil
}

// SplitRoutingDefaultTemplate is the default template for Split Routing.
const SplitRoutingDefaultTemplate = `
{{- define "SplitRoutingRules"}}
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
		{{if .VPNConfig.IPv4.IsValid}}
		iifname != {{.VPNConfig.Device.Name}} ip daddr {{.VPNConfig.IPv4}} fib saddr type != local counter drop
		{{end}}
		{{if .VPNConfig.IPv6.IsValid}}
		iifname != {{.VPNConfig.Device.Name}} ip6 daddr {{.VPNConfig.IPv6}} fib saddr type != local counter drop
		{{end}}
	}

	chain splitrouting {
		# restore mark from conntracking
		ct mark != 0 meta mark set ct mark counter
		meta mark != 0 counter accept

		# mark packets in exclude sets
		ip daddr @excludes4 counter meta mark set {{.SplitRouting.FirewallMark}}
		ip6 daddr @excludes6 counter meta mark set {{.SplitRouting.FirewallMark}}

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
		meta mark {{.SplitRouting.FirewallMark}} ct mark set meta mark counter
	}

	chain postrouting {
		type nat hook postrouting priority srcnat; policy accept;

		# masquerare tunnel/exclude traffic to make sure the source IP
		# matches the outgoing interface
		ct mark {{.SplitRouting.FirewallMark}} counter masquerade
	}

	chain rejectipversion {
		# used to reject unsupported ip version on the tunnel device

		# make sure exclude traffic is not filtered
		ct mark {{.SplitRouting.FirewallMark}} counter accept

		# use tcp reset and icmp admin prohibited
		meta l4proto tcp counter reject with tcp reset
		counter reject with icmpx admin-prohibited
	}

	chain rejectforward {
		type filter hook forward priority filter; policy accept;

		# reject unsupported ip versions when forwarding packets,
		# add matching jump rule to rejectipversion if necessary
		{{if .VPNConfig.IPv4.IsValid}}
		meta oifname {{.VPNConfig.Device.Name}} meta nfproto ipv6 counter jump rejectipversion
		{{end}}
		{{if .VPNConfig.IPv6.IsValid}}
		meta oifname {{.VPNConfig.Device.Name}} meta nfproto ipv4 counter jump rejectipversion
		{{end}}
	}

	chain rejectoutput {
		type filter hook output priority filter; policy accept;

		# reject unsupported ip versions when sending packets,
		# add matching jump rule to rejectipversion if necessary
		{{if .VPNConfig.IPv4.IsValid}}
		meta oifname {{.VPNConfig.Device.Name}} meta nfproto ipv6 counter jump rejectipversion
		{{end}}
		{{if .VPNConfig.IPv6.IsValid}}
		meta oifname {{.VPNConfig.Device.Name}} meta nfproto ipv4 counter jump rejectipversion
		{{end}}
	}
}
{{end -}}
`

// getCommandListSplitRouting returns the command list identified by name for SplitRouting.
func getCommandListSplitRouting(name string) *CommandList {
	var cl *CommandList
	switch name {
	case "SplitRoutingSetupRouting":
		// Setup Routing
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f -", Stdin: `{{template "SplitRoutingRules" .}}`},
				{Line: "{{.Executables.IP}} -4 route add 0.0.0.0/0 dev {{.VPNConfig.Device.Name}} table {{.SplitRouting.RoutingTable}}"},
				{Line: "{{.Executables.IP}} -4 rule add iif {{.VPNConfig.Device.Name}} table main pref {{.SplitRouting.RulePriority1}}"},
				{Line: "{{.Executables.IP}} -4 rule add not fwmark {{.SplitRouting.FirewallMark}} table {{.SplitRouting.RoutingTable}} pref {{.SplitRouting.RulePriority2}}"},
				{Line: "{{.Executables.Sysctl}} -q net.ipv4.conf.all.src_valid_mark=1"},
				{Line: "{{.Executables.IP}} -6 route add ::/0 dev {{.VPNConfig.Device.Name}} table {{.SplitRouting.RoutingTable}}"},
				{Line: "{{.Executables.IP}} -6 rule add iif {{.VPNConfig.Device.Name}} table main pref {{.SplitRouting.RulePriority1}}"},
				{Line: "{{.Executables.IP}} -6 rule add not fwmark {{.SplitRouting.FirewallMark}} table {{.SplitRouting.RoutingTable}} pref {{.SplitRouting.RulePriority2}}"},
			},
			defaultTemplate: SplitRoutingDefaultTemplate,
		}
	case "SplitRoutingTeardownRouting":
		// Teardown Routing
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.IP}} -4 rule delete table {{.SplitRouting.RoutingTable}}"},
				{Line: "{{.Executables.IP}} -4 rule delete iif {{.VPNConfig.Device.Name}} table main"},
				{Line: "{{.Executables.IP}} -6 rule delete table {{.SplitRouting.RoutingTable}}"},
				{Line: "{{.Executables.IP}} -6 rule delete iif {{.VPNConfig.Device.Name}} table main"},
				{Line: "{{.Executables.Nft}} -f - delete table inet oc-daemon-routing"},
			},
			defaultTemplate: SplitRoutingDefaultTemplate,
		}
	case "SplitRoutingSetExcludes":
		// Set Excludes
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				// flush existing entries
				// add entries
				{Line: "{{.Executables.Nft}} -f -",
					Stdin: `flush set inet oc-daemon-routing excludes4
flush set inet oc-daemon-routing excludes6
{{range .Addresses -}}
{{if .Addr.Is6 -}}
add element inet oc-daemon-routing excludes6 { {{.}} }
{{else -}}
add element inet oc-daemon-routing excludes4 { {{.}} }
{{end -}}
{{end}}`},
			},
			defaultTemplate: SplitRoutingDefaultTemplate,
		}
	case "SplitRoutingCleanup":
		// Cleanup
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.IP}} -4 rule delete pref {{.SplitRouting.RulePriority1}}"},
				{Line: "{{.Executables.IP}} -4 rule delete pref {{.SplitRouting.RulePriority2}}"},
				{Line: "{{.Executables.IP}} -6 rule delete pref {{.SplitRouting.RulePriority1}}"},
				{Line: "{{.Executables.IP}} -6 rule delete pref {{.SplitRouting.RulePriority2}}"},
				{Line: "{{.Executables.IP}} -4 route flush table {{.SplitRouting.RoutingTable}}"},
				{Line: "{{.Executables.IP}} -6 route flush table {{.SplitRouting.RoutingTable}}"},
				{Line: "{{.Executables.Nft}} -f - delete table inet oc-daemon-routing"},
			},
			defaultTemplate: SplitRoutingDefaultTemplate,
		}
	default:
		return nil

	}

	cl.template = template.Must(template.New("Template").Parse(cl.defaultTemplate))
	return cl
}

// TrafPolDefaultTemplate is the default template for Traffic Policing.
const TrafPolDefaultTemplate = `
{{- define "TrafPolRules"}}
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
                iifname @allowdevs ct mark {{.SplitRouting.FirewallMark}} counter accept

                # accept traffic on allowed devices
                iifname @allowdevs oifname @allowdevs counter accept
        }
}
{{end}}`

// getCommandListTrafPol returns the command list identified by name for Traffic Policing.
func getCommandListTrafPol(name string) *CommandList {
	var cl *CommandList
	switch name {
	case "TrafPolSetFilterRules":
		// Set Filter Rules
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f -", Stdin: `{{template "TrafPolRules" .}}`},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	case "TrafPolUnsetFilterRules":
		// Unset Filter Rules
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f - delete table inet oc-daemon-filter"},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	case "TrafPolAddAllowedDevice":
		// Add Allowed Device
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f - add element inet oc-daemon-filter allowdevs { {{.Device}} }"},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	case "TrafPolRemoveAllowedDevice":
		// Remove Allowed Device
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f - delete element inet oc-daemon-filter allowdevs { {{.Device}} }"},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	case "TrafPolFlushAllowedHosts":
		// Flush Allowed Hosts
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f - flush set inet oc-daemon-filter allowhosts4"},
				{Line: "{{.Executables.Nft}} -f - flush set inet oc-daemon-filter allowhosts6"},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	case "TrafPolAddAllowedHost":
		// Add Allowed Host
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f -",
					Stdin: `
				{{if .AllowedIP.Addr.Is4}}
				add element inet oc-daemon-filter allowhosts4 { {{.AllowedIP}} }
				{{else}}
				add element inet oc-daemon-filter allowhosts6 { {{.AllowedIP}} }
				{{end}}
				`,
				},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	case "TrafPolAddPortalPorts":
		// Remove Portal Ports
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f - add element inet oc-daemon-filter allowports { {{range $i, $port := .TrafficPolicing.PortalPorts}}{{if $i}}, {{end}}{{$port}}{{end}} }"},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	case "TrafPolRemovePortalPorts":
		// Remove Portal Ports
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f - delete element inet oc-daemon-filter allowports { {{range $i, $port := .TrafficPolicing.PortalPorts}}{{if $i}}, {{end}}{{$port}}{{end}} }"},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	case "TrafPolCleanup":
		// Cleanup
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Nft}} -f - delete table inet oc-daemon-filter"},
			},
			defaultTemplate: TrafPolDefaultTemplate,
		}
	default:
		return nil
	}

	cl.template = template.Must(template.New("Template").Parse(cl.defaultTemplate))
	return cl
}

// getCommandListVPNSetup returns the command list identified by name for VPNSetup.
func getCommandListVPNSetup(name string) *CommandList {
	var cl *CommandList
	switch name {
	case "VPNSetupSetupVPNDevice":
		// Setup VPN Device
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				// set mtu on device
				{Line: "{{.Executables.IP}} link set {{.VPNConfig.Device.Name}} mtu {{.VPNConfig.Device.MTU}}"},
				// set device up
				{Line: "{{.Executables.IP}} link set {{.VPNConfig.Device.Name}} up"},
				// set ipv4 and ipv6 addresses on device
				{Line: "{{if .VPNConfig.IPv4.IsValid}}{{.Executables.IP}} address add {{.VPNConfig.IPv4}} dev {{.VPNConfig.Device.Name}}{{end}}"},
				{Line: "{{if .VPNConfig.IPv6.IsValid}}{{.Executables.IP}} address add {{.VPNConfig.IPv6}} dev {{.VPNConfig.Device.Name}}{{end}}"},
			},
			defaultTemplate: "",
		}
	case "VPNSetupTeardownVPNDevice":
		// Teardown VPN Device
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.IP}} link set {{.VPNConfig.Device.Name}} down"},
			},
			defaultTemplate: "",
		}
	case "VPNSetupSetupDNSServer":
		// Setup DNS server
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Resolvectl}} dns {{.VPNConfig.Device.Name}} {{.DNSProxy.Address}}"},
			},
			defaultTemplate: "",
		}
	case "VPNSetupSetupDNSDomains":
		// Setup DNS domains
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Resolvectl}} domain {{.VPNConfig.Device.Name}} {{.VPNConfig.DNS.DefaultDomain}} ~."},
			},
			defaultTemplate: "",
		}
	case "VPNSetupSetupDNSDefaultRoute":
		// Setup DNS Default Route
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Resolvectl}} default-route {{.VPNConfig.Device}} yes"},
			},
			defaultTemplate: "",
		}
	case "VPNSetupSetupDNS":
		// Setup DNS
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Resolvectl}} dns {{.VPNConfig.Device.Name}} {{.DNSProxy.Address}}"},
				{Line: "{{.Executables.Resolvectl}} domain {{.VPNConfig.Device.Name}} {{.VPNConfig.DNS.DefaultDomain}} ~."},
				{Line: "{{.Executables.Resolvectl}} default-route {{.VPNConfig.Device.Name}} yes"},
				{Line: "{{.Executables.Resolvectl}} flush-caches"},
				{Line: "{{.Executables.Resolvectl}} reset-server-features"},
			},
			defaultTemplate: "",
		}
	case "VPNSetupTeardownDNS":
		// Teardown DNS
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Resolvectl}} revert {{.VPNConfig.Device.Name}}"},
				{Line: "{{.Executables.Resolvectl}} flush-caches"},
				{Line: "{{.Executables.Resolvectl}} reset-server-features"},
			},
			defaultTemplate: "",
		}
	case "VPNSetupEnsureDNS":
		// Ensure DNS
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Resolvectl}} status {{.VPNConfig.Device.Name}} --no-pager"},
			},
			defaultTemplate: "",
		}
	case "VPNSetupCleanup":
		// Cleanup
		cl = &CommandList{
			Name: name,
			Commands: []*Command{
				{Line: "{{.Executables.Resolvectl}} revert {{.OpenConnect.VPNDevice}}"},
				{Line: "{{.Executables.IP}} link delete {{.OpenConnect.VPNDevice}}"},
			},
			defaultTemplate: "",
		}
	default:
		return nil
	}

	cl.template = template.Must(template.New("Template").Parse(cl.defaultTemplate))
	return cl
}

// getCommandList returns the command list identified by name.
func getCommandList(name string) *CommandList {
	if strings.HasPrefix(name, "SplitRouting") {
		return getCommandListSplitRouting(name)
	}
	if strings.HasPrefix(name, "TrafPol") {
		return getCommandListTrafPol(name)
	}
	if strings.HasPrefix(name, "VPNSetup") {
		return getCommandListVPNSetup(name)
	}
	return nil
}

// Cmd is a command ready to run.
type Cmd struct {
	Cmd   string
	Args  []string
	Stdin string
}

// Run runs the command.
func (c *Cmd) Run(ctx context.Context) (stdout, stderr []byte, err error) {
	return execs.RunCmd(ctx, c.Cmd, c.Stdin, c.Args...)
}

// GetCmds returns a list of Cmds ready to run.
func GetCmds(name string, data any) ([]*Cmd, error) {
	cl := getCommandList(name)
	if cl == nil {
		return nil, fmt.Errorf("could not find command list %s", name)
	}
	var commands []*Cmd
	for _, c := range cl.Commands {
		// execute template for command line
		line, err := cl.executeTemplate(c.Line, data)
		if err != nil {
			return nil, fmt.Errorf("could not execute template for command line: %w", err)
		}

		// execute template for stdin
		stdin, err := cl.executeTemplate(c.Stdin, data)
		if err != nil {
			return nil, fmt.Errorf("could not execute template for stdin: %w", err)
		}

		// extract command from command line
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		command := fields[0]

		// extract arguments from command line
		args := []string{}
		if len(fields) > 1 {
			args = fields[1:]
		}
		commands = append(commands, &Cmd{
			Cmd:   command,
			Args:  args,
			Stdin: stdin,
		})
	}
	return commands, nil
}
