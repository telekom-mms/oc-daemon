package trafpol

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/oc-daemon/internal/cpd"
	"github.com/telekom-mms/oc-daemon/internal/devmon"
	"github.com/telekom-mms/oc-daemon/internal/dnsmon"
)

// TrafPol is a traffic policing component
type TrafPol struct {
	config *Config
	devmon *devmon.DevMon
	dnsmon *dnsmon.DNSMon
	cpd    *cpd.CPD

	// capPortal indicates if a captive portal is detected
	capPortal bool

	allowDevs  *AllowDevs
	allowHosts *AllowHosts

	loopDone chan struct{}
	done     chan struct{}
}

// handleDeviceUpdate handles a device update
func (t *TrafPol) handleDeviceUpdate(ctx context.Context, u *devmon.Update) {
	// skip physical devices and only allow virtual devices
	if u.Type == "device" {
		return
	}

	// add or remove virtual device to/from allowed devices
	if u.Add {
		t.allowDevs.Add(ctx, u.Device)
		return
	}
	t.allowDevs.Remove(ctx, u.Device)
}

// handleDNSUpdate handles a dns config update
func (t *TrafPol) handleDNSUpdate() {
	// update allowed hosts
	t.allowHosts.Update()

	// triger captive portal detection
	t.cpd.Probe()
}

// handleCPDReport handles a CPD report
func (t *TrafPol) handleCPDReport(ctx context.Context, report *cpd.Report) {
	if !report.Detected {
		// no captive portal detected
		// check if there was a portal before
		if t.capPortal {
			// refresh all IPs, maybe they pointed to a
			// portal host in case of dns-based portals
			t.allowHosts.Update()

			// remove ports from allowed ports
			removePortalPorts(ctx)
			t.capPortal = false
		}
		return
	}

	// add ports to allowed ports
	if !t.capPortal {
		addPortalPorts(ctx)
		t.capPortal = true
	}
}

// start starts the traffic policing component
func (t *TrafPol) start() {
	log.Debug("TrafPol starting")
	defer close(t.loopDone)

	// create context
	ctx := context.Background()

	// set firewall config
	setFilterRules(ctx, t.config.FirewallMark)
	defer unsetFilterRules(ctx)

	// add CPD hosts to allowed hosts
	for _, h := range t.cpd.Hosts() {
		t.allowHosts.Add(h)
	}

	// start allowed hosts
	t.allowHosts.Start()
	defer t.allowHosts.Stop()

	// start captive portal detection
	t.cpd.Start()
	defer t.cpd.Stop()

	// start device monitor
	t.devmon.Start()
	defer t.devmon.Stop()

	// start dns monitor
	t.dnsmon.Start()
	defer t.dnsmon.Stop()

	// enter main loop
	for {
		select {
		case u := <-t.devmon.Updates():
			// Device Update
			log.WithField("update", u).Debug("TrafPol got DevMon update")
			t.handleDeviceUpdate(ctx, u)

		case <-t.dnsmon.Updates():
			// DNS Update
			log.Debug("TrafPol got DNSMon update")
			t.handleDNSUpdate()

		case r := <-t.cpd.Results():
			// CPD Result
			log.WithField("result", r).Debug("TrafPol got CPD result")
			t.handleCPDReport(ctx, r)

		case <-t.done:
			// shutdown
			return
		}
	}
}

// Start starts the traffic policing component
func (t *TrafPol) Start() {
	go t.start()
}

// Stop stops the traffic policing component
func (t *TrafPol) Stop() {
	close(t.done)

	// wait for everything
	<-t.loopDone
	log.Debug("TrafPol stopped")
}

// NewTrafPol returns a new traffic policing component
func NewTrafPol(config *Config) *TrafPol {
	allowHosts := NewAllowHosts(config)
	for _, h := range config.AllowedHosts {
		allowHosts.Add(h)
	}
	return &TrafPol{
		config: config,
		devmon: devmon.NewDevMon(),
		dnsmon: dnsmon.NewDNSMon(dnsmon.NewConfig()),
		cpd:    cpd.NewCPD(cpd.NewConfig()),

		allowDevs:  NewAllowDevs(),
		allowHosts: allowHosts,

		loopDone: make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Cleanup cleans up old configuration after a failed shutdown
func Cleanup() {
	cleanupFilterRules()
}
