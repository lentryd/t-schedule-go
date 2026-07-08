// Package messages renders the bot's reusable reply screens, mirroring
// src/messages/*.ts.
package messages

import (
	"context"
	"fmt"

	"t-schedule/internal/store"
	"t-schedule/internal/telegram/messagemanager"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// CalendarInfo sends (or edits, if possible) the main "your calendar" screen,
// mirroring messages/calendarInfo.ts.
func CalendarInfo(ctx context.Context, b *tgbot.Bot, chatID int64, user *store.UserData, session *store.SessionData, cq *models.CallbackQuery, additionalMessage string) error {
	if user.CalendarID != "" {
		session.State = "done"
	} else {
		session.State = "set_student"
	}

	if additionalMessage != "" {
		additionalMessage += "\n\n"
	}

	welcome := "Я помогу вам перенести ваше расписание в Google Calendar и Apple iCalendar."
	if user.CalendarID != "" {
		welcome = "С помощью кнопок ниже, вы можете легко добавить ваше расписание в Google Calendar или Apple iCalendar."
	}

	text := additionalMessage + welcome
	keyboard := calendarInfoKeyboard(user)

	if messagemanager.CanEditMessage(session, cq) {
		_, err := b.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   cq.Message.Message.ID,
			Text:        text,
			ReplyMarkup: keyboard,
		})
		return err
	}

	messagemanager.ClearMessagesAfter(ctx, b, chatID, session, cq)

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

func calendarInfoKeyboard(user *store.UserData) models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton

	if user.CalendarID != "" {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "Google Calendar", URL: fmt.Sprintf("https://calendar.google.com/calendar/render?cid=%s", user.CalendarID)},
			{Text: "Apple iCalendar", CallbackData: "iCal"},
		})
	}

	if user.CalendarID != "" {
		empty := ""
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "Поделиться расписанием", SwitchInlineQuery: &empty},
		})
	} else {
		empty := ""
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "Перенести расписание", SwitchInlineQueryCurrentChat: &empty},
		})
	}

	if user.CalendarID != "" && !user.HasEnteredEmail {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "Разукрасить календарь", CallbackData: "set_email"},
		})
	}

	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}
