package scope

import (
	"strings"
	"testing"

	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
)

func TestC5ScopeDerivesExactFiveDomainCleanClosure(t *testing.T) {
	inventory := referenceInventoryForTest()
	findings := Evaluate(AssessmentInput{
		Before: inventory,
		After:  inventory,

		ChildStarted:      true,
		ChildPID:          4217,
		ProcessGroupID:    4217,
		ProcessGroupOwned: true,
		PreparedBinding:   "sha256:" + strings.Repeat("1", 64),
		ObservedBinding:   "sha256:" + strings.Repeat("1", 64),
		ReceiptValidated:  true,

		ReadinessAccepted:  true,
		ReadinessEndpoint:  "127.0.0.1:43127",
		ConnectionAttempts: 1,
		Measurements: Measurements{
			ModuleSHA256: strings.Repeat("2", 64),
		},
	})
	wantDomains := []contractmodel.ScopeDomain{
		contractmodel.ScopeTargetInventory,
		contractmodel.ScopeChildBindings,
		contractmodel.ScopeImportResolution,
		contractmodel.ScopeServiceBindings,
		contractmodel.ScopeSentinelInheritance,
	}
	if len(findings) != len(wantDomains) {
		t.Fatalf("scope finding count = %d, want %d", len(findings), len(wantDomains))
	}
	for index, finding := range findings {
		if !finding.Valid() || finding.Domain != wantDomains[index] ||
			finding.State != contractmodel.ScopeCheckClean ||
			finding.Violation != "" || len(finding.Facts) == 0 ||
			finding.Diagnostic.Valid() {
			t.Fatalf("clean finding %d = %#v", index, finding)
		}
	}
}

func TestC5ScopeViolationOutranksMissingAcrossForbiddenControls(t *testing.T) {
	before := referenceInventoryForTest()
	after := before
	after.entries = append(after.entries, Entry{
		Path: "countershape-runtime.mjs", Mode: "100644", Size: 1,
		SHA256: strings.Repeat("3", 64),
	})
	after.digest = strings.Repeat("4", 64)
	findings := Evaluate(AssessmentInput{
		Before: before,
		After:  after,

		ReceiptContradicted: true,
		SentinelInherited:   true,
		NodePathDeclared:    true,
		Measurements: Measurements{
			ImportAccepts:  1,
			ServiceAccepts: 2,
			Ambiguous:      true,
			Diagnostic:     NewDiagnostic(CodeProbeCleanup),
		},
	})
	want := map[contractmodel.ScopeDomain]contractmodel.ScopeViolation{
		contractmodel.ScopeTargetInventory:     contractmodel.ViolationTargetSourcePresent,
		contractmodel.ScopeChildBindings:       contractmodel.ViolationChildBinding,
		contractmodel.ScopeImportResolution:    contractmodel.ViolationImportResolution,
		contractmodel.ScopeServiceBindings:     contractmodel.ViolationServiceBinding,
		contractmodel.ScopeSentinelInheritance: contractmodel.ViolationSentinelInherited,
	}
	if len(findings) != len(want) {
		t.Fatalf("scope finding count = %d, want %d", len(findings), len(want))
	}
	for _, finding := range findings {
		if !finding.Valid() || finding.State != contractmodel.ScopeCheckViolated ||
			finding.Violation != want[finding.Domain] {
			t.Fatalf("violation did not outrank missing for %s: %#v", finding.Domain, finding)
		}
	}
}

func referenceInventoryForTest() Inventory {
	return Inventory{
		entries: []Entry{
			{
				Path: "candidate-role.json", Mode: "100644", Size: 20,
				SHA256: referenceRoleDigests[20],
			},
			{Path: "fixture", Mode: "040700"},
			{
				Path: "fixture/server_child_bind.mjs", Mode: "100755",
				Size: referenceProgramBytes, SHA256: referenceProgramSHA256,
			},
		},
		digest: strings.Repeat("a", 64),
	}
}
