package cmdtmpl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// CommandTemplates are command templates.
type CommandTemplates struct {
	templates *template.Template
}

func NewCommandTemplates(templates string) *CommandTemplates {
	t := template.Must(template.New("Templates").Parse(templates))
	return &CommandTemplates{
		templates: t,
	}
}

// LoadTemplates loads the command templates.
func (ct *CommandTemplates) Load() error {
	// TODO: try to load templates from filesystem and update ct.templates
	return nil
}

// Command consists of a command line to be executed and an optional Stdin to
// be passed to the command on execution.
type Command struct {
	Line  string
	Stdin string

	cmd   string
	args  []string
	stdin string
}

// executeTemplate executes the template on data and returns the resulting
// output as string.
func (ct *CommandTemplates) executeTemplate(tmpl string, data any) (string, error) {
	t, err := ct.templates.Clone()
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

	line := buf.String()
	return line, nil
}

// Run runs the command and returns its output.
func (ct *CommandTemplates) RunCommand(ctx context.Context, cmd *Command, data any) (stdout, stderr []byte, err error) {
	// execute template for command line
	line, err := ct.executeTemplate(cmd.Line, data)
	if err != nil {
		return nil, nil, err
	}

	// execute template for stdin
	stdin, err := ct.executeTemplate(cmd.Stdin, data)
	if err != nil {
		return nil, nil, err
	}

	// extract command from command line
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, nil, nil
	}
	command := fields[0]

	// extract arguments from command line
	args := []string{}
	if len(fields) > 1 {
		args = fields[1:]
	}

	// run command
	return execs.RunCmd(ctx, command, stdin, args...)
}

type CommandList struct {
	Name     string
	Commands []*Command

	defaultTemplate string
	template        *template.Template
}

var commandLists map[string]*CommandList

const SplitRoutingDefaultTemplate = `
{{- define "RoutingRules"}}
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
		{{if .IPv4Address}}
		iifname != {{.Device}} ip daddr {{.IPv4Address}} fib saddr type != local counter drop
		{{end}}
		{{if .IPv6Address}}
		iifname != {{.Device}} ip6 daddr {{.IPv6Address}} fib saddr type != local counter drop
		{{end}}
	}

	chain splitrouting {
		# restore mark from conntracking
		ct mark != 0 meta mark set ct mark counter
		meta mark != 0 counter accept

		# mark packets in exclude sets
		ip daddr @excludes4 counter meta mark set {{.FWMark}}
		ip6 daddr @excludes6 counter meta mark set {{.FWMark}}

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
		meta mark {{.FWMark}} ct mark set meta mark counter
	}

	chain postrouting {
		type nat hook postrouting priority srcnat; policy accept;

		# masquerare tunnel/exclude traffic to make sure the source IP
		# matches the outgoing interface
		ct mark {{.FWMark}} counter masquerade
	}

	chain rejectipversion {
		# used to reject unsupported ip version on the tunnel device

		# make sure exclude traffic is not filtered
		ct mark {{.FWMark}} counter accept

		# use tcp reset and icmp admin prohibited
		meta l4proto tcp counter reject with tcp reset
		counter reject with icmpx admin-prohibited
	}

	chain rejectforward {
		type filter hook forward priority filter; policy accept;

		# reject unsupported ip versions when forwarding packets,
		# add matching jump rule to rejectipversion if necessary
		{{if .IPv4Address}}
		meta oifname {{.Device}} meta nfproto ipv6 counter jump rejectipversion
		{{end}}
		{{if .IPv6Address}}
		meta oifname {{.Device}} meta nfproto ipv4 counter jump rejectipversion
		{{end}}
	}

	chain rejectoutput {
		type filter hook output priority filter; policy accept;

		# reject unsupported ip versions when sending packets,
		# add matching jump rule to rejectipversion if necessary
		{{if .IPv4Address}}
		meta oifname {{.Device}} meta nfproto ipv6 counter jump rejectipversion
		{{end}}
		{{if .IPv6Address}}
		meta oifname {{.Device}} meta nfproto ipv4 counter jump rejectipversion
		{{end}}
	}
}
{{end -}}
`

