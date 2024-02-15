package vpncscript

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/telekom-mms/oc-daemon/internal/daemon"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// createConfigGeneral creates the general configuration in config from env
func createConfigGeneral(env *env, config *vpnconfig.Config) error {
	// set gateway address
	if env.vpnGateway != "" {
		gateway := net.ParseIP(env.vpnGateway)
		config.Gateway = gateway
	}

	// set PID
	if env.vpnPID != "" {
		pid, err := strconv.Atoi(env.vpnPID)
		if err != nil {
			return fmt.Errorf("could not convert PID: %w", err)
		}
		config.PID = pid
	}

	// set timeout
	if env.idleTimeout != "" {
		timeout, err := strconv.Atoi(env.idleTimeout)
		if err != nil {
			return fmt.Errorf("could not convert timeout: %w", err)
		}
		config.Timeout = timeout
	}

	return nil
}

// createConfigDevice creates the device configuration in config from env
func createConfigDevice(env *env, config *vpnconfig.Config) error {
	// set device name
	if env.tunDev != "" {
		config.Device.Name = env.tunDev
	}

	// set device mtu
	if env.internalIP4MTU != "" {
		mtu, err := strconv.Atoi(env.internalIP4MTU)
		if err != nil {
			return fmt.Errorf("could not convert MTU: %w", err)
		}
		config.Device.MTU = mtu
	}

	return nil
}

// createConfigIPv4 creates the IPv4 configuration in config from env
func createConfigIPv4(env *env, config *vpnconfig.Config) error {
	// set ip
	if env.internalIP4Address != "" {
		ip := net.ParseIP(env.internalIP4Address)
		config.IPv4.Address = ip
	}

	// set netmask
	if env.internalIP4NetmaskLen != "" {
		maskLen, err := strconv.Atoi(env.internalIP4NetmaskLen)
		if err != nil {
			return fmt.Errorf("could not convert IPv4 netmask length: %w", err)
		}
		mask := net.CIDRMask(maskLen, 32)
		config.IPv4.Netmask = mask
	}
	// TODO: parse dotted decimal representation in internalIP4Netmask?

	return nil
}

// createConfigIPv6 creates the IPv6 configuration in config from env
func createConfigIPv6(env *env, config *vpnconfig.Config) error {
	// set ip and netmask
	// internalIP6Netmask should contain IP in CIDR representation
	if env.internalIP6Netmask != "" {
		ip, ipnet, err := net.ParseCIDR(env.internalIP6Netmask)
		if err != nil {
			return fmt.Errorf("could not parse IPv6 netmask: %w", err)
		}
		config.IPv6.Address = ip
		config.IPv6.Netmask = ipnet.Mask
	}

	return nil
}

// createConfigDNS creates the DNS configuration in config from env
func createConfigDNS(env *env, config *vpnconfig.Config) error {
	// set default domain
	if env.ciscoDefDomain != "" {
		config.DNS.DefaultDomain = env.ciscoDefDomain
	}

	// set ipv4 and ipv6 servers
	parse := func(list string) ([]net.IP, error) {
		ips := []net.IP{}
		for _, d := range strings.Split(list, " ") {
			ip := net.ParseIP(d)
			if ip == nil {
				return nil, fmt.Errorf("could not parse DNS server IP address %s", d)
			}
			ips = append(ips, ip)
		}
		return ips, nil
	}
	if env.internalIP4DNS != "" {
		ips, err := parse(env.internalIP4DNS)
		if err != nil {
			return err
		}
		config.DNS.ServersIPv4 = ips
	}
	if env.internalIP6DNS != "" {
		ips, err := parse(env.internalIP6DNS)
		if err != nil {
			return err
		}
		config.DNS.ServersIPv6 = ips
	}

	return nil
}

// createConfigSplit creates the split routing configuration in config from env
func createConfigSplit(env *env, config *vpnconfig.Config) error {
	// set ipv4 and ipv6 excludes
	parse := func(list []string) ([]*net.IPNet, error) {
		ipnets := []*net.IPNet{}
		for _, e := range list {
			_, ipnet, err := net.ParseCIDR(e)
			if err != nil {
				return nil, fmt.Errorf("could not parse exclude IP address: %w", err)
			}
			ipnets = append(ipnets, ipnet)
		}
		return ipnets, nil
	}
	if len(env.ciscoSplitExc) != 0 {
		ipnets, err := parse(env.ciscoSplitExc)
		if err != nil {
			return err
		}
		config.Split.ExcludeIPv4 = ipnets
	}
	if len(env.ciscoIPv6SplitExc) != 0 {
		ipnets, err := parse(env.ciscoIPv6SplitExc)
		if err != nil {
			return err
		}
		config.Split.ExcludeIPv6 = ipnets
	}

	// set dns excludes
	config.Split.ExcludeDNS = env.dnsSplitExc

	// set exclude virtual subnets only IPv5 flag
	config.Split.ExcludeVirtualSubnetsOnlyIPv4 =
		env.bypassVirtualSubnetsOnlyV4

	return nil
}

// createConfigFlags creates the flags configuration in config from env
func createConfigFlags(env *env, config *vpnconfig.Config) {
	config.Flags.DisableAlwaysOnVPN = env.disableAlwaysOnVPN
}

// createConfig creates a VPN configuration from env
func createConfig(env *env) (*vpnconfig.Config, error) {
	config := vpnconfig.New()

	// set general configuration
	if err := createConfigGeneral(env, config); err != nil {
		return nil, err
	}

	// set device configuration
	if err := createConfigDevice(env, config); err != nil {
		return nil, err
	}

	// set ipv4 configuration
	if err := createConfigIPv4(env, config); err != nil {
		return nil, err
	}

	// set ipv6 configuration
	if err := createConfigIPv6(env, config); err != nil {
		return nil, err
	}

	// set DNS configuration
	if err := createConfigDNS(env, config); err != nil {
		return nil, err
	}

	// set split routing configuration
	if err := createConfigSplit(env, config); err != nil {
		return nil, err
	}

	// set flags configuration
	createConfigFlags(env, config)

	return config, nil
}

// createConfigUpdate creates a VPN configuration update from env
func createConfigUpdate(env *env) (*daemon.VPNConfigUpdate, error) {
	update := daemon.NewVPNConfigUpdate()
	update.Reason = env.reason
	update.Token = env.token
	if env.reason == "connect" {
		c, err := createConfig(env)
		if err != nil {
			return nil, err
		}
		update.Config = c
	}
	return update, nil
}
