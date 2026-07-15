package model

import (
	"bytes"
	"path"
	"strings"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const HTTPStartAuthorityV1 = "NODE_CORE_INHERITED_LOOPBACK_LISTENER_V1"

// HTTPStartSpec is the sole U4 start profile: one Node-core service at one
// immutable repository-relative program. Runtime-owned listener/readiness
// descriptors and paths never enter this declaration.
type HTTPStartSpec struct {
	digest         domain.Digest
	canonicalBytes []byte
	entrypoint     string
}

func NewHTTPStartSpec(entrypoint string) (HTTPStartSpec, error) {
	if !fixedNodeProgram(entrypoint) {
		return HTTPStartSpec{}, refuse(CodeInvalidStartSpec, "entrypoint must be one clean repository-relative .js/.mjs/.cjs path")
	}
	identity := struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Authority     string   `json:"authority"`
		LogicalArgv   []string `json:"logical_argv"`
		SetupArgv     []string `json:"setup_argv"`
	}{domain.SchemaVersion, "HTTPStartSpec", "http-start/v1", HTTPStartAuthorityV1, []string{"node", entrypoint}, []string{}}
	digest, canonicalBytes, err := digestTyped("HTTPStartSpec", identity)
	if err != nil {
		return HTTPStartSpec{}, err
	}
	return HTTPStartSpec{digest: digest, canonicalBytes: canonicalBytes, entrypoint: entrypoint}, nil
}

func fixedNodeProgram(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "-") ||
		strings.Contains(value, "\\") || path.Clean(value) != value || value == "." || value == ".." ||
		strings.HasPrefix(value, "../") || strings.Contains(value, "/../") {
		return false
	}
	switch path.Ext(value) {
	case ".js", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

func (s HTTPStartSpec) Valid() bool {
	rebuilt, err := NewHTTPStartSpec(s.entrypoint)
	return err == nil && rebuilt.digest == s.digest && bytes.Equal(rebuilt.canonicalBytes, s.canonicalBytes)
}
func (s HTTPStartSpec) Digest() domain.Digest  { return s.digest }
func (s HTTPStartSpec) CanonicalBytes() []byte { return append([]byte(nil), s.canonicalBytes...) }
func (s HTTPStartSpec) Executable() string     { return "node" }
func (s HTTPStartSpec) Entrypoint() string     { return s.entrypoint }
func (s HTTPStartSpec) LogicalArgv() []string  { return []string{"node", s.entrypoint} }

const HTTPSeedOverlayAuthorityV1 = "PRIVATE_SEED_ROOT_EXCLUSIVE_REOPEN_REHASH_V1"

type HTTPFixtureRecipe struct {
	digest         domain.Digest
	canonicalBytes []byte
}

func NewHTTPFixtureRecipe() (HTTPFixtureRecipe, error) {
	identity := struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Authority     string   `json:"authority"`
		RootPolicy    string   `json:"root_policy"`
		PathPolicy    string   `json:"path_policy"`
		Modes         []string `json:"regular_file_modes"`
		WritePolicy   string   `json:"write_policy"`
		VerifyPolicy  string   `json:"verify_policy"`
	}{
		domain.SchemaVersion, "HTTPFixtureRecipe", "http-fixture-recipe/v1", HTTPSeedOverlayAuthorityV1,
		"NEW_PRIVATE_FIXTURE_ROOT", "STRICT_ORDER_NO_ALIAS_PREFIX_COLLISION_OR_GIT_METADATA",
		[]string{string(SeedMode0644)}, "EXCLUSIVE_NO_OVERWRITE_SYNCED", "REOPEN_REGULAR_MODE_SIZE_BYTES_AND_DOMAIN_DIGEST",
	}
	digest, canonicalBytes, err := digestTyped("HTTPFixtureRecipe", identity)
	if err != nil {
		return HTTPFixtureRecipe{}, err
	}
	return HTTPFixtureRecipe{digest: digest, canonicalBytes: canonicalBytes}, nil
}

func (r HTTPFixtureRecipe) Valid() bool {
	rebuilt, err := NewHTTPFixtureRecipe()
	return err == nil && rebuilt.digest == r.digest && bytes.Equal(rebuilt.canonicalBytes, r.canonicalBytes)
}
func (r HTTPFixtureRecipe) Digest() domain.Digest  { return r.digest }
func (r HTTPFixtureRecipe) CanonicalBytes() []byte { return append([]byte(nil), r.canonicalBytes...) }

const (
	ReadinessSuccessByte  byte = 0x01
	ReadinessProtocolV1        = "ONE_BYTE_0X01_THEN_EOF_V1"
	ReadinessSignalNameV1      = "ready-byte"
)

type HTTPReadinessContract struct {
	digest         domain.Digest
	canonicalBytes []byte
}

func NewHTTPReadinessContract() (HTTPReadinessContract, error) {
	identity := struct {
		SchemaVersion string `json:"schema_version"`
		Kind          string `json:"kind"`
		Version       string `json:"version"`
		SignalName    string `json:"signal_name"`
		Protocol      string `json:"protocol"`
		SuccessByte   int    `json:"success_byte"`
		SuccessLength int    `json:"success_length"`
		EOFRequired   bool   `json:"eof_required"`
		HTTPProbe     bool   `json:"http_probe"`
	}{domain.SchemaVersion, "HTTPReadinessContract", "http-readiness/v1", ReadinessSignalNameV1, ReadinessProtocolV1, int(ReadinessSuccessByte), 1, true, false}
	digest, canonicalBytes, err := digestTyped("HTTPReadinessContract", identity)
	if err != nil {
		return HTTPReadinessContract{}, err
	}
	return HTTPReadinessContract{digest: digest, canonicalBytes: canonicalBytes}, nil
}

func (r HTTPReadinessContract) Valid() bool {
	rebuilt, err := NewHTTPReadinessContract()
	return err == nil && rebuilt.digest == r.digest && bytes.Equal(rebuilt.canonicalBytes, r.canonicalBytes)
}
func (r HTTPReadinessContract) Digest() domain.Digest { return r.digest }
func (r HTTPReadinessContract) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r HTTPReadinessContract) SignalName() string { return ReadinessSignalNameV1 }
func (r HTTPReadinessContract) Protocol() string   { return ReadinessProtocolV1 }
func (r HTTPReadinessContract) SuccessByte() byte  { return ReadinessSuccessByte }
