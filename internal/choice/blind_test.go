package choice

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestBlindDTOOrderIgnoresSupportCountAndCandidateInputOrder(t *testing.T) {
	choicepoint := testDomainDigest(t, "blind-order-choicepoint")
	firstSpecs := []outcomeSpec{
		{candidate: "candidate:a", status: "200", kind: "shared"},
		{candidate: "candidate:b", status: "200", kind: "shared"},
		{candidate: "candidate:c", status: "404", kind: "other"},
	}
	reorderedSpecs := []outcomeSpec{firstSpecs[2], firstSpecs[0], firstSpecs[1]}
	reweightedSpecs := []outcomeSpec{
		{candidate: "candidate:a", status: "200", kind: "shared"},
		{candidate: "candidate:b", status: "404", kind: "other"},
		{candidate: "candidate:c", status: "404", kind: "other"},
	}

	_, first := testConfirmed(t, firstSpecs...)
	_, reordered := testConfirmed(t, reorderedSpecs...)
	_, reweighted := testConfirmed(t, reweightedSpecs...)
	firstGroups := mustBlindGroups(t, choicepoint, first)
	reorderedGroups := mustBlindGroups(t, choicepoint, reordered)
	reweightedGroups := mustBlindGroups(t, choicepoint, reweighted)

	want := blindGroupPresentationOrder(firstGroups)
	if got := blindGroupPresentationOrder(reorderedGroups); !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate input order changed blind presentation order\nwant: %#v\n got: %#v", want, got)
	}
	if got := blindGroupPresentationOrder(reweightedGroups); !reflect.DeepEqual(got, want) {
		t.Fatalf("support counts changed blind presentation order\nwant: %#v\n got: %#v", want, got)
	}

	firstSupport := blindGroupSupportByFingerprint(firstGroups)
	reweightedSupport := blindGroupSupportByFingerprint(reweightedGroups)
	if reflect.DeepEqual(firstSupport, reweightedSupport) {
		t.Fatal("fixture did not actually change support counts")
	}
}

func TestBlindAliasBindsFingerprintNotFirstSupportingMember(t *testing.T) {
	choicepoint := testDomainDigest(t, "blind-alias-choicepoint")
	_, first := testConfirmed(t,
		outcomeSpec{candidate: "candidate:a", status: "200", kind: "shared"},
		outcomeSpec{candidate: "candidate:b", status: "200", kind: "shared"},
		outcomeSpec{candidate: "candidate:c", status: "404", kind: "other"},
	)
	_, second := testConfirmed(t,
		outcomeSpec{candidate: "candidate:d", status: "200", kind: "shared"},
		outcomeSpec{candidate: "candidate:e", status: "200", kind: "shared"},
		outcomeSpec{candidate: "candidate:f", status: "404", kind: "other"},
	)

	firstShared := blindGroupWithSupport(t, mustBlindGroups(t, choicepoint, first), 2)
	secondShared := blindGroupWithSupport(t, mustBlindGroups(t, choicepoint, second), 2)
	if firstShared.fingerprint != secondShared.fingerprint {
		t.Fatal("fixture shared groups do not have the same exact fingerprint")
	}
	if firstShared.refs[0].ID() == secondShared.refs[0].ID() {
		t.Fatal("fixture did not change the first supporting member")
	}
	if firstShared.alias != secondShared.alias {
		t.Fatalf("alias changed with supporting membership: %q != %q", firstShared.alias, secondShared.alias)
	}
	otherChoicepoint := testDomainDigest(t, "other-blind-alias-choicepoint")
	otherAlias, err := blindAlias(otherChoicepoint, firstShared.fingerprint)
	if err != nil {
		t.Fatalf("derive alias for other Choicepoint: %v", err)
	}
	if otherAlias == firstShared.alias {
		t.Fatal("alias failed to bind the exact Choicepoint digest")
	}
}

