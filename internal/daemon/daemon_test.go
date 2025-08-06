package daemon

import (
	"encoding/json"
	"encoding/xml"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
	"github.com/telekom-mms/oc-daemon/internal/ocrunner"
	"github.com/telekom-mms/oc-daemon/internal/trafpol"
	"github.com/telekom-mms/oc-daemon/internal/vpnsetup"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
	"github.com/telekom-mms/oc-daemon/pkg/xmlprofile"
	"github.com/telekom-mms/tnd/pkg/tnd"
)

// socketServer is a socket API server for testing.
type socketServer struct{ r chan *api.Request }

func (s *socketServer) Requests() chan *api.Request { return s.r }
func (s *socketServer) Shutdown()                   {}
func (s *socketServer) Start() error                { return nil }
func (s *socketServer) Stop()                       {}

// dbusService is a D-Bus API service for testing.
type dbusService struct{ r chan *dbusapi.Request }

func (d *dbusService) Requests() chan *dbusapi.Request { return d.r }
func (d *dbusService) SetProperty(string, any)         {}
func (d *dbusService) Start() error                    { return nil }
func (d *dbusService) Stop()                           {}

// tndDetector is a TND detector for testing.
type tndDetector struct{ r chan bool }

func (t *tndDetector) SetServers(map[string]string)  {}
func (t *tndDetector) GetServers() map[string]string { return nil }
func (t *tndDetector) SetDialer(*net.Dialer)         {}
func (t *tndDetector) GetDialer() *net.Dialer        { return nil }
func (t *tndDetector) Start() error                  { return nil }
func (t *tndDetector) Stop()                         {}
func (t *tndDetector) Probe()                        {}
func (t *tndDetector) Results() chan bool            { return t.r }

// vpnSetup is VPN Setup for testing.
type vpnSetup struct{}

func (v *vpnSetup) GetState() *vpnsetup.State  { return nil }
func (v *vpnSetup) Setup(*daemoncfg.Config)    {}
func (v *vpnSetup) Start()                     {}
func (v *vpnSetup) Stop()                      {}
func (v *vpnSetup) Teardown(*daemoncfg.Config) {}

// trafPolicer is TrafPol for testing.
type trafPolicer struct{ s chan bool }

func (t *trafPolicer) AddAllowedAddr(netip.Addr) bool    { return false }
func (t *trafPolicer) CPDStatus() <-chan bool            { return t.s }
func (t *trafPolicer) GetState() *trafpol.State          { return nil }
func (t *trafPolicer) RemoveAllowedAddr(netip.Addr) bool { return false }
func (t *trafPolicer) Start() error                      { return nil }
func (t *trafPolicer) Stop()                             {}

// sleepMonitor is SleepMon for testing.
type sleepMonitor struct{ e chan bool }

func (s *sleepMonitor) Events() chan bool { return s.e }
func (s *sleepMonitor) Start() error      { return nil }
func (s *sleepMonitor) Stop()             {}

// ocRunner is OC-Runner for testing.
type ocRunner struct{ e chan *ocrunner.ConnectEvent }

func (o *ocRunner) Connect(*daemoncfg.Config, []string) {}
func (o *ocRunner) Disconnect()                         {}
func (o *ocRunner) Events() chan *ocrunner.ConnectEvent { return o.e }
func (o *ocRunner) Start()                              {}
func (o *ocRunner) Stop()                               {}

// profMonitor is Profile monitor for testing.
type profMonitor struct{ u chan struct{} }

func (p *profMonitor) Start() error           { return nil }
func (p *profMonitor) Stop()                  {}
func (p *profMonitor) Updates() chan struct{} { return p.u }

// getTestDaemon returns a Daemon for testing.
func getTestDaemon() *Daemon {
	return &Daemon{
		config:  daemoncfg.NewConfig(),
		status:  vpnstatus.New(),
		errors:  make(chan error, 1),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
		profile: xmlprofile.NewProfile(),

		server:   &socketServer{r: make(chan *api.Request)},
		dbus:     &dbusService{r: make(chan *dbusapi.Request)},
		tnd:      &tndDetector{r: make(chan bool)},
		vpnsetup: &vpnSetup{},
		trafpol:  &trafPolicer{s: make(chan bool)},
		sleepmon: &sleepMonitor{e: make(chan bool)},
		runner:   &ocRunner{e: make(chan *ocrunner.ConnectEvent)},
		profmon:  &profMonitor{u: make(chan struct{})},
	}
}

// TestDaemonSetStatusTrustedNetwork tests setStatusTrustedNetwork of Daemon.
func TestDaemonSetStatusTrustedNetwork(t *testing.T) {
	d := getTestDaemon()
	for i, trusted := range []struct {
		v    bool
		want vpnstatus.TrustedNetwork
	}{
		{true, vpnstatus.TrustedNetworkTrusted},
		{true, vpnstatus.TrustedNetworkTrusted},
		{false, vpnstatus.TrustedNetworkNotTrusted},
		{false, vpnstatus.TrustedNetworkNotTrusted},
	} {
		d.setStatusTrustedNetwork(trusted.v)
		got := d.status.TrustedNetwork
		if got != trusted.want {
			t.Errorf("%d: set %t, got %d, want %d",
				i, trusted.v, got, trusted.want)
		}
	}
}

