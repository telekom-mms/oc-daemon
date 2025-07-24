package daemon

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
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

type socketServer struct{ r chan *api.Request }

func (s *socketServer) Requests() chan *api.Request { return s.r }
func (s *socketServer) Shutdown()                   {}
func (s *socketServer) Start() error                { return nil }
func (s *socketServer) Stop()                       {}

type dbusService struct{ r chan *dbusapi.Request }

func (d *dbusService) Requests() chan *dbusapi.Request    { return d.r }
func (d *dbusService) SetProperty(name string, value any) {}
func (d *dbusService) Start() error                       { return nil }
func (d *dbusService) Stop()                              {}

type tndDetector struct{ r chan bool }

func (t *tndDetector) SetServers(map[string]string)  {}
func (t *tndDetector) GetServers() map[string]string { return nil }
func (t *tndDetector) SetDialer(dialer *net.Dialer)  {}
func (t *tndDetector) GetDialer() *net.Dialer        { return nil }
func (t *tndDetector) Start() error                  { return nil }
func (t *tndDetector) Stop()                         {}
func (t *tndDetector) Probe()                        {}
func (t *tndDetector) Results() chan bool            { return t.r }

type vpnSetup struct{}

func (v *vpnSetup) GetState() *vpnsetup.State       { return nil }
func (v *vpnSetup) Setup(conf *daemoncfg.Config)    {}
func (v *vpnSetup) Start()                          {}
func (v *vpnSetup) Stop()                           {}
func (v *vpnSetup) Teardown(conf *daemoncfg.Config) {}

type trafPolicer struct{ s chan bool }

func (t *trafPolicer) AddAllowedAddr(addr netip.Addr) bool    { return false }
func (t *trafPolicer) CPDStatus() <-chan bool                 { return t.s }
func (t *trafPolicer) GetState() *trafpol.State               { return nil }
func (t *trafPolicer) RemoveAllowedAddr(addr netip.Addr) bool { return false }
func (t *trafPolicer) Start() error                           { return nil }
func (t *trafPolicer) Stop()                                  {}

type sleepMonitor struct{ e chan bool }

func (s *sleepMonitor) Events() chan bool { return s.e }
func (s *sleepMonitor) Start() error      { return nil }
func (s *sleepMonitor) Stop()             {}

type ocRunner struct{ e chan *ocrunner.ConnectEvent }

func (o *ocRunner) Connect(config *daemoncfg.Config, env []string) {}
func (o *ocRunner) Disconnect()                                    {}
func (o *ocRunner) Events() chan *ocrunner.ConnectEvent            { return o.e }
func (o *ocRunner) Start()                                         {}
func (o *ocRunner) Stop()                                          {}

type profMonitor struct{ u chan struct{} }

func (p *profMonitor) Start() error           { return nil }
func (p *profMonitor) Stop()                  {}
func (p *profMonitor) Updates() chan struct{} { return p.u }

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
	tndNewDetector = func(config *tnd.Config) tnd.TND {
		return &tndDetector{r: make(chan bool)}
	}

	// TODO: test start error

	// check with no tnd tnd servers and tnd not running
	d := getTestDaemon()
	d.tnd = nil
	d.checkTND()

	// check with tnd servers and tnd not running, start
	d = getTestDaemon()
	d.tnd = nil
	d.profile.AutomaticVPNPolicy.TrustedHTTPSServerList = []xmlprofile.TrustedHTTPSServer{
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
	if err := d.checkTND(); err != nil {
		t.Error(err)
	}
	if d.status.TNDState != vpnstatus.TNDStateActive {
		t.Errorf("TND State should be %d but it's %d",
			vpnstatus.TNDStateActive, d.status.TNDState)
	}

	// check again with tnd servers, tnd running, start again
	if err := d.checkTND(); err != nil {
		t.Error(err)
	}
	if d.status.TNDState != vpnstatus.TNDStateActive {
		t.Errorf("TND State should be %d but it's %d",
			vpnstatus.TNDStateActive, d.status.TNDState)
	}

	// check again without tnd servers, stop tnd
	d.profile.AutomaticVPNPolicy.TrustedHTTPSServerList = nil
	if err := d.checkTND(); err != nil {
		t.Error(err)
	}
	if d.status.TNDState != vpnstatus.TNDStateInactive {
		t.Errorf("TND State should be %d but it's %d",
			vpnstatus.TNDStateInactive, d.status.TNDState)
	}

	// check again without tnd servers, not running, stop again
	if err := d.checkTND(); err != nil {
		t.Error(err)
	}
	if d.status.TNDState != vpnstatus.TNDStateInactive {
		t.Errorf("TND State should be %d but is %d",
			vpnstatus.TNDStateInactive, d.status.TNDState)
	}
}

