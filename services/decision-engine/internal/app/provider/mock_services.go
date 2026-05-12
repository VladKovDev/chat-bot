package provider

import (
	"context"
	"strings"
	"time"

	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
)

const providerTimeout = 2800 * time.Millisecond

type MockBookingService struct {
	dataset *appseed.Dataset
}

type MockWorkspaceService struct {
	dataset *appseed.Dataset
}

type MockPaymentService struct {
	dataset *appseed.Dataset
}

type MockAccountService struct {
	dataset *appseed.Dataset
}

type MockPricingService struct {
	dataset *appseed.Dataset
}

func NewMockBookingService(dataset *appseed.Dataset) *MockBookingService {
	return &MockBookingService{dataset: resolveDataset(dataset)}
}

func NewMockWorkspaceService(dataset *appseed.Dataset) *MockWorkspaceService {
	return &MockWorkspaceService{dataset: resolveDataset(dataset)}
}

func NewMockPaymentService(dataset *appseed.Dataset) *MockPaymentService {
	return &MockPaymentService{dataset: resolveDataset(dataset)}
}

func NewMockAccountService(dataset *appseed.Dataset) *MockAccountService {
	return &MockAccountService{dataset: resolveDataset(dataset)}
}

func NewMockPricingService(dataset *appseed.Dataset) *MockPricingService {
	return &MockPricingService{dataset: resolveDataset(dataset)}
}

func (s *MockBookingService) LookupBooking(ctx context.Context, req BookingLookupRequest) (BookingLookupResponse, ActionAudit, error) {
	start := time.Now()
	audit := baseAudit("booking")
	callCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	if err := providerContextErr(callCtx); err != nil {
		return BookingLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("booking", CodeProviderUnavailable, "booking provider unavailable")
	}
	req.Identifier = normalizeLookupPhone(req.Identifier, req.IdentifierType)

	if !validBookingIdentifier(req.Identifier, req.IdentifierType) {
		return BookingLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusInvalid, CodeInvalidIdentifier), providerError("booking", CodeInvalidIdentifier, "invalid booking identifier")
	}
	if isSyntheticUnavailable("booking", req.Identifier) {
		return BookingLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("booking", CodeProviderUnavailable, "booking provider unavailable")
	}

	dataset := resolveDataset(s.dataset)
	if dataset == nil {
		return BookingLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("booking", CodeProviderUnavailable, "booking dataset unavailable")
	}

	result, err := dataset.LookupBooking(req.Identifier)
	if err != nil {
		finalAudit, providerErr := finalizeLookupError("booking", audit, start, err)
		return BookingLookupResponse{Source: SourceMockExternal}, finalAudit, providerErr
	}
	if result["status"] == StatusNotFound {
		return BookingLookupResponse{Found: false, Source: SourceMockExternal}, finalizeAudit(audit, start, StatusNotFound, CodeBookingNotFound), nil
	}

	return BookingLookupResponse{
		Found:         true,
		BookingNumber: stringValue(result["booking_number"]),
		Service:       stringValue(result["service"]),
		Master:        stringValue(result["master"]),
		Date:          stringValue(result["date"]),
		Time:          stringValue(result["time"]),
		Status:        stringValue(result["booking_status"]),
		Price:         intValue(result["price"]),
		Source:        SourceMockExternal,
	}, finalizeAudit(audit, start, StatusFound, ""), nil
}

func (s *MockWorkspaceService) LookupWorkspaceBooking(ctx context.Context, req WorkspaceLookupRequest) (WorkspaceLookupResponse, ActionAudit, error) {
	start := time.Now()
	audit := baseAudit("workspace_booking")
	callCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	if err := providerContextErr(callCtx); err != nil {
		return WorkspaceLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("workspace_booking", CodeProviderUnavailable, "workspace provider unavailable")
	}

	if !validWorkspaceIdentifier(req.Identifier, req.IdentifierType) {
		return WorkspaceLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusInvalid, CodeInvalidIdentifier), providerError("workspace_booking", CodeInvalidIdentifier, "invalid workspace identifier")
	}
	if isSyntheticUnavailable("workspace_booking", req.Identifier) {
		return WorkspaceLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("workspace_booking", CodeProviderUnavailable, "workspace provider unavailable")
	}

	dataset := resolveDataset(s.dataset)
	if dataset == nil {
		return WorkspaceLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("workspace_booking", CodeProviderUnavailable, "workspace dataset unavailable")
	}

	result, err := dataset.LookupWorkspaceBooking(req.Identifier)
	if err != nil {
		finalAudit, providerErr := finalizeLookupError("workspace_booking", audit, start, err)
		return WorkspaceLookupResponse{Source: SourceMockExternal}, finalAudit, providerErr
	}
	if result["status"] == StatusNotFound {
		return WorkspaceLookupResponse{Found: false, Source: SourceMockExternal}, finalizeAudit(audit, start, StatusNotFound, CodeWorkspaceNotFound), nil
	}

	return WorkspaceLookupResponse{
		Found:         true,
		BookingNumber: stringValue(result["booking_number"]),
		WorkspaceType: stringValue(result["workspace_type"]),
		Date:          stringValue(result["date"]),
		Time:          stringValue(result["time"]),
		Duration:      stringValue(result["duration"]),
		Status:        stringValue(result["booking_status"]),
		DurationHours: intValue(result["duration_hours"]),
		Source:        SourceMockExternal,
	}, finalizeAudit(audit, start, StatusFound, ""), nil
}

