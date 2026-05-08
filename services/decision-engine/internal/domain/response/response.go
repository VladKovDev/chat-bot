package response

import (
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

// Response represents the decision engine's response to a message
type Response struct {
	Text    string
	Options []string
	State   state.State
}