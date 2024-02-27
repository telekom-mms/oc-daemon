package devmon

import (
	"net"
	"testing"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// TestDevMonStartStop tests Start and Stop of DevMon
func TestDevMonStartStop(t *testing.T) {
	// test without LinkUpdates
	RegisterLinkUpdates = func(d *DevMon) (chan netlink.LinkUpdate, error) {
		return nil, nil
	}

	devMon := NewDevMon()
	if err := devMon.Start(); err != nil {
		t.Error(err)
	}
	devMon.Stop()

	// test with LinkUpdates
	for i, test := range []struct {
		update netlink.LinkUpdate
		want   *Update
	}{
		{
			// invalid link event
			update: netlink.LinkUpdate{
				Header: unix.NlMsghdr{Type: unix.RTM_NEWADDR},
				Link:   &netlink.Device{},
			},
			want: nil,
		},
		{
			// new link event
			update: netlink.LinkUpdate{
				Header: unix.NlMsghdr{Type: unix.RTM_NEWLINK},
				Link:   &netlink.Device{},
			},
			want: &Update{Add: true, Type: "device"},
		},
		{
			// del link event
			update: netlink.LinkUpdate{
				Header: unix.NlMsghdr{Type: unix.RTM_DELLINK},
				Link:   &netlink.Device{},
			},
			want: &Update{Add: false, Type: "device"},
		},
		{
			// loopback device
			update: netlink.LinkUpdate{
				Header: unix.NlMsghdr{Type: unix.RTM_NEWLINK},
				Link: &netlink.Device{
					LinkAttrs: netlink.LinkAttrs{Flags: net.FlagLoopback},
				},
			},
			want: &Update{Add: true, Type: "loopback"},
		},
		{
			// device that is actually virtual
			update: netlink.LinkUpdate{
				Header: unix.NlMsghdr{Type: unix.RTM_NEWLINK},
				Link: &netlink.Device{
					LinkAttrs: netlink.LinkAttrs{Name: "lo"},
				},
			},
			want: &Update{Add: true, Type: "virtual"},
		},
		{
			// device with invalid symlink
			update: netlink.LinkUpdate{
				Header: unix.NlMsghdr{Type: unix.RTM_NEWLINK},
				Link: &netlink.Device{
					LinkAttrs: netlink.LinkAttrs{Name: "device-does-not-exist"},
				},
			},
			want: &Update{Add: true, Type: "device"},
		},
	} {
		// send test update in goroutine spawned in RegisterLinkUpdates
		// and signal sending complete
		sendDone := make(chan struct{})
		RegisterLinkUpdates = func(d *DevMon) (chan netlink.LinkUpdate, error) {
			updates := make(chan netlink.LinkUpdate)
			go func(up netlink.LinkUpdate) {
				defer close(sendDone)
				updates <- up
			}(test.update)
			return updates, nil
		}

		// start monitor, wait for result/sending complete and check
		// result, stop monitor
		devMon := NewDevMon()
		if err := devMon.Start(); err != nil {
			t.Error(err)
		}
		if test.want != nil {
			up := <-devMon.Updates()
			if up.Add != test.want.Add || up.Type != test.want.Type {
				t.Errorf("test %d, got %v,  want %v", i, up, test.want)
			}
		}
		<-sendDone
		devMon.Stop()
	}

	// test event after stop
	sendDone := make(chan struct{})
	RegisterLinkUpdates = func(d *DevMon) (chan netlink.LinkUpdate, error) {
		updates := make(chan netlink.LinkUpdate)
		go func() {
			defer close(sendDone)

			// wait for monitor closed
			<-d.closed

			// send update
			up := netlink.LinkUpdate{}
			up.Header.Type = unix.RTM_NEWLINK
			up.Link = &netlink.Device{}
			updates <- up
		}()
		return updates, nil
	}

	devMon = NewDevMon()
	if err := devMon.Start(); err != nil {
		t.Error(err)
	}
	devMon.Stop()
	<-sendDone

	// test with unexpected close and LinkUpdates
	runOnce := false
	RegisterLinkUpdates = func(d *DevMon) (chan netlink.LinkUpdate, error) {
		updates := make(chan netlink.LinkUpdate)
		if !runOnce {
			// on first run, close updates
			runOnce = true
			close(updates)
		} else {
			// on subsequent run, send update
			go func() {
				up := netlink.LinkUpdate{}
				up.Header.Type = unix.RTM_NEWLINK
				up.Link = &netlink.Device{}
				updates <- up
			}()
		}
		return updates, nil
	}

	devMon = NewDevMon()
	if err := devMon.Start(); err != nil {
		t.Error(err)
	}
	up := <-devMon.Updates()
	if !up.Add {
		t.Errorf("add should be true")
	}
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
