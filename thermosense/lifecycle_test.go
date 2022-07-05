package thermosense

import (
	"errors"
	"os"
	"testing"

	"github.com/malkhamis/heatsink"
)

func TestSensor_lifeCycle(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		FileContent  string
		ExpectedTemp float64
	}{
		"with-fraction": {FileContent: "35124", ExpectedTemp: 35.124},
		"no-fraction":   {FileContent: "35000", ExpectedTemp: 35.0},
		"only-fraction": {FileContent: "00178", ExpectedTemp: 0.178},
		"zero":          {FileContent: "00000", ExpectedTemp: 0.0},
	}

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	sensor, err := New(tmpFile.Name())
	if err != nil {
		t.Fatalf("expected no error creating a new sensor, got: %v", err)
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {

			if err := tmpFile.Truncate(0); err != nil {
				t.Fatal(err)
			}
			if _, err := tmpFile.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			if _, err := tmpFile.WriteString(testCase.FileContent); err != nil {
				t.Fatal(err)
			}

			actualTemp, err := sensor.Temperature()
			if err != nil {
				t.Errorf("expected no error reading temperature, got: %v", err)
			}
			if testCase.ExpectedTemp != actualTemp {
				t.Errorf(
					"actual temperature does not match expected\nwant: %.3f\n got: %.3f",
					testCase.ExpectedTemp, actualTemp,
				)
			}
		})
	}

	err = sensor.Close()
	if err != nil {
		t.Fatalf("expected no error closing the sensor, got: %v", err)
	}
	_, err = sensor.Temperature()
	if !errors.Is(err, heatsink.ErrThermoSensorClosed) {
		t.Errorf(
			"unexpected error reading temperature from closed sensor\nwant: %v\n got: %v",
			heatsink.ErrThermoSensorClosed, err,
		)
	}

	_, err = sensor.devFile.Seek(0, 0)
	if !errors.Is(err, os.ErrClosed) {
		t.Errorf(
			"expected a call to Close() to also close the device file\nwant: %v\n got: %v",
			os.ErrClosed, err,
		)
	}

	err = sensor.Close()
	if !errors.Is(err, heatsink.ErrThermoSensorClosed) {
		t.Errorf(
			"unexpected error closing a closed sensor\nwant: %v\n got: %v",
			heatsink.ErrThermoSensorClosed, err,
		)
	}
}
