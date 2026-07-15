package domain

import (
	"bytes"
	"regexp"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
)

var (
	digestPattern       = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	candidateKeyPattern = regexp.MustCompile(`^candidate:[0-9a-f]{64}$`)
)

// Digest is an already-established semantic or byte digest. Parsing validates
// its closed textual profile but does not reinterpret what the digest proves.
type Digest string

func ParseDigest(raw string) (Digest, error) {
	if !utf8.ValidString(raw) || !digestPattern.MatchString(raw) {
		return "", refuse(ErrInvalidDigest, raw)
	}
	return Digest(raw), nil
}

func MustDigest(raw string) Digest {
	digest, err := ParseDigest(raw)
	if err != nil {
		panic(err)
	}
	return digest
}

func (d Digest) String() string { return string(d) }

func (d Digest) Valid() bool {
	return utf8.ValidString(string(d)) && digestPattern.MatchString(string(d))
}

// CandidateExecutionKey binds semantic execution identity. Display labels,
// producer names, branch spelling, candidate order, stimulus identity, and
// support counts are intentionally absent. The wire form is deliberately one
// exact profile: candidate: followed by a lowercase SHA-256 hex payload.
type CandidateExecutionKey struct{ text string }

type CandidateExecutionIdentity struct {
	TreeIdentityDigest          Digest `json:"tree_identity_digest"`
	MaterializationPolicyDigest Digest `json:"materialization_policy_digest"`
	WorldPlanDigest             Digest `json:"world_plan_digest"`
	AdapterDigest               Digest `json:"adapter_digest"`
	RunnerDigest                Digest `json:"runner_digest"`
	ProjectionDefinitionDigest  Digest `json:"projection_definition_digest"`
}

// CandidateExecutionBinding retains the exact semantic identity whose digest is
// exposed as a CandidateExecutionKey. A bare key parsed from wire text remains a
// reference only; execution evidence can be allocated only from this binding.
// The binding proves structural agreement among declarations, not that a host
// actually materialized or executed them.
type CandidateExecutionBinding struct {
	key            CandidateExecutionKey
	identity       CandidateExecutionIdentity
	canonicalBytes []byte
}

func NewCandidateExecutionBinding(identity CandidateExecutionIdentity) (CandidateExecutionBinding, error) {
	for _, field := range []struct {
		name   string
		digest Digest
	}{
		{"tree_identity_digest", identity.TreeIdentityDigest},
		{"materialization_policy_digest", identity.MaterializationPolicyDigest},
		{"world_plan_digest", identity.WorldPlanDigest},
		{"adapter_digest", identity.AdapterDigest},
		{"runner_digest", identity.RunnerDigest},
		{"projection_definition_digest", identity.ProjectionDefinitionDigest},
	} {
		if !field.digest.Valid() {
			return CandidateExecutionBinding{}, refuse(ErrInvalidDigest, field.name)
		}
	}
	digest, canonicalBytes, err := digestTyped("CandidateExecutionKey", identity)
	if err != nil {
		return CandidateExecutionBinding{}, err
	}
	return CandidateExecutionBinding{
		key:            CandidateExecutionKey{text: "candidate:" + digest.String()[len("sha256:"):]},
		identity:       identity,
		canonicalBytes: append([]byte(nil), canonicalBytes...),
	}, nil
}

func NewCandidateExecutionKey(identity CandidateExecutionIdentity) (CandidateExecutionKey, error) {
	binding, err := NewCandidateExecutionBinding(identity)
	if err != nil {
		return CandidateExecutionKey{}, err
	}
	return binding.Key(), nil
}

func (b CandidateExecutionBinding) Valid() bool {
	if !b.key.Valid() || len(b.canonicalBytes) == 0 {
		return false
	}
	rebuilt, err := NewCandidateExecutionBinding(b.identity)
	return err == nil && rebuilt.key == b.key && bytes.Equal(rebuilt.canonicalBytes, b.canonicalBytes)
}

func (b CandidateExecutionBinding) Key() CandidateExecutionKey { return b.key }

func (b CandidateExecutionBinding) Identity() CandidateExecutionIdentity { return b.identity }

func (b CandidateExecutionBinding) CanonicalBytes() []byte {
	return append([]byte(nil), b.canonicalBytes...)
}

func ParseCandidateExecutionKey(raw string) (CandidateExecutionKey, error) {
	if !utf8.ValidString(raw) || !candidateKeyPattern.MatchString(raw) {
		return CandidateExecutionKey{}, refuse(ErrInvalidCandidateKey, raw)
	}
	return CandidateExecutionKey{text: raw}, nil
}

func (k CandidateExecutionKey) String() string { return k.text }

func (k CandidateExecutionKey) Valid() bool {
	return utf8.ValidString(k.text) && candidateKeyPattern.MatchString(k.text)
}

// ProjectionFingerprint hashes canonical projection bytes. Supplying merely
// valid JSON is insufficient: callers must provide the exact canonical form.
type ProjectionFingerprint struct{ digest Digest }

func NewProjectionFingerprint(canonicalBytes []byte) (ProjectionFingerprint, error) {
	normalized, err := canon.Canonicalize(canonicalBytes)
	if err != nil {
		return ProjectionFingerprint{}, err
	}
	if !bytes.Equal(normalized, canonicalBytes) {
		return ProjectionFingerprint{}, refuse("NONCANONICAL_PROJECTION", "projection bytes are not canonical")
	}
	digest, err := canon.DigestBytes("ProjectionFingerprint", canonicalBytes)
	if err != nil {
		return ProjectionFingerprint{}, err
	}
	parsed, err := ParseDigest(digest.String())
	if err != nil {
		return ProjectionFingerprint{}, err
	}
	return ProjectionFingerprint{digest: parsed}, nil
}

func ParseProjectionFingerprint(raw string) (ProjectionFingerprint, error) {
	digest, err := ParseDigest(raw)
	if err != nil {
		return ProjectionFingerprint{}, err
	}
	return ProjectionFingerprint{digest: digest}, nil
}

func (p ProjectionFingerprint) String() string { return p.digest.String() }

func (p ProjectionFingerprint) Valid() bool {
	return p.digest.Valid()
}

// digestTyped is the only domain-owned typed-identity boundary. canon validates
// strings, keys, floats, and integer range before producing canonical bytes.
func digestTyped(kind string, value any) (Digest, []byte, error) {
	if !utf8.ValidString(kind) || kind == "" {
		return "", nil, refuse(ErrEmptyIdentityField, "identity kind")
	}
	digest, canonicalBytes, err := canon.DigestTyped(kind, value)
	if err != nil {
		return "", nil, err
	}
	parsed, err := ParseDigest(digest.String())
	if err != nil {
		return "", nil, err
	}
	return parsed, canonicalBytes, nil
}
