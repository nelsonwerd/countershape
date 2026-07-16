package model

import (
	"bytes"
	"path"
	"strconv"
	"strings"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/runnerprofile"
)

const (
	HTTPStartAuthorityV1         = "NODE_CORE_INHERITED_LOOPBACK_LISTENER_V1"
	HTTPPortableStartAuthorityV1 = "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1"
	HTTPStartVersionV1           = "http-start/v1"
	HTTPPortableStartVersionV1   = "http-start/child-bind-pipe-ready/v1"
)

// HTTPStartSpec is the sole U4 start profile: one Node-core service at one
// immutable repository-relative program. Runtime-owned listener/readiness
// descriptors and paths never enter this declaration.
type HTTPStartSpec struct {
	digest         domain.Digest
	canonicalBytes []byte
	entrypoint     string
	authority      string
}

func NewHTTPStartSpec(entrypoint string) (HTTPStartSpec, error) {
	return newHTTPStartSpec(entrypoint, HTTPStartAuthorityV1)
}

// NewPortableHTTPStartSpec constructs the only v1 HTTP start authority that a
// standalone ContractBundle may retain. The historical inherited-listener
// constructor remains valid for sealed history but is deliberately distinct.
func NewPortableHTTPStartSpec(entrypoint string) (HTTPStartSpec, error) {
	return newHTTPStartSpec(entrypoint, HTTPPortableStartAuthorityV1)
}

func newHTTPStartSpec(entrypoint, authority string) (HTTPStartSpec, error) {
	if authority != HTTPStartAuthorityV1 && authority != HTTPPortableStartAuthorityV1 {
		return HTTPStartSpec{}, refuse(CodeInvalidStartSpec, "HTTP start authority is outside the closed v1 roster")
	}
	validEntrypoint := fixedNodeProgram(entrypoint)
	if authority == HTTPPortableStartAuthorityV1 {
		validEntrypoint = runnerprofile.ValidRepositoryNodeEntrypoint(entrypoint)
	}
	if !validEntrypoint {
		return HTTPStartSpec{}, refuse(CodeInvalidStartSpec, "entrypoint must be one exact repository-relative Node script path")
	}
	version := HTTPStartVersionV1
	if authority == HTTPPortableStartAuthorityV1 {
		version = HTTPPortableStartVersionV1
	}
	identity := struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Authority     string   `json:"authority"`
		LogicalArgv   []string `json:"logical_argv"`
		SetupArgv     []string `json:"setup_argv"`
	}{domain.SchemaVersion, "HTTPStartSpec", version, authority, []string{"node", entrypoint}, []string{}}
	digest, canonicalBytes, err := digestTyped("HTTPStartSpec", identity)
	if err != nil {
		return HTTPStartSpec{}, err
	}
	return HTTPStartSpec{digest: digest, canonicalBytes: canonicalBytes, entrypoint: entrypoint, authority: authority}, nil
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
	rebuilt, err := newHTTPStartSpec(s.entrypoint, s.authority)
	return err == nil && rebuilt.digest == s.digest && bytes.Equal(rebuilt.canonicalBytes, s.canonicalBytes)
}
func (s HTTPStartSpec) Digest() domain.Digest  { return s.digest }
func (s HTTPStartSpec) CanonicalBytes() []byte { return append([]byte(nil), s.canonicalBytes...) }
func (s HTTPStartSpec) Executable() string     { return "node" }
func (s HTTPStartSpec) Entrypoint() string     { return s.entrypoint }
func (s HTTPStartSpec) LogicalArgv() []string  { return []string{"node", s.entrypoint} }
func (s HTTPStartSpec) Authority() string      { return s.authority }
func (s HTTPStartSpec) Version() string {
	if s.authority == HTTPPortableStartAuthorityV1 {
		return HTTPPortableStartVersionV1
	}
	return HTTPStartVersionV1
}

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
	ReadinessSuccessByte           byte = 0x01
	ReadinessProtocolV1                 = "ONE_BYTE_0X01_THEN_EOF_V1"
	ReadinessSignalNameV1               = "ready-byte"
	PortableReadinessProtocolV1         = "ASCII_COUNTERSHAPE_READY_V1_SPACE_PORT_LF_THEN_EOF_V1"
	PortableReadinessSignalNameV1       = "ready-port-frame"
	PortableReadinessFramePrefix        = "COUNTERSHAPE_READY_V1 "
	PortableReadinessFrameMax           = 32
	HTTPReadinessVersionV1              = "http-readiness/v1"
	HTTPPortableReadinessVersionV1      = "http-readiness/child-port-frame/v1"
)

