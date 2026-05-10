package operator

import "strings"

func NormalizeReason(reason Reason) Reason {
	trimmed := Reason(strings.TrimSpace(string(reason)))
	if trimmed == "" {
		return ReasonManualRequest
	}
	return trimmed
}

func ValidateReason(reason Reason) error {
	switch NormalizeReason(reason) {
	case ReasonManualRequest, ReasonLowConfidenceRepeated, ReasonComplaint, ReasonBusinessError:
		return nil
	default:
		return ErrInvalidReason
	}
}

func NormalizeStatus(status QueueStatus) QueueStatus {
	return QueueStatus(strings.TrimSpace(string(status)))
}

func ValidateStatus(status QueueStatus) error {
	switch NormalizeStatus(status) {
	case QueueStatusWaiting, QueueStatusAccepted, QueueStatusClosed:
		return nil
	default:
		return ErrInvalidStatus
	}
}
