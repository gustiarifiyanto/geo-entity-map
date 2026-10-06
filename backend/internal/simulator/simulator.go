// Package simulator produces dummy sensor readings for IoT devices until real
// hardware is connected. Readings go through the same validation and service
// as the device endpoint, so they are stored exactly like real ones.
package simulator

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/service"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/validation"
)

const (
	// backfillWindow is filled with history when a device has no recent data,
	// so its chart is not empty right after the sensor is set up.
	backfillWindow = 24 * time.Hour
	backfillStep   = 15 * time.Minute
	// pruneEvery is how often old readings are deleted, counted in ticks.
	pruneEvery = 60
)

// profile shapes the dummy values of one metric: a level, a daily swing
// (peaking mid-afternoon) and random noise.
type profile struct {
	base, dailySwing, noise float64
}

var profiles = map[model.Metric]profile{
	model.MetricTemperature: {base: 29, dailySwing: 4, noise: 0.4},
	model.MetricWaterLevel:  {base: 150, dailySwing: 20, noise: 6},
	model.MetricWindSpeed:   {base: 4, dailySwing: 2, noise: 0.8},
}

// Simulator writes one reading per tick for every configured sensor.
type Simulator struct {
	sensors *service.SensorService
	val     *validation.Validator
	rng     *rand.Rand
	ticks   int
}

// New returns a simulator. seed makes the values reproducible in tests.
func New(sensors *service.SensorService, val *validation.Validator, seed uint64) *Simulator {
	return &Simulator{sensors: sensors, val: val, rng: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
}

// Run ticks immediately and then every interval until ctx is done. Errors
// are logged, never fatal: dummy data must not take the server down.
func (s *Simulator) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := s.Tick(ctx); err != nil && ctx.Err() == nil {
			slog.Warn("sensor simulator tick failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Tick adds a reading for every sensor (or a day of history for a sensor
// without recent data) and prunes old readings now and then.
func (s *Simulator) Tick(ctx context.Context) error {
	if s.ticks%pruneEvery == 0 {
		if _, err := s.sensors.Prune(ctx); err != nil {
			return fmt.Errorf("prune readings: %w", err)
		}
	}
	s.ticks++

	configs, err := s.sensors.Sensors(ctx)
	if err != nil {
		return err
	}
	now := s.sensors.Now()
	for _, cfg := range configs {
		if err := s.simulate(ctx, cfg, now); err != nil {
			return fmt.Errorf("simulate %s: %w", cfg.EntityID, err)
		}
	}
	return nil
}

func (s *Simulator) simulate(ctx context.Context, cfg model.SensorConfig, now time.Time) error {
	spec, ok := cfg.Metric.Spec()
	p, hasProfile := profiles[cfg.Metric]
	if !ok || !hasProfile {
		return nil // a metric without a profile is simply not simulated
	}

	latest, err := s.sensors.LatestReading(ctx, cfg.EntityID, cfg.Metric)
	if err != nil {
		return err
	}
	times := []time.Time{now}
	prev := p.base
	if latest == nil || latest.RecordedAt.Before(now.Add(-backfillWindow)) {
		times = times[:0]
		for t := now.Add(-backfillWindow + backfillStep); !t.After(now); t = t.Add(backfillStep) {
			times = append(times, t)
		}
	} else {
		prev = latest.Value
	}

	readings := make([]model.Reading, 0, len(times))
	for _, t := range times {
		value := s.next(p, spec, prev, t)
		prev = value
		// The same rules as a real device: range and timestamp.
		at := t.Format(time.RFC3339)
		in := model.ReadingInput{Value: &value, RecordedAt: &at}
		fields, err := s.val.Reading(&in, spec, now)
		if err != nil {
			return err
		}
		if fields != nil {
			return fmt.Errorf("simulated reading rejected: %v", fields)
		}
		readings = append(readings, model.Reading{Value: value, RecordedAt: t})
	}
	return s.sensors.Record(ctx, cfg.EntityID, cfg.Metric, readings)
}

// next moves the previous value part of the way toward the profile's level
// for time t, adds noise, keeps it in the metric's range and rounds it to
// one decimal.
func (s *Simulator) next(p profile, spec model.MetricSpec, prev float64, t time.Time) float64 {
	hours := float64(t.Hour()) + float64(t.Minute())/60
	target := p.base + p.dailySwing*math.Sin(2*math.Pi*(hours-9)/24)
	v := prev + 0.3*(target-prev) + s.rng.NormFloat64()*p.noise
	v = math.Max(spec.Min, math.Min(spec.Max, v))
	return math.Round(v*10) / 10
}
