//go:build darwin && cgo

package http_invoices

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/spec"
	"github.com/nelsonwerd/countershape/testkit/httpfixture"
)

type projectedInvoiceBody struct {
	FixtureVersion string         `json:"fixture_version"`
	InvoiceID      string         `json:"invoice_id"`
	Kind           string         `json:"kind"`
	Metadata       map[string]any `json:"metadata"`
	RequestID      string         `json:"request_id"`
	ScratchRoot    string         `json:"scratch_root"`
}

func TestHTTPInvoiceReferenceStudy(t *testing.T) {
	result := runReferenceStudy(t)
	if result.Observation.Status() != observe.ObservationComplete || result.Observation.CompletedMatrices() != 3 {
		t.Fatalf("observation status=%s matrices=%d", result.Observation.Status(), result.Observation.CompletedMatrices())
	}
	if !result.HasOutcomeMap || !result.OutcomeMap.Divergence() ||
		result.OutcomeMap.DistinctProjectionCount() != 3 || len(result.OutcomeMap.Entries()) != 3 ||
		len(result.OutcomeMap.Exclusions()) != 1 {
		t.Fatalf("unexpected invoice map: entries=%d exclusions=%d distinct=%d divergence=%v",
			len(result.OutcomeMap.Entries()), len(result.OutcomeMap.Exclusions()),
			result.OutcomeMap.DistinctProjectionCount(), result.OutcomeMap.Divergence())
	}
	if len(result.Trials) != 12 {
		t.Fatalf("trial evidence count=%d, want 12", len(result.Trials))
	}

	seenRootsByKind := map[string]map[string]struct{}{}
	seenAttempts := map[domain.Digest]struct{}{}
	seenWorlds := map[domain.Digest]struct{}{}
	seenObservations := map[domain.Digest]struct{}{}
	representativeProjection := map[httpfixture.CandidateRole][]byte{}
	representativeBodies := map[httpfixture.CandidateRole][]byte{}
	wantStatus := map[httpfixture.CandidateRole]int{
		httpfixture.Forbidden: 403, httpfixture.ConcealNotFound: 404,
		httpfixture.MetadataDisclosure: 200,
	}
	for _, trial := range result.Trials {
		if !trial.Admitted || !trial.Projected || trial.ProjectionRejection != nil {
			t.Fatalf("trial %d role=%s was not admitted and projected", trial.Slot.Ordinal(), trial.Role)
		}
		process := trial.Result.Process()
		preTerm := process.PreTermGroupProbe()
		if primary, present := process.PrimaryControl(); present || primary != "" ||
			!process.Started() || !process.ProcessGroupOwned() || !process.DirectChildWaited() ||
			!process.DrainsComplete() || !process.FinalGroupProbeClean() ||
			preTerm != "ABSENT" || process.TermSent() || process.KillSent() ||
			process.ExitCode() != 0 || process.ExitSignal() != "" || process.DiagnosticCode() != "" ||
			process.TeardownError() || process.OrphanRisk() {
			t.Fatalf("trial %d role=%s lacks clean cooperative service lifecycle: primary=%q/%t started=%t group-owned=%t waited=%t drains=%t final-clean=%t pre-term=%s term=%t kill=%t exit=%d signal=%q diagnostic=%q teardown=%t orphan=%t",
				trial.Slot.Ordinal(), trial.Role, primary, present, process.Started(), process.ProcessGroupOwned(),
				process.DirectChildWaited(), process.DrainsComplete(), process.FinalGroupProbeClean(),
				preTerm, process.TermSent(), process.KillSent(), process.ExitCode(),
				process.ExitSignal(), process.DiagnosticCode(), process.TeardownError(), process.OrphanRisk())
		}
		assertFreshOwnedRoots(t, trial, seenRootsByKind)
		attemptDigest := trial.Result.FinalizedAttempt().ArtifactDigest()
		worldDigest := trial.Result.World().Digest()
		observationDigest := trial.Observation.Digest()
		if _, duplicate := seenAttempts[attemptDigest]; duplicate {
			t.Fatalf("reused attempt %s", attemptDigest)
		}
		if _, duplicate := seenWorlds[worldDigest]; duplicate {
			t.Fatalf("reused world %s", worldDigest)
		}
		if _, duplicate := seenObservations[observationDigest]; duplicate {
			t.Fatalf("reused HTTP observation %s", observationDigest)
		}
		seenAttempts[attemptDigest] = struct{}{}
		seenWorlds[worldDigest] = struct{}{}
		seenObservations[observationDigest] = struct{}{}

		seed, hasSeed := trial.Result.HTTPSeedOverlay()
		readiness, hasReadiness := trial.Result.HTTPReadiness()
		exchange, hasExchange := trial.Result.HTTPExchange()
		invocation, hasInvocation := trial.Result.HTTPInvocationEvidence()
		if !hasSeed || !seed.Digest().Valid() || seed.Root() != trial.Result.Roots().Fixture() || len(seed.Entries()) != 1 ||
			!hasReadiness || !readiness.Digest().Valid() || !readiness.Accepted() || readiness.Protocol() != counterhttp.ReadinessProtocolV1 ||
			readiness.ListenerFD() != 3 || readiness.ReadinessFD() != 4 || readiness.BytesObserved() != 1 ||
			readiness.ObservedByte() != counterhttp.ReadinessSuccessByte || !readiness.EOFObserved() || readiness.DiagnosticCode() != "" ||
			!hasExchange || !exchange.Digest().Valid() || exchange.ConnectionAttempts() != 1 || !exchange.RequestComplete() ||
			exchange.RequestWrittenBytes() != int64(len(exchange.RequestWire())) || !exchange.ResponseParsed() ||
			exchange.ResponseOverflow() || exchange.DiagnosticCode() != "" ||
			!hasInvocation || !invocation.Digest().Valid() || !invocation.Validated() {
			t.Fatalf("trial %d lacks strict HTTP physical lineage", trial.Slot.Ordinal())
		}
		encoded, err := counterhttp.EncodeRequest(result.Stimulus, readiness.Port())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded.Bytes(), exchange.RequestWire()) || encoded.ByteLength() != int(exchange.RequestWrittenBytes()) {
			t.Fatalf("trial %d request bytes escaped the exact stimulus encoder", trial.Slot.Ordinal())
		}
		response, err := counterhttp.ParseResponse(exchange.ResponseWire(), result.CapturePolicy)
		if err != nil {
			t.Fatal(err)
		}
		body := decodeInvoiceBody(t, response.Body())
		if body.FixtureVersion != httpfixture.FixtureVersion || body.InvoiceID != "inv-204" ||
			body.RequestID == "" || body.ScratchRoot != trial.Result.Roots().State() ||
			!bytes.Contains(response.Body(), []byte(body.RequestID)) ||
			!bytes.Contains(response.Body(), []byte(body.ScratchRoot)) {
			t.Fatalf("trial %d lost exact captured volatile evidence: %#v", trial.Slot.Ordinal(), body)
		}
		status, parsedStatus := exchange.Status()
		if !parsedStatus || status != response.Status() {
			t.Fatalf("trial %d status lineage mismatch", trial.Slot.Ordinal())
		}
		if trial.Role == httpfixture.Alternating {
			if trial.Slot.Repetition()%2 == 0 {
				if status != 404 || body.Kind != "not_found" {
					t.Fatalf("alternating even trial = %d/%s", status, body.Kind)
				}
			} else if status != 200 || body.Kind != "authorized_metadata" {
				t.Fatalf("alternating odd trial = %d/%s", status, body.Kind)
			}
		} else if status != wantStatus[trial.Role] {
			t.Fatalf("role %s status=%d, want %d", trial.Role, status, wantStatus[trial.Role])
		}
		assertRoleBody(t, trial.Role, body)
		cachePath := filepath.Join(trial.Result.Roots().State(), httpfixture.ContaminationFilename)
		_, cacheErr := os.Lstat(cachePath)
		if trial.Role == httpfixture.ConcealNotFound ||
			(trial.Role == httpfixture.Alternating && trial.Slot.Repetition()%2 == 0) {
			if cacheErr != nil {
				t.Fatalf("conceal path did not leave its own-world cache marker: %v", cacheErr)
			}
		} else if !errors.Is(cacheErr, os.ErrNotExist) {
			t.Fatalf("role %s observed cross-world cache contamination: %v", trial.Role, cacheErr)
		}
		transcript := trial.Projection.Transcript()
		if !trial.Projection.Digest().Valid() || len(transcript) != len(result.ProjectionDefinition.Operations()) ||
			!projectionTranscriptComplete(transcript) {
			t.Fatalf("trial %d lacks visible projection evidence", trial.Slot.Ordinal())
		}
		if first, present := representativeProjection[trial.Role]; present {
			if !bytes.Equal(first, trial.Projection.ProjectionBytes()) && trial.Role != httpfixture.Alternating {
				t.Fatalf("stable role %s changed projection across fresh worlds", trial.Role)
			}
		} else {
			representativeProjection[trial.Role] = trial.Projection.ProjectionBytes()
		}
		if first, present := representativeBodies[trial.Role]; present && trial.Role != httpfixture.Alternating {
			if bytes.Equal(first, response.Body()) {
				t.Fatalf("stable role %s lost volatile capture differences", trial.Role)
			}
		} else if !present {
			representativeBodies[trial.Role] = response.Body()
		}
	}
	if bytes.Equal(representativeProjection[httpfixture.ConcealNotFound], representativeProjection[httpfixture.MetadataDisclosure]) {
		t.Fatal("fresh product roots did not restore the actual B/C split")
	}
}

