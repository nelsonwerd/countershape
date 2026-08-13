package clistudy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/reference/app"
)

const (
	evidenceArtifactSchema = "countershape/u7-study-artifact/v1"
	evidenceTrialSchema    = "countershape/u7-study-trial/v1"
	evidenceDomainCLI      = "cli"
	evidenceStatusGreen    = "GREEN"
	maximumSafeJSONInteger = int64(1<<53 - 1)
	maximumEvidenceFile    = 16 << 20
	maximumEvidenceTree    = 64 << 20
)

// evidencePayload is presentation data derived after the corresponding core
// value has already been constructed and revalidated. It is never accepted as
// semantic input and cannot authorize a completed study by itself.
type evidencePayload map[string]any

type deterministicEvidencePayloads struct {
	sourceSpec     evidencePayload
	worldPlan      evidencePayload
	ruling         evidencePayload
	decisionRecord evidencePayload
	contractBundle evidencePayload
}

type freshEvidencePayloads struct {
	worldInstance  evidencePayload
	attempts       evidencePayload
	measurements   evidencePayload
	captures       evidencePayload
	confirmation   evidencePayload
	target         evidencePayload
	finalized      evidencePayload
	classification evidencePayload
}

// phaseEvidenceReceipt remains package-private so callers cannot mint GREEN
// phase rows from arbitrary hash strings. The completed workflow derives each
// receipt from one retained typed physical or execution authority.
type phaseEvidenceReceipt struct {
	phase     string
	trial     int
	authority domain.Digest
}

type evidencePublicationInput struct {
	ordinal              int
	physicalRunAuthority domain.Digest
	phaseTrials          []phaseEvidenceReceipt
	deterministic        deterministicEvidencePayloads
	fresh                freshEvidencePayloads
}

// EvidenceManifestEntry is one exact terminal file projection. It carries no
// semantic interpretation; the SHA-256 is over the exact canonical JSON line.
type EvidenceManifestEntry struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// publicationProjection is the private byte-and-filesystem result returned by
// the evidence writer. It deliberately carries no study-semantic validity:
// only publishCompletedEvidence may merge it with an independently validated
// StudyInspection.
type publicationProjection struct {
	ordinal              int
	physicalRunAuthority domain.Digest
	manifest             []EvidenceManifestEntry
	manifestSHA256       string
	deterministic        map[string]string
	fresh                map[string]string
}

type studyInspectionSeal struct{ marker byte }

var issuedStudyInspection = &studyInspectionSeal{marker: 1}

type publicationReadyCLIStudySeal struct{ marker byte }

var issuedPublicationReadyCLIStudy = &publicationReadyCLIStudySeal{marker: 1}

// publicationReadyCLIStudy is minted only by the completion path after its
// full static validation and terminal live official replay. It carries a
// previously validated defensive inspection plus the exact private evidence
// input, so the machine publication path cannot accept a raw or forged
// CompletedCLIStudy while avoiding a duplicate semantic replay.
type publicationReadyCLIStudy struct {
	ordinal              int
	physicalRunAuthority domain.Digest
	inspection           StudyInspection
	boundInspection      StudyInspection
	input                evidencePublicationInput
	boundInput           evidencePublicationInput
	seal                 *publicationReadyCLIStudySeal
}

func (ready publicationReadyCLIStudy) bound() bool {
	return ready.seal == issuedPublicationReadyCLIStudy && ready.ordinal >= 1 && ready.ordinal <= 3 &&
		ready.physicalRunAuthority.Valid() && ready.inspection.seal == issuedStudyInspection &&
		ready.inspection.ordinal == ready.ordinal &&
		ready.inspection.physicalRunAuthority == ready.physicalRunAuthority &&
		ready.inspection.publication.empty() && ready.input.ordinal == ready.ordinal &&
		ready.input.physicalRunAuthority == ready.physicalRunAuthority && len(ready.input.phaseTrials) == 98 &&
		reflect.DeepEqual(ready.inspection, ready.boundInspection) &&
		reflect.DeepEqual(ready.input, ready.boundInput)
}

func (ready publicationReadyCLIStudy) valid() bool {
	return ready.bound() && ready.inspection.Valid()
}

// ArtifactFact is one immutable semantic artifact retained by the completed
// study. CanonicalBytes returns a defensive copy.
type ArtifactFact struct {
	name      string
	authority domain.Digest
	canonical []byte
}

func (fact ArtifactFact) Name() string           { return fact.name }
func (fact ArtifactFact) Authority() string      { return fact.authority.String() }
func (fact ArtifactFact) CanonicalBytes() []byte { return append([]byte(nil), fact.canonical...) }

// PhaseFact is one exact logical-trial owner. Component authorities are empty
// only when that component does not exist for the named kind.
type PhaseFact struct {
	phase       string
	trial       int
	kind        string
	owner       domain.Digest
	world       domain.Digest
	attempt     domain.Digest
	measurement domain.Digest
	capture     domain.Digest
	projection  domain.Digest
}

func (fact PhaseFact) Phase() string                { return fact.phase }
func (fact PhaseFact) Trial() int                   { return fact.trial }
func (fact PhaseFact) Kind() string                 { return fact.kind }
func (fact PhaseFact) OwnerAuthority() string       { return fact.owner.String() }
func (fact PhaseFact) WorldAuthority() string       { return fact.world.String() }
func (fact PhaseFact) AttemptAuthority() string     { return fact.attempt.String() }
func (fact PhaseFact) MeasurementAuthority() string { return fact.measurement.String() }
func (fact PhaseFact) CaptureAuthority() string     { return fact.capture.String() }
func (fact PhaseFact) ProjectionAuthority() string  { return fact.projection.String() }

