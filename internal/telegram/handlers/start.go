package handlers

import (
	"context"
	"strings"

	"t-schedule/internal/telegram/messagemanager"
	"t-schedule/internal/telegram/messages"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Start handles /start, mirroring methods/command.start.ts.
func (d *Deps) Start(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	msg := update.Message
	session := state.SessionFromContext(ctx)

	messagemanager.TrackIncoming(ctx, b, msg.Chat.ID, session, msg, "start")

	if err := messages.CalendarInfo(ctx, b, msg.Chat.ID, state.UserFromContext(ctx), session, nil, ""); err != nil {
		logError(ctx, "start", err)
	}
}

// commandPayload returns the text following "/command" (and an optional
// "@botname"), mirroring Telegraf's ctx.payload.
func commandPayload(text string) string {
	fields := strings.SplitN(text, " ", 2)
	if len(fields) < 2 {
		return ""
	}
	return strings.TrimSpace(fields[1])
}
