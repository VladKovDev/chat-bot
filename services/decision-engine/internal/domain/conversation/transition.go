package conversation

import (
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

// ResponseLoader interface for loading responses from JSON
type ResponseLoader interface {
	GetResponse(key string) (message string, options []string, ok bool)
}

// TransitionHandler is a function that executes transition logic and returns response key
type TransitionHandler func(ctx HandlerContext) (nextState state.State, responseKey string, err error)

// HandlerContext provides access to services and data during transition
type HandlerContext struct {
	// User input data
	UserText         string
	SelectedCategory string

	// Services and dependencies
	ResponseLoader ResponseLoader
	Logger         interface{} // TODO: define logger interface

	// Custom data for intermediate results
	Data map[string]interface{}
}

// SetData stores custom data in handler context
func (h *HandlerContext) SetData(key string, value interface{}) {
	if h.Data == nil {
		h.Data = make(map[string]interface{})
	}
	h.Data[key] = value
}

// GetData retrieves custom data from handler context
func (h *HandlerContext) GetData(key string) (interface{}, bool) {
	if h.Data == nil {
		return nil, false
	}
	val, ok := h.Data[key]
	return val, ok
}

// GetResponseByKey loads response by key using ResponseLoader
func (h *HandlerContext) GetResponseByKey(key string) (response.Response, error) {
	message, options, ok := h.ResponseLoader.GetResponse(key)
	if !ok {
		return response.Response{}, fmt.Errorf("response key not found: %s", key)
	}
	return response.Response{
		Text:    message,
		Options: options,
	}, nil
}

type transitionKey struct {
	From  state.State
	Event state.Event
}

type transitionResult struct {
	To      state.State
	Handler TransitionHandler
}

// TransitionContext is kept for backward compatibility
// Use HandlerContext in new handlers instead
type TransitionContext struct {
	UserText         string
	SelectedCategory string
}

var transitions = map[transitionKey]transitionResult{
	// New
	{From: state.StateNew, Event: state.EventUnknown}: {
		To: state.StateWaitingForCategory,
		Handler: func(ctx HandlerContext) (state.State, string, error) {
			// Example: Simple handler that returns static key
			// You can add logic here: call services, check conditions, etc.
			return state.StateWaitingForCategory, "start", nil
		},
	},

	// Example: Handler with service call and conditional logic
	// {From: state.StateWaitingForCategory, Event: state.EventCategorySelected}: {
	//     To: state.StateWaitingClarification,
	//     Handler: func(ctx HandlerContext) (state.State, string, error) {
	//         // Call service to get category info
	//         categoryInfo, err := ctx.CategoryService.GetCategory(ctx.SelectedCategory)
	//         if err != nil {
	//             return state.StateWaitingForCategory, "category_error", err
	//         }
	//
	//         // Store data for later use
	//         ctx.SetData("categoryInfo", categoryInfo)
	//
	//         // Return different response based on category
	//         if categoryInfo.NeedsClarification {
	//             return state.StateWaitingClarification, "needs_clarification", nil
	//         }
	//         return state.StateSolutionOffered, "solution_ready", nil
	//     },
	// },

	// Example: Handler with multiple steps
	// {From: state.StateWaitingClarification, Event: state.EventMessageReceived}: {
	//     To: state.StateSolutionOffered,
	//     Handler: func(ctx HandlerContext) (state.State, string, error) {
	//         // Step 1: Analyze user input
	//         analysis := ctx.AnalysisService.Analyze(ctx.UserText)
	//         ctx.SetData("analysis", analysis)
	//
	//         // Step 2: Check if we need more info
	//         if analysis.Confidence < 0.5 {
	//             return state.StateWaitingClarification, "need_more_details", nil
	//         }
	//
	//         // Step 3: Generate solution
	//         solution, err := ctx.SolutionService.Generate(analysis)
	//         if err != nil {
	//             return state.StateWaitingClarification, "generation_error", err
	//         }
	//
	//         ctx.SetData("solution", solution)
	//         return state.StateSolutionOffered, "solution_generated", nil
	//     },
	// },
}

// Transition executes state transition using handler
func (c *Conversation) Transition(event state.Event, ctx HandlerContext) (state.State, string, error) {
	key := transitionKey{From: c.State, Event: event}
	result, ok := transitions[key]
	if !ok {
		return c.State, "", fmt.Errorf("%w: state=%s, event=%s", ErrInvalidTransition, c.State, event)
	}

	// Execute handler with context
	nextState, responseKey, err := result.Handler(ctx)
	if err != nil {
		return c.State, "", fmt.Errorf("transition handler error: %w", err)
	}

	return nextState, responseKey, nil
}

// TransitionWithResponse executes state transition and loads response from JSON
func (c *Conversation) TransitionWithResponse(event state.Event, ctx HandlerContext) (state.State, response.Response, error) {
	nextState, responseKey, err := c.Transition(event, ctx)
	if err != nil {
		return c.State, response.Response{}, err
	}

	message, options, ok := ctx.ResponseLoader.GetResponse(responseKey)
	if !ok {
		return nextState, response.Response{}, fmt.Errorf("response key not found: %s", responseKey)
	}

	return nextState, response.Response{
		Text:    message,
		Options: options,
		State:   nextState,
	}, nil
}