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

// SetExecutables configures all executables from config.
func SetExecutables(config *daemoncfg.Executables) {
	ip = config.IP
	sysctl = config.Sysctl
	nft = config.Nft
	resolvectl = config.Resolvectl
}
