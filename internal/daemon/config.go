package daemon

import (
	"encoding/json"
	"os"

	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/cpd"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
	"github.com/telekom-mms/oc-daemon/internal/execs"
	"github.com/telekom-mms/oc-daemon/internal/ocrunner"
	"github.com/telekom-mms/oc-daemon/internal/splitrt"
	"github.com/telekom-mms/oc-daemon/internal/trafpol"
	"github.com/telekom-mms/tnd/pkg/tnd"
)

var (
	// configDir is the directory for the configuration
	configDir = "/var/lib/oc-daemon"

	// ConfigFile is the default config file
	ConfigFile = configDir + "/oc-daemon.json"

	// DefaultDNSServer is the default DNS server address, i.e., listen
	// address of systemd-resolved
	DefaultDNSServer = "127.0.0.53:53"
)

// Config is an OC-Daemon configuration
type Config struct {
	Config  string `json:"-"`
	Verbose bool

	SocketServer    *api.Config
	CPD             *cpd.Config
	DNSProxy        *dnsproxy.Config
	OpenConnect     *ocrunner.Config
	Executables     *execs.Config
	SplitRouting    *splitrt.Config
	TrafficPolicing *trafpol.Config
	TND             *tnd.Config
}

// Valid returns whether config is valid
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

// Load loads the configuration from the config file
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

// NewConfig returns a new Config
func NewConfig() *Config {
	return &Config{
		Config:  ConfigFile,
		Verbose: false,

		SocketServer:    api.NewConfig(),
		CPD:             cpd.NewConfig(),
		DNSProxy:        dnsproxy.NewConfig(),
		OpenConnect:     ocrunner.NewConfig(),
		Executables:     execs.NewConfig(),
		SplitRouting:    splitrt.NewConfig(),
		TrafficPolicing: trafpol.NewConfig(),
		TND:             tnd.NewConfig(),
	}
}
