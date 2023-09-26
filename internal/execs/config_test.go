package execs

import (
	"reflect"
	"testing"
)

// TestConfigValid tests Valid of Config
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

// TestNewConfig tests NewConfig
func TestNewConfig(t *testing.T) {
	want := &Config{IP, Nft, Resolvectl, Sysctl}
	got := NewConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
