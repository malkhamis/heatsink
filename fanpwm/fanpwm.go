// Package fanpwm provides an implementation of the heatsink.FanDriver interface
package fanpwm

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/malkhamis/heatsink"
)

// compile-time check for interface implementation and dependency inversion
var _ heatsink.FanDriver = (*Driver)(nil)

// Driver is a two-speed fan driver that is backed by an underlying file. It assumes that the
// physical fan controller can only be set to either a minimum or a maximum speed. Instances
// of this type are safe for concurrent use although it is not recommended to be used that way
type Driver struct {
	name        string
	devFile     wrOnlyFile `deep:"-"`
	minSpeedVal string
	maxSpeedVal string
	pwmPeriod   time.Duration
	// unsetCurPWM is used to send a stop signal to the currently running
	// go routine that performs the PWM as per a call to SetDutyCycle()
	unsetCurPWM chan struct{}
	closeSignal chan struct{}
	closeMutex  sync.Mutex
	isBusy      sync.Mutex
	wg          sync.WaitGroup
}

// New returns a new unstarted two-speed fan driver. The given file should typically represent a
// PWM-device and looks like '/sys/class/hwmon/hwmon[x]/pwm[y]'. The returned instance will
// have the exclusive write access to the given file and it will remain open until Close() is
// called. For details about options and defaults, see the documentation for type 'Option'
func New(filename string, options ...Option) (*Driver, error) {

	devFile, err := os.OpenFile(filename, os.O_EXCL|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	driver := &Driver{ // defaults
		name:        filename,
		minSpeedVal: "0",
		maxSpeedVal: "255",
		pwmPeriod:   50 * time.Millisecond,
		devFile:     devFile,
		unsetCurPWM: make(chan struct{}),
		closeSignal: make(chan struct{}),
	}
	for _, applyOption := range options {
		if applyOption == nil {
			continue
		}
		applyOption(driver)
	}

	// So SetDutyCycle() does not block on the very first call
	driver.startAsyncNopPWM()
	return driver, nil
}

// SetDutyCycle is a non-blocking method that uses the given duty cycle ratio to perform PWM.
// dcRatio must be in the range [0.0, 1.0]. If dcRatio is less than 0.0, it will be set to
// 0.0 and if it is greater than 1.0, it will be set to 1.0
func (dr *Driver) SetDutyCycle(dcRatio float64) (err error) {
	dr.isBusy.Lock()
	defer dr.isBusy.Unlock()

	if dr.isClosed() {
		return heatsink.ErrFanDriverClosed
	}
	dr.unsetCurPWM <- struct{}{}

	durationDn, durationUp, isFlatPulse := dr.calcDurations(dcRatio)
	err = dr.tryGenSinglePulse(durationDn, durationUp)
	if err != nil || isFlatPulse {
		dr.startAsyncNopPWM()
	}
	if err != nil {
		return fmt.Errorf("generating initial pulse: %w", err)
	}
	if isFlatPulse {
		return nil
	}

	dr.startAsyncPWM(durationDn, durationUp)
	return nil
}

// Close closes open files and releases held resources. If the driver is already closed, it
// returns heatsink.ErrFanDriverClosed
func (dr *Driver) Close() error {

	dr.closeMutex.Lock()
	defer dr.closeMutex.Unlock()
	if dr.isClosed() {
		return heatsink.ErrFanDriverClosed
	}
	close(dr.closeSignal)

	dr.isBusy.Lock()
	defer dr.isBusy.Unlock()
	dr.wg.Wait()
	close(dr.unsetCurPWM)

	err1 := dr.setSpeedMax()
	err2 := dr.devFile.Close()
	if err1 != nil {
		return fmt.Errorf("failed to set fan speed to max while closing driver: %w", err1)
	}
	if err2 != nil {
		return fmt.Errorf("failed to close device file while closing driver: %w", err2)
	}

	return nil
}

// Name returns the name of this fan driver
func (dr *Driver) Name() string {
	return dr.name
}
