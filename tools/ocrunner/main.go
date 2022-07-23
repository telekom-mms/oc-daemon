package main

import (
	"flag"
	"time"

	"github.com/T-Systems-MMS/oc-daemon/internal/ocrunner"
	log "github.com/sirupsen/logrus"
)

func main() {
	// set log level
	log.SetLevel(log.DebugLevel)

	// set command line arguments
	authenticate := flag.Bool("authenticate", false, "authenticate client")
	connect := flag.Bool("connect", false, "connect client")
	disconnect := flag.Bool("disconnect", false, "disconnect client")

	cert := flag.String("cert", "client.crt", "set client certificate `file`")
	key := flag.String("key", "client.key", "set client key `file`")
	ca := flag.String("ca", "ca.crt", "set ca `file`")
	profile := flag.String("profile", "./profile.xml", "set XML profile `file`")
	script := flag.String("script", "vpncscript", "set script `file`")
	server := flag.String("server", "", "set `server`")

	flag.Parse()

	// check parameters
	if !*authenticate && !*connect {
		log.Fatal("OC-Runner got neither authenticate nor connect from command line")
	}

	// authenticate client
	a := ocrunner.NewAuthenticate()
	if *authenticate {
		a.Certificate = *cert
		a.Key = *key
		a.CA = *ca
		a.XMLProfile = *profile
		a.Script = *script
		a.Server = *server
		a.Authenticate()
	}

	// connect client
	c := ocrunner.NewConnect(*profile, *script, "oc-daemon-tun0")
	done := make(chan struct{})
	go c.Start()
	go func() {
		for e := range c.Events() {
			log.WithField("event", e).Debug("OC-Runner got event")
			if !e.Connect {
				break
			}
		}
		done <- struct{}{}
	}()
	if *connect {
		c.Connect(&a.Login, []string{})
	}

	// disconnect client
	if *disconnect {
		time.Sleep(10 * time.Second)
		c.Disconnect()
		c.Stop()
	}

	// wait for command exit
	<-done
	time.Sleep(1 * time.Second)
}
