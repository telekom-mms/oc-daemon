package trafpol

import (
	"context"
	"errors"
	"net/netip"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// setFilterRules sets the filter rules.
func setFilterRules(ctx context.Context, config *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("TrafPolSetFilterRules", config)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get set filter rules commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("TrafPol could not run set filter rules command")
		}
	}
}

// unsetFilterRules unsets the filter rules.
func unsetFilterRules(ctx context.Context, config *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("TrafPolUnsetFilterRules", config)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get unset filter rules commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("TrafPol could not run unset filter rules command")
		}
	}
}

// setAllowedDevices sets devices as allowed devices.
func setAllowedDevices(ctx context.Context, conf *daemoncfg.Config, devices []string) {
	data := &struct {
		daemoncfg.Config
		Devices []string
	}{
		Config:  *conf,
		Devices: devices,
	}
	cmds, err := cmdtmpl.GetCmds("TrafPolSetAllowedDevices", data)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get set allowed devices commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"devices": devices,
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("TrafPol could not run set allowed devices command")
		}
	}
}

// setAllowedIPs set the allowed hosts.
func setAllowedIPs(ctx context.Context, conf *daemoncfg.Config, ips []netip.Prefix) {
	// set allowed hosts
	data := &struct {
		daemoncfg.Config
		AllowedIPs []netip.Prefix
	}{
		Config:     *conf,
		AllowedIPs: ips,
	}
	cmds, err := cmdtmpl.GetCmds("TrafPolSetAllowedHosts", data)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get set allowed hosts commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"hosts":   ips,
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("TrafPol could not run set allowed hosts command")
		}
	}
}

// setAllowedPorts sets ports (for a captive portal) as the allowed ports.
func setAllowedPorts(ctx context.Context, conf *daemoncfg.Config, ports []uint16) {
	data := &struct {
		daemoncfg.Config
		Ports []uint16
	}{
		Config: *conf,
		Ports:  ports,
	}
	cmds, err := cmdtmpl.GetCmds("TrafPolSetAllowedPorts", data)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get set allowed ports commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"ports":   ports,
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
				"error":   err,
			}).Error("TrafPol could not run set allowed ports command")
		}
	}
}

// cleanupFilterRules cleans up the filter rules after a failed shutdown.
func cleanupFilterRules(ctx context.Context, conf *daemoncfg.Config) {
	cmds, err := cmdtmpl.GetCmds("TrafPolCleanup", conf)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get cleanup commands")
	}
	for _, c := range cmds {
		if _, _, err := c.Run(ctx); err == nil {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
			}).Warn("TrafPol cleaned up configuration")
		}
	}
}
