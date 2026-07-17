package http

import (
	"bytes"
	"slices"
	"sort"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/reduce"
)

type HTTPReducerID string

const (
	HTTPBodyAbsent   HTTPReducerID = "http.body.absent"
	HTTPSeedRemove   HTTPReducerID = "http.seed.remove"
	HTTPQueryRemove  HTTPReducerID = "http.query.remove"
	HTTPHeaderRemove HTTPReducerID = "http.header.remove"
	HTTPQueryFlag    HTTPReducerID = "http.query.flag"
	HTTPBodyShrink   HTTPReducerID = "http.body.shrink"
	HTTPSeedShrink   HTTPReducerID = "http.seed.shrink"
	HTTPQueryShrink  HTTPReducerID = "http.query.shrink"
	HTTPHeaderShrink HTTPReducerID = "http.header.shrink"
)

var httpReducerOrder = []HTTPReducerID{
	HTTPBodyAbsent,
	HTTPSeedRemove,
	HTTPQueryRemove,
	HTTPHeaderRemove,
	HTTPQueryFlag,
	HTTPBodyShrink,
	HTTPSeedShrink,
	HTTPQueryShrink,
	HTTPHeaderShrink,
}

func DefaultHTTPReducers() []HTTPReducerID {
	return append([]HTTPReducerID(nil), httpReducerOrder...)
}

type HTTPReductionPolicyConfig struct {
	Anchor          HTTPStimulus
	PinnedSeedPaths []string
	EnabledRules    []HTTPReducerID
}

type httpPinnedSeedIdentity struct {
	Path           string `json:"path"`
	Mode           string `json:"mode"`
	ContentsDigest string `json:"contents_digest"`
}

type httpReductionPolicyIdentity struct {
	SchemaVersion          string                   `json:"schema_version"`
	Kind                   string                   `json:"kind"`
	Version                string                   `json:"version"`
	FixedMethod            string                   `json:"fixed_method"`
	FixedPath              string                   `json:"fixed_path"`
	FixedExecutionShape    string                   `json:"fixed_execution_shape"`
	PinnedSeeds            []httpPinnedSeedIdentity `json:"pinned_seeds"`
	EnabledRulesInPriority []string                 `json:"enabled_rules_in_priority_order"`
}

// HTTPReductionPolicy fixes method/path/one-request shape and exact pinned
// seed files. Ordered query/header members, body, and unpinned seeds remain
// typed reducer inputs.
type HTTPReductionPolicy struct {
	digest         domain.Digest
	canonicalBytes []byte
	anchor         HTTPStimulus
	pinnedPaths    []string
	pinned         []httpPinnedSeedIdentity
	enabled        []HTTPReducerID
	measureDef     domain.Digest
	reducerSet     reduce.ReducerSet
}