// MUTATION_ANCHOR: invoice-fixture-request-facts-must-drive-policy
func TestHTTPInvoiceRequestFactsDrivePhysicalPolicy(t *testing.T) {
	testCases := []struct {
		name       string
		mutate     func(*testing.T, *counterhttp.HTTPStimulusConfig)
		wantStatus int
		wantKind   string
	}{
		{
			name: "method",
			mutate: func(_ *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Method = counterhttp.MethodPOST
			},
			wantStatus: 405,
			wantKind:   "method_not_allowed",
		},
		{
			name: "path",
			mutate: func(_ *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Path = "/v1/invoices/inv-else"
			},
			wantStatus: 404,
			wantKind:   "route_not_found",
		},
		{
			name: "ordered-query",
			mutate: func(_ *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Query[0], config.Query[1] = config.Query[1], config.Query[0]
			},
			wantStatus: 400,
			wantKind:   "request_order_or_shape_error",
		},
		{
			name: "ordered-query-tags",
			mutate: func(_ *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Query[2], config.Query[3] = config.Query[3], config.Query[2]
			},
			wantStatus: 422,
			wantKind:   "tag_policy_mismatch",
		},
		{
			name: "required-tenant-header",
			mutate: func(_ *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Headers = append(config.Headers[:1], config.Headers[2:]...)
			},
			wantStatus: 400,
			wantKind:   "tenant_context_mismatch",
		},
		{
			name: "required-role-header",
			mutate: func(t *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Headers[2] = mustInvoiceHeader(t, "x-countershape-role", "viewer")
			},
			wantStatus: 403,
			wantKind:   "role_not_authorized",
		},
		{
			name: "audit-flag-presence",
			mutate: func(t *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Query[4] = mustInvoiceQueryValue(t, "audit", "")
			},
			wantStatus: 428,
			wantKind:   "audit_contract_error",
		},
		{
			name: "required-audit-header",
			mutate: func(t *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Headers[3] = mustInvoiceHeader(t, "x-countershape-audit", "optional")
			},
			wantStatus: 428,
			wantKind:   "audit_contract_error",
		},
		{
			name: "ordered-header-tags",
			mutate: func(_ *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Headers[4], config.Headers[5] = config.Headers[5], config.Headers[4]
			},
			wantStatus: 422,
			wantKind:   "tag_policy_mismatch",
		},
		{
			name: "ordered-header-shape",
			mutate: func(_ *testing.T, config *counterhttp.HTTPStimulusConfig) {
				config.Headers[0], config.Headers[1] = config.Headers[1], config.Headers[0]
			},
			wantStatus: 400,
			wantKind:   "request_order_or_shape_error",
		},
		{
			name: "present-empty-body",
			mutate: func(t *testing.T, config *counterhttp.HTTPStimulusConfig) {
				body, err := counterhttp.PresentBody(nil)
				if err != nil {
					t.Fatal(err)
				}
				config.Body = body
			},
			wantStatus: 415,
			wantKind:   "body_not_allowed",
		},
		{
			name: "present-body-content",
			mutate: func(t *testing.T, config *counterhttp.HTTPStimulusConfig) {
				body, err := counterhttp.PresentBody([]byte(`{"unexpected":true}`))
				if err != nil {
					t.Fatal(err)
				}
				config.Body = body
			},
			wantStatus: 415,
			wantKind:   "body_not_allowed",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			stimulus := mutateInvoiceStimulus(t, testCase.mutate)
			result := runOneMatrixStudy(t, stimulus)
			assertPhysicalPolicyOutcome(t, result, testCase.wantStatus, testCase.wantKind)
		})
	}
}

