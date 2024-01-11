package dnsmon

import (
	"errors"
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestDNSMonStart(t *testing.T) {
	dnsMon := NewDNSMon(NewConfig())
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	dnsMon.watcher = watcher

	// test valid file events
	go dnsMon.start()
	<-dnsMon.Updates()
	for _, name := range []string{
		dnsMon.config.ETCResolvConf,
		dnsMon.config.StubResolvConf,
		dnsMon.config.SystemdResolvConf,
	} {
		dnsMon.watcher.Events <- fsnotify.Event{Name: name}
		<-dnsMon.Updates()
	}

	// test invalid file event
	dnsMon.watcher.Events <- fsnotify.Event{Name: "something else"}

	// test error event
	dnsMon.watcher.Errors <- errors.New("test error")

	// test unexpected close
	watcher.Close()
	<-dnsMon.closed
}

// TestDNSMonStartStop tests Start and Stop of DNSMon
func TestDNSMonStartStop(t *testing.T) {
	dnsMon := NewDNSMon(NewConfig())
	dnsMon.Start()
	dnsMon.Stop()
}

// TestDNSMonUpdates tests Updates of DNSMon
func TestDNSMonUpdates(t *testing.T) {
	dnsMon := NewDNSMon(NewConfig())
	got := dnsMon.Updates()
	want := dnsMon.updates
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewDNSMon tests NewDNSMon
func TestNewDNSMon(t *testing.T) {
	config := NewConfig()
	dnsMon := NewDNSMon(config)
	if dnsMon.config != config {
		t.Errorf("got %v, want %v", dnsMon.config, config)
	}
	if dnsMon.updates == nil ||
		dnsMon.done == nil ||
		dnsMon.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
