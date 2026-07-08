package handlers

import (
	"context"
	"strconv"
	"strings"

	"t-schedule/internal/telegram/messages"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// inlineStudent answers an inline query while the bot is waiting for a
// student pick, mirroring methods/inline.student.ts.
func (d *Deps) inlineStudent(ctx context.Context, b *tgbot.Bot, iq *models.InlineQuery) {
	query := strings.TrimSpace(iq.Query)

	if query == "" {
		_, _ = b.AnswerInlineQuery(ctx, &tgbot.AnswerInlineQueryParams{
			InlineQueryID: iq.ID,
			Results: []models.InlineQueryResult{
				&models.InlineQueryResultArticle{
					ID:                  "tips",
					Title:               "Начните вводить свою фамилию",
					InputMessageContent: models.InputTextMessageContent{MessageText: "/start"},
				},
			},
		})
		return
	}

	all := d.Store.StudentList()
	var matches []models.InlineQueryResult
	lowerQuery := strings.ToLower(query)

	for _, student := range all {
		if len(matches) >= 10 {
			break
		}
		if strings.HasPrefix(strings.ToLower(student.FullName), lowerQuery) {
			_, department := messages.FormatStudent(student)
			id := strconv.FormatInt(student.ID, 10)
			matches = append(matches, &models.InlineQueryResultArticle{
				ID:                  id,
				Title:               student.FullName,
				Description:         department,
				InputMessageContent: models.InputTextMessageContent{MessageText: "/student " + id},
			})
		}
	}

	if len(matches) == 0 {
		matches = []models.InlineQueryResult{
			&models.InlineQueryResultArticle{
				ID:                  "tips",
				Title:               "Не удалось найти студента 😔",
				Description:         "Если фамилия введена правильно, то нажмите сюда",
				InputMessageContent: models.InputTextMessageContent{MessageText: "/auth"},
			},
		}
	}

	_, _ = b.AnswerInlineQuery(ctx, &tgbot.AnswerInlineQueryParams{InlineQueryID: iq.ID, Results: matches})
}
