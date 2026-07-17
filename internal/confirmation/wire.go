package confirmation

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"sort"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/reduce"
	"github.com/nelsonwerd/countershape/internal/world"
)

const freshConfirmationMemberCount = 38

// Record is the inert, strict durable representation of FreshConfirmation.
// It can be inspected after restart but cannot recreate the live execution
// seal required by Choicepoint promotion.
type Record struct {
	digest                domain.Digest
	canonicalBytes        []byte
	planDigest            domain.Digest
	originalBaseline      compare.CandidateOutcomeMap
	reducedBaseline       compare.CandidateOutcomeMap
	confirmed             compare.CandidateOutcomeMap
	projectionProofs      []ProjectionProof
	reductionRun          reduce.ReductionRunRecord
	reductionGradeBytes   []byte
	reductionGradeDigest  domain.Digest
	reductionGradeStatus  string
	reducedArtifact       compare.OutcomeArtifactDigest
	confirmedArtifact     compare.OutcomeArtifactDigest
	challengeDigest       domain.Digest
	priorLedgerDigest     domain.Digest
	physicalFactDigests   []domain.Digest
	processDigests        []domain.Digest
	rootLayoutDigests     []domain.Digest
	invocationDigests     []domain.Digest
	invocationFileDigests []domain.Digest
	executionBindings     []domain.Digest
}

