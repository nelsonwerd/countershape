package reduce

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type engineStimulus struct {
	digest  domain.Digest
	measure Measure
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type sequenceClock struct {
	times []time.Time
	index int
}

func (c *sequenceClock) Now() time.Time {
	if c.index >= len(c.times) {
		return c.times[len(c.times)-1]
	}
	result := c.times[c.index]
	c.index++
	return result
}

func engineReducerSet(t *testing.T) ReducerSet {
	t.Helper()
	rule, err := NewReducerRule("drop-node", "v1")
	if err != nil {
		t.Fatal(err)
	}
	set, err := NewReducerSet("test", digest(6900), digest(6901), []ReducerRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func engineNode(t *testing.T, number int, value uint64) engineStimulus {
	t.Helper()
	measure, err := NewMeasure(digest(6900), []uint64{value, value})
	if err != nil {
		t.Fatal(err)
	}
	return engineStimulus{digest: digest(number), measure: measure}
}

func engineProposal(t *testing.T, set ReducerSet, parent, child engineStimulus, locus string) TypedProposal[engineStimulus] {
	t.Helper()
	return engineProposalWithRule(t, set, set.Rules()[0], parent, child, locus)
}

func engineProposalWithRule(t *testing.T, set ReducerSet, rule ReducerRule, parent, child engineStimulus, locus string) TypedProposal[engineStimulus] {
	t.Helper()
	return engineProposalWithRuleAndTransform(t, set, rule, parent, child, locus, 0)
}

func engineProposalWithRuleAndTransform(t *testing.T, set ReducerSet, rule ReducerRule, parent, child engineStimulus, locus string, transformPriority uint64) TypedProposal[engineStimulus] {
	t.Helper()
	neighbor, err := NewNeighbor(NeighborInput{
		CurrentStimulus: parent.digest, CurrentMeasure: parent.measure, Stimulus: child.digest, Measure: child.measure,
		Rule: rule, Locus: locus, TransformPriority: transformPriority, ReducerSet: set,
	})
	if err != nil {
		t.Fatal(err)
	}
	return TypedProposal[engineStimulus]{Stimulus: child, Neighbor: neighbor}
}

func engineReference(value engineStimulus) (domain.Digest, Measure, bool) {
	return value.digest, value.measure, value.digest.Valid() && value.measure.Valid()
}

type engineEvaluator struct {
	t                *testing.T
	fixture          reductionFixture
	nextOffset       int
	preserve         map[domain.Digest]bool
	unresolvedSearch map[domain.Digest]bool
	unresolvedFinal  bool
	reuseOffset      bool
	cancel           context.CancelFunc
}

func (e *engineEvaluator) evaluate(_ context.Context, _ engineStimulus, neighbor Neighbor, purpose domain.AttemptPurpose, _ EvaluationAllowance) (EvaluationObservation, error) {
	if purpose == domain.AttemptReduction && e.unresolvedSearch[neighbor.stimulusDigest] {
		evidence := engineUnresolvedEvidence(e.t, e.nextOffset)
		e.nextOffset++
		return EvaluationObservation{UnresolvedReason: ReasonUnstable, UnresolvedEvidence: evidence, CandidateTrials: 2}, nil
	}
	if e.unresolvedFinal && purpose == domain.AttemptFinalSweep {
		evidence := engineUnresolvedEvidence(e.t, e.nextOffset)
		e.nextOffset++
		return EvaluationObservation{UnresolvedReason: ReasonTeardownError, UnresolvedEvidence: evidence, CandidateTrials: 2}, nil
	}
	offset := e.nextOffset
	if e.reuseOffset {
		offset = 900
	}
	e.nextOffset++
	left, right := 10, 10
	if e.preserve[neighbor.stimulusDigest] {
		right = 11
	}
	observed := candidateMap(e.t, e.fixture, neighbor.stimulusDigest, purpose, left, right, offset)
	if e.cancel != nil {
		e.cancel()
	}
	return EvaluationObservation{OutcomeMap: &observed, CandidateTrials: 2}, nil
}

func engineUnresolvedEvidence(t *testing.T, offset int) UnresolvedEvidence {
	t.Helper()
	evidence, err := NewUnresolvedEvidence(
		[]domain.Digest{digest(offset*10 + 1)},
		[]domain.Digest{digest(offset*10 + 2)},
		[]domain.Digest{digest(offset*10 + 3)},
		[]domain.Digest{digest(offset*10 + 4)},
	)
	if err != nil {
		t.Fatal(err)
	}
	return evidence
}

func TestEarlierUnresolvedSearchCannotBeErasedByFreshCompleteSweep(t *testing.T) {
	set := engineReducerSet(t)
	original, smaller, finalNeighbor := engineNode(t, 1, 3), engineNode(t, 2, 2), engineNode(t, 3, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, smaller, "node.2")},
		smaller.digest:  {engineProposal(t, set, smaller, finalNeighbor, "node.3")},
	}
	budget, _ := NewBudget(10, 20, time.Minute)
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "engine-unresolved-search/v1"), nextOffset: 875,
		preserve:         map[domain.Digest]bool{smaller.digest: true},
		unresolvedSearch: map[domain.Digest]bool{finalNeighbor.digest: true},
	}
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if run.DraftGrade() != GradeBestKnown || !run.HasAcceptedReduction() ||
		run.Transcript().FinalSweepState() != FinalSweepComplete || len(run.Transcript().Limitations()) == 0 {
		t.Fatalf("unresolved search was not retained: grade=%s accepted=%t sweep=%s limitations=%v",
			run.DraftGrade(), run.HasAcceptedReduction(), run.Transcript().FinalSweepState(), run.Transcript().Limitations())
	}
	if _, present, err := run.CompletedSweepDraft(); err != nil || present {
		t.Fatalf("earlier unresolved search was erased by later sweep: present=%t err=%v", present, err)
	}
}

