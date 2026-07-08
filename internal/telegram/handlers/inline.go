package handlers

import (
	"context"

	"t-schedule/internal/telegram/state"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Inline dispatches inline queries by session state, mirroring
// methods/inline.common.ts.
func (d *Deps) Inline(ctx context.Context, b *tgbot.Bot, update *models.Update) {
	iq := update.InlineQuery
	if iq.ChatType != "sender" && iq.ChatType != "private" {
		return
	}

	session := state.SessionFromContext(ctx)

	if session.State == state.StateSetStudent {
		d.inlineStudent(ctx, b, iq)
		return
	}
	d.inlineShare(ctx, b, iq)
}
