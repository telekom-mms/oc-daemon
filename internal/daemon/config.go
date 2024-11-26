package daemon

import (
	"encoding/json"
	"net/netip"
	"os"

	"github.com/telekom-mms/oc-daemon/internal/config"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/tnd/pkg/tnd"
)

var (
	// configDir is the directory for the configuration.
	configDir = "/var/lib/oc-daemon"

	// ConfigFile is the default config file.
	ConfigFile = configDir + "/oc-daemon.json"

	// DefaultDNSServer is the default DNS server address, i.e., listen
	// address of systemd-resolved.
	DefaultDNSServer = "127.0.0.53:53"
)

// Config is an OC-Daemon configuration.
// TODO: move this into separate package
// TODO: use it in cmdtmpl?
// TODO: add runtime config login with json:"-"?
// TODO: add runtime config VPNConfig with json:"-"?
type Config struct {
	Config  string `json:"-"`
	Verbose bool

	SocketServer    *config.SocketServer
	CPD             *config.CPD
	DNSProxy        *config.DNSProxy
	OpenConnect     *config.OpenConnect
	Executables     *config.Executables
	SplitRouting    *config.SplitRouting
	TrafficPolicing *config.TrafficPolicing
	TND             *tnd.Config
}

// String returns the configuration as string.
func (c *Config) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// Valid returns whether config is valid.
func (c *Config) Valid() bool {
	if c == nil ||
		!c.SocketServer.Valid() ||
		!c.CPD.Valid() ||
		!c.DNSProxy.Valid() ||
		!c.OpenConnect.Valid() ||
		!c.Executables.Valid() ||
		!c.SplitRouting.Valid() ||
		!c.TrafficPolicing.Valid() ||
		!c.TND.Valid() {
		// invalid
		return false
	}
	return true
}

// Load loads the configuration from the config file.
func (c *Config) Load() error {
	// read file contents
	file, err := os.ReadFile(c.Config)
	if err != nil {
		return err
	}

	// parse config
	if err := json.Unmarshal(file, c); err != nil {
		return err
	}

	return nil
}

func (c *Config) GetConfig() *config.Config {
	conf := &config.Config{
		Verbose: c.Verbose,

		// Socket Server
		SocketServer: c.SocketServer,

		// CPD
		CPD: c.CPD,

		// DNS Proxy
		DNSProxy: c.DNSProxy,

		// OpenConnect
		OpenConnect: c.OpenConnect,

		// Executables
		Executables: c.Executables,

		// SplitRouting
		SplitRouting: c.SplitRouting,

		// TrafficPolicing
		TrafficPolicing: &config.TrafficPolicing{
			AllowedHosts:      c.TrafficPolicing.AllowedHosts,
			PortalPorts:       c.TrafficPolicing.PortalPorts,
			FirewallMark:      c.TrafficPolicing.FirewallMark,
			ResolveTimeout:    c.TrafficPolicing.ResolveTimeout,
			ResolveTries:      c.TrafficPolicing.ResolveTries,
			ResolveTriesSleep: c.TrafficPolicing.ResolveTriesSleep,
			ResolveTimer:      c.TrafficPolicing.ResolveTimer,
			ResolveTTL:        c.TrafficPolicing.ResolveTTL,
		},

		// TND
		TND: c.TND,
	}
	return conf
}

func getVPNConfig(vpnconf *vpnconfig.Config) *config.VPNConfig {
	return &config.VPNConfig{
		Gateway: netip.MustParseAddr(vpnconf.Gateway.String()), // TODO: improve
		PID:     vpnconf.PID,
		Timeout: vpnconf.Timeout,
		// TODO: fix
	}
}

// NewConfig returns a new Config.
func NewConfig() *Config {
	return &Config{
		Config:  ConfigFile,
		Verbose: false,

		SocketServer:    config.NewSocketServer(),
		CPD:             config.NewCPD(),
		DNSProxy:        config.NewDNSProxy(),
		OpenConnect:     config.NewOpenConnect(),
		Executables:     config.NewExecutables(),
		SplitRouting:    config.NewSplitRouting(),
		TrafficPolicing: config.NewTrafficPolicing(),
		TND:             tnd.NewConfig(),
	}
}
