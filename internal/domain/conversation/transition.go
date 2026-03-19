package conversation

import "fmt"

type transitionKey struct {
	From  State
	Event Event
}

type transitionResult struct {
	To       State
	Response func(ctx TransitionContext) BotResponse
}

type TransitionContext struct {
	UserText         string
	SelectedCategory string
}

var transitions = map[transitionKey]transitionResult{
	// New
	{From: StateNew, Event: EventUnknown}: {
		To: StateWaitingForCategory,
		Response: func(_ TransitionContext) BotResponse {
			return BotResponse{
				Text:    "Здравствуйте! Пожалуйста, выберите категорию вашего вопроса:",
				Buttons: categoryButtons(),
			}
		},
	},
	// WaitingForCategory
	{From: StateWaitingForCategory, Event: EventCategorySelected}: {
		To: StateWaitingClarification,
		Response: func(ctx TransitionContext) BotResponse {
			return BotResponse{
				Text: "Спасибо! Можете уточнить ваш вопрос?",
			}
		},
	},
	// WaitingClarification
	{From: StateWaitingClarification, Event: EventUnknown}: {
		To: StateSolutionOffered,
		Response: func(ctx TransitionContext) BotResponse {
			return BotResponse{
				Text:    "Спасибо за уточнение! Я предлагаю следующее решение...",
				Buttons: resolvedButtons(),
			}
		},
	},
	// SolutionOffered
	{From: StateSolutionOffered, Event: EventNotResolved}: {
		To: StateEscalatedToOperator,
		Response: func(_ TransitionContext) BotResponse {
			return BotResponse{
				Text:               "Извините, что не смог помочь. Я передаю ваш вопрос оператору, который свяжется с вами в ближайшее время.",
				ShouldCreateTicket: true,
				TicketCategory:     "Уточнение решения",
			}
		},
	},
	{From: StateSolutionOffered, Event: EventResolved}: {
		To: StateClosed,
		Response: func(_ TransitionContext) BotResponse {
			return BotResponse{
				Text: "Рад был помочь! Если у вас возникнут еще вопросы, не стесняйтесь обращаться.",
			}
		},
	},
	// Closed
	{From: StateClosed, Event: EventMessageReceived}: {
		To: StateWaitingForCategory,
		Response: func(_ TransitionContext) BotResponse {
			return BotResponse{
				Text:    "Здравствуйте! Пожалуйста, выберите категорию вашего вопроса:",
				Buttons: categoryButtons(),
			}
		},
	},
}

func (c *Conversation) Transition(event Event, ctx TransitionContext) (State, BotResponse, error) {
	key := transitionKey{From: c.State, Event: event}
	result, ok := transitions[key]
	if !ok {
		return c.State, BotResponse{}, fmt.Errorf("%w: state=%s, event=%s", ErrInvalidTransition, c.State, event)
	}
	return result.To, result.Response(ctx), nil
}
