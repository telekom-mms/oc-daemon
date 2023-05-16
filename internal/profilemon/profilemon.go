package profilemon

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

// ProfileMon is a XML profile monitor
type ProfileMon struct {
	file    string
	updates chan struct{}
	done    chan struct{}
	hash    [sha256.Size]byte
}

// sendUpdate sends an update over the updates channel
func (p *ProfileMon) sendUpdate() {
	// send an update or abort if we are shutting down
	select {
	case p.updates <- struct{}{}:
	case <-p.done:
	}
}

// handleEvent compares file hashes to see if the file changed and sends an
// update notification
func (p *ProfileMon) handleEvent() {
	b, err := os.ReadFile(p.file)
	if err != nil {
		log.WithError(err).Error("Could not read xml profile in watcher")
		return
	}

	hash := sha256.Sum256(b)
	if bytes.Equal(hash[:], p.hash[:]) {
		return
	}

	p.hash = hash
	p.sendUpdate()
}

// start starts the profile monitor
func (p *ProfileMon) start() {
	defer close(p.updates)

	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Fatal("XML Profile watcher create error")
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.WithError(err).Error("XML Profile watcher close error")
		}
	}()

	// add xml profile folder to watcher
	dir := filepath.Dir(p.file)
	if err := watcher.Add(dir); err != nil {
		log.WithError(err).Debug("XML Profile watcher add profile dir error")
	}

	// watch file
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Error("XML Profile watcher got unexpected " +
					"close of events channel")
				return
			}
			if event.Name == p.file {
				log.WithFields(log.Fields{
					"name": event.Name,
					"op":   event.Op,
				}).Debug("XML Profile watcher handling file event")
				p.handleEvent()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				log.Error("XML Profile watcher got unexpected " +
					"close of errors channel")
				return
			}
			log.WithError(err).Error("XML Profile watcher error event")

		case <-p.done:
			return
		}
	}
}

// Start starts the profile monitor
func (p *ProfileMon) Start() {
	go p.start()
}

// Stop stops the profile monitor
func (p *ProfileMon) Stop() {
	close(p.done)
	for range p.updates {
		// wait for channel shutdown
	}
}

// Updates returns the channel for profile updates
func (p *ProfileMon) Updates() chan struct{} {
	return p.updates
}

// NewProfileMon returns a new profile monitor
func NewProfileMon(file string) *ProfileMon {
	return &ProfileMon{
		file:    file,
		updates: make(chan struct{}),
		done:    make(chan struct{}),
	}
}
