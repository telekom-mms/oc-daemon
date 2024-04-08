// Package daemon contains the OC-Daemon.
package daemon

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
	"github.com/telekom-mms/oc-daemon/internal/execs"
	"github.com/telekom-mms/oc-daemon/internal/ocrunner"
	"github.com/telekom-mms/oc-daemon/internal/profilemon"
	"github.com/telekom-mms/oc-daemon/internal/sleepmon"
	"github.com/telekom-mms/oc-daemon/internal/trafpol"
	"github.com/telekom-mms/oc-daemon/internal/vpnsetup"
	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
	"github.com/telekom-mms/oc-daemon/pkg/xmlprofile"
	"github.com/telekom-mms/tnd/pkg/tnd"
	"golang.org/x/sys/unix"
)

// Daemon is used to run the daemon.
type Daemon struct {
	config *Config

	server *api.Server
	dbus   *dbusapi.Service

	tnd tnd.TND

	vpnsetup *vpnsetup.VPNSetup
	trafpol  *trafpol.TrafPol

	sleepmon *sleepmon.SleepMon

	status *vpnstatus.Status

	runner *ocrunner.Connect

	// token is used for client authentication
	token string

	// channel for errors
	errors chan error

	// channels for shutdown
	done   chan struct{}
	closed chan struct{}

	// profile is the xml profile
	profile *xmlprofile.Profile
	profmon *profilemon.ProfileMon

	// disableTrafPol determines if traffic policing should be disabled,
	// overrides other traffic policing settings
	disableTrafPol bool
}

// setStatusTrustedNetwork sets the trusted network status in status.
func (d *Daemon) setStatusTrustedNetwork(trusted bool) {
	// convert bool to trusted network status
	trustedNetwork := vpnstatus.TrustedNetworkNotTrusted
	if trusted {
		trustedNetwork = vpnstatus.TrustedNetworkTrusted
	}

	// check status change
	if d.status.TrustedNetwork == trustedNetwork {
		// status not changed
		return
	}

	// status changed
	d.status.TrustedNetwork = trustedNetwork
	d.dbus.SetProperty(dbusapi.PropertyTrustedNetwork, trustedNetwork)
}

// setStatusConnectionState sets the connection state in status.
func (d *Daemon) setStatusConnectionState(connectionState vpnstatus.ConnectionState) {
	if d.status.ConnectionState == connectionState {
		// state not changed
		return
	}

	// state changed
	d.status.ConnectionState = connectionState
	d.dbus.SetProperty(dbusapi.PropertyConnectionState, connectionState)
}

// setStatusIP sets the IP in status.
func (d *Daemon) setStatusIP(ip string) {
	if d.status.IP == ip {
		// ip not changed
		return
	}

	// ip changed
	d.status.IP = ip
	d.dbus.SetProperty(dbusapi.PropertyIP, ip)
}

// setStatusDevice sets the device in status.
func (d *Daemon) setStatusDevice(device string) {
	if d.status.Device == device {
		// device not changed
		return
	}

	// device changed
	d.status.Device = device
	d.dbus.SetProperty(dbusapi.PropertyDevice, device)
}

// setStatusServer sets the current server in status.
func (d *Daemon) setStatusServer(server string) {
	if d.status.Server == server {
		// connected server not changed
		return
	}

	// connected server changed
	d.status.Server = server
	d.dbus.SetProperty(dbusapi.PropertyServer, server)
}

// setStatusConnectedAt sets the connection time in status.
func (d *Daemon) setStatusConnectedAt(connectedAt int64) {
	if d.status.ConnectedAt == connectedAt {
		// connection time not changed
		return
	}

	// connection time changed
	d.status.ConnectedAt = connectedAt
	d.dbus.SetProperty(dbusapi.PropertyConnectedAt, connectedAt)
}

