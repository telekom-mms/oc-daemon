// Package daemon contains the OC-Daemon.
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
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
	config *daemoncfg.Config

	server api.SocketAPI
	dbus   dbusapi.DBusAPI

	tnd tnd.TND

	vpnsetup vpnsetup.Setup
	trafpol  trafpol.Policer

	sleepmon sleepmon.Monitor

	status *vpnstatus.Status

	runner ocrunner.Runner

	// token is used for client authentication
	token string

	// channel for errors
	errors chan error

	// channels for shutdown
	done   chan struct{}
	closed chan struct{}

	// profile is the xml profile
	profile *xmlprofile.Profile
	profmon profilemon.Monitor

	// disableTrafPol determines if traffic policing should be disabled,
	// overrides other traffic policing settings
	disableTrafPol bool

	// serverIP is the IP address of the current VPN server
	serverIP netip.Addr

	// serverIPAllowed indicates whether server IP was added to
	// the allowed addresses
	serverIPAllowed bool
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
	log.WithField("TrustedNetwork", trustedNetwork).Info("Daemon changed TrustedNetwork status")
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
	log.WithField("ConnectionState", connectionState).Info("Daemon changed ConnectionState status")
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
	log.WithField("IP", ip).Info("Daemon changed IP status")
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
	log.WithField("Device", device).Info("Daemon changed Device status")
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
	log.WithField("Server", server).Info("Daemon changed Server status")
	d.status.Server = server
	d.dbus.SetProperty(dbusapi.PropertyServer, server)
}

// setStatusServerIP sets the current server IP in status.
func (d *Daemon) setStatusServerIP(serverIP string) {
	if d.status.ServerIP == serverIP {
		// connected server IP not changed
		return
	}

	// connected server IP changed
	log.WithField("ServerIP", serverIP).Info("Daemon changed Server IP status")
	d.status.ServerIP = serverIP
	d.dbus.SetProperty(dbusapi.PropertyServerIP, serverIP)
}

// setStatusConnectedAt sets the connection time in status.
func (d *Daemon) setStatusConnectedAt(connectedAt int64) {
	if d.status.ConnectedAt == connectedAt {
		// connection time not changed
		return
	}

	// connection time changed
	log.WithField("ConnectedAt", connectedAt).Info("Daemon changed ConnectedAt status")
	d.status.ConnectedAt = connectedAt
	d.dbus.SetProperty(dbusapi.PropertyConnectedAt, connectedAt)
}

// setStatusServers sets the vpn servers in status.
func (d *Daemon) setStatusServers(servers []string) {
	if slices.Equal(d.status.Servers, servers) {
		// servers not changed
		return
	}

	// servers changed
	log.WithField("Servers", servers).Info("Daemon changed Servers status")
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
	log.WithField("OCRunning", ocrunning).Info("Daemon changed OCRunning status")
	d.status.OCRunning = ocrunning
	d.dbus.SetProperty(dbusapi.PropertyOCRunning, ocrunning)
}

// setStatusOCPID sets the openconnect PID in status.
func (d *Daemon) setStatusOCPID(pid uint32) {
	if d.status.OCPID == pid {
		// OC PID not changed
		return
	}

	// OC PID changed
	log.WithField("OCPID", pid).Info("Daemon changed OCPID status")
	d.status.OCPID = pid
	d.dbus.SetProperty(dbusapi.PropertyOCPID, pid)
}

// setStatusTrafPolState sets the TrafPol state in status.
func (d *Daemon) setStatusTrafPolState(state vpnstatus.TrafPolState) {
	if d.status.TrafPolState == state {
		// TrafPol state not changed
		return
	}

	// TrafPol state changed
	log.WithField("TrafPolState", state).Info("Daemon changed TrafPolState status")
	d.status.TrafPolState = state
	d.dbus.SetProperty(dbusapi.PropertyTrafPolState, state)
}

// setStatusAllowedHosts sets the allowed hosts in status.
func (d *Daemon) setStatusAllowedHosts(hosts []string) {
	if slices.Equal(d.status.AllowedHosts, hosts) {
		// allowed hosts not changed
		return
	}

	// allowed hosts changed
	log.WithField("AllowedHosts", hosts).Info("Daemon changed AllowedHosts status")
	d.status.AllowedHosts = hosts
	d.dbus.SetProperty(dbusapi.PropertyAllowedHosts, hosts)
}

