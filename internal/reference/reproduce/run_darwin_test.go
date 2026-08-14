package reproduce

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/reference/app"
)

func TestCollectorBuildsOnlyTheBoundedArtifactClosure(t *testing.T) {
	set := collectExactStudySetThroughHandlers(t)
	copied := *set
	closure, err := set.Close()
	if err != nil || !closure.Valid() {
		t.Fatalf("close exact snapshots: closure-valid=%t err=%v", closure.Valid(), err)
	}
	if closure.Status() != ClosureStatus || closure.Authority() != ClosureAuthority || closure.Ceiling() != AuthorityCeiling {
		t.Fatalf("closure policy changed: status=%q authority=%q ceiling=%q",
			closure.Status(), closure.Authority(), closure.Ceiling())
	}
	for _, limitation := range []string{
		"FULL_STUDY_RESOURCE_BOUND_UNVALIDATED",
		"HTTP_FINALIZED_CONTRACT_RUN_AUTHORITY_ABSENT",
		"HTTP_CONTRACT_EXECUTION_CLASSIFICATION_AUTHORITY_ABSENT",
		"TWO_DOMAIN_TARGET_RUN_CLASSIFICATION_REPRODUCTION_UNMET",
		"CLI_OFFICIAL_EXECUTION_GENERALIZATION_UNVALIDATED",
		"U7R_SELF_RECEIPT_ABSENT",
	} {
		if !slices.Contains(closure.Unreceipted(), limitation) {
			t.Fatalf("closure omitted mandatory limitation %q", limitation)
		}
	}

	domains := closure.Domains()
	if len(domains) != 2 || domains[0].Domain() != "http" || domains[0].Authority() != HTTPAuthority ||
		domains[1].Domain() != "cli" || domains[1].Authority() != CLIAuthority ||
		domains[0].Authority() == domains[1].Authority() {
		t.Fatalf("mixed authority projection = %#v", domains)
	}
	for _, domain := range domains {
		runs := domain.Runs()
		if len(runs) != 3 {
			t.Fatalf("%s run count = %d", domain.Domain(), len(runs))
		}
		for index, run := range runs {
			if run.Ordinal() != index+1 || len(run.Manifest()) != 111 || len(run.DeterministicSHA256()) != 5 ||
				len(run.FreshSHA256()) != 8 || !validSHA256(run.ManifestSHA256()) {
				t.Fatalf("%s run %d is incomplete", domain.Domain(), index+1)
			}
		}
	}

	// Every synthetic payload deliberately contains semantic-looking hostile
	// text. Close admits only its opaque object shape and byte identity; none of
	// those caller-authored words can replace the fixed ceiling or limitations.
	policySurface := strings.Join([]string{
		closure.Status(), closure.Authority(), closure.Ceiling(),
		domains[0].Authority(), domains[1].Authority(), strings.Join(closure.Unreceipted(), "\n"),
	}, "\n")
	if strings.Contains(policySurface, "FORGED_CONFORMS_DO_NOT_TRUST") ||
		strings.Contains(policySurface, "FORGED_FINALIZED_RUN_DO_NOT_TRUST") {
		t.Fatalf("opaque payload text escaped into closure authority: %s", policySurface)
	}

	// All public aggregate accessors must be defensive. Mutating their results
	// cannot corrupt the package-minted closure.
	limitations := closure.Unreceipted()
	limitations[0] = "FORGED_LIMITATION"
	domains[0] = DomainSummary{}
	runs := closure.Domains()[0].Runs()
	manifest := runs[0].Manifest()
	manifest[0].SHA256 = strings.Repeat("0", 64)
	deterministic := runs[0].DeterministicSHA256()
	deterministic["source_spec_sha256"] = strings.Repeat("0", 64)
	fresh := runs[0].FreshSHA256()
	fresh["world_instance_sha256"] = strings.Repeat("0", 64)
	if !closure.Valid() || closure.Unreceipted()[0] == "FORGED_LIMITATION" || closure.Domains()[0].Domain() != "http" ||
		closure.Domains()[0].Runs()[0].Manifest()[0].SHA256 == strings.Repeat("0", 64) ||
		closure.Domains()[0].Runs()[0].DeterministicSHA256()["source_spec_sha256"] == strings.Repeat("0", 64) ||
		closure.Domains()[0].Runs()[0].FreshSHA256()["world_instance_sha256"] == strings.Repeat("0", 64) {
		t.Fatal("closure exposed mutable internal state")
	}
	if (Closure{}).Valid() {
		t.Fatal("zero closure was valid")
	}
	if second, err := copied.Close(); ErrorCode(err) != "COLLECTOR_CLOSED" || second.Valid() {
		t.Fatalf("copied collector bypassed one-shot state: closure-valid=%t err=%v", second.Valid(), err)
	}
	if second, err := set.Close(); ErrorCode(err) != "COLLECTOR_CLOSED" || second.Valid() {
		t.Fatalf("collector was not one-shot: closure-valid=%t err=%v", second.Valid(), err)
	}
}

