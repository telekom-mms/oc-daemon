package client

import (
	"errors"
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

	// not existing system config with "-system-settings"
	clientSystemConfig = func() string {
		return filepath.Join(dir, "system-config")
	}
	defer func() { clientSystemConfig = client.SystemConfig }()

	if err := run([]string{"test",
		"-system-settings",
	}); err == nil || err == flag.ErrHelp {
		t.Errorf("no config should return error, got: %v", err)
	}

	// invalid config with "-system-settings"
	if err := os.WriteFile(filepath.Join(dir, "system-config"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"test",
		"-system-settings",
	}); err == nil || err == flag.ErrHelp {
		t.Errorf("invalid config should return error, got: %v", err)
	}

	// not existing user/system config
	clientLoadUserSystemConfig = func() *client.Config {
		return nil
	}
	defer func() { clientLoadUserSystemConfig = client.LoadUserSystemConfig }()

	// invalid command
	if err := run([]string{"test",
		"-cert", "cert-file",
		"-key", "key-file",
		"-user-cert", "user-cert-file",
		"-user-key", "user-key-file",
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
		"-cert", "cert-file",
		"-key", "key-file",
		"-user-cert", "user-cert-file",
		"-user-key", "user-key-file",
		"-ca", "ca-file",
		"-profile", "profile-file",
		"-server", "test-server",
		"-user", "test-user",
		"save",
	}); err != nil {
		t.Errorf("save should not return error, got: %v", err)
	}

	// test commands
	clientNewClient = func(*client.Config) (client.Client, error) {
		return nil, errors.New("test error")
	}
	defer func() { clientNewClient = client.NewClient }()

	for _, cmd := range []string{
		"list",
		"",
		"connect",
		"disconnect",
		"reconnect",
		"status",
		"monitor",
	} {
		if err := run([]string{"test",
			"-cert", "cert-file",
			"-key", "key-file",
			"-user-cert", "user-cert-file",
			"-user-key", "user-key-file",
			"-ca", "ca-file",
			"-profile", "profile-file",
			"-server", "test-server",
			"-user", "test-user",
			cmd,
		}); err == nil || err == flag.ErrHelp {
			t.Errorf("command %s should return error, got: %v", cmd, err)
		}
	}
}
