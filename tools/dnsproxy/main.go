/*
Dnsproxy is a DNS proxy example.
*/
package main

import (
	"flag"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/dnsproxy"
)

var (
	// parsed server address and remotes
	address = "127.0.0.1:5301"
	remotes = make(map[string][]string)
	watches = []string{}
)

// parseCommandLine parses command line arguments
func parseCommandLine() {
	addr := flag.String("address", address, "set local listen `address`")
	rems := flag.String("remotes", "", "set `remotes` as comma-separated "+
		"list of domain:remote pairs")
	wchs := flag.String("watches", "", "set watches as comma-separated "+
		"list of `domains`")
	flag.Parse()

	// parse local listen address
	if *addr != "" {
		// TODO: check listen address
		address = *addr
	}

	// make sure there are domain:remote pairs
	if *rems == "" {
		log.Fatal("DNS-Proxy got no domain:remote pairs from command line")
	}

	// parse all domain:remote pairs
	for _, arg := range strings.Split(*rems, ",") {
		// parse domain and remote address
		i := strings.Index(arg, ":")
		if i == -1 || len(arg) < i+2 {
			log.Fatal("DNS-Proxy got invalid domain:remote pair from command line")
		}
		domain := arg[:i]
		remote := arg[i+1:]

		// TODO: check domain name
		// TODO: check remote address and port

		// add domain and remote to remotes
		log.WithFields(log.Fields{
			"domain": domain,
			"remote": remote,
		}).Debug("DNS-Proxy got remote from command line")
		remotes[domain] = append(remotes[domain], remote)
	}

	// parse watches
	if *wchs != "" {
		watches = strings.Split(*wchs, ",")
		for _, w := range watches {
			log.WithField("watch", w).Debug("DNS-Proxy got watch from command line")
		}
	}
}

func main() {
	log.SetLevel(log.DebugLevel)
	parseCommandLine()
	c := dnsproxy.NewConfig()
	c.Address = address
	c.ListenUDP = true
	c.ListenTCP = true
	p := dnsproxy.NewProxy(c)
	p.SetRemotes(remotes)
	p.SetWatches(watches)
	go p.Start()
	for r := range p.Reports() {
		log.WithField("report", r).Debug("DNS-Proxy got watched domain report")
		r.Done()
	}
}
