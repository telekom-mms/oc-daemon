package devmon

import (
	"log"
	"net"
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

	// test with del link event
	devMon = NewDevMon()
	linkUpdates = func(updates chan netlink.LinkUpdate, done chan struct{}) {
		up := netlink.LinkUpdate{}
		up.Header.Type = unix.RTM_DELLINK
		up.Link = &netlink.Device{}
		updates <- up
	}
	devMon.Start()
	up := <-devMon.updates
	if up.Add {
		t.Errorf("add should be false")
	}
	devMon.Stop()

	// test with invalid event
	devMon = NewDevMon()
	linkUpdates = func(updates chan netlink.LinkUpdate, done chan struct{}) {
		up := netlink.LinkUpdate{}
		up.Header.Type = unix.RTM_NEWADDR
		up.Link = &netlink.Device{}
		updates <- up
		up = netlink.LinkUpdate{}
		up.Header.Type = unix.RTM_DELLINK
		up.Link = &netlink.Device{}
		updates <- up
	}
	devMon.Start()
	up = <-devMon.updates
	if up.Add {
		t.Errorf("add should be false")
	}
	devMon.Stop()

	// test loopback
	devMon = NewDevMon()
	linkUpdates = func(updates chan netlink.LinkUpdate, done chan struct{}) {
		up := netlink.LinkUpdate{}
		up.Header.Type = unix.RTM_NEWLINK
		up.Link = &netlink.Device{LinkAttrs: netlink.LinkAttrs{Flags: net.FlagLoopback}}
		updates <- up
	}
	devMon.Start()
	up = <-devMon.updates
	if up.Type != "loopback" {
		t.Errorf("type should be loopback")
	}
	devMon.Stop()

	// test device that is actually virtual
	devMon = NewDevMon()
	linkUpdates = func(updates chan netlink.LinkUpdate, done chan struct{}) {
		up := netlink.LinkUpdate{}
		up.Header.Type = unix.RTM_NEWLINK
		up.Link = &netlink.Device{LinkAttrs: netlink.LinkAttrs{Name: "lo"}}
		updates <- up
	}
	devMon.Start()
	up = <-devMon.updates
	if up.Type != "virtual" {
		t.Errorf("type should be virtual")
	}
	devMon.Stop()

	// test device with invalid symlink
	devMon = NewDevMon()
	linkUpdates = func(updates chan netlink.LinkUpdate, done chan struct{}) {
		up := netlink.LinkUpdate{}
		up.Header.Type = unix.RTM_NEWLINK
		up.Link = &netlink.Device{LinkAttrs: netlink.LinkAttrs{Name: "device-does-not-exist"}}
		updates <- up
	}
	devMon.Start()
	up = <-devMon.updates
	if up.Type != "device" {
		t.Errorf("type should be device")
	}
	devMon.Stop()
	// test stop during event handling
	devMon = NewDevMon()
	linkUpdates = func(updates chan netlink.LinkUpdate, done chan struct{}) {
		up := netlink.LinkUpdate{}
		up.Header.Type = unix.RTM_DELLINK
		up.Link = &netlink.Device{}
		updates <- up
	}
	devMon.Start()
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
		devMon.done == nil ||
		devMon.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
