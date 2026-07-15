package observe

import (
	"bytes"
	"sort"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	minScheduledCandidates  = 2
	maxScheduledCandidates  = 4
	maxScheduledRepetitions = 5
)

type rotatedScheduleIdentity struct {
	SchemaVersion string   `json:"schema_version"`
	Kind          string   `json:"kind"`
	Roster        []string `json:"candidate_roster"`
	Repetitions   int      `json:"repetitions"`
	Rotation      string   `json:"rotation"`
	Sequential    bool     `json:"sequential"`
}

// ScheduledTrial is one evidence position in a sequential rotated schedule.
// The ordinal is copied into WorldInstance by the adapter and therefore remains
// attached to the attempt evidence. It is not an outcome or map label.
type ScheduledTrial struct {
	candidateKey domain.CandidateExecutionKey
	repetition   int
	position     int
	ordinal      int
}

func (t ScheduledTrial) CandidateKey() domain.CandidateExecutionKey { return t.candidateKey }
func (t ScheduledTrial) Repetition() int                            { return t.repetition }
func (t ScheduledTrial) Position() int                              { return t.position }
func (t ScheduledTrial) Ordinal() int                               { return t.ordinal }

func (t ScheduledTrial) valid(candidateCount, repetitions int) bool {
	return t.candidateKey.Valid() && t.repetition >= 0 && t.repetition < repetitions &&
		t.position >= 0 && t.position < candidateCount && t.ordinal >= 0 &&
		t.ordinal == t.repetition*candidateCount+t.position
}

// RotatedSchedule is deterministic for a candidate roster: candidate keys are
// first put in canonical order, then repetition r starts at r mod n. Execution
// remains sequential; this type grants no concurrency authority.
type RotatedSchedule struct {
	digest      domain.Digest
	canonical   []byte
	roster      []domain.CandidateExecutionKey
	repetitions int
	trials      []ScheduledTrial
}

func NewRotatedSchedule(roster []domain.CandidateExecutionKey, repetitions int) (RotatedSchedule, error) {
	if len(roster) < minScheduledCandidates || len(roster) > maxScheduledCandidates {
		return RotatedSchedule{}, &domain.Error{Code: "INVALID_OBSERVATION_ROSTER", Detail: "candidate count outside 2..4"}
	}
	if repetitions < 1 || repetitions > maxScheduledRepetitions {
		return RotatedSchedule{}, &domain.Error{Code: "INVALID_OBSERVATION_REPEATS", Detail: "repeat count outside 1..5"}
	}
	canonicalRoster := append([]domain.CandidateExecutionKey(nil), roster...)
	sort.Slice(canonicalRoster, func(i, j int) bool {
		return canonicalRoster[i].String() < canonicalRoster[j].String()
	})
	for index, candidate := range canonicalRoster {
		if !candidate.Valid() {
			return RotatedSchedule{}, &domain.Error{Code: "INVALID_OBSERVATION_ROSTER", Detail: "invalid candidate key"}
		}
		if index > 0 && candidate == canonicalRoster[index-1] {
			return RotatedSchedule{}, &domain.Error{Code: domain.ErrDuplicateCandidateKey, Detail: candidate.String()}
		}
	}

	trials := make([]ScheduledTrial, 0, len(canonicalRoster)*repetitions)
	for repetition := 0; repetition < repetitions; repetition++ {
		for position := range canonicalRoster {
			candidate := canonicalRoster[(repetition+position)%len(canonicalRoster)]
			trials = append(trials, ScheduledTrial{
				candidateKey: candidate,
				repetition:   repetition,
				position:     position,
				ordinal:      repetition*len(canonicalRoster) + position,
			})
		}
	}
	rosterIdentity := make([]string, len(canonicalRoster))
	for index, candidate := range canonicalRoster {
		rosterIdentity[index] = candidate.String()
	}
	digest, canonicalBytes, err := canon.DigestTyped("RotatedSchedule", rotatedScheduleIdentity{
		SchemaVersion: domain.SchemaVersion,
		Kind:          "RotatedSchedule",
		Roster:        rosterIdentity,
		Repetitions:   repetitions,
		Rotation:      string(domain.ScheduleRotationStartByRepetitionV1),
		Sequential:    true,
	})
	if err != nil {
		return RotatedSchedule{}, &domain.Error{Code: "INVALID_OBSERVATION_SCHEDULE_IDENTITY", Detail: err.Error()}
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return RotatedSchedule{}, &domain.Error{Code: "INVALID_OBSERVATION_SCHEDULE_IDENTITY", Detail: err.Error()}
	}
	return RotatedSchedule{
		digest:      parsed,
		canonical:   append([]byte(nil), canonicalBytes...),
		roster:      canonicalRoster,
		repetitions: repetitions,
		trials:      trials,
	}, nil
}

func (s RotatedSchedule) Valid() bool {
	if !s.digest.Valid() || len(s.canonical) == 0 {
		return false
	}
	rebuilt, err := NewRotatedSchedule(s.roster, s.repetitions)
	return err == nil && rebuilt.digest == s.digest && bytes.Equal(rebuilt.canonical, s.canonical) &&
		sameScheduledTrials(rebuilt.trials, s.trials)
}

func (s RotatedSchedule) Digest() domain.Digest { return s.digest }

func (s RotatedSchedule) CanonicalBytes() []byte {
	return append([]byte(nil), s.canonical...)
}

func (s RotatedSchedule) Rotation() domain.ScheduleRotation {
	return domain.ScheduleRotationStartByRepetitionV1
}

func (s RotatedSchedule) Roster() []domain.CandidateExecutionKey {
	return append([]domain.CandidateExecutionKey(nil), s.roster...)
}

