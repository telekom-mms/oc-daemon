package splitrt

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// addDefaultRouteIPv4 adds default routing for IPv4.
func addDefaultRouteIPv4(ctx context.Context, device, rtTable, rulePrio1, fwMark, rulePrio2 string) {
	// set default route
	if err := execs.RunIP4Route(ctx, "add", "0.0.0.0/0", "dev", device,
		"table", rtTable); err != nil {
		log.WithError(err).Error("SplitRouting error setting ipv4 default route")
	}

	// set routing rules
	if err := execs.RunIP4Rule(ctx, "add", "iif", device, "table", "main",
		"pref", rulePrio1); err != nil {
		log.WithError(err).Error("SplitRouting error setting ipv4 routing rule 1")
	}
	if err := execs.RunIP4Rule(ctx, "add", "not", "fwmark", fwMark,
		"table", rtTable, "pref", rulePrio2); err != nil {
		log.WithError(err).Error("SplitRouting error setting ipv4 routing rule 2")
	}

	// set src_valid_mark with sysctl
	if err := execs.RunSysctl(ctx, "-q",
		"net.ipv4.conf.all.src_valid_mark=1"); err != nil {
		log.WithError(err).Error("SplitRouting error setting ipv4 sysctl")
	}
}

// addDefaultRouteIPv6 adds default routing for IPv6.
func addDefaultRouteIPv6(ctx context.Context, device, rtTable, rulePrio1, fwMark, rulePrio2 string) {
	// set default route
	if err := execs.RunIP6Route(ctx, "add", "::/0", "dev", device, "table",
		rtTable); err != nil {
		log.WithError(err).Error("SplitRouting error setting ipv6 default route")
	}

	// set routing rules
	if err := execs.RunIP6Rule(ctx, "add", "iif", device, "table", "main",
		"pref", rulePrio1); err != nil {
		log.WithError(err).Error("SplitRouting error setting ipv6 routing rule 1")
	}
	if err := execs.RunIP6Rule(ctx, "add", "not", "fwmark", fwMark,
		"table", rtTable, "pref", rulePrio2); err != nil {
		log.WithError(err).Error("SplitRouting error setting ipv6 routing rule 2")
	}
}

// deleteDefaultRouteIPv4 removes default routing for IPv4.
func deleteDefaultRouteIPv4(ctx context.Context, device, rtTable string) {
	// delete routing rules
	if err := execs.RunIP4Rule(ctx, "delete", "table", rtTable); err != nil {
		log.WithError(err).Error("SplitRouting error deleting ipv4 routing rule 2")
	}
	if err := execs.RunIP4Rule(ctx, "delete", "iif", device, "table",
		"main"); err != nil {
		log.WithError(err).Error("SplitRouting error deleting ipv4 routing rule 1")
	}
}

// deleteDefaultRouteIPv6 removes default routing for IPv6.
func deleteDefaultRouteIPv6(ctx context.Context, device, rtTable string) {
	// delete routing rules
	if err := execs.RunIP6Rule(ctx, "delete", "table", rtTable); err != nil {
		log.WithError(err).Error("SplitRouting error deleting ipv6 routing rule 2")
	}
	if err := execs.RunIP6Rule(ctx, "delete", "iif", device, "table",
		"main"); err != nil {
		log.WithError(err).Error("SplitRouting error deleting ipv6 routing rule 1")
	}
}

// cleanupRouting cleans up the routing configuration after a failed shutdown.
func cleanupRouting(ctx context.Context, rtTable, rulePrio1, rulePrio2 string) {
	// delete ipv4 routing rules
	if err := execs.RunIP4Rule(ctx, "delete", "pref", rulePrio1); err == nil {
		log.Debug("SplitRouting cleaned up ipv4 routing rule 1")
	}
	if err := execs.RunIP4Rule(ctx, "delete", "pref", rulePrio2); err == nil {
		log.Debug("SplitRouting cleaned up ipv4 routing rule 2")
	}

	// delete ipv6 routing rules
	if err := execs.RunIP6Rule(ctx, "delete", "pref", rulePrio1); err == nil {
		log.Debug("SplitRouting cleaned up ipv6 routing rule 1")
	}
	if err := execs.RunIP6Rule(ctx, "delete", "pref", rulePrio2); err == nil {
		log.Debug("SplitRouting cleaned up ipv6 routing rule 2")
	}

	// flush ipv4 routing table
	if err := execs.RunIP4Route(ctx, "flush", "table", rtTable); err == nil {
		log.Debug("SplitRouting cleaned up ipv4 routing table")
	}

	// flush ipv6 routing table
	if err := execs.RunIP6Route(ctx, "flush", "table", rtTable); err == nil {
		log.Debug("SplitRouting cleaned up ipv6 routing table")
	}
}