func TestStudySetRefusesHostileArtifactForgeries(t *testing.T) {
	baseline := exactSnapshotRoster(t)
	driftedHTTP2 := captureStudySnapshot(t, "http", 2, func(path string, exact []byte) ([]byte, bool) {
		if path == deterministicArtifactPaths["world_plan_sha256"] {
			exact = bytes.Replace(exact,
				[]byte("FORGED_CONFORMS_DO_NOT_TRUST"),
				[]byte("DRIFTED_CONFORMS_DO_NOT_TRUST"), 1)
		}
		return exact, true
	})
	forgedCLI3Trial := captureStudySnapshot(t, "cli", 3, func(path string, exact []byte) ([]byte, bool) {
		if path == "phases/contract/trial-010.json" {
			exact = bytes.Replace(exact, []byte(`"status":"GREEN"`), []byte(`"status":"CONFORMS"`), 1)
		}
		return exact, true
	})
	missingHTTP1File := captureStudySnapshot(t, "http", 1, func(path string, exact []byte) ([]byte, bool) {
		return exact, path != freshArtifactPaths["captures_sha256"]
	})
	arrayPayloadHTTP1 := captureStudySnapshot(t, "http", 1, func(path string, exact []byte) ([]byte, bool) {
		if path == deterministicArtifactPaths["source_spec_sha256"] {
			return deterministicArtifactBytes("http", "source_spec_sha256", json.RawMessage(`[]`)), true
		}
		return exact, true
	})

	tests := []struct {
		name  string
		input testSnapshotRoster
		code  string
	}{
		{name: "partial roster", input: testSnapshotRoster{HTTP: baseline.HTTP[:2], CLI: baseline.CLI}, code: "ROSTER"},
		{name: "cross-domain relabel", input: mutateSnapshotRoster(baseline, func(input *testSnapshotRoster) { input.HTTP[0] = input.CLI[0] }), code: "ARTIFACT"},
		{name: "ordinal substitution", input: mutateSnapshotRoster(baseline, func(input *testSnapshotRoster) { input.HTTP[0], input.HTTP[1] = input.HTTP[1], input.HTTP[0] }), code: "ARTIFACT"},
		{name: "deterministic drift", input: mutateSnapshotRoster(baseline, func(input *testSnapshotRoster) { input.HTTP[1] = driftedHTTP2 }), code: "DETERMINISTIC"},
		{name: "classification in trial status", input: mutateSnapshotRoster(baseline, func(input *testSnapshotRoster) { input.CLI[2] = forgedCLI3Trial }), code: "TRIAL"},
		{name: "missing artifact", input: mutateSnapshotRoster(baseline, func(input *testSnapshotRoster) { input.HTTP[0] = missingHTTP1File }), code: "ROSTER"},
		{name: "non-object semantic payload", input: mutateSnapshotRoster(baseline, func(input *testSnapshotRoster) { input.HTTP[0] = arrayPayloadHTTP1 }), code: "ARTIFACT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			closure, err := closeSnapshotRoster(test.input)
			if err == nil || closure.Valid() || ErrorCode(err) != test.code ||
				!strings.HasPrefix(err.Error(), "U7_REPRODUCE_"+test.code+":") {
				t.Fatalf("hostile closure valid=%t err=%v code=%q, want %s", closure.Valid(), err, ErrorCode(err), test.code)
			}
			wrapped := fmt.Errorf("outer: %w", err)
			if ErrorCode(wrapped) != test.code {
				t.Fatalf("wrapped refusal code = %q, want %q", ErrorCode(wrapped), test.code)
			}
		})
	}
	duplicate := NewCollector()
	if err := duplicate.capture("http", 1, baseline.HTTP[0]); err != nil {
		t.Fatalf("first capture: %v", err)
	}
	if err := duplicate.capture("http", 1, baseline.HTTP[0]); ErrorCode(err) != "DUPLICATE" {
		t.Fatalf("duplicate capture err=%v code=%q", err, ErrorCode(err))
	}
	if _, err := (*Collector)(nil).Close(); ErrorCode(err) != "COLLECTOR" {
		t.Fatalf("nil collector err=%v code=%q", err, ErrorCode(err))
	}
	if ErrorCode(errors.New("foreign")) != "" {
		t.Fatal("foreign error acquired reproduce authority")
	}
}

