package daemon

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"strconv"
	"syscall"

	"github.com/T-Systems-MMS/oc-daemon/internal/api"
	"github.com/T-Systems-MMS/oc-daemon/internal/dnsproxy"
	"github.com/T-Systems-MMS/oc-daemon/internal/ocrunner"
	"github.com/T-Systems-MMS/oc-daemon/internal/sleepmon"
	"github.com/T-Systems-MMS/oc-daemon/internal/splitrt"
	"github.com/T-Systems-MMS/oc-daemon/internal/trafpol"
	"github.com/T-Systems-MMS/oc-daemon/internal/xmlprofile"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnconfig"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnstatus"
	"github.com/T-Systems-MMS/tnd/pkg/trustnet"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	// dnsAddr is the DNS listen address
	dnsAddr = "127.0.0.1:4253" // TODO: change?

	// defaultDNSServer is the default DNS server address
	// TODO: check if ok. use variable?
	defaultDNSServer = "127.0.0.53:53"
)

var (
	// cpdServers is a list of CPD servers, e.g., used by browsers
	cpdServers = []string{
		"connectivity-check.ubuntu.com", // ubuntu
		"detectportal.firefox.com",      // firefox
		"www.gstatic.com",               // chrome
		"clients3.google.com",           // chromium
		"nmcheck.gnome.org",             // gnome
	}
)

// Daemon is used to run the daemon
type Daemon struct {
	server *api.Server

	dns *dnsproxy.Proxy
	tnd *trustnet.TND

	splitrt *splitrt.SplitRouting
	trafpol *trafpol.TrafPol

	sleepmon *sleepmon.SleepMon

	status *vpnstatus.Status

	runner *ocrunner.Connect

	// token is used for client authentication
	token string

	// channels for shutdown
	done   chan struct{}
	closed chan struct{}

	// profile is the xml profile
	profile *xmlprofile.Profile

	// disableTrafPol determines if traffic policing should be disabled,
	// overrides other traffic policing settings
	disableTrafPol bool
}

// setStatusTrustedNetwork sets the trusted network status in status
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
}

// setStatusConnectionState sets the connection state in status
func (d *Daemon) setStatusConnectionState(connectionState vpnstatus.ConnectionState) {
	if d.status.ConnectionState == connectionState {
		// state not changed
		return
	}

	// state changed
	d.status.ConnectionState = connectionState
}

// setStatusIP sets the IP in status
func (d *Daemon) setStatusIP(ip string) {
	if d.status.IP == ip {
		// ip not changed
		return
	}

	// ip changed
	d.status.IP = ip
}

// setStatusDevice sets the device in status
func (d *Daemon) setStatusDevice(device string) {
	if d.status.Device == device {
		// device not changed
		return
	}

	// device changed
	d.status.Device = device
}

// connectVPN connects to the VPN using login info from client request
func (d *Daemon) connectVPN(login *ocrunner.LoginInfo) {
	// allow only one connection
	if d.status.Running {
		return
	}

	// ignore invalid login information
	if !login.Valid() {
		return
	}

	// update status
	d.status.Running = true
	d.setStatusConnectionState(vpnstatus.ConnectionStateConnecting)

	// connect using runner
	env := []string{"oc_daemon_token=" + d.token}
	d.runner.Connect(login, env)
}

// disconnectVPN disconnects from the VPN
func (d *Daemon) disconnectVPN() {
	// update status
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnecting)
	d.status.Running = false

	// stop runner
	if d.runner == nil {
		return
	}
	d.runner.Disconnect()
}

// sendVPNStatus sends the VPN status back to the client
func (d *Daemon) sendVPNStatus(request *api.Request) {
	// send OK with VPN status
	b, err := d.status.JSON()
	if err != nil {
		log.WithError(err).Fatal("Daemon could not convert status to JSON")
	}
	request.Reply(b)
}

