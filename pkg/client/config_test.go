package client

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestConfigCopy tests Copy of Config.
func TestConfigCopy(t *testing.T) {
	// test nil
	if (*Config)(nil).Copy() != nil {
		t.Error("copy of nil should be nil")
	}

	// test new config
	want := NewConfig()
	got := want.Copy()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test slice manipulation after copy
	src := NewConfig()
	src.ExtraEnv = append(src.ExtraEnv, "TestEnv=Before")
	src.ExtraArgs = append(src.ExtraArgs, "--TestArg=Before")

	dst := src.Copy()
	if !reflect.DeepEqual(dst, src) {
		t.Errorf("%v and %v should still match", dst, src)
	}

	dst.ExtraEnv[0] = "TestEnv=After"
	dst.ExtraArgs[0] = "--TestArg=After"
	if reflect.DeepEqual(dst, src) {
		t.Errorf("%v and %v should not match anymore", dst, src)
	}
}

// TestConfigEmpty tests Empty of Config.
func TestConfigEmpty(t *testing.T) {
	// test empty
	c := &Config{}
	want := true
	got := c.Empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test not empty
	c.User = "User"
	want = false
	got = c.Empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test nil
	c = nil
	want = true
	got = c.Empty()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestConfigValid tests Valid of Config.
func TestConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Config{
		nil,
		{},
	} {
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	valid := NewConfig()
	valid.ClientCertificate = "test-cert"
	valid.ClientKey = "test-key"
	valid.VPNServer = "test-server"
	want := true
	got := valid.Valid()

	if got != want {
		t.Errorf("got %t, want %t for %v", got, want, valid)
	}
}

// TestConfigExpand tests Expand of Config.
func TestConfigExpand(t *testing.T) {
	// get user home dir
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	// test expanding dirs
	want := NewConfig()
	want.ClientCertificate = filepath.Join(home, "test-certs/cert")
	want.ClientKey = filepath.Join(home, "test-certs/key")
	want.CACertificate = filepath.Join(home, "test-certs/ca")

	got := NewConfig()
	got.ClientCertificate = "~/test-certs/cert"
	got.ClientKey = "$HOME/test-certs/key"
	got.CACertificate = filepath.Join(home, "test-certs/ca")
	got.Expand()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewConfig tests NewConfig.
func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if c.Empty() {
		t.Errorf("got empty, want not empty")
	}
}

// TestLoadConfig tests Save of Config and LoadConfig.
func TestLoadConfig(t *testing.T) {
	// create test config
	want := &Config{
		ClientCertificate: "/some/cert",
		ClientKey:         "/some/key",
		CACertificate:     "/some/ca",
		XMLProfile:        "/some/profile",
		VPNServer:         "server.example.com",
		User:              "user1",
		UserGroup:         "group1",

		OpenConnect: "openconnect",
		Protocol:    "test",
		UserAgent:   "agent",
		Quiet:       true,
		NoProxy:     true,
		ExtraEnv:    []string{"oc_daemon_var_is_not=used"},
		ExtraArgs:   []string{"--arg-does-not=exist"},
	}

	// create temporary file
	f, err := os.CreateTemp("", "oc-client-config")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()

	// test not existing file
	if _, err := LoadConfig(f.Name() + "does not exists"); err == nil {
		t.Error("loading not existing file should return error")
	}

	// test with empty file
	if _, err := LoadConfig(f.Name()); err == nil {
		t.Error("loading empty file should return error")
	}

	// save config to temporary file with error
	jsonMarshalIndent = func(any, string, string) ([]byte, error) {
		return nil, errors.New("test error")
	}
	if err := want.Save(f.Name()); err == nil {
		t.Error("save with error should return error")
	}
	jsonMarshalIndent = json.MarshalIndent

	// save config to temporary file
	if err := want.Save(f.Name()); err != nil {
		t.Error(err)
	}

	// load config from temporary file
	got, err := LoadConfig(f.Name())
	if err != nil {
		t.Error(err)
	}

	// make sure configs are equal
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestLoadUserSystemConfig tests LoadUserSystemConfig.
func TestLoadUserSystemConfig(t *testing.T) {
	// clean up after tests
	oldSys := SystemConfigDirPath
	defer func() { SystemConfigDirPath = oldSys }()
	defer func() { osUserConfigDir = os.UserConfigDir }()

	// set new system config path
	SystemConfigDirPath = t.TempDir()

	// test with user config path error and no system config
	osUserConfigDir = func() (string, error) {
		return "", errors.New("test error")
	}
	if c := LoadUserSystemConfig(); c != nil {
		t.Errorf("got %v, want nil", c)
	}

	// set new user config path
	userConfDir := t.TempDir()
	osUserConfigDir = func() (string, error) {
		return userConfDir, nil
	}

	// test no existing configs
	if c := LoadUserSystemConfig(); c != nil {
		t.Errorf("got %v, want nil", c)
	}

	// create test config
	conf := NewConfig()

	// test with system config only
	if err := os.MkdirAll(filepath.Dir(SystemConfig()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := conf.Save(SystemConfig()); err != nil {
		t.Fatal(err)
	}
	if c := LoadUserSystemConfig(); c == nil {
		t.Errorf("got nil, want %v", conf)
	}

	// test with user config
	if err := os.MkdirAll(filepath.Dir(UserConfig()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := conf.Save(UserConfig()); err != nil {
		t.Fatal(err)
	}
	if c := LoadUserSystemConfig(); c == nil {
		t.Errorf("got nil, want %v", conf)
	}
}
