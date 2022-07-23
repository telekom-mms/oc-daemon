package vpncscript

import (
	"reflect"
	"testing"
)

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

// TestParseBypassVSubnetsXML tests parsing the bypass virtual subnets only v4
// setting from xml
func TestParseBypassVSubnetsXMLXML(t *testing.T) {
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
