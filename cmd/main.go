package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	dotenv "github.com/joho/godotenv"
	envconf "github.com/sethvargo/go-envconfig"
)

type AppConfig struct {
	BotApiKey string `env:"BOT_TOKEN, required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := dotenv.Load(); err != nil {
		log.Println("Warning! No .env file found")
	}

	var c AppConfig

	envconf.MustProcess(ctx, &c)

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(c.BotApiKey, opts...)
	if err != nil {
		panic(err)
	}

	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}
