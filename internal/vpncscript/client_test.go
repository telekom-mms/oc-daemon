package vpncscript

import (
	"path/filepath"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/daemon"
)

// TestRunClient tests runClient.
func TestRunClient(t *testing.T) {
	sockfile := filepath.Join(t.TempDir(), "sockfile")
	config := api.NewConfig()
	config.SocketFile = sockfile

	// without errors
	server := api.NewServer(config)
	go func() {
		for r := range server.Requests() {
			r.Close()
		}
	}()
	server.Start()
	runClient(sockfile, &daemon.VPNConfigUpdate{})
	server.Stop()

	// with error reply
	server = api.NewServer(config)
	go func() {
		for r := range server.Requests() {
			r.Error("test error")
			r.Close()
		}
	}()
	server.Start()
	runClient(sockfile, &daemon.VPNConfigUpdate{})
	server.Stop()
}