func NewHTTPReductionPolicy(config HTTPReductionPolicyConfig) (HTTPReductionPolicy, error) {
	if !config.Anchor.Valid() {
		return HTTPReductionPolicy{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_POLICY"}
	}
	enabled, err := normalizeHTTPReducers(config.EnabledRules)
	if err != nil {
		return HTTPReductionPolicy{}, err
	}
	pinnedPaths := append([]string(nil), config.PinnedSeedPaths...)
	sort.Strings(pinnedPaths)
	for index := range pinnedPaths {
		if pinnedPaths[index] == "" || index > 0 && pinnedPaths[index-1] == pinnedPaths[index] {
			return HTTPReductionPolicy{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_PIN_SET"}
		}
	}
	seedByPath := make(map[string]HTTPSeedFile, len(config.Anchor.Seeds()))
	for _, seed := range config.Anchor.Seeds() {
		seedByPath[seed.Path()] = seed
	}
	pinned := make([]httpPinnedSeedIdentity, len(pinnedPaths))
	for index, path := range pinnedPaths {
		seed, present := seedByPath[path]
		if !present {
			return HTTPReductionPolicy{}, &domain.Error{Code: "HTTP_REDUCTION_PIN_NOT_IN_ANCHOR", Detail: path}
		}
		contents, err := canon.DigestBytes("HTTPReductionPinnedSeedContents", seed.Contents())
		if err != nil {
			return HTTPReductionPolicy{}, err
		}
		pinned[index] = httpPinnedSeedIdentity{Path: path, Mode: string(seed.Mode()), ContentsDigest: contents.String()}
	}
	measureDef, err := httpReductionMeasureDefinition()
	if err != nil {
		return HTTPReductionPolicy{}, err
	}
	rules := make([]reduce.ReducerRule, len(enabled))
	ruleNames := make([]string, len(enabled))
	for index, id := range enabled {
		rules[index], err = reduce.NewReducerRule(string(id), "v1")
		if err != nil {
			return HTTPReductionPolicy{}, err
		}
		ruleNames[index] = string(id)
	}
	identity := httpReductionPolicyIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPReductionPolicy", Version: "http-reduction-policy/v1",
		FixedMethod: string(config.Anchor.Method()), FixedPath: config.Anchor.Path(),
		FixedExecutionShape: string(domain.OneLoopbackHTTPRequest), PinnedSeeds: pinned,
		EnabledRulesInPriority: ruleNames,
	}
	digest, canonicalBytes, err := digestTyped("HTTPReductionPolicy", identity)
	if err != nil {
		return HTTPReductionPolicy{}, err
	}
	reducerSet, err := reduce.NewReducerSet("http", measureDef, digest, rules)
	if err != nil {
		return HTTPReductionPolicy{}, err
	}
	return HTTPReductionPolicy{
		digest: digest, canonicalBytes: canonicalBytes, anchor: config.Anchor,
		pinnedPaths: pinnedPaths, pinned: pinned, enabled: enabled,
		measureDef: measureDef, reducerSet: reducerSet,
	}, nil
}

func normalizeHTTPReducers(input []HTTPReducerID) ([]HTTPReducerID, error) {
	if len(input) == 0 {
		return nil, &domain.Error{Code: "HTTP_EMPTY_REDUCER_SET"}
	}
	priorities := make(map[HTTPReducerID]int, len(httpReducerOrder))
	for index, id := range httpReducerOrder {
		priorities[id] = index
	}
	seen := make(map[HTTPReducerID]struct{}, len(input))
	result := append([]HTTPReducerID(nil), input...)
	for _, id := range result {
		if _, valid := priorities[id]; !valid {
			return nil, &domain.Error{Code: "HTTP_UNKNOWN_REDUCER_RULE", Detail: string(id)}
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, &domain.Error{Code: "HTTP_DUPLICATE_REDUCER_RULE", Detail: string(id)}
		}
		seen[id] = struct{}{}
	}
	sort.Slice(result, func(left, right int) bool { return priorities[result[left]] < priorities[result[right]] })
	return result, nil
}

func httpReductionMeasureDefinition() (domain.Digest, error) {
	digest, _, err := digestTyped("HTTPReductionMeasureDefinition", struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Ordering      string   `json:"ordering"`
		Dimensions    []string `json:"dimensions"`
		Fixed         []string `json:"fixed_fields"`
	}{
		domain.SchemaVersion, "HTTPReductionMeasureDefinition", "http-reduction-measure/v1", "LEXICOGRAPHIC",
		[]string{"structural_and_presence_atoms", "mutable_payload_bytes"},
		[]string{"method", "path", "one_request_profile", "pinned_seed_paths_modes_contents"},
	})
	return digest, err
}

func (p HTTPReductionPolicy) Valid() bool {
	rebuilt, err := NewHTTPReductionPolicy(HTTPReductionPolicyConfig{
		Anchor: p.anchor, PinnedSeedPaths: p.pinnedPaths, EnabledRules: p.enabled,
	})
	return err == nil && rebuilt.digest == p.digest && bytes.Equal(rebuilt.canonicalBytes, p.canonicalBytes) &&
		rebuilt.measureDef == p.measureDef && rebuilt.reducerSet.Digest() == p.reducerSet.Digest()
}

func (p HTTPReductionPolicy) Digest() domain.Digest  { return p.digest }
func (p HTTPReductionPolicy) CanonicalBytes() []byte { return append([]byte(nil), p.canonicalBytes...) }
func (p HTTPReductionPolicy) EnabledRules() []HTTPReducerID {
	return append([]HTTPReducerID(nil), p.enabled...)
}
func (p HTTPReductionPolicy) PinnedSeedPaths() []string {
	return append([]string(nil), p.pinnedPaths...)
}
func (p HTTPReductionPolicy) MeasureDefinitionDigest() domain.Digest { return p.measureDef }
func (p HTTPReductionPolicy) ReducerSet() reduce.ReducerSet          { return p.reducerSet }

