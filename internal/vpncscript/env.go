package vpncscript

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// env contains openconnect parameters passed through the environment:
//
// List of parameters passed through environment
//
//   - reason                           -- why this script was called, one of:
//     --                                  pre-init connect disconnect reconnect
//     --                                  attempt-reconnect
//
//   - VPNGATEWAY                       -- VPN gateway address (always present)
//
//   - VPNPID                           -- PID of the process controlling the VPN (OpenConnect v9.0+)
//
//   - TUNDEV                           -- tunnel device (always present)
//
//   - IDLE_TIMEOUT                     -- gateway's idle timeout in seconds (OpenConnect v8.06+); unused
//
//   - INTERNAL_IP4_ADDRESS             -- address (always present)
//
//   - INTERNAL_IP4_MTU                 -- MTU (often unset)
//
//   - INTERNAL_IP4_NETMASK             -- netmask (often unset)
//
//   - INTERNAL_IP4_NETMASKLEN          -- netmask length (often unset)
//
//   - INTERNAL_IP4_NETADDR             -- address of network (only present if netmask is set)
//
//   - INTERNAL_IP4_DNS                 -- list of DNS servers
//
//   - INTERNAL_IP4_NBNS                -- list of WINS servers
//
//   - INTERNAL_IP6_ADDRESS             -- IPv6 address
//
//   - INTERNAL_IP6_NETMASK             -- IPv6 netmask
//
//   - INTERNAL_IP6_DNS                 -- IPv6 list of dns servers
//
//   - CISCO_DEF_DOMAIN                 -- default domain name
//
//   - CISCO_BANNER                     -- banner from server
//
//   - CISCO_SPLIT_DNS                  -- DNS search domain list
//
//   - CISCO_SPLIT_INC                  -- number of networks in split-network-list
//
//   - CISCO_SPLIT_INC_%d_ADDR          -- network address
//
//   - CISCO_SPLIT_INC_%d_MASK          -- subnet mask (for example: 255.255.255.0)
//
//   - CISCO_SPLIT_INC_%d_MASKLEN       -- subnet masklen (for example: 24)
//
//   - CISCO_SPLIT_INC_%d_PROTOCOL      -- protocol (often just 0); unused
//
//   - CISCO_SPLIT_INC_%d_SPORT         -- source port (often just 0); unused
//
//   - CISCO_SPLIT_INC_%d_DPORT         -- destination port (often just 0); unused
//
//   - CISCO_IPV6_SPLIT_INC             -- number of networks in IPv6 split-network-list
//
//   - CISCO_IPV6_SPLIT_INC_%d_ADDR     -- IPv6 network address
//
//   - CISCO_IPV6_SPLIT_INC_$%d_MASKLEN -- IPv6 subnet masklen
//
//     The split tunnel variables above have *_EXC* counterparts for network
//     addresses to be excluded from the VPN tunnel.
//
// Other/undocumented environment variables
//   - CISCO_CSTP_OPTIONS               -- List of all CSTP headers, includes dynamic
//     --                                  DNS-based Split Exclude and Bypass Virtual
//     --                                  Subnets Only settings in nested
//     --                                  X-CSTP-Post-Auth-XML header
type env struct {
	// general settings
	reason      string
	vpnGateway  string
	vpnPID      string
	tunDev      string
	idleTimeout string

	// IPv4 settings
	internalIP4Address    string
	internalIP4MTU        string
	internalIP4Netmask    string
	internalIP4NetmaskLen string
	internalIP4NetAddr    string
	internalIP4DNS        string
	internalIP4NBNS       string

	// IPv6 settings
	internalIP6Address string
	internalIP6Netmask string
	internalIP6DNS     string

	// cisco settings
	ciscoDefDomain    string
	ciscoBanner       string
	ciscoSplitDNS     string
	ciscoSplitInc     []string
	ciscoSplitExc     []string
	ciscoIPv6SplitInc []string
	ciscoIPv6SplitExc []string

	// other/undocumented settings
	ciscoCSTPOptions []string
	dnsSplitExc      []string

	bypassVirtualSubnetsOnlyV4 bool
	disableAlwaysOnVPN         bool

	// openconnect daemon token, socket file, verbosity
	token      string
	socketFile string
	verbose    bool
}

// parseEnvironmentSplit parses split include/exclude parameters identified by
// prefix and returns them; note: only uses ADDR and MASKLEN
// TODO: this might not work with IPv6, check and fix it
func parseEnvironmentSplit(prefix string) []string {
	splits := []string{}

	// get number of settings
	num := os.Getenv(prefix)
	if num == "" {
		return splits
	}

	// make sure it is a number
	n, err := strconv.Atoi(num)
	if err != nil {
		log.WithError(err).Error("VPNCScript could not convert number of split settings")
		return splits
	}

	// parse all addresses and masklens
	for i := 0; i < n; i++ {
		// construct name of address and masklen environment variables
		addr := fmt.Sprintf("%s_%d_ADDR", prefix, i)
		masklen := fmt.Sprintf("%s_%d_MASKLEN", prefix, i)

		// get values of address and masklen environment variables
		a := os.Getenv(addr)
		m := os.Getenv(masklen)

		// add address/masklen to split parameters
		if a == "" || m == "" {
			continue
		}
		s := fmt.Sprintf("%s/%s", a, m)
		splits = append(splits, s)
	}

	// return all split parameters
	return splits
}

