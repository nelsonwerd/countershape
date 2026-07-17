//go:build darwin && arm64

package compiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/adapters/cli"
	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/emit/node/internal/compilation"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	contractfixtures "github.com/nelsonwerd/countershape/testkit/contracts"
)

const physicalSubjectWatchdog = "setTimeout(() => process.exit(70), 15_000).unref();\n"

type generatedContractHooks struct {
	beforeRun  func(t *testing.T, bundleRoot, targetRoot string)
	afterStart func(t *testing.T, runtimeParent, bundleRoot, targetRoot string) error
}

func TestGeneratedCLIContractRunsFromPrivateTargetInventory(t *testing.T) {
	subject := []byte(`import { readFileSync, statSync, writeFileSync } from "node:fs";
import { basename, dirname, join } from "node:path";

const expectedKeys = [
  "APP_MODE", "COUNTERSHAPE_ATTEMPT_ID", "COUNTERSHAPE_EVIDENCE_ROOT", "COUNTERSHAPE_FIXTURE_ROOT",
  "COUNTERSHAPE_SCHEDULE_ORDINAL", "COUNTERSHAPE_SCHEDULE_REPETITION", "COUNTERSHAPE_STATE_ROOT", "HOME",
  "LANG", "NO_COLOR", "TMPDIR", "TZ", "XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME",
  "__CF_USER_TEXT_ENCODING",
];
const keys = Object.keys(process.env).sort();
const userTextEncoding = /^0x([0-9A-F]+):0x0:0x0$/.exec(process.env.__CF_USER_TEXT_ENCODING ?? "");
const valid = JSON.stringify(keys) === JSON.stringify(expectedKeys) && basename(process.cwd()) === "candidate" &&
  process.env.APP_MODE === "public-contract-test" && process.env.OPTIONAL_FLAG === undefined &&
  process.env.PATH === undefined && process.env.HTTP_PROXY === undefined && process.env.npm_config_registry === undefined &&
  process.env.NODE_OPTIONS === undefined &&
  process.env.COUNTERSHAPE_TEST_SECRET === undefined && /^attempt:[0-9a-f]{64}$/.test(process.env.COUNTERSHAPE_ATTEMPT_ID) &&
  process.env.COUNTERSHAPE_SCHEDULE_ORDINAL === "0" && process.env.COUNTERSHAPE_SCHEDULE_REPETITION === "0" &&
  userTextEncoding !== null && Number.parseInt(userTextEncoding[1], 16) === process.getuid();
if (!valid) process.exit(65);
writeFileSync(join(process.cwd(), "private-copy-only.txt"), "private\n", { mode: 0o600 });
process.stdout.write("ok\n");
`)
	stdout, stderr, err := runGeneratedCLISubject(t, subject)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "CONFORMS", "NONE", true)
}

func TestGeneratedCLIContractWaitsForExactPresentStdinEOF(t *testing.T) {
	subject := []byte(`const chunks = [];
process.stdin.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
process.stdin.on("end", () => {
  if (!Buffer.concat(chunks).equals(Buffer.from("contract-input"))) process.exit(65);
  process.stdout.write("ok\n");
});
process.stdin.resume();
`)
	stdout, stderr, err := runGeneratedCLISubject(t, subject)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "CONFORMS", "NONE", true)
}

