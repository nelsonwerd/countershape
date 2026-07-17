//go:build darwin && arm64

package program

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCopiedHarnessLifecycleStateMachines(t *testing.T) {
	harness, err := Bytes(HarnessPath)
	if err != nil {
		t.Fatal(err)
	}
	spawnImport := []byte(`import { spawn } from "node:child_process";`)
	if bytes.Count(harness, spawnImport) != 1 {
		t.Fatalf("harness has %d spawn import seams; want exactly one", bytes.Count(harness, spawnImport))
	}
	harness = bytes.Replace(harness, spawnImport, []byte(`import { spawn as __countershapeRealSpawn } from "node:child_process";
let spawn = __countershapeRealSpawn;`), 1)
	root := t.TempDir()
	harnessPath := filepath.Join(root, "instrumented-harness.mjs")
	testPath := filepath.Join(root, "lifecycle-test.mjs")
	exports := []byte(`
export const __countershapeLifecycleTest = Object.freeze({
  makeRotatingLatch,
  waitForLatch,
  awaitOwnerDecision,
  awaitSpawnConfirmation,
  childOwner,
  waitForGroupAbsence,
  cappedCapture,
  pollCLIPhase,
  pollReadinessPhase,
  pollHTTPProbePhase,
  beginStdinHandoff,
  readinessCapture,
  beginRawHTTPExchange,
  rawHTTPExchange,
  finalJoinComplete,
  refreshNaturalTerminal,
  recordFinalOutputOverflow,
  finalizeChild,
  runCLI,
  startRootWatcher,
  startRootWatcherPair,
  closeRootWatchers,
  setSpawnForTest(value) { spawn = value; },
  resetSpawnForTest() { spawn = __countershapeRealSpawn; },
});
`)
	if err := os.WriteFile(harnessPath, append(harness, exports...), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testPath, []byte(copiedLifecycleTestSource), 0o600); err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, node, testPath, harnessPath)
	command.Dir = root
	command.Env = []string{"LANG=C", "LC_ALL=C", "NO_COLOR=1", "TZ=UTC"}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			t.Fatalf("copied lifecycle test exceeded its deadline: %v", ctx.Err())
		}
		t.Fatalf("copied lifecycle test failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("copied lifecycle test emitted output; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

const copiedLifecycleTestSource = `import assert from "node:assert/strict";
import { EventEmitter } from "node:events";
import { mkdir, rm, writeFile } from "node:fs/promises";
import { createServer } from "node:net";
import { join } from "node:path";
import { Writable } from "node:stream";
import { pathToFileURL } from "node:url";

const module = await import(pathToFileURL(process.argv[2]).href);
const lifecycle = module.__countershapeLifecycleTest;
assert.ok(Object.isFrozen(lifecycle));
assert.deepEqual(Object.keys(lifecycle), [
  "makeRotatingLatch",
  "waitForLatch",
  "awaitOwnerDecision",
  "awaitSpawnConfirmation",
  "childOwner",
  "waitForGroupAbsence",
  "cappedCapture",
  "pollCLIPhase",
  "pollReadinessPhase",
  "pollHTTPProbePhase",
  "beginStdinHandoff",
  "readinessCapture",
  "beginRawHTTPExchange",
  "rawHTTPExchange",
  "finalJoinComplete",
  "refreshNaturalTerminal",
  "recordFinalOutputOverflow",
  "finalizeChild",
  "runCLI",
  "startRootWatcher",
  "startRootWatcherPair",
  "closeRootWatchers",
  "setSpawnForTest",
  "resetSpawnForTest",
]);

const event = (observedAt) => ({ observedAt: BigInt(observedAt) });

{
  const latch = lifecycle.makeRotatingLatch();
  const stale = latch.snapshot();
  latch.wake();
  assert.equal(
    await lifecycle.waitForLatch(latch, stale, process.hrtime.bigint() + 1_000_000_000n),
    "state",
  );
  let polls = 0;
  let visible = false;
  const deadline = process.hrtime.bigint() + 1_000_000_000n;
  const decision = await lifecycle.awaitOwnerDecision(deadline, latch, () => {
    polls += 1;
    if (polls === 1) {
      visible = true;
      latch.wake();
      return null;
    }
    return visible ? { code: 0, kind: "exit" } : null;
  });
  assert.deepEqual(decision, { code: 0, kind: "exit" });
  assert.equal(polls, 2);

  const owner = { exit: null, lifecycleError: null, spawned: null, spawnError: null };
  const spawnLatch = lifecycle.makeRotatingLatch();
  const confirmation = lifecycle.awaitSpawnConfirmation(owner, spawnLatch, 1000);
  setImmediate(() => {
    owner.spawned = { observedAt: process.hrtime.bigint(), pid: 123 };
    spawnLatch.wake();
  });
  assert.equal(await confirmation, true);
}

{
  const latch = lifecycle.makeRotatingLatch();
  const child = new EventEmitter();
  child.pid = 12345;
  const owner = lifecycle.childOwner(child, latch);
  assert.equal(owner.spawned, null);
  child.emit("spawn");
  assert.equal(owner.spawned.pid, 12345);
  child.pid = 54321;
  assert.equal(owner.spawned.pid, 12345);
  child.emit("error", Object.assign(new Error("late child error"), { code: "EIO" }));
  assert.equal(owner.spawnError, null);
  assert.equal(owner.lifecycleError.code, "EIO");

  const failedChild = new EventEmitter();
  failedChild.pid = 22222;
  const failedLatch = lifecycle.makeRotatingLatch();
  const failedOwner = lifecycle.childOwner(failedChild, failedLatch);
  const confirmation = lifecycle.awaitSpawnConfirmation(failedOwner, failedLatch, 1000);
  setImmediate(() => failedChild.emit("error", Object.assign(new Error("missing"), { code: "ENOENT" })));
  assert.equal(await confirmation, false);
  assert.equal(failedOwner.spawned, null);
  assert.equal(failedOwner.spawnError.code, "ENOENT");
  assert.equal(failedOwner.lifecycleError, null);

  const invalidChild = new EventEmitter();
  invalidChild.pid = 33333;
  const invalidOwner = lifecycle.childOwner(invalidChild, lifecycle.makeRotatingLatch());
  invalidChild.emit("exit", 0, null);
  assert.equal(invalidOwner.spawned, null);
  assert.equal(invalidOwner.lifecycleError.code, "EXIT_BEFORE_SPAWN");
}

{
  const deadline = 100n;
  const cli = () => ({
    outputOverflow: { event: null },
    owner: { exit: null, lifecycleError: null, spawnError: null },
    stdin: { complete: true, failure: null },
  });
  let state = cli();
  state.outputOverflow.event = event(99);
  state.stdin.failure = event(99);
  state.owner.spawnError = event(99);
  state.owner.exit = { ...event(99), code: 0, kind: "exit", signal: null };
  assert.deepEqual(lifecycle.pollCLIPhase(state, 100n, deadline), { kind: "overflow" });
  state.outputOverflow.event = null;
  assert.deepEqual(lifecycle.pollCLIPhase(state, 100n, deadline), { kind: "timeout" });
  assert.deepEqual(lifecycle.pollCLIPhase(state, 99n, deadline), { kind: "stdin-failure" });
  state.stdin.failure = null;
  assert.deepEqual(lifecycle.pollCLIPhase(state, 100n, deadline), { kind: "timeout" });
  assert.deepEqual(lifecycle.pollCLIPhase(state, 99n, deadline), { kind: "error" });
  state.owner.spawnError = null;
  assert.equal(lifecycle.pollCLIPhase(state, 99n, deadline), state.owner.exit);
  state.outputOverflow.event = event(100);
  assert.deepEqual(lifecycle.pollCLIPhase(state, 100n, deadline), { kind: "timeout" });
  state.outputOverflow.event = null;
  state.stdin.complete = false;
  assert.equal(lifecycle.pollCLIPhase(state, 99n, deadline), null);
}

{
  const deadline = 100n;
  const frame = { ...event(99), frame: Buffer.from("COUNTERSHAPE_READY_V1 1\n"), kind: "frame" };
  const state = {
    outputOverflow: { event: event(99) },
    owner: { exit: { ...event(99), code: 0, signal: null }, lifecycleError: null, spawnError: event(99) },
    readiness: { decisionResult: frame, errorEvent: null },
  };
  assert.deepEqual(lifecycle.pollReadinessPhase(state, 100n, deadline), { kind: "overflow" });
  state.outputOverflow.event = null;
  assert.deepEqual(lifecycle.pollReadinessPhase(state, 100n, deadline), { kind: "timeout" });
  assert.deepEqual(lifecycle.pollReadinessPhase(state, 99n, deadline), { kind: "error" });
  state.owner.spawnError = null;
  assert.deepEqual(lifecycle.pollReadinessPhase(state, 99n, deadline), { kind: "exit" });
  state.owner.exit = null;
  assert.equal(lifecycle.pollReadinessPhase(state, 99n, deadline), frame);
  state.outputOverflow.event = event(100);
  assert.deepEqual(lifecycle.pollReadinessPhase(state, 100n, deadline), { kind: "timeout" });
}

{
  const deadline = 100n;
  const socket = {
    bytes: () => Buffer.from("response"),
    overflow: null,
    success: event(99),
    transportFailure: event(99),
  };
  const state = {
    outputOverflow: { event: event(99) },
    owner: { exit: { ...event(99), code: 1, signal: null }, lifecycleError: null, spawnError: null },
    socket,
  };
  assert.deepEqual(lifecycle.pollHTTPProbePhase(state, 100n, deadline), { kind: "overflow" });
  state.outputOverflow.event = null;
  assert.deepEqual(lifecycle.pollHTTPProbePhase(state, 100n, deadline), { kind: "timeout" });
  assert.deepEqual(lifecycle.pollHTTPProbePhase(state, 99n, deadline), { kind: "transport" });
  socket.transportFailure = null;
  state.owner.exit = null;
  assert.deepEqual(lifecycle.pollHTTPProbePhase(state, 99n, deadline), {
    kind: "success", value: Buffer.from("response"),
  });
  socket.overflow = event(100);
  assert.deepEqual(lifecycle.pollHTTPProbePhase(state, 100n, deadline), { kind: "timeout" });
}

{
  const latch = lifecycle.makeRotatingLatch();
  const received = [];
  let release;
  const sink = new Writable({
    write(chunk, _encoding, callback) {
      received.push(Buffer.from(chunk));
      release = callback;
    },
  });
  const handoff = lifecycle.beginStdinHandoff(sink, Buffer.from("contract-input"), latch);
  assert.equal(handoff.complete, false);
  assert.equal(handoff.activated, false);
  assert.equal(sink.writableLength, 0);
  assert.equal(handoff.activate(), true);
  assert.ok(sink.writableLength > 0);
  release();
  await handoff.done;
  assert.equal(handoff.failure, null);
  assert.equal(handoff.complete, true);
  assert.equal(sink.writableFinished, true);
  assert.equal(sink.writableLength, 0);
  assert.deepEqual(Buffer.concat(received), Buffer.from("contract-input"));

  const empty = lifecycle.beginStdinHandoff(new Writable({
    write(_chunk, _encoding, callback) { callback(); },
  }), Buffer.alloc(0), lifecycle.makeRotatingLatch());
  assert.equal(empty.activate(), true);
  await empty.done;
  assert.equal(empty.present, true);
  assert.equal(empty.complete, true);

  const failed = lifecycle.beginStdinHandoff(new Writable({
    write(_chunk, _encoding, callback) {
      callback(Object.assign(new Error("closed"), { code: "EPIPE" }));
    },
  }), Buffer.from("x"), lifecycle.makeRotatingLatch());
  assert.equal(failed.activate(), true);
  await failed.done;
  assert.equal(failed.complete, false);
  assert.equal(failed.failure.reason, "TRANSPORT_FAILED");

  class ControlledStdin extends EventEmitter {
    constructor() {
      super();
      this.destroyed = false;
      this.writableFinished = false;
      this.writableLength = 1;
    }
    destroy() { this.destroyed = true; }
    end(bytes) { this.bytes = Buffer.from(bytes); }
    finish() {
      this.writableFinished = true;
      this.writableLength = 0;
      this.emit("finish");
    }
    close() { this.emit("close"); }
  }
  const preactivation = new ControlledStdin();
  const preactivationHandoff = lifecycle.beginStdinHandoff(
    preactivation, Buffer.from("must-not-write"), lifecycle.makeRotatingLatch(),
  );
  preactivation.emit("error", new Error("spawn failed pipe"));
  preactivation.close();
  await preactivationHandoff.done;
  assert.equal(preactivationHandoff.activated, false);
  assert.equal(preactivationHandoff.failure, null);
  assert.equal(preactivation.bytes, undefined);

  const controlled = new ControlledStdin();
  const controlledHandoff = lifecycle.beginStdinHandoff(
    controlled, Buffer.from("controlled"), lifecycle.makeRotatingLatch(),
  );
  assert.equal(controlled.bytes, undefined);
  assert.equal(controlledHandoff.activate(), true);
  assert.deepEqual(controlled.bytes, Buffer.from("controlled"));
  controlled.finish();
  assert.equal(controlledHandoff.complete, true);
  assert.equal(controlledHandoff.settled, false);
  controlled.emit("error", new Error("late EPIPE"));
  assert.equal(controlledHandoff.failure.reason, "TRANSPORT_FAILED");
  assert.equal(controlled.destroyed, true);
  assert.equal(controlledHandoff.settled, false);
  controlled.close();
  await controlledHandoff.done;
  assert.equal(controlledHandoff.settled, true);
  assert.notEqual(controlledHandoff.settledAt, null);
}

{
  class SyntheticReadable extends EventEmitter {
    constructor() {
      super();
      this.closed = false;
    }
    close() {
      if (this.closed) return;
      this.closed = true;
      this.emit("close");
    }
    destroy() { this.close(); }
  }
  class SyntheticStdin extends SyntheticReadable {
    constructor() {
      super();
      this.endCalls = 0;
      this.writableFinished = false;
      this.writableLength = 0;
    }
    end() { this.endCalls += 1; }
  }
  const stdout = new SyntheticReadable();
  const stderr = new SyntheticReadable();
  const stdin = new SyntheticStdin();
  const child = new EventEmitter();
  child.pid = 44444;
  child.stdout = stdout;
  child.stderr = stderr;
  child.stdin = stdin;
  child.unref = () => {};
  lifecycle.setSpawnForTest(() => {
    setImmediate(() => {
      child.emit("error", Object.assign(new Error("async spawn failure"), { code: "ENOENT" }));
      for (const stream of [stdout, stderr, stdin]) {
        stream.emit("error", new Error("pre-spawn pipe failure"));
        stream.close();
      }
    });
    return child;
  });
  try {
    const facts = { ineligible_reasons: [], internal_failure: false, predicate_match: false, tamper: false };
    await lifecycle.runCLI({
      runtime: { argv: [], stdin: Buffer.from("must-not-write"), stdinPresent: true },
      source: { limits: { probe_ms: 100, stderr_bytes: 16, stdout_bytes: 16, teardown_ms: 100 } },
    }, { candidate: process.cwd() }, Object.create(null), facts);
    assert.equal(stdin.endCalls, 0);
    assert.deepEqual(facts.ineligible_reasons, ["START_FAILED"]);
    assert.equal(facts.internal_failure, false);
  } finally {
    lifecycle.resetSpawnForTest();
  }
}

{
  const subject = join(process.cwd(), "absent-stdin-subject.mjs");
  await writeFile(subject, 'process.stdout.write("ok\\n");\n');
  const facts = { ineligible_reasons: [], internal_failure: false, predicate_match: false, tamper: false };
  await lifecycle.runCLI({
    runtime: { argv: [subject], stdin: Buffer.alloc(0), stdinPresent: false },
    source: { limits: { probe_ms: 1000, stderr_bytes: 16 << 10, stdout_bytes: 32 << 10, teardown_ms: 800 } },
    profileFields: [{ field_id: "cli.stdout.bytes" }],
    predicate: {
      selected: ["cli.stdout.bytes"],
      allowed: [Buffer.from('{"fields":[{"field_id":"cli.stdout.bytes","value":{"base64":"b2sK","tag":"BYTES"}}]}')],
    },
  }, { candidate: process.cwd() }, Object.create(null), facts);
  assert.deepEqual(facts, {
    ineligible_reasons: [], internal_failure: false, predicate_match: true, tamper: false,
  });
  await rm(subject);
}

{
  class ControlledReadable extends EventEmitter {
    constructor() {
      super();
      this.destroyed = false;
    }
    destroy() { this.destroyed = true; }
  }

  const cleanStream = new ControlledReadable();
  const cleanCapture = lifecycle.cappedCapture(
    cleanStream, 16, lifecycle.makeRotatingLatch(), () => assert.fail("unexpected overflow"),
  );
  cleanCapture.activate();
  cleanStream.emit("data", Buffer.from("clean"));
  cleanStream.emit("end");
  assert.notEqual(cleanCapture.endEvent, null);
  assert.equal(cleanCapture.settled, false);
  cleanStream.emit("close");
  await cleanCapture.done;
  assert.equal(cleanCapture.settled, true);
  assert.equal(cleanCapture.error, false);
  assert.deepEqual(cleanCapture.bytes(), Buffer.from("clean"));

  const lateErrorStream = new ControlledReadable();
  const lateErrorCapture = lifecycle.cappedCapture(
    lateErrorStream, 16, lifecycle.makeRotatingLatch(), () => assert.fail("unexpected overflow"),
  );
  lateErrorCapture.activate();
  lateErrorStream.emit("data", Buffer.from("retained"));
  lateErrorStream.emit("end");
  assert.equal(lateErrorCapture.settled, false);
  lateErrorStream.emit("error", new Error("late capture failure"));
  assert.equal(lateErrorCapture.error, true);
  assert.notEqual(lateErrorCapture.errorEvent, null);
  assert.equal(lateErrorCapture.settled, false);
  lateErrorStream.emit("close");
  await lateErrorCapture.done;
  assert.deepEqual(lateErrorCapture.bytes(), Buffer.from("retained"));

  const readyStream = new ControlledReadable();
  const readiness = lifecycle.readinessCapture(readyStream, 32, lifecycle.makeRotatingLatch());
  readiness.activate();
  const readyFrame = Buffer.from("COUNTERSHAPE_READY_V1 1\n");
  readyStream.emit("data", readyFrame);
  readyStream.emit("end");
  assert.equal(readiness.ended, true);
  assert.equal(readiness.settled, false);
  assert.equal(readiness.decisionResult.kind, "frame");
  readyStream.emit("error", new Error("late readiness failure"));
  assert.equal(readiness.error, true);
  assert.deepEqual(lifecycle.pollReadinessPhase({
    outputOverflow: { event: null },
    owner: { exit: null, lifecycleError: null, spawnError: null },
    readiness,
  }, process.hrtime.bigint(), process.hrtime.bigint() + 1_000_000_000n), { kind: "failure" });
  assert.equal(readiness.settled, false);
  readyStream.emit("close");
  await readiness.done;
  assert.equal(readiness.settled, true);

  const incompleteStream = new ControlledReadable();
  const incomplete = lifecycle.readinessCapture(incompleteStream, 32, lifecycle.makeRotatingLatch());
  incomplete.activate();
  incompleteStream.emit("close");
  await incomplete.done;
  assert.equal(incomplete.error, true);
  assert.equal(incomplete.decisionResult.kind, "failure");
}

{
  const owner = { exit: null, spawnError: null, spawned: { pid: 123 } };
  const resources = [{ settled: true }, { settled: true }];
  assert.equal(lifecycle.finalJoinComplete(owner, resources), false);
  owner.exit = { code: 0, signal: null, signalCountAtObservation: 0 };
  resources[1].settled = false;
  assert.equal(lifecycle.finalJoinComplete(owner, resources), false);
  resources[1].settled = true;
  assert.equal(lifecycle.finalJoinComplete(owner, resources), true);
  owner.exit.observedAt = 99n;
  resources[0].settledAt = 99n;
  resources[1].settledAt = 99n;
  assert.equal(lifecycle.finalJoinComplete(owner, resources, 100n), true);
  resources[1].settledAt = 100n;
  assert.equal(lifecycle.finalJoinComplete(owner, resources, 100n), false);
  resources[1].settledAt = 99n;
  owner.exit.observedAt = 100n;
  assert.equal(lifecycle.finalJoinComplete(owner, resources, 100n), false);
  owner.exit.observedAt = 99n;
  assert.equal(lifecycle.refreshNaturalTerminal(owner), owner.exit);
  assert.equal(lifecycle.refreshNaturalTerminal(owner, 100n), owner.exit);
  owner.exit.observedAt = 100n;
  assert.equal(lifecycle.refreshNaturalTerminal(owner, 100n), null);
  owner.exit.observedAt = 99n;
  owner.exit.signalCountAtObservation = 1;
  assert.equal(lifecycle.refreshNaturalTerminal(owner), null);

  const failedSpawn = { exit: null, spawnError: event(1), spawned: null };
  assert.equal(lifecycle.finalJoinComplete(failedSpawn, resources), true);
}

{
  const facts = () => ({ ineligible_reasons: [], internal_failure: false });
  const settledAt = process.hrtime.bigint();
  const failedOwner = {
    child: { unref() {} }, exit: null, lifecycleError: null,
    spawned: null, spawnError: { code: "ENOENT", observedAt: settledAt },
  };
  const preSpawnCapture = {
    activated: false, destroy() {}, destroyError: false, error: true, role: "capture",
    settled: true, settledAt,
  };
  const preSpawnFacts = facts();
  await lifecycle.finalizeChild(
    failedOwner, [preSpawnCapture], 100, preSpawnFacts, false, lifecycle.makeRotatingLatch(),
  );
  assert.deepEqual(preSpawnFacts.ineligible_reasons, []);
  assert.equal(preSpawnFacts.internal_failure, false);

  const readinessOwner = {
    child: { unref() {} }, exit: null, lifecycleError: null,
    spawned: null, spawnError: { code: "ENOENT", observedAt: process.hrtime.bigint() },
  };
  const lateReadiness = {
    activated: true, destroy() {}, destroyError: false, error: true, role: "readiness",
    settled: true, settledAt: process.hrtime.bigint(),
  };
  const readinessFacts = facts();
  await lifecycle.finalizeChild(
    readinessOwner, [lateReadiness], 100, readinessFacts, false, lifecycle.makeRotatingLatch(),
  );
  assert.deepEqual(readinessFacts.ineligible_reasons, ["READINESS_FAILED"]);

  const broadFacts = facts();
  await lifecycle.finalizeChild(
    { child: { unref() {} }, exit: null, lifecycleError: null, spawned: null, spawnError: null },
    [], 100, broadFacts, false,
    { snapshot() { throw new Error("latch invariant"); }, wake() {} },
  );
  assert.equal(broadFacts.internal_failure, true);
  assert.deepEqual(broadFacts.ineligible_reasons, ["TEARDOWN_FAILED", "ORPHAN_RISK"]);

  const unrefFacts = facts();
  await lifecycle.finalizeChild(
    {
      child: { unref() { throw new Error("unref invariant"); } },
      exit: null,
      lifecycleError: null,
      signal() { return "absent"; },
      spawned: { observedAt: process.hrtime.bigint(), pid: 999999 },
      spawnError: null,
    },
    [], 0, unrefFacts, false, lifecycle.makeRotatingLatch(),
  );
  assert.equal(unrefFacts.internal_failure, true);
  assert.deepEqual(unrefFacts.ineligible_reasons, ["TEARDOWN_FAILED", "ORPHAN_RISK"]);
}

{
  const before = { ineligible_reasons: [] };
  lifecycle.recordFinalOutputOverflow(before, event(99), 100n, false);
  assert.deepEqual(before.ineligible_reasons, ["OUTPUT_LIMIT"]);
  const at = { ineligible_reasons: [] };
  lifecycle.recordFinalOutputOverflow(at, event(100), 100n, false);
  assert.deepEqual(at.ineligible_reasons, []);
  const after = { ineligible_reasons: [] };
  lifecycle.recordFinalOutputOverflow(after, event(101), 100n, false);
  assert.deepEqual(after.ineligible_reasons, []);
  const completed = { ineligible_reasons: [] };
  lifecycle.recordFinalOutputOverflow(completed, event(100), 100n, true);
  assert.deepEqual(completed.ineligible_reasons, ["OUTPUT_LIMIT"]);
  const absent = { ineligible_reasons: [] };
  lifecycle.recordFinalOutputOverflow(absent, null, 100n, true);
  lifecycle.recordFinalOutputOverflow(absent, event(99), null, true);
  assert.deepEqual(absent.ineligible_reasons, []);
}

{
  const calls = [];
  const errors = lifecycle.closeRootWatchers(
    { closed: false, handle: { close() { calls.push("first"); throw new Error("first close"); } } },
    { closed: false, handle: { close() { calls.push("second"); } } },
  );
  assert.deepEqual(calls, ["first", "second"]);
  assert.equal(errors.length, 1);

  const expired = await lifecycle.waitForGroupAbsence(999999, process.hrtime.bigint());
  assert.deepEqual(expired, { absent: false, probeError: false });
  const missing = await lifecycle.waitForGroupAbsence(
    999999, process.hrtime.bigint() + 100_000_000n,
  );
  assert.deepEqual(missing, { absent: true, probeError: false });
}

{
  const watchRoot = join(process.cwd(), "watch-positive-control");
  await mkdir(watchRoot);
  const watcher = lifecycle.startRootWatcher(watchRoot);
  const path = join(watchRoot, "witness.txt");
  await writeFile(path, "one");
  for (let attempt = 0; attempt < 100 && watcher.events.length === 0; attempt += 1) {
    await new Promise((resolve) => setTimeout(resolve, 5));
  }
  assert.ok(watcher.events.length > 0);
  await writeFile(path, "two");
  await rm(path);
  assert.deepEqual(lifecycle.closeRootWatchers(watcher), []);
  assert.equal(watcher.closed, true);

  assert.throws(
    () => lifecycle.startRootWatcherPair(watchRoot, join(watchRoot, "missing")),
    (error) => error?.reason === "ENVIRONMENT_INVALID",
  );
}

{
  const response = Buffer.from("HTTP/1.1 200 OK\r\ncontent-length: 0\r\n\r\n", "ascii");
  const server = createServer({ allowHalfOpen: true }, (socket) => {
    let request = Buffer.alloc(0);
    socket.on("data", (chunk) => {
      request = Buffer.concat([request, Buffer.from(chunk)]);
      if (request.includes(Buffer.from("\r\n\r\n"))) socket.end(response);
    });
  });
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  try {
    const port = server.address().port;
    const state = {
      latch: lifecycle.makeRotatingLatch(),
      outputOverflow: { event: null },
      owner: { exit: null, lifecycleError: null, spawnError: null },
      socket: null,
    };
    const raw = await lifecycle.rawHTTPExchange({
      body: Buffer.alloc(0), bodyPresent: false,
      capture: { body_bytes: 16, header_bytes: 128, status_line_bytes: 64 },
      headers: [], method: "GET", path: "/", query: [],
    }, port, process.hrtime.bigint() + 1_000_000_000n, state);
    assert.deepEqual(raw, response);
    assert.equal(state.socket.settled, true);
    assert.notEqual(state.socket.requestFinished, null);
    assert.notEqual(state.socket.responseEOF, null);
    assert.equal(state.socket.success.observedAt, state.socket.closeEvent.observedAt);
  } finally {
    await new Promise((resolve) => server.close(resolve));
  }
}
`
