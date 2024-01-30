package client

import (
	"errors"
	"testing"

	"github.com/telekom-mms/oc-daemon/pkg/client"
	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
)

// testClient is a Client for testing.
type testClient struct {
	querErr error
	status  *vpnstatus.Status
	authErr error
	connErr error
	discErr error
	subsErr error
	subsCha chan *vpnstatus.Status
}

func (t *testClient) SetConfig(config *client.Config)            {}
func (t *testClient) GetConfig() *client.Config                  { return nil }
func (t *testClient) SetEnv(env []string)                        {}
func (t *testClient) GetEnv() []string                           { return nil }
func (t *testClient) SetLogin(login *logininfo.LoginInfo)        {}
func (t *testClient) GetLogin() *logininfo.LoginInfo             { return nil }
func (t *testClient) Ping() error                                { return nil }
func (t *testClient) Query() (*vpnstatus.Status, error)          { return t.status, t.querErr }
func (t *testClient) Subscribe() (chan *vpnstatus.Status, error) { return t.subsCha, t.subsErr }
func (t *testClient) Authenticate() error                        { return t.authErr }
func (t *testClient) Connect() error                             { return t.connErr }
func (t *testClient) Disconnect() error                          { return t.discErr }
func (t *testClient) Close() error                               { return nil }

// TestListServers tests listServers.
func TestListServers(t *testing.T) {
	defer func() { clientNewClient = client.NewClient }()

	// test with query error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{querErr: errors.New("test error")}, nil
	}

	if err := listServers(); err == nil {
		t.Error("query error should return error")
	}

	// test without error
	clientNewClient = func(*client.Config) (client.Client, error) {
		status := vpnstatus.New()
		status.Servers = []string{"server1", "server2"}
		return &testClient{status: status}, nil
	}

	if err := listServers(); err != nil {
		t.Error(err)
	}
}

// TestConnectVPN tests connectVPN.
func TestConnectVPN(t *testing.T) {
	defer func() { clientNewClient = client.NewClient }()

	// test with connect error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{connErr: errors.New("test error")}, nil
	}

	if err := connectVPN(); err == nil {
		t.Error("connect error should return error")
	}

	// test with authenticate error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{authErr: errors.New("test error")}, nil
	}

	if err := connectVPN(); err == nil {
		t.Error("authenticate error should return error")
	}
	// test without error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{}, nil
	}

	if err := connectVPN(); err != nil {
		t.Error(err)
	}
}

// TestDisconnectVPN tests disconnectVPN.
func TestDisconnectVPN(t *testing.T) {
	// test with disconnect error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{discErr: errors.New("test error")}, nil
	}

	if err := disconnectVPN(); err == nil {
		t.Error("disconnect error should return error")
	}

	// test without error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{}, nil
	}

	if err := disconnectVPN(); err != nil {
		t.Error(err)
	}
}

// TestReconnectVPN tests reconnectVPN.
func TestReconnectVPN(t *testing.T) {
	defer func() { clientNewClient = client.NewClient }()

	// test with query error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{querErr: errors.New("test error")}, nil
	}

	if err := reconnectVPN(); err == nil {
		t.Error("query error should return error")
	}

	// test with oc always running
	reconnectSleep = 0
	clientNewClient = func(*client.Config) (client.Client, error) {
		status := vpnstatus.New()
		status.OCRunning = vpnstatus.OCRunningRunning
		return &testClient{status: status}, nil
	}

	if err := reconnectVPN(); err == nil {
		t.Error("oc always running should return error")
	}
}

// TestGetStatus tests getStatus.
func TestGetStatus(t *testing.T) {
	defer func() { clientNewClient = client.NewClient }()

	// test with query error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{querErr: errors.New("test error")}, nil
	}

	if err := getStatus(); err == nil {
		t.Error("query error should return error")
	}

	// test with json, without verbose
	json = true
	verbose = false
	clientNewClient = func(*client.Config) (client.Client, error) {
		status := vpnstatus.New()
		status.Servers = []string{"server1", "server2"}
		return &testClient{status: status}, nil
	}

	if err := getStatus(); err != nil {
		t.Error(err)
	}

	// test without json, with verbose
	json = false
	verbose = true
	clientNewClient = func(*client.Config) (client.Client, error) {
		status := vpnstatus.New()
		status.Servers = []string{"server1", "server2"}
		return &testClient{status: status}, nil
	}

	if err := getStatus(); err != nil {
		t.Error(err)
	}

	// test without json, with verbose, with connectedAt and config
	json = false
	verbose = true
	clientNewClient = func(*client.Config) (client.Client, error) {
		status := vpnstatus.New()
		status.ConnectedAt = 1
		status.VPNConfig = vpnconfig.New()
		return &testClient{status: status}, nil
	}

	if err := getStatus(); err != nil {
		t.Error(err)
	}

	// test without json, without verbose, without connectedAt
	json = false
	verbose = false
	clientNewClient = func(*client.Config) (client.Client, error) {
		status := vpnstatus.New()
		status.Servers = []string{"server1", "server2"}
		return &testClient{status: status}, nil
	}

	if err := getStatus(); err != nil {
		t.Error(err)
	}
}

// TestMonitor tests monitor.
func TestMonitor(t *testing.T) {
	defer func() { clientNewClient = client.NewClient }()

	// test with subscribe error
	clientNewClient = func(*client.Config) (client.Client, error) {
		return &testClient{subsErr: errors.New("test error")}, nil
	}

	if err := monitor(); err == nil {
		t.Error("subscribe error should return error")
	}

	// test without error
	clientNewClient = func(*client.Config) (client.Client, error) {
		c := make(chan *vpnstatus.Status)
		go func() {
			c <- vpnstatus.New()
			close(c)
		}()
		return &testClient{subsCha: c}, nil
	}

	if err := monitor(); err != nil {
		t.Error(err)
	}
}