type HTTPReadinessContract struct {
	digest         domain.Digest
	canonicalBytes []byte
	signalName     string
	protocol       string
}

func NewHTTPReadinessContract() (HTTPReadinessContract, error) {
	return newHTTPReadinessContract(ReadinessSignalNameV1, ReadinessProtocolV1)
}

// NewPortableHTTPReadinessContract binds the child-selected literal loopback
// port to one dedicated inherited pipe. The bounded frame is parsed by the
// process owner; it is not stdout or application response data.
func NewPortableHTTPReadinessContract() (HTTPReadinessContract, error) {
	return newHTTPReadinessContract(PortableReadinessSignalNameV1, PortableReadinessProtocolV1)
}

func newHTTPReadinessContract(signalName, protocol string) (HTTPReadinessContract, error) {
	legacy := signalName == ReadinessSignalNameV1 && protocol == ReadinessProtocolV1
	portable := signalName == PortableReadinessSignalNameV1 && protocol == PortableReadinessProtocolV1
	if !legacy && !portable {
		return HTTPReadinessContract{}, refuse(CodeInvalidStartSpec, "HTTP readiness signal and protocol are cross-paired")
	}
	var digest domain.Digest
	var canonicalBytes []byte
	var err error
	if legacy {
		// Preserve the historical U4 bytes exactly. Adding optional zero-value
		// fields here would silently rewrite every sealed inherited-listener plan.
		digest, canonicalBytes, err = digestTyped("HTTPReadinessContract", struct {
			SchemaVersion string `json:"schema_version"`
			Kind          string `json:"kind"`
			Version       string `json:"version"`
			SignalName    string `json:"signal_name"`
			Protocol      string `json:"protocol"`
			SuccessByte   int    `json:"success_byte"`
			SuccessLength int    `json:"success_length"`
			EOFRequired   bool   `json:"eof_required"`
			HTTPProbe     bool   `json:"http_probe"`
		}{domain.SchemaVersion, "HTTPReadinessContract", HTTPReadinessVersionV1, signalName, protocol, int(ReadinessSuccessByte), 1, true, false})
	} else {
		digest, canonicalBytes, err = digestTyped("HTTPReadinessContract", struct {
			SchemaVersion string `json:"schema_version"`
			Kind          string `json:"kind"`
			Version       string `json:"version"`
			SignalName    string `json:"signal_name"`
			Protocol      string `json:"protocol"`
			FramePrefix   string `json:"frame_prefix"`
			FrameMaxBytes int    `json:"frame_max_bytes"`
			EOFRequired   bool   `json:"eof_required"`
			HTTPProbe     bool   `json:"http_probe"`
		}{
			domain.SchemaVersion, "HTTPReadinessContract", HTTPPortableReadinessVersionV1, signalName, protocol,
			PortableReadinessFramePrefix, PortableReadinessFrameMax, true, false,
		})
	}
	if err != nil {
		return HTTPReadinessContract{}, err
	}
	return HTTPReadinessContract{
		digest: digest, canonicalBytes: canonicalBytes, signalName: signalName, protocol: protocol,
	}, nil
}

