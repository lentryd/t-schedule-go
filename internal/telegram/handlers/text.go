package handlers

import (
	"context"

	"t-schedule/internal/telegram/messagemanager"
	"t-schedule/internal/telegram/messages"
	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Text dispatches plain text messages by session state, mirroring
// methods/text.common.ts.
func (d *Deps) Text(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	msg := update.Message
	session := state.SessionFromContext(ctx)

	messagemanager.TrackIncoming(ctx, b, msg.Chat.ID, session, msg, "")

	switch session.State {
	case state.StateSetEmail:
		d.textEmail(ctx, b, update)
	default:
		if err := messages.CalendarInfo(ctx, b, msg.Chat.ID, state.UserFromContext(ctx), session, nil, "Простите, я не понимаю вас. Попробуйте снова"); err != nil {
			logError(ctx, "text.default", err)
		}
	}
}
