package client

import (
	"bytes"
	"os"
	"time"

	"github.com/T-Systems-MMS/oc-daemon/internal/api"
	"github.com/T-Systems-MMS/oc-daemon/internal/ocrunner"
	log "github.com/sirupsen/logrus"
)

const (
	// runDir is the daemons run dir
	runDir = "/run/oc-daemon"

	// daemon socket file
	sockFile = runDir + "/daemon.sock"

	// oc runner settings
	vpncScript = "/usr/bin/oc-daemon-vpncscript"

	// maxReconnectTries is the maximum amount or reconnect retries
	maxReconnectTries = 5
)

// readXMLProfile reads the contents of the XML profile
func readXMLProfile() []byte {
	b, err := os.ReadFile(xmlProfile)
	if err != nil {
		return nil
	}
	return b
}

// authenticateVPN authenticates user for vpn connection and returns login info
func authenticateVPN() *ocrunner.LoginInfo {
	// authenticate
	auth := ocrunner.NewAuthenticate()
	auth.Certificate = config.ClientCertificate
	auth.Key = config.ClientKey
	auth.CA = config.CACertificate
	auth.XMLProfile = xmlProfile
	auth.Script = vpncScript
	auth.Server = config.VPNServer
	auth.User = config.User
	auth.Authenticate()

	return &auth.Login
}

// authenticateConnectVPN authenticates the user and connects to the VPN
func authenticateConnectVPN(client *api.Client) {
	// try to read current xml profile
	pre := readXMLProfile()

	// autenticate user for vpn connection
	login := authenticateVPN()

	// warn user if profile changed
	post := readXMLProfile()
	if !bytes.Equal(pre, post) {
		time.Sleep(2 * time.Second)
		log.Warnln("XML Profile was updated. Connection attempt " +
			"might fail. Please, check status and reconnect " +
			"if necessary.")
		time.Sleep(2 * time.Second)
	}

	// send login info to daemon
	client.Connect(login)
}

// connectVPN connects to the VPN if necessary
func connectVPN() {
	// create client
	client := api.NewClient(sockFile)

	// get status
	status := client.Query()
	if status == nil {
		return
	}

	// check if we need to start the VPN connection
	if status.TrustedNetwork {
		log.Println("Trusted network detected, nothing to do")
		return
	}
	if status.Connected {
		log.Println("VPN already connected, nothing to do")
		return
	}
	if status.Running {
		log.Println("OpenConnect client already running, nothing to do")
		return
	}

	// authenticate and connect
	authenticateConnectVPN(client)
}

// disconnectVPN disconnects the VPN
func disconnectVPN() {
	// create client
	client := api.NewClient(sockFile)

	// check status
	status := client.Query()
	if status == nil {
		return
	}
	if !status.Running {
		log.Println("OpenConnect client is not running, nothing to do")
		return
	}

	// disconnect
	client.Disconnect()
}

// reconnectVPN reconnects to the VPN
func reconnectVPN() {
	// create client
	client := api.NewClient(sockFile)

	// check status
	status := client.Query()
	if status == nil {
		log.Fatal("error reconnecting to VPN")
	}

	// disconnect if needed
	if status.Running {
		// send disconnect request
		client.Disconnect()
	}

	// wait for status to switch to untrusted network and not running
	try := 0
	for {
		status := client.Query()
		if status == nil {
			log.Fatal("error reconnecting to VPN")
		}

		if !status.TrustedNetwork &&
			!status.Connected &&
			!status.Running {
			// authenticate and connect
			authenticateConnectVPN(client)
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

// getStatus gets the VPN status from the daemon
func getStatus() {
	client := api.NewClient(sockFile)
	status := client.Query()
	if status == nil {
		return
	}
	log.Printf("Trusted Network: %t", status.TrustedNetwork)
	log.Printf("Running: %t", status.Running)
	log.Printf("Connected: %t", status.Connected)
	log.Printf("Config: %+v", status.Config)
}
