package transition

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// Engine executes state transitions based on events
type Engine struct {
	transitions  map[session.Mode]map[session.Event]*TransitionConfig
	globalEvents map[session.Event]*GlobalEventConfig
	logger       logger.Logger
}

// NewEngine creates a new transition engine
func NewEngine(
	transitions map[session.Mode]map[session.Event]*TransitionConfig,
	globalEvents map[session.Event]*GlobalEventConfig,
	logger logger.Logger,
) *Engine {
	return &Engine{
		transitions:  transitions,
		globalEvents: globalEvents,
		logger:       logger,
	}
}

// Execute executes a transition from the current mode with the given event.
// It returns the next mode and actions to execute.
func (e *Engine) Execute(ctx context.Context, current session.Mode, event session.Event) (*TransitionResult, error) {
	e.logger.Debug("executing transition",
		e.logger.String("from", string(current)),
		e.logger.String("event", string(event)))

	// 1. Check global events first (can be triggered from any state)
	if global, ok := e.globalEvents[event]; ok {
		e.logger.Info("global event triggered",
			e.logger.String("event", string(event)),
			e.logger.String("from_state", string(current)),
			e.logger.String("to_state", string(global.To)))

		return &TransitionResult{
			NextMode:    global.To,
			Actions:     global.Actions,
			ResponseKey: global.ResponseKey,
		}, nil
	}

	// 2. Find matching state transition
	if stateTransitions, ok := e.transitions[current]; ok {
		if trans, ok := stateTransitions[event]; ok {
			e.logger.Info("transition found",
				e.logger.String("from", string(current)),
				e.logger.String("event", string(event)),
				e.logger.String("to", string(trans.To)))

			return &TransitionResult{
				NextMode:    trans.To,
				Actions:     trans.Actions,
				ResponseKey: trans.ResponseKey,
			}, nil
		}
	}

	// 3. No transition found - stay in current state with default response
	e.logger.Debug("no transition found, staying in current state",
		e.logger.String("state", string(current)),
		e.logger.String("event", string(event)))

	return &TransitionResult{
		NextMode:    current,
		Actions:     []string{},
		ResponseKey: "", // Will be handled by presenter as fallback
	}, fmt.Errorf("no transition found for mode %s and event %s", current, event)
}
