package conversation

import (
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

type transitionKey struct {
	From  state.State
	Event state.Event
}

type transitionResult struct {
	To       state.State
	Response func(ctx TransitionContext) response.Response
}

type TransitionContext struct {
	UserText         string
	SelectedCategory string
}

var transitions = map[transitionKey]transitionResult{
	// New
	{From: state.StateNew, Event: state.EventUnknown}: {
		To: state.StateWaitingForCategory,
		Response: func(_ TransitionContext) response.Response {
			return response.Response{
				Text:    "Здравствуйте! Пожалуйста, выберите категорию вашего вопроса:",
				Options: []string{"Техническая поддержка", "Биллинг", "Общие вопросы"},
				State:   state.StateWaitingForCategory,
			}
		},
	},
}

func (c *Conversation) Transition(event state.Event, ctx TransitionContext) (state.State, response.Response, error) {
	key := transitionKey{From: c.State, Event: event}
	result, ok := transitions[key]
	if !ok {
		return c.State, response.Response{}, fmt.Errorf("%w: state=%s, event=%s", ErrInvalidTransition, c.State, event)
	}
	return result.To, result.Response(ctx), nil
}
