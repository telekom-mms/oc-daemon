package cmdtmpl

import (
	"context"
	"os"
	"path/filepath"
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

// TestLoadCommandLists tests LoadCommandLists.
func TestLoadCommandLists(t *testing.T) {
	dir := t.TempDir()

	// not existing file
	if err := LoadCommandLists(filepath.Join(dir, "does not exist")); err == nil {
		t.Errorf("not existing file should return error")
	}

	// invalid json file
	f := filepath.Join(dir, "command-lists.json")
	if err := os.WriteFile(f, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadCommandLists(f); err == nil {
		t.Errorf("invalid json should return error")
	}

	b := []byte(`[{"Name":"does not exist"}]`)
	if err := os.WriteFile(f, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadCommandLists(f); err == nil {
		t.Errorf("invalid command list name should return error")
	}

	// valid file, update command lists
	oldTrafPolCleanup := CommandLists[TrafPolCleanup].Commands
	oldVPNSetupCleanup := CommandLists[VPNSetupCleanup].Commands
	defer func() {
		// cleanup after test
		CommandLists[TrafPolCleanup].Commands = oldTrafPolCleanup
		CommandLists[VPNSetupCleanup].Commands = oldVPNSetupCleanup
	}()

	// load TrafPolCleanup and VPNSetupCleanup from file
	b = []byte(`[
{
	"Name":"TrafPolCleanup",
	"Commands": [
		{"Line": "echo TrafPol"},
		{"Line": "echo", "Stdin": "Cleanup"}
	]
},
{
	"Name":"VPNSetupCleanup",
	"Commands": [
		{"Line": "echo VPNSetup"},
		{"Line": "echo", "Stdin": "Cleanup"}
	]
}
]`)
	if err := os.WriteFile(f, b, 0600); err != nil {
		t.Fatal(err)
	}

	if err := LoadCommandLists(f); err != nil {
		t.Errorf("loading valid file returned error: %s", err)
	}

	// make sure all lists still contain commands
	for k, v := range CommandLists {
		if len(v.Commands) == 0 {
			t.Errorf("invalid command list length in %s after load", k)
		}
	}
	// check TrafPolCleanup
	commands := CommandLists[TrafPolCleanup].Commands
	if commands[0].Line != "echo TrafPol" ||
		commands[0].Stdin != "" ||
		commands[1].Line != "echo" ||
		commands[1].Stdin != "Cleanup" {

		t.Errorf("invalid command in TrafPolCleanup after successfull load: %v, %v",
			commands[0], commands[1])
	}
	// check VPNSetupCleanup
	commands = CommandLists[VPNSetupCleanup].Commands
	if commands[0].Line != "echo VPNSetup" ||
		commands[0].Stdin != "" ||
		commands[1].Line != "echo" ||
		commands[1].Stdin != "Cleanup" {

		t.Errorf("invalid command in VPNSetupCleanup after successfull load: %v, %v",
			commands[0], commands[1])
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
		"TrafPolSetAllowedHosts",
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

// TestRunCmd tests RunCmd.
func TestRunCmd(t *testing.T) {
	ctx := context.Background()

	// test not existing
	dir := t.TempDir()
	if _, _, err := RunCmd(ctx, filepath.Join(dir, "does/not/exist"), ""); err == nil {
		t.Errorf("running not existing command should fail: %v", err)
	}

	// test existing
	if _, _, err := RunCmd(ctx, "echo", "", "this", "is", "a", "test"); err != nil {
		t.Errorf("running echo failed: %v", err)
	}

	// test with stdin
	if _, _, err := RunCmd(ctx, "echo", "this is a test"); err != nil {
		t.Errorf("running echo failed: %v", err)
	}

	// test stdout
	stdout, stderr, err := RunCmd(ctx, "cat", "this is a test")
	if err != nil || string(stdout) != "this is a test" {
		t.Errorf("running echo failed: %s, %s, %v", stdout, stderr, err)
	}

	// test stderr and error
	stdout, stderr, err = RunCmd(ctx, "cat", "", "does/not/exist")
	if err == nil || string(stderr) != "cat: does/not/exist: No such file or directory\n" {
		t.Errorf("running echo failed: %s, %s, %v", stdout, stderr, err)
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
		// "TrafPolSetAllowedHosts", // skip, requires hosts
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
		"TrafPolSetAllowedHosts",
		"TrafPolSetAllowedPorts",

		// VPN Setup
		"VPNSetupSetExcludes",
	} {
		if _, err := GetCmds(name, daemoncfg.NewConfig()); err == nil {
			t.Errorf("insufficient data should return error for list %s", name)
		}
	}
}
