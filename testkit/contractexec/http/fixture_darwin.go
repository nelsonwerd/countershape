//go:build darwin && arm64 && cgo

// Package http provides public-constructor-only C5 HTTP contract-execution
// fixtures. It retains no alternate execution, admission, or target authority.
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/contractexec"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	node "github.com/nelsonwerd/countershape/internal/emit/node"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
	"github.com/nelsonwerd/countershape/testkit/httpfixture"
	httpstudy "github.com/nelsonwerd/countershape/testkit/studies/http_invoices"
)

const (
	SystemGit  = "/usr/bin/git"
	SystemNode = "/opt/homebrew/bin/node"
)

type physicalLineage struct {
	checkpoint httpstudy.StudyResult
	baseline   httpstudy.StudyResult
	minimized  httpstudy.StudyResult
	confirmed  httpstudy.StudyResult
	reduction  reducer.ReductionRun
	grade      grade.Result

	source            contractsource.PortableSource
	original          choice.CanonicalArtifact
	minimizedArtifact choice.CanonicalArtifact
	reveals           []choice.CandidateReveal
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

type decisionVariant string

const (
	decisionObserved404 decisionVariant = "observed-404"
	decisionCustom401   decisionVariant = "custom-401"
	decisionAllowMany   decisionVariant = "allow-many"
)

// Fixture retains the one capability consumed by black-box C5 tests.
// StoreRoot is diagnostics-only and cannot mint an execution target.
type Fixture struct {
	Target    contractexec.OfficialTarget
	StoreRoot string
}

var sharedCompilation struct {
	lineageOnce sync.Once
	lineageErr  error
	lineage     physicalLineage
	root        string

	observedOnce sync.Once
	observedErr  error
	observed     residueFixture

	custom401Once sync.Once
	custom401Err  error
	custom401     residueFixture

	allowManyOnce sync.Once
	allowManyErr  error
	allowMany     residueFixture

	sourceMu sync.Mutex
	sources  map[string]sourceRepositoryFixture
}

// RunMain releases process-scoped physical lineage, compiled residues, and
// immutable source repositories after the complete test binary schedule.
func RunMain(m *testing.M) int {
	code := m.Run()
	if sharedCompilation.root != "" {
		if err := os.RemoveAll(sharedCompilation.root); err != nil {
			fmt.Fprintf(os.Stderr, "C5 physical fixture cleanup failed: %v\n", err)
			if code == 0 {
				code = 1
			}
		}
	}
	return code
}

func NewReferenceTarget(t testing.TB) Fixture {
	t.Helper()
	return newRoleTarget(t, sharedResidue(t, decisionObserved404), httpfixture.ConcealNotFound, "C5 HTTP reference")
}

func NewDifferingTarget(t testing.TB) Fixture {
	t.Helper()
	return newRoleTarget(t, sharedResidue(t, decisionObserved404), httpfixture.Forbidden, "C5 HTTP differing")
}

func NewCustom401Target(t testing.TB, role httpfixture.CandidateRole) Fixture {
	t.Helper()
	return newRoleTarget(t, sharedResidue(t, decisionCustom401), role, "C5 HTTP custom 401")
}

func NewAllowManyTarget(t testing.TB, role httpfixture.CandidateRole) Fixture {
	t.Helper()
	return newRoleTarget(t, sharedResidue(t, decisionAllowMany), role, "C5 HTTP allow-many")
}

func ReferenceFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	files, err := httpfixture.PortableCandidateFiles(role)
	if err != nil {
		t.Fatal(err)
	}
	return cloneFiles(files)
}

func EarlyExitFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		`    writeAll(readinessFD, Buffer.from(`,
		"    process.exit(23);\n    writeAll(readinessFD, Buffer.from(",
	)
}

func PartialReadinessFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"Buffer.from(`COUNTERSHAPE_READY_V1 ${address.port}\\n`, \"ascii\")",
		"Buffer.from(\"COUNTERSHAPE_READY_V1 43127\", \"ascii\")",
	)
}

func MalformedReadinessFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"Buffer.from(`COUNTERSHAPE_READY_V1 ${address.port}\\n`, \"ascii\")",
		"Buffer.from(\"COUNTERSHAPE_READY_V1 01\\n\", \"ascii\")",
	)
}

func WithheldReadinessFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"    writeAll(readinessFD, Buffer.from(`COUNTERSHAPE_READY_V1 ${address.port}\\n`, \"ascii\"));\n"+
			"    closeSync(readinessFD);",
		"    return;",
	)
}

func WrongEndpointFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"    writeAll(readinessFD, Buffer.from(`COUNTERSHAPE_READY_V1 ${address.port}\\n`, \"ascii\"));\n"+
			"    closeSync(readinessFD);",
		`    const resetServer = createServer({ allowHalfOpen: false }, (socket) => {
      socket.destroy();
      resetServer.close();
    });
    resetServer.listen({ host: "127.0.0.1", port: 0, exclusive: true }, () => {
      const resetAddress = resetServer.address();
      if (resetAddress === null || typeof resetAddress !== "object") {
        die("reset endpoint did not bind");
      }
      writeAll(readinessFD, Buffer.from(`+"`COUNTERSHAPE_READY_V1 ${resetAddress.port}\\n`"+`, "ascii"));
      closeSync(readinessFD);
    });`,
	)
}

func DecoyReadinessFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"    writeAll(readinessFD, Buffer.from(`COUNTERSHAPE_READY_V1 ${address.port}\\n`, \"ascii\"));\n"+
			"    closeSync(readinessFD);",
		`    const decoyServer = createServer({ allowHalfOpen: true }, (socket) => {
      let responded = false;
      socket.on("data", () => {
        if (responded) return;
        responded = true;
        const body = Buffer.from(JSON.stringify({
          fixture_version: fixtureVersion,
          invoice_id: seed.invoice_id,
          kind: "decoy",
          metadata: {},
          request_id: attemptID,
          scratch_root: stateRoot,
        }), "utf8");
        socket.end(Buffer.concat([
          Buffer.from(
            "HTTP/1.1 418 I'm a Teapot\r\n" +
            "Content-Type: application/json\r\n" +
            `+"`Content-Length: ${body.length}\\r\\n`"+` +
            "Connection: close\r\n\r\n",
            "latin1",
          ),
          body,
        ]));
        server.close();
        decoyServer.close();
      });
    });
    decoyServer.listen({ host: "127.0.0.1", port: 0, exclusive: true }, () => {
      const decoyAddress = decoyServer.address();
      if (decoyAddress === null || typeof decoyAddress !== "object") {
        die("decoy endpoint did not bind");
      }
      writeAll(readinessFD, Buffer.from(`+"`COUNTERSHAPE_READY_V1 ${decoyAddress.port}\\n`"+`, "ascii"));
      closeSync(readinessFD);
    });`,
	)
}

func MalformedResponseFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"    socket.end(Buffer.concat([responseHead, body]));",
		`    socket.end(Buffer.from("NOT_HTTP\r\n\r\n", "latin1"));`,
	)
}

func PartialResponseFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"    socket.end(Buffer.concat([responseHead, body]));",
		`    socket.end(Buffer.from("HTTP/1.1 404 Not Found\r\nContent-Length: 4\r\n\r\nno", "latin1"));`,
	)
}

func OverflowResponseFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"    socket.end(Buffer.concat([responseHead, body]));",
		"    socket.end(Buffer.alloc(128 * 1024, 0x41));",
	)
}

func ResponseTimeoutFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	files := rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"const server = createServer({ allowHalfOpen: false }, (socket) => {",
		"const server = createServer({ allowHalfOpen: true }, (socket) => {",
	)
	return rewritePortableProgram(
		t,
		files,
		"    socket.end(Buffer.concat([responseHead, body]));\n    server.close();",
		"    void responseHead;\n    void body;",
	)
}

func MissingInvocationFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"    syncExclusive(\n      join(evidenceRoot, \"http-invocation.json\"),",
		"    if (false) syncExclusive(\n      join(evidenceRoot, \"http-invocation.json\"),",
	)
}

func MalformedInvocationFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		`      Buffer.from(JSON.stringify({
        attempt_id: attemptID,
        invocation_count: 1,
        kind: "HTTPFixtureInvocationEvidence",
        request_byte_count: requestBytes.length,
        request_byte_sha256: requestSHA256,
        schema_version: "countershape/v1",
        stimulus_digest: stimulusDigest,
      }), "utf8"),`,
		`      Buffer.from("{", "utf8"),`,
	)
}

func ChildBindingMismatchFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	return rewritePortableProgram(
		t,
		ReferenceFiles(t, role),
		"        attempt_id: attemptID,",
		`        attempt_id: attemptID.slice(0, -1) + (attemptID.endsWith("0") ? "1" : "0"),`,
	)
}