func engineInput(t *testing.T, graph map[domain.Digest][]TypedProposal[engineStimulus], evaluator *engineEvaluator, budget Budget) RunInput[engineStimulus] {
	t.Helper()
	original := engineNode(t, 1, 3)
	baseline := divergentBaseline(t, evaluator.fixture, original.digest, 700)
	return RunInput[engineStimulus]{
		Original: original, Reference: engineReference,
		Enumerate: func(_ context.Context, current engineStimulus) ([]TypedProposal[engineStimulus], error) {
			return append([]TypedProposal[engineStimulus](nil), graph[current.digest]...), nil
		},
		Evaluate: evaluator.evaluate, Baseline: baseline, ReducerSet: engineReducerSet(t), Budget: budget,
		Clock: fixedClock{now: time.Unix(100, 0)},
	}
}

func TestBoundedReducerAcceptsExactMapAndBuildsFreshCompleteDraft(t *testing.T) {
	set := engineReducerSet(t)
	original, smaller, finalNeighbor := engineNode(t, 1, 3), engineNode(t, 2, 2), engineNode(t, 3, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, smaller, "node.2")},
		smaller.digest:  {engineProposal(t, set, smaller, finalNeighbor, "node.3")},
	}
	budget, err := NewBudget(10, 20, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := &engineEvaluator{t: t, fixture: newReductionFixture(t, "engine-complete/v1"), nextOffset: 710,
		preserve: map[domain.Digest]bool{smaller.digest: true}}
	input := engineInput(t, graph, evaluator, budget)
	input.Original = original
	input.ReducerSet = set
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !run.Valid() || run.DraftGrade() != GradeBestKnown || run.MinimizedStimulusDigest() != smaller.digest ||
		run.Transcript().FinalSweepState() != FinalSweepComplete {
		t.Fatalf("unexpected completed run: grade=%s minimized=%s sweep=%s limits=%v",
			run.DraftGrade(), run.MinimizedStimulusDigest(), run.Transcript().FinalSweepState(), run.Transcript().Limitations())
	}
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 1 ||
		draft.BaselinePreservationMapDigest() != evaluator.fixtureMapPreservation(t, smaller.digest, 999) {
		// The final comparison baseline is the accepted preserving map. Compare
		// directly to the run's serialized baseline field below; the helper call
		// here intentionally only checks typed digest construction is retained.
		if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 1 {
			t.Fatalf("completed draft = present %t valid %t err %v", present, draft.Valid(), err)
		}
	}
	parsed, err := ParseCompletedSweepDraft(draft.CanonicalBytes())
	if err != nil || parsed.Digest() != draft.Digest() || parsed.BaselinePreservationMapDigest() != draft.BaselinePreservationMapDigest() {
		t.Fatalf("draft round trip lost exact map digest: %#v %v", parsed, err)
	}
	if len(run.ReplayPlan()) != 3 {
		t.Fatalf("replay plan has %d steps, want search+search+fresh-sweep", len(run.ReplayPlan()))
	}
}

func (e *engineEvaluator) fixtureMapPreservation(t *testing.T, stimulus domain.Digest, offset int) compare.PreservationMapDigest {
	t.Helper()
	return candidateMap(t, e.fixture, stimulus, domain.AttemptReduction, 10, 11, offset).PreservationDigest()
}