// MUTATION_ANCHOR: invoice-fixture-tenant-seed-shape-trap-must-change-labeled-map
func TestHTTPInvoiceTenantSeedShapeTrapIsPhysicalAndLabelSensitive(t *testing.T) {
	referenceStimulus, err := newInvoiceStimulus()
	if err != nil {
		t.Fatal(err)
	}
	tenantlessStimulus, err := newInvoiceStimulusWithSeed(httpfixture.TenantlessSeedJSON())
	if err != nil {
		t.Fatal(err)
	}
	if referenceStimulus.Digest() == tenantlessStimulus.Digest() ||
		referenceStimulus.ExecutionPayloadDigest() == tenantlessStimulus.ExecutionPayloadDigest() {
		t.Fatal("tenant seed removal did not change stimulus and execution-payload authority")
	}
	referenceWire, err := counterhttp.EncodeRequest(referenceStimulus, 43210)
	if err != nil {
		t.Fatal(err)
	}
	tenantlessWire, err := counterhttp.EncodeRequest(tenantlessStimulus, 43210)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(referenceWire.Bytes(), tenantlessWire.Bytes()) {
		t.Fatal("tenant seed removal unexpectedly changed method/path/query/header/body request bytes")
	}

	reference := runOneMatrixStudy(t, referenceStimulus)
	tenantless := runOneMatrixStudy(t, tenantlessStimulus)
	wantReference := map[httpfixture.CandidateRole]int{
		httpfixture.Forbidden: 403, httpfixture.ConcealNotFound: 404, httpfixture.MetadataDisclosure: 200,
	}
	wantTenantless := map[httpfixture.CandidateRole]int{
		httpfixture.Forbidden: 200, httpfixture.ConcealNotFound: 500, httpfixture.MetadataDisclosure: 500,
	}
	for role, want := range wantReference {
		if got := statusForRole(t, reference, role); got != want {
			t.Fatalf("reference role %s status=%d, want %d", role, got, want)
		}
	}
	for role, want := range wantTenantless {
		if got := statusForRole(t, tenantless, role); got != want {
			t.Fatalf("tenantless role %s status=%d, want %d", role, got, want)
		}
	}
	referenceProjections := representativeProjections(reference)
	tenantlessProjections := representativeProjections(tenantless)
	if bytes.Equal(referenceProjections[httpfixture.ConcealNotFound], referenceProjections[httpfixture.MetadataDisclosure]) {
		t.Fatal("reference B/C labels unexpectedly collapsed before the shape trap")
	}
	if !bytes.Equal(tenantlessProjections[httpfixture.ConcealNotFound], tenantlessProjections[httpfixture.MetadataDisclosure]) {
		t.Fatal("tenantless shape trap did not produce the deliberate B/C 500/500 equality")
	}
	for _, result := range []StudyResult{reference, tenantless} {
		for _, trial := range result.Trials {
			invocation, present := trial.Result.HTTPInvocationEvidence()
			process := trial.Result.Process()
			if !trial.Admitted || !trial.Projected || trial.ProjectionRejection != nil ||
				process.TeardownError() || process.OrphanRisk() || process.ExitCode() != 0 ||
				process.ExitSignal() != "" || process.TermSent() || process.KillSent() ||
				!present || !invocation.Validated() || !invocation.AttemptValidated() ||
				!invocation.StimulusValidated() || !invocation.RequestByteCountValidated() ||
				!invocation.RequestDigestValidated() || !invocation.InvocationCountValidated() {
				t.Fatalf("shape-trap role %s was not one clean eligible exact invocation", trial.Role)
			}
		}
	}
	for _, trial := range tenantless.Trials {
		exchange, present := trial.Result.HTTPExchange()
		if !present {
			t.Fatalf("tenantless role %s has no HTTP exchange", trial.Role)
		}
		response, err := counterhttp.ParseResponse(exchange.ResponseWire(), tenantless.CapturePolicy)
		if err != nil {
			t.Fatal(err)
		}
		body := decodeInvoiceBody(t, response.Body())
		wantKind := "tenant_seed_missing"
		if trial.Role == httpfixture.Forbidden {
			wantKind = "authorized_without_tenant_seed"
		}
		if body.Kind != wantKind || body.Metadata == nil || len(body.Metadata) != 0 {
			t.Fatalf("tenantless role %s kind=%s metadata=%#v", trial.Role, body.Kind, body.Metadata)
		}
	}
	t.Log("TENANT_SEED_SHAPE_TRAP exact_labeled_statuses reference=403|404|200 tenantless=200|500|500")
}

