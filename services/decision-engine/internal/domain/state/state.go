package state

type State string

// General states
const (
	StateNew                  State = "new"
	StateWaitingForCategory   State = "waiting_for_category"
	StateWaitingClarification State = "waiting_clarification"
	StateSolutionOffered      State = "solution_offered"
	StateEscalatedToOperator  State = "escalated_to_operator"
	StateClosed               State = "closed"
)

// Booking & Reservation states
const (
	StateBookingTypeSelection           State = "booking_type_selection"
	StateBookingClientMenu              State = "booking_client_menu"
	StateBookingClientHasNumber         State = "booking_client_has_number"
	StateBookingClientNumberInput       State = "booking_client_number_input"
	StateBookingClientNumberFound       State = "booking_client_number_found"
	StateBookingClientReschedule        State = "booking_client_reschedule"
	StateBookingClientRescheduleDate    State = "booking_client_reschedule_date"
	StateBookingClientRescheduleConfirm State = "booking_client_reschedule_confirm"
	StateBookingClientCancel            State = "booking_client_cancel"
	StateBookingClientCancelConfirm     State = "booking_client_cancel_confirm"
	StateBookingClientDetails           State = "booking_client_details"
	StateBookingClientIssue             State = "booking_client_issue"
	StateBookingClientNotFound          State = "booking_client_not_found"
	StateBookingClientSearchContact     State = "booking_client_search_contact"
	StateBookingClientNewService        State = "booking_client_new_service"
	StateBookingClientNewMaster         State = "booking_client_new_master"
	StateBookingClientNewDateTime       State = "booking_client_new_datetime"
	StateBookingClientNewConfirm        State = "booking_client_new_confirm"
	StateBookingMasterMenu              State = "booking_master_menu"
	StateBookingMasterSchedule          State = "booking_master_schedule"
	StateBookingMasterRecordActions     State = "booking_master_record_actions"
	StateBookingMasterConfirm           State = "booking_master_confirm"
	StateBookingMasterCancel            State = "booking_master_cancel"
	StateBookingMasterReschedule        State = "booking_master_reschedule"
	StateBookingMasterIssue             State = "booking_master_issue"
)

// Workspace rental states
const (
	StateWorkspaceInfo              State = "workspace_info"
	StateWorkspaceBooking           State = "workspace_booking"
	StateWorkspaceTypeSelection     State = "workspace_type_selection"
	StateWorkspaceDateTimeSelection State = "workspace_datetime_selection"
	StateWorkspaceAvailabilityCheck State = "workspace_availability_check"
	StateWorkspaceBookingConfirm    State = "workspace_booking_confirm"
	StateWorkspaceManageHasNumber   State = "workspace_manage_has_number"
	StateWorkspaceManageNumberInput State = "workspace_manage_number_input"
	StateWorkspaceManageFound       State = "workspace_manage_found"
	StateWorkspaceManageCancel      State = "workspace_manage_cancel"
	StateWorkspaceManageReschedule  State = "workspace_manage_reschedule"
	StateWorkspaceManageDetails     State = "workspace_manage_details"
	StateWorkspaceManageNotFound    State = "workspace_manage_not_found"
	StateWorkspaceIssue             State = "workspace_issue"
	StateWorkspaceIssueType         State = "workspace_issue_type"
)

// Payment & Finance states
const (
	StatePaymentIssue              State = "payment_issue"
	StatePaymentIssueType          State = "payment_issue_type"
	StatePaymentIdInput            State = "payment_id_input"
	StatePaymentFound              State = "payment_found"
	StatePaymentNotFound           State = "payment_not_found"
	StatePaymentRetry              State = "payment_retry"
	StatePaymentRefund             State = "payment_refund"
	StatePaymentRefundIdInput      State = "payment_refund_id_input"
	StatePaymentRefundCheck        State = "payment_refund_check"
	StatePaymentRefundAvailable    State = "payment_refund_available"
	StatePaymentRefundNotAvailable State = "payment_refund_not_available"
	StatePaymentHistory            State = "payment_history"
	StatePaymentHistoryDetails     State = "payment_history_details"
)

// Site/App Technical issues states
const (
	StateTechIssueCategory        State = "tech_issue_category"
	StateTechIssueDetails         State = "tech_issue_details"
	StateTechIssueDevice          State = "tech_issue_device"
	StateTechIssueBrowser         State = "tech_issue_browser"
	StateTechIssueStep            State = "tech_issue_step"
	StateTechIssueSolution        State = "tech_issue_solution"
	StateTechIssueEscalate        State = "tech_issue_escalate"
	StateTechIssueCollectInfo     State = "tech_issue_collect_info"
	StateTechIssueBasicSolution   State = "tech_issue_basic_solution"
	StateTechIssuePersist         State = "tech_issue_persist"
	StateTechTicketCreated        State = "tech_ticket_created"
	StateTechLoginProblem         State = "tech_login_problem"
	StateTechBookingError         State = "tech_booking_error"
	StateTechSiteNotLoading       State = "tech_site_not_loading"
)

