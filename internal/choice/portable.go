package choice

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
)

func newPortableFieldRegistry(profile projectionprofile.Profile, expectation projectiontranslate.ExpectationDomain) (FieldRegistry, error) {
	if !profile.Valid() || profile.Digest() == profile.Binding().Digest() || !expectation.Valid() ||
		expectation.ProfileDigest() != profile.Digest() || !bytes.Equal(expectation.ProfileBytes(), profile.CanonicalBytes()) {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "portable registry requires a distinct exact derived profile")
	}
	descriptors := profile.Fields()
	definitions := make([]FieldDefinition, len(descriptors))
	for index, descriptor := range descriptors {
		fieldType, ok := portableFieldType(descriptor.PortableTag)
		if !ok {
			return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "portable profile contains an unsupported value tag")
		}
		definitions[index] = FieldDefinition{
			ID: descriptor.FieldID, Path: []string{descriptor.FieldID}, Type: fieldType,
			AllowMissing: descriptor.AllowMissing, AllowNull: descriptor.AllowNull,
		}
	}
	registry, err := newFieldRegistry(definitions, true)
	if err != nil {
		return FieldRegistry{}, err
	}
	registry.sourceKinds = make(map[string]string, len(descriptors))
	for _, descriptor := range descriptors {
		registry.sourceKinds[descriptor.FieldID] = descriptor.SourceKind
	}
	identity := struct {
		SchemaVersion              string                    `json:"schema_version"`
		Kind                       string                    `json:"kind"`
		ProfileDigest              string                    `json:"portable_profile_digest"`
		ProjectionDefinitionDigest string                    `json:"projection_definition_digest"`
		ExpectationDomainDigest    string                    `json:"expectation_domain_digest"`
		Fields                     []fieldDefinitionIdentity `json:"fields"`
	}{
		SchemaVersion: domain.SchemaVersion, Kind: "ChoicePortableFieldRegistry",
		ProfileDigest: profile.Digest().String(), ProjectionDefinitionDigest: profile.Binding().Digest().String(),
		ExpectationDomainDigest: expectation.Digest().String(),
		Fields:                  make([]fieldDefinitionIdentity, len(definitions)),
	}
	for index, definition := range definitions {
		identity.Fields[index] = fieldDefinitionIdentity{
			ID: definition.ID, Path: append([]string(nil), definition.Path...), Type: string(definition.Type),
			AllowMissing: definition.AllowMissing, AllowNull: definition.AllowNull,
		}
	}
	digestRaw, _, err := canon.DigestTyped("ChoicePortableFieldRegistry", identity)
	if err != nil {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "portable registry identity could not be derived")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return FieldRegistry{}, refusal(CodeInvalidFieldRegistry, "portable registry digest is invalid")
	}
	registry.fieldRegistryDigest = digest
	registry.projectionDefinitionDigest = profile.Binding().Digest()
	registry.profileDigest = profile.Digest()
	registry.expectationDomain = expectation
	registry.mode = fieldRegistryPortable
	return registry, nil
}

func portableFieldType(tag portablevalue.Tag) (FieldType, bool) {
	switch tag {
	case portablevalue.TagString:
		return FieldString, true
	case portablevalue.TagInteger:
		return FieldInteger, true
	case portablevalue.TagBoolean:
		return FieldBoolean, true
	case portablevalue.TagBytes:
		return FieldBytes, true
	case portablevalue.TagOrderedStringList:
		return FieldOrderedStringList, true
	case portablevalue.TagCanonicalJSON:
		return FieldCanonicalJSON, true
	default:
		return "", false
	}
}

