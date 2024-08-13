// Package dbusapi contains the D-Bus API.
package dbusapi

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	log "github.com/sirupsen/logrus"
)

// D-Bus object path and interface.
const (
	Path      = "/com/telekom_mms/oc_daemon/Daemon"
	Interface = "com.telekom_mms.oc_daemon.Daemon"
)

// PropertiesChanged is the DBus properties changed signal.
const PropertiesChanged = "org.freedesktop.DBus.Properties.PropertiesChanged"

// Properties.
const (
	PropertyTrustedNetwork  = "TrustedNetwork"
	PropertyConnectionState = "ConnectionState"
	PropertyIP              = "IP"
	PropertyDevice          = "Device"
	PropertyServer          = "Server"
	PropertyServerIP        = "ServerIP"
	PropertyConnectedAt     = "ConnectedAt"
	PropertyServers         = "Servers"
	PropertyOCRunning       = "OCRunning"
	PropertyTrafPolState    = "TrafPolState"
	PropertyAllowedHosts    = "AllowedHosts"
	PropertyTNDState        = "TNDState"
	PropertyTNDServers      = "TNDServers"
	PropertyVPNConfig       = "VPNConfig"
)

// Property "Trusted Network" states.
const (
	TrustedNetworkUnknown uint32 = iota
	TrustedNetworkNotTrusted
	TrustedNetworkTrusted
)

// Property "Connection State" states.
const (
	ConnectionStateUnknown uint32 = iota
	ConnectionStateDisconnected
	ConnectionStateConnecting
	ConnectionStateConnected
	ConnectionStateDisconnecting
)

// Property "IP" values.
const (
	IPInvalid = ""
)

// Property "Device" values.
const (
	DeviceInvalid = ""
)

// Property "Server" values.
const (
	ServerInvalid = ""
)

// Property "ServerIP" values.
const (
	ServerIPInvalid = ""
)

// Property "Connected At" values.
const (
	ConnectedAtInvalid int64 = -1
)

// Property "Servers" values.
var (
	ServersInvalid []string
)

// Property "OCRunning" values.
const (
	OCRunningUnknown uint32 = iota
	OCRunningNotRunning
	OCRunningRunning
)

// Property "TrafPol State" states.
const (
	TrafPolStateUnknown uint32 = iota
	TrafPolStateInactive
	TrafPolStateActive
)

// Property "Allowed Hosts" values.
var (
	AllowedHostsInvalid []string
)

// Property "TND State" states.
const (
	TNDStateUnknown uint32 = iota
	TNDStateInactive
	TNDStateActive
)

// Property "TND Servers" values.
var (
	TNDServersInvalid []string
)

// Property "VPNConfig" values.
const (
	VPNConfigInvalid = ""
)

// Methods.
const (
	MethodConnect    = Interface + ".Connect"
	MethodDisconnect = Interface + ".Disconnect"
)

// Request Names.
const (
	RequestConnect    = "Connect"
	RequestDisconnect = "Disconnect"
)

// Request is a D-Bus client request.
type Request struct {
	Name       string
	Parameters []any
	Results    []any
	Error      error

	wait chan struct{}
	done chan struct{}
}

// Close completes the request handling.
func (r *Request) Close() {
	close(r.wait)
}

// Wait waits for the completion of request handling.
func (r *Request) Wait() {
	select {
	case <-r.wait:
	case <-r.done:
		r.Error = errors.New("Request aborted")
	}
}

// daemon defines daemon interface methods.
type daemon struct {
	requests chan *Request
	done     chan struct{}
}

