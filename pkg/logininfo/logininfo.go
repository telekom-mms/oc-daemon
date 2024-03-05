// Package logininfo contains the login information for OpenConnect.
package logininfo

import (
	"encoding/json"
	"strings"
)

// LoginInfo is login information for OpenConnect
type LoginInfo struct {
	Server      string
	Cookie      string
	Host        string
	ConnectURL  string
	Fingerprint string
	Resolve     string
}

// Copy returns a copy of LoginInfo
func (l *LoginInfo) Copy() *LoginInfo {
	if l == nil {
		return nil
	}
	cp := *l
	return &cp
}

// Valid returns if the login information is valid
func (l *LoginInfo) Valid() bool {
	if l == nil ||
		l.Server == "" ||
		l.Cookie == "" ||
		l.Host == "" ||
		l.Fingerprint == "" {
		// invalid
		return false
	}

	return true
}

// ParseLine extracts login information from line
func (l *LoginInfo) ParseLine(line string) {
	// get key, value pair from line
	s := strings.SplitN(line, "=", 2)
	if len(s) != 2 {
		return
	}
	key, value := s[0], s[1]

	// strip leading and trailing "'" character from value
	value = strings.TrimPrefix(value, "'")
	value = strings.TrimSuffix(value, "'")

	// get cookie, host, fingerprint
	switch key {
	case "COOKIE":
		l.Cookie = value
	case "HOST":
		l.Host = value
	case "CONNECT_URL":
		l.ConnectURL = value
	case "FINGERPRINT":
		l.Fingerprint = value
	case "RESOLVE":
		l.Resolve = value
	}
}

// jsonMarshal is json.Marshal for testing.
var jsonMarshal = json.Marshal

// JSON returns the login info as JSON
func (l *LoginInfo) JSON() ([]byte, error) {
	b, err := jsonMarshal(l)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// FromJSON parses and returns the login info in b
func FromJSON(b []byte) (*LoginInfo, error) {
	l := &LoginInfo{}
	err := json.Unmarshal(b, l)
	if err != nil {
		return nil, err
	}

	return l, nil
}
