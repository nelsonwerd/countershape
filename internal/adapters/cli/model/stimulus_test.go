package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

func testDigest(character string) domain.Digest {
	return domain.MustDigest("sha256:" + strings.Repeat(character, 64))
}

func testProjectionBinding(t *testing.T) domain.ProjectionDefinitionBinding {
	t.Helper()
	binding, _ := testProjectionPair(t)
	return binding
}

func testProjectionPair(t *testing.T) (domain.ProjectionDefinitionBinding, CLIProjectionAuthority) {
	return testProjectionPairWithRegistry(t, "a")
}

func testProjectionPairWithRegistry(t *testing.T, registryCharacter string) (domain.ProjectionDefinitionBinding, CLIProjectionAuthority) {
	t.Helper()
	fields := []string{"cli.stdout.bytes"}
	configurationDigest, _, err := digestTyped("CLIProjectionConfiguration", struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Fields        []string `json:"fields"`
	}{domain.SchemaVersion, "CLIProjectionConfiguration", "cli-projection/v1", fields})
	if err != nil {
		t.Fatal(err)
	}
	implementationRaw, err := canon.DigestBytes(
		"CLIProjectionImplementation",
		[]byte("cli-projection/v1\x00closed-field-registry\x00visible-pure-operations\x00exact-canonical"),
	)
	if err != nil {
		t.Fatal(err)
	}
	implementationDigest, err := domain.ParseDigest(implementationRaw.String())
	if err != nil {
		t.Fatal(err)
	}
	operation := resolvedProjectionOperation{
		Name:      "cli.require-eligible-capture/v1",
		Semantics: "test-visible-semantics",
	}
	ruleRaw, err := canon.DigestBytes(
		"CLIProjectionOperationRule", []byte(operation.Name+"\x00"+operation.Semantics),
	)
	if err != nil {
		t.Fatal(err)
	}
	operation.RuleDigest = ruleRaw.String()
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterCLI, ImplementationDigest: implementationDigest,
		ConfigurationDigest: configurationDigest, AcceptedChannels: []string{"exit", "stderr", "stdout"},
		Operations: []domain.ProjectionOperationBinding{{Name: operation.Name, RuleDigest: domain.MustDigest(operation.RuleDigest)}},
		Comparator: domain.ProjectionComparatorExact, FieldRegistryDigest: testDigest(registryCharacter),
	})
	if err != nil {
		t.Fatal(err)
	}
	identity := resolvedProjectionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIProjectionDefinition", Version: "cli-projection/v1",
		Fields: fields, Operations: []resolvedProjectionOperation{operation},
		FieldRegistryDigest:     binding.FieldRegistryDigest().String(),
		ImplementationDigest:    binding.ImplementationDigest().String(),
		ConfigurationDigest:     binding.ConfigurationDigest().String(),
		ProjectionBindingDigest: binding.Digest().String(),
	}
	canonical, err := canon.CanonicalizeTyped(identity)
	if err != nil {
		t.Fatal(err)
	}
	digestRaw, err := canon.DigestBytes("CLIProjectionDefinition", canonical)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		t.Fatal(err)
	}
	authority, err := ResolveCLIProjectionAuthority(digest, canonical, binding)
	if err != nil {
		t.Fatal(err)
	}
	return binding, authority
}

