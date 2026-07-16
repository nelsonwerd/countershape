//go:build darwin && cgo

package world

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/runnerprofile"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

const portableNegativeEntrypoint = "fixture/case.mjs"

type portableHTTPNegativeHarness struct {
	binding        httpmodel.HTTPExecutionBinding
	candidate      gitobj.BoundCandidate
	registry       ToolRegistry
	allocationRoot string
}

func newPortableHTTPNegativeHarness(t *testing.T, program string, readinessMS int64) portableHTTPNegativeHarness {
	t.Helper()
	ctx := context.Background()
	gitExecutable := resolvedExecutable(t, "git")
	nodeExecutable := resolvedExecutable(t, "node")
	repositoryFixture, err := gitrepo.Init(ctx, gitExecutable, resolvedPrivateTempDir(t), gitrepo.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	commits := make([]string, 2)
	for index, marker := range []string{"first", "second"} {
		commits[index], err = repositoryFixture.CommitFiles(ctx, []gitrepo.File{
			{Path: portableNegativeEntrypoint, Mode: "100644", Content: []byte(program)},
			{Path: "candidate-marker.txt", Mode: "100644", Content: []byte(marker + "\n")},
		}, "", "portable negative "+marker)
		if err != nil {
			t.Fatal(err)
		}
		if err := repositoryFixture.UpdateRef(ctx, "refs/heads/"+marker, commits[index]); err != nil {
			t.Fatal(err)
		}
	}
	repository, err := gitobj.OpenRepository(ctx, gitobj.OpenConfig{
		GitExecutable: gitExecutable, Repository: repositoryFixture.Root, ScratchRoot: resolvedPrivateTempDir(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("close repository: %v", err)
		}
	})
	first, err := repository.Pin(ctx, "refs/heads/first")
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.Pin(ctx, "refs/heads/second")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := gitobj.SelectTrees(first, second)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := gitobj.NewPolicy(16, 1<<20, 1<<18)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := gitobj.InspectSelected(ctx, selected, policy, resolvedPrivateTempDir(t))
	if err != nil {
		t.Fatal(err)
	}

	start, err := httpmodel.NewPortableHTTPStartSpec(portableNegativeEntrypoint)
	if err != nil {
		t.Fatal(err)
	}
	readiness, err := httpmodel.NewPortableHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	capture, err := httpmodel.NewHTTPCapturePolicy(httpmodel.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 16 << 10, HeaderCount: 32, BodyBytes: 16 << 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	fixtureRecipe, err := httpmodel.NewHTTPFixtureRecipe()
	if err != nil {
		t.Fatal(err)
	}
	projection, err := httpmodel.NewHTTPProjectionAuthority()
	if err != nil {
		t.Fatal(err)
	}
	stimulus, err := httpmodel.NewHTTPStimulus(httpmodel.HTTPStimulusConfig{
		Method: httpmodel.MethodGET, Path: "/", Query: []httpmodel.HTTPQueryEntry{},
		Headers: []httpmodel.HTTPRequestHeader{}, Body: httpmodel.AbsentBody(), Seeds: []httpmodel.HTTPSeedFile{},
	})
	if err != nil {
		t.Fatal(err)
	}
	runnerDigest, err := runnerprofile.HTTPPortableDigest()
	if err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest: selected.Digest(), MaterializationPolicyDigest: policy.Digest(),
		ComparisonEnvelopeDigest: worldTestDigest("3"),
		Adapter: domain.Adapter{
			Domain: domain.AdapterHTTP, AdapterVersion: runnerprofile.HTTPAdapterVersionV1, RunnerDigest: runnerDigest,
		},
		ExecutionShape: domain.OneLoopbackHTTPRequest, StartArgv: start.LogicalArgv(), SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{
			{Name: "LANG", Value: "C"}, {Name: "LC_ALL", Value: "C"}, {Name: "NO_COLOR", Value: "1"},
			{Name: "NODE_NO_WARNINGS", Value: "1"}, {Name: "TZ", Value: "UTC"},
		},
		SecretSlots: []domain.SecretSlot{}, FixtureRecipeDigest: fixtureRecipe.Digest(),
		Readiness:           domain.Readiness{Kind: domain.FixtureOwnedReadiness, SignalName: readiness.SignalName()},
		CapturePolicyDigest: capture.Digest(), ProjectionDefinition: projection.Binding(),
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: 2, ConfirmationRepeats: 2,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1,
		},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: runnerprofile.NodeToolConstraintV1}},
		Budgets: domain.Budgets{
			CandidateCount: 2, MaterializedEntryCount: 16, MaterializedBytesPerWorld: 1 << 20,
			SingleBlobBytes: 1 << 18, ReadinessMS: readinessMS, ProbeMS: 200, TeardownMS: 600,
			StdoutBytes: 16 << 10, StderrBytes: 16 << 10, HTTPBodyBytes: capture.BodyBytes(),
			ProposedShrinkStimuli: 4, TotalCandidateTrials: 8, ShrinkWallMS: 10_000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := httpmodel.BindExecution(plan, stimulus, start, capture, readiness, projection)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := declaration.Bind(plan)
	if err != nil || len(bound) != 2 {
		t.Fatalf("bind selected candidates: count=%d err=%v", len(bound), err)
	}
	registry, err := NewToolRegistry(ctx, resolvedPrivateTempDir(t), ToolSpec{
		Name: "node", AbsolutePath: nodeExecutable,
		VersionConstraint: runnerprofile.NodeToolConstraintV1, VersionArgs: []string{"--version"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return portableHTTPNegativeHarness{
		binding: binding, candidate: bound[0], registry: registry, allocationRoot: resolvedPrivateTempDir(t),
	}
}

func resolvedExecutable(t *testing.T, name string) string {
	t.Helper()
	path := os.Getenv("COUNTERSHAPE_" + strings.ToUpper(name))
	if path == "" {
		var err error
		path, err = exec.LookPath(name)
		if err != nil {
			t.Skipf("%s is not installed", name)
		}
	} else if !filepath.IsAbs(path) {
		t.Fatalf("explicit %s test executable is not absolute", name)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func (h portableHTTPNegativeHarness) execute(t *testing.T, nonce string) Result {
	t.Helper()
	result, err := ExecuteHTTP(context.Background(), HTTPRequest{
		Binding: h.binding, Candidate: h.candidate, Tools: h.registry, AllocationRoot: h.allocationRoot,
		Purpose: domain.AttemptDiscovery, InstanceNonce: nonce, ScheduleOrdinal: 0,
	})
	if err != nil {
		t.Fatalf("ExecuteHTTP returned an internal error instead of a finalized result: %v", err)
	}
	return result
}

func TestHTTPPortableReadinessFailuresRemainFinalizedReceipts(t *testing.T) {
	const spawnFDHolder = `
import { spawn } from "node:child_process";
spawn(process.execPath, ["-e", "setTimeout(() => {}, 25)"], { stdio: ["ignore", "ignore", "ignore", 3] });
`
	tests := []struct {
		name       string
		program    string
		wantBytes  []byte
		wantEOF    bool
		diagnostic string
	}{
		{
			name: "exit-before-readiness", program: spawnFDHolder + `process.exit(0);`,
			wantBytes: nil, wantEOF: true, diagnostic: "HTTP_PROCESS_EXITED_BEFORE_READINESS",
		},
		{
			name: "malformed-frame-eof",
			program: `import { closeSync, writeSync } from "node:fs";
writeSync(3, Buffer.from("COUNTERSHAPE_READY_V1 00080\n", "ascii"));
closeSync(3);
setTimeout(() => {}, 1000);`,
			wantBytes: []byte("COUNTERSHAPE_READY_V1 00080\n"), wantEOF: true,
			diagnostic: "HTTP_READINESS_PROTOCOL_REJECTED",
		},
		{
			name: "valid-frame-held-open",
			program: `import { writeSync } from "node:fs";
writeSync(3, Buffer.from("COUNTERSHAPE_READY_V1 43127\n", "ascii"));
setTimeout(() => {}, 1000);`,
			wantBytes: []byte("COUNTERSHAPE_READY_V1 43127\n"), wantEOF: false,
			diagnostic: "HTTP_READINESS_TIMEOUT",
		},
		{
			name: "valid-frame-bytes-exit-before-eof-acceptance",
			program: `import { writeSync } from "node:fs";
writeSync(3, Buffer.from("COUNTERSHAPE_READY_V1 43127\n", "ascii"));
` + spawnFDHolder + `process.exit(0);`,
			wantBytes: []byte("COUNTERSHAPE_READY_V1 43127\n"), wantEOF: true,
			diagnostic: "HTTP_PROCESS_EXITED_BEFORE_READINESS",
		},
	}
	for _, test := range tests {
		harness := newPortableHTTPNegativeHarness(t, test.program, 80)
		result := harness.execute(t, "portable-negative-"+test.name)
		wantStates := []domain.AttemptState{
			domain.AttemptAllocated, domain.AttemptMaterializing, domain.AttemptStarting,
			domain.AttemptTearingDown, domain.AttemptFinalized,
		}
		if !slices.Equal(result.StateHistory(), wantStates) {
			t.Fatalf("state history=%v, want %v", result.StateHistory(), wantStates)
		}
		if primary, present := result.FinalizedAttempt().PrimaryControl(); !present || primary != domain.ControlReadinessError {
			t.Fatalf("finalized primary=%q/%t", primary, present)
		}
		process := result.Process()
		if primary, present := process.PrimaryControl(); !present || primary != domain.ControlReadinessError ||
			process.DiagnosticCode() != test.diagnostic || !process.Started() || !process.ProcessGroupOwned() ||
			!process.DirectChildWaited() || !process.DrainsComplete() || !process.FinalGroupProbeClean() ||
			process.TeardownError() || process.OrphanRisk() {
			t.Fatalf("process receipt did not retain a clean controlled readiness failure: primary=%q/%t diagnostic=%q receipt=%+v",
				primary, present, process.DiagnosticCode(), process)
		}
		environment := process.Environment()
		if !slices.Contains(environment, "COUNTERSHAPE_HTTP_READINESS_FD=3") {
			t.Fatalf("portable environment omitted readiness fd3: %q", environment)
		}
		for _, entry := range environment {
			if strings.HasPrefix(entry, "COUNTERSHAPE_HTTP_LISTEN_FD=") || strings.HasPrefix(entry, "COUNTERSHAPE_HTTP_PORT=") {
				t.Fatalf("portable child inherited listener authority: %q", entry)
			}
		}
		readiness, present := result.HTTPReadiness()
		if !present || !readiness.Valid() || readiness.Accepted() || readiness.ListenerFDPresent() ||
			readiness.Authority() != httpPortableReadinessAuthority ||
			readiness.Protocol() != httpmodel.PortableReadinessProtocolV1 ||
			readiness.ReadinessFD() != httpPortableReadinessChildFD ||
			readiness.WorldDigest() != result.World().Digest() ||
			readiness.AttemptArtifactDigest() != result.FinalizedAttempt().ArtifactDigest() ||
			readiness.ExecutionBindingDigest() != harness.binding.Digest() ||
			!slices.Equal(readiness.FrameBytes(), test.wantBytes) || readiness.EOFObserved() != test.wantEOF ||
			readiness.DiagnosticCode() != test.diagnostic || !readiness.FrameDigest().Valid() {
			t.Fatalf("readiness receipt mismatch: present=%t receipt=%+v bytes=%q", present, readiness, readiness.FrameBytes())
		}
		if _, present := result.HTTPExchange(); present {
			t.Fatal("rejected readiness produced an exchange receipt")
		}
		if _, present := result.HTTPInvocationEvidence(); present {
			t.Fatalf("%s: rejected readiness produced an invocation receipt", test.name)
		}
	}
}

func TestAcceptedPortableReadinessWithoutRequestRemainsInvariantFailure(t *testing.T) {
	accepted := httpReadinessPhysical{protocol: httpmodel.PortableReadinessProtocolV1, accepted: true}
	if rejectedPortableReadinessOnly(httpmodel.HTTPPortableExecutionAuthorityV1, accepted) {
		t.Fatal("accepted portable readiness was permitted to omit its encoded request")
	}
}

func TestPortableReadinessPortIsAReportNotListenerOwnership(t *testing.T) {
	frame, err := httpmodel.NewHTTPReadyPortFrame(43127)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := newHTTPReadinessReceipt(
		httpTestDigest(t, "world", "reported-not-owned"),
		httpTestDigest(t, "attempt", "reported-not-owned"),
		httpTestDigest(t, "binding", "reported-not-owned"),
		httpReadinessPhysical{
			protocol: httpmodel.PortableReadinessProtocolV1, readinessFD: httpPortableReadinessChildFD,
			endpoint: "127.0.0.1:43127", port: 43127, frameBytes: frame.CanonicalBytes(),
			bytesObserved: int64(len(frame.CanonicalBytes())), observedByte: frame.CanonicalBytes()[0],
			eofObserved: true, accepted: true,
		},
	)
	if err != nil || !receipt.Valid() {
		t.Fatalf("construct child-reported port receipt: %v", err)
	}
	// No socket is opened by this test. Receipt validity deliberately depends
	// only on exact child-reported bytes and joined execution identity; it is
	// not kernel evidence that the reporting process owns the named listener.
	if receipt.Endpoint() != "127.0.0.1:43127" || receipt.ListenerFDPresent() {
		t.Fatal("child-reported endpoint characterization changed")
	}
}
