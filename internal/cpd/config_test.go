package cpd

import (
	"testing"
	"time"
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
		{
			Host:               "some.host.example.com",
			HTTPTimeout:        3 * time.Second,
			ProbeCount:         5,
			ProbeTimer:         150 * time.Second,
			ProbeTimerDetected: 10 * time.Second,
		},
	} {
		if !valid.Valid() {
			t.Errorf("config should be valid: %v", valid)
		}
	}
}

// TestNewConfig tests NewConfig
func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if !c.Valid() {
		t.Errorf("new config should be valid")
	}
}
