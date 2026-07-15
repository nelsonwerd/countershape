package cli

import (
	"bytes"
	"slices"
	"sort"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/reduce"
)

// CLIReducerID is the closed v1 set of typed CLI transforms. The order in
// DefaultCLIReducers is search order; reducer-set identity also binds that
// order through CLIReductionPolicy.
type CLIReducerID string

const (
	CLIStdinAbsent       CLIReducerID = "cli.stdin.absent"
	CLIFixtureRemove     CLIReducerID = "cli.fixture.remove"
	CLIEnvironmentRemove CLIReducerID = "cli.environment.remove"
	CLIArgvRemove        CLIReducerID = "cli.argv.remove"
	CLIEnvironmentAbsent CLIReducerID = "cli.environment.absent"
	CLIStdinShrink       CLIReducerID = "cli.stdin.shrink"
	CLIFixtureShrink     CLIReducerID = "cli.fixture.shrink"
	CLIEnvironmentShrink CLIReducerID = "cli.environment.shrink"
	CLIArgvShrink        CLIReducerID = "cli.argv.shrink"
)

var cliReducerOrder = []CLIReducerID{
	CLIStdinAbsent,
	CLIFixtureRemove,
	CLIEnvironmentRemove,
	CLIArgvRemove,
	CLIEnvironmentAbsent,
	CLIStdinShrink,
	CLIFixtureShrink,
	CLIEnvironmentShrink,
	CLIArgvShrink,
}

func DefaultCLIReducers() []CLIReducerID {
	return append([]CLIReducerID(nil), cliReducerOrder...)
}

type CLIReductionPolicyConfig struct {
	Anchor             CLIStimulus
	PinnedFixturePaths []string
	EnabledRules       []CLIReducerID
}

type cliPinnedFixtureIdentity struct {
	Path           string `json:"path"`
	Mode           string `json:"mode"`
	ContentsDigest string `json:"contents_digest"`
}

type cliReductionPolicyIdentity struct {
	SchemaVersion          string                     `json:"schema_version"`
	Kind                   string                     `json:"kind"`
	Version                string                     `json:"version"`
	FixedExecutable        string                     `json:"fixed_executable"`
	FixedBaseArgv          []string                   `json:"fixed_base_argv"`
	FixedCWDPolicy         string                     `json:"fixed_cwd_policy"`
	PinnedFixtures         []cliPinnedFixtureIdentity `json:"pinned_fixtures"`
	EnabledRulesInPriority []string                   `json:"enabled_rules_in_priority_order"`
}

// CLIReductionPolicy is canonical scope authority. It pins executable, base
// argv, CWD policy, and any fixture files declared semantically fixed. It does
// not bind reducible argv, stdin, environment, or unpinned fixture content.
type CLIReductionPolicy struct {
	digest         domain.Digest
	canonicalBytes []byte
	anchor         CLIStimulus
	pinnedPaths    []string
	pinned         []cliPinnedFixtureIdentity
	enabled        []CLIReducerID
	measureDef     domain.Digest
	reducerSet     reduce.ReducerSet
}

