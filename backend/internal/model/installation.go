package model

import (
	"slices"
	"strings"
	"time"
)

// Capability is a feature that only some entity types have.
type Capability string

const (
	// CapInstallation: the entity has installation dates (start, target, completion).
	CapInstallation Capability = "installation"
)

// TypeCapabilities lists the capabilities of each entity type. Types that are
// not listed have none. This is the single source of truth; the frontend reads
// it from GET /api/meta.
var TypeCapabilities = map[EntityType][]Capability{
	TypeIoTDevice: {CapInstallation, CapReadings},
	TypeFacility:  {CapInstallation},
}

// Has reports whether entity type t has capability c.
func (t EntityType) Has(c Capability) bool {
	return slices.Contains(TypeCapabilities[t], c)
}

// InstallationStatus is derived from the dates and today's date; it is never stored.
type InstallationStatus string

const (
	InstallationScheduled       InstallationStatus = "scheduled"
	InstallationInProgress      InstallationStatus = "in_progress"
	InstallationOverdue         InstallationStatus = "overdue"
	InstallationCompletedOnTime InstallationStatus = "completed_on_time"
	InstallationCompletedLate   InstallationStatus = "completed_late"
)

// DateLayout is how installation dates are written: a calendar day, no time.
const DateLayout = "2006-01-02"

// Installation is the installation schedule of an entity plus values
// computed from it for "today".
type Installation struct {
	EntityID    string             `json:"entity_id"`
	StartedOn   string             `json:"started_on"`
	TargetOn    string             `json:"target_on"`
	CompletedOn *string            `json:"completed_on"`
	Status      InstallationStatus `json:"status"`
	PlannedDays int                `json:"planned_days"`
	ElapsedDays int                `json:"elapsed_days"`
	DaysLate    int                `json:"days_late"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// InstallationInput is the request body for setting an entity's installation dates.
// CompletedOn is a pointer so "not completed" (null) differs from an empty string.
type InstallationInput struct {
	StartedOn   string  `json:"started_on" validate:"required,date"`
	TargetOn    string  `json:"target_on" validate:"required,date"`
	CompletedOn *string `json:"completed_on" validate:"omitempty,date"`
}

// Normalize trims the dates and treats an empty completion date as not completed.
// It must run before validation.
func (in *InstallationInput) Normalize() {
	in.StartedOn = strings.TrimSpace(in.StartedOn)
	in.TargetOn = strings.TrimSpace(in.TargetOn)
	if in.CompletedOn != nil {
		c := strings.TrimSpace(*in.CompletedOn)
		if c == "" {
			in.CompletedOn = nil
		} else {
			in.CompletedOn = &c
		}
	}
}
