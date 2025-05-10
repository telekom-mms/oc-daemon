// Package client contains code for OC-Daemon clients.
package client

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
)

// Client is an OC-Daemon client.
type Client interface {
	SetConfig(config *Config)
	GetConfig() *Config

	SetEnv(env []string)
	GetEnv() []string

	SetLogin(login *logininfo.LoginInfo)
	GetLogin() *logininfo.LoginInfo

	Ping() error
	Query() (*vpnstatus.Status, error)
	Subscribe() (chan *vpnstatus.Status, error)

	Authenticate() error
	Connect() error
	Disconnect() error

	DumpState() (string, error)

	Close() error
}

// DBusClient is an OC-Daemon client that uses the D-Bus API of OC-Daemon.
type DBusClient struct {
	mutex sync.Mutex

	// config is the client configuration
	config *Config

	// conn is the D-Bus connection
	conn *dbus.Conn

	// signals is the channel for the D-Bus signals
	signals chan *dbus.Signal

	// env are extra environment variables set during execution of
	// `openconnect --authenticate`
	env []string

	// login contains information required to connect to the VPN, produced
	// by successful authentication
	login *logininfo.LoginInfo

	// subscribed specifies whether the client is subscribed to
	// PropertiesChanged D-Bus signals
	subscribed bool

	// update is used for vpn status updates
	updates chan *vpnstatus.Status

	// done signals termination of the client
	done chan struct{}

	// closed signals termination of the client is complete
	closed chan struct{}
}

// SetConfig sets the client config.
func (d *DBusClient) SetConfig(config *Config) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.config = config.Copy()
}

// GetConfig returns the client config.
func (d *DBusClient) GetConfig() *Config {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.config.Copy()
}

// SetEnv sets additional environment variables.
func (d *DBusClient) SetEnv(env []string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.env = append(env[:0:0], env...)
}

// GetEnv returns the additional environment variables.
func (d *DBusClient) GetEnv() []string {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return append(d.env[:0:0], d.env...)
}

// SetLogin sets the login information.
func (d *DBusClient) SetLogin(login *logininfo.LoginInfo) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.login = login.Copy()
}

// GetLogin returns the login information.
func (d *DBusClient) GetLogin() *logininfo.LoginInfo {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.login.Copy()
}

// dbusConnectSystemBus calls dbus.ConnectSystemBus.
var dbusConnectSystemBus = func() (*dbus.Conn, error) {
	return dbus.ConnectSystemBus()
}