func (s *MockWorkspaceService) CheckWorkspaceAvailability(ctx context.Context, req WorkspaceAvailabilityRequest) (WorkspaceAvailabilityResponse, ActionAudit, error) {
	start := time.Now()
	audit := baseAudit("workspace_booking")
	callCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	if err := providerContextErr(callCtx); err != nil {
		return WorkspaceAvailabilityResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("workspace_booking", CodeProviderUnavailable, "workspace provider unavailable")
	}

	workspaceType := strings.TrimSpace(req.WorkspaceType)
	date := strings.TrimSpace(req.Date)
	if !validAvailabilityRequest(date, workspaceType) {
		return WorkspaceAvailabilityResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusInvalid, CodeInvalidIdentifier), providerError("workspace_booking", CodeInvalidIdentifier, "invalid workspace availability request")
	}

	dataset := resolveDataset(s.dataset)
	if dataset == nil {
		return WorkspaceAvailabilityResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("workspace_booking", CodeProviderUnavailable, "workspace dataset unavailable")
	}

	if errCase, ok := workspaceAvailabilityError(dataset, date, workspaceType); ok {
		return WorkspaceAvailabilityResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, errCase.Code), providerError("workspace_booking", errCase.Code, errCase.Message)
	}

	if availabilityOutOfFixtureRange(date) {
		return WorkspaceAvailabilityResponse{Found: false, Date: date, WorkspaceType: workspaceType, Source: SourceMockExternal}, finalizeAudit(audit, start, StatusNotFound, CodeWorkspaceNotFound), nil
	}

	return WorkspaceAvailabilityResponse{
		Found:         true,
		WorkspaceType: workspaceType,
		Date:          date,
		Available:     workspaceAvailable(dataset, date, workspaceType),
		Source:        SourceMockExternal,
	}, finalizeAudit(audit, start, StatusFound, ""), nil
}

func (s *MockPaymentService) LookupPayment(ctx context.Context, req PaymentLookupRequest) (PaymentLookupResponse, ActionAudit, error) {
	start := time.Now()
	audit := baseAudit("payment")
	callCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	if err := providerContextErr(callCtx); err != nil {
		return PaymentLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("payment", CodeProviderUnavailable, "payment provider unavailable")
	}

	if !validPaymentIdentifier(req.Identifier, req.IdentifierType) {
		return PaymentLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusInvalid, CodeInvalidIdentifier), providerError("payment", CodeInvalidIdentifier, "invalid payment identifier")
	}

	dataset := resolveDataset(s.dataset)
	if dataset == nil {
		return PaymentLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("payment", CodeProviderUnavailable, "payment dataset unavailable")
	}

	result, err := dataset.LookupPayment(req.Identifier)
	if err != nil {
		finalAudit, providerErr := finalizeLookupError("payment", audit, start, err)
		return PaymentLookupResponse{Source: SourceMockExternal}, finalAudit, providerErr
	}
	if result["status"] == StatusNotFound {
		return PaymentLookupResponse{Found: false, Source: SourceMockExternal}, finalizeAudit(audit, start, StatusNotFound, CodePaymentNotFound), nil
	}

	return PaymentLookupResponse{
		Found:     true,
		PaymentID: stringValue(result["payment_id"]),
		Amount:    intValue(result["amount"]),
		Currency:  firstStringValue("RUB", result["currency"]),
		Date:      stringValue(result["date"]),
		Status:    stringValue(result["payment_status"]),
		Purpose:   stringValue(result["purpose"]),
		CreatedAt: firstStringValue(stringValue(result["date"]), result["created_at"]),
		Source:    SourceMockExternal,
	}, finalizeAudit(audit, start, StatusFound, ""), nil
}

