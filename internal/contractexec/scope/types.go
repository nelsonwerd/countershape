// Package scope owns the bounded, profile-specific observations used by the
// standalone HTTP runner. It reports exact local facts and deliberately makes
// no host-wide or containment claim.
package scope

import (
	"errors"

	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
)

const (
	CodeProbeAbsent       = "HTTP_SCOPE_PROBE_ABSENT"
	CodeProbeIntegrity    = "HTTP_SCOPE_PROBE_INTEGRITY"
	CodeProbeCleanup      = "HTTP_SCOPE_PROBE_CLEANUP"
	CodeInventoryInvalid  = "HTTP_SCOPE_INVENTORY_INVALID"
	CodeInventoryLimit    = "HTTP_SCOPE_INVENTORY_LIMIT"
	CodeInventoryChanged  = "HTTP_SCOPE_INVENTORY_CHANGED"
	CodeInventorySpecial  = "HTTP_SCOPE_INVENTORY_SPECIAL_ENTRY"
	CodeInventoryRead     = "HTTP_SCOPE_INVENTORY_READ"
	CodeInventoryIdentity = "HTTP_SCOPE_INVENTORY_IDENTITY"
)

// Diagnostic is a closed, bounded scope failure. Detail is intentionally
// absent: raw host errors remain operational data rather than semantic input.
type Diagnostic struct{ code string }

func NewDiagnostic(code string) Diagnostic {
	switch code {
	case CodeProbeAbsent, CodeProbeIntegrity, CodeProbeCleanup, CodeInventoryInvalid,
		CodeInventoryLimit, CodeInventoryChanged, CodeInventorySpecial,
		CodeInventoryRead, CodeInventoryIdentity:
		return Diagnostic{code: code}
	default:
		return Diagnostic{}
	}
}

func (d Diagnostic) Valid() bool { return NewDiagnostic(d.code).code == d.code && d.code != "" }
func (d Diagnostic) Code() string {
	if !d.Valid() {
		return ""
	}
	return d.code
}

// Error retains a bounded code while preserving a private operational cause.
type Error struct {
	Diagnostic Diagnostic
	Cause      error
}

