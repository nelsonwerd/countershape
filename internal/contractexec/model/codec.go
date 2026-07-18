package model

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

var (
	studyIDPattern = regexp.MustCompile(`^study:[0-9a-f]{64}$`)
	noncePattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	sha1OIDPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha2OIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	versionPattern = regexp.MustCompile(`^v?[1-9][0-9]*\.[0-9]+(?:\.[0-9]+)?(?:[-+][0-9A-Za-z.-]+)?$`)
	modePattern    = regexp.MustCompile(`^100(?:[1357][0-7]{2}|[0-7][1357][0-7]|[0-7]{2}[1357])$`)
)

func canonicalObject(kind string, wire map[string]any) ([]byte, domain.Digest, error) {
	exact, err := canon.CanonicalizeTyped(wire)
	if err != nil {
		return nil, "", refuse(CodeInvalidWitness, "body could not be canonicalized", err)
	}
	if len(exact) == 0 || len(exact) > canon.MaxInputBytes {
		return nil, "", refuse(CodeLimitExceeded, "canonical body exceeds the object ceiling", nil)
	}
	digest, err := canon.DigestBytes(kind, exact)
	if err != nil {
		return nil, "", refuse(CodeInvalidWitness, "typed digest could not be derived", err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return nil, "", refuse(CodeInvalidWitness, "typed digest is outside the domain profile", err)
	}
	return exact, parsed, nil
}

func parseExactObject(exact []byte, kind string, expected domain.Digest, invalidCode, digestCode string) (canon.Value, error) {
	if len(exact) == 0 || len(exact) > canon.MaxInputBytes || !expected.Valid() {
		return canon.Value{}, refuse(invalidCode, "body or expected digest is invalid", nil)
	}
	root, err := canon.Parse(exact)
	if err != nil {
		return canon.Value{}, refuse(invalidCode, "body is not strict JSON", err)
	}
	canonical, err := root.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, exact) {
		return canon.Value{}, refuse(invalidCode, "body is not exact canonical JSON", err)
	}
	actual, err := canon.DigestBytes(kind, exact)
	if err != nil || actual.String() != expected.String() {
		return canon.Value{}, refuse(digestCode, "expected typed digest differs from the exact body", err)
	}
	return root, nil
}

func requireRoster(value canon.Value, names ...string) error {
	members, ok := value.Members()
	if !ok || len(members) != len(names) {
		return refuse(CodeInvalidWitness, "object member roster differs", nil)
	}
	expected := append([]string(nil), names...)
	sort.Strings(expected)
	for index, member := range members {
		if member.Name != expected[index] {
			return refuse(CodeInvalidWitness, "object member roster differs", nil)
		}
	}
	return nil
}

func member(value canon.Value, name string) (canon.Value, error) {
	result, ok := value.LookupMember(name)
	if !ok {
		return canon.Value{}, refuse(CodeInvalidWitness, fmt.Sprintf("missing %s", name), nil)
	}
	return result, nil
}

func textMember(value canon.Value, name string) (string, error) {
	entry, err := member(value, name)
	if err != nil {
		return "", err
	}
	text, ok := entry.Text()
	if !ok {
		return "", refuse(CodeInvalidWitness, fmt.Sprintf("%s is not text", name), nil)
	}
	return text, nil
}

func integerMember(value canon.Value, name string) (int64, error) {
	entry, err := member(value, name)
	if err != nil {
		return 0, err
	}
	integer, ok := entry.Int64()
	if !ok {
		return 0, refuse(CodeInvalidWitness, fmt.Sprintf("%s is not an exact integer", name), nil)
	}
	return integer, nil
}

func boolMember(value canon.Value, name string) (bool, error) {
	entry, err := member(value, name)
	if err != nil {
		return false, err
	}
	boolean, ok := entry.Boolean()
	if !ok {
		return false, refuse(CodeInvalidWitness, fmt.Sprintf("%s is not a Boolean", name), nil)
	}
	return boolean, nil
}

func digestMember(value canon.Value, name string) (domain.Digest, error) {
	text, err := textMember(value, name)
	if err != nil {
		return "", err
	}
	digest, err := domain.ParseDigest(text)
	if err != nil {
		return "", refuse(CodeInvalidWitness, fmt.Sprintf("%s is not a digest", name), err)
	}
	return digest, nil
}

func arrayMember(value canon.Value, name string) ([]canon.Value, error) {
	entry, err := member(value, name)
	if err != nil {
		return nil, err
	}
	items, ok := entry.Elements()
	if !ok {
		return nil, refuse(CodeInvalidWitness, fmt.Sprintf("%s is not an array", name), nil)
	}
	return items, nil
}

func validBoundedText(value string, maximum int) bool {
	if value == "" || len(value) > maximum || !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validAbsoluteExecutablePath(value string) bool {
	if !validBoundedText(value, 4096) || !strings.HasPrefix(value, "/") || strings.Contains(value, "//") {
		return false
	}
	for _, component := range strings.Split(value, "/")[1:] {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	return true
}

func cloneBytes(input []byte) []byte { return append([]byte(nil), input...) }

func cloneDigests(input []domain.Digest) []domain.Digest {
	return append([]domain.Digest(nil), input...)
}

func plainValue(value canon.Value) (any, error) {
	switch value.Kind() {
	case canon.KindNull:
		return nil, nil
	case canon.KindBoolean:
		boolean, _ := value.Boolean()
		return boolean, nil
	case canon.KindInteger:
		integer, _ := value.Int64()
		return integer, nil
	case canon.KindString:
		text, _ := value.Text()
		return text, nil
	case canon.KindArray:
		entries, _ := value.Elements()
		result := make([]any, len(entries))
		for index, entry := range entries {
			converted, err := plainValue(entry)
			if err != nil {
				return nil, err
			}
			result[index] = converted
		}
		return result, nil
	case canon.KindObject:
		members, _ := value.Members()
		result := make(map[string]any, len(members))
		for _, entry := range members {
			converted, err := plainValue(entry.Value)
			if err != nil {
				return nil, err
			}
			result[entry.Name] = converted
		}
		return result, nil
	default:
		return nil, refuse(CodeInvalidWitness, "canonical value has an impossible kind", nil)
	}
}
