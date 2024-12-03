package execs

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
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

// TestRunIP tests RunIP.
func TestRunIP(t *testing.T) {
	want := []string{"ip address show"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunIP(context.Background(), "address", "show")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunIPLink tests RunIPLink.
func TestRunIPLink(t *testing.T) {
	want := []string{"ip link show"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunIPLink(context.Background(), "show")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunIPAddress tests RunIPAddress.
func TestRunIPAddress(t *testing.T) {
	want := []string{"ip address show"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunIPAddress(context.Background(), "show")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunIP4Route tests RunIP4Route.
func TestRunIP4Route(t *testing.T) {
	want := []string{"ip -4 route show"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunIP4Route(context.Background(), "show")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunIP6Route tests RunIP6Route.
func TestRunIP6Route(t *testing.T) {
	want := []string{"ip -6 route show"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunIP6Route(context.Background(), "show")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunIP4Rule tests RunIP4Rule.
func TestRunIP4Rule(t *testing.T) {
	want := []string{"ip -4 rule show"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunIP4Rule(context.Background(), "show")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunIP6Rule tests RunIP6Rule.
func TestRunIP6Rule(t *testing.T) {
	want := []string{"ip -6 rule show"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunIP6Rule(context.Background(), "show")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunSysctl tests RunSysctl.
func TestRunSysctl(t *testing.T) {
	want := []string{"sysctl -q net.ipv4.conf.all.src_valid_mark=1"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunSysctl(context.Background(), "-q", "net.ipv4.conf.all.src_valid_mark=1")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunNft tests RunNft.
func TestRunNft(t *testing.T) {
	want := []string{"nft -f - list tables"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, s string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " ")+" "+s)
		return nil, nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	_, _, _ = RunNft(context.Background(), "list tables")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestRunResolvectl tests RunResolvectl.
func TestRunResolvectl(t *testing.T) {
	want := []string{"resolvectl dns"}
	got := []string{}

	oldRunCmd := RunCmd
	RunCmd = func(_ context.Context, cmd string, _ string, arg ...string) ([]byte, []byte, error) {
		got = append(got, cmd+" "+strings.Join(arg, " "))
		return []byte("OK"), nil, nil
	}
	defer func() { RunCmd = oldRunCmd }()

	if b, _, err := RunResolvectl(context.Background(), "dns"); err != nil || string(b) != "OK" {
		t.Errorf("invalid return values %s, %v", b, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
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