func TestSharedBlindAliasExpandsEverySupportingOutcome(t *testing.T) {
	choicepoint := testDomainDigest(t, "blind-expansion-choicepoint")
	_, confirmed := testConfirmed(t,
		outcomeSpec{candidate: "candidate:a", status: "200", kind: "shared"},
		outcomeSpec{candidate: "candidate:b", status: "200", kind: "shared"},
		outcomeSpec{candidate: "candidate:c", status: "404", kind: "other"},
	)
	shared := blindGroupWithSupport(t, mustBlindGroups(t, choicepoint, confirmed), 2)
	view := BlindView{aliases: map[string][]ConfirmedOutcomeRef{
		shared.alias: append([]ConfirmedOutcomeRef(nil), shared.refs...),
	}}
	resolved, err := view.resolveAliases([]string{shared.alias})
	if err != nil {
		t.Fatalf("resolve shared alias: %v", err)
	}
	if len(resolved) != len(shared.refs) {
		t.Fatalf("shared alias resolved %d outcomes, want %d", len(resolved), len(shared.refs))
	}
	want := map[string]struct{}{}
	for _, ref := range shared.refs {
		want[ref.ID().String()] = struct{}{}
	}
	for _, ref := range resolved {
		if _, present := want[ref.ID().String()]; !present {
			t.Fatalf("shared alias expanded foreign outcome %s", ref.ID())
		}
		delete(want, ref.ID().String())
	}
	if len(want) != 0 {
		t.Fatalf("shared alias omitted supporting outcomes: %#v", want)
	}
}

func TestBlindDTOIdentityHasClosedCandidateNeutralWireShape(t *testing.T) {
	identity := blindDTOIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "BlindChoicepoint",
		ChoicepointDigest:       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Scenario:                "Which exact witnessed behavior should be accepted?",
		Scope:                   blindScope,
		OriginalStimulusBase64:  "e30=",
		MinimizedStimulusBase64: "e30=",
		Reduction: BlindReductionFacts{
			GradeStatus: "UNRECEIPTED", ProposalLimit: 1, CandidateTrialLimit: 1,
			WallLimitMS: 1, EvaluationCount: 1, UnresolvedCount: 0,
			Limitations:      []string{},
			ReducerSetDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			FinalSweepState:  "COMPLETE",
		},
		DiscoveryRepeatsPerCandidate: 1, ConfirmationRepeatsPerCandidate: 1,
		ProjectionMode: "EXACT", ProjectionOperations: []BlindProjectionOperation{{
			Name: "exact", RuleDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		}},
		SelectableFields: []string{WholeProjectionFieldID}, DifferingFields: []string{WholeProjectionFieldID},
		Cards: []BlindCard{{Alias: "blind:opaque", Fields: []BlindField{{
			FieldID: WholeProjectionFieldID, Tag: string(ValueString), Text: "opaque", Boolean: false,
			CanonicalJSONBase64: "",
		}}}},
		TrustWarnings: []string{"exact witness only"},
	}
	_, exact, err := canon.DigestTyped("BlindChoicepoint", identity)
	if err != nil {
		t.Fatalf("canonicalize blind DTO fixture: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(exact, &root); err != nil {
		t.Fatalf("decode blind DTO fixture: %v", err)
	}
	assertExactObjectKeys(t, "blind DTO", root,
		"schema_version", "kind", "choicepoint_digest", "scenario", "scope",
		"original_stimulus_base64", "minimized_stimulus_base64", "reduction",
		"discovery_repeats_per_candidate", "confirmation_repeats_per_candidate",
		"projection_mode", "projection_operations", "selectable_fields", "differing_fields",
		"cards", "trust_warnings",
	)
	assertExactObjectKeys(t, "blind DTO reduction", mustObjectMember(t, root, "reduction"),
		"grade_status", "proposal_limit", "candidate_trial_limit", "wall_limit_ms",
		"evaluation_count", "unresolved_count", "limitations", "reducer_set_digest", "final_sweep_state",
	)
	operation := mustObjectArrayMember(t, root, "projection_operations", 1)[0]
	assertExactObjectKeys(t, "blind DTO projection operation", mustObject(t, operation), "name", "rule_digest")
	card := mustObjectArrayMember(t, root, "cards", 1)[0]
	cardObject := mustObject(t, card)
	assertExactObjectKeys(t, "blind DTO card", cardObject, "alias", "fields")
	field := mustObjectArrayMember(t, cardObject, "fields", 1)[0]
	assertExactObjectKeys(t, "blind DTO field", mustObject(t, field),
		"field_id", "tag", "text", "boolean", "canonical_json_base64",
	)
}

