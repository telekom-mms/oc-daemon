package vpncscript

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/api"
	"github.com/telekom-mms/oc-daemon/internal/daemon"
	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
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
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if err := runClient(sockfile, &daemon.VPNConfigUpdate{}); err != nil {
		t.Fatal(err)
	}
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

	// test with maximum payload length
	server = api.NewServer(config)
	go func() {
		for r := range server.Requests() {
			r.Close()
		}
	}()
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if err := runClient(sockfile, getConfUpdate(api.MaxPayloadLength)); err != nil {
		t.Fatal(err)
	}
	server.Stop()

	// test with more than maximum payload length
	server = api.NewServer(config)
	go func() {
		for r := range server.Requests() {
			r.Close()
		}
	}()
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	if err := runClient(sockfile, getConfUpdate(api.MaxPayloadLength+1)); err == nil {
		t.Fatal("too long message should return error")
	}
	server.Stop()
}
