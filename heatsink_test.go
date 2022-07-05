package heatsink

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-test/deep"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		inConfig *Config
		outErr   error
	}{
		"valid": {
			inConfig: &Config{
				Fan:            &fakeFanDriver{},
				MinTemperature: 10,
				MaxTemperature: 20,
				Sensors:        []ThermoSensor{&fakeThermoSensor{}, &fakeThermoSensor{}},
			},
			outErr: nil,
		},
		"config-is-nil": {
			inConfig: nil,
			outErr:   errNoConfig,
		},
		"fan-is-nil": {
			inConfig: &Config{
				Fan:            nil,
				MinTemperature: 10,
				MaxTemperature: 20,
				Sensors:        []ThermoSensor{&fakeThermoSensor{}, &fakeThermoSensor{}},
			},
			outErr: errNoFan,
		},
		"sensors-empty": {
			inConfig: &Config{
				Fan:            &fakeFanDriver{},
				MinTemperature: 10,
				MaxTemperature: 20,
				Sensors:        []ThermoSensor{},
			},
			outErr: errNoSensors,
		},
		"sensor-is-nil": {
			inConfig: &Config{
				Fan:            &fakeFanDriver{},
				MinTemperature: 10,
				MaxTemperature: 20,
				Sensors:        []ThermoSensor{&fakeThermoSensor{}, nil},
			},
			outErr: errNilSensor,
		},
		"temperatures-min-max-equal": {
			inConfig: &Config{
				Fan:            &fakeFanDriver{},
				MinTemperature: 10,
				MaxTemperature: 10,
				Sensors:        []ThermoSensor{&fakeThermoSensor{}, &fakeThermoSensor{}},
			},
			outErr: errBadTemps,
		},
		"temperatures-min-larger-than-max": {
			inConfig: &Config{
				Fan:            &fakeFanDriver{},
				MinTemperature: 12,
				MaxTemperature: 10,
				Sensors:        []ThermoSensor{&fakeThermoSensor{}, &fakeThermoSensor{}},
			},
			outErr: errBadTemps,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, actualErr := New(testCase.inConfig)
			if !errors.Is(actualErr, testCase.outErr) {
				t.Fatalf("unexpected error\nwant: %v\n got: %v", testCase.outErr, actualErr)
			}
		})
	}
}

func TestNew_defaults(t *testing.T) {
	orig := deep.CompareUnexportedFields
	deep.CompareUnexportedFields = true
	defer func() { deep.CompareUnexportedFields = orig }()

	fd := &fakeFanDriver{onName: "cpu-fan1"}
	ths := &fakeThermoSensor{}

	expected := &Heatsink{
		name:      "heatsink/cpu-fan1",
		chkPeriod: 1 * time.Second,
		dcCalc:    newDutyCyclerPowPi(35, 45),
		fan:       fd,
		sensors:   []ThermoSensor{ths},
		isStopped: make(chan struct{}),
		logger:    zap.NewNop(),
	}

	config := &Config{
		Fan:            fd,
		Sensors:        []ThermoSensor{ths},
		MinTemperature: 35,
		MaxTemperature: 45,
	}
	actual, err := New(config)
	if err != nil {
		t.Fatalf("expected no error creating a heatsink, got: %v", err)
	}

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Fatal("actual heatsink does not match expected\n", strings.Join(diff, "\n"))
	}
}

func TestNew_validOptions(t *testing.T) {
	orig := deep.CompareUnexportedFields
	deep.CompareUnexportedFields = true
	defer func() { deep.CompareUnexportedFields = orig }()

	logger := zap.NewExample()
	sensors := []ThermoSensor{&fakeThermoSensor{}}
	fanDriver := &fakeFanDriver{}

	expected := &Heatsink{
		name:      t.Name(),
		chkPeriod: 100 * time.Millisecond,
		dcCalc:    newDutyCyclerPowPi(0, 10),
		fan:       fanDriver,
		sensors:   sensors,
		isStopped: make(chan struct{}),
		logger:    logger,
	}

	config := &Config{
		Fan:            fanDriver,
		Sensors:        sensors,
		MinTemperature: 0,
		MaxTemperature: 10,
	}
	actual, err := New(
		config,
		nil, // should be ignored
		OptName(t.Name()),
		OptLogger(logger),
		OptTemperatureCheckPeriod(100*time.Millisecond),
		OptFanResponse(FanResponsePowPi),
	)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Fatal("actual heatsink does not match expected\n", strings.Join(diff, "\n"))
	}
}