func ForbiddenScopeFiles(t testing.TB, role httpfixture.CandidateRole) []gitrepo.File {
	t.Helper()
	files := ChildBindingMismatchFiles(t, role)
	files = rewritePortableProgram(
		t,
		files,
		`const repetition = requiredEnvironment("COUNTERSHAPE_SCHEDULE_REPETITION");`,
		`const repetition = requiredEnvironment("COUNTERSHAPE_SCHEDULE_REPETITION");
const importCanaryModule = requiredEnvironment("COUNTERSHAPE_C5_IMPORT_CANARY_MODULE");
const serviceCanarySocket = requiredEnvironment("COUNTERSHAPE_C5_SERVICE_CANARY_SOCKET");
const { pathToFileURL } = await import("node:url");
const { createConnection } = await import("node:net");
await import(pathToFileURL(importCanaryModule).href);
await new Promise((resolve, reject) => {
  const client = createConnection(serviceCanarySocket, () => client.end());
  client.once("close", resolve);
  client.once("error", reject);
});`,
	)
	return append(files,
		gitrepo.File{
			Path: "countershape-runtime.mjs", Mode: "100644",
			Content: []byte("export const inertCountershapeSource = true;\n"),
		},
		gitrepo.File{
			Path: "node_modules/countershape/index.mjs", Mode: "100644",
			Content: []byte("export const inertCountershapeDependency = true;\n"),
		},
	)
}

func NewControlTarget(t testing.TB, files []gitrepo.File, label string) Fixture {
	t.Helper()
	return newTarget(t, sharedResidue(t, decisionObserved404), files, label)
}

func newRoleTarget(
	t testing.TB,
	residue residueFixture,
	role httpfixture.CandidateRole,
	label string,
) Fixture {
	t.Helper()
	return newTarget(t, residue, ReferenceFiles(t, role), label+" "+string(role))
}

func sharedResidue(t testing.TB, variant decisionVariant) residueFixture {
	t.Helper()
	lineage := sharedLineage(t)
	var (
		once     *sync.Once
		target   *residueFixture
		buildErr *error
	)
	switch variant {
	case decisionObserved404:
		once, target, buildErr = &sharedCompilation.observedOnce, &sharedCompilation.observed, &sharedCompilation.observedErr
	case decisionCustom401:
		once, target, buildErr = &sharedCompilation.custom401Once, &sharedCompilation.custom401, &sharedCompilation.custom401Err
	case decisionAllowMany:
		once, target, buildErr = &sharedCompilation.allowManyOnce, &sharedCompilation.allowMany, &sharedCompilation.allowManyErr
	default:
		t.Fatalf("unknown C5 decision variant %q", variant)
	}
	once.Do(func() {
		*target, *buildErr = buildResidue(
			filepath.Join(sharedCompilation.root, "residue-"+string(variant)),
			"C5 physical HTTP "+string(variant),
			lineage,
			variant,
		)
	})
	if *buildErr != nil || target.store == nil || !target.residue.Valid() {
		t.Fatalf("build C5 %s residue: valid=%t err=%v", variant, target.residue.Valid(), *buildErr)
	}
	return *target
}

func sharedLineage(t testing.TB) physicalLineage {
	t.Helper()
	resolvedGit := resolvedTool(t, "git", SystemGit)
	resolvedNode := resolvedTool(t, "node", SystemNode)
	sharedCompilation.lineageOnce.Do(func() {
		root, err := os.MkdirTemp("", "countershape-c5-compiled-")
		if err == nil {
			err = os.Chmod(root, 0o700)
		}
		if err == nil {
			root, err = filepath.EvalSymlinks(root)
		}
		if err != nil {
			if root != "" {
				err = errors.Join(err, os.RemoveAll(root))
			}
			sharedCompilation.lineageErr = err
			return
		}
		sharedCompilation.root = root
		sharedCompilation.lineage, sharedCompilation.lineageErr =
			buildPhysicalLineage(root, resolvedGit, resolvedNode)
	})
	if sharedCompilation.lineageErr != nil || !sharedCompilation.lineage.source.Valid() {
		t.Fatalf("build exact C5 physical lineage: %v", sharedCompilation.lineageErr)
	}
	return sharedCompilation.lineage
}

