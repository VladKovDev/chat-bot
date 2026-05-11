package presenter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

const (
	quickReplyActionSend         = "send_text"
	quickReplyActionSelectIntent = "select_intent"
	quickReplyActionOperator     = "request_operator"
)

type QuickReplyConfig struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Action  string         `json:"action"`
	Payload map[string]any `json:"payload,omitempty"`
	Order   int            `json:"order,omitempty"`
}

// ResponseConfig represents a response template from JSON
type ResponseConfig struct {
	Message      string             `json:"message"`
	Options      []string           `json:"options,omitempty"`
	QuickReplies []QuickReplyConfig `json:"quick_replies,omitempty"`
}

// Loader loads response templates from JSON file
type Presenter struct {
	configPath string
	responses  map[string]*ResponseConfig
	logger     logger.Logger
}

// NewPresenter creates a new presenter
func NewPresenter(configPath string, logs ...logger.Logger) (*Presenter, error) {
	log := logger.Noop()
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	}

	p := &Presenter{
		configPath: configPath,
		responses:  make(map[string]*ResponseConfig),
		logger:     log,
	}

	if err := p.load(); err != nil {
		return nil, err
	}

	return p, nil

}

// Present creates a response from a template key and state
func (p *Presenter) Present(responseKey string, st state.State) (response.Response, error) {
	return p.Render(RenderInput{
		ResponseKey: responseKey,
		State:       st,
	})
}

// load loads all response templates from JSON file
func (p *Presenter) load() error {
	data, err := os.ReadFile(p.configPath + "/responses.json")
	if err != nil {
		return fmt.Errorf("failed to read responses file: %w", err)
	}

	var responses map[string]*ResponseConfig
	if err := json.Unmarshal(data, &responses); err != nil {
		return fmt.Errorf("failed to parse responses: %w", err)
	}

	p.responses = responses
	return nil
}

// Load returns a response config by key
func (p *Presenter) GetResponse(key string) (*ResponseConfig, error) {
	if response, ok := p.responses[key]; ok {
		return response, nil
	}

	return nil, ErrKeyNotFound
}

// GetAll returns all response configs
func (p *Presenter) GetAll() map[string]*ResponseConfig {
	return p.responses
}

// GetAllKeys returns all loaded response keys
func (p *Presenter) GetAllKeys() []string {
	keys := make([]string, 0, len(p.responses))

	for key := range p.responses {
		keys = append(keys, key)
	}

	return keys
}

func (c *ResponseConfig) legacyOptions() []string {
	if len(c.Options) > 0 {
		return append([]string(nil), c.Options...)
	}

	if len(c.QuickReplies) == 0 {
		return nil
	}

	options := make([]string, 0, len(c.QuickReplies))
	for _, quickReply := range c.QuickReplies {
		if quickReply.Label == "" {
			continue
		}
		options = append(options, quickReply.Label)
	}

	return options
}

func buildResponseQuickReplies(responseKey string, c *ResponseConfig) []response.QuickReply {
	if len(c.QuickReplies) > 0 {
		replies := make([]response.QuickReply, 0, len(c.QuickReplies))
		for _, quickReply := range c.QuickReplies {
			replies = append(replies, response.QuickReply{
				ID:      quickReply.ID,
				Label:   quickReply.Label,
				Action:  quickReply.Action,
				Payload: clonePayload(quickReply.Payload),
				Order:   quickReply.Order,
			})
		}
		return replies
	}

	if len(c.Options) == 0 {
		return nil
	}

	replies := make([]response.QuickReply, 0, len(c.Options))
	for index, option := range c.Options {
		replies = append(replies, legacyOptionQuickReply(responseKey, option, index))
	}

	return replies
}

func legacyOptionQuickReply(responseKey, option string, order int) response.QuickReply {
	label := strings.TrimSpace(option)
	payloadText := sanitizeLegacyOptionText(label)
	if payloadText == "" {
		payloadText = label
	}

	reply := response.QuickReply{
		ID:    slugifyQuickReplyID(payloadText),
		Label: label,
		Order: order,
	}

	intentKey := canonicalLegacyOptionKey(responseKey, payloadText)
	switch intentKey {
	case "request_operator":
		reply.Action = quickReplyActionOperator
	case "return_to_menu":
		reply.Action = quickReplyActionSelectIntent
		reply.Payload = map[string]any{
			"intent": "return_to_menu",
			"text":   "главное меню",
		}
		case "ask_booking_info",
		"ask_workspace_info",
		"ask_workspace_prices",
		"ask_workspace_rules",
		"ask_workspace_status",
		"ask_payment_status",
		"payment_not_passed",
		"payment_not_activated",
		"ask_site_problem",
		"login_not_working",
		"code_not_received",
		"ask_account_help",
		"ask_account_status",
		"account_code_not_received",
		"forgot_password",
		"ask_services_info",
		"ask_prices",
		"ask_rules",
		"ask_location",
		"ask_faq",
		"ask_faq_booking",
		"ask_faq_cancellation",
		"ask_faq_workspace",
		"show_contacts",
		"report_complaint",
		"ask_cancellation_rules",
		"ask_reschedule_rules",
		"ask_booking_status",
		"general_question":
		reply.Action = quickReplyActionSelectIntent
		reply.Payload = map[string]any{
			"intent": intentKey,
			"text":   payloadText,
		}
	default:
		reply.Action = quickReplyActionSend
		reply.Payload = map[string]any{
			"text": payloadText,
		}
	}

	return reply
}

