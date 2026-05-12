package provider

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	appseed "github.com/VladKovDev/chat-bot/internal/app/seed"
)

func TestMockProvidersCoverSuccessNotFoundInvalidAndUnavailable(t *testing.T) {
	t.Parallel()

	dataset := mustLoadDataset(t)
	ctx := context.Background()

	t.Run("booking", func(t *testing.T) {
		t.Parallel()

		svc := NewMockBookingService(dataset)

		response, audit, err := svc.LookupBooking(ctx, BookingLookupRequest{
			Identifier:     "BRG-482910",
			IdentifierType: "booking_number",
		})
		if err != nil {
			t.Fatalf("lookup success: %v", err)
		}
		if !response.Found || response.BookingNumber != "BRG-482910" {
			t.Fatalf("success response = %+v", response)
		}
		assertAuditFields(t, audit, "booking", StatusFound, "")

		response, audit, err = svc.LookupBooking(ctx, BookingLookupRequest{
			Identifier:     "BRG-404000",
			IdentifierType: "booking_number",
		})
		if err != nil {
			t.Fatalf("lookup not_found: %v", err)
		}
		if response.Found {
			t.Fatalf("not_found response = %+v, want found=false", response)
		}
		if response.Source != SourceMockExternal {
			t.Fatalf("not_found source = %q, want %q", response.Source, SourceMockExternal)
		}
		assertAuditFields(t, audit, "booking", StatusNotFound, CodeBookingNotFound)

		response, audit, err = svc.LookupBooking(ctx, BookingLookupRequest{
			Identifier:     "89991234567",
			IdentifierType: "phone",
		})
		if err != nil {
			t.Fatalf("lookup by phone with 8-prefix: %v", err)
		}
		if !response.Found || response.BookingNumber != "BRG-482910" {
			t.Fatalf("phone success response = %+v", response)
		}
		assertAuditFields(t, audit, "booking", StatusFound, "")

		response, audit, err = svc.LookupBooking(ctx, BookingLookupRequest{
			Identifier:     "BRG-ABC",
			IdentifierType: "booking_number",
		})
		if response.Source != SourceMockExternal {
			t.Fatalf("invalid source = %q, want %q", response.Source, SourceMockExternal)
		}
		assertProviderErrorCode(t, err, CodeInvalidIdentifier)
		assertAuditFields(t, audit, "booking", StatusInvalid, CodeInvalidIdentifier)

		response, audit, err = svc.LookupBooking(ctx, BookingLookupRequest{
			Identifier:     "BRG-503503",
			IdentifierType: "booking_number",
		})
		if response.Source != SourceMockExternal {
			t.Fatalf("unavailable source = %q, want %q", response.Source, SourceMockExternal)
		}
		assertProviderErrorCode(t, err, CodeProviderUnavailable)
		assertAuditFields(t, audit, "booking", StatusUnavailable, CodeProviderUnavailable)
	})

	t.Run("workspace", func(t *testing.T) {
		t.Parallel()

		svc := NewMockWorkspaceService(dataset)

		response, audit, err := svc.LookupWorkspaceBooking(ctx, WorkspaceLookupRequest{
			Identifier:     "WS-1001",
			IdentifierType: "workspace_booking",
		})
		if err != nil {
			t.Fatalf("lookup success: %v", err)
		}
		if !response.Found || response.BookingNumber != "WS-1001" {
			t.Fatalf("success response = %+v", response)
		}
		assertAuditFields(t, audit, "workspace_booking", StatusFound, "")

		response, audit, err = svc.LookupWorkspaceBooking(ctx, WorkspaceLookupRequest{
			Identifier:     "WS-4040",
			IdentifierType: "workspace_booking",
		})
		if err != nil {
			t.Fatalf("lookup not_found: %v", err)
		}
		if response.Found {
			t.Fatalf("not_found response = %+v, want found=false", response)
		}
		assertAuditFields(t, audit, "workspace_booking", StatusNotFound, CodeWorkspaceNotFound)

		_, audit, err = svc.LookupWorkspaceBooking(ctx, WorkspaceLookupRequest{
			Identifier:     "WS-ABCD",
			IdentifierType: "workspace_booking",
		})
		assertProviderErrorCode(t, err, CodeInvalidIdentifier)
		assertAuditFields(t, audit, "workspace_booking", StatusInvalid, CodeInvalidIdentifier)

		_, audit, err = svc.LookupWorkspaceBooking(ctx, WorkspaceLookupRequest{
			Identifier:     "WS-5030",
			IdentifierType: "workspace_booking",
		})
		assertProviderErrorCode(t, err, CodeProviderUnavailable)
		assertAuditFields(t, audit, "workspace_booking", StatusUnavailable, CodeProviderUnavailable)

		availability, audit, err := svc.CheckWorkspaceAvailability(ctx, WorkspaceAvailabilityRequest{
			Date:          "2026-05-20",
			WorkspaceType: "hot_seat",
		})
		if err != nil {
			t.Fatalf("availability success: %v", err)
		}
		if !availability.Found || !availability.Available || availability.Source != SourceMockExternal {
			t.Fatalf("availability success response = %+v", availability)
		}
		assertAuditFields(t, audit, "workspace_booking", StatusFound, "")

		availability, audit, err = svc.CheckWorkspaceAvailability(ctx, WorkspaceAvailabilityRequest{
			Date:          "2027-01-01",
			WorkspaceType: "hot_seat",
		})
		if err != nil {
			t.Fatalf("availability not_found: %v", err)
		}
		if availability.Found {
			t.Fatalf("availability not_found response = %+v, want found=false", availability)
		}
		assertAuditFields(t, audit, "workspace_booking", StatusNotFound, CodeWorkspaceNotFound)

		_, audit, err = svc.CheckWorkspaceAvailability(ctx, WorkspaceAvailabilityRequest{
			Date:          "not-a-date",
			WorkspaceType: "hot_seat",
		})
		assertProviderErrorCode(t, err, CodeInvalidIdentifier)
		assertAuditFields(t, audit, "workspace_booking", StatusInvalid, CodeInvalidIdentifier)

		_, audit, err = svc.CheckWorkspaceAvailability(ctx, WorkspaceAvailabilityRequest{
			Date:          "2026-05-20",
			WorkspaceType: "provider_error",
		})
		assertProviderErrorCode(t, err, CodeInvalidIdentifier)
		assertAuditFields(t, audit, "workspace_booking", StatusInvalid, CodeInvalidIdentifier)

		_, audit, err = svc.CheckWorkspaceAvailability(ctx, WorkspaceAvailabilityRequest{
			Date:          "2026-05-20",
			WorkspaceType: "office_4_8",
		})
		assertProviderErrorCode(t, err, CodeProviderUnavailable)
		assertAuditFields(t, audit, "workspace_booking", StatusUnavailable, CodeProviderUnavailable)
	})

	t.Run("payment", func(t *testing.T) {
		t.Parallel()

		svc := NewMockPaymentService(dataset)

		response, audit, err := svc.LookupPayment(ctx, PaymentLookupRequest{
			Identifier:     "PAY-123456",
			IdentifierType: "payment_id",
		})
		if err != nil {
			t.Fatalf("lookup success: %v", err)
		}
		if !response.Found || response.PaymentID != "PAY-123456" {
			t.Fatalf("success response = %+v", response)
		}
		assertAuditFields(t, audit, "payment", StatusFound, "")

		response, audit, err = svc.LookupPayment(ctx, PaymentLookupRequest{
			Identifier:     "PAY-404000",
			IdentifierType: "payment_id",
		})
		if err != nil {
			t.Fatalf("lookup not_found: %v", err)
		}
		if response.Found {
			t.Fatalf("not_found response = %+v, want found=false", response)
		}
		assertAuditFields(t, audit, "payment", StatusNotFound, CodePaymentNotFound)

		_, audit, err = svc.LookupPayment(ctx, PaymentLookupRequest{
			Identifier:     "PAY",
			IdentifierType: "payment_id",
		})
		assertProviderErrorCode(t, err, CodeInvalidIdentifier)
		assertAuditFields(t, audit, "payment", StatusInvalid, CodeInvalidIdentifier)

		_, audit, err = svc.LookupPayment(ctx, PaymentLookupRequest{
			Identifier:     "PAY-ERROR-503",
			IdentifierType: "payment_id",
		})
		assertProviderErrorCode(t, err, CodeProviderUnavailable)
		assertAuditFields(t, audit, "payment", StatusUnavailable, CodeProviderUnavailable)
	})

	t.Run("account", func(t *testing.T) {
		t.Parallel()

		svc := NewMockAccountService(dataset)

		response, audit, err := svc.LookupAccount(ctx, AccountLookupRequest{
			Identifier:     "user1@example.com",
			IdentifierType: "email",
		})
		if err != nil {
			t.Fatalf("lookup success: %v", err)
		}
		if !response.Found || response.AccountID != "usr-100001" {
			t.Fatalf("success response = %+v", response)
		}
		assertAuditFields(t, audit, "user_account", StatusFound, "")

		response, audit, err = svc.LookupAccount(ctx, AccountLookupRequest{
			Identifier:     "missing@example.com",
			IdentifierType: "email",
		})
		if err != nil {
			t.Fatalf("lookup not_found: %v", err)
		}
		if response.Found {
			t.Fatalf("not_found response = %+v, want found=false", response)
		}
		assertAuditFields(t, audit, "user_account", StatusNotFound, CodeAccountNotFound)

		response, audit, err = svc.LookupAccount(ctx, AccountLookupRequest{
			Identifier:     "89991234567",
			IdentifierType: "phone",
		})
		if err != nil {
			t.Fatalf("lookup account by phone with 8-prefix: %v", err)
		}
		if !response.Found || response.AccountID != "usr-100001" {
			t.Fatalf("phone success response = %+v", response)
		}
		assertAuditFields(t, audit, "user_account", StatusFound, "")

		_, audit, err = svc.LookupAccount(ctx, AccountLookupRequest{
			Identifier:     "broken-email",
			IdentifierType: "email",
		})
		assertProviderErrorCode(t, err, CodeInvalidIdentifier)
		assertAuditFields(t, audit, "user_account", StatusInvalid, CodeInvalidIdentifier)

		_, audit, err = svc.LookupAccount(ctx, AccountLookupRequest{
			Identifier:     "error-user@example.com",
			IdentifierType: "email",
		})
		assertProviderErrorCode(t, err, CodeProviderUnavailable)
		assertAuditFields(t, audit, "user_account", StatusUnavailable, CodeProviderUnavailable)
	})

	t.Run("pricing", func(t *testing.T) {
		t.Parallel()

		svc := NewMockPricingService(dataset)

		response, audit, err := svc.ListPrices(ctx, PricingLookupRequest{
			Catalog: "workspace",
		})
		if err != nil {
			t.Fatalf("lookup success: %v", err)
		}
		if !response.Found || response.KnowledgeKey != "workspace.prices" {
			t.Fatalf("success response = %+v", response)
		}
		assertAuditFields(t, audit, "pricing", StatusFound, "")

		response, audit, err = svc.ListPrices(ctx, PricingLookupRequest{
			Catalog: "pricing-missing",
		})
		if err != nil {
			t.Fatalf("lookup not_found: %v", err)
		}
		if response.Found {
			t.Fatalf("not_found response = %+v, want found=false", response)
		}
		assertAuditFields(t, audit, "pricing", StatusNotFound, CodePricingNotFound)

		response, audit, err = svc.ListPrices(ctx, PricingLookupRequest{
			Catalog: "workspace_rules",
		})
		if err != nil {
			t.Fatalf("rules lookup success: %v", err)
		}
		if !response.Found || response.KnowledgeKey != "workspace.rules" {
			t.Fatalf("rules response = %+v", response)
		}
		assertAuditFields(t, audit, "pricing", StatusFound, "")

		_, audit, err = svc.ListPrices(ctx, PricingLookupRequest{
			Catalog: "   ",
		})
		assertProviderErrorCode(t, err, CodeInvalidIdentifier)
		assertAuditFields(t, audit, "pricing", StatusInvalid, CodeInvalidIdentifier)

		_, audit, err = svc.ListPrices(ctx, PricingLookupRequest{
			Catalog: "pricing-error",
		})
		assertProviderErrorCode(t, err, CodeProviderUnavailable)
		assertAuditFields(t, audit, "pricing", StatusUnavailable, CodeProviderUnavailable)
	})
}