func TestGeneratedCLIReadmeShellCommandSupportsPathsWithSpaces(t *testing.T) {
	bundle, err := Compile(testCompilationInput(t))
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	bundleRoot := filepath.Join(parent, "bundle root with spaces")
	targetRoot := filepath.Join(parent, "target root with spaces")
	runtimeParent := filepath.Join(parent, "runtime root with spaces")
	runnerHome := filepath.Join(parent, "runner home with spaces")
	for _, directory := range []string{bundleRoot, filepath.Join(targetRoot, "fixture"), runtimeParent, runnerHome} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeGeneratedBundle(t, bundleRoot, bundle)
	subject := append([]byte(physicalSubjectWatchdog), []byte(`const chunks = [];
process.stdin.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
process.stdin.on("end", () => {
  if (!Buffer.concat(chunks).equals(Buffer.from("contract-input"))) process.exit(65);
  process.stdout.write("ok\n");
});
process.stdin.resume();
`)...)
	writeExactTestFile(t, filepath.Join(targetRoot, "fixture", "subject.mjs"), subject, 0o644)

	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	node, err = filepath.Abs(node)
	if err != nil {
		t.Fatal(err)
	}
	node, err = filepath.EvalSymlinks(node)
	if err != nil {
		t.Fatal(err)
	}
	readme := string(bundle.Files()[0].Content())
	runSection := strings.Index(readme, "## Run it\n")
	if runSection < 0 {
		t.Fatal("README has no run section")
	}
	fenceStart := strings.Index(readme[runSection:], "```text\n")
	if fenceStart < 0 {
		t.Fatal("README run section has no command fence")
	}
	fenceStart += runSection + len("```text\n")
	fenceEnd := strings.Index(readme[fenceStart:], "\n```")
	if fenceEnd < 0 {
		t.Fatal("README command fence is unterminated")
	}
	script := readme[fenceStart : fenceStart+fenceEnd]
	for placeholder, value := range map[string]string{
		"<absolute-prepared-target-root>": targetRoot,
		"<absolute-node-executable>":      node,
		"<absolute-bundle-root>":          bundleRoot,
	} {
		script = strings.ReplaceAll(script, placeholder, value)
	}
	if strings.Contains(script, "<absolute-") {
		t.Fatal("README shell command retains an unresolved placeholder")
	}
	if !strings.Contains(script, " &&\n") {
		t.Fatal("README shell command does not fail closed when changing target directory")
	}

	bundleBefore := snapshotGeneratedTree(t, bundleRoot)
	targetBefore := snapshotGeneratedTree(t, targetRoot)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	shell := os.Getenv("COUNTERSHAPE_SH")
	if shell == "" {
		shell = "/bin/sh"
	}
	shell, err = filepath.Abs(shell)
	if err != nil {
		t.Fatal(err)
	}
	shell, err = filepath.EvalSymlinks(shell)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, shell, "-c", script)
	command.Env = []string{
		"COUNTERSHAPE_TEST_SECRET=must-not-reach-subject", "HOME=" + runnerHome,
		"LANG=C", "LC_ALL=C", "NODE_OPTIONS=--no-warnings", "NO_COLOR=1",
		"PATH=/ambient/path/must/not/reach/subject", "TMPDIR=" + runtimeParent, "TZ=UTC",
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("README shell command exceeded its outer deadline: %v", ctx.Err())
	}
	assertGeneratedDiagnostic(t, stdout.String(), stderr.String(), runErr, "CONFORMS", "NONE", true)
	if !equalSnapshot(bundleBefore, snapshotGeneratedTree(t, bundleRoot)) ||
		!equalSnapshot(targetBefore, snapshotGeneratedTree(t, targetRoot)) {
		t.Fatal("README shell command mutated its supplied bundle or target root")
	}
	for label, directory := range map[string]string{"runtime": runtimeParent, "home": runnerHome} {
		entries, readErr := os.ReadDir(directory)
		if readErr != nil || len(entries) != 0 {
			t.Fatalf("README shell command left %s residue: entries=%v err=%v", label, entries, readErr)
		}
	}

	probeRoot := filepath.Join(parent, "node probe root with spaces")
	if err := os.MkdirAll(probeRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(probeRoot, "node probe with spaces")
	marker := filepath.Join(probeRoot, "node-reached")
	writeExactTestFile(t, probe, []byte("#!/bin/sh\n: > \"$COUNTERSHAPE_NODE_PROBE_MARKER\"\nexit 93\n"), 0o700)
	missingTarget := filepath.Join(parent, "missing target with spaces")
	negativeScript := readme[fenceStart : fenceStart+fenceEnd]
	for placeholder, value := range map[string]string{
		"<absolute-prepared-target-root>": missingTarget,
		"<absolute-node-executable>":      probe,
		"<absolute-bundle-root>":          bundleRoot,
	} {
		negativeScript = strings.ReplaceAll(negativeScript, placeholder, value)
	}
	negativeContext, negativeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer negativeCancel()
	negativeCommand := exec.CommandContext(negativeContext, shell, "-c", negativeScript)
	negativeCommand.Env = []string{
		"COUNTERSHAPE_NODE_PROBE_MARKER=" + marker,
		"HOME=" + runnerHome, "LANG=C", "LC_ALL=C", "NO_COLOR=1",
		"PATH=/usr/bin:/bin", "TMPDIR=" + runtimeParent, "TZ=UTC",
	}
	if negativeErr := negativeCommand.Run(); negativeErr == nil {
		t.Fatal("README shell command unexpectedly succeeded for a missing prepared target")
	}
	if negativeContext.Err() != nil {
		t.Fatalf("README missing-target refusal exceeded its deadline: %v", negativeContext.Err())
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("README missing-target refusal reached Node probe: %v", statErr)
	}
}

