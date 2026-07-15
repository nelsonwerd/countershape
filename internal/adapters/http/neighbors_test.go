package http

import (
	"bytes"
	"fmt"
	"slices"
	"testing"
)

func httpNeighborFixture(t *testing.T) HTTPStimulus {
	t.Helper()
	query := make([]HTTPQueryEntry, 0, 5)
	for _, pair := range [][2]string{{"actor", "tenant-a"}, {"tag", "first"}, {"tag", "second"}, {"empty", ""}} {
		entry, err := QueryValue(pair[0], pair[1])
		if err != nil {
			t.Fatal(err)
		}
		query = append(query, entry)
	}
	flag, err := QueryFlag("audit")
	if err != nil {
		t.Fatal(err)
	}
	query = append(query, flag)
	headers := make([]HTTPRequestHeader, 0, 3)
	for _, pair := range [][2]string{{"x-tag", "first"}, {"x-tag", "second"}, {"x-empty", ""}} {
		header, headerErr := NewRequestHeader(pair[0], pair[1])
		if headerErr != nil {
			t.Fatal(headerErr)
		}
		headers = append(headers, header)
	}
	body, err := PresentBody([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	seed, err := NewSeedFile("invoice-seed.json", []byte(`{"invoice_id":"inv-204"}`), SeedMode0644)
	if err != nil {
		t.Fatal(err)
	}
	noise, err := NewSeedFile("z-noise.txt", []byte("noise"), SeedMode0644)
	if err != nil {
		t.Fatal(err)
	}
	stimulus, err := NewHTTPStimulus(HTTPStimulusConfig{
		Method: MethodGET, Path: "/v1/invoices/inv-204", Query: query, Headers: headers,
		Body: body, Seeds: []HTTPSeedFile{seed, noise},
	})
	if err != nil {
		t.Fatal(err)
	}
	return stimulus
}

func httpNeighborPolicy(t *testing.T, anchor HTTPStimulus) HTTPReductionPolicy {
	t.Helper()
	policy, err := NewHTTPReductionPolicy(HTTPReductionPolicyConfig{
		Anchor: anchor, PinnedSeedPaths: []string{"invoice-seed.json"}, EnabledRules: DefaultHTTPReducers(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestHTTPNeighborsAreDeterministicUniqueReplayableAndStrictlySmaller(t *testing.T) {
	stimulus := httpNeighborFixture(t)
	policy := httpNeighborPolicy(t, stimulus)
	first, err := EnumerateHTTPNeighbors(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EnumerateHTTPNeighbors(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 || len(first) != len(second) {
		t.Fatalf("neighbor counts = %d/%d", len(first), len(second))
	}
	before, err := MeasureHTTPStimulus(stimulus, policy)
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
		replayed, replayErr := ReplayHTTPNeighbor(stimulus, policy, left.ReplayRecipe())
		if replayErr != nil || replayed.Digest() != left.Stimulus().Digest() {
			t.Fatalf("neighbor %d replay mismatch: %v", index, replayErr)
		}
		if left.Stimulus().Method() != stimulus.Method() || left.Stimulus().Path() != stimulus.Path() {
			t.Fatalf("neighbor %d mutated fixed method/path", index)
		}
		assertPinnedHTTPSeed(t, left.Stimulus(), "invoice-seed.json", []byte(`{"invoice_id":"inv-204"}`))
	}
}

func TestHTTPGeneratedNeighborsRespectProviderInvariants(t *testing.T) {
	testCases := []struct {
		name         string
		method       HTTPMethod
		path         string
		body         []byte
		middleValue  string
		seedContents [][]byte
	}{
		{
			name: "present-empty-and-duplicate-multimaps", method: MethodGET, path: "/generated/empty",
			body: []byte{}, middleValue: "", seedContents: [][]byte{[]byte("noise")},
		},
		{
			name: "single-byte-and-duplicate-payloads", method: MethodPOST, path: "/generated/one",
			body: []byte("x"), middleValue: "repeat", seedContents: [][]byte{[]byte("same"), []byte("same")},
		},
		{
			name: "multi-byte-payloads", method: MethodPATCH, path: "/generated/many",
			body: []byte{0, 1, 2, 3, 4, 5, 6}, middleValue: "abcdefgh",
			seedContents: [][]byte{[]byte{9, 8, 7, 6}, []byte("abcdefgh"), []byte{}},
		},
	}

	for caseIndex, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			query := make([]HTTPQueryEntry, 0, 5)
			for _, item := range []struct {
				name  string
				value string
			}{
				{name: "tag", value: "repeat"},
				{name: "middle", value: testCase.middleValue},
				{name: "tag", value: "repeat"},
				{name: "empty", value: ""},
			} {
				entry, err := QueryValue(item.name, item.value)
				if err != nil {
					t.Fatal(err)
				}
				query = append(query, entry)
			}
			flag, err := QueryFlag("audit")
			if err != nil {
				t.Fatal(err)
			}
			query = append(query, flag)

			headers := make([]HTTPRequestHeader, 0, 3)
			for _, item := range []struct {
				name  string
				value string
			}{
				{name: "x-order", value: "repeat"},
				{name: "x-middle", value: testCase.middleValue},
				{name: "x-order", value: "repeat"},
			} {
				header, headerErr := NewRequestHeader(item.name, item.value)
				if headerErr != nil {
					t.Fatal(headerErr)
				}
				headers = append(headers, header)
			}
			body, err := PresentBody(testCase.body)
			if err != nil {
				t.Fatal(err)
			}
			pinnedContents := []byte(fmt.Sprintf(`{"case":%d}`, caseIndex))
			pinned, err := NewSeedFile("a-pinned.json", pinnedContents, SeedMode0644)
			if err != nil {
				t.Fatal(err)
			}
			seeds := []HTTPSeedFile{pinned}
			for index, contents := range testCase.seedContents {
				seed, seedErr := NewSeedFile(fmt.Sprintf("b-noise-%02d.txt", index), contents, SeedMode0644)
				if seedErr != nil {
					t.Fatal(seedErr)
				}
				seeds = append(seeds, seed)
			}
			stimulus, err := NewHTTPStimulus(HTTPStimulusConfig{
				Method: testCase.method, Path: testCase.path, Query: query, Headers: headers, Body: body, Seeds: seeds,
			})
			if err != nil {
				t.Fatal(err)
			}
			policy, err := NewHTTPReductionPolicy(HTTPReductionPolicyConfig{
				Anchor: stimulus, PinnedSeedPaths: []string{"a-pinned.json"}, EnabledRules: DefaultHTTPReducers(),
			})
			if err != nil {
				t.Fatal(err)
			}

			first, err := EnumerateHTTPNeighbors(stimulus, policy)
			if err != nil {
				t.Fatal(err)
			}
			second, err := EnumerateHTTPNeighbors(stimulus, policy)
			if err != nil {
				t.Fatal(err)
			}
			// Body/seeds/headers can emit at most absence-or-removal plus five
			// shrink transforms. A query member can additionally become a flag.
			maximumNeighbors := 6 + 6*len(stimulus.Seeds()) + 7*len(stimulus.Query()) + 6*len(stimulus.Headers())
			if len(first) == 0 || len(first) != len(second) || len(first) > maximumNeighbors {
				t.Fatalf("neighbor count %d/%d is outside finite bound %d", len(first), len(second), maximumNeighbors)
			}
			before, err := MeasureHTTPStimulus(stimulus, policy)
			if err != nil {
				t.Fatal(err)
			}
			proposalDigests := make(map[string]struct{}, len(first))
			stimulusDigests := make(map[string]struct{}, len(first))
			queryDuplicateSurvivorsObserved := false
			headerDuplicateSurvivorsObserved := false
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
				replayed, replayErr := ReplayHTTPNeighbor(stimulus, policy, left.ReplayRecipe())
				if replayErr != nil || replayed.Digest() != left.Stimulus().Digest() ||
					!bytes.Equal(replayed.CanonicalBytes(), left.Stimulus().CanonicalBytes()) ||
					!bytes.Equal(replayed.ExecutionPayloadCanonicalBytes(), left.Stimulus().ExecutionPayloadCanonicalBytes()) {
					t.Fatalf("neighbor %d replay did not reconstruct the exact proposed stimulus: %v", index, replayErr)
				}
				if left.Stimulus().Method() != stimulus.Method() || left.Stimulus().Path() != stimulus.Path() {
					t.Fatalf("neighbor %d changed a fixed HTTP scope field", index)
				}
				assertPinnedHTTPSeed(t, left.Stimulus(), "a-pinned.json", pinnedContents)

				recipe := left.ReplayRecipe()
				if recipe.Rule == HTTPQueryRemove && recipe.Index == 1 {
					survivors := left.Stimulus().Query()
					if len(survivors) >= 2 {
						firstValue, firstPresent := survivors[0].Value()
						secondValue, secondPresent := survivors[1].Value()
						if survivors[0].Name() == "tag" && survivors[1].Name() == "tag" &&
							firstPresent && secondPresent && firstValue == "repeat" && secondValue == "repeat" {
							queryDuplicateSurvivorsObserved = true
						}
					}
				}
				if recipe.Rule == HTTPHeaderRemove && recipe.Index == 1 {
					survivors := left.Stimulus().Headers()
					if len(survivors) >= 2 && survivors[0].Name() == "x-order" && survivors[1].Name() == "x-order" &&
						survivors[0].Value() == "repeat" && survivors[1].Value() == "repeat" {
						headerDuplicateSurvivorsObserved = true
					}
				}
			}
			if !queryDuplicateSurvivorsObserved || !headerDuplicateSurvivorsObserved {
				t.Fatalf("ordered duplicate survivors were not preserved: query=%t header=%t", queryDuplicateSurvivorsObserved, headerDuplicateSurvivorsObserved)
			}
		})
	}
}

func TestHTTPNeighborsKeepFlagEmptyAbsentAndOmissionDistinct(t *testing.T) {
	stimulus := httpNeighborFixture(t)
	policy := httpNeighborPolicy(t, stimulus)
	neighbors, err := EnumerateHTTPNeighbors(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	var absentBody HTTPStimulus
	var emptyAsFlag HTTPStimulus
	var emptyRemoved HTTPStimulus
	var emptyHeaderRemoved HTTPStimulus
	for _, neighbor := range neighbors {
		recipe := neighbor.ReplayRecipe()
		switch recipe.Rule {
		case HTTPBodyAbsent:
			absentBody = neighbor.Stimulus()
		case HTTPQueryFlag:
			if recipe.Index == 3 {
				emptyAsFlag = neighbor.Stimulus()
			}
		case HTTPQueryRemove:
			if recipe.Index == 3 {
				emptyRemoved = neighbor.Stimulus()
			}
		case HTTPHeaderRemove:
			if recipe.Index == 2 {
				emptyHeaderRemoved = neighbor.Stimulus()
			}
		}
	}
	if !absentBody.Valid() || stimulus.Body().Presence() != PresencePresent || absentBody.Body().Presence() != PresenceAbsent {
		t.Fatal("present-empty body did not produce distinct absence")
	}
	if !emptyAsFlag.Valid() || !emptyRemoved.Valid() || emptyAsFlag.Digest() == emptyRemoved.Digest() {
		t.Fatal("query present-empty collapsed into flag or omission")
	}
	if emptyAsFlag.Query()[3].Presence() != PresenceAbsent || len(emptyRemoved.Query()) != len(stimulus.Query())-1 {
		t.Fatal("query flag/omission tags were not retained")
	}
	if !emptyHeaderRemoved.Valid() || len(emptyHeaderRemoved.Headers()) != len(stimulus.Headers())-1 {
		t.Fatal("present-empty header did not remain distinct from omission")
	}
}

func TestHTTPOrderedMultimapRemovalPreservesSurvivorOrderAndDuplicates(t *testing.T) {
	stimulus := httpNeighborFixture(t)
	policy := httpNeighborPolicy(t, stimulus)
	neighbors, err := EnumerateHTTPNeighbors(stimulus, policy)
	if err != nil {
		t.Fatal(err)
	}
	queryFound := false
	headerFound := false
	for _, neighbor := range neighbors {
		recipe := neighbor.ReplayRecipe()
		if recipe.Rule == HTTPQueryRemove && recipe.Index == 1 {
			query := neighbor.Stimulus().Query()
			if len(query) != 4 || query[0].Name() != "actor" || query[1].Name() != "tag" {
				t.Fatalf("ordered query survivors changed: %#v", query)
			}
			value, _ := query[1].Value()
			if value != "second" {
				t.Fatalf("duplicate query order changed: %q", value)
			}
			queryFound = true
		}
		if recipe.Rule == HTTPHeaderRemove && recipe.Index == 0 {
			headers := neighbor.Stimulus().Headers()
			if len(headers) != 2 || headers[0].Name() != "x-tag" || headers[0].Value() != "second" || headers[1].Name() != "x-empty" {
				t.Fatalf("ordered header survivors changed: %#v", headers)
			}
			headerFound = true
		}
	}
	if !queryFound || !headerFound {
		t.Fatalf("missing order cases query=%t header=%t", queryFound, headerFound)
	}
}

func TestHTTPReductionPolicyCanonicalizesRuleOrderAndRejectsInvalidScope(t *testing.T) {
	stimulus := httpNeighborFixture(t)
	forward := DefaultHTTPReducers()
	reversed := append([]HTTPReducerID(nil), forward...)
	slices.Reverse(reversed)
	left, err := NewHTTPReductionPolicy(HTTPReductionPolicyConfig{
		Anchor: stimulus, PinnedSeedPaths: []string{"invoice-seed.json"}, EnabledRules: forward,
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewHTTPReductionPolicy(HTTPReductionPolicyConfig{
		Anchor: stimulus, PinnedSeedPaths: []string{"invoice-seed.json"}, EnabledRules: reversed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if left.Digest() != right.Digest() || !bytes.Equal(left.CanonicalBytes(), right.CanonicalBytes()) {
		t.Fatal("enabled-rule input permutation changed canonical policy")
	}
	if _, err := NewHTTPReductionPolicy(HTTPReductionPolicyConfig{
		Anchor: stimulus, PinnedSeedPaths: []string{"missing.json"}, EnabledRules: forward,
	}); err == nil {
		t.Fatal("missing pinned seed entered a policy")
	}
	if _, err := NewHTTPReductionPolicy(HTTPReductionPolicyConfig{
		Anchor: stimulus, PinnedSeedPaths: []string{"invoice-seed.json"}, EnabledRules: []HTTPReducerID{HTTPQueryRemove, HTTPQueryRemove},
	}); err == nil {
		t.Fatal("duplicate rule entered a policy")
	}
	if _, err := NewHTTPReductionPolicy(HTTPReductionPolicyConfig{
		Anchor: stimulus, PinnedSeedPaths: []string{"invoice-seed.json"}, EnabledRules: []HTTPReducerID{"http.unknown"},
	}); err == nil {
		t.Fatal("unknown rule entered a policy")
	}
}

func assertPinnedHTTPSeed(t *testing.T, stimulus HTTPStimulus, path string, contents []byte) {
	t.Helper()
	for _, seed := range stimulus.Seeds() {
		if seed.Path() == path {
			if seed.Mode() != SeedMode0644 || !bytes.Equal(seed.Contents(), contents) {
				t.Fatalf("pinned seed %s mutated", path)
			}
			return
		}
	}
	t.Fatalf("pinned seed %s was removed", path)
}
