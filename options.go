package heatsink

import (
	"time"

	"go.uber.org/zap"
)

// Option is used to pass optional parameters to the Heatsink factory function
type Option func(*Config, *Heatsink)

type fanResponse int

// Values that can be passed to option 'OptFanResponse'
const (
	FanResponsePowPi fanResponse = iota
	FanResponseLinear
)

// OptFanResponse controls how the fan speed is adjusted in response to temperature changes.
// The following mechanisms are supported:
//  FanResponseLinear: ideal for unpredictable temperatures -- dutyCucle(x) = x
//  FanResponsePowPi: ideal for unsustained temperature spikes (quiet) -- f(x) = x**π
//
// (default: FanResponsePowPi)
func OptFanResponse(meth fanResponse) Option {
	return func(config *Config, hs *Heatsink) {
		switch meth {
		case FanResponseLinear:
			hs.dcCalc = newDutyCyclerLinear(config.MinTemperature, config.MaxTemperature)
		default:
			hs.dcCalc = newDutyCyclerPowPi(config.MinTemperature, config.MaxTemperature)
		}
	}
}

// OptTemperatureCheckPeriod is the waiting time between temperature checks. If d is less than
// or equal to zero, it is set to the default value
//
// (default: 1 second)
func OptTemperatureCheckPeriod(d time.Duration) Option {
	return func(_ *Config, hs *Heatsink) {
		if d > 0 {
			hs.chkPeriod = d
		}
	}
}

// OptLogger is the logger that will be used by the heatsink. If logger is nil, it is set to the
// default value
//
// (default: noop logger)
func OptLogger(logger *zap.Logger) Option {
	return func(_ *Config, hs *Heatsink) {
		if logger == nil {
			logger = zap.NewNop()
		}
		hs.logger = logger
	}
}

// OptName sets the name of the heatsink. if name is empty, it is set to the default value
//
// (default: "heatsink/<fan.name>")
func OptName(name string) Option {
	return func(_ *Config, hs *Heatsink) {
		if name != "" {
			hs.name = name
		}
	}
}