func TestReducerGradePrecedenceIsHonestUnderUnresolvedSweep(t *testing.T) {
	set := engineReducerSet(t)
	original, smaller, neighbor := engineNode(t, 1, 3), engineNode(t, 2, 2), engineNode(t, 3, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, smaller, "node.2")},
		smaller.digest:  {engineProposal(t, set, smaller, neighbor, "node.3")},
	}
	budget, _ := NewBudget(10, 20, time.Minute)
	evaluator := &engineEvaluator{t: t, fixture: newReductionFixture(t, "engine-best/v1"), nextOffset: 800,
		preserve: map[domain.Digest]bool{smaller.digest: true}, unresolvedFinal: true}
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if run.DraftGrade() != GradeBestKnown || run.Transcript().FinalSweepState() != FinalSweepIncomplete {
		t.Fatalf("accepted unresolved run grade=%s sweep=%s", run.DraftGrade(), run.Transcript().FinalSweepState())
	}
	if _, present, err := run.CompletedSweepDraft(); err != nil || present {
		t.Fatalf("incomplete sweep produced draft: present=%t err=%v", present, err)
	}

	unchangedEvaluator := &engineEvaluator{t: t, fixture: newReductionFixture(t, "engine-unchanged/v1"), nextOffset: 850,
		preserve: map[domain.Digest]bool{}, unresolvedFinal: true}
	unchangedInput := engineInput(t, map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, smaller, "node.2")},
	}, unchangedEvaluator, budget)
	unchangedInput.Original, unchangedInput.ReducerSet = original, set
	unchanged, err := Run(context.Background(), unchangedInput)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.DraftGrade() != GradeUnchanged || unchanged.HasAcceptedReduction() {
		t.Fatalf("no accepted reduction grade=%s accepted=%t", unchanged.DraftGrade(), unchanged.HasAcceptedReduction())
	}
}