func NewCLIReductionPolicy(config CLIReductionPolicyConfig) (CLIReductionPolicy, error) {
	if !config.Anchor.Valid() {
		return CLIReductionPolicy{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_POLICY"}
	}
	enabled, err := normalizeCLIReducers(config.EnabledRules)
	if err != nil {
		return CLIReductionPolicy{}, err
	}
	pinnedPaths := append([]string(nil), config.PinnedFixturePaths...)
	sort.Strings(pinnedPaths)
	for index := range pinnedPaths {
		if pinnedPaths[index] == "" || index > 0 && pinnedPaths[index-1] == pinnedPaths[index] {
			return CLIReductionPolicy{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_PIN_SET"}
		}
	}
	fixtureByPath := make(map[string]CLIFixtureFile, len(config.Anchor.Fixtures()))
	for _, fixture := range config.Anchor.Fixtures() {
		fixtureByPath[fixture.Path()] = fixture
	}
	pinned := make([]cliPinnedFixtureIdentity, len(pinnedPaths))
	for index, path := range pinnedPaths {
		fixture, present := fixtureByPath[path]
		if !present {
			return CLIReductionPolicy{}, &domain.Error{Code: "CLI_REDUCTION_PIN_NOT_IN_ANCHOR", Detail: path}
		}
		contents, err := canon.DigestBytes("CLIReductionPinnedFixtureContents", fixture.Contents())
		if err != nil {
			return CLIReductionPolicy{}, err
		}
		pinned[index] = cliPinnedFixtureIdentity{
			Path: path, Mode: string(fixture.Mode()), ContentsDigest: contents.String(),
		}
	}
	measureDef, err := cliReductionMeasureDefinition()
	if err != nil {
		return CLIReductionPolicy{}, err
	}
	rules := make([]reduce.ReducerRule, len(enabled))
	ruleNames := make([]string, len(enabled))
	for index, id := range enabled {
		rules[index], err = reduce.NewReducerRule(string(id), "v1")
		if err != nil {
			return CLIReductionPolicy{}, err
		}
		ruleNames[index] = string(id)
	}
	identity := cliReductionPolicyIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIReductionPolicy", Version: "cli-reduction-policy/v1",
		FixedExecutable: config.Anchor.Executable(), FixedBaseArgv: config.Anchor.BaseArgv(),
		FixedCWDPolicy: string(config.Anchor.CWDPolicy()), PinnedFixtures: pinned,
		EnabledRulesInPriority: ruleNames,
	}
	digest, canonicalBytes, err := digestCLIReductionTyped("CLIReductionPolicy", identity)
	if err != nil {
		return CLIReductionPolicy{}, err
	}
	reducerSet, err := reduce.NewReducerSet("cli", measureDef, digest, rules)
	if err != nil {
		return CLIReductionPolicy{}, err
	}
	return CLIReductionPolicy{
		digest: digest, canonicalBytes: canonicalBytes, anchor: config.Anchor,
		pinnedPaths: pinnedPaths, pinned: pinned, enabled: enabled,
		measureDef: measureDef, reducerSet: reducerSet,
	}, nil
}

func normalizeCLIReducers(input []CLIReducerID) ([]CLIReducerID, error) {
	if len(input) == 0 {
		return nil, &domain.Error{Code: "CLI_EMPTY_REDUCER_SET"}
	}
	priorities := make(map[CLIReducerID]int, len(cliReducerOrder))
	for index, id := range cliReducerOrder {
		priorities[id] = index
	}
	seen := make(map[CLIReducerID]struct{}, len(input))
	result := append([]CLIReducerID(nil), input...)
	for _, id := range result {
		if _, valid := priorities[id]; !valid {
			return nil, &domain.Error{Code: "CLI_UNKNOWN_REDUCER_RULE", Detail: string(id)}
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, &domain.Error{Code: "CLI_DUPLICATE_REDUCER_RULE", Detail: string(id)}
		}
		seen[id] = struct{}{}
	}
	sort.Slice(result, func(left, right int) bool { return priorities[result[left]] < priorities[result[right]] })
	return result, nil
}

func cliReductionMeasureDefinition() (domain.Digest, error) {
	digest, _, err := digestCLIReductionTyped("CLIReductionMeasureDefinition", struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Ordering      string   `json:"ordering"`
		Dimensions    []string `json:"dimensions"`
		Fixed         []string `json:"fixed_fields"`
	}{
			domain.SchemaVersion, "CLIReductionMeasureDefinition", "cli-reduction-measure/v1", "LEXICOGRAPHIC",
			[]string{"structural_and_presence_atoms", "mutable_payload_bytes"},
			[]string{"executable", "base_argv", "cwd_policy", "pinned_fixture_paths_modes_contents"},
	})
	return digest, err
}

