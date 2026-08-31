package handlers

import (
	"context"

	"t-schedule/internal/store"
	"t-schedule/internal/telegram/messagemanager"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// replyMarkup returns nil for an empty keyboard: Telegram rejects
// `{"inline_keyboard": null}` with "field \"inline_keyboard\" must be of type
// Array", so the field has to be omitted entirely instead.
func replyMarkup(keyboard ...models.InlineKeyboardMarkup) models.ReplyMarkup {
	if len(keyboard) == 0 || len(keyboard[0].InlineKeyboard) == 0 {
		return nil
	}
	return keyboard[0]
}

// replyHTML is reply() for messages that carry HTML formatting, mirroring the
// `fmt`/`code()` templates from telegraf/format used in methods/*.ts.
func (d *Deps) replyHTML(ctx context.Context, b *tgbot.Bot, chatID int64, session *store.SessionData, text string, keyboard ...models.InlineKeyboardMarkup) {
	msg, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: replyMarkup(keyboard...),
	})
	if err != nil {
		logError(ctx, "replyHTML", err)
		return
	}
	messagemanager.TrackSentMessage(session, msg.ID)
}

// reply sends a plain text message and tracks it in the session, mirroring
// the common `ctx.reply(...).then((message) => messageManager(ctx, message))`
// pattern used throughout methods/*.ts.
func (d *Deps) reply(ctx context.Context, b *tgbot.Bot, chatID int64, session *store.SessionData, text string, keyboard ...models.InlineKeyboardMarkup) {
	msg, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: replyMarkup(keyboard...),
	})
	if err != nil {
		logError(ctx, "reply", err)
		return
	}
	messagemanager.TrackSentMessage(session, msg.ID)
}