func ParseRecord(exact []byte) (Record, error) {
	value, err := canon.Parse(exact)
	if err != nil {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_WIRE", "bytes are outside the strict canonical profile", err)
	}
	canonical, err := value.CanonicalChecked()
	members, object := value.Members()
	if err != nil || !bytes.Equal(canonical, exact) || !object || len(members) != freshConfirmationMemberCount {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_WIRE", "wire is nonexact or has unknown/missing members", err)
	}
	var identity freshConfirmationIdentity
	if err := json.Unmarshal(exact, &identity); err != nil {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_WIRE", "typed wire extraction failed", err)
	}
	digestRaw, err := canon.DigestBytes("FreshConfirmation", exact)
	if err != nil {
		return Record{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return Record{}, err
	}
	return parseRecordIdentity(identity, exact, digest)
}

func parseRecordIdentity(identity freshConfirmationIdentity, exact []byte, expected domain.Digest) (Record, error) {
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "FreshConfirmation" ||
		identity.Scope != confirmationScope || identity.InvocationTrustNonclaim != invocationTrustNonclaim ||
		identity.ConfirmationPhase != string(domain.AttemptConfirmation) || identity.ConfirmationScheduleOffset != 1 ||
		identity.ConfirmationTrialCount < 2 || identity.PreservationRelation != string(compare.PreservationEqual) ||
		identity.PreservationReason != "EXACT_PRESERVATION_MAP_MATCH" {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "closed semantic facts disagree", nil)
	}
	plan, err := parseDigest(identity.WorldPlanDigest, "world plan")
	if err != nil {
		return Record{}, err
	}
	challenge, err := parseDigest(identity.ChallengeDigest, "challenge")
	if err != nil {
		return Record{}, err
	}
	priorLedger, err := parseDigest(identity.PriorEvidenceLedgerDigest, "prior evidence ledger")
	if err != nil {
		return Record{}, err
	}
	originalBytes, err := decodeBase64(identity.OriginalBaselineMapBase64, "original baseline")
	if err != nil {
		return Record{}, err
	}
	reducedBytes, err := decodeBase64(identity.ReducedBaselineMapBase64, "reduced baseline")
	if err != nil {
		return Record{}, err
	}
	confirmedBytes, err := decodeBase64(identity.ConfirmedMapBase64, "confirmed map")
	if err != nil {
		return Record{}, err
	}
	original, err := compare.ParseCandidateOutcomeMap(originalBytes)
	if err != nil {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "original baseline map is not strict", err)
	}
	reduced, err := compare.ParseCandidateOutcomeMap(reducedBytes)
	if err != nil {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "reduced baseline map is not strict", err)
	}
	confirmed, err := compare.ParseCandidateOutcomeMap(confirmedBytes)
	if err != nil {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "confirmed map is not strict", err)
	}
	reducedArtifact, err := compare.ParseOutcomeArtifactDigest(identity.ReducedBaselineArtifact)
	if err != nil {
		return Record{}, err
	}
	reducedPreservation, err := compare.ParsePreservationMapDigest(identity.ReducedBaselinePreservation)
	if err != nil {
		return Record{}, err
	}
	confirmedArtifact, err := compare.ParseOutcomeArtifactDigest(identity.ConfirmedArtifact)
	if err != nil {
		return Record{}, err
	}
	confirmedPreservation, err := compare.ParsePreservationMapDigest(identity.ConfirmedPreservation)
	if err != nil {
		return Record{}, err
	}
	if original.PlanDigest() != plan || reduced.PlanDigest() != plan || confirmed.PlanDigest() != plan ||
		reduced.ArtifactDigest() != reducedArtifact || reduced.PreservationDigest() != reducedPreservation ||
		confirmed.ArtifactDigest() != confirmedArtifact || confirmed.PreservationDigest() != confirmedPreservation ||
		confirmed.Phase() != domain.AttemptConfirmation || confirmed.ScheduleStartOffset() != 1 ||
		confirmed.ScheduleDigest().String() != identity.ConfirmationScheduleDigest {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "map identity or schedule binding disagrees", nil)
	}
	assessment, assessmentErr := requireExactLabeledConfirmation(reduced, confirmed)
	if assessmentErr != nil || assessment.ReasonCode() != identity.PreservationReason {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "exact labeled map was not preserved", nil)
	}

	runBytes, err := decodeBase64(identity.ReductionRunBase64, "reduction run")
	if err != nil {
		return Record{}, err
	}
	transcriptBytes, err := decodeBase64(identity.ReductionTranscriptBase64, "reduction transcript")
	if err != nil {
		return Record{}, err
	}
	runRecord, err := reduce.ParseReductionRunRecord(runBytes, transcriptBytes)
	if err != nil {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "reduction wire is not strict", err)
	}
	runDigest, err := parseDigest(identity.ReductionRunDigest, "reduction run")
	if err != nil {
		return Record{}, err
	}
	transcriptDigest, err := parseDigest(identity.ReductionTranscriptDigest, "reduction transcript")
	if err != nil {
		return Record{}, err
	}
	if runRecord.Digest() != runDigest || runRecord.Transcript().Digest() != transcriptDigest ||
		runRecord.BaselineOutcomeMapDigest() != original.ArtifactDigest() ||
		runRecord.BaselinePreservationMapDigest() != original.PreservationDigest() ||
		runRecord.MinimizedStimulusDigest() != reduced.StimulusDigest() {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "reduction record does not bind the map lineage", nil)
	}
	if err := validateGradeWire(identity, runDigest); err != nil {
		return Record{}, err
	}
	gradeBytes, err := decodeBase64(identity.ReductionGradeBase64, "reduction grade")
	if err != nil {
		return Record{}, err
	}
	gradeDigest, err := parseDigest(identity.ReductionGradeDigest, "reduction grade")
	if err != nil {
		return Record{}, err
	}
	proofs, err := validateProjectionProofWire(identity.ProjectionProofs, confirmed)
	if err != nil {
		return Record{}, err
	}

	physical, err := validatePhysicalWire(identity, challenge, plan, confirmed)
	if err != nil {
		return Record{}, err
	}
	if !sameStringDigests(identity.BatchDigests, confirmed.BatchDigests()) ||
		!sameStringDigests(identity.AttemptDigests, confirmed.EvidenceAttemptDigests()) ||
		!sameStringDigests(identity.WorldDigests, confirmed.EvidenceWorldDigests()) ||
		!sameStringDigests(identity.ObservationDigests, confirmed.EvidenceObservationDigests()) {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "stored evidence lists differ from the confirmed map", nil)
	}
	rebuiltLedger, err := buildPriorEvidenceLedgerRecord(runRecord, original, reduced)
	if err != nil || rebuiltLedger != priorLedger {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "prior evidence ledger does not reconstruct", err)
	}
	if err := rejectRecordOverlap(runRecord, original, reduced, confirmed); err != nil {
		return Record{}, err
	}

	digestRaw, rebuilt, err := canon.DigestTyped("FreshConfirmation", identity)
	if err != nil || digestRaw.String() != expected.String() || !bytes.Equal(rebuilt, exact) {
		return Record{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "record identity does not round trip exactly", err)
	}
	return Record{
		digest: expected, canonicalBytes: append([]byte(nil), exact...), planDigest: plan,
		originalBaseline: original, reducedBaseline: reduced, confirmed: confirmed,
		projectionProofs: proofs, reductionRun: runRecord,
		reductionGradeBytes: append([]byte(nil), gradeBytes...), reductionGradeDigest: gradeDigest,
		reductionGradeStatus: identity.ReductionGradeStatus,
		reducedArtifact:      reducedArtifact, confirmedArtifact: confirmedArtifact, challengeDigest: challenge,
		priorLedgerDigest: priorLedger, physicalFactDigests: physical.factDigests, processDigests: physical.processDigests,
		rootLayoutDigests: physical.rootDigests, invocationDigests: physical.invocationDigests,
		invocationFileDigests: physical.invocationFileDigests, executionBindings: physical.executionBindings,
	}, nil
}

