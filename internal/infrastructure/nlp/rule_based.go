package nlp

import (
	"strings"

	"github.com/VladKovDev/chat-bot/internal/domain/conversation"
)

type RuleBased struct{}

func NewRuleBased() *RuleBased {
	return &RuleBased{}
}

var eventKw = map[conversation.Event][]string{
	conversation.EventRequestOperator: {
		"оператор",
		"человек",
		"живой",
		"позовите",
		"не бот",
	},
	conversation.EventNotResolved: {
		"не помогло",
		"не помог",
		"не работает",
		"не решил",
	},
	conversation.EventResolved:{
		"спасибо",
		"решено",
		"помогло",
		"помог",
	},
	conversation.EventCategorySelected: {
		"вход",
		"пароль",
		"платежи",
		"другое",
	},
}


func (r *RuleBased) Classify(textRow string) (conversation.Event, error) {
	text := Normalize(textRow)

	return keywords(text), nil
}

func keywords(text string) conversation.Event {
	for event, kw := range eventKw {
		for _, k := range kw {
			if strings.Contains(text, k) {
				return event
			}
		}
	}
	return conversation.EventMessageReceived
}