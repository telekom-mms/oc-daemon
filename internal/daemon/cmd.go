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
	// Version is the daemon version, to be set at compile time.
	Version = "unknown"
)

// command line argument names.
const (
	argConfig  = "config"
	argVerbose = "verbose"
	argVersion = "version"
)

// osMkdirAll is os.MkdirAll for testing.
var osMkdirAll = os.MkdirAll

// prepareFolders prepares directories used by the daemon.
func prepareFolders(config *Config) error {
	for _, file := range []string{
		config.Config,
		config.SocketServer.SocketFile,
		config.OpenConnect.XMLProfile,
		config.OpenConnect.PIDFile,
	} {
		dir := filepath.Dir(file)
		if err := osMkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("Daemon could not create dir %s: %w", dir, err)
		}
	}

	return nil
}

// flagIsSet returns whether flag with name is set as command line argument.
func flagIsSet(flags *flag.FlagSet, name string) bool {
	isSet := false
	flags.Visit(func(f *flag.Flag) {
		if name == f.Name {
			isSet = true
		}
	})
	return isSet
}

// run is the main entry point for the daemon.
func run(args []string) error {
	// parse command line arguments
	defaults := NewConfig()
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	cfgFile := flags.String(argConfig, defaults.Config, "set config `file`")
	verbose := flags.Bool(argVerbose, defaults.Verbose, "enable verbose output")
	version := flags.Bool(argVersion, false, "print version")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	// print version?
	if *version {
		fmt.Println(Version)
		return nil
	}

	// log version
	log.WithField("version", Version).Info("Starting Daemon")

	// load config
	config := NewConfig()
	if flagIsSet(flags, argConfig) {
		config.Config = *cfgFile
	}
	if err := config.Load(); err != nil {
		log.WithError(err).Warn("Daemon could not load config, using default config")
	}
	if !config.Valid() {
		config = NewConfig()
		log.Warn("Daemon loaded invalid config, using default config")
	}

	// check executables
	if err := config.Executables.CheckExecutables(); err != nil {
		return fmt.Errorf("Daemon could not find all executables: %w", err)
	}

	// overwrite config settings with command line arguments
	if flagIsSet(flags, argVerbose) {
		config.Verbose = *verbose
	}

	// set verbose log level
	if config.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	// prepare directories
	if err := prepareFolders(config); err != nil {
		return err
	}

	// start daemon
	daemon := NewDaemon(config)
	if err := daemon.Start(); err != nil {
		return err
	}
	defer daemon.Stop()

	// catch interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// wait for interrupt signal or daemon error
	var err error
	select {
	case <-c:
	case err = <-daemon.Errors():
	}

	return err
}

// Run is the main entry point for the daemon.
func Run() {
	if err := run(os.Args); err != nil {
		if err != flag.ErrHelp {
			log.Fatal(err)
		}
		return
	}
}