// updateStatusFromProperties updates status from D-Bus properties in props.
func updateStatusFromProperties(status *vpnstatus.Status, props map[string]dbus.Variant) error {
	// create a temporary status, try to set all values in temporary
	// status, if we received valid properties (no type conversion or JSON
	// parsing errors) set real status
	temp := vpnstatus.New()
	for _, dest := range []*vpnstatus.Status{temp, status} {
		for k, v := range props {
			var err error
			switch k {
			case dbusapi.PropertyTrustedNetwork:
				err = v.Store(&dest.TrustedNetwork)
			case dbusapi.PropertyConnectionState:
				err = v.Store(&dest.ConnectionState)
			case dbusapi.PropertyIP:
				err = v.Store(&dest.IP)
			case dbusapi.PropertyDevice:
				err = v.Store(&dest.Device)
			case dbusapi.PropertyServer:
				err = v.Store(&dest.Server)
			case dbusapi.PropertyServerIP:
				err = v.Store(&dest.ServerIP)
			case dbusapi.PropertyConnectedAt:
				err = v.Store(&dest.ConnectedAt)
			case dbusapi.PropertyServers:
				err = v.Store(&dest.Servers)
			case dbusapi.PropertyOCRunning:
				err = v.Store(&dest.OCRunning)
			case dbusapi.PropertyOCPID:
				err = v.Store(&dest.OCPID)
			case dbusapi.PropertyTrafPolState:
				err = v.Store(&dest.TrafPolState)
			case dbusapi.PropertyAllowedHosts:
				err = v.Store(&dest.AllowedHosts)
			case dbusapi.PropertyCaptivePortal:
				err = v.Store(&dest.CaptivePortal)
			case dbusapi.PropertyTNDState:
				err = v.Store(&dest.TNDState)
			case dbusapi.PropertyTNDServers:
				err = v.Store(&dest.TNDServers)
			case dbusapi.PropertyVPNConfig:
				s := dbusapi.VPNConfigInvalid
				if err := v.Store(&s); err != nil {
					return err
				}
				if s == dbusapi.VPNConfigInvalid {
					dest.VPNConfig = nil
				} else {
					config, err := vpnconfig.NewFromJSON([]byte(s))
					if err != nil {
						return err
					}
					dest.VPNConfig = config
				}
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ping calls the ping method to check if OC-Daemon is running.
var ping = func(d *DBusClient) error {
	return d.conn.Object(dbusapi.Interface, dbusapi.Path).
		Call("org.freedesktop.DBus.Peer.Ping", 0).Err
}

// Ping pings the OC-Daemon to check if it is running.
func (d *DBusClient) Ping() error {
	return ping(d)
}

// query retrieves the D-Bus properties from the daemon.
var query = func(d *DBusClient) (map[string]dbus.Variant, error) {
	// get all properties
	props := make(map[string]dbus.Variant)
	if err := d.conn.Object(dbusapi.Interface, dbusapi.Path).
		Call("org.freedesktop.DBus.Properties.GetAll", 0, dbusapi.Interface).
		Store(props); err != nil {
		return nil, err
	}

	// return properties
	return props, nil
}

// Query retrieves the VPN status.
func (d *DBusClient) Query() (*vpnstatus.Status, error) {
	// get properties
	props, err := query(d)
	if err != nil {
		return nil, err
	}

	// get status from properties
	status := vpnstatus.New()
	if err := updateStatusFromProperties(status, props); err != nil {
		return nil, err
	}

	// return current status
	return status, nil
}

// handlePropertiesChanged handles a PropertiesChanged D-Bus signal.
func handlePropertiesChanged(s *dbus.Signal, status *vpnstatus.Status) *vpnstatus.Status {
	// make sure it's a properties changed signal
	if s.Path != dbusapi.Path || s.Name != dbusapi.PropertiesChanged {
		return nil
	}

	// check properties changed signal
	if v, ok := s.Body[0].(string); !ok || v != dbusapi.Interface {
		return nil
	}

	// get changed properties, update current status
	changed, ok := s.Body[1].(map[string]dbus.Variant)
	if !ok {
		return nil
	}

	err := updateStatusFromProperties(status, changed)
	if err != nil {
		return nil
	}

	// get invalidated properties
	invalid, ok := s.Body[2].([]string)
	if !ok {
		return nil
	}
	for _, name := range invalid {
		// not expected to happen currently, but handle it anyway
		switch name {
		case dbusapi.PropertyTrustedNetwork:
			status.TrustedNetwork = vpnstatus.TrustedNetworkUnknown
		case dbusapi.PropertyConnectionState:
			status.ConnectionState = vpnstatus.ConnectionStateUnknown
		case dbusapi.PropertyIP:
			status.IP = dbusapi.IPInvalid
		case dbusapi.PropertyDevice:
			status.Device = dbusapi.DeviceInvalid
		case dbusapi.PropertyServer:
			status.Server = dbusapi.ServerInvalid
		case dbusapi.PropertyServerIP:
			status.ServerIP = dbusapi.ServerIPInvalid
		case dbusapi.PropertyConnectedAt:
			status.ConnectedAt = dbusapi.ConnectedAtInvalid
		case dbusapi.PropertyServers:
			status.Servers = dbusapi.ServersInvalid
		case dbusapi.PropertyOCRunning:
			status.OCRunning = vpnstatus.OCRunningUnknown
		case dbusapi.PropertyOCPID:
			status.OCPID = dbusapi.OCPIDInvalid
		case dbusapi.PropertyTrafPolState:
			status.TrafPolState = vpnstatus.TrafPolStateUnknown
		case dbusapi.PropertyAllowedHosts:
			status.AllowedHosts = dbusapi.AllowedHostsInvalid
		case dbusapi.PropertyCaptivePortal:
			status.CaptivePortal = vpnstatus.CaptivePortalUnknown
		case dbusapi.PropertyTNDState:
			status.TNDState = vpnstatus.TNDStateUnknown
		case dbusapi.PropertyTNDServers:
			status.TNDServers = dbusapi.TNDServersInvalid
		case dbusapi.PropertyVPNConfig:
			status.VPNConfig = nil
		}
	}

	return status
}

// setSubscribed tries to set subscribed to true and returns true if successful.
func (d *DBusClient) setSubscribed() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.subscribed {
		// already subscribed
		return false
	}
	d.subscribed = true
	return true
}

// isSubscribed returns whether subscribed is set.
func (d *DBusClient) isSubscribed() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.subscribed
}

// connAddmatchSignal is dbus conn.AddMatchSignal for testing.
var connAddMatchSignal = func(conn *dbus.Conn, options ...dbus.MatchOption) error {
	return conn.AddMatchSignal(options...)
}

// connSignal is dbus conn.Signal for testing.
var connSignal = func(conn *dbus.Conn, ch chan<- *dbus.Signal) {
	conn.Signal(ch)
}

// Subscribe subscribes to PropertiesChanged D-Bus signals, converts incoming
// PropertiesChanged signals to VPN status updates and sends those updates
// over the returned channel.
func (d *DBusClient) Subscribe() (chan *vpnstatus.Status, error) {
	// make sure this only runs once
	if ok := d.setSubscribed(); !ok {
		return nil, fmt.Errorf("already subscribed")
	}

	// query current status to get initial values
	status, err := d.Query()
	if err != nil {
		return nil, err
	}

	// subscribe to properties changed signals
	if err := connAddMatchSignal(d.conn,
		dbus.WithMatchSender(dbusapi.Interface),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchPathNamespace(dbusapi.Path),
	); err != nil {
		return nil, err
	}

	// handle signals
	connSignal(d.conn, d.signals)

	// handle properties
	go func() {
		defer close(d.closed)
		defer close(d.updates)

		// send initial status
		select {
		case d.updates <- status.Copy():
		case <-d.done:
			return
		}

		// handle signals
		for s := range d.signals {
			// get status update from signal
			update := handlePropertiesChanged(s, status.Copy())
			if update == nil {
				// invalid update
				continue
			}

			// valid update, save it as current status
			status = update.Copy()

			// send status update
			select {
			case d.updates <- update:
			case <-d.done:
				return
			}
		}
	}()

	return d.updates, nil
}

// checkStatus checks if client is not connected to a trusted network and the
// VPN is not already running.
func (d *DBusClient) checkStatus() error {
	status, err := d.Query()
	if err != nil {
		return fmt.Errorf("could not query OC-Daemon: %w", err)
	}

	// check if we need to start the VPN connection
	if status.TrustedNetwork.Trusted() {
		return fmt.Errorf("trusted network detected, nothing to do")
	}
	if status.ConnectionState.Connected() {
		return fmt.Errorf("VPN already connected, nothing to do")
	}
	if status.OCRunning.Running() {
		return fmt.Errorf("OpenConnect client already running, nothing to do")
	}
	return nil
}

// execCommand is exec.Command for testing.
var execCommand = exec.Command

// authenticate runs OpenConnect in authentication mode.
var authenticate = func(d *DBusClient) error {
	// create openconnect command:
	//
	// openconnect \
	//   --protocol=anyconnect \
	//   --certificate="$CLIENT_CERT" \
	//   --sslkey="$PRIVATE_KEY" \
	//   --cafile="$CA_CERT" \
	//   --xmlconfig="$XML_CONFIG" \
	//   --authenticate \
	//   --quiet \
	//   "$SERVER"
	//
	config := d.GetConfig()
	protocol := fmt.Sprintf("--protocol=%s", config.Protocol)
	// some VPN servers reject connections from other clients,
	// set default user agent to AnyConnect
	userAgent := fmt.Sprintf("--useragent=%s", config.UserAgent)
	certificate := fmt.Sprintf("--certificate=%s", config.ClientCertificate)
	sslKey := fmt.Sprintf("--sslkey=%s", config.ClientKey)
	mcaCertificate := fmt.Sprintf("--mca-certificate=%s", config.UserCertificate)
	mcaKey := fmt.Sprintf("--mca-key=%s", config.UserKey)
	caFile := fmt.Sprintf("--cafile=%s", config.CACertificate)
	xmlConfig := fmt.Sprintf("--xmlconfig=%s", config.XMLProfile)
	user := fmt.Sprintf("--user=%s", config.User)

	parameters := []string{
		protocol,
		userAgent,
		certificate,
		sslKey,
		xmlConfig,
		"--authenticate",
	}
	if config.UserCertificate != "" {
		parameters = append(parameters, mcaCertificate)
	}
	if config.UserKey != "" {
		parameters = append(parameters, mcaKey)
	}
	if config.Quiet {
		parameters = append(parameters, "--quiet")
	}
	if config.NoProxy {
		parameters = append(parameters, "--no-proxy")
	}
	if config.CACertificate != "" {
		parameters = append(parameters, caFile)
	}
	if config.User != "" {
		parameters = append(parameters, user)
	}
	if config.Password != "" {
		// read password from stdin and switch to non-interactive mode
		parameters = append(parameters, "--passwd-on-stdin")
		parameters = append(parameters, "--non-inter")
	}
	parameters = append(parameters, config.ExtraArgs...)
	parameters = append(parameters, config.VPNServer)

	command := execCommand(config.OpenConnect, parameters...)

	// run command: allow user input, show stderr, buffer stdout
	var b bytes.Buffer
	command.Stdin = os.Stdin
	if config.Password != "" {
		// disable user input, pass password via stdin
		command.Stdin = bytes.NewBufferString(config.Password)
	}
	command.Stdout = &b
	command.Stderr = os.Stderr
	command.Env = append(os.Environ(), config.ExtraEnv...)
	command.Env = append(command.Env, d.GetEnv()...)
	if err := command.Run(); err != nil {
		// TODO: handle failed program start?
		return err
	}

	// parse login info, cookie from command line in buffer:
	//
	// COOKIE=3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...
	// HOST=10.0.0.1
	// CONNECT_URL='https://vpnserver.example.com'
	// FINGERPRINT=469bb424ec8835944d30bc77c77e8fc1d8e23a42
	// RESOLVE='vpnserver.example.com:10.0.0.1'
	//
	login := &logininfo.LoginInfo{}
	login.Server = config.VPNServer
	scanner := bufio.NewScanner(&b)
	for scanner.Scan() {
		line := scanner.Text()
		login.ParseLine(line)
	}
	d.SetLogin(login)

	return nil
}

// Authenticate authenticates the client on the VPN server.
func (d *DBusClient) Authenticate() error {
	// check status
	if err := d.checkStatus(); err != nil {
		return err
	}

	// authenticate
	return authenticate(d)
}

// connect sends a connect request with login info to the daemon.
var connect = func(d *DBusClient) error {
	// call connect
	login := d.GetLogin()
	return d.conn.Object(dbusapi.Interface, dbusapi.Path).
		Call(dbusapi.MethodConnect, 0,
			login.Server,
			login.Cookie,
			login.Host,
			login.ConnectURL,
			login.Fingerprint,
			login.Resolve,
		).Store()
}

// Connect connects the client with the VPN server, requires successful
// authentication with Authenticate.
func (d *DBusClient) Connect() error {
	// check status
	if err := d.checkStatus(); err != nil {
		return err
	}

	// send login info to daemon
	return connect(d)
}

// disconnect sends a disconnect request to the daemon.
var disconnect = func(d *DBusClient) error {
	// call disconnect
	return d.conn.Object(dbusapi.Interface, dbusapi.Path).
		Call(dbusapi.MethodDisconnect, 0).Store()
}

// Disconnect disconnects the client from the VPN server.
func (d *DBusClient) Disconnect() error {
	// check status
	status, err := d.Query()
	if err != nil {
		return fmt.Errorf("could not query OC-Daemon: %w", err)
	}
	if !status.OCRunning.Running() {
		return fmt.Errorf("OpenConnect client is not running, nothing to do")
	}

	// disconnect
	return disconnect(d)
}

// dumpState sends a dump state request to the daemon.
var dumpState = func(d *DBusClient) (string, error) {
	// call dump state
	state := ""
	err := d.conn.Object(dbusapi.Interface, dbusapi.Path).
		Call(dbusapi.MethodDumpState, 0).Store(&state)
	return state, err
}

// DumpState returns the internal state of the OC-Daemon as string.
func (d *DBusClient) DumpState() (string, error) {
	return dumpState(d)
}

// Close closes the DBusClient.
func (d *DBusClient) Close() error {
	var err error

	if d.conn != nil {
		err = d.conn.Close()
	}

	if d.isSubscribed() {
		close(d.done)
		<-d.closed
	}

	return err
}

// NewDBusClient returns a new DBusClient.
func NewDBusClient(config *Config) (*DBusClient, error) {
	// connect to system bus
	conn, err := dbusConnectSystemBus()
	if err != nil {
		return nil, err
	}

	// create client
	client := &DBusClient{
		config:  config,
		conn:    conn,
		signals: make(chan *dbus.Signal, 10),
		updates: make(chan *vpnstatus.Status),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}

	return client, nil
}

// NewClient returns a new Client.
func NewClient(config *Config) (Client, error) {
	return NewDBusClient(config)
}