// Connect is the "Connect" method of the D-Bus interface.
func (d daemon) Connect(sender dbus.Sender, server, cookie, host, connectURL, fingerprint, resolve string) *dbus.Error {
	log.WithField("sender", sender).Debug("Received D-Bus Connect() call")
	request := &Request{
		Name:       RequestConnect,
		Parameters: []any{server, cookie, host, connectURL, fingerprint, resolve},
		wait:       make(chan struct{}),
		done:       d.done,
	}
	select {
	case d.requests <- request:
	case <-d.done:
		return dbus.NewError(Interface+".ConnectAborted", []any{"Connect aborted"})
	}

	request.Wait()
	if request.Error != nil {
		return dbus.NewError(Interface+".ConnectAborted", []any{request.Error.Error()})
	}
	return nil
}

// Disconnect is the "Disconnect" method of the D-Bus interface.
func (d daemon) Disconnect(sender dbus.Sender) *dbus.Error {
	log.WithField("sender", sender).Debug("Received D-Bus Connect() call")
	request := &Request{
		Name: RequestDisconnect,
		wait: make(chan struct{}),
		done: d.done,
	}
	select {
	case d.requests <- request:
	case <-d.done:
		return dbus.NewError(Interface+".DisconnectAborted", []any{"Disconnect aborted"})
	}

	request.Wait()
	if request.Error != nil {
		return dbus.NewError(Interface+".DisconnectAborted", []any{request.Error.Error()})
	}
	return nil
}

// propertyUpdate is an update of a property.
type propertyUpdate struct {
	name  string
	value any
}

// Service is a D-Bus Service.
type Service struct {
	conn  dbusConn
	props propProperties

	requests chan *Request
	propUps  chan *propertyUpdate
	done     chan struct{}
	closed   chan struct{}
}

// dbusConn is an interface for dbus.Conn to allow for testing.
type dbusConn interface {
	Close() error
	Export(v any, path dbus.ObjectPath, iface string) error
	RequestName(name string, flags dbus.RequestNameFlags) (dbus.RequestNameReply, error)
}

// dbusConnectSystemBus encapsulates dbus.ConnectSystemBus to allow for testing.
var dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
	return dbus.ConnectSystemBus(opts...)
}

// propProperties is an interface for prop.Properties to allow for testing.
type propProperties interface {
	Introspection(iface string) []introspect.Property
	SetMust(iface, property string, v any)
}

// propExport encapsulates prop.Export to allow for testing.
var propExport = func(conn dbusConn, path dbus.ObjectPath, props prop.Map) (propProperties, error) {
	return prop.Export(conn.(*dbus.Conn), path, props)
}

// start starts the service.
func (s *Service) start() {
	defer close(s.closed)
	defer func() { _ = s.conn.Close() }()

	// helper for setting initial property values
	setInitialProps := func() {
		s.props.SetMust(Interface, PropertyTrustedNetwork, TrustedNetworkUnknown)
		s.props.SetMust(Interface, PropertyConnectionState, ConnectionStateUnknown)
		s.props.SetMust(Interface, PropertyIP, IPInvalid)
		s.props.SetMust(Interface, PropertyDevice, DeviceInvalid)
		s.props.SetMust(Interface, PropertyServer, ServerInvalid)
		s.props.SetMust(Interface, PropertyServerIP, ServerIPInvalid)
		s.props.SetMust(Interface, PropertyConnectedAt, ConnectedAtInvalid)
		s.props.SetMust(Interface, PropertyServers, ServersInvalid)
		s.props.SetMust(Interface, PropertyOCRunning, OCRunningUnknown)
		s.props.SetMust(Interface, PropertyTrafPolState, TrafPolStateUnknown)
		s.props.SetMust(Interface, PropertyAllowedHosts, AllowedHostsInvalid)
		s.props.SetMust(Interface, PropertyTNDState, TNDStateUnknown)
		s.props.SetMust(Interface, PropertyTNDServers, TNDServersInvalid)
		s.props.SetMust(Interface, PropertyVPNConfig, VPNConfigInvalid)
	}

	// set properties values to emit properties changed signal and make
	// sure existing clients get updated values after restart
	setInitialProps()

	// main loop
	for {
		select {
		case u := <-s.propUps:
			// update property
			log.WithFields(log.Fields{
				"name":  u.name,
				"value": u.value,
			}).Debug("D-Bus updating property")
			s.props.SetMust(Interface, u.name, u.value)

		case <-s.done:
			log.Debug("D-Bus service stopping")
			// set properties values to unknown/invalid to emit
			// properties changed signal and inform clients
			setInitialProps()
			return
		}
	}
}

