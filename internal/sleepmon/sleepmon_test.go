package sleepmon

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

// TestSleepMonHandleSignal tests handleSignal of SleepMon.
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

// testRWC is a reader writer closer for testing.
type testRWC struct{}

func (t *testRWC) Read([]byte) (int, error)  { return 0, nil }
func (t *testRWC) Write([]byte) (int, error) { return 0, nil }
func (t *testRWC) Close() error              { return nil }

// TestSleepMonStartEvents tests start of SleepMon, events.
func TestSleepMonStartEvents(t *testing.T) {
	s := NewSleepMon()
	conn, err := dbus.NewConn(&testRWC{})
	if err != nil {
		t.Fatal(err)
	}
	s.conn = conn
	go s.start()

	// signal
	s.sigs <- &dbus.Signal{}

	// unexpected close
	close(s.sigs)
	<-s.closed
}

// TestSleepMonStartErrors tests Start of SleepMon, errors.
func TestSleepMonStartErrors(t *testing.T) {
	// match signal error
	oldAddMatchSignal := connAddMatchSignal
	connAddMatchSignal = func(conn *dbus.Conn, options ...dbus.MatchOption) error {
		return errors.New("test error")
	}
	defer func() { connAddMatchSignal = oldAddMatchSignal }()

	s := NewSleepMon()
	if err := s.Start(); err == nil {
		t.Error("Start should return error")
	}

	// system bus error
	dbusConnectSystemBus = func(opts ...dbus.ConnOption) (*dbus.Conn, error) {
		return nil, errors.New("test error")
	}
	defer func() { dbusConnectSystemBus = dbus.ConnectSystemBus }()

	s = NewSleepMon()
	if err := s.Start(); err == nil {
		t.Error("Start should return error")
	}
}

// TestSleepMonStartStop tests Start and Stop of SleepMon.
func TestSleepMonStartStop(t *testing.T) {
	s := NewSleepMon()
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	s.Stop()
}

// TestSleepMonEvents tests Events of SleepMon.
func TestSleepMonEvents(t *testing.T) {
	s := NewSleepMon()
	got := s.Events()
	want := s.events
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewSleepMon tests NewSleepMon.
func TestNewSleepMon(t *testing.T) {
	s := NewSleepMon()
	if s.sigs == nil ||
		s.events == nil ||
		s.done == nil ||
		s.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
