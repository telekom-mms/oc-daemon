package dnsmon

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

// DNSMon is a DNS monitor
type DNSMon struct {
	config  *Config
	watcher *fsnotify.Watcher
	updates chan struct{}
	done    chan struct{}
	closed  chan struct{}
}

// isResolvConfEvent checks if event is a resolv.conf file event
func (d *DNSMon) isResolvConfEvent(event fsnotify.Event) bool {
	switch event.Name {
	case d.config.ETCResolvConf:
		return true
	case d.config.StubResolvConf:
		return true
	case d.config.SystemdResolvConf:
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
	defer close(d.closed)
	defer close(d.updates)
	defer func() {
		if err := d.watcher.Close(); err != nil {
			log.WithError(err).Error("DNSMon file watcher close error")
		}
	}()

	// send initial update
	d.sendUpdate()

	// watch the files
	for {
		select {
		case event, ok := <-d.watcher.Events:
			if !ok {
				log.Error("DNSMon got unexpected close of events channel")
				return
			}
			if d.isResolvConfEvent(event) {
				log.WithFields(log.Fields{
					"name": event.Name,
					"op":   event.Op,
				}).Debug("DNSMon handling resolv.conf event")
				d.sendUpdate()
			}

		case err, ok := <-d.watcher.Errors:
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
	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Fatal("DNSMon file watcher error")
	}

	// add resolv.conf folders to watcher
	for _, dir := range d.config.resolvConfDirs() {
		if err := watcher.Add(dir); err != nil {
			log.WithError(err).WithField("dir", dir).Debug("DNSMon add resolv.conf dir error")
		}
	}

	d.watcher = watcher
	go d.start()
}

// Stop stops the DNSMon
func (d *DNSMon) Stop() {
	close(d.done)
	<-d.closed
}

// Updates returns the channel for dns config updates
func (d *DNSMon) Updates() chan struct{} {
	return d.updates
}

// NewDNSMon returns a new DNSMon
func NewDNSMon(config *Config) *DNSMon {
	return &DNSMon{
		config:  config,
		updates: make(chan struct{}),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),
	}
}
