package parity

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

var directSelectorOraclePrecedence = []string{
	"ORPHAN_RISK", "CLEANUP_FAILED", "TEARDOWN_FAILED", "OUTPUT_LIMIT", "TIMEOUT", "TRANSPORT_FAILED",
	"CAPTURE_FAILED", "RESPONSE_PARSE_FAILED", "PROJECTION_FAILED", "READINESS_FAILED", "START_FAILED",
	"ENVIRONMENT_INVALID", "FIXTURE_OVERLAY_FAILED", "SOURCE_COPY_FAILED", "EXECUTION_ROOT_FAILED",
	"SOURCE_INVENTORY_INVALID",
}

var ownerSelectorOracleReasons = []string{
	"MATERIALIZATION_ERROR", "SETUP_ERROR", "START_ERROR", "READINESS_ERROR", "PROBE_TRANSPORT_ERROR",
	"TIMEOUT", "CANCELLED", "OUTPUT_LIMIT", "PROJECTION_REJECTED", "ORPHAN_RISK", "TEARDOWN_ERROR",
	"UNSUPPORTED_GIT_MODE", "MISSING_OBJECT", "BUDGET_EXHAUSTED",
}

func TestDirectResultSelectorExhaustiveGoNodeMatrix(t *testing.T) {
	assertDirectSelectorParity(t, "semantic conform", nil, false, true, false, directSelectorOK("CONFORMS", "NONE"))
	assertDirectSelectorParity(t, "semantic contradiction", nil, false, false, false, directSelectorOK("CONTRADICTS", "PREDICATE_MISMATCH"))
	assertDirectSelectorParity(t, "internal only", nil, true, true, false, directSelectorOK("HARNESS_FAILURE", "INTERNAL_INVARIANT_FAILED"))
	assertDirectSelectorParity(t, "tamper only", nil, false, true, true, directSelectorOK("TAMPER_DETECTED", "COMPANION_INTEGRITY_MISMATCH"))
	assertDirectSelectorParity(t, "tamper before internal", nil, true, true, true, directSelectorOK("TAMPER_DETECTED", "COMPANION_INTEGRITY_MISMATCH"))

	for index, reason := range directSelectorOraclePrecedence {
		assertDirectSelectorParity(t, "single "+reason, []string{reason}, false, true, false, directSelectorOK("INELIGIBLE_EXECUTION", reason))
		if index < 3 {
			assertDirectSelectorParity(t, "safety before tamper and internal "+reason, []string{reason}, true, false, true, directSelectorOK("INELIGIBLE_EXECUTION", reason))
			continue
		}
		assertDirectSelectorParity(t, "internal before ordinary "+reason, []string{reason}, true, false, false, directSelectorOK("HARNESS_FAILURE", "INTERNAL_INVARIANT_FAILED"))
		assertDirectSelectorParity(t, "tamper before ordinary "+reason, []string{reason}, false, false, true, directSelectorOK("TAMPER_DETECTED", "COMPANION_INTEGRITY_MISMATCH"))
		assertDirectSelectorParity(t, "tamper before internal and ordinary "+reason, []string{reason}, true, false, true, directSelectorOK("TAMPER_DETECTED", "COMPANION_INTEGRITY_MISMATCH"))
	}

	for higher := 0; higher < len(directSelectorOraclePrecedence); higher++ {
		for lower := higher + 1; lower < len(directSelectorOraclePrecedence); lower++ {
			higherReason := directSelectorOraclePrecedence[higher]
			lowerReason := directSelectorOraclePrecedence[lower]
			want := directSelectorOK("INELIGIBLE_EXECUTION", higherReason)
			assertDirectSelectorParity(t, higherReason+" before "+lowerReason, []string{higherReason, lowerReason}, false, false, false, want)
			assertDirectSelectorParity(t, lowerReason+" before "+higherReason, []string{lowerReason, higherReason}, false, false, false, want)
		}
	}

	triples := []struct {
		name     string
		reasons  []string
		internal bool
		tamper   bool
		outcome  string
		reason   string
	}{
		{"three safety families", []string{"TEARDOWN_FAILED", "CLEANUP_FAILED", "ORPHAN_RISK"}, true, true, "INELIGIBLE_EXECUTION", "ORPHAN_RISK"},
		{"cleanup teardown ordinary", []string{"OUTPUT_LIMIT", "TEARDOWN_FAILED", "CLEANUP_FAILED"}, true, true, "INELIGIBLE_EXECUTION", "CLEANUP_FAILED"},
		{"teardown two ordinary", []string{"TIMEOUT", "TEARDOWN_FAILED", "OUTPUT_LIMIT"}, true, true, "INELIGIBLE_EXECUTION", "TEARDOWN_FAILED"},
		{"tamper internal two ordinary", []string{"TIMEOUT", "OUTPUT_LIMIT"}, true, true, "TAMPER_DETECTED", "COMPANION_INTEGRITY_MISMATCH"},
		{"internal two ordinary", []string{"TIMEOUT", "OUTPUT_LIMIT"}, true, false, "HARNESS_FAILURE", "INTERNAL_INVARIANT_FAILED"},
		{"two concurrent ordinary", []string{"TIMEOUT", "OUTPUT_LIMIT"}, false, false, "INELIGIBLE_EXECUTION", "OUTPUT_LIMIT"},
	}
	for _, test := range triples {
		assertDirectSelectorParity(t, test.name, test.reasons, test.internal, false, test.tamper, directSelectorOK(test.outcome, test.reason))
	}

	wantRefusal := `{"code":"INVALID_RESULT_FACTS","status":"REFUSED"}`
	assertDirectSelectorParity(t, "direct cancellation forbidden", []string{"CANCELLED"}, false, false, false, wantRefusal)
	assertDirectSelectorParity(t, "unknown reason forbidden", []string{"OTHER"}, false, false, false, wantRefusal)
	assertDirectSelectorParity(t, "duplicate reason forbidden", []string{"TIMEOUT", "TIMEOUT"}, false, false, false, wantRefusal)
}

