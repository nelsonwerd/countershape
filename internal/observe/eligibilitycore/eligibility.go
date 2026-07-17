// Package eligibilitycore owns the pure two-tag observation eligibility
// disposition shared by TrialFact admission and standalone semantic parity.
// It imports only domain values and carries no world, runner, or I/O closure.
package eligibilitycore

import "github.com/nelsonwerd/countershape/internal/domain"

type FactKind string

const (
	BehaviorCaptured  FactKind = "BEHAVIOR_CAPTURED"
	ControlIneligible FactKind = "CONTROL_INELIGIBLE"
)

type Decision struct {
	eligible bool
	reasons  []domain.ControlReason
}

func (d Decision) IsEligible() bool { return d.eligible }
func (d Decision) Reasons() []domain.ControlReason {
	return append([]domain.ControlReason(nil), d.reasons...)
}

// Select validates one realizable post-admission owner fact. One optional
// primary control must come first; only teardown/orphan findings may follow.
// ENVELOPE_REJECTED is pre-admission and therefore cannot enter.
func Select(kind FactKind, reasons []domain.ControlReason) (Decision, error) {
	if kind == BehaviorCaptured {
		if len(reasons) != 0 {
			return Decision{}, &domain.Error{Code: "INVALID_OWNER_FACTS", Detail: "captured behavior cannot carry controls"}
		}
		return Decision{eligible: true}, nil
	}
	if kind != ControlIneligible || len(reasons) == 0 || len(reasons) > 3 {
		return Decision{}, &domain.Error{Code: "INVALID_OWNER_FACTS", Detail: "owner fact kind or control count is invalid"}
	}
	seen := make(map[domain.ControlReason]struct{}, len(reasons))
	validV1 := func(reason domain.ControlReason) bool {
		switch reason {
		case domain.ControlMaterializationError, domain.ControlSetupError, domain.ControlStartError,
			domain.ControlReadinessError, domain.ControlProbeTransportError, domain.ControlTimeout,
			domain.ControlCancelled, domain.ControlOutputLimit, domain.ControlProjectionRejected,
			domain.ControlOrphanRisk, domain.ControlTeardownError, domain.ControlUnsupportedGitMode,
			domain.ControlMissingObject, domain.ControlBudgetExhausted:
			return true
		default:
			return false
		}
	}
	isTeardown := func(reason domain.ControlReason) bool {
		return reason == domain.ControlTeardownError || reason == domain.ControlOrphanRisk
	}
	for index, reason := range reasons {
		if !validV1(reason) {
			return Decision{}, &domain.Error{Code: "INVALID_OWNER_FACTS", Detail: "control is outside post-admission owner facts"}
		}
		if _, duplicate := seen[reason]; duplicate {
			return Decision{}, &domain.Error{Code: "INVALID_OWNER_FACTS", Detail: "owner controls repeat"}
		}
		seen[reason] = struct{}{}
		if index > 0 && !isTeardown(reason) {
			return Decision{}, &domain.Error{Code: "INVALID_OWNER_FACTS", Detail: "only teardown controls may follow the first control"}
		}
	}
	return Decision{reasons: append([]domain.ControlReason(nil), reasons...)}, nil
}