// setStatusCaptivePortal sets the captive portal state in status.
func (d *Daemon) setStatusCaptivePortal(capPortal vpnstatus.CaptivePortal) {
	if d.status.CaptivePortal == capPortal {
		// state not changed
		return
	}

	// state changed
	log.WithField("CaptivePortal", capPortal).Info("Daemon changed CaptivePortal status")
	d.status.CaptivePortal = capPortal
	d.dbus.SetProperty(dbusapi.PropertyCaptivePortal, capPortal)
}

// setStatusTNDState sets the TND state in status.
func (d *Daemon) setStatusTNDState(state vpnstatus.TNDState) {
	if d.status.TNDState == state {
		// TND state not changed
		return
	}

	// TND state changed
	log.WithField("TNDState", state).Info("Daemon changed TNDState status")
	d.status.TNDState = state
	d.dbus.SetProperty(dbusapi.PropertyTNDState, state)
}

// setStatusTNDServers sets the TND servers in status.
func (d *Daemon) setStatusTNDServers(servers []string) {
	if slices.Equal(d.status.TNDServers, servers) {
		// TND servers not changed
		return
	}

	// TND servers changed
	log.WithField("TNDServers", servers).Info("Daemon changed TNDServers status")
	d.status.TNDServers = servers
	d.dbus.SetProperty(dbusapi.PropertyTNDServers, servers)
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
	s := string(b)
	log.WithField("VPNConfig", s).Info("Daemon changed VPNConfig status")
	d.dbus.SetProperty(dbusapi.PropertyVPNConfig, s)
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

	// set server address
	if serverIP, err := netip.ParseAddr(strings.Trim(login.Host, "[]")); err == nil {
		d.serverIP = serverIP
	}

	// update status
	d.setStatusOCRunning(true)
	d.setStatusServer(login.Server)
	d.setStatusServerIP(d.serverIP.String())
	d.setStatusConnectionState(vpnstatus.ConnectionStateConnecting)

	// add server address to allowed addrs in trafpol
	if d.trafpol != nil && d.serverIP.IsValid() {
		d.serverIPAllowed = d.trafpol.AddAllowedAddr(d.serverIP)
	}

	// save login and connect using runner
	d.config.LoginInfo = login
	env := []string{
		"oc_daemon_token=" + d.token,
		"oc_daemon_socket_file=" + d.config.SocketServer.SocketFile,
		"oc_daemon_verbose=" + strconv.FormatBool(d.config.Verbose),
	}
	d.runner.Connect(d.config.Copy(), env)
}

// disconnectVPN disconnects from the VPN.
func (d *Daemon) disconnectVPN() {
	// check if vpn is flagged as running
	if !d.status.OCRunning.Running() {
		log.WithField("error", "vpn not running").
			Error("Daemon disconnect error")
		return
	}

	// update status
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnecting)

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
	d.config.VPNConfig = daemoncfg.GetVPNConfig(config)
	d.vpnsetup.Setup(d.config.Copy())

	// set traffic policing setting from Disable Always On VPN setting
	// in configuration
	d.disableTrafPol = config.Flags.DisableAlwaysOnVPN

	// save config
	d.setStatusVPNConfig(config)
	ip := ""
	for _, p := range []netip.Prefix{d.config.VPNConfig.IPv4, d.config.VPNConfig.IPv6} {
		// this assumes either a single IPv4 or a single IPv6 address
		// is configured on a vpn device
		if p.IsValid() {
			ip = p.Addr().String()
		}
	}
	d.setStatusIP(ip)
	d.setStatusDevice(config.Device.Name)

	d.setStatusConnectionState(vpnstatus.ConnectionStateConnected)
	d.setStatusConnectedAt(time.Now().Unix())
	log.Info("Daemon configured VPN connection")
}