func TestGeneratedCLIContractClassifiesPredicateContradiction(t *testing.T) {
	stdout, stderr, err := runGeneratedCLISubject(t, []byte("process.stdout.write(\"different\\n\");\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "CONTRADICTS", "PREDICATE_MISMATCH", false)
}

func TestGeneratedCLIContractIgnoresUnselectedCompletionDifferences(t *testing.T) {
	tests := []struct {
		name    string
		subject []byte
	}{
		{"exit-code-2", []byte("process.stdout.write(\"ok\\n\", () => process.exit(2));\n")},
		{"signal-term", []byte("process.stdout.write(\"ok\\n\", () => process.kill(process.pid, \"SIGTERM\"));\n")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := runGeneratedCLISubject(t, test.subject)
			assertGeneratedDiagnostic(t, stdout, stderr, err, "CONFORMS", "NONE", true)
		})
	}
}

func TestGeneratedCLIContractPreservesSelectedSignalCompletion(t *testing.T) {
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	input := testCLICompletionCompilationInput(t, source, "SIGNALED")
	subject := []byte(`process.stdin.on("end", () => process.kill(process.pid, "SIGTERM"));
process.stdin.resume();
`)
	stdout, stderr, runErr := runGeneratedCLISubjectWithInput(t, input, subject)
	assertGeneratedDiagnostic(t, stdout, stderr, runErr, "CONFORMS", "NONE", true)
}

func TestGeneratedCLIContractClassifiesTimeout(t *testing.T) {
	stdout, stderr, err := runGeneratedCLISubject(t, []byte("setInterval(() => {}, 1000);\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "TIMEOUT", false)
}

func TestGeneratedCLIContractEnforcesExactOutputCaps(t *testing.T) {
	stdoutCap := bytes.Repeat([]byte("x"), 32<<10)
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	input := testCLICompilationInput(t, source, [][]byte{stdoutCap}, compilation.ActionAllowObserved)
	stdout, stderr, err := runGeneratedCLISubjectWithInput(t, input, []byte("process.stdout.write(Buffer.alloc(32 << 10, 0x78));\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "CONFORMS", "NONE", true)

	stdout, stderr, err = runGeneratedCLISubject(t, []byte("process.stdout.write(Buffer.alloc((32 << 10) + 1, 0x78));\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "OUTPUT_LIMIT", false)

	stdout, stderr, err = runGeneratedCLISubject(t, []byte("process.stderr.write(Buffer.alloc(16 << 10, 0x79)); process.stdout.write(\"ok\\n\");\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "CONFORMS", "NONE", true)

	stdout, stderr, err = runGeneratedCLISubject(t, []byte("process.stderr.write(Buffer.alloc((16 << 10) + 1, 0x79)); process.stdout.write(\"ok\\n\");\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "OUTPUT_LIMIT", false)
}

func TestGeneratedCLIContractAcceptsSecondAllowManyTuple(t *testing.T) {
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	input := testCLICompilationInput(t, source, [][]byte{[]byte("first\n"), []byte("second\n")}, compilation.ActionAllowObserved)
	stdout, stderr, runErr := runGeneratedCLISubjectWithInput(t, input, []byte("process.stdout.write(\"second\\n\");\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, runErr, "CONFORMS", "NONE", true)
}

func TestGeneratedCLIContractCustomExpectationContradicts(t *testing.T) {
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	input := testCLICompilationInput(t, source, [][]byte{[]byte("expected\n")}, compilation.ActionCustomExpectation)
	stdout, stderr, runErr := runGeneratedCLISubjectWithInput(t, input, []byte("process.stdout.write(\"observed\\n\");\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, runErr, "CONTRADICTS", "PREDICATE_MISMATCH", false)
}

func TestGeneratedCLIContractCustomExpectationConforms(t *testing.T) {
	source, err := contractfixtures.CLISource()
	if err != nil {
		t.Fatal(err)
	}
	input := testCLICompilationInput(t, source, [][]byte{[]byte("expected\n")}, compilation.ActionCustomExpectation)
	stdout, stderr, runErr := runGeneratedCLISubjectWithInput(t, input, []byte("process.stdout.write(\"expected\\n\");\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, runErr, "CONFORMS", "NONE", true)
}

func TestGeneratedCLIContractDistinguishesAbsentAndPresentEmptyStdin(t *testing.T) {
	empty, err := cli.PresentStdin([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name                string
		stdin               cli.CLIStdin
		wantCharacterDevice bool
	}{
		{"absent", cli.AbsentStdin(), true},
		{"present-empty", empty, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, sourceErr := contractfixtures.CLISourceWithStdin(test.stdin)
			if sourceErr != nil {
				t.Fatal(sourceErr)
			}
			input := testCLICompilationInput(t, source, [][]byte{[]byte("ok\n")}, compilation.ActionAllowObserved)
			subject := []byte(fmt.Sprintf(`import { fstatSync, readFileSync } from "node:fs";
const metadata = fstatSync(0);
const bytes = readFileSync(0);
if (bytes.length !== 0 || metadata.isCharacterDevice() !== %t) process.stdout.write("wrong\n");
else process.stdout.write("ok\n");
`, test.wantCharacterDevice))
			stdout, stderr, runErr := runGeneratedCLISubjectWithInput(t, input, subject)
			assertGeneratedDiagnostic(t, stdout, stderr, runErr, "CONFORMS", "NONE", true)
		})
	}
}

func TestGeneratedHTTPContractRunsFromPrivateTargetInventory(t *testing.T) {
	source, sourceErr := contractfixtures.HTTPSource()
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	input := testHTTPCompilationInputForSource(t, source, "200")
	server := []byte(`import { createServer } from "node:http";
import { closeSync, readdirSync, statSync, writeSync } from "node:fs";
import { basename, dirname } from "node:path";

const server = createServer((request, response) => {
  const expectedKeys = [
    "COUNTERSHAPE_ATTEMPT_ID", "COUNTERSHAPE_EVIDENCE_ROOT", "COUNTERSHAPE_FIXTURE_ROOT",
    "COUNTERSHAPE_HTTP_READINESS_FD", "COUNTERSHAPE_HTTP_STIMULUS_DIGEST", "COUNTERSHAPE_SCHEDULE_ORDINAL",
    "COUNTERSHAPE_SCHEDULE_REPETITION", "COUNTERSHAPE_STATE_ROOT", "HOME", "LANG", "NO_COLOR", "TMPDIR", "TZ",
    "XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME",
    "__CF_USER_TEXT_ENCODING",
  ];
  const userTextEncoding = /^0x([0-9A-F]+):0x0:0x0$/.exec(process.env.__CF_USER_TEXT_ENCODING ?? "");
  const valid = JSON.stringify(Object.keys(process.env).sort()) === JSON.stringify(expectedKeys) &&
    basename(process.cwd()) === "candidate" && process.env.COUNTERSHAPE_HTTP_READINESS_FD === "3" &&
    process.env.COUNTERSHAPE_HTTP_STIMULUS_DIGEST === "__EXPECTED_STIMULUS_DIGEST__" &&
    process.env.COUNTERSHAPE_SCHEDULE_ORDINAL === "0" && process.env.COUNTERSHAPE_SCHEDULE_REPETITION === "0" &&
    process.env.PATH === undefined && process.env.HTTP_PROXY === undefined && process.env.npm_config_registry === undefined &&
    process.env.NODE_OPTIONS === undefined && process.env.COUNTERSHAPE_TEST_SECRET === undefined &&
    userTextEncoding !== null && Number.parseInt(userTextEncoding[1], 16) === process.getuid() &&
    request.method === "GET" && request.url === "/contract?contract&mode=exact" &&
    request.headers["x-countershape-contract"] === "v1" && request.headers["content-length"] === undefined;
  const body = Buffer.from(JSON.stringify({
    request_id: "runtime-smoke",
    scratch_root: process.env.COUNTERSHAPE_STATE_ROOT,
    kind: valid ? "ok" : "invalid-request",
    metadata: { mode: "test" },
  }));
  response.statusCode = valid ? 200 : 500;
  response.setHeader("content-type", "application/json");
  response.setHeader("content-length", String(body.length));
  response.end(body);
});

server.listen(0, "127.0.0.1", () => {
  const address = server.address();
  writeSync(3, ` + "`COUNTERSHAPE_READY_V1 ${address.port}\\n`" + `);
  closeSync(3);
});
`)
	server = bytes.ReplaceAll(server, []byte("__EXPECTED_STIMULUS_DIGEST__"), []byte(source.StimulusDigest().String()))
	stdout, stderr, err := runGeneratedHTTPSubjectWithInput(t, input, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "CONFORMS", "NONE", true)
}

func TestGeneratedHTTPContractClassifiesNaturalNonzeroExitAfterResponse(t *testing.T) {
	server := []byte(`import { createServer } from "node:http";
import { closeSync, writeSync } from "node:fs";

const server = createServer((_request, response) => {
  const body = Buffer.from(JSON.stringify({
    request_id: "natural-exit",
    scratch_root: process.env.COUNTERSHAPE_STATE_ROOT,
    kind: "ok",
    metadata: { mode: "test" },
  }));
  response.statusCode = 200;
  response.setHeader("content-type", "application/json");
  response.setHeader("content-length", String(body.length));
  response.end(body, () => {
    process.exitCode = 64;
    server.close();
  });
});

server.listen(0, "127.0.0.1", () => {
  const address = server.address();
  writeSync(3, ` + "`COUNTERSHAPE_READY_V1 ${address.port}\\n`" + `);
  closeSync(3);
});
`)
	stdout, stderr, err := runGeneratedHTTPSubject(t, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "TRANSPORT_FAILED", false)
}

func TestGeneratedHTTPContractKeepsExitBeforeReadinessAsReadinessFailure(t *testing.T) {
	stdout, stderr, err := runGeneratedHTTPSubject(t, []byte("process.exit(64);\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "READINESS_FAILED", false)
}

func TestGeneratedHTTPContractRequiresReadinessEOF(t *testing.T) {
	server := []byte(`import { writeSync } from "node:fs";
writeSync(3, "COUNTERSHAPE_READY_V1 1\n");
setInterval(() => {}, 1000);
`)
	stdout, stderr, err := runGeneratedHTTPSubject(t, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "READINESS_FAILED", false)
}

func TestGeneratedHTTPContractPrioritizesOutputOverflowBeforeReadiness(t *testing.T) {
	server := []byte(`process.stdout.write(Buffer.alloc((32 << 10) + 1, 0x78));
setInterval(() => {}, 1000);
`)
	stdout, stderr, err := runGeneratedHTTPSubject(t, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "OUTPUT_LIMIT", false)
}

func TestGeneratedHTTPContractInterruptsProbeOnProcessOutputOverflow(t *testing.T) {
	server := []byte(`import { createServer } from "node:http";
import { closeSync, writeSync } from "node:fs";

const server = createServer(() => {
  process.stdout.write(Buffer.alloc((32 << 10) + 1, 0x78));
});
server.listen(0, "127.0.0.1", () => {
  writeSync(3, ` + "`COUNTERSHAPE_READY_V1 ${server.address().port}\n`" + `);
  closeSync(3);
});
`)
	stdout, stderr, err := runGeneratedHTTPSubject(t, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "OUTPUT_LIMIT", false)
}

func TestGeneratedHTTPContractRoutesEmptyClosedResponseToParser(t *testing.T) {
	server := []byte(`import { createServer } from "node:net";
import { closeSync, writeSync } from "node:fs";

const server = createServer({ allowHalfOpen: true }, (socket) => {
  socket.on("data", () => {});
  socket.on("end", () => {
    socket.end();
    server.close();
  });
});
server.listen(0, "127.0.0.1", () => {
  writeSync(3, ` + "`COUNTERSHAPE_READY_V1 ${server.address().port}\n`" + `);
  closeSync(3);
});
`)
	stdout, stderr, err := runGeneratedHTTPSubject(t, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "RESPONSE_PARSE_FAILED", false)
}

func TestGeneratedHTTPContractTreatsHTTP500AsEligibleBehavior(t *testing.T) {
	server := generatedHTTPJSONServer(500, "process.env.COUNTERSHAPE_STATE_ROOT", "ok")
	stdout, stderr, err := runGeneratedHTTPSubject(t, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "CONTRADICTS", "PREDICATE_MISMATCH", false)

	source, sourceErr := contractfixtures.HTTPSource()
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	input := testHTTPCompilationInputForSource(t, source, "500")
	stdout, stderr, err = runGeneratedHTTPSubjectWithInput(t, input, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "CONFORMS", "NONE", true)
}

func TestGeneratedHTTPContractClassifiesProbeTimeout(t *testing.T) {
	server := []byte(`import { createServer } from "node:net";
import { closeSync, writeSync } from "node:fs";
const server = createServer({ allowHalfOpen: true }, (socket) => socket.on("data", () => {}));
server.listen(0, "127.0.0.1", () => {
  writeSync(3, ` + "`COUNTERSHAPE_READY_V1 ${server.address().port}\n`" + `);
  closeSync(3);
});
`)
	stdout, stderr, err := runGeneratedHTTPSubject(t, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "TIMEOUT", false)
}

func TestGeneratedHTTPContractClassifiesSocketReset(t *testing.T) {
	server := []byte(`import { createServer, Socket } from "node:net";
import { closeSync, writeSync } from "node:fs";
if (typeof Socket.prototype.resetAndDestroy !== "function") process.exit(66);
const server = createServer((socket) => socket.once("data", () => {
  try { socket.resetAndDestroy(); }
  catch { socket.end(); server.close(); }
}));
server.listen(0, "127.0.0.1", () => {
  writeSync(3, ` + "`COUNTERSHAPE_READY_V1 ${server.address().port}\n`" + `);
  closeSync(3);
});
`)
	stdout, stderr, err := runGeneratedHTTPSubject(t, server)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "TRANSPORT_FAILED", false)
}

func TestGeneratedHTTPContractClassifiesProjectionFailure(t *testing.T) {
	stdout, stderr, err := runGeneratedHTTPSubject(t, generatedHTTPJSONServer(200, `"not-the-state-root"`, "ok"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "PROJECTION_FAILED", false)
}

func TestGeneratedHTTPContractEnforcesExactBodyCap(t *testing.T) {
	for _, test := range []struct {
		name     string
		bodySize int
		outcome  string
		reason   string
		pass     bool
	}{
		{"exact", 64 << 10, "CONFORMS", "NONE", true},
		{"plus-one", (64 << 10) + 1, "INELIGIBLE_EXECUTION", "OUTPUT_LIMIT", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := []byte(fmt.Sprintf(`import { createServer } from "node:http";
import { closeSync, writeSync } from "node:fs";
const limit = %d;
const value = { request_id: "body-cap", scratch_root: "", kind: "ok", metadata: { mode: "test" }, padding: "" };
const server = createServer((_request, response) => {
  value.scratch_root = process.env.COUNTERSHAPE_STATE_ROOT;
  const empty = Buffer.byteLength(JSON.stringify(value));
  value.padding = "x".repeat(limit - empty);
  const body = Buffer.from(JSON.stringify(value));
  if (body.length !== limit) process.exit(65);
  response.statusCode = 200;
  response.setHeader("content-type", "application/json");
  response.setHeader("content-length", String(body.length));
  response.end(body);
});
server.listen(0, "127.0.0.1", () => {
  writeSync(3, `+"`COUNTERSHAPE_READY_V1 ${server.address().port}\n`"+`);
  closeSync(3);
});
`, test.bodySize))
			stdout, stderr, err := runGeneratedHTTPSubject(t, server)
			assertGeneratedDiagnostic(t, stdout, stderr, err, test.outcome, test.reason, test.pass)
		})
	}
}

func TestGeneratedHTTPContractRejectsMalformedReadinessFrames(t *testing.T) {
	frames := []struct {
		name  string
		frame []byte
	}{
		{"empty", []byte{}},
		{"partial", []byte("COUNTERSHAPE_READY_V1")},
		{"oversized", bytes.Repeat([]byte("x"), 33)},
		{"wrong-prefix", []byte("COUNTERSHAPE_READY_V2 1\n")},
		{"port-zero", []byte("COUNTERSHAPE_READY_V1 0\n")},
		{"port-too-large", []byte("COUNTERSHAPE_READY_V1 65536\n")},
		{"leading-zero", []byte("COUNTERSHAPE_READY_V1 01\n")},
		{"plus-sign", []byte("COUNTERSHAPE_READY_V1 +1\n")},
		{"extra-space", []byte("COUNTERSHAPE_READY_V1  1\n")},
		{"crlf", []byte("COUNTERSHAPE_READY_V1 1\r\n")},
		{"double-lf", []byte("COUNTERSHAPE_READY_V1 1\n\n")},
		{"nul", []byte("COUNTERSHAPE_READY_V1 1\x00\n")},
		{"high-bit", []byte{'C', 0xff, '\n'}},
		{"trailing-byte", []byte("COUNTERSHAPE_READY_V1 1\nx")},
		{"two-frames", []byte("COUNTERSHAPE_READY_V1 1\nCOUNTERSHAPE_READY_V1 1\n")},
	}
	for _, test := range frames {
		t.Run(test.name, func(t *testing.T) {
			encoded := base64.StdEncoding.EncodeToString(test.frame)
			server := []byte(fmt.Sprintf(`import { closeSync, writeSync } from "node:fs";
writeSync(3, Buffer.from(%q, "base64"));
closeSync(3);
setInterval(() => {}, 1000);
`, encoded))
			stdout, stderr, err := runGeneratedHTTPSubject(t, server)
			assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "READINESS_FAILED", false)
		})
	}
}

func TestGeneratedHTTPContractRejectsMissingReadinessFrame(t *testing.T) {
	stdout, stderr, err := runGeneratedHTTPSubject(t, []byte("setInterval(() => {}, 1000);\n"))
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "READINESS_FAILED", false)
}

func TestGeneratedHTTPContractDistinguishesAbsentAndPresentEmptyBody(t *testing.T) {
	empty, err := counterhttp.PresentBody([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name              string
		body              counterhttp.HTTPBody
		contentLengthJSON string
	}{
		{"absent", counterhttp.AbsentBody(), "undefined"},
		{"present-empty", empty, `"0"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, sourceErr := contractfixtures.HTTPSourceWithBody(test.body)
			if sourceErr != nil {
				t.Fatal(sourceErr)
			}
			input := testHTTPCompilationInputForSource(t, source, "200")
			server := []byte(fmt.Sprintf(`import { createServer } from "node:http";
import { closeSync, writeSync } from "node:fs";
const server = createServer((request, response) => {
  let observed = 0;
  request.on("data", (chunk) => { observed += chunk.length; });
  request.on("end", () => {
    const valid = observed === 0 && request.headers["content-length"] === %s;
    const body = Buffer.from(JSON.stringify({
      request_id: "body-presence", scratch_root: process.env.COUNTERSHAPE_STATE_ROOT,
      kind: valid ? "ok" : "invalid", metadata: { mode: "test" },
    }));
    response.statusCode = 200;
    response.setHeader("content-type", "application/json");
    response.setHeader("content-length", String(body.length));
    response.end(body);
  });
});
server.listen(0, "127.0.0.1", () => {
  writeSync(3, `+"`COUNTERSHAPE_READY_V1 ${server.address().port}\n`"+`);
  closeSync(3);
});
`, test.contentLengthJSON))
			stdout, stderr, runErr := runGeneratedHTTPSubjectWithInput(t, input, server)
			assertGeneratedDiagnostic(t, stdout, stderr, runErr, "CONFORMS", "NONE", true)
		})
	}
}

func TestGeneratedContractEntrypointRejectsEachChangedCompanionBeforeHarnessImport(t *testing.T) {
	for _, path := range []string{"README.md", "contract.test.mjs", "decision.json", "fixture.json", "harness.mjs"} {
		t.Run(path, func(t *testing.T) {
			hooks := generatedContractHooks{beforeRun: func(t *testing.T, bundleRoot, _ string) {
				exact, err := os.ReadFile(filepath.Join(bundleRoot, path))
				if err != nil {
					t.Fatal(err)
				}
				rewriteGeneratedBundleFile(t, bundleRoot, path, append(exact, []byte("// hostile companion drift\n")...), false)
			}}
			stdout, stderr, err := runGeneratedCLISubjectWithInputAndHooks(
				t, testCompilationInput(t), []byte("process.stdout.write(\"ok\\n\");\n"), hooks,
			)
			assertGeneratedDiagnostic(t, stdout, stderr, err, "TAMPER_DETECTED", "COMPANION_INTEGRITY_MISMATCH", false)
		})
	}
}

func TestGeneratedContractReportsMalformedJSONFileEnvelopes(t *testing.T) {
	aliases := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"missing-lf", func(exact []byte) []byte { return append([]byte(nil), exact[:len(exact)-1]...) }},
		{"crlf", func(exact []byte) []byte { return append(append([]byte(nil), exact[:len(exact)-1]...), '\r', '\n') }},
		{"double-lf", func(exact []byte) []byte { return append(append([]byte(nil), exact...), '\n') }},
	}
	for _, path := range []string{"decision.json", "fixture.json", "manifest.json"} {
		for _, alias := range aliases {
			t.Run(path+"/"+alias.name, func(t *testing.T) {
				hooks := generatedContractHooks{beforeRun: func(t *testing.T, bundleRoot, _ string) {
					exact, err := os.ReadFile(filepath.Join(bundleRoot, path))
					if err != nil {
						t.Fatal(err)
					}
					rewriteGeneratedBundleFile(t, bundleRoot, path, alias.mutate(exact), path != "manifest.json")
				}}
				stdout, stderr, err := runGeneratedCLISubjectWithInputAndHooks(
					t, testCompilationInput(t), []byte("process.stdout.write(\"ok\\n\");\n"), hooks,
				)
				assertGeneratedDiagnostic(t, stdout, stderr, err, "MALFORMED_CONTRACT", "CONTRACT_DATA_INVALID", false)
			})
		}
	}
}

func TestGeneratedContractReportsHarnessFailureForVerifiedLoaderFailure(t *testing.T) {
	hooks := generatedContractHooks{beforeRun: func(t *testing.T, bundleRoot, _ string) {
		path := "harness.mjs"
		exact, err := os.ReadFile(filepath.Join(bundleRoot, path))
		if err != nil {
			t.Fatal(err)
		}
		rewriteGeneratedBundleFile(
			t, bundleRoot, path,
			append(exact, []byte("throw new Error(\"test-edge verified loader failure\");\n")...),
			true,
		)
	}}
	stdout, stderr, err := runGeneratedCLISubjectWithInputAndHooks(
		t, testCompilationInput(t), []byte("process.stdout.write(\"ok\\n\");\n"), hooks,
	)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "HARNESS_FAILURE", "INTERNAL_INVARIANT_FAILED", false)
}

func TestGeneratedContractRejectsSourceSymlinkWithoutFollowing(t *testing.T) {
	hooks := generatedContractHooks{beforeRun: func(t *testing.T, _ string, targetRoot string) {
		if err := os.Symlink("subject.mjs", filepath.Join(targetRoot, "fixture", "hostile-link.mjs")); err != nil {
			t.Fatal(err)
		}
	}}
	stdout, stderr, err := runGeneratedCLISubjectWithInputAndHooks(
		t, testCompilationInput(t), []byte("process.stdout.write(\"ok\\n\");\n"), hooks,
	)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "SOURCE_INVENTORY_INVALID", false)
}

func TestGeneratedContractRejectsTransientTargetWriteRestoredBeforeFinalScan(t *testing.T) {
	subject := []byte(`import { existsSync, writeFileSync } from "node:fs";
import { join } from "node:path";
const state = process.env.COUNTERSHAPE_STATE_ROOT;
writeFileSync(join(state, "watch-ready"), "ready\n", { mode: 0o600 });
const awaitRelease = () => {
  if (existsSync(join(state, "watch-release"))) process.stdout.write("ok\n");
  else setTimeout(awaitRelease, 5);
};
awaitRelease();
`)
	hooks := generatedContractHooks{afterStart: func(
		t *testing.T,
		runtimeParent, _, targetRoot string,
	) error {
		ready, err := waitForGeneratedAttemptFile(runtimeParent, filepath.Join("state", "watch-ready"), 5*time.Second)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, "fixture", "subject.mjs")
		exact, err := os.ReadFile(target)
		if err != nil || len(exact) == 0 {
			return fmt.Errorf("read target for transient mutation: %w", err)
		}
		handle, err := os.OpenFile(target, os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("open target for transient mutation: %w", err)
		}
		mutated := []byte{exact[0] ^ 0x01}
		if _, err = handle.WriteAt(mutated, 0); err == nil {
			err = handle.Sync()
		}
		if err == nil {
			_, err = handle.WriteAt(exact[:1], 0)
		}
		if err == nil {
			err = handle.Sync()
		}
		closeErr := handle.Close()
		if err != nil {
			return fmt.Errorf("transient target write: %w", err)
		}
		if closeErr != nil {
			return fmt.Errorf("close transient target writer: %w", closeErr)
		}
		time.Sleep(100 * time.Millisecond)
		writeExactTestFile(t, filepath.Join(filepath.Dir(ready), "watch-release"), []byte("release\n"), 0o600)
		return nil
	}}
	stdout, stderr, err := runGeneratedCLISubjectWithInputAndHooks(t, testCompilationInput(t), subject, hooks)
	assertGeneratedDiagnostic(t, stdout, stderr, err, "INELIGIBLE_EXECUTION", "SOURCE_INVENTORY_INVALID", false)
}

func generatedHTTPJSONServer(status int, scratchRoot, kind string) []byte {
	return []byte(fmt.Sprintf(`import { createServer } from "node:http";
import { closeSync, writeSync } from "node:fs";
const server = createServer((_request, response) => {
  const body = Buffer.from(JSON.stringify({
    request_id: "runtime-matrix", scratch_root: %s, kind: %q, metadata: { mode: "test" },
  }));
  response.statusCode = %d;
  response.setHeader("content-type", "application/json");
  response.setHeader("content-length", String(body.length));
  response.end(body);
});
server.listen(0, "127.0.0.1", () => {
  writeSync(3, `+"`COUNTERSHAPE_READY_V1 ${server.address().port}\n`"+`);
  closeSync(3);
});
`, scratchRoot, kind, status))
}

func runGeneratedCLISubject(t *testing.T, subject []byte) (string, string, error) {
	t.Helper()
	return runGeneratedCLISubjectWithInput(t, testCompilationInput(t), subject)
}

func runGeneratedCLISubjectWithInput(
	t *testing.T,
	input compilation.Input,
	subject []byte,
) (string, string, error) {
	t.Helper()
	return runGeneratedCLISubjectWithInputAndHooks(t, input, subject, generatedContractHooks{})
}

func runGeneratedCLISubjectWithInputAndHooks(
	t *testing.T,
	input compilation.Input,
	subject []byte,
	hooks generatedContractHooks,
) (string, string, error) {
	t.Helper()
	bundle, err := Compile(input)
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	bundleRoot := filepath.Join(parent, "generated bundle")
	targetRoot := filepath.Join(parent, "target source")
	if err := os.Mkdir(bundleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(bundleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(targetRoot, "fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(targetRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(targetRoot, "fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGeneratedBundle(t, bundleRoot, bundle)
	subject = append([]byte(physicalSubjectWatchdog), subject...)
	writeExactTestFile(t, filepath.Join(targetRoot, "fixture", "subject.mjs"), subject, 0o644)
	if hooks.beforeRun != nil {
		hooks.beforeRun(t, bundleRoot, targetRoot)
	}
	return runGeneratedContractWithHooks(t, bundleRoot, targetRoot, hooks)
}

func runGeneratedHTTPSubject(t *testing.T, server []byte) (string, string, error) {
	t.Helper()
	return runGeneratedHTTPSubjectWithInput(t, testHTTPCompilationInput(t), server)
}

func runGeneratedHTTPSubjectWithInput(
	t *testing.T,
	input compilation.Input,
	server []byte,
) (string, string, error) {
	t.Helper()
	bundle, err := Compile(input)
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	bundleRoot := filepath.Join(parent, "generated http bundle")
	targetRoot := filepath.Join(parent, "http target source")
	if err := os.Mkdir(bundleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(bundleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(targetRoot, "fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(targetRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(targetRoot, "fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGeneratedBundle(t, bundleRoot, bundle)
	server = append([]byte(physicalSubjectWatchdog), server...)
	writeExactTestFile(t, filepath.Join(targetRoot, "fixture", "server.mjs"), server, 0o644)
	return runGeneratedContract(t, bundleRoot, targetRoot)
}

func writeGeneratedBundle(t *testing.T, bundleRoot string, bundle model.ContractBundle) {
	t.Helper()
	for _, file := range bundle.Files() {
		writeExactTestFile(t, filepath.Join(bundleRoot, file.Path()), file.Content(), 0o644)
	}
}

func writeExactTestFile(t *testing.T, path string, content []byte, mode fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func rewriteGeneratedBundleFile(
	t *testing.T,
	bundleRoot, path string,
	content []byte,
	repinManifest bool,
) {
	t.Helper()
	writeExactTestFile(t, filepath.Join(bundleRoot, path), content, 0o644)
	if !repinManifest {
		return
	}
	manifestPath := filepath.Join(bundleRoot, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest := decodeTestJSONObject(t, manifestBytes)
	entries, ok := manifest["files"].([]any)
	if !ok || len(entries) != model.ContractFileCount-1 {
		t.Fatal("generated manifest has no exact protected-file roster")
	}
	found := false
	for _, raw := range entries {
		entry, entryOK := raw.(map[string]any)
		if !entryOK {
			t.Fatal("generated manifest entry is not an object")
		}
		if entry["path"] == path {
			entry["byte_count"] = len(content)
			entry["byte_sha256"] = testRawSHA256(content)
			found = true
		}
	}
	if !found {
		t.Fatalf("generated manifest omits %s", path)
	}
	writeExactTestFile(t, manifestPath, append(canonicalTestJSON(t, manifest), '\n'), 0o644)
}

func waitForGeneratedAttemptFile(runtimeParent, relative string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(runtimeParent)
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "countershape-") {
				continue
			}
			candidate := filepath.Join(runtimeParent, entry.Name(), relative)
			if info, statErr := os.Lstat(candidate); statErr == nil && info.Mode().IsRegular() {
				return candidate, nil
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return "", fmt.Errorf("timed out waiting for generated attempt file %s", relative)
}

func runGeneratedContract(t *testing.T, bundleRoot, targetRoot string) (string, string, error) {
	t.Helper()
	return runGeneratedContractWithHooks(t, bundleRoot, targetRoot, generatedContractHooks{})
}

func runGeneratedContractWithHooks(
	t *testing.T,
	bundleRoot, targetRoot string,
	hooks generatedContractHooks,
) (string, string, error) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	node, err = filepath.Abs(node)
	if err != nil {
		t.Fatal(err)
	}
	node, err = filepath.EvalSymlinks(node)
	if err != nil {
		t.Fatal(err)
	}
	runtimeParent := filepath.Join(filepath.Dir(bundleRoot), "runtime temp")
	if err := os.Mkdir(runtimeParent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(runtimeParent, 0o700); err != nil {
		t.Fatal(err)
	}
	runnerHome := filepath.Join(filepath.Dir(bundleRoot), "runner home")
	if err := os.Mkdir(runnerHome, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(runnerHome, 0o700); err != nil {
		t.Fatal(err)
	}
	bundleBefore := snapshotGeneratedTree(t, bundleRoot)
	targetBefore := snapshotGeneratedTree(t, targetRoot)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, node, "--test", "--test-reporter=tap", filepath.Join(bundleRoot, "contract.test.mjs"))
	command.Dir = targetRoot
	command.Env = []string{
		"COUNTERSHAPE_TEST_SECRET=must-not-reach-subject", "HTTP_PROXY=http://127.0.0.1:1",
		"HOME=" + runnerHome, "LANG=C", "LC_ALL=C", "NODE_OPTIONS=--no-warnings", "NO_COLOR=1",
		"PATH=/ambient/path/must/not/reach/subject", "TMPDIR=" + runtimeParent, "TZ=UTC",
		"__CF_USER_TEXT_ENCODING=must-not-reach-subject", "npm_config_registry=https://ambient.invalid/",
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if hooks.afterStart == nil {
		err = command.Run()
	} else {
		err = command.Start()
		if err == nil {
			if hookErr := hooks.afterStart(t, runtimeParent, bundleRoot, targetRoot); hookErr != nil {
				cancel()
				_ = command.Wait()
				t.Fatalf("generated contract after-start hook failed: %v", hookErr)
			}
			err = command.Wait()
		}
	}
	if ctx.Err() != nil {
		t.Fatalf("generated contract exceeded outer test deadline: %v\nstdout:\n%s\nstderr:\n%s", ctx.Err(), stdout.String(), stderr.String())
	}
	if bundleAfter := snapshotGeneratedTree(t, bundleRoot); !equalSnapshot(bundleBefore, bundleAfter) {
		t.Fatalf("generated contract mutated its bundle root:\nbefore=%v\nafter=%v", bundleBefore, bundleAfter)
	}
	if targetAfter := snapshotGeneratedTree(t, targetRoot); !equalSnapshot(targetBefore, targetAfter) {
		t.Fatalf("generated contract mutated its supplied target:\nbefore=%v\nafter=%v", targetBefore, targetAfter)
	}
	entries, readErr := os.ReadDir(runtimeParent)
	if readErr != nil || len(entries) != 0 {
		t.Fatalf("generated contract left runtime temp residue: entries=%v err=%v", entries, readErr)
	}
	homeEntries, homeErr := os.ReadDir(runnerHome)
	if homeErr != nil || len(homeEntries) != 0 {
		t.Fatalf("generated contract wrote runner home residue: entries=%v err=%v", homeEntries, homeErr)
	}
	return stdout.String(), stderr.String(), err
}

func snapshotGeneratedTree(t *testing.T, root string) []string {
	t.Helper()
	var snapshot []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		links := uint64(0)
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			links = uint64(stat.Nlink)
		}
		digest := "-"
		if info.Mode().IsRegular() {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			sum := sha256.Sum256(content)
			digest = hex.EncodeToString(sum[:])
		}
		snapshot = append(snapshot, fmt.Sprintf("%s|%s|%d|%d|%s", filepath.ToSlash(relative), info.Mode(), info.Size(), links, digest))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(snapshot)
	return snapshot
}

func equalSnapshot(left, right []string) bool {
	return strings.Join(left, "\n") == strings.Join(right, "\n")
}

func assertGeneratedDiagnostic(
	t *testing.T,
	stdout, stderr string,
	err error,
	outcome, reason string,
	wantPass bool,
) {
	t.Helper()
	if wantPass && err != nil {
		t.Fatalf("generated contract failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if !wantPass && err == nil {
		t.Fatalf("generated contract unexpectedly passed:\n%s", stdout)
	}
	want := "# COUNTERSHAPE_RESULT_V1|" + outcome + "|" + reason + "\n"
	if stderr != "" || strings.Count(stdout, "COUNTERSHAPE_RESULT_V1") != 1 || strings.Count(stdout, want) != 1 {
		t.Fatalf("machine diagnostic mismatch; stderr=%q want=%q\n%s", stderr, want, stdout)
	}
	if strings.ContainsAny(stdout+stderr, "\r\x00\x1b") {
		t.Fatalf("generated contract emitted CR, NUL, or ANSI bytes: stdout=%q stderr=%q", stdout, stderr)
	}
	if wantPass && (!strings.Contains(stdout, "# pass 1") || !strings.Contains(stdout, "# fail 0")) {
		t.Fatalf("unexpected passing TAP summary:\n%s", stdout)
	}
	if !wantPass && (!strings.Contains(stdout, "# pass 0") || !strings.Contains(stdout, "# fail 1")) {
		t.Fatalf("unexpected failing TAP summary:\n%s", stdout)
	}
}
