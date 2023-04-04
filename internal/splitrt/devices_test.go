package splitrt

import (
	"reflect"
	"testing"

	"github.com/T-Systems-MMS/oc-daemon/internal/devmon"
)

// getTestDevMonUpdate returns a DevMon Update for testing
func getTestDevMonUpdate() *devmon.Update {
	return &devmon.Update{
		Add:    true,
		Device: "tun0",
		Type:   "device",
		Index:  1,
	}
}

// TestDevicesAdd tests Add of Devices
func TestDevicesAdd(t *testing.T) {
	d := NewDevices()
	update := getTestDevMonUpdate()
	d.Add(update)
	want := update
	got := d.m[update.Index]
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestDevicesRemove tests Remove of Devices
func TestDevicesRemove(t *testing.T) {
	d := NewDevices()
	update := getTestDevMonUpdate()

	// add device
	d.Add(update)
	want := update
	got := d.m[update.Index]
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}

	// test removing device
	d.Remove(update)
	want = nil
	got = d.m[update.Index]
	if got != want {
		t.Errorf("got %p, want %p", got, want)
	}
}

// TestDevicesGetReal tests GetReal of Devices
func TestDevicesGetReal(t *testing.T) {
	d := NewDevices()
	realDev := getTestDevMonUpdate()
	virtDev := getTestDevMonUpdate()
	virtDev.Type = "virtual"
	virtDev.Index = 2
	d.Add(realDev)
	d.Add(virtDev)

	want := []int{realDev.Index}
	got := d.GetReal()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDevicesGetVirtual tests GetVirtual of Devices
func TestDevicesGetVirtual(t *testing.T) {
	d := NewDevices()
	realDev := getTestDevMonUpdate()
	virtDev := getTestDevMonUpdate()
	virtDev.Type = "virtual"
	virtDev.Index = 2
	d.Add(realDev)
	d.Add(virtDev)

	want := []int{virtDev.Index}
	got := d.GetVirtual()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDevicesGetAll tests GetAll of Devices
func TestDevicesGetAll(t *testing.T) {
	d := NewDevices()
	realDev := getTestDevMonUpdate()
	virtDev := getTestDevMonUpdate()
	virtDev.Type = "virtual"
	virtDev.Index = 2
	d.Add(realDev)
	d.Add(virtDev)

	want1 := []int{realDev.Index, virtDev.Index}
	want2 := []int{virtDev.Index, realDev.Index}
	got := d.GetAll()
	if !reflect.DeepEqual(got, want1) &&
		!reflect.DeepEqual(got, want2) {
		t.Errorf("got %v, want %v or %v", got, want1, want2)
	}
}

// TestNewDevices tests NewDevices
func TestNewDevices(t *testing.T) {
	d := NewDevices()
	if d.m == nil {
		t.Errorf("got nil, want != nil")
	}
}
