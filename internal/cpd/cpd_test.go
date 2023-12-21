package cpd

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestCPDProbeCheck(t *testing.T) {
	// status code , detected, early stop
	t.Run("stop during probe", func(_ *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer ts.Close()
		c := NewCPD(NewConfig())
		c.config.Host = ts.Listener.Addr().String()
		c.config.ProbeWait = 0
		close(c.done)
		c.probe()
	})

	t.Run("redirect without url", func(_ *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusFound)
		}))
		defer ts.Close()
		c := NewCPD(NewConfig())
		c.config.Host = ts.Listener.Addr().String()
		c.config.ProbeWait = 0
		c.check()
	})

	t.Run("invalid server", func(_ *testing.T) {
		c := NewCPD(NewConfig())
		c.config.Host = ""
		c.config.ProbeWait = 0
		c.check()
	})

	t.Run("invalid content length", func(_ *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()
		c := NewCPD(NewConfig())
		c.config.Host = ts.Listener.Addr().String()
		c.config.ProbeWait = 0
		c.check()
	})
}

func TestCPDHandleProbeRequest(t *testing.T) {
	c := NewCPD(NewConfig())
	close(c.done)

	c.handleProbeRequest()
	if !c.running {
		t.Error("should be running")
	}

	c.running = true
	c.handleProbeRequest()
	if !c.running {
		t.Error("should still be running")
	}
}

func TestCPDHandleProbeReport(t *testing.T) {
	c := NewCPD(NewConfig())

	go func() {
		c.probes <- struct{}{}
		<-c.reports
		close(c.done)
	}()

	// TODO: read and compare reports?
	c.handleProbeReport(&Report{Detected: true})
	if !c.detected {
		t.Error("detected should be true")
	}
	if !c.running {
		t.Error("should be running")
	}
	//c.running = true
	c.handleProbeReport(&Report{})
}

func TestCPDHandleTimer(t *testing.T) {
	for _, detected := range []bool{
		false,
		true,
	} {
		c := NewCPD(NewConfig())
		c.timer = time.NewTimer(0)
		c.detected = detected
		c.handleTimer()

		if c.running != true {
			t.Error("probe should be running")
		}
	}
}

// TestCPDStartStop tests Start and Stop of CPD
func TestCPDStartStop(t *testing.T) {
	// start and stop immediately
	c := NewCPD(NewConfig())
	c.Start()
	c.Stop()

	// start and stop with timer event, probe result
	conf := NewConfig()
	conf.Host = ""
	conf.ProbeTimer = 0
	conf.ProbeWait = 0
	c = NewCPD(conf)
	c.Start()
	r := <-c.Results()
	c.Stop()

	if r.Detected {
		t.Error("detected should be false")
	}
}

// TestCPDHosts tests Hosts of CPD
func TestCPDHosts(t *testing.T) {
	config := NewConfig()
	config.Host = "test"
	c := NewCPD(config)
	want := []string{"test"}
	got := c.Hosts()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestCPDProbe tests Probe of CPD
func TestCPDProbe(t *testing.T) {
	// status code 204, not detected
	t.Run("not detected", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer ts.Close()
		c := NewCPD(NewConfig())
		c.config.Host = ts.Listener.Addr().String()
		c.config.ProbeWait = 0
		c.Start()
		c.Probe()
		r := <-c.Results()
		if r.Detected {
			t.Error("should not be detected")
		}
		c.Stop()
	})

	// status code 302, detected
	t.Run("detected", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://example.com", http.StatusFound)
		}))
		defer ts.Close()
		c := NewCPD(NewConfig())
		c.config.Host = ts.Listener.Addr().String()
		c.config.ProbeWait = 0
		c.Start()
		c.Probe()
		r := <-c.Results()
		if !r.Detected {
			t.Error("should be detected")
		}
		c.Stop()
	})
}

// TestCPDResults tests Results of CPD
func TestCPDResults(t *testing.T) {
	c := NewCPD(NewConfig())
	want := c.reports
	got := c.Results()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewCPD tests NewCPD
func TestNewCPD(t *testing.T) {
	config := NewConfig()
	c := NewCPD(config)
	if !reflect.DeepEqual(c.config, config) {
		t.Errorf("got %v, want %v", c.config, config)
	}
	if c.reports == nil ||
		c.probes == nil ||
		c.done == nil ||
		c.closed == nil ||
		c.probeReports == nil {

		t.Errorf("got nil, want != nil")
	}
}
