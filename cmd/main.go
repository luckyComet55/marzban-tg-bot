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
	"github.com/go-telegram/ui/keyboard/inline"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	dotenv "github.com/joho/godotenv"
	envconf "github.com/sethvargo/go-envconfig"

	pcl "github.com/luckyComet55/marzban-proto-contract/gen/go/contract"
	"github.com/luckyComet55/marzban-tg-bot/internal/fsm"
	"github.com/luckyComet55/marzban-tg-bot/internal/handler"
	"github.com/luckyComet55/marzban-tg-bot/internal/middleware"
	"github.com/luckyComet55/marzban-tg-bot/internal/repository"
	repo "github.com/luckyComet55/marzban-tg-bot/internal/repository"
)

type AppConfig struct {
	BotApiKey       string  `env:"BOT_TOKEN, required"`
	AuthorizedUsers []int64 `env:"AUTHORIZED_USER_IDS, required"`
	ServerURL       string  `env:"SERVER_URL, required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := dotenv.Load(); err != nil {
		log.Println("Warning! No .env file found")
	}

	var c AppConfig

	envconf.MustProcess(ctx, &c)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	whitelistMidleware := middleware.NewWhitelistMiddleware(c.AuthorizedUsers, logger)

	conn, err := grpc.NewClient(c.ServerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	grpcClient := pcl.NewMarzbanManagementPanelClient(conn)

	userRepo := repo.NewUserRepository(grpcClient, logger)
	proxyRepo := repo.NewProxyRepository(logger)
	adminStateMashine := fsm.NewFSM(repository.ADMIN_STATE_DEFAULT).
		Transition(repo.ADMIN_STATE_DEFAULT, "lu", repo.ADMIN_STATE_DEFAULT).
		Transition(repo.ADMIN_STATE_DEFAULT, "lp", repo.ADMIN_STATE_DEFAULT).
		Transition(repo.ADMIN_STATE_DEFAULT, "cu", repo.ADMIN_STATE_CREATE_USER_INPUT_NAME).
		Transition(repo.ADMIN_STATE_CREATE_USER_INPUT_NAME, "next", repo.ADMIN_STATE_CREATE_USER_SELECT_PROXY).
		Transition(repo.ADMIN_STATE_CREATE_USER_SELECT_PROXY, "up", repo.ADMIN_STATE_CREATE_USER_SUBMIT_DATA).
		Transition(repo.ADMIN_STATE_CREATE_USER_SUBMIT_DATA, "s", repo.ADMIN_STATE_DEFAULT)

	adminRepo := repo.NewAdminRepository(adminStateMashine)

	handlerWrapper := handler.NewMessageHandler(adminRepo, userRepo, proxyRepo, logger)
	everithingHandler := middleware.WithWhitelist(whitelistMidleware, handlerWrapper.HandleUpdate)
	startHandler := middleware.WithWhitelist(whitelistMidleware, handlerWrapper.HandleStart)
	cancelHandler := middleware.WithWhitelist(whitelistMidleware, handlerWrapper.HandleCancel)

	adminStateMashine.OnTransition(func(from, to fsm.State, event fsm.Event, ctx *fsm.FSMContext) error {
		b := ctx.Meta["tgbot"].(*bot.Bot)
		mes := ctx.Meta["tgmes"].(*models.Update)
		c := ctx.Meta["tgctx"].(context.Context)

		if from == repo.ADMIN_STATE_DEFAULT && event == "lu" {
			handlerWrapper.ListUsers(c, b, mes)
			return nil
		}

		if from == repo.ADMIN_STATE_DEFAULT && event == "lp" {
			handlerWrapper.ListProxies(c, b, mes)
			return nil
		}

		return nil
	})

	inlineCallbackFactory := func(fsmCtx *fsm.FSMContext, transitionEvent fsm.Event) inline.OnSelect {
		return func(ctx context.Context, b *bot.Bot, mes models.MaybeInaccessibleMessage, data []byte) {
			logger.Info(fmt.Sprintf("triggering event %s", transitionEvent))
			logger.Info("transition data", "data", string(data))

			fsmCtx.Meta["tgbot"] = b
			fsmCtx.Meta["tgchat"] = mes.Message.Chat.ID
			fsmCtx.Meta["tgctx"] = ctx

			adminID := mes.Message.Chat.ID

			exists, err := adminRepo.CheckAdminExists(adminID)
			if err != nil {
				logger.Error(err.Error())
				return
			}
			if !exists {
				logger.Error(fmt.Sprintf("no admin with ID %d", adminID))
				return
			}

			if err := adminRepo.TriggerAdminTransition(adminID, transitionEvent, string(data)); err != nil {
				logger.Error(err.Error())
				return
			}
		}
	}

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
		logger.Info(fmt.Sprintf("%v", ctx.Input))
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

		inlineCallback := inlineCallbackFactory(ctx, "up")

		proxies, err := proxyRepo.ListProxies()
		if err != nil {
			logger.Error(err.Error())
			return err
		}

		km := inline.New(b).Row()

		for _, p := range proxies {
			km.Button(p.ProxyName, []byte(p.ProxyName), inlineCallback)
		}

		b.SendMessage(c, &bot.SendMessageParams{
			ChatID:      u,
			Text:        "Select user proxy configuration from list",
			ReplyMarkup: km,
		})

		return nil
	})

	adminStateMashine.OnExit(repo.ADMIN_STATE_CREATE_USER_SELECT_PROXY, func(ctx *fsm.FSMContext) error {
		logger.Info("setting proxy name", "proxy", ctx.Input.(string))
		ctx.Data["proxy"] = ctx.Input.(string)
		return nil
	})

	adminStateMashine.OnEnter(repo.ADMIN_STATE_CREATE_USER_SUBMIT_DATA, func(ctx *fsm.FSMContext) error {
		b := ctx.Meta["tgbot"].(*bot.Bot)
		u := ctx.Meta["tgchat"].(int64)
		c := ctx.Meta["tgctx"].(context.Context)

		username := ctx.Data["username"].(string)
		proxy := ctx.Data["proxy"].(string)

		km := inline.New(b).
			Row().
			Button("Submit", []byte{}, inlineCallbackFactory(ctx, "s")).
			Button("Cancel", []byte{}, inlineCallbackFactory(ctx, "cnl"))

		b.SendMessage(c, &bot.SendMessageParams{
			Text:        fmt.Sprintf("Username: %s\nProxy config: %s", username, proxy),
			ChatID:      u,
			ReplyMarkup: km,
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

		km := inline.New(b).
			Row().
			Button("List users", []byte{}, inlineCallbackFactory(ctx, "lu")).
			Button("List proxies", []byte{}, inlineCallbackFactory(ctx, "lp")).
			Row().
			Button("Create user", []byte{}, inlineCallbackFactory(ctx, "cu"))

		b.SendMessage(c, &bot.SendMessageParams{
			ChatID:      u,
			Text:        "Select action",
			ReplyMarkup: km,
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
