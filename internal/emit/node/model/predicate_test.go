package model

import (
	"bytes"
	"encoding/base64"
	"slices"
	"testing"
	"unicode/utf16"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

const (
	sourceProfileGoldenBody   = `{"adapter_domain":"CLI","launch_profile":"NODE_REPO_SCRIPT_V1","runtime_family":"NODE","scope":"DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE","semantic_profile":"countershape-node-core-exact/v1","start_profile":"DIRECT_CHILD_V1","subject_entrypoint":"fixture/subject.mjs"}`
	sourceProfileGoldenDigest = "sha256:68de7ca7aa438e978b1f5af0c8cebc5775582eef95e21dc7ccfa9b2392ab480c"
)

func TestExactValueWireCoversClosedPortableAlgebra(t *testing.T) {
	integer, err := portablevalue.Integer("-42")
	if err != nil {
		t.Fatal(err)
	}
	text, err := portablevalue.String("")
	if err != nil {
		t.Fatal(err)
	}
	opaque, err := portablevalue.Bytes([]byte{0xff, 0x00, 0x61})
	if err != nil {
		t.Fatal(err)
	}
	list, err := portablevalue.OrderedStringList([]string{"", "same", "same", "last"})
	if err != nil {
		t.Fatal(err)
	}
	canonicalJSON, err := portablevalue.CanonicalJSON([]byte(`{"__proto__":{"safe":true},"n":1}`))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		value portablevalue.Value
		want  string
	}{
		{"missing", portablevalue.Missing(), `{"tag":"MISSING"}`},
		{"null", portablevalue.Null(), `{"tag":"NULL"}`},
		{"boolean false", portablevalue.Boolean(false), `{"tag":"BOOLEAN","value":false}`},
		{"integer", integer, `{"canonical":"-42","tag":"INTEGER"}`},
		{"empty string", text, `{"tag":"STRING","value":""}`},
		{"opaque bytes", opaque, `{"base64":"/wBh","tag":"BYTES"}`},
		{"ordered list", list, `{"tag":"ORDERED_STRING_LIST","values":["","same","same","last"]}`},
		{"canonical json", canonicalJSON, `{"canonical_base64":"` + base64.StdEncoding.EncodeToString([]byte(`{"__proto__":{"safe":true},"n":1}`)) + `","tag":"CANONICAL_JSON"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := NewExactValue(test.value)
			if err != nil || !value.Valid() || string(value.CanonicalBytes()) != test.want {
				t.Fatalf("exact value = %s, valid=%t, err=%v; want %s", value.CanonicalBytes(), value.Valid(), err, test.want)
			}
			copyBytes := value.CanonicalBytes()
			copyBytes[0] ^= 0xff
			if string(value.CanonicalBytes()) != test.want {
				t.Fatal("canonical getter mutation changed exact value")
			}
		})
	}
	missing, _ := NewExactValue(portablevalue.Missing())
	emptyString, _ := NewExactValue(text)
	emptyBytesValue, _ := portablevalue.Bytes([]byte{})
	emptyBytes, _ := NewExactValue(emptyBytesValue)
	emptyListValue, _ := portablevalue.OrderedStringList([]string{})
	emptyList, _ := NewExactValue(emptyListValue)
	if bytes.Equal(missing.CanonicalBytes(), emptyString.CanonicalBytes()) ||
		bytes.Equal(emptyString.CanonicalBytes(), emptyBytes.CanonicalBytes()) ||
		bytes.Equal(emptyBytes.CanonicalBytes(), emptyList.CanonicalBytes()) {
		t.Fatal("missing and present-empty values collapsed")
	}
}

func TestExactValueMutationFirewallRetainsAbsenceOpaqueOrderAndJSONBytes(t *testing.T) {
	missing, err := NewExactValue(portablevalue.Missing())
	if err != nil {
		t.Fatal(err)
	}
	emptyPortable, err := portablevalue.String("")
	if err != nil {
		t.Fatal(err)
	}
	empty, err := NewExactValue(emptyPortable)
	if err != nil || bytes.Equal(missing.CanonicalBytes(), empty.CanonicalBytes()) ||
		string(missing.CanonicalBytes()) != `{"tag":"MISSING"}` {
		t.Fatalf("MISSING collapsed into present empty string: missing=%s empty=%s err=%v", missing.CanonicalBytes(), empty.CanonicalBytes(), err)
	}

	opaquePortable, err := portablevalue.Bytes([]byte{0xff, 0x00, 0x61})
	if err != nil {
		t.Fatal(err)
	}
	opaque, err := NewExactValue(opaquePortable)
	if err != nil || string(opaque.CanonicalBytes()) != `{"base64":"/wBh","tag":"BYTES"}` {
		t.Fatalf("opaque bytes crossed a text conversion: %s, %v", opaque.CanonicalBytes(), err)
	}

	orderedPortable, err := portablevalue.OrderedStringList([]string{"z", "", "z", "a"})
	if err != nil {
		t.Fatal(err)
	}
	ordered, err := NewExactValue(orderedPortable)
	if err != nil || string(ordered.CanonicalBytes()) != `{"tag":"ORDERED_STRING_LIST","values":["z","","z","a"]}` {
		t.Fatalf("ordered list was sorted or deduplicated: %s, %v", ordered.CanonicalBytes(), err)
	}

	jsonBytes := []byte(`{"html":"<"}`)
	jsonPortable, err := portablevalue.CanonicalJSON(jsonBytes)
	if err != nil {
		t.Fatal(err)
	}
	exactJSON, err := NewExactValue(jsonPortable)
	wantJSONWire := `{"canonical_base64":"` + base64.StdEncoding.EncodeToString(jsonBytes) + `","tag":"CANONICAL_JSON"}`
	if err != nil || string(exactJSON.CanonicalBytes()) != wantJSONWire {
		t.Fatalf("canonical JSON crossed generic stringify escaping: %s, %v", exactJSON.CanonicalBytes(), err)
	}
}

func TestPredicateCanonicalizesCompleteTupleSetByUnsignedBytes(t *testing.T) {
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := projectiontranslate.Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	profile := resolved.Profile()
	selected := []string{string(counterhttp.HTTPFieldStatus), string(counterhttp.HTTPFieldBodyKind)}
	tuple := func(status, kind string) ExactTuple {
		integer, integerErr := portablevalue.Integer(status)
		if integerErr != nil {
			t.Fatal(integerErr)
		}
		text, textErr := portablevalue.String(kind)
		if textErr != nil {
			t.Fatal(textErr)
		}
		integerValue, _ := NewExactValue(integer)
		textValue, _ := NewExactValue(text)
		statusField, _ := NewExactField(selected[0], integerValue)
		kindField, _ := NewExactField(selected[1], textValue)
		result, tupleErr := NewExactTuple([]ExactField{statusField, kindField})
		if tupleErr != nil {
			t.Fatal(tupleErr)
		}
		return result
	}
	first := tuple("500", "z")
	second := tuple("200", "a")
	predicate, err := NewPredicate(
		testModelDigest('a'), profile, selected, []ExactTuple{first, second, first},
	)
	if err != nil || !predicate.Valid() {
		t.Fatalf("NewPredicate() = valid %t, %v", predicate.Valid(), err)
	}
	allowed := predicate.AllowedTuples()
	if len(allowed) != 2 || bytes.Compare(allowed[0].CanonicalBytes(), allowed[1].CanonicalBytes()) >= 0 {
		t.Fatalf("allowed tuples were not byte-deduplicated and unsigned-sorted: %#v", allowed)
	}
	reversed, err := NewPredicate(
		testModelDigest('a'), profile, selected, []ExactTuple{second, first},
	)
	if err != nil || !bytes.Equal(predicate.CanonicalBytes(), reversed.CanonicalBytes()) {
		t.Fatalf("construction order changed predicate: %v", err)
	}
	selectedCopy := predicate.SelectedFields()
	selectedCopy[0] = "http.body.kind"
	allowedCopy := predicate.AllowedTuples()
	allowedCopy[0].canonical[0] ^= 0xff
	if !slices.Equal(predicate.SelectedFields(), selected) || !predicate.Valid() {
		t.Fatal("predicate defensive getters changed authority")
	}
	if _, err := NewPredicate(
		testModelDigest('a'), profile, []string{selected[1], selected[0]}, []ExactTuple{first},
	); !IsCode(err, "INVALID_PREDICATE") {
		t.Fatalf("profile-order alias refusal = %v", err)
	}
}

func TestPredicateUsesUnsignedCanonicalTupleByteOrder(t *testing.T) {
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := projectiontranslate.Resolve(definition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	const fieldID = "http.body.kind"
	tuple := func(text string) ExactTuple {
		portable, valueErr := portablevalue.String(text)
		if valueErr != nil {
			t.Fatal(valueErr)
		}
		value, valueErr := NewExactValue(portable)
		if valueErr != nil {
			t.Fatal(valueErr)
		}
		field, fieldErr := NewExactField(fieldID, value)
		if fieldErr != nil {
			t.Fatal(fieldErr)
		}
		result, tupleErr := NewExactTuple([]ExactField{field})
		if tupleErr != nil {
			t.Fatal(tupleErr)
		}
		return result
	}
	ascii := tuple("a")
	accented := tuple("é")
	privateUse := tuple("\ue000")
	emoji := tuple("😀")
	predicate, err := NewPredicate(
		testModelDigest('b'), resolved.Profile(), []string{fieldID},
		[]ExactTuple{emoji, privateUse, accented, ascii},
	)
	if err != nil {
		t.Fatal(err)
	}
	allowed := predicate.AllowedTuples()
	for index := 1; index < len(allowed); index++ {
		if bytes.Compare(allowed[index-1].CanonicalBytes(), allowed[index].CanonicalBytes()) >= 0 {
			t.Fatalf("tuple bodies are not unsigned-byte sorted at %d", index)
		}
	}
	indexOf := func(target ExactTuple) int {
		for index, candidate := range allowed {
			if bytes.Equal(candidate.CanonicalBytes(), target.CanonicalBytes()) {
				return index
			}
		}
		return -1
	}
	if indexOf(ascii) >= indexOf(accented) || indexOf(privateUse) >= indexOf(emoji) {
		t.Fatalf("unsigned UTF-8 tuple order drifted: ascii=%d accented=%d private=%d emoji=%d",
			indexOf(ascii), indexOf(accented), indexOf(privateUse), indexOf(emoji))
	}
	privateUTF16 := utf16.Encode([]rune("\ue000"))
	emojiUTF16 := utf16.Encode([]rune("😀"))
	if privateUTF16[0] <= emojiUTF16[0] {
		t.Fatal("UTF-16 reversal fixture no longer distinguishes canonical byte order")
	}
	reversed, err := NewPredicate(
		testModelDigest('b'), resolved.Profile(), []string{fieldID},
		[]ExactTuple{ascii, accented, privateUse, emoji},
	)
	if err != nil || !bytes.Equal(predicate.CanonicalBytes(), reversed.CanonicalBytes()) {
		t.Fatalf("input ordering changed unsigned-byte canonical predicate: %v", err)
	}
}

func TestExactTupleNeverSynthesizesCartesianProduct(t *testing.T) {
	value := func(text string) ExactValue {
		portable, err := portablevalue.String(text)
		if err != nil {
			t.Fatal(err)
		}
		exact, err := NewExactValue(portable)
		if err != nil {
			t.Fatal(err)
		}
		return exact
	}
	field := func(id, text string) ExactField {
		result, err := NewExactField(id, value(text))
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	left, err := NewExactTuple([]ExactField{
		field("cli.stdout.json.mode", "config"), field("cli.stdout.json.source", "config"),
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewExactTuple([]ExactField{
		field("cli.stdout.json.mode", "argv"), field("cli.stdout.json.source", "argv"),
	})
	if err != nil {
		t.Fatal(err)
	}
	cross, err := NewExactTuple([]ExactField{
		field("cli.stdout.json.mode", "config"), field("cli.stdout.json.source", "argv"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(left.CanonicalBytes(), cross.CanonicalBytes()) || bytes.Equal(right.CanonicalBytes(), cross.CanonicalBytes()) {
		t.Fatal("correlated tuple identity collapsed into independent field values")
	}
}

func TestSourceProfileUsesExactSevenMemberContractDomain(t *testing.T) {
	wire := sourceProfileWire{
		RuntimeFamily: RuntimeFamilyNodeV1, SemanticProfile: "countershape-node-core-exact/v1",
		AdapterDomain: string(domain.AdapterCLI), LaunchProfile: "NODE_REPO_SCRIPT_V1",
		SubjectEntrypoint: "fixture/subject.mjs", StartProfile: "DIRECT_CHILD_V1",
		Scope: DeclaredSourceScopeV1,
	}
	digest, canonical, err := canon.DigestTyped(ContractSourceProfileDomain, wire)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical) != sourceProfileGoldenBody || digest.String() != sourceProfileGoldenDigest {
		t.Fatalf("independent ContractSourceProfile golden drifted: body=%s digest=%s", canonical, digest.String())
	}
	profile := SourceProfile{
		adapter: domain.AdapterCLI, entrypoint: wire.SubjectEntrypoint, start: wire.StartProfile,
		digest: parsed, canonical: canonical,
	}
	if !profile.Valid() {
		t.Fatal("exact ContractSourceProfile did not validate")
	}
	value, err := canon.Parse(profile.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	members, object := value.Members()
	if !object || len(members) != 7 || bytes.HasSuffix(profile.CanonicalBytes(), []byte("\n")) {
		t.Fatalf("source profile body object=%t members=%d body=%q", object, len(members), profile.CanonicalBytes())
	}
	wrongDomain, _ := canon.DigestBytes("SourceProfile", canonical)
	wrongDigest, _ := domain.ParseDigest(wrongDomain.String())
	profile.digest = wrongDigest
	if profile.Valid() {
		t.Fatal("same source-profile body under the wrong digest domain validated")
	}
	withLF := append(append([]byte(nil), canonical...), '\n')
	lfDigest, _ := canon.DigestBytes(ContractSourceProfileDomain, withLF)
	profile.digest, _ = domain.ParseDigest(lfDigest.String())
	if profile.Valid() {
		t.Fatal("LF-suffixed source-profile digest validated against the no-LF body")
	}
	for _, invalid := range []SourceProfile{
		{adapter: domain.AdapterCLI, entrypoint: "fixture/subject.mjs", start: httpPortableStartProfileV1},
		{adapter: domain.AdapterHTTP, entrypoint: "fixture/subject.mjs", start: contractsource.CLIStartProfileV1},
		{adapter: domain.AdapterCLI, entrypoint: "../subject.mjs", start: contractsource.CLIStartProfileV1},
	} {
		invalidWire := sourceProfileWire{
			RuntimeFamily: RuntimeFamilyNodeV1, SemanticProfile: contractsource.SourceProfileV1,
			AdapterDomain: string(invalid.adapter), LaunchProfile: contractsource.LaunchProfileV1,
			SubjectEntrypoint: invalid.entrypoint, StartProfile: invalid.start, Scope: DeclaredSourceScopeV1,
		}
		invalidDigest, invalidCanonical, invalidErr := canon.DigestTyped(ContractSourceProfileDomain, invalidWire)
		if invalidErr != nil {
			t.Fatal(invalidErr)
		}
		invalid.digest, _ = domain.ParseDigest(invalidDigest.String())
		invalid.canonical = invalidCanonical
		if invalid.Valid() {
			t.Fatalf("self-consistent invalid source-profile pair validated: %#v", invalidWire)
		}
	}
}

func FuzzExactOpaqueBytesRemainExact(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xff, 0x00, 0x61})
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 1024 {
			t.Skip()
		}
		portable, err := portablevalue.Bytes(input)
		if err != nil {
			t.Fatal(err)
		}
		exact, err := NewExactValue(portable)
		if err != nil || !exact.Valid() {
			t.Fatalf("NewExactValue() valid=%t err=%v", exact.Valid(), err)
		}
		got, ok := exact.PortableValue().BytesValue()
		if !ok || !bytes.Equal(got, input) {
			t.Fatal("opaque bytes changed through emitter model")
		}
	})
}

func testModelDigest(fill byte) domain.Digest {
	return domain.MustDigest("sha256:" + string(bytes.Repeat([]byte{fill}, 64)))
}