// Account & Access states
const (
	StateAccountLogin             State = "account_login"
	StateAccountForgotPassword    State = "account_forgot_password"
	StateAccountResetCode         State = "account_reset_code"
	StateAccountResetNotReceived  State = "account_reset_not_received"
	StateAccountResetSuccess      State = "account_reset_success"
	StateAccountManagement        State = "account_management"
	StateAccountChangeData        State = "account_change_data"
	StateAccountUpdateData        State = "account_update_data"
	StateAccountUpdateSuccess     State = "account_update_success"
	StateAccountManageContacts    State = "account_manage_contacts"
	StateAccountDelete            State = "account_delete"
	StateAccountDeleteConfirm     State = "account_delete_confirm"
	StateAccountDeleteSuccess     State = "account_delete_success"
	StateAccountLinkContact       State = "account_link_contact"
	StateAccountRoleMaster        State = "account_role_master"
	StateAccountRoleClient        State = "account_role_client"
)

// Services & Rules states
const (
	StateServicesInfo         State = "services_info"
	StateServicesPrices       State = "services_prices"
	StateServicesMasters      State = "services_masters"
	StateServicesLocation     State = "services_location"
	StateServicesRules        State = "services_rules"
	StateServicesCancellation State = "services_cancellation"
	StateServicesRefund       State = "services_refund"
	StateServicesCoworking    State = "services_coworking"
	StateServicesFAQ          State = "services_faq"
	StateServicesFAQInput     State = "services_faq_input"
	StateServicesFAQNotFound  State = "services_faq_not_found"
)

// Complaints & Incidents states
const (
	StateComplaintType         State = "complaint_type"
	StateComplaintDetails      State = "complaint_details"
	StateComplaintDateTime     State = "complaint_date_time"
	StateComplaintRelatedOrder State = "complaint_related_order"
	StateComplaintCompensation State = "complaint_compensation"
	StateComplaintEscalate     State = "complaint_escalate"
)

// Other / Free input states
const (
	StateOtherInput         State = "other_input"
	StateOtherClassify       State = "other_classify"
	StateOtherClassified     State = "other_classified"
	StateOtherNotClassified  State = "other_not_classified"
	StateOtherClarify        State = "other_clarify"
	StateOtherCategories     State = "other_categories"
)

// Complex scenarios states
const (
	StateComplexPaymentBooking        State = "complex_payment_booking"
	StateComplexPaymentRestore        State = "complex_payment_restore"
	StateComplexMasterConflict        State = "complex_master_conflict"
	StateComplexWorkspaceCompensation State = "complex_workspace_compensation"
)

// Escalation states
const (
	StateEscalationOperator         State = "escalation_operator"
	StateEscalationOperatorRequest  State = "escalation_operator_request"
	StateEscalationContextSent      State = "escalation_context_sent"
)

// Error handling states
const (
	StateErrorDataMissing       State = "error_data_missing"
	StateErrorUserNotFound      State = "error_user_not_found"
	StateErrorBookingNotFound   State = "error_booking_not_found"
	StateErrorRepeated          State = "error_repeated"
	StateErrorGeneric           State = "error_generic"
	StateErrorNetwork           State = "error_network"
	StateErrorServer            State = "error_server"
	StateErrorValidation        State = "error_validation"
	StateErrorUnauthorized      State = "error_unauthorized"
)

// Support & Contact states
const (
	StateSupportContact   State = "support_contact"
	StateSupportChat      State = "support_chat"
	StateSupportPhone     State = "support_phone"
	StateSupportEmail     State = "support_email"
)

// Search states
const (
	StateSearchByContact          State = "search_by_contact"
	StateSearchByContactFound     State = "search_by_contact_found"
	StateSearchByContactNotFound  State = "search_by_contact_not_found"
)

// My bookings/workspace bookings
const (
	StateMyBookings           State = "my_bookings"
	StateMyBookingsEmpty      State = "my_bookings_empty"
	StateMyWorkspaceBookings  State = "my_workspace_bookings"
)

// Rating & Feedback
const (
	StateRateHelp   State = "rate_help"
	StateRateThanks State = "rate_thanks"
)

// Context menu
const (
	StateContextMenu State = "context_menu"
)

// Farewell
const (
	StateGoodbye State = "goodbye"
)

