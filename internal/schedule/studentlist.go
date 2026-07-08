package schedule

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"t-schedule/internal/pkg/eduapi"
	"t-schedule/internal/store"
)

const studentListMaxAge = 24 * time.Hour

const (
	educationSpaceSchoolX     = 1
	educationSpaceTUniversity = 4
)

// StudentListUpdater refreshes the cached student list, mirroring
// updateStudentList()/fetchAndAddStudents() in synchronizeCalendar.ts.
type StudentListUpdater struct {
	store *store.Store

	lastUpdated time.Time
}

func NewStudentListUpdater(st *store.Store) *StudentListUpdater {
	return &StudentListUpdater{store: st}
}

// Run refreshes the student list unless it was updated recently, unless
// ignoreTime is set.
func (u *StudentListUpdater) Run(ctx context.Context, ignoreTime bool) error {
	if !ignoreTime && time.Since(u.lastUpdated) < studentListMaxAge {
		return nil
	}

	providers, err := u.store.AllProviders(ctx)
	if err != nil {
		return err
	}

	var schoolX, tUniversity []store.ProviderRecord
	for _, p := range providers {
		switch p.Data.EducationSpaceID {
		case educationSpaceSchoolX:
			schoolX = append(schoolX, p)
		case educationSpaceTUniversity:
			tUniversity = append(tUniversity, p)
		}
	}

	var students []store.Student
	students = append(students, fetchFromAny(ctx, u.store, schoolX)...)
	students = append(students, fetchFromAny(ctx, u.store, tUniversity)...)

	if err := u.store.SetStudentList(ctx, students, time.Now()); err != nil {
		return err
	}

	u.lastUpdated = time.Now()
	slog.Info("studentlist: updated", "students", len(students))
	return nil
}

// fetchFromAny tries providers in random order until one returns a non-empty
// student list, mirroring fetchAndAddStudents().
func fetchFromAny(ctx context.Context, st *store.Store, providers []store.ProviderRecord) []store.Student {
	remaining := append([]store.ProviderRecord(nil), providers...)

	for len(remaining) > 0 {
		idx := rand.Intn(len(remaining))
		provider := remaining[idx]
		remaining = append(remaining[:idx], remaining[idx+1:]...)

		client := eduapi.NewClient(st, provider.Data, provider.ID)
		students, err := client.GetStudentList(ctx)
		if err != nil {
			slog.Error("studentlist: error fetching from provider", "provider", provider.ID, "error", err)
			continue
		}
		if len(students) > 0 {
			return students
		}
	}
	return nil
}
