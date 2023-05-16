package api

import (
	"fmt"
	"net"
	"time"

	"github.com/T-Systems-MMS/oc-daemon/pkg/logininfo"
	"github.com/T-Systems-MMS/oc-daemon/pkg/vpnstatus"
)

const (
	// connectTimeout is the timeout for the client connection attempt
	connectTimeout = 30 * time.Second

	// clientTimeout is the timeout for the entire request/response
	// exchange initiated by the client after a successful connection
	clientTimeout = 30 * time.Second
)

// Client is a Daemon API client
type Client struct {
	sockFile string
}

// Request sends msg to the server and returns the server's response
func (c *Client) Request(msg *Message) (*Message, error) {
	// connect to daemon
	conn, err := net.DialTimeout("unix", c.sockFile, connectTimeout)
	if err != nil {
		return nil, fmt.Errorf("client dial error: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// set timeout for entire request/response message exchange
	deadline := time.Now().Add(clientTimeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("client set deadline error: %w", err)
	}

	// send message to daemon
	err = WriteMessage(conn, msg)
	if err != nil {
		return nil, fmt.Errorf("client send message error: %w", err)
	}

	// receive reply
	reply, err := ReadMessage(conn)
	if err != nil {
		return nil, fmt.Errorf("client receive message error: %w", err)
	}

	return reply, nil
}

// Query retrieves the VPN status from the daemon
func (c *Client) Query() (*vpnstatus.Status, error) {
	msg := NewMessage(TypeVPNQuery, nil)
	// send query to daemon

	// handle response
	reply, err := c.Request(msg)
	if err != nil {
		return nil, err
	}
	switch reply.Type {
	case TypeOK:
		// parse status in reply
		status, err := vpnstatus.NewFromJSON(reply.Value)
		if err != nil {
			return nil, fmt.Errorf("client received invalid status: %w", err)
		}
		return status, nil

	case TypeError:
		err := fmt.Errorf("%s", reply.Value)
		return nil, fmt.Errorf("client received error reply: %w", err)
	}
	return nil, fmt.Errorf("client received invalid reply")
}

// Connect sends a connect request with login info to the daemon
func (c *Client) Connect(login *logininfo.LoginInfo) error {
	// convert login to json
	b, err := login.JSON()
	if err != nil {
		return fmt.Errorf("client could not convert login info to JSON: %w", err)
	}

	// create connect request
	msg := NewMessage(TypeVPNConnect, b)

	// send request to server
	reply, err := c.Request(msg)
	if err != nil {
		return err
	}
	if reply.Type == TypeError {
		err := fmt.Errorf("%s", reply.Value)
		return fmt.Errorf("client received error reply: %w", err)
	}
	return nil
}

// Disconnect sends a disconnect request to the daemon
func (c *Client) Disconnect() error {
	// send disconnect request
	msg := NewMessage(TypeVPNDisconnect, nil)
	reply, err := c.Request(msg)
	if err != nil {
		return err
	}
	if reply.Type == TypeError {
		err := fmt.Errorf("%s", reply.Value)
		return fmt.Errorf("client received error reply: %w", err)
	}
	return nil
}

// NewClient returns a new Client
func NewClient(sockFile string) *Client {
	return &Client{
		sockFile: sockFile,
	}
}
