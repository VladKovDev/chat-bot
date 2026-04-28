package response

import "github.com/VladKovDev/chat-bot/internal/domain/state"

type Response struct {
	Text    string
	Options []string
	State   state.State
}