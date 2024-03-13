package dbusapi

import (
	"errors"
	"reflect"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// TestRequestWaitClose tests Wait and Close of Request.
func TestRequestWaitClose(_ *testing.T) {
	// test closing
	r := Request{
		Name: "test1",
		wait: make(chan struct{}),
		done: make(chan struct{}),
	}
	go func() {
		r.Close()
	}()
	r.Wait()

	// test aborting
	done := make(chan struct{})
	r = Request{
		Name: "test2",
		wait: make(chan struct{}),
		done: done,
	}
	go func() {
		close(done)
	}()
	r.Wait()
}

// TestDaemonConnectErrors tests Connect of daemon, errors.
func TestDaemonConnectErrors(t *testing.T) {
	// create daemon
	requests := make(chan *Request)
	done := make(chan struct{})
	daemon := daemon{
		requests: requests,
		done:     done,
	}

	// error when handling request
	go func() {
		r := <-requests
		r.Error = errors.New("test error")
		r.Close()
	}()
	if err := daemon.Connect("", "", "", "", "", "", ""); err == nil {
		t.Error("should return error")
	}

	// closed daemon
	close(done)
	if err := daemon.Connect("", "", "", "", "", "", ""); err == nil {
		t.Error("should return error")
	}
}

// TestDaemonConnect tests Connect of daemon.
func TestDaemonConnect(t *testing.T) {
	// create daemon
	requests := make(chan *Request)
	done := make(chan struct{})
	daemon := daemon{
		requests: requests,
		done:     done,
	}

	// run connect and get results
	server, cookie, host, connectURL, fingerprint, resolve :=
		"server", "cookie", "host", "connectURL", "fingerprint", "resolve"
	want := &Request{
		Name:       RequestConnect,
		Parameters: []any{server, cookie, host, connectURL, fingerprint, resolve},
		done:       done,
	}
	got := &Request{}
	go func() {
		r := <-requests
		got = r
		r.Close()
	}()
	err := daemon.Connect("sender", server, cookie, host, connectURL, fingerprint, resolve)
	if err != nil {
		t.Error(err)
	}

	// check results
	if got.Name != want.Name ||
		!reflect.DeepEqual(got.Parameters, want.Parameters) ||
		!reflect.DeepEqual(got.Results, want.Results) ||
		got.Error != want.Error ||
		got.done != want.done {
		// not equal
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDaemonDisconnectErrors tests Disconnect of daemon, errors.
func TestDaemonDisconnectErrors(t *testing.T) {
	// create daemon
	requests := make(chan *Request)
	done := make(chan struct{})
	daemon := daemon{
		requests: requests,
		done:     done,
	}

	// error when handling request
	go func() {
		r := <-requests
		r.Error = errors.New("test error")
		r.Close()
	}()
	if err := daemon.Disconnect(""); err == nil {
		t.Error("should return error")
	}

	// closed daemon
	close(done)
	if err := daemon.Disconnect(""); err == nil {
		t.Error("should return error")
	}
}

// TestDaemonDisconnect tests Disconnect of daemon.
func TestDaemonDisconnect(t *testing.T) {
	// create daemon
	requests := make(chan *Request)
	done := make(chan struct{})
	daemon := daemon{
		requests: requests,
		done:     done,
	}

	// run disconnect and get results
	want := &Request{
		Name: RequestDisconnect,
		done: done,
	}
	got := &Request{}
	go func() {
		r := <-requests
		got = r
		r.Close()
	}()
	err := daemon.Disconnect("sender")
	if err != nil {
		t.Error(err)
	}

	// check results
	if got.Name != want.Name ||
		!reflect.DeepEqual(got.Parameters, want.Parameters) ||
		!reflect.DeepEqual(got.Results, want.Results) ||
		got.Error != want.Error ||
		got.done != want.done {
		// not equal
		t.Errorf("got %v, want %v", got, want)
	}
}

// testConn implements the dbusConn interface for testing.
type testConn struct {
	reqNameReply dbus.RequestNameReply
	reqNameError error
	exportOKNum  int
	exportError  error
}

func (tc *testConn) Close() error {
	return nil
}

func (tc *testConn) Export(any, dbus.ObjectPath, string) error {
	if tc.exportOKNum > 0 {
		tc.exportOKNum--
		return nil
	}
	return tc.exportError
}

func (tc *testConn) RequestName(string, dbus.RequestNameFlags) (dbus.RequestNameReply, error) {
	return tc.reqNameReply, tc.reqNameError
}

// testProperties implements the propProperties interface for testing.
type testProperties struct {
	props map[string]any
}

func (tp *testProperties) Introspection(string) []introspect.Property {
	return nil
}

func (tp *testProperties) SetMust(_, property string, v any) {
	if tp.props == nil {
		// props not set, skip
		return
	}

	// ignore iface, map property to value
	tp.props[property] = v
}

// TestServiceStartStop tests Start and Stop of Service.
func TestServiceStartStop(t *testing.T) {
	// clean up after tests
	oldDbusConnectSystemBus := dbusConnectSystemBus
	oldPropExport := propExport
	defer func() {
		dbusConnectSystemBus = oldDbusConnectSystemBus
		propExport = oldPropExport
	}()

	// no errors
	dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
		return &testConn{
			reqNameReply: dbus.RequestNameReplyPrimaryOwner,
			exportOKNum:  2,
		}, nil
	}
	propExport = func(conn dbusConn, path dbus.ObjectPath, props prop.Map) (propProperties, error) {
		return &testProperties{}, nil
	}
	s := NewService()
	if err := s.Start(); err != nil {
		t.Error(err)
	}
	s.Stop()

	// conn export introspectable error
	dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
		return &testConn{
			reqNameReply: dbus.RequestNameReplyPrimaryOwner,
			exportOKNum:  1,
			exportError:  errors.New("test error"),
		}, nil
	}
	s = NewService()
	if err := s.Start(); err == nil {
		t.Error("conn export introspectable error should return error")
	}

	// props export error
	propExport = func(conn dbusConn, path dbus.ObjectPath, props prop.Map) (propProperties, error) {
		return nil, errors.New("test error")
	}
	s = NewService()
	if err := s.Start(); err == nil {
		t.Error("props export error should return error")
	}

	// conn export methods error
	dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
		return &testConn{
			reqNameReply: dbus.RequestNameReplyPrimaryOwner,
			exportError:  errors.New("test error"),
		}, nil
	}
	s = NewService()
	if err := s.Start(); err == nil {
		t.Error("conn export methods error should return error")
	}

	// bus name alredy taken
	dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
		return &testConn{
			reqNameReply: dbus.RequestNameReplyExists,
		}, nil
	}
	s = NewService()
	if err := s.Start(); err == nil {
		t.Error("bus name already taken should return error")
	}

	// conn request name error
	dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
		return &testConn{
			reqNameError: errors.New("test error"),
		}, nil
	}
	s = NewService()
	if err := s.Start(); err == nil {
		t.Error("conn request name error should return error")
	}

	// dbus connect error
	dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
		return nil, errors.New("test error")
	}
	s = NewService()
	if err := s.Start(); err == nil {
		t.Error("dbus connect error should return error")
	}
}

