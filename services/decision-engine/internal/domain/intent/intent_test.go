package intent

import "testing"

func TestAllIncludesBRDSection13Intents(t *testing.T) {
	t.Parallel()

	all := make(map[string]struct{}, len(All()))
	for _, item := range All() {
		all[item] = struct{}{}
	}

	required := []string{
		"greeting",
		"goodbye",
		"return_to_menu",
		"reset_conversation",
		"request_operator",
		"ask_booking_info",
		"ask_booking_status",
		"ask_cancellation_rules",
		"ask_reschedule_rules",
		"booking_not_found",
		"ask_workspace_info",
		"ask_workspace_prices",
		"ask_workspace_rules",
		"ask_workspace_status",
		"workspace_unavailable",
		"ask_payment_status",
		"payment_not_passed",
		"payment_not_activated",
		"ask_refund_rules",
		"ask_site_problem",
		"login_not_working",
		"code_not_received",
		"ask_account_help",
		"ask_account_status",
		"ask_services_info",
		"ask_prices",
		"ask_rules",
		"ask_location",
		"ask_faq",
		"report_complaint",
		"complaint_master",
		"complaint_premises",
		"general_question",
		"unknown",
	}

	for _, key := range required {
		if _, ok := all[key]; !ok {
			t.Fatalf("intent.All() is missing %q", key)
		}
	}
}
