package telegram_temp

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	"github.com/VladKovDev/chat-bot/internal/worker"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	msgWorker   *worker.MessageWorker
	convService *conversation.Service
}

func NewBot(token string, msgWorker *worker.MessageWorker, convService *conversation.Service) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	return &Bot{
		api:         api,
		msgWorker:   msgWorker,
		convService: convService,
	}, nil
}

func (b *Bot) API() *tgbotapi.BotAPI {
	return b.api
}

func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message != nil {
				if err := HandleUpdate(b, update); err != nil {
					fmt.Printf("error handling update: %v\n", err)
				}
			}
		}
	}
}

func (b *Bot) Shutdown(ctx context.Context) error {
	b.api.StopReceivingUpdates()
	return nil
}