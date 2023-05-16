package profilemon

import (
	"bytes"
	"log"
	"os"
	"testing"
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
	defer os.Remove(f)

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
}

// TestProfileMonStartStop tests Start and Stop of ProfileMon
func TestProfileMonStartStop(t *testing.T) {
	f := createProfileMonTestFile()
	defer os.Remove(f)

	p := NewProfileMon(f)
	p.Start()
	p.Stop()
}

// TestNewProfileMon tests NewProfileMon
func TestNewProfileMon(t *testing.T) {
	f := "some file"
	p := NewProfileMon(f)
	if p.file != f {
		t.Errorf("got %s, want %s", p.file, f)
	}
	if p.updates == nil ||
		p.done == nil {

		t.Errorf("got nil, want != nil")
	}
}
