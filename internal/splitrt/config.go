package splitrt

import "strconv"

var (
	// RoutingTable is the routing table.
	RoutingTable = "42111"

	// RulePriority1 is the first routing rule priority. It must be unique,
	// higher than the local rule, lower than the main and default rules,
	// lower than the second routing rule priority.
	RulePriority1 = "2111"

	// RulePriority2 is the second routing rule priority. It must be unique,
	// higher than the local rule, lower than the main and default rules,
	// higher than the first routing rule priority.
	RulePriority2 = "2112"

	// FirewallMark is the firewall mark used for split routing.
	FirewallMark = RoutingTable
)

// Config is a split routing configuration.
type Config struct {
	RoutingTable  string
	RulePriority1 string
	RulePriority2 string
	FirewallMark  string
}

// Valid returns whether the split routing configuration is valid.
func (c *Config) Valid() bool {
	if c == nil ||
		c.RoutingTable == "" ||
		c.RulePriority1 == "" ||
		c.RulePriority2 == "" ||
		c.FirewallMark == "" {

		return false
	}

	// check routing table value: must be > 0, < 0xFFFFFFFF
	rtTable, err := strconv.ParseUint(c.RoutingTable, 10, 32)
	if err != nil || rtTable == 0 || rtTable >= 0xFFFFFFFF {
		return false
	}

	// check rule priority values: must be > 0, < 32766, prio1 < prio2
	prio1, err := strconv.ParseUint(c.RulePriority1, 10, 16)
	if err != nil {
		return false
	}
	prio2, err := strconv.ParseUint(c.RulePriority2, 10, 16)
	if err != nil {
		return false
	}
	if prio1 == 0 || prio2 == 0 ||
		prio1 >= 32766 || prio2 >= 32766 ||
		prio1 >= prio2 {

		return false
	}

	// check fwmark value: must be 32 bit unsigned int
	if _, err := strconv.ParseUint(c.FirewallMark, 10, 32); err != nil {
		return false
	}

	return true
}

// NewConfig returns a new split routing configuration.
func NewConfig() *Config {
	return &Config{
		RoutingTable:  RoutingTable,
		RulePriority1: RulePriority1,
		RulePriority2: RulePriority2,
		FirewallMark:  FirewallMark,
	}
}