// updateVPNConfigDown updates the VPN config for VPN disconnect.
func (d *Daemon) updateVPNConfigDown() {
	// TODO: only call this from Runner Event only and remove down message?
	// or potentially calling this twice is better than not at all?

	// check if vpn is still flagged as connecting or connected
	if d.status.ConnectionState.Connecting() || d.status.ConnectionState.Connected() {
		log.WithField("error", "vpn still connecting or connected").
			Error("Daemon config down error")
		return
	}

	// disconnecting, tear down configuration
	log.Info("Daemon tearing down vpn configuration")
	if d.status.VPNConfig != nil {
		d.vpnsetup.Teardown(d.config)
	}

	// remove login and VPN config
	d.config.LoginInfo = &logininfo.LoginInfo{}
	d.config.VPNConfig = &daemoncfg.VPNConfig{}

	// save config
	d.setStatusVPNConfig(nil)
	d.setStatusServer("")
	d.setStatusServerIP("")
	d.setStatusConnectedAt(0)
	d.setStatusIP("")
	d.setStatusDevice("")

	log.Info("Daemon unconfigured VPN connection")
}

// handleVPNAttemptReconnect handles a "attempt reconnect" event received from vpncscript.
func (d *Daemon) handleVPNAttemptReconnect() {
	// check if vpn is flagged as running
	if !d.status.OCRunning.Running() {
		log.WithField("error", "vpn not running").
			Error("Daemon got invalid attempt reconnect event")
		return
	}

	// check if we are connected or connecting
	if d.status.ConnectionState != vpnstatus.ConnectionStateConnected &&
		d.status.ConnectionState != vpnstatus.ConnectionStateConnecting {
		log.WithField("error", "vpn not connected and not connecting").
			Error("Daemon got invalid attempt reconnect event")
		return
	}

	d.setStatusConnectionState(vpnstatus.ConnectionStateConnecting)
}