func (p HTTPReductionPolicy) matches(stimulus HTTPStimulus) bool {
	if !p.Valid() || !stimulus.Valid() || stimulus.Method() != p.anchor.Method() || stimulus.Path() != p.anchor.Path() {
		return false
	}
	byPath := make(map[string]HTTPSeedFile, len(stimulus.Seeds()))
	for _, seed := range stimulus.Seeds() {
		byPath[seed.Path()] = seed
	}
	for _, pinned := range p.pinned {
		seed, present := byPath[pinned.Path]
		if !present || string(seed.Mode()) != pinned.Mode {
			return false
		}
		digest, err := canon.DigestBytes("HTTPReductionPinnedSeedContents", seed.Contents())
		if err != nil || digest.String() != pinned.ContentsDigest {
			return false
		}
	}
	return true
}

func (p HTTPReductionPolicy) enabledRule(id HTTPReducerID) bool {
	return slices.Contains(p.enabled, id)
}
func (p HTTPReductionPolicy) rule(id HTTPReducerID) (reduce.ReducerRule, error) {
	if !p.enabledRule(id) {
		return reduce.ReducerRule{}, &domain.Error{Code: "HTTP_REDUCER_RULE_DISABLED", Detail: string(id)}
	}
	return reduce.NewReducerRule(string(id), "v1")
}
func (p HTTPReductionPolicy) seedPinned(path string) bool {
	index := sort.SearchStrings(p.pinnedPaths, path)
	return index < len(p.pinnedPaths) && p.pinnedPaths[index] == path
}

func MeasureHTTPStimulus(stimulus HTTPStimulus, policy HTTPReductionPolicy) (reduce.Measure, error) {
	if !policy.matches(stimulus) {
		return reduce.Measure{}, &domain.Error{Code: "HTTP_STIMULUS_OUTSIDE_REDUCTION_SCOPE"}
	}
	atoms := uint64(len(stimulus.Query()) + len(stimulus.Headers()) + len(stimulus.Seeds()))
	mutableBytes := uint64(0)
	for _, entry := range stimulus.Query() {
		if entry.Present() {
			atoms++
		}
		value, _ := entry.Value()
		mutableBytes += uint64(len(value))
	}
	for _, header := range stimulus.Headers() {
		mutableBytes += uint64(len(header.Value()))
	}
	body := stimulus.Body()
	if body.Present() {
		atoms++
		mutableBytes += uint64(len(body.Bytes()))
	}
	for _, seed := range stimulus.Seeds() {
		mutableBytes += uint64(len(seed.Contents()))
	}
	return reduce.NewMeasure(policy.measureDef, []uint64{atoms, mutableBytes})
}

type HTTPReductionTransform string

const (
	HTTPTransformAbsent     HTTPReductionTransform = "absent"
	HTTPTransformRemove     HTTPReductionTransform = "remove"
	HTTPTransformFlag       HTTPReductionTransform = "flag"
	HTTPTransformEmpty      HTTPReductionTransform = "empty"
	HTTPTransformPrefixHalf HTTPReductionTransform = "prefix-half"
	HTTPTransformSuffixHalf HTTPReductionTransform = "suffix-half"
	HTTPTransformDropFirst  HTTPReductionTransform = "drop-first"
	HTTPTransformDropLast   HTTPReductionTransform = "drop-last"
)

type HTTPReplayRecipe struct {
	Rule      HTTPReducerID
	Index     int
	Transform HTTPReductionTransform
}

type HTTPNeighbor struct {
	stimulus HTTPStimulus
	neighbor reduce.Neighbor
	replay   HTTPReplayRecipe
}

func (n HTTPNeighbor) Stimulus() HTTPStimulus           { return n.stimulus }
func (n HTTPNeighbor) Neighbor() reduce.Neighbor        { return n.neighbor }
func (n HTTPNeighbor) LogicalNeighbor() reduce.Neighbor { return n.neighbor }
func (n HTTPNeighbor) ReplayRecipe() HTTPReplayRecipe   { return n.replay }

