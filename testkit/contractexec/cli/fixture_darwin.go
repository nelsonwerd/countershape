//go:build darwin && arm64 && cgo

// Package cli provides one public-constructor-only C4 contract-execution
// fixture. It is test infrastructure, never an alternate product authority.
package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/contractexec"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	node "github.com/nelsonwerd/countershape/internal/emit/node"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/hostepoch"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/clifixture"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

const (
	SystemGit               = "/usr/bin/git"
	SystemNode              = "/opt/homebrew/bin/node"
	ReferenceStimulusDigest = "sha256:ed949de9ef5e290b68164b1a1453aec8f95fe5ad8c80643fc528c81871a7e254"
)

type headWire struct {
	SchemaVersion        string `json:"schema_version"`
	Kind                 string `json:"kind"`
	StudyID              string `json:"study_id"`
	Revision             int64  `json:"revision"`
	Stage                string `json:"stage"`
	CurrentKind          string `json:"current_kind"`
	CurrentDigest        string `json:"current_digest"`
	LineageRootDigest    string `json:"lineage_root_digest"`
	PreviousHeadDigest   string `json:"previous_head_digest"`
	PreviousObjectDigest string `json:"previous_object_digest"`
}

type choicepointRevealWire struct {
	CandidateExecutionKey string `json:"candidate_execution_key"`
	DisplayRef            string `json:"display_ref"`
	ProducerMetadata      string `json:"producer_metadata"`
}

type choicepointWire struct {
	ComparisonEnvelopeBase64 string                  `json:"comparison_envelope_base64"`
	CandidateBindingsBase64  []string                `json:"candidate_bindings_base64"`
	CandidateReveals         []choicepointRevealWire `json:"candidate_reveals"`
	OriginalStimulusKind     string                  `json:"original_stimulus_kind"`
	OriginalStimulusBase64   string                  `json:"original_stimulus_base64"`
	OriginalStimulusDigest   string                  `json:"original_stimulus_digest"`
	EvidenceReceipts         []domain.ReceiptWire    `json:"evidence_receipts"`
}

type residueFixture struct {
	store   *store.ObjectStore
	root    string
	residue node.Residue
}

type sourceRepositoryFixture struct {
	root string
	ref  string
}

// Fixture retains the one capability consumed by black-box C4 tests. The
// public StoreRoot is diagnostic test state, never enough to mint a target.
type Fixture struct {
	Target    contractexec.OfficialTarget
	StoreRoot string
}

var sharedCompilation struct {
	once           sync.Once
	controlOnce    sync.Once
	conformingOnce sync.Once
	forbiddenOnce  sync.Once
	sourceMu       sync.Mutex
	sources        map[string]sourceRepositoryFixture
	root           string
	residue        residueFixture
	control        residueFixture
	conforming     residueFixture
	forbidden      residueFixture
}

var sharedAdmissionTemplate struct {
	once  sync.Once
	input contractmodel.TargetInput
}

// AdmissionFixture is a fresh store-bound target for the runner's physical
// admission tests. It reuses only a once-live measured TargetInput template;
// every call owns a new private store, attempt, boot measurement, target
// record, and interlock namespace.
type AdmissionFixture struct {
	Target store.ContractTargetRecord
	Epoch  hostepoch.Epoch
}

// RunMain releases the process-scoped compiled and immutable-source fixtures
// after a test binary's full -count schedule. Sharing ruling/residue and
// content-addressed Git objects makes the 50x/20x stability gates practical;
// every call still publishes a fresh materialization, live runtime admission,
// attempt, boot binding, target record, interlock, and process run.
func RunMain(m *testing.M) int {
	code := m.Run()
	if sharedCompilation.root != "" {
		_ = os.RemoveAll(sharedCompilation.root)
	}
	return code
}