func validateProjectionProofWire(
	identities []projectionProofIdentity,
	confirmed compare.CandidateOutcomeMap,
) ([]ProjectionProof, error) {
	wrapped, err := compare.RequireConfirmedOutcomeMap(confirmed)
	if err != nil {
		return nil, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "projection proofs require a confirmed map", err)
	}
	entries := wrapped.ProjectionRoster().Entries()
	if len(identities) != len(entries) {
		return nil, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "projection proof roster differs", nil)
	}
	proofs := make([]ProjectionProof, len(identities))
	retained := 0
	for index, identity := range identities {
		candidate, parseErr := domain.ParseCandidateExecutionKey(identity.CandidateExecutionKey)
		if parseErr != nil || candidate != entries[index].CandidateExecutionKey() ||
			(index > 0 && identity.CandidateExecutionKey <= identities[index-1].CandidateExecutionKey) {
			return nil, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "projection proof candidate order differs", parseErr)
		}
		canonicalProjection, decodeErr := decodeBase64(identity.CanonicalProjectionBase64, "canonical projection")
		if decodeErr != nil {
			return nil, decodeErr
		}
		retained += len(canonicalProjection)
		if retained > maxProjectionProofBytes {
			return nil, refuse("CONFIRMATION_RESOURCE_LIMIT", "aggregate canonical projections exceed the durable v1 proof budget", nil)
		}
		fingerprint, verifyErr := wrapped.ProjectionRoster().Verify(candidate, canonicalProjection)
		if verifyErr != nil || fingerprint != entries[index].ProjectionFingerprint() || fingerprint.String() != identity.ProjectionFingerprint {
			return nil, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "projection proof differs from the exact confirmed roster", verifyErr)
		}
		proofs[index] = ProjectionProof{candidate: candidate, fingerprint: fingerprint, canonical: canonicalProjection}
	}
	return proofs, nil
}

func validateGradeWire(identity freshConfirmationIdentity, runDigest domain.Digest) error {
	gradeBytes, err := decodeBase64(identity.ReductionGradeBase64, "reduction grade")
	if err != nil {
		return err
	}
	value, err := canon.Parse(gradeBytes)
	if err != nil {
		return refuse("INVALID_FRESH_CONFIRMATION_RECORD", "reduction grade is not strict canonical bytes", err)
	}
	members, object := value.Members()
	schemaValue, hasSchema := value.LookupMember("schema_version")
	kindValue, hasKind := value.LookupMember("kind")
	statusValue, hasStatus := value.LookupMember("status")
	runValue, hasRun := value.LookupMember("run_digest")
	schema, schemaText := schemaValue.Text()
	kind, kindText := kindValue.Text()
	status, statusText := statusValue.Text()
	gradeRun, runText := runValue.Text()
	if !object || len(members) != 9 || !hasSchema || !hasKind || !hasStatus || !hasRun ||
		!schemaText || !kindText || !statusText || !runText || schema != domain.SchemaVersion || kind != "ReductionGrade" ||
		status != identity.ReductionGradeStatus || gradeRun != runDigest.String() {
		return refuse("INVALID_FRESH_CONFIRMATION_RECORD", "reduction grade fields disagree", nil)
	}
	switch identity.ReductionGradeStatus {
	case string(reductionStatusUnchanged), string(reductionStatusBestKnown), string(reductionStatusOneMinimal):
	default:
		return refuse("INVALID_FRESH_CONFIRMATION_RECORD", "unknown reduction grade status", nil)
	}
	gradeDigest, err := parseDigest(identity.ReductionGradeDigest, "reduction grade")
	if err != nil {
		return err
	}
	digestRaw, err := canon.DigestBytes("ReductionGrade", gradeBytes)
	if err != nil || digestRaw.String() != gradeDigest.String() {
		return refuse("INVALID_FRESH_CONFIRMATION_RECORD", "reduction grade digest differs", err)
	}
	return nil
}

