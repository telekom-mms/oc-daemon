package client

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/pkg/client"
	"github.com/telekom-mms/oc-daemon/pkg/vpnstatus"
	"github.com/telekom-mms/oc-daemon/pkg/xmlprofile"
)

const (
	// maxReconnectTries is the maximum amount or reconnect retries
	maxReconnectTries = 5
)

// listServers gets the VPN status from the daemon and prints the VPN servers in it
func listServers() {
	// create client
	c, err := client.NewClient(config)
	if err != nil {
		log.WithError(err).Fatal("error creating client")
	}
	defer func() { _ = c.Close() }()

	// get status
	status, err := c.Query()
	if err != nil {
		log.Fatal(err)
	}

	// print servers in status
	fmt.Printf("Servers:\n")
	for _, server := range status.Servers {
		fmt.Printf("  - \"%s\"\n", server)
	}
}

// connectVPN connects to the VPN if necessary
func connectVPN() {
	// create client
	c, err := client.NewClient(config)
	if err != nil {
		log.WithError(err).Fatal("error creating client")
	}
	defer func() { _ = c.Close() }()

	// try to read current xml profile
	pre := xmlprofile.LoadSystemProfile()

	// authenticate
	if err := c.Authenticate(); err != nil {
		log.WithError(err).Fatal("error authenticating user for VPN")
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
		log.WithError(err).Fatal("error connecting to VPN")
	}
}

// disconnectVPN disconnects the VPN
func disconnectVPN() {
	// create client
	c, err := client.NewClient(config)
	if err != nil {
		log.WithError(err).Fatal("error creating client")
	}
	defer func() { _ = c.Close() }()

	// disconnect
	err = c.Disconnect()
	if err != nil {
		log.WithError(err).Fatal("error disconnecting from VPN")
	}
}

// reconnectVPN reconnects to the VPN
func reconnectVPN() {
	// create client
	client, err := client.NewClient(config)
	if err != nil {
		log.WithError(err).Fatal("error creating client")
	}
	defer func() { _ = client.Close() }()

	// check status
	status, err := client.Query()
	if err != nil {
		log.WithError(err).Fatal("error reconnecting to VPN")
	}

	// disconnect if needed
	if status.OCRunning.Running() {
		// send disconnect request
		disconnectVPN()
	}

	// wait for status to switch to untrusted network and not running
	try := 0
	for {
		status, err := client.Query()
		if err != nil {
			log.WithError(err).Fatal("error reconnecting to VPN")
		}

		if !status.TrustedNetwork.Trusted() &&
			!status.ConnectionState.Connected() &&
			!status.OCRunning.Running() {
			// authenticate and connect
			connectVPN()
			return
		}

		try++
		if try >= maxReconnectTries {
			// too many tries, abort
			log.Fatal("error reconnecting to VPN")
		}

		// sleep a second before retry
		time.Sleep(time.Second)
	}

}

// printStatus prints status on the command line
func printStatus(status *vpnstatus.Status) {
	if json {
		// print status as json
		j, err := status.JSON()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
		return
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
		return
	}
	if status.VPNConfig == nil {
		fmt.Printf("VPN Config:\n")
	} else {
		fmt.Printf("VPN Config:       %+v\n", *status.VPNConfig)
	}
}

// getStatus gets the VPN status from the daemon
func getStatus() {
	// create client
	c, err := client.NewClient(config)
	if err != nil {
		log.WithError(err).Fatal("error creating client")
	}
	defer func() { _ = c.Close() }()

	// get status
	status, err := c.Query()
	if err != nil {
		log.Fatal(err)
	}

	// print status
	printStatus(status)
}

// monitor subscribes to VPN status updates from the daemon and displays them
func monitor() {
	// create client
	c, err := client.NewClient(config)
	if err != nil {
		log.WithError(err).Fatal("error creating client")
	}
	defer func() { _ = c.Close() }()

	// get status updates
	updates, err := c.Subscribe()
	if err != nil {
		log.WithError(err).Fatal("error subscribing to status updates")
	}
	for u := range updates {
		log.Println("Got status update:")
		printStatus(u)
	}
}
