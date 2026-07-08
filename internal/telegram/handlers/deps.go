// Package handlers implements the bot's command, callback, inline and text
// handlers, mirroring src/methods/*.ts.
package handlers

import (
	"t-schedule/internal/pkg/gcalendar"
	"t-schedule/internal/schedule"
	"t-schedule/internal/store"
)

// Deps bundles everything a handler needs; passed to every handler
// constructor instead of relying on globals.
type Deps struct {
	Store          *store.Store
	GCal           *gcalendar.Client
	StudentUpdater *schedule.StudentListUpdater
}
