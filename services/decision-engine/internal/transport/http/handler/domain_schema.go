package handler

import (
	"net/http"

	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/intent"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

func (h *Handler) DomainSchema(w http.ResponseWriter, _ *http.Request) {
	h.respondJSON(w, http.StatusOK, DomainSchemaResponse{
		Intents: intent.All(),
		States:  state.All(),
		Actions: action.All(),
		Channels: []string{
			session.ChannelWebsite,
			session.ChannelDevCLI,
		},
		Modes: []string{
			string(session.ModeStandard),
			string(session.ModeWaitingOperator),
			string(session.ModeOperatorConnected),
			string(session.ModeClosed),
		},
		OperatorStatuses: []string{
			string(session.OperatorStatusNone),
			string(session.OperatorStatusWaiting),
			string(session.OperatorStatusConnected),
			string(session.OperatorStatusClosed),
		},
		QuickReplyActions: []string{
			quickReplyActionSend,
			"request_operator",
			"select_intent",
		},
		WebSocketEvents: WebSocketEvent{
			Client: []string{
				"session.start",
				"message.user",
				"quick_reply.selected",
				"operator.close",
			},
			Server: []string{
				"session.started",
				"message.bot",
				"message.operator",
				"handoff.queued",
				"handoff.accepted",
				"handoff.closed",
				"error",
			},
		},
	})
}
