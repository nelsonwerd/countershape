// Package reduction is the outward composition boundary between pure
// reduction evidence and durable store authority. It is the only package that
// can promote a live reduction run to the local one-minimal grade.
package reduction

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/reduce"
	"github.com/nelsonwerd/countershape/internal/store"
)

const limitationDurableSweepAuthorityAbsent = "DURABLE_SWEEP_AUTHORITY_ABSENT"

// GradeStatus is the closed set of outward reduction grades. The quantified
// status is only a label; callers cannot construct a valid Grade from it.
type GradeStatus string

const (
	StatusUnchanged       GradeStatus = GradeStatus(reduce.GradeUnchanged)
	StatusBestKnown       GradeStatus = GradeStatus(reduce.GradeBestKnown)
	StatusOneMinimalUnder GradeStatus = "ONE_MINIMAL_UNDER"
)

// SweepCompletion groups the exact inputs required to request the strong
// transition. Nil means no durable authority was offered and therefore selects
// the honest weak-grade path. A non-nil value is never treated as optional:
// zero, serialized/reconstructed, stale, or mismatched members are refusals
// rather than fallback. An ordinary in-process value assignment retains the
// same opaque authority and is not a reconstruction boundary.
type SweepCompletion struct {
	Store     *store.ReductionSweepStore
	Draft     reduce.CompletedSweepDraft
	Authority store.SweepCompletionAuthority
}

// Error is a stable refusal at the outward reduction boundary.
type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.Cause }

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

type gradeIdentity struct {
	SchemaVersion        string   `json:"schema_version"`
	Kind                 string   `json:"kind"`
	Status               string   `json:"status"`
	RunDigest            string   `json:"run_digest"`
	ReducerSetDigest     string   `json:"reducer_set_digest"`
	CompletedSweepDigest string   `json:"completed_sweep_digest"`
	Limitations          []string `json:"limitations"`
	GlobalMinimumClaimed bool     `json:"global_minimum_claimed"`
	RootCauseClaimed     bool     `json:"root_cause_claimed"`
}

// Grade is construction-safe outward grade evidence. Its canonical identity
// always binds the exact run and reducer set. The quantified grade additionally
// binds the durably validated completed-sweep draft. There is intentionally no
// parser or public constructor: serialized bytes cannot recreate authority.
type Grade struct {
	status         GradeStatus
	runDigest      domain.Digest
	reducerSet     domain.Digest
	completedSweep domain.Digest
	limitations    []string
	digest         domain.Digest
	canonicalBytes []byte
}

func (g Grade) Valid() bool {
	rebuilt, err := newGrade(g.status, g.runDigest, g.reducerSet, g.completedSweep, g.limitations)
	return err == nil && rebuilt.digest == g.digest && bytes.Equal(rebuilt.canonicalBytes, g.canonicalBytes)
}

func (g Grade) Status() GradeStatus      { return g.status }
func (g Grade) RunDigest() domain.Digest { return g.runDigest }
func (g Grade) ReducerSetDigest() domain.Digest {
	return g.reducerSet
}
func (g Grade) Digest() domain.Digest  { return g.digest }
func (g Grade) CanonicalBytes() []byte { return append([]byte(nil), g.canonicalBytes...) }
func (g Grade) Limitations() []string  { return append([]string(nil), g.limitations...) }

// QuantifiedReducerSetDigest is present only for the strong local-minimality
// grade. Weak grades still bind the reducer set in their canonical identity,
// but do not quantify over it.
func (g Grade) QuantifiedReducerSetDigest() (domain.Digest, bool) {
	return g.reducerSet, g.status == StatusOneMinimalUnder && g.Valid()
}

// CompletedSweepDigest is present only when the grade crossed the durable
// authority boundary.
func (g Grade) CompletedSweepDigest() (domain.Digest, bool) {
	return g.completedSweep, g.status == StatusOneMinimalUnder && g.Valid()
}

func (g Grade) String() string {
	if g.status == StatusOneMinimalUnder && g.Valid() {
		return fmt.Sprintf("%s(%s)", g.status, g.reducerSet.String())
	}
	return string(g.status)
}

