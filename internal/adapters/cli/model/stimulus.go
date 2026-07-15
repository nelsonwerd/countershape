// Package model owns the cycle-free CLI execution input authority. The world
// edge may import this package; the higher-level cli adapter may import both
// model and world without creating a dependency cycle.
package model

import (
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	maxExecutableBytes    = 128
	maxArgCount           = 64
	maxArgBytes           = 4096
	maxTotalArgBytes      = 64 << 10
	maxStdinBytes         = 1 << 20
	maxEnvironmentEntries = 64
	maxEnvironmentValue   = 16 << 10
	// JSON expands '<', '>', and '&' to six-byte escapes. This raw aggregate
	// leaves proved headroom below canon's 1 MiB exact-document ceiling.
	maxEnvironmentTotalBytes = 128 << 10
	maxFixtureFiles          = 256
	maxFixturePathBytes      = 1024
	maxFixtureFileBytes      = 1 << 20
	maxFixtureTotalBytes     = 16 << 20
	maxFixtureTotalPathByte  = 256 << 10
)

var (
	executablePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)
	environmentPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
)

const (
	CodeInvalidStimulus      = "CLI_INVALID_STIMULUS"
	CodeInvalidExecutable    = "CLI_INVALID_EXECUTABLE"
	CodeInvalidArgv          = "CLI_INVALID_ARGV"
	CodeInvalidStdin         = "CLI_INVALID_STDIN"
	CodeInvalidEnvironment   = "CLI_INVALID_ENVIRONMENT"
	CodeInvalidFixture       = "CLI_INVALID_FIXTURE"
	CodeFixtureOrder         = "CLI_FIXTURE_ORDER_INVALID"
	CodeFixturePath          = "CLI_FIXTURE_PATH_INVALID"
	CodeFixtureCollision     = "CLI_FIXTURE_PATH_COLLISION"
	CodeInputLimit           = "CLI_INPUT_LIMIT"
	CodeInvalidCWDPolicy     = "CLI_INVALID_CWD_POLICY"
	CodeInvalidBinding       = "CLI_INVALID_EXECUTION_BINDING"
	CodeInvalidCapturePolicy = "CLI_INVALID_CAPTURE_POLICY"
)

// Refusal is a construction failure with a stable machine code.
type Refusal struct {
	Code   string
	Detail string
}

func (r *Refusal) Error() string {
	if r.Detail == "" {
		return r.Code
	}
	return r.Code + ": " + r.Detail
}

func refuse(code, detail string) error { return &Refusal{Code: code, Detail: detail} }

type Presence string

const (
	PresenceAbsent  Presence = "ABSENT"
	PresencePresent Presence = "PRESENT"
)

func validPresence(value Presence) bool {
	return value == PresenceAbsent || value == PresencePresent
}

// CLIStdin is a tagged value: absent is not present with zero bytes.
type CLIStdin struct {
	presence Presence
	bytes    []byte
}

func AbsentStdin() CLIStdin { return CLIStdin{presence: PresenceAbsent, bytes: []byte{}} }

func PresentStdin(value []byte) (CLIStdin, error) {
	if len(value) > maxStdinBytes {
		return CLIStdin{}, refuse(CodeInputLimit, "stdin exceeds the v1 byte ceiling")
	}
	return CLIStdin{presence: PresencePresent, bytes: append([]byte(nil), value...)}, nil
}

func (s CLIStdin) Presence() Presence { return s.presence }
func (s CLIStdin) Present() bool      { return s.presence == PresencePresent }
func (s CLIStdin) Bytes() []byte      { return append([]byte(nil), s.bytes...) }

func (s CLIStdin) valid() bool {
	return validPresence(s.presence) && len(s.bytes) <= maxStdinBytes &&
		(s.presence == PresencePresent || len(s.bytes) == 0)
}

// CLIEnvironmentBinding records one explicit sparse environment decision.
// The constructor sorts bindings by name; caller order is not process order.
type CLIEnvironmentBinding struct {
	name     string
	presence Presence
	value    string
}

