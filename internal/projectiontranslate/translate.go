// Package projectiontranslate is Countershape's sole generic-to-adapter
// branch. It resolves exact adapter-owned projection bindings and strictly
// interprets historical projection bytes without rewriting those bytes.
package projectiontranslate

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectionprofile"
)

type Code string

const (
	CodeInvalidBinding    Code = "INVALID_PROJECTION_BINDING"
	CodeProfileNotFound   Code = "EXACT_PROFILE_NOT_FOUND"
	CodeProfileAmbiguous  Code = "EXACT_PROFILE_AMBIGUOUS"
	CodeProfileMismatch   Code = "RESOLVED_PROFILE_MISMATCH"
	CodeInvalidProjection Code = "INVALID_PROJECTION_WIRE"
	CodeNoncanonical      Code = "NONCANONICAL_PROJECTION_WIRE"
	CodeRosterMismatch    Code = "PROJECTION_FIELD_ROSTER_MISMATCH"
	CodeInvalidTag        Code = "INVALID_PROJECTION_VALUE_TAG"
	CodeInvalidPayload    Code = "INVALID_PROJECTION_VALUE_PAYLOAD"
	CodeTranslationLimit  Code = "PORTABLE_TRANSLATION_LIMIT_EXCEEDED"
)

type Error struct {
	Code   Code
	Field  string
	Detail string
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	result := string(e.Code)
	if e.Field != "" {
		result += ": field=" + fmt.Sprintf("%q", e.Field)
	}
	if e.Detail != "" {
		result += ": " + e.Detail
	}
	return result
}

func IsCode(err error, code Code) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func refuse(code Code, field, detail string) *Error {
	return &Error{Code: code, Field: field, Detail: detail}
}

type adapterArm uint8

const (
	armCLI adapterArm = iota + 1
	armHTTP
)

// Resolved is an opaque exact-binding capability. Its zero value is invalid;
// callers cannot pair a Profile with an adapter arm.
type Resolved struct {
	profile      projectionprofile.Profile
	arm          adapterArm
	stimulusKind string
	seal         *resolvedSeal
}

type resolvedSeal struct{}

var resolvedAuthority = &resolvedSeal{}

// Field is one immutable member of a translated complete tuple.
type Field struct {
	id    string
	value portablevalue.Value
}

func (f Field) ID() string                 { return f.id }
func (f Field) Value() portablevalue.Value { return f.value }

// Tuple is a complete profile-ordered interpretation of exact historical
// projection bytes. It deliberately contains no replacement wire/fingerprint.
type Tuple struct {
	profileDigest domain.Digest
	fields        []Field
}

func (t Tuple) ProfileDigest() domain.Digest { return t.profileDigest }

func (t Tuple) Fields() []Field { return append([]Field(nil), t.fields...) }

func (t Tuple) Value(fieldID string) (portablevalue.Value, bool) {
	for _, field := range t.fields {
		if field.id == fieldID {
			return field.value, true
		}
	}
	return portablevalue.Value{}, false
}

func (t Tuple) Valid() bool {
	if !t.profileDigest.Valid() || len(t.fields) == 0 {
		return false
	}
	values := make([]portablevalue.Value, len(t.fields))
	seen := make(map[string]struct{}, len(t.fields))
	for index, field := range t.fields {
		if field.id == "" || !field.value.Valid() {
			return false
		}
		if _, duplicate := seen[field.id]; duplicate {
			return false
		}
		seen[field.id] = struct{}{}
		values[index] = field.value
	}
	return portablevalue.ValidateTuple(values) == nil
}

func (t Tuple) Equal(other Tuple) bool {
	if !t.Valid() || !other.Valid() || t.profileDigest != other.profileDigest || len(t.fields) != len(other.fields) {
		return false
	}
	for index := range t.fields {
		if t.fields[index].id != other.fields[index].id || !t.fields[index].value.Equal(other.fields[index].value) {
			return false
		}
	}
	return true
}

func (t Tuple) RetainedBytes() int {
	retained := 0
	for _, field := range t.fields {
		retained += len(field.id) + field.value.RetainedBytes()
	}
	return retained
}

func (t Tuple) CompatibilityEncodedBytes() int {
	encoded := 0
	for _, field := range t.fields {
		encoded += len(field.id) + portablevalue.CompatibilityFieldOverheadBytes + field.value.CompatibilityEncodedBytes()
	}
	return encoded
}

func newTuple(profile projectionprofile.Profile, fields []Field) (Tuple, error) {
	profileFields := profile.Fields()
	if !profile.Valid() || len(fields) != len(profileFields) {
		return Tuple{}, refuse(CodeRosterMismatch, "", "translated tuple does not cover the exact profile roster")
	}
	values := make([]portablevalue.Value, len(fields))
	copyFields := make([]Field, len(fields))
	for index, field := range fields {
		descriptor := profileFields[index]
		if field.id != descriptor.FieldID || !field.value.Valid() {
			return Tuple{}, refuse(CodeRosterMismatch, field.id, "translated field differs from the exact profile order or type")
		}
		switch field.value.Tag() {
		case portablevalue.TagMissing:
			if !descriptor.AllowMissing {
				return Tuple{}, refuse(CodeInvalidTag, field.id, "translated missing value is not permitted by the profile")
			}
		case portablevalue.TagNull:
			if !descriptor.AllowNull {
				return Tuple{}, refuse(CodeInvalidTag, field.id, "translated null value is not permitted by the profile")
			}
		default:
			if field.value.Tag() != descriptor.PortableTag {
				return Tuple{}, refuse(CodeInvalidTag, field.id, "translated present tag differs from the profile")
			}
		}
		copyFields[index] = field
		values[index] = field.value
	}
	if err := portablevalue.ValidateTuple(values); err != nil {
		return Tuple{}, refuse(CodeTranslationLimit, "", err.Error())
	}
	return Tuple{profileDigest: profile.Digest(), fields: copyFields}, nil
}