func TestOwnerEligibilitySelectorExhaustiveGoNodeMatrix(t *testing.T) {
	assertOwnerSelectorParity(t, "captured behavior", "BEHAVIOR_CAPTURED", nil, ownerSelectorOK("ELIGIBLE", nil))
	for _, reason := range ownerSelectorOracleReasons {
		assertOwnerSelectorParity(t, "single "+reason, "CONTROL_INELIGIBLE", []string{reason}, ownerSelectorOK("INELIGIBLE", []string{reason}))
	}

	for _, primary := range ownerSelectorOracleReasons {
		if isOwnerTeardownReason(primary) {
			continue
		}
		for _, teardown := range []string{"TEARDOWN_ERROR", "ORPHAN_RISK"} {
			reasons := []string{primary, teardown}
			assertOwnerSelectorParity(t, primary+" plus "+teardown, "CONTROL_INELIGIBLE", reasons, ownerSelectorOK("INELIGIBLE", reasons))
		}
		for _, trailing := range [][]string{{"TEARDOWN_ERROR", "ORPHAN_RISK"}, {"ORPHAN_RISK", "TEARDOWN_ERROR"}} {
			reasons := append([]string{primary}, trailing...)
			assertOwnerSelectorParity(t, primary+" plus "+strings.Join(trailing, " then "), "CONTROL_INELIGIBLE", reasons, ownerSelectorOK("INELIGIBLE", reasons))
		}
	}
	for _, reasons := range [][]string{{"TEARDOWN_ERROR", "ORPHAN_RISK"}, {"ORPHAN_RISK", "TEARDOWN_ERROR"}} {
		assertOwnerSelectorParity(t, strings.Join(reasons, " then "), "CONTROL_INELIGIBLE", reasons, ownerSelectorOK("INELIGIBLE", reasons))
	}

	wantRefusal := `{"code":"INVALID_OWNER_FACTS","status":"REFUSED"}`
	invalid := []struct {
		name    string
		kind    string
		reasons []string
	}{
		{"captured with reason", "BEHAVIOR_CAPTURED", []string{"TIMEOUT"}},
		{"control without reason", "CONTROL_INELIGIBLE", nil},
		{"primary follows primary", "CONTROL_INELIGIBLE", []string{"TIMEOUT", "CANCELLED"}},
		{"primary follows teardown", "CONTROL_INELIGIBLE", []string{"TEARDOWN_ERROR", "TIMEOUT"}},
		{"duplicate teardown", "CONTROL_INELIGIBLE", []string{"TIMEOUT", "TEARDOWN_ERROR", "TEARDOWN_ERROR"}},
		{"four controls", "CONTROL_INELIGIBLE", []string{"TIMEOUT", "TEARDOWN_ERROR", "ORPHAN_RISK", "TEARDOWN_ERROR"}},
		{"pre-admission reason", "CONTROL_INELIGIBLE", []string{"ENVELOPE_REJECTED"}},
		{"unknown kind", "OTHER", nil},
	}
	for _, test := range invalid {
		assertOwnerSelectorParity(t, test.name, test.kind, test.reasons, wantRefusal)
	}
}

func assertDirectSelectorParity(t *testing.T, name string, reasons []string, internal, predicate, tamper bool, want string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		input := fmt.Sprintf(
			`{"ineligible_reasons":%s,"internal_failure":%t,"predicate_match":%t,"tamper":%t}`,
			selectorStringArray(reasons), internal, predicate, tamper,
		)
		assertSelectorParity(t, parityRequest(SelectDirectResult, input), want)
	})
}

func assertOwnerSelectorParity(t *testing.T, name, kind string, reasons []string, want string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		input := fmt.Sprintf(`{"kind":%s,"reasons":%s}`, strconv.Quote(kind), selectorStringArray(reasons))
		assertSelectorParity(t, parityRequest(SelectOwnerEligibility, input), want)
	})
}

func assertSelectorParity(t *testing.T, requestBytes []byte, want string) {
	t.Helper()
	request, err := ParseRequest(requestBytes)
	if err != nil {
		t.Fatalf("selector request was rejected by the driver: %v", err)
	}
	if got := string(Evaluate(request).CanonicalBytes()); got != want {
		t.Fatalf("Go selector result:\n%s\nwant:\n%s", got, want)
	}
	if got := runNodeParity(t, requestBytes); got != want+"\n" {
		t.Fatalf("Node selector result:\n%s\nwant:\n%s", got, want+"\n")
	}
}

func directSelectorOK(outcome, reason string) string {
	return fmt.Sprintf(`{"status":"OK","value":{"outcome":%s,"reason":%s}}`, strconv.Quote(outcome), strconv.Quote(reason))
}

func ownerSelectorOK(eligibility string, reasons []string) string {
	return fmt.Sprintf(`{"status":"OK","value":{"eligibility":%s,"reasons":%s}}`, strconv.Quote(eligibility), selectorStringArray(reasons))
}

func selectorStringArray(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = strconv.Quote(value)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

func isOwnerTeardownReason(reason string) bool {
	return reason == "TEARDOWN_ERROR" || reason == "ORPHAN_RISK"
}