func (p CLIReductionPolicy) Valid() bool {
	rebuilt, err := NewCLIReductionPolicy(CLIReductionPolicyConfig{
		Anchor: p.anchor, PinnedFixturePaths: p.pinnedPaths, EnabledRules: p.enabled,
	})
	return err == nil && rebuilt.digest == p.digest && bytes.Equal(rebuilt.canonicalBytes, p.canonicalBytes) &&
		rebuilt.measureDef == p.measureDef && rebuilt.reducerSet.Digest() == p.reducerSet.Digest()
}

func (p CLIReductionPolicy) Digest() domain.Digest  { return p.digest }
func (p CLIReductionPolicy) CanonicalBytes() []byte { return append([]byte(nil), p.canonicalBytes...) }
func (p CLIReductionPolicy) EnabledRules() []CLIReducerID {
	return append([]CLIReducerID(nil), p.enabled...)
}
func (p CLIReductionPolicy) PinnedFixturePaths() []string {
	return append([]string(nil), p.pinnedPaths...)
}
func (p CLIReductionPolicy) MeasureDefinitionDigest() domain.Digest { return p.measureDef }
func (p CLIReductionPolicy) ReducerSet() reduce.ReducerSet          { return p.reducerSet }

func (p CLIReductionPolicy) matches(stimulus CLIStimulus) bool {
	if !p.Valid() || !stimulus.Valid() || stimulus.Executable() != p.anchor.Executable() ||
		!slices.Equal(stimulus.BaseArgv(), p.anchor.BaseArgv()) || stimulus.CWDPolicy() != p.anchor.CWDPolicy() {
		return false
	}
	byPath := make(map[string]CLIFixtureFile, len(stimulus.Fixtures()))
	for _, fixture := range stimulus.Fixtures() {
		byPath[fixture.Path()] = fixture
	}
	for _, pinned := range p.pinned {
		fixture, present := byPath[pinned.Path]
		if !present || string(fixture.Mode()) != pinned.Mode {
			return false
		}
		digest, err := canon.DigestBytes("CLIReductionPinnedFixtureContents", fixture.Contents())
		if err != nil || digest.String() != pinned.ContentsDigest {
			return false
		}
	}
	return true
}

func (p CLIReductionPolicy) enabledRule(id CLIReducerID) bool {
	return slices.Contains(p.enabled, id)
}

func (p CLIReductionPolicy) rule(id CLIReducerID) (reduce.ReducerRule, error) {
	if !p.enabledRule(id) {
		return reduce.ReducerRule{}, &domain.Error{Code: "CLI_REDUCER_RULE_DISABLED", Detail: string(id)}
	}
	return reduce.NewReducerRule(string(id), "v1")
}

// MeasureCLIStimulus returns the U5 measure without changing the U3 stimulus
// identity. Present-empty remains an atom distinct from absence.
func MeasureCLIStimulus(stimulus CLIStimulus, policy CLIReductionPolicy) (reduce.Measure, error) {
	if !policy.matches(stimulus) {
		return reduce.Measure{}, &domain.Error{Code: "CLI_STIMULUS_OUTSIDE_REDUCTION_SCOPE"}
	}
	atoms := uint64(len(stimulus.Argv()) + len(stimulus.Environment()) + len(stimulus.Fixtures()))
	mutableBytes := uint64(0)
	for _, argument := range stimulus.Argv() {
		mutableBytes += uint64(len(argument))
	}
	stdin := stimulus.Stdin()
	if stdin.Present() {
		atoms++
		mutableBytes += uint64(len(stdin.Bytes()))
	}
	for _, binding := range stimulus.Environment() {
		if binding.Present() {
			atoms++
		}
		mutableBytes += uint64(len(binding.Value()))
	}
	for _, fixture := range stimulus.Fixtures() {
		mutableBytes += uint64(len(fixture.Contents()))
	}
	return reduce.NewMeasure(policy.measureDef, []uint64{atoms, mutableBytes})
}

