package domain

import "fmt"

// Error is a construction refusal. Code is stable machine data; Detail is
// deliberately explanatory and is not part of semantic identity.
type Error struct {
	Code   string
	Detail string
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func refuse(code, detail string) error {
	return &Error{Code: code, Detail: detail}
}

const (
	ErrInvalidDigest             = "INVALID_DIGEST"
	ErrInvalidCandidateKey       = "INVALID_CANDIDATE_KEY"
	ErrEmptyIdentityField        = "EMPTY_IDENTITY_FIELD"
	ErrDuplicateCandidateKey     = "DUPLICATE_CANDIDATE_KEY"
	ErrInvalidWorldPlan          = "INVALID_WORLD_PLAN"
	ErrAmbientInterpolation      = "AMBIENT_INTERPOLATION_FORBIDDEN"
	ErrShellString               = "SHELL_STRING_FORBIDDEN"
	ErrSecretValueInPlan         = "SECRET_VALUE_IN_PLAN"
	ErrDuplicateEnvironmentName  = "DUPLICATE_ENVIRONMENT_NAME"
	ErrInvalidAttemptTransition  = "INVALID_ATTEMPT_TRANSITION"
	ErrInvalidArtifactTransition = "INVALID_ARTIFACT_TRANSITION"
	ErrTerminalArtifact          = "TERMINAL_ARTIFACT"
	ErrReceiptAuthority          = "INVALID_RECEIPT_AUTHORITY"
)
