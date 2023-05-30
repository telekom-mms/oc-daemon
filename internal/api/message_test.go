package api

import (
	"bytes"
	"log"
	"reflect"
	"testing"
)

// TestNewMessage tests NewMessage
func TestNewMessage(t *testing.T) {
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

// TestNewOK tests NewOK
func TestNewOK(t *testing.T) {
	msg := NewOK(nil)
	if msg.Type != TypeOK {
		t.Errorf("got %d, want %d", msg.Type, TypeOK)
	}
}

// TestNewError tests NewError
func TestNewError(t *testing.T) {
	msg := NewError(nil)
	if msg.Type != TypeError {
		t.Errorf("got %d, want %d", msg.Type, TypeError)
	}
}

// TestReadWriteMessage tests ReadMessage and WriteMessage
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
