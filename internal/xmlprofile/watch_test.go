package xmlprofile

import (
	"bytes"
	"log"
	"os"
	"testing"
)

// createWatchTestFile creates a temporary file for Watch testing
func createWatchTestFile() string {
	f, err := os.CreateTemp("", "watch-test")
	if err != nil {
		log.Fatal(err)
	}
	return f.Name()
}

// TestWatchHandleEvent tests handleEvent of Watch
func TestWatchHandleEvent(t *testing.T) {
	f := createWatchTestFile()
	defer os.Remove(f)

	w := NewWatch(f)

	// test with unitialized hash, should update hash and send update
	h := w.hash
	go w.handleEvent()
	<-w.updates
	if bytes.Equal(h[:], w.hash[:]) {
		t.Errorf("got %v, want other", h)
	}

	// test with same file content, hash should stay the same, no update
	h = w.hash
	w.handleEvent()
	if !bytes.Equal(h[:], w.hash[:]) {
		t.Errorf("got %v, want %v", w.hash, h)
	}
}

// TestWatchStartStop tests Start and Stop of Watch
func TestWatchStartStop(t *testing.T) {
	f := createWatchTestFile()
	defer os.Remove(f)

	w := NewWatch(f)
	w.Start()
	w.Stop()
}

// TestNewWatch tests NewWatch
func TestNewWatch(t *testing.T) {
	f := "some file"
	w := NewWatch(f)
	if w.file != f {
		t.Errorf("got %s, want %s", w.file, f)
	}
	if w.updates == nil ||
		w.done == nil {

		t.Errorf("got nil, want != nil")
	}
}
