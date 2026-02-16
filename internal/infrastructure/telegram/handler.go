package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleUpdate(bot *Bot, update tgbotapi.Update) error {
	if !update.Message.IsCommand() {
		return nil
	}

	switch update.Message.Command() {
	case "start":
		return handleStart(bot, update)
	default:
		return handleUnknown(bot, update)
	}
}

func handleStart(bot *Bot, update tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hello! Welcome to the support chat bot.")
	msg.ParseMode = "Markdown"

	_, err := bot.API().Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	return nil
}

func handleUnknown(bot *Bot, update tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command. Use /start to begin.")
	msg.ParseMode = "Markdown"

	_, err := bot.API().Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send unknown command message: %w", err)
	}

	return nil
}