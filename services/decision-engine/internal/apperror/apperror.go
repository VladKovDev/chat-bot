package apperror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Code string

const (
	CodeInvalidRequest      Code = "invalid_request"
	CodeDatabaseUnavailable Code = "database_unavailable"
	CodeProviderUnavailable Code = "provider_unavailable"
	CodeProcessingFailed    Code = "processing_failed"
	CodeInternal            Code = "internal_error"
)

type Error struct {
	Code      Code
	Operation string
	cause     error
}

func Wrap(code Code, operation string, cause error) error {
	if cause == nil {
		return &Error{Code: code, Operation: operation}
	}
	return &Error{Code: code, Operation: operation, cause: cause}
}

func (e *Error) Error() string {
	if e.Operation == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s:%s", e.Code, e.Operation)
}

func (e *Error) Unwrap() error {
	return e.cause
}

func IsAppError(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr)
}

type PublicError struct {
	Code      Code   `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type Envelope struct {
	Success bool        `json:"success,omitempty"`
	Error   PublicError `json:"error"`
}

func PublicFromError(err error, requestID string) PublicError {
	var appErr *Error
	if errors.As(err, &appErr) {
		return NewPublic(appErr.Code, requestID)
	}
	return NewPublic(CodeProcessingFailed, requestID)
}

func NewPublic(code Code, requestID string) PublicError {
	return PublicError{
		Code:      code,
		Message:   Message(code),
		RequestID: requestID,
	}
}

func Message(code Code) string {
	switch code {
	case CodeInvalidRequest:
		return "Некорректный запрос. Проверьте данные и попробуйте снова."
	case CodeDatabaseUnavailable:
		return "Не удалось сохранить или загрузить данные диалога. Попробуйте позже."
	case CodeProviderUnavailable:
		return "Не удалось проверить данные. Попробуйте позже или подключим оператора."
	case CodeInternal:
		return "Внутренняя ошибка сервиса. Попробуйте позже."
	default:
		return "Не удалось обработать сообщение. Попробуйте позже."
	}
}

func Status(code Code) int {
	switch code {
	case CodeInvalidRequest:
		return http.StatusBadRequest
	case CodeDatabaseUnavailable, CodeProviderUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func WriteJSON(w http.ResponseWriter, status int, publicError PublicError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: false,
		Error:   publicError,
	})
}