func TestMockProviderContractDocumentIncludesAllProvidersAndAuditFields(t *testing.T) {
	t.Parallel()

	contract, err := LoadContractDocument(filepath.Join(serviceRoot(t), "contracts", "mock-external-providers-v1.json"))
	if err != nil {
		t.Fatalf("load contract document: %v", err)
	}

	providers := []string{"booking", "workspace_booking", "payment", "user_account", "pricing"}
	for _, providerName := range providers {
		providerContract, ok := contract.Providers[providerName]
		if !ok {
			t.Fatalf("missing provider contract for %s", providerName)
		}
		if len(providerContract.Endpoints) == 0 {
			t.Fatalf("provider %s has no endpoints", providerName)
		}
		if providerContract.Errors.NotFound.Code == "" {
			t.Fatalf("provider %s missing not_found error code", providerName)
		}
		if providerContract.Errors.Invalid.Code == "" {
			t.Fatalf("provider %s missing invalid error code", providerName)
		}
		if providerContract.Errors.Unavailable.Code == "" {
			t.Fatalf("provider %s missing unavailable error code", providerName)
		}
	}

	requiredAuditFields := []string{"provider", "source", "status", "duration_ms"}
	for _, field := range requiredAuditFields {
		if _, ok := contract.ActionLog.Required[field]; !ok {
			t.Fatalf("missing required action log field %q", field)
		}
	}
}

