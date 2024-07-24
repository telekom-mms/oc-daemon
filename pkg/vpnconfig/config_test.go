package vpnconfig

import (
	"encoding/json"
	"log"
	"net"
	"reflect"
	"testing"
)

// TestDNSRemotes tests Remotes of DNS.
func TestDNSRemotes(t *testing.T) {
	// test empty
	c := New()
	if len(c.DNS.Remotes()) != 0 {
		t.Errorf("got %d, want 0", len(c.DNS.Remotes()))
	}

	// test ipv4
	for _, want := range [][]string{
		{"127.0.0.1:53"},
		{"127.0.0.1:53", "192.168.1.1:53"},
		{"127.0.0.1:53", "192.168.1.1:53", "10.0.0.1:53"},
	} {
		c := New()
		for _, ip := range want {
			ip = ip[:len(ip)-3] // remove port
			c.DNS.ServersIPv4 = append(c.DNS.ServersIPv4, net.ParseIP(ip))
		}
		got := c.DNS.Remotes()["."]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test ipv6
	for _, want := range [][]string{
		{"[::1]:53"},
		{"[::1]:53", "[2000::1]:53"},
		{"[::1]:53", "[2000::1]:53", "[2002::1]:53"},
	} {
		c := New()
		for _, ip := range want {
			ip = ip[1 : len(ip)-4] // remove port and brackets
			c.DNS.ServersIPv6 = append(c.DNS.ServersIPv6, net.ParseIP(ip))
		}
		got := c.DNS.Remotes()["."]
		log.Println(got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// test both ipv4 and ipv6
	c = New()
	dns4 := "127.0.0.1"
	dns6 := "::1"
	c.DNS.ServersIPv4 = append(c.DNS.ServersIPv4, net.ParseIP(dns4))
	c.DNS.ServersIPv6 = append(c.DNS.ServersIPv6, net.ParseIP(dns6))

	want := map[string][]string{
		".": {dns4 + ":53", "[" + dns6 + "]:53"},
	}
	got := c.DNS.Remotes()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSplitDNSExcludes tests DNSExcludes of Split.
func TestSplitDNSExcludes(t *testing.T) {
	// test empty
	c := New()
	if len(c.Split.DNSExcludes()) != 0 {
		t.Errorf("got %d, want 0", len(c.Split.DNSExcludes()))
	}

	// test filled
	c = New()
	want := []string{"example.com", "test.com"}
	c.Split.ExcludeDNS = want
	for i, got := range c.Split.DNSExcludes() {
		want := want[i] + "."
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

// getValidTestConfig returns a valid Config for testing.
func getValidTestConfig() *Config {
	c := New()

	c.Gateway = net.IPv4(192, 168, 0, 1)
	c.PID = 123456
	c.Timeout = 300
	c.Device.Name = "tun0"
	c.Device.MTU = 1300
	c.IPv4.Address = net.IPv4(192, 168, 0, 123)
	c.IPv4.Netmask = net.IPv4Mask(255, 255, 255, 0)
	c.IPv6.Address = net.ParseIP("2001:42:42:42::1")
	c.IPv6.Netmask = net.CIDRMask(64, 128)
	c.DNS.DefaultDomain = "mycompany.com"
	c.DNS.ServersIPv4 = []net.IP{net.IPv4(192, 168, 0, 53)}
	c.DNS.ServersIPv6 = []net.IP{net.ParseIP("2001:53:53:53::53")}
	c.Split.ExcludeIPv4 = []*net.IPNet{
		{
			IP:   net.IPv4(0, 0, 0, 0),
			Mask: net.IPv4Mask(255, 255, 255, 255),
		},
		{
			IP:   net.IPv4(10, 0, 0, 0),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		},
	}
	c.Split.ExcludeIPv6 = []*net.IPNet{
		{
			IP:   net.ParseIP("2001:2:3:4::1"),
			Mask: net.CIDRMask(128, 128),
		},
		{
			IP:   net.ParseIP("2001:2:3:5::1"),
			Mask: net.CIDRMask(64, 128),
		},
	}
	c.Split.ExcludeDNS = []string{"this.other.com", "that.other.com"}
	c.Split.ExcludeVirtualSubnetsOnlyIPv4 = true
	c.Flags.DisableAlwaysOnVPN = true

	return c
}

// getTestConfigPID returns a Config with only PID set to 1.
func getTestConfigPID() *Config {
	c := New()
	c.PID = 1
	return c
}

// TestConfigCopy tests Copy of Config.
func TestConfigCopy(t *testing.T) {
	// test nil
	if (*Config)(nil).Copy() != nil {
		t.Error("copy of nil should be nil")
	}

	// test with new config
	want := New()
	got := want.Copy()

	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with full example config
	want = getValidTestConfig()
	got = want.Copy()

	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test without DNS servers and excludes
	want = getValidTestConfig()
	want.DNS.ServersIPv4 = nil
	want.DNS.ServersIPv6 = nil
	want.Split.ExcludeIPv4 = nil
	want.Split.ExcludeIPv6 = nil
	got = want.Copy()

	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test modification after copy
	c1 := getValidTestConfig()
	c2 := c1.Copy()
	c1.PID = 0

	if c1.Equal(c2) {
		t.Error("copies should not be equal after modification")
	}
}

// TestConfigEmpty tests Empty of Config.
func TestConfigEmpty(t *testing.T) {
	// test empty
	c := New()
	if !c.Empty() {
		t.Errorf("%v should be empty", c)
	}

	// test not empty
	for _, ne := range []*Config{
		// only PID set
		getTestConfigPID(),
		// full valid config
		getValidTestConfig(),
	} {
		if ne.Empty() {
			t.Errorf("%v should not be empty", ne)
		}
	}
}

// TestConfigEqual tests Equal of Config.
func TestConfigEqual(t *testing.T) {
	// test not equal
	c1 := New()
	c2 := getValidTestConfig()
	if c1.Equal(c2) {
		t.Errorf("%v and %v should not be equal", c1, c2)
	}

	// test equal
	for _, ne := range [][]*Config{
		// empty
		{New(), New()},
		// only PID set
		{getTestConfigPID(), getTestConfigPID()},
		// full valid config
		{getValidTestConfig(), getValidTestConfig()},
	} {
		c1, c2 := ne[0], ne[1]
		if !c1.Equal(c2) {
			t.Errorf("%v and %v should be equal", c1, c2)
		}
	}
}

// TestConfigValid tests Valid of Config.
func TestConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Config{
		// only device name set, invalid device name
		func() *Config {
			c := New()
			c.Device.Name = "this is too long for a device name"
			return c
		}(),
		// only PID set
		getTestConfigPID(),
	} {
		if invalid.Valid() {
			t.Errorf("%v should not be valid", invalid)
		}
	}

	// test valid
	for _, valid := range []*Config{
		// empty
		New(),
		// full valid config
		getValidTestConfig(),
	} {
		if !valid.Valid() {
			t.Errorf("%v should be valid", valid)
		}
	}
}

// TestConfigJSON tests JSON of Config.
func TestConfigJSON(t *testing.T) {
	c := getValidTestConfig()

	// test without errors
	want := c
	b, err := c.JSON()
	if err != nil {
		t.Error(err)
	}

	got := New()
	err = json.Unmarshal(b, got)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNew tests New.
func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Errorf("got nil, want != nil")
	}
}

// TestNewFromJSON tests NewFromJSON.
func TestNewFromJSON(t *testing.T) {
	// test with valid config
	want := getValidTestConfig()
	b, err := want.JSON()
	if err != nil {
		t.Error(err)
	}
	got, err := NewFromJSON(b)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with invalid config
	if _, err := NewFromJSON(nil); err == nil {
		t.Error("parsing invalid config should return error")
	}
}
