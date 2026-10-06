package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/model"
	"github.com/gustiarifiyanto/geo-entity-map/backend/internal/repository"
)

// InstallationRepository is the schedule storage the installation service depends on.
type InstallationRepository interface {
	Get(ctx context.Context, entityID string) (repository.InstallationRecord, bool, error)
	List(ctx context.Context) ([]repository.InstallationRecord, error)
	Put(ctx context.Context, in model.InstallationInput, entityID string, updatedAt time.Time) error
	Delete(ctx context.Context, entityID string) error
}

// InstallationService manages facility installation schedules and computes
// their status for "today" in the app's time zone.
type InstallationService struct {
	repo     InstallationRepository
	entities EntityLookup
	loc      *time.Location
	now      func() time.Time
}

// NewInstallationService returns a service that decides what "today" is in loc.
func NewInstallationService(repo InstallationRepository, entities EntityLookup, loc *time.Location) *InstallationService {
	return &InstallationService{repo: repo, entities: entities, loc: loc, now: time.Now}
}

// Today is the current calendar day in the app's time zone, at midnight UTC so
// day arithmetic is exact.
func (s *InstallationService) Today() time.Time {
	y, m, d := s.now().In(s.loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Get returns an entity's schedule, nil when it has none, model.ErrNotFound
// for an unknown entity, or model.ErrNoInstallation if its type has no schedule.
func (s *InstallationService) Get(ctx context.Context, entityID string) (*model.Installation, error) {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return nil, err
	}
	rec, found, err := s.repo.Get(ctx, entityID)
	if err != nil || !found {
		return nil, err
	}
	inst, err := compute(rec, s.Today())
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// Put sets an entity's schedule from a validated input and returns it.
func (s *InstallationService) Put(ctx context.Context, entityID string, in model.InstallationInput) (model.Installation, error) {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return model.Installation{}, err
	}
	if err := s.repo.Put(ctx, in, entityID, s.now().UTC().Truncate(time.Second)); err != nil {
		return model.Installation{}, err
	}
	inst, err := s.Get(ctx, entityID)
	if err != nil {
		return model.Installation{}, err
	}
	if inst == nil {
		return model.Installation{}, fmt.Errorf("installation of %s missing right after saving it", entityID)
	}
	return *inst, nil
}

// Delete removes an entity's schedule. Removing a missing schedule is not an error.
func (s *InstallationService) Delete(ctx context.Context, entityID string) error {
	if err := s.checkEntity(ctx, entityID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, entityID)
}

// List returns every schedule of an entity whose type has the installation
// capability: overdue first (most days late first), then the rest by target date.
func (s *InstallationService) List(ctx context.Context) ([]model.Installation, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	today := s.Today()
	list := make([]model.Installation, 0, len(records))
	for _, rec := range records {
		// A schedule outlives a type change; it only counts while the type supports it.
		if !rec.EntityType.Has(model.CapInstallation) {
			continue
		}
		inst, err := compute(rec, today)
		if err != nil {
			return nil, err
		}
		list = append(list, inst)
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if (a.Status == model.InstallationOverdue) != (b.Status == model.InstallationOverdue) {
			return a.Status == model.InstallationOverdue
		}
		if a.DaysLate != b.DaysLate {
			return a.DaysLate > b.DaysLate
		}
		return a.TargetOn < b.TargetOn
	})
	return list, nil
}

func (s *InstallationService) checkEntity(ctx context.Context, entityID string) error {
	e, err := s.entities.Get(ctx, entityID)
	if err != nil {
		return err
	}
	if !e.Type.Has(model.CapInstallation) {
		return model.ErrNoInstallation
	}
	return nil
}

// compute derives the status and day counts of a schedule for today.
func compute(rec repository.InstallationRecord, today time.Time) (model.Installation, error) {
	started, err := time.Parse(model.DateLayout, rec.StartedOn)
	if err != nil {
		return model.Installation{}, fmt.Errorf("parse started_on of %s: %w", rec.EntityID, err)
	}
	target, err := time.Parse(model.DateLayout, rec.TargetOn)
	if err != nil {
		return model.Installation{}, fmt.Errorf("parse target_on of %s: %w", rec.EntityID, err)
	}

	inst := model.Installation{
		EntityID:    rec.EntityID,
		StartedOn:   rec.StartedOn,
		TargetOn:    rec.TargetOn,
		CompletedOn: rec.CompletedOn,
		PlannedDays: days(started, target),
		UpdatedAt:   rec.UpdatedAt,
	}

	// The day the work stopped counting: completion, or today if still running.
	end := today
	switch {
	case rec.CompletedOn != nil:
		if end, err = time.Parse(model.DateLayout, *rec.CompletedOn); err != nil {
			return model.Installation{}, fmt.Errorf("parse completed_on of %s: %w", rec.EntityID, err)
		}
		inst.Status = model.InstallationCompletedOnTime
		if end.After(target) {
			inst.Status = model.InstallationCompletedLate
		}
	case today.Before(started):
		inst.Status = model.InstallationScheduled
	case today.After(target):
		inst.Status = model.InstallationOverdue
	default:
		inst.Status = model.InstallationInProgress
	}
	inst.ElapsedDays = max(0, days(started, end))
	inst.DaysLate = max(0, days(target, end))
	return inst, nil
}

// days counts whole calendar days from a to b (both at midnight UTC).
func days(a, b time.Time) int {
	return int(b.Sub(a) / (24 * time.Hour))
}
