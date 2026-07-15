package reduce

import (
	"bytes"
	"regexp"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

var reducerTokenPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
var reducerLocusPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._:/\[\]-]{0,255}$`)

const maxMeasureDimensions = 32

// Measure is an adapter-defined, canonical, lexicographically ordered tuple.
// Its definition digest names both the dimensions and their ordering. It is a
// U5 reduction artifact and deliberately does not participate in a U3/U4
// stimulus identity.
type Measure struct {
	definitionDigest domain.Digest
	components       []uint64
	digest           domain.Digest
	canonicalBytes   []byte
}

type measureIdentity struct {
	SchemaVersion    string  `json:"schema_version"`
	Kind             string  `json:"kind"`
	DefinitionDigest string  `json:"definition_digest"`
	Components       []int64 `json:"components"`
}

func NewMeasure(definitionDigest domain.Digest, components []uint64) (Measure, error) {
	if !definitionDigest.Valid() || len(components) == 0 || len(components) > maxMeasureDimensions {
		return Measure{}, &domain.Error{Code: "INVALID_REDUCTION_MEASURE"}
	}
	identityComponents := make([]int64, len(components))
	for index, component := range components {
		if component > uint64(canon.MaxSafeInteger) {
			return Measure{}, &domain.Error{Code: "REDUCTION_MEASURE_OVERFLOW"}
		}
		identityComponents[index] = int64(component)
	}
	digest, canonicalBytes, err := digestTyped("ReductionMeasure", measureIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "ReductionMeasure",
		DefinitionDigest: definitionDigest.String(), Components: identityComponents,
	})
	if err != nil {
		return Measure{}, err
	}
	return Measure{
		definitionDigest: definitionDigest,
		components:       append([]uint64(nil), components...),
		digest:           digest,
		canonicalBytes:   append([]byte(nil), canonicalBytes...),
	}, nil
}

func (m Measure) Valid() bool {
	rebuilt, err := NewMeasure(m.definitionDigest, m.components)
	return err == nil && rebuilt.digest == m.digest && bytes.Equal(rebuilt.canonicalBytes, m.canonicalBytes)
}

func (m Measure) DefinitionDigest() domain.Digest { return m.definitionDigest }
func (m Measure) Components() []uint64            { return append([]uint64(nil), m.components...) }
func (m Measure) Digest() domain.Digest           { return m.digest }
func (m Measure) CanonicalBytes() []byte          { return append([]byte(nil), m.canonicalBytes...) }

// Compare returns -1, 0, or 1. Measures with different definitions are not
// comparable and return an error rather than an arbitrary ordering.
func (m Measure) Compare(other Measure) (int, error) {
	if !m.Valid() || !other.Valid() || m.definitionDigest != other.definitionDigest || len(m.components) != len(other.components) {
		return 0, &domain.Error{Code: "INCOMPARABLE_REDUCTION_MEASURE"}
	}
	for index := range m.components {
		if m.components[index] < other.components[index] {
			return -1, nil
		}
		if m.components[index] > other.components[index] {
			return 1, nil
		}
	}
	return 0, nil
}

func strictlyDecreases(before, after Measure) bool {
	comparison, err := after.Compare(before)
	return err == nil && comparison < 0
}

// ReducerRule is the canonical name/version identity of one typed transform.
type ReducerRule struct {
	name           string
	version        string
	digest         domain.Digest
	canonicalBytes []byte
}

type reducerRuleIdentity struct {
	SchemaVersion string `json:"schema_version"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Version       string `json:"version"`
}

func NewReducerRule(name, version string) (ReducerRule, error) {
	if !reducerTokenPattern.MatchString(name) || !reducerTokenPattern.MatchString(version) {
		return ReducerRule{}, &domain.Error{Code: "INVALID_REDUCER_RULE"}
	}
	digest, canonicalBytes, err := digestTyped("ReducerRule", reducerRuleIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "ReducerRule", Name: name, Version: version,
	})
	if err != nil {
		return ReducerRule{}, err
	}
	return ReducerRule{name: name, version: version, digest: digest, canonicalBytes: canonicalBytes}, nil
}

func (r ReducerRule) Valid() bool {
	rebuilt, err := NewReducerRule(r.name, r.version)
	return err == nil && rebuilt.digest == r.digest && bytes.Equal(rebuilt.canonicalBytes, r.canonicalBytes)
}
func (r ReducerRule) Name() string          { return r.name }
func (r ReducerRule) Version() string       { return r.version }
func (r ReducerRule) Digest() domain.Digest { return r.digest }

