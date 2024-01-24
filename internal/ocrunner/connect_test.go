package ocrunner

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
)

// TestConnectStartStop tests Start and Stop of Connect
func TestConnectStartStop(t *testing.T) {
	c := NewConnect(NewConfig())
	c.Start()
	c.Stop()
}

func TestConnectConnect(t *testing.T) {
	// clean up after tests
	defer func() { execCommand = exec.Command }()

	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("cat")
	}

	conf := NewConfig()
	conf.PIDFile = t.TempDir() + "pidfile"
	c := NewConnect(conf)
	c.Start()
	c.Connect(&logininfo.LoginInfo{}, nil)
	<-c.Events()
	<-c.Events()
	c.Stop()
}

// TestConnectDisconnect tests Disconnect of Connect
func TestConnectDisconnect(t *testing.T) {
	c := NewConnect(NewConfig())
	c.Start()
	c.Disconnect()
	c.Stop()
}

// TestConnectEvents tests Events of Connect
func TestConnectEvents(t *testing.T) {
	c := NewConnect(NewConfig())

	want := c.events
	got := c.Events()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewConnect tests NewConnect
func TestNewConnect(t *testing.T) {
	config := NewConfig()
	config.XMLProfile = "/some/profile/file"
	config.VPNCScript = "/some/vpnc/script"
	config.VPNDevice = "tun999"
	c := NewConnect(config)
	if !reflect.DeepEqual(c.config, config) {
		t.Errorf("got %v, want %v", c.config, config)
	}
	if c.exits == nil ||
		c.commands == nil ||
		c.done == nil ||
		c.closed == nil ||
		c.events == nil {

		t.Errorf("got nil, want != nil")
	}
}

// TestCleanupConnect tests CleanupConnect.
func TestCleanupConnect(t *testing.T) {
	// clean up after tests
	oldProcessSignal := processSignal
	defer func() {
		osReadFile = os.ReadFile
		osFindProcess = os.FindProcess
		processSignal = oldProcessSignal
	}()

	// cannot read PID file
	osReadFile = func(string) ([]byte, error) {
		return nil, errors.New("test error")
	}

	CleanupConnect(NewConfig())

	// PID file contains garbage
	osReadFile = func(string) ([]byte, error) {
		return []byte("garbage"), nil
	}

	CleanupConnect(NewConfig())

	// cannot read process cmdline
	reads := 0
	osReadFile = func(string) ([]byte, error) {
		if reads > 0 {
			return nil, errors.New("test error")
		}
		reads++
		return []byte("123"), nil
	}

	CleanupConnect(NewConfig())

	// process cmdline does not contain openconnect (other process)
	reads = 0
	osReadFile = func(string) ([]byte, error) {
		if reads > 0 {
			return []byte("other"), nil
		}
		reads++
		return []byte("123"), nil
	}

	CleanupConnect(NewConfig())

	// cannot find process (process already terminated)
	reads = 0
	osReadFile = func(string) ([]byte, error) {
		if reads > 0 {
			return []byte("openconnect"), nil
		}
		reads++
		return []byte("123"), nil
	}
	osFindProcess = func(int) (*os.Process, error) {
		return nil, errors.New("test error")
	}

	CleanupConnect(NewConfig())

	// stop process
	reads = 0
	osFindProcess = func(int) (*os.Process, error) {
		return &os.Process{}, nil
	}
	processSignal = func(*os.Process, os.Signal) error {
		return nil
	}

	CleanupConnect(NewConfig())
}
