package vpncscript

import (
	"errors"
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

func run(args []string) error {
	// parse command line
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	verbose := flags.Bool("verbose", false, "enable verbose output")
	version := flags.Bool("version", false, "print version")

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	// print version?
	if *version {
		fmt.Println(daemon.Version)
		return nil
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
		return nil
	case "connect", "disconnect":
		c, err := createConfigUpdate(e)
		if err != nil {
			return fmt.Errorf("VPNCScript could not create config update: %w", err)
		}
		log.WithField("update", c).Debug("VPNCScript created config update")
		return runClient(socketFile, c)
	case "attempt-reconnect":
		return nil
	case "reconnect":
		return nil
	default:
		return errors.New("VPNCScript called with unknown reason")
	}
}

// Run is the main entry point of vpnc script
func Run() {
	if err := run(os.Args); err != nil {
		if err != flag.ErrHelp {
			log.Fatal(err)
		}
		return
	}
}
