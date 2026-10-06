package service

import (
	"testing"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/repository"
)

func date(s string) time.Time {
	t, err := time.Parse(model.DateLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestComputeStatus(t *testing.T) {
	today := date("2026-10-06")
	completed := func(s string) *string { return &s }
	tests := []struct {
		name                     string
		start, target            string
		done                     *string
		status                   model.InstallationStatus
		planned, elapsed, behind int
	}{
		{"starts tomorrow", "2026-10-07", "2026-10-20", nil, model.InstallationScheduled, 13, 0, 0},
		{"starts today", "2026-10-06", "2026-10-20", nil, model.InstallationInProgress, 14, 0, 0},
		{"target is today", "2026-10-01", "2026-10-06", nil, model.InstallationInProgress, 5, 5, 0},
		{"target was yesterday", "2026-10-01", "2026-10-05", nil, model.InstallationOverdue, 4, 5, 1},
		{"ten days late", "2026-09-01", "2026-09-26", nil, model.InstallationOverdue, 25, 35, 10},
		{"completed on the target day", "2026-10-01", "2026-10-04", completed("2026-10-04"), model.InstallationCompletedOnTime, 3, 3, 0},
		{"completed early", "2026-10-01", "2026-10-30", completed("2026-10-02"), model.InstallationCompletedOnTime, 29, 1, 0},
		{"completed a day after target", "2026-10-01", "2026-10-04", completed("2026-10-05"), model.InstallationCompletedLate, 3, 4, 1},
		// Completion wins over today's date: it was late, but it is done.
		{"completed late long ago", "2026-01-01", "2026-01-10", completed("2026-01-15"), model.InstallationCompletedLate, 9, 14, 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := compute(repository.InstallationRecord{
				EntityID: "e", StartedOn: tc.start, TargetOn: tc.target, CompletedOn: tc.done,
			}, today)
			if err != nil {
				t.Fatalf("compute: %v", err)
			}
			if got.Status != tc.status || got.PlannedDays != tc.planned || got.ElapsedDays != tc.elapsed || got.DaysLate != tc.behind {
				t.Errorf("got status=%s planned=%d elapsed=%d late=%d; want %s %d %d %d",
					got.Status, got.PlannedDays, got.ElapsedDays, got.DaysLate, tc.status, tc.planned, tc.elapsed, tc.behind)
			}
		})
	}
}

func TestTodayUsesTheAppTimeZone(t *testing.T) {
	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load Asia/Jakarta: %v", err)
	}
	// 18:00 UTC on the 5th is already 01:00 on the 6th in Jakarta (UTC+7).
	instant := time.Date(2026, 10, 5, 18, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		loc  *time.Location
		want string
	}{
		{jakarta, "2026-10-06"},
		{time.UTC, "2026-10-05"},
	} {
		s := NewInstallationService(nil, nil, tc.loc)
		s.now = func() time.Time { return instant }
		if got := s.Today().Format(model.DateLayout); got != tc.want {
			t.Errorf("Today() in %s = %s, want %s", tc.loc, got, tc.want)
		}
	}
}
