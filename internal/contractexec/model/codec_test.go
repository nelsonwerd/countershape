package model

import (
	"bytes"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestTargetPrimitiveBoundsAndDerivedRelations(t *testing.T) {
	bundle := testBundle(t)
	valid := testTarget(t, bundle).Input()
	tests := []struct {
		name   string
		mutate func(*TargetInput)
	}{
		{"study id", func(input *TargetInput) { input.TerminalResidue.StudyID = "study:bad" }},
		{"revision", func(input *TargetInput) { input.TerminalResidue.HeadRevision = 0 }},
		{"oid width", func(input *TargetInput) { input.Tree.CommitOID = input.Tree.CommitOID[:39] }},
		{"tree identity", func(input *TargetInput) { input.Tree.TreeIdentityDigest = testDigest('0') }},
		{"nonce width", func(input *TargetInput) { input.Attempt.InstanceNonce = input.Attempt.InstanceNonce[:63] }},
		{"relative runtime", func(input *TargetInput) {
			input.Runtime.AdmittedExecutablePath = "bin/node"
			input.Runtime.MeasuredProcessExecPath = "bin/node"
		}},
		{"path mismatch", func(input *TargetInput) { input.Runtime.MeasuredProcessExecPath = "/usr/bin/node" }},
		{"version", func(input *TargetInput) { input.Runtime.Version = "unknown" }},
		{"major mismatch", func(input *TargetInput) { input.Runtime.Major++ }},
		{"platform", func(input *TargetInput) { input.Runtime.Platform = "linux" }},
		{"architecture", func(input *TargetInput) { input.Runtime.Architecture = "amd64" }},
		{"mode", func(input *TargetInput) { input.Runtime.ExecutableMode = "120755" }},
		{"nonexecutable mode", func(input *TargetInput) { input.Runtime.ExecutableMode = "100644" }},
		{"empty executable", func(input *TargetInput) { input.Runtime.ExecutableByteCount = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if _, err := NewContractExecutionTarget(input); !IsCode(err, CodeInvalidTarget) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	t.Run("runtime version schema boundary", func(t *testing.T) {
		input := valid
		input.Runtime.Version = "v1." + string(bytes.Repeat([]byte{'0'}, 125))
		input.Runtime.Major = 1
		if len(input.Runtime.Version) != 128 {
			t.Fatalf("maximum fixture width = %d", len(input.Runtime.Version))
		}
		maximum, err := NewContractExecutionTarget(input)
		if err != nil || !maximum.Valid() {
			t.Fatalf("128-byte runtime version was refused: %v", err)
		}
		parsed, err := ParseContractExecutionTarget(maximum.CanonicalBytes(), maximum.Digest())
		if err != nil || !parsed.Valid() {
			t.Fatalf("128-byte runtime version did not round-trip: %v", err)
		}

		overLimit := "v1." + string(bytes.Repeat([]byte{'0'}, 126))
		if len(overLimit) != 129 {
			t.Fatalf("over-limit fixture width = %d", len(overLimit))
		}
		input.Runtime.Version = overLimit
		if _, err := NewContractExecutionTarget(input); !IsCode(err, CodeInvalidTarget) {
			t.Fatalf("129-byte constructor error = %v", err)
		}
		overLimitBody := mutateCanonical(t, maximum.CanonicalBytes(), func(root map[string]any) {
			root["runtime_binding"].(map[string]any)["version"] = overLimit
		})
		overLimitDigest := typedDigestForTest(t, TargetKind, overLimitBody)
		if _, err := ParseContractExecutionTarget(overLimitBody, overLimitDigest); !IsCode(err, CodeInvalidTarget) {
			t.Fatalf("129-byte parser error = %v", err)
		}
	})
	t.Run("legacy clock-derived boot profile", func(t *testing.T) {
		target := testTarget(t, bundle)
		legacy := mutateCanonical(t, target.CanonicalBytes(), func(root map[string]any) {
			root["boot_session_binding"].(map[string]any)["profile"] = "DARWIN_KERN_BOOTTIME_V1"
		})
		legacyDigest := digestForBytes(t, TargetKind, legacy)
		if _, err := ParseContractExecutionTarget(legacy, legacyDigest); !IsCode(err, CodeInvalidTarget) {
			t.Fatalf("legacy clock-derived profile error = %v", err)
		}
	})
}

func TestClosedEnumBoundsAndTypedRoles(t *testing.T) {
	for _, code := range []string{
		"OS_START_ERROR",
		"PRESPAWN_MATERIALIZATION_REVALIDATION_FAILED",
		"PRESPAWN_RUNTIME_REVALIDATION_FAILED",
	} {
		if observation, err := NewStartErrorObservation(code); err != nil || !observation.Valid() {
			t.Fatalf("start error %q invalid: %v", code, err)
		}
	}
	for _, code := range []string{"", "START_ERROR", "os_start_error", "UNKNOWN"} {
		if _, err := NewStartErrorObservation(code); !IsCode(err, CodeInvalidWitness) {
			t.Fatalf("unknown start code %q error = %v", code, err)
		}
	}
	for _, pid := range []int64{1, 2147483647} {
		if observation, err := NewChildPIDObservation(pid); err != nil || !observation.Valid() {
			t.Fatalf("PID %d invalid: %v", pid, err)
		}
	}
	for _, pid := range []int64{0, -1, 2147483648} {
		if _, err := NewChildPIDObservation(pid); !IsCode(err, CodeInvalidWitness) {
			t.Fatalf("PID %d error = %v", pid, err)
		}
	}
	for _, reason := range []domain.ControlReason{
		domain.ControlSetupError,
		domain.ControlTeardownError,
		domain.ControlOrphanRisk,
		domain.ControlEnvelopeRejected,
		domain.ControlUnsupportedGitMode,
		domain.ControlMissingObject,
		domain.ControlBudgetExhausted,
	} {
		if _, err := NewControlledProcessClosure(reason, false, false, testProcessEvidence(t, true)); !IsCode(err, CodeInvalidWitness) {
			t.Fatalf("primary %q error = %v", reason, err)
		}
	}
	wrongRole := testRef(t, EvidenceServiceBindings, '1')
	if _, err := NewCapturedUnprojected(wrongRole); !IsCode(err, CodeInvalidWitness) {
		t.Fatalf("wrong capture role error = %v", err)
	}
}

func TestPrimaryAndCleanupControlsRemainIndependent(t *testing.T) {
	closure, err := NewControlledProcessClosure(
		domain.ControlTimeout, true, true, testProcessEvidence(t, true),
	)
	if err != nil || !closure.Valid() {
		t.Fatal(err)
	}
	primary, ok := closure.Primary()
	if !ok || primary != domain.ControlTimeout || !closure.TeardownError() || !closure.OrphanRisk() {
		t.Fatalf("closure lost independent controls: %#v", closure)
	}
}

func TestScopeDerivationViolationOutranksMissing(t *testing.T) {
	states := [5]ScopeCheckState{ScopeCheckMissing, ScopeCheckViolated, ScopeCheckClean, ScopeCheckMissing, ScopeCheckClean}
	scope := testScope(t, states)
	if scope.State() != ScopeViolated {
		t.Fatalf("scope state = %q", scope.State())
	}
	duplicate := scope.Checks()
	duplicate[4] = duplicate[0]
	if _, err := NewStandaloneScope(duplicate); !IsCode(err, CodeInvalidWitness) {
		t.Fatalf("duplicate scope error = %v", err)
	}
}

func TestPrivateEvidenceCeilingsAndZeroCorrelation(t *testing.T) {
	manifest := testRef(t, EvidencePrivateManifest, '1')
	for _, test := range []struct {
		blobs int64
		bytes int64
		valid bool
	}{
		{0, 0, true},
		{1, 1, true},
		{16, MaxPrivateEvidenceBytes, true},
		{0, 1, false},
		{1, 0, false},
		{17, 17, false},
		{16, MaxPrivateEvidenceBytes + 1, false},
	} {
		summary, err := NewPrivateManifestSummary(manifest, test.blobs, test.bytes)
		if (err == nil && summary.Valid()) != test.valid {
			t.Fatalf("summary (%d,%d) valid=%t err=%v", test.blobs, test.bytes, summary.Valid(), err)
		}
	}
}

func TestCleanWitnessUsesExactSixteenTypedReferences(t *testing.T) {
	witness := testCleanWitness(t, testBundle(t))
	kinds := map[EvidenceKind]struct{}{}
	for _, ref := range witness.process.evidence {
		kinds[ref.kind] = struct{}{}
	}
	if ref, ok := witness.observation.CapturedRef(); ok {
		kinds[ref.kind] = struct{}{}
	}
	if ref, ok := witness.observation.ProjectionRef(); ok {
		kinds[ref.kind] = struct{}{}
	}
	for _, check := range witness.scope.checks {
		ref, _ := check.Evidence()
		kinds[ref.kind] = struct{}{}
	}
	kinds[witness.privateManifest.manifestRef.kind] = struct{}{}
	if len(kinds) != MaxWitnessReferences || len(witness.CanonicalBytes()) > MaxClosedRunWitnessBytes {
		t.Fatalf("witness refs=%d bytes=%d", len(kinds), len(witness.CanonicalBytes()))
	}
}

func TestStrictParsersRejectCoherentlyRehashedUnknownAndDerivedMembers(t *testing.T) {
	bundle := testBundle(t)
	target := testTarget(t, bundle)
	unknownTarget := mutateCanonical(t, target.CanonicalBytes(), func(root map[string]any) {
		root["artifact_digest"] = testDigest('0').String()
	})
	unknownTargetDigest := typedDigestForTest(t, TargetKind, unknownTarget)
	if _, err := ParseContractExecutionTarget(unknownTarget, unknownTargetDigest); !IsCode(err, CodeInvalidTarget) {
		t.Fatalf("unknown target member error = %v", err)
	}
	run, err := NewFinalizedContractRun(target, testDigest('9'), testCleanWitness(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	duplicatedDisposition := mutateCanonical(t, run.CanonicalBytes(), func(root map[string]any) {
		root["terminal_disposition"] = string(DispositionEligibleClean)
	})
	mutatedRunDigest := typedDigestForTest(t, RunKind, duplicatedDisposition)
	if _, err := ParseFinalizedContractRun(duplicatedDisposition, mutatedRunDigest, target); !IsCode(err, CodeInvalidRun) {
		t.Fatalf("serialized disposition error = %v", err)
	}
	execution, err := DeriveContractExecution(bundle, target, run)
	if err != nil {
		t.Fatal(err)
	}
	requestedResult := mutateCanonical(t, execution.CanonicalBytes(), func(root map[string]any) {
		root["result"] = string(ResultContradicts)
	})
	requestedDigest := typedDigestForTest(t, ExecutionKind, requestedResult)
	if _, err := ParseContractExecution(requestedResult, requestedDigest, bundle, target, run); !IsCode(err, CodeInvalidExecution) {
		t.Fatalf("requested result error = %v", err)
	}
}

func mutateCanonical(t *testing.T, exact []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	value, err := canon.Parse(exact)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := plainValue(value)
	if err != nil {
		t.Fatal(err)
	}
	root, ok := plain.(map[string]any)
	if !ok {
		t.Fatal("canonical root is not an object")
	}
	mutate(root)
	result, err := canon.CanonicalizeTyped(root)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func typedDigestForTest(t *testing.T, kind string, exact []byte) domain.Digest {
	t.Helper()
	digest, err := canon.DigestBytes(kind, exact)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestClassifierProfileIdentityGolden(t *testing.T) {
	const want = "sha256:789b54457f24e19a181da1e0d44f45a491d21c5550b622de0f0ef19ec15412f4"
	if got := ClassifierProfileDigest().String(); got != want {
		t.Fatalf("classifier profile digest = %s, want %s", got, want)
	}
}

func TestCanonicalObjectBodiesHaveNoSelfDigestOrMutableSelectors(t *testing.T) {
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
	for name, exact := range map[string][]byte{
		"target": target.CanonicalBytes(), "run": run.CanonicalBytes(), "execution": execution.CanonicalBytes(),
	} {
		root, err := canon.Parse(exact)
		if err != nil {
			t.Fatal(err)
		}
		if _, present := root.LookupMember("artifact_digest"); present {
			t.Fatalf("%s body contains a self digest", name)
		}
		for _, forbidden := range [][]byte{
			[]byte("latest"), []byte("current_result"), []byte("didrun"),
		} {
			if bytes.Contains(exact, forbidden) {
				t.Fatalf("%s body contains forbidden selector %q", name, forbidden)
			}
		}
	}
}
