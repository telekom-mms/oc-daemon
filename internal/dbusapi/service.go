package dbusapi

import (
	"errors"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	log "github.com/sirupsen/logrus"
)

// D-Bus object path and interface
const (
	Path      = "/com/telekom_mms/oc_daemon/Daemon"
	Interface = "com.telekom_mms.oc_daemon.Daemon"
)

// Properties
const (
	PropertyTrustedNetwork  = "TrustedNetwork"
	PropertyConnectionState = "ConnectionState"
	PropertyIP              = "IP"
	PropertyDevice          = "Device"
	PropertyServer          = "Server"
	PropertyConnectedAt     = "ConnectedAt"
	PropertyServers         = "Servers"
	PropertyOCRunning       = "OCRunning"
	PropertyVPNConfig       = "VPNConfig"
)

// Property "Trusted Network" states
const (
	TrustedNetworkUnknown uint32 = iota
	TrustedNetworkNotTrusted
	TrustedNetworkTrusted
)

// Property "Connection State" states
const (
	ConnectionStateUnknown uint32 = iota
	ConnectionStateDisconnected
	ConnectionStateConnecting
	ConnectionStateConnected
	ConnectionStateDisconnecting
)

// Property "IP" values
const (
	IPInvalid = ""
)

// Property "Device" values
const (
	DeviceInvalid = ""
)

// Property "Server" values
const (
	ServerInvalid = ""
)

// Property "Connected At" values
const (
	ConnectedAtInvalid int64 = -1
)

// Property "Servers" values
var (
	ServersInvalid []string
)

// Property "OCRunning" values
const (
	OCRunningUnknown uint32 = iota
	OCRunningNotRunning
	OCRunningRunning
)

// Property "VPNConfig" values
const (
	VPNConfigInvalid = ""
)

// Methods
const (
	MethodConnect    = Interface + ".Connect"
	MethodDisconnect = Interface + ".Disconnect"
)

// Request Names
const (
	RequestConnect    = "Connect"
	RequestDisconnect = "Disconnect"
)

// Request is a D-Bus client request
type Request struct {
	Name       string
	Parameters []any
	Results    []any
	Error      error

	wait chan struct{}
	done chan struct{}
}

// Close completes the request handling
func (r *Request) Close() {
	close(r.wait)
}

// Wait waits for the completion of request handling
func (r *Request) Wait() {
	select {
	case <-r.wait:
	case <-r.done:
		r.Error = errors.New("Request aborted")
	}
}

// daemon defines daemon interface methods
type daemon struct {
	requests chan *Request
	done     chan struct{}
}

// Connect is the "Connect" method of the D-Bus interface
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

// Disconnect is the "Disconnect" method of the D-Bus interface
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

// propertyUpdate is an update of a property
type propertyUpdate struct {
	name  string
	value any
}

// Service is a D-Bus Service
type Service struct {
	requests chan *Request
	propUps  chan *propertyUpdate
	done     chan struct{}
	closed   chan struct{}
}

// dbusConn is an interface for dbus.Conn to allow for testing
type dbusConn interface {
	Close() error
	Export(v any, path dbus.ObjectPath, iface string) error
	RequestName(name string, flags dbus.RequestNameFlags) (dbus.RequestNameReply, error)
}

// dbusConnectSystemBus encapsulates dbus.ConnectSystemBus to allow for testing
var dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
	return dbus.ConnectSystemBus(opts...)
}

// propProperties is an interface for prop.Properties to allow for testing
type propProperties interface {
	Introspection(iface string) []introspect.Property
	SetMust(iface, property string, v any)
}

// propExport encapsulates prop.Export to allow for testing
var propExport = func(conn dbusConn, path dbus.ObjectPath, props prop.Map) (propProperties, error) {
	return prop.Export(conn.(*dbus.Conn), path, props)
}

// start starts the service
func (s *Service) start() {
	defer close(s.closed)

	// connect to session bus
	conn, err := dbusConnectSystemBus()
	if err != nil {
		log.WithError(err).Fatal("Could not connect to D-Bus session bus")
	}
	defer func() { _ = conn.Close() }()

	// request name
	reply, err := conn.RequestName(Interface, dbus.NameFlagDoNotQueue)
	if err != nil {
		log.WithError(err).Fatal("Could not request D-Bus name")
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Requested D-Bus name is already taken")
	}

	// methods
	meths := daemon{s.requests, s.done}
	err = conn.Export(meths, Path, Interface)
	if err != nil {
		log.WithError(err).Fatal("Could not export D-Bus methods")
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
		log.WithError(err).Fatal("Could not export D-Bus properties spec")
	}

	// introspection
	n := &introspect.Node{
		Name: Path,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       Interface,
				Methods:    introspect.Methods(meths),
				Properties: props.Introspection(Interface),
			},
		},
	}
	err = conn.Export(introspect.NewIntrospectable(n), Path,
		"org.freedesktop.DBus.Introspectable")
	if err != nil {
		log.WithError(err).Fatal("Could not export D-Bus introspection")
	}

	// set properties values to emit properties changed signal and make
	// sure existing clients get updated values after restart
	props.SetMust(Interface, PropertyTrustedNetwork, TrustedNetworkNotTrusted)
	props.SetMust(Interface, PropertyConnectionState, ConnectionStateDisconnected)
	props.SetMust(Interface, PropertyIP, IPInvalid)
	props.SetMust(Interface, PropertyDevice, DeviceInvalid)
	props.SetMust(Interface, PropertyServer, ServerInvalid)
	props.SetMust(Interface, PropertyConnectedAt, ConnectedAtInvalid)
	props.SetMust(Interface, PropertyServers, ServersInvalid)
	props.SetMust(Interface, PropertyOCRunning, OCRunningNotRunning)
	props.SetMust(Interface, PropertyVPNConfig, VPNConfigInvalid)

	// main loop
	for {
		select {
		case u := <-s.propUps:
			// update property
			log.WithFields(log.Fields{
				"name":  u.name,
				"value": u.value,
			}).Debug("D-Bus updating property")
			props.SetMust(Interface, u.name, u.value)

		case <-s.done:
			log.Debug("D-Bus service stopping")
			// set properties values to unknown/invalid to emit
			// properties changed signal and inform clients
			props.SetMust(Interface, PropertyTrustedNetwork, TrustedNetworkUnknown)
			props.SetMust(Interface, PropertyConnectionState, ConnectionStateUnknown)
			props.SetMust(Interface, PropertyIP, IPInvalid)
			props.SetMust(Interface, PropertyDevice, DeviceInvalid)
			props.SetMust(Interface, PropertyServer, ServerInvalid)
			props.SetMust(Interface, PropertyConnectedAt, ConnectedAtInvalid)
			props.SetMust(Interface, PropertyServers, ServersInvalid)
			props.SetMust(Interface, PropertyOCRunning, OCRunningUnknown)
			props.SetMust(Interface, PropertyVPNConfig, VPNConfigInvalid)
			return
		}
	}
}

// Start starts the service
func (s *Service) Start() {
	go s.start()
}

// Stop stops the service
func (s *Service) Stop() {
	close(s.done)
	<-s.closed
}

// Requests returns the requests channel of service
func (s *Service) Requests() chan *Request {
	return s.requests
}

// SetProperty sets property with name to value
func (s *Service) SetProperty(name string, value any) {
	select {
	case s.propUps <- &propertyUpdate{name, value}:
	case <-s.done:
	}
}

// NewService returns a new service
func NewService() *Service {
	return &Service{
		requests: make(chan *Request),
		propUps:  make(chan *propertyUpdate),
		done:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
}
