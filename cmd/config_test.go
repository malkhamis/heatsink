package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/malkhamis/heatsink"
	"github.com/malkhamis/heatsink/fanpwm"
	"github.com/malkhamis/heatsink/thermosense"
	"go.uber.org/zap"
)

func Test_config_newHeatsinks(t *testing.T) {

	orig := deep.CompareUnexportedFields
	deep.CompareUnexportedFields = true
	defer func() { deep.CompareUnexportedFields = orig }()

	tmpDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("%s: error removing temporary test directory: %s", tmpDir, err)
		}
	}()

	tmpDirFans, err := ioutil.TempDir(tmpDir, "hwmon")
	if err != nil {
		t.Fatal(err)
	}
	fanFile1, err := ioutil.TempFile(tmpDirFans, "pwm1")
	if err != nil {
		t.Fatal(err)
	}
	fanFile2, err := ioutil.TempFile(tmpDirFans, "pwm2")
	if err != nil {
		t.Fatal(err)
	}

	tmpDirSensorGroup0, err := ioutil.TempDir(tmpDir, "coretemp.0")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ioutil.TempFile(tmpDirSensorGroup0, "garbage") // should be ignored
	if err != nil {
		t.Fatal(err)
	}
	_, err = ioutil.TempFile(tmpDirSensorGroup0, "temp1_input") // should be ignored
	if err != nil {
		t.Fatal(err)
	}
	sensorFile1, err := ioutil.TempFile(tmpDirSensorGroup0, "temp2_input")
	if err != nil {
		t.Fatal(err)
	}
	sensorFile2, err := ioutil.TempFile(tmpDirSensorGroup0, "temp3_input")
	if err != nil {
		t.Fatal(err)
	}

	tmpDirSensorGroup1, err := ioutil.TempDir(tmpDir, "coretemp.1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ioutil.TempFile(tmpDirSensorGroup1, "garbage2")
	if err != nil {
		t.Fatal(err)
	}
	sensorFile3, err := ioutil.TempFile(tmpDirSensorGroup1, "temp1_input")
	if err != nil {
		t.Fatal(err)
	}

	fan1Glob := filepath.Join(tmpDir, "hwmon*", "pwm1*")
	sensorGroup1Glob := filepath.Join(tmpDir, "coretemp.0*", "temp[2-3]_input*")

	fan2Glob := filepath.Join(tmpDir, "hwmon*", "pwm2*")
	sensorGroup2Glob := filepath.Join(tmpDir, "coretemp.1*", "temp[1-9]_input*")

	jsonData := strings.NewReader(fmt.Sprintf(`
		    {
		      "heatsinks": [

		        {
		          "name":"heatsink/1",
		          "min_temp": 30,
		          "max_temp": 50,
		          "temp_check_period": "3s",
		          "fan": {
		            "name": "fan/1",
		            "path_glob": %q,
		            "pwm_period": "22ms",
		            "min_speed_value": "10",
		            "max_speed_value": "200",
								"response_type": "PowPi"
		          },
		          "sensor_path_globs": [%q]
		        },

		        {
		          "name":"heatsink/2",
		          "min_temp": 13,
		          "max_temp": 31,
		          "temp_check_period": "7s",
		          "fan": {
		            "name": "fan/2",
		            "path_glob": %q,
		            "pwm_period": "44ms",
		            "min_speed_value": "34",
		            "max_speed_value": "145",
								"response_type": "linear"
		          },
		          "sensor_path_globs": [%q]
		        }

		      ]
		    }
		  `,
		fan1Glob, sensorGroup1Glob, // heatsink/1 config
		fan2Glob, sensorGroup2Glob, // heatsink/2 config
	))

	logger := zap.NewNop()

	// expected heatsink/1
	fan1, err := fanpwm.New(
		fanFile1.Name(),
		fanpwm.OptName("fan/1"),
		fanpwm.OptPeriodPWM(22*time.Millisecond),
		fanpwm.OptMinSpeedValue("10"),
		fanpwm.OptMaxSpeedValue("200"),
	)
	if err != nil {
		t.Fatal(err)
	}
	sensor1, err := thermosense.New(sensorFile1.Name())
	if err != nil {
		t.Fatal(err)
	}
	sensor2, err := thermosense.New(sensorFile2.Name())
	if err != nil {
		t.Fatal(err)
	}
	heatsink1, err := heatsink.New(
		&heatsink.Config{
			Fan:            fan1,
			Sensors:        []heatsink.ThermoSensor{sensor1, sensor2},
			MinTemperature: 30,
			MaxTemperature: 50,
		},
		heatsink.OptName("heatsink/1"),
		heatsink.OptFanResponse(heatsink.FanResponsePowPi),
		heatsink.OptTemperatureCheckPeriod(3*time.Second),
		heatsink.OptLogger(logger),
	)
	if err != nil {
		t.Fatal(err)
	}

	// expected heatsink/2
	fan2, err := fanpwm.New(
		fanFile2.Name(),
		fanpwm.OptName("fan/2"),
		fanpwm.OptPeriodPWM(44*time.Millisecond),
		fanpwm.OptMinSpeedValue("34"),
		fanpwm.OptMaxSpeedValue("145"),
	)
	if err != nil {
		t.Fatal(err)
	}
	sensor3, err := thermosense.New(sensorFile3.Name())
	if err != nil {
		t.Fatal(err)
	}
	heatsink2, err := heatsink.New(
		&heatsink.Config{
			Fan:            fan2,
			Sensors:        []heatsink.ThermoSensor{sensor3},
			MinTemperature: 13,
			MaxTemperature: 31,
		},
		heatsink.OptName("heatsink/2"),
		heatsink.OptFanResponse(heatsink.FanResponseLinear),
		heatsink.OptTemperatureCheckPeriod(7*time.Second),
		heatsink.OptLogger(logger),
	)
	if err != nil {
		t.Fatal(err)
	}

	expected := []*heatsink.Heatsink{heatsink1, heatsink2}

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := cfg.newHeatsinks()
	if err != nil {
		t.Fatalf("expected no error building heatsinks from json config, got: %v", err)
	}

	if len(actual) != 2 {
		t.Fatalf("expected 2 heatsinks, got: %d", len(actual))
	}
	if diff := deep.Equal(expected, actual); diff != nil {
		t.Fatal("actual deserialized heatsinks doesn't match expected\n", strings.Join(diff, "\n"))
	}
}

