package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
)

func testDigest(character string) Digest {
	return MustDigest("sha256:" + strings.Repeat(character, 64))
}

func testProjectionDefinitionBinding(adapter AdapterDomain) ProjectionDefinitionBinding {
	channels := []string{"http.body", "http.headers", "http.status"}
	if adapter == AdapterCLI {
		channels = []string{"exit", "stderr", "stdout"}
	}
	binding, err := NewProjectionDefinitionBinding(ProjectionDefinitionBindingConfig{
		AdapterDomain: adapter, ImplementationDigest: testDigest("7"), ConfigurationDigest: testDigest("8"),
		AcceptedChannels: channels,
		Operations:       []ProjectionOperationBinding{{Name: "decode", RuleDigest: testDigest("9")}, {Name: "select", RuleDigest: testDigest("a")}},
		Comparator:       ProjectionComparatorExact, FieldRegistryDigest: testDigest("b"),
	})
	if err != nil {
		panic(err)
	}
	return binding
}

func TestAdapterDomainOwnsStableCanonicalStimulusKinds(t *testing.T) {
	if AdapterCLI.CanonicalStimulusKind() != "CLIStimulus" ||
		AdapterHTTP.CanonicalStimulusKind() != "HTTPStimulus" ||
		AdapterDomain("UNKNOWN").CanonicalStimulusKind() != "" {
		t.Fatal("adapter-domain canonical stimulus mapping drifted")
	}
}

func validPlanConfig() WorldPlanConfig {
	return WorldPlanConfig{
		CandidateSetDigest:          testDigest("1"),
		MaterializationPolicyDigest: testDigest("2"),
		ComparisonEnvelopeDigest:    testDigest("3"),
		Adapter:                     Adapter{Domain: AdapterHTTP, AdapterVersion: "http/v1", RunnerDigest: testDigest("4")},
		ExecutionShape:              OneLoopbackHTTPRequest,
		StartArgv:                   []string{"node", "fixture/server.mjs"},
		SetupArgv:                   []string{},
		Environment:                 []EnvironmentEntry{{Name: "NODE_NO_WARNINGS", Value: "1"}, {Name: "LANG", Value: "C"}},
		SecretSlots:                 []SecretSlot{},
		FixtureRecipeDigest:         testDigest("5"),
		Readiness:                   Readiness{Kind: FixtureOwnedReadiness, SignalName: "ready-byte"},
		CapturePolicyDigest:         testDigest("6"),
		ProjectionDefinition:        testProjectionDefinitionBinding(AdapterHTTP),
		RepeatSchedule: RepeatSchedule{DiscoveryRepeats: 3, ConfirmationRepeats: 3,
			Concurrency: ScheduleSequential, Rotation: ScheduleRotationStartByRepetitionV1},
		RequiredTools: []RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets: Budgets{
			CandidateCount:            4,
			MaterializedEntryCount:    25000,
			MaterializedBytesPerWorld: 268435456,
			SingleBlobBytes:           33554432,
			ReadinessMS:               5000,
			ProbeMS:                   3000,
			TeardownMS:                3000,
			StdoutBytes:               1048576,
			StderrBytes:               1048576,
			HTTPBodyBytes:             1048576,
			ProposedShrinkStimuli:     40,
			TotalCandidateTrials:      300,
			ShrinkWallMS:              600000,
		},
	}
}

func validEnvelopeConfig() ComparisonEnvelopeConfig {
	return ComparisonEnvelopeConfig{
		Version: "test/v1",
		Measured: []MeasuredDimension{
			{Name: "operating system", Source: MeasuredWorldInstance, Comparison: CompareExact},
			{Name: "temporary root", Source: MeasuredFilesystemProbe, Comparison: CompareRecordedOnly},
			{Name: "tool version", Source: MeasuredToolReceipt, Comparison: CompareRejectOnVariance},
		},
		RequiredEqual: []RequiredEqualDimension{{Name: "operating system"}},
		Tolerated:     []ToleratedDimension{{Name: "temporary root", Tolerance: ProjectedCapturePlaceholder}},
		Rejected:      []RejectedDimension{{Name: "tool version", ReasonCode: "TOOL_VARIANCE"}},
		Uncontrolled:  []string{"scheduler timing"},
	}
}

