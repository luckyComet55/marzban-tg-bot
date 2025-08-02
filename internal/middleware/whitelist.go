package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type WhitelistMiddleware struct {
	logger         *slog.Logger
	userAllowedIDs []int64
}

func NewWhitelistMiddleware(userAllowedIDs []int64, logger *slog.Logger) *WhitelistMiddleware {
	return &WhitelistMiddleware{
		userAllowedIDs: userAllowedIDs,
		logger:         logger,
	}
}

func (wm *WhitelistMiddleware) IsUserAllowed(userID int64) bool {
	return slices.Contains(wm.userAllowedIDs, userID)
}

func WithWhitelist(whitelist *WhitelistMiddleware, handler bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var userID, chatID int64
		var username string
		if update.Message != nil {
			userID = update.Message.From.ID
			chatID = update.Message.Chat.ID
			username = update.Message.From.Username
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
			chatID = update.CallbackQuery.From.ID
			username = update.CallbackQuery.From.Username
		} else {
			handler(ctx, b, update)
			return
		}
		if !whitelist.IsUserAllowed(userID) {
			whitelist.logger.Warn(fmt.Sprintf("user %s (ID %d) is not in the whitelist", username, userID))

			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				Text:   "You are not allowed to use this",
				ChatID: chatID,
			})
			if err != nil {
				whitelist.logger.Error(err.Error())
			}

			return
		}

		handler(ctx, b, update)
	}
}
