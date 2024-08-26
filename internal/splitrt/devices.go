package splitrt

import (
	"sync"

	"github.com/telekom-mms/oc-daemon/internal/devmon"
)

// Devices is a set of devices.
type Devices struct {
	sync.Mutex
	m map[int]*devmon.Update
}

// Add adds device info in dev to devices.
func (d *Devices) Add(dev *devmon.Update) {
	d.Lock()
	defer d.Unlock()

	d.m[dev.Index] = dev
}

// Remove removes device info in dev from devices.
func (d *Devices) Remove(dev *devmon.Update) {
	d.Lock()
	defer d.Unlock()

	delete(d.m, dev.Index)
}

// getType returns device indexes of all devices that (do not) match typ.
func (d *Devices) getType(match bool, typ string) (indexes []int) {
	for _, v := range d.m {
		if match && v.Type != typ {
			continue
		}
		if !match && v.Type == typ {
			continue
		}
		indexes = append(indexes, v.Index)
	}
	return
}

// GetReal returns device indexes of all real devices.
func (d *Devices) GetReal() []int {
	d.Lock()
	defer d.Unlock()

	return d.getType(true, "device")
}

// GetVirtual returns device indexes of all virtual devices.
func (d *Devices) GetVirtual() []int {
	d.Lock()
	defer d.Unlock()

	return d.getType(false, "device")
}

// GetAll returns all device indexes.
func (d *Devices) GetAll() []int {
	d.Lock()
	defer d.Unlock()

	return d.getType(false, "")
}

// List returns all devices.
func (d *Devices) List() []*devmon.Update {
	d.Lock()
	defer d.Unlock()

	var devices []*devmon.Update
	for _, v := range d.m {
		device := *v
		devices = append(devices, &device)
	}
	return devices
}

// NewDevices returns new Devices.
func NewDevices() *Devices {
	return &Devices{
		m: make(map[int]*devmon.Update),
	}
}
