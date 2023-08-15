package vpncscript

import (
	"os"
	"reflect"
	"testing"
)

// TestParseEnvironmentSplit tests parseEnvironmentSplit
func TestParseEnvironmentSplit(t *testing.T) {
	// test empty environment
	os.Clearenv()
	for _, prefix := range []string{
		"CISCO_SPLIT_INC",
		"CISCO_SPLIT_EXC",
		"CISCO_IPV6_SPLIT_INC",
		"CISCO_IPV6_SPLIT_EXC",
	} {
		want := []string{}
		got := parseEnvironmentSplit(prefix)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test with environment variables set
	os.Setenv("CISCO_SPLIT_EXC", "3")
	os.Setenv("CISCO_SPLIT_EXC_0_ADDR", "192.168.1.0")
	os.Setenv("CISCO_SPLIT_EXC_0_MASKLEN", "24")
	os.Setenv("CISCO_SPLIT_EXC_1_ADDR", "172.16.0.0")
	os.Setenv("CISCO_SPLIT_EXC_1_MASKLEN", "16")
	os.Setenv("CISCO_SPLIT_EXC_2_ADDR", "10.0.0.0")
	os.Setenv("CISCO_SPLIT_EXC_2_MASKLEN", "8")

	want := []string{
		"192.168.1.0/24",
		"172.16.0.0/16",
		"10.0.0.0/8",
	}
	got := parseEnvironmentSplit("CISCO_SPLIT_EXC")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestParseDNSSplitExcXML tests parsing the dns-based split exclude list
// from xml
func TestParseDNSSplitExcXML(t *testing.T) {
	test := func(want, got []string) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test empty xml
	xml1 := ""
	test(nil, parseDNSSplitExcXML(xml1))

	// test valid xml
	xml2 := `<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
	<dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`
	domains := []string{"some.example.com", "other.example.com",
		"www.example.com"}
	test(domains, parseDNSSplitExcXML(xml2))

	// test valid xml with another custom-attr
	xml3 := `<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
	<BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`
	test(domains, parseDNSSplitExcXML(xml3))
}

// TestParseBypassVSubnetsXML tests parsing the bypass virtual subnets only v4
// setting from xml
func TestParseBypassVSubnetsXML(t *testing.T) {
	test := func(want, got bool) {
		if got != want {
			t.Errorf("got %t, want %t", got, want)
		}
	}

	// test empty xml
	xml1 := ""
	test(false, parseBypassVSubnetsXML(xml1))

	// test valid xml with only an other custom attr
	xml2 := `<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`
	test(false, parseBypassVSubnetsXML(xml2))

	// test valid xml
	xml3 := `<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
	<BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`
	test(true, parseBypassVSubnetsXML(xml3))

	// test valid xml with another custom-attr
	xml4 := `<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
	<BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`
	test(true, parseBypassVSubnetsXML(xml4))
}

// TestGetPostAuthXML tests getPostAuthXML
func TestGetPostAuthXML(t *testing.T) {
	// test invalid/not existing
	for _, invalid := range [][]string{
		nil, {}, {"something else"},
	} {
		want := ""
		got := getPostAuthXML(invalid)
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test valid
	valid := []string{`X-CSTP-Post-Auth-XML=<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
	<BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`}
	want := `<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
	<BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`
	got := getPostAuthXML(valid)
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestParseDNSSplitExc tests parsing the dns-based split exclude list
// from CSTP options
func TestParseDNSSplitExc(t *testing.T) {
	test := func(want, got []string) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test empty opts
	opts1 := []string{}
	test(nil, parseDNSSplitExc(opts1))

	// test post auth xml not present
	opts2 := []string{"nothing=valid", "not=there", "a=b"}
	test(nil, parseDNSSplitExc(opts2))

	// test empty post auth xml
	opts3 := []string{"X-CSTP-Post-Auth-XML="}
	test(nil, parseDNSSplitExc(opts3))

	// test valid post auth xml
	opts4 := []string{`X-CSTP-Post-Auth-XML=<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`}
	domains := []string{"some.example.com", "other.example.com",
		"www.example.com"}
	test(domains, parseDNSSplitExc(opts4))

	// test valid post auth xml with another custom-attr
	opts5 := []string{`X-CSTP-Post-Auth-XML=<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
	<BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`}
	test(domains, parseDNSSplitExc(opts5))
}

// TestParseBypassVSubnets tests parseBypassVSubnets
func TestParseBypassVSubnets(t *testing.T) {
	test := func(want, got bool) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %t, want %t", got, want)
		}
	}

	// test empty opts
	opts1 := []string{}
	test(false, parseBypassVSubnets(opts1))

	// test post auth xml not present
	opts2 := []string{"nothing=valid", "not=there", "a=b"}
	test(false, parseBypassVSubnets(opts2))

	// test empty post auth xml
	opts3 := []string{"X-CSTP-Post-Auth-XML="}
	test(false, parseBypassVSubnets(opts3))

	// test valid post auth xml without bypass vsubnets
	opts4 := []string{`X-CSTP-Post-Auth-XML=<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`}
	test(false, parseBypassVSubnets(opts4))

	// test valid post auth xml with bypass vsubnets
	opts5 := []string{`X-CSTP-Post-Auth-XML=<?xml version="1.0" encoding="UTF-8"?>
<config-auth client="vpn" type="complete" aggregate-auth-version="2">
  <config client="vpn" type="private">
    <opaque is-for="vpn-client">
      <custom-attr>
        <dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains>
	<BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4>
      </custom-attr>
    </opaque>
  </config>
</config-auth>`}
	test(true, parseBypassVSubnets(opts5))
}

// TestParseDisableAlwaysOnVPN tests parseDisableAlwaysOnVPN
func TestParseDisableAlwaysOnVPN(t *testing.T) {
	test := func(want, got bool) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %t, want %t", got, want)
		}
	}

	// test empty opts
	opts := []string{}
	test(false, parseDisableAlwaysOnVPN(opts))

	// test incomplete or false opts
	opts = []string{
		"",
		"X-CSTP-Disable-Always-On-VPN",
		"X-CSTP-Disable-Always-On-VPN=",
		"X-CSTP-Disable-Always-On-VPN=false",
	}
	test(false, parseDisableAlwaysOnVPN(opts))

	// test complete
	opts = []string{"X-CSTP-Disable-Always-On-VPN=true"}
	test(true, parseDisableAlwaysOnVPN(opts))
}