func initCommandListsSplitRouting() {
	// TODO: change this?
	t := template.Must(template.New("Template").Parse(SplitRoutingDefaultTemplate))

	// Setup Routing
	setupRouting := &CommandList{
		Name: "SplitRoutingSetupRouting",
		Commands: []*Command{
			{Line: "nft -f -", Stdin: `{{template "RoutingRules" .}}`},
			{Line: "ip -4 route add 0.0.0.0/0 dev {{.Device}} table {{.RTTable}}"},
			{Line: "ip -4 rule add iif {{.Device}} table main pref {{.RulePrio1}}"},
			{Line: "ip -4 rule add not fwmark {{.FWMark}} table {{.RTTable}} pref {{.RulePrio2}}"},
			{Line: "sysctl -q net.ipv4.conf.all.src_valid_mark=1"},
			{Line: "ip -6 route add ::/0 dev {{.Device}} table {{.RTTable}}"},
			{Line: "ip -6 rule add iif {{.Device}} table main pref {{.RulePrio1}}"},
			{Line: "ip -6 rule add not fwmark {{.FWMark}} table {{.RTTable}} pref {{.RulePrio2}}"},
		},
		defaultTemplate: SplitRoutingDefaultTemplate,
		template:        t,
	}
	commandLists[setupRouting.Name] = setupRouting

	// Teardown Routing
	teardownRouting := &CommandList{
		Name: "SplitRoutingTeardownRouting",
		Commands: []*Command{
			{Line: "ip -4 rule delete table {{.RTTable}}"},
			{Line: "ip -4 rule delete iif {{.Device}} table main"},
			{Line: "ip -6 rule delete table {{.RTTable}}"},
			{Line: "ip -6 rule delete iif {{.Device}} table main"},
			{Line: "nft -f - delete table inet oc-daemon-routing"},
		},
		defaultTemplate: SplitRoutingDefaultTemplate,
		template:        t,
	}
	commandLists[teardownRouting.Name] = teardownRouting

	// Add Exclude
	addExclude := &CommandList{
		Name: "SplitRoutingAddExclude",
		Commands: []*Command{
			{Line: "nft -f -",
				Stdin: `
				{{- if .Addr.Is6 -}}
				add element inet oc-daemon-routing excludes6 { {{.}} }
				{{- else -}}
				add element inet oc-daemon-routing excludes4 { {{.}} }
				{{- end}}`},
		},
		defaultTemplate: SplitRoutingDefaultTemplate,
		template:        t,
	}
	commandLists[addExclude.Name] = addExclude

	// Set Excludes
	setExcludes := &CommandList{
		Name: "SplitRoutingSetExcludes",
		Commands: []*Command{
			// flush existing entries
			// add entries
			{Line: "nft -f -",
				Stdin: `flush set inet oc-daemon-routing excludes4
flush set inet oc-daemon-routing excludes6
{{range . -}}
{{if .Addr.Is6 -}}
add element inet oc-daemon-routing excludes6 { {{.}} }
{{else -}}
add element inet oc-daemon-routing excludes4 { {{.}} }
{{end -}}
{{end}}`},
		},
		defaultTemplate: SplitRoutingDefaultTemplate,
		template:        t,
	}
	commandLists[setExcludes.Name] = setExcludes

	// Cleanup
	cleanup := &CommandList{
		Name: "SplitRoutingCleanup",
		Commands: []*Command{
			{Line: "ip -4 rule delete pref {{.RulePrio1}}"},
			{Line: "ip -4 rule delete pref {{.RulePrio2}}"},
			{Line: "ip -6 rule delete pref {{.RulePrio1}}"},
			{Line: "ip -6 rule delete pref {{.RulePrio2}}"},
			{Line: "ip -4 route flush table {{.RTTable}}"},
			{Line: "ip -6 route flush table {{.RTTable}}"},
			{Line: "nft -f - delete table inet oc-daemon-routing"},
		},
		defaultTemplate: SplitRoutingDefaultTemplate,
		template:        t,
	}
	commandLists[cleanup.Name] = cleanup
}

func initCommandLists() {
	commandLists = make(map[string]*CommandList)
	initCommandListsSplitRouting()
}

// TODO: remove?
func init() {
	initCommandLists()
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

	line := buf.String()
	return line, nil
}

func (cl *CommandList) RunCommand(ctx context.Context, cmd *Command, data any) (stdout, stderr []byte, err error) {
	// execute template for command line
	line, err := cl.executeTemplate(cmd.Line, data)
	if err != nil {
		return nil, nil, err
	}

	// execute template for stdin
	stdin, err := cl.executeTemplate(cmd.Stdin, data)
	if err != nil {
		return nil, nil, err
	}

	// extract command from command line
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, nil, nil
	}
	command := fields[0]

	// extract arguments from command line
	args := []string{}
	if len(fields) > 1 {
		args = fields[1:]
	}

	// run command
	return execs.RunCmd(ctx, command, stdin, args...)
}

func (cl *CommandList) Run(ctx context.Context, data any) (stdout, stderr []byte, err error) {
	errs := []error{}
	for _, cmd := range cl.Commands {
		sout, serr, err := cl.RunCommand(ctx, cmd, data)
		stdout = slices.Concat(stdout, sout)
		stderr = slices.Concat(stderr, serr)
		errs = append(errs, err)

		//if err != nil {
		//	log.WithError(err).WithFields(log.Fields{
		//		"command": cmd.Line,
		//		"stdin":   cmd.Stdin,
		//		"stdout":  string(stdout),
		//		"stderr":  string(stderr),
		//	}).Error("Error executing command")
		//}
	}
	err = errors.Join(errs...)
	return
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

// func GetCommands(name string, data any) ([]*Command, error) {
func GetCmds(name string, data any) ([]*Cmd, error) {
	cl, ok := commandLists[name]
	if !ok {
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

//func RunCommands(ctx context.Context, name string, data any) (stdout, stderr []byte, err error) {
//	commands, err := GetCommands(name, data)
//	if err != nil {
//		return nil, nil, err
//	}
//	return commandLists[name].Run(ctx, data)
//}
