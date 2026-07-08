// Package schedule synchronizes student schedules from edu.donstu.ru into
// Google Calendar, mirroring src/synchronizeCalendar.ts.
package schedule

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"t-schedule/internal/pkg/eduapi"
	"t-schedule/internal/pkg/gcalendar"
	"t-schedule/internal/store"

	calendar "google.golang.org/api/calendar/v3"
)

const (
	msInHour       = time.Hour
	userUpdateTime = 10 * time.Minute
)

// Syncer synchronizes calendars for all users whose schedules are outdated.
type Syncer struct {
	store *store.Store
	gcal  *gcalendar.Client

	inProgress bool
}

func NewSyncer(st *store.Store, gc *gcalendar.Client) *Syncer {
	return &Syncer{store: st, gcal: gc}
}

// InProgress reports whether a sync run is currently active, mirroring the
// exported `isProgress` flag in synchronizeCalendar.ts.
func (s *Syncer) InProgress() bool {
	return s.inProgress
}

// Run synchronizes the calendar with the latest schedule data for all users
// whose schedule is stale. Mirrors synchronizeCalendar().
func (s *Syncer) Run(ctx context.Context) error {
	s.inProgress = true
	defer func() { s.inProgress = false }()

	start := time.Now()
	threshold := timestampThreshold()

	users, err := s.store.UsersToUpdate(ctx, threshold)
	if err != nil {
		return err
	}
	slog.Debug("sync: users fetched", "users", len(users))

	var skipped, unchanged, synced, failed int
	for _, user := range users {
		result, err := s.processUser(ctx, user)
		if err != nil {
			failed++
			slog.Error("sync: error processing user", "user", user.ID, "error", err)
			continue
		}
		switch result {
		case resultSkipped:
			skipped++
		case resultUnchanged:
			unchanged++
		case resultSynced:
			synced++
		}
	}

	slog.Info("sync: synchronization completed",
		"users", len(users),
		"skipped", skipped,
		"unchanged", unchanged,
		"synced", synced,
		"failed", failed,
		"duration", time.Since(start),
	)
	return nil
}

type userSyncResult int

const (
	resultSkipped userSyncResult = iota
	resultUnchanged
	resultSynced
)

func (s *Syncer) processUser(ctx context.Context, user store.UserRecord) (userSyncResult, error) {
	data := user.Data
	if data.CalendarID == "" || data.EducationSpaceID == 0 || data.StudentID == 0 {
		slog.Debug("sync: user skipped (incomplete profile)", "user", user.ID)
		return resultSkipped, nil
	}

	logPrefix := "user " + user.ID
	userStart := time.Now()

	currentHash, err := eduapi.GetRaspHash(data.StudentID)
	if err != nil {
		return resultSkipped, err
	}

	if data.RaspHash == currentHash {
		slog.Debug(logPrefix+": schedule unchanged", "duration", time.Since(userStart))
		return resultUnchanged, s.store.UpdateUserScheduleState(ctx, user.ID, data.RaspHash, time.Now())
	}

	raspList, err := s.fetchSchedule(ctx, data)
	if err != nil {
		slog.Warn(logPrefix+": falling back to reserve schedule", "error", err)
		raspList, err = eduapi.GetReserveRasp(data.StudentID)
		if err != nil {
			return resultSkipped, err
		}
	}

	events, err := s.gcal.ListEvents(ctx, data.CalendarID)
	if err != nil {
		return resultSkipped, err
	}

	existing := make([]eduapi.ScheduleFormat, 0, len(events))
	for _, e := range events {
		existing = append(existing, eduapi.FormatEvent(e))
	}

	var toCreate, toUpdate []eduapi.ScheduleFormat
	for _, item := range raspList {
		found := false
		for _, e := range existing {
			if e.RaspID == item.RaspID {
				found = true
				if e.Etag != item.Etag {
					item.ID = e.ID
					item.ColorID = e.ColorID
					toUpdate = append(toUpdate, item)
				}
				break
			}
		}
		if !found {
			toCreate = append(toCreate, item)
		}
	}

	var toDelete []string
	if len(raspList) > 0 {
		for _, e := range existing {
			stillPresent := false
			for _, item := range raspList {
				if item.RaspID == e.RaspID {
					stillPresent = true
					break
				}
			}
			if !stillPresent {
				toDelete = append(toDelete, e.ID)
			}
		}
	}

	for _, item := range toUpdate {
		if _, err := s.gcal.UpdateEvent(ctx, data.CalendarID, toGoogleEvent(item)); err != nil {
			slog.Error(logPrefix+": error updating event", "error", err)
		}
	}
	for _, item := range toCreate {
		if _, err := s.gcal.CreateEvent(ctx, data.CalendarID, toGoogleEvent(item)); err != nil {
			slog.Error(logPrefix+": error creating event", "error", err)
		}
	}
	for _, eventID := range toDelete {
		if err := s.gcal.DeleteEvent(ctx, data.CalendarID, eventID); err != nil {
			slog.Error(logPrefix+": error deleting event", "event", eventID, "error", err)
		}
	}

	slog.Debug(logPrefix+": schedule synced",
		"created", len(toCreate), "updated", len(toUpdate), "deleted", len(toDelete),
		"duration", time.Since(userStart))

	return resultSynced, s.store.UpdateUserScheduleState(ctx, user.ID, currentHash, time.Now())
}

// fetchSchedule picks a random provider for the user's education space and
// fetches the schedule through it, mirroring the random-provider selection
// in processUser().
func (s *Syncer) fetchSchedule(ctx context.Context, data store.UserData) ([]eduapi.ScheduleFormat, error) {
	providers, err := s.store.ProvidersByEducationSpace(ctx, data.EducationSpaceID)
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return nil, errNoProviders
	}

	provider := providers[rand.Intn(len(providers))]
	client := eduapi.NewClient(s.store, provider.Data, provider.ID)

	return client.GetRaspList(ctx, data.StudentID)
}

func toGoogleEvent(item eduapi.ScheduleFormat) *calendar.Event {
	return &calendar.Event{
		Id:          item.ID,
		Summary:     item.Summary,
		Location:    item.Location,
		Description: item.Description,
		ColorId:     item.ColorID,
		Start: &calendar.EventDateTime{
			DateTime: item.Start.DateTime,
			TimeZone: item.Start.TimeZone,
		},
		End: &calendar.EventDateTime{
			DateTime: item.End.DateTime,
			TimeZone: item.End.TimeZone,
		},
	}
}

// timestampThreshold mirrors getTimestampThreshold(): during Moscow working
// hours (7-18, Mon-Fri) schedules older than 10 minutes are refreshed,
// otherwise the window widens to 30 minutes.
func timestampThreshold() time.Time {
	now := time.Now()
	moscow := now.In(moscowLocation())

	if moscow.Hour() >= 7 && moscow.Hour() <= 18 && moscow.Weekday() != time.Sunday {
		return now.Add(-userUpdateTime)
	}
	return now.Add(-msInHour / 2)
}

func moscowLocation() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.FixedZone("Europe/Moscow", 3*60*60)
	}
	return loc
}

var errNoProviders = &noProvidersError{}

type noProvidersError struct{}

func (*noProvidersError) Error() string {
	return "schedule: no providers available for education space"
}