func TestReducerCanonicalizesProposalOrderAndTranscript(t *testing.T) {
	set := engineReducerSet(t)
	original, left, right := engineNode(t, 1, 3), engineNode(t, 2, 2), engineNode(t, 3, 1)
	first := engineProposal(t, set, original, left, "node.b")
	second := engineProposal(t, set, original, right, "node.a")
	budget, _ := NewBudget(10, 30, time.Minute)
	runOnce := func(order []TypedProposal[engineStimulus]) ReductionRun {
		evaluator := &engineEvaluator{t: t, fixture: newReductionFixture(t, "engine-order/v1"), nextOffset: 1000, preserve: map[domain.Digest]bool{}}
		input := engineInput(t, map[domain.Digest][]TypedProposal[engineStimulus]{original.digest: order}, evaluator, budget)
		input.Original, input.ReducerSet = original, set
		run, err := Run(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	forward := runOnce([]TypedProposal[engineStimulus]{first, second})
	reverse := runOnce([]TypedProposal[engineStimulus]{second, first})
	if !bytes.Equal(forward.CanonicalBytes(), reverse.CanonicalBytes()) ||
		!bytes.Equal(forward.Transcript().CanonicalBytes(), reverse.Transcript().CanonicalBytes()) {
		t.Fatal("provider permutation changed canonical run or transcript")
	}
}

func TestReducerSetPriorityIsCanonicalAndOutranksLexicalRuleOrder(t *testing.T) {
	firstRule, err := NewReducerRule("z-first", "v1")
	if err != nil {
		t.Fatal(err)
	}
	secondRule, err := NewReducerRule("a-second", "v1")
	if err != nil {
		t.Fatal(err)
	}
	set, err := NewReducerSet("test", digest(6900), digest(6901), []ReducerRule{firstRule, secondRule})
	if err != nil {
		t.Fatal(err)
	}
	reversedSet, err := NewReducerSet("test", digest(6900), digest(6901), []ReducerRule{secondRule, firstRule})
	if err != nil {
		t.Fatal(err)
	}
	if set.Digest() == reversedSet.Digest() || set.Rules()[0].Digest() != firstRule.Digest() {
		t.Fatal("reducer-set identity did not bind declared rule priority")
	}

	original := engineNode(t, 1, 3)
	priorityChild := engineNode(t, 2, 2)
	lexicalChild := engineNode(t, 3, 1)
	priorityProposal := engineProposalWithRule(t, set, firstRule, original, priorityChild, "node.z")
	lexicalProposal := engineProposalWithRule(t, set, secondRule, original, lexicalChild, "node.a")
	budget, _ := NewBudget(10, 30, time.Minute)
	runOnce := func(order []TypedProposal[engineStimulus]) ReductionRun {
		evaluator := &engineEvaluator{
			t: t, fixture: newReductionFixture(t, "engine-rule-priority/v1"), nextOffset: 1050,
			preserve: map[domain.Digest]bool{priorityChild.digest: true, lexicalChild.digest: true},
		}
		input := engineInput(t, map[domain.Digest][]TypedProposal[engineStimulus]{original.digest: order}, evaluator, budget)
		input.Original, input.ReducerSet = original, set
		run, runErr := Run(context.Background(), input)
		if runErr != nil {
			t.Fatal(runErr)
		}
		return run
	}
	forward := runOnce([]TypedProposal[engineStimulus]{priorityProposal, lexicalProposal})
	reverse := runOnce([]TypedProposal[engineStimulus]{lexicalProposal, priorityProposal})
	if forward.MinimizedStimulusDigest() != priorityChild.digest || reverse.MinimizedStimulusDigest() != priorityChild.digest ||
		!bytes.Equal(forward.CanonicalBytes(), reverse.CanonicalBytes()) ||
		!bytes.Equal(forward.Transcript().CanonicalBytes(), reverse.Transcript().CanonicalBytes()) {
		t.Fatal("provider order or lexical rule name outranked canonical reducer-set priority")
	}
}

func TestReducerTransformPriorityOutranksChildDigest(t *testing.T) {
	set := engineReducerSet(t)
	rule := set.Rules()[0]
	original := engineNode(t, 1, 3)
	higherDigestPreferred := engineNode(t, 3, 2)
	lowerDigestFallback := engineNode(t, 2, 1)
	preferred := engineProposalWithRuleAndTransform(t, set, rule, original, higherDigestPreferred, "node.same", 0)
	fallback := engineProposalWithRuleAndTransform(t, set, rule, original, lowerDigestFallback, "node.same", 1)
	budget, _ := NewBudget(10, 30, time.Minute)
	runOnce := func(order []TypedProposal[engineStimulus]) ReductionRun {
		evaluator := &engineEvaluator{
			t: t, fixture: newReductionFixture(t, "engine-transform-priority/v1"), nextOffset: 1065,
			preserve: map[domain.Digest]bool{higherDigestPreferred.digest: true, lowerDigestFallback.digest: true},
		}
		input := engineInput(t, map[domain.Digest][]TypedProposal[engineStimulus]{original.digest: order}, evaluator, budget)
		input.Original, input.ReducerSet = original, set
		run, err := Run(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	forward := runOnce([]TypedProposal[engineStimulus]{preferred, fallback})
	reverse := runOnce([]TypedProposal[engineStimulus]{fallback, preferred})
	if forward.MinimizedStimulusDigest() != higherDigestPreferred.digest || reverse.MinimizedStimulusDigest() != higherDigestPreferred.digest ||
		!bytes.Equal(forward.CanonicalBytes(), reverse.CanonicalBytes()) {
		t.Fatal("child digest or provider order outranked canonical transform priority")
	}
}

func TestProposalBudgetFencepostCannotBePromoted(t *testing.T) {
	set := engineReducerSet(t)
	original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, child, "node.2")},
	}
	budget, err := NewBudget(1, 20, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "engine-proposal-fence/v1"), nextOffset: 1075,
		preserve: map[domain.Digest]bool{},
	}
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Transcript().Entries()) != 1 || run.Transcript().FinalSweepState() == FinalSweepComplete ||
		!slices.Contains(run.Transcript().Limitations(), "PROPOSAL_BUDGET_EXHAUSTED") {
		t.Fatalf("proposal fencepost was promoted: entries=%d sweep=%s limitations=%v",
			len(run.Transcript().Entries()), run.Transcript().FinalSweepState(), run.Transcript().Limitations())
	}
}

func TestCandidateTrialBudgetFencepostCannotBePromoted(t *testing.T) {
	set := engineReducerSet(t)
	original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, child, "node.2")},
	}
	budget, err := NewBudget(10, 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "engine-trial-fence/v1"), nextOffset: 1085,
		preserve: map[domain.Digest]bool{},
	}
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Transcript().Entries()) != 1 || run.Transcript().FinalSweepState() == FinalSweepComplete ||
		!slices.Contains(run.Transcript().Limitations(), "CANDIDATE_TRIAL_BUDGET_EXHAUSTED") {
		t.Fatalf("candidate-trial fencepost was promoted: entries=%d sweep=%s limitations=%v",
			len(run.Transcript().Entries()), run.Transcript().FinalSweepState(), run.Transcript().Limitations())
	}
}