// Resolve requires the complete exact binding and uses adapter-owned
// constructors. No domain label, registry digest, or operation name is enough.
func Resolve(binding domain.ProjectionDefinitionBinding) (Resolved, error) {
	if !binding.Valid() {
		return Resolved{}, refuse(CodeInvalidBinding, "", "projection binding is invalid")
	}
	switch binding.AdapterDomain() {
	case domain.AdapterCLI:
		return resolveCLI(binding)
	case domain.AdapterHTTP:
		return resolveHTTP(binding)
	default:
		return Resolved{}, refuse(CodeInvalidBinding, "", "projection adapter domain is unsupported")
	}
}

func (r Resolved) Valid() bool {
	if r.seal != resolvedAuthority || !r.profile.Valid() || (r.arm != armCLI && r.arm != armHTTP) {
		return false
	}
	return (r.arm == armCLI && r.profile.AdapterDomain() == domain.AdapterCLI && r.stimulusKind == "CLIStimulus") ||
		(r.arm == armHTTP && r.profile.AdapterDomain() == domain.AdapterHTTP && r.stimulusKind == "HTTPStimulus")
}

func (r Resolved) Profile() projectionprofile.Profile { return r.profile }

func (r Resolved) ExpectedStimulusKind() string {
	if !r.Valid() {
		return ""
	}
	return r.stimulusKind
}

// Translate strictly interprets exact historical bytes under this resolved
// arm. It never canonicalizes nonexact input into acceptance.
func (r Resolved) Translate(exact []byte) (Tuple, error) {
	if !r.Valid() {
		return Tuple{}, refuse(CodeProfileMismatch, "", "resolved profile is invalid")
	}
	switch r.arm {
	case armCLI:
		return r.translateCLI(exact)
	case armHTTP:
		return r.translateHTTP(exact)
	default:
		return Tuple{}, refuse(CodeProfileMismatch, "", "resolved adapter arm is unknown")
	}
}

// StrictTranslate treats Profile as inert identity: it independently resolves
// the retained binding and demands exact profile identity before translation.
func StrictTranslate(profile projectionprofile.Profile, exact []byte) (Tuple, error) {
	if !profile.Valid() {
		return Tuple{}, refuse(CodeProfileMismatch, "", "portable profile is invalid")
	}
	resolved, err := Resolve(profile.Binding())
	if err != nil {
		return Tuple{}, err
	}
	if resolved.profile.Digest() != profile.Digest() || !bytes.Equal(resolved.profile.CanonicalBytes(), profile.CanonicalBytes()) {
		return Tuple{}, refuse(CodeProfileMismatch, "", "caller profile is not the uniquely resolved adapter profile")
	}
	return resolved.Translate(exact)
}

func exactBindingMatch(candidate, requested domain.ProjectionDefinitionBinding) bool {
	return candidate.Valid() && requested.Valid() && exactIdentityMatch(
		candidate.Digest(), candidate.CanonicalBytes(), requested.Digest(), requested.CanonicalBytes(),
	)
}

func exactIdentityMatch(candidateDigest domain.Digest, candidateBytes []byte, requestedDigest domain.Digest, requestedBytes []byte) bool {
	return candidateDigest.Valid() && requestedDigest.Valid() && candidateDigest == requestedDigest && bytes.Equal(candidateBytes, requestedBytes)
}

func newResolvedProfile(name, version string, binding domain.ProjectionDefinitionBinding, fields []projectionprofile.Descriptor, arm adapterArm, stimulusKind string) (Resolved, error) {
	profile, err := projectionprofile.NewDerived(projectionprofile.DerivedConfig{
		TranslatorName: name, TranslatorVersion: version, Binding: binding, Fields: fields,
	})
	if err != nil {
		return Resolved{}, refuse(CodeInvalidBinding, "", err.Error())
	}
	return Resolved{profile: profile, arm: arm, stimulusKind: stimulusKind, seal: resolvedAuthority}, nil
}

func parseExact(exact []byte) (canon.Value, error) {
	if len(exact) == 0 || len(exact) > canon.MaxInputBytes {
		return canon.Value{}, refuse(CodeTranslationLimit, "", "projection bytes are empty or exceed the canonical ceiling")
	}
	value, err := canon.Parse(exact)
	if err != nil {
		return canon.Value{}, refuse(CodeInvalidProjection, "", "projection bytes do not parse under the canonical profile")
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		return canon.Value{}, refuse(CodeNoncanonical, "", "projection bytes are not exact canonical bytes")
	}
	return value, nil
}

func exactObject(value canon.Value, names ...string) ([]canon.Value, error) {
	members, object := value.Members()
	if !object || len(members) != len(names) {
		return nil, refuse(CodeInvalidProjection, "", "object member roster differs")
	}
	values := make([]canon.Value, len(names))
	for index, name := range names {
		if members[index].Name != name {
			return nil, refuse(CodeInvalidProjection, "", "object member name or canonical order differs")
		}
		values[index] = members[index].Value
	}
	return values, nil
}

func exactText(value canon.Value) (string, error) {
	text, ok := value.Text()
	if !ok {
		return "", refuse(CodeInvalidPayload, "", "value is not a string")
	}
	return text, nil
}

func exactArray(value canon.Value) ([]canon.Value, error) {
	elements, ok := value.Elements()
	if !ok {
		return nil, refuse(CodeInvalidPayload, "", "value is not an array")
	}
	return elements, nil
}
