package vpncscript

import (
	"net"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// TestCreateConfigSplit tests createConfigSplit
func TestCreateConfigSplit(t *testing.T) {
	// create test environment
	env := &env{
		ciscoSplitInc:              []string{},
		ciscoSplitExc:              []string{"172.16.0.0/16"},
		ciscoIPv6SplitInc:          []string{},
		ciscoIPv6SplitExc:          []string{},
		dnsSplitExc:                []string{"some.example.com", "other.example.com", "www.example.com"},
		bypassVirtualSubnetsOnlyV4: true,
	}

	// create expected values
	ipv4 := []*net.IPNet{
		{
			IP:   net.IPv4(172, 16, 0, 0),
			Mask: net.IPv4Mask(255, 255, 0, 0),
		},
	}
	dns := []string{"some.example.com", "other.example.com", "www.example.com"}
	vnet := true
	got := vpnconfig.New()

	// update config from test environment
	createConfigSplit(env, got)

	// check results
	if len(got.Split.ExcludeIPv4) != len(ipv4) {
		t.Errorf("got %v, want %v", got.Split.ExcludeIPv4, ipv4)
	} else {
		for i, exclude := range got.Split.ExcludeIPv4 {
			if exclude.String() != ipv4[i].String() {
				t.Errorf("got %v, want %v", exclude, ipv4[i])
			}
		}
	}
	if !reflect.DeepEqual(got.Split.ExcludeDNS, dns) {
		t.Errorf("got %v, want %v", got.DNS, dns)
	}
	if got.Split.ExcludeVirtualSubnetsOnlyIPv4 != vnet {
		t.Errorf("got %t, want %t", got.Split.ExcludeVirtualSubnetsOnlyIPv4, vnet)
	}
}

// TestCreateConfigUpdate tests createConfigUpdate
func TestCreateConfigUpdate(t *testing.T) {
	// create test environment
	env := &env{
		reason:                     "connect",
		vpnGateway:                 "10.1.1.1",
		vpnPID:                     "12345",
		tunDev:                     "tun0",
		idleTimeout:                "300",
		internalIP4Address:         "192.168.1.123",
		internalIP4MTU:             "1300",
		internalIP4Netmask:         "255.255.255.0",
		internalIP4NetmaskLen:      "24",
		internalIP4NetAddr:         "192.168.1.0",
		internalIP4DNS:             "192.168.1.1",
		internalIP4NBNS:            "192.168.1.1",
		ciscoDefDomain:             "example.com",
		ciscoBanner:                "some banner",
		ciscoSplitInc:              []string{}, // splits are tested in TestCreateConfigSplit
		ciscoSplitExc:              []string{},
		ciscoIPv6SplitInc:          []string{},
		ciscoIPv6SplitExc:          []string{},
		dnsSplitExc:                []string{"some.example.com", "other.example.com", "www.example.com"},
		bypassVirtualSubnetsOnlyV4: true,
		disableAlwaysOnVPN:         true,
		token:                      "some token",
	}

	// create expected values based on test environment
	reason := "connect"
	token := "some token"
	config := &vpnconfig.Config{
		Gateway: net.IPv4(10, 1, 1, 1),
		PID:     12345,
		Timeout: 300,
		Device: vpnconfig.Device{
			Name: "tun0",
			MTU:  1300,
		},
		IPv4: vpnconfig.Address{
			Address: net.IPv4(192, 168, 1, 123),
			Netmask: net.IPv4Mask(255, 255, 255, 0),
		},
		DNS: vpnconfig.DNS{
			DefaultDomain: "example.com",
			ServersIPv4:   []net.IP{net.IPv4(192, 168, 1, 1)},
		},
		Split: vpnconfig.Split{
			ExcludeDNS: []string{"some.example.com", "other.example.com", "www.example.com"},

			ExcludeVirtualSubnetsOnlyIPv4: true,
		},
		Flags: vpnconfig.Flags{
			DisableAlwaysOnVPN: true,
		},
	}

	// pare environment and get update
	got := createConfigUpdate(env)

	// compare results
	if got.Reason != reason {
		t.Errorf("got %s, want %s", got.Reason, reason)
	}
	if got.Token != token {
		t.Errorf("got %s, want %s", got.Token, token)
	}
	if !reflect.DeepEqual(got.Config, config) {
		t.Errorf("got:\n%#v\nwant:\n%#v", got.Config, config)
	}
}
