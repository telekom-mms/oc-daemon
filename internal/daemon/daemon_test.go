package daemon

import (
	"fmt"
	"net"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/ocrunner"
	"github.com/telekom-mms/oc-daemon/internal/trafpol"
	"github.com/telekom-mms/oc-daemon/internal/vpnsetup"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
	"github.com/telekom-mms/oc-daemon/pkg/xmlprofile"
	"github.com/vishvananda/netlink"
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

func TestDaemonCheckTND(t *testing.T) {
	// no tnd
	d := getTestDaemon()
	d.tnd = nil
	d.checkTND()

	// start tnd
	// note: cannot start without systemd-resolved
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
	fmt.Println(d.checkTND())

	// start tnd, already running
	d = getTestDaemon()
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
	d.checkTND()

	// stop tnd
	d.profile.AutomaticVPNPolicy.TrustedHTTPSServerList = nil
	d.checkTND()
}

func TestDaemonCheckTrafPol(t *testing.T) {
	// no trafpol
	d := getTestDaemon()
	d.trafpol = nil
	d.checkTrafPol()

	d = getTestDaemon()
	d.checkTrafPol()

	d = getTestDaemon()
	d.disableTrafPol = true
	d.checkTrafPol()

	//
	d = getTestDaemon()
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	d.status.TrustedNetwork = vpnstatus.TrustedNetworkTrusted
	d.checkTrafPol()

	d = getTestDaemon()
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	d.checkTrafPol()

	rlu := devmon.RegisterLinkUpdates
	defer func() { devmon.RegisterLinkUpdates = rlu }()

	d = getTestDaemon()
	d.trafpol = nil
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	devmon.RegisterLinkUpdates = func(d *devmon.DevMon) (chan netlink.LinkUpdate, error) {
		return nil, fmt.Errorf("test error")
	}
	fmt.Println(d.checkTrafPol())

	d = getTestDaemon()
	d.trafpol = nil
	d.profile.AutomaticVPNPolicy.AlwaysOn.Flag = true
	devmon.RegisterLinkUpdates = func(d *devmon.DevMon) (chan netlink.LinkUpdate, error) {
		return nil, nil
	}
	d.serverIP = netip.MustParseAddr("192.168.1.1")
	fmt.Println(d.checkTrafPol())
}

func TestDaemonHandleClientRequest(t *testing.T) {
	d := getTestDaemon()
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeOK, nil)))
	api.ReadMessage(c1)

	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, nil)))
	api.ReadMessage(c1)

	confup := NewVPNConfigUpdate()
	b, _ := confup.JSON()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	api.ReadMessage(c1)

	confup.Reason = "pre-init"
	b, _ = confup.JSON()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	api.ReadMessage(c1)

	// TODO: updateVPNConfigDown cov
	confup.Reason = "disconnect"
	b, _ = confup.JSON()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	api.ReadMessage(c1)

	// TODO: handleVPNAttemptReconnect cov
	confup.Reason = "attempt-reconnect"
	b, _ = confup.JSON()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	api.ReadMessage(c1)

	// TODO: handleVPNReconnect cov
	confup.Reason = "reconnect"
	b, _ = confup.JSON()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	api.ReadMessage(c1)

	// TODO: updateVPNConfigUp cov
	confup.Reason = "connect"
	confup.Config = vpnconfig.New()
	b, _ = confup.JSON()
	go d.handleClientRequest(api.NewRequest(c2, api.NewMessage(api.TypeVPNConfigUpdate, b)))
	api.ReadMessage(c1)
}

func TestDaemonHandleDBusRequest(t *testing.T) {
	d := getTestDaemon()
	r := dbusapi.NewRequest("", make(chan struct{}))
	d.handleDBusRequest(r)

	// TODO: connectVPN cov
	r = dbusapi.NewRequest(dbusapi.RequestConnect, make(chan struct{}))
	r.Parameters = []any{"", "", "", "", "", ""}
	d.handleDBusRequest(r)

	// TODO: disconnectVPN cov
	r = dbusapi.NewRequest(dbusapi.RequestDisconnect, make(chan struct{}))
	d.handleDBusRequest(r)

	// dump state
	r = dbusapi.NewRequest(dbusapi.RequestDumpState, make(chan struct{}))
	d.handleDBusRequest(r)
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
}

func TestDaemonHandleSleepMonEvent(t *testing.T) {
	d := getTestDaemon()

	d.handleSleepMonEvent(true)

	d.status.OCRunning = vpnstatus.OCRunningRunning
	d.handleSleepMonEvent(false)
}

func TestDaemonHandleProfileUpdate(t *testing.T) {
	d := getTestDaemon()

	d.handleProfileUpdate()
	// TODO: setStatusServers cov
}

func TestDaemonHandleCPDStatusUpdate(t *testing.T) {
	d := getTestDaemon()

	d.handleCPDStatusUpdate(false)
	d.handleCPDStatusUpdate(false)

	d.handleCPDStatusUpdate(true)
	d.handleCPDStatusUpdate(true)
}

func TestDaemonStartStopTND(t *testing.T) {
	// without tnd
	d := getTestDaemon()
	d.tnd = nil
	d.checkTND()
	go d.start()
	d.Stop()

	// start tnd
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
	d.checkTND()

	// with tnd
	d = getTestDaemon()
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
	go d.start()
	d.tnd.(*tndDetector).r <- false
	d.tnd.(*tndDetector).r <- false
	d.tnd.(*tndDetector).r <- true
	d.tnd.(*tndDetector).r <- true
	d.Stop()
}

func TestTest(t *testing.T) {
	d := getTestDaemon()
	go d.start()

	//d.server.Requests() <- &api.Request{}

	//d.dbus.(*dbusService).r <- &dbusapi.Request{}

	d.tnd.(*tndDetector).r <- false
	d.tnd.(*tndDetector).r <- true
	d.tnd.(*tndDetector).r <- false
	d.tnd.(*tndDetector).r <- false

	//d.trafpol.(*trafPolicer).s <- true
	//d.trafpol.(*trafPolicer).s <- false

	d.sleepmon.(*sleepMonitor).e <- true
	d.sleepmon.(*sleepMonitor).e <- false

	d.runner.(*ocRunner).e <- &ocrunner.ConnectEvent{}

	d.profmon.(*profMonitor).u <- struct{}{}

	time.Sleep(time.Second)
	d.Stop()

	d = getTestDaemon()
	d.Start()
	time.Sleep(time.Second)
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
