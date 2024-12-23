// Package execs contains external executables.
package execs

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// executables.
var (
	ip         = daemoncfg.ExecutablesIP
	sysctl     = daemoncfg.ExecutablesSysctl
	nft        = daemoncfg.ExecutablesNft
	resolvectl = daemoncfg.ExecutablesResolvectl
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
func SetExecutables(config *daemoncfg.Executables) {
	ip = config.IP
	sysctl = config.Sysctl
	nft = config.Nft
	resolvectl = config.Resolvectl
}