func ReplayHTTPNeighbor(parent HTTPStimulus, policy HTTPReductionPolicy, recipe HTTPReplayRecipe) (HTTPStimulus, error) {
	if !policy.matches(parent) || !policy.enabledRule(recipe.Rule) {
		return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
	}
	config := HTTPStimulusConfig{
		Method: parent.Method(), Path: parent.Path(), Query: parent.Query(), Headers: parent.Headers(),
		Body: parent.Body(), Seeds: parent.Seeds(),
	}
	switch recipe.Rule {
	case HTTPBodyAbsent:
		if recipe.Index != -1 || recipe.Transform != HTTPTransformAbsent || !config.Body.Present() {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		config.Body = AbsentBody()
	case HTTPSeedRemove:
		if recipe.Transform != HTTPTransformRemove || !httpValidIndex(recipe.Index, len(config.Seeds)) || policy.seedPinned(config.Seeds[recipe.Index].Path()) {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		config.Seeds = httpRemoveAt(config.Seeds, recipe.Index)
	case HTTPQueryRemove:
		if recipe.Transform != HTTPTransformRemove || !httpValidIndex(recipe.Index, len(config.Query)) {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		config.Query = httpRemoveAt(config.Query, recipe.Index)
	case HTTPHeaderRemove:
		if recipe.Transform != HTTPTransformRemove || !httpValidIndex(recipe.Index, len(config.Headers)) {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		config.Headers = httpRemoveAt(config.Headers, recipe.Index)
	case HTTPQueryFlag:
		if recipe.Transform != HTTPTransformFlag || !httpValidIndex(recipe.Index, len(config.Query)) || !config.Query[recipe.Index].Present() {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		entry, err := QueryFlag(config.Query[recipe.Index].Name())
		if err != nil {
			return HTTPStimulus{}, err
		}
		config.Query[recipe.Index] = entry
	case HTTPBodyShrink:
		if recipe.Index != -1 || !config.Body.Present() {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		value, err := shrinkHTTPBytes(config.Body.Bytes(), recipe.Transform)
		if err != nil {
			return HTTPStimulus{}, err
		}
		config.Body, err = PresentBody(value)
		if err != nil {
			return HTTPStimulus{}, err
		}
	case HTTPSeedShrink:
		if !httpValidIndex(recipe.Index, len(config.Seeds)) || policy.seedPinned(config.Seeds[recipe.Index].Path()) {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		seed := config.Seeds[recipe.Index]
		value, err := shrinkHTTPBytes(seed.Contents(), recipe.Transform)
		if err != nil {
			return HTTPStimulus{}, err
		}
		config.Seeds[recipe.Index], err = NewSeedFile(seed.Path(), value, seed.Mode())
		if err != nil {
			return HTTPStimulus{}, err
		}
	case HTTPQueryShrink:
		if !httpValidIndex(recipe.Index, len(config.Query)) || !config.Query[recipe.Index].Present() {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		entry := config.Query[recipe.Index]
		oldValue, _ := entry.Value()
		value, err := shrinkHTTPText(oldValue, recipe.Transform)
		if err != nil {
			return HTTPStimulus{}, err
		}
		config.Query[recipe.Index], err = QueryValue(entry.Name(), value)
		if err != nil {
			return HTTPStimulus{}, err
		}
	case HTTPHeaderShrink:
		if !httpValidIndex(recipe.Index, len(config.Headers)) {
			return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
		}
		header := config.Headers[recipe.Index]
		value, err := shrinkHTTPText(header.Value(), recipe.Transform)
		if err != nil {
			return HTTPStimulus{}, err
		}
		config.Headers[recipe.Index], err = NewRequestHeader(header.Name(), value)
		if err != nil {
			return HTTPStimulus{}, err
		}
	default:
		return HTTPStimulus{}, &domain.Error{Code: "HTTP_INVALID_REDUCTION_REPLAY"}
	}
	return NewHTTPStimulus(config)
}

type httpNeighborCandidate struct {
	child     HTTPStimulus
	recipe    HTTPReplayRecipe
	rule      reduce.ReducerRule
	locus     string
	priority  int
	transform int
	measure   reduce.Measure
}

func EnumerateHTTPNeighbors(current HTTPStimulus, policy HTTPReductionPolicy) ([]HTTPNeighbor, error) {
	currentMeasure, err := MeasureHTTPStimulus(current, policy)
	if err != nil {
		return nil, err
	}
	var candidates []httpNeighborCandidate
	for priority, ruleID := range policy.enabled {
		rule, err := policy.rule(ruleID)
		if err != nil {
			return nil, err
		}
		for _, recipe := range httpRecipes(current, policy, ruleID) {
			child, replayErr := ReplayHTTPNeighbor(current, policy, recipe)
			if replayErr != nil {
				continue
			}
			measure, measureErr := MeasureHTTPStimulus(child, policy)
			if measureErr != nil {
				return nil, measureErr
			}
			if comparison, compareErr := measure.Compare(currentMeasure); compareErr != nil || comparison >= 0 {
				return nil, &domain.Error{Code: "HTTP_NONDECREASING_TYPED_NEIGHBOR"}
			}
			candidates = append(candidates, httpNeighborCandidate{
				child: child, recipe: recipe, rule: rule, locus: httpRecipeLocus(recipe),
				priority: priority, transform: httpTransformPriority(recipe.Transform), measure: measure,
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
	result := make([]HTTPNeighbor, 0, len(candidates))
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
		replayed, err := ReplayHTTPNeighbor(current, policy, candidate.recipe)
		if err != nil || replayed.Digest() != candidate.child.Digest() {
			return nil, &domain.Error{Code: "HTTP_REDUCTION_REPLAY_MISMATCH"}
		}
		result = append(result, HTTPNeighbor{stimulus: candidate.child, neighbor: logical, replay: candidate.recipe})
	}
	return result, nil
}

func httpRecipes(current HTTPStimulus, policy HTTPReductionPolicy, rule HTTPReducerID) []HTTPReplayRecipe {
	var result []HTTPReplayRecipe
	switch rule {
	case HTTPBodyAbsent:
		if current.Body().Present() {
			result = append(result, HTTPReplayRecipe{rule, -1, HTTPTransformAbsent})
		}
	case HTTPSeedRemove:
		seeds := current.Seeds()
		for index := len(seeds) - 1; index >= 0; index-- {
			if !policy.seedPinned(seeds[index].Path()) {
				result = append(result, HTTPReplayRecipe{rule, index, HTTPTransformRemove})
			}
		}
	case HTTPQueryRemove:
		for index := len(current.Query()) - 1; index >= 0; index-- {
			result = append(result, HTTPReplayRecipe{rule, index, HTTPTransformRemove})
		}
	case HTTPHeaderRemove:
		for index := len(current.Headers()) - 1; index >= 0; index-- {
			result = append(result, HTTPReplayRecipe{rule, index, HTTPTransformRemove})
		}
	case HTTPQueryFlag:
		query := current.Query()
		for index := len(query) - 1; index >= 0; index-- {
			if query[index].Present() {
				result = append(result, HTTPReplayRecipe{rule, index, HTTPTransformFlag})
			}
		}
	case HTTPBodyShrink:
		if current.Body().Present() {
			for _, transform := range availableHTTPByteTransforms(current.Body().Bytes()) {
				result = append(result, HTTPReplayRecipe{rule, -1, transform})
			}
		}
	case HTTPSeedShrink:
		seeds := current.Seeds()
		for index := len(seeds) - 1; index >= 0; index-- {
			if policy.seedPinned(seeds[index].Path()) {
				continue
			}
			for _, transform := range availableHTTPByteTransforms(seeds[index].Contents()) {
				result = append(result, HTTPReplayRecipe{rule, index, transform})
			}
		}
	case HTTPQueryShrink:
		query := current.Query()
		for index := len(query) - 1; index >= 0; index-- {
			if !query[index].Present() {
				continue
			}
			value, _ := query[index].Value()
			for _, transform := range availableHTTPTextTransforms(value) {
				result = append(result, HTTPReplayRecipe{rule, index, transform})
			}
		}
	case HTTPHeaderShrink:
		headers := current.Headers()
		for index := len(headers) - 1; index >= 0; index-- {
			for _, transform := range availableHTTPTextTransforms(headers[index].Value()) {
				result = append(result, HTTPReplayRecipe{rule, index, transform})
			}
		}
	}
	return result
}

func httpRecipeLocus(recipe HTTPReplayRecipe) string {
	switch recipe.Rule {
	case HTTPBodyAbsent, HTTPBodyShrink:
		return "body"
	case HTTPSeedRemove, HTTPSeedShrink:
		return "seed[" + httpDecimal(recipe.Index) + "]"
	case HTTPQueryRemove, HTTPQueryFlag, HTTPQueryShrink:
		return "query[" + httpDecimal(recipe.Index) + "]"
	default:
		return "header[" + httpDecimal(recipe.Index) + "]"
	}
}

func httpTransformPriority(value HTTPReductionTransform) int {
	switch value {
	case HTTPTransformAbsent:
		return 0
	case HTTPTransformRemove:
		return 1
	case HTTPTransformFlag:
		return 2
	case HTTPTransformEmpty:
		return 3
	case HTTPTransformPrefixHalf:
		return 4
	case HTTPTransformSuffixHalf:
		return 5
	case HTTPTransformDropFirst:
		return 6
	case HTTPTransformDropLast:
		return 7
	default:
		return 99
	}
}

func availableHTTPByteTransforms(input []byte) []HTTPReductionTransform {
	seen := map[string]struct{}{}
	result := make([]HTTPReductionTransform, 0, 5)
	for _, transform := range []HTTPReductionTransform{HTTPTransformEmpty, HTTPTransformPrefixHalf, HTTPTransformSuffixHalf, HTTPTransformDropFirst, HTTPTransformDropLast} {
		value, err := shrinkHTTPBytes(input, transform)
		if err != nil || len(value) >= len(input) {
			continue
		}
		if _, duplicate := seen[string(value)]; duplicate {
			continue
		}
		seen[string(value)] = struct{}{}
		result = append(result, transform)
	}
	return result
}

func availableHTTPTextTransforms(input string) []HTTPReductionTransform {
	seen := map[string]struct{}{}
	result := make([]HTTPReductionTransform, 0, 5)
	for _, transform := range []HTTPReductionTransform{HTTPTransformEmpty, HTTPTransformPrefixHalf, HTTPTransformSuffixHalf, HTTPTransformDropFirst, HTTPTransformDropLast} {
		value, err := shrinkHTTPText(input, transform)
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

func shrinkHTTPBytes(input []byte, transform HTTPReductionTransform) ([]byte, error) {
	if len(input) == 0 {
		return nil, &domain.Error{Code: "HTTP_EMPTY_SHRINK_INPUT"}
	}
	var value []byte
	switch transform {
	case HTTPTransformEmpty:
		value = []byte{}
	case HTTPTransformPrefixHalf:
		value = input[:len(input)/2]
	case HTTPTransformSuffixHalf:
		value = input[len(input)-len(input)/2:]
	case HTTPTransformDropFirst:
		value = input[1:]
	case HTTPTransformDropLast:
		value = input[:len(input)-1]
	default:
		return nil, &domain.Error{Code: "HTTP_INVALID_SHRINK_TRANSFORM"}
	}
	return append([]byte(nil), value...), nil
}

func shrinkHTTPText(input string, transform HTTPReductionTransform) (string, error) {
	if !utf8.ValidString(input) || input == "" {
		return "", &domain.Error{Code: "HTTP_EMPTY_SHRINK_INPUT"}
	}
	runes := []rune(input)
	var value []rune
	switch transform {
	case HTTPTransformEmpty:
		value = []rune{}
	case HTTPTransformPrefixHalf:
		value = runes[:len(runes)/2]
	case HTTPTransformSuffixHalf:
		value = runes[len(runes)-len(runes)/2:]
	case HTTPTransformDropFirst:
		value = runes[1:]
	case HTTPTransformDropLast:
		value = runes[:len(runes)-1]
	default:
		return "", &domain.Error{Code: "HTTP_INVALID_SHRINK_TRANSFORM"}
	}
	return string(value), nil
}

func httpValidIndex(index, length int) bool { return index >= 0 && index < length }

func httpRemoveAt[T any](input []T, index int) []T {
	result := make([]T, 0, len(input)-1)
	result = append(result, input[:index]...)
	return append(result, input[index+1:]...)
}

func httpDecimal(value int) string {
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
