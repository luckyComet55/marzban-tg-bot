package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	repo "github.com/luckyComet55/marzban-tg-bot/internal/repository"
)

type MessageHandler struct {
	logger          *slog.Logger
	adminRepository repo.AdminRepository
}

func NewMessageHandler(adminRepo repo.AdminRepository, logger *slog.Logger) *MessageHandler {
	return &MessageHandler{
		logger:          logger,
		adminRepository: adminRepo,
	}
}

func (mh *MessageHandler) HandleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	var adminID int64
	if update.Message != nil {
		adminID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		adminID = update.CallbackQuery.From.ID
	} else {
		return
	}

	adminState, ok := mh.adminRepository.GetAdminState(adminID)
	if !ok {
		mh.adminRepository.SetAdminState(adminID, repo.ADMIN_STATE_DEFAULT)
		adminState = repo.ADMIN_STATE_DEFAULT
	}

	mh.logger.Debug(fmt.Sprintf("handling user %d with state %s", adminID, adminState))

	switch adminState {
	case repo.ADMIN_STATE_DEFAULT:
		mh.handleDefaultState(ctx, b, update)
	case repo.ADMIN_STATE_CREATE_USER_INPUT_NAME:
		mh.handleInputUsername(ctx, b, update)
	case repo.ADMIN_STATE_CREATE_USER_SELECT_PROXY:
		mh.handleSelectProxy(ctx, b, update)
	case repo.ADMIN_STATE_CREATE_USER_SUBMIT_DATA:
		mh.handleSubmitData(ctx, b, update)
	}
}

func (mh *MessageHandler) handleDefaultState(ctx context.Context, b *bot.Bot, update *models.Update) {

}

func (mh *MessageHandler) handleInputUsername(ctx context.Context, b *bot.Bot, update *models.Update) {

}

func (mh *MessageHandler) handleSelectProxy(ctx context.Context, b *bot.Bot, update *models.Update) {

}

func (mh *MessageHandler) handleSubmitData(ctx context.Context, b *bot.Bot, update *models.Update) {

}
