package api

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	// MaxPayloadLength is the maximum allowed length of a message payload
	MaxPayloadLength = 2048
)

// Message types
const (
	TypeNone = iota
	TypeOK
	TypeError
	TypeVPNConnect
	TypeVPNDisconnect
	TypeVPNQuery
	TypeVPNConfigUpdate
	TypeUndefined
)

// Header is a message header
type Header struct {
	Type   uint16
	Length uint16
}

// Message is an API message
type Message struct {
	Header
	Value []byte
}

// NewMessage returns a new message with type t and payload p
func NewMessage(t uint16, p []byte) *Message {
	if len(p) > MaxPayloadLength {
		return nil
	}
	return &Message{
		Header: Header{
			Type:   t,
			Length: uint16(len(p)),
		},
		Value: p,
	}
}

// NewOK returns a new OK message with payload p
func NewOK(p []byte) *Message {
	return NewMessage(TypeOK, p)
}

// NewError returns a new error message with payload p
func NewError(p []byte) *Message {
	return NewMessage(TypeError, p)
}

// ReadMessage returns the next message from r
func ReadMessage(r io.Reader) (*Message, error) {
	// read header
	h := &Header{}
	err := binary.Read(r, binary.LittleEndian, h)
	if err != nil {
		return nil, err
	}

	// check if message is valid
	if h.Type == TypeNone || h.Type >= TypeUndefined {
		return nil, errors.New("invalid message type")
	}
	if h.Length > MaxPayloadLength {
		return nil, errors.New("invalid message length")
	}

	// read payload
	b := make([]byte, h.Length)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}

	// return message
	m := &Message{
		Header: *h,
		Value:  b,
	}
	return m, nil
}

// WriteMessage writes message m to r
func WriteMessage(w io.Writer, m *Message) error {
	// write header
	err := binary.Write(w, binary.LittleEndian, m.Header)
	if err != nil {
		return err
	}

	// write payload
	if len(m.Value) == 0 {
		return nil
	}
	err = binary.Write(w, binary.LittleEndian, m.Value)
	if err != nil {
		return err
	}

	return nil
}
