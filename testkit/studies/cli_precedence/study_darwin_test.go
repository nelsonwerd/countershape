//go:build darwin && cgo

package cli_precedence

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/spec"
	"github.com/nelsonwerd/countershape/internal/world"
	"github.com/nelsonwerd/countershape/testkit/clifixture"
)

func TestCLIReferencePrecedenceStudy(t *testing.T) {
	ambientHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(ambientHome, ".countershape-precedence.json"), []byte(`{"mode":"ambient-must-not-cross"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", ambientHome)
	t.Setenv("COUNTERSHAPE_AMBIENT_SENTINEL", "must-not-cross")
	config := referenceConfig(t)
	result, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if result.Observation.Status() != observe.ObservationComplete || result.Observation.CompletedMatrices() != 3 {
		t.Fatalf("observation status=%s matrices=%d", result.Observation.Status(), result.Observation.CompletedMatrices())
	}
	if !result.HasOutcomeMap || !result.OutcomeMap.Divergence() || result.OutcomeMap.DistinctProjectionCount() != 3 ||
		len(result.OutcomeMap.Entries()) != 3 || len(result.OutcomeMap.Exclusions()) != 0 {
		t.Fatalf("unexpected exact outcome map: entries=%d exclusions=%d distinct=%d divergence=%v",
			len(result.OutcomeMap.Entries()), len(result.OutcomeMap.Exclusions()),
			result.OutcomeMap.DistinctProjectionCount(), result.OutcomeMap.Divergence())
	}
	for _, batch := range result.Observation.Batches() {
		classification := batch.Classification()
		if classification.Status() != observe.ObservedStable || classification.EligibleTrials() != 3 ||
			classification.RequiredTrials() != 3 || !strings.HasPrefix(classification.BoundedLabel(), "OBSERVED_STABLE(3/3,") {
			t.Fatalf("candidate %s classification=%s eligible=%d/%d label=%q", batch.CandidateKey(),
				classification.Status(), classification.EligibleTrials(), classification.RequiredTrials(), classification.BoundedLabel())
		}
	}
	if len(result.Trials) != 9 {
		t.Fatalf("trial evidence count=%d, want 9", len(result.Trials))
	}
	seenRootsByKind := map[string]map[string]struct{}{}
	seenAttempts := map[domain.Digest]struct{}{}
	seenWorlds := map[domain.Digest]struct{}{}
	representativeProjection := map[clifixture.CandidateRole][]byte{}
	expected := map[clifixture.CandidateRole]map[cli.CLIFieldID]string{
		clifixture.ConfigFirst: {
			cli.CLIFieldStdoutJSONMode:   "config",
			cli.CLIFieldStdoutJSONSource: "config",
		},
		clifixture.EnvironmentFirst: {
			cli.CLIFieldStdoutJSONMode:   "env",
			cli.CLIFieldStdoutJSONSource: "env",
		},
		clifixture.ArgvFirst: {
			cli.CLIFieldStdoutJSONMode:   "argv",
			cli.CLIFieldStdoutJSONSource: "argv",
		},
	}
	for _, trial := range result.Trials {
		if !trial.Admitted {
			t.Fatalf("complete study exposed unadmitted trial %d", trial.Slot.Ordinal())
		}
		attemptRoot := trial.Result.Roots().Attempt()
		assertFreshOwnedRoots(t, trial, seenRootsByKind)
		if info, statErr := os.Lstat(attemptRoot); statErr != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("attempt root facts changed: %s %+v %v", attemptRoot, info, statErr)
		}
		attemptDigest := trial.Result.FinalizedAttempt().ArtifactDigest()
		worldDigest := trial.Result.World().Digest()
		if _, duplicate := seenAttempts[attemptDigest]; duplicate {
			t.Fatalf("reused attempt digest %s", attemptDigest)
		}
		if _, duplicate := seenWorlds[worldDigest]; duplicate {
			t.Fatalf("reused world digest %s", worldDigest)
		}
		seenAttempts[attemptDigest] = struct{}{}
		seenWorlds[worldDigest] = struct{}{}

		process := trial.Result.Process()
		if process.ExecutionAuthorityMarker() != world.CLIExecutionAuthorityV1 || !process.PhysicalExecutionEntered() ||
			process.ExecutionBindingDigest() != result.Binding.Digest() ||
			process.StimulusDigest() != result.Stimulus.Digest() ||
			!slices.Equal(process.DeclaredLogicalArgv(), result.Binding.LogicalArgv()) ||
			!slices.Equal(process.LogicalArgv(), result.Binding.LogicalArgv()) ||
			process.InvocationEvidenceStatus() != world.CLIInvocationValidated ||
			process.InvocationEvidencePresence() != world.CLIInvocationPresencePresent ||
			!process.InvocationEvidencePresent() || !process.InvocationEvidenceValidated() {
			t.Fatalf("trial %d lacks strict CLI receipt lineage", trial.Slot.Ordinal())
		}
		stdin := result.Stimulus.Stdin()
		stdinDigest, digestErr := canon.DigestBytes("CLIStdinBytes", stdin.Bytes())
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		parsedStdinDigest, parseErr := domain.ParseDigest(stdinDigest.String())
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		if process.StdinPresence() != string(stdin.Presence()) ||
			process.StdinBytes() != int64(len(stdin.Bytes())) || process.StdinDigest() != parsedStdinDigest ||
			process.StdoutCaptureLimit() != result.CapturePolicy.StdoutBytes() ||
			process.StderrCaptureLimit() != result.CapturePolicy.StderrBytes() {
			t.Fatalf("trial %d did not receipt physical stdin/capture authorities", trial.Slot.Ordinal())
		}
		if stdin.Present() {
			if process.StdinDelivery() != "PRESENT_EXPLICIT_PIPE_WRITER" ||
				!process.StdinPipeAllocated() || !process.StdinWriterStarted() || !process.StdinHandoffAttempted() ||
				process.StdinWrittenBytes() != int64(len(stdin.Bytes())) || !process.StdinDeliveryComplete() ||
				process.StdinDeliveryErrorCode() != "" {
				t.Fatalf("trial %d lacks exact present stdin handoff evidence", trial.Slot.Ordinal())
			}
		} else if process.StdinDelivery() != "ABSENT_NULL_DEVICE" ||
			process.StdinPipeAllocated() || process.StdinWriterStarted() || process.StdinHandoffAttempted() ||
			process.StdinWrittenBytes() != 0 || process.StdinDeliveryComplete() || process.StdinDeliveryErrorCode() != "" {
			t.Fatalf("trial %d collapsed absent stdin into a present pipe", trial.Slot.Ordinal())
		}
		for _, environment := range process.Environment() {
			if environment == "HOME="+ambientHome || strings.Contains(environment, "must-not-cross") ||
				strings.HasPrefix(environment, "COUNTERSHAPE_AMBIENT_SENTINEL=") {
				t.Fatalf("ambient environment crossed into trial %d: %q", trial.Slot.Ordinal(), environment)
			}
		}
		overlay, overlayPresent := trial.Result.CLIFixtureOverlay()
		invocation, invocationPresent := trial.Result.CLIInvocationEvidence()
		if !overlayPresent || !overlay.Valid() || overlay.Root() != trial.Result.Roots().Fixture() ||
			!invocationPresent || !invocation.Valid() || invocation.Status() != world.CLIInvocationValidated ||
			!slices.Equal(invocation.ExpectedLogicalArgv(), result.Stimulus.LogicalArgv()) {
			t.Fatalf("trial %d lacks exact fixture/invocation receipts", trial.Slot.Ordinal())
		}
		if !trial.Observation.ProjectionEligible() || !trial.Projected ||
			len(trial.Projection.Operations()) == 0 || len(trial.Projection.SourceLinks()) == 0 ||
			!trial.Projection.Derivation().Valid() {
			t.Fatalf("trial %d lacks inspectable projection evidence", trial.Slot.Ordinal())
		}
		roleExpected, knownRole := expected[trial.Role]
		if !trial.Role.Valid() || !knownRole {
			t.Fatalf("trial %d has unknown immutable role %q", trial.Slot.Ordinal(), trial.Role)
		}
		for field, want := range roleExpected {
			if got, present := projectedString(trial.Projection, field); !present || got != want {
				t.Fatalf("role %s field %s=%q,%v want %q", trial.Role, field, got, present, want)
			}
		}
		if first, present := representativeProjection[trial.Role]; present {
			if !slices.Equal(first, trial.Projection.ProjectionBytes()) {
				t.Fatalf("role %s changed exact behavior projection across fresh trials", trial.Role)
			}
		} else {
			representativeProjection[trial.Role] = trial.Projection.ProjectionBytes()
		}
	}
	if len(result.RolesByFingerprint()) != 3 {
		t.Fatalf("display role-to-fingerprint helper lost a candidate")
	}
}

func TestCLIStudyUsesStrictSourceCompilerAndPlanBoundSchedule(t *testing.T) {
	result, err := Run(context.Background(), behaviorConfig(t, BehaviorPrecedence, 1))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := spec.ParseSource(result.SourceSpecBytes)
	if err != nil {
		t.Fatalf("study did not retain an admitted inert SourceSpec: %v", err)
	}
	compiled, err := spec.Compile(parsed, result.ProjectionDefinition.Binding())
	if err != nil {
		t.Fatalf("study SourceSpec did not resolve through the strict compiler: %v", err)
	}
	if parsed.Digest() != result.SourceSpecDigest || compiled.Digest() != result.Plan.Digest() ||
		!bytes.Equal(compiled.CanonicalBytes(), result.Plan.CanonicalBytes()) {
		t.Fatal("study plan is not the exact product of its retained strict SourceSpec")
	}
	schedule := result.Observation.Schedule()
	if schedule.Rotation() != result.Plan.ScheduleRotation() ||
		schedule.CandidateCount() != result.Plan.Budgets().CandidateCount ||
		schedule.Repetitions() != result.Plan.RepeatSchedule().DiscoveryRepeats {
		t.Fatal("physical observation schedule escaped the compiled plan contract")
	}
	otherProjection, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{
		Fields: []cli.CLIFieldID{cli.CLIFieldStderrText},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := spec.Compile(parsed, otherProjection.Binding()); err == nil {
		t.Fatal("study SourceSpec resolved with a different projection capability")
	}
	unknown := append([]byte(nil), result.SourceSpecBytes[:len(result.SourceSpecBytes)-1]...)
	unknown = append(unknown, []byte(`,"unknown":true}`)...)
	if _, err := spec.ParseSource(unknown); err == nil {
		t.Fatal("study source route admitted an unknown field")
	}
}

func TestOptionalReceiptMeasurementKeepsAbsenceExplicit(t *testing.T) {
	absent, err := canonicalOptionalReceipt(false, "", string(world.CLIInvocationNotInspected), string(world.CLIInvocationPresenceUnknown))
	if err != nil {
		t.Fatal(err)
	}
	absentBytes, err := absent.CanonicalChecked()
	if err != nil {
		t.Fatal(err)
	}
	wantAbsent := `{"digest":"","physical_presence":"UNKNOWN","presence":"ABSENT","status":"NOT_INSPECTED_PRE_PROCESS"}`
	if string(absentBytes) != wantAbsent {
		t.Fatalf("absent optional receipt measurement = %s, want %s", absentBytes, wantAbsent)
	}
	digest, digestErr := canon.DigestBytes("OptionalReceiptTest", []byte("present"))
	if digestErr != nil {
		t.Fatal(digestErr)
	}
	parsed, parseErr := domain.ParseDigest(digest.String())
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	present, err := canonicalOptionalReceipt(true, parsed, string(world.CLIInvocationValidated), string(world.CLIInvocationPresencePresent))
	if err != nil {
		t.Fatal(err)
	}
	presentBytes, err := present.CanonicalChecked()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(presentBytes, []byte(`"presence":"PRESENT"`)) ||
		!bytes.Contains(presentBytes, []byte(`"physical_presence":"PRESENT"`)) ||
		!bytes.Contains(presentBytes, []byte(parsed.String())) {
		t.Fatalf("present optional receipt measurement lost authority: %s", presentBytes)
	}
	if _, err := canonicalOptionalReceipt(false, parsed, "", ""); err == nil {
		t.Fatal("absent optional receipt accepted a digest")
	}
}

func TestCLIPresentEmptyStdinCrossesWorldAndAdapter(t *testing.T) {
	assertCLIPresentStdinCrossesWorldAndAdapter(t, []byte{})
}

func TestCLIPresentBytesStdinCrossesWorldAndAdapter(t *testing.T) {
	assertCLIPresentStdinCrossesWorldAndAdapter(t, []byte("opaque study stdin"))
}

func assertCLIPresentStdinCrossesWorldAndAdapter(t *testing.T, input []byte) {
	t.Helper()
	config := behaviorConfig(t, BehaviorPrecedence, 1)
	config.StdinPresent = true
	config.StdinBytes = append([]byte(nil), input...)
	result, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	assertAllBatchStatus(t, result, observe.ObservedStable)
	wantDigest, err := canon.DigestBytes("CLIStdinBytes", input)
	if err != nil {
		t.Fatal(err)
	}
	parsedDigest, err := domain.ParseDigest(wantDigest.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trials) != 3 {
		t.Fatalf("present stdin trial count=%d, want 3", len(result.Trials))
	}
	for _, trial := range result.Trials {
		process := trial.Result.Process()
		if !trial.Admitted || !trial.Projected || !trial.Observation.ProjectionEligible() ||
			!process.PhysicalExecutionEntered() ||
			!slices.Equal(process.DeclaredLogicalArgv(), result.Binding.LogicalArgv()) ||
			!slices.Equal(process.LogicalArgv(), result.Binding.LogicalArgv()) ||
			process.StdinPresence() != "PRESENT" || process.StdinBytes() != int64(len(input)) ||
			process.StdinDigest() != parsedDigest || process.StdinDelivery() != "PRESENT_EXPLICIT_PIPE_WRITER" ||
			!process.StdinPipeAllocated() || !process.StdinWriterStarted() || !process.StdinHandoffAttempted() ||
			process.StdinWrittenBytes() != int64(len(input)) || !process.StdinDeliveryComplete() ||
			process.StdinDeliveryErrorCode() != "" ||
			process.StdoutCaptureLimit() != result.CapturePolicy.StdoutBytes() ||
			process.StderrCaptureLimit() != result.CapturePolicy.StderrBytes() {
			t.Fatalf("present stdin did not cross the exact physical/adapter seam: %+v", process)
		}
	}
}

func TestCLIMaterializationControlCrossesAdapterAndExcludedMap(t *testing.T) {
	result, err := Run(context.Background(), behaviorConfig(t, BehaviorMaterializeFailure, 1))
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasOutcomeMap || len(result.Observation.Batches()) != 3 ||
		len(result.OutcomeMap.Entries()) != 2 || len(result.OutcomeMap.Exclusions()) != 1 ||
		result.OutcomeMap.Exclusions()[0].Classification != observe.Uncomparable || len(result.Trials) != 3 {
		t.Fatalf("materialization-control trial count=%d, want 3", len(result.Trials))
	}
	controlled := 0
	var controlledKey domain.CandidateExecutionKey
	for _, trial := range result.Trials {
		process := trial.Result.Process()
		primary, hasPrimary := trial.Result.FinalizedAttempt().PrimaryControl()
		controls := trial.Observation.Controls()
		if process.PhysicalExecutionEntered() {
			if !trial.Admitted || !trial.Projected || !trial.Observation.ProjectionEligible() || hasPrimary || len(controls) != 0 {
				t.Fatalf("unaffected peer did not remain exact behavior: %+v", process)
			}
			continue
		}
		controlled++
		controlledKey = trial.Slot.CandidateKey()
		if !trial.Admitted || trial.Projected || trial.ProjectionRejection != nil ||
			!trial.Observation.Valid() || trial.Observation.ProjectionEligible() || !hasPrimary ||
			primary != domain.ControlMissingObject || len(controls) != 1 ||
			controls[0] != domain.ControlMissingObject || len(process.LogicalArgv()) != 0 ||
			!slices.Equal(process.DeclaredLogicalArgv(), result.Binding.LogicalArgv()) ||
			process.StdoutCaptureLimit() != 0 || process.StderrCaptureLimit() != 0 ||
			process.StdinDelivery() != "NOT_APPLIED" || process.StdinPipeAllocated() ||
			process.StdinWriterStarted() || process.StdinHandoffAttempted() ||
			process.StdinWrittenBytes() != 0 || process.StdinDeliveryComplete() ||
			process.InvocationEvidenceStatus() != world.CLIInvocationNotInspected ||
			process.InvocationEvidencePresence() != world.CLIInvocationPresenceUnknown {
			t.Fatalf("pre-process control conflated declaration with physical execution: %+v", process)
		}
	}
	if controlled != 1 || result.OutcomeMap.Exclusions()[0].CandidateKey != controlledKey {
		t.Fatalf("materialization fault controlled %d candidates, want exactly 1", controlled)
	}
}

func TestCLIReferenceDisplayPermutationPreservesMapWithFreshAttempts(t *testing.T) {
	firstConfig := referenceConfig(t)
	firstConfig.DisplayLabels = map[clifixture.CandidateRole]string{
		clifixture.ConfigFirst:      "candidate one",
		clifixture.EnvironmentFirst: "candidate two",
		clifixture.ArgvFirst:        "candidate three",
	}
	firstConfig.ProducerMetadata = "producer-a"
	firstAmbientHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(firstAmbientHome, ".countershape-precedence.json"), []byte(`{"mode":"ambient-first"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", firstAmbientHome)
	first, err := Run(context.Background(), firstConfig)
	if err != nil {
		t.Fatal(err)
	}

	secondConfig := referenceConfig(t)
	secondConfig.CandidateOrder = []clifixture.CandidateRole{
		clifixture.ArgvFirst,
		clifixture.EnvironmentFirst,
		clifixture.ConfigFirst,
	}
	secondConfig.DisplayLabels = map[clifixture.CandidateRole]string{
		clifixture.ConfigFirst:      "renamed z",
		clifixture.EnvironmentFirst: "renamed y",
		clifixture.ArgvFirst:        "renamed x",
	}
	secondConfig.ProducerMetadata = "producer-b"
	secondAmbientHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(secondAmbientHome, ".countershape-precedence.json"), []byte(`{"mode":"ambient-second"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", secondAmbientHome)
	second, err := Run(context.Background(), secondConfig)
	if err != nil {
		t.Fatal(err)
	}
	if first.OutcomeMap.PreservationDigest() != second.OutcomeMap.PreservationDigest() {
		t.Fatalf("candidate order/display metadata changed exact preservation map: %s != %s",
			first.OutcomeMap.PreservationDigest().String(), second.OutcomeMap.PreservationDigest().String())
	}
	if first.Plan.Digest() != second.Plan.Digest() || first.Binding.Digest() != second.Binding.Digest() ||
		first.ProjectionDefinition.Digest() != second.ProjectionDefinition.Digest() ||
		first.Observation.Schedule().Digest() != second.Observation.Schedule().Digest() ||
		first.OutcomeMap.ScheduleDigest() != second.OutcomeMap.ScheduleDigest() {
		t.Fatal("display order or labels leaked into plan, binding, projection, or schedule authority")
	}
	if !sameOutcomeEntries(first.OutcomeMap.Entries(), second.OutcomeMap.Entries()) {
		t.Fatal("display permutation changed the exact candidate-key-to-fingerprint map")
	}
	if first.OutcomeMap.ArtifactDigest() == second.OutcomeMap.ArtifactDigest() {
		t.Fatal("fresh execution reused the complete evidence artifact digest")
	}
	if !slices.Equal(first.CanonicalCandidateRoster(), second.CanonicalCandidateRoster()) {
		t.Fatal("candidate permutation changed the canonical opaque roster")
	}
	assertDigestSetsDisjoint(t, "attempt", first.OutcomeMap.EvidenceAttemptDigests(), second.OutcomeMap.EvidenceAttemptDigests())
	assertDigestSetsDisjoint(t, "world", first.OutcomeMap.EvidenceWorldDigests(), second.OutcomeMap.EvidenceWorldDigests())
	assertDigestSetsDisjoint(t, "observation", first.OutcomeMap.EvidenceObservationDigests(), second.OutcomeMap.EvidenceObservationDigests())
	for role, firstProjection := range representativeProjections(first) {
		secondProjection, present := representativeProjections(second)[role]
		if !present || !slices.Equal(firstProjection, secondProjection) {
			t.Fatalf("role %s changed behavior projection under display permutation", role)
		}
	}
}

func TestCLIReferenceControlClassificationMatrix(t *testing.T) {
	t.Run("nonzero-is-eligible", func(t *testing.T) {
		config := behaviorConfig(t, BehaviorNonzeroExit, 2)
		result, err := Run(context.Background(), config)
		if err != nil {
			t.Fatal(err)
		}
		assertAllBatchStatus(t, result, observe.ObservedStable)
		for _, trial := range result.Trials {
			completion, present := trial.Observation.Completion()
			code, exited := completion.ExitCode()
			if !present || !exited || code != 7 || !trial.Observation.ProjectionEligible() || !trial.Projected {
				t.Fatalf("nonzero trial %d was not eligible behavior", trial.Slot.Ordinal())
			}
		}
	})

	t.Run("alternating-uses-all-trials", func(t *testing.T) {
		config := behaviorConfig(t, BehaviorAlternating, 2)
		result, err := Run(context.Background(), config)
		if err != nil {
			t.Fatal(err)
		}
		assertAllBatchStatus(t, result, observe.Unstable)
		assertExcludedOutcomeMap(t, result, observe.Unstable)
		for _, batch := range result.Observation.Batches() {
			histogram := batch.Classification().Histogram()
			if len(histogram) != 2 || histogram[0].Count != 1 || histogram[1].Count != 1 {
				t.Fatalf("alternating histogram=%#v", histogram)
			}
		}
	})

	for _, controlled := range []struct {
		name     string
		behavior Behavior
		reason   domain.ControlReason
	}{
		{"timeout", BehaviorTimeout, domain.ControlTimeout},
		{"output-limit", BehaviorOutputLimit, domain.ControlOutputLimit},
	} {
		controlled := controlled
		t.Run(controlled.name, func(t *testing.T) {
			result, err := Run(context.Background(), behaviorConfig(t, controlled.behavior, 1))
			if err != nil {
				t.Fatal(err)
			}
			assertAllBatchStatus(t, result, observe.Uncomparable)
			assertExcludedOutcomeMap(t, result, observe.Uncomparable)
			for _, trial := range result.Trials {
				primary, present := trial.Result.FinalizedAttempt().PrimaryControl()
				if !present || primary != controlled.reason || trial.Projected || trial.ProjectionRejection != nil ||
					trial.Observation.ProjectionEligible() {
					t.Fatalf("%s trial %d control=%s,%v projected=%v", controlled.name, trial.Slot.Ordinal(), primary, present, trial.Projected)
				}
				if _, projectErr := result.ProjectionDefinition.Project(trial.Observation); projectErr == nil {
					t.Fatalf("%s trial %d projected when exercised directly", controlled.name, trial.Slot.Ordinal())
				}
			}
		})
	}

	for _, rejected := range []struct {
		name             string
		behavior         Behavior
		invocationStatus world.CLIInvocationEvidenceStatus
		filePresence     world.CLIInvocationEvidencePresence
		captureEligible  bool
	}{
		{"malformed-projection", BehaviorMalformedProjection, world.CLIInvocationValidated, world.CLIInvocationPresencePresent, true},
		{"empty-strict-json", BehaviorEmptyOutput, world.CLIInvocationValidated, world.CLIInvocationPresencePresent, true},
		{"missing-invocation", BehaviorMissingInvocation, world.CLIInvocationAbsent, world.CLIInvocationPresenceAbsent, false},
		{"malformed-invocation", BehaviorMalformedInvocation, world.CLIInvocationMalformed, world.CLIInvocationPresencePresent, false},
	} {
		rejected := rejected
		t.Run(rejected.name, func(t *testing.T) {
			result, err := Run(context.Background(), behaviorConfig(t, rejected.behavior, 1))
			if err != nil {
				t.Fatal(err)
			}
			assertAllBatchStatus(t, result, observe.Uncomparable)
			assertExcludedOutcomeMap(t, result, observe.Uncomparable)
			for _, trial := range result.Trials {
				invocation := trial.Observation.InvocationReceipt()
				if trial.Result.FinalizedAttempt().HasControls() || trial.Projected || trial.ProjectionRejection == nil ||
					trial.Observation.ProjectionEligible() != rejected.captureEligible ||
					trial.Result.Process().InvocationEvidenceStatus() != rejected.invocationStatus ||
					trial.Result.Process().InvocationEvidencePresence() != rejected.filePresence ||
					invocation.Status() != rejected.invocationStatus || invocation.FilePresence() != rejected.filePresence {
					t.Fatalf("%s trial %d did not retain typed post-capture rejection", rejected.name, trial.Slot.Ordinal())
				}
			}
		})
	}

	t.Run("signal-is-disjoint-completion", func(t *testing.T) {
		config := behaviorConfig(t, BehaviorSignal, 1)
		config.ProjectionFields = []cli.CLIFieldID{cli.CLIFieldCompletionKind, cli.CLIFieldExitSignal}
		result, err := Run(context.Background(), config)
		if err != nil {
			t.Fatal(err)
		}
		assertAllBatchStatus(t, result, observe.ObservedStable)
		for _, trial := range result.Trials {
			completion, present := trial.Observation.Completion()
			signal, signaled := completion.Signal()
			if !present || !signaled || signal == "" || trial.Result.FinalizedAttempt().HasControls() || !trial.Projected {
				t.Fatalf("signal trial %d collapsed completion: %q,%v", trial.Slot.Ordinal(), signal, signaled)
			}
		}
	})

	t.Run("incomplete-budget-stops-before-partial-matrix", func(t *testing.T) {
		config := behaviorConfig(t, BehaviorPrecedence, 2)
		config.MaxTotalTrials = 3
		result, err := Run(context.Background(), config)
		if err != nil {
			t.Fatal(err)
		}
		if result.Observation.Status() != observe.ObservationIncomplete || result.Observation.CompletedMatrices() != 1 || len(result.Trials) != 3 {
			t.Fatalf("incomplete run status=%s matrices=%d trials=%d", result.Observation.Status(), result.Observation.CompletedMatrices(), len(result.Trials))
		}
		assertAllBatchStatus(t, result, observe.Incomplete)
		assertExcludedOutcomeMap(t, result, observe.Incomplete)
		for _, trial := range result.Trials {
			if !trial.Admitted {
				t.Fatalf("whole-matrix budget stop exposed partial trial %d", trial.Slot.Ordinal())
			}
		}
	})
}

func referenceConfig(t *testing.T) Config {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git, err := filepath.EvalSymlinks("/usr/bin/git")
	if err != nil {
		t.Fatal(err)
	}
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is not installed")
	}
	node, err := filepath.EvalSymlinks(nodePath)
	if err != nil {
		t.Fatal(err)
	}
	return DefaultConfig(root, git, node)
}

func behaviorConfig(t *testing.T, behavior Behavior, repetitions int) Config {
	t.Helper()
	config := referenceConfig(t)
	config.Behavior = behavior
	config.Repetitions = repetitions
	config.MaxTotalTrials = repetitions * 3
	return config
}

func projectedString(result cli.CLIProjectionResult, field cli.CLIFieldID) (string, bool) {
	for _, projected := range result.Fields() {
		if projected.ID() == field {
			return projected.Value().String()
		}
	}
	return "", false
}

func assertAllBatchStatus(t *testing.T, result StudyResult, want observe.BatchStatus) {
	t.Helper()
	if len(result.Observation.Batches()) != 3 {
		t.Fatalf("batch count=%d, want 3", len(result.Observation.Batches()))
	}
	for _, batch := range result.Observation.Batches() {
		if batch.Classification().Status() != want {
			t.Fatalf("candidate %s status=%s, want %s", batch.CandidateKey(), batch.Classification().Status(), want)
		}
	}
}

func assertExcludedOutcomeMap(t *testing.T, result StudyResult, want observe.BatchStatus) {
	t.Helper()
	if !result.HasOutcomeMap || len(result.OutcomeMap.Entries()) != 0 ||
		len(result.OutcomeMap.Exclusions()) != 3 || result.OutcomeMap.DistinctProjectionCount() != 0 ||
		result.OutcomeMap.Divergence() {
		t.Fatalf("excluded map status=%s has map=%v entries=%d exclusions=%d distinct=%d divergence=%v",
			want, result.HasOutcomeMap, len(result.OutcomeMap.Entries()), len(result.OutcomeMap.Exclusions()),
			result.OutcomeMap.DistinctProjectionCount(), result.OutcomeMap.Divergence())
	}
	seen := make(map[domain.CandidateExecutionKey]struct{}, 3)
	for _, exclusion := range result.OutcomeMap.Exclusions() {
		if exclusion.Classification != want {
			t.Fatalf("candidate %s exclusion=%s, want %s", exclusion.CandidateKey, exclusion.Classification, want)
		}
		if _, duplicate := seen[exclusion.CandidateKey]; duplicate {
			t.Fatalf("duplicate outcome-map exclusion for %s", exclusion.CandidateKey)
		}
		seen[exclusion.CandidateKey] = struct{}{}
	}
	if _, err := compare.RequireDivergence(result.OutcomeMap); err == nil {
		t.Fatalf("%s exclusion map granted divergent-baseline authority", want)
	}
}

func assertFreshOwnedRoots(t *testing.T, trial TrialEvidence, seenByKind map[string]map[string]struct{}) {
	t.Helper()
	roots := trial.Result.Roots()
	attemptRoot := roots.Attempt()
	paths := map[string]string{
		"attempt": attemptRoot, "candidate-parent": roots.CandidateParent(), "fixture": roots.Fixture(),
		"home": roots.Home(), "temporary": roots.Temporary(), "xdg-config": roots.XDGConfig(),
		"xdg-cache": roots.XDGCache(), "xdg-data": roots.XDGData(), "xdg-state": roots.XDGState(),
		"state": roots.State(), "evidence": roots.Evidence(), "working-directory": trial.Result.Process().WorkingDirectory(),
	}
	seenWithinAttempt := make(map[string]string, len(paths))
	for kind, path := range paths {
		if path == "" {
			t.Fatalf("trial %d has empty %s root", trial.Slot.Ordinal(), kind)
		}
		if seenByKind[kind] == nil {
			seenByKind[kind] = make(map[string]struct{})
		}
		if _, duplicate := seenByKind[kind][path]; duplicate {
			t.Fatalf("trial %d reused physical %s root %q", trial.Slot.Ordinal(), kind, path)
		}
		seenByKind[kind][path] = struct{}{}
		if priorKind, duplicate := seenWithinAttempt[path]; duplicate {
			t.Fatalf("trial %d aliases physical %s and %s roots at %q", trial.Slot.Ordinal(), priorKind, kind, path)
		}
		seenWithinAttempt[path] = kind
		if kind == "attempt" {
			continue
		}
		relative, err := filepath.Rel(attemptRoot, path)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			t.Fatalf("trial %d %s root escaped attempt root: %q relative=%q err=%v", trial.Slot.Ordinal(), kind, path, relative, err)
		}
	}
}

func sameOutcomeEntries(left, right []compare.Entry) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].CandidateKey != right[index].CandidateKey ||
			left[index].ProjectionFingerprint != right[index].ProjectionFingerprint {
			return false
		}
	}
	return true
}

func representativeProjections(result StudyResult) map[clifixture.CandidateRole][]byte {
	projections := make(map[clifixture.CandidateRole][]byte)
	for _, trial := range result.Trials {
		if trial.Projected {
			if _, present := projections[trial.Role]; !present {
				projections[trial.Role] = trial.Projection.ProjectionBytes()
			}
		}
	}
	return projections
}

func assertDigestSetsDisjoint(t *testing.T, kind string, left, right []domain.Digest) {
	t.Helper()
	seen := make(map[domain.Digest]struct{}, len(left))
	for _, digest := range left {
		seen[digest] = struct{}{}
	}
	for _, digest := range right {
		if _, reused := seen[digest]; reused {
			t.Fatalf("fresh runs reused %s digest %s", kind, digest)
		}
	}
}
