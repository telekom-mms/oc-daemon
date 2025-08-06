package api

import (
	"bytes"
	"net"
	"testing"
)

// TestRequestType tests Type of Request.
func TestRequestType(t *testing.T) {
	req := &Request{
		msg: NewOK(nil),
	}
	if req.Type() != TypeOK {
		t.Errorf("got %d, want %d", req.Type(), TypeOK)
	}
}

// TestRequestData tests Data of Request.
func TestRequestData(t *testing.T) {
	// test with no data
	req := &Request{
		msg: NewOK(nil),
	}
	if req.Data() != nil {
		t.Errorf("got %v, want nil", req.Data())
	}

	// test with data
	data := []byte("some data")
	req = &Request{
		msg: NewOK(data),
	}
	if !bytes.Equal(req.Data(), data) {
		t.Errorf("got %s, want %s", req.Data(), data)
	}
}

// TestRequestReply tests Reply of Request.
func TestRequestReply(t *testing.T) {
	req := &Request{}
	reply := []byte("this is a reply")
	req.Reply(reply)
	if !bytes.Equal(req.reply, reply) {
		t.Errorf("got %s, want %s", req.reply, reply)
	}
}

// TestRequestError tests Error of Request.
func TestRequestError(t *testing.T) {
	req := &Request{}
	err := "this is an error"
	req.Error(err)
	if req.err != err {
		t.Errorf("got %s, want %s", req.err, err)
	}
}

// TestRequestCloseErrors tests Close of Request, errors.
func TestRequestCloseErrors(t *testing.T) {
	// helper for getting request with closed conn
	getReq := func() *Request {
		c1, c2 := net.Pipe()
		req := &Request{
			conn: c1,
		}
		if err := c2.Close(); err != nil {
			t.Fatal(err)
		}
		return req
	}

	// test OK with error
	req := getReq()
	req.Close()

	// test Error with error
	req = getReq()
	req.Error("fail")
	req.Close()
}

// TestRequestClose tests Close of Request.
func TestRequestClose(t *testing.T) {
	// test OK
	c1, c2 := net.Pipe()
	req := &Request{
		conn: c1,
	}
	go req.Close()
	msg, err := ReadMessage(c2)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != TypeOK {
		t.Errorf("got %d, want %d", msg.Type, TypeOK)
	}

	// test Error
	c1, c2 = net.Pipe()
	req = &Request{
		conn: c1,
	}
	req.Error("fail")
	go req.Close()
	msg, err = ReadMessage(c2)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != TypeError {
		t.Errorf("got %d, want %d", msg.Type, TypeError)
	}
}

// TestNewRequest tests NewRequest.
func TestNewRequest(t *testing.T) {
	c1, c2 := net.Pipe()
	defer func() {
		_ = c1.Close()
		_ = c2.Close()
	}()
	m := NewOK(nil)
	r := NewRequest(c1, m)
	if r.msg != m ||
		r.conn != c1 {
		t.Error("invalid request")
	}
}
