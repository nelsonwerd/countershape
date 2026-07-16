// Package projectionprofile defines the exact, ordered downstream
// interpretation identity for one adapter-owned projection binding. It does
// not resolve an adapter and is not translation authority by itself; only
// projectiontranslate may issue a resolved use of a Profile.
package projectionprofile

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

type Code string

const (
	CodeInvalidProfile     Code = "INVALID_PROFILE"
	CodeInvalidDescriptor  Code = "INVALID_DESCRIPTOR"
	CodeDuplicateField     Code = "DUPLICATE_FIELD"
	CodeUnsupportedMapping Code = "UNSUPPORTED_PORTABLE_MAPPING"
	CodeLimitExceeded      Code = "LIMIT_EXCEEDED"
)

const (
	maxFields             = 64
	maxNameBytes          = 128
	maxPathDepth          = 8
	maxSourceKindBytes    = 64
	maxMissingPolicyBytes = 128
	maxDescriptorBytes    = 256 * 1024
)

type Error struct {
	Code   Code
	Detail string
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Detail == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Detail
}

func IsCode(err error, code Code) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func refuse(code Code, detail string) *Error { return &Error{Code: code, Detail: detail} }

// Descriptor is the exact adapter-owned source declaration paired with its
// closed portable interpretation. SourceKind and MissingPolicy deliberately
// retain the adapter's historical spellings.
type Descriptor struct {
	FieldID       string
	Channel       string
	SourcePath    []string
	SourceKind    string
	MissingPolicy string
	PortableTag   portablevalue.Tag
	AllowMissing  bool
	AllowNull     bool
}

// DerivedConfig is accepted only at the unique projectiontranslate call site,
// a restriction enforced by Countershape's architecture checker. A Profile is
// structural identity; projectiontranslate.Resolved is the use authority.
type DerivedConfig struct {
	TranslatorName    string
	TranslatorVersion string
	Binding           domain.ProjectionDefinitionBinding
	Fields            []Descriptor
}

type Profile struct {
	digest            domain.Digest
	canonical         []byte
	translatorName    string
	translatorVersion string
	binding           domain.ProjectionDefinitionBinding
	fields            []Descriptor
}

type descriptorIdentity struct {
	FieldID       string   `json:"field_id"`
	Channel       string   `json:"channel"`
	SourcePath    []string `json:"source_path"`
	SourceKind    string   `json:"source_value_kind"`
	MissingPolicy string   `json:"missing_policy"`
	PresentTag    string   `json:"present_tag"`
	AllowMissing  bool     `json:"allow_missing"`
	AllowNull     bool     `json:"allow_null"`
}

type profileIdentity struct {
	SchemaVersion           string               `json:"schema_version"`
	Kind                    string               `json:"kind"`
	TranslatorName          string               `json:"translator_name"`
	TranslatorVersion       string               `json:"translator_version"`
	AdapterDomain           string               `json:"adapter_domain"`
	ProjectionBindingDigest string               `json:"projection_definition_binding_digest"`
	ProjectionBindingBase64 string               `json:"projection_definition_binding_base64"`
	Fields                  []descriptorIdentity `json:"fields"`
}

