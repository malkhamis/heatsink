package fanpwm

import (
	"time"
)

// Option is used to pass optional parameters to the Driver factory function
type Option func(*Driver)

// OptPeriodPWM specifies the period of a PWM signal. If d <= 0, it is set to the default value
//
// (default: 50 millisecond)
func OptPeriodPWM(d time.Duration) Option {
	return func(dr *Driver) {
		if d <= 0 {
			d = 50 * time.Millisecond
		}
		dr.pwmPeriod = d
	}
}

// OptMinSpeedValue specifies the value which is written to the fan file to cause the fan to
// spin at the minimum speed. If val is empty, it is set to the default value
//
// (default: "0")
func OptMinSpeedValue(val string) Option {
	return func(dr *Driver) {
		if val == "" {
			val = "0"
		}
		dr.minSpeedVal = val
	}
}

// OptMaxSpeedValue specifies the value which is written to the fan file to cause the fan to
// spin at the maximum speed. If val is empty, it is set to the default value
//
// (default: "255")
func OptMaxSpeedValue(val string) Option {
	return func(dr *Driver) {
		if val == "" {
			val = "255"
		}
		dr.maxSpeedVal = val
	}
}

// OptName sets the name of the fan driver. if name is empty, it is set to the default value
//
// (default: filename)
func OptName(name string) Option {
	return func(dr *Driver) {
		if name != "" {
			dr.name = name
		}
	}
}