func testCapturePolicy(t *testing.T) CLICapturePolicy {
	t.Helper()
	policy, err := NewCLICapturePolicy(CLICapturePolicyConfig{StdoutBytes: 1024, StderrBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func testProjectionAuthority(t *testing.T) CLIProjectionAuthority {
	t.Helper()
	_, authority := testProjectionPair(t)
	return authority
}

func testPlan(t *testing.T, startArgv []string) domain.WorldPlan {
	t.Helper()
	recipe, err := NewCLIFixtureRecipe()
	if err != nil {
		t.Fatal(err)
	}
	policy := testCapturePolicy(t)
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest: testDigest("1"), MaterializationPolicyDigest: testDigest("2"),
		ComparisonEnvelopeDigest: testDigest("3"),
		Adapter:                  domain.Adapter{Domain: domain.AdapterCLI, AdapterVersion: "cli/v1", RunnerDigest: testDigest("4")},
		ExecutionShape:           domain.OneCLIInvocation, StartArgv: startArgv, SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}}, SecretSlots: []domain.SecretSlot{},
		FixtureRecipeDigest: recipe.Digest(), Readiness: domain.Readiness{Kind: domain.ReadinessNone},
		CapturePolicyDigest: policy.Digest(), ProjectionDefinition: testProjectionBinding(t),
		RepeatSchedule: domain.RepeatSchedule{DiscoveryRepeats: 2, ConfirmationRepeats: 2,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 100, MaterializedBytesPerWorld: 1 << 20,
			SingleBlobBytes: 1 << 19, ReadinessMS: 0, ProbeMS: 1000, TeardownMS: 1000,
			StdoutBytes: 1024, StderrBytes: 1024, HTTPBodyBytes: 1024,
			ProposedShrinkStimuli: 10, TotalCandidateTrials: 20, ShrinkWallMS: 10000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func testStimulus(t *testing.T, argv []string, stdin CLIStdin, environment []CLIEnvironmentBinding, fixtures []CLIFixtureFile) CLIStimulus {
	t.Helper()
	stimulus, err := NewCLIStimulus(CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture/precedence.mjs"}, Argv: argv,
		Stdin: stdin, Environment: environment, Fixtures: fixtures, CWDPolicy: CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	return stimulus
}

func TestCLIStimulusPreservesOrderAndTaggedAbsence(t *testing.T) {
	absentEnv, err := AbsentEnvironment("APP_MODE")
	if err != nil {
		t.Fatal(err)
	}
	presentEmptyEnv, err := PresentEnvironment("APP_MODE", "")
	if err != nil {
		t.Fatal(err)
	}
	absent := testStimulus(t, []string{"--first", "a", "--second", "b"}, AbsentStdin(), []CLIEnvironmentBinding{absentEnv}, nil)
	presentEmptyStdin, err := PresentStdin([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	present := testStimulus(t, []string{"--first", "a", "--second", "b"}, presentEmptyStdin, []CLIEnvironmentBinding{presentEmptyEnv}, nil)
	reordered := testStimulus(t, []string{"--second", "b", "--first", "a"}, AbsentStdin(), []CLIEnvironmentBinding{absentEnv}, nil)

	if absent.Digest() == present.Digest() || absent.ExecutionPayloadDigest() == present.ExecutionPayloadDigest() {
		t.Fatal("absent input collapsed into present-empty input")
	}
	if absent.Digest() == reordered.Digest() || absent.ExecutionPayloadDigest() == reordered.ExecutionPayloadDigest() {
		t.Fatal("argv order did not affect exact identity")
	}
	want := []string{"node", "fixture/precedence.mjs", "--first", "a", "--second", "b"}
	if got := absent.LogicalArgv(); !equalStrings(got, want) {
		t.Fatalf("logical argv = %#v, want %#v", got, want)
	}
	var payload struct {
		LogicalArgv []string `json:"logical_argv"`
	}
	if err := json.Unmarshal(absent.ExecutionPayloadCanonicalBytes(), &payload); err != nil {
		t.Fatalf("decode canonical execution payload: %v", err)
	}
	if !equalStrings(payload.LogicalArgv, want) {
		t.Fatalf("canonical execution payload argv = %#v, want %#v", payload.LogicalArgv, want)
	}
	if !absent.Valid() || !present.Valid() || !reordered.Valid() {
		t.Fatal("constructed stimulus did not retain construction authority")
	}
}

func TestCLIStimulusRejectsDirectArgvGrammarBypasses(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		baseArgv   []string
		argv       []string
	}{
		{"shell executable", "sh", nil, nil},
		{"shell payload", "node", []string{"fixture.mjs"}, []string{"bash"}},
		{"absolute path", "node", []string{"fixture.mjs"}, []string{"/tmp/value"}},
		{"parent traversal", "node", []string{"fixture.mjs"}, []string{"../escape"}},
		{"unclean nested path", "node", []string{"fixture.mjs"}, []string{"nested/../escape"}},
		{"home shorthand", "node", []string{"fixture.mjs"}, []string{"~/.config"}},
		{"backslash path", "node", []string{"fixture.mjs"}, []string{`bad\path`}},
		{"ambient dollar", "node", []string{"fixture.mjs"}, []string{"$HOME"}},
		{"ambient command substitution", "node", []string{"fixture.mjs"}, []string{"`whoami`"}},
		{"control character", "node", []string{"fixture.mjs"}, []string{"line\nbreak"}},
		{"short flag", "node", []string{"fixture.mjs"}, []string{"-x"}},
		{"embedded long-flag value", "node", []string{"fixture.mjs"}, []string{"--mode=value"}},
		{"node eval script string", "node", []string{"--eval", "process.exit(0)"}, nil},
		{"node stdin program", "node", nil, nil},
		{"node non-script base", "node", []string{"README.md"}, nil},
		{"unprofiled awk script string", "awk", []string{"--source", `BEGIN { system("id") }`}, nil},
	}
	for _, test := range tests {
		_, err := NewCLIStimulus(CLIStimulusConfig{
			Executable: test.executable,
			BaseArgv:   test.baseArgv,
			Argv:       test.argv,
			Stdin:      AbsentStdin(),
			CWDPolicy:  CWDMaterializedRoot,
		})
		if err == nil {
			t.Fatalf("%s: closed direct-argv grammar accepted executable=%q base=%q argv=%q", test.name, test.executable, test.baseArgv, test.argv)
		}
	}
	valid := testStimulus(t, []string{"--mode", "argv"}, AbsentStdin(), nil, nil)
	if !valid.Valid() {
		t.Fatal("closed direct-argv grammar rejected the admitted CLI profile")
	}
}

func TestCLIStimulusDigestBackedByteArtifactsReachDeclaredBoundary(t *testing.T) {
	stdinBytes := bytes.Repeat([]byte("s"), maxStdinBytes)
	stdin, err := PresentStdin(stdinBytes)
	if err != nil {
		t.Fatal(err)
	}
	fixtures := make([]CLIFixtureFile, 16)
	for index := range fixtures {
		contents := bytes.Repeat([]byte{byte(index)}, maxFixtureFileBytes)
		fixtures[index], err = NewFixtureFile(fmt.Sprintf("data/%02d.bin", index), contents, FixtureMode0644)
		if err != nil {
			t.Fatal(err)
		}
	}
	stimulus, err := NewCLIStimulus(CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture.mjs"}, Stdin: stdin,
		Fixtures: fixtures, CWDPolicy: CWDMaterializedRoot,
	})
	if err != nil || !stimulus.Valid() {
		t.Fatalf("declared byte-artifact boundary was not constructible: %v", err)
	}
	if len(stimulus.CanonicalBytes()) >= 1<<20 || len(stimulus.ExecutionPayloadCanonicalBytes()) >= 1<<20 {
		t.Fatal("byte artifacts were embedded into canonical stimulus identity instead of digest-backed")
	}

	changedStdin := append([]byte(nil), stdinBytes...)
	changedStdin[len(changedStdin)-1] ^= 0xff
	changed, err := PresentStdin(changedStdin)
	if err != nil {
		t.Fatal(err)
	}
	changedStimulus, err := NewCLIStimulus(CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture.mjs"}, Stdin: changed,
		Fixtures: fixtures, CWDPolicy: CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stimulus.ExecutionPayloadDigest() == changedStimulus.ExecutionPayloadDigest() || stimulus.Digest() == changedStimulus.Digest() {
		t.Fatal("same-length stdin byte substitution did not change digest-backed stimulus identity")
	}
}

func TestCLIStimulusNormalizesEnvironmentSetButNotValues(t *testing.T) {
	a, _ := PresentEnvironment("APP_MODE", "config")
	b, _ := AbsentEnvironment("OPTIONAL_MODE")
	left := testStimulus(t, nil, AbsentStdin(), []CLIEnvironmentBinding{a, b}, nil)
	right := testStimulus(t, nil, AbsentStdin(), []CLIEnvironmentBinding{b, a}, nil)
	if left.Digest() != right.Digest() || !bytes.Equal(left.CanonicalBytes(), right.CanonicalBytes()) {
		t.Fatal("sparse environment set depends on caller order")
	}
	bindings := left.Environment()
	if len(bindings) != 2 || bindings[0].Name() != "APP_MODE" || bindings[1].Name() != "OPTIONAL_MODE" {
		t.Fatalf("environment not normalized: %#v", bindings)
	}
	if _, err := PresentEnvironment("HOME", "/ambient"); err == nil {
		t.Fatal("runner-owned HOME was accepted")
	}
}

func TestCLIStimulusEnvironmentAggregateMatchesCanonicalHeadroom(t *testing.T) {
	const entries = 32
	names := make([]string, entries)
	nameBytes := 0
	for index := range names {
		names[index] = fmt.Sprintf("ENV_%02d", index)
		nameBytes += len(names[index])
	}
	valueBytes := (maxEnvironmentTotalBytes - nameBytes) / entries
	if valueBytes > maxEnvironmentValue || valueBytes*entries+nameBytes != maxEnvironmentTotalBytes {
		t.Fatal("test vector does not land on the exact aggregate environment boundary")
	}
	bindings := make([]CLIEnvironmentBinding, entries)
	for index, name := range names {
		binding, err := PresentEnvironment(name, strings.Repeat("v", valueBytes))
		if err != nil {
			t.Fatal(err)
		}
		bindings[index] = binding
	}
	stimulus := testStimulus(t, nil, AbsentStdin(), bindings, nil)
	if !stimulus.Valid() {
		t.Fatal("exact aggregate environment boundary lost stimulus authority")
	}
	escaped := make([]CLIEnvironmentBinding, entries)
	for index, name := range names {
		binding, err := PresentEnvironment(name, strings.Repeat("<", valueBytes))
		if err != nil {
			t.Fatal(err)
		}
		escaped[index] = binding
	}
	escapedStimulus := testStimulus(t, nil, AbsentStdin(), escaped, nil)
	if !escapedStimulus.Valid() || len(escapedStimulus.CanonicalBytes()) >= 1<<20 ||
		len(escapedStimulus.ExecutionPayloadCanonicalBytes()) >= 1<<20 {
		t.Fatal("escape-heavy exact environment boundary exceeded canonical identity headroom")
	}

	over := append([]CLIEnvironmentBinding(nil), bindings...)
	over[0], _ = PresentEnvironment(names[0], strings.Repeat("v", valueBytes+1))
	if _, err := NewCLIStimulus(CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture/precedence.mjs"},
		Stdin: AbsentStdin(), Environment: over, CWDPolicy: CWDMaterializedRoot,
	}); err == nil {
		t.Fatal("environment above the aggregate identity byte ceiling was accepted")
	}
}

func TestCLIStimulusFixtureValidationIsStrict(t *testing.T) {
	a, _ := NewFixtureFile("config/app.json", []byte("{}"), FixtureMode0644)
	b, _ := NewFixtureFile("input.txt", []byte("x"), FixtureMode0644)
	attributes, attributesErr := NewFixtureFile(".gitattributes", []byte("*.txt -text\n"), FixtureMode0644)
	if attributesErr != nil {
		t.Fatalf("ordinary dotfile adjacent to reserved .git namespace was rejected: %v", attributesErr)
	}
	stimulus := testStimulus(t, nil, AbsentStdin(), nil, []CLIFixtureFile{attributes, a, b})
	if !stimulus.Valid() || stimulus.Measure().FixtureFiles() != 3 || stimulus.Measure().FixtureContentBytes() != 15 {
		t.Fatal("valid ordered fixture set lost measure or authority")
	}
	if _, err := NewCLIStimulus(CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture.mjs"}, Stdin: AbsentStdin(),
		Fixtures: []CLIFixtureFile{b, a}, CWDPolicy: CWDMaterializedRoot,
	}); err == nil {
		t.Fatal("out-of-order fixture set was accepted")
	}
	parent, _ := NewFixtureFile("config", []byte("x"), FixtureMode0644)
	if _, err := NewCLIStimulus(CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture.mjs"}, Stdin: AbsentStdin(),
		Fixtures: []CLIFixtureFile{parent, a}, CWDPolicy: CWDMaterializedRoot,
	}); err == nil {
		t.Fatal("file/directory prefix collision was accepted")
	}
	for _, path := range []string{
		"../escape", "/absolute", "a//b", "a\\b", "a/./b",
		".git", ".GIT/config", "nested/.Git/objects/x",
	} {
		if _, err := NewFixtureFile(path, nil, FixtureMode0644); err == nil {
			t.Fatalf("unsafe fixture path %q was accepted", path)
		}
	}
}

func TestCLIStimulusCopiesAllCallerOwnedBytes(t *testing.T) {
	stdinBytes := []byte("stdin")
	stdin, _ := PresentStdin(stdinBytes)
	fixtureBytes := []byte("fixture")
	fixture, _ := NewFixtureFile("input.txt", fixtureBytes, FixtureMode0644)
	argv := []string{"--mode", "argv"}
	stimulus := testStimulus(t, argv, stdin, nil, []CLIFixtureFile{fixture})
	digest := stimulus.Digest()
	stdinBytes[0] = 'X'
	fixtureBytes[0] = 'X'
	argv[0] = "--changed"
	gotStdin := stimulus.Stdin().Bytes()
	gotFixture := stimulus.Fixtures()[0].Contents()
	gotStdin[0] = 'Y'
	gotFixture[0] = 'Y'
	if stimulus.Digest() != digest || !stimulus.Valid() || string(stimulus.Stdin().Bytes()) != "stdin" || string(stimulus.Fixtures()[0].Contents()) != "fixture" {
		t.Fatal("caller mutation changed immutable stimulus authority")
	}
}

func TestBindExecutionRequiresExactPlanPrefixAndRetainsPayload(t *testing.T) {
	stimulus := testStimulus(t, []string{"--mode", "argv"}, AbsentStdin(), nil, nil)
	plan := testPlan(t, []string{"node", "fixture/precedence.mjs"})
	binding, err := BindExecution(plan, stimulus, testCapturePolicy(t), testProjectionAuthority(t))
	if err != nil {
		t.Fatal(err)
	}
	if !binding.Valid() || binding.PlanDigest() != plan.Digest() || binding.StimulusDigest() != stimulus.Digest() ||
		binding.ExecutionPayloadDigest() != stimulus.ExecutionPayloadDigest() ||
		!equalStrings(binding.LogicalArgv(), stimulus.LogicalArgv()) {
		t.Fatal("execution binding did not retain exact plan/stimulus authority")
	}
	wrong := testPlan(t, []string{"node", "fixture/other.mjs"})
	if _, err := BindExecution(wrong, stimulus, testCapturePolicy(t), testProjectionAuthority(t)); err == nil {
		t.Fatal("plan with a different fixed argv prefix was accepted")
	}
}

func TestBindExecutionResolvesExactCaptureAndProjectionAuthorities(t *testing.T) {
	stimulus := testStimulus(t, []string{"--mode", "argv"}, AbsentStdin(), nil, nil)
	plan := testPlan(t, []string{"node", "fixture/precedence.mjs"})
	policy := testCapturePolicy(t)
	projection := testProjectionAuthority(t)
	binding, err := BindExecution(plan, stimulus, policy, projection)
	if err != nil || !binding.Valid() || binding.CapturePolicyDigest() != policy.Digest() ||
		binding.AdapterProjectionDefinitionDigest() != projection.Digest() {
		t.Fatalf("exact resolved execution contracts were not retained: binding=%+v err=%v", binding, err)
	}

	wrongPolicy, err := NewCLICapturePolicy(CLICapturePolicyConfig{StdoutBytes: 1024, StderrBytes: 2048})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BindExecution(plan, stimulus, wrongPolicy, projection); err == nil {
		t.Fatal("independently valid but cross-paired capture policy resolved against the plan")
	}
	_, wrongProjection := testProjectionPairWithRegistry(t, "c")
	if _, err := BindExecution(plan, stimulus, policy, wrongProjection); err == nil {
		t.Fatal("independently valid but cross-paired adapter projection resolved against the plan")
	}
}

func TestReducerNeutralMeasurePreservesPresentEmptyUnit(t *testing.T) {
	presentEmpty, _ := PresentStdin(nil)
	absent := testStimulus(t, nil, AbsentStdin(), nil, nil).Measure()
	present := testStimulus(t, nil, presentEmpty, nil, nil).Measure()
	if absent.StdinPresenceUnits() != 0 || present.StdinPresenceUnits() != 1 || present.StdinBytes() != 0 {
		t.Fatal("measure collapsed present-empty stdin into absence")
	}
	if len(present.WellFoundedTuple()) == 0 {
		t.Fatal("measure did not expose well-founded tuple data")
	}
}