// OfficialFact is one fresh target, run, and classification graph projected
// from an inferred typed runner result. It never accepts a requested outcome.
type OfficialFact struct {
	trial                         int
	attempt                       domain.Digest
	target                        domain.Digest
	targetCanonicalSHA256         string
	bundle                        domain.Digest
	residueHead                   domain.Digest
	run                           domain.Digest
	runCanonicalSHA256            string
	classification                domain.Digest
	classificationCanonicalSHA256 string
	result                        string
	exitCode                      int
	recoveryEqual                 bool
}

func (fact OfficialFact) Trial() int                      { return fact.trial }
func (fact OfficialFact) AttemptAuthority() string        { return fact.attempt.String() }
func (fact OfficialFact) TargetAuthority() string         { return fact.target.String() }
func (fact OfficialFact) TargetCanonicalSHA256() string   { return fact.targetCanonicalSHA256 }
func (fact OfficialFact) BundleAuthority() string         { return fact.bundle.String() }
func (fact OfficialFact) ResidueHeadAuthority() string    { return fact.residueHead.String() }
func (fact OfficialFact) RunAuthority() string            { return fact.run.String() }
func (fact OfficialFact) RunCanonicalSHA256() string      { return fact.runCanonicalSHA256 }
func (fact OfficialFact) ClassificationAuthority() string { return fact.classification.String() }
func (fact OfficialFact) ClassificationCanonicalSHA256() string {
	return fact.classificationCanonicalSHA256
}
func (fact OfficialFact) Result() string      { return fact.result }
func (fact OfficialFact) ExitCode() int       { return fact.exitCode }
func (fact OfficialFact) RecoveryEqual() bool { return fact.recoveryEqual }

// ReductionFact retains the two deliberately different reduction outcomes:
// the incomplete main BEST_KNOWN run and the independently completed,
// durably swept auxiliary ONE_MINIMAL_UNDER run.
type ReductionFact struct {
	name              string
	run               domain.Digest
	transcript        domain.Digest
	grade             domain.Digest
	status            string
	reducerSet        domain.Digest
	minimizedStimulus domain.Digest
	finalSweepState   string
	completedSweep    domain.Digest
	gradeLimitations  []string
}

func (fact ReductionFact) Name() string                       { return fact.name }
func (fact ReductionFact) RunAuthority() string               { return fact.run.String() }
func (fact ReductionFact) TranscriptAuthority() string        { return fact.transcript.String() }
func (fact ReductionFact) GradeAuthority() string             { return fact.grade.String() }
func (fact ReductionFact) Status() string                     { return fact.status }
func (fact ReductionFact) ReducerSetAuthority() string        { return fact.reducerSet.String() }
func (fact ReductionFact) MinimizedStimulusAuthority() string { return fact.minimizedStimulus.String() }
func (fact ReductionFact) FinalSweepState() string            { return fact.finalSweepState }
func (fact ReductionFact) CompletedSweepAuthority() string    { return fact.completedSweep.String() }
func (fact ReductionFact) GradeLimitations() []string {
	return append([]string(nil), fact.gradeLimitations...)
}

var expectedInspectionArtifacts = []string{
	"source-spec",
	"world-plan",
	"ruling",
	"decision-record",
	"contract-bundle",
}

var expectedInspectionNonclaims = []string{
	"BROAD_IMPORTED_REPOSITORY_BEHAVIOR_UNVALIDATED",
	"CLI_OFFICIAL_EXECUTION_GENERALIZATION_UNVALIDATED",
	"FULL_STUDY_RESOURCE_BOUND_UNVALIDATED",
	"FULL_WORKFLOW_RACE_FREEDOM_UNVALIDATED",
	"HOSTILE_CONTAINMENT_UNVALIDATED",
	"NETWORK_DENIAL_UNVALIDATED",
	"POST_SPAWN_OPERATOR_FAILURE_TEXT_UNVALIDATED",
	"RUNTIME_ESCAPED_WRITE_BEHAVIOR_UNPROVEN",
}

// StudyInspection is the defensive semantic snapshot available to U7D.
// Inspect returns a valid unpublished value. Publication only adds an exact
// terminal byte projection; it never upgrades or replaces semantic facts.
type StudyInspection struct {
	ordinal              int
	physicalRunAuthority domain.Digest
	artifacts            []ArtifactFact
	phaseFacts           []PhaseFact
	officialFacts        []OfficialFact
	reductionFacts       []ReductionFact
	nonclaims            []string
	publication          publicationProjection
	seal                 *studyInspectionSeal
}

// Inspect revalidates the opaque completed study and returns a defensive,
// unpublished semantic projection. The later evidence writer can only attach
// terminal filesystem facts to this projection; it cannot construct or repair
// any of the semantic authorities below.
func Inspect(completed CompletedCLIStudy) (StudyInspection, error) {
	if err := validateCompletedCLIStudy(completed); err != nil {
		return StudyInspection{}, fmt.Errorf("CLI_STUDY_INSPECTION_REFUSED: %w", err)
	}
	inspection, err := projectCompletedInspection(completed)
	if err != nil || !inspection.Valid() || !inspection.publication.empty() {
		return StudyInspection{}, errors.Join(err, fmt.Errorf("CLI_STUDY_INSPECTION_PROJECTION_REFUSED"))
	}
	return inspection, nil
}

