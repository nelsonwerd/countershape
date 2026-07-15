package world

import (
	"errors"
	"fmt"
)

// RefusalCode is stable machine data for failures that prevent an attempt
// boundary from being constructed. Candidate exits and detected lifecycle
// controls are returned in Result instead of being converted into Refusal.
type RefusalCode string

const (
	CodeInvalidRequest      RefusalCode = "INVALID_WORLD_REQUEST"
	CodeInvalidAllocation   RefusalCode = "INVALID_ATTEMPT_ALLOCATION"
	CodeToolRejected        RefusalCode = "EXECUTION_TOOL_REJECTED"
	CodePlanProfileRejected RefusalCode = "WORLD_PLAN_PROFILE_REJECTED"
	CodeMarkerWriteFailed   RefusalCode = "ATTEMPT_MARKER_WRITE_FAILED"
	CodeUnsupportedPlatform RefusalCode = "UNSUPPORTED_EXECUTION_PLATFORM"
)

type Refusal struct {
	Code   RefusalCode
	Detail string
	Cause  error
}

func (r *Refusal) Error() string {
	if r.Detail == "" {
		return string(r.Code)
	}
	return fmt.Sprintf("%s: %s", r.Code, r.Detail)
}

func (r *Refusal) Unwrap() error { return r.Cause }

func refuse(code RefusalCode, detail string, cause error) error {
	return &Refusal{Code: code, Detail: detail, Cause: cause}
}

func RefusalCodeOf(err error) (RefusalCode, bool) {
	var target *Refusal
	if !errors.As(err, &target) {
		return "", false
	}
	return target.Code, true
}