// TestDaemonCheckTrafPol tests checkTrafPol of Daemon.
func TestDaemonCheckTrafPol(t *testing.T) {
	// cleanup after tests
	oldTrafPolNewTafPol := trafpolNewTrafPol
	defer func() { trafpolNewTrafPol = oldTrafPolNewTafPol }()
	trafpolNewTrafPol = func(c *daemoncfg.Config) trafpol.Policer {
		return &trafPolicer{s: make(chan bool)}
	}

	// TODO: test start error

	// check with TrafPol disabled and no running TrafPol
	d := getTestDaemon()
	d.trafpol = nil
	d.status.TrafPolState = vpnstatus.TrafPolStateInactive
	d.disableTrafPol = true
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateInactive {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateInactive, d.status.TrafPolState)
	}

	// check with TrafPol disabled and running TrafPol, stop
	d = getTestDaemon()
	d.disableTrafPol = true
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateDisabled {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateDisabled, d.status.TrafPolState)
	}

	// check without trafpol/alwayson setting and no running TrafPol
	d = getTestDaemon()
	d.trafpol = nil
	d.status.TrafPolState = vpnstatus.TrafPolStateInactive
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateInactive {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateInactive, d.status.TrafPolState)
	}

	// check without trafpol/alwayson setting and running TrafPol, stop
	d = getTestDaemon()
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateInactive {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateInactive, d.status.TrafPolState)
	}

	// check with trusted network and no running TrafPol
	d = getTestDaemon()
	d.trafpol = nil
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	d.status.TrustedNetwork = vpnstatus.TrustedNetworkTrusted
	d.status.TrafPolState = vpnstatus.TrafPolStateInactive
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateInactive {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateInactive, d.status.TrafPolState)
	}

	// check with trusted network and running TrafPol, stop
	d = getTestDaemon()
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	d.status.TrustedNetwork = vpnstatus.TrustedNetworkTrusted
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateInactive {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateInactive, d.status.TrafPolState)
	}

	// check with trafpol/alwayson setting and no running TrafPol, start
	d = getTestDaemon()
	d.trafpol = nil
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateActive {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateActive, d.status.TrafPolState)
	}

	// check with trafpol/alwayson setting, server IP and no running TrafPol, start
	d = getTestDaemon()
	d.trafpol = nil
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	d.serverIP = netip.MustParseAddr("10.0.0.1")
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateActive {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateActive, d.status.TrafPolState)
	}

	// check with trafpol/alwayson setting and running TrafPol
	d = getTestDaemon()
	d.status.TrafPolState = vpnstatus.TrafPolStateActive
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	if err := d.checkTrafPol(); err != nil {
		t.Error(err)
	}
	if d.status.TrafPolState != vpnstatus.TrafPolStateActive {
		t.Errorf("TrafPol State should be %d but is %d",
			vpnstatus.TrafPolStateActive, d.status.TrafPolState)
	}
}

