package bot

import (
	"context"
	"log"
	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/config"
	"reseller-bot/pkg/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	cfg     *config.Config
	api     *tgbotapi.BotAPI
	client  *backend.Client
	db      *db.DB
	session *SessionManager
}

func NewBot(cfg *config.Config, database *db.DB, client *backend.Client) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	return &Bot{
		cfg:     cfg,
		api:     api,
		client:  client,
		db:      database,
		session: NewSessionManager(),
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	log.Printf("Authorized on account %s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping bot...")
			return
		case update := <-updates:
			go b.handleUpdate(update)
		}
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in update handler: %v", r)
		}
	}()

	if update.Message != nil {
		b.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
	}
}
