package splitrt

import (
	"context"
	"net/netip"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/config"
)

// addExclude adds exclude address to netfilter.
// TODO: remove and only use setExcludes?
func addExclude(ctx context.Context, conf *config.Config, address netip.Prefix) {
	log.WithField("address", address).Debug("SplitRouting adding exclude to netfilter")

	data := &struct {
		config.Config
		Address netip.Prefix
	}{
		Config:  *conf,
		Address: address,
	}
	cmds, err := cmdtmpl.GetCmds("SplitRoutingAddExclude", data)
	if err != nil {
		log.WithError(err).Error("SplitRouting could not get add exclude commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil {
			log.WithFields(log.Fields{
				"address": address,
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("SplitRouting could not run add exclude command")
		}
	}
}

// setExcludes resets the excludes to addresses in netfilter.
func setExcludes(ctx context.Context, conf *config.Config, addresses []netip.Prefix) {
	data := &struct {
		config.Config
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
