package xmlprofile

import (
	"encoding/xml"
	"os"
	"reflect"
	"testing"
)

// TestProfileGetAllowedHosts tests GetAllowedHosts of Profile
func TestProfileGetAllowedHosts(t *testing.T) {
	p := NewProfile()

	// test empty
	var want []string
	got := p.GetAllowedHosts()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test filled
	p.AutomaticVPNPolicy.AlwaysOn.AllowedHosts =
		"192.168.1.1,somecompany.com,10.0.0.0/8"
	want = []string{
		"192.168.1.1",
		"somecompany.com",
		"10.0.0.0/8",
	}
	got = p.GetAllowedHosts()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestProfileGetVPNServers tests GetVPNServers of Profile
func TestProfileGetVPNServers(t *testing.T) {
	p := NewProfile()

	// test empty
	var want []string
	got := p.GetVPNServers()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test filled
	p.ServerList.HostEntry = []HostEntry{
		{
			HostName:    "vpn1.mycompany.com",
			HostAddress: "vpn1.mycompany.com",
		},
		{
			HostName:    "vpn2.mycompany.com",
			HostAddress: "vpn2.mycompany.com",
		},
	}
	want = []string{
		"vpn1.mycompany.com",
		"vpn2.mycompany.com",
	}
	got = p.GetVPNServers()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestProfileGetVPNServerHostNames tests GetVPNServerHostNames of Profile
func TestProfileGetVPNServerHostNames(t *testing.T) {
	p := NewProfile()

	// test empty
	var want []string
	got := p.GetVPNServerHostNames()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test filled
	p.ServerList.HostEntry = []HostEntry{
		{
			HostName: "vpn1.mycompany.com",
		},
		{
			HostName: "vpn2.mycompany.com",
		},
	}
	want = []string{
		"vpn1.mycompany.com",
		"vpn2.mycompany.com",
	}
	got = p.GetVPNServerHostNames()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestProfileGetTNDServers tests GetTNDServers of Profile
func TestProfileGetTNDServers(t *testing.T) {
	p := NewProfile()

	// test empty
	var want []string
	got := p.GetTNDServers()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test filled
	p.AutomaticVPNPolicy.TrustedHTTPSServerList = []TrustedHTTPSServer{
		{
			Address:         "tnd1.mycompany.com",
			Port:            "443",
			CertificateHash: "hash of tnd1 certificate",
		},
		{
			Address:         "tnd2.mycompany.com",
			Port:            "443",
			CertificateHash: "hash of tnd2 certificate",
		},
	}
	want = []string{
		"tnd1.mycompany.com",
		"tnd2.mycompany.com",
	}
	got = p.GetTNDServers()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestProfileGetTNDHTTPSServers tests GetTNDHTTPSServers of Profile
func TestProfileGetTNDHTTPSServers(t *testing.T) {
	p := NewProfile()

	// test empty
	var wantURLs []string
	var wantHashes []string
	gotURLs, gotHashes := p.GetTNDHTTPSServers()
	if !reflect.DeepEqual(gotURLs, wantURLs) ||
		!reflect.DeepEqual(gotHashes, wantHashes) {

		t.Errorf("got %v, %v, want %v, %v",
			gotURLs, gotHashes, wantURLs, wantHashes)
	}

	// test filled
	p.AutomaticVPNPolicy.TrustedHTTPSServerList = []TrustedHTTPSServer{
		{
			Address:         "tnd1.mycompany.com",
			Port:            "443",
			CertificateHash: "hash of tnd1 certificate",
		},
		{
			Address:         "tnd2.mycompany.com",
			Port:            "443",
			CertificateHash: "hash of tnd2 certificate",
		},
	}
	wantURLs = []string{
		"https://tnd1.mycompany.com:443",
		"https://tnd2.mycompany.com:443",
	}
	wantHashes = []string{
		"hash of tnd1 certificate",
		"hash of tnd2 certificate",
	}
	gotURLs, gotHashes = p.GetTNDHTTPSServers()
	if !reflect.DeepEqual(gotURLs, wantURLs) ||
		!reflect.DeepEqual(gotHashes, wantHashes) {

		t.Errorf("got %v, %v, want %v, %v",
			gotURLs, gotHashes, wantURLs, wantHashes)
	}
}

// TestProfileGetAlwaysOn tests GetAlwaysOn of Profile
func TestProfileGetAlwaysOn(t *testing.T) {
	p := NewProfile()

	want := false
	got := p.GetAlwaysOn()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestNewProfile tests NewProfile
func TestNewProfile(t *testing.T) {
	p := NewProfile()
	if p == nil {
		t.Errorf("got nil, want != nil")
	}
}

// TestLoadProfile tests LoadProfile
func TestLoadProfile(t *testing.T) {
	empty := NewProfile()

	// test not existing file
	if _, err := LoadProfile("does not exists"); err == nil {
		t.Error("got err == nil, want err != nil")
	}

	// test empty file
	f, err := os.CreateTemp("", "xmlprofile-test")
	if err != nil {
		t.Error(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()

	if _, err := LoadProfile(f.Name()); err == nil {
		t.Error("got err == nil, want err != nil")
	}

	// test empty config in file
	b, err := xml.Marshal(empty)
	if err != nil {
		t.Error(err)
	}
	if _, err := f.Write(b); err != nil {
		t.Error(err)
	}
	p, err := LoadProfile(f.Name())
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(p, empty) {
		t.Errorf("got %v, want %v", p, empty)
	}
}
