package daemon

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

var (
	// Version is the daemon version, to be set at compile time
	Version = "unknown"
)

// command line argument names
const (
	argConfig  = "config"
	argVerbose = "verbose"
	argVersion = "version"
)

// prepareFolders prepares directories used by the daemon
func prepareFolders(config *Config) {
	for _, file := range []string{
		config.Config,
		config.SocketServer.SocketFile,
		config.OpenConnect.XMLProfile,
		config.OpenConnect.PIDFile,
	} {
		dir := filepath.Dir(file)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.WithError(err).WithField("dir", dir).
				Fatal("Daemon could not create dir")
		}
	}
}

// flagIsSet returns whether flag with name is set as command line argument
func flagIsSet(name string) bool {
	isSet := false
	flag.Visit(func(f *flag.Flag) {
		if name == f.Name {
			isSet = true
		}
	})
	return isSet
}

// Run is the main entry point for the daemon
func Run() {
	// parse command line arguments
	defaults := NewConfig()
	cfgFile := flag.String(argConfig, defaults.Config, "set config `file`")
	verbose := flag.Bool(argVerbose, defaults.Verbose, "enable verbose output")
	version := flag.Bool(argVersion, false, "print version")
	flag.Parse()

	// print version?
	if *version {
		fmt.Println(Version)
		os.Exit(0)
	}

	// load config
	config := NewConfig()
	if flagIsSet(argConfig) {
		config.Config = *cfgFile
	}
	if err := config.Load(); err != nil {
		log.WithError(err).Warn("Daemon could not load config, using default config")
	}
	if !config.Valid() {
		config = NewConfig()
		log.Warn("Daemon loaded invalid config, using default config")
	}

	// overwrite config settings with command line arguments
	if flagIsSet(argVerbose) {
		config.Verbose = *verbose
	}

	// set verbose log level
	if config.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	// prepare directories
	prepareFolders(config)

	// start daemon
	daemon := NewDaemon(config)
	daemon.Start()

	// catch interrupt and clean up
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	daemon.Stop()
}