// MUTATION_ANCHOR: invoice-study-d-must-use-all-trials-never-majority
func TestHTTPInvoiceAlternatingCandidateUsesAllTrialsNeverMajority(t *testing.T) {
	result := runReferenceStudy(t)
	batch := batchForRole(t, result, httpfixture.Alternating)
	classification := batch.Classification()
	if classification.Status() != observe.Unstable || classification.EligibleTrials() != 3 ||
		classification.RequiredTrials() != 3 || len(classification.Histogram()) != 2 {
		t.Fatalf("D classification=%s eligible=%d/%d histogram=%#v",
			classification.Status(), classification.EligibleTrials(), classification.RequiredTrials(), classification.Histogram())
	}
	counts := []int{classification.Histogram()[0].Count, classification.Histogram()[1].Count}
	slices.Sort(counts)
	if !slices.Equal(counts, []int{1, 2}) {
		t.Fatalf("D histogram counts=%v, want exact 1/2", counts)
	}
	fingerprints := result.RolesByFingerprint()
	want := map[domain.ProjectionFingerprint]bool{
		fingerprints[httpfixture.ConcealNotFound]:    true,
		fingerprints[httpfixture.MetadataDisclosure]: true,
	}
	for _, bin := range classification.Histogram() {
		if !want[bin.Fingerprint] {
			t.Fatalf("D histogram includes a non-B/C fingerprint %s", bin.Fingerprint)
		}
	}
	if len(result.OutcomeMap.Exclusions()) != 1 ||
		result.OutcomeMap.Exclusions()[0].CandidateKey != batch.CandidateKey() ||
		result.OutcomeMap.Exclusions()[0].Classification != observe.Unstable {
		t.Fatal("D instability was not retained as the exact outcome-map exclusion")
	}
}

// MUTATION_ANCHOR: invoice-study-must-assert-exact-labeled-candidate-outcome-map
func TestHTTPInvoiceOutcomeMapUsesExactCandidateLabelsNotGroupShape(t *testing.T) {
	result := runReferenceStudy(t)
	if len(result.OutcomeMap.Entries()) != 3 || result.OutcomeMap.DistinctProjectionCount() != 3 {
		t.Fatal("reference map lost its exact three eligible labeled entries")
	}
	representative := representativeProjections(result)
	seenRoles := map[httpfixture.CandidateRole]bool{}
	for _, entry := range result.OutcomeMap.Entries() {
		role, present := result.CandidateRoles[entry.CandidateKey]
		if !present || role == httpfixture.Alternating || seenRoles[role] {
			t.Fatalf("entry has duplicate, excluded, or unknown candidate label: %s", entry.CandidateKey)
		}
		fingerprint, err := domain.NewProjectionFingerprint(representative[role])
		if err != nil {
			t.Fatal(err)
		}
		if entry.ProjectionFingerprint != fingerprint {
			t.Fatalf("candidate %s/%s was paired with another role's fingerprint", entry.CandidateKey, role)
		}
		seenRoles[role] = true
	}
	for _, role := range []httpfixture.CandidateRole{
		httpfixture.Forbidden, httpfixture.ConcealNotFound, httpfixture.MetadataDisclosure,
	} {
		if !seenRoles[role] {
			t.Fatalf("exact labeled map omitted %s", role)
		}
	}
}

