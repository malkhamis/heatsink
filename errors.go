package heatsink

import "errors"

// Sentinel errors that are wrapped and returned by this package
var (
	ErrControllerStopped  error = constErr("thermal controller is stopped")
	ErrFanDriverClosed    error = constErr("fan driver is closed")
	ErrThermoSensorClosed error = constErr("thermal sensor is closed")
)

// Sentinel errors that are defined to ease testing
var (
	errNoConfig = errors.New("no configuration given")
)

type constErr string

func (ce constErr) Error() string {
	return string(ce)
}
