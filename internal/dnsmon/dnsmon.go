package dnsmon

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

const (
	// resolv.conf files in /etc and /run/systemd/resolve
	etc               = "/etc"
	etcResolvConf     = etc + "/resolv.conf"
	systemdResolveDir = "/run/systemd/resolve"
	systemdResolvConf = systemdResolveDir + "/resolv.conf"
	stubResolvConf    = systemdResolveDir + "/stub-resolv.conf"
)

// DNSMon is a DNS monitor
type DNSMon struct {
	updates chan struct{}
	done    chan struct{}
}

// isResolvConfEvent checks if event is a resolv.conf file event
func isResolvConfEvent(event fsnotify.Event) bool {
	switch event.Name {
	case etcResolvConf:
		return true
	case stubResolvConf:
		return true
	case systemdResolvConf:
		return true
	}
	return false
}

// sendUpdate sends an update over the updates channel
func (d *DNSMon) sendUpdate() {
	// send an update or abort if we are shutting down
	select {
	case d.updates <- struct{}{}:
	case <-d.done:
	}
}

// start starts the DNSMon
func (d *DNSMon) start() {
	defer close(d.updates)

	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Fatal("DNSMon file watcher error")
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.WithError(err).Error("DNSMon file watcher close error")
		}
	}()

	// add resolv.conf folders to watcher
	if err := watcher.Add(etc); err != nil {
		log.WithError(err).Debug("DNSMon add etc dir error")
	}
	if err := watcher.Add(systemdResolveDir); err != nil {
		log.WithError(err).Debug("DNSMon add systemd dir error")
	}

	// send initial update
	d.sendUpdate()

	// watch the files
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Error("DNSMon got unexpected close of events channel")
				return
			}
			if isResolvConfEvent(event) {
				log.WithFields(log.Fields{
					"name": event.Name,
					"op":   event.Op,
				}).Debug("DNSMon handling resolv.conf event")
				d.sendUpdate()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				log.Error("DNSMon got unexpected close of errors channel")
				return
			}
			log.WithError(err).Error("DNSMon watcher error event")

		case <-d.done:
			return
		}
	}
}

// Start starts the DNSMon
func (d *DNSMon) Start() {
	go d.start()
}

// Stop stops the DNSMon
func (d *DNSMon) Stop() {
	close(d.done)
	for range d.updates {
		// wait for channel shutdown
	}
}

// Updates returns the channel for dns config updates
func (d *DNSMon) Updates() chan struct{} {
	return d.updates
}

// NewDNSMon returns a new DNSMon
func NewDNSMon() *DNSMon {
	return &DNSMon{
		updates: make(chan struct{}),
		done:    make(chan struct{}),
	}
}
