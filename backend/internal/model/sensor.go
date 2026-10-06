package model

import "time"

// CapReadings: the entity is a sensor that sends readings.
const CapReadings Capability = "readings"

// Metric is what a sensor measures.
type Metric string

const (
	MetricTemperature Metric = "temperature"
	MetricWaterLevel  Metric = "water_level"
	MetricWindSpeed   Metric = "wind_speed"
)

// MetricSpec describes a metric: its unit and the range of plausible values.
// Values outside the range are rejected as a faulty device.
type MetricSpec struct {
	ID    Metric  `json:"id"`
	Label string  `json:"label"`
	Unit  string  `json:"unit"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

// Metrics lists every metric. This is the single source of truth; the
// frontend reads it from GET /api/meta.
var Metrics = []MetricSpec{
	{MetricTemperature, "Temperature", "°C", -50, 80},
	{MetricWaterLevel, "Water level", "cm", 0, 2000},
	{MetricWindSpeed, "Wind speed", "m/s", 0, 100},
}

// Spec returns the specification of m, or false for an unknown metric.
func (m Metric) Spec() (MetricSpec, bool) {
	for _, s := range Metrics {
		if s.ID == m {
			return s, true
		}
	}
	return MetricSpec{}, false
}

// Valid reports whether m is one of the allowed metrics.
func (m Metric) Valid() bool {
	_, ok := m.Spec()
	return ok
}

const (
	// ReadingRetention is how long readings are kept.
	ReadingRetention = 7 * 24 * time.Hour
	// MaxReadingClockSkew is how far in the future a device clock may be.
	MaxReadingClockSkew = 5 * time.Minute
)

// SensorConfig is the sensor set up on an entity. The API key hash is never
// serialized; clients only learn whether a key exists.
type SensorConfig struct {
	EntityID     string     `json:"entity_id"`
	Metric       Metric     `json:"metric"`
	HasAPIKey    bool       `json:"has_api_key"`
	KeyCreatedAt *time.Time `json:"key_created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SensorInput is the request body for choosing a sensor's metric.
type SensorInput struct {
	Metric Metric `json:"metric" validate:"required,metric"`
}

// DeviceKey is returned once, when a device API key is created.
type DeviceKey struct {
	APIKey       string    `json:"api_key"`
	KeyCreatedAt time.Time `json:"key_created_at"`
}

// Reading is one measurement.
type Reading struct {
	Value      float64   `json:"value"`
	RecordedAt time.Time `json:"recorded_at"`
}

// ReadingInput is the request body a device sends. RecordedAt (RFC3339) is
// optional; when missing, the time the backend received the reading is used.
// It is a string so a bad format becomes a field error, not a broken body.
type ReadingInput struct {
	Value      *float64 `json:"value" validate:"required,finite"`
	RecordedAt *string  `json:"recorded_at" validate:"omitempty,rfc3339"`
}

// Readings is the response of GET /api/entities/{id}/readings.
type Readings struct {
	Metric   MetricSpec `json:"metric"`
	Readings []Reading  `json:"readings"`
	Latest   *Reading   `json:"latest"`
}