func resolvedTool(t testing.TB, name, path string) string {
	t.Helper()
	if _, err := os.Lstat(path); err != nil {
		t.Skipf("exact %s unavailable at %s: %v", name, path, err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) || filepath.Clean(resolved) != resolved {
		t.Fatalf("resolve exact %s: path=%q err=%v", name, resolved, err)
	}
	return resolved
}

func buildPhysicalLineage(root, gitExecutable, nodeExecutable string) (physicalLineage, error) {
	noisy, err := referenceStimulusWithNoise()
	if err != nil {
		return physicalLineage{}, err
	}
	config := physicalConfig(root, gitExecutable, nodeExecutable)
	config.StimulusOverride = &noisy
	checkpoint, err := httpstudy.Run(context.Background(), config)
	if err != nil {
		return physicalLineage{}, fmt.Errorf("baseline checkpoint: %w", err)
	}
	baselineStudy, err := httpstudy.Run(context.Background(), config)
	if err != nil {
		return physicalLineage{}, fmt.Errorf("baseline divergence: %w", err)
	}
	if !checkpoint.HasOutcomeMap || !baselineStudy.HasOutcomeMap ||
		compare.AssessPreservation(checkpoint.OutcomeMap, baselineStudy.OutcomeMap).Relation() != compare.PreservationEqual ||
		bytes.Equal(checkpoint.OutcomeMap.CanonicalBytes(), baselineStudy.OutcomeMap.CanonicalBytes()) {
		return physicalLineage{}, errors.New("fresh baseline pair did not retain equivalent distinct evidence")
	}
	baseline, err := compare.RequireDivergence(baselineStudy.OutcomeMap)
	if err != nil {
		return physicalLineage{}, fmt.Errorf(
			"baseline did not diverge: checkpoint={%s} baseline={%s}: %w",
			physicalStudyDiagnostic(checkpoint),
			physicalStudyDiagnostic(baselineStudy),
			err,
		)
	}
	policy, err := counterhttp.NewHTTPReductionPolicy(counterhttp.HTTPReductionPolicyConfig{
		Anchor:          noisy,
		PinnedSeedPaths: []string{httpfixture.SeedFilename},
		EnabledRules:    []counterhttp.HTTPReducerID{counterhttp.HTTPSeedRemove},
	})
	if err != nil {
		return physicalLineage{}, err
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineStudy.Plan)
	if err != nil {
		return physicalLineage{}, err
	}
	var minimized httpstudy.StudyResult
	reductionRun, err := reducer.Run(context.Background(), reducer.RunInput[counterhttp.HTTPStimulus]{
		Original: noisy,
		Reference: func(stimulus counterhttp.HTTPStimulus) (domain.Digest, reducer.Measure, bool) {
			measure, measureErr := counterhttp.MeasureHTTPStimulus(stimulus, policy)
			return stimulus.Digest(), measure, measureErr == nil
		},
		Enumerate: func(
			_ context.Context,
			stimulus counterhttp.HTTPStimulus,
		) ([]reducer.TypedProposal[counterhttp.HTTPStimulus], error) {
			neighbors, enumerateErr := counterhttp.EnumerateHTTPNeighbors(stimulus, policy)
			if enumerateErr != nil {
				return nil, enumerateErr
			}
			proposals := make([]reducer.TypedProposal[counterhttp.HTTPStimulus], len(neighbors))
			for index, neighbor := range neighbors {
				proposals[index] = reducer.TypedProposal[counterhttp.HTTPStimulus]{
					Stimulus: neighbor.Stimulus(),
					Neighbor: neighbor.Neighbor(),
				}
			}
			return proposals, nil
		},
		Evaluate: func(
			ctx context.Context,
			stimulus counterhttp.HTTPStimulus,
			_ reducer.Neighbor,
			purpose domain.AttemptPurpose,
			allowance reducer.EvaluationAllowance,
		) (reducer.EvaluationObservation, error) {
			evaluation := physicalConfig(root, gitExecutable, nodeExecutable)
			requiredTrials := uint64(evaluation.MaxTotalTrials)
			if allowance.RemainingCandidateTrials < requiredTrials ||
				!time.Now().Before(allowance.WallDeadline) {
				return reducer.EvaluationObservation{}, context.DeadlineExceeded
			}
			evaluation.Purpose = purpose
			evaluation.StimulusOverride = &stimulus
			runContext, cancel := context.WithDeadline(ctx, allowance.WallDeadline)
			defer cancel()
			result, runErr := httpstudy.Run(runContext, evaluation)
			if runErr != nil {
				return reducer.EvaluationObservation{}, runErr
			}
			if !result.HasOutcomeMap {
				return reducer.EvaluationObservation{}, errors.New("reducer evaluation has no outcome map")
			}
			outcome := result.OutcomeMap
			if compare.AssessPreservation(baseline.OutcomeMap(), outcome).Relation() == compare.PreservationEqual {
				minimized = result
			}
			return reducer.EvaluationObservation{
				OutcomeMap:      &outcome,
				CandidateTrials: uint64(len(result.Trials)),
			}, nil
		},
		Baseline:   baseline,
		ReducerSet: policy.ReducerSet(),
		Budget:     budget,
	})
	if err != nil {
		return physicalLineage{}, fmt.Errorf("physical reducer: %w", err)
	}
	if !reductionRun.Valid() || !reductionRun.HasAcceptedReduction() || !minimized.HasOutcomeMap ||
		minimized.Stimulus.Digest() != reductionRun.MinimizedStimulusDigest() {
		return physicalLineage{}, errors.New("physical reduction did not retain its accepted fresh study")
	}
	sweepDraft, present, err := reductionRun.CompletedSweepDraft()
	if err != nil || !present || !sweepDraft.Valid() {
		return physicalLineage{}, fmt.Errorf("completed reduction sweep: present=%t err=%w", present, err)
	}
	sweepStore, err := store.OpenReductionSweepStore(filepath.Join(root, "sweep-store"))
	if err != nil {
		return physicalLineage{}, err
	}
	sweepAuthority, err := sweepStore.Publish(context.Background(), sweepDraft)
	if err != nil {
		return physicalLineage{}, err
	}
	finalGrade, err := grade.Finalize(context.Background(), reductionRun, &grade.SweepCompletion{
		Store:     sweepStore,
		Draft:     sweepDraft,
		Authority: sweepAuthority,
	})
	if err != nil || !finalGrade.Valid() {
		return physicalLineage{}, fmt.Errorf("finalize physical reduction grade: %w", err)
	}
	reducedBaseline, err := compare.RequireDivergence(minimized.OutcomeMap)
	if err != nil {
		return physicalLineage{}, err
	}
	confirmationConfig := physicalConfig(root, gitExecutable, nodeExecutable)
	confirmationConfig.Purpose = domain.AttemptConfirmation
	confirmationConfig.StimulusOverride = &minimized.Stimulus
	confirmationConfig.Confirmation = &httpstudy.ConfirmationInput{
		ReducedBaseline: reducedBaseline,
		ReductionRun:    reductionRun,
		ReductionResult: finalGrade,
	}
	confirmed, err := httpstudy.Run(context.Background(), confirmationConfig)
	if err != nil {
		return physicalLineage{}, fmt.Errorf("physical confirmation: %w", err)
	}
	source, err := portableSource(confirmed)
	if err != nil {
		return physicalLineage{}, err
	}
	original, err := choice.NewCanonicalArtifact(
		"HTTPStimulus", baselineStudy.Stimulus.Digest(), baselineStudy.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		return physicalLineage{}, err
	}
	minimizedArtifact, err := choice.NewCanonicalArtifact(
		"HTTPStimulus", confirmed.Stimulus.Digest(), confirmed.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		return physicalLineage{}, err
	}
	reveals := make([]choice.CandidateReveal, len(confirmed.CandidateBindings))
	for index, binding := range confirmed.CandidateBindings {
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: binding.Key(),
			DisplayRef:            string(confirmed.CandidateRoles[binding.Key()]),
			ProducerMetadata:      "local deterministic HTTP fixture",
		}
	}
	lineage := physicalLineage{
		checkpoint: checkpoint, baseline: baselineStudy, minimized: minimized, confirmed: confirmed,
		reduction: reductionRun, grade: finalGrade, source: source,
		original: original, minimizedArtifact: minimizedArtifact, reveals: reveals,
	}
	if err := validatePhysicalLineage(lineage); err != nil {
		return physicalLineage{}, err
	}
	return lineage, nil
}

