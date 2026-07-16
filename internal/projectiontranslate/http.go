package projectiontranslate

import (
	"strconv"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

const (
	httpTranslatorName    = "HTTP_PROJECTION_TO_PORTABLE"
	httpTranslatorVersion = "v1"
)

func resolveHTTP(binding domain.ProjectionDefinitionBinding) (Resolved, error) {
	definition, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		return Resolved{}, refuse(CodeInvalidBinding, "", "adapter-owned HTTP definition reconstruction failed")
	}
	if !exactBindingMatch(definition.Binding(), binding) {
		return Resolved{}, refuse(CodeProfileNotFound, "", "fixed HTTP definition does not match the exact projection binding")
	}
	registry := counterhttp.HTTPFieldRegistry()
	expectedRegistry := []counterhttp.HTTPFieldID{
		counterhttp.HTTPFieldStatus, counterhttp.HTTPFieldContentType,
		counterhttp.HTTPFieldBodyKind, counterhttp.HTTPFieldBodyMetadata,
	}
	if len(registry) != len(expectedRegistry) {
		return Resolved{}, refuse(CodeInvalidBinding, "", "HTTP translator v1 registry cardinality drifted")
	}
	for index, expected := range expectedRegistry {
		if registry[index].ID() != expected {
			return Resolved{}, refuse(CodeInvalidBinding, "", "HTTP translator v1 registry order drifted")
		}
	}
	descriptors := make([]projectionprofile.Descriptor, len(registry))
	for index, descriptor := range registry {
		tag, ok := portableTagForSourceKind(descriptor.ValueKind())
		if !ok {
			return Resolved{}, refuse(CodeInvalidBinding, string(descriptor.ID()), "HTTP descriptor has no closed portable mapping")
		}
		descriptors[index] = projectionprofile.Descriptor{
			FieldID: string(descriptor.ID()), Channel: string(descriptor.Channel()), SourcePath: descriptor.Path(),
			SourceKind: descriptor.ValueKind(), MissingPolicy: descriptor.MissingPolicy(), PortableTag: tag,
			AllowMissing: descriptor.MissingPolicy() == "TAGGED_MISSING",
		}
	}
	return newResolvedProfile(httpTranslatorName, httpTranslatorVersion, binding, descriptors, armHTTP, "HTTPStimulus")
}

func (r Resolved) translateHTTP(exact []byte) (Tuple, error) {
	root, err := parseExact(exact)
	if err != nil {
		return Tuple{}, err
	}
	members, err := exactObject(root, "Fields", "Kind", "SchemaVersion")
	if err != nil {
		return Tuple{}, err
	}
	kind, err := exactText(members[1])
	if err != nil || kind != "HTTPProjection" {
		return Tuple{}, refuse(CodeInvalidProjection, "", "HTTP projection kind differs")
	}
	version, err := exactText(members[2])
	if err != nil || version != domain.SchemaVersion {
		return Tuple{}, refuse(CodeInvalidProjection, "", "HTTP projection schema version differs")
	}
	fieldValues, err := exactArray(members[0])
	if err != nil {
		return Tuple{}, err
	}
	profileFields := r.profile.Fields()
	if len(fieldValues) != len(profileFields) {
		return Tuple{}, refuse(CodeRosterMismatch, "", "HTTP projection field count differs from the resolved profile")
	}
	fields := make([]Field, len(fieldValues))
	for index, fieldValue := range fieldValues {
		value, fieldID, parseErr := translateHTTPField(profileFields[index], fieldValue)
		if parseErr != nil {
			return Tuple{}, parseErr
		}
		fields[index] = Field{id: fieldID, value: value}
	}
	return newTuple(r.profile, fields)
}

