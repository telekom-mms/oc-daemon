package vpncscript

import (
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/daemon"
)

const (
	runDir     = "/run/oc-daemon"
	socketFile = runDir + "/daemon.sock"
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

	// parse environment variables
	e := parseEnvironment()

	// set verbosity from command line or environment
	if *verbose || e.verbose {
		log.SetLevel(log.DebugLevel)
	}

	// set socket file from environment
	socketFile := socketFile
	if e.socketFile != "" {
		socketFile = e.socketFile
	}

	printDebugEnvironment()
	log.WithField("env", e).Debug("VPNCScript parsed environment")

	// handle reason environment variable
	switch e.reason {
	case "pre-init":
		return
	case "connect", "disconnect":
		c := createConfigUpdate(e)
		log.WithField("update", c).Debug("VPNCScript created config update")
		runClient(socketFile, c)
	case "attempt-reconnect":
		return
	case "reconnect":
		return
	default:
		log.Fatal("VPNCScript called with unknown reason")
	}
}