func AbsentEnvironment(name string) (CLIEnvironmentBinding, error) {
	if err := validateEnvironmentName(name); err != nil {
		return CLIEnvironmentBinding{}, err
	}
	return CLIEnvironmentBinding{name: name, presence: PresenceAbsent}, nil
}

func PresentEnvironment(name, value string) (CLIEnvironmentBinding, error) {
	if err := validateEnvironmentName(name); err != nil {
		return CLIEnvironmentBinding{}, err
	}
	if !validEnvironmentValue(value) {
		return CLIEnvironmentBinding{}, refuse(CodeInvalidEnvironment, "environment value is invalid or exceeds the v1 byte ceiling")
	}
	return CLIEnvironmentBinding{name: name, presence: PresencePresent, value: value}, nil
}

func (b CLIEnvironmentBinding) Name() string       { return b.name }
func (b CLIEnvironmentBinding) Presence() Presence { return b.presence }
func (b CLIEnvironmentBinding) Present() bool      { return b.presence == PresencePresent }
func (b CLIEnvironmentBinding) Value() string      { return b.value }

func (b CLIEnvironmentBinding) valid() bool {
	if validateEnvironmentName(b.name) != nil || !validPresence(b.presence) {
		return false
	}
	if b.presence == PresenceAbsent {
		return b.value == ""
	}
	return validEnvironmentValue(b.value)
}

func validEnvironmentValue(value string) bool {
	if !validText(value, maxEnvironmentValue) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validateEnvironmentName(name string) error {
	if !validText(name, 128) || !environmentPattern.MatchString(name) {
		return refuse(CodeInvalidEnvironment, "environment name is outside the closed v1 spelling")
	}
	switch name {
	case "HOME", "PATH", "TMPDIR", "TMP", "TEMP", "PWD", "OLDPWD", "SHELL",
		"NODE_OPTIONS", "BASH_ENV", "ENV", "RUBYOPT", "PERL5OPT", "PYTHONPATH", "PYTHONHOME", "GODEBUG":
		return refuse(CodeInvalidEnvironment, "runner-owned environment name cannot be stimulus-controlled")
	}
	if strings.HasPrefix(name, "XDG_") || strings.HasPrefix(name, "COUNTERSHAPE_") || strings.HasPrefix(name, "DYLD_") {
		return refuse(CodeInvalidEnvironment, "runner-owned environment namespace cannot be stimulus-controlled")
	}
	return nil
}

type FixtureMode string

const (
	FixtureMode0644 FixtureMode = "100644"
)

// CLIFixtureFile is one bounded regular file beneath the private fixture root.
// Symlinks, directories, devices, and caller-selected arbitrary modes do not
// have constructors.
type CLIFixtureFile struct {
	path     string
	contents []byte
	mode     FixtureMode
}

func NewFixtureFile(relativePath string, contents []byte, mode FixtureMode) (CLIFixtureFile, error) {
	if err := validateFixturePath(relativePath); err != nil {
		return CLIFixtureFile{}, err
	}
	if mode != FixtureMode0644 {
		return CLIFixtureFile{}, refuse(CodeInvalidFixture, "fixture must use one supported regular-file mode")
	}
	if len(contents) > maxFixtureFileBytes {
		return CLIFixtureFile{}, refuse(CodeInputLimit, "fixture file exceeds the per-file byte ceiling")
	}
	return CLIFixtureFile{path: relativePath, contents: append([]byte(nil), contents...), mode: mode}, nil
}

func (f CLIFixtureFile) Path() string      { return f.path }
func (f CLIFixtureFile) Contents() []byte  { return append([]byte(nil), f.contents...) }
func (f CLIFixtureFile) Mode() FixtureMode { return f.mode }

func (f CLIFixtureFile) valid() bool {
	return validateFixturePath(f.path) == nil &&
		f.mode == FixtureMode0644 &&
		len(f.contents) <= maxFixtureFileBytes
}

func validateFixturePath(value string) error {
	if !validText(value, maxFixturePathBytes) || strings.HasPrefix(value, "/") ||
		strings.Contains(value, "\\") || path.Clean(value) != value || value == "." {
		return refuse(CodeFixturePath, "fixture path must be a clean relative slash path")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return refuse(CodeFixturePath, "fixture path contains an empty or traversal segment")
		}
		// MUTATION_ANCHOR: reserved-git-metadata-must-be-rejected-before-world
		if strings.EqualFold(segment, ".git") {
			return refuse(CodeFixturePath, "fixture path contains a reserved metadata segment")
		}
	}
	return nil
}

