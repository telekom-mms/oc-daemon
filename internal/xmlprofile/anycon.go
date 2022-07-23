package xmlprofile

// TrustedHTTPSServer is a trusted https server in the profile
type TrustedHTTPSServer struct {
	Address         string `xml:"Address"`
	Port            string `xml:"Port"`
	CertificateHash string `xml:"CertificateHash"`
}

// AllowCaptivePortalRemediation specifies if captive portal remediation is
// allowed in the profile
type AllowCaptivePortalRemediation struct {
	Flag                            string `xml:",chardata"`
	CaptivePortalRemediationTimeout string `xml:"CaptivePortalRemediationTimeout"`
}

// ConnectFailurePolicy specifies the connect failure policy in the profile
type ConnectFailurePolicy struct {
	Flag                           string                        `xml:",chardata"`
	AllowCaptivePortalRemediation  AllowCaptivePortalRemediation `xml:"AllowCaptivePortalRemediation"`
	ApplyLastVPNLocalResourceRules string                        `xml:"ApplyLastVPNLocalResourceRules"`
}

// AlwaysOn contains the AlwaysOn settings in the profile
type AlwaysOn struct {
	Flag                 bool                 `xml:",chardata"`
	ConnectFailurePolicy ConnectFailurePolicy `xml:"ConnectFailurePolicy"`
	AllowVPNDisconnect   string               `xml:"AllowVPNDisconnect"`
	AllowedHosts         string               `xml:"AllowedHosts"`
}

// AutomaticVPNPolicy contains the automatic vpn policy in the profile
type AutomaticVPNPolicy struct {
	Flag                   string               `xml:",chardata"`
	TrustedDNSDomains      []string             `xml:"TrustedDNSDomains"`
	TrustedDNSServers      []string             `xml:"TrustedDNSServers"`
	TrustedHTTPSServerList []TrustedHTTPSServer `xml:"TrustedHttpsServerList>TrustedHttpsServer"`
	TrustedNetworkPolicy   string               `xml:"TrustedNetworkPolicy"`
	UntrustedNetworkPolicy string               `xml:"UntrustedNetworkPolicy"`
	AlwaysOn               AlwaysOn             `xml:"AlwaysOn"`
}

// MobileHostEntryInfo contains the mobile host entry info in the profile
type MobileHostEntryInfo struct {
	NetworkRoaming            string   `xml:"NetworkRoaming"`
	CertificatePolicy         string   `xml:"CertificatePolicy"`
	ConnectOnDemand           string   `xml:"ConnectOnDemand"`
	AlwaysConnectDomainList   []string `xml:"AlwaysConnectDomainList"`
	NeverConnectDomainList    []string `xml:"NeverConnectDomainList"`
	ConnectIfNeededDomainList []string `xml:"ConnectIfNeededDomainList"`
	ActivateOnImport          string   `xml:"ActivateOnImport"`
}

// StandardAuthenticationOnly specifies standard authentication in the profile
type StandardAuthenticationOnly struct {
	Flag                           string `xml:",chardata"`
	AuthMethodDuringIKENegotiation string `xml:"AuthMethodDuringIKENegotiation"`
	IKEIdentity                    string `xml:"IKEIdentity"`
}

// PrimaryProtocol specifies primary protocol in the profile
type PrimaryProtocol struct {
	Flag                       string                     `xml:",chardata"`
	StandardAuthenticationOnly StandardAuthenticationOnly `xml:"StandardAuthenticationOnly"`
}

// Pin is a pin in the profile
type Pin struct {
	Subject string `xml:"Subject,attr"`
	Issuer  string `xml:"Issuer,attr"`
}

// HostEntry is a host entry in the profile
type HostEntry struct {
	HostName                string              `xml:"HostName"`
	HostAddress             string              `xml:"HostAddress"`
	UserGroup               string              `xml:"UserGroup"`
	BackupServerList        []string            `xml:"BackupServerList>HostAddress"`
	LoadBalancingServerList []string            `xml:"LoadBalancingServerList>HostAddress"`
	AutomaticSCEPHost       string              `xml:"AutomaticSCEPHost"`
	CAURL                   string              `xml:"CAURL"`
	MobileHostEntryInfo     MobileHostEntryInfo `xml:"MobileHostEntryInfo"`
	PrimaryProtocol         PrimaryProtocol     `xml:"PrimaryProtocol"`
	CertificatePinList      []Pin               `xml:"CertificatePinList>Pin"`
}

// ServerList is the server list in the profile
type ServerList struct {
	HostEntry []HostEntry `xml:"HostEntry"`
}

// AnyConnectProfile is the anyconnet profile
type AnyConnectProfile struct {
	AutomaticVPNPolicy AutomaticVPNPolicy `xml:"ClientInitialization>AutomaticVPNPolicy"`
	ServerList         ServerList         `xml:"ServerList"`
}
