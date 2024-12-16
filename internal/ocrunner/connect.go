// Package ocrunner contains the openconnect runner.
package ocrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// ConnectEvent is a connect runner event.
type ConnectEvent struct {
	// Connect indicates connect and disconnect
	// TODO: use a Type with more values?
	Connect bool

	// PID is the process ID of the running openconnect process
	PID uint32

	// config is the daemon configuration.
	config *daemoncfg.Config

	// Env are extra environment variables set during execution
	env []string
}

// Connect is a openconnect connection runner.
type Connect struct {
	// openconnect command
	command *exec.Cmd

	// channel for openconnect exits
	exits chan struct{}

	// channels for commands from user
	commands chan *ConnectEvent
	done     chan struct{}
	closed   chan struct{}

	// channel for user facing events
	events chan *ConnectEvent
}

// function wrappers for testing.
var (
	userLookup      = user.Lookup
	userLookupGroup = user.LookupGroup
	osChown         = os.Chown
	osReadFile      = os.ReadFile
	osWriteFile     = os.WriteFile
	osFindProcess   = os.FindProcess
	processSignal   = func(process *os.Process, sig os.Signal) error {
		return process.Signal(sig)
	}
	execCommand = exec.Command
)

// sendEvent sends event over the event channel.
func (c *Connect) sendEvent(event *ConnectEvent) {
	select {
	case c.events <- event:
	case <-c.done:
	}
}

// setPIDOwner sets the owner of the pid file.
func (c *Connect) setPIDOwner(config *daemoncfg.Config) {
	if config.OpenConnect.PIDOwner == "" {
		// do not change owner
		return
	}

	user, err := userLookup(config.OpenConnect.PIDOwner)
	if err != nil {
		log.WithError(err).Error("OC-Runner could not get UID of pid file owner")
		return
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		log.WithError(err).Error("OC-Runner could not convert UID of pid file owner to int")
		return
	}

	if err := osChown(config.OpenConnect.PIDFile, uid, -1); err != nil {
		log.WithError(err).Error("OC-Runner could not change owner of pid file")
	}
}

// setPIDGroup sets the group of the pid file.
func (c *Connect) setPIDGroup(config *daemoncfg.Config) {
	if config.OpenConnect.PIDGroup == "" {
		// do not change group
		return
	}

	group, err := userLookupGroup(config.OpenConnect.PIDGroup)
	if err != nil {
		log.WithError(err).Error("OC-Runner could not get GID of pid file group")
		return
	}

	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		log.WithError(err).Error("OC-Runner could not convert GID of pid file group to int")
		return
	}

	if err := osChown(config.OpenConnect.PIDFile, -1, gid); err != nil {
		log.WithError(err).Error("OC-Runner could not change group of pid file")
	}
}

// savePidFile saves the running command to pid file.
func (c *Connect) savePidFile(config *daemoncfg.Config) {
	if c.command == nil || c.command.Process == nil {
		return
	}

	// get pid
	pid := fmt.Sprintf("%d\n", c.command.Process.Pid)

	// convert permissions
	perm, err := strconv.ParseUint(config.OpenConnect.PIDPermissions, 8, 32)
	if err != nil {
		log.WithError(err).Error("OC-Runner could not convert permissions of pid file to uint")
		return
	}

	// write pid to file with permissions
	err = osWriteFile(config.OpenConnect.PIDFile, []byte(pid), os.FileMode(perm))
	if err != nil {
		log.WithError(err).Error("OC-Runner writing pid error")
		return
	}

	// set owner and group
	c.setPIDOwner(config)
	c.setPIDGroup(config)
}

// getPID returns the PID of the running command.
func (c *Connect) getPID() uint32 {
	if c.command == nil || c.command.Process == nil || c.command.Process.Pid < 0 {
		return 0
	}
	return uint32(c.command.Process.Pid)
}

