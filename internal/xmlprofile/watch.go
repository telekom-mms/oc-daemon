package xmlprofile

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

type Watch struct {
	file    string
	updates chan struct{}
	done    chan struct{}
	hash    [sha256.Size]byte
}

// sendUpdate sends an update over the updates channel
func (w *Watch) sendUpdate() {
	// send an update or abort if we are shutting down
	select {
	case w.updates <- struct{}{}:
	case <-w.done:
	}
}

// handleEvent compares file hashes to see if the file changed and sends an
// update notification
func (w *Watch) handleEvent() {
	b, err := os.ReadFile(w.file)
	if err != nil {
		log.WithError(err).Error("Could not read xml profile in watcher")
		return
	}

	hash := sha256.Sum256(b)
	if bytes.Equal(hash[:], w.hash[:]) {
		return
	}

	w.hash = hash
	w.sendUpdate()
}

// start starts watching
func (w *Watch) start() {
	defer close(w.updates)

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
	dir := filepath.Dir(w.file)
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
			if event.Name == w.file {
				log.WithFields(log.Fields{
					"name": event.Name,
					"op":   event.Op,
				}).Debug("XML Profile watcher handling file event")
				w.handleEvent()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				log.Error("XML Profile watcher got unexpected " +
					"close of errors channel")
				return
			}
			log.WithError(err).Error("XML Profile watcher error event")

		case <-w.done:
			return
		}
	}
}

// Start starts watching
func (w *Watch) Start() {
	go w.start()
}

// Stop stops watching
func (w *Watch) Stop() {
	close(w.done)
	for range w.updates {
		// wait for channel shutdown
	}
}

// NewWatch returns a new Watch
func NewWatch(file string) *Watch {
	return &Watch{
		file:    file,
		updates: make(chan struct{}),
		done:    make(chan struct{}),
	}
}
