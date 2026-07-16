package projectiontranslate

import (
	"encoding/base64"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

const (
	cliTranslatorName    = "CLI_PROJECTION_TO_PORTABLE"
	cliTranslatorVersion = "v1"
)

func resolveCLI(binding domain.ProjectionDefinitionBinding) (Resolved, error) {
	registry := cli.CLIFieldRegistry()
	expectedRegistry := []cli.CLIFieldID{
		cli.CLIFieldCompletionKind, cli.CLIFieldExitCode, cli.CLIFieldExitSignal, cli.CLIFieldStdoutBytes,
		cli.CLIFieldStderrText, cli.CLIFieldStdoutJSONMode, cli.CLIFieldStdoutJSONSource,
	}
	if len(registry) != len(expectedRegistry) {
		return Resolved{}, refuse(CodeInvalidBinding, "", "CLI translator v1 registry cardinality drifted")
	}
	for index, expected := range expectedRegistry {
		if registry[index].ID() != expected {
			return Resolved{}, refuse(CodeInvalidBinding, "", "CLI translator v1 registry order drifted")
		}
	}
	matches := 0
	var matched cli.CLIProjectionDefinition
	for mask := 1; mask < 1<<len(registry); mask++ {
		fields := make([]cli.CLIFieldID, 0, len(registry))
		for index, descriptor := range registry {
			if mask&(1<<index) != 0 {
				fields = append(fields, descriptor.ID())
			}
		}
		candidate, err := cli.NewCLIProjectionDefinition(cli.CLIProjectionDefinitionConfig{Fields: fields})
		if err != nil {
			return Resolved{}, refuse(CodeInvalidBinding, "", "adapter-owned CLI definition reconstruction failed")
		}
		if exactBindingMatch(candidate.Binding(), binding) {
			matches++
			matched = candidate
		}
	}
	if matches == 0 {
		return Resolved{}, refuse(CodeProfileNotFound, "", "no CLI field subset matches the exact projection binding")
	}
	if matches != 1 {
		return Resolved{}, refuse(CodeProfileAmbiguous, "", "more than one CLI field subset matches the exact projection binding")
	}
	descriptors := make([]projectionprofile.Descriptor, len(matched.Fields()))
	byID := make(map[cli.CLIFieldID]cli.CLIFieldDescriptor, len(registry))
	for _, descriptor := range registry {
		byID[descriptor.ID()] = descriptor
	}
	for index, fieldID := range matched.Fields() {
		descriptor := byID[fieldID]
		tag, ok := portableTagForSourceKind(string(descriptor.ValueKind()))
		if !ok {
			return Resolved{}, refuse(CodeInvalidBinding, string(fieldID), "CLI descriptor has no closed portable mapping")
		}
		descriptors[index] = projectionprofile.Descriptor{
			FieldID: string(descriptor.ID()), Channel: string(descriptor.Channel()), SourcePath: descriptor.Path(),
			SourceKind: string(descriptor.ValueKind()), MissingPolicy: descriptor.MissingPolicy(), PortableTag: tag,
			AllowMissing: cliMissingPolicy(descriptor.MissingPolicy()),
		}
	}
	return newResolvedProfile(cliTranslatorName, cliTranslatorVersion, binding, descriptors, armCLI, "CLIStimulus")
}

func cliMissingPolicy(policy string) bool {
	switch policy {
	case "TAGGED_MISSING", "TAGGED_MISSING_FOR_SIGNAL", "TAGGED_MISSING_FOR_EXIT":
		return true
	default:
		return false
	}
}

func portableTagForSourceKind(kind string) (portablevalue.Tag, bool) {
	switch kind {
	case "UTF8_STRING":
		return portablevalue.TagString, true
	case "SAFE_INTEGER":
		return portablevalue.TagInteger, true
	case "BYTES":
		return portablevalue.TagBytes, true
	case "ORDERED_STRING_LIST":
		return portablevalue.TagOrderedStringList, true
	case "CANONICAL_JSON_OBJECT":
		return portablevalue.TagCanonicalJSON, true
	default:
		return "", false
	}
}

func (r Resolved) translateCLI(exact []byte) (Tuple, error) {
	root, err := parseExact(exact) // MUTANT_P07A_SKIP_EXACT_CANONICAL
	if err != nil {
		return Tuple{}, err
	}
	members, err := exactObject(root, "fields", "kind", "schema_version")
	if err != nil {
		return Tuple{}, err
	}
	kind, err := exactText(members[1])
	if err != nil || kind != "CLIProjection" {
		return Tuple{}, refuse(CodeInvalidProjection, "", "CLI projection kind differs")
	}
	version, err := exactText(members[2])
	if err != nil || version != "cli-projection/v1" {
		return Tuple{}, refuse(CodeInvalidProjection, "", "CLI projection schema version differs")
	}
	fieldValues, err := exactArray(members[0])
	if err != nil {
		return Tuple{}, err
	}
	profileFields := r.profile.Fields()
	if len(fieldValues) != len(profileFields) {
		return Tuple{}, refuse(CodeRosterMismatch, "", "CLI projection field count differs from the resolved profile")
	}
	fields := make([]Field, len(fieldValues))
	for index, fieldValue := range fieldValues {
		fieldMembers, parseErr := exactObject(fieldValue, "field_id", "value")
		if parseErr != nil {
			return Tuple{}, parseErr
		}
		fieldID, parseErr := exactText(fieldMembers[0])
		if parseErr != nil || fieldID != profileFields[index].FieldID { // MUTANT_P07A_IGNORE_CLI_ROSTER_ORDER
			return Tuple{}, refuse(CodeRosterMismatch, fieldID, "CLI projection field ID or registry order differs")
		}
		value, parseErr := translateCLIValue(profileFields[index], fieldMembers[1])
		if parseErr != nil {
			return Tuple{}, parseErr
		}
		fields[index] = Field{id: fieldID, value: value}
	}
	return newTuple(r.profile, fields)
}

func translateCLIValue(descriptor projectionprofile.Descriptor, encoded canon.Value) (portablevalue.Value, error) {
	tagValue, present := encoded.LookupMember("tag")
	if !present {
		return portablevalue.Value{}, refuse(CodeInvalidTag, descriptor.FieldID, "CLI value omits its tag")
	}
	tag, err := exactText(tagValue)
	if err != nil {
		return portablevalue.Value{}, refuse(CodeInvalidTag, descriptor.FieldID, "CLI value tag is not a string")
	}
	switch tag {
	case "MISSING":
		if _, err := exactObject(encoded, "tag"); err != nil || !descriptor.AllowMissing {
			return portablevalue.Value{}, refuse(CodeInvalidPayload, descriptor.FieldID, "CLI missing value shape or policy differs")
		}
		return portablevalue.Missing(), nil
	case "STRING":
		members, err := exactObject(encoded, "tag", "value")
		if err != nil || descriptor.PortableTag != portablevalue.TagString {
			return portablevalue.Value{}, refuse(CodeInvalidPayload, descriptor.FieldID, "CLI string value shape or field type differs")
		}
		text, err := exactText(members[1])
		if err != nil {
			return portablevalue.Value{}, refuse(CodeInvalidPayload, descriptor.FieldID, "CLI string payload is not a string")
		}
		value, err := portablevalue.String(text)
		if err != nil {
			return portablevalue.Value{}, refuse(CodeTranslationLimit, descriptor.FieldID, err.Error())
		}
		return value, nil
	case "INTEGER":
		members, err := exactObject(encoded, "canonical", "tag")
		if err != nil || descriptor.PortableTag != portablevalue.TagInteger {
			return portablevalue.Value{}, refuse(CodeInvalidPayload, descriptor.FieldID, "CLI integer value shape or field type differs")
		}
		canonical, err := exactText(members[0])
		if err != nil {
			return portablevalue.Value{}, refuse(CodeInvalidPayload, descriptor.FieldID, "CLI integer spelling is not a string")
		}
		value, err := portablevalue.Integer(canonical)
		if err != nil {
			return portablevalue.Value{}, refuse(CodeInvalidPayload, descriptor.FieldID, err.Error())
		}
		return value, nil
	case "BYTES":
		members, err := exactObject(encoded, "base64", "tag")
		if err != nil || descriptor.PortableTag != portablevalue.TagBytes {
			return portablevalue.Value{}, refuse(CodeInvalidPayload, descriptor.FieldID, "CLI byte value shape or field type differs")
		}
		encodedBytes, err := exactText(members[0])
		if err != nil || len(encodedBytes) > base64.StdEncoding.EncodedLen(portablevalue.MaxBytesValueBytes) {
			return portablevalue.Value{}, refuse(CodeTranslationLimit, descriptor.FieldID, "CLI byte base64 exceeds the portable ceiling")
		}
		decoded, err := base64.StdEncoding.Strict().DecodeString(encodedBytes)
		if err != nil || base64.StdEncoding.EncodeToString(decoded) != encodedBytes {
			return portablevalue.Value{}, refuse(CodeInvalidPayload, descriptor.FieldID, "CLI byte base64 is not canonical padded encoding")
		}
		value, err := portablevalue.Bytes(decoded) // MUTANT_P07A_BYTES_THROUGH_UTF8
		if err != nil {
			return portablevalue.Value{}, refuse(CodeTranslationLimit, descriptor.FieldID, err.Error())
		}
		return value, nil
	default:
		return portablevalue.Value{}, refuse(CodeInvalidTag, descriptor.FieldID, "CLI value has an unknown tag")
	}
}
