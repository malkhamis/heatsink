package heatsink

import (
	"testing"
)

func TestDutyCycler_Linear(t *testing.T) {
	t.Parallel()

	dc := newDutyCyclerLinear(10, 20)
	cases := map[string]struct {
		inTemp          float64
		expectedDcRatio float64
	}{
		"at-min":         {inTemp: 10.0, expectedDcRatio: 0.0},
		"below-min":      {inTemp: 9.00, expectedDcRatio: 0.0},
		"at-max":         {inTemp: 20.0, expectedDcRatio: 1.0},
		"above-max":      {inTemp: 25.0, expectedDcRatio: 1.0},
		"one-quarter":    {inTemp: 12.5, expectedDcRatio: 0.25},
		"mid-point":      {inTemp: 15.0, expectedDcRatio: 0.5},
		"three-quarters": {inTemp: 17.5, expectedDcRatio: 0.75},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			actual := dc.ratio(testCase.inTemp)
			if actual != testCase.expectedDcRatio {
				t.Fatalf(
					"actual dcRatio does not match expected\nwant: %.2f\n got: %.2f",
					testCase.expectedDcRatio, actual,
				)
			}
		})
	}
}

func TestDutyCycler_PowPi(t *testing.T) {
	t.Parallel()

	dc := newDutyCyclerPowPi(10, 20)
	cases := map[string]struct {
		inTemp          float64
		expectedDcRatio float64
	}{
		"at-min":    {inTemp: 10.0, expectedDcRatio: 0.0},
		"below-min": {inTemp: 9.00, expectedDcRatio: 0.0},
		"at-max":    {inTemp: 20.0, expectedDcRatio: 1.0},
		"above-max": {inTemp: 25.0, expectedDcRatio: 1.0},
		// the following two cases might fail on different platforms/CPUs. If that happens, I
		// should resort to rounding these numbers
		"60-20": {inTemp: 16, expectedDcRatio: 0.200928525861769902149944755365140736103057861328125000},
		"80-50": {inTemp: 18, expectedDcRatio: 0.496075998354999436745771390633308328688144683837890625},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			actual := dc.ratio(testCase.inTemp)
			if actual != testCase.expectedDcRatio {
				t.Fatalf(
					"actual dcRatio does not match expected\nwant: %.64f\n got: %.64f",
					testCase.expectedDcRatio, actual,
				)
			}
		})
	}
}
