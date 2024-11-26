package trafpol

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/config"
	"github.com/telekom-mms/oc-daemon/internal/execs"
)

// TestFilterFunctionsErrors tests filter functions, errors.
func TestFilterFunctionsErrors(_ *testing.T) {
	oldRunCmd := execs.RunCmd
	execs.RunCmd = func(context.Context, string, string, ...string) ([]byte, []byte, error) {
		return nil, nil, errors.New("test error")
	}
	defer func() { execs.RunCmd = oldRunCmd }()

	ctx := context.Background()

	// filter rules
	conf := config.NewConfig()
	conf.TrafficPolicing.FirewallMark = "123"
	setFilterRules(ctx, conf)
	unsetFilterRules(ctx, conf)

	// allowed devices
	addAllowedDevice(ctx, conf, "eth0")
	removeAllowedDevice(ctx, conf, "eth0")

	// allowed IPs
	setAllowedIPs(ctx, conf, []netip.Prefix{
		netip.MustParsePrefix("192.168.1.1/32"),
		netip.MustParsePrefix("2000::1/128"),
	})

	// portal ports
	addPortalPorts(ctx, conf, []uint16{80, 443})
	removePortalPorts(ctx, conf, []uint16{80, 443})
}
