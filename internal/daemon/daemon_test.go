package daemon

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/api"
)

func TestDaemonHandleClientRequest(t *testing.T) {
	d := &Daemon{}
	c1, c2 := net.Pipe()
	r := api.NewRequest(c1, api.NewMessage(api.TypeVPNConfigUpdate, []byte{}))
	go d.handleClientRequest(r)
	c2.Close()
}

func TestNewDaemon(t *testing.T) {
	c := NewConfig()
	c.OpenConnect.XMLProfile = filepath.Join(t.TempDir(), "does-not-exist")
	d := NewDaemon(c)

	if d == nil {
		t.Fatal("daemon is nil")
	}

	if d.config != c {
		t.Fatal("wrong config")
	}

	for i, s := range []any{
		d.server,
		d.dbus,
		d.sleepmon,
		d.vpnsetup,
		d.runner,
		d.status,
		d.done,
		d.closed,
		d.profile,
		d.profmon,
	} {
		if s == nil {
			t.Errorf("%d: unexpected nil", i)
		}
	}
}
