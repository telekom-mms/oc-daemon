package cmdtmpl

import (
	"context"
	"encoding/json"
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

	// invalid command list name in file
	lists := []*CommandList{
		{Name: "does not exist"},
	}
	b, err := json.Marshal(lists)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadCommandLists(f); err == nil {
		t.Errorf("invalid command list name should return error")
	}

	// valid file, update command lists
	oldTrafPolCleanup := commandLists[TrafPolCleanup].Commands
	oldVPNSetupCleanup := commandLists[VPNSetupCleanup].Commands
	defer func() {
		commandLists[TrafPolCleanup].Commands = oldTrafPolCleanup
		commandLists[VPNSetupCleanup].Commands = oldVPNSetupCleanup
	}()

	lists = []*CommandList{
		{Name: TrafPolCleanup},
		{Name: VPNSetupCleanup},
	}

	b, err = json.Marshal(lists)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, b, 0600); err != nil {
		t.Fatal(err)
	}

	if err := LoadCommandLists(f); err != nil {
		t.Errorf("loading valid file returned error: %s", err)
	}
	if len(oldTrafPolCleanup) == len(commandLists[TrafPolCleanup].Commands) ||
		len(oldVPNSetupCleanup) == len(commandLists[VPNSetupCleanup].Commands) {

		t.Error("loading valid file did not change command lists")
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
