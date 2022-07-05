package heatsink

import "math"

// compile-time check for interface implementation
var (
	_ dutyCycler = (*dutyCyclerLinear)(nil)
	_ dutyCycler = (*dutyCyclerPowPi)(nil)
)

type dutyCyclerLinear struct {
	minTemp float64
	maxTemp float64
	tRange  float64
}

func newDutyCyclerLinear(minTemp, maxTemp float64) *dutyCyclerLinear {
	return &dutyCyclerLinear{
		minTemp: minTemp,
		maxTemp: maxTemp,
		tRange:  maxTemp - minTemp,
	}
}

func (dc *dutyCyclerLinear) ratio(temp float64) float64 {
	if temp >= dc.maxTemp {
		return 1.0
	}
	if temp <= dc.minTemp {
		return 0.0
	}
	dcRatio := (temp - dc.minTemp) / dc.tRange
	return dcRatio
}

type dutyCyclerPowPi struct {
	minTemp float64
	maxTemp float64
	tRange  float64
}

func newDutyCyclerPowPi(minTemp, maxTemp float64) *dutyCyclerPowPi {
	return &dutyCyclerPowPi{
		minTemp: minTemp,
		maxTemp: maxTemp,
		tRange:  maxTemp - minTemp,
	}
}

func (dc *dutyCyclerPowPi) ratio(temp float64) float64 {
	if temp >= dc.maxTemp {
		return 1.0
	}
	if temp <= dc.minTemp {
		return 0.0
	}
	fraction := (temp - dc.minTemp) / dc.tRange
	dcRatio := math.Pow(fraction, math.Pi)
	return dcRatio
}