// setStatusServers sets the vpn servers in status.
func (d *Daemon) setStatusServers(servers []string) {
	if reflect.DeepEqual(d.status.Servers, servers) {
		// servers not changed
		return
	}

	// servers changed
	d.status.Servers = servers
	d.dbus.SetProperty(dbusapi.PropertyServers, servers)
}

// setStatusOCRunning sets the openconnect running state in status.
func (d *Daemon) setStatusOCRunning(running bool) {
	ocrunning := vpnstatus.OCRunningNotRunning
	if running {
		ocrunning = vpnstatus.OCRunningRunning
	}
	if d.status.OCRunning == ocrunning {
		// OC running state not changed
		return
	}

	// OC running state changed
	d.status.OCRunning = ocrunning
	d.dbus.SetProperty(dbusapi.PropertyOCRunning, ocrunning)
}

// setStatusVPNConfig sets the VPN config in status.
func (d *Daemon) setStatusVPNConfig(config *vpnconfig.Config) {
	if d.status.VPNConfig.Equal(config) {
		// config not changed
		return
	}

	// config changed
	d.status.VPNConfig = config

	if config == nil {
		// remove config
		d.dbus.SetProperty(dbusapi.PropertyVPNConfig, dbusapi.VPNConfigInvalid)
		return
	}

	// update json config
	b, err := config.JSON()
	if err != nil {
		log.WithError(err).Error("Daemon could not convert status to JSON")
		d.dbus.SetProperty(dbusapi.PropertyVPNConfig, dbusapi.VPNConfigInvalid)
		return
	}
	d.dbus.SetProperty(dbusapi.PropertyVPNConfig, string(b))
}

// connectVPN connects to the VPN using login info from client request.
func (d *Daemon) connectVPN(login *logininfo.LoginInfo) {
	// allow only one connection
	if d.status.OCRunning.Running() {
		return
	}

	// ignore invalid login information
	if !login.Valid() {
		return
	}

	// update status
	d.setStatusOCRunning(true)
	d.setStatusServer(login.Server)
	d.setStatusConnectionState(vpnstatus.ConnectionStateConnecting)

	// connect using runner
	env := []string{
		"oc_daemon_token=" + d.token,
		"oc_daemon_socket_file=" + d.config.SocketServer.SocketFile,
		"oc_daemon_verbose=" + strconv.FormatBool(d.config.Verbose),
	}
	d.runner.Connect(login, env)
}

// disconnectVPN disconnects from the VPN.
func (d *Daemon) disconnectVPN() {
	// update status
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnecting)
	d.setStatusOCRunning(false)

	// stop runner
	if d.runner == nil {
		return
	}
	d.runner.Disconnect()
}

// updateVPNConfigUp updates the VPN config for VPN connect.
func (d *Daemon) updateVPNConfigUp(config *vpnconfig.Config) {
	// check if old and new config differ
	if config.Equal(d.status.VPNConfig) {
		log.WithField("error", "old and new vpn configs are equal").
			Error("Daemon config up error")
		return
	}

	// check if vpn is flagged as running
	if !d.status.OCRunning.Running() {
		log.WithField("error", "vpn not running").
			Error("Daemon config up error")
		return
	}

	// check if we are already connected
	if d.status.ConnectionState.Connected() {
		log.WithField("error", "vpn already connected").
			Error("Daemon config up error")
		return
	}

	// connecting, set up configuration
	log.Info("Daemon setting up vpn configuration")
	d.vpnsetup.Setup(config)

	// set traffic policing setting from Disable Always On VPN setting
	// in configuration
	d.disableTrafPol = config.Flags.DisableAlwaysOnVPN

	// save config
	d.setStatusVPNConfig(config)
	ip := ""
	for _, addr := range []net.IP{config.IPv4.Address, config.IPv6.Address} {
		// this assumes either a single IPv4 or a single IPv6 address
		// is configured on a vpn device
		if addr != nil {
			ip = addr.String()
		}
	}
	d.setStatusIP(ip)
	d.setStatusDevice(config.Device.Name)
}

