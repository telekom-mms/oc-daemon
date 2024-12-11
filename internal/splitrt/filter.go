package splitrt

import (
	"context"
	"net/netip"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// setExcludes resets the excludes to addresses in netfilter.
func setExcludes(ctx context.Context, conf *daemoncfg.Config, addresses []netip.Prefix) {
	data := &struct {
		daemoncfg.Config
		Addresses []netip.Prefix
	}{
		Config:    *conf,
		Addresses: addresses,
	}
	cmds, err := cmdtmpl.GetCmds("SplitRoutingSetExcludes", data)
	if err != nil {
		log.WithError(err).Error("SplitRouting could not get set excludes commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil {
			log.WithFields(log.Fields{
				"addresses": addresses,
				"command":   c.Cmd,
				"args":      c.Args,
				"stdin":     c.Stdin,
				"stdout":    string(stdout),
				"stderr":    string(stderr),
				"error":     err,
			}).Error("SplitRouting could not run set excludes command")
		}
	}
}
