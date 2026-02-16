package worker

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/contracts"
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
)

type MessageWorker struct {
	convService *conversation.Service
}

func NewMessageWorker(convService *conversation.Service) *MessageWorker {
	return &MessageWorker{
		convService: convService,
	}
}

func (w *MessageWorker) HandleMessage(ctx context.Context, msg contracts.IncomingMessage) error {
	conv, err := w.convService.LoadConversation(ctx, msg.Channel, msg.ChatID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}

	switch conv.State {
	case conversation.StateNew:
		return w.handleNewConversation(ctx, msg, conv)
	case conversation.StateInProgress:
		return w.handleInProgressConversation(ctx, msg, conv)
	case conversation.StateClosed:
		return w.handleClosedConversation(ctx, msg)
	default:
		return fmt.Errorf("unknown conversation state: %s", conv.State)
	}
}

func (w *MessageWorker) handleNewConversation(ctx context.Context, msg contracts.IncomingMessage, conv *conversation.Conversation) error {
	// TODO: Implement new conversation logic
	// For now, just return nil to indicate success
	fmt.Printf("New conversation: ID=%s, Channel=%s, ChatID=%d, Text=%s\n",
		conv.ID, conv.Channel, conv.ChatID, msg.Text)
	return nil
}

func (w *MessageWorker) handleInProgressConversation(ctx context.Context, msg contracts.IncomingMessage, conv *conversation.Conversation) error {
	// TODO: Implement in-progress conversation logic
	fmt.Printf("In-progress conversation: ID=%s, Channel=%s, ChatID=%d, Text=%s\n",
		conv.ID, conv.Channel, conv.ChatID, msg.Text)
	return nil
}

func (w *MessageWorker) handleClosedConversation(ctx context.Context, msg contracts.IncomingMessage) error {
	// TODO: Implement closed conversation logic
	// May create a new conversation or show a message
	fmt.Printf("Message to closed conversation: Channel=%s, ChatID=%d, Text=%s\n",
		msg.Channel, msg.ChatID, msg.Text)
	return nil
}
