package splitrt

import (
	"github.com/telekom-mms/oc-daemon/internal/devmon"
)

// Devices is a set of devices
type Devices struct {
	m map[int]*devmon.Update
}

// Add adds device info in dev to devices
func (d *Devices) Add(dev *devmon.Update) {
	d.m[dev.Index] = dev
}

// Remove removes device info in dev from devices
func (d *Devices) Remove(dev *devmon.Update) {
	delete(d.m, dev.Index)
}

// getType returns device indexes of all devices that (do not) match typ
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

// GetReal returns device indexes of all real devices
func (d *Devices) GetReal() []int {
	return d.getType(true, "device")
}

// GetVirtual returns device indexes of all virtual devices
func (d *Devices) GetVirtual() []int {
	return d.getType(false, "device")
}

// GetAll returns all device indexes
func (d *Devices) GetAll() []int {
	return d.getType(false, "")
}

// NewDevices returns new Devices
func NewDevices() *Devices {
	return &Devices{
		m: make(map[int]*devmon.Update),
	}
}
