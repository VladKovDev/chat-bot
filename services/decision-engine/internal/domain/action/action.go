package action

import (
	"context"

	"github.com/VladKovDev/chat-bot/internal/domain/session"
)

// Action represents a business operation that can be executed during a transition
type Action interface {
	Execute(ctx context.Context, data ActionData) error
}

// ActionData contains information needed to execute an action
type ActionData struct {
	Session   *session.Session
	UserText  string
	Context   map[string]interface{}
}

// Action keys - string identifiers for actions that LLM can return
const (
	// System actions
	ActionEscalateOperator  string = "escalate_operator"  // Эскалация на оператора
	ActionResetConversation string = "reset_conversation" // Сброс диалога
	ActionLogAnalytics      string = "log_analytics"      // Логирование аналитики

	// Booking & Reservation actions
	ActionCreateBooking         string = "create_booking"          // Создать запись
	ActionCancelBooking         string = "cancel_booking"          // Отменить запись
	ActionRescheduleBooking     string = "reschedule_booking"      // Перенести запись
	ActionSearchBooking         string = "search_booking"          // Найти запись
	ActionConfirmBooking        string = "confirm_booking"         // Подтвердить запись
	ActionViewBookingDetails    string = "view_booking_details"    // Просмотр деталей записи
	ActionReportBookingProblem  string = "report_booking_problem"  // Сообщить о проблеме с записью

	// Workspace actions
	ActionBookWorkspace      string = "book_workspace"       // Забронировать место
	ActionCancelWorkspace    string = "cancel_workspace"     // Отменить бронь
	ActionRescheduleWorkspace string = "reschedule_workspace" // Перенести бронь
	ActionSearchWorkspace    string = "search_workspace"     // Найти бронь
	ActionViewWorkspaceDetails string = "view_workspace_details" // Просмотр деталей брони
	ActionReportWorkspaceIssue string = "report_workspace_issue" // Сообщить о проблеме с местом

	// Payment actions
	ActionProcessPayment     string = "process_payment"      // Обработать оплату
	ActionRetryPayment       string = "retry_payment"        // Повторить оплату
	ActionRequestRefund      string = "request_refund"       // Запросить возврат
	ActionProcessRefund      string = "process_refund"       // Обработать возврат
	ActionViewPaymentHistory string = "view_payment_history" // Просмотр истории платежей
	ActionViewPaymentDetails string = "view_payment_details" // Просмотр деталей платежа

	// Account actions
	ActionLogin              string = "login"               // Вход в аккаунт
	ActionResetPassword      string = "reset_password"       // Сброс пароля
	ActionUpdateAccountData  string = "update_account_data"  // Обновить данные аккаунта
	ActionDeleteAccount      string = "delete_account"       // Удалить аккаунт
	ActionLinkContact        string = "link_contact"         // Привязать контакт
	ActionUnlinkContact      string = "unlink_contact"       // Отвязать контакт

	// Support & Issue actions
	ActionCreateTicket       string = "create_ticket"        // Создать тикет
	ActionContactSupport     string = "contact_support"      // Связаться с поддержкой
	ActionReportIssue        string = "report_issue"         // Сообщить о проблеме
	ActionReportComplaint    string = "report_complaint"     // Подать жалобу

	// Notification actions
	ActionSendConfirmation   string = "send_confirmation"   // Отправить подтверждение
	ActionSendReminder       string = "send_reminder"        // Отправить напоминание
	ActionSendNotification   string = "send_notification"   // Отправить уведомление

	// Context actions
	ActionSaveContext        string = "save_context"         // Сохранить контекст
	ActionClearContext       string = "clear_context"        // Очистить контекст
	ActionUpdateContext      string = "update_context"       // Обновить контекст
)

// All returns all available action keys as a slice of strings
func All() []string {
	return []string{
		// System
		ActionEscalateOperator,
		ActionResetConversation,
		ActionLogAnalytics,
		// Booking
		ActionCreateBooking,
		ActionCancelBooking,
		ActionRescheduleBooking,
		ActionSearchBooking,
		ActionConfirmBooking,
		ActionViewBookingDetails,
		ActionReportBookingProblem,
		// Workspace
		ActionBookWorkspace,
		ActionCancelWorkspace,
		ActionRescheduleWorkspace,
		ActionSearchWorkspace,
		ActionViewWorkspaceDetails,
		ActionReportWorkspaceIssue,
		// Payment
		ActionProcessPayment,
		ActionRetryPayment,
		ActionRequestRefund,
		ActionProcessRefund,
		ActionViewPaymentHistory,
		ActionViewPaymentDetails,
		// Account
		ActionLogin,
		ActionResetPassword,
		ActionUpdateAccountData,
		ActionDeleteAccount,
		ActionLinkContact,
		ActionUnlinkContact,
		// Support
		ActionCreateTicket,
		ActionContactSupport,
		ActionReportIssue,
		ActionReportComplaint,
		// Notification
		ActionSendConfirmation,
		ActionSendReminder,
		ActionSendNotification,
		// Context
		ActionSaveContext,
		ActionClearContext,
		ActionUpdateContext,
	}
}