func (s *MockAccountService) LookupAccount(ctx context.Context, req AccountLookupRequest) (AccountLookupResponse, ActionAudit, error) {
	start := time.Now()
	audit := baseAudit("user_account")
	callCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	if err := providerContextErr(callCtx); err != nil {
		return AccountLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("user_account", CodeProviderUnavailable, "account provider unavailable")
	}
	req.Identifier = normalizeLookupPhone(req.Identifier, req.IdentifierType)

	if !validAccountIdentifier(req.Identifier, req.IdentifierType) {
		return AccountLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusInvalid, CodeInvalidIdentifier), providerError("user_account", CodeInvalidIdentifier, "invalid account identifier")
	}

	dataset := resolveDataset(s.dataset)
	if dataset == nil {
		return AccountLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("user_account", CodeProviderUnavailable, "account dataset unavailable")
	}

	result, err := dataset.LookupUser(req.Identifier)
	if err != nil {
		finalAudit, providerErr := finalizeLookupError("user_account", audit, start, err)
		return AccountLookupResponse{Source: SourceMockExternal}, finalAudit, providerErr
	}
	if result["status"] == StatusNotFound {
		return AccountLookupResponse{Found: false, Source: SourceMockExternal}, finalizeAudit(audit, start, StatusNotFound, CodeAccountNotFound), nil
	}

	return AccountLookupResponse{
		Found:     true,
		AccountID: stringValue(result["user_id"]),
		Email:     stringValue(result["email"]),
		Phone:     stringValue(result["phone"]),
		Status:    stringValue(result["account_status"]),
		Source:    SourceMockExternal,
	}, finalizeAudit(audit, start, StatusFound, ""), nil
}

func (s *MockPricingService) ListPrices(ctx context.Context, req PricingLookupRequest) (PricingLookupResponse, ActionAudit, error) {
	start := time.Now()
	audit := baseAudit("pricing")
	callCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	if err := providerContextErr(callCtx); err != nil {
		return PricingLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("pricing", CodeProviderUnavailable, "pricing provider unavailable")
	}
	catalog := strings.TrimSpace(req.Catalog)

	if catalog == "" {
		return PricingLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusInvalid, CodeInvalidIdentifier), providerError("pricing", CodeInvalidIdentifier, "invalid pricing catalog")
	}
	if isSyntheticUnavailable("pricing", catalog) {
		return PricingLookupResponse{Source: SourceMockExternal}, finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable), providerError("pricing", CodeProviderUnavailable, "pricing provider unavailable")
	}

	knowledgeKey := pricingKnowledgeKey(catalog)
	if knowledgeKey == "" {
		return PricingLookupResponse{Found: false, Source: SourceMockExternal}, finalizeAudit(audit, start, StatusNotFound, CodePricingNotFound), nil
	}

	return PricingLookupResponse{
		Found:        true,
		KnowledgeKey: knowledgeKey,
		Source:       SourceMockExternal,
	}, finalizeAudit(audit, start, StatusFound, ""), nil
}

func providerContextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func resolveDataset(dataset *appseed.Dataset) *appseed.Dataset {
	if dataset != nil {
		return dataset
	}

	loaded, err := appseed.Load(".")
	if err != nil {
		return nil
	}

	return loaded
}

func baseAudit(providerName string) ActionAudit {
	return ActionAudit{
		Provider: providerName,
		Source:   SourceMockExternal,
	}
}

func finalizeAudit(audit ActionAudit, start time.Time, status, errorCode string) ActionAudit {
	audit.Status = status
	audit.DurationMS = time.Since(start).Milliseconds()
	audit.ErrorCode = errorCode
	return audit
}

func finalizeErrorAudit(audit ActionAudit, start time.Time, status, errorCode string) ActionAudit {
	return finalizeAudit(audit, start, status, errorCode)
}

func finalizeLookupError(providerName string, audit ActionAudit, start time.Time, err error) (ActionAudit, error) {
	if seedErr, ok := err.(appseed.ProviderError); ok {
		code := seedErr.Code
		if code == "" {
			code = CodeProviderUnavailable
		}
		finalAudit := finalizeErrorAudit(audit, start, StatusUnavailable, code)
		return finalAudit, providerError(providerName, code, seedErr.Message)
	}

	finalAudit := finalizeErrorAudit(audit, start, StatusUnavailable, CodeProviderUnavailable)
	return finalAudit, providerError(providerName, CodeProviderUnavailable, err.Error())
}

func providerError(providerName, code, message string) error {
	return &Error{
		Provider: providerName,
		Code:     code,
		Message:  message,
	}
}

func stringValue(value any) string {
	str, _ := value.(string)
	return str
}

func firstStringValue(fallback string, value any) string {
	if str := stringValue(value); str != "" {
		return str
	}
	return fallback
}

func intValue(value any) int {
	number, _ := value.(int)
	return number
}

