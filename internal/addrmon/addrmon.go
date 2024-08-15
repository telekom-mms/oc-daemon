// Package addrmon contains the address monitor.
package addrmon

import (
	"fmt"
	"net/netip"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// Update is an address update.
type Update struct {
	Add     bool
	Address netip.Prefix
	Index   int
}

// AddrMon is an address monitor.
type AddrMon struct {
	events  chan netlink.AddrUpdate
	updates chan *Update
	upsDone chan struct{}
	done    chan struct{}
	closed  chan struct{}
}

// sendUpdate sends an address update.
func (a *AddrMon) sendUpdate(update *Update) {
	select {
	case a.updates <- update:
	case <-a.done:
	}
}

// netlinkAddrSubscribeWithOptions is netlink.AddrSubscribeWithOptions for testing.
var netlinkAddrSubscribeWithOptions = netlink.AddrSubscribeWithOptions

// RegisterAddrUpdates registers for addr update events.
var RegisterAddrUpdates = func(a *AddrMon) (chan netlink.AddrUpdate, error) {
	// register for addr update events
	events := make(chan netlink.AddrUpdate)
	options := netlink.AddrSubscribeOptions{
		ListExisting: true,
	}
	if err := netlinkAddrSubscribeWithOptions(events, a.upsDone, options); err != nil {
		return nil, fmt.Errorf("could not subscribe to address events: %w", err)
	}

	return events, nil
}

// start starts the address monitor.
func (a *AddrMon) start() {
	defer close(a.closed)
	defer close(a.updates)
	defer close(a.upsDone)

	// handle events
	for {
		select {
		case e, ok := <-a.events:
			if !ok {
				// unexpected close of events, try to re-open
				log.Error("AddrMon got unexpected close of addr events")
				events, err := RegisterAddrUpdates(a)
				if err != nil {
					log.WithError(err).Error("AddrMon register addr updates error")
				}
				a.events = events
				break
			}

			// forward event as address update
			ip, ok := netip.AddrFromSlice(e.LinkAddress.IP)
			if !ok || !ip.IsValid() {
				log.WithField("LinkAddress", e.LinkAddress).
					Error("AddrMon got invalid IP in addr event")
				continue
			}
			ones, _ := e.LinkAddress.Mask.Size()
			addr := netip.PrefixFrom(ip, ones)
			u := &Update{
				Address: addr,
				Index:   e.LinkIndex,
				Add:     e.NewAddr,
			}
			a.sendUpdate(u)

		case <-a.done:
			// drain events and wait for channel shutdown; this
			// could take until the next addr update
			go func() {
				for range a.events {
					// wait for channel shutdown
					log.Debug("AddrMon dropping event after stop")
				}
			}()

			// stop address monitor
			return
		}
	}
}

// Start starts the address monitor.
func (a *AddrMon) Start() error {
	// register for addr update events
	events, err := RegisterAddrUpdates(a)
	if err != nil {
		return err
	}
	a.events = events

	go a.start()
	return nil
}

// Stop stops the address monitor.
func (a *AddrMon) Stop() {
	close(a.done)
	<-a.closed
}

// Updates returns the address updates channel.
func (a *AddrMon) Updates() chan *Update {
	return a.updates
}

// NewAddrMon returns a new address monitor.
func NewAddrMon() *AddrMon {
	return &AddrMon{
		updates: make(chan *Update),
		upsDone: make(chan struct{}),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
}
