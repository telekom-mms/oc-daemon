package vpnstatus

import (
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// TestTrustedNetworkTrusted tests Trusted of TrustedNetwork.
func TestTrustedNetworkTrusted(t *testing.T) {
	// test not trusted
	for i, notTrusted := range []TrustedNetwork{
		TrustedNetworkUnknown,
		TrustedNetworkNotTrusted,
	} {
		if notTrusted.Trusted() {
			t.Errorf("should not be trusted: %d, %s", i, notTrusted)
		}
	}

	// test trusted
	if !TrustedNetworkTrusted.Trusted() {
		t.Errorf("should be trusted: %s", TrustedNetworkTrusted)
	}
}

// TestTrustedNetworkString tests String of TrustedNetwork.
func TestTrustedNetworkString(t *testing.T) {
	for v, s := range map[TrustedNetwork]string{
		// valid
		TrustedNetworkUnknown:    "unknown",
		TrustedNetworkNotTrusted: "not trusted",
		TrustedNetworkTrusted:    "trusted",

		// invalid
		123456: "",
	} {
		if v.String() != s {
			t.Errorf("got %s, want %s", v.String(), s)
		}
	}
}

// TestConnectionStateConnecting tests Connecting of ConnectionState.
func TestConnectionStateConnecting(t *testing.T) {
	// test not connecting
	for i, notConnecting := range []ConnectionState{
		ConnectionStateUnknown,
		ConnectionStateDisconnected,
		ConnectionStateConnected,
		ConnectionStateDisconnecting,
	} {
		if notConnecting.Connecting() {
			t.Errorf("should not be connecting: %d, %s", i, notConnecting)
		}
	}

	// test connecting
	if !ConnectionStateConnecting.Connecting() {
		t.Errorf("should be connecting: %s", ConnectionStateConnecting)
	}
}

// TestConnectionStateConnected tests Connected of ConnectionState.
func TestConnectionStateConnected(t *testing.T) {
	// test not connected
	for i, notConnected := range []ConnectionState{
		ConnectionStateUnknown,
		ConnectionStateDisconnected,
		ConnectionStateConnecting,
		ConnectionStateDisconnecting,
	} {
		if notConnected.Connected() {
			t.Errorf("should not be connected: %d, %s", i, notConnected)
		}
	}

	// test connected
	if !ConnectionStateConnected.Connected() {
		t.Errorf("should be connected: %s", ConnectionStateConnected)
	}
}

// TestConnectionStateDisconnected tests Disconnected of ConnectionState.
func TestConnectionStateDisconnected(t *testing.T) {
	// test not disconnected
	for i, notDisconnected := range []ConnectionState{
		ConnectionStateUnknown,
		ConnectionStateConnecting,
		ConnectionStateConnected,
		ConnectionStateDisconnecting,
	} {
		if notDisconnected.Disconnected() {
			t.Errorf("should not be disconnected: %d, %s", i, notDisconnected)
		}
	}

	// test disconnected
	if !ConnectionStateDisconnected.Disconnected() {
		t.Errorf("should be disconnected: %s", ConnectionStateDisconnected)
	}
}

// TestConnectionStateString tests String of ConnectionState.
func TestConnectionStateString(t *testing.T) {
	for v, s := range map[ConnectionState]string{
		// valid
		ConnectionStateUnknown:       "unknown",
		ConnectionStateDisconnected:  "disconnected",
		ConnectionStateConnecting:    "connecting",
		ConnectionStateConnected:     "connected",
		ConnectionStateDisconnecting: "disconnecting",

		// invalid
		123456: "",
	} {
		if v.String() != s {
			t.Errorf("got %s, want %s", v.String(), s)
		}
	}
}

// TestOCRunningRunning tests Running of OCRunning.
func TestOCRunningRunning(t *testing.T) {
	// test not running
	for i, notRunning := range []OCRunning{
		OCRunningUnknown,
		OCRunningNotRunning,
	} {
		if notRunning.Running() {
			t.Errorf("should not be running: %d, %s", i, notRunning)
		}
	}

	// test running
	if !OCRunningRunning.Running() {
		t.Errorf("should be running: %s", OCRunningRunning)
	}
}

// TestOCRunningString tests String of OCRunning.
func TestOCRunningString(t *testing.T) {
	for v, s := range map[OCRunning]string{
		// valid
		OCRunningUnknown:    "unknown",
		OCRunningNotRunning: "not running",
		OCRunningRunning:    "running",

		// invalid
		123456: "",
	} {
		if v.String() != s {
			t.Errorf("got %s, want %s", v.String(), s)
		}
	}
}

// TestTrafPolStateString tests String of TrafPolState.
func TestTrafPolStateString(t *testing.T) {
	for v, s := range map[TrafPolState]string{
		// valid
		TrafPolStateUnknown:  "unknown",
		TrafPolStateInactive: "inactive",
		TrafPolStateActive:   "active",
		TrafPolStateDisabled: "disabled",

		// invalid
		123456: "",
	} {
		if v.String() != s {
			t.Errorf("got %s, want %s", v.String(), s)
		}
	}
}

// TestCaptivePortalString tests String of CaptivePortal.
func TestCaptivePortalString(t *testing.T) {
	for v, s := range map[CaptivePortal]string{
		// valid
		CaptivePortalUnknown:     "unknown",
		CaptivePortalNotDetected: "not detected",
		CaptivePortalDetected:    "detected",

		// invalid
		123456: "",
	} {
		if v.String() != s {
			t.Errorf("got %s, want %s", v.String(), s)
		}
	}
}

// TestTNDStateString tests String of TNDState.
func TestTNDStateString(t *testing.T) {
	for v, s := range map[TNDState]string{
		// valid
		TNDStateUnknown:  "unknown",
		TNDStateInactive: "inactive",
		TNDStateActive:   "active",

		// invalid
		123456: "",
	} {
		if v.String() != s {
			t.Errorf("got %s, want %s", v.String(), s)
		}
	}
}

// TestStatusCopy tests Copy of Status.
func TestStatusCopy(t *testing.T) {
	// test nil
	if (*Status)(nil).Copy() != nil {
		t.Error("copy of nil status should be nil")
	}

	// test valid
	for _, want := range []*Status{
		New(),
		{
			TrustedNetwork:  TrustedNetworkNotTrusted,
			ConnectionState: ConnectionStateConnected,
			IP:              "192.168.1.1",
			Device:          "test-tun0",
			Server:          "test server 1",
			ServerIP:        "10.0.0.1",
			ConnectedAt:     1700000000,
			Servers:         []string{"test server 1", "test server 2"},
			OCRunning:       OCRunningRunning,
			OCPID:           12345,
			TrafPolState:    TrafPolStateActive,
			AllowedHosts:    []string{"test.example.com"},
			CaptivePortal:   CaptivePortalNotDetected,
			TNDState:        TNDStateActive,
			TNDServers:      []string{"tnd1.local:abcdef..."},
			VPNConfig:       vpnconfig.New(),
		},
	} {
		got := want.Copy()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

// TestJSON tests JSON and NewFromJSON of Status.
func TestJSON(t *testing.T) {
	// test without json errors
	s := New()
	b, err := s.JSON()
	if err != nil {
		t.Fatal(err)
	}
	n, err := NewFromJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(n, s) {
		t.Errorf("got %v, want %v", n, s)
	}

	// unmarshal error
	if _, err := NewFromJSON(nil); err == nil {
		t.Error("unmarshal error should return error")
	}
}

// TestNew tests New.
func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Errorf("got nil, want != nil")
	}
}
