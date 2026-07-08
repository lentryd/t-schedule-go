// Package store wraps Firestore access for users, sessions, providers and
// the cached student list (mirrors src/utils/database.ts).
package store

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DetectProjectID instructs New to detect the Firestore project ID from the
// service account credentials instead of a hardcoded value.
const DetectProjectID = firestore.DetectProjectID

const (
	usersCollection     = "users"
	sessionsCollection  = "sessions"
	providersCollection = "providers"
	sysCollection       = "sys"
	studentListDocID    = "studentList"
)

// Store provides access to all Firestore-backed data used by the bot.
type Store struct {
	client *firestore.Client

	mu          sync.RWMutex
	studentList []Student
}

// New creates a Firestore-backed Store using the given service account
// credentials file (same file used for Google Calendar).
func New(ctx context.Context, projectID, credentialsPath string) (*Store, error) {
	client, err := firestore.NewClient(ctx, projectID, option.WithAuthCredentialsFile(option.ServiceAccount, credentialsPath))
	if err != nil {
		return nil, err
	}

	s := &Store{client: client}
	go s.watchStudentList(ctx)

	return s, nil
}

func (s *Store) Close() error {
	return s.client.Close()
}

// watchStudentList keeps the in-memory student list cache updated, mirroring
// the onSnapshot() subscription in database.ts.
func (s *Store) watchStudentList(ctx context.Context) {
	docRef := s.client.Collection(sysCollection).Doc(studentListDocID)

	it := docRef.Snapshots(ctx)
	defer it.Stop()

	for {
		snap, err := it.Next()
		if err != nil {
			if err != iterator.Done {
				slog.Error("studentList snapshot listener stopped", "error", err)
			}
			return
		}
		if !snap.Exists() {
			continue
		}

		var data studentListDoc
		if err := snap.DataTo(&data); err != nil {
			slog.Error("failed to decode studentList snapshot", "error", err)
			continue
		}

		s.mu.Lock()
		s.studentList = data.List
		s.mu.Unlock()
	}
}

// StudentList returns a snapshot of the cached student list.
func (s *Store) StudentList() []Student {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Student, len(s.studentList))
	copy(out, s.studentList)
	return out
}

// FindStudent looks up a cached student by ID.
func (s *Store) FindStudent(id int64) (Student, bool) {
	for _, student := range s.StudentList() {
		if student.ID == id {
			return student, true
		}
	}
	return Student{}, false
}

// SetStudentList overwrites the sys/studentList document.
func (s *Store) SetStudentList(ctx context.Context, students []Student, timestamp time.Time) error {
	_, err := s.client.Collection(sysCollection).Doc(studentListDocID).Set(ctx, studentListDoc{
		List:      students,
		Timestamp: timestamp,
	})
	return err
}

// User returns the user document for the given Telegram user ID, or the
// zero value if it doesn't exist yet.
func (s *Store) User(ctx context.Context, userID int64) (UserData, error) {
	doc, err := s.userDoc(userID).Get(ctx)
	if isNotFound(err) {
		return UserData{}, nil
	}
	if err != nil {
		return UserData{}, err
	}

	var data UserData
	if err := doc.DataTo(&data); err != nil {
		return UserData{}, err
	}
	return data, nil
}

// SaveUser persists the user document.
func (s *Store) SaveUser(ctx context.Context, userID int64, data UserData) error {
	_, err := s.userDoc(userID).Set(ctx, data)
	return err
}

func (s *Store) userDoc(userID int64) *firestore.DocumentRef {
	return s.client.Collection(usersCollection).Doc(userKey(userID))
}

// Session returns the session document for the given Telegram user ID, or a
// zero-value session with an empty state if it doesn't exist yet.
func (s *Store) Session(ctx context.Context, userID int64) (SessionData, error) {
	doc, err := s.sessionDoc(userID).Get(ctx)
	if isNotFound(err) {
		return SessionData{RecentMessageIDs: []int64{}, CommandMessageIDs: []int64{}}, nil
	}
	if err != nil {
		return SessionData{}, err
	}

	var data SessionData
	if err := doc.DataTo(&data); err != nil {
		return SessionData{}, err
	}
	return data, nil
}

