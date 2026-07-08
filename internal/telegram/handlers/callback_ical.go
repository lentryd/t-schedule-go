package handlers

import (
	"context"
	"fmt"

	"t-schedule/internal/telegram/messagemanager"
	"t-schedule/internal/telegram/messages"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// callbackICal sends Apple iCalendar subscription instructions, mirroring
// methods/callback.ical.ts.
func (d *Deps) callbackICal(ctx context.Context, b *tgbot.Bot, cq *models.CallbackQuery) {
	chatID := cq.Message.Message.Chat.ID
	session := state.SessionFromContext(ctx)
	user := state.UserFromContext(ctx)

	if user.CalendarID == "" {
		if err := messages.CalendarInfo(ctx, b, chatID, user, session, cq, ""); err != nil {
			logError(ctx, "callbackICal.calendarInfo", err)
		}
		return
	}

	messagemanager.ClearMessagesAfter(ctx, b, chatID, session, cq)

	link := fmt.Sprintf("https://calendar.google.com/calendar/ical/%s/public/basic.ics", user.CalendarID)
	caption := fmt.Sprintf(
		"Инструкция по добавлению календаря в Apple iCalendar\n\n1. Скопируйте ссылку вашего календаря:\n%s\n2. В приложении \"Календарь\" нажмите \"Календари\"\n3. Нажмите \"Добавить\n4. Нажмите \"Добавить подписной календарь\"\n5. Вставьте ссылку на календарь\n6. Нажмите \"Подписаться\"\n7. Нажмите \"Добавить\"",
		link,
	)

	msg, err := b.SendVideo(ctx, &tgbot.SendVideoParams{
		ChatID:  chatID,
		Video:   &models.InputFileString{Data: "https://drive.google.com/uc?id=1IXNCIeRYVuL624KDmE2wLOFsKPOUTb_Z&export=download"},
		Caption: caption,
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Google Calendar", URL: fmt.Sprintf("https://calendar.google.com/calendar/render?cid=%s", user.CalendarID)}},
		}},
	})
	if err != nil {
		logError(ctx, "callbackICal.sendVideo", err)
		return
	}
	messagemanager.TrackSentMessage(session, msg.ID)
}