func exactValueFromPortable(value portablevalue.Value) (ExactValue, error) {
	if !value.Valid() {
		return ExactValue{}, refusal(CodeInvalidFieldType, "portable translator returned an invalid value")
	}
	switch value.Tag() {
	case portablevalue.TagMissing:
		return MissingValue(), nil
	case portablevalue.TagNull:
		return NullValue(), nil
	case portablevalue.TagBoolean:
		boolean, ok := value.BooleanValue()
		if !ok {
			break
		}
		return BooleanValue(boolean), nil
	case portablevalue.TagInteger:
		integer, ok := value.IntegerText()
		if !ok {
			break
		}
		return IntegerValue(integer)
	case portablevalue.TagString:
		text, ok := value.StringText()
		if !ok {
			break
		}
		return StringValue(text)
	case portablevalue.TagBytes:
		body, ok := value.BytesValue()
		if !ok {
			break
		}
		return BytesValue(body)
	case portablevalue.TagOrderedStringList:
		members, ok := value.OrderedStrings()
		if !ok {
			break
		}
		return OrderedStringListValue(members)
	case portablevalue.TagCanonicalJSON:
		exact, ok := value.CanonicalJSONBytes()
		if !ok {
			break
		}
		return CanonicalJSONBytes(exact)
	}
	return ExactValue{}, refusal(CodeInvalidFieldType, "portable translator value could not be converted exactly")
}

func portableValueFromExact(value ExactValue) (portablevalue.Value, error) {
	switch value.tag {
	case ValueMissing:
		return portablevalue.Missing(), nil
	case ValueNull:
		return portablevalue.Null(), nil
	case ValueBoolean:
		return portablevalue.Boolean(value.boolean), nil
	case ValueInteger:
		return portablevalue.Integer(value.text)
	case ValueString:
		return portablevalue.String(value.text)
	case ValueBytes:
		return portablevalue.Bytes(value.opaque)
	case ValueOrderedStringList:
		return portablevalue.OrderedStringListFromCanonical(value.canonical)
	case ValueCanonicalJSON:
		return portablevalue.CanonicalJSON(value.canonical)
	default:
		return portablevalue.Value{}, refusal(CodeInvalidFieldType, "Choice exact value has an unknown portable tag")
	}
}

func confirmedOutcomeSetFromTranslations(translations projectiontranslate.ConfirmedTranslations) (ConfirmedOutcomeSet, error) {
	if !translations.Valid() {
		return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcomeSet, "portable outcomes require one sealed proof-first translation")
	}
	registry, err := newPortableFieldRegistry(translations.Profile(), translations.ExpectationDomain())
	if err != nil {
		return ConfirmedOutcomeSet{}, err
	}
	outcomes := translations.Outcomes()
	set := ConfirmedOutcomeSet{
		registry: registry, ordered: make([]confirmedOutcome, 0, len(outcomes)),
		byID: make(map[string]confirmedOutcome, len(outcomes)), outcomeMapDigest: translations.OutcomeMapDigest(),
		preservationDigest: translations.PreservationDigest(), valid: true,
	}
	seenCandidates := make(map[string]struct{}, len(outcomes))
	for index, translated := range outcomes {
		candidate := translated.CandidateExecutionKey()
		fingerprint := translated.ProjectionFingerprint()
		portableTuple := translated.Tuple()
		if !candidate.Valid() || !fingerprint.Valid() || !portableTuple.Valid() ||
			portableTuple.ProfileDigest() != registry.profileDigest ||
			(index > 0 && candidate.String() <= outcomes[index-1].CandidateExecutionKey().String()) {
			return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcomeSet, "portable translated outcome order or identity differs")
		}
		if _, duplicate := seenCandidates[candidate.String()]; duplicate {
			return ConfirmedOutcomeSet{}, refusal(CodeDuplicateConfirmedCandidate, "portable translation repeats a candidate")
		}
		fields := portableTuple.Fields()
		if len(fields) != len(registry.orderedIDs) {
			return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcomeSet, "portable tuple does not cover the profile registry")
		}
		tuple := CompleteTuple{Fields: make([]FieldValue, len(fields))}
		for fieldIndex, field := range fields {
			if field.ID() != registry.orderedIDs[fieldIndex] {
				return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcomeSet, "portable tuple field order differs from the profile")
			}
			exact, convertErr := exactValueFromPortable(field.Value())
			if convertErr != nil {
				return ConfirmedOutcomeSet{}, convertErr
			}
			rebuilt, rebuildErr := portableValueFromExact(exact)
			if rebuildErr != nil || !bytes.Equal(rebuilt.IdentityBytes(), field.Value().IdentityBytes()) {
				return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcomeSet, "portable value conversion changed exact identity")
			}
			tuple.Fields[fieldIndex] = FieldValue{FieldID: field.ID(), Value: exact}
		}
		if _, validateErr := registry.validateTuple(tuple); validateErr != nil {
			return ConfirmedOutcomeSet{}, validateErr
		}
		id, idErr := makeConfirmedOutcomeID(candidate, fingerprint)
		if idErr != nil {
			return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcome, "portable confirmed outcome identity could not be derived")
		}
		if _, duplicate := set.byID[id.text]; duplicate {
			return ConfirmedOutcomeSet{}, refusal(CodeDuplicateConfirmedOutcome, "portable confirmed outcome identity occurs more than once")
		}
		projection := translated.CanonicalProjection()
		computed, fingerprintErr := domain.NewProjectionFingerprint(projection)
		if fingerprintErr != nil || computed != fingerprint {
			return ConfirmedOutcomeSet{}, refusal(CodeProjectionFingerprintMismatch, fmt.Sprintf("portable outcome %d lost original fingerprint authority", index))
		}
		seenCandidates[candidate.String()] = struct{}{}
		outcome := confirmedOutcome{
			ref:   ConfirmedOutcomeRef{id: id, candidate: candidate, fingerprint: fingerprint},
			tuple: cloneTuple(tuple), canonicalProjection: append([]byte(nil), projection...),
		}
		set.byID[id.text] = outcome
		set.ordered = append(set.ordered, outcome)
	}
	sort.Slice(set.ordered, func(i, j int) bool { return set.ordered[i].ref.id.text < set.ordered[j].ref.id.text })
	seal, err := makeConfirmedOutcomeSetSeal(set.registry, set.outcomeMapDigest, set.preservationDigest, set.ordered)
	if err != nil {
		return ConfirmedOutcomeSet{}, refusal(CodeInvalidConfirmedOutcomeSet, "portable confirmed universe seal could not be derived")
	}
	set.seal = seal
	for index := range set.ordered {
		set.ordered[index].ref.universe = seal
		set.byID[set.ordered[index].ref.id.text] = set.ordered[index]
	}
	return set, nil
}