// projectCompletedInspection is an inert projector. Public Inspect validates
// its result; the machine path binds it into a private witness that validates
// immediately before any evidence write.
func projectCompletedInspection(completed CompletedCLIStudy) (StudyInspection, error) {
	phaseFacts := make([]PhaseFact, len(completed.phaseFacts))
	for index, fact := range completed.phaseFacts {
		phaseFacts[index] = PhaseFact{
			phase: fact.phase, trial: fact.trial, kind: fact.kind,
			owner: fact.authority, world: fact.world, attempt: fact.attempt,
			measurement: fact.measurement, capture: fact.capture, projection: fact.projection,
		}
	}
	officialFacts := make([]OfficialFact, len(completed.officialTrials))
	for index, trial := range completed.officialTrials {
		officialFacts[index] = OfficialFact{
			trial: index + 1, attempt: trial.attemptDigest, target: trial.targetDigest,
			targetCanonicalSHA256: trial.targetCanonicalSHA256,
			bundle:                trial.bundleDigest, residueHead: trial.residueHeadDigest,
			run: trial.runDigest, runCanonicalSHA256: trial.runCanonicalSHA256,
			classification:                trial.classificationDigest,
			classificationCanonicalSHA256: trial.classificationCanonicalSHA256,
			result:                        trial.result, exitCode: trial.exitCode, recoveryEqual: trial.recoveryEqual,
		}
	}
	weakGrade := completed.weakGrade.Grade()
	strongGrade := completed.strongGrade.Grade()
	strongSweep, strongSweepPresent := strongGrade.CompletedSweepDigest()
	if !strongSweepPresent {
		return StudyInspection{}, fmt.Errorf("CLI_STUDY_INSPECTION_STRONG_SWEEP_REFUSED")
	}
	reductionFacts := []ReductionFact{
		{
			name: "main-weak", run: completed.weakRun.Digest(), transcript: completed.weakRun.Transcript().Digest(),
			grade: weakGrade.Digest(), status: string(weakGrade.Status()), reducerSet: completed.weakRun.ReducerSet().Digest(),
			minimizedStimulus: completed.weakRun.MinimizedStimulusDigest(),
			finalSweepState:   string(completed.weakRun.Transcript().FinalSweepState()),
			gradeLimitations:  weakGrade.Limitations(),
		},
		{
			name: "auxiliary-strong", run: completed.strongRun.Digest(), transcript: completed.strongRun.Transcript().Digest(),
			grade: strongGrade.Digest(), status: string(strongGrade.Status()), reducerSet: completed.strongRun.ReducerSet().Digest(),
			minimizedStimulus: completed.strongRun.MinimizedStimulusDigest(),
			finalSweepState:   string(completed.strongRun.Transcript().FinalSweepState()), completedSweep: strongSweep,
			gradeLimitations: strongGrade.Limitations(),
		},
	}
	inspection := StudyInspection{
		ordinal: completed.ordinal, physicalRunAuthority: completed.physicalRunAuthority,
		artifacts: []ArtifactFact{
			{name: "source-spec", authority: completed.confirmed.SourceSpecDigest, canonical: append([]byte(nil), completed.confirmed.SourceSpecBytes...)},
			{name: "world-plan", authority: completed.confirmed.Plan.Digest(), canonical: completed.confirmed.Plan.CanonicalBytes()},
			{name: "ruling", authority: completed.decision.Digest(), canonical: completed.durableRuling.Record().CanonicalBytes()},
			{name: "decision-record", authority: completed.decision.Digest(), canonical: completed.decision.CanonicalBytes()},
			{name: "contract-bundle", authority: completed.bundle.Digest(), canonical: completed.bundle.CanonicalBytes()},
		},
		phaseFacts: phaseFacts, officialFacts: officialFacts, reductionFacts: reductionFacts,
		nonclaims: append([]string(nil), expectedInspectionNonclaims...), seal: issuedStudyInspection,
	}
	return inspection, nil
}

