package daemon

import (
	"net"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
	"github.com/telekom-mms/oc-daemon/internal/ocrunner"
	"github.com/telekom-mms/oc-daemon/internal/trafpol"
	"github.com/telekom-mms/oc-daemon/internal/vpnsetup"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
	"github.com/telekom-mms/oc-daemon/pkg/xmlprofile"
)

type socketServer struct{}

func (s *socketServer) Requests() chan *api.Request { return nil }
func (s *socketServer) Shutdown()                   {}
func (s *socketServer) Start() error                { return nil }
func (s *socketServer) Stop()                       {}

type dbusService struct{}

func (d *dbusService) Requests() chan *dbusapi.Request    { return nil }
func (d *dbusService) SetProperty(name string, value any) {}
func (d *dbusService) Start() error                       { return nil }
func (d *dbusService) Stop()                              {}

type tndDetector struct{}

func (t *tndDetector) SetServers(map[string]string)  {}
func (t *tndDetector) GetServers() map[string]string { return nil }
func (t *tndDetector) SetDialer(dialer *net.Dialer)  {}
func (t *tndDetector) GetDialer() *net.Dialer        { return nil }
func (t *tndDetector) Start() error                  { return nil }
func (t *tndDetector) Stop()                         {}
func (t *tndDetector) Probe()                        {}
func (t *tndDetector) Results() chan bool            { return nil }

type vpnSetup struct{}

func (v *vpnSetup) GetState() *vpnsetup.State       { return nil }
func (v *vpnSetup) Setup(conf *daemoncfg.Config)    {}
func (v *vpnSetup) Start()                          {}
func (v *vpnSetup) Stop()                           {}
func (v *vpnSetup) Teardown(conf *daemoncfg.Config) {}

type trafPolicer struct{}

func (t *trafPolicer) AddAllowedAddr(addr netip.Addr) bool    { return false }
func (t *trafPolicer) CPDStatus() <-chan bool                 { return nil }
func (t *trafPolicer) GetState() *trafpol.State               { return nil }
func (t *trafPolicer) RemoveAllowedAddr(addr netip.Addr) bool { return false }
func (t *trafPolicer) Start() error                           { return nil }
func (t *trafPolicer) Stop()                                  {}

type sleepMonitor struct{}

func (s *sleepMonitor) Events() chan bool { return nil }
func (s *sleepMonitor) Start() error      { return nil }
func (s *sleepMonitor) Stop()             {}

type ocRunner struct{}

func (o *ocRunner) Connect(config *daemoncfg.Config, env []string) {}
func (o *ocRunner) Disconnect()                                    {}
func (o *ocRunner) Events() chan *ocrunner.ConnectEvent            { return nil }
func (o *ocRunner) Start()                                         {}
func (o *ocRunner) Stop()                                          {}

type profMonitor struct{}

func (p *profMonitor) Start() error           { return nil }
func (p *profMonitor) Stop()                  {}
func (p *profMonitor) Updates() chan struct{} { return nil }

func TestTest(t *testing.T) {
	d := &Daemon{
		config:  daemoncfg.NewConfig(),
		status:  vpnstatus.New(),
		errors:  make(chan error, 1),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
		profile: xmlprofile.NewProfile(),

		server:   &socketServer{},
		dbus:     &dbusService{},
		tnd:      &tndDetector{},
		vpnsetup: &vpnSetup{},
		trafpol:  &trafPolicer{},
		sleepmon: &sleepMonitor{},
		runner:   &ocRunner{},
		profmon:  &profMonitor{},
	}
	go d.start()
	time.Sleep(time.Second)
	d.Stop()

	d = &Daemon{
		config:  daemoncfg.NewConfig(),
		status:  vpnstatus.New(),
		errors:  make(chan error, 1),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
		profile: xmlprofile.NewProfile(),

		server:   &socketServer{},
		dbus:     &dbusService{},
		tnd:      &tndDetector{},
		vpnsetup: &vpnSetup{},
		trafpol:  &trafPolicer{},
		sleepmon: &sleepMonitor{},
		runner:   &ocRunner{},
		profmon:  &profMonitor{},
	}
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
