// Package thermosense provides an implementation of the heatsink.ThermoSensor interface
package thermosense

import (
	"os"
	"sync"

	"github.com/malkhamis/heatsink"
)

// compile-time check for interface implementation and dependency inversion
var _ heatsink.ThermoSensor = (*Sensor)(nil)

// Sensor represents a thermal sensor that is backed by an underlying file. It assumes that the
// temperature is an integer value whose unit of measurement is millidegree celsius. Instances
// of this type are safe for concurrent use
type Sensor struct {
	name    string
	devFile rdOnlyFile `deep:"-"`
	mutex   sync.Mutex
	closed  bool
}

// New returns a new thermal sensor. The given file should typically represent a digital
// temperature sensor and looks like '/sys/class/hwmon/hwmon[x]/temp[y]_input'. The given file
// will remain open until Close() is called. For details about options and defaults, see the
// documentation for type 'Option'
func New(filename string, options ...Option) (*Sensor, error) {

	devFile, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	sensor := &Sensor{
		name:    filename,
		devFile: devFile,
	}
	for _, applyOption := range options {
		if applyOption == nil {
			continue
		}
		applyOption(sensor)
	}

	return sensor, nil
}

// Temperature returns the current temperature as well as any error encountered. If the sensor
// is closed, it returns heatsink.ErrThermoSensorClosed. Concurrent calls to this method by multiple
// go routines will be serialized
func (s *Sensor) Temperature() (float64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.temperature()
}

// Close closes this sensor and releases held resources. If the sensor was previously closed, it
// returns heatsink.ErrThermoSensorClosed
func (s *Sensor) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.close()
}

// Name returns the name of this sensor
func (s *Sensor) Name() string {
	return s.name
}
