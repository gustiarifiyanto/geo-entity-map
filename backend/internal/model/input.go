package model

import (
	"bytes"
	"encoding/json"
	"strings"
)

// EntityInput is the request body for creating or fully updating an entity.
// Coordinates are pointers so a missing value can be told apart from 0.
type EntityInput struct {
	Name        string          `json:"name" validate:"required,max=100"`
	Type        EntityType      `json:"type" validate:"required,entity_type"`
	Status      EntityStatus    `json:"status" validate:"required,entity_status"`
	Latitude    *float64        `json:"latitude" validate:"required,latitude"`
	Longitude   *float64        `json:"longitude" validate:"required,longitude"`
	Description string          `json:"description" validate:"max=500"`
	Attributes  json.RawMessage `json:"attributes" validate:"json_object"`
}

// Normalize trims string fields and treats a JSON null for attributes as unset.
// It must run before validation.
func (in *EntityInput) Normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Attributes = bytes.TrimSpace(in.Attributes)
	if len(in.Attributes) == 0 || bytes.Equal(in.Attributes, []byte("null")) {
		in.Attributes = nil
	}
}

// LocationInput is the request body for updating only an entity's coordinates.
type LocationInput struct {
	Latitude  *float64 `json:"latitude" validate:"required,latitude"`
	Longitude *float64 `json:"longitude" validate:"required,longitude"`
}
