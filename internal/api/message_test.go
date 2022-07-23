package api

import (
	"bytes"
	"reflect"
	"testing"
)

// TestReadWriteMessage tests reading and writing messages
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
