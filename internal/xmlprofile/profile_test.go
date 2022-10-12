package xmlprofile

import (
	"os"
	"reflect"
	"testing"
)

// TestProfileGetAllowedHosts tests GetAllowedHosts of Profile
func TestProfileGetAllowedHosts(t *testing.T) {
	p := NewXMLProfile("does not exist")
	p.Parse()

	// test empty
	var want []string
	got := p.GetAllowedHosts()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test filled
	p.profile.AutomaticVPNPolicy.AlwaysOn.AllowedHosts =
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
	p := NewXMLProfile("does not exist")
	p.Parse()

	// test empty
	var want []string
	got := p.GetVPNServers()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test filled
	p.profile.ServerList.HostEntry = []HostEntry{
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
	p := NewXMLProfile("does not exist")
	p.Parse()

	// test empty
	var want []string
	got := p.GetVPNServerHostNames()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test filled
	p.profile.ServerList.HostEntry = []HostEntry{
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
	p := NewXMLProfile("does not exist")
	p.Parse()

	// test empty
	var want []string
	got := p.GetTNDServers()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test filled
	p.profile.AutomaticVPNPolicy.TrustedHTTPSServerList = []TrustedHTTPSServer{
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
	p := NewXMLProfile("does not exist")
	p.Parse()

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
	p.profile.AutomaticVPNPolicy.TrustedHTTPSServerList = []TrustedHTTPSServer{
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
	p := NewXMLProfile("does not exist")
	p.Parse()

	want := false
	got := p.GetAlwaysOn()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestProfileParse tests Parse of Profile
func TestProfileParse(t *testing.T) {
	empty := &AnyConnectProfile{}

	// test not existing file
	p := NewXMLProfile("does not exist")
	p.Parse()
	if !reflect.DeepEqual(p.profile, empty) {
		t.Errorf("got %v, want %v", p.profile, empty)
	}

	// test empty file
	f := createWatchTestFile()
	defer os.Remove(f)

	p = NewXMLProfile(f)
	p.Parse()
	if !reflect.DeepEqual(p.profile, empty) {
		t.Errorf("got %v, want %v", p.profile, empty)
	}
}

// TestProfileStartStop tests Start and Stop of Profile
func TestProfileStartStop(t *testing.T) {
	f := createWatchTestFile()
	defer os.Remove(f)

	p := NewXMLProfile(f)
	p.Start()
	p.Stop()
}

// TestProfileUpdates tests Updates of Profile
func TestProfileUpdates(t *testing.T) {
	p := NewXMLProfile("profile.xml")
	want := p.watch.updates
	got := p.Updates()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewXMLProfile tests NewXMLProfile
func TestNewXMLProfile(t *testing.T) {
	f := "some file"
	p := NewXMLProfile(f)
	if p.file != f {
		t.Errorf("got %s, want %s", p.file, f)
	}
	if p.watch == nil {
		t.Errorf("got nil, want != nil")
	}
}
