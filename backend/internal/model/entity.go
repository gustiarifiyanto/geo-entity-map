// Package model defines the core domain types. It is the single source of
// truth for the allowed entity types and statuses.
package model

import (
	"encoding/json"
	"slices"
	"time"
)

// EntityType classifies what real-world object an entity represents.
type EntityType string

const (
	TypeVehicle   EntityType = "vehicle"
	TypeIoTDevice EntityType = "iot_device"
	TypeFacility  EntityType = "facility"
)

// EntityTypes lists every allowed entity type. Add new types here only.
var EntityTypes = []EntityType{
	TypeVehicle,
	TypeIoTDevice,
	TypeFacility,
}

// Valid reports whether t is one of the allowed entity types.
func (t EntityType) Valid() bool {
	return slices.Contains(EntityTypes, t)
}

// EntityStatus describes the operational state of an entity.
type EntityStatus string

const (
	StatusActive      EntityStatus = "active"
	StatusInactive    EntityStatus = "inactive"
	StatusMaintenance EntityStatus = "maintenance"
)

// EntityStatuses lists every allowed entity status. Add new statuses here only.
var EntityStatuses = []EntityStatus{
	StatusActive,
	StatusInactive,
	StatusMaintenance,
}

// Valid reports whether s is one of the allowed entity statuses.
func (s EntityStatus) Valid() bool {
	return slices.Contains(EntityStatuses, s)
}

// Entity is an object with a geographic location.
type Entity struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        EntityType   `json:"type"`
	Status      EntityStatus `json:"status"`
	Latitude    float64      `json:"latitude"`
	Longitude   float64      `json:"longitude"`
	Description string       `json:"description"`
	// Attributes is a JSON object, or nil when not set (serialized as null).
	Attributes json.RawMessage `json:"attributes"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}
