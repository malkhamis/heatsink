package heatsink

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FanDriver controls the speed of a physical fan
type FanDriver interface {
	// SetDutyCycle set the fan speed according to the given duty cycle ratio. If the fan driver
	// is closed, it should return ErrFanDriverClosed
	SetDutyCycle(dcRatio float64) error
	// Name returns the name of this fan driver
	Name() string
	io.Closer
}

// ThermoSensor is a device that provides temperature readings
type ThermoSensor interface {
	// Temperature returns the current temperature reading of this sensor. If the sensor is
	// closed, it should return ErrThermoSensorClosed
	Temperature() (float64, error)
	// Name returns the name of this sensor
	Name() string
	io.Closer
}

// dutyCycler is a type that converts a temperature to a duty cycle ratio
type dutyCycler interface {
	ratio(temp float64) (dcRatio float64)
}

// Heatsink represents a physical heatsink package with thermal monitor and control
type Heatsink struct {
	name       string
	sensors    []ThermoSensor
	fan        FanDriver
	dcCalc     dutyCycler
	chkPeriod  time.Duration
	isStopped  chan struct{}
	closeMutex sync.Mutex
	logger     *zap.Logger
}

// New returns a new heatsink instance. For details about configs, options, and
// defaults, see the documentation for types 'Config' and 'Option'
func New(config *Config, options ...Option) (*Heatsink, error) {

	if config == nil {
		return nil, errNoConfig
	}
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	hs := &Heatsink{
		name:      "heatsink/" + config.Fan.Name(),
		dcCalc:    newDutyCyclerPowPi(config.MinTemperature, config.MaxTemperature),
		chkPeriod: 1 * time.Second,
		fan:       config.Fan,
		sensors:   append([]ThermoSensor{}, config.Sensors...),
		isStopped: make(chan struct{}),
		logger:    zap.NewNop(),
	}
	for _, applyOption := range options {
		if applyOption == nil {
			continue
		}
		applyOption(config, hs)
	}

	return hs, nil
}

// StartThermalControl continuously monitors temperatures and adjusts the heatsink fan. If the
// heatsink is stopped, it returns ErrControllerStopped. It always returns a non-nil error
func (hs *Heatsink) StartThermalControl() error {

	defer func() {
		cerr := hs.StopThermalControl()
		if errors.Is(cerr, ErrControllerStopped) {
			return
		}
		if cerr != nil {
			hs.logger.Error(
				"failed to properly stop thermal control after encountering an error",
				zap.Error(cerr), zap.String("heatsink_name", hs.name),
			)
		}
		hs.logger.Info("stopped thermal control", zap.String("heatsink_name", hs.name))
	}()

	hs.logger.Info(
		"started thermal control",
		zap.String("heatsink_name", hs.name),
	)

loop:
	for ; ; time.Sleep(hs.chkPeriod) {

		select {
		case <-hs.isStopped:
			break loop
		default:
		}

		temp, err := hs.maxCoreTemp()
		if err != nil {
			return fmt.Errorf("determining max core temperature: %w", err)
		}

		dcRatio := hs.dcCalc.ratio(temp)
		err = hs.fan.SetDutyCycle(dcRatio)
		if err != nil {
			return fmt.Errorf("setting fan's duty cycle: %w", err)
		}
	}

	return ErrControllerStopped
}

// StopThermalControl stops monitoring temperatures, controlling fan speed, andreleases all
// held resources. It safe to call it multiple times by multiple go routines as subsequent
// calls will return ErrControllerStopped with no side effects
func (hs *Heatsink) StopThermalControl() error {
	hs.closeMutex.Lock()
	defer hs.closeMutex.Unlock()

	select {
	case <-hs.isStopped:
		return ErrControllerStopped
	default:
		close(hs.isStopped)
	}

	var errs multiErrs
	if err := hs.fan.Close(); err != nil {
		err = fmt.Errorf("error closing fan: %w", err)
		errs = append(errs, err)
	}
	for _, sensor := range hs.sensors {
		if err := sensor.Close(); err != nil {
			err = fmt.Errorf("error closing sensor: %w", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (hs *Heatsink) maxCoreTemp() (max float64, err error) {

	max = math.SmallestNonzeroFloat64
	var errs multiErrs

	for _, thermoSensor := range hs.sensors {
		temp, err := thermoSensor.Temperature()
		if err != nil {
			err = fmt.Errorf("thermo sensor '%s': %w", thermoSensor.Name(), err)
			errs = append(errs, err)
			continue
		}
		if temp > max {
			max = temp
		}
	}

	if len(errs) == len(hs.sensors) {
		return math.MaxFloat64, errs
	}
	for _, e := range errs {
		hs.logger.Error("failed to read temperature", zap.Error(e))
	}

	return max, nil
}

type multiErrs []error

func (me multiErrs) Error() string {
	if len(me) == 1 {
		return me[0].Error()
	}
	var sb strings.Builder
	for _, err := range me {
		fmt.Fprintf(&sb, "\n  - %s", err)
	}
	return sb.String()
}