type CWDPolicy string

const (
	CWDMaterializedRoot CWDPolicy = "MATERIALIZED_ROOT"
)

func (p CWDPolicy) Valid() bool {
	return p == CWDMaterializedRoot
}

type CLIStimulusConfig struct {
	Executable  string
	BaseArgv    []string
	Argv        []string
	Stdin       CLIStdin
	Environment []CLIEnvironmentBinding
	Fixtures    []CLIFixtureFile
	CWDPolicy   CWDPolicy
}

// CLIStimulus is immutable typed input. Its digest binds the distinction
// between fixed base argv and reducible per-stimulus argv in addition to the
// full physical execution payload.
type CLIStimulus struct {
	digest                         domain.Digest
	canonicalBytes                 []byte
	executionPayloadDigest         domain.Digest
	executionPayloadCanonicalBytes []byte
	executable                     string
	baseArgv                       []string
	argv                           []string
	stdin                          CLIStdin
	environment                    []CLIEnvironmentBinding
	fixtures                       []CLIFixtureFile
	cwdPolicy                      CWDPolicy
	measure                        CLIStimulusMeasure
}

type stdinIdentity struct {
	Presence    string `json:"presence"`
	ByteLength  int    `json:"byte_length"`
	BytesDigest string `json:"bytes_digest"`
}

type environmentIdentity struct {
	Name     string `json:"name"`
	Presence string `json:"presence"`
	Value    string `json:"value"`
}

type fixtureIdentity struct {
	Path           string `json:"path"`
	Mode           string `json:"mode"`
	ContentsBytes  int    `json:"contents_bytes"`
	ContentsDigest string `json:"contents_digest"`
}

type executionPayloadIdentity struct {
	SchemaVersion string                `json:"schema_version"`
	Kind          string                `json:"kind"`
	Executable    string                `json:"executable"`
	LogicalArgv   []string              `json:"logical_argv"`
	Stdin         stdinIdentity         `json:"stdin"`
	Environment   []environmentIdentity `json:"environment"`
	Fixtures      []fixtureIdentity     `json:"fixtures"`
	CWDPolicy     string                `json:"cwd_policy"`
}

type stimulusIdentity struct {
	SchemaVersion          string                `json:"schema_version"`
	Kind                   string                `json:"kind"`
	ExecutionPayloadDigest string                `json:"execution_payload_digest"`
	Executable             string                `json:"executable"`
	BaseArgv               []string              `json:"base_argv"`
	Argv                   []string              `json:"argv"`
	Stdin                  stdinIdentity         `json:"stdin"`
	Environment            []environmentIdentity `json:"environment"`
	Fixtures               []fixtureIdentity     `json:"fixtures"`
	CWDPolicy              string                `json:"cwd_policy"`
	Measure                measureIdentity       `json:"measure"`
}

