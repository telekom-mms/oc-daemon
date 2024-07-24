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
			Server:          "test server 1",
			ConnectedAt:     1700000000,
			Servers:         []string{"test server 1", "test server 2"},
			OCRunning:       OCRunningRunning,
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
