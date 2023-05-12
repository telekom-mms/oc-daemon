package client

import (
	"fmt"

	"github.com/T-Systems-MMS/oc-daemon/internal/api"
	"github.com/T-Systems-MMS/oc-daemon/internal/ocrunner"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnstatus"
)

const (
	// runDir is the daemons run dir
	runDir = "/run/oc-daemon"

	// daemon socket file
	sockFile = runDir + "/daemon.sock"
)

// Client is a VPN client
type Client struct {
	ClientCertificate string
	ClientKey         string
	CACertificate     string
	XMLProfile        string
	VPNCScript        string
	VPNServer         string
	User              string
	Password          string

	client *api.Client
	login  *ocrunner.LoginInfo
}

// Query retrieves the current status from OC-Daemon
func (c *Client) Query() (*vpnstatus.Status, error) {
	status, err := c.client.Query()
	if err != nil {
		return nil, err
	}
	return status, nil
}

// checkStatus checks if client is not connected to a trusted network and the
// VPN is not already running
func (c *Client) checkStatus() error {
	status, err := c.Query()
	if err != nil {
		return fmt.Errorf("could not query OC-Daemon: %w", err)
	}

	// check if we need to start the VPN connection
	if status.TrustedNetwork {
		return fmt.Errorf("trusted network detected, nothing to do")
	}
	if status.Connected {
		return fmt.Errorf("VPN already connected, nothing to do")
	}
	if status.Running {
		return fmt.Errorf("OpenConnect client already running, nothing to do")
	}
	return nil
}

// Authenticate authenticates the client on the VPN server
func (c *Client) Authenticate() error {
	// check status
	if err := c.checkStatus(); err != nil {
		return err
	}

	// authenticate
	auth := ocrunner.NewAuthenticate()
	auth.Certificate = c.ClientCertificate
	auth.Key = c.ClientKey
	auth.CA = c.CACertificate
	auth.XMLProfile = c.XMLProfile
	auth.Script = c.VPNCScript
	auth.Server = c.VPNServer
	auth.User = c.User
	auth.Password = c.Password

	if err := auth.Authenticate(); err != nil {
		return err
	}
	c.login = &auth.Login

	return nil
}

// Connect connects the client with the VPN server, requires successful
// authentication with Authenticate
func (c *Client) Connect() error {
	// check status
	if err := c.checkStatus(); err != nil {
		return err
	}

	// send login info to daemon
	return c.client.Connect(c.login)
}

// Disconnect disconnects the client from the VPN server
func (c *Client) Disconnect() error {
	// check status
	status, err := c.Query()
	if err != nil {
		return fmt.Errorf("could not query OC-Daemon: %w", err)
	}
	if !status.Running {
		return fmt.Errorf("OpenConnect client is not running, nothing to do")
	}

	// disconnect
	return c.client.Disconnect()
}

// NewClient returns a new client
func NewClient() *Client {
	return &Client{
		client: api.NewClient(sockFile),
	}
}