// setupRouting sets up routing using config
// TODO: move somewhere else?
func (d *Daemon) setupRouting(config *vpnconfig.Config) {
	if d.splitrt != nil {
		return
	}
	d.splitrt = splitrt.NewSplitRouting(config)
	d.splitrt.Start()
}

// teardownRouting tears down the routing configuration
func (d *Daemon) teardownRouting() {
	if d.splitrt == nil {
		return
	}
	d.splitrt.Stop()
	d.splitrt = nil
}

// setupDNS sets up DNS using config
// TODO: move somewhere else?
func (d *Daemon) setupDNS(config *vpnconfig.Config) {
	// configure dns proxy
	// TODO: improve this

	// set remotes
	remotes := config.DNS.Remotes()
	d.dns.SetRemotes(remotes)

	// set watches
	excludes := config.Split.DNSExcludes()
	log.WithField("excludes", excludes).Debug("Daemon setting DNS Split Excludes")
	d.dns.SetWatches(excludes)

	// update dns configuration of host
	setVPNDNS(config, dnsAddr)
}

// teardownDNS tears down the DNS configuration
func (d *Daemon) teardownDNS() {
	remotes := map[string][]string{
		".": []string{defaultDNSServer},
	}
	d.dns.SetRemotes(remotes)
	d.dns.SetWatches([]string{})
	unsetVPNDNS(d.status.Config)
}

