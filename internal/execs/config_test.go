package execs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestConfigValid tests Valid of Config.
func TestConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Config{
		nil,
		{},
	} {
		if invalid.Valid() {
			t.Errorf("config should be invalid: %v", invalid)
		}
	}

	// test valid
	for _, valid := range []*Config{
		NewConfig(),
		{"/test/ip", "/test/nft", "/test/resolvectl", "/test/sysctl"},
	} {
		if !valid.Valid() {
			t.Errorf("config should be valid: %v", valid)
		}
	}
}

// TestConfigCheckExecutables tests CheckExecutables of Config.
func TestConfigCheckExecutables(t *testing.T) {
	// create temporary dir for executables
	dir, err := os.MkdirTemp("", "execs-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// create executable file paths
	ip := filepath.Join(dir, "ip")
	nft := filepath.Join(dir, "nft")
	resolvectl := filepath.Join(dir, "resolvectl")
	sysctl := filepath.Join(dir, "sysctl")

	// create config with executables
	c := &Config{
		IP:         ip,
		Nft:        nft,
		Resolvectl: resolvectl,
		Sysctl:     sysctl,
	}

	// test with not all files existing, create files in the process
	for _, f := range []string{
		ip, nft, resolvectl, sysctl,
	} {
		// test
		if got := c.CheckExecutables(); got == nil {
			t.Errorf("got nil, want != nil")
		}

		// create executable file
		if err := os.WriteFile(f, []byte{}, 0777); err != nil {
			t.Fatal(err)
		}
	}

	// test with all files existing
	if got := c.CheckExecutables(); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// TestNewConfig tests NewConfig.
func TestNewConfig(t *testing.T) {
	want := &Config{IP, Nft, Resolvectl, Sysctl}
	got := NewConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