// updateVPNConfigDown updates the VPN config for VPN disconnect.
func (d *Daemon) updateVPNConfigDown() {
	// TODO: only call this from Runner Event only and remove down message?
	// or potentially calling this twice is better than not at all?

	// check if vpn is still flagged as running
	if d.status.OCRunning.Running() {
		log.WithField("error", "vpn still running").
			Error("Daemon config down error")
		return
	}

	// check if vpn is still flagged as connected
	if d.status.ConnectionState.Connected() {
		log.WithField("error", "vpn still connected").
			Error("Daemon config down error")
		return
	}

	// disconnecting, tear down configuration
	log.Info("Daemon tearing down vpn configuration")
	if d.status.VPNConfig != nil {
		d.vpnsetup.Teardown(d.status.VPNConfig)
	}

	// save config
	d.setStatusVPNConfig(nil)
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnected)
	d.setStatusServer("")
	d.setStatusConnectedAt(0)
	d.setStatusIP("")
	d.setStatusDevice("")
}

// updateVPNConfig updates the VPN config with config update in client request.
func (d *Daemon) updateVPNConfig(request *api.Request) {
	// parse config
	configUpdate, err := VPNConfigUpdateFromJSON(request.Data())
	if err != nil {
		log.WithError(err).Error("Daemon could not parse config update from JSON")
		request.Error("invalid config update message")
		return
	}

	// check if config update is valid
	if !configUpdate.Valid() {
		log.Error("Daemon got invalid vpn config update")
		request.Error("invalid config update in config update message")
		return
	}

	// check token
	if configUpdate.Token != d.token {
		log.Error("Daemon got invalid token in vpn config update")
		request.Error("invalid token in config update message")
		return
	}

	// handle config update for vpn (dis)connect
	if configUpdate.Reason == "disconnect" {
		d.updateVPNConfigDown()
		return
	}
	d.updateVPNConfigUp(configUpdate.Config)
}

// handleClientRequest handles a client request.
func (d *Daemon) handleClientRequest(request *api.Request) {
	defer request.Close()
	log.Debug("Daemon handling client request")

	switch request.Type() {
	case api.TypeVPNConfigUpdate:
		// update VPN config
		d.updateVPNConfig(request)
	}
}

// handleDBusRequest handles a D-Bus API client request.
func (d *Daemon) handleDBusRequest(request *dbusapi.Request) {
	defer request.Close()
	log.Debug("Daemon handling D-Bus client request")

	switch request.Name {
	case dbusapi.RequestConnect:
		// create login info
		server := request.Parameters[0].(string)
		cookie := request.Parameters[1].(string)
		host := request.Parameters[2].(string)
		connectURL := request.Parameters[3].(string)
		fingerprint := request.Parameters[4].(string)
		resolve := request.Parameters[5].(string)

		login := &logininfo.LoginInfo{
			Server:      server,
			Cookie:      cookie,
			Host:        host,
			ConnectURL:  connectURL,
			Fingerprint: fingerprint,
			Resolve:     resolve,
		}

		// connect VPN
		d.connectVPN(login)

	case dbusapi.RequestDisconnect:
		// diconnect VPN
		d.disconnectVPN()
	}
}

// checkDisconnectVPN checks if we need to disconnect the VPN when handling a
// TND result.
func (d *Daemon) checkDisconnectVPN() {
	if d.status.TrustedNetwork.Trusted() && d.status.OCRunning.Running() {
		// disconnect VPN when switching from untrusted network with
		// active VPN connection to a trusted network
		log.Info("Daemon detected trusted network, disconnecting VPN connection")
		d.disconnectVPN()
	}
}

// handleTNDResult handles a TND result.
func (d *Daemon) handleTNDResult(trusted bool) error {
	log.WithField("trusted", trusted).Debug("Daemon handling TND result")
	d.setStatusTrustedNetwork(trusted)
	d.checkDisconnectVPN()
	return d.checkTrafPol()
}

