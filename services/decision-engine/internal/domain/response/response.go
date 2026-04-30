package response

import (
	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
)

// Response represents the decision engine's response to a message
type Response struct {
	Text    string
	Options []string
	State   conversation.State
}