// TestServiceRequests tests Requests of Service.
func TestServiceRequests(t *testing.T) {
	s := NewService()
	want := s.requests
	got := s.Requests()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestServiceSetProperty tests SetProperty of Service.
func TestServiceSetProperty(t *testing.T) {
	// clean up after tests
	oldDbusConnectSystemBus := dbusConnectSystemBus
	oldPropExport := propExport
	defer func() {
		dbusConnectSystemBus = oldDbusConnectSystemBus
		propExport = oldPropExport
	}()

	dbusConnectSystemBus = func(opts ...dbus.ConnOption) (dbusConn, error) {
		return &testConn{
			reqNameReply: dbus.RequestNameReplyPrimaryOwner,
			exportOKNum:  2,
		}, nil
	}
	properties := &testProperties{props: make(map[string]any)}
	propExport = func(conn dbusConn, path dbus.ObjectPath, props prop.Map) (propProperties, error) {
		return properties, nil
	}
	s := NewService()
	if err := s.Start(); err != nil {
		t.Error(err)
	}

	propName := "test-property"
	want := "test-value"

	s.SetProperty(propName, want)
	s.Stop()

	got := properties.props[propName]
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestNewService tests NewService.
func TestNewService(t *testing.T) {
	s := NewService()
	empty := &Service{}
	if reflect.DeepEqual(s, empty) {
		t.Errorf("got empty, want not empty")
	}
}
