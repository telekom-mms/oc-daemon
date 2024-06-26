package daemon

import (
	"log"
	"reflect"
	"testing"

	"github.com/telekom-mms/oc-daemon/pkg/vpnconfig"
)

// TestVPNConfigUpdateValid tests Valid of VPNConfigUpdate.
func TestVPNConfigUpdateValid(t *testing.T) {
	// test invalid
	u := NewVPNConfigUpdate()

	got := u.Valid()
	want := false
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test invalid disconnect
	u = NewVPNConfigUpdate()
	u.Reason = "disconnect"
	u.Config = vpnconfig.New()

	got = u.Valid()
	want = false
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test invalid connect, no config
	u = NewVPNConfigUpdate()
	u.Reason = "connect"

	got = u.Valid()
	want = false
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test invalid connect, invalid config
	u = NewVPNConfigUpdate()
	u.Reason = "connect"
	u.Config = vpnconfig.New()
	u.Config.Device.Name = "name is too long for a network device"

	got = u.Valid()
	want = false
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test valid disconnect
	u = NewVPNConfigUpdate()
	u.Reason = "disconnect"

	got = u.Valid()
	want = true
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test valid connect
	u = NewVPNConfigUpdate()
	u.Reason = "connect"
	u.Config = vpnconfig.New()

	got = u.Valid()
	want = true
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestVPNConfigUpdateJSON tests JSON and VPNConfigUpdateFromJSON of VPNConfigUpdate.
func TestVPNConfigUpdateJSON(t *testing.T) {
	updates := []*VPNConfigUpdate{}

	// empty
	u := NewVPNConfigUpdate()
	updates = append(updates, u)

	// valid disconnect
	u = NewVPNConfigUpdate()
	u.Reason = "disconnect"
	updates = append(updates, u)

	// valid connect
	u = NewVPNConfigUpdate()
	u.Reason = "connect"
	u.Config = vpnconfig.New()
	updates = append(updates, u)

	for _, u := range updates {
		log.Println(u)

		b, err := u.JSON()
		if err != nil {
			t.Fatal(err)
		}
		n, err := VPNConfigUpdateFromJSON(b)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(u, n) {
			t.Errorf("got %v, want %v", n, u)
		}
	}
}

// TestNewVPNConfigUpdate tests NewUpdate.
func TestNewVPNConfigUpdate(t *testing.T) {
	u := NewVPNConfigUpdate()
	if u == nil {
		t.Errorf("got nil, want != nil")
	}
}
