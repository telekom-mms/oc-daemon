package client

import (
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
)

// saveConfig saves the user config to the user dir
func saveConfig() {
	userConfig := client.UserConfig()
	userDir := filepath.Dir(userConfig)
	if err := os.MkdirAll(userDir, 0700); err != nil {
		log.WithError(err).Fatal("Client could not create user dir")
	}
	if err := config.Save(userConfig); err != nil {
		log.WithError(err).Fatal("Client could not save config to file")
	}
}

// setConfig sets the config from config files and the command line
func setConfig() {
	// define command line arguments
	cfgFile := flag.String("config", "", "set config `file`")
	cert := flag.String("cert", "", "set client certificate `file` or "+
		"PKCS11 URI")
	key := flag.String("key", "", "set client key `file` or PKCS11 URI")
	ca := flag.String("ca", "", "set additional CA certificate `file`")
	profile := flag.String("profile", "", "set XML profile `file`")
	srv := flag.String("server", "", "set server `address`")
	usr := flag.String("user", "", "set `username`")
	sys := flag.Bool("system-settings", false, "use system settings "+
		"instead of user configuration")
	ver := flag.Bool("version", false, "print version")

	// set usage output
	flag.Usage = func() {
		cmd := os.Args[0]
		w := flag.CommandLine.Output()
		usage := func(f string, args ...interface{}) {
			_, err := fmt.Fprintf(w, f, args...)
			if err != nil {
				log.WithError(err).Fatal("Client could not print usage")
			}
		}
		usage("Usage:\n")
		usage("  %s [options] [command]\n", cmd)
		usage("\nOptions:\n")
		flag.PrintDefaults()
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
	flag.Parse()

	// print version?
	if *ver {
		fmt.Println(daemon.Version)
		os.Exit(0)
	}

	// set command
	command = flag.Arg(0)

	// set config
	if *cfgFile != "" {
		// load config from command line
		c, err := client.LoadConfig(*cfgFile)
		if err != nil {
			log.WithError(err).WithField("config", *cfgFile).
				Fatal("Client could not load config")
		}
		c.Expand()
		config = c
	} else {
		// load user or system configuration
		config = client.LoadUserSystemConfig()
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
		systemConfig := client.SystemConfig()
		c, err := client.LoadConfig(systemConfig)
		if err != nil {
			log.WithField("systemConfig", systemConfig).
				WithError(err).
				Fatal("Client could not load system settings from system config")
		}
		config = c
	}
}

// Run is the main entry point of the oc client
func Run() {
	// load configs and parse command line arguments
	setConfig()

	// make sure config is not empty
	if !config.Valid() {
		log.Fatal("Client got invalid configuration. Make sure you " +
			"configure client certificate, client key and vpn " +
			"server as a minimum. See -help for command line " +
			"arguments")
	}

	// handle command
	switch command {
	case "list":
		listServers()
	case "", "connect":
		connectVPN()
	case "disconnect":
		disconnectVPN()
	case "reconnect":
		reconnectVPN()
	case "status":
		getStatus()
	case "monitor":
		monitor()
	case "save":
		saveConfig()
	default:
		log.Fatalf("unknown command: %s", command)
	}
}
