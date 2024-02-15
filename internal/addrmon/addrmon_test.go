package addrmon

import (
	"log"
	"testing"

	"github.com/vishvananda/netlink"
)

// TestAddrMonStartStop tests Start and Stop of AddrMon
func TestAddrMonStartStop(t *testing.T) {
	// clean up after tests
	oldRegisterAddrUpdates := RegisterAddrUpdates
	defer func() {
		netlinkAddrSubscribeWithOptions = netlink.AddrSubscribeWithOptions
		RegisterAddrUpdates = oldRegisterAddrUpdates
	}()

	// test RegisterAddrUpdates without netlink error
	addrMon := NewAddrMon()

	netlinkAddrSubscribeWithOptions = func(ch chan<- netlink.AddrUpdate,
		done <-chan struct{}, options netlink.AddrSubscribeOptions) error {
		return nil
	}

	if err := addrMon.Start(); err != nil {
		t.Error(err)
	}
	addrMon.Stop()

	// test without AddrUpdates
	addrMon = NewAddrMon()

	RegisterAddrUpdates = func(a *AddrMon) (chan netlink.AddrUpdate, error) {
		return nil, nil
	}

	if err := addrMon.Start(); err != nil {
		t.Error(err)
	}
	addrMon.Stop()

	// helper function for AddrUpdates
	addrUpdates := func(updates chan netlink.AddrUpdate, done chan struct{}) {
		for {
			up := netlink.AddrUpdate{}
			select {
			case updates <- up:
			case <-done:
				return

			}
		}
	}

	// test with AddrUpdates
	addrMon = NewAddrMon()

	RegisterAddrUpdates = func(a *AddrMon) (chan netlink.AddrUpdate, error) {
		updates := make(chan netlink.AddrUpdate)
		go addrUpdates(updates, a.upsDone)
		return updates, nil
	}

	if err := addrMon.Start(); err != nil {
		t.Error(err)
	}
	for i := 0; i < 3; i++ {
		log.Println(<-addrMon.Updates())
	}
	addrMon.Stop()

	// test with unexpected close and AddrUpdates
	addrMon = NewAddrMon()
	runOnce := false

	RegisterAddrUpdates = func(a *AddrMon) (chan netlink.AddrUpdate, error) {
		updates := make(chan netlink.AddrUpdate)
		if !runOnce {
			runOnce = true
			close(updates)
		} else {
			go addrUpdates(updates, a.upsDone)
		}
		return updates, nil
	}

	if err := addrMon.Start(); err != nil {
		t.Error(err)
	}
	log.Println(<-addrMon.Updates())
	addrMon.Stop()
}

// TestAddrMonUpdates tests Updates of AddrMon
func TestAddrMonUpdates(t *testing.T) {
	addrMon := NewAddrMon()
	got := addrMon.Updates()
	want := addrMon.updates
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewAddrMon tests NewAddrMon
func TestNewAddrMon(t *testing.T) {
	addrMon := NewAddrMon()
	if addrMon.updates == nil ||
		addrMon.upsDone == nil ||
		addrMon.done == nil ||
		addrMon.closed == nil {

		t.Errorf("got nil, want != nil")
	}
}
