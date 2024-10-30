// Package execs contains external executables.
package execs

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"text/template"
)

// executables.
var (
	ip         = IP
	sysctl     = Sysctl
	nft        = Nft
	resolvectl = Resolvectl
)

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

// RunIP runs the "ip" command with args.
func RunIP(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	return RunCmd(ctx, ip, "", arg...)
}

// RunIPLink runs the "ip link" command with args.
func RunIPLink(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	a := append([]string{"link"}, arg...)
	return RunIP(ctx, a...)
}

// RunIPAddress runs the "ip address" command with args.
func RunIPAddress(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	a := append([]string{"address"}, arg...)
	return RunIP(ctx, a...)
}

// RunIP4Route runs the "ip -4 route" command with args.
func RunIP4Route(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	a := append([]string{"-4", "route"}, arg...)
	return RunIP(ctx, a...)
}

// RunIP6Route runs the "ip -6 route" command with args.
func RunIP6Route(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	a := append([]string{"-6", "route"}, arg...)
	return RunIP(ctx, a...)
}

// RunIP4Rule runs the "ip -4 rule" command with args.
func RunIP4Rule(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	a := append([]string{"-4", "rule"}, arg...)
	return RunIP(ctx, a...)
}

// RunIP6Rule runs the "ip -6 rule" command with args.
func RunIP6Rule(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	a := append([]string{"-6", "rule"}, arg...)
	return RunIP(ctx, a...)
}

// RunSysctl runs the "sysctl" command with args.
func RunSysctl(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	return RunCmd(ctx, sysctl, "", arg...)
}

// RunNft runs the "nft -f -" command and sets stdin to s.
func RunNft(ctx context.Context, s string) (stdout, stderr []byte, err error) {
	return RunCmd(ctx, nft, s, "-f", "-")
}

// RunResolvectl runs the "resolvectl" command with args.
func RunResolvectl(ctx context.Context, arg ...string) (stdout, stderr []byte, err error) {
	return RunCmd(ctx, resolvectl, "", arg...)
}

// SetExecutables configures all executables from config.
func SetExecutables(config *Config) {
	ip = config.IP
	sysctl = config.Sysctl
	nft = config.Nft
	resolvectl = config.Resolvectl
}

// Command consists of a command line to be executed and an optional Stdin to
// be passed to the command on execution.
type Command struct {
	Line  string
	Stdin string
}

// executeTemplate executes the template on data and returns the resulting
// output as string.
func (c *Command) executeTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("CommandTemplate").Parse(tmpl)
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
func (c *Command) Run(ctx context.Context, data any) (stdout, stderr []byte, err error) {
	// execute template for command line
	line, err := c.executeTemplate(c.Line, data)
	if err != nil {
		return nil, nil, err
	}

	// execute template for stdin
	stdin, err := c.executeTemplate(c.Stdin, data)
	if err != nil {
		return nil, nil, err
	}

	// extract command from command line
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil, nil, nil
	}
	cmd := fields[0]

	// extract arguments from command line
	args := []string{}
	if len(fields) > 1 {
		args = fields[1:]
	}

	// run command
	return RunCmd(ctx, cmd, stdin, args...)
}

type CommandList struct {
	Name     string
	Commands []Command
}

func (cl *CommandList) Run(ctx context.Context) {
}

type CommandLists map[string]CommandList

//// Run runs command list identified by name
//func (c CommandLists) Run(ctx context.Context, name string) {
//	for _, cmd := range c[name].Commands {
//		stdout, stderr, err := RunCmd(ctx, cmd.Line, cmd.Stdin, cmd.Args...)
//		if err != nil && cmd.LogError {
//			log.WithError(err).WithFields(log.Fields{
//				"list":    c[name].Name,
//				"command": cmd.Line,
//				"args":    cmd.Args,
//				"stdin":   cmd.Stdin,
//				"stdout":  string(stdout),
//				"stderr":  string(stderr),
//			}).Error("Error executing command in command list")
//		}
//		if err != nil && cmd.OnError == "stop" {
//			return
//		}
//	}
//}
