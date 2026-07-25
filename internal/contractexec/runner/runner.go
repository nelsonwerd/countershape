// Package runner owns Countershape's capability-only physical contract runner.
// It is the sole production edge that combines an OfficialTarget, store-private
// admission, and neutral process mechanics.
package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/nelsonwerd/countershape/internal/contractexec"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	CodeInvalidRequest        = "INVALID_CONTRACT_RUN_REQUEST"
	CodeTargetChanged         = "CONTRACT_RUN_TARGET_CHANGED"
	CodeUnsupportedProfile    = "UNSUPPORTED_CONTRACT_RUN_PROFILE"
	CodeFixtureRejected       = "CONTRACT_RUN_FIXTURE_REJECTED"
	CodeAdmissionRefused      = "CONTRACT_RUN_ADMISSION_REFUSED"
	CodeSpawnClosureFailed    = "CONTRACT_RUN_SPAWN_CLOSURE_FAILED"
	CodeEvidenceClosureFailed = "CONTRACT_RUN_EVIDENCE_CLOSURE_FAILED"
	CodeRecoveryRefused       = "CONTRACT_RUN_RECOVERY_REFUSED"
)

// Error is a stable runner-layer refusal. Causes retain the lower-layer detail
// without turning strings or copied digests into authority.
type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (failure *Error) Error() string {
	if failure == nil {
		return "contract runner failed"
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

// ExecuteCLI consumes one fresh OfficialTarget through the closed CLI profile.
// It accepts no argv, environment, result, tuple, scope verdict, or store
// locator from the caller.
func ExecuteCLI(
	ctx context.Context,
	retained contractexec.OfficialTarget,
) (store.ContractExecutionRecord, error) {
	return executeCLI(ctx, retained)
}

// ResumeCLIClassification resumes only a durable terminal run. It contains no
// process preparation, admission, or start path and therefore cannot retry a
// subject execution.
func ResumeCLIClassification(
	ctx context.Context,
	retained contractexec.OfficialTarget,
) (store.ContractExecutionRecord, error) {
	return resumeCLIClassification(ctx, retained)
}
