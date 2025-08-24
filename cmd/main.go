package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"regexp"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	dotenv "github.com/joho/godotenv"
	envconf "github.com/sethvargo/go-envconfig"

	pcl "github.com/luckyComet55/marzban-proto-contract/gen/go/contract"

	"github.com/luckyComet55/marzban-tg-bot/internal/handler"
	"github.com/luckyComet55/marzban-tg-bot/internal/middleware"
	"github.com/luckyComet55/marzban-tg-bot/internal/repository"
	repo "github.com/luckyComet55/marzban-tg-bot/internal/repository"
	"github.com/luckyComet55/marzban-tg-bot/pkg/fsm"
)

type AppConfig struct {
	BotApiKey       string  `env:"BOT_TOKEN, required"`
	AuthorizedUsers []int64 `env:"AUTHORIZED_USER_IDS, required"`
	ServerURL       string  `env:"SERVER_URL, required"`
	Env             string  `env:"ENV, required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := dotenv.Load(); err != nil {
		log.Println("Warning! No .env file found")
	}

	var c AppConfig

	envconf.MustProcess(ctx, &c)

	logger := configureLogger(c)
	conn, err := grpc.NewClient(c.ServerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	grpcClient := pcl.NewMarzbanManagementPanelClient(conn)

	adminStateMashine := fsm.NewFSM(repository.ADMIN_STATE_DEFAULT).
		Transition(repo.ADMIN_STATE_DEFAULT, "lu", repo.ADMIN_STATE_DEFAULT).
		Transition(repo.ADMIN_STATE_DEFAULT, "lp", repo.ADMIN_STATE_DEFAULT).
		Transition(repo.ADMIN_STATE_DEFAULT, "cu", repo.ADMIN_STATE_CREATE_USER_INPUT_NAME).
		Transition(repo.ADMIN_STATE_CREATE_USER_INPUT_NAME, "next", repo.ADMIN_STATE_CREATE_USER_SELECT_PROXY).
		Transition(repo.ADMIN_STATE_CREATE_USER_SELECT_PROXY, "up", repo.ADMIN_STATE_CREATE_USER_SUBMIT_DATA).
		Transition(repo.ADMIN_STATE_CREATE_USER_SUBMIT_DATA, "s", repo.ADMIN_STATE_DEFAULT).
		Transition(repo.ADMIN_STATE_CREATE_USER_SUBMIT_DATA, "cnl", repo.ADMIN_STATE_DEFAULT)

	userRepo := repo.NewUserRepository(grpcClient, logger.With("component", "userRepo"))
	proxyRepo := repo.NewProxyRepository(logger.With("component", "proxyRepo"))
	adminRepo := repo.NewAdminRepository(adminStateMashine)

	handlerWrapper := handler.NewMessageHandler(adminRepo, userRepo, proxyRepo, logger.With("component", "handlerWrapper"))
	whitelistMidleware := middleware.NewWhitelistMiddleware(c.AuthorizedUsers, logger.With("component", "whitelistMidleware"))
	everithingHandler := middleware.WithWhitelist(whitelistMidleware, handlerWrapper.HandleUpdate)
	startHandler := middleware.WithWhitelist(whitelistMidleware, handlerWrapper.HandleStart)
	cancelHandler := middleware.WithWhitelist(whitelistMidleware, handlerWrapper.HandleCancel)

	adminStateMashine.OnTransition(func(from, to fsm.State, event fsm.Event, ctx *fsm.FSMContext) error {
		logger.Debug("calling on transition")
		b := ctx.Meta["tgbot"].(*bot.Bot)
		u := ctx.Meta["tgchat"].(int64)
		c := ctx.Meta["tgctx"].(context.Context)

		if from == repo.ADMIN_STATE_DEFAULT && event == "lu" {
			users, err := userRepo.GetUsers()
			if err != nil {
				logger.Error(err.Error())
				if _, err := b.SendMessage(c, &bot.SendMessageParams{
					Text:   "Unable to serve you right now, try again later",
					ChatID: u,
				}); err != nil {
					logger.Error(err.Error())
					return err
				}
				return err
			}

			messageTemplate := "\n- username: %s\n  used traffic: %d GiB\n  config url: `%s`"
			userListMessage := fmt.Sprintf("Total of %d users:", len(users))

			for _, user := range users {
				userListMessage += fmt.Sprintf(messageTemplate, user.Username, user.UsedTraffic, user.ConfigUrl)
			}

			if _, err := b.SendMessage(c, &bot.SendMessageParams{
				Text:   userListMessage,
				ChatID: u,
				// ParseMode: models.ParseModeMarkdownV1,
			}); err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		}

		if from == repo.ADMIN_STATE_DEFAULT && event == "lp" {
			proxies, err := proxyRepo.ListProxies()
			if err != nil {
				logger.Error(err.Error())
				if _, err := b.SendMessage(c, &bot.SendMessageParams{
					Text:   "Unable to serve you right now, try again later",
					ChatID: u,
				}); err != nil {
					logger.Error(err.Error())
					return err
				}
				return err
			}

			proxyMessage := fmt.Sprintf("Total of %d proxies:\n", len(proxies))

			for _, p := range proxies {
				proxyMessage += fmt.Sprintf("- %s\n", p.ProxyName)
			}

			if _, err := b.SendMessage(c, &bot.SendMessageParams{
				Text:   proxyMessage,
				ChatID: u,
			}); err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		}

		return nil
	})

	adminStateMashine.OnEnter(repo.ADMIN_STATE_CREATE_USER_INPUT_NAME, func(ctx *fsm.FSMContext) error {
		b := ctx.Meta["tgbot"].(*bot.Bot)
		u := ctx.Meta["tgchat"].(int64)
		c := ctx.Meta["tgctx"].(context.Context)

		b.SendMessage(c, &bot.SendMessageParams{
			Text:   "Input username. It must be 3-32 symbols [a-zA-Z0-9_]",
			ChatID: u,
		})

		return nil
	})

	adminStateMashine.OnExit(repo.ADMIN_STATE_CREATE_USER_INPUT_NAME, func(ctx *fsm.FSMContext) error {
		userName := ctx.Input.(string)

		matchString := "^[a-zA-Z0-9_]{3,32}$"

		isValid, err := regexp.MatchString(matchString, userName)
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		if !isValid {
			return fmt.Errorf("username '%s' does not match pattern %s", userName, matchString)
		}

		ctx.Data["username"] = userName
		return nil
	})

	adminStateMashine.OnEnter(repo.ADMIN_STATE_CREATE_USER_SELECT_PROXY, func(ctx *fsm.FSMContext) error {
		b := ctx.Meta["tgbot"].(*bot.Bot)
		u := ctx.Meta["tgchat"].(int64)
		c := ctx.Meta["tgctx"].(context.Context)

		proxies, err := proxyRepo.ListProxies()
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{make([]models.InlineKeyboardButton, 0)},
		}

		for _, p := range proxies {
			kb.InlineKeyboard[0] = append(kb.InlineKeyboard[0], models.InlineKeyboardButton{Text: p.ProxyName, CallbackData: "up"})
		}

		b.SendMessage(c, &bot.SendMessageParams{
			ChatID:      u,
			Text:        "Select user proxy configuration from list",
			ReplyMarkup: kb,
		})

		return nil
	})

	adminStateMashine.OnExit(repo.ADMIN_STATE_CREATE_USER_SELECT_PROXY, func(ctx *fsm.FSMContext) error {
		ctx.Data["proxy"] = ctx.Input.(string)
		return nil
	})

	adminStateMashine.OnEnter(repo.ADMIN_STATE_CREATE_USER_SUBMIT_DATA, func(ctx *fsm.FSMContext) error {
		b := ctx.Meta["tgbot"].(*bot.Bot)
		u := ctx.Meta["tgchat"].(int64)
		c := ctx.Meta["tgctx"].(context.Context)

		username := ctx.Data["username"].(string)
		proxy := ctx.Data["proxy"].(string)

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Submit", CallbackData: "s"},
				},
				{
					{Text: "Cancel", CallbackData: "cnl"},
				},
			},
		}

		b.SendMessage(c, &bot.SendMessageParams{
			Text:        fmt.Sprintf("Username: %s\nProxy config: %s", username, proxy),
			ChatID:      u,
			ReplyMarkup: kb,
		})

		return nil
	})

	adminStateMashine.OnTransition(func(from, to fsm.State, event fsm.Event, ctx *fsm.FSMContext) error {
		if event != "s" {
			return nil
		}

		b := ctx.Meta["tgbot"].(*bot.Bot)
		u := ctx.Meta["tgchat"].(int64)
		c := ctx.Meta["tgctx"].(context.Context)

		username := ctx.Data["username"].(string)
		proxy := ctx.Data["proxy"].(string)

		userCreateData := repo.UserCreateData{
			Username:      username,
			ProxyProtocol: proxy,
		}

		userData, err := userRepo.CreateUser(userCreateData)
		if err != nil {
			b.SendMessage(c, &bot.SendMessageParams{
				ChatID: u,
				Text:   "Could not add user, try again later",
			})
			return err
		}

		userFormat := "Created user:\nusername: %s\nproxy config: %s\nconfig url: `%s`"
		b.SendMessage(c, &bot.SendMessageParams{
			ChatID:    u,
			Text:      fmt.Sprintf(userFormat, userData.Username, userData.ProxyProtocol, userData.ConfigUrl),
			ParseMode: models.ParseModeMarkdownV1,
		})
		return nil
	})

	adminStateMashine.OnEnter(repo.ADMIN_STATE_DEFAULT, func(ctx *fsm.FSMContext) error {
		b := ctx.Meta["tgbot"].(*bot.Bot)
		u := ctx.Meta["tgchat"].(int64)
		c := ctx.Meta["tgctx"].(context.Context)

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "List users", CallbackData: "lu"},
					{Text: "List proxies", CallbackData: "lp"},
				},
				{
					{Text: "Create user", CallbackData: "cu"},
				},
			},
		}

		b.SendMessage(c, &bot.SendMessageParams{
			ChatID:      u,
			Text:        "Select action",
			ReplyMarkup: kb,
		})
		return nil
	})

	opts := []bot.Option{
		bot.WithDefaultHandler(everithingHandler),
		bot.WithDebug(),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, startHandler),
		bot.WithMessageTextHandler("/cancel", bot.MatchTypeExact, cancelHandler),
	}

	b, err := bot.New(c.BotApiKey, opts...)
	if err != nil {
		panic(err)
	}

	b.Start(ctx)
}

func configureLogger(c AppConfig) *slog.Logger {
	var logger *slog.Logger
	switch c.Env {
	case "dev":
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case "prod":
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		panic(fmt.Sprintf("incorrect env type: %s. possible values: dev, prod", c.Env))
	}
	return logger
}