// TestParseEnvironment tests parseEnvironment
func TestParseEnvironment(t *testing.T) {
	// setup test environment
	os.Clearenv()
	for k, v := range map[string]string{
		"reason":                     "connect",
		"VPNGATEWAY":                 "10.1.1.1",
		"VPNPID":                     "12345",
		"TUNDEV":                     "tun0",
		"IDLE_TIMEOUT":               "300",
		"INTERNAL_IP4_ADDRESS":       "192.168.1.123",
		"INTERNAL_IP4_MTU":           "1300",
		"INTERNAL_IP4_NETMASK":       "255.255.255.0",
		"INTERNAL_IP4_NETMASKLEN":    "24",
		"INTERNAL_IP4_NETADDR":       "192.168.1.0",
		"INTERNAL_IP4_DNS":           "192.168.1.1",
		"INTERNAL_IP4_NBNS":          "192.168.1.1",
		"INTERNAL_IP6_ADDRESS":       "",
		"INTERNAL_IP6_NETMASK":       "",
		"INTERNAL_IP6_DNS":           "",
		"CISCO_DEF_DOMAIN":           "example.com",
		"CISCO_BANNER":               "some banner",
		"CISCO_SPLIT_DNS":            "",
		"CISCO_SPLIT_INC":            "0",
		"CISCO_SPLIT_EXC":            "1",
		"CISCO_SPLIT_EXC_0_ADDR":     "172.16.0.0",
		"CISCO_SPLIT_EXC_0_MASK":     "255.255.0.0",
		"CISCO_SPLIT_EXC_0_MASKLEN":  "16",
		"CISCO_SPLIT_EXC_0_PROTOCOL": "0",
		"CISCO_SPLIT_EXC_0_SPORT":    "0",
		"CISCO_SPLIT_EXC_0_DPORT":    "0",
		"CISCO_IPV6_SPLIT_INC":       "0",
		"CISCO_IPV6_SPLITEXCC":       "0",
		"CISCO_CSTP_OPTIONS": `X-CSTP-Post-Auth-XML=<?xml version="1.0" encoding="UTF-8"?><config-auth client="vpn" type="complete" aggregate-auth-version="2"><config client="vpn" type="private"><opaque is-for="vpn-client"><custom-attr><dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains><BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4></custom-attr></opaque></config></config-auth>
X-CSTP-Disable-Always-On-VPN=true`,
		"oc_daemon_token":       "some token",
		"oc_daemon_socket_file": "/run/oc-daemon/test.socket",
		"oc_daemon_verbose":     "true",
	} {
		os.Setenv(k, v)
	}

	// create expected env struct based on test environment
	want := &env{
		reason:                "connect",
		vpnGateway:            "10.1.1.1",
		vpnPID:                "12345",
		tunDev:                "tun0",
		idleTimeout:           "300",
		internalIP4Address:    "192.168.1.123",
		internalIP4MTU:        "1300",
		internalIP4Netmask:    "255.255.255.0",
		internalIP4NetmaskLen: "24",
		internalIP4NetAddr:    "192.168.1.0",
		internalIP4DNS:        "192.168.1.1",
		internalIP4NBNS:       "192.168.1.1",
		ciscoDefDomain:        "example.com",
		ciscoBanner:           "some banner",
		ciscoSplitInc:         []string{},
		ciscoSplitExc:         []string{"172.16.0.0/16"},
		ciscoIPv6SplitInc:     []string{},
		ciscoIPv6SplitExc:     []string{},
		ciscoCSTPOptions: []string{
			`X-CSTP-Post-Auth-XML=<?xml version="1.0" encoding="UTF-8"?><config-auth client="vpn" type="complete" aggregate-auth-version="2"><config client="vpn" type="private"><opaque is-for="vpn-client"><custom-attr><dynamic-split-exclude-domains><![CDATA[some.example.com,other.example.com,www.example.com]]></dynamic-split-exclude-domains><BypassVirtualSubnetsOnlyV4><![CDATA[true]]></BypassVirtualSubnetsOnlyV4></custom-attr></opaque></config></config-auth>`,
			`X-CSTP-Disable-Always-On-VPN=true`,
		},
		dnsSplitExc:                []string{"some.example.com", "other.example.com", "www.example.com"},
		bypassVirtualSubnetsOnlyV4: true,
		disableAlwaysOnVPN:         true,
		token:                      "some token",
		socketFile:                 "/run/oc-daemon/test.socket",
		verbose:                    true,
	}

	// run test
	got := parseEnvironment()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n%#v\nwant:\n%#v", got, want)
	}
}
