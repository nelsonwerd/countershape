package confirmation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
)

func TestConfirmationNonceBindsRunChallengeAndOrdinal(t *testing.T) {
	challengeA := confirmationTestDigest(1)
	challengeB := confirmationTestDigest(2)
	values := map[string]struct{}{}
	for _, input := range []struct {
		challenge domain.Digest
		ordinal   int
	}{
		{challengeA, 0}, {challengeA, 1}, {challengeB, 0}, {challengeB, 1},
	} {
		nonce, err := confirmationNonce(input.challenge, input.ordinal)
		if err != nil || !strings.HasPrefix(nonce, "u6-confirmation:") {
			t.Fatalf("confirmation nonce %s/%d = %q, %v", input.challenge, input.ordinal, nonce, err)
		}
		if _, duplicate := values[nonce]; duplicate {
			t.Fatalf("confirmation nonce omitted challenge or ordinal: %q", nonce)
		}
		values[nonce] = struct{}{}
	}
	if _, err := confirmationNonce(domain.Digest(""), 0); confirmationErrorCode(err) != "INVALID_CONFIRMATION_NONCE_INPUT" {
		t.Fatalf("invalid challenge refusal = %v", err)
	}
	if _, err := confirmationNonce(challengeA, -1); confirmationErrorCode(err) != "INVALID_CONFIRMATION_NONCE_INPUT" {
		t.Fatalf("negative ordinal refusal = %v", err)
	}
}

func TestParseRecordRejectsCandidateSwapAcrossConfirmationOrdinals(t *testing.T) {
	recordBytes := checkedFreshConfirmationExample(t)
	var record map[string]any
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		t.Fatal(err)
	}
	facts, okFacts := record["physical_fact_bytes_base64"].([]any)
	digests, okDigests := record["physical_fact_digests"].([]any)
	if !okFacts || !okDigests || len(facts) < 2 || len(facts) != len(digests) {
		t.Fatal("checked-in confirmation lacks a physical fact matrix")
	}
	decoded := make([]map[string]any, 2)
	for index := range decoded {
		encoded, stringValue := facts[index].(string)
		if !stringValue {
			t.Fatal("physical fact base64 is not a string")
		}
		body, decodeErr := base64.StdEncoding.Strict().DecodeString(encoded)
		if decodeErr != nil || json.Unmarshal(body, &decoded[index]) != nil {
			t.Fatalf("physical fact %d cannot be decoded: %v", index, decodeErr)
		}
	}
	left, leftOK := decoded[0]["candidate_execution_key"].(string)
	right, rightOK := decoded[1]["candidate_execution_key"].(string)
	if !leftOK || !rightOK || left == right {
		t.Fatal("first two scheduled physical facts do not provide distinct candidates")
	}
	decoded[0]["candidate_execution_key"], decoded[1]["candidate_execution_key"] = right, left
	for index := range decoded {
		body := canonicalConfirmationTestJSON(t, decoded[index])
		digest, digestErr := canon.DigestBytes("FreshExecutionFact", body)
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		facts[index] = base64.StdEncoding.EncodeToString(body)
		digests[index] = digest.String()
	}
	tampered := canonicalConfirmationTestJSON(t, record)
	if _, err := ParseRecord(tampered); confirmationErrorCode(err) != "INVALID_FRESH_CONFIRMATION_RECORD" {
		t.Fatalf("candidate/ordinal reassignment refusal = %v", err)
	}
}

func TestParseRecordRejectsResealedPhysicalFactClosedFieldTampering(t *testing.T) {
	for _, mutation := range []struct {
		field string
		value any
	}{
		{field: "schema_version", value: "countershape/future"},
		{field: "kind", value: "FreshExecutionClaim"},
		{field: "authority", value: "CALLER_ASSERTED"},
		{field: "adapter", value: "GENERIC"},
		{field: "invocation_trust", value: "HOSTILE_PROCESS_ATTESTED"},
		{field: "freshness_semantics", value: "NONCE_ONLY"},
	} {
		recordBytes := checkedFreshConfirmationExample(t)
		var record map[string]any
		if err := json.Unmarshal(recordBytes, &record); err != nil {
			t.Fatal(err)
		}
		facts := record["physical_fact_bytes_base64"].([]any)
		digests := record["physical_fact_digests"].([]any)
		body, err := base64.StdEncoding.Strict().DecodeString(facts[0].(string))
		if err != nil {
			t.Fatal(err)
		}
		var fact map[string]any
		if err := json.Unmarshal(body, &fact); err != nil {
			t.Fatal(err)
		}
		fact[mutation.field] = mutation.value
		body = canonicalConfirmationTestJSON(t, fact)
		digest, err := canon.DigestBytes("FreshExecutionFact", body)
		if err != nil {
			t.Fatal(err)
		}
		facts[0] = base64.StdEncoding.EncodeToString(body)
		digests[0] = digest.String()
		if _, err := ParseRecord(canonicalConfirmationTestJSON(t, record)); confirmationErrorCode(err) != "INVALID_FRESH_CONFIRMATION_RECORD" {
			t.Fatalf("resealed %s refusal = %v", mutation.field, err)
		}
	}
}

