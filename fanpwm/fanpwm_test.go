package fanpwm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/malkhamis/heatsink"
)

func TestNew_defaults(t *testing.T) {

	orig := deep.CompareUnexportedFields
	deep.CompareUnexportedFields = true
	defer func() { deep.CompareUnexportedFields = orig }()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	actualDr, err := New(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := actualDr.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	expectedDr := &Driver{
		name:        tmpFile.Name(),
		minSpeedVal: "0",
		maxSpeedVal: "255",
		pwmPeriod:   50 * time.Millisecond,
		wg:          sync.WaitGroup{},
	}
	expectedDr.wg.Add(1)

	if diff := deep.Equal(expectedDr, actualDr); diff != nil {
		t.Error("actual driver instance does not match expected\n", strings.Join(diff, "\n"))
	}
	if expected, actual := tmpFile.Name(), actualDr.Name(); expected != actual {
		t.Errorf("actual driver name does not match expected\nwant: %q\n got: %q", expected, actual)
	}
	if actualDr.devFile == nil {
		t.Error("device file was not set")
	}
}

func TestNew_validOptions(t *testing.T) {

	orig := deep.CompareUnexportedFields
	deep.CompareUnexportedFields = true
	defer func() { deep.CompareUnexportedFields = orig }()

	tmpFile, cleanupTmpFile := temporaryFile(t)
	defer cleanupTmpFile()

	actualDr, err := New(
		tmpFile.Name(),
		nil, // should be ignored
		OptName(t.Name()),
		OptMinSpeedValue("2"), OptMaxSpeedValue("8"),
		OptPeriodPWM(13*time.Microsecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := actualDr.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	expectedDr := &Driver{
		name:        t.Name(),
		minSpeedVal: "2",
		maxSpeedVal: "8",
		pwmPeriod:   13 * time.Microsecond,
		wg:          sync.WaitGroup{},
	}
	expectedDr.wg.Add(1)

	if diff := deep.Equal(expectedDr, actualDr); diff != nil {
		t.Error("actual driver instance does not match expected\n", strings.Join(diff, "\n"))
	}
	if expected, actual := t.Name(), actualDr.Name(); expected != actual {
		t.Errorf("actual driver name does not match expected\nwant: %q\n got: %q", expected, actual)
	}
	if actualDr.devFile == nil {
		t.Error("device file was not set")
	}
}

func TestNew_invalidOptions(t *testing.T) {

	orig := deep.CompareUnexportedFields
	deep.CompareUnexportedFields = true
	defer func() { deep.CompareUnexportedFields = orig }()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	actualDr, err := New(
		tmpFile.Name(),
		OptName(""),
		OptMinSpeedValue(""), OptMaxSpeedValue(""),
		OptPeriodPWM(-16),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := actualDr.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	expectedDr := &Driver{
		name:        tmpFile.Name(),
		minSpeedVal: "0",
		maxSpeedVal: "255",
		pwmPeriod:   50 * time.Millisecond,
		wg:          sync.WaitGroup{},
	}
	expectedDr.wg.Add(1)

	if diff := deep.Equal(expectedDr, actualDr); diff != nil {
		t.Error("actual driver instance does not match expected\n", strings.Join(diff, "\n"))
	}
	if expected, actual := tmpFile.Name(), actualDr.Name(); expected != actual {
		t.Errorf("actual driver name does not match expected\nwant: %q\n got: %q", expected, actual)
	}
	if actualDr.devFile == nil {
		t.Error("device file was not set")
	}
}

func TestNew_error(t *testing.T) {
	t.Parallel()

	_, err := New("/does/not/exist")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", os.ErrNotExist, err)
	}
}

func TestDriver_SetDutyCycle_errorSync(t *testing.T) {
	t.Parallel()

	driver, devFile := testDriver(t)
	defer func() {
		if err := driver.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	expectedErr := errors.New("simulated error")
	devFile.onSeekErrs = []error{expectedErr, nil, expectedErr}

	actualErr := driver.SetDutyCycle(0.5)
	if !errors.Is(actualErr, expectedErr) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", expectedErr, actualErr)
	}
	actualErr = driver.SetDutyCycle(0.5)
	if !errors.Is(actualErr, expectedErr) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", expectedErr, actualErr)
	}
}

func TestDriver_SetDutyCycle_errorTruncate(t *testing.T) {
	t.Parallel()

	driver, devFile := testDriver(t)
	defer func() {
		if err := driver.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	expectedErr := errors.New("simulated error")
	devFile.onTruncateErrs = []error{expectedErr, nil, expectedErr}

	actualErr := driver.SetDutyCycle(0.5)
	if !errors.Is(actualErr, expectedErr) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", expectedErr, actualErr)
	}
	actualErr = driver.SetDutyCycle(0.5)
	if !errors.Is(actualErr, expectedErr) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", expectedErr, actualErr)
	}
}

func TestDriver_SetDutyCycle_max_min(t *testing.T) {
	t.Parallel()

	tmpFile, cleanupTmpFile := temporaryFile(t)
	defer cleanupTmpFile()
	if _, err := tmpFile.WriteString("hello world"); err != nil {
		t.Fatal(err)
	}

	dr, err := New(tmpFile.Name(), OptMaxSpeedValue("176"), OptMinSpeedValue("9"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dr.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	err = dr.SetDutyCycle(1.0)
	if err != nil {
		t.Fatalf("expected no error setting fan speed to the maximum, got: %v", err)
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	expected := "176"
	actual, err := ioutil.ReadAll(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if expected != string(actual) {
		t.Errorf(
			"actual data written to the file does not match expected\nwant: %q\n got: %q",
			expected, actual,
		)
	}

	err = dr.SetDutyCycle(0.0)
	if err != nil {
		t.Fatalf("expected no error setting fan speed to the maximum, got: %v", err)
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	expected = "9"
	actual, err = ioutil.ReadAll(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if expected != string(actual) {
		t.Errorf(
			"actual data written to the file does not match expected\nwant: %q\n got: %q",
			expected, actual,
		)
	}
}

func TestDriver_concurrentUseAfterClose(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic using driver concurrently during close, got: %v", r)
		}
	}()

	driver, _ := testDriver(t)

	var wg sync.WaitGroup
	wg.Add(100)
	for range iter(100) {
		go func() {
			err := driver.SetDutyCycle(0.5)
			if err != nil && !errors.Is(err, heatsink.ErrFanDriverClosed) {
				t.Errorf("unexpected error\nwant: %v\n got: %v", heatsink.ErrFanDriverClosed, err)
			}
			wg.Done()
		}()
	}

	err := driver.Close()
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()
	err = driver.SetDutyCycle(0.5)
	if !errors.Is(err, heatsink.ErrFanDriverClosed) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", heatsink.ErrFanDriverClosed, err)
	}

	err = driver.Close()
	if !errors.Is(err, heatsink.ErrFanDriverClosed) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", heatsink.ErrFanDriverClosed, err)
	}
}

func TestDriver_Close_concurrently_ShouldNotPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic using closing driver concurrently, got: %v", r)
		}
	}()

	driver, _ := testDriver(t)

	var wg sync.WaitGroup
	wg.Add(200)

	for range iter(100) {
		go func() {
			driver.SetDutyCycle(0.5)
			wg.Done()
		}()
	}

	for range iter(100) {
		go func() {
			driver.Close()
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestDriver_Close_error_closingDevFile(t *testing.T) {
	t.Parallel()

	simErr := errors.New("simulated error1")
	driver, devFile := testDriver(t)
	devFile.onCloseErrs = []error{simErr}

	err := driver.Close()
	if !errors.Is(err, simErr) {
		t.Fatalf("unexpected error\nwant: %s\n got: %s", simErr, err)
	}
}

func TestDriver_Close_error_settingFanSpeedToMax(t *testing.T) {
	t.Parallel()

	simErr := errors.New("simulated error2")
	driver, devFile := testDriver(t)
	devFile.onWriteErrs = []error{simErr}

	err := driver.Close()
	if !errors.Is(err, simErr) {
		t.Fatalf("unexpected error\nwant: %s\n got: %s", simErr, err)
	}
}

func TestDriver_SetDutyCycle_unknownPanicsAreNotSilenced(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("unexpected panics should not be silenced becauase they are bugs")
		}
	}()

	driver, _ := testDriver(t)
	driver.devFile = nil
	err := driver.SetDutyCycle(0.5)
	fmt.Println(err)
}
