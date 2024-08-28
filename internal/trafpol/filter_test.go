package trafpol

import (
	"context"
	"errors"
	"net/netip"
	"testing"

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
	setFilterRules(ctx, "123")
	unsetFilterRules(ctx)

	// allowed devices
	addAllowedDevice(ctx, "eth0")
	removeAllowedDevice(ctx, "eth0")

	// allowed IPs
	setAllowedIPs(ctx, []netip.Prefix{
		netip.MustParsePrefix("192.168.1.1/32"),
		netip.MustParsePrefix("2000::1/128"),
	})

	// portal ports
	addPortalPorts(ctx, []uint16{80, 443})
	removePortalPorts(ctx, []uint16{80, 443})
}
