// Package vpnstatus contains the VPN status.
package vpnstatus

import (
	"encoding/json"

	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// TrustedNetwork is the current trusted network state.
type TrustedNetwork uint32

// TrustedNetwork states
const (
	TrustedNetworkUnknown TrustedNetwork = iota
	TrustedNetworkNotTrusted
	TrustedNetworkTrusted
)

// Trusted returns whether TrustedNetwork is state "trusted".
func (t TrustedNetwork) Trusted() bool {
	return t == TrustedNetworkTrusted
}

// String returns t as string.
func (t TrustedNetwork) String() string {
	switch t {
	case TrustedNetworkUnknown:
		return "unknown"
	case TrustedNetworkNotTrusted:
		return "not trusted"
	case TrustedNetworkTrusted:
		return "trusted"
	}
	return ""
}

// ConnectionState is the current connection state.
type ConnectionState uint32

// ConnectionState states.
const (
	ConnectionStateUnknown ConnectionState = iota
	ConnectionStateDisconnected
	ConnectionStateConnecting
	ConnectionStateConnected
	ConnectionStateDisconnecting
)

// Connecting returns whether ConnectionState is state "connecting".
func (c ConnectionState) Connecting() bool {
	return c == ConnectionStateConnecting
}

// Connected returns whether ConnectionState is state "connected".
func (c ConnectionState) Connected() bool {
	return c == ConnectionStateConnected
}

// Disconnected returns whether ConnectionState is state "disconnected".
func (c ConnectionState) Disconnected() bool {
	return c == ConnectionStateDisconnected
}

// String returns ConnectionState as string.
func (c ConnectionState) String() string {
	switch c {
	case ConnectionStateUnknown:
		return "unknown"
	case ConnectionStateDisconnected:
		return "disconnected"
	case ConnectionStateConnecting:
		return "connecting"
	case ConnectionStateConnected:
		return "connected"
	case ConnectionStateDisconnecting:
		return "disconnecting"
	}
	return ""
}

// OCRunning is the current state of the openconnect client.
type OCRunning uint32

// OCRunning states.
const (
	OCRunningUnknown OCRunning = iota
	OCRunningNotRunning
	OCRunningRunning
)

// Running returns whether OCRunning is in state "running".
func (o OCRunning) Running() bool {
	return o == OCRunningRunning
}

// String returns OCRunning as string.
func (o OCRunning) String() string {
	switch o {
	case OCRunningUnknown:
		return "unknown"
	case OCRunningNotRunning:
		return "not running"
	case OCRunningRunning:
		return "running"
	}
	return ""
}

// TrafPolState is the current TrafPol state.
type TrafPolState uint32

// TrafPolState states.
const (
	TrafPolStateUnknown = iota
	TrafPolStateInactive
	TrafPolStateActive
	TrafPolStateDisabled
)

// String resturns TrafPolState as string.
func (t TrafPolState) String() string {
	switch t {
	case TrafPolStateUnknown:
		return "unknown"
	case TrafPolStateInactive:
		return "inactive"
	case TrafPolStateActive:
		return "active"
	case TrafPolStateDisabled:
		return "disabled"
	}
	return ""
}

// TNDState is the current TND state.
type TNDState uint32

// TNDState states.
const (
	TNDStateUnknown = iota
	TNDStateInactive
	TNDStateActive
)

// String resturns TNDState as string.
func (t TNDState) String() string {
	switch t {
	case TNDStateUnknown:
		return "unknown"
	case TNDStateInactive:
		return "inactive"
	case TNDStateActive:
		return "active"
	}
	return ""
}

// Status is a VPN status.
type Status struct {
	TrustedNetwork  TrustedNetwork
	ConnectionState ConnectionState
	IP              string
	Device          string
	Server          string
	ServerIP        string
	ConnectedAt     int64
	Servers         []string
	OCRunning       OCRunning
	OCPID           uint32
	TrafPolState    TrafPolState
	AllowedHosts    []string
	TNDState        TNDState
	TNDServers      []string
	VPNConfig       *vpnconfig.Config
}

// Copy returns a copy of Status.
func (s *Status) Copy() *Status {
	if s == nil {
		return nil
	}

	return &Status{
		TrustedNetwork:  s.TrustedNetwork,
		ConnectionState: s.ConnectionState,
		IP:              s.IP,
		Device:          s.Device,
		Server:          s.Server,
		ServerIP:        s.ServerIP,
		ConnectedAt:     s.ConnectedAt,
		Servers:         append(s.Servers[:0:0], s.Servers...),
		OCRunning:       s.OCRunning,
		OCPID:           s.OCPID,
		TrafPolState:    s.TrafPolState,
		AllowedHosts:    append(s.AllowedHosts[:0:0], s.AllowedHosts...),
		TNDState:        s.TNDState,
		TNDServers:      append(s.TNDServers[:0:0], s.TNDServers...),
		VPNConfig:       s.VPNConfig.Copy(),
	}
}

// JSON returns the Status as JSON.
func (s *Status) JSON() ([]byte, error) {
	return json.Marshal(s)
}

// NewFromJSON parses and returns the Status in b.
func NewFromJSON(b []byte) (*Status, error) {
	s := New()
	err := json.Unmarshal(b, s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// New returns a new Status.
func New() *Status {
	return &Status{
		ConnectedAt: -1,
	}
}
