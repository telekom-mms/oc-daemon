package vpnconfig

import (
	"encoding/json"
	"log"
	"net"
	"reflect"
	"testing"
)

// TestDNSRemotes tests Remotes of DNS
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

// TestSplitDNSExcludes tests DNSExcludes of Split
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

// TestConfigCopy tests Copy of Config
func TestConfigCopy(t *testing.T) {
	want := getValidTestConfig()
	got := want.Copy()

	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestConfigEmpty tests Empty of Config
func TestConfigEmpty(t *testing.T) {
	// test empty
	c := New()
	want := true
	got := c.Empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test not empty
	c = New()
	c.PID = 1
	want = false
	got = c.Empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestConfigEqual tests Equal of Config
func TestConfigEqual(t *testing.T) {
	// test empty
	c1 := New()
	c2 := New()
	want := true
	got := c1.Equal(c2)
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test not empty
	c1 = New()
	c1.PID = 1
	c2 = New()
	c2.PID = 1
	want = true
	got = c1.Equal(c2)
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// getValidTestConfig returns a valid Config for testing
func getValidTestConfig() *Config {
	c := New()

	c.Gateway = net.IPv4(192, 168, 0, 1)
	c.PID = 123456
	c.Timeout = 300
	c.Device.Name = "tun0"
	c.Device.MTU = 1300
	c.IPv4.Address = net.IPv4(192, 168, 0, 123)
	c.IPv4.Netmask = net.IPv4Mask(255, 255, 255, 0)
	c.DNS.DefaultDomain = "mycompany.com"
	c.DNS.ServersIPv4 = []net.IP{net.IPv4(192, 168, 0, 1)}
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
	c.Split.ExcludeDNS = []string{"this.other.com", "that.other.com"}
	c.Split.ExcludeVirtualSubnetsOnlyIPv4 = true
	c.Flags.DisableAlwaysOnVPN = true

	return c
}

// TestConfigValid tests Valid of Config
func TestConfigValid(t *testing.T) {
	// test empty, valid
	c := New()
	want := true
	got := c.Valid()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test invalid
	c = New()
	c.Device.Name = "this is too long for a device name"
	want = false
	got = c.Valid()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test valid
	c = getValidTestConfig()
	want = true
	got = c.Valid()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestConfigJSON tests JSON of Config
func TestConfigJSON(t *testing.T) {
	c := getValidTestConfig()

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

// TestNew tests New
func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Errorf("got nil, want != nil")
	}
}

// TestNewFromJSON tests NewFromJSON
func TestNewFromJSON(t *testing.T) {
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
}
