package handlers

import (
	"context"
	"fmt"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	shareTipTitle       = "Поделиться расписанием"
	shareTipDescription = "Нажмите, чтобы поделиться расписанием"
	shareTipMessage     = "Вот мое расписание. Если ты хочешь создать подобное, используй @t_schedule_bot."
)

// inlineShare answers an inline query with a "share my schedule" card,
// mirroring methods/inline.share.ts.
func (d *Deps) inlineShare(ctx context.Context, b *tgbot.Bot, iq *models.InlineQuery) {
	if iq.ChatType != "private" {
		return
	}

	user, err := d.Store.User(ctx, iq.From.ID)
	if err != nil || user.CalendarID == "" {
		return
	}

	_, _ = b.AnswerInlineQuery(ctx, &tgbot.AnswerInlineQueryParams{
		InlineQueryID: iq.ID,
		IsPersonal:    true,
		Results: []models.InlineQueryResult{
			&models.InlineQueryResultArticle{
				ID:                  "tip",
				Title:               shareTipTitle,
				Description:         shareTipDescription,
				InputMessageContent: models.InputTextMessageContent{MessageText: shareTipMessage},
				ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{Text: "Google Calendar", URL: fmt.Sprintf("https://calendar.google.com/calendar/render?cid=%s", user.CalendarID)},
						{Text: "Apple iCalendar", URL: fmt.Sprintf("https://calendar.google.com/calendar/ical/%s/public/basic.ics", user.CalendarID)},
					},
				}},
			},
		},
	})
}