func translateHTTPField(descriptor projectionprofile.Descriptor, encoded canon.Value) (portablevalue.Value, string, error) {
	members, err := exactObject(encoded, "CanonicalJSON", "FieldID", "Integer", "String", "Strings", "Tag")
	if err != nil {
		return portablevalue.Value{}, "", err
	}
	canonicalJSON, err := exactText(members[0])
	if err != nil {
		return portablevalue.Value{}, "", refuse(CodeInvalidPayload, descriptor.FieldID, "HTTP CanonicalJSON slot is not a string")
	}
	fieldID, err := exactText(members[1])
	if err != nil || fieldID != descriptor.FieldID {
		return portablevalue.Value{}, fieldID, refuse(CodeRosterMismatch, fieldID, "HTTP projection field ID or registry order differs")
	}
	integer, integerOK := members[2].Int64()
	text, textErr := exactText(members[3])
	stringValues, arrayErr := exactArray(members[4])
	tag, tagErr := exactText(members[5])
	if !integerOK || textErr != nil || arrayErr != nil || tagErr != nil {
		return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP six-slot field payload types differ")
	}
	emptyStrings := len(stringValues) == 0
	switch descriptor.FieldID {
	case string(counterhttp.HTTPFieldStatus):
		if tag != "INTEGER" || descriptor.PortableTag != portablevalue.TagInteger || integer < 200 || integer > 599 || !emptyStrings || text != "" || canonicalJSON != "" {
			return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP status tag, range, or unused slots differ")
		}
		value, valueErr := portablevalue.Integer(strconv.FormatInt(integer, 10))
		if valueErr != nil {
			return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, valueErr.Error())
		}
		return value, fieldID, nil
	case string(counterhttp.HTTPFieldContentType):
		if integer != 0 || text != "" || canonicalJSON != "" {
			return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP content-type unused slots are nonzero")
		}
		if tag == "MISSING" {
			if !emptyStrings || !descriptor.AllowMissing {
				return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP missing content-type carries list values")
			}
			return portablevalue.Missing(), fieldID, nil
		}
		if tag != "STRING_LIST" || emptyStrings || descriptor.PortableTag != portablevalue.TagOrderedStringList {
			return portablevalue.Value{}, fieldID, refuse(CodeInvalidTag, fieldID, "HTTP content-type tag or empty-list state differs")
		}
		if len(stringValues) > portablevalue.MaxListMembers {
			return portablevalue.Value{}, fieldID, refuse(CodeTranslationLimit, fieldID, "HTTP content-type exceeds the portable member ceiling")
		}
		strings := make([]string, len(stringValues))
		for index, encodedString := range stringValues {
			member, valueErr := exactText(encodedString)
			if valueErr != nil {
				return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP Strings slot contains a non-string")
			}
			strings[index] = member
		}
		for _, member := range strings {
			for index := 0; index < len(member); index++ {
				if member[index] < 0x20 || member[index] > 0x7e {
					return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP content-type contains non-printable historical bytes")
				}
			}
		}
		value, valueErr := portablevalue.OrderedStringList(strings) // MUTANT_P07A_PRESERVE_ORDERED_LIST
		if valueErr != nil {
			return portablevalue.Value{}, fieldID, refuse(CodeTranslationLimit, fieldID, valueErr.Error())
		}
		return value, fieldID, nil
	case string(counterhttp.HTTPFieldBodyKind):
		if tag != "STRING" || descriptor.PortableTag != portablevalue.TagString || integer != 0 || !emptyStrings || canonicalJSON != "" {
			return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP body-kind tag or unused slots differ")
		}
		value, valueErr := portablevalue.String(text)
		if valueErr != nil {
			return portablevalue.Value{}, fieldID, refuse(CodeTranslationLimit, fieldID, valueErr.Error())
		}
		return value, fieldID, nil
	case string(counterhttp.HTTPFieldBodyMetadata):
		if tag != "CANONICAL_JSON" || descriptor.PortableTag != portablevalue.TagCanonicalJSON || integer != 0 || !emptyStrings || text != "" || canonicalJSON == "" { // MUTANT_P07A_HTTP_UNUSED_SLOTS
			return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP metadata tag or unused slots differ")
		}
		if len(canonicalJSON) > portablevalue.MaxCanonicalJSONBytes {
			return portablevalue.Value{}, fieldID, refuse(CodeTranslationLimit, fieldID, "HTTP metadata exceeds the portable canonical-JSON ceiling")
		}
		parsed, parseErr := canon.Parse([]byte(canonicalJSON))
		if parseErr != nil || parsed.Kind() != canon.KindObject {
			return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, "HTTP metadata is not a strict canonical JSON object")
		}
		value, valueErr := portablevalue.CanonicalJSON([]byte(canonicalJSON))
		if valueErr != nil {
			if portablevalue.IsCode(valueErr, portablevalue.CodeLimitExceeded) {
				return portablevalue.Value{}, fieldID, refuse(CodeTranslationLimit, fieldID, valueErr.Error())
			}
			return portablevalue.Value{}, fieldID, refuse(CodeInvalidPayload, fieldID, valueErr.Error())
		}
		return value, fieldID, nil
	default:
		return portablevalue.Value{}, fieldID, refuse(CodeRosterMismatch, fieldID, "HTTP field escaped the fixed profile roster")
	}
}
