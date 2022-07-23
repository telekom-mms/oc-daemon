package cpd

import (
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	// host is the host address used for probing
	// TODO: add chrome, android, firefox URLs?
	host = "connectivity-check.ubuntu.com"

	// httpTimeout is the timeout for http requests in seconds
	httpTimeout = 5

	// numProbes is the number of probes to run
	numProbes = 3

	// detectedTimer is the probe timer in case of a detected portal
	// in seconds
	detectedTimer = 15

	// regularTimer is the probe timer in case of no detected portal
	// in seconds
	regularTimer = 300
)

// Report is a captive portal detection report
type Report struct {
	Detected bool
	Host     string
}

// CPD is a captive portal detection instance
type CPD struct {
	reports chan *Report
	probes  chan struct{}
	done    chan struct{}
}

// check probes the http server
func (c *CPD) check() *Report {
	// send http request
	client := &http.Client{
		Timeout: httpTimeout * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("http://" + host)
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
		url, err := resp.Location()
		if err != nil {
			log.WithError(err).Error("CPD could not get location in response")
		}
		return &Report{
			Detected: true,
			Host:     url.Hostname(),
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
		for i := 0; i < numProbes; i++ {
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
	timer := time.NewTimer(regularTimer * time.Second)
	resetTimer := func() {
		if detected {
			timer.Reset(detectedTimer * time.Second)
		} else {
			timer.Reset(regularTimer * time.Second)
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
	return []string{host}
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
func NewCPD() *CPD {
	return &CPD{
		reports: make(chan *Report),
		probes:  make(chan struct{}),
		done:    make(chan struct{}),
	}
}
