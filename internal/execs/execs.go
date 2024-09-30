// Package execs contains external executables.
package execs

import (
	"bytes"
	"context"
	"os/exec"

	log "github.com/sirupsen/logrus"
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

type Command struct {
	Name  string
	Args  []string
	Stdin string

	// error handling?
	LogError bool   // log everything on error? with name, args, stdin/out/err?
	OnError  string // continue, stop? if list of commands
}

type CommandList struct {
	Name     string
	Commands []Command
}

type CommandLists map[string]CommandList

// Run runs command list identified by name
func (c CommandLists) Run(ctx context.Context, name string) {
	for _, cmd := range c[name].Commands {
		stdout, stderr, err := RunCmd(ctx, cmd.Name, cmd.Stdin, cmd.Args...)
		if err != nil && cmd.LogError {
			log.WithError(err).WithFields(log.Fields{
				"list":    c[name].Name,
				"command": cmd.Name,
				"args":    cmd.Args,
				"stdin":   cmd.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("Error executing command in command list")
		}
		if err != nil && cmd.OnError == "stop" {
			return
		}
	}
}