func physicalStudyDiagnostic(result httpstudy.StudyResult) string {
	trials := make([]string, 0, len(result.Trials))
	for _, trial := range result.Trials {
		process := trial.Result.Process()
		primary, hasPrimary := process.PrimaryControl()
		stderr := process.Stderr()
		trials = append(trials, fmt.Sprintf(
			"%s#%d/r%d admitted=%t projected=%t rejected=%t primary=%s/%t diagnostic=%s exit=%d signal=%s stderr-bytes=%d",
			trial.Role,
			trial.Slot.Ordinal(),
			trial.Slot.Repetition(),
			trial.Admitted,
			trial.Projected,
			trial.ProjectionRejection != nil,
			primary,
			hasPrimary,
			process.DiagnosticCode(),
			process.ExitCode(),
			process.ExitSignal(),
			len(stderr),
		))
	}
	return fmt.Sprintf(
		"has-outcome=%t entries=%v exclusions=%v distinct=%d trials=%v",
		result.HasOutcomeMap,
		result.OutcomeMap.Entries(),
		result.OutcomeMap.Exclusions(),
		result.OutcomeMap.DistinctProjectionCount(),
		trials,
	)
}

func physicalConfig(root, gitExecutable, nodeExecutable string) httpstudy.Config {
	config := httpstudy.DefaultConfig(root, gitExecutable, nodeExecutable)
	config.PortableStart = true
	config.ReductionProposalLimit = 2
	config.ReductionTotalCandidateTrials = 36
	config.ReductionWallMS = (3 * time.Minute).Milliseconds()
	return config
}