func testCandidate() CandidateExecutionKey {
	key, err := NewCandidateExecutionKey(CandidateExecutionIdentity{
		TreeIdentityDigest:          testDigest("1"),
		MaterializationPolicyDigest: testDigest("2"),
		WorldPlanDigest:             testDigest("3"),
		AdapterDigest:               testDigest("4"),
		RunnerDigest:                testDigest("5"),
		ProjectionDefinitionDigest:  testDigest("6"),
	})
	if err != nil {
		panic(err)
	}
	return key
}

type measuredWorldFixture struct {
	world        WorldInstance
	measurements InstanceMeasurements
}

func measurementRow(t *testing.T, envelope ComparisonEnvelope, subject Digest, overrides map[string]string) measuredWorldFixture {
	t.Helper()
	planConfig := validPlanConfig()
	planConfig.ComparisonEnvelopeDigest = envelope.Digest()
	plan, err := NewWorldPlan(planConfig)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := NewCandidateExecutionBinding(CandidateExecutionIdentity{
		TreeIdentityDigest: subject, MaterializationPolicyDigest: plan.MaterializationPolicyDigest(),
		WorldPlanDigest: plan.Digest(), AdapterDigest: plan.AdapterDigest(), RunnerDigest: plan.Adapter().RunnerDigest,
		ProjectionDefinitionDigest: plan.ProjectionDefinitionDigest(),
	})
	if err != nil {
		t.Fatal(err)
	}
	world, err := NewWorldInstance(plan, binding, WorldInstanceConfig{
		StimulusDigest: testDigest("e"), AttemptArtifactDigest: subject,
		Purpose: AttemptDiscovery, InstanceNonce: "nonce:" + subject.String(),
		ScheduleOrdinal: int(subject.String()[len("sha256:")]),
	})
	if err != nil {
		t.Fatal(err)
	}
	values := make([]MeasurementValue, 0, len(envelope.Config().Measured))
	for _, dimension := range envelope.Config().Measured {
		text := "fixed:" + dimension.Name
		if override, exists := overrides[dimension.Name]; exists {
			text = override
		}
		value, err := canon.String(text)
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, MeasurementValue{Name: dimension.Name, Source: dimension.Source, Value: value})
	}
	measurements, err := NewInstanceMeasurements(envelope, world, values)
	if err != nil {
		t.Fatal(err)
	}
	return measuredWorldFixture{world: world, measurements: measurements}
}

