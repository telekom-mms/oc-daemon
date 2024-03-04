// Package devmon contains the device monitor.
package devmon

import (
	"fmt"
	"net"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// Update is a device update
type Update struct {
	Add    bool
	Device string
	Type   string
	Index  int
}

// DevMon is a device monitor
type DevMon struct {
	events  chan netlink.LinkUpdate
	updates chan *Update
	upsDone chan struct{}
	done    chan struct{}
	closed  chan struct{}
}

// sendUpdate sends update over the update channel
func (d *DevMon) sendUpdate(update *Update) {
	// send update or abort if we are shutting down
	select {
	case d.updates <- update:
	case <-d.done:
	}
}

// handleLink handles a link update
func (d *DevMon) handleLink(add bool, link netlink.Link) {
	log.WithFields(log.Fields{
		"add":  add,
		"link": link,
	}).Debug("DevMon handling link update")

	// get attributes and link type
	attrs := link.Attrs()
	typ := link.Type()

	// use special type for loop back device
	if attrs.Flags&net.FlagLoopback != 0 {
		typ = "loopback"
	}

	// use special type for device that is actually virtual, e.g., vboxnet
	if typ == "device" {
		sysfs := filepath.Join("/sys/class/net", attrs.Name)
		path, err := filepath.EvalSymlinks(sysfs)
		if err != nil {
			log.WithError(err).Error("DevMon could not eval device symlink")
		} else {
			if path == filepath.Join("/sys/devices/virtual/net",
				attrs.Name) {
				// set device type to virtual
				typ = "virtual"
			}
		}
	}

	// report device update
	update := &Update{
		Add:    add,
		Device: attrs.Name,
		Type:   typ,
		Index:  attrs.Index,
	}
	d.sendUpdate(update)
}

// RegisterLinkUpdates registers for link update events
var RegisterLinkUpdates = func(d *DevMon) (chan netlink.LinkUpdate, error) {
	// register for link update events
	events := make(chan netlink.LinkUpdate)
	options := netlink.LinkSubscribeOptions{
		ListExisting: true,
	}
	if err := netlink.LinkSubscribeWithOptions(events, d.upsDone, options); err != nil {
		return nil, fmt.Errorf("could not subscribe to link updates: %w", err)
	}

	return events, nil
}

// start starts the device monitor
func (d *DevMon) start() {
	defer close(d.closed)
	defer close(d.updates)
	defer close(d.upsDone)

	// handle link update events
	for {
		select {
		case e, ok := <-d.events:
			if !ok {
				// unexpected close of events, try to re-open
				log.Error("DevMon got unexpected close of link events")
				events, err := RegisterLinkUpdates(d)
				if err != nil {
					log.WithError(err).Error("DevMon register link update error")
				}
				d.events = events
				break
			}
			switch e.Header.Type {
			case unix.RTM_NEWLINK:
				d.handleLink(true, e)
			case unix.RTM_DELLINK:
				d.handleLink(false, e)
			default:
				log.WithField("event", e).Error("DevMon got unknown link event")
			}

		case <-d.done:
			// drain events and wait for channel shutdown; this
			// could take until the next link update
			go func() {
				for range d.events {
					// wait for channel shutdown
				}
			}()
			return
		}
	}
}

// Start starts the device monitor
func (d *DevMon) Start() error {
	// register for link update events
	events, err := RegisterLinkUpdates(d)
	if err != nil {
		return err
	}
	d.events = events

	go d.start()
	return nil
}

// Stop stops the device monitor
func (d *DevMon) Stop() {
	close(d.done)
	<-d.closed
}

// Updates returns the Update channel for device updates
func (d *DevMon) Updates() chan *Update {
	return d.updates
}

// NewDevMon returns a new device monitor
func NewDevMon() *DevMon {
	return &DevMon{
		updates: make(chan *Update),
		upsDone: make(chan struct{}),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
}
