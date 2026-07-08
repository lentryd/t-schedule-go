package handlers

import (
	"context"

	"t-schedule/internal/telegram/messages"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Callback dispatches callback queries by data / session state, mirroring
// methods/callback.common.ts.
func (d *Deps) Callback(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq.Message.Message == nil {
		return
	}
	chatID := cq.Message.Message.Chat.ID
	session := state.SessionFromContext(ctx)
	user := state.UserFromContext(ctx)

	switch cq.Data {
	case "iCal":
		d.callbackICal(ctx, b, cq)
		return
	case "cancel":
		if err := messages.CalendarInfo(ctx, b, chatID, user, session, cq, "Действие отменено!"); err != nil {
			logError(ctx, "callback.cancel", err)
		}
		return
	}

	switch session.State {
	case state.StateSetStudent:
		d.callbackStudent(ctx, b, cq)
	case state.StateDone, state.StateSetEmail:
		if cq.Data == "set_email" {
			d.callbackColorize(ctx, b, cq)
		}
	default:
		_, _ = b.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
			CallbackQueryID: cq.ID,
			Text:            "Произошла ошибка, пожалуйста, повторите попытку",
			ShowAlert:       true,
		})
		if err := messages.CalendarInfo(ctx, b, chatID, user, session, cq, ""); err != nil {
			logError(ctx, "callback.default", err)
		}
	}
}
