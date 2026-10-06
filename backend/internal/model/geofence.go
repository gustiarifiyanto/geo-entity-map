package model

import (
	"math"
	"time"
)

// CapGeofence: the entity can have an operating zone (a circle).
const CapGeofence Capability = "geofence"

const (
	// MinGeofenceRadius and MaxGeofenceRadius bound a zone's radius, in meters.
	MinGeofenceRadius = 100
	MaxGeofenceRadius = 50_000

	// earthRadiusMeters is the mean Earth radius used for distances.
	earthRadiusMeters = 6_371_008.8
)

// Geofence is an entity's operating zone plus, computed for the entity's
// current position, how far it is from the center and whether it is inside.
type Geofence struct {
	EntityID        string    `json:"entity_id"`
	CenterLatitude  float64   `json:"center_latitude"`
	CenterLongitude float64   `json:"center_longitude"`
	RadiusM         float64   `json:"radius_m"`
	DistanceM       float64   `json:"distance_m"`
	Inside          bool      `json:"inside"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GeofenceInput is the request body for setting an operating zone.
// Pointers tell a missing value from 0.
type GeofenceInput struct {
	CenterLatitude  *float64 `json:"center_latitude" validate:"required,latitude"`
	CenterLongitude *float64 `json:"center_longitude" validate:"required,longitude"`
	RadiusM         *float64 `json:"radius_m" validate:"required,radius"`
}

// ZoneStatus is the single place that decides where a position stands
// relative to a zone, so manual positions today and tracked positions later
// are judged the same way. The distance is rounded to 0.1 m.
func ZoneStatus(centerLat, centerLng, radiusM, lat, lng float64) (distanceM float64, inside bool) {
	d := math.Round(DistanceMeters(centerLat, centerLng, lat, lng)*10) / 10
	return d, d <= radiusM
}

// DistanceMeters is the great-circle distance between two points (haversine).
func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusMeters * math.Asin(math.Min(1, math.Sqrt(a)))
}
