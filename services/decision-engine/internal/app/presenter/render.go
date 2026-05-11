package presenter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/VladKovDev/chat-bot/internal/app/processor"
	"github.com/VladKovDev/chat-bot/internal/domain/action"
	"github.com/VladKovDev/chat-bot/internal/domain/response"
	"github.com/VladKovDev/chat-bot/internal/domain/session"
	"github.com/VladKovDev/chat-bot/internal/domain/state"
)

const safeRenderFallbackKey = "error_generic"

type SessionContext struct {
	Channel        string
	ExternalUserID string
	ClientID       string
	ActiveTopic    string
	LastIntent     string
	Mode           session.Mode
	OperatorStatus session.OperatorStatus
	FallbackCount  int
	Metadata       map[string]interface{}
}

type RenderInput struct {
	ResponseKey    string
	Template       *ResponseConfig
	State          state.State
	ActionResults  map[string]processor.ActionResult
	SessionContext SessionContext
	Data           map[string]any
}

func NewSessionContext(sess session.Session) SessionContext {
	metadata := make(map[string]interface{}, len(sess.Metadata))
	for key, value := range sess.Metadata {
		metadata[key] = value
	}

	return SessionContext{
		Channel:        sess.Channel,
		ExternalUserID: sess.ExternalUserID,
		ClientID:       sess.ClientID,
		ActiveTopic:    sess.ActiveTopic,
		LastIntent:     sess.LastIntent,
		Mode:           sess.Mode,
		OperatorStatus: sess.OperatorStatus,
		FallbackCount:  sess.FallbackCount,
		Metadata:       metadata,
	}
}

func (p *Presenter) Render(input RenderInput) (response.Response, error) {
	responseKey := strings.TrimSpace(input.ResponseKey)
	cfg := input.Template
	if cfg == nil {
		loaded, err := p.GetResponse(responseKey)
		if err != nil {
			p.logger.Warn("response key missing, using safe fallback",
				p.logger.String("response_key", responseKey))
			return p.fallbackResponse(input.State, responseKey, err)
		}
		cfg = loaded
	}

	text, missing, err := renderTemplateMessage(cfg.Message, input)
	if err != nil {
		p.logger.Warn("response template invalid, using safe fallback",
			p.logger.String("response_key", responseKey),
			p.logger.Err(err))
		return p.fallbackResponse(input.State, responseKey, err)
	}
	if len(missing) > 0 {
		p.logger.Warn("response template missing placeholder data, using safe fallback",
			p.logger.String("response_key", responseKey),
			p.logger.Any("missing_placeholders", missing))
		return p.fallbackResponse(input.State, responseKey, fmt.Errorf("missing placeholder data: %s", strings.Join(missing, ", ")))
	}

	return response.Response{
		Text:         text,
		Options:      cfg.legacyOptions(),
		QuickReplies: buildResponseQuickReplies(responseKey, cfg),
		State:        input.State,
	}, nil
}

func (p *Presenter) fallbackResponse(st state.State, failedResponseKey string, cause error) (response.Response, error) {
	cfg, err := p.GetResponse(safeRenderFallbackKey)
	if err != nil {
		return response.Response{}, fmt.Errorf("failed to render response %q and safe fallback %q is unavailable: %w: %v", failedResponseKey, safeRenderFallbackKey, err, cause)
	}

	text, missing, renderErr := renderTemplateMessage(cfg.Message, RenderInput{State: st})
	if renderErr != nil {
		return response.Response{}, fmt.Errorf("safe fallback %q has invalid template: %w", safeRenderFallbackKey, renderErr)
	}
	if len(missing) > 0 {
		return response.Response{}, fmt.Errorf("safe fallback %q requires missing placeholders: %s", safeRenderFallbackKey, strings.Join(missing, ", "))
	}

	return response.Response{
		Text:         text,
		Options:      cfg.legacyOptions(),
		QuickReplies: buildResponseQuickReplies(safeRenderFallbackKey, cfg),
		State:        st,
	}, nil
}

func renderTemplateMessage(message string, input RenderInput) (string, []string, error) {
	placeholders, err := extractPlaceholders(message)
	if err != nil {
		return "", nil, err
	}
	if len(placeholders) == 0 {
		return message, nil, nil
	}

	values := renderValues(input)
	missing := make([]string, 0)
	rendered := placeholderPattern.ReplaceAllStringFunc(message, func(token string) string {
		match := placeholderPattern.FindStringSubmatch(token)
		if len(match) < 2 {
			return token
		}

		key := match[1]
		value, ok := values[key]
		if !ok || strings.TrimSpace(value) == "" {
			missing = append(missing, key)
			return token
		}
		if strings.HasPrefix(token, "#{") {
			return "#" + value
		}
		return value
	})
	if len(missing) > 0 {
		slices.Sort(missing)
		missing = slices.Compact(missing)
		return "", missing, nil
	}

	return rendered, nil, nil
}