func referenceStimulusWithNoise() (counterhttp.HTTPStimulus, error) {
	query := make([]counterhttp.HTTPQueryEntry, 0, 5)
	for _, input := range [][2]string{
		{"actor_tenant", "tenant-a"},
		{"role", "support"},
		{"tag", "first"},
		{"tag", "second"},
	} {
		entry, err := counterhttp.QueryValue(input[0], input[1])
		if err != nil {
			return counterhttp.HTTPStimulus{}, err
		}
		query = append(query, entry)
	}
	flag, err := counterhttp.QueryFlag("audit")
	if err != nil {
		return counterhttp.HTTPStimulus{}, err
	}
	query = append(query, flag)
	headers := make([]counterhttp.HTTPRequestHeader, 0, 9)
	for _, input := range [][2]string{
		{"accept", "application/json"},
		{"x-countershape-tenant", "tenant-a"},
		{"x-countershape-role", "support"},
		{"x-countershape-audit", "required"},
		{"x-countershape-tag", "first"},
		{"x-countershape-tag", "second"},
		{"x-countershape-trace", "first"},
		{"x-countershape-trace", "second"},
		{"x-countershape-present-empty", ""},
	} {
		header, headerErr := counterhttp.NewRequestHeader(input[0], input[1])
		if headerErr != nil {
			return counterhttp.HTTPStimulus{}, headerErr
		}
		headers = append(headers, header)
	}
	seed, err := counterhttp.NewSeedFile(
		httpfixture.SeedFilename, httpfixture.SeedJSON(), counterhttp.SeedMode0644,
	)
	if err != nil {
		return counterhttp.HTTPStimulus{}, err
	}
	noise, err := counterhttp.NewSeedFile("z-noise.txt", []byte("not-consumed\n"), counterhttp.SeedMode0644)
	if err != nil {
		return counterhttp.HTTPStimulus{}, err
	}
	return counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method:  counterhttp.MethodGET,
		Path:    "/v1/invoices/inv-204",
		Query:   query,
		Headers: headers,
		Body:    counterhttp.AbsentBody(),
		Seeds:   []counterhttp.HTTPSeedFile{seed, noise},
	})
}

func portableSource(study httpstudy.StudyResult) (contractsource.PortableSource, error) {
	if !study.HasConfirmation || !study.Confirmation.Valid() {
		return contractsource.PortableSource{}, errors.New("confirmed HTTP study is absent")
	}
	resolved, err := projectiontranslate.Resolve(study.ProjectionDefinition.Binding())
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	projection, err := httpmodel.ResolveHTTPProjectionAuthority(
		study.ProjectionDefinition.Digest(),
		study.ProjectionDefinition.CanonicalBytes(),
		study.ProjectionDefinition.Binding(),
	)
	if err != nil {
		return contractsource.PortableSource{}, err
	}
	return contractsource.NewHTTPSource(contractsource.HTTPInput{
		Plan:       study.Plan,
		Stimulus:   study.Stimulus,
		Start:      study.StartSpec,
		Capture:    study.CapturePolicy,
		Readiness:  study.Readiness,
		Profile:    resolved.Profile(),
		Projection: projection,
	})
}

func validatePhysicalLineage(lineage physicalLineage) error {
	studies := []httpstudy.StudyResult{
		lineage.checkpoint, lineage.baseline, lineage.minimized, lineage.confirmed,
	}
	seenAttempts := make(map[domain.Digest]struct{}, 48)
	seenWorlds := make(map[domain.Digest]struct{}, 48)
	seenObservations := make(map[domain.Digest]struct{}, 48)
	for index, study := range studies {
		if len(study.Trials) != 12 || !study.HasOutcomeMap ||
			study.StartSpec.Authority() != counterhttp.HTTPPortableStartAuthorityV1 ||
			study.Readiness.Protocol() != counterhttp.PortableReadinessProtocolV1 {
			return fmt.Errorf("physical lineage study %d lacks the exact 12-trial portable profile", index)
		}
		for _, roster := range []struct {
			label  string
			values []domain.Digest
			seen   map[domain.Digest]struct{}
		}{
			{"attempt", study.OutcomeMap.EvidenceAttemptDigests(), seenAttempts},
			{"world", study.OutcomeMap.EvidenceWorldDigests(), seenWorlds},
			{"observation", study.OutcomeMap.EvidenceObservationDigests(), seenObservations},
		} {
			if len(roster.values) != 12 {
				return fmt.Errorf("physical study %d has %d %s digests", index, len(roster.values), roster.label)
			}
			for _, digest := range roster.values {
				if !digest.Valid() {
					return fmt.Errorf("physical study %d has invalid %s digest", index, roster.label)
				}
				if _, duplicate := roster.seen[digest]; duplicate {
					return fmt.Errorf("physical lineage reused %s digest %s", roster.label, digest)
				}
				roster.seen[digest] = struct{}{}
			}
		}
	}
	facts := lineage.confirmed.Confirmation.Draft().PhysicalFacts()
	if len(facts) != 12 {
		return fmt.Errorf("confirmation retained %d physical facts", len(facts))
	}
	record, err := confirmation.ParseRecord(lineage.confirmed.Confirmation.Draft().CanonicalBytes())
	if err != nil {
		return err
	}
	for index, binding := range record.ExecutionBindingDigests() {
		if binding != lineage.source.ExecutionBindingDigest() {
			return fmt.Errorf("confirmation binding %d differs from portable source", index)
		}
	}
	return nil
}

