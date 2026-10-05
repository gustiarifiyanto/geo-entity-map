package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
)

type seedEntity struct {
	name        string
	typ         model.EntityType
	status      model.EntityStatus
	lat, lng    float64
	description string
	attributes  string // JSON object, or "" for none
}

var seedEntities = []seedEntity{
	{"Truck B 1234 XYZ", model.TypeVehicle, model.StatusActive, -6.2088, 106.8456,
		"Delivery truck, central Jakarta route", `{"plate":"B 1234 XYZ","capacity_kg":5000}`},
	{"Van B 9876 ABC", model.TypeVehicle, model.StatusMaintenance, -6.1751, 106.8650,
		"Scheduled engine service", `{"plate":"B 9876 ABC"}`},
	{"Air Quality Sensor #01", model.TypeIoTDevice, model.StatusActive, -6.2297, 106.8295,
		"PM2.5 and PM10 monitoring", `{"firmware":"1.4.2","battery_pct":87}`},
	{"Flood Sensor #07", model.TypeIoTDevice, model.StatusInactive, -6.1352, 106.8133,
		"Water level sensor near the canal", ""},
	{"Main Warehouse", model.TypeFacility, model.StatusActive, -6.1862, 106.7343,
		"Primary distribution center", `{"area_m2":12000}`},
	{"Bandung Depot", model.TypeFacility, model.StatusActive, -6.9175, 107.6191,
		"", ""},
}

// Seed inserts sample entities, but only when the entities table is empty.
func Seed(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entities`).Scan(&count); err != nil {
		return fmt.Errorf("count entities: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, e := range seedEntities {
		var attrs sql.NullString
		if e.attributes != "" {
			attrs = sql.NullString{String: e.attributes, Valid: true}
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO entities
				(id, name, type, status, latitude, longitude, description, attributes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			uuid.NewString(), e.name, e.typ, e.status, e.lat, e.lng, e.description, attrs, now, now)
		if err != nil {
			return fmt.Errorf("insert seed entity %q: %w", e.name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}
