package model

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
)

func testDigest(character byte) domain.Digest {
	return domain.MustDigest("sha256:" + string(bytes.Repeat([]byte{character}, 64)))
}

func testRef(t testing.TB, kind EvidenceKind, character byte) EvidenceRef {
	t.Helper()
	ref, err := newEvidenceRef(kind, testDigest(character))
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func testBundle(t testing.TB) emitmodel.ContractBundle {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	pretty, err := os.ReadFile(filepath.Join(root, "spec", "examples", "v1", "contract-bundle.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	exact, err := canon.Canonicalize(pretty)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := canon.DigestBytes("ContractBundle", exact)
	if err != nil {
		t.Fatal(err)
	}
	parsedDigest, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := emitmodel.ParseContractBundle(exact, parsedDigest)
	if err != nil || !bundle.Valid() {
		t.Fatalf("test ContractBundle invalid: %v", err)
	}
	return bundle
}

func testTarget(t testing.TB, bundle emitmodel.ContractBundle) ContractExecutionTarget {
	t.Helper()
	tree := TreeBinding{
		ObjectFormat:                  "sha1",
		CommitOID:                     string(bytes.Repeat([]byte{'a'}, 40)),
		TreeOID:                       string(bytes.Repeat([]byte{'b'}, 40)),
		PortableTreeDigest:            testDigest('c'),
		MaterializationPolicyDigest:   testDigest('d'),
		MaterializationManifestDigest: testDigest('e'),
	}
	identity, err := pinnedTreeIdentityDigest(tree)
	if err != nil {
		t.Fatal(err)
	}
	tree.TreeIdentityDigest = identity
	target, err := NewContractExecutionTarget(TargetInput{
		ContractBundleDigest: bundle.Digest(),
		TerminalResidue: TerminalResidueBinding{
			StudyID:      "study:" + string(bytes.Repeat([]byte{'1'}, 64)),
			HeadRevision: 8, HeadDigest: testDigest('f'), LineageRootDigest: testDigest('2'),
		},
		Tree: tree,
		Attempt: AttemptBinding{
			ArtifactDigest: testDigest('3'), InstanceNonce: string(bytes.Repeat([]byte{'4'}, 64)),
		},
		BootSession: BootSessionBinding{IdentityDigest: testDigest('5')},
		Runtime: RuntimeBinding{
			AdmittedExecutablePath:  "/opt/homebrew/bin/node",
			MeasuredProcessExecPath: "/opt/homebrew/bin/node",
			Version:                 "25.2.1", Major: 25, Platform: "darwin", Architecture: "arm64",
			ExecutableBytesDigest: testDigest('6'), ExecutableMode: "100755", ExecutableByteCount: 1024,
			ProbeProgramDigest: testDigest('7'),
		},
	})
	if err != nil || !target.Valid() {
		t.Fatalf("test target invalid: %v", err)
	}
	return target
}

func testProcessEvidence(t testing.TB, started bool) []EvidenceRef {
	t.Helper()
	kinds := []EvidenceKind{
		EvidenceMaterializationRevalidation, EvidenceRuntimeRevalidation,
	}
	if started {
		kinds = append(kinds, EvidenceProcessResult, EvidenceWaitResult, EvidenceDrainResult)
	}
	kinds = append(kinds, EvidenceTeardownResult, EvidenceOrphanCheck, EvidenceFinalizationMarker)
	result := make([]EvidenceRef, len(kinds))
	const hexCharacters = "89abcdef"
	for index, kind := range kinds {
		result[index] = testRef(t, kind, hexCharacters[index])
	}
	return result
}

func testScope(t testing.TB, states [5]ScopeCheckState) StandaloneScope {
	t.Helper()
	checks := make([]ScopeCheck, len(scopeDomainOrder))
	for index, scopeDomain := range scopeDomainOrder {
		var err error
		switch states[index] {
		case ScopeCheckClean:
			checks[index], err = NewCleanScopeCheck(scopeDomain, testRef(t, scopeEvidenceKind(scopeDomain), byte('a'+index)))
		case ScopeCheckMissing:
			checks[index], err = NewMissingScopeCheck(scopeDomain)
		case ScopeCheckViolated:
			checks[index], err = NewViolatedScopeCheck(
				scopeDomain,
				testRef(t, scopeEvidenceKind(scopeDomain), byte('a'+index)),
				testViolation(scopeDomain),
			)
		default:
			t.Fatalf("unknown scope state %q", states[index])
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	scope, err := NewStandaloneScope(checks)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func testViolation(scopeDomain ScopeDomain) ScopeViolation {
	switch scopeDomain {
	case ScopeTargetInventory:
		return ViolationTargetSourcePresent
	case ScopeChildBindings:
		return ViolationChildBinding
	case ScopeImportResolution:
		return ViolationImportResolution
	case ScopeServiceBindings:
		return ViolationServiceBinding
	case ScopeSentinelInheritance:
		return ViolationSentinelInherited
	default:
		return ""
	}
}

func testPrivateManifest(t testing.TB) PrivateManifestSummary {
	t.Helper()
	summary, err := NewPrivateManifestSummary(testRef(t, EvidencePrivateManifest, 'f'), 1, 32)
	if err != nil {
		t.Fatal(err)
	}
	return summary
}

func testCleanWitness(t testing.TB, bundle emitmodel.ContractBundle) ClosedRunWitness {
	t.Helper()
	spawn, _ := NewChildPIDObservation(4242)
	process, err := NewCleanProcessClosure(testProcessEvidence(t, true))
	if err != nil {
		t.Fatal(err)
	}
	allowed := bundle.Predicate().AllowedTuples()
	if len(allowed) == 0 {
		t.Fatal("bundle has no allowed tuple")
	}
	observation, err := NewProjectedObservation(
		testRef(t, EvidenceCapturedObservation, 'd'),
		testRef(t, EvidenceProjectionResult, 'e'),
		allowed[0],
	)
	if err != nil {
		t.Fatal(err)
	}
	clean := [5]ScopeCheckState{ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean, ScopeCheckClean}
	witness, err := NewClosedRunWitness(spawn, process, observation, testScope(t, clean), testPrivateManifest(t))
	if err != nil || !witness.Valid() {
		t.Fatalf("clean witness invalid: %v", err)
	}
	return witness
}

func TestCanonicalObjectGraphRoundTripsAndClassifiesWithoutRequestedResult(t *testing.T) {
	bundle := testBundle(t)
	target := testTarget(t, bundle)
	parsedTarget, err := ParseContractExecutionTarget(target.CanonicalBytes(), target.Digest())
	if err != nil || !parsedTarget.Valid() || parsedTarget.Digest() != target.Digest() {
		t.Fatalf("target parse failed: %v", err)
	}
	witness := testCleanWitness(t, bundle)
	run, err := NewFinalizedContractRun(target, testDigest('9'), witness)
	if err != nil || !run.Valid() || run.Disposition() != DispositionEligibleClean {
		t.Fatalf("run invalid: disposition=%q err=%v", run.Disposition(), err)
	}
	parsedRun, err := ParseFinalizedContractRun(run.CanonicalBytes(), run.Digest(), target)
	if err != nil || !parsedRun.Valid() || !parsedRun.MatchesTarget(target) {
		t.Fatalf("run parse failed: %v", err)
	}
	execution, err := DeriveContractExecution(bundle, target, run)
	if err != nil || !execution.Valid() || execution.Result() != ResultConforms {
		t.Fatalf("execution invalid: result=%q err=%v", execution.Result(), err)
	}
	parsedExecution, err := ParseContractExecution(execution.CanonicalBytes(), execution.Digest(), bundle, target, run)
	if err != nil || !parsedExecution.Equal(execution) {
		t.Fatalf("execution parse failed: %v", err)
	}
	if !ClassifierProfileDigest().Valid() {
		t.Fatal("classifier profile digest is invalid")
	}
}

func TestCanonicalGettersAndInputsAreDefensive(t *testing.T) {
	bundle := testBundle(t)
	target := testTarget(t, bundle)
	before := target.CanonicalBytes()
	copyBytes := target.CanonicalBytes()
	copyBytes[0] ^= 0xff
	input := target.Input()
	input.Attempt.InstanceNonce = string(bytes.Repeat([]byte{'0'}, 64))
	if !bytes.Equal(target.CanonicalBytes(), before) || !target.Valid() {
		t.Fatal("target getter exposed retained storage")
	}
	witness := testCleanWitness(t, bundle)
	copyWitness := witness.CanonicalBytes()
	copyWitness[0] ^= 0xff
	checks := witness.StandaloneScope().Checks()
	checks[0] = ScopeCheck{}
	if !witness.Valid() {
		t.Fatal("witness getter exposed retained storage")
	}
}

func TestParentMismatchAndNoncanonicalBytesRefuse(t *testing.T) {
	bundle := testBundle(t)
	target := testTarget(t, bundle)
	run, err := NewFinalizedContractRun(target, testDigest('9'), testCleanWitness(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	otherInput := target.Input()
	otherInput.Attempt.InstanceNonce = string(bytes.Repeat([]byte{'0'}, 64))
	other, err := NewContractExecutionTarget(otherInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFinalizedContractRun(run.CanonicalBytes(), run.Digest(), other); !IsCode(err, CodeTargetRunMismatch) {
		t.Fatalf("cross-target parse error = %v", err)
	}
	noncanonical := append(run.CanonicalBytes(), '\n')
	if _, err := ParseFinalizedContractRun(noncanonical, run.Digest(), target); !IsCode(err, CodeInvalidRun) {
		t.Fatalf("noncanonical parse error = %v", err)
	}
}

func TestC1ExampleProbe(t *testing.T) {
	if os.Getenv("COUNTERSHAPE_C1_EXAMPLE_PROBE") != "1" {
		t.Skip("fixture probe is opt-in")
	}
	bundle := testBundle(t)
	target := testTarget(t, bundle)
	run, err := NewFinalizedContractRun(target, testDigest('9'), testCleanWitness(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	execution, err := DeriveContractExecution(bundle, target, run)
	if err != nil {
		t.Fatal(err)
	}
	encode := func(exact []byte) string { return base64.StdEncoding.EncodeToString(exact) }
	fmt.Printf(
		"COUNTERSHAPE_C1_MODEL_V1|%s|%s|%s|%s|%s|%s\n",
		encode(target.CanonicalBytes()), target.Digest(),
		encode(run.CanonicalBytes()), run.Digest(),
		encode(execution.CanonicalBytes()), execution.Digest(),
	)
}
