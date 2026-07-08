package handlers

import (
	"context"
	"regexp"

	"t-schedule/internal/telegram/messages"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

// textEmail handles the "waiting for an email address" state, mirroring
// methods/text.email.ts.
func (d *Deps) textEmail(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	msg := update.Message
	session := state.SessionFromContext(ctx)
	user := state.UserFromContext(ctx)

	if user.CalendarID == "" {
		empty := ""
		d.reply(ctx, b, msg.Chat.ID, session,
			"Похоже у вас нет календаря. Однако вы можете создать его, введя команду /student или воспользовавшись кнопкой ниже.",
			models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "Создать календарь", SwitchInlineQueryCurrentChat: &empty}},
			}})
		return
	}

	email := emailRegex.FindString(msg.Text)
	if email == "" {
		d.reply(ctx, b, msg.Chat.ID, session,
			"Для настройки цветов в вашем календаре, введите адрес электронной почты, привязанный к вашему Google-аккаунту, который используется для управления календарем.\n\nПример:\nexample@gmail.com",
			models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "Отмена", CallbackData: "cancel"}},
			}})
		return
	}

	waitMsg, err := b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: msg.Chat.ID, Text: "Добавляю почту, пожалуйста, подождите..."})
	if err != nil {
		logError(ctx, "textEmail.wait", err)
		return
	}

	if _, err := d.GCal.CreateUserRule(ctx, user.CalendarID, email); err != nil {
		logError(ctx, "textEmail.createRule", err)
		_, _ = b.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:    msg.Chat.ID,
			MessageID: waitMsg.ID,
			Text:      "Произошла ошибка 😔\nПопробуйте повторить попытку позже.",
			ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "Отмена", CallbackData: "cancel"}},
			}},
		})
		return
	}

	user.HasEnteredEmail = true
	_, _ = b.DeleteMessage(ctx, &tgbot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: waitMsg.ID})

	if err := messages.CalendarInfo(ctx, b, msg.Chat.ID, user, session, nil, "Теперь расписание в профиле "+email+" будет цветное"); err != nil {
		logError(ctx, "textEmail.calendarInfo", err)
	}
}