// SaveSession persists the session document.
func (s *Store) SaveSession(ctx context.Context, userID int64, data SessionData) error {
	_, err := s.sessionDoc(userID).Set(ctx, data)
	return err
}

func (s *Store) sessionDoc(userID int64) *firestore.DocumentRef {
	return s.client.Collection(sessionsCollection).Doc(userKey(userID))
}

// ProviderRecord pairs a provider document with its Firestore document ID.
type ProviderRecord struct {
	ID   string
	Data ProviderData
}

// AddProvider creates a new provider document, mirroring providersCollection.add().
func (s *Store) AddProvider(ctx context.Context, data ProviderData) (string, error) {
	doc, _, err := s.client.Collection(providersCollection).Add(ctx, data)
	if err != nil {
		return "", err
	}
	return doc.ID, nil
}

// UpdateProviderAccessToken updates only the accessToken field of a provider.
func (s *Store) UpdateProviderAccessToken(ctx context.Context, providerID, accessToken string) error {
	_, err := s.client.Collection(providersCollection).Doc(providerID).Update(ctx, []firestore.Update{
		{Path: "accessToken", Value: accessToken},
	})
	return err
}

// ProvidersByEducationSpace returns all providers for the given education space.
func (s *Store) ProvidersByEducationSpace(ctx context.Context, spaceID int64) ([]ProviderRecord, error) {
	iter := s.client.Collection(providersCollection).Where("educationSpaceId", "==", spaceID).Documents(ctx)
	defer iter.Stop()
	return collectProviders(iter)
}

// AllProviders returns every provider document.
func (s *Store) AllProviders(ctx context.Context) ([]ProviderRecord, error) {
	iter := s.client.Collection(providersCollection).Documents(ctx)
	defer iter.Stop()
	return collectProviders(iter)
}

func collectProviders(iter *firestore.DocumentIterator) ([]ProviderRecord, error) {
	var out []ProviderRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var data ProviderData
		if err := doc.DataTo(&data); err != nil {
			return nil, err
		}
		out = append(out, ProviderRecord{ID: doc.Ref.ID, Data: data})
	}
	return out, nil
}

// UserRecord pairs a user document with its Firestore document ID.
type UserRecord struct {
	ID   string
	Data UserData
}

// UsersToUpdate returns users whose lastScheduleUpdate is missing/null or
// older than the threshold, mirroring the Filter.or() in
// synchronizeCalendar.ts's getUsersToUpdate().
func (s *Store) UsersToUpdate(ctx context.Context, threshold time.Time) ([]UserRecord, error) {
	q := s.client.Collection(usersCollection).WhereEntity(firestore.OrFilter{
		Filters: []firestore.EntityFilter{
			firestore.PropertyFilter{Path: "lastScheduleUpdate", Operator: "==", Value: nil},
			firestore.PropertyFilter{Path: "lastScheduleUpdate", Operator: "<", Value: threshold},
		},
	})
	iter := q.Documents(ctx)
	defer iter.Stop()

	var out []UserRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var data UserData
		if err := doc.DataTo(&data); err != nil {
			return nil, err
		}
		out = append(out, UserRecord{ID: doc.Ref.ID, Data: data})
	}
	return out, nil
}

// UpdateUserScheduleState updates raspHash + lastScheduleUpdate for a user by document ID.
func (s *Store) UpdateUserScheduleState(ctx context.Context, docID, raspHash string, lastUpdate time.Time) error {
	_, err := s.client.Collection(usersCollection).Doc(docID).Update(ctx, []firestore.Update{
		{Path: "raspHash", Value: raspHash},
		{Path: "lastScheduleUpdate", Value: lastUpdate},
	})
	return err
}

func userKey(userID int64) string {
	return strconv.FormatInt(userID, 10)
}

func isNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}
