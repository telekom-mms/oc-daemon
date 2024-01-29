package client

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/telekom-mms/oc-daemon/pkg/client"
)

// TestRun tests run.
func TestRun(t *testing.T) {
	// test invalid arg
	if err := run([]string{"test", "-invalid"}); err == nil || err == flag.ErrHelp {
		t.Errorf("invalid argument should return error, got: %v", err)
	}

	// test with "-version"
	if err := run([]string{"test", "-version"}); err != flag.ErrHelp {
		t.Errorf("version should not return error, got: %v", err)
	}

	// test with "-help"
	if err := run([]string{"test", "-help"}); err != flag.ErrHelp {
		t.Errorf("help should return ErrHelp, got: %v", err)
	}

	// test with "status -help"
	if err := run([]string{"test", "status", "-help"}); err != flag.ErrHelp {
		t.Errorf("help should return ErrHelp, got: %v", err)
	}

	// not existing config with "-config"
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")
	if err := run([]string{"test", "-config", cfg}); err == nil || err == flag.ErrHelp {
		t.Errorf("not existing config should return error, got: %v", err)
	}

	// invalid config with "-config"
	if err := os.WriteFile(cfg, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"test", "-config", cfg}); err == nil || err == flag.ErrHelp {
		t.Errorf("invalid config should return error, got: %v", err)
	}

	// not existing system config
	clientLoadUserSystemConfig = func() *client.Config {
		return nil
	}
	defer func() { clientLoadUserSystemConfig = client.LoadUserSystemConfig }()

	// invalid command
	if err := run([]string{"test",
		//"-config", cfg,
		"-cert", "cert-file",
		"-key", "key-file",
		"-ca", "ca-file",
		"-profile", "profile-file",
		"-server", "test-server",
		"-user", "test-user",
		"invalid-command",
	}); err == nil || err == flag.ErrHelp {
		t.Errorf("invalid command should return error, got: %v", err)
	}

	// save user config
	clientUserConfig = func() string {
		return filepath.Join(dir, "user-config")
	}
	defer func() { clientUserConfig = client.UserConfig }()

	if err := run([]string{"test",
		//"-config", cfg,
		"-cert", "cert-file",
		"-key", "key-file",
		"-ca", "ca-file",
		"-profile", "profile-file",
		"-server", "test-server",
		"-user", "test-user",
		"save",
	}); err != nil {
		t.Errorf("save should not return error, got: %v", err)
	}
}
