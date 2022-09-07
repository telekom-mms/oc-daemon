package cpd

import (
	"log"
	"reflect"
	"testing"
)

// testCPDStartStop tests Start and Stop of CPD
func TestCPDStartStop(t *testing.T) {
	c := NewCPD()
	c.Start()
	c.Stop()
}

// TestCPDHosts tests Hosts of CPD
func TestCPDHosts(t *testing.T) {
	c := NewCPD()
	want := []string{host}
	got := c.Hosts()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestCPDProbe tests Probe of CPD
func TestCPDProbe(t *testing.T) {
	c := NewCPD()
	c.Start()
	c.Probe()
	log.Println(<-c.Results())
	c.Stop()
}

// TestCPDResults tests Results of CPD
func TestCPDResults(t *testing.T) {
	c := NewCPD()
	want := c.reports
	got := c.Results()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewCPD tests NewCPD
func TestNewCPD(t *testing.T) {
	c := NewCPD()
	if c.reports == nil ||
		c.probes == nil ||
		c.done == nil {

		t.Errorf("got nil, want != nil")
	}
}
