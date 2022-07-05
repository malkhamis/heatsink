package fanpwm

import (
	"fmt"
	"io"
	"time"
)

type wrOnlyFile interface {
	Truncate(int64) error
	io.Seeker
	io.WriteCloser
}

func (dr *Driver) tryGenSinglePulse(dn, up time.Duration) error {
	// We start by trying to set the min speed first because if the fan is
	// spinning near the min speed and this call is for setting the speed to
	// the min, the human ears may not notice a momentary spike in fan noise.
	// If it happens that the fan is spinning near the max speed and this call
	// is for setting the speed to the max, it would be so noisy for the human
	// ears to care about a momentary reduction in fan noise.
	err := dr.setSpeedMin()
	if err != nil {
		return fmt.Errorf("failed to set min speed: %w", err)
	} else if dn == dr.pwmPeriod {
		return nil
	}
	time.Sleep(dn)

	err = dr.setSpeedMax()
	if err != nil {
		return fmt.Errorf("failed to set max speed: %w", err)
	} else if up == dr.pwmPeriod {
		return nil
	}
	time.Sleep(up)

	return nil
}

func (dr *Driver) startAsyncNopPWM() {
	dr.wg.Add(1)
	go func() {
		defer dr.wg.Done()
		select {
		case <-dr.unsetCurPWM:
			return
		case <-dr.closeSignal:
			return
		}
	}()
}

func (dr *Driver) startAsyncPWM(dn, up time.Duration) {
	dr.wg.Add(1)
	go func() {
		defer dr.wg.Done()
		for {
			// errors are ignore for the following reasons:
			//  - intermitten failures are not worth the effort
			//  - persistent failures indicate there is a bigger problem
			//  - the go routine will keep trying anyway
			//  - expectations are SetDutyCycle() will be called again and
			//    an error will be returned there if it is persistent
			_ = dr.setSpeedMin()
			time.Sleep(dn)
			_ = dr.setSpeedMax()
			time.Sleep(up)
			select {
			case <-dr.unsetCurPWM:
				return
			case <-dr.closeSignal:
				return
			default: // continue
			}
		}
	}()
}

func (dr *Driver) isClosed() bool {
	select {
	case <-dr.closeSignal:
		return true
	default:
		return false
	}
}

func (dr *Driver) calcDurations(dcRatio float64) (dn, up time.Duration, isFlatPulse bool) {
	if dcRatio > 1.0 {
		dcRatio = 1.0
	} else if dcRatio < 0.0 {
		dcRatio = 0.0
	}
	up = time.Duration(dcRatio * float64(dr.pwmPeriod))
	dn = dr.pwmPeriod - up
	isFlatPulse = (up == dr.pwmPeriod) || (dn == dr.pwmPeriod)
	return
}

func (dr *Driver) setSpeedMax() error {
	if _, err := dr.devFile.Seek(0, 0); err != nil {
		return err
	}
	if err := dr.devFile.Truncate(0); err != nil {
		return err
	}
	_, err := dr.devFile.Write([]byte(dr.maxSpeedVal))
	return err
}

func (dr *Driver) setSpeedMin() error {
	if _, err := dr.devFile.Seek(0, 0); err != nil {
		return err
	}
	if err := dr.devFile.Truncate(0); err != nil {
		return err
	}
	_, err := dr.devFile.Write([]byte(dr.minSpeedVal))
	return err
}
