/*
Ocrunner is a OC-Runner example.
*/
package main

import (
	"flag"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/ocrunner"
	"github.com/telekom-mms/oc-daemon/pkg/client"
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

	// create config
	config := client.NewConfig()
	config.ClientCertificate = *cert
	config.ClientKey = *key
	config.CACertificate = *ca
	config.XMLProfile = *profile
	config.VPNServer = *server

	// authenticate client
	a, err := client.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = a.Close() }()
	if *authenticate {
		err := a.Authenticate()
		if err != nil {
			log.Error(err)
		}
	}

	// connect client
	ocrConf := ocrunner.NewConfig()
	ocrConf.XMLProfile = *profile
	ocrConf.VPNCScript = *script
	c := ocrunner.NewConnect(ocrConf)
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
		c.Connect(a.GetLogin(), []string{})
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
