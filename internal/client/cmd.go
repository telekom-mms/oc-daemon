package client

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/daemon"
	"github.com/telekom-mms/oc-daemon/pkg/client"
)

var (
	// config is the OC client config
	config *client.Config

	// command line arguments
	command = ""

	// verbose specifies verbose output
	verbose = false

	// json specifies whether output should be formatted as json
	json = false
)

// clientUserConfig is client.UserConfig for testing.
var clientUserConfig = client.UserConfig

// saveConfig saves the user config to the user dir
func saveConfig() error {
	userConfig := clientUserConfig()
	userDir := filepath.Dir(userConfig)
	if err := os.MkdirAll(userDir, 0700); err != nil {
		return fmt.Errorf("Client could not create user dir: %w", err)
	}
	if err := config.Save(userConfig); err != nil {
		return fmt.Errorf("Client could not save config to file: %w", err)
	}
	return nil
}

// clientLoadUserSystemConfig is client.LoadUserSystemConfig for testing.
var clientLoadUserSystemConfig = client.LoadUserSystemConfig

// clientSystemConfig is client.SystemConfig for testing.
var clientSystemConfig = client.SystemConfig

// setConfig sets the config from config files and the command line
func setConfig(args []string) error {
	// status subcommand
	statusCmd := flag.NewFlagSet("status", flag.ContinueOnError)
	statusCmd.BoolVar(&verbose, "verbose", verbose, "set verbose output")
	statusCmd.BoolVar(&json, "json", json, "set json output")

	// define command line arguments
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	cfgFile := flags.String("config", "", "set config `file`")
	cert := flags.String("cert", "", "set client certificate `file` or "+
		"PKCS11 URI")
	key := flags.String("key", "", "set client key `file` or PKCS11 URI")
	ca := flags.String("ca", "", "set additional CA certificate `file`")
	profile := flags.String("profile", "", "set XML profile `file`")
	srv := flags.String("server", "", "set server `address`")
	usr := flags.String("user", "", "set `username`")
	sys := flags.Bool("system-settings", false, "use system settings "+
		"instead of user configuration")
	ver := flags.Bool("version", false, "print version")

	// set usage output
	flags.Usage = func() {
		cmd := os.Args[0]
		w := flags.Output()
		usage := func(f string, args ...interface{}) {
			_, err := fmt.Fprintf(w, f, args...)
			if err != nil {
				log.WithError(err).Fatal("Client could not print usage")
			}
		}
		usage("Usage:\n")
		usage("  %s [options] [command]\n", cmd)
		usage("\nOptions:\n")
		flags.PrintDefaults()
		usage("\nCommands:\n")
		usage("  connect\n")
		usage("        connect to the VPN (default)\n")
		usage("  disconnect\n")
		usage("        disconnect from the VPN\n")
		usage("  reconnect\n")
		usage("        reconnect to the VPN\n")
		usage("  list\n")
		usage("        list VPN servers in XML Profile\n")
		usage("  status\n")
		usage("        show VPN status\n")
		usage("  monitor\n")
		usage("        monitor VPN status updates\n")
		usage("  save\n")
		usage("        save current settings to user configuration\n")
		usage("\nExamples:\n")
		usage("  %s connect\n", cmd)
		usage("  %s disconnect\n", cmd)
		usage("  %s reconnect\n", cmd)
		usage("  %s status\n", cmd)
		usage("  %s list\n", cmd)
		usage("  %s -server \"My SSL VPN Server\" connect\n", cmd)
		usage("  %s -server \"My SSL VPN Server\" save\n", cmd)
		usage("  %s -user exampleuser connect\n", cmd)
		usage("  %s -user $USER save\n", cmd)
		usage("  %s -system-settings save\n", cmd)
	}

	// parse arguments
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	// print version?
	if *ver {
		fmt.Println(daemon.Version)
		return flag.ErrHelp
	}

	// parse subcommands
	command = flags.Arg(0)
	switch command {
	case "status":
		if err := statusCmd.Parse(args[2:]); err != nil {
			return err
		}
	}

	// set command
	command = flags.Arg(0)

	// set config
	if *cfgFile != "" {
		// load config from command line
		c, err := client.LoadConfig(*cfgFile)
		if err != nil {
			return fmt.Errorf("Client could not load config %s: %w", *cfgFile, err)
		}
		c.Expand()
		config = c
	} else {
		// load user or system configuration
		config = clientLoadUserSystemConfig()
		if config == nil {
			// fall back to default config
			log.Warn("Client could not load user or system config, using default config")
			config = client.NewConfig()
		}
	}

	// set client certificate
	if *cert != "" {
		config.ClientCertificate = *cert
	}

	// set client key
	if *key != "" {
		config.ClientKey = *key
	}

	// set ca certificate
	if *ca != "" {
		config.CACertificate = *ca
	}

	// set xml profile
	if *profile != "" {
		config.XMLProfile = *profile
	}

	// set vpn server
	if *srv != "" {
		config.VPNServer = *srv
	}

	// set username
	if *usr != "" {
		config.User = *usr
	}

	// reset to system settings
	if *sys {
		systemConfig := clientSystemConfig()
		c, err := client.LoadConfig(systemConfig)
		if err != nil {
			return fmt.Errorf("Client could not load system settings from system config %s: %w", systemConfig, err)
		}
		config = c
	}
	return nil
}

func run(args []string) error {
	// load configs and parse command line arguments
	if err := setConfig(args); err != nil {
		return err
	}

	// make sure config is not empty
	if !config.Valid() {
		log.Error("Client got invalid configuration. Make sure you " +
			"configure client certificate, client key and vpn " +
			"server as a minimum. See -help for command line " +
			"arguments")
		return errors.New("invalid configuration")
	}

	// handle command
	switch command {
	case "list":
		return listServers()
	case "", "connect":
		return connectVPN()
	case "disconnect":
		return disconnectVPN()
	case "reconnect":
		return reconnectVPN()
	case "status":
		return getStatus()
	case "monitor":
		return monitor()
	case "save":
		return saveConfig()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// Run is the main entry point of the oc client
func Run() {
	if err := run(os.Args); err != nil {
		if err != flag.ErrHelp {
			log.Fatal(err)
		}
		return
	}
}
