package api

import (
	"strconv"
	"time"
)

var (
	// SocketFile is the unix socket file
	SocketFile = "/run/oc-daemon/daemon.sock"

	// SocketOwner is the owner of the socket file
	SocketOwner = ""

	// SocketGroup is the group of the socket file
	SocketGroup = ""

	// SocketPermissions are the file permissions of the socket file
	SocketPermissions = "0700"

	// RequestTimeout is the timeout for an entire request/response
	// exchange initiated by a client
	RequestTimeout = 30 * time.Second
)

// Config is a server configuration
type Config struct {
	SocketFile        string
	SocketOwner       string
	SocketGroup       string
	SocketPermissions string
	RequestTimeout    time.Duration
}

// Valid returns whether server config is valid
func (c *Config) Valid() bool {
	if c == nil ||
		c.SocketFile == "" ||
		c.RequestTimeout < 0 {
		return false
	}
	if c.SocketPermissions != "" {
		perm, err := strconv.ParseUint(c.SocketPermissions, 8, 32)
		if err != nil {
			return false
		}
		if perm > 0777 {
			return false
		}
	}
	return true
}

// NewConfig returns a new server configuration
func NewConfig() *Config {
	return &Config{
		SocketFile:        SocketFile,
		SocketOwner:       SocketOwner,
		SocketGroup:       SocketGroup,
		SocketPermissions: SocketPermissions,
		RequestTimeout:    RequestTimeout,
	}
}
