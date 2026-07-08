package handlers

import (
	"context"

	"t-schedule/internal/telegram/messagemanager"
	"t-schedule/internal/telegram/messages"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Fallback handles any non-text message (photo, sticker, voice, ...),
// mirroring the generic bot.on('message', ...) handler in index.ts.
func (d *Deps) Fallback(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	msg := update.Message
	session := state.SessionFromContext(ctx)

	messagemanager.TrackIncoming(ctx, b, msg.Chat.ID, session, msg, "")

	if err := messages.CalendarInfo(ctx, b, msg.Chat.ID, state.UserFromContext(ctx), session, nil, "Простите, я не понимаю вас."); err != nil {
		logError(ctx, "fallback.calendarInfo", err)
	}
}