func TestMockProvidersRespectTimeoutBudget(t *testing.T) {
	t.Parallel()

	dataset := mustLoadDataset(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	cases := []struct {
		name string
		call func() (ActionAudit, error)
	}{
		{
			name: "booking",
			call: func() (ActionAudit, error) {
				_, audit, err := NewMockBookingService(dataset).LookupBooking(cancelled, BookingLookupRequest{Identifier: "BRG-482910", IdentifierType: "booking_number"})
				return audit, err
			},
		},
		{
			name: "workspace_booking",
			call: func() (ActionAudit, error) {
				_, audit, err := NewMockWorkspaceService(dataset).LookupWorkspaceBooking(cancelled, WorkspaceLookupRequest{Identifier: "WS-1001", IdentifierType: "workspace_booking"})
				return audit, err
			},
		},
		{
			name: "workspace_availability",
			call: func() (ActionAudit, error) {
				_, audit, err := NewMockWorkspaceService(dataset).CheckWorkspaceAvailability(cancelled, WorkspaceAvailabilityRequest{Date: "2026-05-20", WorkspaceType: "hot_seat"})
				return audit, err
			},
		},
		{
			name: "payment",
			call: func() (ActionAudit, error) {
				_, audit, err := NewMockPaymentService(dataset).LookupPayment(cancelled, PaymentLookupRequest{Identifier: "PAY-123456", IdentifierType: "payment_id"})
				return audit, err
			},
		},
		{
			name: "account",
			call: func() (ActionAudit, error) {
				_, audit, err := NewMockAccountService(dataset).LookupAccount(cancelled, AccountLookupRequest{Identifier: "user1@example.com", IdentifierType: "email"})
				return audit, err
			},
		},
		{
			name: "pricing",
			call: func() (ActionAudit, error) {
				_, audit, err := NewMockPricingService(dataset).ListPrices(cancelled, PricingLookupRequest{Catalog: "workspace"})
				return audit, err
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			audit, err := tc.call()
			elapsed := time.Since(start)
			assertProviderErrorCode(t, err, CodeProviderUnavailable)
			if elapsed >= 3*time.Second {
				t.Fatalf("provider timeout elapsed = %s, want < 3s", elapsed)
			}
			if audit.DurationMS >= 3000 {
				t.Fatalf("audit duration_ms = %d, want < 3000", audit.DurationMS)
			}
			if audit.Status != StatusUnavailable || audit.ErrorCode != CodeProviderUnavailable {
				t.Fatalf("audit = %+v, want unavailable/provider_unavailable", audit)
			}
		})
	}
}

func TestMockProvidersDoNotMutateFixtureData(t *testing.T) {
	t.Parallel()

	dataset := mustLoadDataset(t)
	beforeBookings := dataset.Bookings
	beforeWorkspaces := dataset.WorkspaceBookings
	beforePayments := dataset.Payments
	beforeUsers := dataset.Users
	ctx := context.Background()

	_, _, _ = NewMockBookingService(dataset).LookupBooking(ctx, BookingLookupRequest{Identifier: "BRG-482910", IdentifierType: "booking_number"})
	_, _, _ = NewMockWorkspaceService(dataset).LookupWorkspaceBooking(ctx, WorkspaceLookupRequest{Identifier: "WS-1001", IdentifierType: "workspace_booking"})
	_, _, _ = NewMockWorkspaceService(dataset).CheckWorkspaceAvailability(ctx, WorkspaceAvailabilityRequest{Date: "2026-05-20", WorkspaceType: "hot_seat"})
	_, _, _ = NewMockPaymentService(dataset).LookupPayment(ctx, PaymentLookupRequest{Identifier: "PAY-123456", IdentifierType: "payment_id"})
	_, _, _ = NewMockAccountService(dataset).LookupAccount(ctx, AccountLookupRequest{Identifier: "user1@example.com", IdentifierType: "email"})
	_, _, _ = NewMockPricingService(dataset).ListPrices(ctx, PricingLookupRequest{Catalog: "workspace_rules"})

	if !reflect.DeepEqual(beforeBookings, dataset.Bookings) {
		t.Fatal("booking fixtures mutated by provider calls")
	}
	if !reflect.DeepEqual(beforeWorkspaces, dataset.WorkspaceBookings) {
		t.Fatal("workspace fixtures mutated by provider calls")
	}
	if !reflect.DeepEqual(beforePayments, dataset.Payments) {
		t.Fatal("payment fixtures mutated by provider calls")
	}
	if !reflect.DeepEqual(beforeUsers, dataset.Users) {
		t.Fatal("account fixtures mutated by provider calls")
	}
}

func assertProviderErrorCode(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected provider error %q", want)
	}

	providerErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error type = %T, want *provider.Error", err)
	}
	if providerErr.Code != want {
		t.Fatalf("error code = %q, want %q", providerErr.Code, want)
	}
}

func assertAuditFields(t *testing.T, audit ActionAudit, wantProvider, wantStatus, wantCode string) {
	t.Helper()

	if audit.Provider != wantProvider {
		t.Fatalf("audit provider = %q, want %q", audit.Provider, wantProvider)
	}
	if audit.Source != SourceMockExternal {
		t.Fatalf("audit source = %q, want %q", audit.Source, SourceMockExternal)
	}
	if audit.Status != wantStatus {
		t.Fatalf("audit status = %q, want %q", audit.Status, wantStatus)
	}
	if audit.DurationMS < 0 {
		t.Fatalf("audit duration_ms = %d, want >= 0", audit.DurationMS)
	}
	if audit.ErrorCode != wantCode {
		t.Fatalf("audit error_code = %q, want %q", audit.ErrorCode, wantCode)
	}
}

func mustLoadDataset(t *testing.T) *appseed.Dataset {
	t.Helper()

	dataset, err := appseed.Load(serviceRoot(t))
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}

	return dataset
}

func serviceRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("service root abs: %v", err)
	}

	return root
}