// TestDaemonHandleClientRequest tests handleClientRequest of Daemon.
func TestDaemonHandleClientRequest(t *testing.T) {
	// ok
	d := getTestDaemon()
	c1, c2 := net.Pipe()
	defer c1.Close()
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

	// vpn config update, missing config
	d = getTestDaemon()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, nil)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeError)
	gotValue = string(m.Value)
	wantValue = "invalid config update message"
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, empty config
	d = getTestDaemon()
	confup := NewVPNConfigUpdate()
	b, _ := confup.JSON()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeError)
	gotValue = string(m.Value)
	wantValue = "invalid config update message"
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got error %s, want %s", gotValue, wantValue)
	}

	// vpn config update, pre-init
	d = getTestDaemon()
	confup = NewVPNConfigUpdate()
	confup.Reason = "pre-init"
	b, _ = confup.JSON()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, connect, config not changed
	d = getTestDaemon()
	confup = NewVPNConfigUpdate()
	confup.Reason = "connect"
	confup.Config = vpnconfig.New()
	d.status.VPNConfig = confup.Config
	b, _ = confup.JSON()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, connect, config changed, openconnect not running
	d.status.VPNConfig = nil
	b, _ = confup.JSON()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, connect, config changed, openconnect running and already connected
	d.status.VPNConfig = nil
	d.status.OCRunning = vpnstatus.OCRunningRunning
	d.status.ConnectionState = vpnstatus.ConnectionStateConnected
	b, _ = confup.JSON()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, connect, config changed, openconnect running and connecting
	d.status.VPNConfig = nil
	d.status.OCRunning = vpnstatus.OCRunningRunning
	d.status.ConnectionState = vpnstatus.ConnectionStateConnecting
	confup.Config.Gateway = net.ParseIP("10.0.0.1")
	confup.Config.Device.Name = "oc-daemon-tun0"
	confup.Config.Device.MTU = 1300
	confup.Config.IPv4.Address = net.ParseIP("192.168.1.1")
	confup.Config.IPv4.Netmask = net.CIDRMask(24, 32)
	confup.Config.DNS.ServersIPv4 = []net.IP{net.ParseIP("192.168.1.2")}
	fmt.Println("### IPv4:", confup.Config.IPv4.Address, confup.Config.IPv4.Netmask)
	b, _ = confup.JSON()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}
	gotState := d.status.ConnectionState
	wantState := vpnstatus.ConnectionStateConnected
	if gotState != wantState {
		t.Errorf("got state %d, want %d", gotState, wantState)
	}

	// vpn config update, disconnect
	d = getTestDaemon()
	confup = NewVPNConfigUpdate()
	confup.Reason = "disconnect"
	b, _ = confup.JSON()
	d.status.VPNConfig = vpnconfig.New()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}
	gotConfig := d.status.VPNConfig
	wantConfig := (*vpnconfig.Config)(nil)
	if gotConfig != wantConfig {
		t.Errorf("got state %p, want %p", gotConfig, wantConfig)
	}

	// vpn config update, disconnect, still connected
	d = getTestDaemon()
	d.status.ConnectionState = vpnstatus.ConnectionStateConnected
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, attempt-reconnect, openconnect not running
	d = getTestDaemon()
	confup = NewVPNConfigUpdate()
	confup.Reason = "attempt-reconnect"
	b, _ = confup.JSON()
	d.status.OCRunning = vpnstatus.OCRunningNotRunning
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, attempt-reconnect, openconnect running, not connected
	d = getTestDaemon()
	d.status.OCRunning = vpnstatus.OCRunningRunning
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, attempt-reconnect, openconnect running, connected
	d = getTestDaemon()
	d.status.OCRunning = vpnstatus.OCRunningRunning
	d.status.ConnectionState = vpnstatus.ConnectionStateConnected
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}
	gotState = d.status.ConnectionState
	wantState = vpnstatus.ConnectionStateConnecting
	if gotState != wantState {
		t.Errorf("got state %d, want %d", gotState, wantState)
	}

	// vpn config update, reconnect, openconnect not running
	d = getTestDaemon()
	confup = NewVPNConfigUpdate()
	confup.Reason = "reconnect"
	b, _ = confup.JSON()
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, reconnect, openconnect running, not connecting
	d = getTestDaemon()
	d.status.OCRunning = vpnstatus.OCRunningRunning
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}

	// vpn config update, reconnect, openconnect running, connecting
	d = getTestDaemon()
	d.status.OCRunning = vpnstatus.OCRunningRunning
	d.status.ConnectionState = vpnstatus.ConnectionStateConnecting
	c1, c2 = net.Pipe()
	defer c1.Close()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	m, err = api.ReadMessage(c1)
	if err != nil {
		t.Error(err)
	}
	gotType = m.Type
	wantType = uint16(api.TypeOK)
	gotValue = string(m.Value)
	wantValue = ""
	if gotType != wantType {
		t.Errorf("got message type %d, want %d", gotType, wantType)
	}
	if gotValue != wantValue {
		t.Errorf("got value %s, want %s", gotValue, wantValue)
	}
	gotState = d.status.ConnectionState
	wantState = vpnstatus.ConnectionStateConnected
	if gotState != wantState {
		t.Errorf("got state %d, want %d", gotState, wantState)
	}
}

