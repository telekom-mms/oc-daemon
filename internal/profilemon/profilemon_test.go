package profilemon

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
)

// createProfileMonTestFile creates a temporary file for ProfileMon testing
func createProfileMonTestFile() string {
	f, err := os.CreateTemp("", "profilemon-test")
	if err != nil {
		log.Fatal(err)
	}
	return f.Name()
}

// TestProfileMonHandleEvent tests handleEvent of ProfileMon
func TestProfileMonHandleEvent(t *testing.T) {
	f := createProfileMonTestFile()
	defer func() { _ = os.Remove(f) }()

	p := NewProfileMon(f)

	// test with unitialized hash, should update hash and send update
	h := p.hash
	go p.handleEvent()
	<-p.updates
	if bytes.Equal(h[:], p.hash[:]) {
		t.Errorf("got %v, want other", h)
	}

	// test with same file content, hash should stay the same, no update
	h = p.hash
	p.handleEvent()
	if !bytes.Equal(h[:], p.hash[:]) {
		t.Errorf("got %v, want %v", p.hash, h)
	}

	// test with file error
	d := t.TempDir()
	p = NewProfileMon(filepath.Join(d, "does-not-exist"))
	p.handleEvent()

}

// TestProfileMonStartEvents tests start of ProfileMon, events.
func TestProfileMonStartEvents(t *testing.T) {
	f := createProfileMonTestFile()
	defer func() { _ = os.Remove(f) }()

	p := NewProfileMon(f)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Close() }()
	p.watcher = watcher

	go p.start()

	p.watcher.Events <- fsnotify.Event{}
	p.watcher.Events <- fsnotify.Event{Name: f}
	<-p.Updates()

	p.watcher.Errors <- errors.New("test error")

	if err := watcher.Close(); err != nil {
		t.Error(err)
	}
	<-p.closed
}

// TestProfileMonStartStop tests Start and Stop of ProfileMon
func TestProfileMonStartStop(_ *testing.T) {
	f := createProfileMonTestFile()
	defer func() { _ = os.Remove(f) }()

	p := NewProfileMon(f)
	p.Start()
	p.Stop()
}

// TestProfileMonUpdates tests Updates of ProfileMon.
func TestProfileMonUpdates(t *testing.T) {
	p := NewProfileMon("")
	want := p.updates
	got := p.Updates()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewProfileMon tests NewProfileMon
func TestNewProfileMon(t *testing.T) {
	f := "some file"
	p := NewProfileMon(f)
	if p.file != f {
		t.Errorf("got %s, want %s", p.file, f)
	}
	if p.updates == nil ||
		p.done == nil ||
		p.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