func TestClosureValidReplaysCrossRunTopology(t *testing.T) {
	closure, err := closeSnapshotRoster(exactSnapshotRoster(t))
	if err != nil || !closure.Valid() {
		t.Fatalf("baseline closure valid=%t err=%v", closure.Valid(), err)
	}

	tests := []struct {
		name   string
		mutate func(*Closure)
	}{
		{
			name: "within-domain deterministic drift",
			mutate: func(hostile *Closure) {
				rewriteRunDigest(&hostile.domains[0].runs[1], deterministicArtifactPaths["source_spec_sha256"], digestBytes([]byte("hostile deterministic drift")))
			},
		},
		{
			name: "cross-domain deterministic alias",
			mutate: func(hostile *Closure) {
				alias := hostile.domains[0].runs[0].deterministic["source_spec_sha256"]
				for index := range hostile.domains[1].runs {
					rewriteRunDigest(&hostile.domains[1].runs[index], deterministicArtifactPaths["source_spec_sha256"], alias)
				}
			},
		},
		{
			name: "cross-run fresh alias",
			mutate: func(hostile *Closure) {
				alias := hostile.domains[0].runs[0].fresh["world_instance_sha256"]
				rewriteRunDigest(&hostile.domains[1].runs[2], freshArtifactPaths["captures_sha256"], alias)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hostile := cloneClosureForTest(closure)
			test.mutate(&hostile)
			for _, domain := range hostile.domains {
				for _, run := range domain.runs {
					if !validRunSummary(run) {
						t.Fatal("mutation did not preserve the individual run-summary invariant")
					}
				}
			}
			if hostile.Valid() {
				t.Fatal("cross-run forgery remained valid")
			}
		})
	}

	hostile := cloneClosureForTest(closure)
	hostile.status = "FORGED_CONFORMS"
	if hostile.Valid() {
		t.Fatal("forged policy status remained valid")
	}
	hostile = cloneClosureForTest(closure)
	hostile.domains[0].runs[0].manifest[0].SHA256 = strings.Repeat("0", 64)
	if hostile.Valid() {
		t.Fatal("forged manifest remained valid")
	}
}