func buildResidue(
	root string,
	label string,
	lineage physicalLineage,
	variant decisionVariant,
) (residueFixture, error) {
	objectStore, err := store.OpenObjectStore(root)
	if err != nil {
		return residueFixture{}, err
	}
	studyID, err := store.NewStudyID(label)
	if err != nil {
		return residueFixture{}, err
	}
	head, err := objectStore.CreateStudy(context.Background(), studyID, lineage.confirmed.Plan)
	if err != nil {
		return residueFixture{}, err
	}
	head, err = objectStore.AdvanceBaseline(context.Background(), head, lineage.checkpoint.OutcomeMap)
	if err != nil {
		return residueFixture{}, err
	}
	baseline, err := compare.RequireDivergence(lineage.baseline.OutcomeMap)
	if err != nil {
		return residueFixture{}, err
	}
	head, err = objectStore.AdvanceDivergence(context.Background(), head, baseline)
	if err != nil {
		return residueFixture{}, err
	}
	head, err = objectStore.AdvanceReduction(context.Background(), head, lineage.reduction)
	if err != nil {
		return residueFixture{}, err
	}
	confirmationDraft := lineage.confirmed.Confirmation.Draft()
	stored, err := promotion.PersistConfirmation(context.Background(), objectStore, head, confirmationDraft)
	if err != nil {
		return residueFixture{}, err
	}
	ready, err := promotion.Promote(context.Background(), objectStore, stored, promotion.ChoicepointRequest{
		Scenario:          "Which exact invoice response behavior should become the accepted contract?",
		Plan:              lineage.confirmed.Plan,
		Envelope:          lineage.confirmed.Envelope,
		CandidateBindings: lineage.confirmed.CandidateBindings,
		OriginalStimulus:  lineage.original,
		MinimizedStimulus: lineage.minimizedArtifact,
		CandidateReveals:  lineage.reveals,
		EvidenceReceipts:  []domain.ReceiptReference{},
	})
	if err != nil {
		return residueFixture{}, err
	}
	decision, err := decisionForReady(ready, variant, confirmationDraft.Digest())
	if err != nil {
		return residueFixture{}, err
	}
	ruling, err := promotion.Finalize(context.Background(), objectStore, ready, decision)
	if err != nil {
		return residueFixture{}, err
	}
	preparation, err := promotion.PreparePortableRuling(context.Background(), objectStore, ruling)
	if err != nil {
		return residueFixture{}, err
	}
	prepared, err := node.PrepareCompilation(context.Background(), objectStore, preparation, lineage.source)
	if err != nil {
		return residueFixture{}, err
	}
	bundle, err := node.CompilePrepared(prepared)
	if err != nil {
		return residueFixture{}, err
	}
	published, err := node.PublishPrepared(context.Background(), objectStore, bundle)
	if err != nil || !published.Residue.Valid() || published.Disposition != node.PublicationCreated {
		return residueFixture{}, fmt.Errorf("publish C5 residue: disposition=%s err=%w", published.Disposition, err)
	}
	return residueFixture{store: objectStore, root: root, residue: published.Residue}, nil
}