func (inspection StudyInspection) Valid() bool {
	if inspection.seal != issuedStudyInspection || inspection.ordinal < 1 || inspection.ordinal > 3 ||
		!inspection.physicalRunAuthority.Valid() || len(inspection.artifacts) != len(expectedInspectionArtifacts) ||
		len(inspection.phaseFacts) != 98 || len(inspection.officialFacts) != 10 || len(inspection.reductionFacts) != 2 ||
		!slices.Equal(inspection.nonclaims, expectedInspectionNonclaims) {
		return false
	}
	weak, strong := inspection.reductionFacts[0], inspection.reductionFacts[1]
	if weak.name != "main-weak" || strong.name != "auxiliary-strong" ||
		!weak.run.Valid() || !weak.transcript.Valid() || !weak.grade.Valid() || !weak.reducerSet.Valid() ||
		!weak.minimizedStimulus.Valid() || weak.completedSweep.Valid() || weak.status != "BEST_KNOWN" ||
		weak.finalSweepState != "INCOMPLETE" || len(weak.gradeLimitations) == 0 ||
		!strong.run.Valid() || !strong.transcript.Valid() || !strong.grade.Valid() || !strong.reducerSet.Valid() ||
		!strong.minimizedStimulus.Valid() || !strong.completedSweep.Valid() || strong.status != "ONE_MINIMAL_UNDER" ||
		strong.finalSweepState != "COMPLETE" || len(strong.gradeLimitations) != 0 ||
		weak.run == strong.run || weak.transcript == strong.transcript || weak.grade == strong.grade ||
		weak.minimizedStimulus != strong.minimizedStimulus {
		return false
	}
	seenArtifacts := make(map[string]struct{}, len(inspection.artifacts))
	for index, fact := range inspection.artifacts {
		if fact.name != expectedInspectionArtifacts[index] || !fact.authority.Valid() || len(fact.canonical) == 0 {
			return false
		}
		if _, duplicate := seenArtifacts[fact.authority.String()]; duplicate {
			if index != 3 || inspection.artifacts[2].name != "ruling" ||
				inspection.artifacts[2].authority != fact.authority ||
				!bytes.Equal(inspection.artifacts[2].canonical, fact.canonical) {
				return false
			}
		}
		seenArtifacts[fact.authority.String()] = struct{}{}
	}
	seenPhase := make(map[string]struct{}, len(inspection.phaseFacts))
	for index, fact := range inspection.phaseFacts {
		wantPhase, wantTrial := "contract", index-88+1
		if index < 80 {
			wantPhase, wantTrial = "search", index+1
		} else if index < 88 {
			wantPhase, wantTrial = "confirm", index-80+1
		}
		if fact.phase != wantPhase || fact.trial != wantTrial || fact.kind == "" || !fact.owner.Valid() ||
			fact.owner == inspection.physicalRunAuthority {
			return false
		}
		if _, duplicate := seenPhase[fact.owner.String()]; duplicate {
			return false
		}
		seenPhase[fact.owner.String()] = struct{}{}
	}
	seenOfficial := make(map[string]struct{}, len(inspection.officialFacts)*4)
	for index, fact := range inspection.officialFacts {
		if fact.trial != index+1 || !fact.attempt.Valid() || !fact.target.Valid() || !fact.bundle.Valid() ||
			!fact.residueHead.Valid() || !fact.run.Valid() || !fact.classification.Valid() ||
			!validSHA256(fact.targetCanonicalSHA256) || !validSHA256(fact.runCanonicalSHA256) ||
			!validSHA256(fact.classificationCanonicalSHA256) || fact.result != "CONFORMS" ||
			fact.exitCode != 0 || !fact.recoveryEqual ||
			inspection.phaseFacts[88+index].owner != fact.classification {
			return false
		}
		for _, authority := range []domain.Digest{fact.attempt, fact.target, fact.run, fact.classification} {
			if _, duplicate := seenOfficial[authority.String()]; duplicate {
				return false
			}
			seenOfficial[authority.String()] = struct{}{}
		}
	}
	if !inspection.publication.empty() {
		if !inspection.publication.valid() || inspection.publication.ordinal != inspection.ordinal ||
			inspection.publication.physicalRunAuthority != inspection.physicalRunAuthority {
			return false
		}
	}
	return true
}

func (inspection StudyInspection) Published() bool {
	return inspection.Valid() && inspection.publication.valid()
}

func (inspection StudyInspection) Ordinal() int { return inspection.ordinal }

func (inspection StudyInspection) PhysicalRunAuthority() string {
	return inspection.physicalRunAuthority.String()
}

func (inspection StudyInspection) Artifacts() []ArtifactFact {
	result := append([]ArtifactFact(nil), inspection.artifacts...)
	for index := range result {
		result[index].canonical = append([]byte(nil), result[index].canonical...)
	}
	return result
}

func (inspection StudyInspection) PhaseFacts() []PhaseFact {
	return append([]PhaseFact(nil), inspection.phaseFacts...)
}

func (inspection StudyInspection) OfficialFacts() []OfficialFact {
	return append([]OfficialFact(nil), inspection.officialFacts...)
}

func (inspection StudyInspection) ReductionFacts() []ReductionFact {
	result := append([]ReductionFact(nil), inspection.reductionFacts...)
	for index := range result {
		result[index].gradeLimitations = append([]string(nil), result[index].gradeLimitations...)
	}
	return result
}

func (inspection StudyInspection) Nonclaims() []string {
	return append([]string(nil), inspection.nonclaims...)
}

func (inspection StudyInspection) Manifest() []EvidenceManifestEntry {
	return append([]EvidenceManifestEntry(nil), inspection.publication.manifest...)
}

func (inspection StudyInspection) ManifestSHA256() string {
	return inspection.publication.manifestSHA256
}

func (inspection StudyInspection) DeterministicSHA256() map[string]string {
	return cloneStringMap(inspection.publication.deterministic)
}

func (inspection StudyInspection) FreshSHA256() map[string]string {
	return cloneStringMap(inspection.publication.fresh)
}

// mergeReadyStudyPublicationAfterValidation is called only after the same
// lexical publication path has checked ready.valid and the writer has returned
// a valid terminal projection. It repeats the scalar join and validates the
// final public inspection, but does not replay the already-bound token again.
func mergeReadyStudyPublicationAfterValidation(ready publicationReadyCLIStudy, projection publicationProjection) (StudyInspection, error) {
	if ready.seal != issuedPublicationReadyCLIStudy || ready.ordinal != projection.ordinal ||
		ready.physicalRunAuthority != projection.physicalRunAuthority {
		return StudyInspection{}, evidenceError("INSPECTION", "semantic and publication projections do not join")
	}
	inspection := ready.inspection
	inspection.publication = publicationProjection{
		ordinal: projection.ordinal, physicalRunAuthority: projection.physicalRunAuthority,
		manifest: append([]EvidenceManifestEntry(nil), projection.manifest...), manifestSHA256: projection.manifestSHA256,
		deterministic: cloneStringMap(projection.deterministic), fresh: cloneStringMap(projection.fresh),
	}
	if !inspection.Valid() {
		return StudyInspection{}, evidenceError("INSPECTION", "merged inspection is invalid")
	}
	return inspection, nil
}

