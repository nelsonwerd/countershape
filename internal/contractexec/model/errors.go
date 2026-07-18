package model

import (
	"errors"
	"fmt"
)

const (
	CodeInvalidTarget       = "INVALID_CONTRACT_EXECUTION_TARGET"
	CodeTargetDigest        = "CONTRACT_EXECUTION_TARGET_DIGEST_MISMATCH"
	CodeInvalidWitness      = "INVALID_CLOSED_RUN_WITNESS"
	CodeInvalidRun          = "INVALID_FINALIZED_CONTRACT_RUN"
	CodeRunDigest           = "FINALIZED_CONTRACT_RUN_DIGEST_MISMATCH"
	CodeTargetRunMismatch   = "TARGET_RUN_MISMATCH"
	CodeInvalidExecution    = "INVALID_CONTRACT_EXECUTION"
	CodeExecutionDigest     = "CONTRACT_EXECUTION_DIGEST_MISMATCH"
	CodeClassificationInput = "INVALID_CLASSIFICATION_INPUT"
	CodeLimitExceeded       = "CONTRACT_EXECUTION_LIMIT_EXCEEDED"
)

// Error is a stable construction or parsing refusal. Detail and Cause are
// diagnostics only and never participate in canonical identity.
type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.Cause }

func IsCode(err error, code string) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}
