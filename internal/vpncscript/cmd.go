package vpncscript

import (
	"flag"
	"fmt"
	"os"

	"github.com/T-Systems-MMS/oc-daemon/internal/daemon"
	log "github.com/sirupsen/logrus"
)

// Run is the main entry point of vpnc script
func Run() {
	// parse command line
	verbose := flag.Bool("verbose", false, "enable verbose output")
	version := flag.Bool("version", false, "print version")
	flag.Parse()

	// print version?
	if *version {
		fmt.Println(daemon.Version)
		os.Exit(0)
	}

	// set verbose output
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	// parse environment variables
	e := parseEnvironment()
	printDebugEnvironment()
	log.WithField("env", e).Debug("VPNCScript parsed environment")

	// handle reason environment variable
	switch e.reason {
	case "pre-init":
		return
	case "connect", "disconnect":
		c := createConfigUpdate(e)
		log.WithField("update", c).Debug("VPNCScript created config update")
		runClient(c)
	case "attempt-reconnect":
		return
	case "reconnect":
		return
	default:
		log.Fatal("VPNCScript called with unknown reason")
	}
}
