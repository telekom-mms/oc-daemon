package devmon

import (
	"log"
	"testing"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// TestDevMonStartStop tests Start and Stop of DevMon
func TestDevMonStartStop(t *testing.T) {
	devMon := NewDevMon()

	// test without LinkUpdates
	RegisterLinkUpdates = func(d *DevMon) chan netlink.LinkUpdate {
		return nil
	}
	devMon.Start()
	devMon.Stop()

	// helper function for LinkUpdates
	linkUpdates := func(updates chan netlink.LinkUpdate, done chan struct{}) {
		for {
			up := netlink.LinkUpdate{}
			up.Header.Type = unix.RTM_NEWLINK
			up.Link = &netlink.Device{}
			select {
			case updates <- up:
			case <-done:
				return

			}
		}
	}

	// test with LinkUpdates
	devMon = NewDevMon()
	RegisterLinkUpdates = func(d *DevMon) chan netlink.LinkUpdate {
		updates := make(chan netlink.LinkUpdate)
		go linkUpdates(updates, d.upsDone)
		return updates
	}
	devMon.Start()
	for i := 0; i < 3; i++ {
		log.Println(<-devMon.Updates())
	}
	devMon.Stop()

	// test with unexpected close and LinkUpdates
	devMon = NewDevMon()
	runOnce := false
	RegisterLinkUpdates = func(d *DevMon) chan netlink.LinkUpdate {
		updates := make(chan netlink.LinkUpdate)
		if !runOnce {
			runOnce = true
			close(updates)
		} else {
			go linkUpdates(updates, d.upsDone)
		}
		return updates
	}
	devMon.Start()
	log.Println(<-devMon.Updates())
	devMon.Stop()
}

// TestDevMonUpdates tests Updates of DevMon
func TestDevMonUpdates(t *testing.T) {
	devMon := NewDevMon()
	got := devMon.Updates()
	want := devMon.updates
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewDevMon tests NewDevMon
func TestNewDevMon(t *testing.T) {
	devMon := NewDevMon()
	if devMon.updates == nil ||
		devMon.upsDone == nil ||
		devMon.done == nil {

		t.Errorf("got nil, want != nil")
	}
}
