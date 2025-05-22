// Package cmdtmpl contains command lists for external commands with templates.
package cmdtmpl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
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

	template *template.Template
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

// DefaultTemplate is the default template for the command lists.
const DefaultTemplate = `

{{- /*********************************************************************/ -}}
{{- /*********************************************************************/ -}}
{{- /*********************************************************************/ -}}
{{- /***                                                               ***/ -}}
{{- /***                  Traffic Policing Templates                   ***/ -}}
{{- /***                                                               ***/ -}}
{{- /*********************************************************************/ -}}
{{- /*********************************************************************/ -}}
{{- /*********************************************************************/ -}}

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

                # accept traffic to allowed ports
                udp dport @allowports counter accept
                tcp dport @allowports counter accept

                # accept DNS traffic
                udp dport 53 counter accept
                tcp dport 53 counter accept

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
{{end}}

{{- /*********************************************************************/ -}}
{{- /*********************************************************************/ -}}
{{- /*********************************************************************/ -}}
{{- /***                                                               ***/ -}}
{{- /***                      VPNSetup Templates                       ***/ -}}
{{- /***                                                               ***/ -}}
{{- /*********************************************************************/ -}}
{{- /*********************************************************************/ -}}
{{- /*********************************************************************/ -}}

{{- define "SplitRoutingRules"}}
table inet oc-daemon-routing {
	# set for ipv4 excludes
	set excludes4 {
		type ipv4_addr
		flags interval
		{{if .VPNConfig.Gateway.Is4}}
		elements = { {{.VPNConfig.Gateway}}/32 }
		{{end}}
	}

	# set for ipv6 excludes
	set excludes6 {
		type ipv6_addr
		flags interval
		{{if .VPNConfig.Gateway.Is6}}
		elements = { {{.VPNConfig.Gateway}}/128 }
		{{end}}
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

// LoadedTemplates are the templates loaded from file.
var LoadedTemplates string

// defaultTemplate is the parsed default template for the command lists.
var defaultTemplate = template.Must(template.New("Template").Parse(DefaultTemplate))

// Command list names.
const (
	TrafPolSetFilterRules    = "TrafPolSetFilterRules"
	TrafPolUnsetFilterRules  = "TrafPolUnsetFilterRules"
	TrafPolSetAllowedDevices = "TrafPolSetAllowedDevices"
	TrafPolSetAllowedHosts   = "TrafPolSetAllowedHosts"
	TrafPolSetAllowedPorts   = "TrafPolSetAllowedPorts"
	TrafPolCleanup           = "TrafPolCleanup"

	VPNSetupSetup       = "VPNSetupSetup"
	VPNSetupTeardown    = "VPNSetupTeardown"
	VPNSetupSetExcludes = "VPNSetupSetExcludes"
	VPNSetupSetDNS      = "VPNSetupSetDNS"
	VPNSetupGetDNS      = "VPNSetupGetDNS"
	VPNSetupCleanup     = "VPNSetupCleanup"
)

// CommandLists contains all command lists.
var CommandLists = map[string]*CommandList{

	//////////////////////
	// Traffic Policing //
	//////////////////////

	// Set Filter Rules
	TrafPolSetFilterRules: {
		Name: TrafPolSetFilterRules,
		Commands: []*Command{
			{Line: "{{.Executables.Nft}} -f -", Stdin: `{{template "TrafPolRules" .}}`},
		},
		template: defaultTemplate,
	},

	// Unset Filter Rules
	TrafPolUnsetFilterRules: {
		Name: TrafPolUnsetFilterRules,
		Commands: []*Command{
			{Line: "{{.Executables.Nft}} -f - delete table inet oc-daemon-filter"},
		},
		template: defaultTemplate,
	},

	// Set Allowed Devices
	TrafPolSetAllowedDevices: {
		Name: TrafPolSetAllowedDevices,
		Commands: []*Command{
			{Line: "{{.Executables.Nft}} -f -",
				Stdin: `flush set inet oc-daemon-filter allowdevs
{{range .Devices -}}
add element inet oc-daemon-filter allowdevs { {{.}} }
{{end}}`},
		},
		template: defaultTemplate,
	},

	// Set Allowed Hosts
	TrafPolSetAllowedHosts: {
		Name: TrafPolSetAllowedHosts,
		Commands: []*Command{
			{Line: "{{.Executables.Nft}} -f -",
				Stdin: `flush set inet oc-daemon-filter allowhosts4
