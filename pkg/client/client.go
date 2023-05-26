package client

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/T-Systems-MMS/oc-daemon/internal/api"
	"github.com/T-Systems-MMS/oc-daemon/pkg/logininfo"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnstatus"
)

// Client is an OC-Daemon client
type Client struct {
	// Config is the client configuration
	Config *Config

	// Env are extra environment variables set during execution of
	// `openconnect --authenticate`
	Env []string

	// Login contains information required to connect to the VPN, produced
	// by successful authentication
	Login *logininfo.LoginInfo
}

// Request sends msg to the server and returns the server's response
func (c *Client) Request(msg *api.Message) (*api.Message, error) {
	// connect to daemon
	conn, err := net.DialTimeout("unix", c.Config.SocketFile, c.Config.ConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("client dial error: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// set timeout for entire request/response message exchange
	deadline := time.Now().Add(c.Config.RequestTimeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("client set deadline error: %w", err)
	}

	// send message to daemon
	err = api.WriteMessage(conn, msg)
	if err != nil {
		return nil, fmt.Errorf("client send message error: %w", err)
	}

	// receive reply
	reply, err := api.ReadMessage(conn)
	if err != nil {
		return nil, fmt.Errorf("client receive message error: %w", err)
	}

	return reply, nil
}

// query retrieves the VPN status from the daemon
func (c *Client) query() (*vpnstatus.Status, error) {
	msg := api.NewMessage(api.TypeVPNQuery, nil)
	// send query to daemon

	// handle response
	reply, err := c.Request(msg)
	if err != nil {
		return nil, err
	}
	switch reply.Type {
	case api.TypeOK:
		// parse status in reply
		status, err := vpnstatus.NewFromJSON(reply.Value)
		if err != nil {
			return nil, fmt.Errorf("client received invalid status: %w", err)
		}
		return status, nil

	case api.TypeError:
		err := fmt.Errorf("%s", reply.Value)
		return nil, fmt.Errorf("client received error reply: %w", err)
	}
	return nil, fmt.Errorf("client received invalid reply")
}

// Query retrieves the current status from OC-Daemon
func (c *Client) Query() (*vpnstatus.Status, error) {
	status, err := c.query()
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
	if status.TrustedNetwork.Trusted() {
		return fmt.Errorf("trusted network detected, nothing to do")
	}
	if status.ConnectionState.Connected() {
		return fmt.Errorf("VPN already connected, nothing to do")
	}
	if status.OCRunning.Running() {
		return fmt.Errorf("OpenConnect client already running, nothing to do")
	}
	return nil
}

// authenticate runs OpenConnect in authentication mode
func (c *Client) authenticate() error {
	// create openconnect command:
	//
	// openconnect \
	//   --protocol=anyconnect \
	//   --certificate="$CLIENT_CERT" \
	//   --sslkey="$PRIVATE_KEY" \
	//   --cafile="$CA_CERT" \
	//   --xmlconfig="$XML_CONFIG" \
	//   --authenticate \
	//   --quiet \
	//   "$SERVER"
	//
	certificate := fmt.Sprintf("--certificate=%s", c.Config.ClientCertificate)
	sslKey := fmt.Sprintf("--sslkey=%s", c.Config.ClientKey)
	caFile := fmt.Sprintf("--cafile=%s", c.Config.CACertificate)
	xmlConfig := fmt.Sprintf("--xmlconfig=%s", c.Config.XMLProfile)
	user := fmt.Sprintf("--user=%s", c.Config.User)

	parameters := []string{
		"--protocol=anyconnect",
		certificate,
		sslKey,
		xmlConfig,
		"--authenticate",
		"--quiet",
		"--no-proxy",
	}
	if c.Config.CACertificate != "" {
		parameters = append(parameters, caFile)
	}
	if c.Config.User != "" {
		parameters = append(parameters, user)
	}
	if c.Config.Password != "" {
		// read password from stdin and switch to non-interactive mode
		parameters = append(parameters, "--passwd-on-stdin")
		parameters = append(parameters, "--non-inter")
	}
	parameters = append(parameters, c.Config.VPNServer)

	command := exec.Command("openconnect", parameters...)

	// run command: allow user input, show stderr, buffer stdout
	var b bytes.Buffer
	command.Stdin = os.Stdin
	if c.Config.Password != "" {
		// disable user input, pass password via stdin
		command.Stdin = bytes.NewBufferString(c.Config.Password)
	}
	command.Stdout = &b
	command.Stderr = os.Stderr
	command.Env = append(os.Environ(), c.Env...)
	if err := command.Run(); err != nil {
		// TODO: handle failed program start?
		return err
	}

	// parse login info, cookie from command line in buffer:
	//
	// COOKIE=3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...
	// HOST=10.0.0.1
	// CONNECT_URL='https://vpnserver.example.com'
	// FINGERPRINT=469bb424ec8835944d30bc77c77e8fc1d8e23a42
	// RESOLVE='vpnserver.example.com:10.0.0.1'
	//
	s := b.String()
	c.Login = &logininfo.LoginInfo{}
	for _, line := range strings.Fields(s) {
		c.Login.ParseLine(line)
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
	if err := c.authenticate(); err != nil {
		return err
	}

	return nil
}

// connect sends a connect request with login info to the daemon
func (c *Client) connect() error {
	// convert login to json
	b, err := c.Login.JSON()
	if err != nil {
		return fmt.Errorf("client could not convert login info to JSON: %w", err)
	}

	// create connect request
	msg := api.NewMessage(api.TypeVPNConnect, b)

	// send request to server
	reply, err := c.Request(msg)
	if err != nil {
		return err
	}
	if reply.Type == api.TypeError {
		err := fmt.Errorf("%s", reply.Value)
		return fmt.Errorf("client received error reply: %w", err)
	}
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
	return c.connect()
}

// disconnect sends a disconnect request to the daemon
func (c *Client) disconnect() error {
	// send disconnect request
	msg := api.NewMessage(api.TypeVPNDisconnect, nil)
	reply, err := c.Request(msg)
	if err != nil {
		return err
	}
	if reply.Type == api.TypeError {
		err := fmt.Errorf("%s", reply.Value)
		return fmt.Errorf("client received error reply: %w", err)
	}
	return nil
}

// Disconnect disconnects the client from the VPN server
func (c *Client) Disconnect() error {
	// check status
	status, err := c.Query()
	if err != nil {
		return fmt.Errorf("could not query OC-Daemon: %w", err)
	}
	if !status.OCRunning.Running() {
		return fmt.Errorf("OpenConnect client is not running, nothing to do")
	}

	// disconnect
	return c.disconnect()
}

// NewClient returns a new client
func NewClient(config *Config) *Client {
	return &Client{
		Config: config,
	}
}