// updateVPNConfigUp updates the VPN config for VPN connect
func (d *Daemon) updateVPNConfigUp(config *vpnconfig.Config) {
	// check if old and new config differ
	if config.Equal(d.status.Config) {
		log.WithField("error", "old and new vpn configs are equal").
			Error("Daemon config up error")
		return
	}

	// check if vpn is flagged as running
	if !d.status.Running {
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
	setupVPNDevice(config)
	d.setupRouting(config)
	d.setupDNS(config)

	// set traffic policing setting from Disable Always On VPN setting
	// in configuration
	d.disableTrafPol = config.Flags.DisableAlwaysOnVPN

	// save config
	d.status.Config = config
	d.setStatusConnectionState(vpnstatus.ConnectionStateConnected)
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

// updateVPNConfigDown updates the VPN config for VPN disconnect
func (d *Daemon) updateVPNConfigDown() {
	// TODO: only call this from Runner Event only and remove down message?
	// or potentially calling this twice is better than not at all?

	// check if vpn is still flagged as running
	if d.status.Running {
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
	if d.status.Config != nil {
		teardownVPNDevice(d.status.Config)
		d.teardownRouting()
		d.teardownDNS()
	}

	// save config
	d.status.Config = nil
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnected)
	d.setStatusIP("")
	d.setStatusDevice("")
}

// updateVPNConfig updates the VPN config with config update in client request
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

// handleClientRequest handles a client request
func (d *Daemon) handleClientRequest(request *api.Request) {
	defer request.Close()
	log.Debug("Daemon handling client request")

	switch request.Type() {
	case api.TypeVPNConnect:
		// parse login info
		login, err := ocrunner.LoginInfoFromJSON(request.Data())
		if err != nil {
			log.WithError(err).Error("Daemon could not parse login info JSON")
			request.Error("invalid login info in connect message")
			break
		}

		// connect VPN
		d.connectVPN(login)

	case api.TypeVPNDisconnect:
		// diconnect VPN
		d.disconnectVPN()

	case api.TypeVPNQuery:
		// send vpn status
		d.sendVPNStatus(request)

	case api.TypeVPNConfigUpdate:
		// update VPN config
		d.updateVPNConfig(request)
	}
}

// handleDNSReport handles a DNS report
func (d *Daemon) handleDNSReport(r *dnsproxy.Report) {
	log.WithField("report", r).Debug("Daemon handling DNS report")

	if !d.status.Running { // TODO: fix connected state and change to connected?
		return
	}
	if d.splitrt == nil {
		return
	}

	// forward report to split routing
	select {
	case d.splitrt.DNSReports() <- r:
	case <-d.done:
	}
}

// checkDisconnectVPN checks if we need to disconnect the VPN when handling a
// TND result
func (d *Daemon) checkDisconnectVPN() {
	if d.status.TrustedNetwork.Trusted() && d.status.Running {
		// disconnect VPN when switching from untrusted network with
		// active VPN connection to a trusted network
		log.Info("Daemon detected trusted network, disconnecting VPN connection")
		d.disconnectVPN()
	}
}

// handleTNDResult handles a TND result
func (d *Daemon) handleTNDResult(trusted bool) {
	log.WithField("trusted", trusted).Debug("Daemon handling TND result")
	d.setStatusTrustedNetwork(trusted)
	d.checkDisconnectVPN()
	d.checkTrafPol()
}

// handleRunnerDisconnect handles a disconnect event from the OC runner,
// cleaning up everthing. This is also called when stopping the daemon
func (d *Daemon) handleRunnerDisconnect() {
	// make sure running and connected are not set
	d.status.Running = false
	d.setStatusConnectionState(vpnstatus.ConnectionStateDisconnected)

	// make sure the vpn config is not active any more
	d.updateVPNConfigDown()
}

// handleRunnerEvent handles a connect event from the OC runner
func (d *Daemon) handleRunnerEvent(e *ocrunner.ConnectEvent) {
	log.WithField("event", e).Debug("Daemon handling Runner event")

	if e.Connect {
		// make sure running is set
		d.status.Running = true
		return
	}

	// clean up after disconnect
	d.handleRunnerDisconnect()
}

// handleSleepMonEvent handles a suspend/resume event from SleepMon
func (d *Daemon) handleSleepMonEvent(sleep bool) {
	log.WithField("sleep", sleep).Debug("Daemon handling SleepMon event")

	// disconnect vpn on resume
	if !sleep && d.status.Running {
		d.disconnectVPN()
	}
}

// handleProfileUpdate handles a xml profile update
func (d *Daemon) handleProfileUpdate() {
	log.Debug("Daemon handling XML profile update")
	d.profile.Parse()
	d.stopTND()
	d.stopTrafPol()
	d.checkTrafPol()
	d.checkTND()
}

// cleanup cleans up after a failed shutdown
func (d *Daemon) cleanup() {
	ocrunner.CleanupConnect()
	cleanupVPNConfig(vpnDevice)
	splitrt.Cleanup()
	trafpol.Cleanup()
}

// initToken creates the daemon token for client authentication
func (d *Daemon) initToken() {
	// TODO: is this good enough for us?
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.WithError(err).Fatal("Daemon could not init token")
	}
	d.token = base64.RawURLEncoding.EncodeToString(b)
}

// getAllowedHosts returns the allowed hosts
func (d *Daemon) getAllowedHosts() (hosts []string) {
	// add vpn servers to allowed hosts
	hosts = append(hosts, d.profile.GetVPNServers()...)

	// add tnd servers to allowed hosts
	hosts = append(hosts, d.profile.GetTNDServers()...)

	// add cpd servers to allowed hosts
	hosts = append(hosts, cpdServers...)

	// add allowed hosts from xml profile to allowed hosts
	hosts = append(hosts, d.profile.GetAllowedHosts()...)

	return
}

// initTNDServers sets the TND servers from the xml profile
func (d *Daemon) initTNDServers() {
	urls, hashes := d.profile.GetTNDHTTPSServers()
	for i, url := range urls {
		d.tnd.AddServer(url, hashes[i])
	}
}

// setTNDDialer sets a custom dialer for TND
func (d *Daemon) setTNDDialer() {
	// get mark to be set on socket
	mark, err := strconv.Atoi(splitrt.FWMark)
	if err != nil {
		log.WithError(err).Error("Daemon could not convert FWMark to int")
		return
	}

	// control function that sets socket option on raw connection
	control := func(network, address string, c syscall.RawConn) error {
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

// startTND starts TND if it's not running
func (d *Daemon) startTND() {
	if d.tnd != nil {
		return
	}
	d.tnd = trustnet.NewTND()
	d.initTNDServers()
	d.setTNDDialer()
	d.tnd.Start()
}

// stopTND stops TND if it's running
func (d *Daemon) stopTND() {
	if d.tnd == nil {
		return
	}
	d.tnd.Stop()
	d.tnd = nil
}

// checkTND checks if TND should be running and starts or stops it
func (d *Daemon) checkTND() {
	if len(d.profile.GetTNDServers()) == 0 {
		d.stopTND()
		return
	}
	d.startTND()
}

// getTNDResults returns the TND results channel
// TODO: move this into TND code?
func (d *Daemon) getTNDResults() chan bool {
	if d.tnd == nil {
		return nil
	}
	return d.tnd.Results()
}

// startTrafPol starts traffic policing if it's not running
func (d *Daemon) startTrafPol() {
	if d.trafpol != nil {
		return
	}
	d.trafpol = trafpol.NewTrafPol(d.getAllowedHosts())
	d.trafpol.Start()
}

// stopTrafPol stops traffic policing if it's running
func (d *Daemon) stopTrafPol() {
	if d.trafpol == nil {
		return
	}
	d.trafpol.Stop()
	d.trafpol = nil
}

// checkTrafPol checks if traffic policing should be running and
// starts or stops it
func (d *Daemon) checkTrafPol() {
	// check if traffic policing is disabled in the daemon
	if d.disableTrafPol {
		d.stopTrafPol()
		return
	}

	// check if traffic policing is enabled in the xml profile
	if !d.profile.GetAlwaysOn() {
		d.stopTrafPol()
		return
	}

	// check if we are connected to a trusted network
	if d.status.TrustedNetwork.Trusted() {
		d.stopTrafPol()
		return
	}

	d.startTrafPol()
}

// start starts the daemon
func (d *Daemon) start() {
	defer close(d.closed)

	// cleanup after a failed shutdown
	d.cleanup()

	// init token
	d.initToken()

	// start sleep monitor
	d.sleepmon.Start()

	// start traffic policing
	d.checkTrafPol()
	defer d.stopTrafPol()

	// start TND
	d.checkTND()
	defer d.stopTND()

	// start DNS-Proxy
	d.dns.Start()
	defer d.dns.Stop()

	// start OC runner
	d.runner.Start()
	defer d.handleRunnerDisconnect() // clean up vpn config
	defer d.runner.Stop()

	// start unix server
	d.server.Start()
	defer d.server.Stop()

	// start xml profile watching
	d.profile.Start()
	defer d.profile.Stop()

	// run main loop
	for {
		select {
		case req := <-d.server.Requests():
			d.handleClientRequest(req)

		case r := <-d.dns.Reports():
			d.handleDNSReport(r)

		case r := <-d.getTNDResults():
			d.handleTNDResult(r)

		case e := <-d.runner.Events():
			d.handleRunnerEvent(e)

		case e := <-d.sleepmon.Events():
			d.handleSleepMonEvent(e)

		case <-d.profile.Updates():
			d.handleProfileUpdate()

		case <-d.done:
			return
		}
	}
}

// Start starts the daemon
func (d *Daemon) Start() {
	go d.start()
}

// Stop stops the daemon
func (d *Daemon) Stop() {
	// stop daemon and wait for main loop termination
	close(d.done)
	<-d.closed
}

// NewDaemon returns a new Daemon
func NewDaemon() *Daemon {
	// parse xml profile
	profile := xmlprofile.NewXMLProfile(xmlProfile)
	profile.Parse()

	return &Daemon{
		server: api.NewServer(sockFile),

		sleepmon: sleepmon.NewSleepMon(),

		dns: dnsproxy.NewProxy(dnsAddr),

		runner: ocrunner.NewConnect(xmlProfile, vpncScript, vpnDevice),

		status: vpnstatus.New(),

		done:   make(chan struct{}),
		closed: make(chan struct{}),

		profile: profile,
	}
}