// Local strings avoid importing the outward reduction package into this inert
// parser solely to compare enum spellings.
type reductionStatus string

const (
	reductionStatusUnchanged  reductionStatus = "UNCHANGED"
	reductionStatusBestKnown  reductionStatus = "BEST_KNOWN"
	reductionStatusOneMinimal reductionStatus = "ONE_MINIMAL_UNDER"
)

type physicalWireSummary struct {
	factDigests           []domain.Digest
	processDigests        []domain.Digest
	rootDigests           []domain.Digest
	invocationDigests     []domain.Digest
	invocationFileDigests []domain.Digest
	executionBindings     []domain.Digest
}

func validatePhysicalWire(
	identity freshConfirmationIdentity,
	challenge, plan domain.Digest,
	confirmed compare.CandidateOutcomeMap,
) (physicalWireSummary, error) {
	count := identity.ConfirmationTrialCount
	lists := [][]string{
		identity.PhysicalFactBytesBase64, identity.PhysicalFactDigests, identity.ProcessDigests,
		identity.RootLayoutDigests, identity.InvocationReceiptDigests, identity.InvocationFileDigests,
	}
	for _, list := range lists {
		if len(list) != count {
			return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical fact list cardinality differs", nil)
		}
	}
	summary := physicalWireSummary{
		factDigests: make([]domain.Digest, count), processDigests: make([]domain.Digest, count),
		rootDigests: make([]domain.Digest, count), invocationDigests: make([]domain.Digest, count),
		invocationFileDigests: make([]domain.Digest, count), executionBindings: make([]domain.Digest, count),
	}
	attempts := make([]domain.Digest, count)
	worlds := make([]domain.Digest, count)
	candidateCounts := map[domain.CandidateExecutionKey]int{}
	roster := confirmed.CandidateRoster()
	if len(roster) == 0 || count%len(roster) != 0 {
		return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical candidate matrix cardinality is invalid", nil)
	}
	schedule, err := observe.NewPhaseRotatedSchedule(roster, count/len(roster), domain.AttemptConfirmation)
	if err != nil || !schedule.Valid() || schedule.TotalTrials() != count ||
		schedule.Digest().String() != identity.ConfirmationScheduleDigest ||
		schedule.StartOffset() != identity.ConfirmationScheduleOffset {
		return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical schedule identity differs", err)
	}
	scheduledTrials := schedule.Trials()
	for index := 0; index < count; index++ {
		factBytes, err := decodeBase64(identity.PhysicalFactBytesBase64[index], "physical fact")
		if err != nil {
			return physicalWireSummary{}, err
		}
		fact, err := world.InspectFreshExecutionFactWire(factBytes)
		if err != nil {
			return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical fact failed its world-owned strict parser", err)
		}
		factDigest, err := parseDigest(identity.PhysicalFactDigests[index], "physical fact")
		if err != nil {
			return physicalWireSummary{}, err
		}
		if fact.Digest() != factDigest {
			return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical fact digest differs", err)
		}
		wantNonce, nonceErr := confirmationNonce(challenge, index)
		candidate := fact.CandidateKey()
		if fact.PlanDigest() != plan || fact.StimulusDigest() != confirmed.StimulusDigest() ||
			fact.Purpose() != domain.AttemptConfirmation || fact.ScheduleOrdinal() != index || nonceErr != nil ||
			fact.InstanceNonce() != wantNonce || !candidateInRoster(roster, candidate) ||
			candidate != scheduledTrials[index].CandidateKey() {
			return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical fact lineage differs", nil)
		}
		attempts[index] = fact.AttemptArtifactDigest()
		worlds[index] = fact.WorldDigest()
		summary.processDigests[index] = fact.ProcessDigest()
		summary.rootDigests[index] = fact.RootLayoutDigest()
		summary.invocationDigests[index] = fact.InvocationReceiptDigest()
		summary.invocationFileDigests[index] = fact.InvocationFileDigest()
		summary.executionBindings[index] = fact.ExecutionBindingDigest()
		if summary.processDigests[index].String() != identity.ProcessDigests[index] ||
			summary.rootDigests[index].String() != identity.RootLayoutDigests[index] ||
			summary.invocationDigests[index].String() != identity.InvocationReceiptDigests[index] ||
			summary.invocationFileDigests[index].String() != identity.InvocationFileDigests[index] ||
			!summary.executionBindings[index].Valid() {
			return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical summary lists differ", nil)
		}
		summary.factDigests[index] = factDigest
		candidateCounts[candidate]++
	}
	for _, candidate := range roster {
		if candidateCounts[candidate] != count/len(roster) {
			return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical candidate matrix is incomplete", nil)
		}
	}
	if duplicateDigest(summary.factDigests) || duplicateDigest(summary.processDigests) || duplicateDigest(summary.rootDigests) ||
		duplicateDigest(summary.invocationDigests) || duplicateDigest(summary.invocationFileDigests) {
		return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical evidence was reused", nil)
	}
	sortDigests(attempts)
	sortDigests(worlds)
	if !sameDigests(attempts, confirmed.EvidenceAttemptDigests()) || !sameDigests(worlds, confirmed.EvidenceWorldDigests()) {
		return physicalWireSummary{}, refuse("INVALID_FRESH_CONFIRMATION_RECORD", "physical fact coverage differs from map", nil)
	}
	return summary, nil
}