// TestDaemonHandleDBusRequest tests handleDBusRequest of Daemon.
func TestDaemonHandleDBusRequest(t *testing.T) {
	// other
	d := getTestDaemon()
	r := dbusapi.NewRequest("", make(chan struct{}))
	go d.handleDBusRequest(r)
	r.Wait()

	// connect, openconnect running
	d = getTestDaemon()
	r = dbusapi.NewRequest(dbusapi.RequestConnect, make(chan struct{}))
	r.Parameters = []any{"", "", "", "", "", ""}
	d.status.OCRunning = vpnstatus.OCRunningRunning
	go d.handleDBusRequest(r)
	r.Wait()

	// connect, openconnect not running, login invalid
	d = getTestDaemon()
	r = dbusapi.NewRequest(dbusapi.RequestConnect, make(chan struct{}))
	r.Parameters = []any{"", "", "", "", "", ""}
	d.status.OCRunning = vpnstatus.OCRunningNotRunning
	go d.handleDBusRequest(r)
	r.Wait()

	// connect, openconnect not running, login valid, no server ip
	d = getTestDaemon()
	r = dbusapi.NewRequest(dbusapi.RequestConnect, make(chan struct{}))
	r.Parameters = []any{"server", "cookie", "host", "connectURL", "fingerprint", "resolve"}
	d.status.OCRunning = vpnstatus.OCRunningNotRunning
	go d.handleDBusRequest(r)
	r.Wait()
	gotState := d.status.ConnectionState
	wantState := vpnstatus.ConnectionStateConnecting
	if gotState != wantState {
		t.Errorf("got state %d, want %d", gotState, wantState)
	}

	// connect, openconnect not running, login valid, server ip
	d = getTestDaemon()
	r = dbusapi.NewRequest(dbusapi.RequestConnect, make(chan struct{}))
	r.Parameters = []any{"server", "cookie", "10.0.0.1", "connectURL", "fingerprint", "resolve"}
	d.status.OCRunning = vpnstatus.OCRunningNotRunning
	go d.handleDBusRequest(r)
	r.Wait()
	gotState = d.status.ConnectionState
	wantState = vpnstatus.ConnectionStateConnecting
	if gotState != wantState {
		t.Errorf("got state %d, want %d", gotState, wantState)
	}
	gotServerIP := d.serverIP
	wantServerIP := netip.MustParseAddr("10.0.0.1")
	if gotServerIP != wantServerIP {
		t.Errorf("got server ip %s, want %s", gotServerIP, wantServerIP)
	}

	// disconnect, openconnect not running
	d = getTestDaemon()
	r = dbusapi.NewRequest(dbusapi.RequestDisconnect, make(chan struct{}))
	go d.handleDBusRequest(r)
	r.Wait()

	// disconnect, openconnect running, without runner
	d = getTestDaemon()
	d.status.OCRunning = vpnstatus.OCRunningRunning
	d.runner = nil
	r = dbusapi.NewRequest(dbusapi.RequestDisconnect, make(chan struct{}))
	go d.handleDBusRequest(r)
	r.Wait()
	gotState = d.status.ConnectionState
	wantState = vpnstatus.ConnectionStateDisconnecting
	if gotState != wantState {
		t.Errorf("got state %d, want %d", gotState, wantState)
	}

	// disconnect, openconnect running, with runner
	d = getTestDaemon()
	d.status.OCRunning = vpnstatus.OCRunningRunning
	r = dbusapi.NewRequest(dbusapi.RequestDisconnect, make(chan struct{}))
	go d.handleDBusRequest(r)
	r.Wait()
	gotState = d.status.ConnectionState
	wantState = vpnstatus.ConnectionStateDisconnecting
	if gotState != wantState {
		t.Errorf("got state %d, want %d", gotState, wantState)
	}

	// dump state
	d = getTestDaemon()
	r = dbusapi.NewRequest(dbusapi.RequestDumpState, make(chan struct{}))
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

func TestDaemonHandleTNDResult(t *testing.T) {
	d := getTestDaemon()
	d.handleTNDResult(false)
	d.handleTNDResult(false)

	d.status.OCRunning = vpnstatus.OCRunningRunning
	d.handleTNDResult(true)
}

func TestDaemonHandleRunnerEvent(t *testing.T) {
	d := getTestDaemon()

	d.handleRunnerEvent(&ocrunner.ConnectEvent{Connect: true, PID: 1})

	d.handleRunnerEvent(&ocrunner.ConnectEvent{Connect: true, PID: 1})

	d.serverIPAllowed = true
	d.handleRunnerEvent(&ocrunner.ConnectEvent{})

	// double disconnect event, should not happen? TODO: check/remove?
	//d.handleRunnerEvent(&ocrunner.ConnectEvent{})
}

func TestDaemonHandleSleepMonEvent(t *testing.T) {
	d := getTestDaemon()

	d.handleSleepMonEvent(true)

	d.status.OCRunning = vpnstatus.OCRunningRunning
	d.handleSleepMonEvent(false)
}

func TestDaemonHandleProfileUpdate(t *testing.T) {
	d := getTestDaemon()

	// empty profile
	d.handleProfileUpdate()

	// server in profile
	profile := xmlprofile.NewProfile()
	profile.ServerList.HostEntry = []xmlprofile.HostEntry{
		{HostName: "test"},
	}

	b, err := xml.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "profile.xml")
	if err := os.WriteFile(file, b, 0600); err != nil {
		t.Fatal(err)
	}

	d.config.OpenConnect.XMLProfile = file
	d.handleProfileUpdate()
}