// parseDNSSplitExcXML parses the DNS-based Split Exclude List contained in
// postAuthXML
func parseDNSSplitExcXML(postAuthXML string) []string {
	type DNSSplitExc struct {
		Domains string `xml:"config>opaque>custom-attr>dynamic-split-exclude-domains"`
	}
	d := &DNSSplitExc{}
	err := xml.Unmarshal([]byte(postAuthXML), d)
	if err != nil {
		log.WithError(err).
			Error("VPNCScript could not parse split excludes in post auth XML")
		return nil
	}
	return strings.Split(d.Domains, ",")
}

// parseBypassVSubnetsXML parses the Bypass Virtual Subnets Only V4 setting
// contained in postAuthXML
func parseBypassVSubnetsXML(postAuthXML string) bool {
	type BypassVSubnets struct {
		Bypass bool `xml:"config>opaque>custom-attr>BypassVirtualSubnetsOnlyV4"`
	}
	b := &BypassVSubnets{}
	err := xml.Unmarshal([]byte(postAuthXML), b)
	if err != nil {
		log.WithError(err).
			Error("VPNCScript could not parse bypass virtual subnets in post auth XML")
		return false
	}
	return b.Bypass
}

// getPostAuthXML gets the post auth xml from ciscoCSTPOptions
func getPostAuthXML(ciscoCSTPOptions []string) string {
	for _, opt := range ciscoCSTPOptions {
		pair := strings.SplitN(opt, "=", 2)
		key := pair[0]
		if key != "X-CSTP-Post-Auth-XML" {
			continue
		}
		if len(pair) != 2 || pair[1] == "" {
			return ""
		}
		value := pair[1]
		return value
	}

	return ""
}

// parseDNSSplitExc parses the DNS-based Split Exclude List contained in
// ciscoCSTPOptions
func parseDNSSplitExc(ciscoCSTPOptions []string) []string {
	xml := getPostAuthXML(ciscoCSTPOptions)
	if xml != "" {
		return parseDNSSplitExcXML(xml)
	}

	return nil
}

// parseBypassVSubnets parses the bypass virtual subnets only v4 setting in
// ciscoCSTPOptions
func parseBypassVSubnets(ciscoCSTPOptions []string) bool {
	xml := getPostAuthXML(ciscoCSTPOptions)
	if xml != "" {
		return parseBypassVSubnetsXML(xml)
	}

	return false
}

// parseDisableAlwaysOnVPN parses the disable always on vpn setting in
// ciscoCSTPOptions
func parseDisableAlwaysOnVPN(ciscoCSTPOptions []string) bool {
	for _, opt := range ciscoCSTPOptions {
		pair := strings.SplitN(opt, "=", 2)
		key := pair[0]
		if key != "X-CSTP-Disable-Always-On-VPN" {
			continue
		}
		if len(pair) != 2 || pair[1] != "true" {
			return false
		}
		return true
	}

	return false
}

