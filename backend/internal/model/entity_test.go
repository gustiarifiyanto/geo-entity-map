package model

import (
	"math"
	"testing"
)

func TestEntityTypeValid(t *testing.T) {
	for _, typ := range EntityTypes {
		if !typ.Valid() {
			t.Errorf("EntityType(%q).Valid() = false, want true", typ)
		}
	}
	for _, typ := range []EntityType{"", "car", "Vehicle", " vehicle"} {
		if typ.Valid() {
			t.Errorf("EntityType(%q).Valid() = true, want false", typ)
		}
	}
}

func TestRoleValid(t *testing.T) {
	for _, r := range Roles {
		if !r.Valid() {
			t.Errorf("Role(%q).Valid() = false, want true", r)
		}
	}
	for _, r := range []Role{"", "root", "Admin", "admin "} {
		if r.Valid() {
			t.Errorf("Role(%q).Valid() = true, want false", r)
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got, want := NormalizeEmail("  Budi@Example.COM \t"), "budi@example.com"; got != want {
		t.Errorf("NormalizeEmail = %q, want %q", got, want)
	}
}

func TestEntityStatusValid(t *testing.T) {
	for _, s := range EntityStatuses {
		if !s.Valid() {
			t.Errorf("EntityStatus(%q).Valid() = false, want true", s)
		}
	}
	for _, s := range []EntityStatus{"", "broken", "Active", "active "} {
		if s.Valid() {
			t.Errorf("EntityStatus(%q).Valid() = true, want false", s)
		}
	}
}

func TestTypeCapabilities(t *testing.T) {
	for _, typ := range []EntityType{TypeFacility, TypeIoTDevice} {
		if !typ.Has(CapInstallation) {
			t.Errorf("%q should have the installation capability", typ)
		}
	}
	for _, typ := range []EntityType{TypeVehicle, "unknown"} {
		if typ.Has(CapInstallation) {
			t.Errorf("%q should not have the installation capability", typ)
		}
	}
	for typ := range TypeCapabilities {
		if !typ.Valid() {
			t.Errorf("TypeCapabilities lists unknown type %q", typ)
		}
	}
}

func TestDistanceMeters(t *testing.T) {
	// One degree along a meridian is π/180 × R.
	if got, want := DistanceMeters(0, 0, 1, 0), 111_195.08; math.Abs(got-want) > 0.1 {
		t.Errorf("1° of latitude = %.2f m, want %.2f", got, want)
	}
	if got := DistanceMeters(-6.2, 106.8, -6.2, 106.8); got != 0 {
		t.Errorf("same point = %v, want 0", got)
	}
	// Monas to Bundaran HI, Jakarta: about 2.2 km.
	if got := DistanceMeters(-6.175392, 106.827153, -6.195016, 106.822970); got < 2_200 || got > 2_260 {
		t.Errorf("Monas → Bundaran HI = %.0f m, want ≈2 230 m", got)
	}
	// Symmetric.
	if a, b := DistanceMeters(-6.1, 106.7, -6.9, 107.6), DistanceMeters(-6.9, 107.6, -6.1, 106.7); math.Abs(a-b) > 1e-6 {
		t.Errorf("not symmetric: %v vs %v", a, b)
	}
}

func TestZoneStatusEdge(t *testing.T) {
	const radius = 5_000.0
	// A point exactly `radius` north of the center (on a meridian).
	degPerMeter := 180 / (math.Pi * earthRadiusMeters)
	onEdge := radius * degPerMeter
	if d, inside := ZoneStatus(0, 0, radius, onEdge, 0); !inside || d != radius {
		t.Errorf("on the edge: distance %v, inside %v; want %v, true", d, inside, radius)
	}
	if d, inside := ZoneStatus(0, 0, radius, (radius+0.2)*degPerMeter, 0); inside {
		t.Errorf("0.2 m past the edge: distance %v counted as inside", d)
	}
}