// TestDaemonSetStatusConnectionState tests setStatusConnectionState of Daemon.
func TestDaemonSetStatusConnectionState(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []vpnstatus.ConnectionState{
		vpnstatus.ConnectionStateDisconnected,
		vpnstatus.ConnectionStateDisconnected,
		vpnstatus.ConnectionStateConnecting,
		vpnstatus.ConnectionStateConnecting,
		vpnstatus.ConnectionStateConnected,
		vpnstatus.ConnectionStateConnected,
		vpnstatus.ConnectionStateDisconnecting,
		vpnstatus.ConnectionStateDisconnecting,
		vpnstatus.ConnectionStateUnknown,
		vpnstatus.ConnectionStateUnknown,
	} {
		d.setStatusConnectionState(want)
		got := d.status.ConnectionState
		if got != want {
			t.Errorf("%d: got %d, want %d", i, got, want)
		}
	}
}

// TestDaemonSetStatusIP tests setStatusIP of Daemon.
func TestDaemonSetStatusIP(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []string{
		"192.168.1.1",
		"192.168.1.1",
		"",
		"",
	} {
		d.setStatusIP(want)
		got := d.status.IP
		if got != want {
			t.Errorf("%d: got %s, want %s", i, got, want)
		}
	}
}

// TestDaemonSetStatusDevice tests setStatusDevice of Daemon.
func TestDaemonSetStatusDevice(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []string{
		"oc-daemon-tun0",
		"oc-daemon-tun0",
		"",
		"",
	} {
		d.setStatusDevice(want)
		got := d.status.Device
		if got != want {
			t.Errorf("%d: got %s, want %s", i, got, want)
		}
	}
}

// TestDaemonSetStatusServer tests setStatusServer of Daemon.
func TestDaemonSetStatusServer(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []string{
		"test-server1",
		"test-server1",
		"",
		"",
	} {
		d.setStatusServer(want)
		got := d.status.Server
		if got != want {
			t.Errorf("%d: got %s, want %s", i, got, want)
		}
	}
}

// TestDaemonSetStatusServerIP tests setStatusServerIP of Daemon.
func TestDaemonSetStatusServerIP(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []string{
		"10.0.0.1",
		"10.0.0.1",
		"",
		"",
	} {
		d.setStatusServerIP(want)
		got := d.status.ServerIP
		if got != want {
			t.Errorf("%d: got %s, want %s", i, got, want)
		}
	}
}

// TestDaemonSetStatusConnectedAt tests setStatusConnectedAt of Daemon.
func TestDaemonSetStatusConnectedAt(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []int64{
		1753274054,
		1753274054,
		0,
		0,
	} {
		d.setStatusConnectedAt(want)
		got := d.status.ConnectedAt
		if got != want {
			t.Errorf("%d: got %d, want %d", i, got, want)
		}
	}
}

