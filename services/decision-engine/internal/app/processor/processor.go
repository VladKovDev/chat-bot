package processor

import (
	"context"
	"fmt"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

// Processor executes actions by name
type Processor struct {
	actions map[string]action.Action
	logger  logger.Logger
}

// NewProcessor creates a new action processor
func NewProcessor(logger logger.Logger) *Processor {
	return &Processor{
		actions: make(map[string]action.Action),
		logger:  logger,
	}
}

// Register registers an action with a name
func (p *Processor) Register(name string, act action.Action) {
	p.actions[name] = act
	p.logger.Debug("action registered",
		p.logger.String("name", name),
		p.logger.String("type", fmt.Sprintf("%T", act)))
}

// Execute executes actions by their names
func (p *Processor) Execute(ctx context.Context, actionNames []string, data action.ActionData) error {
	if len(actionNames) == 0 {
		return nil
	}

	p.logger.Debug("executing actions",
		p.logger.Int("count", len(actionNames)),
		p.logger.Any("actions", actionNames))

	for _, name := range actionNames {
		act, ok := p.actions[name]
		if !ok {
			p.logger.Warn("action not found, skipping",
				p.logger.String("action", name))
			continue
		}

		p.logger.Debug("executing action", p.logger.String("name", name))
		if err := act.Execute(ctx, data); err != nil {
			return fmt.Errorf("action %s failed: %w", name, err)
		}
		p.logger.Debug("action executed successfully", p.logger.String("name", name))
	}

	return nil
}