func (s RotatedSchedule) Repetitions() int { return s.repetitions }
func (s RotatedSchedule) CandidateCount() int {
	return len(s.roster)
}

func (s RotatedSchedule) TotalTrials() int { return len(s.trials) }

func (s RotatedSchedule) Trials() []ScheduledTrial {
	return append([]ScheduledTrial(nil), s.trials...)
}

func (s RotatedSchedule) Slot(candidate domain.CandidateExecutionKey, repetition int) (ScheduledTrial, bool) {
	if !s.Valid() || repetition < 0 || repetition >= s.repetitions {
		return ScheduledTrial{}, false
	}
	start := repetition * len(s.roster)
	for _, slot := range s.trials[start : start+len(s.roster)] {
		if slot.candidateKey == candidate {
			return slot, true
		}
	}
	return ScheduledTrial{}, false
}

func sameScheduledTrials(left, right []ScheduledTrial) bool {
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

func (s RotatedSchedule) TrialsForRepetition(repetition int) ([]ScheduledTrial, error) {
	if repetition < 0 || repetition >= s.repetitions || len(s.roster) == 0 {
		return nil, &domain.Error{Code: "INVALID_OBSERVATION_REPETITION"}
	}
	start := repetition * len(s.roster)
	return append([]ScheduledTrial(nil), s.trials[start:start+len(s.roster)]...), nil
}

// TrialBudget is finite by construction. The total-trial budget counts started
// adapter calls, including a partial matrix that cannot be admitted. WallBudget
// bounds the whole orchestration, not an individual adapter attempt.
type TrialBudget struct {
	MaxTotalTrials int
	WallBudget     time.Duration
}

func (b TrialBudget) validate(candidateCount int) error {
	if candidateCount < minScheduledCandidates || b.MaxTotalTrials < 1 {
		return &domain.Error{
			Code:   "INVALID_OBSERVATION_BUDGET",
			Detail: "total-trial budget must be finite and positive",
		}
	}
	if b.WallBudget <= 0 {
		return &domain.Error{Code: "INVALID_OBSERVATION_BUDGET", Detail: "wall budget must be finite and positive"}
	}
	return nil
}

type budgetExhaustion string

const (
	budgetAvailable        budgetExhaustion = "AVAILABLE"
	budgetTrialExhausted   budgetExhaustion = "TOTAL_TRIAL_BUDGET_EXHAUSTED"
	budgetWallExhausted    budgetExhaustion = "WALL_BUDGET_EXHAUSTED"
	budgetMatrixWouldSplit budgetExhaustion = "TOTAL_TRIAL_BUDGET_CANNOT_COMPLETE_MATRIX"
)

// BudgetSnapshot is disclosure evidence for orchestration control flow. It is
// not eligible behavior and never enters a projection fingerprint.
type BudgetSnapshot struct {
	maxTotalTrials  int
	startedTrials   int
	completedTrials int
	wallBudget      time.Duration
	elapsed         time.Duration
	exhaustion      budgetExhaustion
}

func (s BudgetSnapshot) MaxTotalTrials() int       { return s.maxTotalTrials }
func (s BudgetSnapshot) StartedTrials() int        { return s.startedTrials }
func (s BudgetSnapshot) CompletedTrials() int      { return s.completedTrials }
func (s BudgetSnapshot) WallBudget() time.Duration { return s.wallBudget }
func (s BudgetSnapshot) Elapsed() time.Duration    { return s.elapsed }
func (s BudgetSnapshot) Exhaustion() string        { return string(s.exhaustion) }
func (s BudgetSnapshot) Exhausted() bool           { return s.exhaustion != budgetAvailable }

type budgetTracker struct {
	budget          TrialBudget
	startedAt       time.Time
	startedTrials   int
	completedTrials int
	exhaustion      budgetExhaustion
}

func newBudgetTracker(budget TrialBudget, startedAt time.Time) budgetTracker {
	return budgetTracker{budget: budget, startedAt: startedAt, exhaustion: budgetAvailable}
}

func (b *budgetTracker) canStartMatrix(now time.Time, candidateCount int) bool {
	if b.wallExpired(now) {
		b.exhaustion = budgetWallExhausted
		return false
	}
	if b.budget.MaxTotalTrials-b.startedTrials < candidateCount {
		if b.budget.MaxTotalTrials == b.startedTrials {
			b.exhaustion = budgetTrialExhausted
		} else {
			b.exhaustion = budgetMatrixWouldSplit
		}
		return false
	}
	return true
}

func (b *budgetTracker) beginTrial(now time.Time) bool {
	if b.wallExpired(now) {
		b.exhaustion = budgetWallExhausted
		return false
	}
	if b.startedTrials >= b.budget.MaxTotalTrials {
		b.exhaustion = budgetTrialExhausted
		return false
	}
	b.startedTrials++
	return true
}

func (b *budgetTracker) completeTrial(now time.Time) bool {
	if b.wallExpired(now) {
		b.exhaustion = budgetWallExhausted
		return false
	}
	b.completedTrials++
	return true
}

func (b *budgetTracker) withinWall(now time.Time) bool {
	if b.wallExpired(now) {
		b.exhaustion = budgetWallExhausted
		return false
	}
	return true
}

func (b *budgetTracker) wallExpired(now time.Time) bool {
	return !now.Before(b.startedAt.Add(b.budget.WallBudget))
}

func (b *budgetTracker) snapshot(now time.Time) BudgetSnapshot {
	elapsed := now.Sub(b.startedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	return BudgetSnapshot{
		maxTotalTrials: b.budget.MaxTotalTrials, startedTrials: b.startedTrials,
		completedTrials: b.completedTrials, wallBudget: b.budget.WallBudget,
		elapsed: elapsed, exhaustion: b.exhaustion,
	}
}
