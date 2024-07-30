/*
Dbusclient is an example of a D-Bus API client.
*/
package main

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/dbusapi"
)

func main() {
	// connect to system bus
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// subscribe to properties changed signals
	if err = conn.AddMatchSignal(
		dbus.WithMatchSender(dbusapi.Interface),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchPathNamespace(dbusapi.Path),
	); err != nil {
		log.Fatal(err)
	}

	// get initial values of properties
	trustedNetwork := dbusapi.TrustedNetworkUnknown
	connectionState := dbusapi.ConnectionStateUnknown
	ip := dbusapi.IPInvalid
	device := dbusapi.DeviceInvalid
	server := dbusapi.ServerInvalid
	serverIP := dbusapi.ServerInvalid
	connectedAt := dbusapi.ConnectedAtInvalid
	servers := dbusapi.ServersInvalid
	ocRunning := dbusapi.OCRunningUnknown
	trafPolState := dbusapi.TrafPolStateUnknown
	allowedHosts := dbusapi.AllowedHostsInvalid
	vpnConfig := dbusapi.VPNConfigInvalid

	getProperty := func(name string, val any) {
		err = conn.Object(dbusapi.Interface, dbusapi.Path).
			StoreProperty(dbusapi.Interface+"."+name, val)
		if err != nil {
			log.Fatal(err)
		}
	}
	getProperty(dbusapi.PropertyTrustedNetwork, &trustedNetwork)
	getProperty(dbusapi.PropertyConnectionState, &connectionState)
	getProperty(dbusapi.PropertyIP, &ip)
	getProperty(dbusapi.PropertyDevice, &device)
	getProperty(dbusapi.PropertyServer, &server)
	getProperty(dbusapi.PropertyServerIP, &serverIP)
	getProperty(dbusapi.PropertyConnectedAt, &connectedAt)
	getProperty(dbusapi.PropertyServers, &servers)
	getProperty(dbusapi.PropertyOCRunning, &ocRunning)
	getProperty(dbusapi.PropertyTrafPolState, &trafPolState)
	getProperty(dbusapi.PropertyAllowedHosts, &allowedHosts)
	getProperty(dbusapi.PropertyVPNConfig, &vpnConfig)

	log.Println("TrustedNetwork:", trustedNetwork)
	log.Println("ConnectionState:", connectionState)
	log.Println("IP:", ip)
	log.Println("Device:", device)
	log.Println("Server:", server)
	log.Println("ServerIP:", serverIP)
	log.Println("ConnectedAt:", connectedAt)
	log.Println("Servers:", servers)
	log.Println("OCRunning:", ocRunning)
	log.Println("TrafPolState:", trafPolState)
	log.Println("AllowedHosts:", allowedHosts)
	log.Println("VPNConfig:", vpnConfig)

	// handle signals
	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for s := range c {
		// make sure it's a properties changed signal
		if s.Path != dbusapi.Path ||
			s.Name != "org.freedesktop.DBus.Properties.PropertiesChanged" {
			log.Error("Not a properties changed signal")
			continue
		}

		// check properties changed signal
		if v, ok := s.Body[0].(string); !ok || v != dbusapi.Interface {
			log.Error("Not the right properties changed signal")
			continue
		}

		// get changed properties
		changed, ok := s.Body[1].(map[string]dbus.Variant)
		if !ok {
			log.Error("Invalid changed properties in properties changed signal")
			continue
		}
		for name, value := range changed {
			fmt.Printf("Changed property: %s ", name)
			switch name {
			case dbusapi.PropertyTrustedNetwork:
				if err := value.Store(&trustedNetwork); err != nil {
					log.Fatal(err)
				}
				fmt.Println(trustedNetwork)
			case dbusapi.PropertyConnectionState:
				if err := value.Store(&connectionState); err != nil {
					log.Fatal(err)
				}
				fmt.Println(connectionState)
			case dbusapi.PropertyIP:
				if err := value.Store(&ip); err != nil {
					log.Fatal(err)
				}
				fmt.Println(ip)
			case dbusapi.PropertyDevice:
				if err := value.Store(&device); err != nil {
					log.Fatal(err)
				}
				fmt.Println(device)
			case dbusapi.PropertyServer:
				if err := value.Store(&server); err != nil {
					log.Fatal(err)
				}
				fmt.Println(server)
			case dbusapi.PropertyServerIP:
				if err := value.Store(&serverIP); err != nil {
					log.Fatal(err)
				}
				fmt.Println(serverIP)
			case dbusapi.PropertyConnectedAt:
				if err := value.Store(&connectedAt); err != nil {
					log.Fatal(err)
				}
				fmt.Println(connectedAt)
			case dbusapi.PropertyServers:
				if err := value.Store(&servers); err != nil {
					log.Fatal(err)
				}
				fmt.Println(servers)
			case dbusapi.PropertyOCRunning:
				if err := value.Store(&ocRunning); err != nil {
					log.Fatal(err)
				}
				fmt.Println(ocRunning)
			case dbusapi.PropertyTrafPolState:
				if err := value.Store(&trafPolState); err != nil {
					log.Fatal(err)
				}
				fmt.Println(trafPolState)
			case dbusapi.PropertyAllowedHosts:
				if err := value.Store(&allowedHosts); err != nil {
					log.Fatal(err)
				}
				fmt.Println(allowedHosts)
			case dbusapi.PropertyVPNConfig:
				if err := value.Store(&vpnConfig); err != nil {
					log.Fatal(err)
				}
				fmt.Println(vpnConfig)
			}
		}

		// get invalidated properties
		invalid, ok := s.Body[2].([]string)
		if !ok {
			log.Error("Invalid invalidated properties in properties changed signal")
			continue
		}
		for _, name := range invalid {
			// not expected to happen currently, but handle it anyway
			switch name {
			case dbusapi.PropertyTrustedNetwork:
				trustedNetwork = dbusapi.TrustedNetworkUnknown
			case dbusapi.PropertyConnectionState:
				connectionState = dbusapi.ConnectionStateUnknown
			case dbusapi.PropertyIP:
				ip = dbusapi.IPInvalid
			case dbusapi.PropertyDevice:
				device = dbusapi.DeviceInvalid
			case dbusapi.PropertyServer:
				device = dbusapi.ServerInvalid
			case dbusapi.PropertyServerIP:
				device = dbusapi.ServerIPInvalid
			case dbusapi.PropertyConnectedAt:
				connectedAt = dbusapi.ConnectedAtInvalid
			case dbusapi.PropertyServers:
				servers = dbusapi.ServersInvalid
			case dbusapi.PropertyOCRunning:
				ocRunning = dbusapi.OCRunningUnknown
			case dbusapi.PropertyTrafPolState:
				trafPolState = dbusapi.TrafPolStateUnknown
			case dbusapi.PropertyAllowedHosts:
				allowedHosts = dbusapi.AllowedHostsInvalid
			case dbusapi.PropertyVPNConfig:
				vpnConfig = dbusapi.VPNConfigInvalid
			}
			fmt.Printf("Invalidated property: %s\n", name)
		}
	}
}