func TestHTTPInvoiceStudyUsesStrictSourceCompilerAndPlanBoundSchedule(t *testing.T) {
	result := runReferenceStudy(t)
	parsed, err := spec.ParseSource(result.SourceSpecBytes)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := spec.Compile(parsed, result.ProjectionDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Digest() != result.SourceSpecDigest || compiled.Digest() != result.Plan.Digest() ||
		!bytes.Equal(compiled.CanonicalBytes(), result.Plan.CanonicalBytes()) {
		t.Fatal("HTTP study plan is not the exact product of its retained SourceSpec")
	}
	schedule := result.Observation.Schedule()
	if schedule.Rotation() != result.Plan.ScheduleRotation() || schedule.CandidateCount() != 4 ||
		schedule.Repetitions() != 3 || schedule.TotalTrials() != 12 {
		t.Fatal("HTTP physical schedule escaped the compiled plan")
	}
}

func TestHTTPInvoicePermutationPreservesExactMapWithFreshEvidence(t *testing.T) {
	firstConfig := referenceConfig(t)
	firstConfig.DisplayLabels = map[httpfixture.CandidateRole]string{
		httpfixture.Forbidden: "candidate one", httpfixture.ConcealNotFound: "candidate two",
		httpfixture.MetadataDisclosure: "candidate three", httpfixture.Alternating: "candidate four",
	}
	firstConfig.ProducerMetadata = "producer-a"
	first, err := Run(context.Background(), firstConfig)
	if err != nil {
		t.Fatal(err)
	}
	secondConfig := referenceConfig(t)
	secondConfig.CandidateOrder = []httpfixture.CandidateRole{
		httpfixture.Alternating, httpfixture.MetadataDisclosure, httpfixture.ConcealNotFound, httpfixture.Forbidden,
	}
	secondConfig.DisplayLabels = map[httpfixture.CandidateRole]string{
		httpfixture.Forbidden: "renamed z", httpfixture.ConcealNotFound: "renamed y",
		httpfixture.MetadataDisclosure: "renamed x", httpfixture.Alternating: "renamed w",
	}
	secondConfig.ProducerMetadata = "producer-b"
	second, err := Run(context.Background(), secondConfig)
	if err != nil {
		t.Fatal(err)
	}
	if first.Plan.Digest() != second.Plan.Digest() || first.Binding.Digest() != second.Binding.Digest() ||
		first.ProjectionDefinition.Digest() != second.ProjectionDefinition.Digest() ||
		first.Observation.Schedule().Digest() != second.Observation.Schedule().Digest() ||
		first.OutcomeMap.ScheduleDigest() != second.OutcomeMap.ScheduleDigest() {
		t.Fatalf("candidate order or display metadata changed semantic HTTP authority: plan %s/%s binding %s/%s projection %s/%s schedule %s/%s outcome-schedule %s/%s",
			first.Plan.Digest(), second.Plan.Digest(), first.Binding.Digest(), second.Binding.Digest(),
			first.ProjectionDefinition.Digest(), second.ProjectionDefinition.Digest(),
			first.Observation.Schedule().Digest(), second.Observation.Schedule().Digest(),
			first.OutcomeMap.ScheduleDigest(), second.OutcomeMap.ScheduleDigest())
	}
	if !sameOutcomeEntries(first.OutcomeMap.Entries(), second.OutcomeMap.Entries()) ||
		!sameOutcomeExclusions(first.OutcomeMap.Exclusions(), second.OutcomeMap.Exclusions()) {
		t.Fatalf("candidate permutation changed exact map entries or exclusions: first entries=%+v exclusions=%+v batches=%v controlled-trials=%v; second entries=%+v exclusions=%+v batches=%v controlled-trials=%v",
			first.OutcomeMap.Entries(), first.OutcomeMap.Exclusions(), studyBatchSummaries(first), studyControlledTrialSummaries(first),
			second.OutcomeMap.Entries(), second.OutcomeMap.Exclusions(), studyBatchSummaries(second), studyControlledTrialSummaries(second))
	}
	if first.OutcomeMap.PreservationDigest() != second.OutcomeMap.PreservationDigest() {
		t.Fatalf("structurally equal maps produced different preservation authority: %s/%s",
			first.OutcomeMap.PreservationDigest(), second.OutcomeMap.PreservationDigest())
	}
	if first.OutcomeMap.ArtifactDigest() == second.OutcomeMap.ArtifactDigest() {
		t.Fatal("fresh HTTP execution reused the complete outcome artifact")
	}
	assertDigestSetsDisjoint(t, "attempt", first.OutcomeMap.EvidenceAttemptDigests(), second.OutcomeMap.EvidenceAttemptDigests())
	assertDigestSetsDisjoint(t, "world", first.OutcomeMap.EvidenceWorldDigests(), second.OutcomeMap.EvidenceWorldDigests())
	assertDigestSetsDisjoint(t, "observation", first.OutcomeMap.EvidenceObservationDigests(), second.OutcomeMap.EvidenceObservationDigests())
	for role, projection := range representativeProjections(first) {
		if role == httpfixture.Alternating {
			continue
		}
		if !bytes.Equal(projection, representativeProjections(second)[role]) {
			t.Fatalf("role %s changed exact projection under permutation", role)
		}
	}
	if !slices.Equal(first.CanonicalCandidateRoster(), second.CanonicalCandidateRoster()) {
		t.Fatal("candidate permutation changed canonical opaque roster")
	}
}

func studyBatchSummaries(result StudyResult) []string {
	summaries := make([]string, 0, len(result.Observation.Batches()))
	for _, batch := range result.Observation.Batches() {
		classification := batch.Classification()
		summaries = append(summaries, fmt.Sprintf("%s=%s eligible=%d/%d reasons=%v histogram=%v",
			result.CandidateRoles[batch.CandidateKey()], classification.Status(), classification.EligibleTrials(),
			classification.RequiredTrials(), classification.Reasons(), classification.Histogram()))
	}
	return summaries
}

func studyControlledTrialSummaries(result StudyResult) []string {
	summaries := []string{}
	for _, trial := range result.Trials {
		process := trial.Result.Process()
		if !process.TeardownError() && !process.OrphanRisk() {
			continue
		}
		summaries = append(summaries, fmt.Sprintf("%s#%d diagnostic=%s waited=%t drains=%t final-clean=%t term=%t kill=%t exit=%d signal=%s stderr=%q",
			trial.Role, trial.Slot.Ordinal(), process.DiagnosticCode(), process.DirectChildWaited(), process.DrainsComplete(),
			process.FinalGroupProbeClean(), process.TermSent(), process.KillSent(), process.ExitCode(), process.ExitSignal(), process.Stderr()))
	}
	return summaries
}

func TestHTTPInvoiceStudyRecordsTimingAndTrialMultiplication(t *testing.T) {
	config := referenceConfig(t)
	result, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	budget := result.Observation.Budget()
	if result.Plan.Budgets().TotalCandidateTrials != 24 || result.Observation.Schedule().TotalTrials() != 12 ||
		budget.MaxTotalTrials() != 12 || budget.StartedTrials() != 12 || budget.CompletedTrials() != 12 ||
		budget.Elapsed() <= 0 || result.Elapsed <= 0 || budget.Elapsed() > config.WallBudget {
		t.Fatalf("timing/multiplication mismatch: plan=%d schedule=%d max=%d started=%d completed=%d observation=%s study=%s",
			result.Plan.Budgets().TotalCandidateTrials, result.Observation.Schedule().TotalTrials(), budget.MaxTotalTrials(),
			budget.StartedTrials(), budget.CompletedTrials(), budget.Elapsed(), result.Elapsed)
	}
	t.Logf("HTTP_INVOICE_STUDY candidates=4 repeats=3 discovery_trials=12 confirmation_capacity=12 total_plan_trials=24 started=12 completed=12 observation_elapsed=%s study_elapsed=%s",
		budget.Elapsed(), result.Elapsed)
}

func mutateInvoiceStimulus(
	t *testing.T,
	mutate func(*testing.T, *counterhttp.HTTPStimulusConfig),
) counterhttp.HTTPStimulus {
	t.Helper()
	base, err := newInvoiceStimulus()
	if err != nil {
		t.Fatal(err)
	}
	config := counterhttp.HTTPStimulusConfig{
		Method: base.Method(), Path: base.Path(), Query: base.Query(), Headers: base.Headers(),
		Body: base.Body(), Seeds: base.Seeds(),
	}
	mutate(t, &config)
	result, err := counterhttp.NewHTTPStimulus(config)
	if err != nil {
		t.Fatal(err)
	}
	if result.Digest() == base.Digest() || result.ExecutionPayloadDigest() == base.ExecutionPayloadDigest() {
		t.Fatal("request mutation did not change stimulus and execution-payload authority")
	}
	return result
}

func mustInvoiceHeader(t *testing.T, name, value string) counterhttp.HTTPRequestHeader {
	t.Helper()
	header, err := counterhttp.NewRequestHeader(name, value)
	if err != nil {
		t.Fatal(err)
	}
	return header
}

func mustInvoiceQueryValue(t *testing.T, name, value string) counterhttp.HTTPQueryEntry {
	t.Helper()
	entry, err := counterhttp.QueryValue(name, value)
	if err != nil {
		t.Fatal(err)
	}
	return entry
}

func runOneMatrixStudy(t *testing.T, stimulus counterhttp.HTTPStimulus) StudyResult {
	t.Helper()
	config := referenceConfig(t)
	config.Repetitions = 1
	config.MaxTotalTrials = 4
	config.StimulusOverride = &stimulus
	result, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if result.Observation.Status() != observe.ObservationComplete || result.Observation.CompletedMatrices() != 1 ||
		!result.HasOutcomeMap || len(result.Trials) != 4 {
		t.Fatalf("one-matrix physical study is incomplete: status=%s matrices=%d map=%t trials=%d",
			result.Observation.Status(), result.Observation.CompletedMatrices(), result.HasOutcomeMap, len(result.Trials))
	}
	return result
}

func assertPhysicalPolicyOutcome(t *testing.T, result StudyResult, wantStatus int, wantKind string) {
	t.Helper()
	if len(result.OutcomeMap.Entries()) != 4 || len(result.OutcomeMap.Exclusions()) != 0 ||
		result.OutcomeMap.DistinctProjectionCount() != 1 {
		t.Fatalf("request policy case did not produce one eligible projection for all labels: entries=%d exclusions=%d distinct=%d",
			len(result.OutcomeMap.Entries()), len(result.OutcomeMap.Exclusions()), result.OutcomeMap.DistinctProjectionCount())
	}
	seenRoles := map[httpfixture.CandidateRole]bool{}
	for _, trial := range result.Trials {
		if trial.Admitted == false || trial.Projected == false || trial.ProjectionRejection != nil {
			t.Fatalf("request policy role %s was not admitted and projected", trial.Role)
		}
		if seenRoles[trial.Role] {
			t.Fatalf("request policy repeated role %s in one matrix", trial.Role)
		}
		seenRoles[trial.Role] = true

		process := trial.Result.Process()
		preTerm := process.PreTermGroupProbe()
		if primary, present := process.PrimaryControl(); present || primary != "" ||
			!process.DirectChildWaited() || !process.DrainsComplete() || !process.FinalGroupProbeClean() ||
			preTerm != "ABSENT" || process.TermSent() || process.KillSent() ||
			process.ExitCode() != 0 || process.ExitSignal() != "" || process.TeardownError() || process.OrphanRisk() {
			t.Fatalf("request policy role %s was not a clean application outcome: primary=%q/%t waited=%t drains=%t final=%t pre-term=%s term=%t kill=%t exit=%d signal=%q teardown=%t orphan=%t",
				trial.Role, primary, present, process.DirectChildWaited(), process.DrainsComplete(),
				process.FinalGroupProbeClean(), preTerm, process.TermSent(), process.KillSent(),
				process.ExitCode(), process.ExitSignal(), process.TeardownError(), process.OrphanRisk())
		}

		readiness, hasReadiness := trial.Result.HTTPReadiness()
		exchange, hasExchange := trial.Result.HTTPExchange()
		invocation, hasInvocation := trial.Result.HTTPInvocationEvidence()
		if !hasReadiness || !readiness.Accepted() || !hasExchange || !exchange.ResponseParsed() ||
			!hasInvocation || !invocation.Validated() || !invocation.AttemptValidated() ||
			!invocation.StimulusValidated() || !invocation.RequestByteCountValidated() ||
			!invocation.RequestDigestValidated() || !invocation.InvocationCountValidated() {
			t.Fatalf("request policy role %s lost readiness, response, or exact invocation lineage", trial.Role)
		}
		encoded, err := counterhttp.EncodeRequest(result.Stimulus, readiness.Port())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded.Bytes(), exchange.RequestWire()) ||
			exchange.RequestWrittenBytes() != int64(encoded.ByteLength()) {
			t.Fatalf("request policy role %s did not execute the mutated request bytes", trial.Role)
		}
		response, err := counterhttp.ParseResponse(exchange.ResponseWire(), result.CapturePolicy)
		if err != nil {
			t.Fatal(err)
		}
		body := decodeInvoiceBody(t, response.Body())
		status, parsed := exchange.Status()
		if !parsed || status != wantStatus || response.Status() != wantStatus || body.Kind != wantKind ||
			body.Metadata == nil || len(body.Metadata) != 0 || body.RequestID == "" ||
			body.ScratchRoot != trial.Result.Roots().State() {
			t.Fatalf("request policy role %s outcome=%d/%d kind=%s metadata=%#v request=%q root=%q",
				trial.Role, status, response.Status(), body.Kind, body.Metadata, body.RequestID, body.ScratchRoot)
		}
		if _, err := os.Lstat(filepath.Join(trial.Result.Roots().State(), httpfixture.ContaminationFilename)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("request policy role %s reached candidate state before its request decision: %v", trial.Role, err)
		}
	}
	if len(seenRoles) != len(httpfixture.Roles()) {
		t.Fatalf("request policy matrix covered %d roles, want %d", len(seenRoles), len(httpfixture.Roles()))
	}
}