func TestNew_validOptions_defaultFanResponse(t *testing.T) {
	orig := deep.CompareUnexportedFields
	deep.CompareUnexportedFields = true
	defer func() { deep.CompareUnexportedFields = orig }()

	logger := zap.NewNop()
	sensors := []ThermoSensor{&fakeThermoSensor{}}
	fanDriver := &fakeFanDriver{}

	expected := &Heatsink{
		name:      t.Name(),
		chkPeriod: 100 * time.Millisecond,
		dcCalc:    newDutyCyclerLinear(0, 10),
		fan:       fanDriver,
		sensors:   sensors,
		isStopped: make(chan struct{}),
		logger:    logger,
	}

	config := &Config{
		Fan:            fanDriver,
		Sensors:        sensors,
		MinTemperature: 0,
		MaxTemperature: 10,
	}
	actual, err := New(
		config,
		OptName(t.Name()),
		OptLogger(nil),
		OptTemperatureCheckPeriod(100*time.Millisecond),
		OptFanResponse(FanResponseLinear),
	)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Fatal("actual heatsink does not match expected\n", strings.Join(diff, "\n"))
	}
}

func TestNew_invalidOptions(t *testing.T) {
	orig := deep.CompareUnexportedFields
	deep.CompareUnexportedFields = true
	defer func() { deep.CompareUnexportedFields = orig }()

	sensors := []ThermoSensor{&fakeThermoSensor{}}
	fanDriver := &fakeFanDriver{onName: "cpu-fan1"}

	expected := &Heatsink{
		name:      "heatsink/cpu-fan1",
		chkPeriod: 1 * time.Second,
		dcCalc:    newDutyCyclerPowPi(0, 10),
		fan:       fanDriver,
		sensors:   sensors,
		isStopped: make(chan struct{}),
		logger:    zap.NewNop(),
	}

	config := &Config{
		Fan:            fanDriver,
		Sensors:        sensors,
		MinTemperature: 0,
		MaxTemperature: 10,
	}
	actual, err := New(
		config,
		OptName(""),
		OptLogger(nil),
		OptTemperatureCheckPeriod(time.Duration(-10)),
	)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Fatal("actual heatsink does not match expected\n", strings.Join(diff, "\n"))
	}
}

func TestNew_copiesSensors(t *testing.T) {
	t.Parallel()

	sensors := []ThermoSensor{&fakeThermoSensor{}, &fakeThermoSensor{}}
	config := &Config{
		Fan:            &fakeFanDriver{},
		Sensors:        sensors,
		MaxTemperature: 1,
	}
	actual, err := New(config)
	if err != nil {
		t.Fatalf("expected no error creating a heatsink, got: %v", err)
	}

	sensors[0], sensors[1] = nil, nil
	if actual.sensors[0] == nil || actual.sensors[1] == nil {
		t.Fatal("expected heatsink factory function to make a copy of sensors")
	}
}

func TestHeatsink(t *testing.T) {
	t.Parallel()

	fanDriver := &fakeFanDriver{}
	sensor1 := &fakeThermoSensor{onTemperatureVals: []float64{36}}
	sensor2 := &fakeThermoSensor{onTemperatureVals: []float64{40}}
	config := &Config{
		Fan:            fanDriver,
		Sensors:        []ThermoSensor{sensor1, sensor2},
		MinTemperature: 35,
		MaxTemperature: 45,
	}
	hs, err := New(config)
	if err != nil {
		t.Fatalf("expected no error creating a heatsink, got: %v", err)
	}
	hs.dcCalc = &fakeDutyCycler{
		tmpToDC: map[float64]float64{
			40: 0.40,
			36: 0.36,
		},
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err = hs.StartThermalControl()
		if !errors.Is(err, ErrControllerStopped) {
			t.Errorf("unexpected error\nwant: %v\n got: %v", ErrControllerStopped, err)
		}
		wg.Done()
	}()

	for deadline := time.After(100 * time.Millisecond); ; {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for thermal control to set fan's dc ratio")
		default:
		}
		fanDriver.mutex.Lock()
		if len(fanDriver.argSetDutyCycle) <= 0 {
			fanDriver.mutex.Unlock()
			continue
		}

		if expected, actual := 0.40, fanDriver.argSetDutyCycle[0]; expected != actual {
			t.Fatalf(
				"expected max core temperature 40.0 to result in dc ratio %.2f, got: %.2f",
				expected, actual,
			)
		}
		fanDriver.mutex.Unlock()
		break // test passed
	}

	if err := hs.StopThermalControl(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()

	if sensor1.numCloseCalls != 1 {
		t.Errorf(
			"expected sensor1 to be closed exactly once, but it was closed %d times",
			sensor1.numCloseCalls,
		)
	}
	if sensor2.numCloseCalls != 1 {
		t.Errorf(
			"expected sensor2 to be closed exactly once, but it was closed %d times",
			sensor2.numCloseCalls,
		)
	}
	if fanDriver.numCloseCalls != 1 {
		t.Errorf(
			"expected fan driver to be closed exactly once, but it was closed %d times",
			fanDriver.numCloseCalls,
		)
	}

	err = hs.StopThermalControl()
	if !errors.Is(err, ErrControllerStopped) {
		t.Fatalf(
			"unexpected error stopping thermal control twice\nwant: %v\n got: %v",
			ErrControllerStopped, err,
		)
	}
}

