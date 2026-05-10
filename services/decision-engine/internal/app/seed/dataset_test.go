package seed

import (
	"path/filepath"
	"strings"
	"testing"

	apppresenter "github.com/VladKovDev/chat-bot/internal/app/presenter"
	"github.com/VladKovDev/chat-bot/pkg/logger"
)

func TestLoadActualDatasetAndValidateCatalog(t *testing.T) {
	t.Parallel()

	configPath := serviceRoot(t)

	dataset, err := Load(configPath)
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}

	presenter, err := apppresenter.NewPresenter(filepath.Join(configPath, "configs"))
	if err != nil {
		t.Fatalf("new presenter: %v", err)
	}

	catalog, err := apppresenter.LoadIntentCatalog(configPath)
	if err != nil {
		t.Fatalf("load intent catalog: %v", err)
	}

	validator := apppresenter.NewValidator(presenter.GetAll(), logger.Noop())
	if err := validator.ValidateCatalog(catalog); err != nil {
		t.Fatalf("validate intent catalog: %v", err)
	}

	if err := dataset.ValidateCatalog(catalog); err != nil {
		t.Fatalf("validate dataset catalog: %v", err)
	}
}

func TestValidateCatalogFailsOnMissingKnowledgeAndUnknownQuickReplyIntent(t *testing.T) {
	t.Parallel()

	configPath := serviceRoot(t)

	dataset, err := Load(configPath)
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}

	catalog, err := apppresenter.LoadIntentCatalog(configPath)
	if err != nil {
		t.Fatalf("load intent catalog: %v", err)
	}

	mutated := cloneCatalog(catalog)
	mutated.Intents[0].KnowledgeKey = "missing.knowledge"
	mutated.Intents[0].QuickReplies = append(mutated.Intents[0].QuickReplies, apppresenter.QuickReplyConfig{
		ID:     "broken-intent",
		Label:  "Broken",
		Action: "select_intent",
		Payload: map[string]any{
			"intent": "missing_intent",
		},
	})

	err = dataset.ValidateCatalog(mutated)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "missing knowledge_key") {
		t.Fatalf("expected missing knowledge_key error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "quick reply references missing intent") {
		t.Fatalf("expected missing quick reply intent error, got: %v", err)
	}
}

func TestLookupFixturesCoverSuccessNotFoundAndProviderErrors(t *testing.T) {
	t.Parallel()

	dataset, err := Load(serviceRoot(t))
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}

	successCases := []struct {
		name       string
		identifier string
		lookup     func(string) (map[string]any, error)
		wantField  string
		wantValue  any
	}{
		{
			name:       "booking",
			identifier: "BRG-482910",
			lookup:     dataset.LookupBooking,
			wantField:  "booking_status",
			wantValue:  "confirmed",
		},
		{
			name:       "workspace",
			identifier: "WS-1001",
			lookup:     dataset.LookupWorkspaceBooking,
			wantField:  "booking_status",
			wantValue:  "confirmed",
		},
		{
			name:       "payment",
			identifier: "PAY-123456",
			lookup:     dataset.LookupPayment,
			wantField:  "payment_status",
			wantValue:  "completed",
		},
		{
			name:       "user",
			identifier: "user1@example.com",
			lookup:     dataset.LookupUser,
			wantField:  "account_status",
			wantValue:  "active",
		},
	}

	for _, tc := range successCases {
		tc := tc
		t.Run(tc.name+"_success", func(t *testing.T) {
			t.Parallel()

			result, err := tc.lookup(tc.identifier)
			if err != nil {
				t.Fatalf("lookup(%q): %v", tc.identifier, err)
			}
			if got := result["status"]; got != "found" {
				t.Fatalf("status = %#v, want found", got)
			}
			if got := result[tc.wantField]; got != tc.wantValue {
				t.Fatalf("%s = %#v, want %#v", tc.wantField, got, tc.wantValue)
			}
			if got := result["source"]; got != "mock_external" {
				t.Fatalf("source = %#v, want mock_external", got)
			}
		})
	}

	notFoundCases := []struct {
		name       string
		identifier string
		lookup     func(string) (map[string]any, error)
	}{
		{name: "booking", identifier: "BRG-404000", lookup: dataset.LookupBooking},
		{name: "workspace", identifier: "WS-4040", lookup: dataset.LookupWorkspaceBooking},
		{name: "payment", identifier: "PAY-404000", lookup: dataset.LookupPayment},
		{name: "user", identifier: "missing@example.com", lookup: dataset.LookupUser},
	}

	for _, tc := range notFoundCases {
		tc := tc
		t.Run(tc.name+"_not_found", func(t *testing.T) {
			t.Parallel()

			result, err := tc.lookup(tc.identifier)
			if err != nil {
				t.Fatalf("lookup(%q): %v", tc.identifier, err)
			}
			if got := result["status"]; got != "not_found" {
				t.Fatalf("status = %#v, want not_found", got)
			}
		})
	}

	errorCases := []struct {
		name         string
		identifier   string
		lookup       func(string) (map[string]any, error)
		wantProvider string
	}{
		{name: "booking", identifier: "BRG-ERROR-503", lookup: dataset.LookupBooking, wantProvider: providerBooking},
		{name: "workspace", identifier: "WS-ERROR-503", lookup: dataset.LookupWorkspaceBooking, wantProvider: providerWorkspaceBooking},
		{name: "payment", identifier: "PAY-ERROR-503", lookup: dataset.LookupPayment, wantProvider: providerPayment},
		{name: "user", identifier: "error-user@example.com", lookup: dataset.LookupUser, wantProvider: providerUserAccount},
	}

	for _, tc := range errorCases {
		tc := tc
		t.Run(tc.name+"_provider_error", func(t *testing.T) {
			t.Parallel()

			_, err := tc.lookup(tc.identifier)
			if err == nil {
				t.Fatalf("lookup(%q): expected provider error", tc.identifier)
			}

			providerErr, ok := err.(ProviderError)
			if !ok {
				t.Fatalf("error type = %T, want ProviderError", err)
			}
			if providerErr.Provider != tc.wantProvider {
				t.Fatalf("provider = %q, want %q", providerErr.Provider, tc.wantProvider)
			}
			if providerErr.Code != "provider_unavailable" {
				t.Fatalf("code = %q, want provider_unavailable", providerErr.Code)
			}
		})
	}
}

