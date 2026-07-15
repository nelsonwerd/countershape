//go:build darwin && cgo

package world

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
)

type cliBridgeProjectionOperation struct {
	Name       string `json:"name"`
	Semantics  string `json:"semantics"`
	RuleDigest string `json:"rule_digest"`
}

type cliBridgeProjectionIdentity struct {
	SchemaVersion           string                         `json:"schema_version"`
	Kind                    string                         `json:"kind"`
	Version                 string                         `json:"version"`
	Fields                  []string                       `json:"fields"`
	Operations              []cliBridgeProjectionOperation `json:"operations"`
	FieldRegistryDigest     string                         `json:"field_registry_digest"`
	ImplementationDigest    string                         `json:"implementation_digest"`
	ConfigurationDigest     string                         `json:"configuration_digest"`
	ProjectionBindingDigest string                         `json:"projection_definition_binding_digest"`
}

func cliBridgeProjectionPair(t *testing.T) (domain.ProjectionDefinitionBinding, climodel.CLIProjectionAuthority) {
	t.Helper()
	fields := []string{"cli.stdout.bytes"}
	configurationRaw, _, err := canon.DigestTyped("CLIProjectionConfiguration", struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Version       string   `json:"version"`
		Fields        []string `json:"fields"`
	}{domain.SchemaVersion, "CLIProjectionConfiguration", "cli-projection/v1", fields})
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := domain.ParseDigest(configurationRaw.String())
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
	implementation, err := domain.ParseDigest(implementationRaw.String())
	if err != nil {
		t.Fatal(err)
	}
	operation := cliBridgeProjectionOperation{
		Name: "cli.require-eligible-capture/v1", Semantics: "world-test-visible-semantics",
	}
	ruleRaw, err := canon.DigestBytes(
		"CLIProjectionOperationRule", []byte(operation.Name+"\x00"+operation.Semantics),
	)
	if err != nil {
		t.Fatal(err)
	}
	operation.RuleDigest = ruleRaw.String()
	binding, err := domain.NewProjectionDefinitionBinding(domain.ProjectionDefinitionBindingConfig{
		AdapterDomain: domain.AdapterCLI, ImplementationDigest: implementation,
		ConfigurationDigest: configuration, AcceptedChannels: []string{"exit", "stderr", "stdout"},
		Operations: []domain.ProjectionOperationBinding{{Name: operation.Name, RuleDigest: domain.MustDigest(operation.RuleDigest)}},
		Comparator: domain.ProjectionComparatorExact, FieldRegistryDigest: worldTestDigest("b"),
	})
	if err != nil {
		t.Fatal(err)
	}
	identity := cliBridgeProjectionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIProjectionDefinition", Version: "cli-projection/v1",
		Fields: fields, Operations: []cliBridgeProjectionOperation{operation},
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
	authority, err := climodel.ResolveCLIProjectionAuthority(digest, canonical, binding)
	if err != nil {
		t.Fatal(err)
	}
	return binding, authority
}