// NewDerived validates and derives one immutable profile. It never resolves a
// binding against adapter constructors; that stronger operation belongs only
// to projectiontranslate.
func NewDerived(config DerivedConfig) (Profile, error) {
	if !boundedText(config.TranslatorName, maxNameBytes) || !boundedText(config.TranslatorVersion, maxNameBytes) || !config.Binding.Valid() {
		return Profile{}, refuse(CodeInvalidProfile, "translator identity and exact projection binding are required")
	}
	if len(config.Fields) == 0 || len(config.Fields) > maxFields {
		return Profile{}, refuse(CodeLimitExceeded, "profile field roster is empty or exceeds the field ceiling")
	}
	fields := make([]Descriptor, len(config.Fields))
	identities := make([]descriptorIdentity, len(config.Fields))
	seen := make(map[string]struct{}, len(config.Fields))
	seenSources := make(map[string]struct{}, len(config.Fields))
	acceptedChannels := make(map[string]struct{}, len(config.Binding.AcceptedChannels()))
	for _, channel := range config.Binding.AcceptedChannels() {
		acceptedChannels[channel] = struct{}{}
	}
	retained := 0
	for index, raw := range config.Fields {
		field := raw
		if !boundedText(field.FieldID, maxNameBytes) || !boundedText(field.Channel, maxNameBytes) ||
			!boundedText(field.SourceKind, maxSourceKindBytes) || !boundedText(field.MissingPolicy, maxMissingPolicyBytes) ||
			len(field.SourcePath) == 0 || len(field.SourcePath) > maxPathDepth {
			return Profile{}, refuse(CodeInvalidDescriptor, fmt.Sprintf("profile field %d is incomplete or outside its byte ceilings", index))
		}
		if _, duplicate := seen[field.FieldID]; duplicate {
			return Profile{}, refuse(CodeDuplicateField, "profile field ID occurs more than once")
		}
		seen[field.FieldID] = struct{}{}
		if _, accepted := acceptedChannels[field.Channel]; !accepted {
			return Profile{}, refuse(CodeInvalidDescriptor, "profile field channel is not admitted by the exact projection binding")
		}
		field.SourcePath = append([]string(nil), field.SourcePath...)
		sourceKey := field.Channel + "\x00"
		for _, segment := range field.SourcePath {
			if !boundedText(segment, maxNameBytes) {
				return Profile{}, refuse(CodeInvalidDescriptor, "profile source path has an invalid segment")
			}
			sourceKey += fmt.Sprintf("%d:%s", len(segment), segment)
		}
		if _, duplicate := seenSources[sourceKey]; duplicate {
			return Profile{}, refuse(CodeDuplicateField, "profile descriptors alias one adapter source path")
		}
		seenSources[sourceKey] = struct{}{}
		if !validMapping(field.SourceKind, field.PortableTag) {
			return Profile{}, refuse(CodeUnsupportedMapping, "adapter source kind and portable tag disagree")
		}
		allowsMissing, recognizedPolicy := missingPolicy(field.MissingPolicy)
		if !recognizedPolicy || field.AllowMissing != allowsMissing || field.AllowNull {
			return Profile{}, refuse(CodeUnsupportedMapping, "normalized missing/null permissions disagree with the closed adapter policy")
		}
		retained += len(field.FieldID) + len(field.Channel) + len(field.SourceKind) + len(field.MissingPolicy)
		for _, segment := range field.SourcePath {
			retained += len(segment)
		}
		if retained > maxDescriptorBytes {
			return Profile{}, refuse(CodeLimitExceeded, "profile descriptors exceed the aggregate retained-byte ceiling")
		}
		fields[index] = field
		identities[index] = descriptorIdentity{
			FieldID: field.FieldID, Channel: field.Channel, SourcePath: append([]string(nil), field.SourcePath...),
			SourceKind: field.SourceKind, MissingPolicy: field.MissingPolicy, PresentTag: string(field.PortableTag),
			AllowMissing: field.AllowMissing, AllowNull: field.AllowNull,
		}
	}
	identity := profileIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "PortableProjectionProfile",
		TranslatorName: config.TranslatorName, TranslatorVersion: config.TranslatorVersion,
		AdapterDomain:           string(config.Binding.AdapterDomain()),
		ProjectionBindingDigest: config.Binding.Digest().String(),
		ProjectionBindingBase64: base64.StdEncoding.EncodeToString(config.Binding.CanonicalBytes()),
		Fields:                  identities,
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("PortableProjectionProfile", identity)
	if err != nil || len(canonicalBytes) > canon.MaxInputBytes {
		return Profile{}, refuse(CodeLimitExceeded, "profile identity exceeds the canonical object ceiling")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return Profile{}, refuse(CodeInvalidProfile, "profile digest could not be reconstructed")
	}
	return Profile{
		digest: digest, canonical: canonicalBytes, translatorName: config.TranslatorName,
		translatorVersion: config.TranslatorVersion, binding: config.Binding, fields: fields,
	}, nil
}

