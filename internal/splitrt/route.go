package splitrt

import (
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

const (
	// rtTable is the routing table
	rtTable = "42111"

	// rule preferences
	rulePref1 = "2111"
	rulePref2 = "2112"
)

// runCmd runs the cmd
var runCmd = func(cmd string) {
	log.WithField("command", cmd).Debug("Daemon executing command")
	c := exec.Command("bash", "-c", cmd)
	if err := c.Run(); err != nil {
		log.WithFields(log.Fields{
			"command": cmd,
			"error":   err,
		}).Error("Daemon command execution error")
	}
}

// addDefaultRouteIPv4 adds default routing for IPv4
func addDefaultRouteIPv4(device string) {
	// set default route and routing rules
	for _, r := range []string{
		fmt.Sprintf("ip -4 route add 0.0.0.0/0 dev %s table %s",
			device, rtTable),
		fmt.Sprintf("ip -4 rule add iif %s table main pref %s",
			device, rulePref1),
		fmt.Sprintf("ip -4 rule add not fwmark %s table %s pref %s",
			FWMark, rtTable, rulePref2),
	} {
		runCmd(r)
	}

	// set src_valid_mark with sysctl
	sysctl := "sysctl -q net.ipv4.conf.all.src_valid_mark=1"
	runCmd(sysctl)
}

// addDefaultRouteIPv6 adds default routing for IPv6
func addDefaultRouteIPv6(device string) {
	// set default route and routing rules
	for _, r := range []string{
		fmt.Sprintf("ip -6 route add ::/0 dev %s table %s", device,
			rtTable),
		fmt.Sprintf("ip -6 rule add iif %s table main pref %s",
			device, rulePref1),
		fmt.Sprintf("ip -6 rule add not fwmark %s table %s pref %s",
			FWMark, rtTable, rulePref2),
	} {
		runCmd(r)
	}
}

// deleteDefaultRouteIPv4 removes default routing for IPv4
func deleteDefaultRouteIPv4(device string) {
	// delete routing rules
	for _, r := range []string{
		fmt.Sprintf("ip -4 rule delete table %s", rtTable),
		fmt.Sprintf("ip -4 rule delete iif %s table main", device),
	} {
		runCmd(r)
	}
}

// deleteDefaultRouteIPv6 removes default routing for IPv6
func deleteDefaultRouteIPv6(device string) {
	// delete routing rules
	for _, r := range []string{
		fmt.Sprintf("ip -6 rule delete table %s", rtTable),
		fmt.Sprintf("ip -6 rule delete iif %s table main", device),
	} {
		runCmd(r)
	}
}

// runCleanupCmd runs cmd for cleanups
var runCleanupCmd = func(cmd string) {
	log.WithField("command", cmd).Debug("SplitRouting executing routing cleanup command")
	c := exec.Command("bash", "-c", cmd)
	if err := c.Run(); err == nil {
		// some commands might succeed anyway, so just use debug
		log.WithField("command", cmd).Debug("SplitRouting cleaned up routing")
	}
}

// cleanupRouting cleans up the routing configuration after a failed shutdown
func cleanupRouting() {
	// delete routing rules
	for _, r := range []string{
		fmt.Sprintf("ip -4 rule delete pref %s", rulePref1),
		fmt.Sprintf("ip -4 rule delete pref %s", rulePref2),
		fmt.Sprintf("ip -6 rule delete pref %s", rulePref1),
		fmt.Sprintf("ip -6 rule delete pref %s", rulePref2),
		fmt.Sprintf("ip -4 route flush table %s", rtTable),
		fmt.Sprintf("ip -6 route flush table %s", rtTable),
	} {
		runCleanupCmd(r)
	}
}
