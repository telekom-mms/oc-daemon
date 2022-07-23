package api

import (
	"net"
	"time"

	"github.com/T-Systems-MMS/oc-daemon/internal/ocrunner"
	"github.com/T-Systems-MMS/oc-daemon/internal/vpnstatus"
	log "github.com/sirupsen/logrus"
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
func (c *Client) Request(msg *Message) *Message {
	// connect to daemon
	conn, err := net.DialTimeout("unix", c.sockFile, connectTimeout)
	if err != nil {
		log.WithError(err).Fatal("Client dial error")
	}
	defer func() {
		_ = conn.Close()
	}()

	// set timeout for entire request/response message exchange
	deadline := time.Now().Add(clientTimeout)
	if err := conn.SetDeadline(deadline); err != nil {
		log.WithError(err).Fatal("Client set deadline error")
	}

	// send message to daemon
	err = WriteMessage(conn, msg)
	if err != nil {
		log.WithError(err).Fatal("Client send message error")
	}

	// receive reply
	reply, err := ReadMessage(conn)
	if err != nil {
		log.WithError(err).Fatal("Client receive message error")
	}

	return reply
}

// Query retrieves the VPN status from the daemon
func (c *Client) Query() *vpnstatus.Status {
	// send query to daemon
	msg := NewMessage(TypeVPNQuery, nil)
	reply := c.Request(msg)

	// handle response
	switch reply.Type {
	case TypeOK:
		// parse status in reply
		status, err := vpnstatus.NewFromJSON(reply.Value)
		if err != nil {
			log.WithError(err).Fatal("Client received invalid status")
		}
		return status

	case TypeError:
		log.WithField("error", string(reply.Value)).Error("Client received error reply")
	}
	return nil
}

// Connect sends a connect request with login info to the daemon
func (c *Client) Connect(login *ocrunner.LoginInfo) {
	// convert login to json
	b, err := login.JSON()
	if err != nil {
		log.WithError(err).Fatal("Client could not convert login info to JSON")
	}

	// create connect request
	msg := NewMessage(TypeVPNConnect, b)

	// send request to server
	reply := c.Request(msg)
	if reply.Type == TypeError {
		log.WithField("error", string(reply.Value)).Error("Client received error reply")
	}
}

// Disconnect sends a disconnect request to the daemon
func (c *Client) Disconnect() {
	// send disconnect request
	msg := NewMessage(TypeVPNDisconnect, nil)
	reply := c.Request(msg)
	if reply.Type == TypeError {
		log.WithField("error", string(reply.Value)).Error("Client received error reply")
	}
}

// NewClient returns a new Client
func NewClient(sockFile string) *Client {
	return &Client{
		sockFile: sockFile,
	}
}