func TestHeatsink_StartThermalControl_errorReadingMaxCoreTemp(t *testing.T) {
	t.Parallel()

	simErrSensor1 := errors.New("simulated error reading from sensor 1")
	simErrSensor2 := errors.New("simulated error reading from sensor 2")

	sensor1 := &fakeThermoSensor{onTemperatureErrs: []error{simErrSensor1}}
	sensor2 := &fakeThermoSensor{onTemperatureErrs: []error{simErrSensor2}}
	config := &Config{
		Fan:            &fakeFanDriver{},
		Sensors:        []ThermoSensor{sensor1, sensor2},
		MinTemperature: 1,
		MaxTemperature: 2,
	}
	hs, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	err = hs.StartThermalControl()
	var actualErr multiErrs
	if ok := errors.As(err, &actualErr); !ok {
		t.Fatalf("unexpected error type\nwant: %T\n got: %T", actualErr, err)
	}
	if len(actualErr) != len(config.Sensors) {
		t.Fatalf(
			"expected the count of errors to match the count of sensors (%d), got: %d",
			len(config.Sensors), len(actualErr),
		)
	}
	if !errors.Is(actualErr[0], simErrSensor1) {
		t.Errorf(
			"unexpected first error in multiErrs\nwant: %v\n got: %v",
			simErrSensor1, actualErr[0],
		)
	}
	if !errors.Is(actualErr[1], simErrSensor2) {
		t.Errorf(
			"unexpected second error in multiErrs\nwant: %v\n got: %v",
			simErrSensor2, actualErr[1],
		)
	}

	err = hs.StopThermalControl()
	if !errors.Is(err, ErrControllerStopped) {
		t.Errorf(
			"expected the thermal control to be stopped if an error is encountered\n"+
				"want: %v\n got: %v", ErrControllerStopped, err,
		)
	}
}

func TestHeatsink_StartThermalControl_errorSettingDutyCycle(t *testing.T) {
	t.Parallel()

	simErr := errors.New("simulated error setting duty cycle")
	fanDriver := &fakeFanDriver{onSetDutyCycleErrs: []error{simErr}}
	config := &Config{
		Fan:            fanDriver,
		Sensors:        []ThermoSensor{&fakeThermoSensor{onTemperatureVals: []float64{10.0}}},
		MinTemperature: 1,
		MaxTemperature: 2,
	}
	hs, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	actualErr := hs.StartThermalControl()
	if !errors.Is(actualErr, simErr) {
		t.Errorf(
			"unexpected error\nwant: %v\n got: %v",
			simErr, actualErr,
		)
	}

	err = hs.StopThermalControl()
	if !errors.Is(err, ErrControllerStopped) {
		t.Errorf(
			"expected the thermal control to be stopped if an error is encountered\n"+
				"want: %v\n got: %v", ErrControllerStopped, err,
		)
	}
}

