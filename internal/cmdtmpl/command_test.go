package cmdtmpl

import (
	"context"
	"testing"
	"text/template"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// TestExecuteTemplateErrors tests executeTemplate of CommandList, parse error.
func TestExecuteTemplateParseError(t *testing.T) {
	cl := &CommandList{
		template: template.Must(template.New("test").Parse("test")),
	}
	if _, err := cl.executeTemplate("{{ invalid }}", nil); err == nil {
		t.Error("invalid template should not parse correctly")
	}
}

// TestGetCommandList tests getCommandList.
func TestGetCommandList(t *testing.T) {
	// not existing
	for _, name := range []string{
		"TrafPolDoesNotExist",
		"VPNSetupDoesNotExist",
		"DoesNotExist",
	} {
		cl := getCommandList(name)
		if cl != nil {
			t.Errorf("command list %s should not exists, got %s", name, cl.Name)
		}
	}

	// existing
	for _, name := range []string{
		// Traffic Policing
		"TrafPolSetFilterRules",
		"TrafPolUnsetFilterRules",
		"TrafPolSetAllowedDevices",
		"TrafPolFlushAllowedHosts",
		"TrafPolAddAllowedHost",
		"TrafPolSetAllowedPorts",
		"TrafPolCleanup",

		// VPN Setup
		"VPNSetupSetup",
		"VPNSetupTeardown",
		"VPNSetupSetExcludes",
		"VPNSetupSetDNS",
		"VPNSetupGetDNS",
		"VPNSetupCleanup",
	} {
		cl := getCommandList(name)
		if cl.Name != name {
			t.Errorf("command list should be %s, got %s", name, cl.Name)
		}
	}
}

// TestCmdRun tests Run of Cmd.
func TestCmdRun(t *testing.T) {
	cmd := &Cmd{
		Cmd:  "echo",
		Args: []string{"this", "is", "a", "test"},
	}
	stdout, _, err := cmd.Run(context.Background())
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}
	if string(stdout) != "this is a test\n" {
		t.Errorf("unexpected stdout: %s", stdout)
	}
}

// TestGetCmds tets GetCmds.
func TestGetCmds(t *testing.T) {
	// not existing
	if _, err := GetCmds("DoesNotExist", nil); err == nil {
		t.Error("not existing command list should return error")
	}

	// existing, that only need daemon config as input data
	for _, name := range []string{
		// Traffic Policing
		"TrafPolSetFilterRules",
		"TrafPolUnsetFilterRules",
		// TrafPolSetAllowedDevices", // skip, requires devices
		"TrafPolFlushAllowedHosts",
		// "TrafPolAddAllowedHost", // skip, requires host
		//"TrafPolSetAllowedPorts", // skip, requires ports
		"TrafPolCleanup",

		// VPN Setup
		"VPNSetupSetup",
		"VPNSetupTeardown",
		// "VPNSetupSetExcludes", // skip, requires excludes
		"VPNSetupSetDNS",
		"VPNSetupGetDNS",
		"VPNSetupCleanup",
	} {
		if cmds, err := GetCmds(name, daemoncfg.NewConfig()); err != nil ||
			len(cmds) == 0 {

			t.Errorf("got invalid command list for name %s", name)
		}
	}

	// existing, with insufficient input data
	for _, name := range []string{
		// Traffic Policing
		"TrafPolSetAllowedDevices",
		"TrafPolAddAllowedHost",
		"TrafPolSetAllowedPorts",

		// VPN Setup
		"VPNSetupSetExcludes",
	} {
		if _, err := GetCmds(name, daemoncfg.NewConfig()); err == nil {
			t.Errorf("insufficient data should return error for list %s", name)
		}
	}
}
