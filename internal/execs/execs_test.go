package execs

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

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

// TestSetExecutables tests SetExecutables.
func TestSetExecutables(t *testing.T) {
	old := daemoncfg.NewExecutables()
	defer SetExecutables(old)

	config := &daemoncfg.Executables{
		IP:         "/test/ip",
		Sysctl:     "/test/sysctl",
		Nft:        "/test/nft",
		Resolvectl: "/test/resolvectl",
	}
	SetExecutables(config)
	if ip != config.IP ||
		sysctl != config.Sysctl ||
		nft != config.Nft ||
		resolvectl != config.Resolvectl {
		// executables not set properly
		t.Errorf("executables incorrect")
	}
}
