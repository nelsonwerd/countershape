package choice

import (
	"bytes"
	"encoding/base64"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

func TestPortableConfirmedSetPreservesProfileOrderAndExactValues(t *testing.T) {
	confirmed := portableCLIConfirmed(t,
		portableCLIProjection(t, []byte{0x00, 0xff}, "same-context"),
		portableCLIProjection(t, []byte{0xff, 0x00}, "same-context"),
	)
	definitions := confirmed.Registry().Definitions()
	if got, want := fieldIDs(definitions), []string{string(cli.CLIFieldStdoutBytes), string(cli.CLIFieldStderrText)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("portable registry order = %#v, want adapter order %#v", got, want)
	}
	if definitions[0].Type != FieldBytes || definitions[1].Type != FieldString ||
		confirmed.registry.mode != fieldRegistryPortable || !confirmed.registry.profileDigest.Valid() ||
		confirmed.registry.profileDigest == confirmed.registry.projectionDefinitionDigest {
		t.Fatal("portable registry lost its value types, private mode, or distinct profile identity")
	}
	for _, outcome := range confirmed.ordered {
		if got := outcome.tuple.Fields[0].Value.Bytes(); len(got) != 2 {
			t.Fatalf("translated byte payload length = %d, want 2", len(got))
		}
		if outcome.tuple.Fields[1].Value.Tag() != ValueString {
			t.Fatalf("translated stderr tag = %s, want STRING", outcome.tuple.Fields[1].Value.Tag())
		}
	}
	groups, err := buildBlindGroups(testDomainDigest(t, "portable-profile-order"), confirmed)
	if err != nil || len(groups) != 2 {
		t.Fatalf("portable blind groups = %d, %v", len(groups), err)
	}
	differing, err := differingFields(confirmed.registry, groups)
	if err != nil || !reflect.DeepEqual(differing, []string{string(cli.CLIFieldStdoutBytes)}) {
		t.Fatalf("portable differing fields = %#v, %v", differing, err)
	}
}

func TestPortableSelectedFieldsCanonicalizeInProfileOrderNotLexicalOrder(t *testing.T) {
	confirmed := portableCLIConfirmed(t,
		portableCLIProjection(t, []byte{0x00}, "alpha"),
		portableCLIProjection(t, []byte{0x01}, "beta"),
	)
	selected, err := confirmed.registry.resolveSelected([]string{
		string(cli.CLIFieldStderrText), string(cli.CLIFieldStdoutBytes),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(selected))
	for index, field := range selected {
		got[index] = field.String()
	}
	want := []string{string(cli.CLIFieldStdoutBytes), string(cli.CLIFieldStderrText)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("selected portable field order = %#v, want profile order %#v", got, want)
	}
}

func TestPortableConfirmedOutcomesRemainCanonicalIDOrdered(t *testing.T) {
	projections := make([][]byte, 4)
	for index := range projections {
		projections[index] = portableCLIProjection(t, []byte{byte(index)}, "same-context")
	}
	confirmed := portableCLIConfirmed(t, projections...)
	outcomes := confirmed.Outcomes()
	for index := 1; index < len(outcomes); index++ {
		if outcomes[index].ID().String() <= outcomes[index-1].ID().String() {
			t.Fatalf("confirmed outcome IDs are not canonical: %q then %q", outcomes[index-1].ID().String(), outcomes[index].ID().String())
		}
	}
	byCandidate := append([]ConfirmedOutcomeRef(nil), outcomes...)
	sort.Slice(byCandidate, func(i, j int) bool {
		return byCandidate[i].CandidateExecutionKey().String() < byCandidate[j].CandidateExecutionKey().String()
	})
	sameOrder := true
	for index := range outcomes {
		if outcomes[index].ID() != byCandidate[index].ID() {
			sameOrder = false
			break
		}
	}
	if sameOrder {
		t.Fatal("test precondition: candidate-key hashes happened to match canonical outcome-ID order")
	}
}

func TestPortableFourSlotWireRoundTripsOpaqueBytesAndOrderedDuplicatesInRegistryOrder(t *testing.T) {
	registry, err := newFieldRegistry([]FieldDefinition{
		{ID: "z.bytes", Type: FieldBytes},
		{ID: "a.list", Type: FieldOrderedStringList},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	registry.mode = fieldRegistryPortable
	bytesValue, err := BytesValue([]byte{0x00, 0xff, 0x80})
	if err != nil {
		t.Fatal(err)
	}
	listValue, err := OrderedStringListValue([]string{"text/plain", "", "text/plain"})
	if err != nil {
		t.Fatal(err)
	}
	tuple := CompleteTuple{Fields: []FieldValue{
		{FieldID: "z.bytes", Value: bytesValue},
		{FieldID: "a.list", Value: listValue},
	}}
	wire, err := tupleToWire(registry, tuple)
	if err != nil {
		t.Fatal(err)
	}
	if wire.Fields[0].FieldID != "z.bytes" || wire.Fields[0].Value.Text != "AP+A" ||
		wire.Fields[0].Value.CanonicalJSONBase64 != "" || wire.Fields[1].FieldID != "a.list" ||
		wire.Fields[1].Value.Text != "" || wire.Fields[1].Value.CanonicalJSONBase64 == "" {
		t.Fatalf("portable compatibility wire slots/order differ: %#v", wire)
	}
	rebuilt, err := tupleFromWire(registry, wire)
	if err != nil || tupleIdentityKey(registry, rebuilt.Fields) != tupleIdentityKey(registry, tuple.Fields) ||
		!bytes.Equal(rebuilt.Fields[0].Value.Bytes(), []byte{0x00, 0xff, 0x80}) {
		t.Fatalf("portable compatibility round trip changed exact identity: %v", err)
	}
	members, ok := rebuilt.Fields[1].Value.OrderedStrings()
	if !ok || !reflect.DeepEqual(members, []string{"text/plain", "", "text/plain"}) {
		t.Fatalf("ordered duplicate list changed: %#v", members)
	}
	reordered := wire
	reordered.Fields = append([]fieldValueWire(nil), wire.Fields...)
	reordered.Fields[0], reordered.Fields[1] = reordered.Fields[1], reordered.Fields[0]
	if _, err := tupleFromWire(registry, reordered); !IsRefusal(err, CodeDuplicateTupleField) {
		t.Fatalf("lexically reordered portable wire error = %v", err)
	}
}

func TestPortableExactValueWireRejectsHostileBytesAndOrderedLists(t *testing.T) {
	for name, wire := range map[string]exactValueWire{
		"empty-bytes":       {Tag: string(ValueBytes), Text: ""},
		"opaque-bytes":      {Tag: string(ValueBytes), Text: base64.StdEncoding.EncodeToString([]byte{0x00, 0xff, 0x80})},
		"empty-list":        {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: "W10="},
		"empty-member":      {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: "WyIiXQ=="},
		"ordered-duplicate": {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: base64.StdEncoding.EncodeToString([]byte(`["z","","z"]`))},
	} {
		value, err := exactValueFromWire(wire)
		if err != nil || !value.Tag().validPortableCompatibilityTag() {
			t.Fatalf("positive %s wire failed: tag=%s err=%v", name, value.Tag(), err)
		}
	}

	tooManyMembers := make([]string, 257)
	for index := range tooManyMembers {
		tooManyMembers[index] = `""`
	}
	hostileLists := map[string]exactValueWire{
		"dirty-text":         {Tag: string(ValueOrderedStringList), Text: "x", CanonicalJSONBase64: "W10="},
		"dirty-boolean":      {Tag: string(ValueOrderedStringList), Boolean: true, CanonicalJSONBase64: "W10="},
		"empty-slot":         {Tag: string(ValueOrderedStringList)},
		"invalid-base64":     {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: "@@=="},
		"unpadded-base64":    {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: "W10"},
		"whitespace-base64":  {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: "W 10="},
		"noncanonical-json":  {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: base64.StdEncoding.EncodeToString([]byte(`[ "a" ]`))},
		"escaped-equivalent": {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: base64.StdEncoding.EncodeToString([]byte(`["\u0061"]`))},
		"object-root":        {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: "e30="},
		"null-root":          {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: "bnVsbA=="},
		"mixed-array":        {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: base64.StdEncoding.EncodeToString([]byte(`["a",1]`))},
		"invalid-utf8":       {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: base64.StdEncoding.EncodeToString([]byte{0xff})},
		"too-many-members":   {Tag: string(ValueOrderedStringList), CanonicalJSONBase64: base64.StdEncoding.EncodeToString([]byte("[" + strings.Join(tooManyMembers, ",") + "]"))},
	}
	for name, wire := range hostileLists {
		if _, err := exactValueFromWire(wire); !IsRefusal(err, CodeInvalidFieldType) {
			t.Fatalf("hostile ordered-list %s error = %v, want %s", name, err, CodeInvalidFieldType)
		}
	}

	hostileBytes := map[string]exactValueWire{
		"dirty-boolean":        {Tag: string(ValueBytes), Boolean: true},
		"dirty-canonical":      {Tag: string(ValueBytes), Text: "YQ==", CanonicalJSONBase64: "W10="},
		"invalid-alphabet":     {Tag: string(ValueBytes), Text: "@@=="},
		"whitespace":           {Tag: string(ValueBytes), Text: "Y Q=="},
		"missing-padding":      {Tag: string(ValueBytes), Text: "YQ"},
		"nonzero-padding-bits": {Tag: string(ValueBytes), Text: "YR=="},
		"extra-padding":        {Tag: string(ValueBytes), Text: "YQ==="},
		"unknown-tag":          {Tag: "OPAQUE_BYTES", Text: "YQ=="},
	}
	for name, wire := range hostileBytes {
		if _, err := exactValueFromWire(wire); !IsRefusal(err, CodeInvalidFieldType) {
			t.Fatalf("hostile byte %s error = %v, want %s", name, err, CodeInvalidFieldType)
		}
	}
	oversized := exactValueWire{
		Tag: string(ValueBytes), Text: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0xa5}, 64*1024+1)),
	}
	if _, err := exactValueFromWire(oversized); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("oversized byte wire error = %v, want %s", err, CodeInputLimitExceeded)
	}

	registry, err := newFieldRegistry([]FieldDefinition{
		{ID: "bytes", Type: FieldBytes},
		{ID: "list", Type: FieldOrderedStringList},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	registry.mode = fieldRegistryPortable
	stringValue, err := exactValueFromWire(exactValueWire{Tag: string(ValueString), Text: "not-bytes"})
	if err != nil {
		t.Fatal(err)
	}
	jsonValue, err := exactValueFromWire(exactValueWire{Tag: string(ValueCanonicalJSON), CanonicalJSONBase64: "e30="})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.validateSelectedTuple(
		[]FieldID{{text: "bytes"}, {text: "list"}},
		[]FieldValue{{FieldID: "bytes", Value: stringValue}, {FieldID: "list", Value: jsonValue}},
	)
	if !IsRefusal(err, CodeInvalidFieldType) {
		t.Fatalf("registry type substitution error = %v, want %s", err, CodeInvalidFieldType)
	}
}

func (tag ValueTag) validPortableCompatibilityTag() bool {
	return tag == ValueBytes || tag == ValueOrderedStringList
}

func TestPortableEmptyAndAbsenceIdentitiesRemainDistinct(t *testing.T) {
	emptyString, err := StringValue("")
	if err != nil {
		t.Fatal(err)
	}
	emptyBytes, err := BytesValue([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	emptyList, err := OrderedStringListValue([]string{})
	if err != nil {
		t.Fatal(err)
	}
	emptyMemberList, err := OrderedStringListValue([]string{""})
	if err != nil {
		t.Fatal(err)
	}
	values := []ExactValue{MissingValue(), NullValue(), emptyString, emptyBytes, emptyList, emptyMemberList}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := value.identityKey(fieldRegistryPortable)
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("portable empty/absence identities collided at %q", key)
		}
		seen[key] = struct{}{}
	}
	registry, err := newFieldRegistry([]FieldDefinition{
		{ID: "a.missing", Type: FieldString, AllowMissing: true},
		{ID: "b.null", Type: FieldString, AllowNull: true},
		{ID: "c.empty-string", Type: FieldString},
		{ID: "d.empty-bytes", Type: FieldBytes},
		{ID: "e.empty-list", Type: FieldOrderedStringList},
		{ID: "f.empty-member-list", Type: FieldOrderedStringList},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	registry.mode = fieldRegistryPortable
	tuple := CompleteTuple{Fields: make([]FieldValue, len(values))}
	for index, fieldID := range registry.orderedIDs {
		tuple.Fields[index] = FieldValue{FieldID: fieldID, Value: values[index]}
	}
	wire, err := tupleToWire(registry, tuple)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 4; index++ {
		if wire.Fields[index].Value.Boolean || wire.Fields[index].Value.CanonicalJSONBase64 != "" || wire.Fields[index].Value.Text != "" {
			t.Fatalf("empty/absence compatibility slot %d is dirty: %#v", index, wire.Fields[index].Value)
		}
	}
	if wire.Fields[4].Value.Text != "" || wire.Fields[4].Value.Boolean || wire.Fields[4].Value.CanonicalJSONBase64 != "W10=" ||
		wire.Fields[5].Value.Text != "" || wire.Fields[5].Value.Boolean || wire.Fields[5].Value.CanonicalJSONBase64 != "WyIiXQ==" {
		t.Fatalf("ordered-list compatibility slots differ: %#v", wire.Fields[4:])
	}
	rebuilt, err := tupleFromWire(registry, wire)
	if err != nil {
		t.Fatal(err)
	}
	for index := range values {
		if rebuilt.Fields[index].Value.identityKey(fieldRegistryPortable) != values[index].identityKey(fieldRegistryPortable) {
			t.Fatalf("portable empty/absence value %d changed on round trip", index)
		}
	}
	rawBytes := []byte{0x00, 0xff}
	defensiveBytes, err := BytesValue(rawBytes)
	if err != nil {
		t.Fatal(err)
	}
	rawBytes[0] = 0x7f
	gotBytes := defensiveBytes.Bytes()
	gotBytes[1] = 0x7f
	if !bytes.Equal(defensiveBytes.Bytes(), []byte{0x00, 0xff}) {
		t.Fatal("portable byte value exposed caller or getter storage")
	}
	rawList := []string{"first", "second"}
	defensiveList, err := OrderedStringListValue(rawList)
	if err != nil {
		t.Fatal(err)
	}
	rawList[0] = "mutated"
	gotList, _ := defensiveList.OrderedStrings()
	gotList[1] = "mutated"
	again, _ := defensiveList.OrderedStrings()
	if !reflect.DeepEqual(again, []string{"first", "second"}) {
		t.Fatal("portable ordered list exposed caller or getter storage")
	}
}

func TestSelectedTupleContainsExactlySelectedFieldsAndCannotSmuggleContext(t *testing.T) {
	confirmed := portableCLIConfirmed(t,
		portableCLIProjection(t, []byte{0x00}, "same-context"),
		portableCLIProjection(t, []byte{0x01}, "same-context"),
	)
	selected := []string{string(cli.CLIFieldStdoutBytes)}
	customBytes, err := BytesValue([]byte{0x02})
	if err != nil {
		t.Fatal(err)
	}
	expectation, err := newSelectedTuple(confirmed, selected, []FieldValue{{FieldID: selected[0], Value: customBytes}})
	if err != nil {
		t.Fatal(err)
	}
	review, err := NewCustomExpectationReview(confirmed, selected, expectation, "portable-reviewer", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	validated, err := ValidateRuling(confirmed, RulingInput{
		Action: ActionCustomExpectation, SelectedFields: selected, AllowedObserved: []ConfirmedOutcomeRef{},
		CustomExpectation: &expectation, CustomReview: review,
	})
	if err != nil {
		t.Fatal(err)
	}
	compilable := validated.CompileEligibility().(CompilableRuling)
	if allowed := compilable.AllowedTuples(); len(allowed) != 1 || len(allowed[0].Fields) != 1 || allowed[0].Fields[0].FieldID != selected[0] {
		t.Fatalf("custom selected-only predicate widened: %#v", allowed)
	}
	contextValue, _ := StringValue("smuggled")
	_, err = newSelectedTuple(confirmed, selected, []FieldValue{
		{FieldID: selected[0], Value: customBytes},
		{FieldID: string(cli.CLIFieldStderrText), Value: contextValue},
	})
	assertRefusal(t, err, CodeIncompleteTuple)
	_, err = newSelectedTuple(confirmed, selected, []FieldValue{
		{FieldID: string(cli.CLIFieldStderrText), Value: customBytes},
	})
	assertRefusal(t, err, CodeIncompleteTuple)
	_, err = newSelectedTuple(confirmed, selected, []FieldValue{})
	assertRefusal(t, err, CodeIncompleteTuple)
	observed := confirmed.ordered[0].tuple.Fields[0]
	seen, err := newSelectedTuple(confirmed, selected, []FieldValue{observed})
	if err != nil {
		t.Fatal(err)
	}
	seenReview, err := NewCustomExpectationReview(confirmed, selected, seen, "portable-reviewer", reviewDigest())
	if err != nil {
		t.Fatal(err)
	}
	_, err = ValidateRuling(confirmed, RulingInput{
		Action: ActionCustomExpectation, SelectedFields: selected, AllowedObserved: []ConfirmedOutcomeRef{},
		CustomExpectation: &seen, CustomReview: seenReview,
	})
	assertRefusal(t, err, CodeCustomExpectationAlreadySeen)
}

func TestPortableSelectedTupleEnforcesEncodedAggregateCeiling(t *testing.T) {
	definitions := make([]FieldDefinition, 4)
	raw := make([]FieldValue, 4)
	selected := make([]FieldID, 4)
	for index := range definitions {
		fieldID := "portable.string." + string(rune('a'+index))
		definitions[index] = FieldDefinition{ID: fieldID, Type: FieldString}
		value, err := StringValue(strings.Repeat("\\", 60*1024))
		if err != nil {
			t.Fatal(err)
		}
		raw[index] = FieldValue{FieldID: fieldID, Value: value}
		selected[index] = FieldID{text: fieldID}
	}
	registry, err := newFieldRegistry(definitions, true)
	if err != nil {
		t.Fatal(err)
	}
	registry.mode = fieldRegistryPortable
	if _, err := registry.validateSelectedTuple(selected, raw); !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("portable compatibility encoded ceiling error = %v, want %s", err, CodeInputLimitExceeded)
	}
}

func TestPortableSelectedTupleEnforcesCanonicalJSONValueCeiling(t *testing.T) {
	registry, err := newFieldRegistry([]FieldDefinition{{ID: "metadata", Type: FieldCanonicalJSON}}, true)
	if err != nil {
		t.Fatal(err)
	}
	registry.mode = fieldRegistryPortable
	registry.sourceKinds = map[string]string{"metadata": "CANONICAL_JSON_OBJECT"}
	value, err := CanonicalJSONBytes([]byte(`{"payload":"` + strings.Repeat("x", 65*1024) + `"}`))
	if err != nil {
		t.Fatalf("Choice legacy-sized canonical value test precondition: %v", err)
	}
	_, err = registry.validateSelectedTuple(
		[]FieldID{{text: "metadata"}},
		[]FieldValue{{FieldID: "metadata", Value: value}},
	)
	if !IsRefusal(err, CodeInputLimitExceeded) {
		t.Fatalf("portable canonical JSON ceiling error = %v, want %s", err, CodeInputLimitExceeded)
	}
}

func TestPortableCustomCanonicalJSONRetainsHTTPObjectSourceConstraint(t *testing.T) {
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := projectiontranslate.Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	expectation, err := resolved.ExpectationDomain()
	if err != nil {
		t.Fatal(err)
	}
	registry, err := newPortableFieldRegistry(resolved.Profile(), expectation)
	if err != nil {
		t.Fatal(err)
	}
	fieldID := string(counterhttp.HTTPFieldBodyMetadata)
	selected, err := registry.resolveSelected([]string{fieldID})
	if err != nil {
		t.Fatal(err)
	}
	for _, exact := range [][]byte{[]byte(`1`), []byte(`[]`)} {
		value, valueErr := CanonicalJSONBytes(exact)
		if valueErr != nil {
			t.Fatal(valueErr)
		}
		if _, validateErr := registry.validateSelectedTuple(selected, []FieldValue{{FieldID: fieldID, Value: value}}); !IsRefusal(validateErr, CodeInvalidFieldType) {
			t.Fatalf("HTTP metadata %s error = %v, want %s", exact, validateErr, CodeInvalidFieldType)
		}
	}
	object, err := CanonicalJSONBytes([]byte(`{"owner":"tenant"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.validateSelectedTuple(selected, []FieldValue{{FieldID: fieldID, Value: object}}); err != nil {
		t.Fatalf("HTTP metadata object was rejected: %v", err)
	}
}

func TestPortableCustomExpectationRequiresAdapterRealizableExtension(t *testing.T) {
	httpDefinition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	httpResolved, err := projectiontranslate.Resolve(httpDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	httpExpectation, err := httpResolved.ExpectationDomain()
	if err != nil {
		t.Fatal(err)
	}
	httpRegistry, err := newPortableFieldRegistry(httpResolved.Profile(), httpExpectation)
	if err != nil {
		t.Fatal(err)
	}
	statusID := string(counterhttp.HTTPFieldStatus)
	statusSelected, err := httpRegistry.resolveSelected([]string{statusID})
	if err != nil {
		t.Fatal(err)
	}
	status401, err := IntegerValue("401")
	if err != nil {
		t.Fatal(err)
	}
	if values, err := httpRegistry.validateSelectedTuple(statusSelected, []FieldValue{{FieldID: statusID, Value: status401}}); err != nil || len(values) != 1 {
		t.Fatalf("realizable selected-only HTTP 401 expectation = %#v, %v", values, err)
	}
	status600, err := IntegerValue("600")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := httpRegistry.validateSelectedTuple(statusSelected, []FieldValue{{FieldID: statusID, Value: status600}}); !IsRefusal(err, CodeCustomExpectationNotAdapterRealizable) {
		t.Fatalf("impossible HTTP 600 expectation error = %v, want %s", err, CodeCustomExpectationNotAdapterRealizable)
	}

	cliDefinition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{
		cli.CLIFieldCompletionKind, cli.CLIFieldExitCode, cli.CLIFieldExitSignal,
	}})
	if err != nil {
		t.Fatal(err)
	}
	cliResolved, err := projectiontranslate.Resolve(cliDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	cliExpectation, err := cliResolved.ExpectationDomain()
	if err != nil {
		t.Fatal(err)
	}
	cliRegistry, err := newPortableFieldRegistry(cliResolved.Profile(), cliExpectation)
	if err != nil {
		t.Fatal(err)
	}
	completionSelected, err := cliRegistry.resolveSelected([]string{
		string(cli.CLIFieldCompletionKind), string(cli.CLIFieldExitCode),
	})
	if err != nil {
		t.Fatal(err)
	}
	exited, err := StringValue("EXITED")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cliRegistry.validateSelectedTuple(completionSelected, []FieldValue{
		{FieldID: string(cli.CLIFieldCompletionKind), Value: exited},
		{FieldID: string(cli.CLIFieldExitCode), Value: MissingValue()},
	}); !IsRefusal(err, CodeCustomExpectationNotAdapterRealizable) {
		t.Fatalf("impossible EXITED plus missing-code expectation error = %v, want %s", err, CodeCustomExpectationNotAdapterRealizable)
	}

	stdoutDefinition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{
		cli.CLIFieldStdoutBytes, cli.CLIFieldStdoutJSONMode,
	}})
	if err != nil {
		t.Fatal(err)
	}
	stdoutResolved, err := projectiontranslate.Resolve(stdoutDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	stdoutExpectation, err := stdoutResolved.ExpectationDomain()
	if err != nil {
		t.Fatal(err)
	}
	stdoutRegistry, err := newPortableFieldRegistry(stdoutResolved.Profile(), stdoutExpectation)
	if err != nil {
		t.Fatal(err)
	}
	stdoutID := string(cli.CLIFieldStdoutBytes)
	stdoutSelected, err := stdoutRegistry.resolveSelected([]string{stdoutID})
	if err != nil {
		t.Fatal(err)
	}
	nonJSON, err := BytesValue([]byte{0xff})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stdoutRegistry.validateSelectedTuple(stdoutSelected, []FieldValue{{FieldID: stdoutID, Value: nonJSON}}); !IsRefusal(err, CodeCustomExpectationNotAdapterRealizable) {
		t.Fatalf("non-JSON stdout under an active JSON profile error = %v, want %s", err, CodeCustomExpectationNotAdapterRealizable)
	}
}

func TestPortableExpectationDomainIdentityBindsRegistryAndUniverseSeal(t *testing.T) {
	stdoutDefinition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{cli.CLIFieldStdoutBytes}})
	if err != nil {
		t.Fatal(err)
	}
	stderrDefinition, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: []cli.CLIFieldID{cli.CLIFieldStderrText}})
	if err != nil {
		t.Fatal(err)
	}
	stdoutResolved, err := projectiontranslate.Resolve(stdoutDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	stderrResolved, err := projectiontranslate.Resolve(stderrDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	stderrExpectation, err := stderrResolved.ExpectationDomain()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newPortableFieldRegistry(stdoutResolved.Profile(), stderrExpectation); !IsRefusal(err, CodeInvalidFieldRegistry) {
		t.Fatalf("cross-profile expectation-domain pairing error = %v, want %s", err, CodeInvalidFieldRegistry)
	}

	confirmed := portableCLIConfirmed(t,
		portableCLIProjection(t, []byte("left"), "same"),
		portableCLIProjection(t, []byte("right"), "same"),
	)
	original := confirmed.seal
	withoutDomain := confirmed.registry.clone()
	withoutDomain.expectationDomain = projectiontranslate.ExpectationDomain{}
	drifted, err := makeConfirmedOutcomeSetSeal(withoutDomain, confirmed.outcomeMapDigest, confirmed.preservationDigest, confirmed.ordered)
	if err != nil {
		t.Fatal(err)
	}
	if original == drifted || !confirmed.registry.expectationDomain.Valid() {
		t.Fatal("portable expectation-domain identity was omitted from the confirmed-universe seal")
	}
}

func TestTranslateConfirmedAdmitsProofBytesBeforeDefensiveCopy(t *testing.T) {
	binding := p07CLIProjectionBinding(t, cli.CLIFieldStdoutBytes)
	leftProjection := portableCLIBytesProjection(t, []byte("left"))
	rightProjection := portableCLIBytesProjection(t, []byte("right"))
	inputs := []ProjectionProofInput{
		trustedBytesInputForProjection(t, binding.Digest(), "candidate:proof-copy-left", leftProjection),
		trustedBytesInputForProjection(t, binding.Digest(), "candidate:proof-copy-right", rightProjection),
	}
	proofs := []projectiontranslate.ProjectionProof{
		{CandidateExecutionKey: inputs[0].CandidateExecutionKey, CanonicalProjection: bytes.Repeat([]byte{'x'}, 600*1024+1)},
		{CandidateExecutionKey: inputs[1].CandidateExecutionKey, CanonicalProjection: rightProjection},
	}
	_, err := projectiontranslate.TranslateConfirmed(binding, testProjectionRoster(t, inputs), proofs)
	if !projectiontranslate.IsCode(err, projectiontranslate.CodeTranslationLimit) {
		t.Fatalf("oversized proof copy admission error = %v, want %s", err, projectiontranslate.CodeTranslationLimit)
	}
}

func portableCLIBytesProjection(t testing.TB, stdout []byte) []byte {
	t.Helper()
	exact, err := canon.CanonicalizeTyped(map[string]any{
		"fields": []map[string]any{{
			"field_id": string(cli.CLIFieldStdoutBytes),
			"value":    map[string]any{"base64": base64.StdEncoding.EncodeToString(stdout), "tag": "BYTES"},
		}},
		"kind": "CLIProjection", "schema_version": "cli-projection/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return exact
}

func portableCLIConfirmed(t testing.TB, projections ...[]byte) ConfirmedOutcomeSet {
	t.Helper()
	binding := p07CLIProjectionBinding(t, cli.CLIFieldStdoutBytes, cli.CLIFieldStderrText)
	inputs := make([]ProjectionProofInput, len(projections))
	proofs := make([]projectiontranslate.ProjectionProof, len(projections))
	for index, projection := range projections {
		inputs[index] = trustedBytesInputForProjection(t, binding.Digest(), "candidate:portable-choice-"+string(rune('a'+index)), projection)
		proofs[index] = projectiontranslate.ProjectionProof{
			CandidateExecutionKey: inputs[index].CandidateExecutionKey,
			CanonicalProjection:   append([]byte(nil), projection...),
		}
	}
	translations, err := projectiontranslate.TranslateConfirmed(binding, testProjectionRoster(t, inputs), proofs)
	if err != nil {
		t.Fatalf("TranslateConfirmed(): %v", err)
	}
	confirmed, err := confirmedOutcomeSetFromTranslations(translations)
	if err != nil {
		t.Fatalf("confirmedOutcomeSetFromTranslations(): %v", err)
	}
	return confirmed
}

func portableCLIProjection(t testing.TB, stdout []byte, stderr string) []byte {
	t.Helper()
	exact, err := canon.CanonicalizeTyped(map[string]any{
		"fields": []map[string]any{
			{"field_id": string(cli.CLIFieldStdoutBytes), "value": map[string]any{"base64": base64.StdEncoding.EncodeToString(stdout), "tag": "BYTES"}},
			{"field_id": string(cli.CLIFieldStderrText), "value": map[string]any{"tag": "STRING", "value": stderr}},
		},
		"kind": "CLIProjection", "schema_version": "cli-projection/v1",
	})
	if err != nil {
		t.Fatalf("portable CLI projection: %v", err)
	}
	return exact
}

func fieldIDs(definitions []FieldDefinition) []string {
	result := make([]string, len(definitions))
	for index, definition := range definitions {
		result[index] = definition.ID
	}
	return result
}