func renderValues(input RenderInput) map[string]string {
	values := make(map[string]string)

	mergeScalar(values, "channel", input.SessionContext.Channel)
	mergeScalar(values, "external_user_id", input.SessionContext.ExternalUserID)
	mergeScalar(values, "client_id", input.SessionContext.ClientID)
	mergeScalar(values, "active_topic", localizeTopic(input.SessionContext.ActiveTopic))
	mergeScalar(values, "last_intent", input.SessionContext.LastIntent)
	mergeScalar(values, "mode", localizeMode(input.SessionContext.Mode))
	mergeScalar(values, "operator_status", localizeOperatorStatus(input.SessionContext.OperatorStatus))
	mergeScalar(values, "fallback_count", input.SessionContext.FallbackCount)
	mergeAnyMap(values, input.SessionContext.Metadata)
	mergeAnyMap(values, input.Data)

	actionNames := make([]string, 0, len(input.ActionResults))
	for actionName := range input.ActionResults {
		actionNames = append(actionNames, actionName)
	}
	slices.Sort(actionNames)
	for _, actionName := range actionNames {
		result := input.ActionResults[actionName]
		mergeAnyMap(values, result.Data)
		mergeActionAliases(values, actionName, result)
	}

	localizeGenericStatus(values)
	return values
}

func mergeActionAliases(values map[string]string, actionName string, result processor.ActionResult) {
	data := mapFromAny(result.Data)

	switch actionName {
	case action.ActionFindBooking, action.ActionFindWorkspaceBooking:
		if status := stringFromAny(data["booking_status"]); status != "" {
			values["status"] = localizeBookingStatus(status)
		}
	case action.ActionFindPayment:
		if status := stringFromAny(data["payment_status"]); status != "" {
			values["status"] = localizePaymentStatus(status)
		}
		if values["date"] == "" {
			mergeScalar(values, "date", data["created_at"])
		}
	case action.ActionFindUserAccount:
		if status := stringFromAny(data["account_status"]); status != "" {
			values["status"] = localizeAccountStatus(status)
		}
	}

	if workspaceType := stringFromAny(data["workspace_type"]); workspaceType != "" {
		values["workspace_type"] = localizeWorkspaceType(workspaceType)
	}
}

func localizeGenericStatus(values map[string]string) {
	status := values["status"]
	if status == "" {
		return
	}
	if localized := localizeLookupStatus(status); localized != "" {
		values["status"] = localized
	}
}

func mergeAnyMap(values map[string]string, raw any) {
	for key, value := range mapFromAny(raw) {
		mergeScalar(values, key, value)
	}
}

func mapFromAny(raw any) map[string]any {
	switch value := raw.(type) {
	case nil:
		return nil
	case map[string]any:
		return value
	}

	rv := reflect.ValueOf(raw)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

func mergeScalar(values map[string]string, key string, raw any) {
	if key == "" || raw == nil {
		return
	}
	if text := stringFromAny(raw); text != "" {
		values[key] = text
	}
}

func stringFromAny(raw any) string {
	switch value := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	case int:
		return strconv.Itoa(value)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		if value {
			return "да"
		}
		return "нет"
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func localizeBookingStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "confirmed":
		return "подтверждена"
	case "pending":
		return "ожидает подтверждения"
	case "active":
		return "активна"
	case "completed":
		return "завершена"
	case "cancelled":
		return "отменена"
	default:
		return status
	}
}

func localizePaymentStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "completed":
		return "оплачен"
	case "pending":
		return "в обработке"
	case "failed":
		return "не прошел"
	case "debited_not_activated":
		return "списан, услуга не активирована"
	case "refunded":
		return "возвращен"
	default:
		return status
	}
}

func localizeAccountStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "active":
		return "активен"
	case "pending_verification":
		return "ожидает подтверждения"
	case "blocked":
		return "заблокирован"
	case "vip":
		return "VIP"
	default:
		return status
	}
}

func localizeWorkspaceType(workspaceType string) string {
	switch strings.TrimSpace(workspaceType) {
	case "hot_seat":
		return "Горячее место"
	case "fixed_desk":
		return "Фиксированное место"
	case "office_1_3":
		return "Офис на 1-3 человека"
	case "office_4_8":
		return "Офис на 4-8 человек"
	default:
		return workspaceType
	}
}

func localizeLookupStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "found":
		return "найдено"
	case "not_found":
		return "не найдено"
	case "invalid":
		return "некорректные данные"
	case "unavailable":
		return "временно недоступно"
	case "queued_requested":
		return "передано оператору"
	default:
		return ""
	}
}

func localizeTopic(topic string) string {
	switch strings.TrimSpace(topic) {
	case "booking":
		return "записи и бронирование"
	case "workspace":
		return "рабочие места"
	case "payment":
		return "оплата"
	case "tech_issue":
		return "техническая проблема"
	case "account":
		return "аккаунт"
	case "services":
		return "услуги и правила"
	case "complaint":
		return "жалоба"
	case "other":
		return "другое"
	default:
		return topic
	}
}

func localizeMode(mode session.Mode) string {
	switch mode {
	case session.ModeStandard:
		return "бот отвечает"
	case session.ModeWaitingOperator:
		return "ожидаем оператора"
	case session.ModeOperatorConnected:
		return "оператор подключен"
	case session.ModeClosed:
		return "диалог закрыт"
	default:
		return string(mode)
	}
}

func localizeOperatorStatus(status session.OperatorStatus) string {
	switch status {
	case session.OperatorStatusNone:
		return "оператор не подключен"
	case session.OperatorStatusWaiting:
		return "ожидаем оператора"
	case session.OperatorStatusConnected:
		return "оператор подключен"
	case session.OperatorStatusClosed:
		return "обращение закрыто"
	default:
		return string(status)
	}
}