// parseEnvironment parses environment variables and collects
// openconnect settings
func parseEnvironment() *env {
	e := &env{}

	// parse reason:
	// reason -- why this script was called, one of:
	// pre-init connect disconnect reconnect attempt-reconnect
	// TODO: check values
	e.reason = os.Getenv("reason")

	// parse vpn gateway:
	// VPNGATEWAY -- VPN gateway address (always present)
	e.vpnGateway = os.Getenv("VPNGATEWAY")

	// parse vpn PID:
	// VPNPID -- PID of the process controlling the VPN (OpenConnect v9.0+)
	e.vpnPID = os.Getenv("VPNPID")

	// parse tunnel device:
	// TUNDEV -- tunnel device (always present)
	e.tunDev = os.Getenv("TUNDEV")

	// parse idle timeout
	// IDLE_TIMEOUT -- gateway's idle timeout in seconds (OpenConnect
	// v8.06+); unused
	e.idleTimeout = os.Getenv("IDLE_TIMEOUT")

	// parse internal ipv4 address
	// INTERNAL_IP4_ADDRESS -- address (always present)
	e.internalIP4Address = os.Getenv("INTERNAL_IP4_ADDRESS")

	// parse internal ipv4 mtu
	// INTERNAL_IP4_MTU -- MTU (often unset)
	e.internalIP4MTU = os.Getenv("INTERNAL_IP4_MTU")

	// parse internal ipv4 netmask
	// INTERNAL_IP4_NETMASK -- netmask (often unset)
	e.internalIP4Netmask = os.Getenv("INTERNAL_IP4_NETMASK")

	// parse internal ipv4 netmask length
	// INTERNAL_IP4_NETMASKLEN -- netmask length (often unset)
	e.internalIP4NetmaskLen = os.Getenv("INTERNAL_IP4_NETMASKLEN")

	// parse internal ipv4 network address
	// INTERNAL_IP4_NETADDR -- address of network (only present if netmask
	// is set)
	e.internalIP4NetAddr = os.Getenv("INTERNAL_IP4_NETADDR")

	// parse internal ipv4 dns servers
	// INTERNAL_IP4_DNS -- list of DNS servers
	e.internalIP4DNS = os.Getenv("INTERNAL_IP4_DNS")

	// parse internal ipv4 wins servers
	// INTERNAL_IP4_NBNS -- list of WINS servers
	e.internalIP4NBNS = os.Getenv("INTERNAL_IP4_NBNS")

	// parse internal ipv6 address
	// INTERNAL_IP6_ADDRESS -- IPv6 address
	e.internalIP6Address = os.Getenv("INTERNAL_IP6_ADDRESS")

	// parse internal ipv6 netmask
	// INTERNAL_IP6_NETMASK -- IPv6 netmask
	e.internalIP6Netmask = os.Getenv("INTERNAL_IP6_NETMASK")

	// parse internal ipv6 dns servers
	// INTERNAL_IP6_DNS -- IPv6 list of dns servers
	e.internalIP6DNS = os.Getenv("INTERNAL_IP6_DNS")

	// parse default domain
	// CISCO_DEF_DOMAIN -- default domain name
	e.ciscoDefDomain = os.Getenv("CISCO_DEF_DOMAIN")

	// parse banner
	// CISCO_BANNER -- banner from server
	e.ciscoBanner = os.Getenv("CISCO_BANNER")

	// parse split dns
	// CISCO_SPLIT_DNS -- DNS search domain list
	e.ciscoSplitDNS = os.Getenv("CISCO_SPLIT_DNS")

	// parse split include ipv4 network list
	// CISCO_SPLIT_INC             -- number of networks in
	//                                split-network-list
	// CISCO_SPLIT_INC_%d_ADDR     -- network address
	// CISCO_SPLIT_INC_%d_MASK     -- subnet mask (for example:
	//                                255.255.255.0)
	// CISCO_SPLIT_INC_%d_MASKLEN  -- subnet masklen (for example: 24)
	// CISCO_SPLIT_INC_%d_PROTOCOL -- protocol (often just 0); unused
	// CISCO_SPLIT_INC_%d_SPORT    -- source port (often just 0); unused
	// CISCO_SPLIT_INC_%d_DPORT    -- destination port (often just 0);
	//                                unused
	e.ciscoSplitInc = parseEnvironmentSplit("CISCO_SPLIT_INC")

	// parse split exclude ipv4 network list
	e.ciscoSplitExc = parseEnvironmentSplit("CISCO_SPLIT_EXC")

	// parse split include ipv6 network list
	// CISCO_IPV6_SPLIT_INC             -- number of networks in IPv6
	//                                     split-network-list
	// CISCO_IPV6_SPLIT_INC_%d_ADDR     -- IPv6 network address
	// CISCO_IPV6_SPLIT_INC_$%d_MASKLEN -- IPv6 subnet masklen
	e.ciscoIPv6SplitInc = parseEnvironmentSplit("CISCO_IPV6_SPLIT_INC")

	// parse split exclude ipv6 network list
	e.ciscoIPv6SplitExc = parseEnvironmentSplit("CISCO_IPV6_SPLIT_EXC")

	// parse cstp options
	// CISCO_CSTP_OPTIONS -- List of all CSTP headers, includes dynamic
	//                       DNS-based Split Exclude settings in nested
	//                       X-CSTP-Post-Auth-XML header
	e.ciscoCSTPOptions = strings.Split(os.Getenv("CISCO_CSTP_OPTIONS"), "\n")

	// parse dynamic dns-based split exclude list
	e.dnsSplitExc = parseDNSSplitExc(e.ciscoCSTPOptions)

	// parse bypass virtual subnets only v4 setting
	e.bypassVirtualSubnetsOnlyV4 = parseBypassVSubnets(e.ciscoCSTPOptions)

	// parse Disable Always On VPN
	e.disableAlwaysOnVPN = parseDisableAlwaysOnVPN(e.ciscoCSTPOptions)

	// parse openconnect daemon token, socket file, verbosity
	e.token = os.Getenv("oc_daemon_token")
	e.socketFile = os.Getenv("oc_daemon_socket_file")
	e.verbose = false
	if os.Getenv("oc_daemon_verbose") == "true" {
		e.verbose = true
	}

	return e
}

// printDebugEnvironment prints all environment variables as debug output
func printDebugEnvironment() {
	for _, e := range os.Environ() {
		log.WithField("variable", e).Debug("VPNCScript got environment variable")
	}
}