func (projection publicationProjection) valid() bool {
	if projection.ordinal < 1 || projection.ordinal > 3 || !projection.physicalRunAuthority.Valid() ||
		len(projection.manifest) != 111 || !validSHA256(projection.manifestSHA256) ||
		len(projection.deterministic) != 5 || len(projection.fresh) != 8 ||
		validateDigestSeparation(projection.deterministic, projection.fresh) != nil {
		return false
	}
	expectedPaths := make([]string, 0, 111)
	fieldByPath := make(map[string]struct {
		field         string
		deterministic bool
	}, 13)
	for _, payload := range evidencePayloadRoster(evidencePublicationInput{}) {
		expectedPaths = append(expectedPaths, payload.path)
		fieldByPath[payload.path] = struct {
			field         string
			deterministic bool
		}{field: payload.field, deterministic: payload.deterministic}
	}
	for _, phase := range trialPhaseCounts {
		for trial := 1; trial <= phase.count; trial++ {
			expectedPaths = append(expectedPaths, fmt.Sprintf("phases/%s/trial-%03d.json", phase.phase, trial))
		}
	}
	sort.Strings(expectedPaths)
	seen := make(map[string]struct{}, len(projection.manifest))
	for index, entry := range projection.manifest {
		if entry.Path == "" || entry.Mode != "0600" || entry.Bytes < 1 ||
			entry.Bytes > maximumEvidenceFile || !validSHA256(entry.SHA256) || entry.Path != expectedPaths[index] {
			return false
		}
		if _, duplicate := seen[entry.Path]; duplicate {
			return false
		}
		seen[entry.Path] = struct{}{}
		if artifact, present := fieldByPath[entry.Path]; present {
			observed := projection.fresh[artifact.field]
			if artifact.deterministic {
				observed = projection.deterministic[artifact.field]
			}
			if observed != entry.SHA256 {
				return false
			}
		}
	}
	manifestLine, err := canonicalJSONLine(projection.manifest)
	if err != nil || sha256Hex(manifestLine) != projection.manifestSHA256 {
		return false
	}
	return true
}

func (projection publicationProjection) empty() bool {
	return projection.ordinal == 0 && !projection.physicalRunAuthority.Valid() &&
		len(projection.manifest) == 0 && projection.manifestSHA256 == "" &&
		len(projection.deterministic) == 0 && len(projection.fresh) == 0
}

type namedEvidencePayload struct {
	field         string
	path          string
	payload       evidencePayload
	deterministic bool
}

type deterministicArtifactLine struct {
	SchemaVersion string          `json:"schema_version"`
	Domain        string          `json:"domain"`
	Artifact      string          `json:"artifact"`
	Payload       json.RawMessage `json:"payload"`
}

type freshArtifactLine struct {
	SchemaVersion string          `json:"schema_version"`
	Domain        string          `json:"domain"`
	Ordinal       int             `json:"ordinal"`
	Artifact      string          `json:"artifact"`
	Payload       json.RawMessage `json:"payload"`
}

type trialArtifactLine struct {
	SchemaVersion string `json:"schema_version"`
	Domain        string `json:"domain"`
	Ordinal       int    `json:"ordinal"`
	Phase         string `json:"phase"`
	Trial         int    `json:"trial"`
	Status        string `json:"status"`
}

var evidenceDirectories = []string{
	"deterministic",
	"fresh",
	"phases",
	"phases/search",
	"phases/confirm",
	"phases/contract",
}

var trialPhaseCounts = []struct {
	phase string
	count int
}{
	{phase: "search", count: 80},
	{phase: "confirm", count: 8},
	{phase: "contract", count: 10},
}

// publishEvidenceInput writes the frozen protocol only through the app-owned
// root capability, then asks that same capability to prove the exact terminal
// roster. Only the completed-study adapter can construct the private input.
func publishEvidenceInput(ctx context.Context, input evidencePublicationInput, workspace *app.EvidenceWorkspace) (publicationProjection, error) {
	if ctx == nil {
		return publicationProjection{}, evidenceError("INPUT", "publication context is absent")
	}
	if err := ctx.Err(); err != nil {
		return publicationProjection{}, evidenceError("CANCELED", "before publication")
	}
	if input.ordinal < 1 || input.ordinal > 3 || workspace == nil || !input.physicalRunAuthority.Valid() {
		return publicationProjection{}, evidenceError("INPUT", "ordinal, workspace, or run authority is invalid")
	}
	planned, deterministicDigests, freshDigests, err := planEvidenceFiles(input)
	if err != nil {
		return publicationProjection{}, err
	}
	for _, relative := range evidenceDirectories {
		if err := ctx.Err(); err != nil {
			return publicationProjection{}, evidenceError("CANCELED", "before directory "+relative)
		}
		if err := workspace.CreateDirectory(relative); err != nil {
			return publicationProjection{}, evidenceError("DIRECTORY", relative+": "+err.Error())
		}
	}
	paths := make([]string, 0, len(planned))
	for relative := range planned {
		paths = append(paths, relative)
	}
	sort.Strings(paths)
	manifest := make([]EvidenceManifestEntry, 0, len(paths))
	workspaceFiles := make([]app.EvidenceFile, 0, len(paths))
	for _, relative := range paths {
		if err := ctx.Err(); err != nil {
			return publicationProjection{}, evidenceError("CANCELED", "before file "+relative)
		}
		exact := planned[relative]
		if err := workspace.WriteFileExclusive(relative, exact); err != nil {
			return publicationProjection{}, evidenceError("WRITE", relative+": "+err.Error())
		}
		digest := sha256Hex(exact)
		manifest = append(manifest, EvidenceManifestEntry{Path: relative, Mode: "0600", Bytes: int64(len(exact)), SHA256: digest})
		workspaceFiles = append(workspaceFiles, app.EvidenceFile{Path: relative, Bytes: int64(len(exact)), SHA256: digest})
	}
	if err := ctx.Err(); err != nil {
		return publicationProjection{}, evidenceError("CANCELED", "before terminal manifest verification")
	}
	if err := workspace.VerifyExactManifest(app.EvidenceManifest{
		Directories: append([]string(nil), evidenceDirectories...),
		Files:       workspaceFiles,
	}); err != nil {
		return publicationProjection{}, evidenceError("TERMINAL", err.Error())
	}
	if err := ctx.Err(); err != nil {
		return publicationProjection{}, evidenceError("CANCELED", "after terminal manifest verification")
	}
	manifestLine, err := canonicalJSONLine(manifest)
	if err != nil {
		return publicationProjection{}, evidenceError("ENCODE", "manifest")
	}
	projection := publicationProjection{
		ordinal: input.ordinal, physicalRunAuthority: input.physicalRunAuthority,
		manifest: append([]EvidenceManifestEntry(nil), manifest...), manifestSHA256: sha256Hex(manifestLine),
		deterministic: cloneStringMap(deterministicDigests), fresh: cloneStringMap(freshDigests),
	}
	if !projection.valid() {
		return publicationProjection{}, evidenceError("INSPECTION", "terminal projection is invalid")
	}
	return projection, nil
}