flush set inet oc-daemon-filter allowhosts6
{{range .AllowedIPs -}}
{{if .Addr.Is4 -}}
add element inet oc-daemon-filter allowhosts4 { {{.}} }
{{else -}}
add element inet oc-daemon-filter allowhosts6 { {{.}} }
{{end -}}
{{end}}`,
			},
		},
		template: defaultTemplate,
	},

	// Set Allowed Ports
	TrafPolSetAllowedPorts: {
		Name: TrafPolSetAllowedPorts,
		Commands: []*Command{
			{Line: "{{.Executables.Nft}} -f -",
				Stdin: `flush set inet oc-daemon-filter allowports
{{range .Ports -}}
add element inet oc-daemon-filter allowports { {{.}} }
{{end}}`},
		},
		template: defaultTemplate,
	},

	// Cleanup
	TrafPolCleanup: {
		Name: TrafPolCleanup,
		Commands: []*Command{
			{Line: "{{.Executables.Nft}} -f - delete table inet oc-daemon-filter"},
		},
		template: defaultTemplate,
	},

	//////////////
	// VPNSetup //
	//////////////

	// Setup VPN
	VPNSetupSetup: {
		Name: VPNSetupSetup,
		Commands: []*Command{
			// Device setup:
			// - set mtu on device
			// - set device up
			// - set ipv4 and ipv6 addresses on device
			{Line: "{{.Executables.IP}} link set {{.VPNConfig.Device.Name}} mtu {{.VPNConfig.Device.MTU}}"},
			{Line: "{{.Executables.IP}} link set {{.VPNConfig.Device.Name}} up"},
			{Line: "{{if .VPNConfig.IPv4.IsValid}}{{.Executables.IP}} address add {{.VPNConfig.IPv4}} dev {{.VPNConfig.Device.Name}}{{end}}"},
			{Line: "{{if .VPNConfig.IPv6.IsValid}}{{.Executables.IP}} address add {{.VPNConfig.IPv6}} dev {{.VPNConfig.Device.Name}}{{end}}"},
			// Routing setup
			{Line: "{{.Executables.Nft}} -f -", Stdin: `{{template "SplitRoutingRules" .}}`},
			{Line: "{{.Executables.IP}} -4 route add 0.0.0.0/0 dev {{.VPNConfig.Device.Name}} table {{.SplitRouting.RoutingTable}}"},
			{Line: "{{.Executables.IP}} -4 rule add iif {{.VPNConfig.Device.Name}} table main pref {{.SplitRouting.RulePriority1}}"},
			{Line: "{{.Executables.IP}} -4 rule add not fwmark {{.SplitRouting.FirewallMark}} table {{.SplitRouting.RoutingTable}} pref {{.SplitRouting.RulePriority2}}"},
			{Line: "{{.Executables.Sysctl}} -q net.ipv4.conf.all.src_valid_mark=1"},
			{Line: "{{.Executables.IP}} -6 route add ::/0 dev {{.VPNConfig.Device.Name}} table {{.SplitRouting.RoutingTable}}"},
			{Line: "{{.Executables.IP}} -6 rule add iif {{.VPNConfig.Device.Name}} table main pref {{.SplitRouting.RulePriority1}}"},
			{Line: "{{.Executables.IP}} -6 rule add not fwmark {{.SplitRouting.FirewallMark}} table {{.SplitRouting.RoutingTable}} pref {{.SplitRouting.RulePriority2}}"},
			// DNS setup:
			// - set DNS Proxy
			// - set Domains
			// - set default DNS route
			// - flush caches
			// - reset server features
			{Line: "{{.Executables.Resolvectl}} dns {{.VPNConfig.Device.Name}} {{.DNSProxy.Address}}"},
			{Line: "{{.Executables.Resolvectl}} domain {{.VPNConfig.Device.Name}} {{.VPNConfig.DNS.DefaultDomain}} ~."},
			{Line: "{{.Executables.Resolvectl}} default-route {{.VPNConfig.Device.Name}} yes"},
			{Line: "{{.Executables.Resolvectl}} flush-caches"},
			{Line: "{{.Executables.Resolvectl}} reset-server-features"},
		},
		template: defaultTemplate,
	},

	// Teardown VPN
	VPNSetupTeardown: {
		Name: VPNSetupTeardown,
		Commands: []*Command{
			// Device teardown
			{Line: "{{.Executables.IP}} link set {{.VPNConfig.Device.Name}} down"},
			// Routing teardown
			{Line: "{{.Executables.IP}} -4 rule delete table {{.SplitRouting.RoutingTable}}"},
			{Line: "{{.Executables.IP}} -4 rule delete iif {{.VPNConfig.Device.Name}} table main"},
			{Line: "{{.Executables.IP}} -6 rule delete table {{.SplitRouting.RoutingTable}}"},
			{Line: "{{.Executables.IP}} -6 rule delete iif {{.VPNConfig.Device.Name}} table main"},
			{Line: "{{.Executables.Nft}} -f - delete table inet oc-daemon-routing"},
			// DNS teardown
			{Line: "{{.Executables.Resolvectl}} revert {{.VPNConfig.Device.Name}}"},
			{Line: "{{.Executables.Resolvectl}} flush-caches"},
			{Line: "{{.Executables.Resolvectl}} reset-server-features"},
		},
		template: defaultTemplate,
	},

	// Set Excludes
	VPNSetupSetExcludes: {
		Name: VPNSetupSetExcludes,
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
		template: defaultTemplate,
	},

	// Set DNS settings
	VPNSetupSetDNS: {
		Name: VPNSetupSetDNS,
		Commands: []*Command{
			{Line: "{{.Executables.Resolvectl}} dns {{.VPNConfig.Device.Name}} {{.DNSProxy.Address}}"},
			{Line: "{{.Executables.Resolvectl}} domain {{.VPNConfig.Device.Name}} {{.VPNConfig.DNS.DefaultDomain}} ~."},
			{Line: "{{.Executables.Resolvectl}} default-route {{.VPNConfig.Device}} yes"},
		},
		template: defaultTemplate,
	},

	// Get DNS settings
	VPNSetupGetDNS: {
		Name: VPNSetupGetDNS,
		Commands: []*Command{
			{Line: "{{.Executables.Resolvectl}} status {{.VPNConfig.Device.Name}} --no-pager"},
		},
		template: defaultTemplate,
	},

	// Cleanup
	VPNSetupCleanup: {
		Name: VPNSetupCleanup,
		Commands: []*Command{
			// DNS cleanup
			{Line: "{{.Executables.Resolvectl}} revert {{.OpenConnect.VPNDevice}}"},
			// Device cleanup
			{Line: "{{.Executables.IP}} link delete {{.OpenConnect.VPNDevice}}"},
			// Routing cleanup
			{Line: "{{.Executables.IP}} -4 rule delete pref {{.SplitRouting.RulePriority1}}"},
			{Line: "{{.Executables.IP}} -4 rule delete pref {{.SplitRouting.RulePriority2}}"},
			{Line: "{{.Executables.IP}} -6 rule delete pref {{.SplitRouting.RulePriority1}}"},
			{Line: "{{.Executables.IP}} -6 rule delete pref {{.SplitRouting.RulePriority2}}"},
			{Line: "{{.Executables.IP}} -4 route flush table {{.SplitRouting.RoutingTable}}"},
			{Line: "{{.Executables.IP}} -6 route flush table {{.SplitRouting.RoutingTable}}"},
			{Line: "{{.Executables.Nft}} -f - delete table inet oc-daemon-routing"},
		},
		template: defaultTemplate,
	},
}

// LoadTemplates loads the templates from file.
func LoadTemplates(file string) error {
	// read file contents
	f, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// parse file contents
	s := string(f)
	t, err := template.New("Template").Parse(s)
	if err == nil {
		// save loaded templates
		LoadedTemplates = s
		defaultTemplate = t
	}
	return err
}

// LoadCommandLists loads the command lists from file.
func LoadCommandLists(file string) error {
	// read file contents
	f, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// parse entries in file
	lists := []*CommandList{}
	if err := json.Unmarshal(f, &lists); err != nil {
		return err
	}

	// check entries in file
	for _, cl := range lists {
		// check valid names
		switch cl.Name {
		case TrafPolSetFilterRules:
		case TrafPolUnsetFilterRules:
		case TrafPolSetAllowedDevices:
		case TrafPolSetAllowedHosts:
		case TrafPolSetAllowedPorts:
		case TrafPolCleanup:

		case VPNSetupSetup:
		case VPNSetupTeardown:
		case VPNSetupSetExcludes:
		case VPNSetupSetDNS:
		case VPNSetupGetDNS:
		case VPNSetupCleanup:

		default:
			// invalid name
			return fmt.Errorf("invalid command list name %s", cl.Name)
		}
	}

	// entries in file valid, update command lists
	for _, cl := range lists {
		cl.template = defaultTemplate
		CommandLists[cl.Name] = cl
	}

	return nil
}

// getCommandList returns the command list identified by name.
func getCommandList(name string) *CommandList {
	return CommandLists[name]
}

// Cmd is a command ready to run.
type Cmd struct {
	Cmd   string
	Args  []string
	Stdin string
}

// RunCmd runs the cmd with args and sets stdin to s, returns stdout and stderr.
var RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) (stdout, stderr []byte, err error) {
	c := exec.CommandContext(ctx, cmd, arg...)
	if s != "" {
		c.Stdin = bytes.NewBufferString(s)
	}
	var outbuf, errbuf bytes.Buffer
	c.Stdout = &outbuf
	c.Stderr = &errbuf
	err = c.Run()
	stdout = outbuf.Bytes()
	stderr = errbuf.Bytes()
	return
}

// Run runs the command.
func (c *Cmd) Run(ctx context.Context) (stdout, stderr []byte, err error) {
	return RunCmd(ctx, c.Cmd, c.Stdin, c.Args...)
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
