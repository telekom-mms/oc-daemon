package ocrunner

import (
	"reflect"
	"testing"
)

// TestConfigValid tests Valid of Config.
func TestConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Config{
		nil,
		{},
		{
			OpenConnect:    "openconnect",
			XMLProfile:     "/test/profile",
			VPNCScript:     "/test/vpncscript",
			VPNDevice:      "test-device",
			PIDFile:        "/test/pid",
			PIDPermissions: "invalid",
		},
		{
			OpenConnect:    "openconnect",
			XMLProfile:     "/test/profile",
			VPNCScript:     "/test/vpncscript",
			VPNDevice:      "test-device",
			PIDFile:        "/test/pid",
			PIDPermissions: "1234",
		},
	} {
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	for _, valid := range []*Config{
		NewConfig(),
		{
			OpenConnect:    "openconnect",
			XMLProfile:     "/test/profile",
			VPNCScript:     "/test/vpncscript",
			VPNDevice:      "test-device",
			PIDFile:        "/test/pid",
			PIDPermissions: "777",
		},
	} {
		want := true
		got := valid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, valid)
		}
	}
}

// TestNewConfig tests NewConfig.
func TestNewConfig(t *testing.T) {
	want := &Config{
		OpenConnect: OpenConnect,

		XMLProfile: XMLProfile,
		VPNCScript: VPNCScript,
		VPNDevice:  VPNDevice,

		PIDFile:        PIDFile,
		PIDOwner:       PIDOwner,
		PIDGroup:       PIDGroup,
		PIDPermissions: PIDPermissions,

		NoProxy:   NoProxy,
		ExtraEnv:  ExtraEnv,
		ExtraArgs: ExtraArgs,
	}
	got := NewConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
