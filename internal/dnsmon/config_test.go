package dnsmon

import (
	"reflect"
	"testing"
)

// TestConfigResolvConfDirs tests resolvConfDirs of Config
func TestConfigResolvConfDirs(t *testing.T) {
	config := &Config{
		ETCResolvConf:     "/test/etc/resolv.conf",
		StubResolvConf:    "/test/etc/stub-resolv.conf",
		SystemdResolvConf: "/test/etc/systemd-resolv.conf",
	}
	want := []string{"/test/etc"}
	got := config.resolvConfDirs()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNewConfig tests NewConfig
func TestNewConfig(t *testing.T) {
	want := &Config{
		ETCResolvConf:     ETCResolvConf,
		StubResolvConf:    StubResolvConf,
		SystemdResolvConf: SystemdResolvConf,
	}
	got := NewConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