func cliBridgeBinding(
	t *testing.T,
	stdin climodel.CLIStdin,
	environment []climodel.CLIEnvironmentBinding,
	fixtures []climodel.CLIFixtureFile,
) climodel.CLIExecutionBinding {
	t.Helper()
	recipe, err := climodel.NewCLIFixtureRecipe()
	if err != nil {
		t.Fatal(err)
	}
	capturePolicy, err := climodel.NewCLICapturePolicy(climodel.CLICapturePolicyConfig{
		StdoutBytes: 1 << 16, StderrBytes: 1 << 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	projectionBinding, projectionAuthority := cliBridgeProjectionPair(t)
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest:          worldTestDigest("1"),
		MaterializationPolicyDigest: worldTestDigest("2"),
		ComparisonEnvelopeDigest:    worldTestDigest("3"),
		Adapter: domain.Adapter{
			Domain: domain.AdapterCLI, AdapterVersion: "cli/v1", RunnerDigest: worldTestDigest("4"),
		},
		ExecutionShape:       domain.OneCLIInvocation,
		StartArgv:            []string{"node", "fixture/bridge.mjs", "--mode", "report"},
		SetupArgv:            []string{},
		Environment:          []domain.EnvironmentEntry{{Name: "LANG", Value: "C"}},
		SecretSlots:          []domain.SecretSlot{},
		FixtureRecipeDigest:  recipe.Digest(),
		Readiness:            domain.Readiness{Kind: domain.ReadinessNone},
		CapturePolicyDigest:  capturePolicy.Digest(),
		ProjectionDefinition: projectionBinding,
		RepeatSchedule: domain.RepeatSchedule{DiscoveryRepeats: 2, ConfirmationRepeats: 2,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 32, MaterializedBytesPerWorld: 1 << 20,
			SingleBlobBytes: 1 << 18, ProbeMS: 1000, TeardownMS: 1000,
			StdoutBytes: 1 << 16, StderrBytes: 1 << 16, HTTPBodyBytes: 1 << 16,
			ProposedShrinkStimuli: 4, TotalCandidateTrials: 8, ShrinkWallMS: 10_000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	stimulus, err := climodel.NewCLIStimulus(climodel.CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{"fixture/bridge.mjs", "--mode", "report"}, Argv: []string{"--u3"},
		Stdin: stdin, Environment: environment, Fixtures: fixtures, CWDPolicy: climodel.CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := climodel.BindExecution(plan, stimulus, capturePolicy, projectionAuthority)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func TestCLIFixtureOverlayBindsRecipeAndReopensExactBytes(t *testing.T) {
	fixture, err := climodel.NewFixtureFile("config/input.txt", []byte("opaque\n"), climodel.FixtureMode0644)
	if err != nil {
		t.Fatal(err)
	}
	binding := cliBridgeBinding(t, climodel.AbsentStdin(), nil, []climodel.CLIFixtureFile{fixture})
	root := resolvedPrivateTempDir(t)
	receipt, err := materializeCLIFixtures(binding, root)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Valid() || receipt.Authority() != climodel.CLIFixtureOverlayAuthority ||
		receipt.FixtureRecipeDigest() != binding.FixtureRecipeDigest() || len(receipt.Entries()) != 1 {
		t.Fatalf("fixture receipt lost recipe authority: %+v", receipt)
	}
	tamperedRoot := receipt
	tamperedRoot.root += "-tampered"
	if tamperedRoot.Valid() {
		t.Fatal("fixture overlay receipt accepted a self-inconsistent root")
	}
	tamperedEntry := receipt
	tamperedEntry.entries = append([]CLIFixtureEntryReceipt(nil), receipt.entries...)
	tamperedEntry.entries[0].bytes++
	if tamperedEntry.Valid() {
		t.Fatal("fixture overlay receipt accepted altered entry facts")
	}
	tamperedCanonical := receipt
	tamperedCanonical.canonicalBytes = append(tamperedCanonical.CanonicalBytes(), '\n')
	if tamperedCanonical.Valid() {
		t.Fatal("fixture overlay receipt accepted noncanonical sealed bytes")
	}
	path := filepath.Join(root, "config", "input.txt")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o644 {
		t.Fatalf("fixture facts = %+v, %v", info, err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != "opaque\n" {
		t.Fatalf("fixture bytes = %q, %v", bytes, err)
	}
	if _, err := materializeCLIFixtures(binding, root); err == nil {
		t.Fatal("exclusive fixture materialization overwrote an existing destination")
	}
}

func TestCLIEnvironmentIsSparseScheduledAndCollisionClosed(t *testing.T) {
	absent, err := climodel.AbsentEnvironment("OPTIONAL_MODE")
	if err != nil {
		t.Fatal(err)
	}
	present, err := climodel.PresentEnvironment("APP_MODE", "probe")
	if err != nil {
		t.Fatal(err)
	}
	roots := processRoots(t)
	fixtureRoot := filepath.Join(roots.attempt, "fixture")
	if err := os.Mkdir(fixtureRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	roots.fixture = fixtureRoot
	environment, err := buildCLIEnvironment(
		[]domain.EnvironmentEntry{{Name: "LANG", Value: "C"}},
		[]climodel.CLIEnvironmentBinding{present, absent}, roots, "attempt:u3", 3, 2,
	)
	if err != nil {
		t.Fatal(err)
	}
	joined := "\x00" + strings.Join(environment, "\x00") + "\x00"
	for _, want := range []string{
		"APP_MODE=probe", "COUNTERSHAPE_FIXTURE_ROOT=" + fixtureRoot,
		"COUNTERSHAPE_SCHEDULE_ORDINAL=3", "COUNTERSHAPE_SCHEDULE_REPETITION=1",
	} {
		if !strings.Contains(joined, "\x00"+want+"\x00") {
			t.Fatalf("environment omitted %q: %#v", want, environment)
		}
	}
	if strings.Contains(joined, "\x00OPTIONAL_MODE=") {
		t.Fatalf("ABSENT stimulus environment became a process value: %#v", environment)
	}
	collision, err := climodel.PresentEnvironment("LANG", "other")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildCLIEnvironment(
		[]domain.EnvironmentEntry{{Name: "LANG", Value: "C"}},
		[]climodel.CLIEnvironmentBinding{collision}, roots, "attempt:u3", 0, 2,
	); err == nil {
		t.Fatal("stimulus environment overrode an immutable plan entry")
	}
	for _, name := range []string{"NODE_OPTIONS", "PYTHONPATH", "GODEBUG", "DYLD_INSERT_LIBRARIES"} {
		if !reservedCLIEnvironmentName(name) {
			t.Fatalf("runner-owned environment %q was not reserved", name)
		}
	}
}

func TestCLIInvocationEvidenceRequiresAttemptAndFullLogicalArgv(t *testing.T) {
	binding := cliBridgeBinding(t, climodel.AbsentStdin(), nil, nil)
	evidenceRoot := resolvedPrivateTempDir(t)
	wantArgv := binding.LogicalArgv()
	bytes := []byte(`{"attempt_id":"attempt:u3","logical_argv":["node","fixture/bridge.mjs","--mode","report","--u3"]}`)
	if err := os.WriteFile(filepath.Join(evidenceRoot, cliInvocationFilename), bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := inspectCLIInvocationEvidence(binding, evidenceRoot, "attempt:u3", wantArgv)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Valid() || receipt.Status() != CLIInvocationValidated || !receipt.AttemptIDValidated() ||
		!receipt.LogicalArgvValidated() || !receipt.FileDigest().Valid() {
		t.Fatalf("valid invocation evidence was not bound: %+v", receipt)
	}
	bytes = []byte(`{"attempt_id":"attempt:u3","logical_argv":["node","fixture/bridge.mjs","--mode","report"]}`)
	if err := os.WriteFile(filepath.Join(evidenceRoot, cliInvocationFilename), bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err = inspectCLIInvocationEvidence(binding, evidenceRoot, "attempt:u3", wantArgv)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status() != CLIInvocationSchemaMismatch || receipt.LogicalArgvValidated() {
		t.Fatalf("truncated logical argv was accepted: %+v", receipt)
	}
}

func resealCLIInvocationReceipt(t *testing.T, receipt CLIInvocationEvidenceReceipt) CLIInvocationEvidenceReceipt {
	t.Helper()
	digest, canonicalBytes, err := canon.DigestTyped("CLIInvocationEvidenceReceipt", receipt.identity())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		t.Fatal(err)
	}
	receipt.digest = parsed
	receipt.canonicalBytes = append([]byte(nil), canonicalBytes...)
	return receipt
}

func TestCLIInvocationReceiptRejectsSelfSealedStatusContradictions(t *testing.T) {
	binding := cliBridgeBinding(t, climodel.AbsentStdin(), nil, nil)
	evidenceRoot := resolvedPrivateTempDir(t)
	wantArgv := binding.LogicalArgv()
	path := filepath.Join(evidenceRoot, cliInvocationFilename)

	if err := os.WriteFile(path, []byte(`{"attempt_id":"attempt:u3","logical_argv":["node","fixture/bridge.mjs","--mode","report","--u3"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	validated, err := inspectCLIInvocationEvidence(binding, evidenceRoot, "attempt:u3", wantArgv)
	if err != nil || !validated.Valid() || validated.Presence() != CLIInvocationPresencePresent {
		t.Fatalf("validated receipt invalid: %+v, %v", validated, err)
	}
	changedCanonical := validated
	changedCanonical.canonicalBytes = append(changedCanonical.CanonicalBytes(), '\n')
	if changedCanonical.Valid() {
		t.Fatal("noncanonical receipt bytes survived identity recomputation")
	}

	contradictoryAbsent := validated
	contradictoryAbsent.status = CLIInvocationAbsent
	contradictoryAbsent.presence = CLIInvocationPresencePresent
	contradictoryAbsent.fileDigest = ""
	contradictoryAbsent.bytes = 0
	contradictoryAbsent.attemptIDValidated = false
	contradictoryAbsent.logicalArgvValidated = false
	contradictoryAbsent = resealCLIInvocationReceipt(t, contradictoryAbsent)
	if contradictoryAbsent.Valid() {
		t.Fatal("self-sealed ABSENT receipt claimed a present file")
	}

	if err := os.WriteFile(path, []byte(`{"attempt_id":`), 0o600); err != nil {
		t.Fatal(err)
	}
	malformed, err := inspectCLIInvocationEvidence(binding, evidenceRoot, "attempt:u3", wantArgv)
	if err != nil || !malformed.Valid() || malformed.Status() != CLIInvocationMalformed || !malformed.FileDigest().Valid() {
		t.Fatalf("malformed receipt facts invalid: %+v, %v", malformed, err)
	}
	malformed.fileDigest = ""
	malformed = resealCLIInvocationReceipt(t, malformed)
	if malformed.Valid() {
		t.Fatal("self-sealed malformed receipt omitted the reopened-byte digest")
	}

	if err := os.WriteFile(path, make([]byte, maxCLIInvocationBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	tooLarge, err := inspectCLIInvocationEvidence(binding, evidenceRoot, "attempt:u3", wantArgv)
	if err != nil || !tooLarge.Valid() || tooLarge.Status() != CLIInvocationTooLarge || tooLarge.Bytes() <= maxCLIInvocationBytes {
		t.Fatalf("too-large receipt facts invalid: %+v, %v", tooLarge, err)
	}
	tooLarge.bytes = maxCLIInvocationBytes
	tooLarge = resealCLIInvocationReceipt(t, tooLarge)
	if tooLarge.Valid() {
		t.Fatal("self-sealed TOO_LARGE receipt omitted the observed oversized length")
	}

	unknownRead := validated
	unknownRead.status = CLIInvocationReadFailed
	unknownRead.presence = CLIInvocationPresenceUnknown
	unknownRead.fileDigest = ""
	unknownRead.bytes = 0
	unknownRead.attemptIDValidated = false
	unknownRead.logicalArgvValidated = false
	unknownRead = resealCLIInvocationReceipt(t, unknownRead)
	if !unknownRead.Valid() {
		t.Fatal("READ_FAILED could not preserve unknown path presence")
	}
	unknownRead.presence = CLIInvocationPresenceAbsent
	unknownRead = resealCLIInvocationReceipt(t, unknownRead)
	if unknownRead.Valid() {
		t.Fatal("READ_FAILED overclaimed known path absence")
	}
}

type cliShortEPIPEWriter struct {
	written int
}

func (w *cliShortEPIPEWriter) Write(bytes []byte) (int, error) {
	if w.written == 0 && len(bytes) > 1 {
		w.written++
		return 1, nil
	}
	return 0, syscall.EPIPE
}

func (*cliShortEPIPEWriter) Close() error { return nil }

func TestBindPhysicalCLIStdinRejectsPresenceCollapseAndSubstitution(t *testing.T) {
	digest := func(value []byte) domain.Digest {
		t.Helper()
		encoded, err := canon.DigestBytes("CLIStdinBytes", value)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := domain.ParseDigest(encoded.String())
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	presentEmpty := processCLILineage{
		stdinPresence: "PRESENT", stdinDigest: digest(nil), stdinDelivery: "NOT_APPLIED",
	}
	physicalPresentEmpty := physicalProcessResult{
		physicalExecutionEntered: true, started: true,
		stdinPresence: processStdinPresent, stdinDigest: digest(nil),
		stdinPipeAllocated: true, stdinWriterStarted: true, stdinHandoffAttempted: true, stdinComplete: true,
	}
	if err := bindPhysicalCLIStdin(&presentEmpty, physicalPresentEmpty); err != nil {
		t.Fatalf("exact present-empty physical stdin was rejected: %v", err)
	}
	if !presentEmpty.stdinPipeAllocated || !presentEmpty.stdinWriterStarted ||
		!presentEmpty.stdinHandoffAttempted || !presentEmpty.stdinDeliveryComplete {
		t.Fatalf("present-empty physical edges were not bound: %+v", presentEmpty)
	}
	spawnFailed := processCLILineage{
		stdinPresence: "PRESENT", stdinDigest: digest(nil), stdinDelivery: "NOT_APPLIED",
	}
	spawnFailurePhysical := physicalProcessResult{
		physicalExecutionEntered: true, spawnAttempted: true, started: false,
		stdinPresence: processStdinPresent, stdinDigest: digest(nil), stdinPipeAllocated: true,
	}
	if err := bindPhysicalCLIStdin(&spawnFailed, spawnFailurePhysical); err != nil ||
		spawnFailed.stdinDelivery != "NOT_APPLIED" || !spawnFailed.stdinPipeAllocated ||
		spawnFailed.stdinWriterStarted || spawnFailed.stdinHandoffAttempted || spawnFailed.stdinDeliveryComplete {
		t.Fatalf("failed Start claimed stdin delivery: lineage=%+v err=%v", spawnFailed, err)
	}
	absentSubstitute := physicalPresentEmpty
	absentSubstitute.stdinPresence = processStdinAbsent
	absentExpected := processCLILineage{
		stdinPresence: "PRESENT", stdinDigest: digest(nil), stdinDelivery: "NOT_APPLIED",
	}
	if err := bindPhysicalCLIStdin(&absentExpected, absentSubstitute); err == nil {
		t.Fatal("absent stdin substituted for present-empty stdin")
	}
	expectedBytes := processCLILineage{
		stdinPresence: "PRESENT", stdinBytes: 2, stdinDigest: digest([]byte("ab")),
		stdinDelivery: "NOT_APPLIED",
	}
	sameLengthSubstitute := physicalProcessResult{
		physicalExecutionEntered: true, started: true,
		stdinPresence: processStdinPresent, stdinDeclared: 2, stdinDigest: digest([]byte("cd")),
		stdinPipeAllocated: true, stdinWriterStarted: true, stdinHandoffAttempted: true,
		stdinWritten: 2, stdinComplete: true,
	}
	if err := bindPhysicalCLIStdin(&expectedBytes, sameLengthSubstitute); err == nil {
		t.Fatal("same-length stdin substitution escaped physical digest authority")
	}
}

func TestCLIExplicitStdinPipeReceiptsShortEPIPE(t *testing.T) {
	results := make(chan stdinWriteResult, 1)
	writeExactStdin(&cliShortEPIPEWriter{}, []byte("opaque"), results)
	result := <-results
	if result.complete || result.written != 1 || result.errorCode != "EPIPE" {
		t.Fatalf("short EPIPE was not reported honestly: %+v", result)
	}
	stdinDigest, err := canon.DigestBytes("CLIStdinBytes", nil)
	if err != nil {
		t.Fatal(err)
	}
	parsedStdinDigest, err := domain.ParseDigest(stdinDigest.String())
	if err != nil {
		t.Fatal(err)
	}
	lineage := processCLILineage{
		authority: CLIExecutionAuthorityV1, declaredLogicalArgv: []string{"fixture"},
		executionBindingDigest: worldTestDigest("1"), stimulusDigest: worldTestDigest("2"),
		executionPayloadDigest: worldTestDigest("3"), fixtureRecipeDigest: worldTestDigest("4"),
		stdinPresence: "ABSENT", stdinDigest: parsedStdinDigest, stdinDelivery: "NOT_APPLIED",
		scheduleOrdinal: 3, candidateCount: 2, scheduleRepetition: 1,
		invocationEvidenceStatus:   CLIInvocationNotInspected,
		invocationEvidencePresence: CLIInvocationPresenceUnknown,
	}
	if !lineage.valid() {
		t.Fatal("valid ordinal/candidate-count/repetition lineage was rejected")
	}
	lineage.invocationEvidencePresence = CLIInvocationPresenceAbsent
	if lineage.valid() {
		t.Fatal("NOT_INSPECTED invocation lineage accepted known absence")
	}
	lineage.invocationEvidencePresence = CLIInvocationPresenceUnknown
	lineage.scheduleRepetition = 0
	if lineage.valid() {
		t.Fatal("schedule repetition escaped ordinal/candidate-count binding")
	}
}

func TestCLIPreprocessReceiptCannotForgePhysicalExecutionMutationGuard(t *testing.T) {
	executable := buildProcessFixture(t)
	harness := newExecutionHarness(
		t, executable, []string{"fixture", "--mode", "report"}, 1000, 400, 1<<16, 1<<16,
	)
	request := harness.request("u3-preprocess-physical-guard")
	allocated, err := allocateAttempt(request)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := allocated.domainAttempt.Advance(domain.AttemptMaterializing)
	if err != nil {
		t.Fatal(err)
	}
	tool, err := harness.registry.resolvePlan(harness.plan)
	if err != nil {
		t.Fatal(err)
	}
	stdinDigest, err := digestProcessStdin(processStdin{presence: processStdinAbsent})
	if err != nil {
		t.Fatal(err)
	}
	declaredArgv := []string{"fixture", "--mode", "report", "--declared-only"}
	lineage := processCLILineage{
		authority: CLIExecutionAuthorityV1, declaredLogicalArgv: declaredArgv,
		executionBindingDigest: worldTestDigest("1"), stimulusDigest: worldTestDigest("2"),
		executionPayloadDigest: worldTestDigest("3"), fixtureRecipeDigest: worldTestDigest("4"),
		cwdPolicy: "MATERIALIZED_ROOT", stdinPresence: "ABSENT", stdinDigest: stdinDigest,
		stdinDelivery: "NOT_APPLIED", scheduleOrdinal: 0, candidateCount: 2, scheduleRepetition: 0,
		invocationEvidenceStatus: CLIInvocationNotInspected, invocationEvidencePresence: CLIInvocationPresenceUnknown,
	}
	result, err := finalizeWithoutProcessWithLineageAndTool(
		allocated, attempt, gitobj.MaterializationReceipt{}, domain.ControlMissingObject,
		withReceiptDiagnostic(diagnosticMaterializationFailed, os.ErrNotExist), tool, lineage,
	)
	if err != nil {
		t.Fatal(err)
	}
	process := result.Process()
	if process.PhysicalExecutionEntered() || len(process.LogicalArgv()) != 0 ||
		strings.Join(process.DeclaredLogicalArgv(), "\x00") != strings.Join(declaredArgv, "\x00") ||
		process.StdoutCaptureLimit() != 0 || process.StderrCaptureLimit() != 0 ||
		process.StdinDelivery() != "NOT_APPLIED" || process.StdinPipeAllocated() ||
		process.StdinWriterStarted() || process.StdinHandoffAttempted() || process.StdinWrittenBytes() != 0 ||
		process.StdinDeliveryComplete() || process.InvocationEvidenceStatus() != CLIInvocationNotInspected ||
		process.InvocationEvidencePresence() != CLIInvocationPresenceUnknown {
		t.Fatalf("pre-process receipt forged physically applied execution: %+v", process)
	}
}
