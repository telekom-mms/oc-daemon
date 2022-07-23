package client

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/T-Systems-MMS/oc-daemon/internal/daemon"
	"github.com/T-Systems-MMS/oc-daemon/internal/xmlprofile"
	log "github.com/sirupsen/logrus"
)

var (
	// dirNams it the directory name appended to path names
	dirName = "/oc-daemon"

	// userDir is the directory for the user config
	userDir = ""

	// userConfig is the user config file
	userConfig = ""

	// systemDir is the system-wide config dir
	systemDir = "/var/lib" + dirName

	// configFile is the config file name appended to path names
	configFile = "/oc-client.json"

	// systemConfig is the system-wide config file
	systemConfig = systemDir + configFile

	// xmlProfile is the anyconnect xml profile
	xmlProfile = systemDir + "/profile.xml"

	// config is the OC client config
	config *ClientConfig

	// command line arguments
	command = ""
)

// prepareFolder prepares the user directory
func prepareFolder() {
	c, err := os.UserConfigDir()
	if err != nil {
		log.WithError(err).Fatal("Client could not get user config dir")
	}
	userDir = c + dirName
	if err := os.MkdirAll(userDir, 0700); err != nil {
		log.WithError(err).Fatal("Client could not create user dir")
	}

	userConfig = userDir + configFile
}

// expandPath expands tilde and environment variables in path
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		path = filepath.Join("$HOME", path[1:])
	}
	return os.ExpandEnv(path)
}

// expandPaths expands the paths in config
func expandPaths() {
	config.ClientCertificate = expandPath(config.ClientCertificate)
	config.ClientKey = expandPath(config.ClientKey)
	config.CACertificate = expandPath(config.CACertificate)
}

// expandUser expands the username in config
func expandUser() {
	config.User = os.ExpandEnv(config.User)
}

// expandConfig expands variables in config
func expandConfig() {
	expandPaths()
	expandUser()
}

// loadConfig loads the user config from the config file
func loadConfig() {
	// try user config
	config = loadClientConfig(userConfig)
	if config != nil {
		return
	}

	// try system config
	config = loadClientConfig(systemConfig)
	if config != nil {
		return
	}

	// return new config
	config = newClientConfig()
}

// saveConfig saves the user config to the user dir
func saveConfig() {
	config.save(userConfig)
}

// parseCommandLine parses the command line
func parseCommandLine() {
	// define command line arguments
	cert := flag.String("cert", "", "set client certificate `file` or "+
		"PKCS11 URI")
	key := flag.String("key", "", "set client key `file` or PKCS11 URI")
	ca := flag.String("ca", "", "set additional CA certificate `file`")
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
		config = loadClientConfig(systemConfig)
		if config == nil {
			log.WithField("systemConfig", systemConfig).
				Fatal("Client could not load system settings from system config")
		}
	}
}

// listServers lists the (d)tls servers in the xml profile
func listServers() {
	p := xmlprofile.NewXMLProfile(xmlProfile)
	p.Parse()
	for _, s := range p.GetVPNServerHostNames() {
		log.Printf("Server: %#v", s)
	}
}

// Run is the main entry point of the oc client
func Run() {
	// prepare directory
	prepareFolder()

	// load configuration
	loadConfig()

	// parse command line arguments
	parseCommandLine()

	// make sure config is not empty
	if config.empty() {
		log.Fatal("Cannot run with empty configuration. You need to " +
			"configure client certificate, client key and vpn " +
			"server first. See -help for command line arguments")
	}

	// handle command
	switch command {
	case "list":
		listServers()
	case "", "connect":
		expandConfig()
		connectVPN()
	case "disconnect":
		disconnectVPN()
	case "reconnect":
		expandConfig()
		reconnectVPN()
	case "status":
		getStatus()
	case "save":
		saveConfig()
	default:
		log.Fatalf("unknown command: %s", command)
	}
}
