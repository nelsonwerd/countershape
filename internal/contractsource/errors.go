// Package contractsource owns the byte-complete, adapter-reconstructed source
// witness consumed by the standalone contract emitter. It is pure: no store,
// filesystem, process, Git, clock, environment, or network authority lives
// here.
package contractsource

import "errors"

type Code string

const (
	CodePortableSourceRequired  Code = "PORTABLE_SOURCE_REQUIRED"
	CodeNonportableStartProfile Code = "NONPORTABLE_START_PROFILE"
	CodeSourceAuthorityMismatch Code = "PORTABLE_SOURCE_AUTHORITY_MISMATCH"
	CodeSourceWireInvalid       Code = "PORTABLE_SOURCE_WIRE_INVALID"
	CodeSourceLimitExceeded     Code = "PORTABLE_SOURCE_LIMIT_EXCEEDED"
)

type Refusal struct {
	Code   Code
	Detail string
	Cause  error
}

func (r *Refusal) Error() string {
	if r == nil {
		return "<nil>"
	}
	if r.Detail == "" {
		return string(r.Code)
	}
	return string(r.Code) + ": " + r.Detail
}

func (r *Refusal) Unwrap() error { return r.Cause }

func IsCode(err error, code Code) bool {
	actual, ok := CodeOf(err)
	return ok && actual == code
}

func CodeOf(err error) (Code, bool) {
	var target *Refusal
	if !errors.As(err, &target) || target == nil {
		return "", false
	}
	return target.Code, true
}

func refuse(code Code, detail string, cause error) error {
	return &Refusal{Code: code, Detail: detail, Cause: cause}
}