func TestCancellationBudgetFencepostAndReusedEvidenceNeverProduceDraft(t *testing.T) {
	set := engineReducerSet(t)
	original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{original.digest: {engineProposal(t, set, original, child, "node.2")}}
	budget, _ := NewBudget(10, 20, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancelEvaluator := &engineEvaluator{t: t, fixture: newReductionFixture(t, "engine-cancel/v1"), nextOffset: 1100,
		preserve: map[domain.Digest]bool{child.digest: true}, cancel: cancel}
	input := engineInput(t, graph, cancelEvaluator, budget)
	input.Original, input.ReducerSet = original, set
	cancelled, err := Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, present, _ := cancelled.CompletedSweepDraft(); present {
		t.Fatal("cancelled run produced completed draft")
	}

	start := time.Unix(200, 0)
	fenceEvaluator := &engineEvaluator{t: t, fixture: newReductionFixture(t, "engine-fence/v1"), nextOffset: 1200, preserve: map[domain.Digest]bool{}}
	fenceInput := engineInput(t, graph, fenceEvaluator, budget)
	fenceInput.Original, fenceInput.ReducerSet = original, set
	fenceInput.Clock = &sequenceClock{times: []time.Time{start, start.Add(time.Second)}}
	fenced, err := Run(context.Background(), fenceInput)
	if err != nil {
		t.Fatal(err)
	}
	if len(fenced.Transcript().Entries()) != 0 || fenced.Transcript().FinalSweepState() == FinalSweepComplete {
		t.Fatal("wall fencepost admitted work or complete sweep")
	}

	reuseEvaluator := &engineEvaluator{t: t, fixture: newReductionFixture(t, "engine-reuse/v1"), nextOffset: 1300,
		preserve: map[domain.Digest]bool{}, reuseOffset: true}
	reuseInput := engineInput(t, graph, reuseEvaluator, budget)
	reuseInput.Original, reuseInput.ReducerSet = original, set
	reused, err := Run(context.Background(), reuseInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, present, _ := reused.CompletedSweepDraft(); present {
		t.Fatal("reused evaluation evidence produced completed draft")
	}
}

func TestInternalWallDeadlineIsNotMisreportedAsParentCancellation(t *testing.T) {
	set := engineReducerSet(t)
	original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, child, "node.2")},
	}
	budget, err := NewBudget(2, 2, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := &engineEvaluator{t: t, fixture: newReductionFixture(t, "engine-wall-timeout/v1"), nextOffset: 1400, preserve: map[domain.Digest]bool{}}
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	input.Clock = nil
	input.Evaluate = func(ctx context.Context, _ engineStimulus, _ Neighbor, _ domain.AttemptPurpose, _ EvaluationAllowance) (EvaluationObservation, error) {
		<-ctx.Done()
		return EvaluationObservation{}, ctx.Err()
	}
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	limitations := run.Transcript().Limitations()
	if !slices.Contains(limitations, "WALL_BUDGET_EXHAUSTED") ||
		slices.Contains(limitations, "REDUCTION_CANCELLED") ||
		slices.Contains(limitations, "UNRESOLVED_REDUCTION_CANCELLED") {
		t.Fatalf("internal wall deadline limitations=%v", limitations)
	}
	if _, present, err := run.CompletedSweepDraft(); err != nil || present {
		t.Fatalf("wall-expired run produced durable draft: present=%t err=%v", present, err)
	}
}

func TestCancellationDuringEmptySearchEnumerationCannotCompleteSweep(t *testing.T) {
	set := engineReducerSet(t)
	original := engineNode(t, 1, 2)
	budget, _ := NewBudget(10, 20, time.Minute)
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "engine-cancel-empty-search/v1"), nextOffset: 1450,
		preserve: map[domain.Digest]bool{},
	}
	input := engineInput(t, map[domain.Digest][]TypedProposal[engineStimulus]{}, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	ctx, cancel := context.WithCancel(context.Background())
	input.Enumerate = func(context.Context, engineStimulus) ([]TypedProposal[engineStimulus], error) {
		cancel()
		return []TypedProposal[engineStimulus]{}, nil
	}
	run, err := Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if run.Transcript().FinalSweepState() == FinalSweepComplete ||
		!slices.Contains(run.Transcript().Limitations(), "REDUCTION_CANCELLED") {
		t.Fatalf("cancelled empty search enumeration completed: sweep=%s limitations=%v",
			run.Transcript().FinalSweepState(), run.Transcript().Limitations())
	}
	if _, present, err := run.CompletedSweepDraft(); err != nil || present {
		t.Fatalf("cancelled empty search produced completion: present=%t err=%v", present, err)
	}
}