func validBookingIdentifier(identifier, identifierType string) bool {
	identifier = strings.TrimSpace(identifier)
	switch identifierType {
	case "", "booking_number":
		return matchesAny(identifier, "^BRG-\\d{6}$", "^БРГ-\\d{6}$", "^BRG-ERROR-503$")
	case "phone":
		return matchesAny(identifier, "^\\+7 \\(\\d{3}\\) \\d{3}-\\d{2}-\\d{2}$", "^\\d{10}$", "^[78]\\d{10}$")
	default:
		return false
	}
}

func validWorkspaceIdentifier(identifier, identifierType string) bool {
	identifier = strings.TrimSpace(identifier)
	switch identifierType {
	case "", "workspace_booking":
		return matchesAny(identifier, "^WS-\\d{4}$", "^WRK-(HOT|FIX|OFC1|OFC4)-\\d{3}$", "^WS-ERROR-503$")
	default:
		return false
	}
}

func validPaymentIdentifier(identifier, identifierType string) bool {
	identifier = strings.TrimSpace(identifier)
	switch identifierType {
	case "", "payment_id":
		return matchesAny(identifier, "^PAY-[A-Z0-9-]{3,}$")
	case "booking_number":
		return matchesAny(identifier, "^BRG-\\d{6}$", "^БРГ-\\d{6}$")
	case "workspace_booking":
		return matchesAny(identifier, "^WS-\\d{4}$", "^WRK-(HOT|FIX|OFC1|OFC4)-\\d{3}$")
	case "order_id":
		return matchesAny(identifier, "^ORDER-[A-Z0-9-]{3,}$")
	default:
		return false
	}
}

func validAccountIdentifier(identifier, identifierType string) bool {
	identifier = strings.TrimSpace(identifier)
	switch identifierType {
	case "", "user_id":
		return matchesAny(identifier, "^usr-\\d{6}$")
	case "email":
		return matchesAny(identifier, "^[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}$")
	case "phone":
		return matchesAny(identifier, "^\\+7 \\(\\d{3}\\) \\d{3}-\\d{2}-\\d{2}$", "^\\d{10}$", "^[78]\\d{10}$")
	default:
		return false
	}
}

func normalizeLookupPhone(identifier, identifierType string) string {
	if identifierType != "phone" {
		return identifier
	}

	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return trimmed
	}

	digits := make([]rune, 0, len(trimmed))
	for _, r := range trimmed {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}

	switch len(digits) {
	case 10:
		return "7" + string(digits)
	case 11:
		if digits[0] == '8' {
			digits[0] = '7'
		}
		return string(digits)
	default:
		return identifier
	}
}

func pricingKnowledgeKey(catalog string) string {
	switch strings.ToLower(strings.TrimSpace(catalog)) {
	case "workspace":
		return "workspace.prices"
	case "services":
		return "services.prices"
	case "workspace_rules", "workspace-rules", "rules":
		return "workspace.rules"
	default:
		return ""
	}
}

func validAvailabilityRequest(date, workspaceType string) bool {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return false
	}
	return isKnownWorkspaceType(workspaceType)
}

func isKnownWorkspaceType(workspaceType string) bool {
	switch strings.TrimSpace(workspaceType) {
	case "hot_seat", "fixed_desk", "office_1_3", "office_4_8":
		return true
	default:
		return false
	}
}

func workspaceAvailabilityError(dataset *appseed.Dataset, date, workspaceType string) (appseed.ProviderErrorCase, bool) {
	return dataset.ProviderErrorCase("workspace_booking", "availability:"+date+":"+workspaceType)
}

func availabilityOutOfFixtureRange(date string) bool {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return true
	}
	return parsed.Before(time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)) ||
		parsed.After(time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC))
}

func workspaceAvailable(dataset *appseed.Dataset, date, workspaceType string) bool {
	for _, fixture := range dataset.WorkspaceBookings.Items {
		if fixture.WorkspaceType != workspaceType {
			continue
		}
		if workspaceFixtureISODate(fixture.Date) != date {
			continue
		}
		switch fixture.Status {
		case "confirmed", "pending", "active":
			return false
		}
	}
	return true
}

func workspaceFixtureISODate(date string) string {
	parsed, err := time.Parse("02.01.2006", strings.TrimSpace(date))
	if err != nil {
		return strings.TrimSpace(date)
	}
	return parsed.Format("2006-01-02")
}

func isSyntheticUnavailable(providerName, identifier string) bool {
	key := strings.ToUpper(strings.TrimSpace(identifier))

	switch providerName {
	case "booking":
		return key == "BRG-503503" || key == "BRG-ERROR-503"
	case "workspace_booking":
		return key == "WS-5030" || key == "WS-ERROR-503"
	case "payment":
		return key == "PAY-ERROR-503"
	case "user_account":
		return strings.EqualFold(strings.TrimSpace(identifier), "error-user@example.com")
	case "pricing":
		return strings.EqualFold(strings.TrimSpace(identifier), "pricing-error")
	default:
		return false
	}
}
