package client

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
)

// TestDBusClientSetGetConfig tests SetConfig and GetConfig of DBusClient.
func TestDBusClientSetGetConfig(t *testing.T) {
	client := &DBusClient{}
	want := NewConfig()
	client.SetConfig(want)
	got := client.GetConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDBusClientSetGetEnv tests SetEnv and GetEnv of DBusClient.
func TestDBusClientSetGetEnv(t *testing.T) {
	client := &DBusClient{}
	want := []string{"test=test"}
	client.SetEnv(want)
	got := client.GetEnv()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDBusClientSetGetLogin tests SetLogin and GetLogin of DBusClient.
func TestDBusClientSetGetLogin(t *testing.T) {
	client := &DBusClient{}
	want := &logininfo.LoginInfo{}
	client.SetLogin(want)
	got := client.GetLogin()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDBusClientPing tests Ping of DBusClient.
func TestDBusClientPing(t *testing.T) {
	client := &DBusClient{}
	ping = func(*DBusClient) error {
		return nil
	}
	err := client.Ping()
	if err != nil {
		t.Error(err)
	}
}

// TestDBusClientQuery tests Query of DBusClient.
func TestDBusClientQuery(t *testing.T) {
	// clean up after tests
	oldQuery := query
	defer func() { query = oldQuery }()

	// create test client
	client := &DBusClient{}

	// test query error
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, errors.New("test error")
	}
	if _, err := client.Query(); err == nil {
		t.Error(err)
	}

	// test empty properties
	want := vpnstatus.New()
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}
	got, err := client.Query()
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %p, want %p", got, want)
	}

	// test filled properties, valid properties
	for _, props := range []map[string]dbus.Variant{
		{
			dbusapi.PropertyTrustedNetwork:  dbus.MakeVariant(dbusapi.TrustedNetworkUnknown),
			dbusapi.PropertyConnectionState: dbus.MakeVariant(dbusapi.ConnectionStateUnknown),
			dbusapi.PropertyIP:              dbus.MakeVariant(dbusapi.IPInvalid),
			dbusapi.PropertyDevice:          dbus.MakeVariant(dbusapi.DeviceInvalid),
			dbusapi.PropertyServer:          dbus.MakeVariant(dbusapi.ServerInvalid),
			dbusapi.PropertyServerIP:        dbus.MakeVariant(dbusapi.ServerIPInvalid),
			dbusapi.PropertyConnectedAt:     dbus.MakeVariant(dbusapi.ConnectedAtInvalid),
			dbusapi.PropertyServers:         dbus.MakeVariant(dbusapi.ServersInvalid),
			dbusapi.PropertyOCRunning:       dbus.MakeVariant(dbusapi.OCRunningUnknown),
			dbusapi.PropertyTrafPolState:    dbus.MakeVariant(dbusapi.TrafPolStateUnknown),
			dbusapi.PropertyAllowedHosts:    dbus.MakeVariant(dbusapi.AllowedHostsInvalid),
			dbusapi.PropertyTNDState:        dbus.MakeVariant(dbusapi.TNDStateUnknown),
			dbusapi.PropertyTNDServers:      dbus.MakeVariant(dbusapi.TNDServersInvalid),
			dbusapi.PropertyVPNConfig:       dbus.MakeVariant(dbusapi.VPNConfigInvalid),
		},
		{
			dbusapi.PropertyVPNConfig: dbus.MakeVariant("{}"),
		},
	} {
		query = func(*DBusClient) (map[string]dbus.Variant, error) {
			return props, nil
		}
		if _, err = client.Query(); err != nil {
			t.Error(err)
		}
	}

	// test filled properties, invalid properties
	for _, props := range []map[string]dbus.Variant{
		{dbusapi.PropertyTrustedNetwork: dbus.MakeVariant("invalid")},
		{dbusapi.PropertyVPNConfig: dbus.MakeVariant(1.23)},
		{dbusapi.PropertyVPNConfig: dbus.MakeVariant(1234)},
	} {
		query = func(*DBusClient) (map[string]dbus.Variant, error) {
			return props, nil
		}
		if _, err := client.Query(); err == nil {
			t.Error("invalid property should return error")
		}
	}
}