func TestWorldPlanDeterministicAndImmutable(t *testing.T) {
	config := validPlanConfig()
	plan, err := NewWorldPlan(config)
	if err != nil {
		t.Fatal(err)
	}
	originalBytes := plan.CanonicalBytes()
	originalDigest := plan.Digest()

	config.StartArgv[0] = "mutated"
	config.Environment[0].Value = "mutated"
	returnedArgv := plan.StartArgv()
	returnedArgv[0] = "also-mutated"
	returnedEnvironment := plan.Environment()
	returnedEnvironment[0].Value = "also-mutated"
	if plan.StartArgv()[0] != "node" || !bytes.Equal(plan.CanonicalBytes(), originalBytes) || plan.Digest() != originalDigest {
		t.Fatal("source or accessor slice mutation changed immutable plan")
	}

	reordered := validPlanConfig()
	reordered.Environment[0], reordered.Environment[1] = reordered.Environment[1], reordered.Environment[0]
	reorderedPlan, err := NewWorldPlan(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if reorderedPlan.Digest() != originalDigest {
		t.Fatal("declared-unordered environment changed plan identity")
	}

	argvChanged := validPlanConfig()
	argvChanged.StartArgv[1] = "fixture/other-server.mjs"
	changedPlan, err := NewWorldPlan(argvChanged)
	if err != nil {
		t.Fatal(err)
	}
	if changedPlan.Digest() == originalDigest {
		t.Fatal("ordered argv did not change plan identity")
	}
}

func TestWorldPlanRetainsDeclaredSequentialRotationMutationGuard(t *testing.T) {
	plan, err := NewWorldPlan(validPlanConfig())
	if err != nil {
		t.Fatal(err)
	}
	canonical := plan.CanonicalBytes()
	if plan.ScheduleConcurrency() != ScheduleSequential ||
		plan.ScheduleRotation() != ScheduleRotationStartByRepetitionV1 ||
		!bytes.Contains(canonical, []byte(`"concurrency":"SEQUENTIAL"`)) ||
		!bytes.Contains(canonical, []byte(`"rotation":"ROTATE_START_BY_REPETITION_V1"`)) ||
		bytes.Contains(canonical, []byte("NOT_ESTABLISHED_IN_U1")) {
		t.Fatalf("world plan lost the declared schedule policy: %s", canonical)
	}
	invalid := validPlanConfig()
	invalid.RepeatSchedule.Rotation = "NOT_ESTABLISHED_IN_U1"
	if _, err := NewWorldPlan(invalid); err == nil {
		t.Fatal("world plan accepted a schedule outside the closed declared policy")
	}
}

func TestProjectionDefinitionBindingPreservesPipelineOrderAndNormalizesChannels(t *testing.T) {
	base := ProjectionDefinitionBindingConfig{
		AdapterDomain: AdapterHTTP, ImplementationDigest: testDigest("1"), ConfigurationDigest: testDigest("2"),
		AcceptedChannels: []string{"http.status", "http.body", "http.headers"},
		Operations:       []ProjectionOperationBinding{{Name: "decode", RuleDigest: testDigest("3")}, {Name: "select", RuleDigest: testDigest("4")}},
		Comparator:       ProjectionComparatorExact, FieldRegistryDigest: testDigest("5"),
	}
	first, err := NewProjectionDefinitionBinding(base)
	if err != nil {
		t.Fatal(err)
	}
	reorderedChannels := base
	reorderedChannels.AcceptedChannels = []string{"http.body", "http.headers", "http.status"}
	second, err := NewProjectionDefinitionBinding(reorderedChannels)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() != second.Digest() || !first.Valid() || !second.Valid() {
		t.Fatal("accepted-channel set order changed the full definition identity")
	}
	reorderedOperations := base
	reorderedOperations.Operations = []ProjectionOperationBinding{base.Operations[1], base.Operations[0]}
	third, err := NewProjectionDefinitionBinding(reorderedOperations)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() == third.Digest() {
		t.Fatal("ordered projection pipeline collapsed into a set identity")
	}
}

func TestStrictAuthorityWireParsersRoundTripAndRejectUnknownMembers(t *testing.T) {
	config := validPlanConfig()
	plan, err := NewWorldPlan(config)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := ParseProjectionDefinitionBinding(config.ProjectionDefinition.CanonicalBytes())
	if err != nil || projection.Digest() != config.ProjectionDefinition.Digest() {
		t.Fatalf("projection binding round trip: %v", err)
	}
	parsedPlan, err := ParseWorldPlan(plan.CanonicalBytes(), projection)
	if err != nil || parsedPlan.Digest() != plan.Digest() {
		t.Fatalf("world plan round trip: %v", err)
	}
	binding, err := NewCandidateExecutionBinding(CandidateExecutionIdentity{
		TreeIdentityDigest: testDigest("c"), MaterializationPolicyDigest: plan.MaterializationPolicyDigest(),
		WorldPlanDigest: plan.Digest(), AdapterDigest: plan.AdapterDigest(), RunnerDigest: plan.Adapter().RunnerDigest,
		ProjectionDefinitionDigest: plan.ProjectionDefinitionDigest(),
	})
	if err != nil {
		t.Fatal(err)
	}
	parsedBinding, err := ParseCandidateExecutionBinding(binding.CanonicalBytes())
	if err != nil || parsedBinding.Key() != binding.Key() {
		t.Fatalf("candidate binding round trip: %v", err)
	}

	for name, exact := range map[string][]byte{
		"projection": config.ProjectionDefinition.CanonicalBytes(),
		"plan":       plan.CanonicalBytes(),
		"candidate":  binding.CanonicalBytes(),
	} {
		var identity map[string]any
		if err := json.Unmarshal(exact, &identity); err != nil {
			t.Fatal(err)
		}
		identity["unknown_member"] = "must refuse"
		encoded, err := json.Marshal(identity)
		if err != nil {
			t.Fatal(err)
		}
		unknown, err := canon.Canonicalize(encoded)
		if err != nil {
			t.Fatal(err)
		}
		accepted := false
		switch name {
		case "projection":
			_, err = ParseProjectionDefinitionBinding(unknown)
		case "plan":
			_, err = ParseWorldPlan(unknown, projection)
		case "candidate":
			_, err = ParseCandidateExecutionBinding(unknown)
		}
		accepted = err == nil
		if accepted {
			t.Fatalf("%s parser accepted an unknown member", name)
		}
	}
}

func TestWorldPlanRequiresMatchingProjectionDefinitionCapability(t *testing.T) {
	zero := validPlanConfig()
	zero.ProjectionDefinition = ProjectionDefinitionBinding{}
	if _, err := NewWorldPlan(zero); err == nil {
		t.Fatal("bare or zero projection reference produced an executable world plan")
	}
	mismatch := validPlanConfig()
	mismatch.ProjectionDefinition = testProjectionDefinitionBinding(AdapterCLI)
	if _, err := NewWorldPlan(mismatch); err == nil {
		t.Fatal("CLI projection definition was relabeled into an HTTP world plan")
	}
}

func TestWorldPlanRejectsInformationLossAndPolicyBypasses(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*WorldPlanConfig)
		code   string
	}{
		{"direct-shell", func(c *WorldPlanConfig) { c.StartArgv = []string{"/bin/sh", "-c", "echo ok"} }, ErrShellString},
		{"env-shell", func(c *WorldPlanConfig) { c.StartArgv = []string{"/usr/bin/env", "-u", "X", "bash", "-c", "echo ok"} }, ErrShellString},
		{"ambient", func(c *WorldPlanConfig) { c.Environment = []EnvironmentEntry{{Name: "LANG", Value: "$HOME"}} }, ErrAmbientInterpolation},
		{"undeclared-env", func(c *WorldPlanConfig) { c.Environment = []EnvironmentEntry{{Name: "ARBITRARY", Value: "public?"}} }, ErrInvalidWorldPlan},
		{"secret", func(c *WorldPlanConfig) { c.Environment = []EnvironmentEntry{{Name: "API_TOKEN", Value: "value"}} }, ErrInvalidWorldPlan},
		{"split-output", func(c *WorldPlanConfig) { c.StartArgv = []string{"node", "x.mjs", "--output", "/tmp/out"} }, ErrInvalidWorldPlan},
		{"joined-output", func(c *WorldPlanConfig) { c.StartArgv = []string{"node", "x.mjs", "--out=/tmp/out"} }, ErrInvalidWorldPlan},
		{"invalid-utf8", func(c *WorldPlanConfig) { c.Adapter.AdapterVersion = string([]byte{0xff}) }, ErrInvalidWorldPlan},
		{"tool-name-space", func(c *WorldPlanConfig) { c.RequiredTools[0].Name = "node cli" }, ErrInvalidWorldPlan},
		{"tool-name-unicode", func(c *WorldPlanConfig) { c.RequiredTools[0].Name = "nodé" }, ErrInvalidWorldPlan},
		{"tool-version-whitespace", func(c *WorldPlanConfig) { c.RequiredTools[0].VersionConstraint = " \t " }, ErrInvalidWorldPlan},
		{"tool-version-control", func(c *WorldPlanConfig) { c.RequiredTools[0].VersionConstraint = "v1\x00" }, ErrInvalidWorldPlan},
		{"secret-name-bound", func(c *WorldPlanConfig) {
			c.SecretSlots = []SecretSlot{{Name: strings.Repeat("A", 129), Presence: SecretAbsent}}
		}, ErrInvalidWorldPlan},
		{"readiness-whitespace", func(c *WorldPlanConfig) { c.Readiness.SignalName = " \t " }, ErrInvalidWorldPlan},
		{"readiness-control", func(c *WorldPlanConfig) { c.Readiness.SignalName = "ready\n" }, ErrInvalidWorldPlan},
		{"readiness-bound", func(c *WorldPlanConfig) { c.Readiness.SignalName = strings.Repeat("r", 129) }, ErrInvalidWorldPlan},
		{"trial-budget", func(c *WorldPlanConfig) { c.Budgets.TotalCandidateTrials = 23 }, ErrInvalidWorldPlan},
		{"secret-overlap", func(c *WorldPlanConfig) {
			c.Environment = []EnvironmentEntry{{Name: "LANG", Value: "C"}}
			c.SecretSlots = []SecretSlot{{Name: "LANG", Presence: SecretRequiredNoCapture}}
		}, ErrSecretValueInPlan},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validPlanConfig()
			test.mutate(&config)
			_, err := NewWorldPlan(config)
			var domainError *Error
			if !errors.As(err, &domainError) || domainError.Code != test.code {
				t.Fatalf("got %v, want code %s", err, test.code)
			}
		})
	}

	replacement := validPlanConfig()
	replacement.Readiness.SignalName = "\ufffd"
	if _, err := NewWorldPlan(replacement); err != nil {
		t.Fatalf("valid replacement rune should remain representable: %v", err)
	}
}

