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
