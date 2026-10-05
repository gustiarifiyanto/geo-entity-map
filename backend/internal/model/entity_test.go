package model

import "testing"

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