func Test_newConfig_errNilReader(t *testing.T) {
	t.Parallel()

	_, err := newConfig(nil, nil)
	if !errors.Is(err, errNoJsonConfig) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", errNoJsonConfig, err)
	}
}

func Test_newConfig_setsDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := newConfig(strings.NewReader(`{"heatsinks":[{"fan":{}}]}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	expected, actual := "PowPi", cfg.Heatsinks[0].Fan.RespType
	if actual != expected {
		t.Fatalf(
			"expected fan response type to be set to '%s' if not given, got: '%s'",
			expected, actual,
		)
	}
}

func Test_newConfig_errBadJson(t *testing.T) {
	t.Parallel()

	_, err := newConfig(strings.NewReader(`{ bad json`), nil)
	var expected *json.SyntaxError
	if ok := errors.As(err, &expected); !ok {
		t.Fatalf("unexpected error type\nwant: %T\n got: %T", expected, err)
	}
}

func Test_newConfig_errNoHeatsinkConfig(t *testing.T) {
	t.Parallel()

	_, err := newConfig(strings.NewReader(`{}`), nil)
	if !errors.Is(err, errNoHeatsinkConfig) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", errNoHeatsinkConfig, err)
	}
}

func Test_config_newHeatsinks_error_tempChkPeriod_wrongType(t *testing.T) {
	t.Parallel()

	jsonData := strings.NewReader(`
    {
      "heatsinks": [
        {
          "temp_check_period": "3 s"
        }
      ]
    }
  `)

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cfg.newHeatsinks()
	if !errors.Is(err, errBadDuration) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", errBadDuration, err)
	}
}

func Test_config_newHeatsinks_errorCreatingSensor_badGlob(t *testing.T) {
	t.Parallel()

	jsonData := strings.NewReader(`
    {
      "heatsinks": [
        {
          "sensor_path_globs":["/tmp/[[BAD PATTERN"]
        }
      ]
    }
  `)

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cfg.newHeatsinks()
	if !errors.Is(err, filepath.ErrBadPattern) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", filepath.ErrBadPattern, err)
	}
}

func Test_config_newHeatsinks_errorCreatingSensor_noGlobMatches(t *testing.T) {
	t.Parallel()

	jsonData := strings.NewReader(`
    {
      "heatsinks": [
        {
          "sensor_path_globs": ["/tmp/file/not/exists"]
        }
      ]
    }
  `)

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cfg.newHeatsinks()
	if !errors.Is(err, errGlobNoMatches) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", errGlobNoMatches, err)
	}
}

func Test_config_newHeatsinks_errorCreatingFan_badGlob(t *testing.T) {
	t.Parallel()

	sensorFile, cleanup := temporaryFile(t)
	defer cleanup()

	jsonData := strings.NewReader(fmt.Sprintf(`
    {
      "heatsinks": [
        {
          "max_temp": 10,
          "fan": {
            "path_glob": "/tmp/[[BAD PATTERN"
          },
          "sensor_path_globs": [%q]
        }
      ]
    }
  `, sensorFile.Name(),
	))

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cfg.newHeatsinks()
	if !errors.Is(err, filepath.ErrBadPattern) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", filepath.ErrBadPattern, err)
	}
}

func Test_config_newHeatsinks_errorCreatingFan_noGlobMatches(t *testing.T) {
	t.Parallel()

	sensorFile, cleanup := temporaryFile(t)
	defer cleanup()

	jsonData := strings.NewReader(fmt.Sprintf(`
    {
      "heatsinks": [
        {
          "max_temp": 10,
          "fan": {
            "path_glob": "/tmp/file/not/exist"
          },
          "sensor_path_globs": [%q]
        }
      ]
    }
  `, sensorFile.Name(),
	))

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cfg.newHeatsinks()
	if !errors.Is(err, errGlobNoMatches) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", errGlobNoMatches, err)
	}
}

func Test_config_newHeatsinks_errorCreatingFan_globeTooManyMatches(t *testing.T) {
	t.Parallel()

	f1, cleanup := temporaryFile(t)
	defer cleanup()
	f1, cleanup = temporaryFile(t)
	defer cleanup()
	glob := filepath.Join(filepath.Dir(f1.Name()), "*")

	sensorFile, cleanup := temporaryFile(t)
	defer cleanup()

	jsonData := strings.NewReader(fmt.Sprintf(`
    {
      "heatsinks": [
        {
          "max_temp": 10,
          "fan": {
            "path_glob": %q
          },
          "sensor_path_globs": [%q]
        }
      ]
    }
  `, glob, sensorFile.Name(),
	))

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cfg.newHeatsinks()
	if !errors.Is(err, errGlobTooManyMatches) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", errGlobTooManyMatches, err)
	}
}

func Test_config_newHeatsinks_fan_pwmPeriod_wrongType(t *testing.T) {
	t.Parallel()

	sensorFile, cleanup := temporaryFile(t)
	defer cleanup()

	jsonData := strings.NewReader(fmt.Sprintf(`
    {
      "heatsinks": [
        {
          "max_temp": 10,
          "fan": {
            "pwm_period": "3 s"
          },
          "sensor_path_globs": [%q]
        }
      ]
    }
  `, sensorFile.Name(),
	))

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cfg.newHeatsinks()
	if !errors.Is(err, errBadDuration) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", errBadDuration, err)
	}
}

func Test_config_newHeatsinks_errorCreatingSensor(t *testing.T) {
	t.Parallel()

	badSensorFile, err := os.OpenFile(
		filepath.Join(os.TempDir(), t.Name()),
		os.O_CREATE,
		os.ModeSticky, // trouble maker
	)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(badSensorFile.Name())

	jsonData := strings.NewReader(fmt.Sprintf(`
		{
		  "heatsinks": [

		    {
		      "name":"heatsink/1",
		      "min_temp": 30,
		      "max_temp": 50,
		      "temp_check_period": "3s",
		      "sensor_path_globs": [%q]
		    }

		  ]
		}
	`, badSensorFile.Name(),
	))

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.newHeatsinks()
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf(
			"unexpected error creating a heatsink with a non-regular sensor file\nwant: %v\n got: %v",
			os.ErrPermission, err,
		)
	}

}

func Test_config_newHeatsinks_errorCreatingFan(t *testing.T) {
	t.Parallel()

	badFanFile, err := os.OpenFile(
		filepath.Join(os.TempDir(), t.Name()),
		os.O_CREATE,
		os.ModeSticky, // trouble maker
	)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(badFanFile.Name())

	sensorFile, cleanup := temporaryFile(t)
	defer cleanup()

	jsonData := strings.NewReader(fmt.Sprintf(`
		{
		  "heatsinks": [

		    {
		      "name":"heatsink/1",
		      "min_temp": 30,
		      "max_temp": 50,
		      "temp_check_period": "3s",
		      "sensor_path_globs": [%q],
					"fan": {
						"path_glob": %q,
						"pwm_period": "22ms",
						"min_speed_value": "10",
						"max_speed_value": "200"
					}
		    }

		  ]
		}
	`, sensorFile.Name(), badFanFile.Name(),
	))

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.newHeatsinks()
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf(
			"unexpected error creating a heatsink with a non-regular fan file\nwant: %v\n got: %v",
			os.ErrPermission, err,
		)
	}

}

func Test_config_newHeatsinks_error_badFanResponseType(t *testing.T) {
	t.Parallel()

	sensorFile, cleanup := temporaryFile(t)
	defer cleanup()

	fanFile, cleanup := temporaryFile(t)
	defer cleanup()

	jsonData := strings.NewReader(fmt.Sprintf(`
		{
		  "heatsinks": [

		    {
		      "name":"heatsink/1",
		      "min_temp": 1,
		      "max_temp": 10,
		      "temp_check_period": "3s",
		      "sensor_path_globs": [%q],
					"fan": {
						"path_glob": %q,
						"pwm_period": "22ms",
						"min_speed_value": "10",
						"max_speed_value": "200",
						"response_type": "sublinear"
					}
		    }

		  ]
		}
	`, sensorFile.Name(), fanFile.Name(),
	))

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.newHeatsinks()
	if !errors.Is(err, errFanRespTypeUnknwon) {
		t.Fatalf("unexpected error\nwant: %v\n got: %v", errFanRespTypeUnknwon, err)
	}

}

func Test_config_newHeatsinks_errorCreatingHeatsink(t *testing.T) {
	t.Parallel()

	sensorFile, cleanup := temporaryFile(t)
	defer cleanup()

	fanFile, cleanup := temporaryFile(t)
	defer cleanup()

	jsonData := strings.NewReader(fmt.Sprintf(`
		{
		  "heatsinks": [

		    {
		      "name":"heatsink/1",
		      "min_temp": 10,
		      "max_temp": 3,
		      "temp_check_period": "3s",
		      "sensor_path_globs": [%q],
					"fan": {
						"path_glob": %q,
						"pwm_period": "22ms",
						"min_speed_value": "10",
						"max_speed_value": "200"
					}
		    }

		  ]
		}
	`, sensorFile.Name(), fanFile.Name(),
	))

	cfg, err := newConfig(jsonData, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.newHeatsinks()

	// asserting against error strings is bad. However, it is better than introducing a custom
	// error type, overriding the returned error, or exporting an error from the other package
	// for the sake of this test. This should be good for now
	expectedString := "maximum temperature must be greater than the minimum"
	if !strings.Contains(err.Error(), expectedString) {
		t.Fatalf("expected error to contain the following string: '%s'", expectedString)
	}

}
