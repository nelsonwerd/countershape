package observe

import (
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestRotatedScheduleIsCanonicalSequentialAndEvidenceOrdinaled(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	left := fixture.binding(t, 1).Key()
	right := fixture.binding(t, 2).Key()
	schedule, err := NewRotatedSchedule([]domain.CandidateExecutionKey{right, left}, 3)
	if err != nil {
		t.Fatal(err)
	}
	roster := schedule.Roster()
	if len(roster) != 2 || roster[0].String() >= roster[1].String() {
		t.Fatalf("canonical roster = %#v", roster)
	}
	want := [][]domain.CandidateExecutionKey{
		{roster[0], roster[1]},
		{roster[1], roster[0]},
		{roster[0], roster[1]},
	}
	for repetition := range want {
		trials, err := schedule.TrialsForRepetition(repetition)
		if err != nil {
			t.Fatal(err)
		}
		for position, trial := range trials {
			if trial.CandidateKey() != want[repetition][position] ||
				trial.Repetition() != repetition || trial.Position() != position ||
				trial.Ordinal() != repetition*2+position {
				t.Fatalf("trial[%d][%d] = %#v", repetition, position, trial)
			}
		}
	}
}

func TestRotatedScheduleRefusesDuplicateOrUnboundedInputs(t *testing.T) {
	fixture := newExecutionFixture(t, 3, 3)
	key := fixture.binding(t, 1).Key()
	for _, test := range []struct {
		name    string
		roster  []domain.CandidateExecutionKey
		repeats int
	}{
		{name: "one candidate", roster: []domain.CandidateExecutionKey{key}, repeats: 3},
		{name: "duplicate", roster: []domain.CandidateExecutionKey{key, key}, repeats: 3},
		{name: "zero repeats", roster: []domain.CandidateExecutionKey{key, fixture.binding(t, 2).Key()}, repeats: 0},
		{name: "too many repeats", roster: []domain.CandidateExecutionKey{key, fixture.binding(t, 2).Key()}, repeats: 6},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewRotatedSchedule(test.roster, test.repeats); err == nil {
				t.Fatal("expected refusal")
			}
		})
	}
}

func TestBudgetTrackerStopsBeforeSplittingATrialBoundMatrix(t *testing.T) {
	start := time.Unix(100, 0)
	tracker := newBudgetTracker(TrialBudget{MaxTotalTrials: 3, WallBudget: time.Second}, start)
	if !tracker.canStartMatrix(start, 2) {
		t.Fatal("first complete matrix should fit")
	}
	for index := 0; index < 2; index++ {
		if !tracker.beginTrial(start) || !tracker.completeTrial(start.Add(time.Millisecond)) {
			t.Fatal("first matrix unexpectedly exhausted budget")
		}
	}
	if tracker.canStartMatrix(start.Add(2*time.Millisecond), 2) {
		t.Fatal("one remaining trial must not authorize a partial comparison matrix")
	}
	snapshot := tracker.snapshot(start.Add(2 * time.Millisecond))
	if snapshot.StartedTrials() != 2 || snapshot.CompletedTrials() != 2 ||
		snapshot.Exhaustion() != "TOTAL_TRIAL_BUDGET_CANNOT_COMPLETE_MATRIX" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestBudgetTrackerTreatsExactWallBoundaryAsExhausted(t *testing.T) {
	start := time.Unix(200, 0)
	tracker := newBudgetTracker(TrialBudget{MaxTotalTrials: 4, WallBudget: time.Second}, start)
	if !tracker.canStartMatrix(start.Add(time.Second-time.Nanosecond), 2) {
		t.Fatal("instant before deadline should remain available")
	}
	if tracker.canStartMatrix(start.Add(time.Second), 2) {
		t.Fatal("exact deadline must be exhausted")
	}
	if tracker.snapshot(start.Add(time.Second)).Exhaustion() != "WALL_BUDGET_EXHAUSTED" {
		t.Fatal("wall exhaustion was not disclosed")
	}
}
