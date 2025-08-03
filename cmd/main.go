package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"

	dotenv "github.com/joho/godotenv"
	envconf "github.com/sethvargo/go-envconfig"

	"github.com/luckyComet55/marzban-tg-bot/internal/handler"
	"github.com/luckyComet55/marzban-tg-bot/internal/middleware"
	"github.com/luckyComet55/marzban-tg-bot/internal/repository"
)

type AppConfig struct {
	BotApiKey       string  `env:"BOT_TOKEN, required"`
	AuthorizedUsers []int64 `env:"AUTHORIZED_USER_IDS, required"`
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
	adminRepo := repository.NewAdminRepository()

	handlerWrapper := handler.NewMessageHandler(adminRepo, logger)
	everithingHandler := middleware.WithWhitelist(whitelistMidleware, handlerWrapper.HandleUpdate)

	opts := []bot.Option{
		bot.WithDefaultHandler(everithingHandler),
		bot.WithDebug(),
	}

	b, err := bot.New(c.BotApiKey, opts...)
	if err != nil {
		panic(err)
	}

	b.Start(ctx)
}