// ReducerSet binds the adapter domain, exact rule set, measure definition, and
// adapter policy scope. The policy fixes anchors, pins, execution shape, and
// transform classes; run/proposal identities separately bind current reducible
// values. Its digest is the quantifier in ONE_MINIMAL_UNDER, not a display label.
type ReducerSet struct {
	adapterDomain     string
	measureDefinition domain.Digest
	scopeDigest       domain.Digest
	rules             []ReducerRule
	digest            domain.Digest
	canonicalBytes    []byte
}

type reducerSetRuleIdentity struct {
	Priority int64  `json:"priority"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Digest   string `json:"digest"`
}

type reducerSetIdentity struct {
	SchemaVersion           string                   `json:"schema_version"`
	Kind                    string                   `json:"kind"`
	AdapterDomain           string                   `json:"adapter_domain"`
	MeasureDefinitionDigest string                   `json:"measure_definition_digest"`
	ScopeDigest             string                   `json:"scope_digest"`
	Rules                   []reducerSetRuleIdentity `json:"rules"`
}

func NewReducerSet(adapterDomain string, measureDefinition, scopeDigest domain.Digest, rules []ReducerRule) (ReducerSet, error) {
	if !reducerTokenPattern.MatchString(adapterDomain) || !measureDefinition.Valid() || !scopeDigest.Valid() || len(rules) == 0 || len(rules) > 64 {
		return ReducerSet{}, &domain.Error{Code: "INVALID_REDUCER_SET"}
	}
	// Input order is semantic rule priority, not incidental provider order. The
	// adapter policy constructs this list canonically, and the reducer-set digest
	// binds every priority explicitly so the engine cannot silently substitute a
	// lexical rule ordering.
	normalized := append([]ReducerRule(nil), rules...)
	identities := make([]reducerSetRuleIdentity, len(normalized))
	seen := make(map[domain.Digest]struct{}, len(normalized))
	for index, rule := range normalized {
		if !rule.Valid() {
			return ReducerSet{}, &domain.Error{Code: "INVALID_REDUCER_SET_RULES"}
		}
		if _, duplicate := seen[rule.digest]; duplicate {
			return ReducerSet{}, &domain.Error{Code: "INVALID_REDUCER_SET_RULES"}
		}
		seen[rule.digest] = struct{}{}
		identities[index] = reducerSetRuleIdentity{
			Priority: int64(index), Name: rule.name, Version: rule.version, Digest: rule.digest.String(),
		}
	}
	digest, canonicalBytes, err := digestTyped("ReducerSet", reducerSetIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "ReducerSet", AdapterDomain: adapterDomain,
		MeasureDefinitionDigest: measureDefinition.String(), ScopeDigest: scopeDigest.String(), Rules: identities,
	})
	if err != nil {
		return ReducerSet{}, err
	}
	return ReducerSet{
		adapterDomain: adapterDomain, measureDefinition: measureDefinition, scopeDigest: scopeDigest,
		rules: normalized, digest: digest, canonicalBytes: canonicalBytes,
	}, nil
}

func (s ReducerSet) Valid() bool {
	rebuilt, err := NewReducerSet(s.adapterDomain, s.measureDefinition, s.scopeDigest, s.rules)
	return err == nil && rebuilt.digest == s.digest && bytes.Equal(rebuilt.canonicalBytes, s.canonicalBytes)
}
func (s ReducerSet) AdapterDomain() string                  { return s.adapterDomain }
func (s ReducerSet) MeasureDefinitionDigest() domain.Digest { return s.measureDefinition }
func (s ReducerSet) ScopeDigest() domain.Digest             { return s.scopeDigest }
func (s ReducerSet) Rules() []ReducerRule                   { return append([]ReducerRule(nil), s.rules...) }
func (s ReducerSet) Digest() domain.Digest                  { return s.digest }
func (s ReducerSet) CanonicalBytes() []byte                 { return append([]byte(nil), s.canonicalBytes...) }

func (s ReducerSet) Contains(rule ReducerRule) bool {
	_, present := s.rulePriority(rule)
	return present
}

func (s ReducerSet) rulePriority(rule ReducerRule) (int, bool) {
	if !s.Valid() || !rule.Valid() {
		return 0, false
	}
	for priority, candidate := range s.rules {
		if candidate.digest == rule.digest {
			return priority, true
		}
	}
	return 0, false
}

func digestTyped(kind string, value any) (domain.Digest, []byte, error) {
	digest, canonicalBytes, err := canon.DigestTyped(kind, value)
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return "", nil, err
	}
	return parsed, canonicalBytes, nil
}