func planEvidenceFiles(input evidencePublicationInput) (map[string][]byte, map[string]string, map[string]string, error) {
	payloads := evidencePayloadRoster(input)
	if err := validatePhysicalRunAuthorityPayloads(input.physicalRunAuthority, payloads); err != nil {
		return nil, nil, nil, err
	}
	planned := make(map[string][]byte, 111)
	payloadBytes := make(map[string][]byte, len(payloads))
	payloadDigests := make(map[string]string, len(payloads))
	deterministicDigests := make(map[string]string, 5)
	freshDigests := make(map[string]string, 8)
	for _, entry := range payloads {
		raw, err := canonicalEvidencePayload(entry.field, entry.payload)
		if err != nil {
			return nil, nil, nil, err
		}
		digest := sha256Hex(raw)
		for priorField, prior := range payloadBytes {
			if bytes.Equal(prior, raw) || payloadDigests[priorField] == digest {
				return nil, nil, nil, evidenceError("PAYLOAD_ALIAS", entry.field+" duplicates "+priorField)
			}
		}
		payloadBytes[entry.field] = append([]byte(nil), raw...)
		payloadDigests[entry.field] = digest
		var exact []byte
		if entry.deterministic {
			exact, err = canonicalJSONLine(deterministicArtifactLine{
				SchemaVersion: evidenceArtifactSchema, Domain: evidenceDomainCLI, Artifact: entry.field, Payload: raw,
			})
		} else {
			exact, err = canonicalJSONLine(freshArtifactLine{
				SchemaVersion: evidenceArtifactSchema, Domain: evidenceDomainCLI, Ordinal: input.ordinal,
				Artifact: entry.field, Payload: raw,
			})
		}
		if err != nil {
			return nil, nil, nil, evidenceError("ENCODE", entry.field)
		}
		if len(exact) < 1 || len(exact) > maximumEvidenceFile {
			return nil, nil, nil, evidenceError("SIZE", entry.field)
		}
		if _, duplicate := planned[entry.path]; duplicate {
			return nil, nil, nil, evidenceError("ROSTER", "duplicate "+entry.path)
		}
		planned[entry.path] = exact
		if entry.deterministic {
			deterministicDigests[entry.field] = sha256Hex(exact)
		} else {
			freshDigests[entry.field] = sha256Hex(exact)
		}
	}
	phaseFiles, err := planPhaseTrialFiles(input.ordinal, input.phaseTrials)
	if err != nil {
		return nil, nil, nil, err
	}
	for relative, exact := range phaseFiles {
		if _, duplicate := planned[relative]; duplicate {
			return nil, nil, nil, evidenceError("ROSTER", "duplicate "+relative)
		}
		planned[relative] = exact
	}
	if len(planned) != 111 || len(deterministicDigests) != 5 || len(freshDigests) != 8 {
		return nil, nil, nil, evidenceError("ROSTER", fmt.Sprintf("constructed %d files", len(planned)))
	}
	totalBytes := 0
	for relative, exact := range planned {
		if len(exact) < 1 || len(exact) > maximumEvidenceFile || totalBytes > maximumEvidenceTree-len(exact) {
			return nil, nil, nil, evidenceError("SIZE", relative)
		}
		totalBytes += len(exact)
	}
	if err := validateDigestSeparation(deterministicDigests, freshDigests); err != nil {
		return nil, nil, nil, err
	}
	return planned, deterministicDigests, freshDigests, nil
}

func planPhaseTrialFiles(ordinal int, receipts []phaseEvidenceReceipt) (map[string][]byte, error) {
	expected := 0
	for _, phase := range trialPhaseCounts {
		expected += phase.count
	}
	if len(receipts) != expected {
		return nil, evidenceError("PHASE_RECEIPTS", fmt.Sprintf("received %d receipts, want %d", len(receipts), expected))
	}
	planned := make(map[string][]byte, expected)
	seen := make(map[string]string, expected)
	index := 0
	for _, phase := range trialPhaseCounts {
		for trial := 1; trial <= phase.count; trial++ {
			receipt := receipts[index]
			index++
			relative := fmt.Sprintf("phases/%s/trial-%03d.json", phase.phase, trial)
			if receipt.phase != phase.phase || receipt.trial != trial || !receipt.authority.Valid() {
				return nil, evidenceError("PHASE_RECEIPTS", relative)
			}
			raw, err := rawSHA256(receipt.authority)
			if err != nil || !validSHA256(raw) {
				return nil, evidenceError("PHASE_AUTHORITY", relative)
			}
			if prior, duplicate := seen[raw]; duplicate {
				return nil, evidenceError("PHASE_AUTHORITY_ALIAS", relative+" duplicates "+prior)
			}
			seen[raw] = relative
			exact, err := canonicalJSONLine(trialArtifactLine{
				SchemaVersion: evidenceTrialSchema, Domain: evidenceDomainCLI, Ordinal: ordinal,
				Phase: phase.phase, Trial: trial, Status: evidenceStatusGreen,
			})
			if err != nil {
				return nil, evidenceError("ENCODE", relative)
			}
			planned[relative] = exact
		}
	}
	return planned, nil
}