func TestDatasetFixturesHaveStableIDsAndProviderCoverage(t *testing.T) {
	t.Parallel()

	dataset, err := Load(serviceRoot(t))
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}

	assertUniqueIDs := func(kind string, ids []string) {
		t.Helper()

		seen := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			if strings.TrimSpace(id) == "" {
				t.Fatalf("%s contains empty id", kind)
			}
			if _, exists := seen[id]; exists {
				t.Fatalf("%s contains duplicate id %q", kind, id)
			}
			seen[id] = struct{}{}
		}
	}

	knowledgeIDs := make([]string, 0, len(dataset.KnowledgeBase.Articles))
	for _, article := range dataset.KnowledgeBase.Articles {
		knowledgeIDs = append(knowledgeIDs, article.ID)
	}
	assertUniqueIDs("knowledge articles", knowledgeIDs)

	bookingIDs := make([]string, 0, len(dataset.Bookings.Items))
	for _, fixture := range dataset.Bookings.Items {
		bookingIDs = append(bookingIDs, fixture.ID)
	}
	assertUniqueIDs("booking fixtures", bookingIDs)

	workspaceIDs := make([]string, 0, len(dataset.WorkspaceBookings.Items))
	for _, fixture := range dataset.WorkspaceBookings.Items {
		workspaceIDs = append(workspaceIDs, fixture.ID)
	}
	assertUniqueIDs("workspace fixtures", workspaceIDs)

	paymentIDs := make([]string, 0, len(dataset.Payments.Items))
	for _, fixture := range dataset.Payments.Items {
		paymentIDs = append(paymentIDs, fixture.ID)
	}
	assertUniqueIDs("payment fixtures", paymentIDs)

	userIDs := make([]string, 0, len(dataset.Users.Items))
	for _, fixture := range dataset.Users.Items {
		userIDs = append(userIDs, fixture.ID)
	}
	assertUniqueIDs("user fixtures", userIDs)

	operatorIDs := make([]string, 0, len(dataset.Operators.Items))
	for _, fixture := range dataset.Operators.Items {
		operatorIDs = append(operatorIDs, fixture.ID)
	}
	assertUniqueIDs("operator fixtures", operatorIDs)

	for _, provider := range dataset.Providers.Providers {
		if len(provider.SuccessFixtureIDs) == 0 {
			t.Fatalf("provider %s has no success fixtures", provider.Provider)
		}
		if len(provider.NotFoundIdentifiers) == 0 {
			t.Fatalf("provider %s has no not_found identifiers", provider.Provider)
		}
		if len(provider.ErrorCases) == 0 {
			t.Fatalf("provider %s has no error cases", provider.Provider)
		}

		errorCaseIDs := make([]string, 0, len(provider.ErrorCases))
		for _, errorCase := range provider.ErrorCases {
			errorCaseIDs = append(errorCaseIDs, errorCase.ID)
		}
		assertUniqueIDs("provider "+provider.Provider+" error cases", errorCaseIDs)
	}
}

func cloneCatalog(catalog *apppresenter.IntentCatalog) *apppresenter.IntentCatalog {
	cloned := &apppresenter.IntentCatalog{
		Intents: make([]apppresenter.IntentDefinition, len(catalog.Intents)),
	}

	for i, intentDef := range catalog.Intents {
		intentCopy := intentDef
		intentCopy.Examples = append([]string(nil), intentDef.Examples...)
		intentCopy.ResultResponseKeys = append([]string(nil), intentDef.ResultResponseKeys...)
		intentCopy.E2ECoverage = append([]string(nil), intentDef.E2ECoverage...)
		intentCopy.QuickReplies = append([]apppresenter.QuickReplyConfig(nil), intentDef.QuickReplies...)
		cloned.Intents[i] = intentCopy
	}

	return cloned
}

func serviceRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("service root abs: %v", err)
	}

	return root
}
