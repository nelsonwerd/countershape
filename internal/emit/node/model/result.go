package model

// DirectReason is one closed standalone-harness ineligibility reason. It is
// intentionally distinct from domain.ControlReason: the generated runtime has
// direct materialization, capture, companion-integrity, and cleanup stages
// that do not fabricate owner-event authority.
type DirectReason string

const (
	DirectOrphanRisk             DirectReason = "ORPHAN_RISK"
	DirectCleanupFailed          DirectReason = "CLEANUP_FAILED"
	DirectTeardownFailed         DirectReason = "TEARDOWN_FAILED"
	DirectOutputLimit            DirectReason = "OUTPUT_LIMIT"
	DirectTimeout                DirectReason = "TIMEOUT"
	DirectTransportFailed        DirectReason = "TRANSPORT_FAILED"
	DirectCaptureFailed          DirectReason = "CAPTURE_FAILED"
	DirectResponseParseFailed    DirectReason = "RESPONSE_PARSE_FAILED"
	DirectProjectionFailed       DirectReason = "PROJECTION_FAILED"
	DirectReadinessFailed        DirectReason = "READINESS_FAILED"
	DirectStartFailed            DirectReason = "START_FAILED"
	DirectEnvironmentInvalid     DirectReason = "ENVIRONMENT_INVALID"
	DirectFixtureOverlayFailed   DirectReason = "FIXTURE_OVERLAY_FAILED"
	DirectSourceCopyFailed       DirectReason = "SOURCE_COPY_FAILED"
	DirectExecutionRootFailed    DirectReason = "EXECUTION_ROOT_FAILED"
	DirectSourceInventoryInvalid DirectReason = "SOURCE_INVENTORY_INVALID"
)

var remainingDirectReasonPrecedence = [...]DirectReason{
	DirectOutputLimit,
	DirectTimeout,
	DirectTransportFailed,
	DirectCaptureFailed,
	DirectResponseParseFailed,
	DirectProjectionFailed,
	DirectReadinessFailed,
	DirectStartFailed,
	DirectEnvironmentInvalid,
	DirectFixtureOverlayFailed,
	DirectSourceCopyFailed,
	DirectExecutionRootFailed,
	DirectSourceInventoryInvalid,
}

// DirectResultFacts is the complete pure input to the standalone result
// selector. It carries no cancellation injection surface, process handle, or
// expected outcome.
type DirectResultFacts struct {
	IneligibleReasons []DirectReason
	InternalFailure   bool
	PredicateMatch    bool
	Tamper            bool
}

// DirectResult is the closed machine outcome emitted by the generated
// contract's TAP diagnostic.
type DirectResult struct {
	Outcome string
	Reason  string
}

// SelectDirectResult applies the standalone safety and semantic precedence.
// Reasons are a set at this boundary; input order cannot influence the result.
func SelectDirectResult(facts DirectResultFacts) (DirectResult, error) {
	valid := make(map[DirectReason]struct{}, len(remainingDirectReasonPrecedence)+3)
	for _, reason := range append([]DirectReason{
		DirectOrphanRisk, DirectCleanupFailed, DirectTeardownFailed,
	}, remainingDirectReasonPrecedence[:]...) {
		valid[reason] = struct{}{}
	}
	seen := make(map[DirectReason]struct{}, len(facts.IneligibleReasons))
	for _, reason := range facts.IneligibleReasons {
		if _, ok := valid[reason]; !ok {
			return DirectResult{}, refuse("INVALID_RESULT_FACTS", "ineligible reason is outside the closed roster", nil)
		}
		if _, duplicate := seen[reason]; duplicate {
			return DirectResult{}, refuse("INVALID_RESULT_FACTS", "ineligible reasons repeat", nil)
		}
		seen[reason] = struct{}{}
	}
	for _, reason := range []DirectReason{DirectOrphanRisk, DirectCleanupFailed, DirectTeardownFailed} {
		if _, ok := seen[reason]; ok {
			return DirectResult{Outcome: "INELIGIBLE_EXECUTION", Reason: string(reason)}, nil
		}
	}
	if facts.Tamper {
		return DirectResult{Outcome: "TAMPER_DETECTED", Reason: "COMPANION_INTEGRITY_MISMATCH"}, nil
	}
	if facts.InternalFailure {
		return DirectResult{Outcome: "HARNESS_FAILURE", Reason: "INTERNAL_INVARIANT_FAILED"}, nil
	}
	for _, reason := range remainingDirectReasonPrecedence {
		if _, ok := seen[reason]; ok {
			return DirectResult{Outcome: "INELIGIBLE_EXECUTION", Reason: string(reason)}, nil
		}
	}
	if len(seen) != 0 {
		return DirectResult{}, refuse("INVALID_RESULT_FACTS", "ineligible reason set escaped precedence", nil)
	}
	if facts.PredicateMatch {
		return DirectResult{Outcome: "CONFORMS", Reason: "NONE"}, nil
	}
	return DirectResult{Outcome: "CONTRADICTS", Reason: "PREDICATE_MISMATCH"}, nil
}
