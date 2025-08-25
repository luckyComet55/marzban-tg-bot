package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	repo "github.com/luckyComet55/marzban-tg-bot/internal/repository"
	"github.com/luckyComet55/marzban-tg-bot/pkg/fsm"
)

type MessageHandler struct {
	logger          *slog.Logger
	adminRepository repo.AdminRepository
	userRepository  repo.UserRepository
	proxyRepository repo.ProxyRepository
}

func NewMessageHandler(adminRepo repo.AdminRepository, userRepo repo.UserRepository, proxyRepo repo.ProxyRepository, logger *slog.Logger) *MessageHandler {
	return &MessageHandler{
		logger:          logger,
		adminRepository: adminRepo,
		userRepository:  userRepo,
		proxyRepository: proxyRepo,
	}
}

func (mh *MessageHandler) HandleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	adminID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	exists, err := mh.adminRepository.CheckAdminExists(adminID)
	if err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	if !exists {
		if err := mh.adminRepository.AddAdmin(adminID); err != nil {
			mh.logger.Error(err.Error())
			if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
				Text:   "Unable to serve you, try again later",
				ChatID: chatID,
			}); err != nil {
				mh.logger.Error(err.Error())
			}
			return
		}
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgbot", b); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgmes", update); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgchat", chatID); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgctx", ctx); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	if err := mh.adminRepository.SetAdminState(adminID, repo.ADMIN_STATE_DEFAULT); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}
}

func (mh *MessageHandler) HandleCancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	adminID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	exists, err := mh.adminRepository.CheckAdminExists(adminID)
	if err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you right now, try again later",
			ChatID: chatID,
		})
		return
	}
	if !exists {
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "To start using bot enter /start",
			ChatID: chatID,
		})
		return
	}

	if err := mh.adminRepository.RemoveAdmin(adminID); err != nil {
		mh.logger.Error(fmt.Sprintf("error while deleting admin: %s", err.Error()))
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to delete you, try again later",
			ChatID: chatID,
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		Text:   "Successfully deleted you",
		ChatID: chatID,
	})
}

func (mh *MessageHandler) HandleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	var adminID, chatID int64
	if update.Message != nil {
		adminID = update.Message.From.ID
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		adminID = update.CallbackQuery.From.ID
		chatID = update.CallbackQuery.Message.Message.Chat.ID
	} else {
		return
	}

	exists, err := mh.adminRepository.CheckAdminExists(adminID)
	if err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you right now, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}
	if !exists {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "To start using bot enter /start",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	var adminInput, transitionName string
	transitionName = "next"
	if update.Message != nil {
		adminInput = update.Message.Text
	} else if update.CallbackQuery != nil {
		queryData := strings.Split(update.CallbackQuery.Data, ":")
		adminInput = queryData[1]
		if queryData[0] != "" {
			transitionName = queryData[0]
		}
	} else {
		adminInput = ""
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgbot", b); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgmes", update); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgchat", chatID); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgctx", ctx); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
		return
	}

	mh.logger.Debug("user input is", "input", adminInput)

	if err := mh.adminRepository.TriggerAdminTransition(adminID, fsm.Event(transitionName), adminInput); err != nil {
		mh.logger.Error(err.Error())
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		}); err != nil {
			mh.logger.Error(err.Error())
		}
	}
}
