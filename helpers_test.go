package heatsink

import "sync"

var (
	_ FanDriver    = (*fakeFanDriver)(nil)
	_ ThermoSensor = (*fakeThermoSensor)(nil)
	_ dutyCycler   = (*fakeDutyCycler)(nil)
)

type fakeFanDriver struct {
	argSetDutyCycle    []float64
	onSetDutyCycleErrs []error
	numCloseCalls      int
	onCloseErrs        []error
	onName             string
	mutex              sync.Mutex
}

func (ffd *fakeFanDriver) SetDutyCycle(dcRatio float64) (err error) {
	ffd.mutex.Lock()
	defer ffd.mutex.Unlock()

	ffd.argSetDutyCycle = append(ffd.argSetDutyCycle, dcRatio)
	if len(ffd.onSetDutyCycleErrs) > 0 {
		err = ffd.onSetDutyCycleErrs[0]
		ffd.onSetDutyCycleErrs = ffd.onSetDutyCycleErrs[1:]
	}
	return
}

func (ffd *fakeFanDriver) Close() (err error) {
	ffd.mutex.Lock()
	defer ffd.mutex.Unlock()

	ffd.numCloseCalls++
	if len(ffd.onCloseErrs) > 0 {
		err = ffd.onCloseErrs[0]
		ffd.onCloseErrs = ffd.onCloseErrs[1:]
	}
	return
}

func (ffd *fakeFanDriver) Name() string {
	return ffd.onName
}

type fakeThermoSensor struct {
	onTemperatureErrs []error
	onTemperatureVals []float64
	onCloseErrs       []error
	numCloseCalls     int
	onName            string
	mutex             sync.Mutex
}

func (fts *fakeThermoSensor) Temperature() (temp float64, err error) {
	fts.mutex.Lock()
	defer fts.mutex.Unlock()

	if len(fts.onTemperatureVals) > 0 {
		temp = fts.onTemperatureVals[0]
		fts.onTemperatureVals = fts.onTemperatureVals[1:]
	}
	if len(fts.onTemperatureErrs) > 0 {
		err = fts.onTemperatureErrs[0]
		fts.onTemperatureErrs = fts.onTemperatureErrs[1:]
	}
	return
}

func (fts *fakeThermoSensor) Close() (err error) {
	fts.mutex.Lock()
	defer fts.mutex.Unlock()

	fts.numCloseCalls++
	if len(fts.onCloseErrs) > 0 {
		err = fts.onCloseErrs[0]
		fts.onCloseErrs = fts.onCloseErrs[1:]
	}
	return
}

func (fts *fakeThermoSensor) Name() string {
	return fts.onName
}

type fakeDutyCycler struct {
	tmpToDC map[float64]float64
}

func (fdc *fakeDutyCycler) ratio(temp float64) (dcRatio float64) {
	return fdc.tmpToDC[temp]
}
