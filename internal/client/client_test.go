package client

import (
	"errors"
	"testing"

	"github.com/telekom-mms/oc-daemon/pkg/client"
	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
)

type testClient struct {
	querErr error
	authErr error
	connErr error
	status  *vpnstatus.Status
}

func (t *testClient) SetConfig(config *client.Config)            {}
func (t *testClient) GetConfig() *client.Config                  { return nil }
func (t *testClient) SetEnv(env []string)                        {}
func (t *testClient) GetEnv() []string                           { return nil }
func (t *testClient) SetLogin(login *logininfo.LoginInfo)        {}
func (t *testClient) GetLogin() *logininfo.LoginInfo             { return nil }
func (t *testClient) Ping() error                                { return nil }
func (t *testClient) Query() (*vpnstatus.Status, error)          { return t.status, t.querErr }
func (t *testClient) Subscribe() (chan *vpnstatus.Status, error) { return nil, nil }
func (t *testClient) Authenticate() error                        { return t.authErr }
func (t *testClient) Connect() error                             { return t.connErr }
func (t *testClient) Disconnect() error                          { return nil }
func (t *testClient) Close() error                               { return nil }

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
		status := vpnstatus.New()
		status.Servers = []string{"server1", "server2"}
		return &testClient{status: status}, nil
	}

	if err := connectVPN(); err != nil {
		t.Error(err)
	}
}
