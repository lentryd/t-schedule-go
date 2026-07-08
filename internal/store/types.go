package store

import "time"

// UserData mirrors the `users` Firestore collection (src/utils/database.ts UserData).
type UserData struct {
	StudentID          int64     `firestore:"studentId,omitempty"`
	CalendarID         string    `firestore:"calendarId,omitempty"`
	EducationSpaceID   int64     `firestore:"educationSpaceId,omitempty"`
	RaspHash           string    `firestore:"raspHash,omitempty"`
	LastScheduleUpdate time.Time `firestore:"lastScheduleUpdate"`
	HasEnteredEmail    bool      `firestore:"hasEnteredEmail"`
}

// SessionData mirrors the `sessions` Firestore collection.
type SessionData struct {
	State             string  `firestore:"state"`
	RecentMessageIDs  []int64 `firestore:"recentMessageIds"`
	CommandMessageIDs []int64 `firestore:"commandMessageIds"`
}

// ProviderData mirrors the `providers` Firestore collection.
type ProviderData struct {
	UserID           int64  `firestore:"userId"`
	EducationSpaceID int64  `firestore:"educationSpaceId"`
	UserName         string `firestore:"userName"`
	Password         string `firestore:"password"`
	AccessToken      string `firestore:"accessToken,omitempty"`
}

// Student mirrors an entry in the sys/studentList document.
type Student struct {
	ID        int64  `firestore:"id"`
	Course    int64  `firestore:"course"`
	SpaceID   int64  `firestore:"spaceID"`
	FullName  string `firestore:"fullName"`
	ShortName string `firestore:"shortName"`
}

// studentListDoc is the shape of the sys/studentList document.
type studentListDoc struct {
	List      []Student `firestore:"list"`
	Timestamp time.Time `firestore:"timestamp"`
}
