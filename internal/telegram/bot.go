// Package telegram wires up the go-telegram/bot client: middleware,
// handlers, and the polling/webhook entrypoint, mirroring src/index.ts.
package telegram

import (
	"context"
	"log/slog"

	"t-schedule/internal/config"
	"t-schedule/internal/pkg/gcalendar"
	"t-schedule/internal/schedule"
	"t-schedule/internal/store"
	"t-schedule/internal/telegram/handlers"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// New creates and configures the bot, registering every command/callback/
// inline/text handler.
func New(cfg *config.Config, st *store.Store, gcal *gcalendar.Client, studentUpdater *schedule.StudentListUpdater) (*tgbot.Bot, error) {
	deps := &handlers.Deps{Store: st, GCal: gcal, StudentUpdater: studentUpdater}

	opts := []tgbot.Option{
		tgbot.WithMiddlewares(SessionUserMiddleware(st)),
		tgbot.WithDefaultHandler(defaultHandler(deps)),
	}
	if cfg.Debug {
		opts = append(opts, tgbot.WithDebug())
	}

	b, err := tgbot.New(cfg.BotToken, opts...)
	if err != nil {
		return nil, err
	}

	b.RegisterHandler(tgbot.HandlerTypeMessageText, "start", tgbot.MatchTypeCommand, deps.Start)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "auth", tgbot.MatchTypeCommand, deps.Auth)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "student", tgbot.MatchTypeCommand, deps.Student)
	b.RegisterHandler(tgbot.HandlerTypeMessageText, "colorize", tgbot.MatchTypeCommand, deps.Colorize)

	return b, nil
}

// defaultHandler routes every update that isn't one of the registered
// commands, mirroring the combination of bot.on(message('text')),
// bot.on(callbackQuery('data')), bot.on('inline_query') and the generic
// bot.on('message') fallback in index.ts.
func defaultHandler(deps *handlers.Deps) tgbot.HandlerFunc {
	return func(ctx context.Context, b *tgbot.Bot, update *models.Update) {
		switch {
		case update.CallbackQuery != nil:
			deps.Callback(ctx, b, update)
		case update.InlineQuery != nil:
			deps.Inline(ctx, b, update)
		case update.Message != nil && update.Message.Text != "":
			deps.Text(ctx, b, update)
		case update.Message != nil:
			deps.Fallback(ctx, b, update)
		default:
			slog.Debug("unhandled update", "update_id", update.ID)
		}
	}
}