// PortableRulingInspection is non-authorizing semantic metadata. It lets the
// store-backed promotion layer distinguish portable decisions from parseable
// legacy history without parsing mode strings. Only promotion may turn this
// inspection into current-head preparation authority for P07B.
type PortableRulingInspection struct {
	decisionDigest domain.Digest
	profileDigest  domain.Digest
	selectedFields []string
	seal           *portableInspectionSeal
}

type portableInspectionSeal struct{}

var portableInspectionAuthority = &portableInspectionSeal{}

// InspectPortableRuling refuses legacy whole-projection history with the exact
// stable code required by the P07 boundary. A successful result remains inert:
// it proves semantic shape, not that the DecisionRecord is the current head.
func InspectPortableRuling(decision DecisionRecord) (PortableRulingInspection, error) {
	if !decision.Valid() {
		return PortableRulingInspection{}, refusal(CodeInvalidSessionState, "portable inspection requires an exact DecisionRecord")
	}
	if decision.choicepoint.mode == choicepointLegacyWhole {
		return PortableRulingInspection{}, refusal(CodeLegacyWholeProjectionNotPortable, "legacy whole-projection rulings are parseable history only")
	}
	if decision.choicepoint.mode != choicepointPortable || !decision.choicepoint.confirmed.registry.profileDigest.Valid() {
		return PortableRulingInspection{}, refusal(CodeInvalidSessionState, "DecisionRecord lacks closed portable profile authority")
	}
	if _, compilable := decision.CompilableRuling(); !compilable {
		return PortableRulingInspection{}, refusal(CodeNoncompilablePredicate, "noncompilable action cannot enter portable preparation")
	}
	return PortableRulingInspection{
		decisionDigest: decision.digest, profileDigest: decision.choicepoint.confirmed.registry.profileDigest,
		selectedFields: decision.SelectedFields(), seal: portableInspectionAuthority,
	}, nil
}

func (p PortableRulingInspection) Valid() bool {
	return p.seal == portableInspectionAuthority && p.decisionDigest.Valid() && p.profileDigest.Valid() && len(p.selectedFields) > 0
}

func (p PortableRulingInspection) DecisionDigest() domain.Digest { return p.decisionDigest }
func (p PortableRulingInspection) ProfileDigest() domain.Digest  { return p.profileDigest }
func (p PortableRulingInspection) SelectedFields() []string {
	return append([]string(nil), p.selectedFields...)
}
