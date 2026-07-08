package handlers

import (
	"context"
	"strings"

	"t-schedule/internal/pkg/eduapi"
	"t-schedule/internal/store"
	"t-schedule/internal/telegram/messagemanager"
	"t-schedule/internal/telegram/messages"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Auth handles /auth <login> <password>, mirroring methods/command.auth.ts.
func (d *Deps) Auth(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	msg := update.Message
	session := state.SessionFromContext(ctx)
	user := state.UserFromContext(ctx)

	messagemanager.TrackIncoming(ctx, b, msg.Chat.ID, session, msg, "auth")

	fields := strings.Fields(commandPayload(msg.Text))
	if len(fields) != 2 {
		d.reply(ctx, b, msg.Chat.ID, session,
			"Для авторизации, введите команду /auth с вашим логином и паролем от личного кабинета на edu.donstu.ru, в формате `/auth login password`, где `login` - ваш логин, а `password` - ваш пароль.\n\nПример:\n/auth example@gmail.com 123456",
			models.InlineKeyboardMarkup{})
		return
	}
	userName, password := fields[0], fields[1]

	waitMsg, err := b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: msg.Chat.ID, Text: "Проверяю данные, пожалуйста, подождите"})
	if err != nil {
		logError(ctx, "auth.wait", err)
		return
	}

	result, err := eduapi.TryAuth(userName, password)
	if err != nil || result == nil {
		d.handleAuthError(ctx, b, msg.Chat.ID, waitMsg.ID, session)
		return
	}

	user.StudentID = result.StudentID
	user.EducationSpaceID = result.SpaceID

	providerID, err := d.Store.AddProvider(ctx, store.ProviderData{
		UserID:           msg.From.ID,
		UserName:         userName,
		Password:         password,
		AccessToken:      result.AccessToken,
		EducationSpaceID: result.SpaceID,
	})
	if err != nil {
		logError(ctx, "auth.addProvider", err)
	} else if err := d.StudentUpdater.Run(ctx, true); err != nil {
		logError(ctx, "auth.updateStudentList", err)
	}
	_ = providerID

	summary := msg.From.FirstName
	if student, ok := d.Store.FindStudent(result.StudentID); ok && student.ShortName != "" {
		summary = student.ShortName
	}

	if user.CalendarID == "" {
		calendarID, err := d.GCal.CreateCalendar(ctx, summary)
		if err != nil {
			logError(ctx, "auth.createCalendar", err)
			d.handleAuthError(ctx, b, msg.Chat.ID, waitMsg.ID, session)
			return
		}
		user.CalendarID = calendarID
	}

	if _, err := d.GCal.CreateUserRule(ctx, user.CalendarID, userName); err != nil {
		logError(ctx, "auth.createRule", err)
	}

	_, _ = b.DeleteMessage(ctx, &tgbot.DeleteMessageParams{ChatID: msg.Chat.ID, MessageID: waitMsg.ID})

	if err := messages.CalendarInfo(ctx, b, msg.Chat.ID, user, session, nil, "Вы успешно авторизовались!"); err != nil {
		logError(ctx, "auth.calendarInfo", err)
	}
}

func (d *Deps) handleAuthError(ctx context.Context, b *tgbot.Bot, chatID int64, waitMessageID int, session *store.SessionData) {
	const errText = "Произошла ошибка 😔\nПопробуйте повторить попытку позже."
	keyboard := models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "Отмена", CallbackData: "cancel"}},
	}}

	if _, err := b.EditMessageText(ctx, &tgbot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   waitMessageID,
		Text:        errText,
		ReplyMarkup: keyboard,
	}); err != nil {
		_, _ = b.DeleteMessage(ctx, &tgbot.DeleteMessageParams{ChatID: chatID, MessageID: waitMessageID})
		d.reply(ctx, b, chatID, session, errText, keyboard)
	}
}
