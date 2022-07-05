package heatsink

import "errors"

// internal errors defined to ease testing
var (
	errNoFan     = errors.New("no fan given")
	errNoSensors = errors.New("no thermal sensors given")
	errNilSensor = errors.New("a given sensor cannot be nil")
	errBadTemps  = errors.New("maximum temperature must be greater than the minimum")
)

// Config is used to pass configuration to the heatsink factory function
type Config struct {
	// Fan is an instance that controls a physical fan, e.g. a fan attached to a CPU heatsink
	Fan FanDriver
	// Sensors are used to obtain temperature readings periodically
	Sensors []ThermoSensor
	// MinTemperature is the temperature below which the fan should spin at the minimum speed
	MinTemperature float64
	// MaxTemperature is the temperature above which the fan should spin at the maximum speed
	MaxTemperature float64
}

func (c *Config) validate() error {
	if c.Fan == nil {
		return errNoFan
	}
	if len(c.Sensors) == 0 {
		return errNoSensors
	}
	for _, sensor := range c.Sensors {
		if sensor == nil {
			return errNilSensor
		}
	}
	if c.MinTemperature >= c.MaxTemperature {
		return errBadTemps
	}
	return nil
}