func statusForRole(t *testing.T, result StudyResult, role httpfixture.CandidateRole) int {
	t.Helper()
	found := false
	status := 0
	for _, trial := range result.Trials {
		if trial.Role != role {
			continue
		}
		if found {
			t.Fatalf("one-matrix result repeated role %s", role)
		}
		found = true
		exchange, present := trial.Result.HTTPExchange()
		if !present {
			t.Fatalf("role %s has no HTTP exchange", role)
		}
		var parsed bool
		status, parsed = exchange.Status()
		if !parsed {
			t.Fatalf("role %s has no parsed HTTP status", role)
		}
	}
	if !found {
		t.Fatalf("one-matrix result omitted role %s", role)
	}
	return status
}

func referenceConfig(t *testing.T) Config {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git is not installed")
	}
	git, err := filepath.EvalSymlinks(gitPath)
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
	return DefaultConfig(t.TempDir(), git, node)
}

func runReferenceStudy(t *testing.T) StudyResult {
	t.Helper()
	result, err := Run(context.Background(), referenceConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func decodeInvoiceBody(t *testing.T, body []byte) projectedInvoiceBody {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var result projectedInvoiceBody
	if err := decoder.Decode(&result); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("invoice body contains trailing JSON or malformed suffix: %v", err)
	}
	return result
}

func assertRoleBody(t *testing.T, role httpfixture.CandidateRole, body projectedInvoiceBody) {
	t.Helper()
	switch role {
	case httpfixture.Forbidden:
		if body.Kind != "forbidden" || body.Metadata == nil || len(body.Metadata) != 0 {
			t.Fatalf("A body=%#v", body)
		}
	case httpfixture.ConcealNotFound:
		if body.Kind != "not_found" || body.Metadata == nil || len(body.Metadata) != 0 {
			t.Fatalf("B body=%#v", body)
		}
	case httpfixture.MetadataDisclosure:
		amount, amountPresent := body.Metadata["amount_cents"].(json.Number)
		if body.Kind != "authorized_metadata" || body.Metadata == nil || !amountPresent ||
			body.Metadata["currency"] != "USD" || body.Metadata["owner_tenant"] != "tenant-b" ||
			amount.String() != "4200" {
			t.Fatalf("C disclosure metadata=%#v", body.Metadata)
		}
	case httpfixture.Alternating:
		if body.Kind == "not_found" {
			if body.Metadata == nil || len(body.Metadata) != 0 {
				t.Fatalf("D conceal metadata=%#v", body.Metadata)
			}
		} else if body.Kind == "authorized_metadata" {
			amount, amountPresent := body.Metadata["amount_cents"].(json.Number)
			if !amountPresent || amount.String() != "4200" || body.Metadata["currency"] != "USD" ||
				body.Metadata["owner_tenant"] != "tenant-b" {
				t.Fatalf("D disclosure metadata=%#v", body.Metadata)
			}
		} else {
			t.Fatalf("D body=%#v", body)
		}
	default:
		t.Fatalf("unknown role %q", role)
	}
}

func projectionTranscriptComplete(transcript []counterhttp.HTTPProjectionTraceEntry) bool {
	for _, entry := range transcript {
		if entry.Operation().Name() == "" || !entry.Operation().RuleDigest().Valid() || len(entry.SourceLinks()) == 0 {
			return false
		}
	}
	return len(transcript) > 0
}

func batchForRole(t *testing.T, result StudyResult, role httpfixture.CandidateRole) observe.StableBatch {
	t.Helper()
	for _, batch := range result.Observation.Batches() {
		if result.CandidateRoles[batch.CandidateKey()] == role {
			return batch
		}
	}
	t.Fatalf("missing batch for role %s", role)
	return observe.StableBatch{}
}

func assertFreshOwnedRoots(t *testing.T, trial TrialEvidence, seenByKind map[string]map[string]struct{}) {
	t.Helper()
	roots := trial.Result.Roots()
	for kind, path := range map[string]string{
		"attempt": roots.Attempt(), "candidate-parent": roots.CandidateParent(), "fixture": roots.Fixture(),
		"home": roots.Home(), "temporary": roots.Temporary(), "xdg-config": roots.XDGConfig(),
		"xdg-cache": roots.XDGCache(), "xdg-data": roots.XDGData(), "xdg-state": roots.XDGState(),
		"state": roots.State(), "evidence": roots.Evidence(),
	} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || !strings.HasPrefix(path, roots.Attempt()+string(os.PathSeparator)) && kind != "attempt" {
			t.Fatalf("trial %d has invalid %s root %q", trial.Slot.Ordinal(), kind, path)
		}
		seen := seenByKind[kind]
		if seen == nil {
			seen = map[string]struct{}{}
			seenByKind[kind] = seen
		}
		if _, duplicate := seen[path]; duplicate {
			t.Fatalf("trial %d reused %s root %s", trial.Slot.Ordinal(), kind, path)
		}
		seen[path] = struct{}{}
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

func sameOutcomeExclusions(left, right []compare.Exclusion) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].CandidateKey != right[index].CandidateKey ||
			left[index].Classification != right[index].Classification {
			return false
		}
	}
	return true
}

func representativeProjections(result StudyResult) map[httpfixture.CandidateRole][]byte {
	output := map[httpfixture.CandidateRole][]byte{}
	for _, trial := range result.Trials {
		if trial.Projected {
			if _, present := output[trial.Role]; !present {
				output[trial.Role] = trial.Projection.ProjectionBytes()
			}
		}
	}
	return output
}

func assertDigestSetsDisjoint(t *testing.T, kind string, left, right []domain.Digest) {
	t.Helper()
	seen := make(map[domain.Digest]struct{}, len(left))
	for _, digest := range left {
		seen[digest] = struct{}{}
	}
	for _, digest := range right {
		if _, duplicate := seen[digest]; duplicate {
			t.Fatalf("fresh runs reused %s evidence %s", kind, digest)
		}
	}
}
