package cpd

import (
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// Report is a captive portal detection report
type Report struct {
	Detected bool
	Host     string
}

// CPD is a captive portal detection instance
type CPD struct {
	config  *Config
	reports chan *Report
	probes  chan struct{}
	done    chan struct{}
	closed  chan struct{}

	// internal probe reports
	probeReports chan *Report

	// timer for periodic checks
	timer *time.Timer

	// is a captive portal detected, are probes currently running or
	// have to run again?
	detected bool
	running  bool
	runAgain bool
}

// resetTimer resets the timer
func (c *CPD) resetTimer() {
	if c.detected {
		c.timer.Reset(c.config.ProbeTimerDetected)
	} else {
		c.timer.Reset(c.config.ProbeTimer)
	}
}

// check probes the http server
func (c *CPD) check() *Report {
	// send http request
	client := &http.Client{
		Timeout: c.config.HTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("http://" + c.config.Host)
	if err != nil {
		log.WithError(err).Debug("CPD GET error")
		return &Report{}
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.WithError(err).Error("CPD read error")
		return &Report{}
	}
	_ = resp.Body.Close()

	// check response code
	switch resp.StatusCode {
	case http.StatusNoContent:
		// 204, what we want, no captive portal
		return &Report{}

	case http.StatusFound:
		// 302, redirect, captive portal detected
		hostname := ""
		if url, err := resp.Location(); err != nil {
			log.WithError(err).Error("CPD could not get location in response")
		} else {
			hostname = url.Hostname()
		}
		return &Report{
			Detected: true,
			Host:     hostname,
		}
	default:
		// other, captive protal detected
		return &Report{
			Detected: true,
		}
	}
}

// probe probes the http server
func (c *CPD) probe() {
	// TODO: improve this?
	r := &Report{}
	for i := 0; i < c.config.ProbeCount; i++ {
		r = c.check()
		if r.Detected {
			break
		}
		time.Sleep(time.Second)
	}
	select {
	case c.probeReports <- r:
	case <-c.done:
		return
	}
}

// sendReport is a helper for sending a probe report
func (c *CPD) sendReport(r *Report) {
	// try sending the report back,
	// in the meantime read incoming probe requests and set the
	// runAgain flag accordingly,
	// stop when the report is sent or we are shutting down
	for {
		select {
		case c.reports <- r:
			c.running = false
			if c.runAgain {
				// we must trigger another probe
				c.runAgain = false
				c.running = true
				go c.probe()
			}
			c.detected = r.Detected
			return
		case <-c.probes:
			c.runAgain = true
		case <-c.done:
			return
		}
	}
}

// handleProbeRequest handles a probe request.
func (c *CPD) handleProbeRequest() {
	if c.running {
		c.runAgain = true
		return
	}
	c.running = true
	go c.probe()

}

// handleProbeReport handles an internal probe report.
func (c *CPD) handleProbeReport(r *Report) {
	// send probe report
	c.sendReport(r)

	// reset periodic probing timer
	if c.running {
		// probing still active and new report about
		// to arrive, so wait for it before resetting
		// the timer
		return
	}
	if !c.timer.Stop() {
		<-c.timer.C
	}
	c.resetTimer()
}

// handleTimer handles a timer event.
func (c *CPD) handleTimer() {
	if !c.running && !c.runAgain {
		// no probes active, trigger new probe
		log.Debug("periodic CPD timer")
		c.running = true
		go c.probe()
	}

	// reset timer
	c.resetTimer()
}

// start starts the captive portal detection
func (c *CPD) start() {
	defer close(c.closed)
	defer close(c.reports)

	// set timer for periodic checks
	c.timer = time.NewTimer(c.config.ProbeTimer)

	for {
		select {
		case <-c.probes:
			c.handleProbeRequest()

		case r := <-c.probeReports:
			c.handleProbeReport(r)

		case <-c.timer.C:
			c.handleTimer()

		case <-c.done:
			if !c.timer.Stop() {
				<-c.timer.C
			}
			return
		}
	}
}

// Start starts the captive portal detection
func (c *CPD) Start() {
	go c.start()
}

// Stop stops the captive portal detection
func (c *CPD) Stop() {
	close(c.done)
	<-c.closed
}

// Hosts returns the host addresses used for captive protal detection
func (c *CPD) Hosts() []string {
	return []string{c.config.Host}
}

// Probe triggers the captive portal detection
func (c *CPD) Probe() {
	c.probes <- struct{}{}
}

// Results returns the results channel
func (c *CPD) Results() chan *Report {
	return c.reports
}

// NewCPD returns a new CPD
func NewCPD(config *Config) *CPD {
	return &CPD{
		config:  config,
		reports: make(chan *Report),
		probes:  make(chan struct{}),
		done:    make(chan struct{}),
		closed:  make(chan struct{}),

		probeReports: make(chan *Report),
	}
}
