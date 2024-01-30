package client

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/pkg/client"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
	"github.com/telekom-mms/oc-daemon/pkg/xmlprofile"
)

var (
	// maxReconnectTries is the maximum amount or reconnect retries
	maxReconnectTries = 5

	// reconnectSleep is the sleep time between reconnect retries.
	reconnectSleep = time.Second
)

// clientNewClient is client.NewClient for testing.
var clientNewClient = client.NewClient

// listServers gets the VPN status from the daemon and prints the VPN servers in it
func listServers() error {
	// create client
	c, err := clientNewClient(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer func() { _ = c.Close() }()

	// get status
	status, err := c.Query()
	if err != nil {
		return fmt.Errorf("error getting server list: %w", err)
	}

	// print servers in status
	fmt.Printf("Servers:\n")
	for _, server := range status.Servers {
		fmt.Printf("  - \"%s\"\n", server)
	}

	return nil
}

// connectVPN connects to the VPN if necessary
func connectVPN() error {
	// create client
	c, err := clientNewClient(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer func() { _ = c.Close() }()

	// try to read current xml profile
	pre := xmlprofile.LoadSystemProfile()

	// authenticate
	if err := c.Authenticate(); err != nil {
		return fmt.Errorf("error authenticating user for VPN: %w", err)
	}

	// warn user if profile changed
	post := xmlprofile.LoadSystemProfile()
	if !pre.Equal(post) {
		time.Sleep(2 * time.Second)
		log.Warnln("XML Profile was updated. Connection attempt " +
			"might fail. Please, check status and reconnect " +
			"if necessary.")
		time.Sleep(2 * time.Second)
	}

	// connect
	if err := c.Connect(); err != nil {
		return fmt.Errorf("error connecting to VPN: %w", err)
	}

	return nil
}

// disconnectVPN disconnects the VPN
func disconnectVPN() error {
	// create client
	c, err := clientNewClient(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer func() { _ = c.Close() }()

	// disconnect
	err = c.Disconnect()
	if err != nil {
		return fmt.Errorf("error disconnecting from VPN: %w", err)
	}

	return nil
}

// reconnectVPN reconnects to the VPN
func reconnectVPN() error {
	// create client
	client, err := clientNewClient(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// check status
	status, err := client.Query()
	if err != nil {
		return fmt.Errorf("error reconnecting to VPN: %w", err)
	}

	// disconnect if needed
	if status.OCRunning.Running() {
		// send disconnect request
		if err := disconnectVPN(); err != nil {
			return err
		}
	}

	// wait for status to switch to untrusted network and not running
	try := 0
	for {
		status, err := client.Query()
		if err != nil {
			return fmt.Errorf("error reconnecting to VPN: %w", err)
		}

		if !status.TrustedNetwork.Trusted() &&
			!status.ConnectionState.Connected() &&
			!status.OCRunning.Running() {
			// authenticate and connect
			return connectVPN()
		}

		try++
		if try >= maxReconnectTries {
			// too many tries, abort
			return fmt.Errorf("error reconnecting to VPN: too many tries")
		}

		// sleep a second before retry
		time.Sleep(reconnectSleep)
	}
}

// printStatus prints status on the command line
func printStatus(status *vpnstatus.Status) error {
	if json {
		// print status as json
		j, err := status.JSON()
		if err != nil {
			return err
		}
		fmt.Println(string(j))
		return nil
	}

	// print status
	fmt.Printf("Trusted Network:  %s\n", status.TrustedNetwork)
	fmt.Printf("Connection State: %s\n", status.ConnectionState)
	fmt.Printf("IP:               %s\n", status.IP)
	fmt.Printf("Device:           %s\n", status.Device)
	fmt.Printf("Current Server:   %s\n", status.Server)

	if status.ConnectedAt <= 0 {
		fmt.Printf("Connected At:\n")
	} else {
		connectedAt := time.Unix(status.ConnectedAt, 0)
		fmt.Printf("Connected At:     %s\n", connectedAt)
	}

	fmt.Printf("Servers:\n")
	for _, server := range status.Servers {
		fmt.Printf("  - \"%s\"\n", server)
	}

	fmt.Printf("OC Running:       %s\n", status.OCRunning)

	// verbose output
	if !verbose {
		return nil
	}
	if status.VPNConfig == nil {
		fmt.Printf("VPN Config:\n")
	} else {
		fmt.Printf("VPN Config:       %+v\n", *status.VPNConfig)
	}

	return nil
}

// getStatus gets the VPN status from the daemon
func getStatus() error {
	// create client
	c, err := clientNewClient(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer func() { _ = c.Close() }()

	// get status
	status, err := c.Query()
	if err != nil {
		return fmt.Errorf("error getting status: %w", err)
	}

	// print status
	return printStatus(status)
}

// monitor subscribes to VPN status updates from the daemon and displays them
func monitor() error {
	// create client
	c, err := clientNewClient(config)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	defer func() { _ = c.Close() }()

	// get status updates
	updates, err := c.Subscribe()
	if err != nil {
		return fmt.Errorf("error subscribing to status updates: %w", err)
	}
	for u := range updates {
		log.Println("Got status update:")
		if err := printStatus(u); err != nil {
			return err
		}
	}

	return nil
}