func TestDaemonHandleCPDStatusUpdate(t *testing.T) {
	d := getTestDaemon()

	d.handleCPDStatusUpdate(false)
	d.handleCPDStatusUpdate(false)

	d.handleCPDStatusUpdate(true)
	d.handleCPDStatusUpdate(true)
}

//func TestDaemonStartStopTND(t *testing.T) {
//	// without tnd
//	d := getTestDaemon()
//	d.tnd = nil
//	d.checkTND()
//	go d.start()
//	d.Stop()
//
//	// start tnd
//	d = getTestDaemon()
//	d.tnd = nil
//	d.profile.AutomaticVPNPolicy.TrustedHTTPSServerList = []xmlprofile.TrustedHTTPSServer{
//		{
//			Address:         "tnd1.mycompany.com",
//			Port:            "443",
//			CertificateHash: "hash of tnd1 certificate",
//		},
//		{
//			Address:         "tnd2.mycompany.com",
//			Port:            "443",
//			CertificateHash: "hash of tnd2 certificate",
//		},
//	}
//	d.checkTND()
//
//	// with tnd
//	d = getTestDaemon()
//	d.profile.AutomaticVPNPolicy.TrustedHTTPSServerList = []xmlprofile.TrustedHTTPSServer{
//		{
//			Address:         "tnd1.mycompany.com",
//			Port:            "443",
//			CertificateHash: "hash of tnd1 certificate",
//		},
//		{
//			Address:         "tnd2.mycompany.com",
//			Port:            "443",
//			CertificateHash: "hash of tnd2 certificate",
//		},
//	}
//	go d.start()
//	d.tnd.(*tndDetector).r <- false
//	d.tnd.(*tndDetector).r <- false
//	d.tnd.(*tndDetector).r <- true
//	d.tnd.(*tndDetector).r <- true
//	d.Stop()
//}
//
//func TestTest(t *testing.T) {
//	d := getTestDaemon()
//	go d.start()
//
//	//d.server.Requests() <- &api.Request{}
//
//	//d.dbus.(*dbusService).r <- &dbusapi.Request{}
//
//	d.tnd.(*tndDetector).r <- false
//	d.tnd.(*tndDetector).r <- true
//	d.tnd.(*tndDetector).r <- false
//	d.tnd.(*tndDetector).r <- false
//
//	//d.trafpol.(*trafPolicer).s <- true
//	//d.trafpol.(*trafPolicer).s <- false
//
//	d.sleepmon.(*sleepMonitor).e <- true
//	d.sleepmon.(*sleepMonitor).e <- false
//
//	d.runner.(*ocRunner).e <- &ocrunner.ConnectEvent{}
//
//	d.profmon.(*profMonitor).u <- struct{}{}
//
//	time.Sleep(time.Second)
//	d.Stop()
//
//	d = getTestDaemon()
//	d.Start()
//	time.Sleep(time.Second)
//	d.Stop()
//}

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
