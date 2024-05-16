package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
)

const (
	// TokenLength is the length of the message token in bytes.
	TokenLength = 16
)

var (
	// token is the message token.
	token [TokenLength]byte
)

// Message types.
const (
	TypeNone = iota
	TypeOK
	TypeError
	TypeVPNConfigUpdate
	TypeUndefined
)

// Header is a message header.
type Header struct {
	Type   uint16
	Length uint32
	Token  [TokenLength]byte
}

// Message is an API message.
type Message struct {
	Header
	Value []byte
}

// NewMessage returns a new message with type t and payload p.
func NewMessage(t uint16, p []byte) *Message {
	return &Message{
		Header: Header{
			Type:   t,
			Length: uint32(len(p)),
			Token:  token,
		},
		Value: p,
	}
}

// NewOK returns a new OK message with payload p.
func NewOK(p []byte) *Message {
	return NewMessage(TypeOK, p)
}

// NewError returns a new error message with payload p.
func NewError(p []byte) *Message {
	return NewMessage(TypeError, p)
}

// ReadMessage returns the next message from r.
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
	if h.Token != token {
		return nil, errors.New("invalid message token")
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

// WriteMessage writes message m to r.
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

// GetToken generates and returns the message token as string. This should be
// used once on the server side before the server is started. Token must be
// passed to the client side.
func GetToken() (string, error) {
	_, err := rand.Read(token[:])
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token[:]), nil
}

// SetToken sets the message token from string. This should be used on the
// client side before sending requests to the server. Token must match token on
// the server side.
func SetToken(s string) error {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	copy(token[:], b)
	return nil
}