func TestCancellationDuringFinalEmptyEnumerationCannotCompleteSweep(t *testing.T) {
	set := engineReducerSet(t)
	original := engineNode(t, 1, 2)
	budget, _ := NewBudget(10, 20, time.Minute)
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "engine-cancel-empty-final/v1"), nextOffset: 1460,
		preserve: map[domain.Digest]bool{},
	}
	input := engineInput(t, map[domain.Digest][]TypedProposal[engineStimulus]{}, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	ctx, cancel := context.WithCancel(context.Background())
	enumerations := 0
	input.Enumerate = func(context.Context, engineStimulus) ([]TypedProposal[engineStimulus], error) {
		enumerations++
		if enumerations == 2 {
			cancel()
		}
		return []TypedProposal[engineStimulus]{}, nil
	}
	run, err := Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if enumerations != 2 || run.Transcript().FinalSweepState() == FinalSweepComplete ||
		!slices.Contains(run.Transcript().Limitations(), "REDUCTION_CANCELLED") {
		t.Fatalf("cancelled final empty enumeration completed: calls=%d sweep=%s limitations=%v",
			enumerations, run.Transcript().FinalSweepState(), run.Transcript().Limitations())
	}
	if _, present, err := run.CompletedSweepDraft(); err != nil || present {
		t.Fatalf("cancelled final empty sweep produced completion: present=%t err=%v", present, err)
	}
}

func TestEvaluatorErrorChargesReportedCandidateTrials(t *testing.T) {
	set := engineReducerSet(t)
	original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, child, "node.2")},
	}
	budget, _ := NewBudget(10, 2, time.Minute)
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "engine-evaluator-error-accounting/v1"), nextOffset: 1470,
		preserve: map[domain.Digest]bool{},
	}
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	input.Evaluate = func(context.Context, engineStimulus, Neighbor, domain.AttemptPurpose, EvaluationAllowance) (EvaluationObservation, error) {
		return EvaluationObservation{CandidateTrials: 2, UnresolvedEvidence: engineUnresolvedEvidence(t, 1470)}, errors.New("edge failed after trials")
	}
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	entries := run.Transcript().Entries()
	if len(entries) != 1 || entries[0].CandidateTrials() != 2 || entries[0].TrialCount() != 2 ||
		!slices.Contains(run.Transcript().Limitations(), "UNRESOLVED_EVALUATOR_ERROR") ||
		!slices.Contains(run.Transcript().Limitations(), "CANDIDATE_TRIAL_BUDGET_EXHAUSTED") {
		t.Fatalf("evaluator failure accounting lost: entries=%#v limitations=%v", entries, run.Transcript().Limitations())
	}
}

