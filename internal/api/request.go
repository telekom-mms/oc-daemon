package api

import (
	"net"

	log "github.com/sirupsen/logrus"
)

// Request is a request from a client
type Request struct {
	msg   *Message
	reply []byte
	err   string
	conn  net.Conn
}

// Type returns the type of the request
func (r *Request) Type() uint16 {
	return r.msg.Type
}

// Data returns the data in the API request
func (r *Request) Data() []byte {
	return r.msg.Value
}

// Reply sets the data in the reply for this request
func (r *Request) Reply(b []byte) {
	r.reply = b
}

// Error sets the error reply message for this request
func (r *Request) Error(msg string) {
	r.err = msg
}

// sendOK sends an ok message back to the client
func (r *Request) sendOK() {
	o := NewOK(r.reply)
	if err := WriteMessage(r.conn, o); err != nil {
		log.WithError(err).Error("Daemon message send error")
	}
}

// sendError sends an error back to the client
func (r *Request) sendError() {
	e := NewError([]byte(r.err))
	if err := WriteMessage(r.conn, e); err != nil {
		log.WithError(err).Error("Daemon message send error")
	}
}

// Close closes the API request
func (r *Request) Close() {
	defer func() {
		_ = r.conn.Close()
	}()

	if r.err != "" {
		r.sendError()
		return
	}
	r.sendOK()
}
