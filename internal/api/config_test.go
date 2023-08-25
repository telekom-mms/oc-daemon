package api

import "testing"

// TestConfigValid tests Valid of Config
func TestConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Config{
		nil,
		{},
		{SocketFile: "test.sock", SocketPermissions: "invalid"},
		{SocketFile: "test.sock", SocketPermissions: "1234"},
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
		{SocketFile: "test.sock", SocketPermissions: "777"},
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
	sc := NewConfig()
	if !sc.Valid() {
		t.Errorf("config is not valid")
	}
}