// handleRunnerDisconnect handles a disconnect event from the OC runner,
// cleaning up everthing. This is also called when stopping the daemon.
func (d *Daemon) handleRunnerDisconnect() {
	// make sure running and connected are not set
	d.setStatusOCRunning(false)
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnected)
	d.setStatusServer("")
	d.setStatusConnectedAt(0)

	// make sure the vpn config is not active any more
	d.updateVPNConfigDown()
}

// handleRunnerEvent handles a connect event from the OC runner.
func (d *Daemon) handleRunnerEvent(e *ocrunner.ConnectEvent) {
	log.WithField("event", e).Debug("Daemon handling Runner event")

	if e.Connect {
		// make sure running is set
		d.setStatusOCRunning(true)
		return
	}

	// clean up after disconnect
	d.handleRunnerDisconnect()
}

// handleVPNSetupEvent handles a VPN setup event.
func (d *Daemon) handleVPNSetupEvent(event *vpnsetup.Event) {
	switch event.Type {
	case vpnsetup.EventSetupOK:
		d.setStatusConnectionState(vpnstatus.ConnectionStateConnected)
		d.setStatusConnectedAt(time.Now().Unix())
		log.Info("Daemon configured VPN connection")
	case vpnsetup.EventTeardownOK:
		log.Info("Daemon unconfigured VPN connection")
	}
}

// handleSleepMonEvent handles a suspend/resume event from SleepMon.
func (d *Daemon) handleSleepMonEvent(sleep bool) {
	log.WithField("sleep", sleep).Debug("Daemon handling SleepMon event")

	// disconnect vpn on resume
	if !sleep && d.status.OCRunning.Running() {
		d.disconnectVPN()
	}
}

// readXMLProfile reads the XML profile from file.
func readXMLProfile(xmlProfile string) *xmlprofile.Profile {
	profile, err := xmlprofile.LoadProfile(xmlProfile)
	if err != nil {
		// invalid config, use empty config
		log.WithError(err).Error("Could not read XML profile")
		profile = xmlprofile.NewProfile()
	}
	return profile
}

// handleProfileUpdate handles a xml profile update.
func (d *Daemon) handleProfileUpdate() error {
	log.Debug("Daemon handling XML profile update")
	d.profile = readXMLProfile(d.config.OpenConnect.XMLProfile)
	d.stopTND()
	d.stopTrafPol()
	if err := d.checkTrafPol(); err != nil {
		return err
	}
	if err := d.checkTND(); err != nil {
		return err
	}
	d.setStatusServers(d.profile.GetVPNServerHostNames())
	return nil
}

// cleanup cleans up after a failed shutdown.
func (d *Daemon) cleanup(ctx context.Context) {
	ocrunner.CleanupConnect(d.config.OpenConnect)
	vpnsetup.Cleanup(ctx, d.config.OpenConnect.VPNDevice, d.config.SplitRouting)
	trafpol.Cleanup(ctx)
}

// initToken creates the daemon token for client authentication.
func (d *Daemon) initToken() error {
	// TODO: is this good enough for us?
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	d.token = base64.RawURLEncoding.EncodeToString(b)
	return nil
}

// getProfileAllowedHosts returns the allowed hosts.
func (d *Daemon) getProfileAllowedHosts() (hosts []string) {
	// add vpn servers to allowed hosts
	hosts = append(hosts, d.profile.GetVPNServers()...)

	// add tnd servers to allowed hosts
	hosts = append(hosts, d.profile.GetTNDServers()...)

	// add allowed hosts from xml profile to allowed hosts
	hosts = append(hosts, d.profile.GetAllowedHosts()...)

	return
}

// initTNDServers sets the TND servers from the xml profile.
func (d *Daemon) initTNDServers() {
	servers := d.profile.GetTNDHTTPSServers()
	d.tnd.SetServers(servers)
}