// All returns all available states as a slice of strings
func All() []string {
	return []string{
		// General
		string(StateNew),
		string(StateWaitingForCategory),
		string(StateWaitingClarification),
		string(StateSolutionOffered),
		string(StateEscalatedToOperator),
		string(StateClosed),
		// Booking & Reservation
		string(StateBookingTypeSelection),
		string(StateBookingClientMenu),
		string(StateBookingClientHasNumber),
		string(StateBookingClientNumberInput),
		string(StateBookingClientNumberFound),
		string(StateBookingClientReschedule),
		string(StateBookingClientRescheduleDate),
		string(StateBookingClientRescheduleConfirm),
		string(StateBookingClientCancel),
		string(StateBookingClientCancelConfirm),
		string(StateBookingClientDetails),
		string(StateBookingClientIssue),
		string(StateBookingClientNotFound),
		string(StateBookingClientSearchContact),
		string(StateBookingClientNewService),
		string(StateBookingClientNewMaster),
		string(StateBookingClientNewDateTime),
		string(StateBookingClientNewConfirm),
		string(StateBookingMasterMenu),
		string(StateBookingMasterSchedule),
		string(StateBookingMasterRecordActions),
		string(StateBookingMasterConfirm),
		string(StateBookingMasterCancel),
		string(StateBookingMasterReschedule),
		string(StateBookingMasterIssue),
		// Workspace rental
		string(StateWorkspaceInfo),
		string(StateWorkspaceBooking),
		string(StateWorkspaceTypeSelection),
		string(StateWorkspaceDateTimeSelection),
		string(StateWorkspaceAvailabilityCheck),
		string(StateWorkspaceBookingConfirm),
		string(StateWorkspaceManageHasNumber),
		string(StateWorkspaceManageNumberInput),
		string(StateWorkspaceManageFound),
		string(StateWorkspaceManageCancel),
		string(StateWorkspaceManageReschedule),
		string(StateWorkspaceManageDetails),
		string(StateWorkspaceManageNotFound),
		string(StateWorkspaceIssue),
		string(StateWorkspaceIssueType),
		// Payment & Finance
		string(StatePaymentIssue),
		string(StatePaymentIssueType),
		string(StatePaymentIdInput),
		string(StatePaymentFound),
		string(StatePaymentNotFound),
		string(StatePaymentRetry),
		string(StatePaymentRefund),
		string(StatePaymentRefundIdInput),
		string(StatePaymentRefundCheck),
		string(StatePaymentRefundAvailable),
		string(StatePaymentRefundNotAvailable),
		string(StatePaymentHistory),
		string(StatePaymentHistoryDetails),
		// Site/App Technical issues
		string(StateTechIssueCategory),
		string(StateTechIssueDetails),
		string(StateTechIssueDevice),
		string(StateTechIssueBrowser),
		string(StateTechIssueStep),
		string(StateTechIssueSolution),
		string(StateTechIssueEscalate),
		string(StateTechIssueCollectInfo),
		string(StateTechIssueBasicSolution),
		string(StateTechIssuePersist),
		string(StateTechTicketCreated),
		string(StateTechLoginProblem),
		string(StateTechBookingError),
		string(StateTechSiteNotLoading),
		// Account & Access
		string(StateAccountLogin),
		string(StateAccountForgotPassword),
		string(StateAccountResetCode),
		string(StateAccountResetNotReceived),
		string(StateAccountResetSuccess),
		string(StateAccountManagement),
		string(StateAccountChangeData),
		string(StateAccountUpdateData),
		string(StateAccountUpdateSuccess),
		string(StateAccountManageContacts),
		string(StateAccountDelete),
		string(StateAccountDeleteConfirm),
		string(StateAccountDeleteSuccess),
		string(StateAccountLinkContact),
		string(StateAccountRoleMaster),
		string(StateAccountRoleClient),
		// Services & Rules
		string(StateServicesInfo),
		string(StateServicesPrices),
		string(StateServicesMasters),
		string(StateServicesLocation),
		string(StateServicesRules),
		string(StateServicesCancellation),
		string(StateServicesRefund),
		string(StateServicesCoworking),
		string(StateServicesFAQ),
		string(StateServicesFAQInput),
		string(StateServicesFAQNotFound),
		// Complaints & Incidents
		string(StateComplaintType),
		string(StateComplaintDetails),
		string(StateComplaintDateTime),
		string(StateComplaintRelatedOrder),
		string(StateComplaintCompensation),
		string(StateComplaintEscalate),
		// Other / Free input
		string(StateOtherInput),
		string(StateOtherClassify),
		string(StateOtherClassified),
		string(StateOtherNotClassified),
		string(StateOtherClarify),
		string(StateOtherCategories),
		// Complex scenarios
		string(StateComplexPaymentBooking),
		string(StateComplexPaymentRestore),
		string(StateComplexMasterConflict),
		string(StateComplexWorkspaceCompensation),
		// Escalation
		string(StateEscalationOperator),
		string(StateEscalationOperatorRequest),
		string(StateEscalationContextSent),
		// Error handling
		string(StateErrorDataMissing),
		string(StateErrorUserNotFound),
		string(StateErrorBookingNotFound),
		string(StateErrorRepeated),
		string(StateErrorGeneric),
		string(StateErrorNetwork),
		string(StateErrorServer),
		string(StateErrorValidation),
		string(StateErrorUnauthorized),
		// Support & Contact
		string(StateSupportContact),
		string(StateSupportChat),
		string(StateSupportPhone),
		string(StateSupportEmail),
		// Search
		string(StateSearchByContact),
		string(StateSearchByContactFound),
		string(StateSearchByContactNotFound),
		// My bookings/workspace bookings
		string(StateMyBookings),
		string(StateMyBookingsEmpty),
		string(StateMyWorkspaceBookings),
		// Rating & Feedback
		string(StateRateHelp),
		string(StateRateThanks),
		// Context menu
		string(StateContextMenu),
		// Farewell
		string(StateGoodbye),
	}
}
