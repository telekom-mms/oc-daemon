// Package sleepmon contains the sleep monitor.
package sleepmon

import (
	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
)

const (
	// object path, destination, interface, signals, methods, properties.
	path            = "/org/freedesktop/login1"
	dest            = "org.freedesktop.login1"
	iface           = dest + ".Manager"
	prepareForSleep = iface + ".PrepareForSleep"
)

// SleepMon is a suspend/hibernate monitor.
type SleepMon struct {
	conn   *dbus.Conn
	sigs   chan *dbus.Signal
	events chan bool
	done   chan struct{}
	closed chan struct{}
}

// sendEvent sends sleep over the event channel.
func (s *SleepMon) sendEvent(sleep bool) {
	select {
	case s.events <- sleep:
	case <-s.done:
	}
}

// handleSignal handles signal.
func (s *SleepMon) handleSignal(signal *dbus.Signal) {
	log.WithField("signal", signal).Debug("SleepMon got signal")
	switch signal.Name {
	case prepareForSleep:
		// handle prepare for sleep signal,
		if len(signal.Body) < 1 {
			log.Error("SleepMon got invalid prepare for sleep signal")
			return
		}

		// is it sleep or resume?
		sleep, ok := signal.Body[0].(bool)
		if !ok {
			log.Error("SleepMon could not parse prepare for sleep signal")
			return
		}
		log.WithField("sleep", sleep).Debug("SleepMon got prepare for sleep signal")

		// send event
		s.sendEvent(sleep)
	}

}

// start starts the sleep monitor.
func (s *SleepMon) start() {
	defer close(s.closed)
	defer close(s.events)
	defer func() {
		_ = s.conn.Close()
	}()

	// handle login signals
	for {
		select {
		case sig, ok := <-s.sigs:
			if !ok {
				log.Error("SleepMon got unexpected close of signals channel")
				return
			}
			s.handleSignal(sig)
		case <-s.done:
			return
		}
	}
}

// dbusConnectSystemBus is dbus.ConnectSystemBus for testing.
var dbusConnectSystemBus = dbus.ConnectSystemBus

// connAddMatchSignal is dbus conn.AddMatchSignal for testing.
var connAddMatchSignal = func(conn *dbus.Conn, options ...dbus.MatchOption) error {
	return conn.AddMatchSignal(options...)
}

// Start starts the sleep monitor.
func (s *SleepMon) Start() error {
	// connect to system bus
	conn, err := dbusConnectSystemBus()
	if err != nil {
		log.WithError(err).Error("SleepMon could not connect to system bus")
		return err
	}
	s.conn = conn

	// subscribe to login signals
	if err = connAddMatchSignal(conn,
		dbus.WithMatchObjectPath(path),
		dbus.WithMatchInterface(iface),
	); err != nil {
		log.WithError(err).Error("SleepMon could not subscribe to login signals")
		_ = s.conn.Close()
		return err
	}

	// set channel for signals
	conn.Signal(s.sigs)

	go s.start()
	return nil
}

// Stop stops the sleep monitor.
func (s *SleepMon) Stop() {
	close(s.done)
	<-s.closed
}

// Events returns the sleep event channel.
func (s *SleepMon) Events() chan bool {
	return s.events
}

// NewSleepMon returns a new sleep monitor.
func NewSleepMon() *SleepMon {
	return &SleepMon{
		sigs:   make(chan *dbus.Signal, 10),
		events: make(chan bool),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}