// handleVPNReconnect handles a "reconnect" event received from vpncscript.
func (d *Daemon) handleVPNReconnect() {
	// check if vpn is flagged as running
	if !d.status.OCRunning.Running() {
		log.WithField("error", "vpn not running").
			Error("Daemon got invalid reconnect event")
		return
	}

	// check if we are connecting
	if d.status.ConnectionState != vpnstatus.ConnectionStateConnecting {
		log.WithField("error", "vpn not connecting").
			Error("Daemon got invalid reconnect event")
		return
	}

	d.setStatusConnectionState(vpnstatus.ConnectionStateConnected)
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

	// handle config update for vpn pre-init, connect, disconnect,
	// attempt-reconnect, reconnect
	log.WithField("reason", configUpdate.Reason).
		Info("Daemon got OpenConnect event from VPNCScript")
	switch configUpdate.Reason {
	case "connect":
		d.updateVPNConfigUp(configUpdate.Config)
	case "disconnect":
		d.updateVPNConfigDown()
	case "attempt-reconnect":
		d.handleVPNAttemptReconnect()
	case "reconnect":
		d.handleVPNReconnect()
	}
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

// dumpState returns the internal daemon state as json string.
func (d *Daemon) dumpState() string {
	// define state type
	type State struct {
		DaemonConfig     *daemoncfg.Config
		TrafficPolicing  *trafpol.State
		VPNSetup         *vpnsetup.State
		CommandLists     map[string]*cmdtmpl.CommandList
		CommandTemplates string
	}

	// collect internal state
	c := d.config.Copy()
	c.LoginInfo.Cookie = "HIDDEN" // hide cookie
	state := State{
		DaemonConfig:     c,
		CommandLists:     cmdtmpl.CommandLists,
		CommandTemplates: cmdtmpl.LoadedTemplates,
	}
	if d.trafpol != nil {
		state.TrafficPolicing = d.trafpol.GetState()
	}
	if d.vpnsetup != nil {
		state.VPNSetup = d.vpnsetup.GetState()
	}

	// convert to json
	b, err := json.Marshal(state)
	if err != nil {
		log.WithError(err).Error("Daemon could not convert internal state to JSON")
		return ""
	}

	return string(b)
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
		log.Info("Daemon got connect request from client")
		d.connectVPN(login)

	case dbusapi.RequestDisconnect:
		// disconnect VPN
		log.Info("Daemon got disconnect request from client")
		d.disconnectVPN()

	case dbusapi.RequestDumpState:
		// dump state
		state := d.dumpState()
		log.WithField("state", state).Info("Daemon got dump state request from client")
		request.Results = []any{state}
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
// cleaning up everything. This is also called when stopping the daemon.
func (d *Daemon) handleRunnerDisconnect() {
	// make sure running and connected are not set
	d.setStatusOCRunning(false)
	d.setStatusOCPID(0)
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnected)
	d.setStatusServer("")
	d.setStatusServerIP("")
	d.setStatusConnectedAt(0)

	// make sure the vpn config is not active any more
	d.updateVPNConfigDown()

	// remove server ip from allowed addrs and delete it
	if d.trafpol != nil && d.serverIPAllowed {
		d.trafpol.RemoveAllowedAddr(d.serverIP)
	}
	d.serverIP = netip.Addr{}
	d.serverIPAllowed = false
}

// handleRunnerEvent handles a connect event from the OC runner.
func (d *Daemon) handleRunnerEvent(e *ocrunner.ConnectEvent) {
	log.WithField("event", e).Debug("Daemon handling Runner event")

	if e.Connect {
		// make sure running is set
		d.setStatusOCRunning(true)
		d.setStatusOCPID(e.PID)
		return
	}

	// clean up after disconnect
	d.handleRunnerDisconnect()
}

// handleSleepMonEvent handles a suspend/resume event from SleepMon.
func (d *Daemon) handleSleepMonEvent(sleep bool) {
	log.WithField("sleep", sleep).Debug("Daemon handling SleepMon event")

	// disconnect vpn on resume
	if !sleep && d.status.OCRunning.Running() {
		log.Info("Daemon resuming after sleep, disconnecting")
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
	log.Info("Daemon got XML profile update")
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

// handleCPDStatusUpdate handles a CPD status update.
func (d *Daemon) handleCPDStatusUpdate(detected bool) {
	log.WithField("detected", detected).Debug("Daemon handling CPD status update")

	if detected {
		d.setStatusCaptivePortal(vpnstatus.CaptivePortalDetected)
		return
	}
	d.setStatusCaptivePortal(vpnstatus.CaptivePortalNotDetected)
}

// cleanup cleans up after a failed shutdown.
func (d *Daemon) cleanup(ctx context.Context) {
	ocrunner.CleanupConnect(d.config.OpenConnect)
	vpnsetup.Cleanup(ctx, d.config)
	trafpol.Cleanup(ctx, d.config)
}

// initToken creates the daemon token for client authentication.
func (d *Daemon) initToken() error {
	token, err := api.GetToken()
	if err != nil {
		return err
	}
	d.token = token
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
	log.Info("Daemon starting TND")
	d.tnd = tnd.NewDetector(d.config.TND)
	servers := d.profile.GetTNDHTTPSServers()
	d.tnd.SetServers(servers)
	d.setTNDDialer()
	if err := d.tnd.Start(); err != nil {
		return fmt.Errorf("Daemon could not start TND: %w", err)
	}

	// update tnd status
	var s []string
	for k, v := range servers {
		s = append(s, fmt.Sprintf("%s:%s", k, v))
	}
	d.setStatusTNDState(vpnstatus.TNDStateActive)
	d.setStatusTNDServers(s)

	return nil
}

// stopTND stops TND if it's running.
func (d *Daemon) stopTND() {
	if d.tnd == nil {
		return
	}
	log.Info("Daemon stopping TND")
	d.tnd.Stop()
	d.tnd = nil

	// update tnd status
	d.setStatusTNDState(vpnstatus.TNDStateInactive)
	d.setStatusTNDServers(nil)
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
	log.Info("Daemon starting TrafPol")
	c := d.config.Copy()
	c.TrafficPolicing.AllowedHosts = append(c.TrafficPolicing.AllowedHosts, d.getProfileAllowedHosts()...)
	d.trafpol = trafpol.NewTrafPol(c)
	if err := d.trafpol.Start(); err != nil {
		return fmt.Errorf("Daemon could not start TrafPol: %w", err)
	}

	// update trafpol status
	d.setStatusTrafPolState(vpnstatus.TrafPolStateActive)
	d.setStatusAllowedHosts(c.TrafficPolicing.AllowedHosts)
	d.setStatusCaptivePortal(vpnstatus.CaptivePortalNotDetected)

	if d.serverIP.IsValid() {
		// VPN connection active, allow server IP
		d.serverIPAllowed = d.trafpol.AddAllowedAddr(d.serverIP)
	}

	return nil
}

// stopTrafPol stops traffic policing if it's running.
func (d *Daemon) stopTrafPol() {
	if d.trafpol == nil {
		return
	}
	log.Info("Daemon stopping TrafPol")
	d.trafpol.Stop()
	d.trafpol = nil
	d.serverIPAllowed = false

	// update trafpol status
	if d.disableTrafPol {
		d.setStatusTrafPolState(vpnstatus.TrafPolStateDisabled)
	} else {
		d.setStatusTrafPolState(vpnstatus.TrafPolStateInactive)
	}
	d.setStatusAllowedHosts(nil)
	d.setStatusCaptivePortal(vpnstatus.CaptivePortalUnknown)
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
	defer d.profmon.Stop()
	defer d.stopTrafPol()
	defer d.stopTND()
	defer d.vpnsetup.Stop()
	defer d.server.Stop()
	defer d.runner.Stop()
	defer d.handleRunnerDisconnect() // clean up vpn config
	defer d.dbus.Stop()
	defer d.server.Shutdown()

	// run main loop
	log.Info("Daemon started")
	for {
		var cpdStatus <-chan bool
		if d.trafpol != nil {
			cpdStatus = d.trafpol.CPDStatus()
		}

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

		case e := <-d.sleepmon.Events():
			d.handleSleepMonEvent(e)

		case <-d.profmon.Updates():
			if err := d.handleProfileUpdate(); err != nil {
				// send error event and stop daemon
				d.errors <- fmt.Errorf("Daemon could not handle Profile update: %w", err)
				return
			}

		case s := <-cpdStatus:
			d.handleCPDStatusUpdate(s)

		case <-d.done:
			log.Info("Daemon stopping")
			return
		}
	}
}

// Start starts the daemon.
func (d *Daemon) Start() error {
	// create context
	ctx := context.Background()

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

	// start xml profile monitor
	err := d.profmon.Start()
	if err != nil {
		err = fmt.Errorf("Daemon could not start ProfileMon: %w", err)
		goto cleanup_profmon
	}

	// start VPN setup
	d.vpnsetup.Start()

	// start OC runner
	d.runner.Start()

	// start unix server
	err = d.server.Start()
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

	// set initial status
	d.setStatusTrustedNetwork(false)
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnected)
	d.setStatusServers(d.profile.GetVPNServerHostNames())
	d.setStatusConnectedAt(0)
	d.setStatusOCRunning(false)
	d.setStatusTrafPolState(vpnstatus.TrafPolStateInactive)
	d.setStatusTNDState(vpnstatus.TNDStateInactive)

	// start traffic policing
	err = d.checkTrafPol()
	if err != nil {
		goto cleanup_trafpol
	}

	// start TND
	err = d.checkTND()
	if err != nil {
		goto cleanup_tnd
	}

	go d.start()
	return nil

	// clean up after error
cleanup_tnd:
	d.stopTrafPol()
cleanup_trafpol:
	d.dbus.Stop()
	d.server.Stop()
cleanup_dbus:
	d.server.Stop()
cleanup_unix:
	d.runner.Stop()
	d.vpnsetup.Stop()
	d.profmon.Stop()
cleanup_profmon:
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
func NewDaemon(config *daemoncfg.Config) *Daemon {
	return &Daemon{
		config: config,

		server: api.NewServer(config.SocketServer),
		dbus:   dbusapi.NewService(),

		sleepmon: sleepmon.NewSleepMon(),

		vpnsetup: vpnsetup.NewVPNSetup(dnsproxy.NewProxy(config.DNSProxy)),

		runner: ocrunner.NewConnect(),

		status: vpnstatus.New(),

		errors: make(chan error, 1),

		done:   make(chan struct{}),
		closed: make(chan struct{}),

		profile: readXMLProfile(config.OpenConnect.XMLProfile),
		profmon: profilemon.NewProfileMon(config.OpenConnect.XMLProfile),
	}
}
