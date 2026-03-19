package telegram_temp

import (
	"context"
	"fmt"
	"time"

	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

func HandleUpdate(bot *Bot, update tgbotapi.Update) error {
	if !update.Message.IsCommand() {
		return handleMessage(bot, update)
	}

	switch update.Message.Command() {
	case "start":
		return handleStart(bot, update)
	default:
		return handleUnknown(bot, update)
	}
}


func handleMessage(bot *Bot, update tgbotapi.Update) error {
	ctx := context.Background()
	
	incomingMsg := contracts.IncomingMessage{
		EventID:   uuid.New(),
		Channel:   conversation.ChannelTelegram,
		ChatID:    update.Message.Chat.ID,
		Text:      update.Message.Text,
		Timestamp: time.Now(),
	}
	
	if err := bot.msgWorker.HandleMessage(ctx, incomingMsg); err != nil {
		return fmt.Errorf("failed to handle message: %w", err)
	}
	return nil
}

func handleStart(bot *Bot, update tgbotapi.Update) error {
	return nil
}

func handleUnknown(bot *Bot, update tgbotapi.Update) error {
	return nil
}
