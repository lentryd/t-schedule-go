package messages

import (
	"context"
	"fmt"
	"strconv"

	"t-schedule/internal/store"
	"t-schedule/internal/telegram/messagemanager"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// FormatStudent renders a student's name + department, mirroring
// formatStudent() in utils/format.ts.
func FormatStudent(student store.Student) (fullName, department string) {
	dep := fmt.Sprintf("%d курс (Т-университет)", student.Course)
	if student.SpaceID == 1 {
		dep = fmt.Sprintf("%d курс (Школа Икс)", student.Course)
	}
	return student.FullName, dep
}

// ConfirmStudent asks the user to confirm the selected student, mirroring
// messages/confirmStudent.ts.
func ConfirmStudent(ctx context.Context, b *tgbot.Bot, chatID int64, session *store.SessionData, currentStudent *store.Student, student store.Student, viaBot bool) error {
	newName, newDept := FormatStudent(student)

	text := ""
	if currentStudent != nil {
		oldName, oldDept := FormatStudent(*currentStudent)
		text = fmt.Sprintf("%s\n%s\n\nЗаменить на\n\n", oldName, oldDept)
	}
	text += fmt.Sprintf("%s\n%s\n\nВсе верно?", newName, newDept)

	noButton := models.InlineKeyboardButton{Text: "Нет", CallbackData: "cancel"}
	if viaBot {
		empty := ""
		noButton = models.InlineKeyboardButton{Text: "Нет", SwitchInlineQueryCurrentChat: &empty}
	}

	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Да", CallbackData: strconv.FormatInt(student.ID, 10)},
				noButton,
			},
		},
	}

	msg, err := b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		return err
	}

	messagemanager.TrackSentMessage(session, msg.ID)
	return nil
}
