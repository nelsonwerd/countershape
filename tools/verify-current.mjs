#!/usr/bin/env node

import { createHash } from "node:crypto";
import { lstat, readdir, realpath } from "node:fs/promises";
import { arch, platform } from "node:os";
import { join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

import {
	acquireVerificationLock,
	admitTools,
	buildChildEnvironment,
	childResult,
	cleanupVerificationResources,
	createPrivateRoots,
	finalizeVerificationResources,
	repositoryRoot,
} from "./verify-runtime-authority.mjs";

const modulePath = "github.com/nelsonwerd/countershape";
const generalJobs = 1;
const goTestParallelism = 2;
const authorityNames = Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]);
const extendedChildTimeoutMS = 30 * 60 * 1000;
const exactExtendedChildTimeoutStepIDs = Object.freeze([
	"architecture-p07b-c-c5",
	"architecture-p07b-c-c5-selftest",
]);
export const sealedC6AArchitectureStepIDs = Object.freeze([
	"architecture-p07b-c-c6",
	"architecture-p07b-c-c6-selftest",
]);

const exactU7PhaseCapsuleKeys = Object.freeze([
	"boundary", "parent", "profile", "state", "receipt", "productAuthority", "productBehavior",
]);
export const exactU7PhaseContracts = Object.freeze({
	U7P: Object.freeze({ boundary: "U7P", parent: "C6B", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT", productAuthority: "NONE", productBehavior: "INHERITED_UNREPROVEN" }),
	U7M: Object.freeze({ boundary: "U7M", parent: "U7P", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT", productAuthority: "NONE", productBehavior: "INHERITED_UNREPROVEN" }),
	U7A: Object.freeze({ boundary: "U7A", parent: "U7M", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT", productAuthority: "U7_REFERENCE_APPLICATION", productBehavior: "CANDIDATE_UNRECEIPTED" }),
	U7B: Object.freeze({ boundary: "U7B", parent: "U7A", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT", productAuthority: "U7_REFERENCE_APPLICATION", productBehavior: "CANDIDATE_UNRECEIPTED" }),
	U7N: Object.freeze({ boundary: "U7N", parent: "U7B", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT", productAuthority: "NONE", productBehavior: "INHERITED_UNREPROVEN" }),
	U7C: Object.freeze({ boundary: "U7C", parent: "U7N", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT", productAuthority: "U7_REFERENCE_APPLICATION", productBehavior: "CANDIDATE_UNRECEIPTED" }),
	U7D: Object.freeze({ boundary: "U7D", parent: "U7C", profile: "SOURCE_FULL", state: "SOURCE_CANDIDATE", receipt: "ABSENT", productAuthority: "U7_REFERENCE_APPLICATION", productBehavior: "CANDIDATE_UNRECEIPTED" }),
});
const exactU7GovernedCounts = Object.freeze({ U7P: 0, U7M: 0, U7A: 10, U7B: 19, U7N: 19, U7C: 30, U7D: 34 });
const u7ArchitectureCaseDigest = "da362a62ba6afde90fcb2464e8d7cbb8d191145f45a18458c807179ccfaa5bb9";
const u7PhaseCapsuleStart = "<!-- U7-PHASE:START -->";
const u7PhaseCapsuleEnd = "<!-- U7-PHASE:END -->";
const u7PhaseCapsuleIndexStep = Object.freeze({
	id: "u7-phase-capsule-index", tool: "git", tools: Object.freeze(["git"]),
	args: Object.freeze(["--no-replace-objects", "show", ":docs/HANDOFF_MODE_C.md"]),
});

export function validateU7PhaseCapsule(capsule) {
	if (capsule === null || typeof capsule !== "object" || Array.isArray(capsule) ||
		JSON.stringify(Object.keys(capsule)) !== JSON.stringify(exactU7PhaseCapsuleKeys)) {
		throw new VerificationError("VERIFY_U7_PHASE_CONTRACT", "capsule shape");
	}
	const expected = exactU7PhaseContracts[capsule.boundary];
	if (expected === undefined) {
		throw new VerificationError("VERIFY_U7_PHASE_UNSUPPORTED", String(capsule.boundary));
	}
	if (JSON.stringify(capsule) !== JSON.stringify(expected)) {
		throw new VerificationError("VERIFY_U7_PHASE_CONTRACT", capsule.boundary);
	}
	return capsule.boundary;
}

export function exactU7ArchitectureOutput(phase) {
	const governed = exactU7GovernedCounts[phase];
	if (governed === undefined) throw new VerificationError("VERIFY_U7_PHASE_UNSUPPORTED", String(phase));
	return `U7_ARCHITECTURE_OK phase=${phase} governed=${governed} protected=67\n`;
}

export function exactU7ArchitectureSelftestOutput(phase) {
	if (!Object.hasOwn(exactU7PhaseContracts, phase)) {
		throw new VerificationError("VERIFY_U7_PHASE_UNSUPPORTED", String(phase));
	}
	return `U7 architecture defensive self-test passed: active=${phase} cases=100 clean=8 hostile=92 digest=${u7ArchitectureCaseDigest}\n`;
}

// Sealed C0 compatibility invariants now enforced by verify-runtime-authority.mjs:
// GOFLAGS: "-mod=readonly -buildvcs=false -p=1" and review-first VERIFY_STALE_LOCK refusal.

export class VerificationError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function extendedChildTimeoutMSByStepID() {
	return Object.freeze(Object.assign(Object.create(null), {
		"architecture-p07b-c-c5": extendedChildTimeoutMS,
		"architecture-p07b-c-c5-selftest": extendedChildTimeoutMS,
	}));
}

function extendedChildExecutionPolicy() {
	return Object.freeze(Object.assign(Object.create(null), {
		timeoutMS: extendedChildTimeoutMS,
	}));
}

export function childExecutionPolicyForStepID(stepID) {
	const timeoutMSByStepID = extendedChildTimeoutMSByStepID();
	const executionPolicy = extendedChildExecutionPolicy();
	const keys = Object.keys(timeoutMSByStepID);
	if (!Object.isFrozen(timeoutMSByStepID) ||
		Object.getPrototypeOf(timeoutMSByStepID) !== null ||
		JSON.stringify(keys) !== JSON.stringify(exactExtendedChildTimeoutStepIDs) ||
		keys.some((id) => timeoutMSByStepID[id] !== extendedChildTimeoutMS) ||
		!Object.isFrozen(executionPolicy) ||
		Object.getPrototypeOf(executionPolicy) !== null ||
		Reflect.ownKeys(executionPolicy).length !== 1 ||
		executionPolicy.timeoutMS !== extendedChildTimeoutMS) {
		throw new VerificationError("VERIFY_CHILD_TIMEOUT_POLICY_INVALID", keys.join(","));
	}
	return typeof stepID === "string" && Object.hasOwn(timeoutMSByStepID, stepID)
		? executionPolicy
		: undefined;
}

export async function currentStepChildResult(step, admitted, childEnvironment, dependencies = {}) {
	const executionPolicy = childExecutionPolicyForStepID(step?.id);
	return executionPolicy === undefined
		? childResult(step, admitted, childEnvironment, dependencies)
		: childResult(step, admitted, childEnvironment, dependencies, executionPolicy);
}

function u7OccurrenceCount(text, value) {
	return text.split(value).length - 1;
}

function u7VisibleMarkdownAuthorityBody(body) {
	const visible = [];
	let htmlComment = false;
	let fence;
	for (const originalLine of body.split("\n")) {
		if (fence !== undefined) {
			visible.push("");
			const close = new RegExp(`^ {0,3}${fence.character === "`" ? "`" : "~"}{${fence.length},}\\s*$`, "u");
			if (close.test(originalLine)) fence = undefined;
			continue;
		}
		let line = "";
		let cursor = 0;
		while (cursor < originalLine.length) {
			if (htmlComment) {
				const end = originalLine.indexOf("-->", cursor);
				if (end === -1) { cursor = originalLine.length; continue; }
				htmlComment = false;
				cursor = end + 3;
				continue;
			}
			const start = originalLine.indexOf("<!--", cursor);
			if (start === -1) { line += originalLine.slice(cursor); cursor = originalLine.length; continue; }
			line += originalLine.slice(cursor, start);
			htmlComment = true;
			cursor = start + 4;
		}
		if (/^(?: {4}|\t)/u.test(line)) { visible.push(""); continue; }
		if (/^ {0,3}<(?:\/?[A-Za-z][A-Za-z0-9-]*(?:\s|\/?>)|\?|![A-Z]|!\[CDATA\[)/u.test(line)) {
			throw new VerificationError("VERIFY_U7_PHASE_MARKDOWN_RAW_HTML", line.slice(0, 128));
		}
		const opening = /^ {0,3}(`{3,}|~{3,})/u.exec(line);
		if (opening) { fence = { character: opening[1][0], length: opening[1].length }; visible.push(""); continue; }
		visible.push(line);
	}
	if (htmlComment) throw new VerificationError("VERIFY_U7_PHASE_MARKDOWN_HTML_COMMENT", "unterminated");
	if (fence !== undefined) throw new VerificationError("VERIFY_U7_PHASE_MARKDOWN_FENCE", "unterminated");
	return visible.join("\n");
}

export function parseU7PhaseCapsule(text) {
	if (typeof text !== "string" || u7OccurrenceCount(text, u7PhaseCapsuleStart) !== 1 ||
		u7OccurrenceCount(text, u7PhaseCapsuleEnd) !== 1) {
		throw new VerificationError("VERIFY_U7_PHASE_CAPSULE", "one start and one end marker required");
	}
	const start = text.indexOf(u7PhaseCapsuleStart);
	const end = text.indexOf(u7PhaseCapsuleEnd);
	if (start < 0 || end <= start || (start !== 0 && text[start - 1] !== "\n") ||
		text[start + u7PhaseCapsuleStart.length] !== "\n" || text[end - 1] !== "\n" ||
		!["", "\n"].includes(text.slice(end + u7PhaseCapsuleEnd.length, end + u7PhaseCapsuleEnd.length + 1))) {
		throw new VerificationError("VERIFY_U7_PHASE_CAPSULE", "marker order or line shape");
	}
	const heading = "## Current state\n";
	if (u7OccurrenceCount(text, heading) !== 1) throw new VerificationError("VERIFY_U7_PHASE_CAPSULE", "Current state heading");
	const sectionStart = text.indexOf(heading) + heading.length;
	const next = text.indexOf("\n## ", sectionStart);
	const sectionEnd = next === -1 ? text.length : next + 1;
	if (start < sectionStart || end >= sectionEnd || !/^\s*$/u.test(text.slice(sectionStart, start))) {
		throw new VerificationError("VERIFY_U7_PHASE_CAPSULE", "first authority in Current state");
	}
	const startSentinel = "COUNTERSHAPE_VISIBLE_U7_PHASE_START";
	const endSentinel = "COUNTERSHAPE_VISIBLE_U7_PHASE_END";
	if (text.includes(startSentinel) || text.includes(endSentinel)) throw new VerificationError("VERIFY_U7_PHASE_CAPSULE", "reserved sentinel");
	const visible = u7VisibleMarkdownAuthorityBody(text.replace(u7PhaseCapsuleStart, startSentinel).replace(u7PhaseCapsuleEnd, endSentinel));
	if (u7OccurrenceCount(visible, startSentinel) !== 1 || u7OccurrenceCount(visible, endSentinel) !== 1 ||
		visible.indexOf(startSentinel) >= visible.indexOf(endSentinel)) {
		throw new VerificationError("VERIFY_U7_PHASE_CAPSULE", "markers must be visible operational Markdown");
	}
	const body = text.slice(start, end + u7PhaseCapsuleEnd.length);
	const match = /^<!-- U7-PHASE:START -->\n### Active U7 phase contract\n\n- \*\*Namespace:\*\* `countershape\/u7-unit-paths\/v3`\n- \*\*Boundary:\*\* `([A-Z0-9]+)`\n- \*\*Parent:\*\* `([A-Z0-9.]+)`\n- \*\*Verification profile:\*\* `(SOURCE_FULL|RECEIPT_RECONCILIATION)`\n- \*\*State:\*\* `(SOURCE_CANDIDATE|RECEIPT_CANDIDATE)`\n- \*\*Topology:\*\* `U7P -> U7M -> U7A -> U7B -> U7N -> U7C -> U7D -> U7R`\n- \*\*Inherited receipts:\*\* `C3P=PRESENT; C3=PRESENT; C6A=PRESENT`\n- \*\*Receipt U7D:\*\* `(ABSENT|PRESENT)`\n- \*\*Product authority:\*\* `(NONE|U7_REFERENCE_APPLICATION)`\n- \*\*Product behavior:\*\* `(INHERITED_UNREPROVEN|CANDIDATE_UNRECEIPTED|SOURCE_RECEIPT_RECONCILIATION)`\n<!-- U7-PHASE:END -->$/u.exec(body);
	if (!match) throw new VerificationError("VERIFY_U7_PHASE_CAPSULE", "exact capsule shape");
	return Object.freeze({ boundary: match[1], parent: match[2], profile: match[3], state: match[4], receipt: match[5], productAuthority: match[6], productBehavior: match[7] });
}

export async function loadU7PhaseCapsuleFromIndex(state, runner = currentStepChildResult) {
	const result = await runner(u7PhaseCapsuleIndexStep, state.admitted, state.childEnvironment);
	if (result.error || result.signal || result.status !== 0 || result.stderr !== "") {
		const outcome = result.error?.message ?? (result.signal ? `signal=${result.signal}` : `exit=${result.status}`);
		throw new VerificationError("VERIFY_U7_PHASE_SOURCE", `${outcome}: stderr=${JSON.stringify(result.stderr)}`);
	}
	return parseU7PhaseCapsule(result.stdout);
}

const goGeneralCommon = Object.freeze(["-mod=readonly", "-buildvcs=false", `-p=${generalJobs}`]);
const goSerialCommon = Object.freeze(["-mod=readonly", "-buildvcs=false", "-p=1"]);
const inheritedP07CompatibilityStdout = "U7 inherited P07 compatibility exact: parent=C6B blocks=11 protected_paths=4 c3p_plan=FULL_EXPLICIT_C6A_AUTHORITY legacy_live_entrypoints=RETIRED_SUCCESSOR_INCOMPATIBLE\n";

export const sensitiveGoPackages = Object.freeze([
	`${modulePath}/internal/contractexec/runner`,
	`${modulePath}/internal/emit/node/compiler`,
	`${modulePath}/internal/emit/node/program/v1`,
	`${modulePath}/internal/processmechanics`,
	`${modulePath}/internal/store`,
	`${modulePath}/internal/world`,
	`${modulePath}/testkit/contractexec/cli`,
	`${modulePath}/testkit/studies/cli_precedence`,
	`${modulePath}/testkit/studies/http_invoices`,
]);

export const c5SensitiveGoPackages = Object.freeze([
	`${modulePath}/internal/contractexec/http`,
	`${modulePath}/internal/contractexec/scope`,
	`${modulePath}/testkit/contractexec/http`,
]);

export const currentSteps = Object.freeze([
	Object.freeze({ id: "workspace-no-ds-store", kind: "guard" }),
	Object.freeze({
		id: "architecture-u7", kind: "u7-phase-architecture", tool: "node", tools: Object.freeze(["node"]),
		path: "tools/check-u7-architecture.mjs", marker: "U7_ARCHITECTURE_OK phase=",
	}),
	Object.freeze({
		id: "go-package-partition", kind: "package-guard", tool: "go", tools: Object.freeze(["go"]),
		args: Object.freeze(["list", "-mod=readonly", "-buildvcs=false", "./..."]), marker: "PACKAGE_PARTITION exact",
	}),
	Object.freeze({ id: "go-build", tool: "go", tools: Object.freeze(["go", "cc", "cxx"]), args: Object.freeze(["build", goGeneralCommon[0], goGeneralCommon[1], goGeneralCommon[2], "./..."]) }),
	Object.freeze({ id: "go-vet", tool: "go", tools: Object.freeze(["go", "cc", "cxx"]), args: Object.freeze(["vet", goGeneralCommon[0], goGeneralCommon[1], goGeneralCommon[2], "./..."]) }),
	Object.freeze({
		id: "go-test-general", tool: "go", tools: Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]), packageClass: "general",
		args: Object.freeze(["test", goGeneralCommon[0], goGeneralCommon[1], goGeneralCommon[2], `-parallel=${goTestParallelism}`, "-count=1", "-timeout=20m"]),
	}),
	Object.freeze({
		id: "go-test-sensitive-serial", tool: "go", tools: Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]), packageClass: "sensitive",
		args: Object.freeze(["test", goSerialCommon[0], goSerialCommon[1], goSerialCommon[2], `-parallel=${goTestParallelism}`, "-count=1", "-timeout=20m"]),
	}),
	Object.freeze({
		id: "go-test-sensitive-c5-serial", tool: "go", tools: Object.freeze(["go", "node", "git", "sh", "cc", "cxx"]), packageClass: "c5Sensitive",
		args: Object.freeze(["test", goSerialCommon[0], goSerialCommon[1], goSerialCommon[2], `-parallel=${goTestParallelism}`, "-count=1", "-timeout=20m"]),
	}),
	Object.freeze({
		id: "go-package-partition-revalidation", kind: "package-revalidation", tool: "go", tools: Object.freeze(["go"]),
		args: Object.freeze(["list", "-mod=readonly", "-buildvcs=false", "./..."]), marker: "PACKAGE_PARTITION_REVALIDATED exact",
	}),
	Object.freeze({ id: "verification-runner-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/verify-current-selftest.mjs", marker: "verification runner self-test passed:" }),
	Object.freeze({
		id: "go-repetition-runner-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/verify-go-test-repetition.mjs",
		args: Object.freeze(["--self-test"]), marker: "Go repetition verifier self-test passed:",
	}),
	Object.freeze({ id: "architecture-u5", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-u5-architecture.mjs", marker: "U5 architecture boundary OK" }),
	Object.freeze({ id: "architecture-u5-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-u5-architecture-selftest.mjs", marker: "U5 architecture self-test passed:" }),
	Object.freeze({ id: "architecture-u6", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-u6-architecture.mjs", marker: "U6 architecture boundary OK" }),
	Object.freeze({ id: "architecture-u6-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-u6-architecture-selftest.mjs", marker: "U6 architecture checker self-test OK" }),
	Object.freeze({ id: "architecture-p07b-a1", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-architecture.mjs", marker: "P07B source architecture boundary OK" }),
	Object.freeze({ id: "architecture-p07b-a1-selftest", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-architecture-selftest.mjs", marker: "P07B architecture self-test OK" }),
	Object.freeze({
		id: "runtime-example-p07b-a2-2", tool: "node", tools: Object.freeze(["node", "go"]), path: "tools/generate-p07b-a2-runtime-example.mjs",
		args: Object.freeze(["--check"]), marker: "P07B A2.2 runtime ContractBundle example exact (",
	}),
	Object.freeze({
		id: "planning-example-p07b-a2-2", tool: "node", tools: Object.freeze(["node", "go"]), path: "tools/generate-p07-planning-example.mjs",
		args: Object.freeze(["--exercise"]),
		marker: "P07 planning example: real A2.2 bundle conformed and intact entrypoint refused companion tamper before harness load",
	}),
	Object.freeze({
		id: "planning-validator-selftest", tool: "node", tools: Object.freeze(["node", "go"]), path: "tools/validate-planning.mjs",
		args: Object.freeze(["--self-test"]), marker: "planning validator self-test: ok (",
	}),
	Object.freeze({
		id: "recovery-process-p07b-a2-2", tool: "node", tools: Object.freeze(["node", "go"]), path: "tools/verify-p07b-a2-recovery-process.mjs",
		marker: "P07B A2.2 fresh-process recovery OK",
	}),
	Object.freeze({
		id: "human-surface-p07b-a2-2-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/capture-p07b-a2-human-surface.mjs",
		args: Object.freeze(["--self-test"]), marker: "P07B A2.2 human-surface renderer self-test OK",
	}),
	Object.freeze({
		id: "human-surface-p07b-a2-2-check", tool: "node", tools: Object.freeze(["node"]), path: "tools/capture-p07b-a2-human-surface.mjs",
		args: Object.freeze(["--check"]), marker: "P07B A2.2 human-surface capture exact",
	}),
	Object.freeze({ id: "architecture-p07b-a2-2", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-a2-architecture.mjs", marker: "P07B A2.2 architecture boundary OK" }),
	Object.freeze({ id: "architecture-p07b-a2-2-selftest", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-a2-architecture-selftest.mjs", marker: "P07B A2.2 architecture defensive self-test OK" }),
	Object.freeze({ id: "architecture-p07b-b", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-b-architecture.mjs", marker: "P07B B architecture boundary OK" }),
	Object.freeze({ id: "architecture-p07b-b-selftest", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]), path: "tools/check-p07b-b-architecture-selftest.mjs", marker: "P07B B architecture defensive self-test OK" }),
	Object.freeze({ id: "architecture-p07b-c-c1", tool: "node", tools: Object.freeze(["node", "go"]), path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--c1"]), marker: "P07B-C C1 architecture boundary OK" }),
	Object.freeze({ id: "architecture-p07b-c-c1-selftest", tool: "node", tools: Object.freeze(["node", "go"]), path: "tools/check-p07b-c-architecture-selftest.mjs", args: Object.freeze(["--c1"]), marker: "P07B-C C1 architecture defensive self-test OK" }),
	Object.freeze({
		id: "architecture-p07b-c-c2", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--c2"]),
		marker: "P07B-C C2 cumulative architecture boundary OK",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c2-selftest", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture-selftest.mjs", args: Object.freeze(["--c2"]),
		marker: "P07B-C C2 cumulative architecture defensive self-test OK",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c2-nonhead", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c2-nonhead-persistence"]),
		marker: "P07B-C C2 Go JSON target execution OK (c2-nonhead-persistence: 6 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c2-interlock", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c2-interlock"]),
		marker: "P07B-C C2 Go JSON target execution OK (c2-interlock: 7 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c2-private-evidence", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c2-private-evidence"]),
		marker: "P07B-C C2 Go JSON target execution OK (c2-private-evidence: 6 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c2-public-surface", tool: "node", tools: Object.freeze(["node", "go", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c2-public-surface"]),
		marker: "P07B-C C2 Go JSON target execution OK (c2-public-surface: 2 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c3", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--c3"]),
		marker: "P07B-C C3 cumulative architecture boundary OK",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c3-selftest", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture-selftest.mjs", args: Object.freeze(["--c3"]),
		marker: "P07B-C C3 cumulative architecture defensive self-test OK (12 metadata cases; 35 Go JSON parser cases; 58 raw predecessor/parser cases)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c3-official-target", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c3-official-target"]),
		marker: "P07B-C C3 Go JSON target execution OK (c3-official-target: 10 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c3-single-target", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c3-single-target"]),
		marker: "P07B-C C3 Go JSON target execution OK (c3-single-target: 7 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c3-hostepoch", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c3-hostepoch"]),
		marker: "P07B-C C3 Go JSON target execution OK (c3-hostepoch: 8 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c3-noderuntime", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c3-noderuntime"]),
		marker: "P07B-C C3 Go JSON target execution OK (c3-noderuntime: 12 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c3-store-bridge", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c3-store-bridge"]),
		marker: "P07B-C C3 Go JSON target execution OK (c3-store-bridge: 6 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c4", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--c4"]),
		marker: "P07B-C C4 cumulative architecture boundary OK",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c4-selftest", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture-selftest.mjs", args: Object.freeze(["--c4"]),
		marker: "P07B-C C4 cumulative architecture defensive self-test OK (14 metadata cases; 42 Go JSON parser cases; 7 command cases; 5 owner-reference parser cases)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c4-processmechanics-parity", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c4-processmechanics-parity"]),
		marker: "P07B-C C4 Go JSON target execution OK (c4-processmechanics-parity: 12 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c4-admission-permit", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c4-admission-permit"]),
		marker: "P07B-C C4 Go JSON target execution OK (c4-admission-permit: 7 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c4-cli-closure", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c4-cli-closure"]),
		marker: "P07B-C C4 Go JSON target execution OK (c4-cli-closure: 4 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c4-finalized-run-release", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c4-finalized-run-release"]),
		marker: "P07B-C C4 Go JSON target execution OK (c4-finalized-run-release: 5 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c4-classification-recovery", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c4-classification-recovery"]),
		marker: "P07B-C C4 Go JSON target execution OK (c4-classification-recovery: 2 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c4-authority-race", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c4-authority-race"]),
		marker: "P07B-C C4 Go JSON target execution OK (c4-authority-race: 4 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c5", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--c5"]),
		marker: "P07B-C C5 cumulative architecture boundary OK",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c5-selftest", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture-selftest.mjs", args: Object.freeze(["--c5"]),
		marker: "P07B-C C5 cumulative architecture defensive self-test OK (14 metadata cases; 60 Go JSON parser cases; 6 command cases)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c5-http-behavior", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c5-http-behavior"]),
		marker: "P07B-C C5 Go JSON target execution OK (c5-http-behavior: 4 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c5-readiness-teardown", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c5-readiness-teardown"]),
		marker: "P07B-C C5 Go JSON target execution OK (c5-readiness-teardown: 9 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c5-scope-closure", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c5-scope-closure"]),
		marker: "P07B-C C5 Go JSON target execution OK (c5-scope-closure: 9 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c5-cross-profile-parity", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c5-cross-profile-parity"]),
		marker: "P07B-C C5 Go JSON target execution OK (c5-cross-profile-parity: 22 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "go-json-p07b-c-c5-http-authority-race", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-p07b-c-architecture.mjs", args: Object.freeze(["--run-go-json", "c5-http-authority-race"]),
		marker: "P07B-C C5 Go JSON target execution OK (c5-http-authority-race: 5 passed, 0 skipped)",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c6", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-sealed-c6a-architecture.mjs", args: Object.freeze(["--c6"]), rootAuthority: "SEALED_C6A",
		marker: "P07B-C C6 cumulative architecture boundary OK",
	}),
	Object.freeze({
		id: "architecture-p07b-c-c6-selftest", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-sealed-c6a-architecture.mjs", args: Object.freeze(["--c6-selftest"]), rootAuthority: "SEALED_C6A",
		marker: "P07B-C C6 cumulative architecture defensive self-test OK (11 metadata cases; 22 Go JSON lifecycle cases)",
	}),
	Object.freeze({
		id: "architecture-p07b-c-inherited-compatibility", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]), path: "tools/check-u7-plan.mjs",
		args: Object.freeze(["--verify-inherited-p07-compatibility"]), marker: "U7 inherited P07 compatibility exact:", exactStdout: inheritedP07CompatibilityStdout,
	}),
	Object.freeze({
		id: "architecture-p07b-c-unit-scope-selftest", tool: "node", tools: Object.freeze(["node"]), path: "tools/check-p07b-c-unit-scope.mjs",
		args: Object.freeze(["--self-test"]), marker: "P07B-C unit scope self-test passed:",
	}),
	Object.freeze({
		id: "architecture-u7-plan-selftest", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-u7-plan.mjs", args: Object.freeze(["--self-test"]), marker: "U7 plan contract defensive self-test passed:",
	}),
	Object.freeze({
		id: "architecture-u7-scope-selftest", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-u7-scope.mjs", args: Object.freeze(["--self-test"]), marker: "U7 independent staged-scope defensive self-test passed:",
	}),
	Object.freeze({
		id: "architecture-u7-study-harness-protocol", tool: "node", tools: Object.freeze(["node", "go", "git"]),
		path: "tools/check-u7-study-harness.mjs", args: Object.freeze(["--protocol-check"]),
		marker: "U7_STUDY_HARNESS_PROTOCOL_OK protocol=countershape/u7-study-harness/v2 digest=",
	}),
	Object.freeze({
		id: "architecture-u7-study-harness-selftest", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/check-u7-study-harness-selftest.mjs", args: Object.freeze(["--phase", "U7N"]),
		marker: "U7 study harness defensive self-test passed: active=U7N cases=",
	}),
	Object.freeze({
		id: "u7-final-runbook-selftest", tool: "node", tools: Object.freeze(["node", "go", "git", "sh", "cc", "cxx"]),
		path: "tools/print-u7-final-runbook.mjs", args: Object.freeze(["--self-test"]),
		marker: "U7 final runbook renderer defensive self-test passed:",
	}),
	Object.freeze({
		id: "architecture-u7-selftest", kind: "u7-phase-architecture-selftest", tool: "node", tools: Object.freeze(["node"]),
		path: "tools/check-u7-architecture-selftest.mjs", marker: "U7 architecture defensive self-test passed: active=",
	}),
	Object.freeze({ id: "workspace-no-ds-store-terminal", kind: "guard" }),
	Object.freeze({ id: "authority-revalidation", kind: "authority-guard" }),
	Object.freeze({ id: "verification-resource-finalization", kind: "finalization-guard" }),
]);

export const historicalOnly = Object.freeze([
	Object.freeze({ id: "architecture-u1", scripts: Object.freeze(["tools/check-u1-boundary.mjs"]), status: "docs/status/U1.md" }),
	Object.freeze({ id: "architecture-u2", scripts: Object.freeze(["tools/check-u2-boundary.mjs"]), status: "docs/status/U2.md" }),
	Object.freeze({ id: "architecture-u3", scripts: Object.freeze(["tools/check-u3-architecture.mjs"]), status: "docs/status/U3.md" }),
	Object.freeze({ id: "architecture-u4", scripts: Object.freeze(["tools/check-u4-architecture.mjs"]), status: "docs/status/U4.md" }),
	Object.freeze({ id: "mutation-u1", scripts: Object.freeze(["tools/mutate-u1.mjs", "tools/test-mutate-u1.mjs"]), status: "docs/status/U1.md" }),
	Object.freeze({ id: "mutation-u2", scripts: Object.freeze(["tools/mutate-u2.mjs", "tools/test-mutate-u2.mjs"]), status: "docs/status/U4.md" }),
	Object.freeze({ id: "mutation-u3", scripts: Object.freeze(["tools/mutate-u3.mjs"]), status: "docs/status/U4.md" }),
	Object.freeze({ id: "mutation-u4", scripts: Object.freeze(["tools/mutate-u4.mjs"]), status: "docs/status/U4.md" }),
	Object.freeze({ id: "mutation-u5", scripts: Object.freeze(["tools/mutate-u5.mjs", "tools/test-mutate-u5.mjs"]), status: "docs/status/U5.md" }),
	Object.freeze({ id: "mutation-u6-p07a-b", scripts: Object.freeze(["tools/mutate-u6.mjs"]), status: "docs/status/P07A-RULING.md" }),
	Object.freeze({ id: "mutation-p07b-a1", scripts: Object.freeze(["tools/mutate-p07b.mjs"]), status: "docs/status/P07B-A1-SOURCE.md" }),
	Object.freeze({ id: "mutation-p07b-a2-1", scripts: Object.freeze(["tools/mutate-p07b-a2-authority.mjs"]), status: "docs/status/P07B-A2-1-AUTHORITY.md" }),
	Object.freeze({ id: "architecture-p07b-c-plan-selftest", scripts: Object.freeze(["tools/check-p07b-c-plan.mjs"]), status: "docs/status/U7P-AUTHORITY.md" }),
	Object.freeze({ id: "architecture-p07b-c-c3p-receipt-selftest", scripts: Object.freeze(["tools/check-p07b-c-c3p-receipt.mjs"]), status: "docs/status/U7P-AUTHORITY.md" }),
	Object.freeze({ id: "architecture-p07b-c-c3p-receipt", scripts: Object.freeze(["tools/check-p07b-c-c3p-receipt.mjs"]), status: "docs/status/U7M-P07-RECEIPT-SUCCESSOR-COMPATIBILITY.md" }),
]);

function childToolNames(step) {
	if (!step || typeof step !== "object" || typeof step.tool !== "string") {
		throw new VerificationError("VERIFY_PLAN_PRIMARY_TOOL_REQUIRED", step?.id ?? "unnamed step");
	}
	if (!Array.isArray(step.tools) || step.tools.length === 0) {
		throw new VerificationError("VERIFY_PLAN_TOOL_ROSTER_REQUIRED", step.id ?? step.tool);
	}
	if (step.tools[0] !== step.tool) {
		throw new VerificationError("VERIFY_PLAN_PRIMARY_TOOL_MISMATCH", `${step.id ?? step.tool}: ${step.tools[0]} != ${step.tool}`);
	}
	const known = new Set(authorityNames);
	const seen = new Set();
	for (const name of step.tools) {
		if (typeof name !== "string" || !known.has(name)) {
			throw new VerificationError("VERIFY_PLAN_TOOL_UNKNOWN", `${step.id ?? step.tool}: ${String(name)}`);
		}
		if (seen.has(name)) throw new VerificationError("VERIFY_PLAN_TOOL_DUPLICATE", `${step.id ?? step.tool}: ${name}`);
		seen.add(name);
	}
	return [...step.tools];
}

export function rootAuthorityForStep(step) {
	const sealedContracts = Object.freeze(Object.assign(Object.create(null), {
		"architecture-p07b-c-c6": Object.freeze({ args: Object.freeze(["--c6"]), path: "tools/check-sealed-c6a-architecture.mjs" }),
		"architecture-p07b-c-c6-selftest": Object.freeze({ args: Object.freeze(["--c6-selftest"]), path: "tools/check-sealed-c6a-architecture.mjs" }),
	}));
	const contract = step && typeof step === "object" ? sealedContracts[step.id] : undefined;
	if (contract === undefined) {
		if (step && typeof step === "object" && Object.hasOwn(step, "rootAuthority")) {
			throw new VerificationError("VERIFY_ROOT_AUTHORITY_UNEXPECTED", `${step.id ?? "unnamed step"}:${String(step.rootAuthority)}`);
		}
		return "LIVE";
	}
	if (step.rootAuthority !== "SEALED_C6A" || step.path !== contract.path ||
		JSON.stringify(step.args) !== JSON.stringify(contract.args)) {
		throw new VerificationError("VERIFY_ROOT_AUTHORITY_CONTRACT", step.id);
	}
	return "SEALED_C6A";
}

function slash(value) {
	return value.split(sep).join("/");
}

async function requireRegularRepositoryFile(root, relativePath) {
	const absolute = resolve(root, relativePath);
	const fromRoot = relative(root, absolute);
	if (fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
		throw new VerificationError("VERIFY_PLAN_PATH_ESCAPE", relativePath);
	}
	const components = slash(relativePath).split("/");
	let cursor = root;
	for (let index = 0; index < components.length; index += 1) {
		const component = components[index];
		if (!component || component === "." || component === "..") {
			throw new VerificationError("VERIFY_PLAN_PATH_INVALID", relativePath);
		}
		cursor = join(cursor, component);
		let metadata;
		try {
			metadata = await lstat(cursor);
		} catch (error) {
			throw new VerificationError("VERIFY_PLAN_FILE_MISSING", `${relativePath}: ${error.code ?? error.message}`);
		}
		if (metadata.isSymbolicLink()) throw new VerificationError("VERIFY_PLAN_SYMLINK", relativePath);
		const final = index === components.length - 1;
		if ((!final && !metadata.isDirectory()) || (final && !metadata.isFile())) {
			throw new VerificationError("VERIFY_PLAN_FILE_NONREGULAR", relativePath);
		}
	}
}

export async function validateRepositoryPlan(root = repositoryRoot, steps = currentSteps, historical = historicalOnly) {
	const absoluteRoot = resolve(root);
	const canonicalRoot = await realpath(absoluteRoot);
	if (absoluteRoot !== canonicalRoot) throw new VerificationError("VERIFY_REPOSITORY_ROOT_NOT_CANONICAL", `${absoluteRoot} != ${canonicalRoot}`);
	const paths = new Set(["tools/verify-current.mjs", "tools/verify-runtime-authority.mjs"]);
	for (const step of steps) {
		rootAuthorityForStep(step);
		if (step.tool) childToolNames(step);
		else if (Object.hasOwn(step, "tools")) throw new VerificationError("VERIFY_PLAN_PRIMARY_TOOL_REQUIRED", step.id ?? "unnamed step");
		if (step.path) paths.add(step.path);
	}
	for (const row of historical) {
		for (const path of row.scripts) paths.add(path);
		paths.add(row.status);
	}
	for (const path of [...paths].sort()) await requireRegularRepositoryFile(canonicalRoot, path);
}

const ignoredArtifactRoots = new Set([".git", ".didrun", ".didrun-history", ".countershape", "node_modules"]);
const artifactScanLimits = Object.freeze({
	maxDepth: 128,
	maxEntries: 250_000,
	maxRelativePathBytes: 4096,
});

export async function assertNoDSStore(root = repositoryRoot, limits = artifactScanLimits) {
	for (const [name, minimum] of [["maxDepth", 0], ["maxEntries", 1], ["maxRelativePathBytes", 1]]) {
		if (!Number.isSafeInteger(limits?.[name]) || limits[name] < minimum) {
			throw new VerificationError("VERIFY_FINDER_SCAN_LIMIT_INVALID", name);
		}
	}
	const findings = [];
	let inspectedEntries = 0;
	async function walk(directory, relativeDirectory, depth) {
		if (depth > limits.maxDepth) {
			throw new VerificationError("VERIFY_FINDER_SCAN_DEPTH_LIMIT", slash(relativeDirectory));
		}
		const entries = await readdir(directory, { withFileTypes: true });
		entries.sort((left, right) => left.name.localeCompare(right.name, "en"));
		for (const entry of entries) {
			const relativePath = relativeDirectory ? `${relativeDirectory}/${entry.name}` : entry.name;
			inspectedEntries += 1;
			if (inspectedEntries > limits.maxEntries) {
				throw new VerificationError("VERIFY_FINDER_SCAN_ENTRY_LIMIT", String(inspectedEntries));
			}
			if (Buffer.byteLength(relativePath, "utf8") > limits.maxRelativePathBytes) {
				throw new VerificationError("VERIFY_FINDER_SCAN_PATH_LIMIT", slash(relativePath));
			}
			const status = await lstat(join(directory, entry.name));
			if (entry.name === ".DS_Store") findings.push(relativePath);
			if (status.isDirectory() && !(relativeDirectory === "" && ignoredArtifactRoots.has(entry.name))) {
				await walk(join(directory, entry.name), relativePath, depth + 1);
			}
		}
	}
	await walk(root, "", 0);
	if (findings.length > 0) {
		throw new VerificationError("VERIFY_FINDER_ARTIFACT_PRESENT", findings.map(slash).join(","));
	}
}

export function childArguments(step, root = repositoryRoot) {
	const suffix = [...(step.args ?? [])];
	return step.path ? [resolve(root, step.path), ...suffix] : suffix;
}

export function partitionGoPackages(
	stdout,
	sensitive = sensitiveGoPackages,
	c5Sensitive = c5SensitiveGoPackages,
) {
	if (typeof stdout !== "string" || !stdout.endsWith("\n") || stdout.includes("\r") || stdout.includes("\0")) {
		throw new VerificationError("VERIFY_PACKAGE_LIST_INVALID", "go list framing");
	}
	const packages = stdout.slice(0, -1).split("\n");
	if (packages.length === 0 || packages.some((path) => path.length === 0 ||
		(path !== modulePath && !path.startsWith(`${modulePath}/`)) || path.includes("//") || path.split("/").includes("..")) ||
		new Set(packages).size !== packages.length) {
		throw new VerificationError("VERIFY_PACKAGE_LIST_INVALID", "go list package roster");
	}
	const sortedPackages = [...packages].sort();
	if (!Array.isArray(sensitive)) {
		throw new VerificationError("VERIFY_SENSITIVE_PACKAGE_ROSTER_INVALID", String(sensitive));
	}
	if (!Array.isArray(c5Sensitive)) {
		throw new VerificationError("VERIFY_C5_SENSITIVE_PACKAGE_ROSTER_INVALID", String(c5Sensitive));
	}
	const sortedSensitive = [...sensitive].sort();
	const sortedC5Sensitive = [...c5Sensitive].sort();
	if (JSON.stringify(sortedSensitive) !== JSON.stringify(sensitive) ||
		new Set(sensitive).size !== sensitive.length ||
		sensitive.some((path) => !sortedPackages.includes(path))) {
		throw new VerificationError("VERIFY_SENSITIVE_PACKAGE_ROSTER_INVALID", sensitive.join(","));
	}
	if (JSON.stringify(sortedC5Sensitive) !== JSON.stringify(c5Sensitive) ||
		new Set(c5Sensitive).size !== c5Sensitive.length ||
		c5Sensitive.some((path) => !sortedPackages.includes(path))) {
		throw new VerificationError("VERIFY_C5_SENSITIVE_PACKAGE_ROSTER_INVALID", c5Sensitive.join(","));
	}
	const completeSensitive = [...sensitive, ...c5Sensitive];
	const sensitiveSet = new Set(completeSensitive);
	if (sensitiveSet.size !== completeSensitive.length) {
		throw new VerificationError("VERIFY_SENSITIVE_PACKAGE_GROUPS_OVERLAP", completeSensitive.join(","));
	}
	if (JSON.stringify(sensitive) !== JSON.stringify(sensitiveGoPackages)) {
		throw new VerificationError("VERIFY_SENSITIVE_PACKAGE_ROSTER_INVALID", sensitive.join(","));
	}
	if (JSON.stringify(c5Sensitive) !== JSON.stringify(c5SensitiveGoPackages)) {
		throw new VerificationError("VERIFY_C5_SENSITIVE_PACKAGE_ROSTER_INVALID", c5Sensitive.join(","));
	}
	const general = sortedPackages.filter((path) => !sensitiveSet.has(path));
	if (general.length === 0 || general.length + completeSensitive.length !== sortedPackages.length ||
		new Set([...general, ...completeSensitive]).size !== sortedPackages.length) {
		throw new VerificationError(
			"VERIFY_PACKAGE_PARTITION_INVALID",
			`general=${general.length} sensitive=${sensitive.length} c5Sensitive=${c5Sensitive.length} all=${sortedPackages.length}`,
		);
	}
	return Object.freeze({
		all: Object.freeze(sortedPackages),
		general: Object.freeze(general),
		sensitive: Object.freeze([...sensitive]),
		c5Sensitive: Object.freeze([...c5Sensitive]),
	});
}

export function packageArguments(step, partition) {
	if (!step?.packageClass) return childArguments(step);
	if (!partition || !Object.hasOwn(partition, step.packageClass) ||
		!["general", "sensitive", "c5Sensitive"].includes(step.packageClass)) {
		throw new VerificationError("VERIFY_PACKAGE_PARTITION_UNAVAILABLE", step?.id ?? "unnamed step");
	}
	const packages = partition[step.packageClass];
	if (!Array.isArray(packages) || packages.length === 0) {
		throw new VerificationError("VERIFY_PACKAGE_PARTITION_UNAVAILABLE", `${step.id}: ${step.packageClass}`);
	}
	return [...childArguments(step), ...packages];
}

export function revalidatePackagePartition(initial, current) {
	for (const name of ["all", "general", "sensitive", "c5Sensitive"]) {
		if (!initial || !current || JSON.stringify(initial[name]) !== JSON.stringify(current[name])) {
			throw new VerificationError("VERIFY_PACKAGE_PARTITION_CHANGED", name);
		}
	}
}

function writeChildFrames(write, step, stream, source) {
	if (!source) return;
	const lines = source.split(/\r?\n/u);
	if (lines.at(-1) === "") lines.pop();
	for (const line of lines) write(`CHILD ${step.id} ${stream} ${JSON.stringify(line)}\n`);
}

export function rosterDigest(steps = currentSteps, historical = historicalOnly) {
	return createHash("sha256").update(JSON.stringify({ steps, historical })).digest("hex");
}

export async function executeCurrentPlan({
	steps = currentSteps,
	historical = historicalOnly,
	admitted,
	childEnvironment,
	executor,
	write = (value) => process.stdout.write(value),
	clock = () => performance.now(),
}) {
	write(`COUNTERSHAPE_VERIFY_V1 platform=${platform()}/${arch()} general_jobs=${generalJobs} nested_jobs=1 gomaxprocs=2 roster_sha256:${rosterDigest(steps, historical)}\n`);
	for (const name of authorityNames) {
		const tool = admitted[name];
		write(`AUTHORITY ${name} path=${JSON.stringify(tool.path)} sha256:${tool.sha256}\n`);
	}
	write(`ROSTER CURRENT ${steps.map((step) => step.id).join(",")}\n`);
	const sealedRows = steps.filter((step) => rootAuthorityForStep(step) === "SEALED_C6A").map((step) => step.id);
	write(`ROSTER ROOT_AUTHORITY live=${steps.length - sealedRows.length} sealed_c6a=${sealedRows.length}:${sealedRows.join(",")}\n`);
	for (const row of historical) {
		write(`HISTORICAL-ONLY NOT-RUN ${row.id} scripts=${row.scripts.join(",")} status=${row.status}\n`);
	}
	let passed = 0;
	for (let index = 0; index < steps.length; index += 1) {
		const step = steps[index];
		const ordinal = String(index + 1).padStart(2, "0");
		const total = String(steps.length).padStart(2, "0");
		write(`CURRENT [${ordinal}/${total}] START ${step.id}\n`);
		const started = clock();
		let result;
		try {
			result = await executor(step, { admitted, childEnvironment });
		} catch (error) {
			result = { status: 1, signal: null, error, stdout: "", stderr: "" };
		}
		writeChildFrames(write, step, "STDOUT", result.stdout);
		writeChildFrames(write, step, "STDERR", result.stderr);
		if (!result.error && !result.signal && result.status === 0 && step.exactStdout !== undefined &&
			(result.stdout !== step.exactStdout || result.stderr !== "")) {
			result = { ...result, status: 1, error: new VerificationError("VERIFY_SUCCESS_OUTPUT_MISMATCH", `${step.id}: stdout=${JSON.stringify(result.stdout)} stderr=${JSON.stringify(result.stderr)}`) };
		} else if (!result.error && !result.signal && result.status === 0 && step.marker && !result.stdout.includes(step.marker)) {
			result = { ...result, status: 1, error: new VerificationError("VERIFY_SUCCESS_MARKER_MISSING", `${step.id}: ${step.marker}`) };
		}
		const duration = Math.max(0, Math.round(clock() - started));
		if (result.error || result.signal || result.status !== 0) {
			const outcome = result.error?.message ?? (result.signal ? `signal=${result.signal}` : `exit=${result.status}`);
			write(`CURRENT [${ordinal}/${total}] FAIL ${step.id} duration_ms=${duration} detail=${JSON.stringify(outcome)}\n`);
			write(`RESULT FAIL current=${passed}/${steps.length} historical_not_run=${historical.length}\n`);
			return 1;
		}
		passed += 1;
		write(`CURRENT [${ordinal}/${total}] PASS ${step.id} duration_ms=${duration}\n`);
	}
	write(`RESULT PASS current=${passed}/${steps.length} historical_not_run=${historical.length}\n`);
	return 0;
}

export async function dispatchCurrentStep(step, state) {
	if (!state || typeof state !== "object") {
		throw new VerificationError("VERIFY_EXECUTOR_STATE_REQUIRED", step?.id ?? "unnamed step");
	}
	if (step.kind === "guard") {
		await assertNoDSStore(state.finderRoot ?? repositoryRoot);
		return { status: 0, signal: null, error: null, stdout: "workspace contains no .DS_Store artifacts\n", stderr: "" };
	}
	if (step.kind === "u7-phase-architecture") {
		if (Object.hasOwn(state, "u7Phase") && state.u7Phase !== undefined) {
			throw new VerificationError("VERIFY_U7_PHASE_ALREADY_ADMITTED", String(state.u7Phase));
		}
		const capsule = await (state.loadU7PhaseCapsule ?? (() => loadU7PhaseCapsuleFromIndex(state)))();
		const phase = validateU7PhaseCapsule(capsule);
		const dynamicStep = { ...step, args: ["--phase", phase] };
		const runner = state.currentStepRunner ?? currentStepChildResult;
		const result = await runner(dynamicStep, state.admitted, state.childEnvironment);
		if (result.error || result.signal || result.status !== 0) return result;
		const expected = exactU7ArchitectureOutput(phase);
		if (result.stdout !== expected || result.stderr !== "") {
			throw new VerificationError("VERIFY_U7_ARCHITECTURE_OUTPUT", `${phase}: stdout=${JSON.stringify(result.stdout)} stderr=${JSON.stringify(result.stderr)}`);
		}
		state.u7Phase = phase;
		return result;
	}
	if (step.kind === "u7-phase-architecture-selftest") {
		if (typeof state.u7Phase !== "string") {
			throw new VerificationError("VERIFY_U7_PHASE_EARLY_GATE_MISSING", step.id);
		}
		const terminalPhase = validateU7PhaseCapsule(await (state.loadU7PhaseCapsule ?? (() => loadU7PhaseCapsuleFromIndex(state)))());
		if (terminalPhase !== state.u7Phase) {
			throw new VerificationError("VERIFY_U7_PHASE_CHANGED", `${state.u7Phase} -> ${terminalPhase}`);
		}
		const dynamicStep = { ...step, args: ["--phase", terminalPhase] };
		const runner = state.currentStepRunner ?? currentStepChildResult;
		const result = await runner(dynamicStep, state.admitted, state.childEnvironment);
		if (result.error || result.signal || result.status !== 0) return result;
		const expected = exactU7ArchitectureSelftestOutput(terminalPhase);
		if (result.stdout !== expected || result.stderr !== "") {
			throw new VerificationError("VERIFY_U7_ARCHITECTURE_SELFTEST_OUTPUT", `${terminalPhase}: stdout=${JSON.stringify(result.stdout)} stderr=${JSON.stringify(result.stderr)}`);
		}
		return result;
	}
	if (step.kind === "package-guard") {
		const result = await currentStepChildResult(step, state.admitted, state.childEnvironment);
		if (result.error || result.signal || result.status !== 0) return result;
		state.packagePartition = partitionGoPackages(result.stdout);
		return {
			...result,
			stdout: `${result.stdout}PACKAGE_PARTITION exact general=${state.packagePartition.general.length} sensitive=${state.packagePartition.sensitive.length} c5_sensitive=${state.packagePartition.c5Sensitive.length} total=${state.packagePartition.all.length}\n`,
		};
	}
	if (step.kind === "package-revalidation") {
		if (!state.packagePartition) throw new VerificationError("VERIFY_PACKAGE_PARTITION_UNAVAILABLE", step.id);
		const result = await currentStepChildResult(step, state.admitted, state.childEnvironment);
		if (result.error || result.signal || result.status !== 0) return result;
		const currentPartition = partitionGoPackages(result.stdout);
		revalidatePackagePartition(state.packagePartition, currentPartition);
		return {
			...result,
			stdout: `${result.stdout}PACKAGE_PARTITION_REVALIDATED exact general=${currentPartition.general.length} sensitive=${currentPartition.sensitive.length} c5_sensitive=${currentPartition.c5Sensitive.length} total=${currentPartition.all.length}\n`,
		};
	}
	if (step.kind === "authority-guard") {
		await admitTools(process.env, state.admitted);
		return { status: 0, signal: null, error: null, stdout: "all admitted tool authorities revalidated\n", stderr: "" };
	}
	if (step.kind === "finalization-guard") {
		await finalizeVerificationResources(state.lock, state.roots.runRoot);
		state.resourcesFinalized = true;
		return {
			status: 0, signal: null, error: null,
			stdout: `private run root removed and verifier lock released for pid ${state.lock.pid}\n`, stderr: "",
		};
	}
	if (step.packageClass) {
		return await currentStepChildResult(step, state.admitted, state.childEnvironment, {
			args: packageArguments(step, state.packagePartition),
		});
	}
	return await currentStepChildResult(step, state.admitted, state.childEnvironment);
}

async function main() {
	if (process.argv.length !== 2) {
		throw new VerificationError("VERIFY_ARGUMENTS", "no arguments are accepted");
	}
	if (platform() !== "darwin" || arch() !== "arm64") {
		throw new VerificationError("VERIFY_PLATFORM_UNSUPPORTED", `${platform()}/${arch()}; Darwin reference baseline required`);
	}
	const lock = await acquireVerificationLock();
	let primaryFailure;
	let executionState;
	let roots;
	try {
		await validateRepositoryPlan();
		const admitted = await admitTools();
		roots = await createPrivateRoots(repositoryRoot, admitted);
		const childEnvironment = buildChildEnvironment(admitted, roots);
			executionState = {
				admitted, childEnvironment, finderRoot: repositoryRoot, lock, packagePartition: undefined,
				resourcesFinalized: false, roots, u7Phase: undefined,
			};
		const status = await executeCurrentPlan({
			admitted,
			childEnvironment,
			executor: async (step) => dispatchCurrentStep(step, executionState),
		});
		process.exitCode = status;
	} catch (error) {
		primaryFailure = error;
		throw error;
	} finally {
		try {
			if (!executionState?.resourcesFinalized) await cleanupVerificationResources(lock, roots);
		} catch (cleanupFailure) {
			if (primaryFailure) throw new AggregateError([primaryFailure, cleanupFailure], "verification and cleanup both failed");
			throw cleanupFailure;
		}
	}
}

async function classifyEntry(entry) {
	if (entry === undefined) return Object.freeze({ kind: "IMPORTED" });
	const source = await realpath(fileURLToPath(import.meta.url));
	let supplied;
	try {
		supplied = resolve(entry);
		if (await realpath(supplied) !== source) return Object.freeze({ kind: "IMPORTED" });
	} catch {
		return Object.freeze({ kind: "IMPORTED" });
	}
	if (supplied === source) return Object.freeze({ kind: "CANONICAL", canonical: source, supplied });
	return Object.freeze({ kind: "NONCANONICAL", canonical: source, supplied });
}

const entry = await classifyEntry(process.argv[1]);
if (entry.kind === "NONCANONICAL") {
	throw new VerificationError(
		"VERIFY_NONCANONICAL_ENTRY",
		`${JSON.stringify(entry.supplied)} != ${JSON.stringify(entry.canonical)}`,
	);
}
if (entry.kind === "CANONICAL") {
	await main();
}