func (e *Error) Error() string {
	if e == nil || !e.Diagnostic.Valid() {
		return "HTTP scope observation failed"
	}
	return e.Diagnostic.Code()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func DiagnosticOf(err error) (Diagnostic, bool) {
	var failure *Error
	if !errors.As(err, &failure) || failure == nil || !failure.Diagnostic.Valid() {
		return Diagnostic{}, false
	}
	return failure.Diagnostic, true
}

func fail(code string, cause error) error {
	return &Error{Diagnostic: NewDiagnostic(code), Cause: cause}
}

// Measurements closes the two explicit canaries. Counts greater than zero are
// positive observations; Ambiguous means probe integrity or cleanup failed.
type Measurements struct {
	ImportAccepts  int64
	ServiceAccepts int64
	ModuleSHA256   string
	Ambiguous      bool
	Diagnostic     Diagnostic
}

// Finding is one canonical-domain draft plus the bounded private facts used to
// derive it. Missing findings carry no evidence body; their diagnostic is
// retained by the runner's finalization summary.
type Finding struct {
	Domain     contractmodel.ScopeDomain
	State      contractmodel.ScopeCheckState
	Kind       contractmodel.EvidenceKind
	Violation  contractmodel.ScopeViolation
	Facts      map[string]any
	Diagnostic Diagnostic
}

func (f Finding) Valid() bool {
	switch f.State {
	case contractmodel.ScopeCheckClean:
		return f.Domain != "" && f.Kind != "" && f.Violation == "" && len(f.Facts) != 0
	case contractmodel.ScopeCheckViolated:
		return f.Domain != "" && f.Kind != "" && f.Violation != "" && len(f.Facts) != 0
	case contractmodel.ScopeCheckMissing:
		return f.Domain != "" && f.Kind != "" && f.Violation == "" && f.Diagnostic.Valid()
	default:
		return false
	}
}

// AssessmentInput contains only parent-observed facts. Callers cannot supply a
// scope state; Evaluate derives each state in the fixed five-domain order.
type AssessmentInput struct {
	Before, After Inventory

	ChildStarted        bool
	ChildPID            int
	ProcessGroupID      int
	ProcessGroupOwned   bool
	PreparedBinding     string
	ObservedBinding     string
	ReceiptValidated    bool
	ReceiptContradicted bool

	ReadinessAccepted  bool
	ReadinessEndpoint  string
	ConnectionAttempts int

	SentinelInherited bool
	NodePathDeclared  bool
	Measurements      Measurements
}

// Evaluate applies violation-over-missing precedence independently for every
// domain. It always returns five drafts in canonical order.
func Evaluate(input AssessmentInput) []Finding {
	enrolled := input.Before.ReferenceHTTPFixture()
	unchanged := input.Before.Equal(input.After)
	targetViolation, targetViolated := input.After.TargetViolation()

	findings := make([]Finding, 0, 5)
	add := func(
		domain contractmodel.ScopeDomain,
		kind contractmodel.EvidenceKind,
		clean bool,
		violated bool,
		violation contractmodel.ScopeViolation,
		facts map[string]any,
		diagnostic Diagnostic,
	) {
		state := contractmodel.ScopeCheckMissing
		if violated {
			state = contractmodel.ScopeCheckViolated
		} else if clean {
			state = contractmodel.ScopeCheckClean
		}
		if state == contractmodel.ScopeCheckMissing && !diagnostic.Valid() {
			diagnostic = NewDiagnostic(CodeProbeIntegrity)
		}
		if state != contractmodel.ScopeCheckViolated {
			violation = ""
		}
		if state != contractmodel.ScopeCheckMissing {
			diagnostic = Diagnostic{}
		}
		findings = append(findings, Finding{
			Domain: domain, State: state, Kind: kind, Violation: violation,
			Facts: facts, Diagnostic: diagnostic,
		})
	}

	add(
		contractmodel.ScopeTargetInventory,
		contractmodel.EvidenceTargetInventory,
		enrolled && unchanged,
		targetViolated,
		targetViolation,
		map[string]any{
			"before_inventory_sha256": input.Before.Digest(),
			"after_inventory_sha256":  input.After.Digest(),
			"before_entry_count":      input.Before.EntryCount(),
			"after_entry_count":       input.After.EntryCount(),
			"unchanged":               unchanged,
			"reference_enrolled":      enrolled,
		},
		NewDiagnostic(CodeInventoryIdentity),
	)

	bound := input.ChildStarted && input.ChildPID > 0 && input.ProcessGroupOwned &&
		input.ProcessGroupID == input.ChildPID && input.PreparedBinding != "" &&
		input.PreparedBinding == input.ObservedBinding && input.ReceiptValidated
	add(
		contractmodel.ScopeChildBindings,
		contractmodel.EvidenceChildBindings,
		enrolled && bound,
		input.ReceiptContradicted,
		contractmodel.ViolationChildBinding,
		map[string]any{
			"child_pid":            input.ChildPID,
			"process_group_id":     input.ProcessGroupID,
			"process_group_owned":  input.ProcessGroupOwned,
			"prepared_binding":     input.PreparedBinding,
			"observed_binding":     input.ObservedBinding,
			"receipt_validated":    input.ReceiptValidated,
			"receipt_contradicted": input.ReceiptContradicted,
			"reference_enrolled":   enrolled,
		},
		NewDiagnostic(CodeProbeIntegrity),
	)

	importViolated := input.Measurements.ImportAccepts > 0 || input.NodePathDeclared
	add(
		contractmodel.ScopeImportResolution,
		contractmodel.EvidenceImportResolution,
		enrolled && unchanged && !importViolated && !input.Measurements.Ambiguous,
		importViolated,
		contractmodel.ViolationImportResolution,
		map[string]any{
			"accepted_connections":  input.Measurements.ImportAccepts,
			"node_path_declared":    input.NodePathDeclared,
			"outside_module_sha256": input.Measurements.ModuleSHA256,
			"measurement_ambiguous": input.Measurements.Ambiguous,
			"reference_enrolled":    enrolled,
		},
		input.Measurements.Diagnostic,
	)

	serviceViolated := input.Measurements.ServiceAccepts > 0
	serviceClean := enrolled && unchanged && input.ReadinessAccepted &&
		input.ReadinessEndpoint != "" && input.ConnectionAttempts == 1 &&
		!serviceViolated && !input.Measurements.Ambiguous
	add(
		contractmodel.ScopeServiceBindings,
		contractmodel.EvidenceServiceBindings,
		serviceClean,
		serviceViolated,
		contractmodel.ViolationServiceBinding,
		map[string]any{
			"accepted_canary_connections": input.Measurements.ServiceAccepts,
			"child_reported_endpoint":     input.ReadinessEndpoint,
			"readiness_accepted":          input.ReadinessAccepted,
			"connection_attempts":         input.ConnectionAttempts,
			"measurement_ambiguous":       input.Measurements.Ambiguous,
			"reference_enrolled":          enrolled,
			"listener_ownership_claimed":  false,
		},
		input.Measurements.Diagnostic,
	)

	add(
		contractmodel.ScopeSentinelInheritance,
		contractmodel.EvidenceSentinelInheritance,
		enrolled && !input.SentinelInherited,
		input.SentinelInherited,
		contractmodel.ViolationSentinelInherited,
		map[string]any{
			"named_parent_roster":            ParentSentinels(),
			"child_environment_omits_roster": !input.SentinelInherited,
			"reference_enrolled":             enrolled,
		},
		NewDiagnostic(CodeProbeIntegrity),
	)
	return findings
}

var parentSentinels = [...]string{
	"COUNTERSHAPE_C5_PARENT_CREDENTIAL_SENTINEL",
	"COUNTERSHAPE_C5_PARENT_SECRET_SENTINEL",
}

func ParentSentinels() []string {
	return append([]string(nil), parentSentinels[:]...)
}

func IsParentSentinel(name string) bool {
	for _, candidate := range parentSentinels {
		if name == candidate {
			return true
		}
	}
	return false
}
