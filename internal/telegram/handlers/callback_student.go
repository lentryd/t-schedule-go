package handlers

import (
	"context"
	"strconv"

	"t-schedule/internal/store"
	"t-schedule/internal/telegram/messagemanager"
	"t-schedule/internal/telegram/messages"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const callbackErrorMessage = "Произошла ошибка 😔\nПопробуйте снова через некоторое время"

// callbackStudent handles the "choose this student" callback, mirroring
// methods/callback.student.ts.
func (d *Deps) callbackStudent(ctx context.Context, b *tgbot.Bot, cq *models.CallbackQuery) {
	chatID := cq.Message.Message.Chat.ID
	session := state.SessionFromContext(ctx)
	user := state.UserFromContext(ctx)

	studentID, err := strconv.ParseInt(cq.Data, 10, 64)
	if err != nil {
		d.callbackError(ctx, b, chatID, session, cq)
		return
	}

	student, ok := d.Store.FindStudent(studentID)
	if !ok {
		d.callbackError(ctx, b, chatID, session, cq)
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: cq.ID, Text: "Создаю календарь, пожалуйста, подождите"})

	const waitMessage = "Создаю календарь, пожалуйста, подождите"
	if messagemanager.CanEditMessage(session, cq) {
		_, _ = b.EditMessageText(ctx, &tgbot.EditMessageTextParams{ChatID: chatID, MessageID: cq.Message.Message.ID, Text: waitMessage})
	} else {
		messagemanager.ClearMessagesAfter(ctx, b, chatID, session, cq)
		msg, err := b.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: waitMessage})
		if err == nil {
			messagemanager.TrackSentMessage(session, msg.ID)
			cq.Message.Message = msg
		}
	}

	calendarID := user.CalendarID
	if calendarID == "" {
		created, err := d.GCal.CreateCalendar(ctx, student.ShortName)
		if err != nil {
			logError(ctx, "callbackStudent.createCalendar", err)
			d.callbackError(ctx, b, chatID, session, cq)
			return
		}
		calendarID = created
	}

	user.StudentID = studentID
	user.CalendarID = calendarID
	user.EducationSpaceID = student.SpaceID

	if err := messages.CalendarInfo(ctx, b, chatID, user, session, cq, "Ваш календарь готов!"); err != nil {
		logError(ctx, "callbackStudent.calendarInfo", err)
	}
}

func (d *Deps) callbackError(ctx context.Context, b *tgbot.Bot, chatID int64, session *store.SessionData, cq *models.CallbackQuery) {
	keyboard := models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		{{Text: "Попробовать снова", SwitchInlineQueryCurrentChat: strPtr("")}},
	}}

	if messagemanager.CanEditMessage(session, cq) {
		_, _ = b.EditMessageText(ctx, &tgbot.EditMessageTextParams{ChatID: chatID, MessageID: cq.Message.Message.ID, Text: callbackErrorMessage, ReplyMarkup: keyboard})
		return
	}

	messagemanager.ClearMessagesAfter(ctx, b, chatID, session, cq)
	d.reply(ctx, b, chatID, session, callbackErrorMessage, keyboard)
}

func strPtr(s string) *string { return &s }