// TestDBusClientSubscribe tests Subscribe of DBusClient.
func TestDBusClientSubscribe(t *testing.T) {
	// clean up after tests
	oldQuery := query
	defer func() { query = oldQuery }()
	oldAddMatchSignal := connAddMatchSignal
	defer func() { connAddMatchSignal = oldAddMatchSignal }()
	oldSignal := connSignal
	defer func() { connSignal = oldSignal }()

	// test without errors
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}
	connAddMatchSignal = func(*dbus.Conn, ...dbus.MatchOption) error {
		return nil
	}
	connSignal = func(*dbus.Conn, chan<- *dbus.Signal) {}

	client := &DBusClient{
		updates: make(chan *vpnstatus.Status),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
	if _, err := client.Subscribe(); err != nil {
		t.Error(err)
	}
	_ = client.Close()

	// test without errors, signals
	client = &DBusClient{
		signals: make(chan *dbus.Signal),
		updates: make(chan *vpnstatus.Status),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
	s, err := client.Subscribe()
	if err != nil {
		t.Error(err)
	}

	// read initial status
	<-s

	// handle signals
	for _, sig := range []*dbus.Signal{
		{},
		{
			Path: dbusapi.Path,
			Name: dbusapi.PropertiesChanged,
			Body: []any{"", "", ""},
		},
		{
			Path: dbusapi.Path,
			Name: dbusapi.PropertiesChanged,
			Body: []any{dbusapi.Interface, "", ""},
		},
		{
			Path: dbusapi.Path,
			Name: dbusapi.PropertiesChanged,
			Body: []any{dbusapi.Interface, map[string]dbus.Variant{
				dbusapi.PropertyTrustedNetwork: dbus.MakeVariant("invalid"),
			}, ""},
		},
		{
			Path: dbusapi.Path,
			Name: dbusapi.PropertiesChanged,
			Body: []any{dbusapi.Interface, map[string]dbus.Variant{}, ""},
		},
		{
			Path: dbusapi.Path,
			Name: dbusapi.PropertiesChanged,
			Body: []any{dbusapi.Interface, map[string]dbus.Variant{}, []string{
				dbusapi.PropertyTrustedNetwork,
				dbusapi.PropertyConnectionState,
				dbusapi.PropertyIP,
				dbusapi.PropertyDevice,
				dbusapi.PropertyServer,
				dbusapi.PropertyServerIP,
				dbusapi.PropertyConnectedAt,
				dbusapi.PropertyServers,
				dbusapi.PropertyOCRunning,
				dbusapi.PropertyTrafPolState,
				dbusapi.PropertyAllowedHosts,
				dbusapi.PropertyTNDState,
				dbusapi.PropertyTNDServers,
				dbusapi.PropertyVPNConfig,
			}},
		},
	} {
		client.signals <- sig
	}
	close(client.signals)
	_ = client.Close()

	// test add match signal error
	client = &DBusClient{}

	connAddMatchSignal = func(*dbus.Conn, ...dbus.MatchOption) error {
		return errors.New("test error")
	}

	if _, err := client.Subscribe(); err == nil {
		t.Error("subscribe with add match signal error should return error")
	}

	// test query error
	client = &DBusClient{}

	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, errors.New("test error")
	}
	if _, err := client.Subscribe(); err == nil {
		t.Error("subscribe with query error should return error")
	}

	// test double subscribe
	client = &DBusClient{}

	_, _ = client.Subscribe()
	if _, err := client.Subscribe(); err == nil {
		t.Error("double subscribe should return error")
	}
}

// TestDBusClientAuthenticate tests Authenticate of DBusClient.
func TestDBusClientAuthenticate(t *testing.T) {
	// clean up after tests
	oldQuery := query
	defer func() { query = oldQuery }()
	defer func() { execCommand = exec.Command }()

	// create test client
	conf := NewConfig()
	conf.UserCertificate = "/test/user-cert"
	conf.UserKey = "/test/user-key"
	conf.CACertificate = "/test/ca"
	conf.User = "test-user"
	conf.Password = "test-passwd"
	client := &DBusClient{config: conf}

	// test with status errors
	for _, v := range []map[string]dbus.Variant{
		nil,
		{dbusapi.PropertyTrustedNetwork: dbus.MakeVariant(dbusapi.TrustedNetworkTrusted)},
		{dbusapi.PropertyConnectionState: dbus.MakeVariant(dbusapi.ConnectionStateConnected)},
		{dbusapi.PropertyOCRunning: dbus.MakeVariant(dbusapi.OCRunningRunning)},
	} {
		query = func(*DBusClient) (map[string]dbus.Variant, error) {
			if v == nil {
				return nil, errors.New("test error")
			}
			return v, nil
		}
		if err := client.Authenticate(); err == nil {
			t.Errorf("authenticate should return error for %v", v)
		}
	}

	// test without status errors
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}

	// test with exec error
	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("")
	}
	if err := client.Authenticate(); err == nil {
		t.Error("authenticate should return error")
	}

	// test without exec error
	want := &logininfo.LoginInfo{
		Cookie: "TestCookie",
	}
	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("echo", "COOKIE=TestCookie")
	}
	if err := client.Authenticate(); err != nil {
		t.Errorf("authenticate returned error: %v", err)
	}
	got := client.GetLogin()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test cookie with spaces
	want = &logininfo.LoginInfo{
		Cookie: "Test cookie with spaces!",
	}
	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("echo", "COOKIE='Test cookie with spaces!'")
	}
	if err := client.Authenticate(); err != nil {
		t.Errorf("authenticate returned error: %v", err)
	}
	got = client.GetLogin()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDBusClientConnect tests Connect of DBusClient.