func decisionForReady(
	ready promotion.Ready,
	variant decisionVariant,
	confirmationDigest domain.Digest,
) (choice.DecisionRecord, error) {
	selected := []string{string(counterhttp.HTTPFieldStatus)}
	var (
		draft choice.RulingDraftInput
		err   error
	)
	switch variant {
	case decisionObserved404:
		draft = choice.RulingDraftInput{
			Action:         choice.ActionAllowObserved,
			SelectedFields: selected,
			AllowedAliases: aliasesForStatuses(ready.Record(), "404"),
		}
	case decisionAllowMany:
		draft = choice.RulingDraftInput{
			Action:         choice.ActionAllowObserved,
			SelectedFields: selected,
			AllowedAliases: aliasesForStatuses(ready.Record(), "403", "404"),
		}
	case decisionCustom401:
		status, valueErr := choice.IntegerValue("401")
		if valueErr != nil {
			return choice.DecisionRecord{}, valueErr
		}
		expectation, tupleErr := choice.NewSelectedTuple(
			ready.Record(),
			selected,
			[]choice.FieldValue{{FieldID: string(counterhttp.HTTPFieldStatus), Value: status}},
		)
		if tupleErr != nil {
			return choice.DecisionRecord{}, tupleErr
		}
		draft = choice.RulingDraftInput{
			Action:               choice.ActionCustomExpectation,
			SelectedFields:       selected,
			AllowedAliases:       []string{},
			CustomExpectation:    &expectation,
			CustomReviewer:       "c5-http-401-reviewer",
			CustomReviewEvidence: confirmationDigest,
		}
	default:
		return choice.DecisionRecord{}, fmt.Errorf("unknown C5 decision variant %q", variant)
	}
	if len(draft.AllowedAliases) == 0 && variant != decisionCustom401 {
		return choice.DecisionRecord{}, fmt.Errorf("C5 %s decision matched no observed status", variant)
	}
	session, err := choice.NewSession(ready.Record())
	if err != nil {
		return choice.DecisionRecord{}, err
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
			return choice.DecisionRecord{}, err
		}
	}
	session, err = session.Propose(draft)
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	session, _, err = session.Reveal()
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	session, err = session.Visit(choice.SurfaceProvenance)
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	session, err = session.Revise(draft, "")
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	_, decision, err := session.Finalize(
		"countershape-c5-test",
		"Bind the exact reviewed standalone HTTP status contract.",
		[]domain.ReceiptReference{},
	)
	if err != nil || !decision.Valid() {
		return choice.DecisionRecord{}, fmt.Errorf("finalize C5 %s decision: %w", variant, err)
	}
	if !slices.Equal(decision.SelectedFields(), selected) {
		return choice.DecisionRecord{}, errors.New("C5 decision changed the selected field")
	}
	return decision, nil
}

func aliasesForStatuses(record choice.ChoicepointRecord, statuses ...string) []string {
	wanted := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		wanted[status] = struct{}{}
	}
	session, err := choice.NewSession(record)
	if err != nil {
		return nil
	}
	aliases := make([]string, 0, len(statuses))
	for _, card := range session.BlindDTO().Cards() {
		for _, field := range card.Fields {
			if field.FieldID != string(counterhttp.HTTPFieldStatus) ||
				field.Tag != string(choice.ValueInteger) {
				continue
			}
			if _, accepted := wanted[field.Text]; accepted {
				aliases = append(aliases, card.Alias)
				delete(wanted, field.Text)
			}
		}
	}
	if len(wanted) != 0 {
		return nil
	}
	return aliases
}

func newTarget(
	t testing.TB,
	residue residueFixture,
	files []gitrepo.File,
	label string,
) Fixture {
	t.Helper()
	source := sharedSourceRepository(t, files)
	repository, err := gitobj.OpenRepository(context.Background(), gitobj.OpenConfig{
		GitExecutable: SystemGit,
		Repository:    source.root,
		ScratchRoot:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("open %s source: %v", label, err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	target, err := contractexec.PublishOfficialTarget(
		context.Background(),
		contractexec.PublishOfficialTargetRequest{
			Store:          residue.store,
			Residue:        residue.residue,
			Repository:     repository,
			DisplayRef:     source.ref,
			NodeExecutable: SystemNode,
		},
	)
	if err != nil || !target.Valid() {
		t.Fatalf("publish %s target: valid=%t err=%v", label, target.Valid(), err)
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
	digest, err := canon.DigestBytes("C5HTTPSourceRepositoryFixture", exact)
	if err != nil {
		t.Fatal(err)
	}
	key := digest.String()
	sharedCompilation.sourceMu.Lock()
	defer sharedCompilation.sourceMu.Unlock()
	if source, ok := sharedCompilation.sources[key]; ok {
		return source
	}
	parent := filepath.Join(sharedCompilation.root, "sources")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	gitFixture, err := gitrepo.Init(context.Background(), SystemGit, parent, gitrepo.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := gitFixture.CommitFiles(context.Background(), cloneFiles(files), "", "C5 cached source "+key)
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

func cloneFiles(files []gitrepo.File) []gitrepo.File {
	result := make([]gitrepo.File, len(files))
	for index, file := range files {
		result[index] = file
		result[index].Content = append([]byte(nil), file.Content...)
	}
	return result
}

func rewritePortableProgram(
	t testing.TB,
	files []gitrepo.File,
	anchor string,
	replacement string,
) []gitrepo.File {
	t.Helper()
	result := cloneFiles(files)
	rewritten := 0
	for index := range result {
		if result[index].Path != httpfixture.PortableEntrypoint {
			continue
		}
		if bytes.Count(result[index].Content, []byte(anchor)) != 1 {
			t.Fatalf("portable fixture rewrite anchor count for %q is not one", anchor)
		}
		result[index].Content = bytes.Replace(
			result[index].Content,
			[]byte(anchor),
			[]byte(replacement),
			1,
		)
		rewritten++
	}
	if rewritten != 1 {
		t.Fatalf("portable fixture roster contained %d entrypoints, want one", rewritten)
	}
	return result
}