func (r HTTPReadinessContract) Valid() bool {
	rebuilt, err := newHTTPReadinessContract(r.signalName, r.protocol)
	return err == nil && rebuilt.digest == r.digest && bytes.Equal(rebuilt.canonicalBytes, r.canonicalBytes)
}
func (r HTTPReadinessContract) Digest() domain.Digest { return r.digest }
func (r HTTPReadinessContract) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r HTTPReadinessContract) SignalName() string { return r.signalName }
func (r HTTPReadinessContract) Protocol() string   { return r.protocol }
func (r HTTPReadinessContract) Version() string {
	if r.protocol == PortableReadinessProtocolV1 {
		return HTTPPortableReadinessVersionV1
	}
	return HTTPReadinessVersionV1
}
func (r HTTPReadinessContract) SuccessByte() byte {
	if r.protocol == ReadinessProtocolV1 {
		return ReadinessSuccessByte
	}
	return 0
}

// PortableFrameProfile reports the exact child-bind readiness wire profile.
// A false result means this is the historical one-byte readiness contract.
func (r HTTPReadinessContract) PortableFrameProfile() (prefix string, maxBytes int, eofRequired bool, ok bool) {
	if r.protocol != PortableReadinessProtocolV1 {
		return "", 0, false, false
	}
	return PortableReadinessFramePrefix, PortableReadinessFrameMax, true, true
}

// HTTPReadyPortFrame is the complete portable readiness pipe payload. Parsing
// this value consumes the entire byte slice, which makes EOF part of the
// authority rather than an ambient stream convention.
type HTTPReadyPortFrame struct {
	port      uint16
	canonical []byte
}

func NewHTTPReadyPortFrame(port int) (HTTPReadyPortFrame, error) {
	if port < 1 || port > 65535 {
		return HTTPReadyPortFrame{}, refuse(CodeInvalidReadiness, "portable readiness port must be in 1..65535")
	}
	canonical := []byte(PortableReadinessFramePrefix + strconv.Itoa(port) + "\n")
	if len(canonical) > PortableReadinessFrameMax {
		return HTTPReadyPortFrame{}, refuse(CodeInvalidReadiness, "portable readiness frame exceeds its closed ceiling")
	}
	return HTTPReadyPortFrame{port: uint16(port), canonical: canonical}, nil
}

func ParseHTTPReadyPortFrame(exact []byte) (HTTPReadyPortFrame, error) {
	if len(exact) == 0 || len(exact) > PortableReadinessFrameMax ||
		!bytes.HasPrefix(exact, []byte(PortableReadinessFramePrefix)) || exact[len(exact)-1] != '\n' {
		return HTTPReadyPortFrame{}, refuse(CodeInvalidReadiness, "portable readiness frame has invalid prefix, length, or terminator")
	}
	digits := exact[len(PortableReadinessFramePrefix) : len(exact)-1]
	if len(digits) == 0 || (len(digits) > 1 && digits[0] == '0') {
		return HTTPReadyPortFrame{}, refuse(CodeInvalidReadiness, "portable readiness port is not canonical decimal")
	}
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return HTTPReadyPortFrame{}, refuse(CodeInvalidReadiness, "portable readiness port contains non-decimal text")
		}
	}
	port, err := strconv.Atoi(string(digits))
	if err != nil {
		return HTTPReadyPortFrame{}, refuse(CodeInvalidReadiness, "portable readiness port could not be decoded")
	}
	rebuilt, err := NewHTTPReadyPortFrame(port)
	if err != nil || !bytes.Equal(rebuilt.canonical, exact) {
		return HTTPReadyPortFrame{}, refuse(CodeInvalidReadiness, "portable readiness frame is outside the exact profile")
	}
	return rebuilt, nil
}

func (f HTTPReadyPortFrame) Valid() bool {
	rebuilt, err := ParseHTTPReadyPortFrame(f.canonical)
	return err == nil && rebuilt.port == f.port
}
func (f HTTPReadyPortFrame) Port() uint16 { return f.port }
func (f HTTPReadyPortFrame) CanonicalBytes() []byte {
	return append([]byte(nil), f.canonical...)
}
