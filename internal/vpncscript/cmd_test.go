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

	// prepare environment with not existing sockfile
	os.Clearenv()
	sockfile := filepath.Join(t.TempDir(), "sockfile")
	t.Setenv("oc_daemon_socket_file", sockfile)
	t.Setenv("oc_daemon_verbose", "true")

	// test with errors
	for _, v := range []string{
		"connect",
		"disconnect",
		"invalid",
	} {
		t.Setenv("reason", v)
		if err := run([]string{"test"}); err == nil {
			t.Errorf("%s: should return error", v)
		}
	}

	// test without errors
	for _, v := range []string{
		"pre-init",
		"attempt-reconnect",
		"reconnect",
	} {
		t.Setenv("reason", v)
		if err := run([]string{"test"}); err != nil {
			t.Errorf("%s: should not return error, got: %v", v, err)
		}
	}
}
