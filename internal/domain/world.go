package domain

import (
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const SchemaVersion = "countershape/v1"

type AdapterDomain string

const (
	AdapterCLI  AdapterDomain = "CLI"
	AdapterHTTP AdapterDomain = "HTTP"
)

type ExecutionShape string

const (
	OneCLIInvocation       ExecutionShape = "ONE_CLI_INVOCATION"
	OneLoopbackHTTPRequest ExecutionShape = "ONE_LOOPBACK_HTTP_REQUEST"
)

type ReadinessKind string

const (
	ReadinessNone         ReadinessKind = "NONE"
	FixtureOwnedReadiness ReadinessKind = "FIXTURE_OWNED_SIGNAL"
)

type Readiness struct {
	Kind       ReadinessKind `json:"kind"`
	SignalName string        `json:"signal_name"`
}

type Adapter struct {
	Domain         AdapterDomain `json:"domain"`
	AdapterVersion string        `json:"adapter_version"`
	RunnerDigest   Digest        `json:"runner_digest"`
}

type EnvironmentEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SecretPresence string

const (
	SecretAbsent            SecretPresence = "ABSENT"
	SecretRequiredNoCapture SecretPresence = "REQUIRED_WITHOUT_VALUE_CAPTURE"
)

type SecretSlot struct {
	Name     string         `json:"name"`
	Presence SecretPresence `json:"presence"`
}

type RequiredTool struct {
	Name              string `json:"name"`
	VersionConstraint string `json:"version_constraint"`
}

type RepeatSchedule struct {
	DiscoveryRepeats    int `json:"discovery_repeats"`
	ConfirmationRepeats int `json:"confirmation_repeats"`
}

type Budgets struct {
	CandidateCount            int   `json:"candidate_count"`
	MaterializedEntryCount    int   `json:"materialized_entry_count"`
	MaterializedBytesPerWorld int64 `json:"materialized_bytes_per_world"`
	SingleBlobBytes           int64 `json:"single_blob_bytes"`
	ReadinessMS               int64 `json:"readiness_ms"`
	ProbeMS                   int64 `json:"probe_ms"`
	TeardownMS                int64 `json:"teardown_ms"`
	StdoutBytes               int64 `json:"stdout_bytes"`
	StderrBytes               int64 `json:"stderr_bytes"`
	HTTPBodyBytes             int64 `json:"http_body_bytes"`
	ProposedShrinkStimuli     int   `json:"proposed_shrink_stimuli"`
	TotalCandidateTrials      int   `json:"total_candidate_trials"`
	ShrinkWallMS              int64 `json:"shrink_wall_ms"`
}

type WorldPlanConfig struct {
	CandidateSetDigest          Digest
	MaterializationPolicyDigest Digest
	ComparisonEnvelopeDigest    Digest
	Adapter                     Adapter
	ExecutionShape              ExecutionShape
	StartArgv                   []string
	SetupArgv                   []string
	Environment                 []EnvironmentEntry
	SecretSlots                 []SecretSlot
	FixtureRecipeDigest         Digest
	Readiness                   Readiness
	CapturePolicyDigest         Digest
	ProjectionDefinition        ProjectionDefinitionBinding
	RepeatSchedule              RepeatSchedule
	RequiredTools               []RequiredTool
	Budgets                     Budgets
}

// WorldPlanDeclarationConfig is the non-authoritative, digest-referencing
// shape used while validating an inert source document. Successful validation
// returns no WorldPlan and cannot manufacture a ProjectionDefinitionBinding.
// Compilation must still resolve the declared digest to the full binding.
type WorldPlanDeclarationConfig struct {
	CandidateSetDigest          Digest
	MaterializationPolicyDigest Digest
	ComparisonEnvelopeDigest    Digest
	Adapter                     Adapter
	ExecutionShape              ExecutionShape
	StartArgv                   []string
	SetupArgv                   []string
	Environment                 []EnvironmentEntry
	SecretSlots                 []SecretSlot
	FixtureRecipeDigest         Digest
	Readiness                   Readiness
	CapturePolicyDigest         Digest
	ProjectionDefinitionDigest  Digest
	RepeatSchedule              RepeatSchedule
	RequiredTools               []RequiredTool
	Budgets                     Budgets
}

// WorldPlan is immutable through its public API. Every default must already be
// materialized by the spec compiler before this constructor is called.
type WorldPlan struct {
	digest                      Digest
	canonicalBytes              []byte
	candidateSetDigest          Digest
	materializationPolicyDigest Digest
	comparisonEnvelopeDigest    Digest
	adapter                     Adapter
	adapterDigest               Digest
	executionShape              ExecutionShape
	startArgv                   []string
	setupArgv                   []string
	environment                 []EnvironmentEntry
	secretSlots                 []SecretSlot
	fixtureRecipeDigest         Digest
	readiness                   Readiness
	capturePolicyDigest         Digest
	projectionDefinition        ProjectionDefinitionBinding
	repeatSchedule              RepeatSchedule
	requiredTools               []RequiredTool
	budgets                     Budgets
}

type worldPlanIdentity struct {
	SchemaVersion               string             `json:"schema_version"`
	Kind                        string             `json:"kind"`
	CandidateSetDigest          Digest             `json:"candidate_set_digest"`
	MaterializationPolicyDigest Digest             `json:"materialization_policy_digest"`
	ComparisonEnvelopeDigest    Digest             `json:"comparison_envelope_digest"`
	Adapter                     Adapter            `json:"adapter"`
	ExecutionShape              ExecutionShape     `json:"execution_shape"`
	StartArgv                   []string           `json:"start_argv"`
	SetupArgv                   []string           `json:"setup_argv"`
	CWDPolicy                   string             `json:"cwd_policy"`
	Environment                 []EnvironmentEntry `json:"environment"`
	SecretSlots                 []SecretSlot       `json:"secret_slots"`
	FixtureRecipeDigest         Digest             `json:"fixture_recipe_digest"`
	Readiness                   Readiness          `json:"readiness"`
	NetworkMode                 string             `json:"network_mode"`
	CapturePolicyDigest         Digest             `json:"capture_policy_digest"`
	ProjectionDefinitionDigest  Digest             `json:"projection_definition_digest"`
	RepeatSchedule              repeatIdentity     `json:"repeat_schedule"`
	RequiredTools               []RequiredTool     `json:"required_tools"`
	Budgets                     Budgets            `json:"budgets"`
	EvidenceReuse               string             `json:"evidence_reuse"`
	TrustBoundary               string             `json:"trust_boundary"`
}

type repeatIdentity struct {
	DiscoveryRepeats    int    `json:"discovery_repeats"`
	ConfirmationRepeats int    `json:"confirmation_repeats"`
	Concurrency         string `json:"concurrency"`
	Rotation            string `json:"rotation"`
}

func NewWorldPlan(config WorldPlanConfig) (WorldPlan, error) {
	if err := validateWorldPlan(config); err != nil {
		return WorldPlan{}, err
	}
	adapterDigest, _, err := digestTyped("AdapterDefinition", config.Adapter)
	if err != nil {
		return WorldPlan{}, err
	}

	environment := append([]EnvironmentEntry(nil), config.Environment...)
	sort.Slice(environment, func(i, j int) bool { return environment[i].Name < environment[j].Name })
	secretSlots := append([]SecretSlot(nil), config.SecretSlots...)
	sort.Slice(secretSlots, func(i, j int) bool { return secretSlots[i].Name < secretSlots[j].Name })
	requiredTools := append([]RequiredTool(nil), config.RequiredTools...)
	sort.Slice(requiredTools, func(i, j int) bool {
		if requiredTools[i].Name == requiredTools[j].Name {
			return requiredTools[i].VersionConstraint < requiredTools[j].VersionConstraint
		}
		return requiredTools[i].Name < requiredTools[j].Name
	})

	identity := worldPlanIdentity{
		SchemaVersion:               SchemaVersion,
		Kind:                        "WorldPlan",
		CandidateSetDigest:          config.CandidateSetDigest,
		MaterializationPolicyDigest: config.MaterializationPolicyDigest,
		ComparisonEnvelopeDigest:    config.ComparisonEnvelopeDigest,
		Adapter:                     config.Adapter,
		ExecutionShape:              config.ExecutionShape,
		StartArgv:                   append([]string(nil), config.StartArgv...),
		SetupArgv:                   append([]string(nil), config.SetupArgv...),
		CWDPolicy:                   "MATERIALIZED_ROOT",
		Environment:                 environment,
		SecretSlots:                 secretSlots,
		FixtureRecipeDigest:         config.FixtureRecipeDigest,
		Readiness:                   config.Readiness,
		NetworkMode:                 "HOST_ALLOWED",
		CapturePolicyDigest:         config.CapturePolicyDigest,
		ProjectionDefinitionDigest:  config.ProjectionDefinition.Digest(),
		RepeatSchedule: repeatIdentity{
			DiscoveryRepeats:    config.RepeatSchedule.DiscoveryRepeats,
			ConfirmationRepeats: config.RepeatSchedule.ConfirmationRepeats,
			Concurrency:         "SEQUENTIAL",
			Rotation:            "NOT_ESTABLISHED_IN_U1",
		},
		RequiredTools: requiredTools,
		Budgets:       config.Budgets,
		EvidenceReuse: "FORBIDDEN",
		TrustBoundary: "TRUSTED_LOCAL_FULL_USER_PERMISSIONS",
	}
	digest, canonicalBytes, err := digestTyped("WorldPlan", identity)
	if err != nil {
		return WorldPlan{}, err
	}

	return WorldPlan{
		digest:                      digest,
		canonicalBytes:              append([]byte(nil), canonicalBytes...),
		candidateSetDigest:          config.CandidateSetDigest,
		materializationPolicyDigest: config.MaterializationPolicyDigest,
		comparisonEnvelopeDigest:    config.ComparisonEnvelopeDigest,
		adapter:                     config.Adapter,
		adapterDigest:               adapterDigest,
		executionShape:              config.ExecutionShape,
		startArgv:                   append([]string(nil), config.StartArgv...),
		setupArgv:                   append([]string(nil), config.SetupArgv...),
		environment:                 environment,
		secretSlots:                 secretSlots,
		fixtureRecipeDigest:         config.FixtureRecipeDigest,
		readiness:                   config.Readiness,
		capturePolicyDigest:         config.CapturePolicyDigest,
		projectionDefinition:        config.ProjectionDefinition,
		repeatSchedule:              config.RepeatSchedule,
		requiredTools:               requiredTools,
		budgets:                     config.Budgets,
	}, nil
}

func validateWorldPlan(config WorldPlanConfig) error {
	if !config.ProjectionDefinition.Valid() {
		return refuse(ErrInvalidWorldPlan, "world plan requires a full projection-definition binding")
	}
	if config.ProjectionDefinition.AdapterDomain() != config.Adapter.Domain {
		return refuse(ErrInvalidWorldPlan, "projection definition and adapter domain disagree")
	}
	return ValidateWorldPlanDeclaration(WorldPlanDeclarationConfig{
		CandidateSetDigest: config.CandidateSetDigest, MaterializationPolicyDigest: config.MaterializationPolicyDigest,
		ComparisonEnvelopeDigest: config.ComparisonEnvelopeDigest, Adapter: config.Adapter,
		ExecutionShape: config.ExecutionShape, StartArgv: config.StartArgv, SetupArgv: config.SetupArgv,
		Environment: config.Environment, SecretSlots: config.SecretSlots, FixtureRecipeDigest: config.FixtureRecipeDigest,
		Readiness: config.Readiness, CapturePolicyDigest: config.CapturePolicyDigest,
		ProjectionDefinitionDigest: config.ProjectionDefinition.Digest(), RepeatSchedule: config.RepeatSchedule,
		RequiredTools: config.RequiredTools, Budgets: config.Budgets,
	})
}

// ValidateWorldPlanDeclaration checks the complete static declaration profile
// but intentionally returns only an error. A digest reference that passes this
// gate is not executable authority and cannot be converted into a plan.
func ValidateWorldPlanDeclaration(config WorldPlanDeclarationConfig) error {
	for _, field := range []struct {
		name   string
		digest Digest
	}{
		{"candidate_set_digest", config.CandidateSetDigest},
		{"materialization_policy_digest", config.MaterializationPolicyDigest},
		{"comparison_envelope_digest", config.ComparisonEnvelopeDigest},
		{"runner_digest", config.Adapter.RunnerDigest},
		{"fixture_recipe_digest", config.FixtureRecipeDigest},
		{"capture_policy_digest", config.CapturePolicyDigest},
		{"projection_definition_digest", config.ProjectionDefinitionDigest},
	} {
		if !field.digest.Valid() {
			return refuse(ErrInvalidDigest, field.name)
		}
	}
	if (config.Adapter.Domain == AdapterCLI && config.Adapter.AdapterVersion != "cli/v1") ||
		(config.Adapter.Domain == AdapterHTTP && config.Adapter.AdapterVersion != "http/v1") {
		return refuse(ErrInvalidWorldPlan, "adapter version is outside the closed v1 profile")
	}
	if (config.Adapter.Domain == AdapterCLI && config.ExecutionShape != OneCLIInvocation) ||
		(config.Adapter.Domain == AdapterHTTP && config.ExecutionShape != OneLoopbackHTTPRequest) {
		return refuse(ErrInvalidWorldPlan, "adapter and execution shape disagree")
	}
	if config.Adapter.Domain != AdapterCLI && config.Adapter.Domain != AdapterHTTP {
		return refuse(ErrInvalidWorldPlan, "unsupported adapter domain")
	}
	if len(config.StartArgv) == 0 {
		return refuse(ErrInvalidWorldPlan, "start argv is empty")
	}
	if err := validateDirectArgv(config.StartArgv); err != nil {
		return err
	}
	if len(config.SetupArgv) > 0 {
		if err := validateDirectArgv(config.SetupArgv); err != nil {
			return err
		}
	}
	if config.Readiness.Kind == ReadinessNone && config.Readiness.SignalName != "" {
		return refuse(ErrInvalidWorldPlan, "NONE readiness has a signal")
	}
	if config.Readiness.Kind == FixtureOwnedReadiness && !validReadinessSignal(config.Readiness.SignalName) {
		return refuse(ErrInvalidWorldPlan, "fixture readiness signal is empty, oversized, or contains control text")
	}
	if config.Readiness.Kind != ReadinessNone && config.Readiness.Kind != FixtureOwnedReadiness {
		return refuse(ErrInvalidWorldPlan, "unsupported readiness kind")
	}
	if config.Adapter.Domain == AdapterCLI && config.Readiness.Kind != ReadinessNone {
		return refuse(ErrInvalidWorldPlan, "CLI v1 cannot declare service readiness")
	}
	if config.Adapter.Domain == AdapterHTTP && config.Readiness.Kind != FixtureOwnedReadiness {
		return refuse(ErrInvalidWorldPlan, "HTTP v1 requires fixture-owned readiness")
	}
	if err := validateBudgets(config.Budgets); err != nil {
		return err
	}
	if config.RepeatSchedule.DiscoveryRepeats < 1 || config.RepeatSchedule.DiscoveryRepeats > 5 ||
		config.RepeatSchedule.ConfirmationRepeats < 1 || config.RepeatSchedule.ConfirmationRepeats > 5 {
		return refuse(ErrInvalidWorldPlan, "repeat count outside 1..5")
	}
	if config.Adapter.Domain == AdapterHTTP && config.Budgets.ReadinessMS < 1 {
		return refuse(ErrInvalidWorldPlan, "HTTP readiness budget must be positive")
	}
	if len(config.Environment) > 5 || len(config.SecretSlots) > 16 || len(config.RequiredTools) > 16 {
		return refuse(ErrInvalidWorldPlan, "declared collection exceeds v1 bound")
	}

	seenEnvironment := map[string]struct{}{}
	for _, entry := range config.Environment {
		if !validEnvironmentName(entry.Name) {
			return refuse(ErrInvalidWorldPlan, "invalid environment name")
		}
		if _, exists := seenEnvironment[entry.Name]; exists {
			return refuse(ErrDuplicateEnvironmentName, entry.Name)
		}
		seenEnvironment[entry.Name] = struct{}{}
		if !publicEnvironmentName(entry.Name) {
			return refuse(ErrInvalidWorldPlan, entry.Name+" is not an allowed public literal environment name")
		}
		if credentialLikeName(entry.Name) {
			return refuse(ErrSecretValueInPlan, entry.Name+" must be represented only as a secret slot")
		}
		if hasAmbientInterpolation(entry.Value) {
			return refuse(ErrAmbientInterpolation, entry.Name)
		}
		if !publicEnvironmentLiteral(entry.Name, entry.Value) {
			return refuse(ErrInvalidWorldPlan, entry.Name+" value is outside the closed public-literal profile")
		}
	}
	seenSecrets := map[string]struct{}{}
	for _, slot := range config.SecretSlots {
		if !validEnvironmentName(slot.Name) || len(slot.Name) > 128 {
			return refuse(ErrInvalidWorldPlan, "invalid secret-slot name")
		}
		if slot.Presence != SecretAbsent && slot.Presence != SecretRequiredNoCapture {
			return refuse(ErrInvalidWorldPlan, "invalid secret-slot presence")
		}
		if _, exists := seenSecrets[slot.Name]; exists {
			return refuse(ErrInvalidWorldPlan, "duplicate secret slot")
		}
		seenSecrets[slot.Name] = struct{}{}
		if _, captured := seenEnvironment[slot.Name]; captured {
			return refuse(ErrSecretValueInPlan, slot.Name)
		}
	}
	seenTools := map[string]struct{}{}
	for _, tool := range config.RequiredTools {
		if !validRequiredToolName(tool.Name) || !validVersionConstraint(tool.VersionConstraint) {
			return refuse(ErrInvalidWorldPlan, "empty tool requirement")
		}
		if _, exists := seenTools[tool.Name]; exists {
			return refuse(ErrInvalidWorldPlan, "duplicate tool requirement")
		}
		seenTools[tool.Name] = struct{}{}
	}
	for _, argv := range [][]string{config.StartArgv, config.SetupArgv} {
		if len(argv) == 0 {
			continue
		}
		command := argv[0]
		if filepath.Base(command) == command && !strings.Contains(command, "/") {
			if _, declared := seenTools[command]; !declared {
				return refuse(ErrInvalidWorldPlan, "bare host command is absent from required_tools: "+command)
			}
		}
	}
	minimumTrials := config.Budgets.CandidateCount *
		(config.RepeatSchedule.DiscoveryRepeats + config.RepeatSchedule.ConfirmationRepeats)
	if config.Budgets.TotalCandidateTrials < minimumTrials {
		return refuse(ErrInvalidWorldPlan, "trial budget cannot cover discovery and confirmation schedules")
	}
	return nil
}

func validRequiredToolName(name string) bool {
	if len(name) < 1 || len(name) > 128 {
		return false
	}
	for index := 0; index < len(name); index++ {
		character := name[index]
		alphaNumeric := character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
		if alphaNumeric || (index > 0 && (character == '.' || character == '_' || character == '+' || character == '-')) {
			continue
		}
		return false
	}
	return true
}

func validVersionConstraint(constraint string) bool {
	if len(constraint) < 1 || len(constraint) > 256 || !utf8.ValidString(constraint) || strings.TrimSpace(constraint) == "" {
		return false
	}
	for _, character := range constraint {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validReadinessSignal(signal string) bool {
	if len(signal) < 1 || len(signal) > 128 || !utf8.ValidString(signal) || strings.TrimSpace(signal) == "" {
		return false
	}
	for _, character := range signal {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validateDirectArgv(argv []string) error {
	if len(argv) == 0 || argv[0] == "" {
		return refuse(ErrInvalidWorldPlan, "direct command is empty")
	}
	if filepath.Base(argv[0]) == "env" {
		return refuse(ErrShellString, "env wrappers are outside the closed direct-exec profile")
	}
	for _, argument := range argv {
		base := filepath.Base(argument)
		if base == "sh" || base == "bash" || base == "zsh" || base == "fish" || base == "dash" ||
			base == "csh" || base == "tcsh" || base == "ksh" || base == "pwsh" || base == "powershell" {
			return refuse(ErrShellString, argument)
		}
	}
	if filepath.Base(argv[0]) != argv[0] || strings.ContainsAny(argv[0], `/\\`) || strings.HasPrefix(argv[0], "-") {
		return refuse(ErrInvalidWorldPlan, "argv[0] must be one bare required-tool name")
	}
	if len(argv) > 64 {
		return refuse(ErrInvalidWorldPlan, "argv exceeds v1 argument count")
	}
	totalBytes := 0
	for index, argument := range argv {
		totalBytes += len(argument)
		if !utf8.ValidString(argument) || strings.IndexByte(argument, 0) >= 0 {
			return refuse(ErrInvalidWorldPlan, "argv contains NUL")
		}
		if len(argument) > 4096 || totalBytes > 65536 {
			return refuse(ErrInvalidWorldPlan, "argv exceeds v1 byte bound")
		}
		if filepath.IsAbs(argument) {
			return refuse(ErrInvalidWorldPlan, "absolute argv element")
		}
		if strings.HasPrefix(argument, "~") || strings.Contains(argument, "\\") || filepath.VolumeName(argument) != "" {
			return refuse(ErrInvalidWorldPlan, "argv token uses unsupported path syntax")
		}
		for _, character := range argument {
			if unicode.IsControl(character) {
				return refuse(ErrInvalidWorldPlan, "argv token contains a control character")
			}
		}
		if index > 0 && strings.HasPrefix(argument, "-") {
			if !strings.HasPrefix(argument, "--") || len(argument) < 3 || strings.ContainsAny(argument, `/=`) {
				return refuse(ErrInvalidWorldPlan, "option is outside the closed long-flag profile")
			}
			for _, character := range argument[2:] {
				if !((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
					(character >= '0' && character <= '9') || character == '.' || character == '_' || character == '-') {
					return refuse(ErrInvalidWorldPlan, "long flag contains an unsupported character")
				}
			}
		}
		if index > 0 && strings.Contains(argument, "/") {
			if strings.HasPrefix(argument, "/") || filepath.Clean(argument) != argument || argument == "." ||
				strings.HasPrefix(argument, "../") || strings.Contains(argument, "/../") {
				return refuse(ErrInvalidWorldPlan, "repo-relative argv path is not clean")
			}
		}
		if hasAmbientInterpolation(argument) {
			return refuse(ErrAmbientInterpolation, "argv")
		}
		for _, prefix := range []string{"--output=", "--out=", "-o="} {
			if strings.HasPrefix(argument, prefix) && filepath.IsAbs(strings.TrimPrefix(argument, prefix)) {
				return refuse(ErrInvalidWorldPlan, "absolute output path")
			}
		}
		if equals := strings.IndexByte(argument, '='); equals >= 0 && filepath.IsAbs(argument[equals+1:]) {
			return refuse(ErrInvalidWorldPlan, "absolute embedded option value")
		}
		if argument == "--output" || argument == "--out" || argument == "-o" {
			if index+1 >= len(argv) || argv[index+1] == "" {
				return refuse(ErrInvalidWorldPlan, "output flag is missing a path")
			}
			if filepath.IsAbs(argv[index+1]) {
				return refuse(ErrInvalidWorldPlan, "absolute output path")
			}
		}
	}
	return nil
}

func hasAmbientInterpolation(value string) bool {
	return strings.Contains(value, "$") || strings.Contains(value, "`")
}

func credentialLikeName(name string) bool {
	upper := strings.ToUpper(name)
	for _, marker := range []string{"TOKEN", "PASSWORD", "PASSWD", "SECRET", "API_KEY", "PRIVATE_KEY", "CREDENTIAL", "AUTH"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

// WorldPlan environment values are identity-bearing bytes. Countershape cannot
// detect arbitrary secrets from their values, so v1 accepts only a closed set
// of conventional public literals.
// Runtime-owned HOME/TMP/PATH/ports and secret material are represented by
// WorldInstance allocations or SecretSlot presence records instead.
func publicEnvironmentName(name string) bool {
	switch name {
	case "LANG", "LC_ALL", "TZ", "NO_COLOR", "NODE_NO_WARNINGS":
		return true
	default:
		return false
	}
}

func publicEnvironmentLiteral(name, value string) bool {
	switch name {
	case "LANG", "LC_ALL":
		return value == "C"
	case "TZ":
		return value == "UTC"
	case "NO_COLOR", "NODE_NO_WARNINGS":
		return value == "1"
	default:
		return false
	}
}

func validEnvironmentName(name string) bool {
	if name == "" {
		return false
	}
	for index, r := range name {
		if index == 0 && !((r >= 'A' && r <= 'Z') || r == '_') {
			return false
		}
		if index > 0 && !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func validateBudgets(b Budgets) error {
	valid := b.CandidateCount >= 2 && b.CandidateCount <= 4 &&
		b.MaterializedEntryCount >= 1 && b.MaterializedEntryCount <= 100000 &&
		b.MaterializedBytesPerWorld >= 1 && b.MaterializedBytesPerWorld <= 1073741824 &&
		b.SingleBlobBytes >= 1 && b.SingleBlobBytes <= 134217728 &&
		b.ReadinessMS >= 0 && b.ReadinessMS <= 30000 &&
		b.ProbeMS >= 1 && b.ProbeMS <= 30000 &&
		b.TeardownMS >= 1 && b.TeardownMS <= 10000 &&
		b.StdoutBytes >= 1 && b.StdoutBytes <= 16777216 &&
		b.StderrBytes >= 1 && b.StderrBytes <= 16777216 &&
		b.HTTPBodyBytes >= 1 && b.HTTPBodyBytes <= 16777216 &&
		b.ProposedShrinkStimuli >= 0 && b.ProposedShrinkStimuli <= 200 &&
		b.TotalCandidateTrials >= 1 && b.TotalCandidateTrials <= 2000 &&
		b.ShrinkWallMS >= 1 && b.ShrinkWallMS <= 3600000
	if !valid || b.SingleBlobBytes > b.MaterializedBytesPerWorld {
		return refuse(ErrInvalidWorldPlan, "budget outside v1 limits")
	}
	return nil
}

func (p WorldPlan) Digest() Digest { return p.digest }

func (p WorldPlan) CanonicalBytes() []byte { return append([]byte(nil), p.canonicalBytes...) }

// CandidateSetDigest exposes the exact pre-plan candidate-set declaration that
// this immutable plan consumed. It is a reference, not executable authority;
// an impure substrate must still bind inspected candidates to the actual plan.
func (p WorldPlan) CandidateSetDigest() Digest { return p.candidateSetDigest }

func (p WorldPlan) StartArgv() []string { return append([]string(nil), p.startArgv...) }

func (p WorldPlan) SetupArgv() []string { return append([]string(nil), p.setupArgv...) }

func (p WorldPlan) Environment() []EnvironmentEntry {
	return append([]EnvironmentEntry(nil), p.environment...)
}

func (p WorldPlan) SecretSlots() []SecretSlot { return append([]SecretSlot(nil), p.secretSlots...) }

func (p WorldPlan) RequiredTools() []RequiredTool {
	return append([]RequiredTool(nil), p.requiredTools...)
}

func (p WorldPlan) Budgets() Budgets { return p.budgets }

func (p WorldPlan) Adapter() Adapter { return p.adapter }

func (p WorldPlan) AdapterDigest() Digest { return p.adapterDigest }

func (p WorldPlan) MaterializationPolicyDigest() Digest { return p.materializationPolicyDigest }

func (p WorldPlan) ComparisonEnvelopeDigest() Digest { return p.comparisonEnvelopeDigest }

func (p WorldPlan) ProjectionDefinitionDigest() Digest { return p.projectionDefinition.Digest() }

func (p WorldPlan) ProjectionDefinitionBinding() ProjectionDefinitionBinding {
	return p.projectionDefinition
}

func (p WorldPlan) ExecutionShape() ExecutionShape { return p.executionShape }

func (p WorldPlan) RepeatSchedule() RepeatSchedule { return p.repeatSchedule }
