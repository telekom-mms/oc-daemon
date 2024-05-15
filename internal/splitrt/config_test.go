package splitrt

import "testing"

// TestConfigValid tests Valid of Config.
func TestConfigValid(t *testing.T) {
	// test invalid
	for _, invalid := range []*Config{
		nil,
		{},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "0",
			RulePriority2: "1",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "32766",
			RulePriority2: "32767",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "2111",
			RulePriority2: "2111",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "2112",
			RulePriority2: "2111",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "65537",
			RulePriority2: "2111",
		},
		{
			RoutingTable:  "42111",
			FirewallMark:  "42111",
			RulePriority1: "2111",
			RulePriority2: "65537",
		},
		{
			RoutingTable:  "0",
			FirewallMark:  "42112",
			RulePriority1: "2222",
			RulePriority2: "2223",
		},
		{
			RoutingTable:  "4294967295",
			FirewallMark:  "42112",
			RulePriority1: "2222",
			RulePriority2: "2223",
		},
		{
			RoutingTable:  "42112",
			FirewallMark:  "4294967296",
			RulePriority1: "2222",
			RulePriority2: "2223",
		},
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
		{
			RoutingTable:  "42112",
			FirewallMark:  "42112",
			RulePriority1: "2222",
			RulePriority2: "2223",
		},
	} {
		want := true
		got := valid.Valid()

		if got != want {
			t.Errorf("got %t, want %t for %v", got, want, valid)
		}
	}
}

// TestNewConfig tests NewConfig.
func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if !c.Valid() {
		t.Errorf("new config should be valid")
	}
}