// setTNDDialer sets a custom dialer for TND.
func (d *Daemon) setTNDDialer() {
	// get mark to be set on socket
	mark, err := strconv.Atoi(d.config.SplitRouting.FirewallMark)
	if err != nil {
		log.WithError(err).Error("Daemon could not convert FWMark to int")
		return
	}

	// control function that sets socket option on raw connection
	control := func(_, _ string, c syscall.RawConn) error {
		// set socket option function for setting mark with SO_MARK
		var soerr error
		setsockopt := func(fd uintptr) {
			soerr = unix.SetsockoptInt(
				int(fd),
				unix.SOL_SOCKET,
				unix.SO_MARK,
				mark,
			)
			if soerr != nil {
				log.WithError(soerr).Error("TND could not set SO_MARK")
			}
		}

		if err := c.Control(setsockopt); err != nil {
			return err
		}
		return soerr
	}

	// create and set dialer
	dialer := &net.Dialer{
		Control: control,
	}
	d.tnd.SetDialer(dialer)
}

// startTND starts TND if it's not running.
func (d *Daemon) startTND() error {
	if d.tnd != nil {
		return nil
	}
	d.tnd = tnd.NewDetector(d.config.TND)
	d.initTNDServers()
	d.setTNDDialer()
	if err := d.tnd.Start(); err != nil {
		return fmt.Errorf("Daemon could not start TND: %w", err)
	}
	return nil
}

// stopTND stops TND if it's running.
func (d *Daemon) stopTND() {
	if d.tnd == nil {
		return
	}
	d.tnd.Stop()
	d.tnd = nil
}

// checkTND checks if TND should be running and starts or stops it.
func (d *Daemon) checkTND() error {
	if len(d.profile.GetTNDServers()) == 0 {
		d.stopTND()
		return nil
	}
	return d.startTND()
}

// getTNDResults returns the TND results channel.
// TODO: move this into TND code?
func (d *Daemon) getTNDResults() chan bool {
	if d.tnd == nil {
		return nil
	}
	return d.tnd.Results()
}

// startTrafPol starts traffic policing if it's not running.
func (d *Daemon) startTrafPol() error {
	if d.trafpol != nil {
		return nil
	}
	c := trafpol.NewConfig()
	c.AllowedHosts = append(c.AllowedHosts, d.getProfileAllowedHosts()...)
	c.FirewallMark = d.config.SplitRouting.FirewallMark
	d.trafpol = trafpol.NewTrafPol(c)
	if err := d.trafpol.Start(); err != nil {
		return fmt.Errorf("Daemon could not start TrafPol: %w", err)
	}
	return nil
}

// stopTrafPol stops traffic policing if it's running.
func (d *Daemon) stopTrafPol() {
	if d.trafpol == nil {
		return
	}
	d.trafpol.Stop()
	d.trafpol = nil
}

// checkTrafPol checks if traffic policing should be running and
// starts or stops it.
func (d *Daemon) checkTrafPol() error {
	// check if traffic policing is disabled in the daemon
	if d.disableTrafPol {
		d.stopTrafPol()
		return nil
	}

	// check if traffic policing is enabled in the xml profile
	if !d.profile.GetAlwaysOn() {
		d.stopTrafPol()
		return nil
	}

	// check if we are connected to a trusted network
	if d.status.TrustedNetwork.Trusted() {
		d.stopTrafPol()
		return nil
	}

	return d.startTrafPol()
}

