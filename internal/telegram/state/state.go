// Package state carries the per-update session/user data through
// context.Context, mirroring the ctx.session/ctx.user getters installed by
// utils/middleware.ts in the Node bot.
package state

import (
	"context"

	"t-schedule/internal/store"

	"github.com/go-telegram/bot/models"
)

// Session states, mirroring the `state` field in src/utils/database.ts /
// the various `session.state = '...'` assignments across methods/*.ts.
const (
	StateSetStudent = "set_student"
	StateSetEmail   = "set_email"
	StateDone       = "done"
)

type contextKey int

const (
	sessionKey contextKey = iota
	userKey
)

// WithSession stores a mutable *store.SessionData in ctx.
func WithSession(ctx context.Context, s *store.SessionData) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

// SessionFromContext returns the session loaded by the middleware for the
// current update. Panics if the middleware wasn't installed - this mirrors
// the Node code's assumption that ctx.session is always available.
func SessionFromContext(ctx context.Context) *store.SessionData {
	return ctx.Value(sessionKey).(*store.SessionData)
}

// WithUser stores a mutable *store.UserData in ctx.
func WithUser(ctx context.Context, u *store.UserData) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext returns the user loaded by the middleware for the current update.
func UserFromContext(ctx context.Context) *store.UserData {
	return ctx.Value(userKey).(*store.UserData)
}

// UserIDFromUpdate extracts the Telegram user ID that triggered the update,
// mirroring the docKey resolution in middleware.ts.
func UserIDFromUpdate(update *models.Update) (int64, bool) {
	switch {
	case update.Message != nil && update.Message.From != nil:
		return update.Message.From.ID, true
	case update.CallbackQuery != nil:
		return update.CallbackQuery.From.ID, true
	case update.InlineQuery != nil && update.InlineQuery.From != nil:
		return update.InlineQuery.From.ID, true
	default:
		return 0, false
	}
}