func newGrade(
	status GradeStatus,
	runDigest, reducerSet, completedSweep domain.Digest,
	limitations []string,
) (Grade, error) {
	normalizedLimitations := append([]string{}, limitations...)
	if !runDigest.Valid() || !reducerSet.Valid() || !validLimitations(normalizedLimitations) {
		return Grade{}, refuse("INVALID_REDUCTION_GRADE", "grade identity is incomplete", nil)
	}
	switch status {
	case StatusOneMinimalUnder:
		// MUTATION_ANCHOR: strong-grade-requires-exact-reducer-set-and-clean-sweep
		if !completedSweep.Valid() || len(normalizedLimitations) != 0 {
			return Grade{}, refuse("INVALID_REDUCTION_GRADE", "strong grade requires one exact limitation-free sweep", nil)
		}
	case StatusBestKnown, StatusUnchanged:
		if completedSweep.Valid() || !slices.Contains(normalizedLimitations, limitationDurableSweepAuthorityAbsent) {
			return Grade{}, refuse("INVALID_REDUCTION_GRADE", "weak grade must disclose missing durable authority", nil)
		}
	default:
		return Grade{}, refuse("INVALID_REDUCTION_GRADE", "unknown grade status", nil)
	}
	identity := gradeIdentity{
		SchemaVersion:        domain.SchemaVersion,
		Kind:                 "ReductionGrade",
		Status:               string(status),
		RunDigest:            runDigest.String(),
		ReducerSetDigest:     reducerSet.String(),
		CompletedSweepDigest: completedSweep.String(),
		Limitations:          normalizedLimitations,
		GlobalMinimumClaimed: false,
		RootCauseClaimed:     false,
	}
	digest, canonicalBytes, err := canon.DigestTyped("ReductionGrade", identity)
	if err != nil {
		return Grade{}, refuse("INVALID_REDUCTION_GRADE", "canonical grade construction failed", err)
	}
	parsedDigest, err := domain.ParseDigest(digest.String())
	if err != nil {
		return Grade{}, refuse("INVALID_REDUCTION_GRADE", "grade digest is invalid", err)
	}
	return Grade{
		status: status, runDigest: runDigest, reducerSet: reducerSet, completedSweep: completedSweep,
		limitations: append([]string(nil), normalizedLimitations...), digest: parsedDigest,
		canonicalBytes: append([]byte(nil), canonicalBytes...),
	}, nil
}

