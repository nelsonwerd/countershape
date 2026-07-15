package gitobj

import (
	"errors"
	"fmt"
)

// RefusalCode is a stable machine-readable reason that no executable source or
// materialized tree was produced. Details are diagnostic only and never enter
// candidate identity.
type RefusalCode string

const (
	CodeInvalidConfig        RefusalCode = "INVALID_GIT_SOURCE_CONFIG"
	CodeUnsupportedPlatform  RefusalCode = "UNSUPPORTED_PLATFORM"
	CodeGitToolRejected      RefusalCode = "GIT_TOOL_REJECTED"
	CodeGitCommandFailed     RefusalCode = "GIT_COMMAND_FAILED"
	CodeRepositoryRejected   RefusalCode = "REPOSITORY_REJECTED"
	CodeAlternatesRejected   RefusalCode = "GIT_ALTERNATES_REJECTED"
	CodeUnsupportedFormat    RefusalCode = "UNSUPPORTED_OBJECT_FORMAT"
	CodeInvalidRef           RefusalCode = "INVALID_GIT_REF"
	CodeMissingObject        RefusalCode = "MISSING_OBJECT"
	CodeMalformedTree        RefusalCode = "MALFORMED_GIT_TREE"
	CodeUnsupportedMode      RefusalCode = "UNSUPPORTED_GIT_MODE"
	CodeUnsafePath           RefusalCode = "UNSAFE_GIT_PATH"
	CodePathCollision        RefusalCode = "TARGET_FILESYSTEM_PATH_COLLISION"
	CodeLFSPointer           RefusalCode = "LFS_POINTER_REJECTED"
	CodeBudgetExceeded       RefusalCode = "MATERIALIZATION_BUDGET_EXCEEDED"
	CodeObjectHashMismatch   RefusalCode = "GIT_OBJECT_HASH_MISMATCH"
	CodeSourceChanged        RefusalCode = "PINNED_SOURCE_CHANGED"
	CodeCandidateSetRejected RefusalCode = "CANDIDATE_SET_REJECTED"
	CodePlanBindingMismatch  RefusalCode = "WORLD_PLAN_BINDING_MISMATCH"
	CodePublicationFailed    RefusalCode = "MATERIALIZATION_PUBLICATION_FAILED"
	CodePublicationAmbiguous RefusalCode = "MATERIALIZATION_PUBLICATION_AMBIGUOUS"
)

// Refusal represents a fail-closed physical-boundary decision.
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
	var refusal *Refusal
	if errors.As(err, &refusal) {
		return refusal.Code, true
	}
	return "", false
}