func sanitizeLegacyOptionText(label string) string {
	trimmed := strings.TrimSpace(label)
	trimmed = strings.TrimLeftFunc(trimmed, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.TrimSpace(trimmed)
}

func canonicalLegacyOptionKey(responseKey, text string) string {
	switch normalizeLegacyOptionKey(text) {
	case "связаться с оператором",
		"связь с оператором",
		"связаться для записи",
		"связаться с администратором",
		"связаться для бронирования",
		"перейти к оператору",
		"связаться прямо сейчас",
		"передать оператору",
		"вызвать администратора":
		return "request_operator"
	case "вернуться в главное меню",
		"вернуться в меню",
		"вернуться в categories",
		"выбрать категорию":
		return "return_to_menu"
	case "записи и бронирование":
		return "ask_booking_info"
	case "рабочие места":
		return "ask_workspace_info"
	case "оплата":
		return "ask_payment_status"
	case "проблемы с сайтом или входом":
		return "ask_site_problem"
	case "аккаунт":
		return "ask_account_help"
	case "услуги и правила":
		return "ask_services_info"
	case "услуги и цены":
		return "ask_prices"
	case "правила":
		return "ask_rules"
	case "адрес и часы работы":
		return "ask_location"
	case "faq":
		return "ask_faq"
	case "жалобы и проблемы":
		return "report_complaint"
	case "другое":
		return "general_question"
	default:
		return canonicalLegacyOptionKeyByResponse(responseKey, text)
	}
}

func canonicalLegacyOptionKeyByResponse(responseKey, text string) string {
	key := normalizeLegacyOptionKey(text)

	switch responseKey {
	case "booking_found":
		switch key {
		case "правила отмены":
			return "ask_cancellation_rules"
		case "контакты":
			return "show_contacts"
		}
	case "booking_not_found", "booking_request_identifier":
		switch key {
		case "ввести данные снова", "ввести номер записи", "проверить статус записи":
			return "ask_booking_status"
		}
	case "booking_cancellation_rules":
		if key == "правила переноса" {
			return "ask_reschedule_rules"
		}
	case "booking_reschedule_rules":
		if key == "правила отмены" {
			return "ask_cancellation_rules"
		}
	case "payment_category", "payment_refund_rules":
		switch key {
		case "статус платежа", "проверить статус платежа":
			return "ask_payment_status"
		case "оплата не прошла":
			return "payment_not_passed"
		case "деньги списались услуга не активирована":
			return "payment_not_activated"
		}
	case "payment_not_found", "payment_request_id":
		switch key {
		case "ввести данные снова", "ввести id платежа":
			return "ask_payment_status"
		}
	case "payment_debited_not_activated":
		if key == "подождать и проверить снова" {
			return "ask_payment_status"
		}
	case "tech_issue_category":
		switch key {
		case "сайт не работает":
			return "ask_site_problem"
		case "не могу войти":
			return "login_not_working"
		case "не приходит код":
			return "code_not_received"
		}
	case "tech_login_problem":
		switch key {
		case "забыл пароль":
			return "forgot_password"
		case "ошибка неверный логин пароль":
			return "login_not_working"
		}
	case "tech_code_not_received":
		if key == "запросить код повторно" {
			return "code_not_received"
		}
	case "account_category":
		switch key {
		case "не приходит код подтверждения":
			return "account_code_not_received"
		case "забыл пароль":
			return "forgot_password"
		}
	case "account_found":
		if key == "изменить пароль" {
			return "forgot_password"
		}
	case "account_not_found":
		if key == "ввести данные снова" {
			return "ask_account_status"
		}
	case "account_code_not_received":
		if key == "запросить код повторно" {
			return "account_code_not_received"
		}
	case "services_category", "services_rules":
		switch key {
		case "услуги и цены":
			return "ask_prices"
		case "правила":
			return "ask_rules"
		case "адрес и часы работы":
			return "ask_location"
		case "faq":
			return "ask_faq"
		}
	case "services_faq":
		switch key {
		case "как записаться":
			return "ask_faq_booking"
		case "как отменить":
			return "ask_faq_cancellation"
		case "аренда места":
			return "ask_faq_workspace"
		case "задать свой вопрос":
			return "general_question"
		}
	case "services_faq_booking", "services_faq_cancellation", "services_faq_workspace":
		if key == "вернуться в faq" {
			return "ask_faq"
		}
	case "services_faq_not_found", "other_not_classified":
		switch key {
		case "выбрать из категорий", "выбрать категорию вручную":
			return "return_to_menu"
		}
	case "greeting":
		switch key {
		case "выбрать категорию":
			return "return_to_menu"
		case "задать вопрос":
			return "general_question"
		}
	case "escalation_to_operator":
		if key == "отменить остаться в боте" {
			return "return_to_menu"
		}
	}

	return ""
}

func normalizeLegacyOptionKey(text string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(text, "ё", "е")))
}

func clonePayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}

	return cloned
}

func slugifyQuickReplyID(label string) string {
	if label == "" {
		return "quick-reply"
	}

	buf := make([]rune, 0, len(label))
	lastDash := false

	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				r = r - 'A' + 'a'
			}
			buf = append(buf, r)
			lastDash = false
			continue
		}

		if lastDash {
			continue
		}

		buf = append(buf, '-')
		lastDash = true
	}

	id := string(buf)
	for len(id) > 0 && id[0] == '-' {
		id = id[1:]
	}
	for len(id) > 0 && id[len(id)-1] == '-' {
		id = id[:len(id)-1]
	}
	if id == "" {
		return "quick-reply"
	}

	return id
}
