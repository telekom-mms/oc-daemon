package dnsproxy

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
		want := false
		got := invalid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, invalid)
		}
	}

	// test valid
	for _, valid := range []*Config{
		NewConfig(),
		{Address: "127.0.0.1:4253", ListenUDP: true},
		{Address: "127.0.0.1:4253", ListenTCP: true},
	} {
		want := true
		got := valid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, valid)
		}
	}
}

// TestNewConfig tests NewConfig
func TestNewConfig(t *testing.T) {
	want := &Config{
		Address:   Address,
		ListenUDP: true,
		ListenTCP: true,
	}
	got := NewConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
