package sleepmon

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

// TestSleepMonHandleSignal tests handleSignal of SleepMon
func TestSleepMonHandleSignal(t *testing.T) {
	s := NewSleepMon()

	// test invalid signals, should not block
	for _, signal := range []*dbus.Signal{
		{},
		{Name: prepareForSleep},
		{Name: prepareForSleep, Body: make([]interface{}, 1)},
	} {
		s.handleSignal(signal)
	}

	// test valid signals
	for _, want := range []bool{true, false} {
		signal := &dbus.Signal{
			Name: prepareForSleep,
			Body: append(make([]interface{}, 0), want),
		}
		go s.handleSignal(signal)
		got := <-s.Events()
		if got != want {
			t.Errorf("got %t, want %t", got, want)
		}
	}
}

// TestSleepMonStartStop tests Start and Stop of SleepMon
func TestSleepMonStartStop(t *testing.T) {
	s := NewSleepMon()
	s.Start()
	s.Stop()
}

// TestSleepMonEvents tests Events of SleepMon
func TestSleepMonEvents(t *testing.T) {
	s := NewSleepMon()
	got := s.Events()
	want := s.events
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewSleepMon tests NewSleepMon
func TestNewSleepMon(t *testing.T) {
	s := NewSleepMon()
	if s.events == nil ||
		s.done == nil {

		t.Errorf("got nil, want != nil")
	}
}
