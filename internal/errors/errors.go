/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errors

import (
	"errors"
	"fmt"
)

type StatusError struct {
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
	StatusReasonNotRequired   StatusReason = "not required"
	StatusReasonUnknown       StatusReason = "unknown"
)

func (e *StatusError) Error() string { return e.Reason.Message }

func (e *StatusError) Status() Reason { return e.Reason }

func IsNotExist(err error) bool {
	return ReasonForError(err) == StatusReasonNotExist
}

func IsAlreadyExists(err error) bool {
	return ReasonForError(err) == StatusReasonAlreadyExist
}

func IsUnderDeletion(err error) bool {
	return ReasonForError(err) == StatusReasonUnderDeletion
}

func IsNotRequired(err error) bool {
	return ReasonForError(err) == StatusReasonNotRequired
}

func ReasonForError(err error) StatusReason {
	if reason := Status(nil); errors.As(err, &reason) {
		return reason.Status().StatusReason
	}
	return StatusReasonUnknown
}

func AlreadyExist(s string) *StatusError {
	return &StatusError{
		Reason: Reason{
			Message:      fmt.Sprintf("resource with name: %s, already exist ", s),
			StatusReason: StatusReasonAlreadyExist,
		},
	}
}

func NotExist(s string) *StatusError {
	return &StatusError{
		Reason: Reason{
			Message:      fmt.Sprintf("resource with name: %s, not exist", s),
			StatusReason: StatusReasonNotExist,
		},
	}
}

func UnderDeletion() *StatusError {
	return &StatusError{
		Reason: Reason{
			Message:      "component is under deletion",
			StatusReason: StatusReasonUnderDeletion,
		},
	}
}

func NotRequired() *StatusError {
	return &StatusError{
		Reason: Reason{
			Message:      "not required",
			StatusReason: StatusReasonNotRequired,
		},
	}
}