type CLIReductionTransform string

const (
	CLITransformAbsent     CLIReductionTransform = "absent"
	CLITransformRemove     CLIReductionTransform = "remove"
	CLITransformEmpty      CLIReductionTransform = "empty"
	CLITransformPrefixHalf CLIReductionTransform = "prefix-half"
	CLITransformSuffixHalf CLIReductionTransform = "suffix-half"
	CLITransformDropFirst  CLIReductionTransform = "drop-first"
	CLITransformDropLast   CLIReductionTransform = "drop-last"
)

type CLIReplayRecipe struct {
	Rule      CLIReducerID
	Index     int
	Transform CLIReductionTransform
}

type CLINeighbor struct {
	stimulus CLIStimulus
	neighbor reduce.Neighbor
	replay   CLIReplayRecipe
}

func (n CLINeighbor) Stimulus() CLIStimulus            { return n.stimulus }
func (n CLINeighbor) Neighbor() reduce.Neighbor        { return n.neighbor }
func (n CLINeighbor) LogicalNeighbor() reduce.Neighbor { return n.neighbor }
func (n CLINeighbor) ReplayRecipe() CLIReplayRecipe    { return n.replay }

// ReplayCLINeighbor applies one typed recipe and always reconstructs through
// NewCLIStimulus. Arbitrary replacement bytes never cross this API.
func ReplayCLINeighbor(parent CLIStimulus, policy CLIReductionPolicy, recipe CLIReplayRecipe) (CLIStimulus, error) {
	if !policy.matches(parent) || !policy.enabledRule(recipe.Rule) {
		return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
	}
	config := CLIStimulusConfig{
		Executable: parent.Executable(), BaseArgv: parent.BaseArgv(), Argv: parent.Argv(),
		Stdin: parent.Stdin(), Environment: parent.Environment(), Fixtures: parent.Fixtures(),
		CWDPolicy: parent.CWDPolicy(),
	}
	switch recipe.Rule {
	case CLIStdinAbsent:
		if recipe.Index != -1 || recipe.Transform != CLITransformAbsent || !config.Stdin.Present() {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		config.Stdin = AbsentStdin()
	case CLIFixtureRemove:
		if recipe.Transform != CLITransformRemove || !validIndex(recipe.Index, len(config.Fixtures)) || policy.fixturePinned(config.Fixtures[recipe.Index].Path()) {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		config.Fixtures = removeAt(config.Fixtures, recipe.Index)
	case CLIEnvironmentRemove:
		if recipe.Transform != CLITransformRemove || !validIndex(recipe.Index, len(config.Environment)) {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		config.Environment = removeAt(config.Environment, recipe.Index)
	case CLIArgvRemove:
		if recipe.Transform != CLITransformRemove || !validIndex(recipe.Index, len(config.Argv)) {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		config.Argv = removeAt(config.Argv, recipe.Index)
	case CLIEnvironmentAbsent:
		if recipe.Transform != CLITransformAbsent || !validIndex(recipe.Index, len(config.Environment)) || !config.Environment[recipe.Index].Present() {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		binding, err := AbsentEnvironment(config.Environment[recipe.Index].Name())
		if err != nil {
			return CLIStimulus{}, err
		}
		config.Environment[recipe.Index] = binding
	case CLIStdinShrink:
		if recipe.Index != -1 || !config.Stdin.Present() {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		value, err := shrinkBytes(config.Stdin.Bytes(), recipe.Transform)
		if err != nil {
			return CLIStimulus{}, err
		}
		config.Stdin, err = PresentStdin(value)
		if err != nil {
			return CLIStimulus{}, err
		}
	case CLIFixtureShrink:
		if !validIndex(recipe.Index, len(config.Fixtures)) || policy.fixturePinned(config.Fixtures[recipe.Index].Path()) {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		fixture := config.Fixtures[recipe.Index]
		value, err := shrinkBytes(fixture.Contents(), recipe.Transform)
		if err != nil {
			return CLIStimulus{}, err
		}
		config.Fixtures[recipe.Index], err = NewFixtureFile(fixture.Path(), value, fixture.Mode())
		if err != nil {
			return CLIStimulus{}, err
		}
	case CLIEnvironmentShrink:
		if !validIndex(recipe.Index, len(config.Environment)) || !config.Environment[recipe.Index].Present() {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		binding := config.Environment[recipe.Index]
		value, err := shrinkText(binding.Value(), recipe.Transform)
		if err != nil {
			return CLIStimulus{}, err
		}
		config.Environment[recipe.Index], err = PresentEnvironment(binding.Name(), value)
		if err != nil {
			return CLIStimulus{}, err
		}
	case CLIArgvShrink:
		if !validIndex(recipe.Index, len(config.Argv)) {
			return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
		}
		value, err := shrinkText(config.Argv[recipe.Index], recipe.Transform)
		if err != nil {
			return CLIStimulus{}, err
		}
		config.Argv[recipe.Index] = value
	default:
		return CLIStimulus{}, &domain.Error{Code: "CLI_INVALID_REDUCTION_REPLAY"}
	}
	return NewCLIStimulus(config)
}

func (p CLIReductionPolicy) fixturePinned(path string) bool {
	index := sort.SearchStrings(p.pinnedPaths, path)
	return index < len(p.pinnedPaths) && p.pinnedPaths[index] == path
}

type cliNeighborCandidate struct {
	child     CLIStimulus
	recipe    CLIReplayRecipe
	rule      reduce.ReducerRule
	locus     string
	priority  int
	transform int
	measure   reduce.Measure
}

func EnumerateCLINeighbors(current CLIStimulus, policy CLIReductionPolicy) ([]CLINeighbor, error) {
	currentMeasure, err := MeasureCLIStimulus(current, policy)
	if err != nil {
		return nil, err
	}
	var candidates []cliNeighborCandidate
	for priority, ruleID := range policy.enabled {
		rule, err := policy.rule(ruleID)
		if err != nil {
			return nil, err
		}
		recipes := cliRecipes(current, policy, ruleID)
		for _, recipe := range recipes {
			child, replayErr := ReplayCLINeighbor(current, policy, recipe)
			if replayErr != nil {
				// Some bounded textual reductions are rejected by the closed CLI
				// grammar. They are not valid direct neighbors.
				continue
			}
			measure, measureErr := MeasureCLIStimulus(child, policy)
			if measureErr != nil {
				return nil, measureErr
			}
			if comparison, compareErr := measure.Compare(currentMeasure); compareErr != nil || comparison >= 0 {
				return nil, &domain.Error{Code: "CLI_NONDECREASING_TYPED_NEIGHBOR"}
			}
			candidates = append(candidates, cliNeighborCandidate{
				child: child, recipe: recipe, rule: rule, locus: cliRecipeLocus(recipe),
				priority: priority, transform: cliTransformPriority(recipe.Transform), measure: measure,
			})
		}
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].priority != candidates[right].priority {
			return candidates[left].priority < candidates[right].priority
		}
		if candidates[left].locus != candidates[right].locus {
			return candidates[left].locus < candidates[right].locus
		}
		if candidates[left].transform != candidates[right].transform {
			return candidates[left].transform < candidates[right].transform
		}
		return candidates[left].child.Digest().String() < candidates[right].child.Digest().String()
	})
	seen := make(map[domain.Digest]struct{}, len(candidates))
	result := make([]CLINeighbor, 0, len(candidates))
	for _, candidate := range candidates {
		if _, duplicate := seen[candidate.child.Digest()]; duplicate {
			continue
		}
		seen[candidate.child.Digest()] = struct{}{}
		logical, err := reduce.NewNeighbor(reduce.NeighborInput{
			CurrentStimulus: current.Digest(), CurrentMeasure: currentMeasure,
			Stimulus: candidate.child.Digest(), Measure: candidate.measure, Rule: candidate.rule,
			Locus: candidate.locus, TransformPriority: uint64(candidate.transform), ReducerSet: policy.reducerSet,
		})
		if err != nil {
			return nil, err
		}
		replayed, err := ReplayCLINeighbor(current, policy, candidate.recipe)
		if err != nil || replayed.Digest() != candidate.child.Digest() {
			return nil, &domain.Error{Code: "CLI_REDUCTION_REPLAY_MISMATCH"}
		}
		result = append(result, CLINeighbor{stimulus: candidate.child, neighbor: logical, replay: candidate.recipe})
	}
	return result, nil
}

func cliRecipes(current CLIStimulus, policy CLIReductionPolicy, rule CLIReducerID) []CLIReplayRecipe {
	var result []CLIReplayRecipe
	switch rule {
	case CLIStdinAbsent:
		if current.Stdin().Present() {
			result = append(result, CLIReplayRecipe{rule, -1, CLITransformAbsent})
		}
	case CLIFixtureRemove:
		fixtures := current.Fixtures()
		for index := len(fixtures) - 1; index >= 0; index-- {
			if !policy.fixturePinned(fixtures[index].Path()) {
				result = append(result, CLIReplayRecipe{rule, index, CLITransformRemove})
			}
		}
	case CLIEnvironmentRemove:
		for index := len(current.Environment()) - 1; index >= 0; index-- {
			result = append(result, CLIReplayRecipe{rule, index, CLITransformRemove})
		}
	case CLIArgvRemove:
		for index := len(current.Argv()) - 1; index >= 0; index-- {
			result = append(result, CLIReplayRecipe{rule, index, CLITransformRemove})
		}
	case CLIEnvironmentAbsent:
		environment := current.Environment()
		for index := len(environment) - 1; index >= 0; index-- {
			if environment[index].Present() {
				result = append(result, CLIReplayRecipe{rule, index, CLITransformAbsent})
			}
		}
	case CLIStdinShrink:
		if current.Stdin().Present() {
			for _, transform := range availableByteTransforms(current.Stdin().Bytes()) {
				result = append(result, CLIReplayRecipe{rule, -1, transform})
			}
		}
	case CLIFixtureShrink:
		fixtures := current.Fixtures()
		for index := len(fixtures) - 1; index >= 0; index-- {
			if policy.fixturePinned(fixtures[index].Path()) {
				continue
			}
			for _, transform := range availableByteTransforms(fixtures[index].Contents()) {
				result = append(result, CLIReplayRecipe{rule, index, transform})
			}
		}
	case CLIEnvironmentShrink:
		environment := current.Environment()
		for index := len(environment) - 1; index >= 0; index-- {
			if !environment[index].Present() {
				continue
			}
			for _, transform := range availableTextTransforms(environment[index].Value()) {
				result = append(result, CLIReplayRecipe{rule, index, transform})
			}
		}
	case CLIArgvShrink:
		argv := current.Argv()
		for index := len(argv) - 1; index >= 0; index-- {
			for _, transform := range availableTextTransforms(argv[index]) {
				result = append(result, CLIReplayRecipe{rule, index, transform})
			}
		}
	}
	return result
}

func cliRecipeLocus(recipe CLIReplayRecipe) string {
	switch recipe.Rule {
	case CLIStdinAbsent, CLIStdinShrink:
		return "stdin"
	case CLIFixtureRemove, CLIFixtureShrink:
		return "fixture[" + decimal(recipe.Index) + "]"
	case CLIEnvironmentRemove, CLIEnvironmentAbsent, CLIEnvironmentShrink:
		return "environment[" + decimal(recipe.Index) + "]"
	default:
		return "argv[" + decimal(recipe.Index) + "]"
	}
}

func cliTransformPriority(value CLIReductionTransform) int {
	switch value {
	case CLITransformAbsent:
		return 0
	case CLITransformRemove:
		return 1
	case CLITransformEmpty:
		return 2
	case CLITransformPrefixHalf:
		return 3
	case CLITransformSuffixHalf:
		return 4
	case CLITransformDropFirst:
		return 5
	case CLITransformDropLast:
		return 6
	default:
		return 99
	}
}

func availableByteTransforms(input []byte) []CLIReductionTransform {
	return uniqueByteTransforms(input, []CLIReductionTransform{
		CLITransformEmpty, CLITransformPrefixHalf, CLITransformSuffixHalf, CLITransformDropFirst, CLITransformDropLast,
	})
}

func uniqueByteTransforms(input []byte, transforms []CLIReductionTransform) []CLIReductionTransform {
	seen := map[string]struct{}{}
	result := make([]CLIReductionTransform, 0, len(transforms))
	for _, transform := range transforms {
		value, err := shrinkBytes(input, transform)
		if err != nil || len(value) >= len(input) {
			continue
		}
		key := string(value)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, transform)
	}
	return result
}

func availableTextTransforms(input string) []CLIReductionTransform {
	seen := map[string]struct{}{}
	result := make([]CLIReductionTransform, 0, 5)
	for _, transform := range []CLIReductionTransform{CLITransformEmpty, CLITransformPrefixHalf, CLITransformSuffixHalf, CLITransformDropFirst, CLITransformDropLast} {
		value, err := shrinkText(input, transform)
		if err != nil || len(value) >= len(input) {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, transform)
	}
	return result
}

func shrinkBytes(input []byte, transform CLIReductionTransform) ([]byte, error) {
	if len(input) == 0 {
		return nil, &domain.Error{Code: "CLI_EMPTY_SHRINK_INPUT"}
	}
	var value []byte
	switch transform {
	case CLITransformEmpty:
		value = []byte{}
	case CLITransformPrefixHalf:
		value = input[:len(input)/2]
	case CLITransformSuffixHalf:
		value = input[len(input)-len(input)/2:]
	case CLITransformDropFirst:
		value = input[1:]
	case CLITransformDropLast:
		value = input[:len(input)-1]
	default:
		return nil, &domain.Error{Code: "CLI_INVALID_SHRINK_TRANSFORM"}
	}
	return append([]byte(nil), value...), nil
}

func shrinkText(input string, transform CLIReductionTransform) (string, error) {
	if !utf8.ValidString(input) || input == "" {
		return "", &domain.Error{Code: "CLI_EMPTY_SHRINK_INPUT"}
	}
	runes := []rune(input)
	var value []rune
	switch transform {
	case CLITransformEmpty:
		value = []rune{}
	case CLITransformPrefixHalf:
		value = runes[:len(runes)/2]
	case CLITransformSuffixHalf:
		value = runes[len(runes)-len(runes)/2:]
	case CLITransformDropFirst:
		value = runes[1:]
	case CLITransformDropLast:
		value = runes[:len(runes)-1]
	default:
		return "", &domain.Error{Code: "CLI_INVALID_SHRINK_TRANSFORM"}
	}
	return string(value), nil
}

func validIndex(index, length int) bool { return index >= 0 && index < length }

func removeAt[T any](input []T, index int) []T {
	result := make([]T, 0, len(input)-1)
	result = append(result, input[:index]...)
	return append(result, input[index+1:]...)
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [24]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[position:])
}

func digestCLIReductionTyped(kind string, value any) (domain.Digest, []byte, error) {
	digest, canonicalBytes, err := canon.DigestTyped(kind, value)
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	return parsed, canonicalBytes, err
}
