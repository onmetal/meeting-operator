package errors

import (
	"errors"
	"fmt"
)

type ErrorStatus struct {
	Reason
}

type Status interface {
	Status() Reason
}

type Reason struct {
	Message      string
	StatusReason StatusReason
}

type StatusReason string

const (
	StatusReasonAlreadyExist  StatusReason = "already exist"
	StatusReasonNotExist      StatusReason = "resource not exist"
	StatusReasonUnderDeletion StatusReason = "under deletion"
	StatusReasonUnknown       StatusReason = "unknown"
)

func (e *ErrorStatus) Error() string { return e.Reason.Message }

func (e *ErrorStatus) Status() Reason { return e.Reason }

func IsNotExist(err error) bool {
	return ReasonForError(err) == StatusReasonNotExist
}

func IsAlreadyExists(err error) bool {
	return ReasonForError(err) == StatusReasonAlreadyExist
}

func IsUnderDeletion(err error) bool {
	return ReasonForError(err) == StatusReasonUnderDeletion
}

func ReasonForError(err error) StatusReason {
	if reason := Status(nil); errors.As(err, &reason) {
		return reason.Status().StatusReason
	}
	return StatusReasonUnknown
}

func AlreadyExist(s string) *ErrorStatus {
	return &ErrorStatus{
		Reason: Reason{
			Message:      fmt.Sprintf("resource with name: %s, already exist ", s),
			StatusReason: StatusReasonAlreadyExist,
		},
	}
}

func NotExist(s string) *ErrorStatus {
	return &ErrorStatus{
		Reason: Reason{
			Message:      fmt.Sprintf("resource with name: %s, not exist", s),
			StatusReason: StatusReasonNotExist,
		},
	}
}

func UnderDeletion() *ErrorStatus {
	return &ErrorStatus{
		Reason: Reason{
			Message:      "component is under deletion",
			StatusReason: StatusReasonUnderDeletion,
		},
	}
}