func TestHeatsink_StartThermalControl_logsErrorFromStoppingThermalControl(t *testing.T) {
	t.Parallel()

	expectedLogMsg := "failed to properly stop thermal control after encountering an error"
	expectedLogMsgFound := false
	// silence the logger because we are not interested in displaying output in test
	loggerCfg := zap.NewDevelopmentConfig()
	loggerCfg.OutputPaths, loggerCfg.ErrorOutputPaths = nil, nil
	interceptedLogger, err := loggerCfg.Build(
		zap.Hooks(
			func(e zapcore.Entry) error {
				if strings.Contains(e.Message, expectedLogMsg) {
					expectedLogMsgFound = true
				}
				return nil
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	sensor1 := &fakeThermoSensor{
		onTemperatureErrs: []error{errors.New("simulated error reading from sensor")},
		onCloseErrs:       []error{errors.New("simulated error closing sensor")},
	}
	config := &Config{
		Fan:            &fakeFanDriver{},
		Sensors:        []ThermoSensor{sensor1},
		MinTemperature: 1,
		MaxTemperature: 2,
	}
	hs, err := New(config, OptLogger(interceptedLogger))
	if err != nil {
		t.Fatal(err)
	}

	_ = hs.StartThermalControl()
	if !expectedLogMsgFound {
		t.Fatalf(
			"the expected log entry when attempting to close thermal control following error "+
				"was not found\n want: '%s'", expectedLogMsg,
		)
	}
}

func TestHeatsink_StartThermalControl_logsErrorIfOneSensorFails(t *testing.T) {
	t.Parallel()

	expectedLogMsg := "failed to read temperature"
	expectedLogMsgFound := make(chan struct{})
	// silence the logger because we are not interested in displaying output in test
	loggerCfg := zap.NewDevelopmentConfig()
	loggerCfg.OutputPaths, loggerCfg.ErrorOutputPaths = nil, nil
	interceptedLogger, err := loggerCfg.Build(
		zap.Hooks(
			func(e zapcore.Entry) error {
				if strings.Contains(e.Message, expectedLogMsg) {
					close(expectedLogMsgFound)
				}
				return nil
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Fan: &fakeFanDriver{},
		Sensors: []ThermoSensor{
			&fakeThermoSensor{onTemperatureErrs: []error{errors.New("simulated error")}},
			&fakeThermoSensor{},
		},
		MinTemperature: 1,
		MaxTemperature: 2,
	}
	hs, err := New(config, OptLogger(interceptedLogger))
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		_ = hs.StartThermalControl()
	}()

	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatalf(
			"the expected log entry if reading from a sensor fails was not found\n want: '%s'",
			expectedLogMsg,
		)
	case <-expectedLogMsgFound:
		return // test passed
	}
}

func TestHeatsink_StopThermalControl_multipleErrs(t *testing.T) {
	t.Parallel()

	simErrFan := errors.New("simulated error for closing fan")
	simErrSensor1 := errors.New("simulated error for closing sensor 1")
	simErrSensor3 := errors.New("simulated error for closing sensor 3")

	fanDriver := &fakeFanDriver{onCloseErrs: []error{simErrFan}}
	sensor1 := &fakeThermoSensor{onCloseErrs: []error{simErrSensor1}}
	sensor2 := &fakeThermoSensor{}
	sensor3 := &fakeThermoSensor{onCloseErrs: []error{simErrSensor3}}
	config := &Config{
		Fan:            fanDriver,
		Sensors:        []ThermoSensor{sensor1, sensor2, sensor3},
		MaxTemperature: 10,
	}
	hs, err := New(config)
	if err != nil {
		t.Fatalf("expected no error creating a heatsink, got: %v", err)
	}

	err = hs.StopThermalControl()
	var actualErr multiErrs
	if !errors.As(err, &actualErr) {
		t.Fatalf("unexpected error type\nwant: %T\n got: %T", multiErrs(nil), err)
	}
	if len(actualErr) != 3 {
		t.Fatalf("expected 3 errors, got: %d", len(actualErr))
	}
	if !errors.Is(actualErr[0], simErrFan) {
		t.Errorf("unexpected error\nwant: %v\n got: %v", simErrFan, actualErr[0])
	}
	if !errors.Is(actualErr[1], simErrSensor1) {
		t.Errorf("unexpected error\nwant: %v\n got: %v", simErrSensor1, actualErr[1])
	}
	if !errors.Is(actualErr[2], simErrSensor3) {
		t.Errorf("unexpected error\nwant: %v\n got: %v", simErrSensor3, actualErr[2])
	}

	expectedErrStr := "\n" +
		"  - error closing fan: simulated error for closing fan\n" +
		"  - error closing sensor: simulated error for closing sensor 1\n" +
		"  - error closing sensor: simulated error for closing sensor 3"
	if actualErrStr := err.Error(); expectedErrStr != actualErrStr {
		t.Errorf("unexpected formatted error string\nwant: %s\n got: %s", expectedErrStr, actualErrStr)
	}
}

func Test_multiErrs_Error_singleErr(t *testing.T) {
	simErr := errors.New("simulated error")
	me := multiErrs{simErr}
	expected := simErr.Error()
	actual := me.Error()
	if expected != actual {
		t.Errorf("actual error string does not match expected\nwant: %q\n got: %q", expected, actual)
	}
}

func Test_constErr_Error(t *testing.T) {
	err := constErr(t.Name())
	if err.Error() != t.Name() {
		t.Fatalf("unexpected error string\nwant: '%s'\n got: '%s'", t.Name(), err)
	}
}
