// Package xmlprofile contains the XML profile.
package xmlprofile

import (
	"encoding/xml"
	"fmt"
	"os"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	// SystemProfile is the file path of the system XML profile.
	SystemProfile = "/var/lib/oc-daemon/profile.xml"
)

// Profile is an XML Profile.
type Profile AnyConnectProfile

// GetAllowedHosts returns the allowed hosts in the XML profile.
func (p *Profile) GetAllowedHosts() (hosts []string) {
	hs := p.AutomaticVPNPolicy.AlwaysOn.AllowedHosts
	for _, h := range strings.Split(hs, ",") {
		if h == "" {
			continue
		}
		log.WithField("host", h).Debug("Getting allowed host from Profile")
		hosts = append(hosts, h)
	}
	return
}

// GetVPNServers returns the VPN servers in the XML profile.
func (p *Profile) GetVPNServers() (servers []string) {
	for _, h := range p.ServerList.HostEntry {
		// skip ipsec servers
		if strings.HasPrefix(h.PrimaryProtocol.Flag, "IPsec") {
			continue
		}

		// add servers to allowed hosts
		for _, s := range append(h.LoadBalancingServerList,
			h.HostAddress) {
			if s != "" {
				log.WithField("server", s).Debug("Getting VPN server from Profile")
				servers = append(servers, s)
			}
		}
	}
	return
}

// GetVPNServerHostNames returns the VPN server hostnames in the xml profile.
func (p *Profile) GetVPNServerHostNames() (servers []string) {
	for _, s := range p.ServerList.HostEntry {
		if strings.HasPrefix(s.PrimaryProtocol.Flag, "IPsec") {
			continue
		}
		servers = append(servers, s.HostName)
	}
	return
}

// GetTNDServers returns the TND servers in the XML profile.
func (p *Profile) GetTNDServers() (servers []string) {
	for _, s := range p.AutomaticVPNPolicy.TrustedHTTPSServerList {
		log.WithField("server", s).Debug("Getting TND server from Profile")
		servers = append(servers, s.Address)
	}
	return
}

// GetTNDHTTPSServers gets the TND HTTPS server URLs and their hashes in the XML profile.
func (p *Profile) GetTNDHTTPSServers() (servers map[string]string) {
	servers = make(map[string]string)
	for _, s := range p.AutomaticVPNPolicy.TrustedHTTPSServerList {
		url := fmt.Sprintf("https://%s:%s", s.Address, s.Port)
		servers[url] = s.CertificateHash
	}
	return
}

// GetAlwaysOn returns the always on flag in the XML profile.
func (p *Profile) GetAlwaysOn() bool {
	return p.AutomaticVPNPolicy.AlwaysOn.Flag
}

// Equal returns whether the profile and other are equal.
func (p *Profile) Equal(other *Profile) bool {
	return reflect.DeepEqual(p, other)
}

// NewProfile returns a new Profile.
func NewProfile() *Profile {
	return &Profile{}
}

// LoadProfile loads the XML profile from file.
func LoadProfile(file string) (*Profile, error) {
	// try to read file
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// try to parse file contents
	p := NewProfile()
	if err := xml.Unmarshal(b, p); err != nil {
		return nil, err
	}

	return p, nil
}

// LoadSystemProfile loads the XML profile from the default system location.
func LoadSystemProfile() *Profile {
	profile, err := LoadProfile(SystemProfile)
	if err != nil {
		return nil
	}
	return profile
}
