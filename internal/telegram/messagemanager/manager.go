// Package messagemanager tracks recent message IDs per session so old bot
// messages can be cleaned up, mirroring src/utils/messageManager.ts.
package messagemanager

import (
	"context"
	"time"

	"t-schedule/internal/store"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const maxEditWindow = 5 * time.Minute

// CanEditMessage reports whether the message behind a callback query can
// still be edited in place, mirroring canEditMessage().
func CanEditMessage(session *store.SessionData, cq *models.CallbackQuery) bool {
	if cq == nil || cq.Message.Message == nil {
		return false
	}

	msg := cq.Message.Message
	for _, id := range session.RecentMessageIDs {
		if id > int64(msg.ID) {
			return false
		}
	}

	return time.Since(time.Unix(int64(msg.Date), 0)) < maxEditWindow
}

// ClearMessagesAfter deletes every tracked message sent after (and
// including) the callback query's message, mirroring clearMessagesAfter().
func ClearMessagesAfter(ctx context.Context, b *tgbot.Bot, chatID int64, session *store.SessionData, cq *models.CallbackQuery) {
	if cq == nil || cq.Message.Message == nil {
		return
	}

	messageID := int64(cq.Message.Message.ID)
	index := indexOf(session.RecentMessageIDs, messageID)
	if index == -1 {
		return
	}

	toClean := session.RecentMessageIDs[index:]
	session.CommandMessageIDs = without(session.CommandMessageIDs, toClean)

	clearMessages(ctx, b, chatID, toClean)
}

// TrackSentMessage records a message the bot just sent, mirroring
// handleSendMessage().
func TrackSentMessage(session *store.SessionData, messageID int) {
	session.RecentMessageIDs = append(session.RecentMessageIDs, int64(messageID))
}

// TrackIncoming handles bookkeeping for an incoming update (command vs plain
// text) and cleans up superseded messages, mirroring handleNewMessage().
func TrackIncoming(ctx context.Context, b *tgbot.Bot, chatID int64, session *store.SessionData, msg *models.Message, isCommand string) {
	if msg == nil {
		return
	}
	if msg.ViaBot != nil {
		isCommand = ""
	}

	switch {
	case isCommand == "start":
		session.CommandMessageIDs = []int64{int64(msg.ID)}
		clearMessages(ctx, b, chatID, session.RecentMessageIDs)
		session.RecentMessageIDs = []int64{int64(msg.ID)}
	case isCommand != "":
		session.CommandMessageIDs = append(session.CommandMessageIDs, int64(msg.ID))
		session.RecentMessageIDs = append(session.RecentMessageIDs, int64(msg.ID))
	default:
		session.RecentMessageIDs = append(session.RecentMessageIDs, int64(msg.ID))
	}

	index := commandIndex(session.CommandMessageIDs, session.RecentMessageIDs)
	if index < len(session.RecentMessageIDs) {
		clearMessages(ctx, b, chatID, session.RecentMessageIDs[index:])
	}
}

func commandIndex(commandMessageIDs, recentMessageIDs []int64) int {
	if len(commandMessageIDs) == 0 {
		return 0
	}
	last := commandMessageIDs[len(commandMessageIDs)-1]
	return indexOf(recentMessageIDs, last) + 1
}

func clearMessages(ctx context.Context, b *tgbot.Bot, chatID int64, messageIDs []int64) {
	for _, id := range messageIDs {
		_, _ = b.DeleteMessage(ctx, &tgbot.DeleteMessageParams{ChatID: chatID, MessageID: int(id)})
	}
}

func indexOf(ids []int64, id int64) int {
	for i, v := range ids {
		if v == id {
			return i
		}
	}
	return -1
}

func without(ids []int64, remove []int64) []int64 {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		skip := false
		for _, r := range remove {
			if id == r {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, id)
		}
	}
	return out
}
