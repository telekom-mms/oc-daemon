package client

import (
	"reflect"
	"testing"

	"github.com/T-Systems-MMS/oc-daemon/internal/dbusapi"
	"github.com/T-Systems-MMS/oc-daemon/pkg/logininfo"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnstatus"
	"github.com/godbus/dbus/v5"
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

// TestDBusClientQuery tests Query of DBusClient
func TestDBusClientQuery(t *testing.T) {
	client := &DBusClient{}
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
