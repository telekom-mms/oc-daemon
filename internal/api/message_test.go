package api

import (
	"bytes"
	"encoding/base64"
	"errors"
	"log"
	"reflect"
	"testing"
)

// TestNewMessage tests NewMessage.
func TestNewMessage(t *testing.T) {
	// message types
	for _, typ := range []uint16{
		TypeNone,
		TypeOK,
		TypeError,
		TypeVPNConfigUpdate,
		TypeUndefined,
	} {
		log.Println("NewMessage with type", typ)
		msg := NewMessage(typ, nil)
		if msg.Type != typ {
			t.Errorf("got %d, want %d", msg.Type, typ)
		}
	}
}

// TestNewOK tests NewOK.
func TestNewOK(t *testing.T) {
	msg := NewOK(nil)
	if msg.Type != TypeOK {
		t.Errorf("got %d, want %d", msg.Type, TypeOK)
	}
}

// TestNewError tests NewError.
func TestNewError(t *testing.T) {
	msg := NewError(nil)
	if msg.Type != TypeError {
		t.Errorf("got %d, want %d", msg.Type, TypeError)
	}
}

// TestReadMessageErrors tests ReadMessage, errors.
func TestReadMessageErrors(t *testing.T) {
	buf := new(bytes.Buffer)

	// empty message
	if _, err := ReadMessage(buf); err == nil {
		t.Errorf("reading empty message should return error")
	}

	// invalid messages
	for _, msg := range []*Message{
		// invalid type
		{Header: Header{Type: TypeUndefined}},

		// short message
		{Header: Header{Type: TypeOK, Length: 4096}},

		// invalid token
		{Header: Header{Type: TypeOK, Token: [16]byte{1}}},
	} {
		if err := WriteMessage(buf, msg); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadMessage(buf); err == nil {
			t.Errorf("reading message %v should return error", msg)
		}
	}
}

// errWriter is a writer that returns an error after n writes.
type errWriter struct{ n int }

func (e *errWriter) Write(p []byte) (int, error) {
	if e.n > 0 {
		e.n--
		return len(p), nil
	}
	return 0, errors.New("test error")
}

// TestWriteMessageErrors tests WriteMessage, errors.
func TestWriteMessageErrors(t *testing.T) {
	msg := NewMessage(TypeOK, []byte("test message"))

	// header error
	w := &errWriter{n: 0}
	if err := WriteMessage(w, msg); err == nil {
		t.Error("write should return error")
	}

	// payload error
	w = &errWriter{n: 1}
	if err := WriteMessage(w, msg); err == nil {
		t.Error("write should return error")
	}
}

// TestReadWriteMessage tests ReadMessage and WriteMessage.
func TestReadWriteMessage(t *testing.T) {
	want := &Message{
		Header: Header{
			Type:   1,
			Length: 3,
		},
		Value: []byte{1, 2, 3},
	}
	buf := new(bytes.Buffer)

	// write message
	err := WriteMessage(buf, want)
	if err != nil {
		t.Error(err)
	}

	// read message
	got, err := ReadMessage(buf)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestGetSetToken tests GetToken and SetToken.
func TestGetSetToken(t *testing.T) {
	// reset token after tests
	defer func() { token = [TokenLength]byte{} }()

	// get new test token
	testToken, err := GetToken()
	if err != nil {
		t.Fatal(err)
	}
	s := base64.RawURLEncoding.EncodeToString(token[:])
	if testToken != s {
		t.Fatal("encoded token should match internal token")
	}

	// set token
	if err := SetToken(testToken); err != nil {
		t.Fatal(err)
	}

	// check token
	s = base64.RawURLEncoding.EncodeToString(token[:])
	if s != testToken {
		t.Fatal("internal token should match encoded token")
	}

	// setting invalid token
	if err := SetToken("not a valid encoded token!"); err == nil {
		t.Fatal("invalid token should return error")
	}
}