// start starts the daemon.
func (d *Daemon) start() {
	defer close(d.closed)
	defer d.sleepmon.Stop()
	defer d.stopTrafPol()
	defer d.stopTND()
	defer d.vpnsetup.Stop()
	defer d.handleRunnerDisconnect() // clean up vpn config
	defer d.runner.Stop()
	defer d.server.Stop()
	defer d.dbus.Stop()
	defer d.profmon.Stop()

	// run main loop
	for {
		select {
		case req := <-d.server.Requests():
			d.handleClientRequest(req)

		case req := <-d.dbus.Requests():
			d.handleDBusRequest(req)

		case r := <-d.getTNDResults():
			if err := d.handleTNDResult(r); err != nil {
				// send error event and stop daemon
				d.errors <- fmt.Errorf("Daemon could not handle TND result: %w", err)
				return
			}

		case e := <-d.runner.Events():
			d.handleRunnerEvent(e)

		case e := <-d.vpnsetup.Events():
			d.handleVPNSetupEvent(e)

		case e := <-d.sleepmon.Events():
			d.handleSleepMonEvent(e)

		case <-d.profmon.Updates():
			if err := d.handleProfileUpdate(); err != nil {
				// send error event and stop daemon
				d.errors <- fmt.Errorf("Daemon could not handle Profile update: %w", err)
				return
			}

		case <-d.done:
			return
		}
	}
}

// Start starts the daemon.
func (d *Daemon) Start() error {
	// create context
	ctx := context.Background()

	// set executables
	execs.SetExecutables(d.config.Executables)

	// cleanup after a failed shutdown
	d.cleanup(ctx)

	// init token
	if err := d.initToken(); err != nil {
		return fmt.Errorf("Daemon could not init token: %w", err)
	}

	// start sleep monitor
	if err := d.sleepmon.Start(); err != nil {
		return fmt.Errorf("Daemon could not start sleep monitor: %w", err)
	}

	// start traffic policing
	if err := d.checkTrafPol(); err != nil {
		return err
	}

	// start TND
	if err := d.checkTND(); err != nil {
		return err
	}

	// start VPN setup
	d.vpnsetup.Start()

	// start OC runner
	d.runner.Start()

	// start unix server
	err := d.server.Start()
	if err != nil {
		err = fmt.Errorf("Daemon could not start Socket API server: %w", err)
		goto cleanup_unix
	}

	// start dbus api service
	err = d.dbus.Start()
	if err != nil {
		err = fmt.Errorf("Daemon could not start D-Bus API: %w", err)
		goto cleanup_dbus
	}

	// start xml profile monitor
	err = d.profmon.Start()
	if err != nil {
		err = fmt.Errorf("Daemon could not start ProfileMon: %w", err)
		goto cleanup_profmon
	}

	// set initial status
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnected)
	d.setStatusServers(d.profile.GetVPNServerHostNames())

	go d.start()
	return nil

	// clean up after error
cleanup_profmon:
	d.dbus.Stop()
cleanup_dbus:
	d.server.Stop()
cleanup_unix:
	d.runner.Stop()
	d.vpnsetup.Stop()
	d.stopTND()
	d.stopTrafPol()
	d.sleepmon.Stop()

	return err
}

// Stop stops the daemon.
func (d *Daemon) Stop() {
	// stop daemon and wait for main loop termination
	close(d.done)
	<-d.closed
}

// Errors returns the error channel of the daemon.
func (d *Daemon) Errors() chan error {
	return d.errors
}

// NewDaemon returns a new Daemon.
func NewDaemon(config *Config) *Daemon {
	return &Daemon{
		config: config,

		server: api.NewServer(config.SocketServer),
		dbus:   dbusapi.NewService(),

		sleepmon: sleepmon.NewSleepMon(),

		vpnsetup: vpnsetup.NewVPNSetup(config.DNSProxy,
			config.SplitRouting),

		runner: ocrunner.NewConnect(config.OpenConnect),

		status: vpnstatus.New(),

		errors: make(chan error, 1),

		done:   make(chan struct{}),
		closed: make(chan struct{}),

		profile: readXMLProfile(config.OpenConnect.XMLProfile),
		profmon: profilemon.NewProfileMon(config.OpenConnect.XMLProfile),
	}
}