func TestBlindRenderedCardsOmitCandidateIdentityAndSupportMetadata(t *testing.T) {
	choicepoint := testDomainDigest(t, "blind-render-choicepoint")
	_, confirmed := testConfirmed(t,
		outcomeSpec{candidate: "candidate:a", status: "200", kind: "shared"},
		outcomeSpec{candidate: "candidate:b", status: "200", kind: "shared"},
		outcomeSpec{candidate: "candidate:c", status: "404", kind: "other"},
	)
	groups := mustBlindGroups(t, choicepoint, confirmed)
	cards, aliases, err := renderBlindGroups(groups)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != len(groups) || len(aliases) != len(groups) {
		t.Fatalf("rendered cards=%d aliases=%d, want %d exact groups", len(cards), len(aliases), len(groups))
	}
	for index, group := range groups {
		if cards[index].Alias != group.alias {
			t.Fatalf("card %d alias = %q, want candidate-neutral %q", index, cards[index].Alias, group.alias)
		}
		encoded, err := json.Marshal(cards[index])
		if err != nil {
			t.Fatalf("encode card %d: %v", index, err)
		}
		for _, ref := range group.refs {
			for _, forbidden := range []string{
				ref.ID().String(), ref.CandidateExecutionKey().String(), ref.ProjectionFingerprint().String(),
			} {
				if bytes.Contains(encoded, []byte(forbidden)) {
					t.Fatalf("card %d leaked candidate or reveal identity %q", index, forbidden)
				}
			}
		}
		if bytes.Contains(encoded, []byte(`"support"`)) || bytes.Contains(encoded, []byte(`"count"`)) {
			t.Fatalf("card %d leaked support metadata: %s", index, encoded)
		}
	}
}

func TestBlindRendererRejectsAliasCollisionAndEmptySupport(t *testing.T) {
	choicepoint := testDomainDigest(t, "blind-render-collision-choicepoint")
	_, confirmed := testConfirmed(t,
		outcomeSpec{candidate: "candidate:a", status: "200", kind: "first"},
		outcomeSpec{candidate: "candidate:b", status: "404", kind: "second"},
	)
	groups := mustBlindGroups(t, choicepoint, confirmed)
	if len(groups) != 2 {
		t.Fatalf("blind group count = %d, want 2", len(groups))
	}
	collided := append([]blindGroup(nil), groups...)
	collided[1].alias = collided[0].alias
	_, _, err := renderBlindGroups(collided)
	assertRefusal(t, err, CodeInvalidConfirmedOutcomeSet)

	empty := append([]blindGroup(nil), groups...)
	empty[0].refs = nil
	_, _, err = renderBlindGroups(empty)
	assertRefusal(t, err, CodeInvalidConfirmedOutcomeSet)
}

func mustBlindGroups(t testing.TB, choicepoint domain.Digest, confirmed ConfirmedOutcomeSet) []blindGroup {
	t.Helper()
	groups, err := buildBlindGroups(choicepoint, confirmed)
	if err != nil {
		t.Fatalf("build blind groups: %v", err)
	}
	return groups
}

func blindGroupPresentationOrder(groups []blindGroup) []string {
	result := make([]string, len(groups))
	for index, group := range groups {
		result[index] = group.fingerprint.String() + "|" + group.alias + "|" + group.orderKey
	}
	return result
}

func blindGroupSupportByFingerprint(groups []blindGroup) map[string]int {
	result := make(map[string]int, len(groups))
	for _, group := range groups {
		result[group.fingerprint.String()] = len(group.refs)
	}
	return result
}

func blindGroupWithSupport(t testing.TB, groups []blindGroup, support int) blindGroup {
	t.Helper()
	for _, group := range groups {
		if len(group.refs) == support {
			return group
		}
	}
	t.Fatalf("no blind group has support %d", support)
	return blindGroup{}
}

func assertExactObjectKeys(t testing.TB, path string, object map[string]any, want ...string) {
	t.Helper()
	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s members differ\nwant: %#v\n got: %#v", path, want, got)
	}
}

func mustObjectMember(t testing.TB, object map[string]any, key string) map[string]any {
	t.Helper()
	value, present := object[key]
	if !present {
		t.Fatalf("object lacks %q", key)
	}
	return mustObject(t, value)
}

func mustObject(t testing.TB, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value is %T, want object", value)
	}
	return object
}

func mustObjectArrayMember(t testing.TB, object map[string]any, key string, wantLength int) []any {
	t.Helper()
	values, ok := object[key].([]any)
	if !ok || len(values) != wantLength {
		t.Fatalf("%q is %T with length %d, want array length %d", key, object[key], len(values), wantLength)
	}
	return values
}
