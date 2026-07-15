package cli

import (
	"bytes"
	"fmt"
	"slices"
	"testing"
)

func cliNeighborFixture(t *testing.T, reverseEnvironment bool) CLIStimulus {
	t.Helper()
	presentEmpty, err := PresentStdin([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	appMode, err := PresentEnvironment("APP_MODE", "")
	if err != nil {
		t.Fatal(err)
	}
	optionalMode, err := AbsentEnvironment("OPTIONAL_MODE")
	if err != nil {
		t.Fatal(err)
	}
	environment := []CLIEnvironmentBinding{appMode, optionalMode}
	if reverseEnvironment {
		environment = []CLIEnvironmentBinding{optionalMode, appMode}
	}
	config, err := NewFixtureFile("config.json", []byte(`{"mode":"config"}`), FixtureMode0644)
	if err != nil {
		t.Fatal(err)
	}
	noise, err := NewFixtureFile("z-noise.txt", []byte("noise"), FixtureMode0644)
	if err != nil {
		t.Fatal(err)
	}
	stimulus, err := NewCLIStimulus(CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture.mjs"},
		Argv: []string{"--mode", "argv", "tail"}, Stdin: presentEmpty,
		Environment: environment, Fixtures: []CLIFixtureFile{config, noise},
		CWDPolicy: CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	return stimulus
}

func cliNeighborPolicy(t *testing.T, anchor CLIStimulus) CLIReductionPolicy {
	t.Helper()
	policy, err := NewCLIReductionPolicy(CLIReductionPolicyConfig{
		Anchor: anchor, PinnedFixturePaths: []string{"config.json"}, EnabledRules: DefaultCLIReducers(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestCLINeighborsAreDeterministicUniqueReplayableAndStrictlySmaller(t *testing.T) {
	stimulus := cliNeighborFixture(t, false)
	policy := cliNeighborPolicy(t, stimulus)
	first, err := EnumerateCLINeighbors(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EnumerateCLINeighbors(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 || len(first) != len(second) {
		t.Fatalf("neighbor counts = %d/%d", len(first), len(second))
	}
	before, err := MeasureCLIStimulus(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]struct{}{}
	for index := range first {
		left := first[index]
		right := second[index]
		if left.LogicalNeighbor().Digest() != right.LogicalNeighbor().Digest() || left.Stimulus().Digest() != right.Stimulus().Digest() {
			t.Fatalf("neighbor %d was nondeterministic", index)
		}
		if !left.Stimulus().Valid() {
			t.Fatalf("neighbor %d is not a valid typed stimulus", index)
		}
		if _, duplicate := seen[left.Stimulus().Digest().String()]; duplicate {
			t.Fatalf("duplicate child digest at %d", index)
		}
		seen[left.Stimulus().Digest().String()] = struct{}{}
		logical := left.LogicalNeighbor()
		if logical.CurrentStimulusDigest() != stimulus.Digest() || logical.StimulusDigest() != left.Stimulus().Digest() ||
			logical.ReducerSetDigest() != policy.ReducerSet().Digest() {
			t.Fatalf("neighbor %d lost reducer bindings", index)
		}
		comparison, compareErr := logical.Measure().Compare(before)
		if compareErr != nil || comparison >= 0 {
			t.Fatalf("neighbor %d did not strictly decrease: %d, %v", index, comparison, compareErr)
		}
		replayed, replayErr := ReplayCLINeighbor(stimulus, policy, left.ReplayRecipe())
		if replayErr != nil || replayed.Digest() != left.Stimulus().Digest() {
			t.Fatalf("neighbor %d replay mismatch: %v", index, replayErr)
		}
		if left.Stimulus().Executable() != stimulus.Executable() ||
			!slices.Equal(left.Stimulus().BaseArgv(), stimulus.BaseArgv()) ||
			left.Stimulus().CWDPolicy() != stimulus.CWDPolicy() {
			t.Fatalf("neighbor %d mutated a fixed CLI field", index)
		}
		assertPinnedCLIFixture(t, left.Stimulus(), "config.json", []byte(`{"mode":"config"}`))
	}
}

func TestCLIGeneratedNeighborsRespectProviderInvariants(t *testing.T) {
	testCases := []struct {
		name              string
		argv              []string
		stdin             []byte
		environmentValues []string
		fixtureContents   [][]byte
	}{
		{
			name: "present-empty-and-duplicate-argv",
			argv: []string{"repeat", "middle-empty", "repeat"}, stdin: []byte{},
			environmentValues: []string{"", "repeat"},
			fixtureContents:   [][]byte{[]byte("noise")},
		},
		{
			name: "single-byte-and-duplicate-payloads",
			argv: []string{"repeat", "middle-one", "repeat", "tail"}, stdin: []byte("x"),
			environmentValues: []string{"repeat", "repeat", "longer-value"},
			fixtureContents:   [][]byte{[]byte("same"), []byte("same")},
		},
		{
			name: "multi-byte-payloads",
			argv: []string{"repeat", "middle-many", "repeat", "tail-a", "tail-b"}, stdin: []byte{0, 1, 2, 3, 4, 5, 6},
			environmentValues: []string{"abcdef", "xy", ""},
			fixtureContents:   [][]byte{[]byte{9, 8, 7, 6}, []byte("abcdefgh"), []byte{}},
		},
	}
	environmentNames := []string{"CASE_ALPHA", "CASE_BETA", "CASE_GAMMA"}

	for caseIndex, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stdin, err := PresentStdin(testCase.stdin)
			if err != nil {
				t.Fatal(err)
			}
			environment := make([]CLIEnvironmentBinding, 0, len(testCase.environmentValues)+1)
			for index, value := range testCase.environmentValues {
				binding, bindingErr := PresentEnvironment(environmentNames[index], value)
				if bindingErr != nil {
					t.Fatal(bindingErr)
				}
				environment = append(environment, binding)
			}
			absent, err := AbsentEnvironment("CASE_OPTIONAL")
			if err != nil {
				t.Fatal(err)
			}
			environment = append(environment, absent)

			pinnedContents := []byte(fmt.Sprintf(`{"case":%d}`, caseIndex))
			pinned, err := NewFixtureFile("a-pinned.json", pinnedContents, FixtureMode0644)
			if err != nil {
				t.Fatal(err)
			}
			fixtures := []CLIFixtureFile{pinned}
			for index, contents := range testCase.fixtureContents {
				fixture, fixtureErr := NewFixtureFile(fmt.Sprintf("b-noise-%02d.txt", index), contents, FixtureMode0644)
				if fixtureErr != nil {
					t.Fatal(fixtureErr)
				}
				fixtures = append(fixtures, fixture)
			}
			stimulus, err := NewCLIStimulus(CLIStimulusConfig{
				Executable: "node", BaseArgv: []string{"fixture.mjs"}, Argv: testCase.argv,
				Stdin: stdin, Environment: environment, Fixtures: fixtures, CWDPolicy: CWDMaterializedRoot,
			})
			if err != nil {
				t.Fatal(err)
			}
			policy, err := NewCLIReductionPolicy(CLIReductionPolicyConfig{
				Anchor: stimulus, PinnedFixturePaths: []string{"a-pinned.json"}, EnabledRules: DefaultCLIReducers(),
			})
			if err != nil {
				t.Fatal(err)
			}

			first, err := EnumerateCLINeighbors(stimulus, policy)
			if err != nil {
				t.Fatal(err)
			}
			second, err := EnumerateCLINeighbors(stimulus, policy)
			if err != nil {
				t.Fatal(err)
			}
			// Each stdin/argv/fixture can emit at most absence-or-removal plus
			// five shrink transforms; an environment binding can additionally
			// emit explicit absence. Pins and transform deduplication only lower it.
			maximumNeighbors := 6 + 6*len(stimulus.Argv()) + 7*len(stimulus.Environment()) + 6*len(stimulus.Fixtures())
			if len(first) == 0 || len(first) != len(second) || len(first) > maximumNeighbors {
				t.Fatalf("neighbor count %d/%d is outside finite bound %d", len(first), len(second), maximumNeighbors)
			}
			before, err := MeasureCLIStimulus(stimulus, policy)
			if err != nil {
				t.Fatal(err)
			}
			proposalDigests := make(map[string]struct{}, len(first))
			stimulusDigests := make(map[string]struct{}, len(first))
			duplicateSurvivorsObserved := false
			for index := range first {
				left, right := first[index], second[index]
				if left.ReplayRecipe() != right.ReplayRecipe() ||
					left.LogicalNeighbor().Digest() != right.LogicalNeighbor().Digest() ||
					!bytes.Equal(left.LogicalNeighbor().CanonicalBytes(), right.LogicalNeighbor().CanonicalBytes()) ||
					!bytes.Equal(left.Stimulus().CanonicalBytes(), right.Stimulus().CanonicalBytes()) {
					t.Fatalf("neighbor %d changed across repeated enumeration", index)
				}
				logical := left.LogicalNeighbor()
				if _, duplicate := proposalDigests[logical.Digest().String()]; duplicate {
					t.Fatalf("duplicate proposal digest at neighbor %d", index)
				}
				proposalDigests[logical.Digest().String()] = struct{}{}
				if _, duplicate := stimulusDigests[left.Stimulus().Digest().String()]; duplicate {
					t.Fatalf("duplicate stimulus digest at neighbor %d", index)
				}
				stimulusDigests[left.Stimulus().Digest().String()] = struct{}{}
				comparison, compareErr := logical.Measure().Compare(before)
				if compareErr != nil || comparison >= 0 {
					t.Fatalf("neighbor %d did not strictly decrease: %d, %v", index, comparison, compareErr)
				}
				if logical.CurrentStimulusDigest() != stimulus.Digest() ||
					logical.StimulusDigest() != left.Stimulus().Digest() ||
					logical.ReducerSetDigest() != policy.ReducerSet().Digest() {
					t.Fatalf("neighbor %d lost proposal bindings", index)
				}
				replayed, replayErr := ReplayCLINeighbor(stimulus, policy, left.ReplayRecipe())
				if replayErr != nil || replayed.Digest() != left.Stimulus().Digest() ||
					!bytes.Equal(replayed.CanonicalBytes(), left.Stimulus().CanonicalBytes()) ||
					!bytes.Equal(replayed.ExecutionPayloadCanonicalBytes(), left.Stimulus().ExecutionPayloadCanonicalBytes()) {
					t.Fatalf("neighbor %d replay did not reconstruct the exact proposed stimulus: %v", index, replayErr)
				}
				if left.Stimulus().Executable() != stimulus.Executable() ||
					!slices.Equal(left.Stimulus().BaseArgv(), stimulus.BaseArgv()) ||
					left.Stimulus().CWDPolicy() != stimulus.CWDPolicy() {
					t.Fatalf("neighbor %d changed a fixed CLI scope field", index)
				}
				assertPinnedCLIFixture(t, left.Stimulus(), "a-pinned.json", pinnedContents)
				if recipe := left.ReplayRecipe(); recipe.Rule == CLIArgvRemove && recipe.Index == 1 &&
					slices.Equal(left.Stimulus().Argv()[:2], []string{"repeat", "repeat"}) {
					duplicateSurvivorsObserved = true
				}
			}
			if !duplicateSurvivorsObserved {
				t.Fatal("ordered duplicate argv survivors were not preserved")
			}
		})
	}
}

func TestCLINeighborsKeepPresentEmptyDistinctFromAbsentAndOmission(t *testing.T) {
	stimulus := cliNeighborFixture(t, false)
	policy := cliNeighborPolicy(t, stimulus)
	neighbors, err := EnumerateCLINeighbors(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	var stdinAbsent CLIStimulus
	var environmentAbsent CLIStimulus
	var environmentRemoved CLIStimulus
	for _, neighbor := range neighbors {
		recipe := neighbor.ReplayRecipe()
		switch recipe.Rule {
		case CLIStdinAbsent:
			stdinAbsent = neighbor.Stimulus()
		case CLIEnvironmentAbsent:
			if neighbor.Stimulus().Environment()[recipe.Index].Name() == "APP_MODE" {
				environmentAbsent = neighbor.Stimulus()
			}
		case CLIEnvironmentRemove:
			if len(neighbor.Stimulus().Environment()) == 1 {
				environmentRemoved = neighbor.Stimulus()
			}
		}
	}
	if !stdinAbsent.Valid() || stdinAbsent.Stdin().Presence() != PresenceAbsent || stimulus.Stdin().Presence() != PresencePresent {
		t.Fatal("present-empty stdin did not produce a distinct absent neighbor")
	}
	if !environmentAbsent.Valid() || !environmentRemoved.Valid() || environmentAbsent.Digest() == environmentRemoved.Digest() {
		t.Fatal("present-empty environment collapsed absent binding into omitted binding")
	}
	foundAbsent := false
	for _, binding := range environmentAbsent.Environment() {
		if binding.Name() == "APP_MODE" {
			foundAbsent = binding.Presence() == PresenceAbsent
		}
	}
	if !foundAbsent {
		t.Fatal("APP_MODE absent binding was not retained")
	}
}

func TestCLIReductionPolicyIsCanonicalAndRejectsInvalidPinsOrRules(t *testing.T) {
	left := cliNeighborFixture(t, false)
	right := cliNeighborFixture(t, true)
	if left.Digest() != right.Digest() {
		t.Fatal("model did not normalize environment permutations")
	}
	leftPolicy := cliNeighborPolicy(t, left)
	rightPolicy := cliNeighborPolicy(t, right)
	if leftPolicy.Digest() != rightPolicy.Digest() || !bytes.Equal(leftPolicy.CanonicalBytes(), rightPolicy.CanonicalBytes()) {
		t.Fatal("normalized environment permutation changed reduction policy")
	}
	if _, err := NewCLIReductionPolicy(CLIReductionPolicyConfig{
		Anchor: left, PinnedFixturePaths: []string{"missing.txt"}, EnabledRules: DefaultCLIReducers(),
	}); err == nil {
		t.Fatal("missing pinned fixture entered a policy")
	}
	if _, err := NewCLIReductionPolicy(CLIReductionPolicyConfig{
		Anchor: left, PinnedFixturePaths: []string{"config.json"}, EnabledRules: []CLIReducerID{CLIArgvRemove, CLIArgvRemove},
	}); err == nil {
		t.Fatal("duplicate rule entered a policy")
	}
	if _, err := NewCLIReductionPolicy(CLIReductionPolicyConfig{
		Anchor: left, PinnedFixturePaths: []string{"config.json"}, EnabledRules: []CLIReducerID{"cli.unknown"},
	}); err == nil {
		t.Fatal("unknown rule entered a policy")
	}
}

func TestCLIOrderedArgvRemovalDoesNotReorderSurvivors(t *testing.T) {
	stimulus := cliNeighborFixture(t, false)
	policy := cliNeighborPolicy(t, stimulus)
	neighbors, err := EnumerateCLINeighbors(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, neighbor := range neighbors {
		recipe := neighbor.ReplayRecipe()
		if recipe.Rule == CLIArgvRemove && recipe.Index == 2 {
			if !slices.Equal(neighbor.Stimulus().Argv(), []string{"--mode", "argv"}) {
				t.Fatalf("argv survivors reordered: %#v", neighbor.Stimulus().Argv())
			}
			found = true
		}
	}
	if !found {
		t.Fatal("expected suffix argv removal neighbor")
	}
}

func assertPinnedCLIFixture(t *testing.T, stimulus CLIStimulus, path string, contents []byte) {
	t.Helper()
	for _, fixture := range stimulus.Fixtures() {
		if fixture.Path() == path {
			if fixture.Mode() != FixtureMode0644 || !bytes.Equal(fixture.Contents(), contents) {
				t.Fatalf("pinned fixture %s mutated", path)
			}
			return
		}
	}
	t.Fatalf("pinned fixture %s was removed", path)
}