func NewCLIStimulus(config CLIStimulusConfig) (CLIStimulus, error) {
	if !validText(config.Executable, maxExecutableBytes) || !executablePattern.MatchString(config.Executable) ||
		config.Executable == "." || config.Executable == ".." {
		return CLIStimulus{}, refuse(CodeInvalidExecutable, "executable must be one bare logical tool name")
	}
	if !config.CWDPolicy.Valid() {
		return CLIStimulus{}, refuse(CodeInvalidCWDPolicy, "unsupported working-directory policy")
	}
	if !config.Stdin.valid() {
		return CLIStimulus{}, refuse(CodeInvalidStdin, "stdin must be explicitly absent or present within the byte ceiling")
	}
	if err := validateArgv(config.Executable, config.BaseArgv, config.Argv); err != nil {
		return CLIStimulus{}, err
	}
	environment, err := normalizeEnvironment(config.Environment)
	if err != nil {
		return CLIStimulus{}, err
	}
	fixtures, err := validateFixtures(config.Fixtures)
	if err != nil {
		return CLIStimulus{}, err
	}

	baseArgv := append([]string{}, config.BaseArgv...)
	argv := append([]string{}, config.Argv...)
	logicalArgv := make([]string, 0, 1+len(baseArgv)+len(argv))
	logicalArgv = append(logicalArgv, config.Executable)
	logicalArgv = append(logicalArgv, baseArgv...)
	// MUTATION_ANCHOR: preserve-argv-order
	logicalArgv = append(logicalArgv, argv...)

	stdin := cloneStdin(config.Stdin)
	environmentIDs := environmentIdentities(environment)
	fixtureIDs, err := fixtureIdentities(fixtures)
	if err != nil {
		return CLIStimulus{}, err
	}
	stdinID, err := stdinIdentityOf(stdin)
	if err != nil {
		return CLIStimulus{}, err
	}
	payloadIdentity := executionPayloadIdentity{
		SchemaVersion: domain.SchemaVersion,
		Kind:          "CLIExecutionPayload",
		Executable:    config.Executable,
		LogicalArgv:   logicalArgv,
		Stdin:         stdinID,
		Environment:   environmentIDs,
		Fixtures:      fixtureIDs,
		CWDPolicy:     string(config.CWDPolicy),
	}
	payloadDigest, payloadBytes, err := digestTyped("CLIExecutionPayload", payloadIdentity)
	if err != nil {
		return CLIStimulus{}, err
	}
	measure := measureOf(argv, stdin, environment, fixtures)
	identity := stimulusIdentity{
		SchemaVersion:          domain.SchemaVersion,
		Kind:                   "CLIStimulus",
		ExecutionPayloadDigest: payloadDigest.String(),
		Executable:             config.Executable,
		BaseArgv:               baseArgv,
		Argv:                   argv,
		Stdin:                  stdinID,
		Environment:            environmentIDs,
		Fixtures:               fixtureIDs,
		CWDPolicy:              string(config.CWDPolicy),
		Measure:                measure.identity(),
	}
	digest, canonicalBytes, err := digestTyped("CLIStimulus", identity)
	if err != nil {
		return CLIStimulus{}, err
	}
	return CLIStimulus{
		digest: digest, canonicalBytes: canonicalBytes,
		executionPayloadDigest: payloadDigest, executionPayloadCanonicalBytes: payloadBytes,
		executable: config.Executable, baseArgv: baseArgv, argv: argv, stdin: stdin,
		environment: environment, fixtures: fixtures, cwdPolicy: config.CWDPolicy, measure: measure,
	}, nil
}

func validateArgv(executable string, baseArgv, argv []string) error {
	if len(baseArgv)+len(argv)+1 > maxArgCount {
		return refuse(CodeInputLimit, "logical argv exceeds the item ceiling")
	}
	total := 0
	for _, value := range append(append([]string{}, baseArgv...), argv...) {
		if !validText(value, maxArgBytes) {
			return refuse(CodeInvalidArgv, "argv item is invalid or exceeds the per-item byte ceiling")
		}
		total += len(value)
	}
	if total > maxTotalArgBytes {
		return refuse(CodeInputLimit, "logical argv exceeds the aggregate byte ceiling")
	}
	logicalArgv := append([]string{executable}, baseArgv...)
	logicalArgv = append(logicalArgv, argv...)
	// MUTATION_ANCHOR: node-must-name-one-fixed-repository-program
	if executable != "node" || !fixedNodeProgram(baseArgv) {
		return refuse(CodeInvalidArgv, "U3 admits only node with one fixed repository-relative script entrypoint in immutable base argv")
	}
	// MUTATION_ANCHOR: reducible-argv-must-use-domain-direct-grammar
	if err := domain.ValidateDirectArgv(logicalArgv); err != nil {
		return refuse(CodeInvalidArgv, err.Error())
	}
	return nil
}

