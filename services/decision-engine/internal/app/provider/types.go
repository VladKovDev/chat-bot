package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

const (
	SourceMockExternal = "mock_external"

	StatusFound       = "found"
	StatusNotFound    = "not_found"
	StatusInvalid     = "invalid"
	StatusUnavailable = "unavailable"

	CodeInvalidIdentifier   = "invalid_identifier"
	CodeProviderUnavailable = "provider_unavailable"
	CodeBookingNotFound     = "booking_not_found"
	CodeWorkspaceNotFound   = "workspace_booking_not_found"
	CodePaymentNotFound     = "payment_not_found"
	CodeAccountNotFound     = "account_not_found"
	CodePricingNotFound     = "pricing_not_found"
)

type BookingProvider interface {
	LookupBooking(context.Context, BookingLookupRequest) (BookingLookupResponse, ActionAudit, error)
}

type WorkspaceBookingProvider interface {
	LookupWorkspaceBooking(context.Context, WorkspaceLookupRequest) (WorkspaceLookupResponse, ActionAudit, error)
	CheckWorkspaceAvailability(context.Context, WorkspaceAvailabilityRequest) (WorkspaceAvailabilityResponse, ActionAudit, error)
}

type PaymentProvider interface {
	LookupPayment(context.Context, PaymentLookupRequest) (PaymentLookupResponse, ActionAudit, error)
}

type AccountProvider interface {
	LookupAccount(context.Context, AccountLookupRequest) (AccountLookupResponse, ActionAudit, error)
}

type PricingProvider interface {
	ListPrices(context.Context, PricingLookupRequest) (PricingLookupResponse, ActionAudit, error)
}

type Error struct {
	Provider string
	Code     string
	Message  string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Provider == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s:%s: %s", e.Provider, e.Code, e.Message)
}

type ActionAudit struct {
	Provider   string `json:"provider"`
	Source     string `json:"source"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	ErrorCode  string `json:"error_code,omitempty"`
}

type BookingLookupRequest struct {
	Identifier     string `json:"identifier"`
	IdentifierType string `json:"identifier_type"`
}

type BookingLookupResponse struct {
	Found         bool   `json:"found"`
	BookingNumber string `json:"booking_number,omitempty"`
	Service       string `json:"service,omitempty"`
	Master        string `json:"master,omitempty"`
	Date          string `json:"date,omitempty"`
	Time          string `json:"time,omitempty"`
	Status        string `json:"status,omitempty"`
	Price         int    `json:"price,omitempty"`
	Source        string `json:"source"`
}

type WorkspaceLookupRequest struct {
	Identifier     string `json:"identifier"`
	IdentifierType string `json:"identifier_type"`
}

type WorkspaceLookupResponse struct {
	Found         bool   `json:"found"`
	BookingNumber string `json:"booking_number,omitempty"`
	WorkspaceType string `json:"workspace_type,omitempty"`
	Date          string `json:"date,omitempty"`
	Time          string `json:"time,omitempty"`
	Duration      string `json:"duration,omitempty"`
	Status        string `json:"status,omitempty"`
	DurationHours int    `json:"duration_hours,omitempty"`
	Source        string `json:"source"`
}

type WorkspaceAvailabilityRequest struct {
	Date          string `json:"date"`
	WorkspaceType string `json:"workspace_type"`
}

type WorkspaceAvailabilityResponse struct {
	Found         bool   `json:"found"`
	WorkspaceType string `json:"workspace_type,omitempty"`
	Date          string `json:"date,omitempty"`
	Available     bool   `json:"available"`
	Source        string `json:"source"`
}

type PaymentLookupRequest struct {
	Identifier     string `json:"identifier"`
	IdentifierType string `json:"identifier_type"`
}

type PaymentLookupResponse struct {
	Found     bool   `json:"found"`
	PaymentID string `json:"payment_id,omitempty"`
	Amount    int    `json:"amount,omitempty"`
	Currency  string `json:"currency,omitempty"`
	Date      string `json:"date,omitempty"`
	Status    string `json:"status,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Source    string `json:"source"`
}

type AccountLookupRequest struct {
	Identifier     string `json:"identifier"`
	IdentifierType string `json:"identifier_type"`
}

type AccountLookupResponse struct {
	Found     bool   `json:"found"`
	AccountID string `json:"account_id,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Status    string `json:"status,omitempty"`
	Source    string `json:"source"`
}

type PricingLookupRequest struct {
	Catalog string `json:"catalog"`
}

type PricingLookupResponse struct {
	Found        bool   `json:"found"`
	KnowledgeKey string `json:"knowledge_key,omitempty"`
	Source       string `json:"source"`
}

type ContractDocument struct {
	Version   string                      `json:"version"`
	Providers map[string]ProviderContract `json:"providers"`
	ActionLog ActionLogContract           `json:"action_log"`
}

type ProviderContract struct {
	Endpoints []ProviderEndpointContract `json:"endpoints"`
	Errors    ProviderErrorsContract     `json:"errors"`
}

type ProviderEndpointContract struct {
	Name            string   `json:"name"`
	Method          string   `json:"method"`
	Path            string   `json:"path"`
	RequestFields   []string `json:"request_fields"`
	ResponseFields  []string `json:"response_fields"`
	TimeoutBehavior string   `json:"timeout_behavior"`
}

type ProviderErrorsContract struct {
	NotFound    ProviderErrorContract `json:"not_found"`
	Invalid     ProviderErrorContract `json:"invalid"`
	Unavailable ProviderErrorContract `json:"unavailable"`
}

type ProviderErrorContract struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ActionLogContract struct {
	Required map[string]string `json:"required"`
	Optional map[string]string `json:"optional,omitempty"`
}

func LoadContractDocument(path string) (*ContractDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read contract document: %w", err)
	}

	var doc ContractDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse contract document: %w", err)
	}

	return &doc, nil
}