func TestCandidateExecutionKeyHasOneExactWireProfile(t *testing.T) {
	key := testCandidate()
	if !key.Valid() || len(key.String()) != len("candidate:")+64 {
		t.Fatalf("candidate key = %q", key)
	}
	for _, invalid := range []string{
		"candidate:a",
		"candidate:" + strings.Repeat("a", 63),
		"candidate:" + strings.Repeat("a", 65),
		"candidate:" + strings.Repeat("A", 64),
		"candidate:" + strings.Repeat("g", 64),
	} {
		if _, err := ParseCandidateExecutionKey(invalid); err == nil {
			t.Fatalf("accepted invalid candidate key %q", invalid)
		}
	}
}

func TestProjectionFingerprintRequiresCanonicalBytes(t *testing.T) {
	if _, err := NewProjectionFingerprint([]byte(`{"b":1,"a":2}`)); err == nil {
		t.Fatal("accepted noncanonical projection bytes")
	}
	left, err := NewProjectionFingerprint([]byte(`{"a":2,"b":1}`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewProjectionFingerprint([]byte(`{"a":2,"b":1}`))
	if err != nil || left != right {
		t.Fatal("canonical projection fingerprint was not deterministic")
	}
}

func TestReceiptGradeRoundTripsVerbatim(t *testing.T) {
	grade := "FUTURE_GRADE(v7): keep spaces / punctuation?!"
	receipt, err := NewDidrunReceipt(grade, "abc123", testDigest("a"))
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := ReceiptFromWire(receipt.Wire())
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.GradeVerbatim() != grade || Unreceipted() != "UNRECEIPTED" {
		t.Fatal("opaque receipt grade was normalized")
	}
}

func advanceToCapturing(t *testing.T, attempt Attempt) Attempt {
	t.Helper()
	var err error
	for _, next := range []AttemptState{AttemptMaterializing, AttemptStarting, AttemptReady, AttemptProbing, AttemptCapturing} {
		attempt, err = attempt.Advance(next)
		if err != nil {
			t.Fatal(err)
		}
	}
	return attempt
}

func TestAttemptRetainsPrimaryAndTeardownControls(t *testing.T) {
	attempt, err := NewAttempt("attempt:a", testDigest("a"), AttemptConfirmation)
	if err != nil {
		t.Fatal(err)
	}
	original := attempt
	if _, err := attempt.Advance(AttemptStarting); err == nil {
		t.Fatal("skipped materialization")
	}
	attempt = advanceToCapturing(t, attempt)
	failed, err := attempt.Fail(ControlTimeout)
	if err != nil {
		t.Fatal(err)
	}
	failed, err = failed.RecordTeardownControl(ControlTeardownError)
	if err != nil {
		t.Fatal(err)
	}
	failed, err = failed.RecordTeardownControl(ControlOrphanRisk)
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := failed.Advance(AttemptFinalized)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := finalized.FinalizedEvidence()
	if err != nil {
		t.Fatal(err)
	}
	primary, ok := evidence.PrimaryControl()
	if !ok || primary != ControlTimeout || len(evidence.TeardownControls()) != 2 || original.State() != AttemptAllocated {
		t.Fatal("attempt lost primary cause, teardown/orphan evidence, or immutability")
	}
	if _, err := failed.RecordTeardownControl(ControlTimeout); err == nil {
		t.Fatal("accepted a primary reason as teardown control")
	}
}

func TestComparisonEnvelopeSeparatesPolicyAssessmentAndWorld(t *testing.T) {
	config := validEnvelopeConfig()
	envelope, err := NewComparisonEnvelope(config)
	if err != nil {
		t.Fatal(err)
	}
	config.Measured[0].Name = "mutated"
	if envelope.Config().Measured[0].Name == "mutated" {
		t.Fatal("source mutation changed envelope")
	}

	unclassified := validEnvelopeConfig()
	unclassified.RequiredEqual = nil
	if _, err := NewComparisonEnvelope(unclassified); err == nil {
		t.Fatal("accepted measured dimension with no required disposition")
	}
	mismatch := validEnvelopeConfig()
	mismatch.Measured[0].Comparison = CompareRecordedOnly
	if _, err := NewComparisonEnvelope(mismatch); err == nil {
		t.Fatal("accepted comparator/disposition disagreement")
	}
	overlap := validEnvelopeConfig()
	overlap.Uncontrolled = append(overlap.Uncontrolled, "temporary root")
	if _, err := NewComparisonEnvelope(overlap); err == nil {
		t.Fatal("accepted measured dimension as uncontrolled")
	}

	leftMeasured := measurementRow(t, envelope, testDigest("a"), nil)
	rightMeasured := measurementRow(t, envelope, testDigest("b"), map[string]string{"temporary root": "different-tolerated-root"})
	assessment, err := AssessComparison(envelope, []InstanceMeasurements{leftMeasured.measurements, rightMeasured.measurements})
	if err != nil {
		t.Fatal(err)
	}
	admitted, ok := assessment.(AdmittedComparison)
	if !ok {
		t.Fatalf("assessment = %T, want AdmittedComparison", assessment)
	}
	token, err := admitted.AdmissionFor(leftMeasured.measurements)
	if err != nil {
		t.Fatal(err)
	}
	changedRequired := measurementRow(t, envelope, testDigest("c"), map[string]string{"operating system": "other-os"})
	rejectedAssessment, err := AssessComparison(envelope, []InstanceMeasurements{leftMeasured.measurements, changedRequired.measurements})
	if err != nil {
		t.Fatal(err)
	}
	if rejected, ok := rejectedAssessment.(RejectedComparison); !ok || len(rejected.ReasonCodes()) == 0 {
		t.Fatalf("required variance = %#v (%T), want rejected comparison", rejectedAssessment, rejectedAssessment)
	}
	if _, err := NewInstanceMeasurements(envelope, leftMeasured.world, []MeasurementValue{}); err == nil {
		t.Fatal("incomplete measured row was accepted")
	}
	world := leftMeasured.world
	if !world.Digest().Valid() || !token.Valid() || token.SubjectDigest() != world.Digest() {
		t.Fatalf("world/token binding = %#v / %#v", world, token)
	}
}

func TestComparisonAdmissionIdentityIncludesExactMeasurementMatrix(t *testing.T) {
	envelope, err := NewComparisonEnvelope(validEnvelopeConfig())
	if err != nil {
		t.Fatal(err)
	}
	firstLeft := measurementRow(t, envelope, testDigest("a"), map[string]string{"temporary root": "matrix-one-left"})
	firstRight := measurementRow(t, envelope, testDigest("b"), map[string]string{"temporary root": "matrix-one-right"})
	secondLeft := measurementRow(t, envelope, testDigest("a"), map[string]string{"temporary root": "matrix-two-left"})
	secondRight := measurementRow(t, envelope, testDigest("b"), map[string]string{"temporary root": "matrix-two-right"})
	firstAssessment, err := AssessComparison(envelope, []InstanceMeasurements{firstLeft.measurements, firstRight.measurements})
	if err != nil {
		t.Fatal(err)
	}
	secondAssessment, err := AssessComparison(envelope, []InstanceMeasurements{secondLeft.measurements, secondRight.measurements})
	if err != nil {
		t.Fatal(err)
	}
	first, firstOK := firstAssessment.(AdmittedComparison)
	second, secondOK := secondAssessment.(AdmittedComparison)
	if !firstOK || !secondOK {
		t.Fatalf("assessments = %T / %T, want two admitted comparisons", firstAssessment, secondAssessment)
	}
	if first.ComparisonBasisDigest() != second.ComparisonBasisDigest() {
		t.Fatal("distinct tolerated measurement matrices unexpectedly changed the stimulus-independent comparison basis")
	}
	if len(first.MeasurementDigests()) != 2 || len(second.MeasurementDigests()) != 2 {
		t.Fatal("comparison admission omitted its exact measurement matrix")
	}
	if first.Digest() == second.Digest() {
		t.Fatal("independently measured matrices collapsed to one admission identity")
	}
}

func FuzzAttemptTransitionsNeverSkipTeardown(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6})
	f.Add([]byte{0, 1, 9, 7})
	f.Fuzz(func(t *testing.T, operations []byte) {
		attempt, err := NewAttempt("fuzz", testDigest("f"), AttemptReduction)
		if err != nil {
			t.Fatal(err)
		}
		states := []AttemptState{AttemptMaterializing, AttemptStarting, AttemptReady, AttemptProbing, AttemptCapturing, AttemptTearingDown, AttemptFinalized}
		for _, operation := range operations {
			switch operation % 10 {
			case 7:
				if next, nextErr := attempt.Fail(ControlTimeout); nextErr == nil {
					attempt = next
				}
			case 8:
				if next, nextErr := attempt.RecordTeardownControl(ControlOrphanRisk); nextErr == nil {
					attempt = next
				}
			default:
				nextState := states[int(operation)%len(states)]
				if next, nextErr := attempt.Advance(nextState); nextErr == nil {
					attempt = next
				}
			}
		}
		if attempt.State() == AttemptFinalized {
			history := attempt.StateHistory()
			if len(history) > 2 {
				foundTeardown := false
				for _, state := range history[:len(history)-1] {
					foundTeardown = foundTeardown || state == AttemptTearingDown
				}
				if !foundTeardown {
					t.Fatal("resource-owning attempt finalized without teardown")
				}
			}
		}
	})
}
