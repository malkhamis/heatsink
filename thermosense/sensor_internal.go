package thermosense

import (
	"fmt"
	"io"
	"math"

	"github.com/malkhamis/heatsink"
)

type rdOnlyFile interface {
	io.ReadSeeker
	io.Closer
}

type tempMilliDegCelsius int

func (t tempMilliDegCelsius) degCelsius() float64 {
	tf := float64(t)
	tempDeg := tf / 1000.0
	fraction := (tf - tempDeg*1000.0) / 1000.0
	tempDeg += fraction
	return tempDeg
}

func (s *Sensor) temperature() (float64, error) {

	if s.closed {
		return math.Inf(1), heatsink.ErrThermoSensorClosed
	}

	if _, err := s.devFile.Seek(0, 0); err != nil {
		return math.Inf(1), err
	}

	var temp tempMilliDegCelsius
	if _, err := fmt.Fscanf(s.devFile, "%d", &temp); err != nil {
		return math.Inf(1), err
	}

	return temp.degCelsius(), nil
}

func (s *Sensor) close() error {
	if s.closed {
		return heatsink.ErrThermoSensorClosed
	}
	s.closed = true

	if err := s.devFile.Close(); err != nil {
		return fmt.Errorf("failed to close device file while closing sensor: %w", err)
	}

	return nil
}
