package handlers

import (
	"context"

	"t-schedule/internal/telegram/messagemanager"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Colorize handles /colorize, mirroring methods/command.colorize.ts.
func (d *Deps) Colorize(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	msg := update.Message
	session := state.SessionFromContext(ctx)
	user := state.UserFromContext(ctx)

	messagemanager.TrackIncoming(ctx, b, msg.Chat.ID, session, msg, "colorize")

	if user.CalendarID == "" {
		empty := ""
		d.reply(ctx, b, msg.Chat.ID, session,
			"Эта команда предназначена для настройки цветов в вашем календаре. Однако, если у вас еще нет календаря, вы можете создать его, введя команду /student или воспользовавшись кнопкой ниже.",
			models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "Создать календарь", SwitchInlineQueryCurrentChat: &empty}},
			}})
		return
	}

	session.State = state.StateSetEmail

	if user.HasEnteredEmail {
		d.reply(ctx, b, msg.Chat.ID, session,
			"Цвета в вашем календаре уже настроены. Однако, если вы хотите добавить еще одного пользователя, нажмите кнопку ниже.",
			models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Добавить пользователя", CallbackData: "set_email"},
					{Text: "Отмена", CallbackData: "cancel"},
				},
			}})
		return
	}

	d.reply(ctx, b, msg.Chat.ID, session,
		"Для настройки цветов в вашем календаре, введите адрес электронной почты, привязанный к вашему Google-аккаунту, который используется для управления календарем.",
		models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Отмена", CallbackData: "cancel"}},
		}})
}
