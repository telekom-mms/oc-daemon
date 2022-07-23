package vpncscript

import (
	"net"
	"strconv"
	"strings"

	"github.com/T-Systems-MMS/oc-daemon/internal/vpnconfig"
	log "github.com/sirupsen/logrus"
)

// createConfigGeneral creates the general configuration in config from env
func createConfigGeneral(env *env, config *vpnconfig.Config) {
	// set gateway address
	if env.vpnGateway != "" {
		gateway := net.ParseIP(env.vpnGateway)
		config.Gateway = gateway
	}

	// set PID
	if env.vpnPID != "" {
		pid, err := strconv.Atoi(env.vpnPID)
		if err != nil {
			log.WithError(err).Fatal("VPNCScript could not convert PID")
		}
		config.PID = pid
	}

	// set timeout
	if env.idleTimeout != "" {
		timeout, err := strconv.Atoi(env.idleTimeout)
		if err != nil {
			log.WithError(err).Fatal("VPNCScript could not convert timeout")
		}
		config.Timeout = timeout
	}
}

// createConfigDevice creates the device configuration in config from env
func createConfigDevice(env *env, config *vpnconfig.Config) {
	// set device name
	if env.tunDev != "" {
		config.Device.Name = env.tunDev
	}

	// set device mtu
	if env.internalIP4MTU != "" {
		mtu, err := strconv.Atoi(env.internalIP4MTU)
		if err != nil {
			log.WithError(err).Fatal("VPNCScript could not convert MTU")
		}
		config.Device.MTU = mtu
	}
}

// createConfigIPv4 creates the IPv4 configuration in config from env
func createConfigIPv4(env *env, config *vpnconfig.Config) {
	// set ip
	if env.internalIP4Address != "" {
		ip := net.ParseIP(env.internalIP4Address)
		config.IPv4.Address = ip
	}

	// set netmask
	if env.internalIP4NetmaskLen != "" {
		maskLen, err := strconv.Atoi(env.internalIP4NetmaskLen)
		if err != nil {
			log.WithError(err).
				Fatal("VPNCScript could not convert IPv4 netmask length")
		}
		mask := net.CIDRMask(maskLen, 32)
		config.IPv4.Netmask = mask
	}
	// TODO: parse dotted decimal representation in internalIP4Netmask?
}

// createConfigIPv6 creates the IPv6 configuration in config from env
func createConfigIPv6(env *env, config *vpnconfig.Config) {
	// set ip and netmask
	// internalIP6Netmask should contain IP in CIDR representation
	if env.internalIP6Netmask == "" {
		// no ipv6 configuration
		return
	}
	ip, ipnet, err := net.ParseCIDR(env.internalIP6Netmask)
	if err != nil {
		log.WithError(err).
			Fatal("VPNCScript could not parse IPv6 netmask")
	}
	config.IPv6.Address = ip
	config.IPv6.Netmask = ipnet.Mask
}

// createConfigDNS creates the DNS configuration in config from env
func createConfigDNS(env *env, config *vpnconfig.Config) {
	// set default domain
	if env.ciscoDefDomain != "" {
		config.DNS.DefaultDomain = env.ciscoDefDomain
	}

	// set ipv4 and ipv6 servers
	parse := func(list string) []net.IP {
		ips := []net.IP{}
		for _, d := range strings.Split(list, " ") {
			ip := net.ParseIP(d)
			if ip == nil {
				log.WithField("ip", d).
					Fatal("VPNCScript could not parse DNS server IP address")
			}
			ips = append(ips, ip)
		}
		return ips
	}
	if env.internalIP4DNS != "" {
		config.DNS.ServersIPv4 = parse(env.internalIP4DNS)
	}
	if env.internalIP6DNS != "" {
		config.DNS.ServersIPv6 = parse(env.internalIP6DNS)
	}
}

// createConfigSplit creates the split routing configuration in config from env
func createConfigSplit(env *env, config *vpnconfig.Config) {
	// set ipv4 and ipv6 excludes
	parse := func(list []string) []*net.IPNet {
		ipnets := []*net.IPNet{}
		for _, e := range list {
			_, ipnet, err := net.ParseCIDR(e)
			if err != nil {
				log.WithError(err).
					Fatal("VPNCScript could not parse exclude IP address")
			}
			ipnets = append(ipnets, ipnet)
		}
		return ipnets
	}
	if len(env.ciscoSplitExc) != 0 {
		config.Split.ExcludeIPv4 = parse(env.ciscoSplitExc)
	}
	if len(env.ciscoIPv6SplitExc) != 0 {
		config.Split.ExcludeIPv6 = parse(env.ciscoIPv6SplitExc)
	}

	// set dns excludes
	config.Split.ExcludeDNS = env.dnsSplitExc

	// set exclude virtual subnets only IPv5 flag
	config.Split.ExcludeVirtualSubnetsOnlyIPv4 =
		env.bypassVirtualSubnetsOnlyV4
}

// createConfigFlags creates the flags configuration in config from env
func createConfigFlags(env *env, config *vpnconfig.Config) {
	config.Flags.DisableAlwaysOnVPN = env.disableAlwaysOnVPN
}

// createConfig creates a VPN configuration from env
func createConfig(env *env) *vpnconfig.Config {
	config := vpnconfig.New()

	// only use settings in env in connect case
	if env.reason != "connect" {
		return config
	}

	// set general configuration
	createConfigGeneral(env, config)

	// set device configuration
	createConfigDevice(env, config)

	// set ipv4 configuration
	createConfigIPv4(env, config)

	// set ipv6 configuration
	createConfigIPv6(env, config)

	// set DNS configuration
	createConfigDNS(env, config)

	// set split routing configuration
	createConfigSplit(env, config)

	// set flags configuration
	createConfigFlags(env, config)

	return config
}

// createConfigUpdate creates a VPN configuration update from env
func createConfigUpdate(env *env) *vpnconfig.ConfigUpdate {
	update := vpnconfig.NewUpdate()
	update.Reason = env.reason
	update.Token = env.token
	if env.reason == "connect" {
		update.Config = createConfig(env)
	}
	return update
}
