package thermosense

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-test/deep"
	"github.com/malkhamis/heatsink"
)

func TestNew_defaults(t *testing.T) {
	t.Parallel()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	actual, err := New(tmpFile.Name())
	if err != nil {
		t.Fatalf("expected no error creating a new sensor, got: %v", err)
	}

	expected := &Sensor{
		name: tmpFile.Name(),
	}
	if diff := deep.Equal(expected, actual); diff != nil {
		t.Error("actual sensor does not match expected\n", strings.Join(diff, "\n"))
	}
	if actual.devFile == nil {
		t.Error("expected the device file to be initialized")
	}
}

func TestNew_validOptions(t *testing.T) {
	t.Parallel()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	actual, err := New(tmpFile.Name(), nil, OptName(t.Name()))
	if err != nil {
		t.Fatalf("expected no error creating a new sensor, got: %v", err)
	}

	expected := &Sensor{
		name: t.Name(),
	}
	if diff := deep.Equal(expected, actual); diff != nil {
		t.Error("actual sensor does not match expected\n", strings.Join(diff, "\n"))
	}

	if actual.devFile == nil {
		t.Error("expected the device file to be initialized")
	}

	if actual.Name() != t.Name() {
		t.Errorf(
			"actual sensor name does not match expected\nwant: %q\n got: %q",
			t.Name(), actual.Name(),
		)
	}
}

func TestNew_invalidOptions(t *testing.T) {
	t.Parallel()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	actual, err := New(tmpFile.Name(), OptName(""))
	if err != nil {
		t.Fatalf("expected no error creating a new sensor, got: %v", err)
	}

	expected := &Sensor{
		name: tmpFile.Name(),
	}
	if diff := deep.Equal(expected, actual); diff != nil {
		t.Error("actual sensor does not match expected\n", strings.Join(diff, "\n"))
	}

	if actual.devFile == nil {
		t.Error("expected the device file to be initialized")
	}

	if actual.Name() != tmpFile.Name() {
		t.Errorf(
			"actual sensor name does not match expected\nwant: %q\n got: %q",
			tmpFile.Name(), actual.Name(),
		)
	}
}

func TestNew_error(t *testing.T) {
	t.Parallel()

	_, err := New("/does/not/exist")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", os.ErrNotExist, err)
	}
}

func TestSensor_Temperature_errorSeek(t *testing.T) {
	t.Parallel()

	simErr := errors.New("simulated error")
	s := &Sensor{
		devFile: &fakeFile{onSeekErrs: []error{simErr}},
	}
	_, err := s.Temperature()
	if !errors.Is(err, simErr) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", simErr, err)
	}
}

func TestSensor_Temperature_errorScan(t *testing.T) {
	t.Parallel()

	simErr := errors.New("simulated error")
	s := &Sensor{
		devFile: &fakeFile{onReadErrs: []error{simErr}},
	}
	_, err := s.Temperature()
	if !errors.Is(err, simErr) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", simErr, err)
	}
}

func TestSensor_Close(t *testing.T) {
	t.Parallel()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	sensor, err := New(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	err = sensor.Close()
	if err != nil {
		t.Fatalf("expected no error closing sensor, got: %v", err)
	}
}

func TestSensor_Close_error(t *testing.T) {
	t.Parallel()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	sensor, err := New(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	simErr := errors.New("simulated error")
	sensor.devFile = &fakeFile{onCloseErrs: []error{simErr}}

	err = sensor.Close()
	if !errors.Is(err, simErr) {
		t.Fatalf("unexpected error closing sensor\nwant: %v\n got: %v", simErr, err)
	}
}

func TestSensor_Close_concurrently(t *testing.T) {
	t.Parallel()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	sensor, err := New(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 1000)
	for range iter(1000) {
		wg.Add(1)
		go func() {
			errCh <- sensor.Close()
			wg.Done()
		}()
	}
	wg.Wait()
	close(errCh)

	numErrThermoSensorClosed, numNilErrs := 0, 0
	for err := range errCh {
		if errors.Is(err, heatsink.ErrThermoSensorClosed) {
			numErrThermoSensorClosed++
		} else if err == nil {
			numNilErrs++
		}
	}

	if numNilErrs != 1 {
		t.Errorf("expected exactly one nil error, got: %d", numNilErrs)
	}
	if numErrThermoSensorClosed != 999 {
		t.Errorf("expected exactly 999 ErrThermoSensorClosed errors, got: %d", numErrThermoSensorClosed)
	}
}

func TestSensor_Close_concurrently_error(t *testing.T) {
	t.Parallel()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	sensor, err := New(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	simErr := errors.New("simulated error")
	sensor.devFile = &fakeFile{onCloseErrs: []error{simErr}}

	var wg sync.WaitGroup
	errCh := make(chan error, 1000)
	for range iter(1000) {
		wg.Add(1)
		go func() {
			errCh <- sensor.Close()
			wg.Done()
		}()
	}
	wg.Wait()
	close(errCh)

	numErrThermoSensorClosed, numSimErr := 0, 0
	for err := range errCh {
		if errors.Is(err, heatsink.ErrThermoSensorClosed) {
			numErrThermoSensorClosed++
		} else if errors.Is(err, simErr) {
			numSimErr++
		}
	}

	if numSimErr != 1 {
		t.Errorf("expected exactly one simulated error (%v), got: %d", simErr, numSimErr)
	}
	if numErrThermoSensorClosed != 999 {
		t.Errorf("expected exactly 999 ErrThermoSensorClosed errors, got: %d", numErrThermoSensorClosed)
	}
}
