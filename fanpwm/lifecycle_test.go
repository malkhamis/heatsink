package fanpwm

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/malkhamis/heatsink"
)

type lifeCycleTest struct {
	t           *testing.T
	driver      *Driver
	sampleCount int
	inDcRatio   float64
	outDurUp    time.Duration
	outDurDn    time.Duration
}

// TestDriver_lifeCycle ensures that calling SetDutyCycle multiple times with different
// params on the same instance does not result in deadlocks, panics, or errors and that pulses
// are generated as expected
func TestDriver_lifeCycle(t *testing.T) {
	ltc := newLifeCycleTest(t)
	ltc.Run()
}

func newLifeCycleTest(t *testing.T) *lifeCycleTest {

	// only needed so New() does not fail as we don't want
	// to manually init internal channels and workGroup here
	// and, therefore, we use New()
	tmpFile, cleanupTmpFile := temporaryFile(t)
	defer cleanupTmpFile()

	driver, err := New(
		tmpFile.Name(),
		OptMinSpeedValue("11"), OptMaxSpeedValue("99"),
		OptPeriodPWM(5*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	// in case it was used before it is initialized with fakeFile, it will panic
	driver.devFile = nil

	lc := &lifeCycleTest{
		t:           t,
		driver:      driver,
		sampleCount: 5,
		inDcRatio:   0.20,
		outDurUp:    1 * time.Millisecond,
		outDurDn:    4 * time.Millisecond,
	}
	return lc
}

func (lc *lifeCycleTest) Run() {
	lc.testDriver_SetDutyCycle_dcRatioBelowMin()
	lc.testDriver_SetDutyCycle_with_inDcRatio()
	lc.testDriver_SetDutyCycle_dcRatioAboveMax()
	lc.testDriver_Close_thenUse()
}

func (lc *lifeCycleTest) testDriver_SetDutyCycle_dcRatioAboveMax() {

	devFile := new(fakeFile)
	lc.driver.devFile = devFile
	defer func() { lc.driver.devFile = nil }()

	if err := lc.driver.SetDutyCycle(123.0); err != nil {
		lc.t.Fatalf("expected no error setting fan speed to value above 1.0, got: %v", err)
	}
	time.Sleep(time.Duration(2) * lc.driver.pwmPeriod)

	if len(devFile.actualWrites) > 2 {
		lc.t.Fatalf(
			"expected no more than two write-file ops when setting the speed to the max, got: %d",
			len(devFile.actualWrites),
		)
	}
}

func (lc *lifeCycleTest) testDriver_SetDutyCycle_dcRatioBelowMin() {

	devFile := new(fakeFile)
	lc.driver.devFile = devFile
	defer func() { lc.driver.devFile = nil }()

	if err := lc.driver.SetDutyCycle(-123.0); err != nil {
		lc.t.Fatalf("expected no error setting fan speed to value below 0.0, got: %v", err)
	}
	time.Sleep(time.Duration(2) * lc.driver.pwmPeriod)

	if len(devFile.actualWrites) > 2 {
		lc.t.Fatalf(
			"expected no more than two write-file ops when setting the speed to the min, got: %d",
			len(devFile.actualWrites),
		)
	}
}

func (lc *lifeCycleTest) testDriver_SetDutyCycle_with_inDcRatio() {

	var (
		totalGoodPulses, totalBadPulses int
		lastErr                         error
	)

	for range iter(10) {
		fakeDevFile := lc.collectPulseSamples()
		lc.testDriver_ensure_writesMinMaxSpeedValsCorrectly(fakeDevFile)
		goodPulses, badPulses, err := lc.testDriver_ensure_dutyCycle(fakeDevFile)
		if err != nil {
			lastErr = err
		}
		totalGoodPulses += goodPulses
		totalBadPulses += badPulses
	}

	if totalGoodPulses <= 0 {
		lc.t.Fatal("pwm accuracy is at 0%%")
	}
	pwmAccuracy := 1 - (float64(totalBadPulses) / float64(totalGoodPulses))
	if pwmAccuracy < 0.80 {
		lc.t.Fatalf("pwm accuracy '%.1f%%' is too low\nlast error: %s", pwmAccuracy*100.0, lastErr)
	}
}

func (lc *lifeCycleTest) testDriver_ensure_writesMinMaxSpeedValsCorrectly(devFile *fakeFile) {

	fileWrCount := 2 * lc.sampleCount
	for i := 0; i < fileWrCount; i++ {
		curWr := devFile.actualWrites[i]

		switch isOddSig, isEvenSig := i&1 == 1, i&1 == 0; {
		case isOddSig && string(curWr.val) != lc.driver.maxSpeedVal:
			lc.t.Fatalf(
				"expected odd-signal[%d] to be UP, i.e. value '%s' written to file, got: %q",
				i, lc.driver.maxSpeedVal, curWr.val,
			)
		case isEvenSig && string(curWr.val) != lc.driver.minSpeedVal:
			lc.t.Fatalf(
				"expected even-signal[%d] to be DOWN, i.e. value '%s' written to file, got: %q",
				i, lc.driver.maxSpeedVal, curWr.val,
			)
		case string(curWr.val) != lc.driver.maxSpeedVal && string(curWr.val) != lc.driver.minSpeedVal:
			lc.t.Fatalf(
				"expected signal[%d] written to file to be either UP('%s') or DOWN('%s'), got: %q",
				i, lc.driver.maxSpeedVal, lc.driver.minSpeedVal, curWr.val,
			)
		default: // all is good so far
		}
	}
}

func (lc *lifeCycleTest) testDriver_ensure_dutyCycle(devFile *fakeFile) (goodPulses, badPulses int, lastErr error) {

	fileWrCount := 2 * lc.sampleCount
	for i := 1; i < fileWrCount; /* == fileTrCount */ i++ {
		isOddSig := i&1 == 1
		prevWr := devFile.actualWrites[i-1]
		curTr := devFile.actualTruncates[i]

		if isOddSig {
			actualDurDn := curTr.ts.Sub(prevWr.ts)
			if lc.outDurDn == actualDurDn.Round(time.Millisecond) {
				goodPulses++
				continue
			}
			if lc.outDurDn == actualDurDn.Truncate(time.Millisecond) {
				goodPulses++
				continue
			}
			badPulses++
			lastErr = fmt.Errorf(
				"actual duration for DOWN-signal[%d] does not match expected\nwant: %s\n got: %s",
				i, lc.outDurDn, actualDurDn,
			)
			continue
		}

		actualDurUp := curTr.ts.Sub(prevWr.ts)
		if lc.outDurUp == actualDurUp.Round(time.Millisecond) {
			goodPulses++
			continue
		}
		if lc.outDurUp == actualDurUp.Truncate(time.Millisecond) {
			goodPulses++
			continue
		}
		lastErr = fmt.Errorf(
			"actual duration for UP-signal[%d] does not match expected\nwant: %s\n got: %s",
			i, lc.outDurUp, actualDurUp,
		)
		badPulses++
	}

	return
}

func (lc *lifeCycleTest) testDriver_Close_thenUse() {
	lc.driver.devFile = new(fakeFile)
	if err := lc.driver.Close(); err != nil {
		lc.t.Fatal(err)
	}
	if err := lc.driver.SetDutyCycle(0.3); !errors.Is(err, heatsink.ErrFanDriverClosed) {
		lc.t.Fatalf("unexpected error\nwant: %s\n got: %s", heatsink.ErrFanDriverClosed, err)
	}
}

func (lc *lifeCycleTest) collectPulseSamples() *fakeFile {

	devFile := new(fakeFile)
	lc.driver.devFile = devFile
	defer func() { lc.driver.devFile = nil }()

	if err := lc.driver.SetDutyCycle(lc.inDcRatio); err != nil {
		lc.t.Fatalf("expected no error setting fan speed, got: %v", err)
	}

	// collect pulse samples in the fake device file
	fileWrCount := 2 * lc.sampleCount
	fileTrCount := 2 * lc.sampleCount
	deadline := time.NewTimer(1 * time.Second).C
	for done := false; !done; {
		select {
		case <-deadline:
			lc.t.Fatalf("deadline exceeded waiting for %d pwm pulses", lc.sampleCount)
		default:
			devFile.mutex.Lock()
			if len(devFile.actualWrites) < fileWrCount {
				devFile.mutex.Unlock()
				continue
			}
			if len(devFile.actualTruncates) < fileTrCount {
				devFile.mutex.Unlock()
				continue
			}
			lc.driver.unsetCurPWM <- struct{}{}
			lc.driver.wg.Add(1)
			go func() { <-lc.driver.unsetCurPWM; lc.driver.wg.Done() }()
			devFile.actualWrites = devFile.actualWrites[:fileWrCount]
			devFile.actualTruncates = devFile.actualTruncates[:fileTrCount]
			done = true
			devFile.mutex.Unlock()
		}
	}

	return devFile
}
