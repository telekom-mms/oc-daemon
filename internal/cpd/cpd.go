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

// start starts the captive portal detection
func (c *CPD) start() {
	defer close(c.reports)

	// probe channels and function
	probeReports := make(chan *Report)
	probesDone := make(chan struct{})
	probeFunc := func() {
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
		case probeReports <- r:
		case <-probesDone:
			return
		}
	}

	// is a captive portal detected, are probes currently running or
	// have to run again?
	detected := false
	running := false
	runAgain := false

	// set timer for periodic checks
	timer := time.NewTimer(c.config.ProbeTimer)
	resetTimer := func() {
		if detected {
			timer.Reset(c.config.ProbeTimerDetected)
		} else {
			timer.Reset(c.config.ProbeTimer)
		}
	}

	// helper for sending a probe report
	reportFunc := func(r *Report) {
		// try sending the report back,
		// in the meantime read incoming probe requests and set the
		// runAgain flag accordingly,
		// stop when the report is sent or we are shutting down
		for {
			select {
			case c.reports <- r:
				running = false
				if runAgain {
					// we must trigger another probe
					runAgain = false
					running = true
					go probeFunc()
				}
				detected = r.Detected
				return
			case <-c.probes:
				runAgain = true
			case <-c.done:
				return
			}
		}
	}

	for {
		select {
		case <-c.probes:
			if running {
				runAgain = true
				break
			}
			running = true
			go probeFunc()

		case r := <-probeReports:
			// send probe report
			reportFunc(r)

			// reset periodic probing timer
			if running {
				// probing still active and new report about
				// to arrive, so wait for it before resetting
				// the timer
				break
			}
			if !timer.Stop() {
				<-timer.C
			}
			resetTimer()

		case <-timer.C:
			if !running && !runAgain {
				// no probes active, trigger new probe
				log.Debug("periodic CPD timer")
				running = true
				go probeFunc()
			}

			// reset timer
			resetTimer()

		case <-c.done:
			close(probesDone)
			if !timer.Stop() {
				<-timer.C
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
	for range c.reports {
		// wait for channel shutdown
	}
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
	}
}