// Start starts the service.
func (s *Service) Start() error {
	// connect to session bus
	conn, err := dbusConnectSystemBus()
	if err != nil {
		return fmt.Errorf("could not connect to D-Bus session bus: %w", err)
	}
	s.conn = conn

	// request name
	reply, err := conn.RequestName(Interface, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("could not request D-Bus name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("requested D-Bus name is already taken")
	}

	// methods
	meths := daemon{s.requests, s.done}
	err = conn.Export(meths, Path, Interface)
	if err != nil {
		return fmt.Errorf("could not export D-Bus methods: %w", err)
	}

	// properties
	propsSpec := prop.Map{
		Interface: {
			PropertyTrustedNetwork: {
				Value:    TrustedNetworkUnknown,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyConnectionState: {
				Value:    ConnectionStateUnknown,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyIP: {
				Value:    IPInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyDevice: {
				Value:    DeviceInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyServer: {
				Value:    ServerInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyServerIP: {
				Value:    ServerIPInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyConnectedAt: {
				Value:    ConnectedAtInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyServers: {
				Value:    ServersInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyOCRunning: {
				Value:    OCRunningUnknown,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyTrafPolState: {
				Value:    TrafPolStateUnknown,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyAllowedHosts: {
				Value:    AllowedHostsInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyTNDState: {
				Value:    TNDStateUnknown,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyTNDServers: {
				Value:    TNDServersInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
			PropertyVPNConfig: {
				Value:    VPNConfigInvalid,
				Writable: false,
				Emit:     prop.EmitTrue,
				Callback: nil,
			},
		},
	}
	props, err := propExport(conn, Path, propsSpec)
	if err != nil {
		return fmt.Errorf("could not export D-Bus properties spec: %w", err)
	}
	s.props = props

	// introspection
	// set names of method arguments
	introMeths := introspect.Methods(meths)
	for _, m := range introMeths {
		if m.Name != "Connect" {
			continue
		}
		m.Args[0].Name = "server"
		m.Args[1].Name = "cookie"
		m.Args[2].Name = "host"
		m.Args[3].Name = "connect_url"
		m.Args[4].Name = "fingerprint"
		m.Args[5].Name = "resolve"

	}
	// set peer interface
	peerData := introspect.Interface{
		Name: "org.freedesktop.DBus.Peer",
		Methods: []introspect.Method{
			{
				Name: "Ping",
			},
			{
				Name: "GetMachineId",
				Args: []introspect.Arg{
					{Name: "machine_uuid", Type: "s", Direction: "out"},
				},
			},
		},
	}
	n := &introspect.Node{
		Name: Path,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			peerData,
			prop.IntrospectData,
			{
				Name:       Interface,
				Methods:    introMeths,
				Properties: props.Introspection(Interface),
			},
		},
	}
	err = conn.Export(introspect.NewIntrospectable(n), Path,
		"org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("could not export D-Bus introspection: %w", err)
	}

	go s.start()
	return nil
}

// Stop stops the service.
func (s *Service) Stop() {
	close(s.done)
	<-s.closed
}

// Requests returns the requests channel of service.
func (s *Service) Requests() chan *Request {
	return s.requests
}

// SetProperty sets property with name to value.
func (s *Service) SetProperty(name string, value any) {
	select {
	case s.propUps <- &propertyUpdate{name, value}:
	case <-s.done:
	}
}

// NewService returns a new service.
func NewService() *Service {
	return &Service{
		requests: make(chan *Request),
		propUps:  make(chan *propertyUpdate),
		done:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
}