func TestDBusClientConnect(t *testing.T) {
	// clean up after tests
	oldQuery := query
	defer func() { query = oldQuery }()
	oldConnect := connect
	defer func() { connect = oldConnect }()

	// create test client
	client := &DBusClient{}

	// test with status errors
	for _, v := range []map[string]dbus.Variant{
		nil,
		{dbusapi.PropertyTrustedNetwork: dbus.MakeVariant(dbusapi.TrustedNetworkTrusted)},
		{dbusapi.PropertyConnectionState: dbus.MakeVariant(dbusapi.ConnectionStateConnected)},
		{dbusapi.PropertyOCRunning: dbus.MakeVariant(dbusapi.OCRunningRunning)},
	} {
		query = func(*DBusClient) (map[string]dbus.Variant, error) {
			if v == nil {
				return nil, errors.New("test error")
			}
			return v, nil
		}
		if err := client.Connect(); err == nil {
			t.Errorf("connect should return error for %v", v)
		}
	}

	// test without status errors
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}
	connect = func(*DBusClient) error {
		return nil
	}
	if err := client.Connect(); err != nil {
		t.Error(err)
	}
}

// TestDBusClientDisconnect tests Disconnect of DBusClient.
func TestDBusClientDisconnect(t *testing.T) {
	// clean up after tests
	oldQuery := query
	defer func() { query = oldQuery }()
	oldDisconnect := disconnect
	defer func() { disconnect = oldDisconnect }()

	// create test client
	client := &DBusClient{}

	// test with status errors
	for _, v := range []map[string]dbus.Variant{
		nil,
		{dbusapi.PropertyOCRunning: dbus.MakeVariant(dbusapi.OCRunningNotRunning)},
	} {
		query = func(*DBusClient) (map[string]dbus.Variant, error) {
			if v == nil {
				return nil, errors.New("test error")
			}
			return v, nil
		}
		if err := client.Disconnect(); err == nil {
			t.Errorf("disconnect should return error for %v", v)
		}
	}

	// test without status errors
	status := vpnstatus.New()
	status.OCRunning = vpnstatus.OCRunningRunning
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		props := map[string]dbus.Variant{
			dbusapi.PropertyOCRunning: dbus.MakeVariant(dbusapi.OCRunningRunning),
		}
		return props, nil
	}
	disconnect = func(*DBusClient) error {
		return nil
	}
	err := client.Disconnect()
	if err != nil {
		t.Error(err)
	}
}

// testRWC is a reader writer closer for testing.
type testRWC struct{}

func (t *testRWC) Read([]byte) (int, error)  { return 0, nil }
func (t *testRWC) Write([]byte) (int, error) { return 0, nil }
func (t *testRWC) Close() error              { return nil }

// TestNewDBusClient tests NewDBusClient.
func TestNewDBusClient(t *testing.T) {
	// clean up after tests
	oldSystemBus := dbusConnectSystemBus
	defer func() { dbusConnectSystemBus = oldSystemBus }()

	// test with system bus error
	dbusConnectSystemBus = func() (*dbus.Conn, error) {
		return nil, errors.New("test error")
	}

	if _, err := NewDBusClient(NewConfig()); err == nil {
		t.Error("system bus error should return error")
	}

	// test without errors
	conn, err := dbus.NewConn(&testRWC{})
	if err != nil {
		t.Fatal(err)
	}
	dbusConnectSystemBus = func() (*dbus.Conn, error) {
		return conn, nil
	}

	config := NewConfig()
	client, err := NewDBusClient(config)
	if err != nil {
		t.Error(err)
	}

	want := config
	got := client.GetConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if err := client.Close(); err != nil {
		t.Error(err)
	}
}

// TestNewClient tests NewClient.
func TestNewClient(t *testing.T) {
	// clean up after tests
	oldSystemBus := dbusConnectSystemBus
	defer func() { dbusConnectSystemBus = oldSystemBus }()

	// test without errors
	dbusConnectSystemBus = func() (*dbus.Conn, error) {
		return nil, nil
	}

	config := NewConfig()
	client, err := NewClient(config)
	if err != nil {
		t.Error(err)
	}

	want := config
	got := client.GetConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if err := client.Close(); err != nil {
		t.Error(err)
	}
}
