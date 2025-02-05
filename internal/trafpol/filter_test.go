package trafpol

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// TestFilterFunctionsErrors tests filter functions, errors.
func TestFilterFunctionsErrors(_ *testing.T) {
	oldRunCmd := cmdtmpl.RunCmd
	cmdtmpl.RunCmd = func(context.Context, string, string, ...string) ([]byte, []byte, error) {
		return nil, nil, errors.New("test error")
	}
	defer func() { cmdtmpl.RunCmd = oldRunCmd }()

	ctx := context.Background()

	// filter rules
	conf := daemoncfg.NewConfig()
	conf.SplitRouting.FirewallMark = "123"
	setFilterRules(ctx, conf)
	unsetFilterRules(ctx, conf)

	// allowed devices
	setAllowedDevices(ctx, conf, []string{"eth0"})
	setAllowedDevices(ctx, conf, []string{"eth0", "eth1"})
	setAllowedDevices(ctx, conf, []string{"eth0"})
	setAllowedDevices(ctx, conf, []string{})

	// allowed IPs
	setAllowedIPs(ctx, conf, []netip.Prefix{
		netip.MustParsePrefix("192.168.1.1/32"),
		netip.MustParsePrefix("2000::1/128"),
	})

	// portal ports
	setAllowedPorts(ctx, conf, []uint16{80, 443})
	setAllowedPorts(ctx, conf, []uint16{})
}