func checkedFreshConfirmationExample(t testing.TB) []byte {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller path unavailable")
	}
	examplePath := filepath.Join(filepath.Dir(source), "..", "..", "spec", "examples", "v1", "choicepoint.valid.json")
	example, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatal(err)
	}
	var choicepoint struct {
		FreshConfirmationBase64 string `json:"fresh_confirmation_base64"`
	}
	if err := json.Unmarshal(example, &choicepoint); err != nil {
		t.Fatal(err)
	}
	recordBytes, err := base64.StdEncoding.Strict().DecodeString(choicepoint.FreshConfirmationBase64)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRecord(recordBytes); err != nil {
		t.Fatalf("checked-in confirmation precondition failed: %v", err)
	}
	return recordBytes
}

func canonicalConfirmationTestJSON(t testing.TB, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canon.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func TestPhysicalLedgerRejectsReuseInEveryPhysicalEvidenceDomain(t *testing.T) {
	base := physicalEvidenceIDs{
		attempt: confirmationTestDigest(10), world: confirmationTestDigest(11), process: confirmationTestDigest(12),
		root: confirmationTestDigest(13), invocation: confirmationTestDigest(14), file: confirmationTestDigest(15),
	}
	mutations := []struct {
		name   string
		mutate func(*physicalEvidenceIDs)
	}{
		{"attempt", func(value *physicalEvidenceIDs) { value.attempt = base.attempt }},
		{"world", func(value *physicalEvidenceIDs) { value.world = base.world }},
		{"process", func(value *physicalEvidenceIDs) { value.process = base.process }},
		{"root", func(value *physicalEvidenceIDs) { value.root = base.root }},
		{"invocation receipt", func(value *physicalEvidenceIDs) { value.invocation = base.invocation }},
		{"invocation file", func(value *physicalEvidenceIDs) { value.file = base.file }},
	}
	for index, mutation := range mutations {
		ledger := newPhysicalLedger()
		if err := ledger.admitIDs(base); err != nil {
			t.Fatal(err)
		}
		fresh := physicalEvidenceIDs{
			attempt: confirmationTestDigest(100 + index*10), world: confirmationTestDigest(101 + index*10),
			process: confirmationTestDigest(102 + index*10), root: confirmationTestDigest(103 + index*10),
			invocation: confirmationTestDigest(104 + index*10), file: confirmationTestDigest(105 + index*10),
		}
		mutation.mutate(&fresh)
		if err := ledger.admitIDs(fresh); confirmationErrorCode(err) != "REUSED_CONFIRMATION_PHYSICAL_EVIDENCE" {
			t.Fatalf("%s reuse refusal = %v", mutation.name, err)
		}
	}
}

func TestPriorEvidenceLedgerRejectsEveryDiscoveryReductionDigestReuse(t *testing.T) {
	ledger := priorEvidenceLedger{
		batch:       map[domain.Digest]struct{}{confirmationTestDigest(201): {}},
		attempt:     map[domain.Digest]struct{}{confirmationTestDigest(202): {}},
		world:       map[domain.Digest]struct{}{confirmationTestDigest(203): {}},
		observation: map[domain.Digest]struct{}{confirmationTestDigest(204): {}},
	}
	tests := []struct {
		name                               string
		batch, attempt, world, observation []domain.Digest
	}{
		{name: "batch", batch: []domain.Digest{confirmationTestDigest(201)}},
		{name: "attempt", attempt: []domain.Digest{confirmationTestDigest(202)}},
		{name: "world", world: []domain.Digest{confirmationTestDigest(203)}},
		{name: "observation", observation: []domain.Digest{confirmationTestDigest(204)}},
	}
	for _, test := range tests {
		err := ledger.rejectEvidence(test.batch, test.attempt, test.world, test.observation)
		if confirmationErrorCode(err) != "REUSED_CONFIRMATION_LINEAGE_EVIDENCE" || !strings.HasPrefix(err.Error(), "REUSED_CONFIRMATION_LINEAGE_EVIDENCE: "+test.name+":") {
			t.Fatalf("%s prior-evidence reuse refusal = %v", test.name, err)
		}
	}
	if err := ledger.rejectEvidence(
		[]domain.Digest{confirmationTestDigest(211)}, []domain.Digest{confirmationTestDigest(212)},
		[]domain.Digest{confirmationTestDigest(213)}, []domain.Digest{confirmationTestDigest(214)},
	); err != nil {
		t.Fatalf("fresh evidence was rejected: %v", err)
	}
}

func TestConfirmationRequiresExactLabeledMapNotPartitionShape(t *testing.T) {
	fixture := newConfirmationMapFixture(t)
	reduced := fixture.outcomeMap(t, domain.AttemptDiscovery, 1, [2]int{10, 20})
	confirmed := fixture.outcomeMap(t, domain.AttemptConfirmation, 2, [2]int{10, 20})
	assessment, err := requireExactLabeledConfirmation(reduced, confirmed)
	if err != nil || !assessment.Valid() || assessment.Relation() != compare.PreservationEqual {
		t.Fatalf("fresh equal labeled map was refused: %v", err)
	}
	relabeled := fixture.outcomeMap(t, domain.AttemptConfirmation, 3, [2]int{20, 10})
	if relabeled.DistinctProjectionCount() != confirmed.DistinctProjectionCount() ||
		len(relabeled.Entries()) != len(confirmed.Entries()) {
		t.Fatal("relabeled-map fixture changed partition shape")
	}
	if _, err := requireExactLabeledConfirmation(reduced, relabeled); confirmationErrorCode(err) != "CONFIRMATION_DOES_NOT_PRESERVE_EXACT_MAP" {
		t.Fatalf("same-shape relabeled confirmation refusal = %v", err)
	}
}

type confirmationMapFixture struct {
	plan     domain.WorldPlan
	envelope domain.ComparisonEnvelope
	bindings [2]domain.CandidateExecutionBinding
}

func newConfirmationMapFixture(t *testing.T) confirmationMapFixture {
	t.Helper()
	envelope, err := domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version:       "confirmation-test/v1",
		Measured:      []domain.MeasuredDimension{{Name: "os", Source: domain.MeasuredWorldInstance, Comparison: domain.CompareExact}},
		RequiredEqual: []domain.RequiredEqualDimension{{Name: "os"}}, Uncontrolled: []string{"scheduler"},
	})
	if err != nil {
		t.Fatal(err)
	}
	projection, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterHTTP, ImplementationDigest: confirmationTestDigest(301),
		ConfigurationDigest: confirmationTestDigest(302), AcceptedChannels: []string{"http.body"},
		Operations: []domain.ProjectionOperationBinding{{Name: "test", RuleDigest: confirmationTestDigest(303)}},
		Comparator: domain.ProjectionComparatorExact, FieldRegistryDigest: confirmationTestDigest(304),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest: confirmationTestDigest(305), MaterializationPolicyDigest: confirmationTestDigest(306),
		ComparisonEnvelopeDigest: envelope.Digest(), Adapter: domain.Adapter{
			Domain: domain.AdapterHTTP, AdapterVersion: "http/v1", RunnerDigest: confirmationTestDigest(307),
		},
		ExecutionShape: domain.OneLoopbackHTTPRequest, StartArgv: []string{"node", "fixture.mjs"}, SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}}, SecretSlots: []domain.SecretSlot{},
		FixtureRecipeDigest: confirmationTestDigest(308), Readiness: domain.Readiness{Kind: domain.FixtureOwnedReadiness, SignalName: "ready"},
		CapturePolicyDigest: confirmationTestDigest(309), ProjectionDefinition: projection,
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: 2, ConfirmationRepeats: 2, Concurrency: domain.ScheduleSequential,
			Rotation: domain.ScheduleRotationStartByRepetitionV1,
		},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "test"}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 100, MaterializedBytesPerWorld: 1 << 20,
			SingleBlobBytes: 1 << 18, ReadinessMS: 1000, ProbeMS: 1000, TeardownMS: 1000,
			StdoutBytes: 1 << 16, StderrBytes: 1 << 16, HTTPBodyBytes: 1 << 16,
			ProposedShrinkStimuli: 4, TotalCandidateTrials: 32, ShrinkWallMS: 60_000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture := confirmationMapFixture{plan: plan, envelope: envelope}
	for index := range fixture.bindings {
		fixture.bindings[index], err = domain.NewCandidateExecutionBinding(domain.CandidateExecutionIdentity{
			TreeIdentityDigest: confirmationTestDigest(320 + index), MaterializationPolicyDigest: plan.MaterializationPolicyDigest(),
			WorldPlanDigest: plan.Digest(), AdapterDigest: plan.AdapterDigest(), RunnerDigest: plan.Adapter().RunnerDigest,
			ProjectionDefinitionDigest: plan.ProjectionDefinitionDigest(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return fixture
}

func (f confirmationMapFixture) outcomeMap(
	t *testing.T,
	purpose domain.AttemptPurpose,
	salt int,
	projections [2]int,
) compare.CandidateOutcomeMap {
	t.Helper()
	roster := []domain.CandidateExecutionKey{f.bindings[0].Key(), f.bindings[1].Key()}
	schedule, err := observe.NewPhaseRotatedSchedule(roster, 2, purpose)
	if err != nil {
		t.Fatal(err)
	}
	trials := [2][]observe.TrialFact{}
	stimulus := confirmationTestDigest(400)
	for repetition := 0; repetition < 2; repetition++ {
		worlds := [2]domain.WorldInstance{}
		attempts := [2]domain.FinalizedAttempt{}
		measurements := make([]domain.InstanceMeasurements, 2)
		for index := range f.bindings {
			slot, present := schedule.Slot(f.bindings[index].Key(), repetition)
			if !present {
				t.Fatal("missing confirmation test schedule slot")
			}
			attemptDigest := confirmationTestDigest(10_000 + salt*100 + repetition*10 + index)
			attempts[index] = confirmationFinalizedAttempt(t, attemptDigest, purpose)
			worlds[index], err = domain.NewWorldInstance(f.plan, f.bindings[index], domain.WorldInstanceConfig{
				StimulusDigest: stimulus, AttemptArtifactDigest: attemptDigest, Purpose: purpose,
				InstanceNonce: fmt.Sprintf("confirmation-test:%d:%d:%d", salt, repetition, index), ScheduleOrdinal: slot.Ordinal(),
			})
			if err != nil {
				t.Fatal(err)
			}
			measured, valueErr := canon.String("darwin-test-basis")
			if valueErr != nil {
				t.Fatal(valueErr)
			}
			measurements[index], err = domain.NewInstanceMeasurements(f.envelope, worlds[index], []domain.MeasurementValue{{
				Name: "os", Source: domain.MeasuredWorldInstance, Value: measured,
			}})
			if err != nil {
				t.Fatal(err)
			}
		}
		assessment, err := domain.AssessComparison(f.envelope, measurements)
		if err != nil {
			t.Fatal(err)
		}
		admitted, ok := assessment.(domain.AdmittedComparison)
		if !ok {
			t.Fatalf("confirmation test comparison = %T, want admitted", assessment)
		}
		for index := range f.bindings {
			token, tokenErr := admitted.AdmissionFor(measurements[index])
			if tokenErr != nil {
				t.Fatal(tokenErr)
			}
			projection, projectionErr := canon.Integer(int64(projections[index]))
			if projectionErr != nil {
				t.Fatal(projectionErr)
			}
			projectionBytes := projection.Canonical()
			observationDigest := confirmationTestDigest(20_000 + salt*100 + repetition*10 + index)
			derivation, derivationErr := observe.NewProjectionDerivation(
				observationDigest, worlds[index].ProjectionDefinitionDigest(), projectionBytes,
				[]byte(`{"kind":"CONFIRMATION_TEST_DERIVATION","operations":["test"],"source_links":["test"]}`),
			)
			if derivationErr != nil {
				t.Fatal(derivationErr)
			}
			capture, captureErr := observe.NewStructuralCapture(
				worlds[index], attempts[index], observationDigest, projectionBytes, derivation,
			)
			if captureErr != nil {
				t.Fatal(captureErr)
			}
			trial, trialErr := observe.NewCapturedTrial(worlds[index], attempts[index], token, capture)
			if trialErr != nil {
				t.Fatal(trialErr)
			}
			trials[index] = append(trials[index], trial)
		}
	}
	batches := make([]observe.StableBatch, 2)
	for index := range f.bindings {
		batches[index], err = observe.ClassifyScheduled(observe.ScheduledBatchInput{
			Schedule: schedule, CandidateKey: f.bindings[index].Key(), Trials: trials[index],
		})
		if err != nil || batches[index].Classification().Status() != observe.ObservedStable {
			t.Fatalf("confirmation test batch %d was not stable: %v", index, err)
		}
	}
	result, err := compare.NewCandidateOutcomeMap(stimulus, f.envelope.Digest(), roster, batches)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func confirmationFinalizedAttempt(t *testing.T, digest domain.Digest, purpose domain.AttemptPurpose) domain.FinalizedAttempt {
	t.Helper()
	attempt, err := domain.NewAttempt("attempt:"+digest.String(), digest, purpose)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []domain.AttemptState{
		domain.AttemptMaterializing, domain.AttemptStarting, domain.AttemptReady, domain.AttemptProbing,
		domain.AttemptCapturing, domain.AttemptTearingDown, domain.AttemptFinalized,
	} {
		attempt, err = attempt.Advance(state)
		if err != nil {
			t.Fatal(err)
		}
	}
	finalized, err := attempt.FinalizedEvidence()
	if err != nil {
		t.Fatal(err)
	}
	return finalized
}

func confirmationTestDigest(value int) domain.Digest {
	return domain.MustDigest(fmt.Sprintf("sha256:%064x", value))
}

func confirmationErrorCode(err error) string {
	if typed, ok := err.(*Error); ok {
		return typed.Code
	}
	return ""
}