func evidencePayloadRoster(input evidencePublicationInput) []namedEvidencePayload {
	return []namedEvidencePayload{
		{field: "source_spec_sha256", path: "deterministic/source-spec.json", payload: input.deterministic.sourceSpec, deterministic: true},
		{field: "world_plan_sha256", path: "deterministic/world-plan.json", payload: input.deterministic.worldPlan, deterministic: true},
		{field: "ruling_sha256", path: "deterministic/ruling.json", payload: input.deterministic.ruling, deterministic: true},
		{field: "decision_record_sha256", path: "deterministic/decision-record.json", payload: input.deterministic.decisionRecord, deterministic: true},
		{field: "contract_bundle_sha256", path: "deterministic/contract-bundle.json", payload: input.deterministic.contractBundle, deterministic: true},
		{field: "world_instance_sha256", path: "fresh/world-instance.json", payload: input.fresh.worldInstance},
		{field: "attempts_sha256", path: "fresh/attempts.json", payload: input.fresh.attempts},
		{field: "measurements_sha256", path: "fresh/measurements.json", payload: input.fresh.measurements},
		{field: "captures_sha256", path: "fresh/captures.json", payload: input.fresh.captures},
		{field: "confirmation_sha256", path: "fresh/confirmation.json", payload: input.fresh.confirmation},
		{field: "contract_execution_target_sha256", path: "fresh/contract-execution-target.json", payload: input.fresh.target},
		{field: "finalized_contract_run_sha256", path: "fresh/finalized-contract-run.json", payload: input.fresh.finalized},
		{field: "contract_execution_sha256", path: "fresh/contract-execution.json", payload: input.fresh.classification},
	}
}

func validatePhysicalRunAuthorityPayloads(authority domain.Digest, payloads []namedEvidencePayload) error {
	if !authority.Valid() {
		return evidenceError("PHYSICAL_RUN_AUTHORITY", "invalid completed-run authority")
	}
	want := authority.String()
	for _, entry := range payloads {
		observed, present := entry.payload["physical_run_authority"]
		if entry.deterministic {
			if present {
				return evidenceError("PHYSICAL_RUN_AUTHORITY", entry.field+" must remain run-independent")
			}
			continue
		}
		value, ok := observed.(string)
		if !present || !ok || value != want {
			return evidenceError("PHYSICAL_RUN_AUTHORITY", entry.field+" is not bound to the completed run")
		}
	}
	return nil
}

func canonicalEvidencePayload(field string, payload evidencePayload) ([]byte, error) {
	if len(payload) == 0 {
		return nil, evidenceError("PAYLOAD", field+" is empty")
	}
	if err := validateJSONValue(payload, 0); err != nil {
		return nil, evidenceError("PAYLOAD", field+": "+err.Error())
	}
	exact, err := canonicalJSONObject(payload)
	if err != nil || len(exact) < 2 || exact[0] != '{' || exact[len(exact)-1] != '}' {
		return nil, evidenceError("PAYLOAD", field+" is not one canonical object")
	}
	return exact, nil
}