// ReferenceFiles returns the exact two-file C4 standalone reference subject.
func ReferenceFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	files, err := clifixture.CandidateFiles(clifixture.ArgvFirst)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// ImportCanaryFiles returns a subject that positively reaches the runner's
// outside-candidate import canary before emitting a valid projected tuple.
func ImportCanaryFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFiles(t, []byte(`import { pathToFileURL } from "node:url";
const modulePath = process.env.COUNTERSHAPE_C4_IMPORT_CANARY_MODULE;
if (typeof modulePath !== "string" || modulePath.length === 0) throw new Error("missing import canary");
await import(pathToFileURL(modulePath).href);
process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

// ServiceCanaryFiles returns a subject that positively reaches the runner's
// private service canary before emitting a valid projected tuple.
func ServiceCanaryFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFiles(t, []byte(`import { connect } from "node:net";
const socket = process.env.COUNTERSHAPE_C4_SERVICE_CANARY_SOCKET;
if (typeof socket !== "string" || socket.length === 0) throw new Error("missing service canary");
await new Promise((resolve, reject) => {
  const client = connect(socket, () => { client.end(); resolve(); });
  client.once("error", reject);
});
process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

// ForbiddenCanaryFiles returns one subject that positively reaches both
// independently measured forbidden-resource canaries before emitting a valid
// projected tuple. Keeping both observations in one real child preserves the
// detector evidence while avoiding a second compiled-target execution in the
// repeated C4 qualification.
func ForbiddenCanaryFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFiles(t, []byte(`import { connect } from "node:net";
import { pathToFileURL } from "node:url";
const modulePath = process.env.COUNTERSHAPE_C4_IMPORT_CANARY_MODULE;
const socket = process.env.COUNTERSHAPE_C4_SERVICE_CANARY_SOCKET;
if (typeof modulePath !== "string" || modulePath.length === 0) throw new Error("missing import canary");
if (typeof socket !== "string" || socket.length === 0) throw new Error("missing service canary");
await import(pathToFileURL(modulePath).href);
await new Promise((resolve, reject) => {
  const client = connect(socket, () => client.end());
  client.once("close", resolve);
  client.once("error", reject);
});
process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

// ChildBindingMismatchFiles writes a well-formed invocation receipt whose
// attempt identity is guaranteed to differ from the runner-owned value.
func ChildBindingMismatchFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFilesWithReceipt(t, invocationReceiptPrelude(true, false), []byte(`process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

// ChildArgvMismatchFiles writes a well-formed invocation receipt whose
// logical argv differs while retaining the exact runner-owned attempt ID.
func ChildArgvMismatchFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFilesWithReceipt(t, invocationReceiptPrelude(false, true), []byte(`process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

// MalformedInvocationFiles writes bounded non-JSON into the fixed invocation
// evidence leaf while otherwise completing a valid projection.
func MalformedInvocationFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFilesWithReceipt(t, malformedInvocationPrelude(), []byte(`process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

// MissingInvocationFiles completes normally without fabricating child-binding
// evidence. The runner must keep that domain missing rather than clean.
func MissingInvocationFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFilesWithReceipt(t, nil, []byte(`process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

// TargetMutationFiles leaves a positive candidate-tree mutation after writing
// a valid invocation receipt. Terminal OfficialTarget reopening must refuse it.
func TargetMutationFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFiles(t, []byte(`import { writeFileSync } from "node:fs";
writeFileSync("unexpected-c4-mutation.txt", "mutation", { encoding: "utf8", flag: "wx", mode: 0o600 });
process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

// CancellationFiles keeps the child live long enough for tests to observe
// durable admission before canceling the caller context.
func CancellationFiles(t testing.TB) []gitrepo.File {
	t.Helper()
	return positiveControlFiles(t, []byte(`await new Promise((resolve) => setTimeout(resolve, 1000));
process.stdout.write('{"mode":"argv","source":"argv"}');
`))
}

func positiveControlFiles(t testing.TB, program []byte) []gitrepo.File {
	t.Helper()
	return positiveControlFilesWithReceipt(t, invocationReceiptPrelude(false, false), program)
}

func positiveControlFilesWithReceipt(t testing.TB, receipt, program []byte) []gitrepo.File {
	t.Helper()
	role := ReferenceFiles(t)[0]
	subject := append([]byte(nil), receipt...)
	subject = append(subject, program...)
	return []gitrepo.File{
		{Path: role.Path, Mode: role.Mode, Content: append([]byte(nil), role.Content...)},
		{Path: clifixture.Entrypoint, Mode: "100755", Content: subject},
	}
}

func invocationReceiptPrelude(attemptMismatch, argvMismatch bool) []byte {
	attempt := `const invocationAttempt = requiredEnvironment("COUNTERSHAPE_ATTEMPT_ID");
`
	if attemptMismatch {
		attempt = `const observedAttempt = requiredEnvironment("COUNTERSHAPE_ATTEMPT_ID");
const replacement = observedAttempt[8] === "0" ? "1" : "0";
const invocationAttempt = observedAttempt.slice(0, 8) + replacement + observedAttempt.slice(9);
`
	}
	logicalArgv := `const invocationLogicalArgv = ["node", basename(process.argv[1]), ...process.argv.slice(2)];
`
	if argvMismatch {
		logicalArgv = `const invocationLogicalArgv = ["node", "different-entrypoint.mjs", ...process.argv.slice(2)];
`
	}
	return []byte(`import { closeSync, fsyncSync, openSync, writeSync } from "node:fs";
import { basename, join } from "node:path";
function requiredEnvironment(name) {
  const value = process.env[name];
  if (typeof value !== "string" || value.length === 0) throw new Error("missing " + name);
  return value;
}
function writeInvocationAll(fd, bytes) {
  let offset = 0;
  while (offset < bytes.length) {
    const count = writeSync(fd, bytes, offset, bytes.length - offset);
    if (count < 1) throw new Error("invocation receipt write made no progress");
    offset += count;
  }
}
` + attempt + `const invocationRoot = requiredEnvironment("COUNTERSHAPE_EVIDENCE_ROOT");
` + logicalArgv + `const invocationBytes = Buffer.from(JSON.stringify({ attempt_id: invocationAttempt, logical_argv: invocationLogicalArgv }), "utf8");
const invocationFD = openSync(join(invocationRoot, "cli-invocation.json"), "wx", 0o600);
writeInvocationAll(invocationFD, invocationBytes);
fsyncSync(invocationFD);
closeSync(invocationFD);
const invocationDirectoryFD = openSync(invocationRoot, "r");
fsyncSync(invocationDirectoryFD);
closeSync(invocationDirectoryFD);
`)
}

func malformedInvocationPrelude() []byte {
	return []byte(`import { closeSync, fsyncSync, openSync, writeSync } from "node:fs";
import { join } from "node:path";
const invocationRoot = process.env.COUNTERSHAPE_EVIDENCE_ROOT;
if (typeof invocationRoot !== "string" || invocationRoot.length === 0) throw new Error("missing evidence root");
const invocationBytes = Buffer.from("not-json", "utf8");
const invocationFD = openSync(join(invocationRoot, "cli-invocation.json"), "wx", 0o600);
let invocationOffset = 0;
while (invocationOffset < invocationBytes.length) {
  const count = writeSync(invocationFD, invocationBytes, invocationOffset, invocationBytes.length - invocationOffset);
  if (count < 1) throw new Error("invocation receipt write made no progress");
  invocationOffset += count;
}
fsyncSync(invocationFD);
closeSync(invocationFD);
const invocationDirectoryFD = openSync(invocationRoot, "r");
fsyncSync(invocationDirectoryFD);
closeSync(invocationDirectoryFD);
`)
}

// NewReferenceTarget builds the exact enrolled reference target through the
// real ruling, compiler, Git materializer, and OfficialTarget issuers.
func NewReferenceTarget(t testing.TB) Fixture {
	t.Helper()
	return newTarget(t, sharedResidue(t), ReferenceFiles(t), "C4 CLI reference target")
}

// NewConformingTarget compiles a second real ruling that selects the exact
// argv-first reference tuple, then publishes the same enrolled reference tree.
// It exists to exercise the physical CONFORMS branch without changing the
// historical reference contract, whose exact ruling remains CONTRADICTS.
func NewConformingTarget(t testing.TB) Fixture {
	t.Helper()
	return newTarget(t, sharedConformingResidue(t), ReferenceFiles(t), "C4 CLI conforming physical target")
}

// NewForbiddenTarget publishes the exact combined forbidden-resource control
// against an independently stored copy of the reference contract. The bundle
// must be byte-identical to the reference ruling, while the separate store
// keeps the store-wide execution interlock meaningful under bounded test load.
func NewForbiddenTarget(t testing.TB) Fixture {
	t.Helper()
	return newTarget(t, sharedForbiddenResidue(t), ForbiddenCanaryFiles(t), "C4 forbidden canaries")
}

// NewAdmissionFixture avoids repeating C3's expensive executable hashing in
// the 50x C4 admission qualification while preserving the exact C2 store,
// C4 owner, and real process-mechanics edges under test.
func NewAdmissionFixture(t testing.TB) AdmissionFixture {
	t.Helper()
	sharedAdmissionTemplate.once.Do(func() {
		reference := NewReferenceTarget(t)
		sharedAdmissionTemplate.input = reference.Target.Model().Input()
	})
	if !sharedAdmissionTemplate.input.ContractBundleDigest.Valid() {
		t.Fatal("shared C4 admission template did not initialize")
	}
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	objectStore, err := store.OpenObjectStore(filepath.Join(parent, "admission-store"))
	if err != nil {
		t.Fatal(err)
	}
	epoch, err := hostepoch.Measure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	input := sharedAdmissionTemplate.input
	attempt, err := objectStore.AllocateConformanceAttempt(context.Background(), store.ConformanceAttemptInput{
		ContractBundleDigest:        input.ContractBundleDigest,
		ResidueHeadDigest:           input.TerminalResidue.HeadDigest,
		TreeIdentityDigest:          input.Tree.TreeIdentityDigest,
		MaterializationPolicyDigest: input.Tree.MaterializationPolicyDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	input.Attempt = contractmodel.AttemptBinding{
		ArtifactDigest: attempt.Digest(), InstanceNonce: attempt.InstanceNonce(),
	}
	input.BootSession = contractmodel.BootSessionBinding{IdentityDigest: epoch.Digest()}
	target, err := contractmodel.NewContractExecutionTarget(input)
	if err != nil {
		t.Fatal(err)
	}
	record, err := objectStore.PersistContractTargetRecord(context.Background(), attempt, target)
	if err != nil || !record.Valid() {
		t.Fatalf("persist C4 admission target: %v", err)
	}
	return AdmissionFixture{Target: record, Epoch: epoch}
}

// NewTarget builds a target with the exact sealed ruling but caller-supplied
// selected Git tree. It exists only so hostile standalone controls exercise
// the same real target graph as the reference subject.
func NewTarget(t testing.TB, files []gitrepo.File, label string) Fixture {
	t.Helper()
	return newTarget(t, sharedControlResidue(t), files, label)
}

// NewMutationTarget gives a terminal target-drift control its own one-shot
// store. The expected refusal leaves admission held, so that store must never
// poison another -count iteration or an unrelated hostile control.
func NewMutationTarget(t testing.TB, files []gitrepo.File, label string) Fixture {
	t.Helper()
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mutation := newResidue(t, parent, "fresh C4 target-mutation CLI contract", nil)
	reference := sharedResidue(t)
	if mutation.store == nil || !mutation.residue.Valid() || mutation.store == reference.store || mutation.root == reference.root ||
		mutation.residue.BundleDigest() != reference.residue.BundleDigest() {
		t.Fatal("fresh C4 target-mutation compilation is not an independent exact reference contract")
	}
	return newTarget(t, mutation, files, label)
}

func newTarget(t testing.TB, residue residueFixture, files []gitrepo.File, label string) Fixture {
	t.Helper()
	if _, err := os.Lstat(SystemNode); err != nil {
		t.Skipf("exact Node unavailable: %v", err)
	}
	source := sharedSourceRepository(t, files)
	repository, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{
		GitExecutable: SystemGit,
		Repository:    source.root,
		ScratchRoot:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	target, err := contractexec.PublishOfficialTarget(context.Background(), contractexec.PublishOfficialTargetRequest{
		Store: residue.store, Residue: residue.residue, Repository: repository,
		DisplayRef: source.ref, NodeExecutable: SystemNode,
	})
	if err != nil || !target.Valid() {
		t.Fatalf("publish %s: %v", label, err)
	}
	t.Cleanup(func() { _ = target.Close() })
	return Fixture{Target: target, StoreRoot: residue.root}
}

func sharedSourceRepository(t testing.TB, files []gitrepo.File) sourceRepositoryFixture {
	t.Helper()
	exact, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := canon.DigestBytes("C4SourceRepositoryFixture", exact)
	if err != nil {
		t.Fatal(err)
	}
	key := digest.String()

	sharedCompilation.sourceMu.Lock()
	defer sharedCompilation.sourceMu.Unlock()
	if source, ok := sharedCompilation.sources[key]; ok {
		return source
	}
	if sharedCompilation.root == "" {
		t.Fatal("shared C4 compilation root did not initialize before source caching")
	}
	parent := filepath.Join(sharedCompilation.root, "sources")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	gitFixture, err := gitrepo.Init(context.Background(), SystemGit, parent, gitrepo.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := gitFixture.CommitFiles(context.Background(), files, "", "C4 cached source "+key)
	if err != nil {
		t.Fatal(err)
	}
	const ref = "refs/heads/target"
	if err := gitFixture.UpdateRef(context.Background(), ref, commit); err != nil {
		t.Fatal(err)
	}
	if sharedCompilation.sources == nil {
		sharedCompilation.sources = make(map[string]sourceRepositoryFixture)
	}
	source := sourceRepositoryFixture{root: gitFixture.Root, ref: ref}
	sharedCompilation.sources[key] = source
	return source
}

func sharedResidue(t testing.TB) residueFixture {
	t.Helper()
	sharedCompilation.once.Do(func() {
		parent, err := os.MkdirTemp("", "countershape-c4-compiled-")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(parent, 0o700); err != nil {
			_ = os.RemoveAll(parent)
			t.Fatal(err)
		}
		sharedCompilation.root = parent
		sharedCompilation.residue = newResidue(t, parent, "shared C4 CLI contract", nil)
	})
	if sharedCompilation.residue.store == nil || !sharedCompilation.residue.residue.Valid() {
		t.Fatal("shared C4 compilation did not initialize")
	}
	return sharedCompilation.residue
}

func sharedControlResidue(t testing.TB) residueFixture {
	t.Helper()
	reference := sharedResidue(t)
	sharedCompilation.controlOnce.Do(func() {
		parent := filepath.Join(sharedCompilation.root, "controls")
		if err := os.Mkdir(parent, 0o700); err != nil {
			t.Fatal(err)
		}
		sharedCompilation.control = newResidue(t, parent, "shared C4 hostile-control CLI contract", nil)
	})
	control := sharedCompilation.control
	if control.store == nil || !control.residue.Valid() || control.store == reference.store || control.root == reference.root ||
		control.residue.BundleDigest() != reference.residue.BundleDigest() {
		t.Fatal("shared C4 hostile-control compilation is not an independent exact reference contract")
	}
	return control
}

func sharedConformingResidue(t testing.TB) residueFixture {
	t.Helper()
	reference := sharedResidue(t)
	sharedCompilation.conformingOnce.Do(func() {
		parent := filepath.Join(sharedCompilation.root, "conforming")
		if err := os.Mkdir(parent, 0o700); err != nil {
			t.Fatal(err)
		}
		sharedCompilation.conforming = newResidue(t, parent, "shared C4 conforming CLI contract", map[string]string{
			string(countercli.CLIFieldStdoutJSONMode):   "argv",
			string(countercli.CLIFieldStdoutJSONSource): "argv",
		})
	})
	if sharedCompilation.conforming.store == nil || !sharedCompilation.conforming.residue.Valid() ||
		sharedCompilation.conforming.store == reference.store || sharedCompilation.conforming.root == reference.root ||
		sharedCompilation.conforming.residue.BundleDigest() == reference.residue.BundleDigest() {
		t.Fatal("shared C4 conforming compilation did not initialize")
	}
	return sharedCompilation.conforming
}

func sharedForbiddenResidue(t testing.TB) residueFixture {
	t.Helper()
	reference := sharedResidue(t)
	sharedCompilation.forbiddenOnce.Do(func() {
		parent := filepath.Join(sharedCompilation.root, "forbidden")
		if err := os.Mkdir(parent, 0o700); err != nil {
			t.Fatal(err)
		}
		sharedCompilation.forbidden = newResidue(t, parent, "shared C4 forbidden CLI contract", nil)
	})
	forbidden := sharedCompilation.forbidden
	if forbidden.store == nil || !forbidden.residue.Valid() || forbidden.store == reference.store || forbidden.root == reference.root ||
		forbidden.residue.BundleDigest() != reference.residue.BundleDigest() {
		t.Fatal("shared C4 forbidden compilation is not an independent exact reference contract")
	}
	return forbidden
}

func repoRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("source location unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func example(t testing.TB, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repoRoot(t), "spec", "examples", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	if len(body) > 0 && body[len(body)-1] == '\n' {
		body = append([]byte(nil), body[:len(body)-1]...)
	}
	return body
}

func publishObject(t testing.TB, objectStore *store.ObjectStore, kind string, digest domain.Digest, exact []byte) {
	t.Helper()
	object, err := store.NewSemanticObject(kind, digest, exact)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := objectStore.Publish(context.Background(), object)
	if err != nil || objectStore.Validate(context.Background(), object, authority) != nil {
		t.Fatalf("publish %s: %v", kind, err)
	}
}

func syncDirectory(t testing.TB, path string) {
	t.Helper()
	handle, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.Sync(); err != nil {
		_ = handle.Close()
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
}

func sourceForChoicepoint(t testing.TB, choicepoint choice.ChoicepointRecord) contractsource.PortableSource {
	t.Helper()
	projection, err := countercli.NewCLIProjectionDefinition(countercli.CLIProjectionDefinitionConfig{
		Fields: []countercli.CLIFieldID{
			countercli.CLIFieldStdoutJSONMode,
			countercli.CLIFieldStdoutJSONSource,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	capture, err := countercli.NewCLICapturePolicy(countercli.CLICapturePolicyConfig{
		StdoutBytes: 64 << 10,
		StderrBytes: 64 << 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	environment, err := countercli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := countercli.NewFixtureFile(
		"config.json", clifixture.ConfigJSON("config"), countercli.FixtureMode0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	stimulus, err := countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: "node",
		BaseArgv:   []string{clifixture.Entrypoint},
		Argv:       []string{"--mode", "argv"},
		Stdin:      countercli.AbsentStdin(),
		Environment: []countercli.CLIEnvironmentBinding{
			environment,
		},
		Fixtures:  []countercli.CLIFixtureFile{fixture},
		CWDPolicy: countercli.CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := choicepoint.WorldPlan()
	if plan.CapturePolicyDigest() != capture.Digest() ||
		plan.ProjectionDefinitionDigest() != projection.Binding().Digest() ||
		plan.ProjectionDefinitionBinding().Digest() != projection.Binding().Digest() {
		t.Fatal("sealed choicepoint capture or projection authority differs")
	}
	resolved, err := projectiontranslate.Resolve(projection.Binding())
	if err != nil {
		t.Fatal(err)
	}
	projectionAuthority, err := climodel.ResolveCLIProjectionAuthority(
		projection.Digest(), projection.CanonicalBytes(), projection.Binding(),
	)
	if err != nil {
		t.Fatal(err)
	}
	source, err := contractsource.NewCLISource(contractsource.CLIInput{
		Plan: plan, Stimulus: stimulus, Capture: capture,
		Profile: resolved.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		t.Fatal(err)
	}
	minimized := choicepoint.MinimizedStimulus()
	if source.Plan().Digest() != plan.Digest() || source.StimulusDigest() != minimized.Digest() ||
		source.StimulusDigest().String() != ReferenceStimulusDigest ||
		source.Profile().Digest() != choicepoint.PortableProfile().Digest() {
		t.Fatal("public CLI source reconstruction differs from sealed choicepoint")
	}
	return source
}

func decodeBase64(t testing.TB, encoded string) []byte {
	t.Helper()
	body, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || base64.StdEncoding.EncodeToString(body) != encoded {
		t.Fatalf("decode exact choicepoint base64: %v", err)
	}
	return body
}

func portableChoicepoint(t testing.TB, legacy choice.ChoicepointRecord) choice.ChoicepointRecord {
	t.Helper()
	var wire choicepointWire
	if err := json.Unmarshal(legacy.CanonicalBytes(), &wire); err != nil {
		t.Fatal(err)
	}
	envelope, err := domain.ParseComparisonEnvelope(decodeBase64(t, wire.ComparisonEnvelopeBase64))
	if err != nil {
		t.Fatal(err)
	}
	bindings := make([]domain.CandidateExecutionBinding, len(wire.CandidateBindingsBase64))
	for index, encoded := range wire.CandidateBindingsBase64 {
		bindings[index], err = domain.ParseCandidateExecutionBinding(decodeBase64(t, encoded))
		if err != nil {
			t.Fatal(err)
		}
	}
	reveals := make([]choice.CandidateReveal, len(wire.CandidateReveals))
	for index, reveal := range wire.CandidateReveals {
		key, parseErr := domain.ParseCandidateExecutionKey(reveal.CandidateExecutionKey)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: key,
			DisplayRef:            reveal.DisplayRef,
			ProducerMetadata:      reveal.ProducerMetadata,
		}
	}
	originalDigest, err := domain.ParseDigest(wire.OriginalStimulusDigest)
	if err != nil {
		t.Fatal(err)
	}
	original, err := choice.NewCanonicalArtifact(
		wire.OriginalStimulusKind, originalDigest, decodeBase64(t, wire.OriginalStimulusBase64),
	)
	if err != nil {
		t.Fatal(err)
	}
	receipts := make([]domain.ReceiptReference, len(wire.EvidenceReceipts))
	for index, receipt := range wire.EvidenceReceipts {
		receipts[index], err = domain.ReceiptFromWire(receipt)
		if err != nil {
			t.Fatal(err)
		}
	}
	portable, err := choice.NewChoicepointRecord(choice.ChoicepointInput{
		Scenario: legacy.Scenario(), Plan: legacy.WorldPlan(), Envelope: envelope,
		CandidateBindings: bindings, OriginalStimulus: original,
		MinimizedStimulus: legacy.MinimizedStimulus(), Confirmation: legacy.ConfirmationRecord(),
		CandidateReveals: reveals, EvidenceReceipts: receipts,
	})
	if err != nil || !portable.Valid() || !portable.PortableProfile().Valid() || portable.Digest() == legacy.Digest() {
		t.Fatalf("reconstruct portable choicepoint: %v", err)
	}
	return portable
}

func decisionForChoicepoint(
	t testing.TB,
	choicepoint choice.ChoicepointRecord,
	desiredFields map[string]string,
) choice.DecisionRecord {
	t.Helper()
	session, err := choice.NewSession(choicepoint)
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness,
		choice.SurfaceMinimizedWitness,
		choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations,
		choice.SurfaceNonassertedFields,
	} {
		session, err = session.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	cards := session.BlindDTO().Cards()
	selected := session.BlindDTO().SelectableFields()
	if len(cards) == 0 || len(selected) == 0 {
		t.Fatal("portable choicepoint has no observed card or selectable field")
	}
	selectedAlias := cards[0].Alias
	if len(desiredFields) > 0 {
		selectedAlias = ""
		for _, card := range cards {
			matched := true
			observed := make(map[string]string, len(card.Fields))
			for _, field := range card.Fields {
				observed[field.FieldID] = field.Text
			}
			for fieldID, want := range desiredFields {
				if observed[fieldID] != want {
					matched = false
					break
				}
			}
			if matched {
				selectedAlias = card.Alias
				break
			}
		}
		if selectedAlias == "" {
			t.Fatalf("portable choicepoint has no card matching exact desired fields: %v", desiredFields)
		}
	}
	draft := choice.RulingDraftInput{
		Action: choice.ActionAllowObserved, SelectedFields: selected,
		AllowedAliases: []string{selectedAlias},
	}
	session, err = session.Propose(draft)
	if err != nil {
		t.Fatal(err)
	}
	session, _, err = session.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	session, err = session.Visit(choice.SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	session, err = session.Revise(draft, "")
	if err != nil {
		t.Fatal(err)
	}
	_, decision, err := session.Finalize(
		"countershape-c4-test", "exact public portable ruling fixture", []domain.ReceiptReference{},
	)
	if err != nil || !decision.Valid() {
		t.Fatalf("finalize portable decision: %v", err)
	}
	return decision
}

func newResidue(t testing.TB, parent, label string, desiredFields map[string]string) residueFixture {
	t.Helper()
	parent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "store")
	objectStore, err := store.OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	choicepoint, err := choice.ParseChoicepointRecord(example(t, "choicepoint.valid.json"))
	if err != nil {
		t.Fatalf("parse choicepoint example: %v", err)
	}
	choicepoint = portableChoicepoint(t, choicepoint)
	decision := decisionForChoicepoint(t, choicepoint, desiredFields)
	source := sourceForChoicepoint(t, choicepoint)
	confirmation := choicepoint.ConfirmationRecord()
	for _, item := range []struct {
		kind   string
		digest domain.Digest
		exact  []byte
	}{
		{"WorldPlan", source.Plan().Digest(), source.Plan().CanonicalBytes()},
		{"FreshConfirmation", confirmation.Digest(), confirmation.CanonicalBytes()},
		{"Choicepoint", choicepoint.Digest(), choicepoint.CanonicalBytes()},
		{"DecisionRecord", decision.Digest(), decision.CanonicalBytes()},
	} {
		publishObject(t, objectStore, item.kind, item.digest, item.exact)
	}
	study, err := store.NewStudyID("C4 contract execution " + label)
	if err != nil {
		t.Fatal(err)
	}
	previous := domain.MustDigest("sha256:" + strings.Repeat("a", 64))
	headBytes, err := canon.CanonicalizeTyped(headWire{
		SchemaVersion: domain.SchemaVersion, Kind: "StudyHead", StudyID: study.String(), Revision: 7,
		Stage: string(store.StageRuling), CurrentKind: "DecisionRecord", CurrentDigest: decision.Digest().String(),
		LineageRootDigest: source.Plan().Digest().String(), PreviousHeadDigest: previous.String(),
		PreviousObjectDigest: choicepoint.Digest().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	studyPath := filepath.Join(root, "studies", strings.TrimPrefix(study.Digest().String(), "sha256:"))
	if err := os.Mkdir(studyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(studyPath, ".head.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Sync(); err != nil {
		_ = lock.Close()
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	head, err := os.OpenFile(filepath.Join(studyPath, "head.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := head.Write(headBytes); err != nil {
		_ = head.Close()
		t.Fatal(err)
	}
	if err := head.Sync(); err != nil {
		_ = head.Close()
		t.Fatal(err)
	}
	if err := head.Close(); err != nil {
		t.Fatal(err)
	}
	syncDirectory(t, studyPath)
	syncDirectory(t, filepath.Dir(studyPath))
	ctx := context.Background()
	ruling, err := promotion.OpenRuling(ctx, objectStore, study)
	if err != nil {
		t.Fatalf("open exact ruling: %v", err)
	}
	preparation, err := promotion.PreparePortableRuling(ctx, objectStore, ruling)
	if err != nil {
		t.Fatalf("prepare portable ruling: %v", err)
	}
	prepared, err := node.PrepareCompilation(ctx, objectStore, preparation, source)
	if err != nil {
		t.Fatalf("prepare exact compiler input: %v", err)
	}
	compiled, err := node.CompilePrepared(prepared)
	if err != nil {
		t.Fatalf("compile exact contract bundle: %v", err)
	}
	publication, err := node.PublishPrepared(ctx, objectStore, compiled)
	if err != nil || !publication.Residue.Valid() {
		t.Fatalf("publish exact terminal residue: %v", err)
	}
	return residueFixture{store: objectStore, root: root, residue: publication.Residue}
}