func TestStudiesReturnsExactDefensiveComposition(t *testing.T) {
	first := Studies()
	if len(first) != 2 || first["http"] == nil || first["cli"] == nil {
		t.Fatalf("installed study map = %#v", first)
	}
	delete(first, "http")
	first["forged"] = func(app.StudyRequest) error { return nil }
	second := Studies()
	keys := make([]string, 0, len(second))
	for key, handler := range second {
		if handler == nil {
			t.Fatalf("installed %s handler is nil", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if !slices.Equal(keys, []string{"cli", "http"}) {
		t.Fatalf("mutated composition leaked across calls: %v", keys)
	}

	invoked := false
	hostile := closeAfterHandler(NewCollector(), "http", func(request app.StudyRequest) error {
		invoked = true
		return request.Workspace.WriteFileExclusive("plausible.json", []byte("{}\n"))
	})
	result := runStudyHandler(t, "http", 1, hostile)
	if !invoked || result.code == app.ExitOK || strings.Contains(result.stdout, `"status":"GREEN"`) || result.stderr != "" {
		t.Fatalf("terminal-gate hostile false-green: invoked=%t result=%#v", invoked, result)
	}

	set := NewCollector()
	wrapped := closeAfterHandler(set, "http", syntheticEvidenceHandler("http", 1, nil))
	firstRun := runStudyHandler(t, "http", 1, wrapped)
	secondRun := runStudyHandler(t, "http", 1, wrapped)
	if firstRun.code != app.ExitOK || secondRun.code == app.ExitOK ||
		strings.Contains(secondRun.stdout, `"status":"GREEN"`) || secondRun.stderr != "" {
		t.Fatalf("duplicate route false-green: first=%#v second=%#v", firstRun, secondRun)
	}
}

func TestCollectorDoesNotRetainAStagedRunWhenTerminalBytesChange(t *testing.T) {
	collector := NewCollector()
	handler := app.CloseStudy(syntheticEvidenceHandler("http", 1, nil), func(request app.StudyRequest, snapshot app.StudyEvidenceSnapshot) (app.StudyTerminalCommit, error) {
		run, err := closeEvidenceRun("http", 1, snapshot)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(request.EvidenceRoot, deterministicArtifactPaths["source_spec_sha256"])
		if err := os.WriteFile(path, []byte("{\"mutated\":true}\n"), 0o600); err != nil {
			return nil, err
		}
		return func() error { return collector.publish("http", 1, run) }, nil
	})
	result := runStudyHandler(t, "http", 1, handler)
	if result.code != app.ExitRefused || !strings.Contains(result.stdout, `"code":"STUDY_EVIDENCE_CLOSURE_REFUSED"`) ||
		strings.Contains(result.stdout, `"status":"GREEN"`) || result.stderr != "" {
		t.Fatalf("post-check mutation false-green: %#v", result)
	}
	if _, err := collector.Close(); ErrorCode(err) != "ROSTER" {
		t.Fatalf("refused staged run entered collector: err=%v code=%q", err, ErrorCode(err))
	}
	retry := runStudyHandler(t, "http", 1, closeAfterHandler(collector, "http", syntheticEvidenceHandler("http", 1, nil)))
	if retry.code != app.ExitOK || retry.stderr != "" || !strings.Contains(retry.stdout, `"status":"GREEN"`) {
		t.Fatalf("refused staged run could not be retried: %#v", retry)
	}
	duplicate := runStudyHandler(t, "http", 1, closeAfterHandler(collector, "http", syntheticEvidenceHandler("http", 1, nil)))
	if duplicate.code == app.ExitOK || duplicate.stderr != "" || strings.Contains(duplicate.stdout, `"status":"GREEN"`) {
		t.Fatalf("successful retry was not retained exactly once: %#v", duplicate)
	}
}

type evidenceMutation func(path string, exact []byte) ([]byte, bool)

type testSnapshotRoster struct {
	HTTP []app.StudyEvidenceSnapshot
	CLI  []app.StudyEvidenceSnapshot
}

func exactSnapshotRoster(t testing.TB) testSnapshotRoster {
	t.Helper()
	input := testSnapshotRoster{HTTP: make([]app.StudyEvidenceSnapshot, 3), CLI: make([]app.StudyEvidenceSnapshot, 3)}
	for index := 0; index < 3; index++ {
		input.HTTP[index] = captureStudySnapshot(t, "http", index+1, nil)
		input.CLI[index] = captureStudySnapshot(t, "cli", index+1, nil)
	}
	return input
}

func mutateSnapshotRoster(input testSnapshotRoster, mutate func(*testSnapshotRoster)) testSnapshotRoster {
	copy := testSnapshotRoster{
		HTTP: append([]app.StudyEvidenceSnapshot(nil), input.HTTP...),
		CLI:  append([]app.StudyEvidenceSnapshot(nil), input.CLI...),
	}
	mutate(&copy)
	return copy
}

func closeSnapshotRoster(input testSnapshotRoster) (Closure, error) {
	set := NewCollector()
	for index, snapshot := range input.HTTP {
		if err := set.capture("http", index+1, snapshot); err != nil {
			return Closure{}, err
		}
	}
	for index, snapshot := range input.CLI {
		if err := set.capture("cli", index+1, snapshot); err != nil {
			return Closure{}, err
		}
	}
	return set.Close()
}

func collectExactStudySetThroughHandlers(t testing.TB) *Collector {
	t.Helper()
	set := NewCollector()
	for _, domain := range []string{"http", "cli"} {
		for ordinal := 1; ordinal <= studyRunCount; ordinal++ {
			handlers := set.studiesWithHandlers(
				syntheticEvidenceHandler("cli", ordinal, nil),
				syntheticEvidenceHandler("http", ordinal, nil),
			)
			handler := handlers[domain]
			result := runStudyHandler(t, domain, ordinal, handler)
			want := fmt.Sprintf("{\"schema_version\":\"%s\",\"domain\":\"%s\",\"ordinal\":%d,\"status\":\"GREEN\"}\n",
				app.StudyDomainResultSchemaVersion, domain, ordinal)
			if result.code != app.ExitOK || result.stdout != want || result.stderr != "" {
				t.Fatalf("collect %s/%d result=%#v", domain, ordinal, result)
			}
		}
	}
	return set
}

func captureStudySnapshot(t testing.TB, domain string, ordinal int, mutate evidenceMutation) app.StudyEvidenceSnapshot {
	t.Helper()
	var captured app.StudyEvidenceSnapshot
	closed := false
	handler := app.CloseStudy(syntheticEvidenceHandler(domain, ordinal, mutate), func(request app.StudyRequest, snapshot app.StudyEvidenceSnapshot) (app.StudyTerminalCommit, error) {
		if request.Domain != domain || request.Ordinal != ordinal {
			return nil, fmt.Errorf("snapshot request changed")
		}
		// Accessor mutations must affect only defensive copies.
		directories := snapshot.Directories()
		files := snapshot.Files()
		if len(directories) > 0 {
			directories[0] = "forged"
		}
		if len(files) > 0 {
			exact := files[0].ExactBytes()
			exact[0] ^= 0xff
		}
		return func() error {
			captured = snapshot
			closed = true
			return nil
		}, nil
	})
	result := runStudyHandler(t, domain, ordinal, handler)
	want := fmt.Sprintf("{\"schema_version\":\"%s\",\"domain\":\"%s\",\"ordinal\":%d,\"status\":\"GREEN\"}\n",
		app.StudyDomainResultSchemaVersion, domain, ordinal)
	if result.code != app.ExitOK || result.stdout != want || result.stderr != "" || !closed {
		t.Fatalf("capture %s/%d result=%#v closed=%t", domain, ordinal, result, closed)
	}
	return captured
}

func syntheticEvidenceHandler(domain string, ordinal int, mutate evidenceMutation) app.StudyHandler {
	return func(request app.StudyRequest) error {
		for _, directory := range exactEvidenceDirectories {
			if err := request.Workspace.CreateDirectory(directory); err != nil {
				return err
			}
		}
		for _, path := range expectedEvidencePaths() {
			exact := testEvidenceBytes(domain, ordinal, path)
			include := true
			if mutate != nil {
				exact, include = mutate(path, append([]byte(nil), exact...))
			}
			if !include {
				continue
			}
			if err := request.Workspace.WriteFileExclusive(path, exact); err != nil {
				return err
			}
		}
		return nil
	}
}

func cloneClosureForTest(input Closure) Closure {
	copy := input
	copy.unreceipted = append([]string(nil), input.unreceipted...)
	copy.domains = make([]DomainSummary, len(input.domains))
	for index, domain := range input.domains {
		copy.domains[index] = DomainSummary{domain: domain.domain, authority: domain.authority, runs: cloneRuns(domain.runs)}
	}
	return copy
}

func rewriteRunDigest(run *RunSummary, path, digest string) {
	found := false
	manifest := make([]manifestEntry, len(run.manifest))
	for index := range run.manifest {
		if run.manifest[index].Path == path {
			run.manifest[index].SHA256 = digest
			found = true
		}
		file := run.manifest[index]
		manifest[index] = manifestEntry{Path: file.Path, Mode: "0600", Bytes: file.Bytes, SHA256: file.SHA256}
	}
	if !found {
		panic("test digest path absent: " + path)
	}
	for field, expected := range deterministicArtifactPaths {
		if path == expected {
			run.deterministic[field] = digest
		}
	}
	for field, expected := range freshArtifactPaths {
		if path == expected {
			run.fresh[field] = digest
		}
	}
	line, err := encodeCanonicalLine(manifest)
	if err != nil {
		panic(err)
	}
	run.manifestSHA = digestBytes(line)
}

func testEvidenceBytes(domain string, ordinal int, path string) []byte {
	for field, expected := range deterministicArtifactPaths {
		if path == expected {
			payload := mustJSON(map[string]any{
				"semantic_claim": "FORGED_CONFORMS_DO_NOT_TRUST",
				"source":         domain + ":" + field,
			})
			return deterministicArtifactBytes(domain, field, payload)
		}
	}
	for field, expected := range freshArtifactPaths {
		if path == expected {
			payload := mustJSON(map[string]any{
				"finalized_contract_run": "FORGED_FINALIZED_RUN_DO_NOT_TRUST",
				"source":                 fmt.Sprintf("%s:%d:%s", domain, ordinal, field),
			})
			return freshArtifactBytes(domain, ordinal, field, payload)
		}
	}
	remainder := strings.TrimPrefix(path, "phases/")
	separator := strings.IndexByte(remainder, '/')
	if separator < 1 {
		panic("unexpected evidence path: " + path)
	}
	phase := remainder[:separator]
	var trial int
	if _, err := fmt.Sscanf(remainder[separator+1:], "trial-%03d.json", &trial); err != nil {
		panic("unexpected trial path: " + path)
	}
	return mustJSONLine(struct {
		SchemaVersion string `json:"schema_version"`
		Domain        string `json:"domain"`
		Ordinal       int    `json:"ordinal"`
		Phase         string `json:"phase"`
		Trial         int    `json:"trial"`
		Status        string `json:"status"`
	}{trialSchema, domain, ordinal, phase, trial, greenStatus})
}

func deterministicArtifactBytes(domain, field string, payload json.RawMessage) []byte {
	return mustJSONLine(struct {
		SchemaVersion string          `json:"schema_version"`
		Domain        string          `json:"domain"`
		Artifact      string          `json:"artifact"`
		Payload       json.RawMessage `json:"payload"`
	}{artifactSchema, domain, field, payload})
}

func freshArtifactBytes(domain string, ordinal int, field string, payload json.RawMessage) []byte {
	return mustJSONLine(struct {
		SchemaVersion string          `json:"schema_version"`
		Domain        string          `json:"domain"`
		Ordinal       int             `json:"ordinal"`
		Artifact      string          `json:"artifact"`
		Payload       json.RawMessage `json:"payload"`
	}{artifactSchema, domain, ordinal, field, payload})
}

func mustJSON(value any) json.RawMessage {
	exact, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return exact
}

func mustJSONLine(value any) []byte {
	exact, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return append(exact, '\n')
}

type studyHandlerRun struct {
	code           int
	stdout, stderr string
}

func runStudyHandler(t testing.TB, domain string, ordinal int, handler app.StudyHandler) studyHandlerRun {
	t.Helper()
	root := privateTestDirectory(t, "route")
	working := privateTestSubdirectory(t, root, "working")
	scratch := privateTestSubdirectory(t, root, "scratch")
	evidence := privateTestSubdirectory(t, root, "evidence")
	tools := privateTestSubdirectory(t, root, "tools")
	node := testExecutable(t, tools, "node")
	git := testExecutable(t, tools, "git")
	environment := map[string]string{
		"COUNTERSHAPE_STUDY_DOMAIN":  domain,
		"COUNTERSHAPE_STUDY_ORDINAL": fmt.Sprintf("%d", ordinal),
		"COUNTERSHAPE_EVIDENCE_ROOT": evidence,
		"COUNTERSHAPE_NODE":          node,
		"COUNTERSHAPE_GIT":           git,
		"TMPDIR":                     scratch,
	}
	var stdout, stderr bytes.Buffer
	code := app.Run([]string{"study", domain, "--json"}, app.Runtime{
		Stdout: &stdout, Stderr: &stderr, WorkingDirectory: working, Context: context.Background(),
		Studies: map[string]app.StudyHandler{domain: handler},
		LookupEnvironment: func(key string) (string, bool) {
			value, present := environment[key]
			return value, present
		},
	})
	return studyHandlerRun{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func privateTestDirectory(t testing.TB, label string) string {
	t.Helper()
	path, err := os.MkdirTemp("", "countershape-u7d-"+label+"-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) || filepath.Clean(resolved) != resolved {
		t.Fatalf("private root cannot be canonicalized: path=%q resolved=%q err=%v", path, resolved, err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(resolved) })
	return resolved
}

func privateTestSubdirectory(t testing.TB, parent, name string) string {
	t.Helper()
	path := filepath.Join(parent, name)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func testExecutable(t testing.TB, root, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
