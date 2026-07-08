package telegram

import (
	"context"
	"log/slog"

	"t-schedule/internal/store"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// SessionUserMiddleware loads the caller's session and user documents from
// Firestore into the context before running the handler, and persists them
// afterwards if the handler didn't error. Mirrors firestoreMiddleware() in
// utils/middleware.ts (one instance per collection there; combined here
// since both are always needed together).
func SessionUserMiddleware(st *store.Store) tgbot.Middleware {
	return func(next tgbot.HandlerFunc) tgbot.HandlerFunc {
		return func(ctx context.Context, b *tgbot.Bot, update *models.Update) {
			userID, ok := state.UserIDFromUpdate(update)
			if !ok {
				next(ctx, b, update)
				return
			}

			session, err := st.Session(ctx, userID)
			if err != nil {
				slog.Error("middleware: failed to load session", "user", userID, "error", err)
				return
			}
			user, err := st.User(ctx, userID)
			if err != nil {
				slog.Error("middleware: failed to load user", "user", userID, "error", err)
				return
			}

			ctx = state.WithSession(ctx, &session)
			ctx = state.WithUser(ctx, &user)

			next(ctx, b, update)

			if err := st.SaveSession(ctx, userID, session); err != nil {
				slog.Error("middleware: failed to save session", "user", userID, "error", err)
			}
			if err := st.SaveUser(ctx, userID, user); err != nil {
				slog.Error("middleware: failed to save user", "user", userID, "error", err)
			}
		}
	}
}