func validLimitations(values []string) bool {
	if values == nil {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

// Result binds an outward grade to the exact live run and transcript that
// earned it. It intentionally does not retain the opaque store authority.
type Result struct {
	runDigest        domain.Digest
	transcriptDigest domain.Digest
	grade            Grade
}

func (r Result) Valid() bool {
	return r.runDigest.Valid() && r.transcriptDigest.Valid() && r.grade.Valid() && r.grade.runDigest == r.runDigest
}

func (r Result) RunDigest() domain.Digest        { return r.runDigest }
func (r Result) TranscriptDigest() domain.Digest { return r.transcriptDigest }
func (r Result) Grade() Grade                    { return r.grade }

// Finalize applies grade precedence to one live run. When completion is
// non-nil, durable validation is deliberately the first operation: callers
// cannot use a malformed run or mismatched draft to probe or bypass an authority.
// A failed offered authority is a refusal, never permission to silently
// downgrade. Passing nil completion is the explicit weak-grade path.
func Finalize(ctx context.Context, run reduce.ReductionRun, completion *SweepCompletion) (Result, error) {
	if completion != nil {
		// Snapshot the offered tuple so a caller cannot race validation against a
		// later binding check by mutating the public composition carrier.
		sweepStore, draft, authority := completion.Store, completion.Draft, completion.Authority
		// Keep this call before all run/draft interpretation. Validate reopens the
		// exact content-addressed object on every transition.
		if err := sweepStore.Validate(ctx, draft, authority); err != nil {
			return Result{}, refuse("REDUCTION_SWEEP_AUTHORITY_REFUSED", "durable completed sweep did not validate", err)
		}
		record, err := validateLiveRun(run)
		if err != nil {
			return Result{}, err
		}
		if err := bindCompletedSweep(run, record, draft); err != nil {
			return Result{}, err
		}
		budgetPlan, planBound := run.Budget().WorldPlanDigest()
		if !planBound || budgetPlan != run.BaselinePlanDigest() {
			return Result{}, refuse(
				"REDUCTION_BUDGET_PLAN_AUTHORITY_ABSENT",
				"strong grade requires the exact compiled WorldPlan shrink budget",
				nil,
			)
		}
		grade, err := newGrade(StatusOneMinimalUnder, run.Digest(), run.ReducerSet().Digest(), draft.Digest(), run.Transcript().Limitations())
		if err != nil {
			return Result{}, err
		}
		return Result{runDigest: run.Digest(), transcriptDigest: run.Transcript().Digest(), grade: grade}, nil
	}

	if _, err := validateLiveRun(run); err != nil {
		return Result{}, err
	}
	status := StatusUnchanged
	if run.HasAcceptedReduction() {
		status = StatusBestKnown
	}
	limitations := run.Transcript().Limitations()
	if !slices.Contains(limitations, limitationDurableSweepAuthorityAbsent) {
		limitations = append(limitations, limitationDurableSweepAuthorityAbsent)
	}
	grade, err := newGrade(status, run.Digest(), run.ReducerSet().Digest(), domain.Digest(""), limitations)
	if err != nil {
		return Result{}, err
	}
	return Result{runDigest: run.Digest(), transcriptDigest: run.Transcript().Digest(), grade: grade}, nil
}

func validateLiveRun(run reduce.ReductionRun) (reduce.ReductionRunRecord, error) {
	if !run.Valid() {
		return reduce.ReductionRunRecord{}, refuse("INVALID_LIVE_REDUCTION_RUN", "run is zero or invalid", nil)
	}
	record, err := reduce.ParseReductionRunRecord(run.CanonicalBytes(), run.Transcript().CanonicalBytes())
	if err != nil {
		return reduce.ReductionRunRecord{}, refuse("INVALID_LIVE_REDUCTION_RUN", "run failed its owning wire parser", err)
	}
	set := run.ReducerSet()
	if record.Digest() != run.Digest() || record.Transcript().Digest() != run.Transcript().Digest() ||
		record.OriginalStimulusDigest() != run.OriginalStimulusDigest() ||
		record.MinimizedStimulusDigest() != run.MinimizedStimulusDigest() ||
		record.ReducerSetDigest() != set.Digest() ||
		record.MeasureDefinitionDigest() != set.MeasureDefinitionDigest() ||
		record.ScopeDigest() != set.ScopeDigest() || record.Grade() != run.DraftGrade() {
		return reduce.ReductionRunRecord{}, refuse("INVALID_LIVE_REDUCTION_RUN", "live fields disagree with exact parsed bytes", nil)
	}
	return record, nil
}

func bindCompletedSweep(run reduce.ReductionRun, record reduce.ReductionRunRecord, draft reduce.CompletedSweepDraft) error {
	expected, present, err := run.CompletedSweepDraft()
	if err != nil || !present {
		return refuse("COMPLETED_SWEEP_RUN_BINDING_MISMATCH", "live run has no exact logical completion", err)
	}
	if !draft.Valid() || expected.Digest() != draft.Digest() || !bytes.Equal(expected.CanonicalBytes(), draft.CanonicalBytes()) {
		return refuse("COMPLETED_SWEEP_RUN_BINDING_MISMATCH", "offered draft is not the live run's exact draft", nil)
	}
	currentMeasure := draft.CurrentMeasure()
	minimizedMeasure := run.MinimizedMeasure()
	set := run.ReducerSet()
	if draft.RunDigest() != run.Digest() || draft.TranscriptDigest() != run.Transcript().Digest() ||
		draft.CurrentStimulusDigest() != run.MinimizedStimulusDigest() ||
		draft.ReducerSetDigest() != set.Digest() ||
		currentMeasure.DefinitionDigest() != set.MeasureDefinitionDigest() ||
		currentMeasure.DefinitionDigest() != minimizedMeasure.DefinitionDigest() ||
		currentMeasure.Digest() != minimizedMeasure.Digest() ||
		!slices.Equal(currentMeasure.Components(), minimizedMeasure.Components()) ||
		draft.BaselineMapDigest() != expected.BaselineMapDigest() ||
		draft.BaselinePreservationMapDigest() != expected.BaselinePreservationMapDigest() ||
		draft.BaselinePreservationMapDigest() != record.BaselinePreservationMapDigest() {
		return refuse("COMPLETED_SWEEP_RUN_BINDING_MISMATCH", "draft lineage or exact measure/map binding differs from the live run", nil)
	}
	return nil
}
