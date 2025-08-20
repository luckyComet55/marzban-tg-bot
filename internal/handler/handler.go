package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/luckyComet55/marzban-tg-bot/internal/fsm"
	repo "github.com/luckyComet55/marzban-tg-bot/internal/repository"
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
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	if !exists {
		if err := mh.adminRepository.AddAdmin(adminID); err != nil {
			mh.logger.Error(err.Error())
			b.SendMessage(ctx, &bot.SendMessageParams{
				Text:   "Unable to serve you, try again later",
				ChatID: chatID,
			})
			return
		}
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgbot", b); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgmes", update); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgchat", chatID); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgctx", ctx); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	if err := mh.adminRepository.SetAdminState(adminID, repo.ADMIN_STATE_DEFAULT); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}
}

func (mh *MessageHandler) HandleCancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	adminID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	if _, ok := mh.adminRepository.GetAdminState(adminID); !ok {
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

	adminState, ok := mh.adminRepository.GetAdminState(adminID)
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "To start using bot enter /start",
			ChatID: chatID,
		})
		return
	}

	mh.logger.Debug(fmt.Sprintf("handling user %d with state %s", adminID, adminState))

	var adminInput, transitionName string
	transitionName = "next"
	if update.Message != nil {
		adminInput = update.Message.Text
	} else if update.CallbackQuery != nil {
		adminInput = update.CallbackQuery.Message.Message.Text
		if update.CallbackQuery.Data != "" {
			transitionName = update.CallbackQuery.Data
		}
	} else {
		adminInput = ""
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgbot", b); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgmes", update); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgchat", chatID); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	if err := mh.adminRepository.SetAdminMeta(adminID, "tgctx", ctx); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you, try again later",
			ChatID: chatID,
		})
		return
	}

	mh.logger.Info("user input is", "input", adminInput)

	if err := mh.adminRepository.TriggerAdminTransition(adminID, fsm.Event(transitionName), adminInput); err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   err.Error(),
			ChatID: chatID,
		})
	}
}

func (mh *MessageHandler) ListUsers(ctx context.Context, b *bot.Bot, update *models.Update) {
	users, err := mh.userRepository.GetUsers()
	if err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you right now, try again later",
			ChatID: update.Message.Chat.ID,
		})
		return
	}

	messageTemplate := `
- username: %s
  used traffic: %d GiB
  config name: %s\n`
	userListMessage := fmt.Sprintf("Total of %d users:", len(users))

	for _, user := range users {
		userListMessage += fmt.Sprintf(messageTemplate, user.Username, user.UsedTraffic, user.ProxyProtocol)
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		Text:   userListMessage,
		ChatID: update.Message.Chat.ID,
	})
}

func (mh *MessageHandler) ListProxies(ctx context.Context, b *bot.Bot, update *models.Update) {
	proxies, err := mh.proxyRepository.ListProxies()
	if err != nil {
		mh.logger.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			Text:   "Unable to serve you right now, try again later",
			ChatID: update.Message.Chat.ID,
		})
		return
	}

	proxyMessage := fmt.Sprintf("Total of %d proxies:\n", len(proxies))

	for _, p := range proxies {
		proxyMessage += fmt.Sprintf("- %s\n", p.ProxyName)
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		Text:   proxyMessage,
		ChatID: update.Message.Chat.ID,
	})
}