// Parse reconstructs one exact profile under the caller's already-retained
// projection binding. It never resolves adapter semantics: callers that need
// that stronger guarantee must still pair the result with adapter-owned
// authority. Unknown members, alternate base64 spellings, normalized aliases,
// and noncanonical JSON all fail the final byte-for-byte regeneration check.
func Parse(exact []byte, expectedBinding domain.ProjectionDefinitionBinding) (Profile, error) {
	if len(exact) == 0 || len(exact) > canon.MaxInputBytes || !expectedBinding.Valid() {
		return Profile{}, refuse(CodeInvalidProfile, "exact profile bytes and projection binding are required")
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return Profile{}, refuse(CodeInvalidProfile, "profile bytes are not strict canonical JSON")
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		return Profile{}, refuse(CodeInvalidProfile, "profile bytes are noncanonical")
	}
	var identity profileIdentity
	if err := json.Unmarshal(exact, &identity); err != nil {
		return Profile{}, refuse(CodeInvalidProfile, "profile typed decode failed")
	}
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "PortableProjectionProfile" ||
		identity.AdapterDomain != string(expectedBinding.AdapterDomain()) ||
		identity.ProjectionBindingDigest != expectedBinding.Digest().String() {
		return Profile{}, refuse(CodeInvalidProfile, "profile identity differs from the expected binding")
	}
	bindingBytes, err := base64.StdEncoding.Strict().DecodeString(identity.ProjectionBindingBase64)
	if err != nil || base64.StdEncoding.EncodeToString(bindingBytes) != identity.ProjectionBindingBase64 ||
		!bytes.Equal(bindingBytes, expectedBinding.CanonicalBytes()) {
		return Profile{}, refuse(CodeInvalidProfile, "profile binding bytes differ from the expected binding")
	}
	parsedBinding, err := domain.ParseProjectionDefinitionBinding(bindingBytes)
	if err != nil || parsedBinding.Digest() != expectedBinding.Digest() ||
		!bytes.Equal(parsedBinding.CanonicalBytes(), expectedBinding.CanonicalBytes()) {
		return Profile{}, refuse(CodeInvalidProfile, "profile binding did not reconstruct exactly")
	}
	descriptors := make([]Descriptor, len(identity.Fields))
	for index, field := range identity.Fields {
		descriptors[index] = Descriptor{
			FieldID: field.FieldID, Channel: field.Channel, SourcePath: append([]string(nil), field.SourcePath...),
			SourceKind: field.SourceKind, MissingPolicy: field.MissingPolicy,
			PortableTag: portablevalue.Tag(field.PresentTag), AllowMissing: field.AllowMissing, AllowNull: field.AllowNull,
		}
	}
	rebuilt, err := NewDerived(DerivedConfig{
		TranslatorName: identity.TranslatorName, TranslatorVersion: identity.TranslatorVersion,
		Binding: parsedBinding, Fields: descriptors,
	})
	if err != nil || !bytes.Equal(rebuilt.CanonicalBytes(), exact) {
		return Profile{}, refuse(CodeInvalidProfile, "profile did not regenerate byte-exactly")
	}
	return rebuilt, nil
}

func validMapping(source string, tag portablevalue.Tag) bool {
	switch source {
	case "UTF8_STRING":
		return tag == portablevalue.TagString
	case "SAFE_INTEGER":
		return tag == portablevalue.TagInteger
	case "BYTES":
		return tag == portablevalue.TagBytes
	case "ORDERED_STRING_LIST":
		return tag == portablevalue.TagOrderedStringList
	case "CANONICAL_JSON_OBJECT":
		return tag == portablevalue.TagCanonicalJSON
	default:
		return false
	}
}

func missingPolicy(policy string) (bool, bool) {
	switch policy {
	case "TAGGED_MISSING", "TAGGED_MISSING_FOR_SIGNAL", "TAGGED_MISSING_FOR_EXIT":
		return true, true
	case "REJECT_CAPTURE", "REJECT_CHANNEL", "REJECT_BODY":
		return false, true
	default:
		return false, false
	}
}

func boundedText(value string, limit int) bool {
	return value != "" && len(value) <= limit && utf8.ValidString(value)
}

func (p Profile) Valid() bool {
	if !p.digest.Valid() || len(p.canonical) == 0 || !p.binding.Valid() {
		return false
	}
	rebuilt, err := NewDerived(DerivedConfig{
		TranslatorName: p.translatorName, TranslatorVersion: p.translatorVersion,
		Binding: p.binding, Fields: p.fields,
	})
	return err == nil && rebuilt.digest == p.digest && bytes.Equal(rebuilt.canonical, p.canonical)
}

func (p Profile) Digest() domain.Digest { return p.digest }

func (p Profile) CanonicalBytes() []byte { return append([]byte(nil), p.canonical...) }

func (p Profile) TranslatorName() string    { return p.translatorName }
func (p Profile) TranslatorVersion() string { return p.translatorVersion }
func (p Profile) AdapterDomain() domain.AdapterDomain {
	return p.binding.AdapterDomain()
}
func (p Profile) Binding() domain.ProjectionDefinitionBinding { return p.binding }

func (p Profile) Fields() []Descriptor {
	if p.fields == nil {
		return nil
	}
	result := make([]Descriptor, len(p.fields))
	for index, field := range p.fields {
		result[index] = field
		result[index].SourcePath = append([]string(nil), field.SourcePath...)
	}
	return result
}
