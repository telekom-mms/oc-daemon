package trafpol

import (
	"context"
	"errors"
	"net/netip"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
)

// setFilterRules sets the filter rules.
func setFilterRules(ctx context.Context, config *Config) {
	data := config // TODO: change?
	cmds, err := cmdtmpl.GetCmds("TrafPolSetFilterRules", data)
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
			}).Error("TrafPol could not run set filter rules command")
		}
	}
}

// unsetFilterRules unsets the filter rules.
func unsetFilterRules(ctx context.Context) {
	data := "" // TODO: change?
	cmds, err := cmdtmpl.GetCmds("TrafPolUnsetFilterRules", data)
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
			}).Error("TrafPol could not run unset filter rules command")
		}
	}
}

// addAllowedDevice adds device to the allowed devices.
func addAllowedDevice(ctx context.Context, device string) {
	data := device // TODO: change?
	cmds, err := cmdtmpl.GetCmds("TrafPolAddAllowedDevice", data)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get add allowed device commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"device":  device,
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("TrafPol could not run add allowed device command")
		}
	}
}

// removeAllowedDevice removes device from the allowed devices.
func removeAllowedDevice(ctx context.Context, device string) {
	data := device // TODO: change?
	cmds, err := cmdtmpl.GetCmds("TrafPolRemoveAllowedDevice", data)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get remove allowed device commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"device":  device,
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("TrafPol could not run remove allowed device command")
		}
	}
}

// setAllowedIPs set the allowed hosts.
func setAllowedIPs(ctx context.Context, ips []netip.Prefix) {
	// we perform all nft commands separately here and not as one atomic
	// operation to avoid issues where the whole update fails because nft
	// runs into "file exists" errors even though we remove duplicates from
	// ips before calling this function and we flush the existing entries

	// flush allowed hosts
	data := "" // TODO: change?
	cmds, err := cmdtmpl.GetCmds("TrafPolFlushAllowedHosts", data)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get flush allowed hosts commands")
	}
	for _, c := range cmds {
		if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"command": c.Cmd,
				"args":    c.Args,
				"stdin":   c.Stdin,
				"stdout":  string(stdout),
				"stderr":  string(stderr),
			}).Error("TrafPol could not run flush allowed hosts command")
		}
	}

	// add allowed hosts
	for _, ip := range ips {
		data := ip // TODO: change?
		cmds, err := cmdtmpl.GetCmds("TrafPolAddAllowedHost", data)
		if err != nil {
			log.WithError(err).Error("TrafPol could not get add allowed host commands")
		}
		for _, c := range cmds {
			if stdout, stderr, err := c.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.WithFields(log.Fields{
					"host":    ip,
					"command": c.Cmd,
					"args":    c.Args,
					"stdin":   c.Stdin,
					"stdout":  string(stdout),
					"stderr":  string(stderr),
				}).Error("TrafPol could not run add allowed host command")
			}
		}
	}
}

// portsToString returns ports as string.
func portsToString(ports []uint16) string {
	s := []string{}
	for _, port := range ports {
		s = append(s, strconv.FormatUint(uint64(port), 10))
	}
	return strings.Join(s, ", ")
}

// addPortalPorts adds ports for a captive portal to the allowed ports.
func addPortalPorts(ctx context.Context, ports []uint16) {
	data := portsToString(ports) // TODO: change?
	cmds, err := cmdtmpl.GetCmds("TrafPolAddPortalPorts", data)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get add portal ports commands")
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
			}).Error("TrafPol could not run add portal ports command")
		}
	}
}

// removePortalPorts removes ports for a captive portal from the allowed ports.
func removePortalPorts(ctx context.Context, ports []uint16) {
	data := portsToString(ports) // TODO: change?
	cmds, err := cmdtmpl.GetCmds("TrafPolRemovePortalPorts", data)
	if err != nil {
		log.WithError(err).Error("TrafPol could not get remove portal ports commands")
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
			}).Error("TrafPol could not run remove portal ports command")
		}
	}
}

// cleanupFilterRules cleans up the filter rules after a failed shutdown.
func cleanupFilterRules(ctx context.Context) {
	data := "" // TODO: change?
	cmds, err := cmdtmpl.GetCmds("TrafPolCleanup", data)
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