func buildPriorEvidenceLedgerRecord(
	record reduce.ReductionRunRecord,
	original, reduced compare.CandidateOutcomeMap,
) (domain.Digest, error) {
	ledger := priorEvidenceLedger{
		batch: map[domain.Digest]struct{}{}, attempt: map[domain.Digest]struct{}{},
		world: map[domain.Digest]struct{}{}, observation: map[domain.Digest]struct{}{},
	}
	ledger.addMap(original)
	for _, entry := range record.Transcript().Entries() {
		addDigests(ledger.batch, entry.BatchDigests())
		addDigests(ledger.attempt, entry.AttemptDigests())
		addDigests(ledger.world, entry.WorldDigests())
		addDigests(ledger.observation, entry.ObservationDigests())
	}
	ledger.addMap(reduced)
	identity := struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Batch         []string `json:"batch_digests"`
		Attempt       []string `json:"attempt_digests"`
		World         []string `json:"world_digests"`
		Observation   []string `json:"observation_digests"`
	}{domain.SchemaVersion, "PriorEvidenceLedger", setStrings(ledger.batch), setStrings(ledger.attempt), setStrings(ledger.world), setStrings(ledger.observation)}
	digestRaw, _, err := canon.DigestTyped("PriorEvidenceLedger", identity)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digestRaw.String())
}

func rejectRecordOverlap(
	record reduce.ReductionRunRecord,
	original, reduced, confirmed compare.CandidateOutcomeMap,
) error {
	ledger := priorEvidenceLedger{
		batch: map[domain.Digest]struct{}{}, attempt: map[domain.Digest]struct{}{},
		world: map[domain.Digest]struct{}{}, observation: map[domain.Digest]struct{}{},
	}
	ledger.addMap(original)
	for _, entry := range record.Transcript().Entries() {
		addDigests(ledger.batch, entry.BatchDigests())
		addDigests(ledger.attempt, entry.AttemptDigests())
		addDigests(ledger.world, entry.WorldDigests())
		addDigests(ledger.observation, entry.ObservationDigests())
	}
	ledger.addMap(reduced)
	return ledger.rejectMap(confirmed)
}

func decodeBase64(raw, label string) ([]byte, error) {
	value, err := base64.StdEncoding.Strict().DecodeString(raw)
	if err != nil || len(value) == 0 {
		return nil, refuse("INVALID_FRESH_CONFIRMATION_RECORD", label+" base64 is invalid", err)
	}
	if base64.StdEncoding.EncodeToString(value) != raw {
		return nil, refuse("INVALID_FRESH_CONFIRMATION_RECORD", label+" base64 is noncanonical", nil)
	}
	return value, nil
}

func parseDigest(raw, label string) (domain.Digest, error) {
	digest, err := domain.ParseDigest(raw)
	if err != nil {
		return "", refuse("INVALID_FRESH_CONFIRMATION_RECORD", label+" digest is invalid", err)
	}
	return digest, nil
}

func sameStringDigests(raw []string, values []domain.Digest) bool {
	if len(raw) != len(values) {
		return false
	}
	for index, value := range values {
		if raw[index] != value.String() {
			return false
		}
	}
	return true
}

