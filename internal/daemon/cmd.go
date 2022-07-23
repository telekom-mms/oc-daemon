package daemon

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"
)

const (
	// configDir is the directory for the configuration
	configDir = "/var/lib/oc-daemon"

	// xmlProfile is the AnyConnect Profile
	xmlProfile = configDir + "/profile.xml"

	// runDir is the directory for runtime files
	runDir = "/run/oc-daemon"

	// sockFile is the unix socket file
	sockFile = runDir + "/daemon.sock"

	// vpncScript is the vpnc-script
	vpncScript = "/usr/bin/oc-daemon-vpncscript"

	// vpnDevice is the vpn network device name
	vpnDevice = "oc-daemon-tun0"
)

var (
	// Version is the daemon version, to be set at compile time
	Version = "unknown"
)

// prepareFolders prepares directories used by the daemon
func prepareFolders() {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.WithError(err).Fatal("Daemon could not create config dir")
	}
	if err := os.MkdirAll(runDir, 0755); err != nil {
		log.WithError(err).Fatal("Daemon could not create run dir")
	}
}

// Run is the main entry point for the daemon
func Run() {
	// parse command line arguments
	verbose := flag.Bool("verbose", false, "enable verbose output")
	version := flag.Bool("version", false, "print version")
	flag.Parse()

	// print version?
	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	// set verbose log level
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	// prepare directories
	prepareFolders()

	// start daemon
	daemon := NewDaemon()
	daemon.Start()

	// catch interrupt and clean up
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	daemon.Stop()
}
