package ocrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
)

const (
	// pidFile is the pid file for openconnect
	pidFile = "/run/oc-daemon/openconnect.pid"
)

// ConnectEvent is a connect runner event
type ConnectEvent struct {
	// Connect indicates connect and disconnect
	// TODO: use a Type with more values?
	Connect bool

	// login info for connect
	login *logininfo.LoginInfo

	// Env are extra environment variables set during execution
	env []string
}

// Connect is a openconnect connection runner
type Connect struct {
	// openconnect command
	command *exec.Cmd

	// channel for openconnect exits
	exits chan struct{}

	// xml profile and vpnc-script paths
	profile string
	script  string

	// tunnel device name
	device string

	// channels for commands from user
	commands chan *ConnectEvent
	done     chan struct{}

	// channel for user facing events
	events chan *ConnectEvent
}

// savePidFile saves the running command to pid file
func (c *Connect) savePidFile() {
	if c.command == nil || c.command.Process == nil {
		return
	}
	pid := fmt.Sprintf("%d\n", c.command.Process.Pid)
	err := os.WriteFile(pidFile, []byte(pid), 0600)
	if err != nil {
		log.WithError(err).Error("OC-Runner writing pid error")
	}
}

// handleConnect establishes the connection by starting openconnect
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
	serverCert := fmt.Sprintf("--servercert=%s", e.login.Fingerprint)
	xmlConfig := fmt.Sprintf("--xmlconfig=%s", c.profile)
	script := fmt.Sprintf("--script=%s", c.script)
	host := e.login.Host
	if e.login.ConnectURL != "" {
		host = e.login.ConnectURL
	}
	parameters := []string{
		xmlConfig,
		script,
		"--cookie-on-stdin",
		host,
		serverCert,
		"--no-proxy",
	}
	if e.login.Resolve != "" {
		resolve := fmt.Sprintf("--resolve=%s", e.login.Resolve)
		parameters = append(parameters, resolve)
	}
	if c.device != "" {
		device := fmt.Sprintf("--interface=%s", c.device)
		parameters = append(parameters, device)
	}
	c.command = exec.Command("openconnect", parameters...)

	// run command, pass login info to stdin
	b := bytes.NewBufferString(e.login.Cookie)
	c.command.Stdin = b
	c.command.Stdout = os.Stdout
	c.command.Stderr = os.Stderr
	c.command.Env = append(os.Environ(), e.env...)

	if err := c.command.Start(); err != nil {
		log.WithError(err).Error("OC-Runner executing connect error")
		c.exits <- struct{}{}
		return
	}

	// save pid and cmd line
	c.savePidFile()

	// signal connect to user
	c.events <- &ConnectEvent{
		Connect: true,
	}

	// wait for program termination and signal disconnect
	go func() {
		if err := c.command.Wait(); err != nil {
			log.WithError(err).
				Error("OC-Runner waiting for connect termination error")
		}
		c.exits <- struct{}{}
	}()

}

// handleDisconnect tears down the connection by stopping openconnect
func (c *Connect) handleDisconnect() {
	if c.command == nil || c.command.Process == nil {
		log.WithField("error", "no openconnect process running").
			Error("OC-Runner disconnect error")
		return
	}
	if err := c.command.Process.Signal(os.Interrupt); err != nil {
		// TODO: handle failed signal?
		log.WithError(err).Error("OC-Runner sending interrupt for disconnect error")
	}
}

// handleOCExit handles openconnect program terminations
func (c *Connect) handleOCExit() {
	// clear command
	c.command = nil

	// signal disconnect to user
	c.events <- &ConnectEvent{}
}

// handleStop handles stopping the runner
func (c *Connect) handleStop() {
	if c.command != nil {
		// TODO: is this ok or ugly?
		c.handleDisconnect()
		<-c.exits
		c.handleOCExit()
	}
}

// start starts the connect runner
func (c *Connect) start() {
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

// Start starts the connect runner
func (c *Connect) Start() {
	go c.start()
}

// Stop stops the connect runner
func (c *Connect) Stop() {
	close(c.done)
	for range c.events {
		// wait for event channel close
	}
}

// Connect connects the vpn by starting openconnect
func (c *Connect) Connect(login *logininfo.LoginInfo, env []string) {
	e := &ConnectEvent{
		Connect: true,
		login:   login,
		env:     env,
	}
	c.commands <- e
}

// Disconnect disconnects the vpn by stopping openconnect
func (c *Connect) Disconnect() {
	e := &ConnectEvent{}
	c.commands <- e
}

// Events returns the connect events channel
func (c *Connect) Events() chan *ConnectEvent {
	return c.events
}

// NewConnect returns a new Connect
func NewConnect(xmlProfile, vpncScript, device string) *Connect {
	return &Connect{
		profile: xmlProfile,
		script:  vpncScript,
		device:  device,

		exits: make(chan struct{}),

		commands: make(chan *ConnectEvent),
		done:     make(chan struct{}),

		events: make(chan *ConnectEvent),
	}
}

// CleanupConnect cleans up connect after a failed shutdown
func CleanupConnect() {
	// get pid from file
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return
	}

	// check if it is running and command line starts with openconnect
	cmdLine, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return
	}

	if !strings.HasPrefix(string(cmdLine), "openconnect") {
		return
	}

	// find process and send interrupt signal
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}

	if err := process.Signal(os.Interrupt); err == nil {
		log.Warn("OC-Runner cleaned up process")
	}
}
