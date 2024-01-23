package client

import (
	"errors"
	"reflect"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
)

// TestDBusClientSetGetConfig tests SetConfig and GetConfig of DBusClient
func TestDBusClientSetGetConfig(t *testing.T) {
	client := &DBusClient{}
	want := NewConfig()
	client.SetConfig(want)
	got := client.GetConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDBusClientSetGetEnv tests SetEnv and GetEnv of DBusClient
func TestDBusClientSetGetEnv(t *testing.T) {
	client := &DBusClient{}
	want := []string{"test=test"}
	client.SetEnv(want)
	got := client.GetEnv()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDBusClientSetGetLogin tests SetLogin and GetLogin of DBusClient
func TestDBusClientSetGetLogin(t *testing.T) {
	client := &DBusClient{}
	want := &logininfo.LoginInfo{}
	client.SetLogin(want)
	got := client.GetLogin()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDBusClientPing tests Ping of DBusClient
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

// TestDBusClientQuery tests Query of DBusClient
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
			dbusapi.PropertyConnectedAt:     dbus.MakeVariant(dbusapi.ConnectedAtInvalid),
			dbusapi.PropertyServers:         dbus.MakeVariant(dbusapi.ServersInvalid),
			dbusapi.PropertyOCRunning:       dbus.MakeVariant(dbusapi.OCRunningUnknown),
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
	connSignal = func(conn *dbus.Conn, ch chan<- *dbus.Signal) {}

	client := &DBusClient{
		updates: make(chan *vpnstatus.Status),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
	if _, err := client.Subscribe(); err != nil {
		t.Error(err)
	}
	client.Close()

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
	<-s
	client.signals <- &dbus.Signal{}
	client.signals <- &dbus.Signal{
		Path: dbusapi.Path,
		Name: dbusapi.PropertiesChanged,
		Body: []any{"", "", ""},
	}
	client.signals <- &dbus.Signal{
		Path: dbusapi.Path,
		Name: dbusapi.PropertiesChanged,
		Body: []any{dbusapi.Interface, "", ""},
	}
	client.signals <- &dbus.Signal{
		Path: dbusapi.Path,
		Name: dbusapi.PropertiesChanged,
		Body: []any{dbusapi.Interface, map[string]dbus.Variant{
			dbusapi.PropertyTrustedNetwork: dbus.MakeVariant("invalid"),
		}, ""},
	}
	client.signals <- &dbus.Signal{
		Path: dbusapi.Path,
		Name: dbusapi.PropertiesChanged,
		Body: []any{dbusapi.Interface, map[string]dbus.Variant{}, ""},
	}
	client.signals <- &dbus.Signal{
		Path: dbusapi.Path,
		Name: dbusapi.PropertiesChanged,
		Body: []any{dbusapi.Interface, map[string]dbus.Variant{}, []string{
			dbusapi.PropertyTrustedNetwork,
			dbusapi.PropertyConnectionState,
			dbusapi.PropertyIP,
			dbusapi.PropertyDevice,
			dbusapi.PropertyConnectedAt,
			dbusapi.PropertyServers,
			dbusapi.PropertyOCRunning,
			dbusapi.PropertyVPNConfig,
		}},
	}

	close(client.signals)
	client.Close()

	// test add match signal error
	client = &DBusClient{}

	connAddMatchSignal = func(*dbus.Conn, ...dbus.MatchOption) error {
		return errors.New("test error")
	}

	if _, err := client.Subscribe(); err == nil {
		t.Error("add match signal should return error")
	}

	// test query error
	client = &DBusClient{}

	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, errors.New("test error")
	}
	if _, err := client.Subscribe(); err == nil {
		t.Error("query error should return error")
	}

	// test double subscribe
	client = &DBusClient{}

	_, _ = client.Subscribe()
	if _, err := client.Subscribe(); err == nil {
		t.Error("doube subscribe should return error")
	}

}

// TestDBusClientAuthenticate tests Authenticate of DBusClient
func TestDBusClientAuthenticate(t *testing.T) {
	client := &DBusClient{}
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}
	want := &logininfo.LoginInfo{
		Cookie: "TestCookie",
	}
	authenticate = func(d *DBusClient) error {
		d.login = want
		return nil
	}
	err := client.Authenticate()
	if err != nil {
		t.Error(err)
	}
	got := client.GetLogin()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDBusClientConnect tests Connect of DBusClient
func TestDBusClientConnect(t *testing.T) {
	client := &DBusClient{}
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		return nil, nil
	}
	connect = func(d *DBusClient) error {
		return nil
	}
	err := client.Connect()
	if err != nil {
		t.Error(err)
	}
}

// TestDBusClientDisconnect tests Disconnect of DBusClient
func TestDBusClientDisconnect(t *testing.T) {
	client := &DBusClient{}
	status := vpnstatus.New()
	status.OCRunning = vpnstatus.OCRunningRunning
	query = func(*DBusClient) (map[string]dbus.Variant, error) {
		props := map[string]dbus.Variant{
			dbusapi.PropertyOCRunning: dbus.MakeVariant(dbusapi.OCRunningRunning),
		}
		return props, nil
	}
	disconnect = func(d *DBusClient) error {
		return nil
	}
	err := client.Disconnect()
	if err != nil {
		t.Error(err)
	}
}

// TestNewDBusClient tests NewDBusClient
func TestNewDBusClient(t *testing.T) {
	dbusConnectSystemBus = func() (*dbus.Conn, error) {
		return nil, nil
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

// TestNewClient tests NewClient
func TestNewClient(t *testing.T) {
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
