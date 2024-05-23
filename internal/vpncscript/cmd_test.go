package vpncscript

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// TestRun tests run.
func TestRun(t *testing.T) {
	// test invalid arg
	if err := run([]string{"test", "-invalid"}); err == nil || err == flag.ErrHelp {
		t.Errorf("invalid argument should return error, got: %v", err)
	}

	// test with "-version"
	if err := run([]string{"test", "-version"}); err != nil {
		t.Errorf("version should not return error, got: %v", err)
	}

	// test with "-help"
	if err := run([]string{"test", "-help"}); err != flag.ErrHelp {
		t.Errorf("help should return ErrHelp, got: %v", err)
	}

	// test with invalid token
	t.Setenv("oc_daemon_token", "this is not a valid encoded token!")
	if err := run([]string{"test"}); err == nil {
		t.Errorf("invalid token should return error")
	}

	// test connect with config creation error, invalid VPN PID
	os.Clearenv()
	t.Setenv("reason", "connect")
	t.Setenv("VPNPID", "not a valid vpn pid!")
	if err := run([]string{"test"}); err == nil {
		t.Errorf("invalid config should return error")
	}

	// prepare environment with not existing sockfile
	os.Clearenv()
	sockfile := filepath.Join(t.TempDir(), "sockfile")
	t.Setenv("oc_daemon_socket_file", sockfile)
	t.Setenv("oc_daemon_verbose", "true")

	// test with errors
	for _, v := range []string{
		"pre-init",
		"connect",
		"disconnect",
		"attempt-reconnect",
		"reconnect",
		"invalid",
	} {
		t.Setenv("reason", v)
		if err := run([]string{"test"}); err == nil {
			t.Errorf("%s: should return error", v)
		}
	}
}