func duplicateDigest(values []domain.Digest) bool {
	seen := make(map[domain.Digest]struct{}, len(values))
	for _, value := range values {
		if _, duplicate := seen[value]; duplicate {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func candidateInRoster(roster []domain.CandidateExecutionKey, candidate domain.CandidateExecutionKey) bool {
	index := sort.Search(len(roster), func(index int) bool { return roster[index].String() >= candidate.String() })
	return index < len(roster) && roster[index] == candidate
}

func cloneRecord(input Record) Record {
	input.canonicalBytes = append([]byte(nil), input.canonicalBytes...)
	input.reductionGradeBytes = append([]byte(nil), input.reductionGradeBytes...)
	input.projectionProofs = cloneProjectionProofs(input.projectionProofs)
	input.physicalFactDigests = append([]domain.Digest(nil), input.physicalFactDigests...)
	input.processDigests = append([]domain.Digest(nil), input.processDigests...)
	input.rootLayoutDigests = append([]domain.Digest(nil), input.rootLayoutDigests...)
	input.invocationDigests = append([]domain.Digest(nil), input.invocationDigests...)
	input.invocationFileDigests = append([]domain.Digest(nil), input.invocationFileDigests...)
	input.executionBindings = append([]domain.Digest(nil), input.executionBindings...)
	return input
}

func (r Record) Valid() bool {
	if !r.digest.Valid() || len(r.canonicalBytes) == 0 || !r.planDigest.Valid() ||
		!r.originalBaseline.ArtifactDigest().Valid() || !r.reducedArtifact.Valid() || !r.confirmedArtifact.Valid() ||
		!r.challengeDigest.Valid() || !r.priorLedgerDigest.Valid() || !r.reductionRun.Digest().Valid() ||
		!r.reductionGradeDigest.Valid() || len(r.reductionGradeBytes) == 0 || len(r.projectionProofs) < 2 ||
		len(r.physicalFactDigests) < 2 || len(r.executionBindings) != len(r.physicalFactDigests) {
		return false
	}
	parsed, err := ParseRecord(r.canonicalBytes)
	return err == nil && parsed.digest == r.digest && bytes.Equal(parsed.canonicalBytes, r.canonicalBytes) &&
		sameDigestsInOrder(parsed.executionBindings, r.executionBindings)
}

func (r Record) Digest() domain.Digest                                  { return r.digest }
func (r Record) CanonicalBytes() []byte                                 { return append([]byte(nil), r.canonicalBytes...) }
func (r Record) PlanDigest() domain.Digest                              { return r.planDigest }
func (r Record) ReducedArtifactDigest() compare.OutcomeArtifactDigest   { return r.reducedArtifact }
func (r Record) ConfirmedArtifactDigest() compare.OutcomeArtifactDigest { return r.confirmedArtifact }
func (r Record) ChallengeDigest() domain.Digest                         { return r.challengeDigest }
func (r Record) OriginalBaselineMap() compare.CandidateOutcomeMap       { return r.originalBaseline }
func (r Record) ReducedBaselineMap() compare.CandidateOutcomeMap        { return r.reducedBaseline }
func (r Record) ConfirmedMap() compare.CandidateOutcomeMap              { return r.confirmed }
func (r Record) ReductionRunRecord() reduce.ReductionRunRecord          { return r.reductionRun }
func (r Record) ReductionRunDigest() domain.Digest                      { return r.reductionRun.Digest() }
func (r Record) ReductionGradeDigest() domain.Digest                    { return r.reductionGradeDigest }
func (r Record) ReductionGradeCanonicalBytes() []byte {
	return append([]byte(nil), r.reductionGradeBytes...)
}
func (r Record) ReductionGradeStatus() string { return r.reductionGradeStatus }
func (r Record) ProjectionProofs() []ProjectionProof {
	return cloneProjectionProofs(r.projectionProofs)
}
func (r Record) PhysicalFactDigests() []domain.Digest {
	return append([]domain.Digest(nil), r.physicalFactDigests...)
}
func (r Record) ExecutionBindingDigests() []domain.Digest {
	return append([]domain.Digest(nil), r.executionBindings...)
}

func sameDigestsInOrder(left, right []domain.Digest) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func cloneProjectionProofs(input []ProjectionProof) []ProjectionProof {
	result := make([]ProjectionProof, len(input))
	for index, proof := range input {
		result[index] = ProjectionProof{
			candidate: proof.candidate, fingerprint: proof.fingerprint,
			canonical: append([]byte(nil), proof.canonical...),
		}
	}
	return result
}
