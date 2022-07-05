package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/malkhamis/heatsink"
	"github.com/malkhamis/heatsink/fanpwm"
	"github.com/malkhamis/heatsink/thermosense"

	"go.uber.org/zap"
)

var (
	errNoJsonConfig       = errors.New("no json config data given")
	errNoHeatsinkConfig   = errors.New("no heatsink config in given json data")
	errBadDuration        = errors.New("error parsing string as duration")
	errGlobNoMatches      = errors.New("no file matches for the given glob(s)")
	errGlobTooManyMatches = errors.New("too many matches for the given globe(s)")
	errFanRespTypeUnknwon = errors.New("unknown fan response type")
)

type config struct {
	Heatsinks []*configHeatsink `json:"heatsinks"`
	logger    *zap.Logger
}

type configHeatsink struct {
	Name            string        `json:"name"`
	Fan             configFan     `json:"fan"`
	SensorPathGlobs configSensors `json:"sensor_path_globs"`
	TempChkPeriod   string        `json:"temp_check_period"`
	MinTemp         float64       `json:"min_temp"`
	MaxTemp         float64       `json:"max_temp"`
	FanRespType     string        `json:"fan_response"`
}

type configFan struct {
	Name        string `json:"name"`
	PathGlob    string `json:"path_glob"`
	PwmPeriod   string `json:"pwm_period"`
	MinSpeedVal string `json:"min_speed_value"`
	MaxSpeedVal string `json:"max_speed_value"`
	// RespType is relevant to configHeatsink. However, presenting it here is user-friendlier
	RespType string `json:"response_type"`
}

type configSensors []string

func newConfig(jsonData io.Reader, logger *zap.Logger) (*config, error) {

	if jsonData == nil {
		return nil, errNoJsonConfig
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	cfg := &config{logger: logger}
	if err := json.NewDecoder(jsonData).Decode(cfg); err != nil {
		return nil, fmt.Errorf("error decoding json config: %w", err)
	}

	for _, hs := range cfg.Heatsinks {
		if hs.Fan.RespType == "" {
			hs.Fan.RespType = "PowPi"
		}
	}

	if len(cfg.Heatsinks) == 0 {
		return nil, errNoHeatsinkConfig
	}

	return cfg, nil
}

func (c *config) newHeatsinks() ([]*heatsink.Heatsink, error) {

	var heatsinks []*heatsink.Heatsink
	for _, hsCfg := range c.Heatsinks {
		hs, err := hsCfg.newHeatsink(c.logger)
		if err != nil {
			return nil, fmt.Errorf("heatsink '%s': %w", hsCfg.Name, err)
		}
		heatsinks = append(heatsinks, hs)
	}

	c.logger.Info(
		"all heatsinks were created successfully",
		zap.Int("heatsink-count", len(heatsinks)),
	)
	return heatsinks, nil
}

func (c *configHeatsink) newHeatsink(logger *zap.Logger) (*heatsink.Heatsink, error) {

	tempChkPeriod, err := time.ParseDuration(c.TempChkPeriod)
	if err != nil && c.TempChkPeriod != "" {
		return nil, fmt.Errorf("%w: %v", errBadDuration, err)
	}
	// otherwise, it is empty and we assume the zero-value will fallback to default

	sensors, err := c.SensorPathGlobs.newSensors(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create all sensors: %w", err)
	}

	fan, err := c.Fan.newFan(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create fan '%s': %w", c.Fan.Name, err)
	}

	var optRespType heatsink.Option
	switch strings.ToLower(c.Fan.RespType) {
	case "linear":
		optRespType = heatsink.OptFanResponse(heatsink.FanResponseLinear)
	case "powpi":
		optRespType = heatsink.OptFanResponse(heatsink.FanResponsePowPi)
	default:
		return nil, fmt.Errorf("%w: '%s'", errFanRespTypeUnknwon, c.Fan.RespType)
	}

	hs, err := heatsink.New(
		&heatsink.Config{
			Fan:            fan,
			Sensors:        sensors,
			MinTemperature: c.MinTemp,
			MaxTemperature: c.MaxTemp,
		},
		optRespType,
		heatsink.OptName(c.Name),
		heatsink.OptTemperatureCheckPeriod(tempChkPeriod),
		heatsink.OptLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create heatsink: %w", err)
	}

	logger.Info(
		"created heatsink",
		zap.String("name", c.Name),
		zap.String("temp_check_period", tempChkPeriod.String()),
		zap.Float64("min_temp", c.MinTemp),
		zap.Float64("max_temp", c.MaxTemp),
	)
	return hs, nil
}

func (c configFan) newFan(logger *zap.Logger) (heatsink.FanDriver, error) {
	period, err := time.ParseDuration(c.PwmPeriod)
	if err != nil && c.PwmPeriod != "" {
		return nil, fmt.Errorf("%w: %v", errBadDuration, err)
	}
	// otherwise, it is empty and we assume the zero-value will fallback to default

	matches, err := filepath.Glob(c.PathGlob)
	if err != nil {
		return nil, fmt.Errorf("invalid glob '%s': %w", c.PathGlob, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("'%s': %w", c.PathGlob, errGlobNoMatches)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("'%s': %w", c.PathGlob, errGlobTooManyMatches)
	}
	filename := matches[0]

	fan, err := fanpwm.New(
		filename,
		fanpwm.OptName(c.Name),
		fanpwm.OptPeriodPWM(period),
		fanpwm.OptMinSpeedValue(c.MinSpeedVal),
		fanpwm.OptMaxSpeedValue(c.MaxSpeedVal),
	)
	if err != nil {
		return nil, fmt.Errorf("'%s': %w", filename, err)
	}

	logger.Info(
		"created PWM fan",
		zap.String("name", c.Name),
		zap.String("filename", filename),
		zap.String("pwm_period", period.String()),
		zap.String("min_speed_value", c.MinSpeedVal),
		zap.String("max_speed_value", c.MaxSpeedVal),
		zap.String("response_type", c.RespType),
	)
	return fan, nil
}

func (c configSensors) newSensors(logger *zap.Logger) ([]heatsink.ThermoSensor, error) {

	var (
		allSensors   []heatsink.ThermoSensor
		allFilenames []string
	)

	for _, pattern := range c {
		sensorFilenames, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob '%s': %w", pattern, err)
		}
		allFilenames = append(allFilenames, sensorFilenames...)
	}

	if len(allFilenames) == 0 {
		return nil, fmt.Errorf("[%s]: %w", strings.Join(c, ", "), errGlobNoMatches)
	}

	for _, filename := range allFilenames {
		filename = filepath.Clean(filename)
		sensor, err := thermosense.New(filename)
		if err != nil {
			return nil, fmt.Errorf("'%s': %w", filename, err)
		}
		logger.Info("created thermo sensor", zap.String("filename", filename))
		allSensors = append(allSensors, sensor)
	}

	return allSensors, nil
}
