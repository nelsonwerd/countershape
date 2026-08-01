// Package http owns the capability-only standalone HTTP subject runner.
package http

import (
	"context"
	"errors"
	"fmt"

	"github.com/nelsonwerd/countershape/internal/contractexec"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	CodeInvalidRequest        = "INVALID_HTTP_CONTRACT_RUN_REQUEST"
	CodeTargetChanged         = "HTTP_CONTRACT_RUN_TARGET_CHANGED"
	CodeUnsupportedProfile    = "UNSUPPORTED_HTTP_CONTRACT_RUN_PROFILE"
	CodeFixtureRejected       = "HTTP_CONTRACT_RUN_FIXTURE_REJECTED"
	CodeAdmissionRefused      = "HTTP_CONTRACT_RUN_ADMISSION_REFUSED"
	CodeSpawnClosureFailed    = "HTTP_CONTRACT_RUN_SPAWN_CLOSURE_FAILED"
	CodeEvidenceClosureFailed = "HTTP_CONTRACT_RUN_EVIDENCE_CLOSURE_FAILED"
	CodeRecoveryRefused       = "HTTP_CONTRACT_RUN_RECOVERY_REFUSED"
)

type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (failure *Error) Error() string {
	if failure == nil {
		return "HTTP contract runner failed"
	}
	if failure.Detail == "" {
		return failure.Code
	}
	return fmt.Sprintf("%s: %s", failure.Code, failure.Detail)
}

func (failure *Error) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.Cause
}

func IsCode(err error, code string) bool {
	var failure *Error
	return errors.As(err, &failure) && failure.Code == code
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

// Execute accepts only a live official target. The emitted source owns the
// launch, environment, request, capture, projection, and expected predicate.
func Execute(
	ctx context.Context,
	retained contractexec.OfficialTarget,
) (store.ContractExecutionRecord, error) {
	return executeHTTP(ctx, retained)
}

// ResumeClassification converges only a durable terminal classification. It
// has no preparation, admission, or subject-start path.
func ResumeClassification(
	ctx context.Context,
	retained contractexec.OfficialTarget,
) (store.ContractExecutionRecord, error) {
	return resumeHTTPClassification(ctx, retained)
}
