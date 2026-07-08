package handlers

import (
	"context"

	"t-schedule/internal/store"
	"t-schedule/internal/telegram/messagemanager"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// reply sends a plain text message and tracks it in the session, mirroring
// the common `ctx.reply(...).then((message) => messageManager(ctx, message))`
// pattern used throughout methods/*.ts.
func (d *Deps) reply(ctx context.Context, b *tgbot.Bot, chatID int64, session *store.SessionData, text string, keyboard models.InlineKeyboardMarkup) {
	msg, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		logError(ctx, "reply", err)
		return
	}
	messagemanager.TrackSentMessage(session, msg.ID)
}