// TestDaemonSetStatusServers tests setStatusServers of Daemon.
func TestDaemonSetStatusServers(t *testing.T) {
	d := getTestDaemon()
	for i, want := range [][]string{
		{"test-server1"},
		{"test-server1"},
		{"test-server1", "test-server2"},
		{"test-server1", "test-server2"},
		{},
		{},
	} {
		d.setStatusServers(want)
		got := d.status.Servers
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

// TestDaemonSetStatusOCRunning tests setStatusOCRunning of Daemon.
func TestDaemonSetStatusOCRunning(t *testing.T) {
	d := getTestDaemon()
	for i, trusted := range []struct {
		v    bool
		want vpnstatus.OCRunning
	}{
		{true, vpnstatus.OCRunningRunning},
		{true, vpnstatus.OCRunningRunning},
		{false, vpnstatus.OCRunningNotRunning},
		{false, vpnstatus.OCRunningNotRunning},
	} {
		d.setStatusOCRunning(trusted.v)
		got := d.status.OCRunning
		if got != trusted.want {
			t.Errorf("%d: set %t, got %d, want %d",
				i, trusted.v, got, trusted.want)
		}
	}
}

// TestDaemonSetStatusOCPID tests setStatusOCPID of Daemon.
func TestDaemonSetStatusOCPID(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []uint32{
		12345,
		12345,
		0,
		0,
	} {
		d.setStatusOCPID(want)
		got := d.status.OCPID
		if got != want {
			t.Errorf("%d: got %d, want %d", i, got, want)
		}
	}
}

// TestDaemonSetStatusTrafPolState tests setStatusTrafPolState of Daemon.
func TestDaemonSetStatusTrafPolState(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []vpnstatus.TrafPolState{
		vpnstatus.TrafPolStateInactive,
		vpnstatus.TrafPolStateInactive,
		vpnstatus.TrafPolStateActive,
		vpnstatus.TrafPolStateActive,
		vpnstatus.TrafPolStateDisabled,
		vpnstatus.TrafPolStateDisabled,
		vpnstatus.TrafPolStateUnknown,
		vpnstatus.TrafPolStateUnknown,
	} {
		d.setStatusTrafPolState(want)
		got := d.status.TrafPolState
		if got != want {
			t.Errorf("%d: got %d, want %d", i, got, want)
		}
	}
}

// TestDaemonSetStatusAllowedHosts tests setStatusAllowedHosts of Daemon.
func TestDaemonSetStatusAllowedHosts(t *testing.T) {
	d := getTestDaemon()
	for i, want := range [][]string{
		{"allowed-host1"},
		{"allowed-host1"},
		{"allowed-host1", "allowed-host2"},
		{"allowed-host1", "allowed-host2"},
		{},
		{},
	} {
		d.setStatusAllowedHosts(want)
		got := d.status.AllowedHosts
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

// TestDaemonSetStatusCaptivePortal tests setStatusCaptivePortal of Daemon.
func TestDaemonSetStatusCaptivePortal(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []vpnstatus.CaptivePortal{
		vpnstatus.CaptivePortalNotDetected,
		vpnstatus.CaptivePortalNotDetected,
		vpnstatus.CaptivePortalDetected,
		vpnstatus.CaptivePortalDetected,
		vpnstatus.CaptivePortalUnknown,
		vpnstatus.CaptivePortalUnknown,
	} {
		d.setStatusCaptivePortal(want)
		got := d.status.CaptivePortal
		if got != want {
			t.Errorf("%d: got %d, want %d", i, got, want)
		}
	}
}

// TestDaemonSetStatusTNDState tests setStatusTNDState of Daemon.
func TestDaemonSetStatusTNDState(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []vpnstatus.TNDState{
		vpnstatus.TNDStateInactive,
		vpnstatus.TNDStateInactive,
		vpnstatus.TNDStateActive,
		vpnstatus.TNDStateActive,
		vpnstatus.TNDStateUnknown,
		vpnstatus.TNDStateUnknown,
	} {
		d.setStatusTNDState(want)
		got := d.status.TNDState
		if got != want {
			t.Errorf("%d: got %d, want %d", i, got, want)
		}
	}
}

// TestDaemonSetStatusTNDServers tests setStatusTNDServers of Daemon.
func TestDaemonSetStatusTNDServers(t *testing.T) {
	d := getTestDaemon()
	for i, want := range [][]string{
		{"tnd-server1"},
		{"tnd-server1"},
		{"tnd-server1", "tnd-server2"},
		{"tnd-server1", "tnd-server2"},
		{},
		{},
	} {
		d.setStatusTNDServers(want)
		got := d.status.TNDServers
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

// TestDaemonSetStatusVPNConfig tests setStatusVPNConfig of Daemon.
func TestDaemonSetStatusVPNConfig(t *testing.T) {
	d := getTestDaemon()
	for i, want := range []*vpnconfig.Config{
		{},
		{},
		nil,
		nil,
	} {
		d.setStatusVPNConfig(want)
		got := d.status.VPNConfig
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

// TestDaemonCheck tests checkTND of Daemon.
func TestDaemonCheckTND(t *testing.T) {
	oldTndNewDetector := tndNewDetector
	defer func() { tndNewDetector = oldTndNewDetector }()
	tndNewDetector = func(*tnd.Config) tnd.TND {
		return &tndDetector{r: make(chan bool)}
	}

	// tnd servers
	tndServers := []xmlprofile.TrustedHTTPSServer{
		{
			Address:         "tnd1.mycompany.com",
			Port:            "443",
			CertificateHash: "hash of tnd1 certificate",
		},
		{
			Address:         "tnd2.mycompany.com",
			Port:            "443",
			CertificateHash: "hash of tnd2 certificate",
		},
	}

	for i, test := range []struct {
		tnd          tnd.TND
		tndServers   []xmlprofile.TrustedHTTPSServer
		wantTNDState vpnstatus.TNDState
	}{
		// check with no tnd servers and tnd not running
		{
			tnd:          nil,
			tndServers:   nil,
			wantTNDState: vpnstatus.TNDStateUnknown,
		},
		// check with tnd servers and tnd not running, start
		{
			tnd:          nil,
			tndServers:   tndServers,
			wantTNDState: vpnstatus.TNDStateActive,
		},
		// check again with tnd servers, tnd running, start again
		{
			tnd:          &tndDetector{r: make(chan bool)},
			tndServers:   tndServers,
			wantTNDState: vpnstatus.TNDStateUnknown,
		},
		// check again without tnd servers, stop tnd
		{
			tnd:          &tndDetector{r: make(chan bool)},
			tndServers:   nil,
			wantTNDState: vpnstatus.TNDStateInactive,
		},
	} {
		d := getTestDaemon()
		d.tnd = test.tnd
		d.profile.AutomaticVPNPolicy.TrustedHTTPSServerList = test.tndServers
		if err := d.checkTND(); err != nil {
			t.Error(err)
		}
		gotTNDState := d.status.TNDState
		if gotTNDState != test.wantTNDState {
			t.Errorf("%d: got TND state %d, want %d",
				i, gotTNDState, test.wantTNDState)
		}
	}
}

// TestDaemonCheckTrafPol tests checkTrafPol of Daemon.
func TestDaemonCheckTrafPol(t *testing.T) {
	// cleanup after tests
	oldTrafPolNewTafPol := trafpolNewTrafPol
	defer func() { trafpolNewTrafPol = oldTrafPolNewTafPol }()
	trafpolNewTrafPol = func(*daemoncfg.Config) trafpol.Policer {
		return &trafPolicer{s: make(chan bool)}
	}

	for i, test := range []struct {
		trafpol              trafpol.Policer
		serverIP             netip.Addr
		profileAlwaysOn      bool
		statusTrustedNetwork vpnstatus.TrustedNetwork
		statusTrafPol        vpnstatus.TrafPolState
		disableTrafPol       bool
		wantTrafPolState     vpnstatus.TrafPolState
	}{
		// check with TrafPol disabled and no running TrafPol
		{
			trafpol:              nil,
			serverIP:             netip.Addr{},
			profileAlwaysOn:      false,
			statusTrustedNetwork: vpnstatus.TrustedNetworkNotTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateInactive,
			disableTrafPol:       true,
			wantTrafPolState:     vpnstatus.TrafPolStateInactive,
		},
		// check with TrafPol disabled and running TrafPol, stop
		{
			trafpol:              &trafPolicer{s: make(chan bool)},
			serverIP:             netip.Addr{},
			profileAlwaysOn:      false,
			statusTrustedNetwork: vpnstatus.TrustedNetworkNotTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateActive,
			disableTrafPol:       true,
			wantTrafPolState:     vpnstatus.TrafPolStateDisabled,
		},
		// check without trafpol/alwayson setting and no running TrafPol
		{
			trafpol:              nil,
			serverIP:             netip.Addr{},
			profileAlwaysOn:      false,
			statusTrustedNetwork: vpnstatus.TrustedNetworkNotTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateInactive,
			disableTrafPol:       false,
			wantTrafPolState:     vpnstatus.TrafPolStateInactive,
		},
		// check without trafpol/alwayson setting and running TrafPol, stop
		{
			trafpol:              &trafPolicer{s: make(chan bool)},
			serverIP:             netip.Addr{},
			profileAlwaysOn:      false,
			statusTrustedNetwork: vpnstatus.TrustedNetworkNotTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateActive,
			disableTrafPol:       false,
			wantTrafPolState:     vpnstatus.TrafPolStateInactive,
		},
		// check with trusted network and no running TrafPol
		{
			trafpol:              nil,
			serverIP:             netip.Addr{},
			profileAlwaysOn:      true,
			statusTrustedNetwork: vpnstatus.TrustedNetworkTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateInactive,
			disableTrafPol:       false,
			wantTrafPolState:     vpnstatus.TrafPolStateInactive,
		},
		// check with trusted network and running TrafPol, stop
		{
			trafpol:              &trafPolicer{s: make(chan bool)},
			serverIP:             netip.Addr{},
			profileAlwaysOn:      true,
			statusTrustedNetwork: vpnstatus.TrustedNetworkTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateActive,
			disableTrafPol:       false,
			wantTrafPolState:     vpnstatus.TrafPolStateInactive,
		},
		// check with trafpol/alwayson setting and no running TrafPol, start
		{
			trafpol:              nil,
			serverIP:             netip.Addr{},
			profileAlwaysOn:      true,
			statusTrustedNetwork: vpnstatus.TrustedNetworkNotTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateInactive,
			disableTrafPol:       false,
			wantTrafPolState:     vpnstatus.TrafPolStateActive,
		},
		// check with trafpol/alwayson setting, server IP and no running TrafPol, start
		{
			trafpol:              nil,
			serverIP:             netip.MustParseAddr("10.0.0.1"),
			profileAlwaysOn:      true,
			statusTrustedNetwork: vpnstatus.TrustedNetworkNotTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateInactive,
			disableTrafPol:       false,
			wantTrafPolState:     vpnstatus.TrafPolStateActive,
		},
		// check with trafpol/alwayson setting and running TrafPol
		{
			trafpol:              &trafPolicer{s: make(chan bool)},
			serverIP:             netip.Addr{},
			profileAlwaysOn:      true,
			statusTrustedNetwork: vpnstatus.TrustedNetworkNotTrusted,
			statusTrafPol:        vpnstatus.TrafPolStateActive,
			disableTrafPol:       false,
			wantTrafPolState:     vpnstatus.TrafPolStateActive,
		},
	} {
		d := getTestDaemon()
		d.trafpol = test.trafpol
		d.serverIP = test.serverIP
		d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = test.profileAlwaysOn
		d.status.TrustedNetwork = test.statusTrustedNetwork
		d.status.TrafPolState = test.statusTrafPol
		d.disableTrafPol = test.disableTrafPol
		if err := d.checkTrafPol(); err != nil {
			t.Error(err)
		}
		gotTrafPolState := d.status.TrafPolState
		if gotTrafPolState != test.wantTrafPolState {
			t.Errorf("%d: got TrafPol state %d, want %d",
				i, gotTrafPolState, test.wantTrafPolState)
		}
	}
}

// TestDaemonHandleClientRequest tests handleClientRequest of Daemon.
func TestDaemonHandleClientRequest(t *testing.T) {
	// ok message
	d := getTestDaemon()
	c1, c2 := net.Pipe()
	defer func() { _ = c1.Close() }()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeOK, nil)))
	m, err := api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType := m.Type
	wantType := uint16(api.TypeOK)
	gotValue := string(m.Value)
	wantValue := ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// filled vpn config
	testVPNConfig := &vpnconfig.Config{
		Gateway: net.ParseIP("10.0.0.1"),
		Device: vpnconfig.Device{
			Name: "oc-daemon-tun0",
			MTU:  1300,
		},
		IPv4: vpnconfig.Address{
			Address: net.ParseIP("192.168.1.1"),
			Netmask: net.CIDRMask(24, 32),
		},
		DNS: vpnconfig.DNS{
			ServersIPv4: []net.IP{net.ParseIP("192.168.1.2")},
		},
	}

	// vpn config update messages
	for i, test := range []struct {
		configUpdate          *VPNConfigUpdate
		statusVPNConfig       *vpnconfig.Config
		statusOCRunning       vpnstatus.OCRunning
		statusConnectionState vpnstatus.ConnectionState
		wantType              uint16
		wantValue             string
		wantConfig            *vpnconfig.Config
		wantState             vpnstatus.ConnectionState
	}{
		// vpn config update, missing config
		{
			configUpdate:          nil,
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningUnknown,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeError,
			wantValue:             "invalid config update message",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, empty config
		{
			configUpdate:          NewVPNConfigUpdate(),
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningUnknown,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeError,
			wantValue:             "invalid config update in config update message",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, pre-init
		{
			configUpdate:          &VPNConfigUpdate{Reason: "pre-init"},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningUnknown,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, connect, config not changed
		{
			configUpdate:          &VPNConfigUpdate{Reason: "connect", Config: vpnconfig.New()},
			statusVPNConfig:       vpnconfig.New(),
			statusOCRunning:       vpnstatus.OCRunningUnknown,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            vpnconfig.New(),
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, connect, config changed, openconnect not running
		{
			configUpdate:          &VPNConfigUpdate{Reason: "connect", Config: vpnconfig.New()},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningUnknown,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, connect, config changed, openconnect running and already connected
		{
			configUpdate:          &VPNConfigUpdate{Reason: "connect", Config: vpnconfig.New()},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningRunning,
			statusConnectionState: vpnstatus.ConnectionStateConnected,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateConnected,
		},
		// vpn config update, connect, config changed, openconnect running and connecting
		{
			configUpdate:          &VPNConfigUpdate{Reason: "connect", Config: testVPNConfig},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningRunning,
			statusConnectionState: vpnstatus.ConnectionStateConnecting,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            testVPNConfig,
			wantState:             vpnstatus.ConnectionStateConnected,
		},
		// vpn config update, disconnect
		{
			configUpdate:          &VPNConfigUpdate{Reason: "disconnect"},
			statusVPNConfig:       vpnconfig.New(),
			statusOCRunning:       vpnstatus.OCRunningUnknown,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, disconnect, still connected
		{
			configUpdate:          &VPNConfigUpdate{Reason: "disconnect"},
			statusVPNConfig:       vpnconfig.New(),
			statusOCRunning:       vpnstatus.OCRunningUnknown,
			statusConnectionState: vpnstatus.ConnectionStateConnected,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            vpnconfig.New(),
			wantState:             vpnstatus.ConnectionStateConnected,
		},
		// vpn config update, attempt-reconnect, openconnect not running
		{
			configUpdate:          &VPNConfigUpdate{Reason: "attempt-reconnect"},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningNotRunning,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, attempt-reconnect, openconnect running, not connected
		{
			configUpdate:          &VPNConfigUpdate{Reason: "attempt-reconnect"},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningRunning,
			statusConnectionState: vpnstatus.ConnectionStateDisconnected,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateDisconnected,
		},
		// vpn config update, attempt-reconnect, openconnect running, connected
		{
			configUpdate:          &VPNConfigUpdate{Reason: "attempt-reconnect"},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningRunning,
			statusConnectionState: vpnstatus.ConnectionStateConnected,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateConnecting,
		},
		// vpn config update, reconnect, openconnect not running
		{
			configUpdate:          &VPNConfigUpdate{Reason: "reconnect"},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningUnknown,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, reconnect, openconnect running, not connecting
		{
			configUpdate:          &VPNConfigUpdate{Reason: "reconnect"},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningRunning,
			statusConnectionState: vpnstatus.ConnectionStateUnknown,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateUnknown,
		},
		// vpn config update, reconnect, openconnect running, connecting
		{
			configUpdate:          &VPNConfigUpdate{Reason: "reconnect"},
			statusVPNConfig:       nil,
			statusOCRunning:       vpnstatus.OCRunningRunning,
			statusConnectionState: vpnstatus.ConnectionStateConnecting,
			wantType:              api.TypeOK,
			wantValue:             "",
			wantConfig:            nil,
			wantState:             vpnstatus.ConnectionStateConnected,
		},
	} {
		d := getTestDaemon()
		d.status.VPNConfig = test.statusVPNConfig
		d.status.OCRunning = test.statusOCRunning
		d.status.ConnectionState = test.statusConnectionState

		b := []byte(nil)
		if test.configUpdate != nil {
			b, _ = test.configUpdate.JSON()
		}

		c1, c2 := net.Pipe()
		defer func() { _ = c1.Close() }()

		go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
		m, err := api.ReadMessage(c1)
		if err != nil {
			t.Error(err)
		}

		gotType := m.Type
		gotValue := string(m.Value)
		gotConfig := d.status.VPNConfig
		gotState := d.status.ConnectionState

		if gotType != test.wantType {
			t.Errorf("%d: got message type %d, want %d", i, gotType, test.wantType)
		}
		if gotValue != test.wantValue {
			t.Errorf("%d: got value \"%s\", want \"%s\"", i, gotValue, test.wantValue)
		}
		if !reflect.DeepEqual(gotConfig, test.wantConfig) {
			t.Errorf("%d: got config %v, want %v", i, gotConfig, test.wantConfig)
		}
		if gotState != test.wantState {
			t.Errorf("%d: got state %d, want %d", i, gotState, test.wantState)
		}
	}
}

// TestDaemonHandleDBusRequest tests handleDBusRequest of Daemon.
func TestDaemonHandleDBusRequest(t *testing.T) {
	for i, test := range []struct {
		request      string
		parameters   []any
		running      vpnstatus.OCRunning
		runner       ocrunner.Runner
		wantState    vpnstatus.ConnectionState
		wantServerIP netip.Addr
	}{
		// other
		{
			request:      "other",
			parameters:   nil,
			running:      vpnstatus.OCRunningUnknown,
			runner:       nil,
			wantState:    vpnstatus.ConnectionStateUnknown,
			wantServerIP: netip.Addr{}},

		// connect, openconnect running
		{
			request:      dbusapi.RequestConnect,
			parameters:   []any{"", "", "", "", "", ""},
			running:      vpnstatus.OCRunningRunning,
			runner:       &ocRunner{e: make(chan *ocrunner.ConnectEvent)},
			wantState:    vpnstatus.ConnectionStateUnknown,
			wantServerIP: netip.Addr{}},
		// connect, openconnect not running, login invalid
		{
			request:      dbusapi.RequestConnect,
			parameters:   []any{"", "", "", "", "", ""},
			running:      vpnstatus.OCRunningNotRunning,
			runner:       &ocRunner{e: make(chan *ocrunner.ConnectEvent)},
			wantState:    vpnstatus.ConnectionStateUnknown,
			wantServerIP: netip.Addr{}},
		// connect, openconnect not running, login valid, no server ip
		{
			request:      dbusapi.RequestConnect,
			parameters:   []any{"server", "cookie", "host", "connectURL", "fingerprint", "resolve"},
			running:      vpnstatus.OCRunningNotRunning,
			runner:       &ocRunner{e: make(chan *ocrunner.ConnectEvent)},
			wantState:    vpnstatus.ConnectionStateConnecting,
			wantServerIP: netip.Addr{}},
		// connect, openconnect not running, login valid, server ip
		{
			request:      dbusapi.RequestConnect,
			parameters:   []any{"server", "cookie", "10.0.0.1", "connectURL", "fingerprint", "resolve"},
			running:      vpnstatus.OCRunningNotRunning,
			runner:       &ocRunner{e: make(chan *ocrunner.ConnectEvent)},
			wantState:    vpnstatus.ConnectionStateConnecting,
			wantServerIP: netip.MustParseAddr("10.0.0.1")},

		// disconnect, openconnect not running
		{
			request:      dbusapi.RequestDisconnect,
			parameters:   nil,
			running:      vpnstatus.OCRunningNotRunning,
			runner:       &ocRunner{e: make(chan *ocrunner.ConnectEvent)},
			wantState:    vpnstatus.ConnectionStateUnknown,
			wantServerIP: netip.Addr{}},
		// disconnect, openconnect running, without runner
		{
			request:      dbusapi.RequestDisconnect,
			parameters:   nil,
			running:      vpnstatus.OCRunningRunning,
			runner:       nil,
			wantState:    vpnstatus.ConnectionStateDisconnecting,
			wantServerIP: netip.Addr{}},
		// disconnect, openconnect running, with runner
		{
			request:      dbusapi.RequestDisconnect,
			parameters:   nil,
			running:      vpnstatus.OCRunningRunning,
			runner:       &ocRunner{e: make(chan *ocrunner.ConnectEvent)},
			wantState:    vpnstatus.ConnectionStateDisconnecting,
			wantServerIP: netip.Addr{}},
	} {
		d := getTestDaemon()
		d.status.OCRunning = test.running
		d.runner = test.runner

		r := dbusapi.NewRequest(test.request, make(chan struct{}))
		r.Parameters = test.parameters

		go d.handleDBusRequest(r)
		r.Wait()

		gotState := d.status.ConnectionState
		if gotState != test.wantState {
			t.Errorf("%d: got state %d, want %d", i, gotState, test.wantState)
		}
		gotServerIP := d.serverIP
		if gotServerIP != test.wantServerIP {
			t.Errorf("%d: got server ip %s, want %s", i, gotServerIP, test.wantServerIP)
		}
	}

	// dump state
	d := getTestDaemon()
	r := dbusapi.NewRequest(dbusapi.RequestDumpState, make(chan struct{}))
	go d.handleDBusRequest(r)
	r.Wait()
	if len(r.Results) != 1 {
		t.Error("invalid dump state results")
	}
	if _, ok := r.Results[0].(string); !ok {
		t.Error("no string in dump state results")
	}
	state := make(map[string]any)
	if err := json.Unmarshal([]byte(r.Results[0].(string)), &state); err != nil {
		t.Error("no json string in dump state results")
	}
}

// TestDaemonHandleTNDResult tests handleTNDResult of Daemon.
func TestDaemonHandleTNDResult(t *testing.T) {
	for i, test := range []struct {
		result      bool
		running     vpnstatus.OCRunning
		wantTrusted vpnstatus.TrustedNetwork
		wantState   vpnstatus.ConnectionState
	}{
		// no trusted network, openconnect not running
		{
			result:      false,
			running:     vpnstatus.OCRunningNotRunning,
			wantTrusted: vpnstatus.TrustedNetworkNotTrusted,
			wantState:   vpnstatus.ConnectionStateUnknown,
		},
		// trusted network, openconnect running
		{
			result:      true,
			running:     vpnstatus.OCRunningRunning,
			wantTrusted: vpnstatus.TrustedNetworkTrusted,
			wantState:   vpnstatus.ConnectionStateDisconnecting,
		},
	} {
		d := getTestDaemon()
		d.status.OCRunning = test.running
		if err := d.handleTNDResult(test.result); err != nil {
			t.Errorf("%d: %v", i, err)
		}
		gotTrusted := d.status.TrustedNetwork
		if gotTrusted != test.wantTrusted {
			t.Errorf("%d: got trusted %d, want %d",
				i, gotTrusted, test.wantTrusted)
		}
		gotState := d.status.ConnectionState
		if gotState != test.wantState {
			t.Errorf("%d: got state %d, want %d",
				i, gotState, test.wantState)
		}
	}
}

// TestDaemonHandleRunnerEvent tests handleRunnerEvent of Daemon.
func TestDaemonHandleRunnerEvent(t *testing.T) {
	for i, test := range []struct {
		event           *ocrunner.ConnectEvent
		serverIPAllowed bool
		want            vpnstatus.OCRunning
	}{
		// connect event
		{
			event:           &ocrunner.ConnectEvent{Connect: true, PID: 1},
			serverIPAllowed: false,
			want:            vpnstatus.OCRunningRunning,
		},
		// disconnect event, no server IP allowed
		{
			event:           &ocrunner.ConnectEvent{},
			serverIPAllowed: false,
			want:            vpnstatus.OCRunningNotRunning,
		},
		// disconnect event, server IP allowed
		{
			event:           &ocrunner.ConnectEvent{},
			serverIPAllowed: true,
			want:            vpnstatus.OCRunningNotRunning,
		},
	} {
		d := getTestDaemon()
		d.serverIPAllowed = test.serverIPAllowed
		d.handleRunnerEvent(test.event)
		got := d.status.OCRunning
		if got != test.want {
			t.Errorf("%d: got running %d, want %d", i, got, test.want)
		}

	}
}

// TestDaemonHandleSleepMonEvent tests handleSleepMonEvent of Daemon.
func TestDaemonHandleSleepMonEvent(t *testing.T) {
	for i, test := range []struct {
		event   bool
		running vpnstatus.OCRunning
		want    vpnstatus.ConnectionState
	}{
		// sleep, openconnect not running
		{
			event:   true,
			running: vpnstatus.OCRunningNotRunning,
			want:    vpnstatus.ConnectionStateUnknown,
		},
		// resume, openconnect not running
		{
			event:   false,
			running: vpnstatus.OCRunningNotRunning,
			want:    vpnstatus.ConnectionStateUnknown,
		},
		// resume, openconnect running
		{
			event:   false,
			running: vpnstatus.OCRunningRunning,
			want:    vpnstatus.ConnectionStateDisconnecting,
		},
	} {
		d := getTestDaemon()
		d.status.OCRunning = test.running
		d.handleSleepMonEvent(test.event)
		got := d.status.ConnectionState
		if got != test.want {
			t.Errorf("%d: got state %d, want %d", i, got, test.want)
		}
	}
}

// TestDaemonHandleProfileUpdate tests handleProfileUpdate of Daemon.
func TestDaemonHandleProfileUpdate(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "profile.xml")

	for i, want := range [][]string{
		// no profile
		nil,
		// server in profile
		{"test"},
		// servers in profile
		{"test1", "test2"},
	} {
		if want != nil {
			// create profile with server(s)
			profile := xmlprofile.NewProfile()
			for _, server := range want {
				profile.ServerList.HostEntry = append(
					profile.ServerList.HostEntry,
					xmlprofile.HostEntry{HostName: server})
			}

			b, err := xml.Marshal(profile)
			if err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(file, b, 0600); err != nil {
				t.Fatal(err)
			}
		}

		// handle update
		d := getTestDaemon()
		d.config.OpenConnect.XMLProfile = file
		if err := d.handleProfileUpdate(); err != nil {
			t.Errorf("%d: %v", i, err)
		}
		got := d.status.Servers
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%d: got servers %v, want %v",
				i, got, want)
		}
	}
}

// TestDaemonHandleCPDStatusUpdate tests handleCPDStatusUpdate of Daemon.
func TestDaemonHandleCPDStatusUpdate(t *testing.T) {
	for i, test := range []struct {
		update bool
		want   vpnstatus.CaptivePortal
	}{
		// no captive portal
		{
			update: false,
			want:   vpnstatus.CaptivePortalNotDetected,
		},
		// captive portal
		{
			update: true,
			want:   vpnstatus.CaptivePortalDetected,
		},
	} {
		d := getTestDaemon()
		d.handleCPDStatusUpdate(test.update)
		got := d.status.CaptivePortal
		if got != test.want {
			t.Errorf("%d: got captive portal %d, want %d",
				i, got, test.want)
		}
	}
}

// TestDaemonStartStop tests Start and Stop of Daemon with some events.
func TestDaemonStartStop(t *testing.T) {
	// set testing functions and cleanup after tests
	oldTndNewDetector := tndNewDetector
	defer func() { tndNewDetector = oldTndNewDetector }()
	tndNewDetector = func(*tnd.Config) tnd.TND {
		return &tndDetector{r: make(chan bool)}
	}
	oldTrafPolNewTafPol := trafpolNewTrafPol
	defer func() { trafpolNewTrafPol = oldTrafPolNewTafPol }()
	trafpolNewTrafPol = func(*daemoncfg.Config) trafpol.Policer {
		return &trafPolicer{s: make(chan bool)}
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "profile.xml")

	// create test daemon
	d := getTestDaemon()
	d.profile = readXMLProfile(file)
	d.config.OpenConnect.XMLProfile = file
	d.tnd = nil
	d.trafpol = nil
	if err := d.Start(); err != nil {
		t.Fatal(err)
	}

	// dbus api request
	c1, c2 := net.Pipe()
	defer func() { _ = c1.Close() }()
	d.server.Requests() <- api.NewRequest(c2, api.NewMessage(api.TypeOK, nil))
	_, _ = api.ReadMessage(c1)

	r := dbusapi.NewRequest("other", make(chan struct{}))
	d.dbus.(*dbusService).r <- r

	// sleepmon events
	d.sleepmon.(*sleepMonitor).e <- true
	d.sleepmon.(*sleepMonitor).e <- false

	// oc runner event
	d.runner.(*ocRunner).e <- &ocrunner.ConnectEvent{}

	// profmon event
	// this changes trafpol and tnd in d
	d.profmon.(*profMonitor).u <- struct{}{}

	d.Stop()

	// create profile
	profile := xmlprofile.NewProfile()
	profile.AutomaticVPNPolicy.TrustedHTTPSServerList = []xmlprofile.TrustedHTTPSServer{
		{
			Address:         "tnd1.mycompany.com",
			Port:            "443",
			CertificateHash: "hash of tnd1 certificate",
		},
		{
			Address:         "tnd2.mycompany.com",
			Port:            "443",
			CertificateHash: "hash of tnd2 certificate",
		},
	}
	profile.AutomaticVPNPolicy.AlwaysOn.Flag = true

	b, err := xml.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file, b, 0600); err != nil {
		t.Fatal(err)
	}

	// create test daemon
	d = getTestDaemon()
	d.profile = readXMLProfile(file)
	d.config.OpenConnect.XMLProfile = file
	d.tnd = nil
	d.trafpol = nil
	if err := d.Start(); err != nil {
		t.Fatal(err)
	}

	// dbus api request
	c1, c2 = net.Pipe()
	defer func() { _ = c1.Close() }()
	d.server.Requests() <- api.NewRequest(c2, api.NewMessage(api.TypeOK, nil))
	_, _ = api.ReadMessage(c1)

	r = dbusapi.NewRequest("other", make(chan struct{}))
	d.dbus.(*dbusService).r <- r

	// sleepmon events
	d.sleepmon.(*sleepMonitor).e <- true
	d.sleepmon.(*sleepMonitor).e <- false

	// oc runner event
	d.runner.(*ocRunner).e <- &ocrunner.ConnectEvent{}

	// tnd event
	d.tnd.(*tndDetector).r <- false

	// trafpol/cpd event
	d.trafpol.(*trafPolicer).s <- false

	// profmon event
	// this changes trafpol and tnd in d
	d.profmon.(*profMonitor).u <- struct{}{}

	d.Stop()
}

// TestDaemonErrors tests Errors of Daemon.
func TestDaemonErrors(t *testing.T) {
	// create daemon
	c := daemoncfg.NewConfig()
	c.OpenConnect.XMLProfile = filepath.Join(t.TempDir(), "does-not-exist")
	d := NewDaemon(c)

	if d.Errors() == nil || d.Errors() != d.errors {
		t.Errorf("invalid errors channel: %v", d.Errors())
	}
}

// TestNewDaemon tests NewDaemon.
func TestNewDaemon(t *testing.T) {
	// create daemon
	c := daemoncfg.NewConfig()
	c.OpenConnect.XMLProfile = filepath.Join(t.TempDir(), "does-not-exist")
	d := NewDaemon(c)

	// check daemon
	if d == nil {
		t.Fatal("daemon is nil")
	}

	if d.config != c {
		t.Fatal("wrong config")
	}

	for i, s := range []any{
		d.server,
		d.dbus,
		d.sleepmon,
		d.vpnsetup,
		d.runner,
		d.status,
		d.errors,
		d.done,
		d.closed,
		d.profile,
		d.profmon,
	} {
		if s == nil {
			t.Errorf("%d: unexpected nil", i)
		}
	}
}