func fixedNodeProgram(baseArgv []string) bool {
	if len(baseArgv) == 0 || strings.HasPrefix(baseArgv[0], "-") || strings.HasPrefix(baseArgv[0], "/") ||
		path.Clean(baseArgv[0]) != baseArgv[0] || baseArgv[0] == "." || baseArgv[0] == ".." {
		return false
	}
	switch path.Ext(baseArgv[0]) {
	case ".js", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

func normalizeEnvironment(input []CLIEnvironmentBinding) ([]CLIEnvironmentBinding, error) {
	if len(input) > maxEnvironmentEntries {
		return nil, refuse(CodeInputLimit, "environment exceeds the entry ceiling")
	}
	result := append([]CLIEnvironmentBinding{}, input...)
	totalBytes := 0
	for _, binding := range result {
		if !binding.valid() {
			return nil, refuse(CodeInvalidEnvironment, "environment contains an invalid binding")
		}
		totalBytes += len(binding.name) + len(binding.value)
	}
	if totalBytes > maxEnvironmentTotalBytes {
		return nil, refuse(CodeInputLimit, "environment exceeds the aggregate identity byte ceiling")
	}
	sort.Slice(result, func(i, j int) bool { return result[i].name < result[j].name })
	for index := 1; index < len(result); index++ {
		if result[index-1].name == result[index].name {
			return nil, refuse(CodeInvalidEnvironment, "environment contains a duplicate name")
		}
	}
	return result, nil
}

func validateFixtures(input []CLIFixtureFile) ([]CLIFixtureFile, error) {
	if len(input) > maxFixtureFiles {
		return nil, refuse(CodeInputLimit, "fixture set exceeds the file ceiling")
	}
	result := cloneFixtures(input)
	totalBytes := 0
	totalPathBytes := 0
	for index, fixture := range result {
		if !fixture.valid() {
			return nil, refuse(CodeInvalidFixture, "fixture set contains an invalid regular file")
		}
		if index > 0 && result[index-1].path >= fixture.path {
			return nil, refuse(CodeFixtureOrder, "fixture files must be supplied in strict path order")
		}
		totalBytes += len(fixture.contents)
		totalPathBytes += len(fixture.path)
	}
	if totalBytes > maxFixtureTotalBytes || totalPathBytes > maxFixtureTotalPathByte {
		return nil, refuse(CodeInputLimit, "fixture set exceeds an aggregate byte ceiling")
	}
	for left := 0; left < len(result); left++ {
		leftFolded := strings.ToLower(result[left].path)
		for right := left + 1; right < len(result); right++ {
			rightFolded := strings.ToLower(result[right].path)
			if leftFolded == rightFolded || strings.HasPrefix(rightFolded, leftFolded+"/") ||
				strings.HasPrefix(leftFolded, rightFolded+"/") {
				return nil, refuse(CodeFixtureCollision, "fixture files collide by path or file/directory prefix")
			}
		}
	}
	return result, nil
}

func (s CLIStimulus) Valid() bool {
	rebuilt, err := NewCLIStimulus(CLIStimulusConfig{
		Executable: s.executable, BaseArgv: s.baseArgv, Argv: s.argv, Stdin: s.stdin,
		Environment: s.environment, Fixtures: s.fixtures, CWDPolicy: s.cwdPolicy,
	})
	return err == nil && rebuilt.digest == s.digest &&
		rebuilt.executionPayloadDigest == s.executionPayloadDigest &&
		string(rebuilt.canonicalBytes) == string(s.canonicalBytes) &&
		string(rebuilt.executionPayloadCanonicalBytes) == string(s.executionPayloadCanonicalBytes)
}

func (s CLIStimulus) Digest() domain.Digest                 { return s.digest }
func (s CLIStimulus) CanonicalBytes() []byte                { return append([]byte(nil), s.canonicalBytes...) }
func (s CLIStimulus) ExecutionPayloadDigest() domain.Digest { return s.executionPayloadDigest }
func (s CLIStimulus) ExecutionPayloadCanonicalBytes() []byte {
	return append([]byte(nil), s.executionPayloadCanonicalBytes...)
}
func (s CLIStimulus) Executable() string { return s.executable }
func (s CLIStimulus) BaseArgv() []string { return append([]string(nil), s.baseArgv...) }
func (s CLIStimulus) Argv() []string     { return append([]string(nil), s.argv...) }
func (s CLIStimulus) BaseLogicalArgv() []string {
	return append([]string{s.executable}, s.baseArgv...)
}
func (s CLIStimulus) LogicalArgv() []string {
	result := append([]string{s.executable}, s.baseArgv...)
	return append(result, s.argv...)
}
func (s CLIStimulus) Stdin() CLIStdin { return cloneStdin(s.stdin) }
func (s CLIStimulus) Environment() []CLIEnvironmentBinding {
	return append([]CLIEnvironmentBinding(nil), s.environment...)
}
func (s CLIStimulus) Fixtures() []CLIFixtureFile  { return cloneFixtures(s.fixtures) }
func (s CLIStimulus) CWDPolicy() CWDPolicy        { return s.cwdPolicy }
func (s CLIStimulus) Measure() CLIStimulusMeasure { return s.measure }

func stdinIdentityOf(value CLIStdin) (stdinIdentity, error) {
	// MUTATION_ANCHOR: stdin-absent-distinct-from-present-empty
	identity := stdinIdentity{Presence: string(value.presence), ByteLength: len(value.bytes)}
	if value.presence == PresenceAbsent {
		return identity, nil
	}
	digest, err := digestExactBytes("CLIStdinBytes", value.bytes)
	if err != nil {
		return stdinIdentity{}, err
	}
	identity.BytesDigest = digest.String()
	return identity, nil
}

func environmentIdentities(values []CLIEnvironmentBinding) []environmentIdentity {
	result := make([]environmentIdentity, len(values))
	for index, value := range values {
		// MUTATION_ANCHOR: environment-absent-distinct-from-present-empty
		result[index] = environmentIdentity{Name: value.name, Presence: string(value.presence), Value: value.value}
	}
	return result
}

func fixtureIdentities(values []CLIFixtureFile) ([]fixtureIdentity, error) {
	result := make([]fixtureIdentity, len(values))
	for index, value := range values {
		digest, err := digestExactBytes("CLIFixtureContents", value.contents)
		if err != nil {
			return nil, err
		}
		result[index] = fixtureIdentity{
			Path: value.path, Mode: string(value.mode),
			ContentsBytes: len(value.contents), ContentsDigest: digest.String(),
		}
	}
	return result, nil
}

func cloneStdin(value CLIStdin) CLIStdin {
	return CLIStdin{presence: value.presence, bytes: append([]byte(nil), value.bytes...)}
}

func cloneFixtures(input []CLIFixtureFile) []CLIFixtureFile {
	result := make([]CLIFixtureFile, len(input))
	for index, file := range input {
		result[index] = CLIFixtureFile{path: file.path, mode: file.mode, contents: append([]byte(nil), file.contents...)}
	}
	return result
}

func validText(value string, limit int) bool {
	return utf8.ValidString(value) && len(value) <= limit && !strings.ContainsRune(value, '\x00')
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
	return parsed, append([]byte(nil), canonicalBytes...), nil
}

func digestExactBytes(kind string, value []byte) (domain.Digest, error) {
	digest, err := canon.DigestBytes(kind, value)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}