func validateJSONValue(value any, depth int) error {
	if depth > 32 {
		return fmt.Errorf("nesting exceeds 32")
	}
	switch typed := value.(type) {
	case nil, bool:
		return nil
	case string:
		if !utf8.ValidString(typed) || strings.ContainsRune(typed, '\x00') {
			return fmt.Errorf("invalid string")
		}
		return nil
	case int:
		if int64(typed) < -maximumSafeJSONInteger || int64(typed) > maximumSafeJSONInteger {
			return fmt.Errorf("unsafe integer")
		}
		return nil
	case int8, int16, int32, int64:
		value := reflect.ValueOf(typed).Int()
		if value < -maximumSafeJSONInteger || value > maximumSafeJSONInteger {
			return fmt.Errorf("unsafe integer")
		}
		return nil
	case uint, uint8, uint16, uint32, uint64:
		value := reflect.ValueOf(typed).Uint()
		if value > uint64(maximumSafeJSONInteger) {
			return fmt.Errorf("unsafe integer")
		}
		return nil
	case float32:
		return validateJSONFloat(float64(typed))
	case float64:
		return validateJSONFloat(typed)
	case []string:
		for _, item := range typed {
			if err := validateJSONValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	case []int:
		for _, item := range typed {
			if err := validateJSONValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	case []any:
		for _, item := range typed {
			if err := validateJSONValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	case map[string]string:
		for key, item := range typed {
			if err := validateJSONKey(key); err != nil {
				return err
			}
			if err := validateJSONValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		for key, item := range typed {
			if err := validateJSONKey(key); err != nil {
				return err
			}
			if err := validateJSONValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	case evidencePayload:
		for key, item := range typed {
			if err := validateJSONKey(key); err != nil {
				return err
			}
			if err := validateJSONValue(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported JSON type %T", value)
	}
}

func validateJSONKey(key string) error {
	if key == "" || !utf8.ValidString(key) || strings.ContainsRune(key, '\x00') {
		return fmt.Errorf("invalid object key")
	}
	return nil
}

func validateJSONFloat(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value == 0 && math.Signbit(value) || math.Trunc(value) != value ||
		value < -float64(maximumSafeJSONInteger) || value > float64(maximumSafeJSONInteger) {
		return fmt.Errorf("unsafe number")
	}
	return nil
}

func canonicalJSONObject(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	exact := output.Bytes()
	if len(exact) == 0 || exact[len(exact)-1] != '\n' {
		return nil, fmt.Errorf("missing encoder newline")
	}
	return append([]byte(nil), exact[:len(exact)-1]...), nil
}

func canonicalJSONLine(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	exact := output.Bytes()
	if len(exact) == 0 || exact[len(exact)-1] != '\n' || bytes.Count(exact, []byte{'\n'}) != 1 {
		return nil, fmt.Errorf("value is not one canonical JSON line")
	}
	return append([]byte(nil), exact...), nil
}

func validateDigestSeparation(deterministic, fresh map[string]string) error {
	seen := make(map[string]string, len(deterministic)+len(fresh))
	for _, group := range []map[string]string{deterministic, fresh} {
		for field, digest := range group {
			if !validSHA256(digest) {
				return evidenceError("DIGEST", field)
			}
			if prior, duplicate := seen[digest]; duplicate {
				return evidenceError("DIGEST_ALIAS", field+" duplicates "+prior)
			}
			seen[digest] = field
		}
	}
	return nil
}

func rawSHA256(value domain.Digest) (string, error) {
	raw := strings.TrimPrefix(value.String(), "sha256:")
	if !validSHA256(raw) {
		return "", fmt.Errorf("invalid SHA-256 authority")
	}
	return raw, nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return false
	}
	nonzero := false
	for _, current := range decoded {
		if current != 0 {
			nonzero = true
		}
	}
	return nonzero && strings.ToLower(value) == value
}

func sha256Hex(exact []byte) string {
	sum := sha256.Sum256(exact)
	return hex.EncodeToString(sum[:])
}

func cloneStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func cloneStudyInspection(input StudyInspection) StudyInspection {
	result := input
	result.artifacts = make([]ArtifactFact, len(input.artifacts))
	for index, fact := range input.artifacts {
		result.artifacts[index] = fact
		result.artifacts[index].canonical = append([]byte(nil), fact.canonical...)
	}
	result.phaseFacts = append([]PhaseFact(nil), input.phaseFacts...)
	result.officialFacts = append([]OfficialFact(nil), input.officialFacts...)
	result.reductionFacts = make([]ReductionFact, len(input.reductionFacts))
	for index, fact := range input.reductionFacts {
		result.reductionFacts[index] = fact
		result.reductionFacts[index].gradeLimitations = append([]string(nil), fact.gradeLimitations...)
	}
	result.nonclaims = append([]string(nil), input.nonclaims...)
	result.publication = publicationProjection{
		ordinal: input.publication.ordinal, physicalRunAuthority: input.publication.physicalRunAuthority,
		manifest:       append([]EvidenceManifestEntry(nil), input.publication.manifest...),
		manifestSHA256: input.publication.manifestSHA256,
		deterministic:  cloneStringMap(input.publication.deterministic),
		fresh:          cloneStringMap(input.publication.fresh),
	}
	return result
}

func cloneEvidencePublicationInput(input evidencePublicationInput) (evidencePublicationInput, error) {
	result := evidencePublicationInput{
		ordinal: input.ordinal, physicalRunAuthority: input.physicalRunAuthority,
		phaseTrials: append([]phaseEvidenceReceipt(nil), input.phaseTrials...),
	}
	clonePayload := func(payload evidencePayload) (evidencePayload, error) {
		cloned := make(evidencePayload, len(payload))
		for key, value := range payload {
			switch typed := value.(type) {
			case string:
				cloned[key] = typed
			case int:
				cloned[key] = typed
			case bool:
				cloned[key] = typed
			case []string:
				cloned[key] = append([]string(nil), typed...)
			default:
				return nil, evidenceError("INPUT", "unsupported publication witness value")
			}
		}
		return cloned, nil
	}
	var err error
	result.deterministic.sourceSpec, err = clonePayload(input.deterministic.sourceSpec)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.deterministic.worldPlan, err = clonePayload(input.deterministic.worldPlan)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.deterministic.ruling, err = clonePayload(input.deterministic.ruling)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.deterministic.decisionRecord, err = clonePayload(input.deterministic.decisionRecord)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.deterministic.contractBundle, err = clonePayload(input.deterministic.contractBundle)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.fresh.worldInstance, err = clonePayload(input.fresh.worldInstance)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.fresh.attempts, err = clonePayload(input.fresh.attempts)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.fresh.measurements, err = clonePayload(input.fresh.measurements)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.fresh.captures, err = clonePayload(input.fresh.captures)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.fresh.confirmation, err = clonePayload(input.fresh.confirmation)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.fresh.target, err = clonePayload(input.fresh.target)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.fresh.finalized, err = clonePayload(input.fresh.finalized)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	result.fresh.classification, err = clonePayload(input.fresh.classification)
	if err != nil {
		return evidencePublicationInput{}, err
	}
	return result, nil
}

func evidenceError(code, detail string) error {
	return fmt.Errorf("U7_CLI_EVIDENCE_%s: %s", code, detail)
}
