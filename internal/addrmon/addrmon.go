package addrmon

import (
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// Update is an address update
type Update struct {
	Add     bool
	Address net.IPNet
	Index   int
}

// AddrMon is an address monitor
type AddrMon struct {
	updates chan *Update
	upsDone chan struct{}
	done    chan struct{}
}

// sendUpdate sends an address update
func (a *AddrMon) sendUpdate(update *Update) {
	select {
	case a.updates <- update:
	case <-a.done:
	}
}

// registerAddrUpdates registers for addr update events
func (a *AddrMon) registerAddrUpdates() chan netlink.AddrUpdate {
	// register for addr update events
	events := make(chan netlink.AddrUpdate)
	options := netlink.AddrSubscribeOptions{
		ListExisting: true,
	}
	if err := netlink.AddrSubscribeWithOptions(events, a.upsDone, options); err != nil {
		log.WithError(err).Fatal("AddrMon address subscribe error")
	}

	return events
}

// start starts the address monitor
func (a *AddrMon) start() {
	defer close(a.updates)
	defer close(a.upsDone)

	// register for addr update events
	events := a.registerAddrUpdates()

	// handle events
	for {
		select {
		case e, ok := <-events:
			if !ok {
				// unexpected close of events, try to re-open
				log.Error("AddrMon got unexpected close of addr events")
				events = a.registerAddrUpdates()
				break
			}

			// forward event as address update
			u := &Update{
				Address: e.LinkAddress,
				Index:   e.LinkIndex,
				Add:     e.NewAddr,
			}
			a.sendUpdate(u)

		case <-a.done:
			// drain events and wait for channel shutdown; this
			// could take until the next addr update
			go func() {
				for range events {
					// wait for channel shutdown
				}
			}()

			// stop address monitor
			return
		}
	}
}

// Start starts the address monitor
func (a *AddrMon) Start() {
	go a.start()
}

// Stop stops the address monitor
func (a *AddrMon) Stop() {
	close(a.done)
	for range a.updates {
		// wait for channel close
	}
}

// Updates returns the address updates channel
func (a *AddrMon) Updates() chan *Update {
	return a.updates
}

// NewAddrMon returns a new address monitor
func NewAddrMon() *AddrMon {
	return &AddrMon{
		updates: make(chan *Update),
		upsDone: make(chan struct{}),
		done:    make(chan struct{}),
	}
}
