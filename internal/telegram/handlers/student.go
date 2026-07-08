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

// Student handles /student <id>, mirroring methods/command.student.ts.
func (d *Deps) Student(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	msg := update.Message
	session := state.SessionFromContext(ctx)
	user := state.UserFromContext(ctx)

	messagemanager.TrackIncoming(ctx, b, msg.Chat.ID, session, msg, "student")

	session.State = state.StateSetStudent

	studentID, err := strconv.ParseInt(commandPayload(msg.Text), 10, 64)
	if err != nil {
		empty := ""
		d.reply(ctx, b, msg.Chat.ID, session,
			"Для указания идентификатора студента, введите команду /student и после неё укажите числовой идентификатор студента.\n\nПример:\n/student 123456",
			models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: "Идентификация по фамилии", SwitchInlineQueryCurrentChat: &empty}},
			}})
		return
	}

	student, ok := d.Store.FindStudent(studentID)
	if !ok {
		d.reply(ctx, b, msg.Chat.ID, session,
			"Студент с указанным идентификатором не найден в базе данных. Пожалуйста, попробуйте выполнить аутентификацию с помощью команды /auth", models.InlineKeyboardMarkup{})
		return
	}

	var currentStudent *store.Student
	if user.StudentID != 0 {
		if s, ok := d.Store.FindStudent(user.StudentID); ok {
			currentStudent = &s
		}
	}

	viaBot := msg.ViaBot != nil

	if err := messages.ConfirmStudent(ctx, b, msg.Chat.ID, session, currentStudent, student, viaBot); err != nil {
		logError(ctx, "student.confirm", err)
	}
}
