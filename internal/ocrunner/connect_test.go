package ocrunner

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/pkg/logininfo"
)

// TestConnectStartStop tests Start and Stop of Connect.
func TestConnectStartStop(_ *testing.T) {
	c := NewConnect()
	c.Start()
	c.Stop()
}

// TestConnectSavePidFile tests savePidFile of Connect.
func TestConnectSavePidFile(t *testing.T) {
	// clean up after tests
	defer func() {
		osWriteFile = os.WriteFile
		userLookup = user.Lookup
		userLookupGroup = user.LookupGroup
		osChown = os.Chown
	}()

	conf := daemoncfg.NewConfig()
	conf.OpenConnect.PIDFile = t.TempDir() + "pidfile"

	// no process
	c := NewConnect()
	c.savePidFile(conf)

	// with chown error
	userLookup = func(string) (*user.User, error) {
		return &user.User{Uid: "10000"}, nil
	}
	userLookupGroup = func(string) (*user.Group, error) {
		return &user.Group{Gid: "10000"}, nil
	}
	osChown = func(string, int, int) error {
		return errors.New("test error")
	}

	conf.OpenConnect.PIDFile = t.TempDir() + "pidfile"
	conf.OpenConnect.PIDOwner = "test"
	conf.OpenConnect.PIDGroup = "test"

	c = NewConnect()
	c.command = &exec.Cmd{Process: &os.Process{Pid: 123}}
	c.savePidFile(conf)

	// with invalid uid/gid
	userLookup = func(string) (*user.User, error) {
		return &user.User{Uid: "invalid"}, nil
	}
	userLookupGroup = func(string) (*user.Group, error) {
		return &user.Group{Gid: "invalid"}, nil
	}

	c = NewConnect()
	c.command = &exec.Cmd{Process: &os.Process{Pid: 123}}
	c.savePidFile(conf)

	// with user/group lookup error
	userLookup = func(string) (*user.User, error) {
		return nil, errors.New("test error")
	}
	userLookupGroup = func(string) (*user.Group, error) {
		return nil, errors.New("test error")
	}

	c = NewConnect()
	c.command = &exec.Cmd{Process: &os.Process{Pid: 123}}
	c.savePidFile(conf)

	// with write error
	osWriteFile = func(string, []byte, fs.FileMode) error {
		return errors.New("test error")
	}

	c = NewConnect()
	c.command = &exec.Cmd{Process: &os.Process{Pid: 123}}
	c.savePidFile(conf)

	// with invalid permissions
	conf.OpenConnect.PIDPermissions = "invalid"

	c = NewConnect()
	c.command = &exec.Cmd{Process: &os.Process{Pid: 123}}
	c.savePidFile(conf)
}

// TestConnectConnect tests Connect of Connect.
func TestConnectConnect(t *testing.T) {
	// clean up after tests
	defer func() { execCommand = exec.Command }()

	conf := daemoncfg.NewConfig()
	conf.OpenConnect.PIDFile = t.TempDir() + "pidfile"
	conf.LoginInfo = &logininfo.LoginInfo{
		Server:      "vpnserver.example.com",
		Cookie:      "3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...",
		Host:        "10.0.0.1",
		ConnectURL:  "https://vpnserver.example.com",
		Fingerprint: "469bb424ec8835944d30bc77c77e8fc1d8e23a42",
		Resolve:     "vpnserver.example.com:10.0.0.1",
	}

	// test with exec error
	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("")
	}

	c := NewConnect()
	c.Start()
	c.Connect(conf, nil)
	c.Stop()

	// test without exec error
	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("sleep", "10")
	}

	c = NewConnect()
	c.Start()
	c.Connect(conf, nil)
	<-c.Events()

	// test double connect
	c.Connect(conf, nil)

	c.Stop()
}

// TestConnectDisconnect tests Disconnect of Connect.
func TestConnectDisconnect(t *testing.T) {
	// clean up after tests
	oldProcessSignal := processSignal
	defer func() {
		processSignal = oldProcessSignal
		execCommand = exec.Command
	}()

	// without connection
	c := NewConnect()
	c.Start()
	c.Disconnect()
	c.Stop()

	// with connection
	conf := daemoncfg.NewOpenConnect()
	conf.PIDFile = t.TempDir() + "pidfile"

	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("sleep", "10")
	}
	c = NewConnect()
	c.Start()
	c.Connect(daemoncfg.NewConfig(), nil)
	<-c.Events()
	c.Disconnect()
	<-c.Events()
	c.Stop()

	// handleDisconnect, signal error
	processSignal = func(*os.Process, os.Signal) error {
		return errors.New("test error")
	}
	c = NewConnect()
	c.command = &exec.Cmd{Process: &os.Process{}}
	c.handleDisconnect()
}

// TestConnectEvents tests Events of Connect.
func TestConnectEvents(t *testing.T) {
	c := NewConnect()

	want := c.events
	got := c.Events()
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestNewConnect tests NewConnect.
func TestNewConnect(t *testing.T) {
	c := NewConnect()

	if c.exits == nil ||
		c.commands == nil ||
		c.done == nil ||
		c.closed == nil ||
		c.events == nil {

		t.Errorf("got nil, want != nil")
	}
}

// TestCleanupConnect tests CleanupConnect.
func TestCleanupConnect(_ *testing.T) {
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

	CleanupConnect(daemoncfg.NewOpenConnect())

	// PID file contains garbage
	osReadFile = func(string) ([]byte, error) {
		return []byte("garbage"), nil
	}

	CleanupConnect(daemoncfg.NewOpenConnect())

	// cannot read process cmdline
	reads := 0
	osReadFile = func(string) ([]byte, error) {
		if reads > 0 {
			return nil, errors.New("test error")
		}
		reads++
		return []byte("123"), nil
	}

	CleanupConnect(daemoncfg.NewOpenConnect())

	// process cmdline does not contain openconnect (other process)
	reads = 0
	osReadFile = func(string) ([]byte, error) {
		if reads > 0 {
			return []byte("other"), nil
		}
		reads++
		return []byte("123"), nil
	}

	CleanupConnect(daemoncfg.NewOpenConnect())

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

	CleanupConnect(daemoncfg.NewOpenConnect())

	// stop process
	reads = 0
	osFindProcess = func(int) (*os.Process, error) {
		return &os.Process{}, nil
	}
	processSignal = func(*os.Process, os.Signal) error {
		return nil
	}

	CleanupConnect(daemoncfg.NewOpenConnect())
}
