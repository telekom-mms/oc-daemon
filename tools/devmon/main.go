/*
Devmon is a device monitor example.
*/
package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
)

func main() {
	log.SetLevel(log.DebugLevel)
	d := devmon.NewDevMon()
	if err := d.Start(); err != nil {
		log.WithError(err).Fatal("could not start DevMon")
	}
	for u := range d.Updates() {
		log.Println(u)
	}
}
