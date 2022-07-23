package main

import (
	"github.com/T-Systems-MMS/oc-daemon/internal/devmon"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)
	d := devmon.NewDevMon()
	go d.Start()
	for u := range d.Updates() {
		log.Println(u)
	}
}