// handleConnect establishes the connection by starting openconnect.
func (c *Connect) handleConnect(e *ConnectEvent) {
	if c.command != nil {
		// command seems to be running, stop here
		log.WithField("error", "openconnect process already running").
			Error("OC-Runner connect error")
		return
	}

	// create openconnect command and
	// use login information from Authenticate():
	//
	// openconnect --cookie-on-stdin $HOST --servercert $FINGERPRINT
	//
	serverCert := fmt.Sprintf("--servercert=%s", e.config.LoginInfo.Fingerprint)
	xmlConfig := fmt.Sprintf("--xmlconfig=%s", e.config.OpenConnect.XMLProfile)
	script := fmt.Sprintf("--script=%s", e.config.OpenConnect.VPNCScript)
	host := e.config.LoginInfo.Host
	if e.config.LoginInfo.ConnectURL != "" {
		host = e.config.LoginInfo.ConnectURL
	}
	parameters := []string{
		xmlConfig,
		script,
		"--cookie-on-stdin",
		host,
		serverCert,
	}
	if e.config.OpenConnect.NoProxy {
		parameters = append(parameters, "--no-proxy")
	}
	if e.config.LoginInfo.Resolve != "" {
		resolve := fmt.Sprintf("--resolve=%s", e.config.LoginInfo.Resolve)
		parameters = append(parameters, resolve)
	}
	if e.config.OpenConnect.VPNDevice != "" {
		device := fmt.Sprintf("--interface=%s", e.config.OpenConnect.VPNDevice)
		parameters = append(parameters, device)
	}
	parameters = append(parameters, e.config.OpenConnect.ExtraArgs...)
	c.command = execCommand(e.config.OpenConnect.OpenConnect, parameters...)

	// run command in own process group so it is not canceled by interrupt
	// signal sent to daemon
	c.command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// run command, pass login info to stdin
	b := bytes.NewBufferString(e.config.LoginInfo.Cookie)
	c.command.Stdin = b
	c.command.Stdout = os.Stdout
	c.command.Stderr = os.Stderr
	c.command.Env = append(os.Environ(), e.config.OpenConnect.ExtraEnv...)
	c.command.Env = append(c.command.Env, e.env...)

	if err := c.command.Start(); err != nil {
		go func() {
			c.exits <- struct{}{}
		}()
		return
	}

	// save pid and cmd line
	c.savePidFile(e.config)

	// signal connect to user
	c.sendEvent(&ConnectEvent{
		Connect: true,
		PID:     c.getPID(),
	})

	// wait for program termination and signal disconnect
	go func() {
		if err := c.command.Wait(); err != nil {
			log.WithError(err).
				Error("OC-Runner waiting for connect termination error")
		}
		c.exits <- struct{}{}
	}()

}

// handleDisconnect tears down the connection by stopping openconnect.
func (c *Connect) handleDisconnect() {
	if c.command == nil || c.command.Process == nil {
		log.WithField("error", "no openconnect process running").
			Error("OC-Runner disconnect error")
		return
	}
	if err := processSignal(c.command.Process, os.Interrupt); err != nil {
		// TODO: handle failed signal?
		log.WithError(err).Error("OC-Runner sending interrupt for disconnect error")
	}
}

// handleOCExit handles openconnect program terminations.
func (c *Connect) handleOCExit() {
	// clear command
	c.command = nil

	// signal disconnect to user
	c.sendEvent(&ConnectEvent{})
}

// handleStop handles stopping the runner.
func (c *Connect) handleStop() {
	if c.command != nil {
		// TODO: is this ok or ugly?
		c.handleDisconnect()
		<-c.exits
		c.handleOCExit()
	}
}

// start starts the connect runner.
func (c *Connect) start() {
	defer close(c.closed)
	defer close(c.events)
	for {
		select {
		case cmd := <-c.commands:
			if cmd.Connect {
				c.handleConnect(cmd)
				break
			}
			c.handleDisconnect()

		case <-c.exits:
			c.handleOCExit()

		case <-c.done:
			c.handleStop()
			return
		}
	}
}

// Start starts the connect runner.
func (c *Connect) Start() {
	go c.start()
}

// Stop stops the connect runner.
func (c *Connect) Stop() {
	close(c.done)
	<-c.closed
}

// Connect connects the vpn by starting openconnect.
func (c *Connect) Connect(config *daemoncfg.Config, env []string) {
	e := &ConnectEvent{
		Connect: true,
		config:  config,
		env:     env,
	}
	c.commands <- e
}

// Disconnect disconnects the vpn by stopping openconnect.
func (c *Connect) Disconnect() {
	e := &ConnectEvent{}
	c.commands <- e
}

// Events returns the connect events channel.
func (c *Connect) Events() chan *ConnectEvent {
	return c.events
}

// NewConnect returns a new Connect.
func NewConnect() *Connect {
	return &Connect{
		exits: make(chan struct{}),

		commands: make(chan *ConnectEvent),
		done:     make(chan struct{}),
		closed:   make(chan struct{}),

		events: make(chan *ConnectEvent),
	}
}

// CleanupConnect cleans up connect after a failed shutdown.
func CleanupConnect(config *daemoncfg.OpenConnect) {
	// get pid from file
	b, err := osReadFile(config.PIDFile)
	if err != nil {
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return
	}

	// check if it is running and command line starts with openconnect
	cmdLine, err := osReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return
	}

	if !strings.HasPrefix(string(cmdLine), config.OpenConnect) {
		return
	}

	// find process and send interrupt signal
	process, err := osFindProcess(pid)
	if err != nil {
		return
	}

	if err := processSignal(process, os.Interrupt); err == nil {
		log.Warn("OC-Runner cleaned up process")
	}
}
