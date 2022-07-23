package xmlprofile

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Profile is an xml Profile
type Profile struct {
	file    string
	profile *AnyConnectProfile
	watch   *Watch
}

// GetAllowedHosts returns the allowed hosts in the XML profile
func (p *Profile) GetAllowedHosts() (hosts []string) {
	hs := p.profile.AutomaticVPNPolicy.AlwaysOn.AllowedHosts
	for _, h := range strings.Split(hs, ",") {
		log.WithField("host", h).Debug("Getting allowed host from Profile")
		hosts = append(hosts, h)
	}
	return
}

// GetVPNServers returns the VPN servers in the XML profile
func (p *Profile) GetVPNServers() (servers []string) {
	for _, h := range p.profile.ServerList.HostEntry {
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

// GetVPNServerHostNames returns the VPN server hostnames in the xml profile
func (p *Profile) GetVPNServerHostNames() (servers []string) {
	for _, s := range p.profile.ServerList.HostEntry {
		if strings.HasPrefix(s.PrimaryProtocol.Flag, "IPsec") {
			continue
		}
		servers = append(servers, s.HostName)
	}
	return
}

// GetTNDServers returns the TND servers in the XML profile
func (p *Profile) GetTNDServers() (servers []string) {
	for _, s := range p.profile.AutomaticVPNPolicy.TrustedHTTPSServerList {
		log.WithField("server", s).Debug("Getting TND server from Profile")
		servers = append(servers, s.Address)
	}
	return
}

// GetTNDHTTPSServers gets the TND HTTPS server URLs and their hashes in the XML profile
func (p *Profile) GetTNDHTTPSServers() (urls, hashes []string) {
	for _, s := range p.profile.AutomaticVPNPolicy.TrustedHTTPSServerList {
		url := fmt.Sprintf("https://%s:%s", s.Address, s.Port)
		urls = append(urls, url)
		hashes = append(hashes, s.CertificateHash)
	}
	if len(urls) != len(hashes) {
		return nil, nil
	}
	return
}

// GetAlwaysOn returns the always on flag in the XML profile
func (p *Profile) GetAlwaysOn() bool {
	return p.profile.AutomaticVPNPolicy.AlwaysOn.Flag
}

// Parse parses the xml profile
func (p *Profile) Parse() {
	// initialize to empty profile in case file is invalid
	p.profile = &AnyConnectProfile{}

	b, err := os.ReadFile(p.file)
	if err != nil {
		log.WithError(err).Error("Could not read xml profile")
		return
	}

	profile := &AnyConnectProfile{}
	if err := xml.Unmarshal(b, profile); err != nil {
		log.WithError(err).Error("Could not parse xml profile")
		return
	}

	p.profile = profile
}

// Start starts watching the profile for changes
func (p *Profile) Start() {
	p.watch.Start()
}

// Stop stops watching the profile
func (p *Profile) Stop() {
	p.watch.Stop()
}

// Updates returns the channel for xml profile updates
func (p *Profile) Updates() chan struct{} {
	return p.watch.updates
}

// NewXMLProfile returns a new Profile
func NewXMLProfile(xmlProfile string) *Profile {
	return &Profile{
		file:  xmlProfile,
		watch: NewWatch(xmlProfile),
	}
}
