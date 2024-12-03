package vpncscript

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/daemon"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// TestRunClient tests runClient.
func TestRunClient(t *testing.T) {
	sockfile := filepath.Join(t.TempDir(), "sockfile")
	config := daemoncfg.NewSocketServer()
	config.SocketFile = sockfile

	// without errors
	server := api.NewServer(config)
	go func() {
		for r := range server.Requests() {
			r.Close()
		}
	}()
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if err := runClient(sockfile, &daemon.VPNConfigUpdate{}); err != nil {
		t.Fatal(err)
	}
	server.Shutdown()
	server.Stop()

	// with error reply
	server = api.NewServer(config)
	go func() {
		for r := range server.Requests() {
			r.Error("test error")
			r.Close()
		}
	}()
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if err := runClient(sockfile, &daemon.VPNConfigUpdate{}); err != nil {
		t.Fatal(err)
	}
	server.Shutdown()
	server.Stop()

	// with "shutting down" error reply
	server = api.NewServer(config)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	server.Shutdown()
	if err := runClient(sockfile, &daemon.VPNConfigUpdate{}); err != nil {
		t.Fatal(err)
	}
	server.Stop()

	// helper for config update creation
	getConfUpdate := func(length int) *daemon.VPNConfigUpdate {
		exclude := "a.too.long.example.com"
		conf := vpnconfig.New()
		conf.Split.ExcludeDNS = []string{exclude}
		confUpdate := daemon.NewVPNConfigUpdate()
		confUpdate.Config = conf

		// check length
		b, err := confUpdate.JSON()
		if err != nil {
			t.Fatal(err)
		}
		n := length - len(b)

		// increase length to maximum
		exclude = strings.Repeat("a", n) + exclude
		conf.Split.ExcludeDNS = []string{exclude}

		return confUpdate
	}

	// test with varying payload lengths
	server = api.NewServer(config)
	go func() {
		for r := range server.Requests() {
			r.Close()
		}
	}()
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	for _, length := range []int{
		2048, 4096, 8192, 65536, 2097152,
	} {
		if err := runClient(sockfile, getConfUpdate(length)); err != nil {
			t.Errorf("length %d returned error: %v", length, err)
		}
	}
	server.Shutdown()
	server.Stop()
}