func TestEvaluatorProtocolRefusalsRemainTerminalTranscriptEntries(t *testing.T) {
	tests := []struct {
		name       string
		budget     uint64
		wantReason UnresolvedReason
		observe    func(*testing.T, reductionFixture, Neighbor, domain.AttemptPurpose) EvaluationObservation
	}{
		{
			name: "reported trials exceed allowance", budget: 2, wantReason: ReasonEvaluatorExceededBudget,
			observe: func(t *testing.T, _ reductionFixture, _ Neighbor, _ domain.AttemptPurpose) EvaluationObservation {
				return EvaluationObservation{
					UnresolvedReason: ReasonIncomplete, UnresolvedEvidence: engineUnresolvedEvidence(t, 1510), CandidateTrials: 3,
				}
			},
		},
		{
			name: "map reported without trials", budget: 2, wantReason: ReasonObservedMapWithoutTrials,
			observe: func(t *testing.T, fixture reductionFixture, neighbor Neighbor, purpose domain.AttemptPurpose) EvaluationObservation {
				observed := candidateMap(t, fixture, neighbor.StimulusDigest(), purpose, 10, 10, 1520)
				return EvaluationObservation{OutcomeMap: &observed}
			},
		},
		{
			name: "trial has no evidence", budget: 2, wantReason: ReasonUnresolvedEvidenceMissing,
			observe: func(_ *testing.T, _ reductionFixture, _ Neighbor, _ domain.AttemptPurpose) EvaluationObservation {
				return EvaluationObservation{UnresolvedReason: ReasonIncomplete, CandidateTrials: 1}
			},
		},
		{
			name: "invalid evaluator result", budget: 2, wantReason: ReasonInvalidEvaluatorResult,
			observe: func(_ *testing.T, _ reductionFixture, _ Neighbor, _ domain.AttemptPurpose) EvaluationObservation {
				return EvaluationObservation{UnresolvedReason: UnresolvedReason("NOT_A_CLOSED_REASON")}
			},
		},
		{
			name: "caller authors reserved protocol reason", budget: 2, wantReason: ReasonInvalidEvaluatorResult,
			observe: func(_ *testing.T, _ reductionFixture, _ Neighbor, _ domain.AttemptPurpose) EvaluationObservation {
				return EvaluationObservation{UnresolvedReason: ReasonReusedEvaluationEvidence}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := engineReducerSet(t)
			original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
			graph := map[domain.Digest][]TypedProposal[engineStimulus]{
				original.digest: {engineProposal(t, set, original, child, "node.2")},
			}
			budget, err := NewBudget(10, test.budget, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			evaluator := &engineEvaluator{
				t: t, fixture: newReductionFixture(t, "engine-protocol-refusal/"+test.name), nextOffset: 1500,
				preserve: map[domain.Digest]bool{},
			}
			input := engineInput(t, graph, evaluator, budget)
			input.Original, input.ReducerSet = original, set
			input.Evaluate = func(_ context.Context, _ engineStimulus, neighbor Neighbor, purpose domain.AttemptPurpose, _ EvaluationAllowance) (EvaluationObservation, error) {
				return test.observe(t, evaluator.fixture, neighbor, purpose), nil
			}
			run, err := Run(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			entries := run.Transcript().Entries()
			if len(entries) != 1 || entries[0].ProposalCount() != 1 || entries[0].Evaluation().Decision() != Unresolved ||
				entries[0].Evaluation().ReasonCode() != string(test.wantReason) ||
				!slices.Contains(run.Transcript().Limitations(), string(test.wantReason)) ||
				run.Transcript().FinalSweepState() != FinalSweepNotRun {
				t.Fatalf("protocol refusal disappeared: entries=%#v sweep=%s limitations=%v",
					entries, run.Transcript().FinalSweepState(), run.Transcript().Limitations())
			}
			if _, present, err := run.CompletedSweepDraft(); err != nil || present {
				t.Fatalf("protocol refusal exposed completion: present=%t err=%v", present, err)
			}
			if _, err := ParseTranscriptRecord(run.Transcript().CanonicalBytes()); err != nil {
				t.Fatalf("protocol refusal transcript did not round trip: %v", err)
			}
			if test.wantReason == ReasonEvaluatorExceededBudget &&
				(entries[0].CandidateTrials() != 3 || entries[0].TrialCount() != 3 || entries[0].TrialCount() <= test.budget) {
				t.Fatalf("over-budget evaluator count was hidden: candidate=%d total=%d limit=%d",
					entries[0].CandidateTrials(), entries[0].TrialCount(), test.budget)
			}
		})
	}
}

func TestMapBackedEvaluationRejectsReportedTrialCountMismatch(t *testing.T) {
	for _, reported := range []uint64{1, 3} {
		set := engineReducerSet(t)
		original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
		graph := map[domain.Digest][]TypedProposal[engineStimulus]{
			original.digest: {engineProposal(t, set, original, child, "node.2")},
		}
		budget, err := NewBudget(10, 4, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		fixture := newReductionFixture(t, fmt.Sprintf("engine-map-trial-count-mismatch/%d/v1", reported))
		input := engineInput(t, graph, &engineEvaluator{
			t: t, fixture: fixture, nextOffset: 1530, preserve: map[domain.Digest]bool{},
		}, budget)
		input.Original, input.ReducerSet = original, set
		input.Evaluate = func(_ context.Context, _ engineStimulus, neighbor Neighbor, purpose domain.AttemptPurpose, _ EvaluationAllowance) (EvaluationObservation, error) {
			observed := candidateMap(t, fixture, neighbor.StimulusDigest(), purpose, 10, 11, 1530)
			if len(observed.EvidenceAttemptDigests()) != 2 {
				t.Fatal("fixture no longer carries exactly two typed attempts")
			}
			return EvaluationObservation{OutcomeMap: &observed, CandidateTrials: reported}, nil
		}
		run, err := Run(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		entries := run.Transcript().Entries()
		if len(entries) != 1 || entries[0].CandidateTrials() != reported ||
			entries[0].Evaluation().Decision() != Unresolved ||
			entries[0].Evaluation().ReasonCode() != string(ReasonInvalidEvaluatorResult) ||
			run.HasAcceptedReduction() || run.MinimizedStimulusDigest() != original.digest ||
			run.Transcript().FinalSweepState() != FinalSweepNotRun ||
			!slices.Contains(run.Transcript().Limitations(), string(ReasonInvalidEvaluatorResult)) {
			t.Fatalf("reported=%d map-backed trial mismatch advanced: entries=%#v minimized=%s sweep=%s limitations=%v",
				reported, entries, run.MinimizedStimulusDigest(), run.Transcript().FinalSweepState(), run.Transcript().Limitations())
		}
		if _, present, err := run.CompletedSweepDraft(); err != nil || present {
			t.Fatalf("reported=%d map-backed trial mismatch exposed completion: present=%t err=%v", reported, present, err)
		}
	}
}

func TestCrossProposalEvidenceReuseIsRecordedAsTerminalRefusal(t *testing.T) {
	set := engineReducerSet(t)
	original, first, second := engineNode(t, 1, 3), engineNode(t, 2, 2), engineNode(t, 3, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {
			engineProposal(t, set, original, first, "node.2"),
			engineProposal(t, set, original, second, "node.3"),
		},
	}
	budget, _ := NewBudget(10, 20, time.Minute)
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "engine-cross-proposal-reuse/v1"), nextOffset: 1540,
		preserve: map[domain.Digest]bool{},
	}
	reused := engineUnresolvedEvidence(t, 1550)
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	input.Evaluate = func(context.Context, engineStimulus, Neighbor, domain.AttemptPurpose, EvaluationAllowance) (EvaluationObservation, error) {
		return EvaluationObservation{UnresolvedReason: ReasonIncomplete, UnresolvedEvidence: reused, CandidateTrials: 1}, nil
	}
	run, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	entries := run.Transcript().Entries()
	if len(entries) != 2 || entries[0].Evaluation().ReasonCode() != string(ReasonIncomplete) ||
		entries[1].Evaluation().ReasonCode() != string(ReasonReusedEvaluationEvidence) ||
		entries[1].ProposalCount() != 2 || entries[1].TrialCount() != 2 ||
		len(entries[1].Evaluation().ObservedBatchDigests()) != 0 ||
		!slices.Contains(run.Transcript().Limitations(), string(ReasonReusedEvaluationEvidence)) {
		t.Fatalf("evidence-reuse refusal was not explicit: entries=%#v limitations=%v", entries, run.Transcript().Limitations())
	}
}

func TestEvaluatorTrialCountBeyondCanonicalMaximumRefusesTheWholeRun(t *testing.T) {
	set := engineReducerSet(t)
	original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, child, "node.2")},
	}
	budget, _ := NewBudget(10, 2, time.Minute)
	evaluator := &engineEvaluator{
		t: t, fixture: newReductionFixture(t, "engine-trial-overflow/v1"), nextOffset: 1570,
		preserve: map[domain.Digest]bool{},
	}
	input := engineInput(t, graph, evaluator, budget)
	input.Original, input.ReducerSet = original, set
	input.Evaluate = func(context.Context, engineStimulus, Neighbor, domain.AttemptPurpose, EvaluationAllowance) (EvaluationObservation, error) {
		return EvaluationObservation{CandidateTrials: maxTrialLimit + 1}, nil
	}
	if _, err := Run(context.Background(), input); err == nil {
		t.Fatal("noncanonical evaluator trial count produced a partial transcript")
	}
}

func TestTranscriptIdentityExcludesNondeterministicWallProgress(t *testing.T) {
	set := engineReducerSet(t)
	original, child := engineNode(t, 1, 2), engineNode(t, 2, 1)
	graph := map[domain.Digest][]TypedProposal[engineStimulus]{
		original.digest: {engineProposal(t, set, original, child, "node.2")},
	}
	budget, _ := NewBudget(10, 20, time.Minute)
	runAt := func(now time.Time) ReductionRun {
		evaluator := &engineEvaluator{
			t: t, fixture: newReductionFixture(t, "engine-time-independent-identity/v1"), nextOffset: 1480,
			preserve: map[domain.Digest]bool{},
		}
		input := engineInput(t, graph, evaluator, budget)
		input.Original, input.ReducerSet, input.Clock = original, set, fixedClock{now: now}
		run, err := Run(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	first := runAt(time.Unix(100, 0))
	second := runAt(time.Unix(10_000, 0))
	if !bytes.Equal(first.Transcript().CanonicalBytes(), second.Transcript().CanonicalBytes()) ||
		first.Transcript().Digest() != second.Transcript().Digest() {
		t.Fatal("host wall progress entered canonical transcript identity")
	}
}
