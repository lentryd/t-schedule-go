package handlers

import (
	"context"

	"t-schedule/internal/telegram/messagemanager"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const colorizePromptMessage = "Для настройки цветов в вашем календаре, введите адрес электронной почты, привязанный к вашему Google-аккаунту, который используется для управления календарем."

// callbackColorize handles the "set_email" callback, mirroring
// methods/callback.colorize.ts.
func (d *Deps) callbackColorize(ctx context.Context, b *tgbot.Bot, cq *models.CallbackQuery) {
	chatID := cq.Message.Message.Chat.ID
	session := state.SessionFromContext(ctx)

	session.State = state.StateSetEmail

	keyboard := models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "Отмена", CallbackData: "cancel"}},
	}}

	if messagemanager.CanEditMessage(session, cq) {
		_, _ = b.EditMessageText(ctx, &tgbot.EditMessageTextParams{ChatID: chatID, MessageID: cq.Message.Message.ID, Text: colorizePromptMessage, ReplyMarkup: keyboard})
		return
	}

	messagemanager.ClearMessagesAfter(ctx, b, chatID, session, cq)
	d.reply(ctx, b, chatID, session, colorizePromptMessage, keyboard)
}
