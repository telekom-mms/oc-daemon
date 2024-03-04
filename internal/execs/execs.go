// Package execs contains external executables.
package execs

import (
	"bytes"
	"context"
	"os/exec"
)

// executables
var (
	ip         = IP
	sysctl     = Sysctl
	nft        = Nft
	resolvectl = Resolvectl
)

// RunCmd runs the cmd with args and sets stdin to s
var RunCmd = func(ctx context.Context, cmd string, s string, arg ...string) error {
	c := exec.CommandContext(ctx, cmd, arg...)
	if s != "" {
		c.Stdin = bytes.NewBufferString(s)
	}
	return c.Run()
}

// RunCmdOutput runs the cmd with args and sets stdin to s, returns output
var RunCmdOutput = func(ctx context.Context, cmd string, s string, arg ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, cmd, arg...)
	if s != "" {
		c.Stdin = bytes.NewBufferString(s)
	}
	return c.Output()
}

// RunIP runs the "ip" command with args
func RunIP(ctx context.Context, arg ...string) error {
	return RunCmd(ctx, ip, "", arg...)
}

// RunIPLink runs the "ip link" command with args
func RunIPLink(ctx context.Context, arg ...string) error {
	a := append([]string{"link"}, arg...)
	return RunIP(ctx, a...)
}

// RunIPAddress runs the "ip address" command with args
func RunIPAddress(ctx context.Context, arg ...string) error {
	a := append([]string{"address"}, arg...)
	return RunIP(ctx, a...)
}

// RunIP4Route runs the "ip -4 route" command with args
func RunIP4Route(ctx context.Context, arg ...string) error {
	a := append([]string{"-4", "route"}, arg...)
	return RunIP(ctx, a...)
}

// RunIP6Route runs the "ip -6 route" command with args
func RunIP6Route(ctx context.Context, arg ...string) error {
	a := append([]string{"-6", "route"}, arg...)
	return RunIP(ctx, a...)
}

// RunIP4Rule runs the "ip -4 rule" command with args
func RunIP4Rule(ctx context.Context, arg ...string) error {
	a := append([]string{"-4", "rule"}, arg...)
	return RunIP(ctx, a...)
}

// RunIP6Rule runs the "ip -6 rule" command with args
func RunIP6Rule(ctx context.Context, arg ...string) error {
	a := append([]string{"-6", "rule"}, arg...)
	return RunIP(ctx, a...)
}

// RunSysctl runs the "sysctl" command with args
func RunSysctl(ctx context.Context, arg ...string) error {
	return RunCmd(ctx, sysctl, "", arg...)
}

// RunNft runs the "nft -f -" command and sets stdin to s
func RunNft(ctx context.Context, s string) error {
	return RunCmd(ctx, nft, s, "-f", "-")
}

// RunResolvectl runs the "resolvectl" command with args
func RunResolvectl(ctx context.Context, arg ...string) error {
	return RunCmd(ctx, resolvectl, "", arg...)
}

// RunResolvectlOutput runs the "resolvectl" command with args, returns output
func RunResolvectlOutput(ctx context.Context, arg ...string) ([]byte, error) {
	return RunCmdOutput(ctx, resolvectl, "", arg...)
}

// SetExecutables configures all executables from config
func SetExecutables(config *Config) {
	ip = config.IP
	sysctl = config.Sysctl
	nft = config.Nft
	resolvectl = config.Resolvectl
}
