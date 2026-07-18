#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { readFile, realpath } from "node:fs/promises";
import { dirname, isAbsolute, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

import { validateSpecification } from "./check-p07b-c-unit-scope.mjs";

export const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");

const legacyPlanningDigests = Object.freeze({
	"spec/schema/v1/contract-execution-target.schema.json": "87b53957021049510902f87fa4464ca68ceca69e177c77bf0eb128c33cc4327e",
	"spec/schema/v1/finalized-contract-run.schema.json": "9dcfe2cd7ae89037ce759a3d3761bf1cc641739accad14f4b2e5bcb658117399",
	"spec/schema/v1/contract-execution.schema.json": "f273c4385cf732b11b2138c2d27ae2a1ef5dc7ad75d8778e85a112338fe4ded7",
	"spec/examples/v1/contract-execution-target.valid.json": "accd602e97a88dc3889aa274279d1fdd17ebeeb78b044c5af206403375875544",
	"spec/examples/v1/finalized-contract-run.valid.json": "192c786b4d357eb7a93065c6e6c907ccb19efd6d52cd3811643a8c2170ba210c",
	"spec/examples/v1/contract-execution.valid.json": "c9063f424a29d3c3efb8acfb7b68c23a9d95579ec8d5383cb6362956ea404ce6",
});

const researchShape = Object.freeze({
	"research/deep-dive/p07b-c/00-scope-and-method.md": "# P07B-C deep dive — scope and method",
	"research/deep-dive/p07b-c/01-authority-model.md": "# Lane 1 — authority and data model",
	"research/deep-dive/p07b-c/02-runtime-security.md": "# Lane 2 — runtime, process, and scoped standalone evidence",
	"research/deep-dive/p07b-c/03-store-service.md": "# Lane 3 — store, service, and crash recovery",
	"research/deep-dive/p07b-c/04-verification.md": "# Lane 4 — verification architecture",
	"research/deep-dive/p07b-c/05-dx-wake-fit.md": "# Lane 5 — operator DX and Wake fit",
	"research/deep-dive/p07b-c/06-adversarial-design.md": "# Lane 6 — alternatives and adversarial interpretation",
	"research/deep-dive/p07b-c/07-synthesis.md": "# P07B-C synthesis",
	"research/deep-dive/p07b-c/08-different-model-red-team.md": "# Different-model red team",
	"research/deep-dive/p07b-c/09-follow-up-verification.md": "# Focused follow-up verification",
	"research/deep-dive/p07b-c/10-executive-briefing.md": "# P07B-C executive briefing",
});

const promptHeadings = Object.freeze([
	"# C0 — lock the corrected P07B-C authority contract",
	"# C1 — implement strict inert semantic algebra",
	"# C2 — add nonhead persistence and private interlock substrate",
	"# C3 — publish one exact pre-spawn target",
	"# C4 — close a CLI candidate-profile run and publish its classification",
	"# C5 — add HTTP and complete standalone-scope evidence",
	"# C6 — cumulative closure, expert surfaces, and receipts",
]);

const authorityDeclarationPath = "spec/verification/p07b-c-c0-authority.json";
const receiptDeclarationPath = "spec/verification/p07b-c-c0-receipt.json";
const statusPath = "docs/status/P07B-C-C0-AUTHORITY.md";
const c0ClaimLabels = Object.freeze([
	"P07B C0 authority plan coherence",
	"P07B C0 plan checker self-test",
	"P07B C0 unit scope self-test",
	"P07B C0 inherited planning validation",
	"P07B C0 cumulative verification",
	"P07B C0 exact staged scope and diff",
	"P07B C0 scoped staged credential scan",
	"P07B C0 didrun chain intact",
]);
const expectedAuthorityDeclaration = Object.freeze({
	schema_version: "countershape/p07b-c-c0-authority/v1",
	semantic_objects: [
		{ name: "ContractExecutionTarget", storage: "IMMUTABLE_NONHEAD", production_issuer: "C3" },
		{ name: "FinalizedContractRun", storage: "IMMUTABLE_NONHEAD", production_issuer: "C4" },
		{ name: "ContractExecution", storage: "IMMUTABLE_NONHEAD", production_issuer: "C4" },
	],
	operational_objects: [
		{ name: "ExecutionInterlock", storage: "STORE_PRIVATE_MUTABLE", authority: "SPAWN_EXCLUSION_ONLY", production_acquirer: "C4" },
	],
	authority_edges: {
		storage_substrate: "C2_NO_PRODUCTION_ISSUER",
		official_target: "C3_ONLY",
		start_claim_and_run_permit: "C4_RUNNER_ONLY",
		run_permit_consumer: "internal/contractexec/runner",
		process_mechanics: "internal/processmechanics_NO_SEMANTIC_OR_PERMIT_AUTHORITY",
		reset: "EXPLICIT_OPERATOR_PLUS_DISTINCT_MEASURED_BOOT_SESSION",
	},
	scope_domains: [
		"TARGET_INVENTORY",
		"CHILD_BINDINGS",
		"IMPORT_RESOLUTION",
		"SERVICE_BINDINGS",
		"NAMED_PARENT_SECRET_SENTINEL_INHERITANCE",
	],
	scope_states: ["COMPLETE", "PARTIAL", "VIOLATED"],
	reserved_surface: "P08",
	wake_boundary: "IMMUTABLE_REF_INERT_PROVENANCE_ONLY",
	unit_order: ["C0", "C1", "C2", "C3", "C4", "C5", "C6"],
});

const requiredText = Object.freeze({
	"docs/SEMANTICS.md": [
		"legacy planning fixtures superseded by the C0 rulings",
		"store-private `ExecutionInterlock`",
		"Only the C4 contract runner may acquire it from an opaque C3-issued official-target capability",
		"fresh target alone is insufficient",
		"classification-only retry and never reruns a process",
	],
	"docs/ARCHITECTURE.md": [
		"sole mutable operational exception to the nonhead model",
		"internal/processmechanics/",
		"private interlock may only block spawn and carries no result/discovery authority",
	],
	"docs/STATE_MACHINES.md": [
		"EXECUTION_INTERLOCK_HELD_FOR_TARGET_AND_BOOT_SESSION",
		"Missing observation is `UNKNOWN`, never `NOT_STARTED`",
		"SPAWN_OUTCOME_UNKNOWN",
		"classification-publication retry performs no spawn",
		"-> CONTRACT_EXECUTION (immutable nonhead)",
	],
	"docs/THREAT_MODEL.md": [
		"C2 storage mechanics cannot issue this authority",
		"Low-level process mechanics never sees that permit or semantic body",
		"It establishes no host-wide absence, network/registry denial, listener ownership, containment, or confidentiality",
	],
	"docs/CONCEPT_BRIEF.md": [
		"classifier-profile-bound conformance/contradiction or ineligible conclusion",
		"A1/A2/B sealed; C0 design locked; C1 next",
		"private interlock is the sole mutable operational exception",
	],
	"docs/CLAIM_VOCABULARY.md": [
		"| `ExecutionInterlock` |",
		"serialized cooperative at-most-once spawn admission only",
		"`COMPLETE | PARTIAL | VIOLATED` scope axes",
		"immutable nonhead classifier-profile-bound conclusion",
	],
	"docs/PROMPT_PACK.md": [
		"prompts/P07B-C-TARGET-RUN-EXECUTION.md",
		"Only the higher contract runner may consume official target/admission authority",
	],
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md": [
		"C0 → C1 → C2 → C3 → C4 → C5 → C6",
		"## Locked v1 semantic tables",
		"`ContractExecutionTarget`",
		"`FinalizedContractRun`",
		"`ContractExecution`",
		"`ExecutionInterlock` is one store-private, store-wide operational CAS",
		"C2 may implement/test storage mechanics but has no production issuer",
		"one opaque process-local `RunPermit`",
		"`SpawnObservation` separately records",
		"Standalone scope is a separate `COMPLETE | PARTIAL | VIOLATED` sum",
		"A bounded canonical `ClosedRunWitness`",
		"Raw/large evidence remains private behind a manifest",
		"Classifier profile v1 is exact complete-tuple membership",
		"machine-readable C0–C6 path allowlist",
		"machine-readable C0 authority declaration",
		"C6a",
		"C6b",
		"Wake integration is an immutable-ref handoff only",
		"## Combined falsification matrix",
	],
	"research/deep-dive/p07b-c/00-scope-and-method.md": [
		"six read-only specialist lanes, synthesis, different-model red team, three single-claim follow-ups, and two post-write adversarial coherence reviews",
		"restored the three-object split",
		"private boot-session `ExecutionInterlock`",
	],
	"research/deep-dive/p07b-c/01-authority-model.md": [
		"Keep three jurisdictions",
		"Process control and standalone-scope closure are separate axes",
		"prevents a fresh target from bypassing unresolved prior-survivor uncertainty",
	],
	"research/deep-dive/p07b-c/02-runtime-security.md": [
		"Go-owned process edge",
		"named parent-secret sentinel inheritance",
		"higher contract runner—not low-level process mechanics",
	],
	"research/deep-dive/p07b-c/03-store-service.md": [
		"typed, store-private publication witnesses",
		"sole mutable operational exception",
		"fresh target alone is insufficient",
	],
	"research/deep-dive/p07b-c/04-verification.md": [
		"black-box/property/parity/recovery/boundary based",
		"same-boot ambiguity blocks all later subject spawn",
		"A single “P07B-C works” claim is prohibited",
	],
	"research/deep-dive/p07b-c/05-dx-wake-fit.md": [
		"contract, candidate, pinned commit, checked fields, result",
		"Countershape does not consume Wake sessions",
	],
	"research/deep-dive/p07b-c/06-adversarial-design.md": [
		"Three persisted objects",
		"Target + run with nonpersisted view",
		"private boot-session interlock",
	],
	"research/deep-dive/p07b-c/07-synthesis.md": [
		"It was wrong to remove the immutable classification",
		"new target after ambiguity",
		"Questions deliberately handed to red team",
	],
	"research/deep-dive/p07b-c/08-different-model-red-team.md": [
		"Blocker 1 — immutable classification was lost",
		"Blocker 2 — start intent is not spawn evidence",
		"## Post-write closure correction",
	],
	"research/deep-dive/p07b-c/09-follow-up-verification.md": [
		"retain the separate object",
		"fresh target/attempt alone is not retry authority",
		"Private evidence: 16 blobs / 64 MiB aggregate per finalized run",
	],
	"research/deep-dive/p07b-c/10-executive-briefing.md": [
		"Three-object jurisdiction stays",
		"different-model critic",
		"same-boot ambiguity blocks every later subject spawn",
		"UNRECEIPTED",
	],
	"docs/HANDOFF_MODE_C.md": [
		"P07B-C C0 deep-dive/scope-lock files are the current working unit",
		"P07B-C C0 research/authority scope lock only",
		"one private boot-session interlock with no result authority",
	],
	[statusPath]: [
		"Machine-readable authority summary: `spec/verification/p07b-c-c0-authority.json`.",
		"C0 implements no C runtime type",
	],
	"docs/VERIFICATION.md": [
		"P07B-C C0 exact authority-declaration/plan and machine-readable staged-unit-scope mutation self-tests",
		"GOFLAGS=-mod=readonly -buildvcs=false -p=1",
	],
	"spec/verification/p07b-c-unit-paths.json": [
		"countershape/p07b-c-unit-paths/v1",
		"\"C0A\"",
		"\"C6B\"",
	],
	"tools/check-p07b-c-unit-scope.mjs": [
		"P07B-C unit scope self-test passed:",
		"--staged",
		"--no-renames",
	],
	"tools/verify-current.mjs": [
		"const goCommon = Object.freeze([\"-mod=readonly\", \"-buildvcs=false\", \"-p=1\"]);",
		"GOFLAGS: \"-mod=readonly -buildvcs=false -p=1\"",
		"architecture-p07b-c-plan-selftest",
		"architecture-p07b-c-unit-scope-selftest",
	],
	"tools/verify-current-selftest.mjs": [
		"GOFLAGS: \"-mod=readonly -buildvcs=false -p=1\"",
	],
});

const controllingPaths = Object.freeze([
	"docs/SEMANTICS.md",
	"docs/ARCHITECTURE.md",
	"docs/STATE_MACHINES.md",
	"docs/THREAT_MODEL.md",
	"docs/CONCEPT_BRIEF.md",
	"docs/CLAIM_VOCABULARY.md",
	"docs/PROMPT_PACK.md",
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md",
	"docs/HANDOFF_MODE_C.md",
]);

const forbiddenAcrossControllingDocs = Object.freeze([
	"persist exactly two semantic truth objects",
	"ContractExecution` is a pure, deterministic, nonpersisted view",
	"exactly-once execution guarantee",
	"resume the candidate process",
]);

const forbiddenByPath = Object.freeze({
	"docs/SEMANTICS.md": [
		"Process closure owns at most one control reason",
		"confidentiality_established = false",
		"another trial allocates a fresh target and attempt",
	],
	"docs/CONCEPT_BRIEF.md": [
		"B in qualification; C future",
		"explicitly nonconfidential",
		"CONTRACT_EXECUTION_EVIDENCE",
	],
	"docs/STATE_MACHINES.md": [
		"CONTRACT_EXECUTION_EVIDENCE",
		"A retry uses a fresh target/attempt",
	],
	"docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md": [
		"C6 must use explicit Go `-p=1`",
		"Run 60/80/120-column match/differ/ineligible/refused/tamper captures",
	],
});

const requiredC0APaths = Object.freeze([
	...Object.keys(requiredText),
	authorityDeclarationPath,
	"tools/check-p07b-c-plan.mjs",
]);

async function readBytes(root, path, overrides) {
	if (overrides.has(path)) {
		const value = overrides.get(path);
		return Buffer.isBuffer(value) ? value : Buffer.from(value, "utf8");
	}
	return readFile(resolve(root, path));
}

async function readText(root, path, overrides) {
	return (await readBytes(root, path, overrides)).toString("utf8");
}

function checkOrderedPromptHeadings(body, errors) {
	let prior = -1;
	for (const heading of promptHeadings) {
		const positions = body.split("\n").flatMap((line, index) => line === heading ? [index] : []);
		if (positions.length !== 1) {
			errors.push(`docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md: heading must occur exactly once: ${JSON.stringify(heading)}`);
			continue;
		}
		if (positions[0] <= prior) errors.push(`docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md: heading order violation: ${JSON.stringify(heading)}`);
		prior = positions[0];
	}
}

function countOccurrences(body, snippet) {
	return body.split(snippet).length - 1;
}

function requireExactlyOnce(body, snippet, path, phase, errors) {
	const count = countOccurrences(body, snippet);
	if (count !== 1) errors.push(`${path}: ${phase} requires exactly one ${JSON.stringify(snippet)}; found ${count}`);
}

function validateReceiptDeclaration(receipt) {
	const errors = [];
	const keys = [
		"claims", "schema_version", "source_commit", "source_tree", "strict_claims_recorded_exact",
		"strict_claims_total", "strict_exit_code",
	];
	if (!receipt || typeof receipt !== "object" || Array.isArray(receipt) ||
		!isDeepStrictEqual(Object.keys(receipt).sort(), keys)) {
		return ["root field roster"];
	}
	if (receipt.schema_version !== "countershape/p07b-c-c0-receipt/v1") errors.push("schema version");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_commit ?? "")) errors.push("source commit");
	if (!/^[0-9a-f]{40}$/u.test(receipt.source_tree ?? "")) errors.push("source tree");
	if (receipt.strict_exit_code !== 0) errors.push("strict exit code");
	if (receipt.strict_claims_recorded_exact !== c0ClaimLabels.length || receipt.strict_claims_total !== c0ClaimLabels.length) {
		errors.push("strict claim counts");
	}
	if (!Array.isArray(receipt.claims) || receipt.claims.length !== c0ClaimLabels.length) {
		errors.push("claim roster length");
	} else {
		for (let index = 0; index < c0ClaimLabels.length; index += 1) {
			const claim = receipt.claims[index];
			if (!claim || typeof claim !== "object" || Array.isArray(claim) ||
				!isDeepStrictEqual(Object.keys(claim).sort(), ["grade", "label"]) ||
				claim.label !== c0ClaimLabels[index] || claim.grade !== "TREE-EXACT") {
				errors.push(`claim ${index + 1}`);
			}
		}
	}
	return errors;
}

function validateReceiptAuthority(receipt, authority) {
	const errors = [];
	if (!authority || typeof authority !== "object" || Array.isArray(authority)) return ["missing Git/didrun authority"];
	if (authority.commit !== receipt.source_commit) errors.push("source commit does not equal admitted Git commit");
	if (authority.tree !== receipt.source_tree) errors.push("source tree does not equal Git commit tree");
	if (authority.subject !== "docs: lock P07B-C execution authority") errors.push("source commit subject");
	if (authority.ancestor_of_head !== true) errors.push("source commit is not an ancestor of HEAD");
	const note = authority.note;
	if (!note || typeof note !== "object" || Array.isArray(note)) {
		errors.push("missing parsed didrun Git note");
		return errors;
	}
	if (note.commit !== receipt.source_commit) errors.push("didrun note commit");
	if (note.tree !== receipt.source_tree) errors.push("didrun note tree");
	if (!Array.isArray(note.claims) || note.claims.length !== c0ClaimLabels.length) {
		errors.push("didrun note claim roster length");
	} else {
		for (let index = 0; index < c0ClaimLabels.length; index += 1) {
			const recorded = note.claims[index];
			if (recorded?.claim?.label !== c0ClaimLabels[index] || recorded?.grade !== "tree-exact" || recorded?.exit_code !== 0) {
				errors.push(`didrun note claim ${index + 1}`);
			}
		}
	}
	if (note.coverage?.total_events !== c0ClaimLabels.length || note.coverage?.by_coverage?.complete !== c0ClaimLabels.length) {
		errors.push("didrun note exact event coverage");
	}
	return errors;
}

async function loadReceiptAuthorityFromGit(root, receipt) {
	const requested = process.env.COUNTERSHAPE_GIT || "/usr/bin/git";
	if (!isAbsolute(requested)) throw new Error("COUNTERSHAPE_GIT must be absolute");
	const git = await realpath(requested);
	const env = { HOME: process.env.HOME || "/", PATH: "/usr/bin:/bin", LANG: "C", LC_ALL: "C", NO_COLOR: "1" };
	const run = (args, accepted = [0]) => {
		const result = spawnSync(git, args, { cwd: root, encoding: "utf8", timeout: 30_000, maxBuffer: 4 * 1024 * 1024, env });
		if (result.error || result.signal || !accepted.includes(result.status) || (result.stderr?.length ?? 0) !== 0) {
			throw new Error(`git ${args[0]} failed (status=${result.status}, signal=${result.signal}, error=${result.error?.message ?? "none"})`);
		}
		return { status: result.status, stdout: result.stdout ?? "" };
	};
	const commit = run(["rev-parse", "--verify", `${receipt.source_commit}^{commit}`]).stdout.trim();
	if (commit !== receipt.source_commit) throw new Error("source commit did not reopen exactly");
	const tree = run(["rev-parse", "--verify", `${receipt.source_commit}^{tree}`]).stdout.trim();
	const subject = run(["show", "-s", "--format=%s", receipt.source_commit]).stdout.trimEnd();
	const ancestor = run(["merge-base", "--is-ancestor", receipt.source_commit, "HEAD"], [0, 1]).status === 0;
	const noteText = run(["notes", "--ref=didrun", "show", receipt.source_commit]).stdout;
	let note;
	try {
		note = JSON.parse(noteText);
	} catch (error) {
		throw new Error(`didrun Git note is not JSON (${error.message})`);
	}
	return { commit, tree, subject, ancestor_of_head: ancestor, note };
}

async function checkStatusPhase(root, overrides, status, handoff, errors, injectedReceiptAuthority) {
	let receiptText;
	try {
		receiptText = await readText(root, receiptDeclarationPath, overrides);
	} catch (error) {
		if (error.code !== "ENOENT") errors.push(`${receiptDeclarationPath}: unreadable (${error.message})`);
	}

	if (receiptText === undefined) {
		requireExactlyOnce(
			status,
			"**State:** provisional C0a source document; every C0 row below is `UNRECEIPTED`",
			statusPath,
			"C0A provisional state",
			errors,
		);
		for (const label of c0ClaimLabels) {
			requireExactlyOnce(status, `| \`${label}\` | \`UNRECEIPTED\` |`, statusPath, "C0A provisional receipt map", errors);
		}
		if (status.includes("C0b reconciled receipt document")) {
			errors.push(`${statusPath}: C0A cannot claim reconciled receipt state without ${receiptDeclarationPath}`);
		}
		if (handoff.includes("<!-- P07B-C-C0A-RECEIPTS:START -->") || handoff.includes("<!-- P07B-C-C0A-RECEIPTS:END -->")) {
			errors.push(`docs/HANDOFF_MODE_C.md: C0A cannot contain a reconciled C0a receipt block`);
		}
		return;
	}

	let receipt;
	try {
		receipt = JSON.parse(receiptText);
	} catch (error) {
		errors.push(`${receiptDeclarationPath}: invalid JSON (${error.message})`);
		return;
	}
	const receiptErrors = validateReceiptDeclaration(receipt);
	for (const error of receiptErrors) errors.push(`${receiptDeclarationPath}: invalid receipt declaration (${error})`);
	if (receiptErrors.length > 0) return;
	let receiptAuthority = injectedReceiptAuthority;
	if (receiptAuthority === undefined) {
		try {
			receiptAuthority = await loadReceiptAuthorityFromGit(root, receipt);
		} catch (error) {
			errors.push(`${receiptDeclarationPath}: Git/didrun authority unavailable (${error.message})`);
			return;
		}
	}
	const authorityErrors = validateReceiptAuthority(receipt, receiptAuthority);
	for (const error of authorityErrors) errors.push(`${receiptDeclarationPath}: Git/didrun authority mismatch (${error})`);
	if (authorityErrors.length > 0) return;

	requireExactlyOnce(
		status,
		`**State:** C0b reconciled receipt document for source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`,
		statusPath,
		"C0B reconciled state",
		errors,
	);
	requireExactlyOnce(status, "## C0a receipt map", statusPath, "C0B receipt heading", errors);
	requireExactlyOnce(status, "- C0a strict exit: `0`", statusPath, "C0B strict result", errors);
	requireExactlyOnce(
		status,
		`- C0a strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\``,
		statusPath,
		"C0B strict claim count",
		errors,
	);
	for (const claim of receipt.claims) {
		requireExactlyOnce(status, `| \`${claim.label}\` | \`${claim.grade}\` |`, statusPath, "C0B receipt map", errors);
	}
	requireExactlyOnce(handoff, "<!-- P07B-C-C0A-RECEIPTS:START -->", "docs/HANDOFF_MODE_C.md", "C0B receipt block", errors);
	requireExactlyOnce(handoff, "<!-- P07B-C-C0A-RECEIPTS:END -->", "docs/HANDOFF_MODE_C.md", "C0B receipt block", errors);
	requireExactlyOnce(
		handoff,
		`C0a source commit \`${receipt.source_commit}\`, tree \`${receipt.source_tree}\`.`,
		"docs/HANDOFF_MODE_C.md",
		"C0B source identity",
		errors,
	);
	requireExactlyOnce(
		handoff,
		`C0a strict claims: \`${receipt.strict_claims_recorded_exact}/${receipt.strict_claims_total} claims recorded-exact\`; strict exit: \`0\`.`,
		"docs/HANDOFF_MODE_C.md",
		"C0B strict result",
		errors,
	);
	for (const claim of receipt.claims) {
		requireExactlyOnce(handoff, `| \`${claim.label}\` | \`${claim.grade}\` |`, "docs/HANDOFF_MODE_C.md", "C0B claim map", errors);
	}
}

export async function checkPlan(root = repositoryRoot, overrides = new Map(), receiptAuthority) {
	const errors = [];
	const bodies = new Map();

	for (const [path, snippets] of Object.entries(requiredText)) {
		let body;
		try {
			body = await readText(root, path, overrides);
			bodies.set(path, body);
		} catch (error) {
			errors.push(`${path}: unreadable (${error.message})`);
			continue;
		}
		for (const snippet of snippets) {
			if (!body.includes(snippet)) errors.push(`${path}: missing required C0 ruling: ${JSON.stringify(snippet)}`);
		}
	}

	const status = bodies.get(statusPath);
	const handoff = bodies.get("docs/HANDOFF_MODE_C.md");
	if (status !== undefined && handoff !== undefined) {
		await checkStatusPhase(root, overrides, status, handoff, errors, receiptAuthority);
	}

	for (const path of controllingPaths) {
		const body = bodies.get(path);
		if (body === undefined) continue;
		for (const snippet of forbiddenAcrossControllingDocs) {
			if (body.includes(snippet)) errors.push(`${path}: contains superseded or overbroad ruling: ${JSON.stringify(snippet)}`);
		}
		for (const snippet of forbiddenByPath[path] ?? []) {
			if (body.includes(snippet)) errors.push(`${path}: contains superseded path-specific ruling: ${JSON.stringify(snippet)}`);
		}
	}

	const prompt = bodies.get("docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md");
	if (prompt !== undefined) checkOrderedPromptHeadings(prompt, errors);

	for (const [path, heading] of Object.entries(researchShape)) {
		const body = bodies.get(path);
		if (body === undefined) continue;
		if (!body.startsWith(`${heading}\n`)) errors.push(`${path}: exact H1 mismatch`);
		if (Buffer.byteLength(body, "utf8") < 1500) errors.push(`${path}: evidence file is not substantive (under 1500 bytes)`);
	}

	for (const [path, expected] of Object.entries(legacyPlanningDigests)) {
		let body;
		try {
			body = await readBytes(root, path, overrides);
		} catch (error) {
			errors.push(`${path}: unreadable legacy planning fixture (${error.message})`);
			continue;
		}
		const actual = createHash("sha256").update(body).digest("hex");
		if (actual !== expected) errors.push(`${path}: C0 must not alter legacy planning fixture; expected ${expected}, got ${actual}`);
	}

	try {
		const declaration = JSON.parse(await readText(root, authorityDeclarationPath, overrides));
		if (!isDeepStrictEqual(declaration, expectedAuthorityDeclaration)) {
			errors.push(`${authorityDeclarationPath}: authority declaration mismatch`);
		}
	} catch (error) {
		errors.push(`${authorityDeclarationPath}: invalid authority declaration (${error.message})`);
	}

	try {
		const specification = validateSpecification(JSON.parse(await readText(root, "spec/verification/p07b-c-unit-paths.json", overrides)));
		for (const path of requiredC0APaths) {
			if (!specification.units.C0A.exact.includes(path)) errors.push(`spec/verification/p07b-c-unit-paths.json: C0A omits required path ${path}`);
		}
	} catch (error) {
		errors.push(`spec/verification/p07b-c-unit-paths.json: invalid (${error.message})`);
	}

	return errors;
}

async function mutatedText(path, transform) {
	return transform(await readText(repositoryRoot, path, new Map()));
}

async function runSelfTest() {
	const baseline = await checkPlan();
	if (baseline.length > 0) throw new Error(`P07B-C C0 plan checker self-test baseline failed:\n${baseline.join("\n")}`);

	const promptPath = "docs/prompts/P07B-C-TARGET-RUN-EXECUTION.md";
	const prompt = await readText(repositoryRoot, promptPath, new Map());
	const authority = JSON.parse(await readText(repositoryRoot, authorityDeclarationPath, new Map()));
	const mutateAuthority = (transform) => {
		const candidate = JSON.parse(JSON.stringify(authority));
		transform(candidate);
		return `${JSON.stringify(candidate, null, 2)}\n`;
	};
	const configPath = "spec/verification/p07b-c-unit-paths.json";
	const configuration = JSON.parse(await readText(repositoryRoot, configPath, new Map()));
	delete configuration.units.C6B;
	const syntheticReceipt = {
		schema_version: "countershape/p07b-c-c0-receipt/v1",
		source_commit: "a".repeat(40),
		source_tree: "b".repeat(40),
		strict_exit_code: 0,
		strict_claims_recorded_exact: c0ClaimLabels.length,
		strict_claims_total: c0ClaimLabels.length,
		claims: c0ClaimLabels.map((label) => ({ label, grade: "TREE-EXACT" })),
	};
	const syntheticAuthority = {
		commit: syntheticReceipt.source_commit,
		tree: syntheticReceipt.source_tree,
		subject: "docs: lock P07B-C execution authority",
		ancestor_of_head: true,
		note: {
			commit: syntheticReceipt.source_commit,
			tree: syntheticReceipt.source_tree,
			claims: c0ClaimLabels.map((label) => ({ claim: { label }, grade: "tree-exact", exit_code: 0 })),
			coverage: { total_events: c0ClaimLabels.length, by_coverage: { complete: c0ClaimLabels.length } },
		},
	};
	const currentStatus = await readText(repositoryRoot, statusPath, new Map());
	const syntheticState = `**State:** C0b reconciled receipt document for source commit \`${syntheticReceipt.source_commit}\`, tree \`${syntheticReceipt.source_tree}\`.`;
	let syntheticStatus = currentStatus
		.replace(/^\*\*State:\*\*.*$/mu, syntheticState)
		.replace(/^## (?:Provisional receipt map|C0a receipt map)$/mu, "## C0a receipt map")
		.replace(/^- C0a strict exit:.*\n?/mu, "")
		.replace(/^- C0a strict claims:.*\n?/mu, "");
	syntheticStatus = syntheticStatus.replace(
		syntheticState,
		`${syntheticState}\n\n- C0a strict exit: \`0\`\n- C0a strict claims: \`${c0ClaimLabels.length}/${c0ClaimLabels.length} claims recorded-exact\``,
	);
	for (const label of c0ClaimLabels) {
		syntheticStatus = syntheticStatus.replace(
			`| \`${label}\` | \`UNRECEIPTED\` |`,
			`| \`${label}\` | \`TREE-EXACT\` |`,
		);
	}
	const receiptBlock = [
		"<!-- P07B-C-C0A-RECEIPTS:START -->",
		"### P07B-C C0a receipt map",
		"",
		`C0a source commit \`${syntheticReceipt.source_commit}\`, tree \`${syntheticReceipt.source_tree}\`.`,
		`C0a strict claims: \`${c0ClaimLabels.length}/${c0ClaimLabels.length} claims recorded-exact\`; strict exit: \`0\`.`,
		"",
		"| Claim label | Verbatim grade |",
		"| --- | --- |",
		...c0ClaimLabels.map((label) => `| \`${label}\` | \`TREE-EXACT\` |`),
		"<!-- P07B-C-C0A-RECEIPTS:END -->",
	].join("\n");
	const currentHandoff = await readText(repositoryRoot, "docs/HANDOFF_MODE_C.md", new Map());
	const syntheticHandoff = `${currentHandoff.replace(
		/\n?<!-- P07B-C-C0A-RECEIPTS:START -->[\s\S]*?<!-- P07B-C-C0A-RECEIPTS:END -->\n?/u,
		"\n",
	).trimEnd()}\n\n${receiptBlock}\n`;
	const syntheticReceiptText = `${JSON.stringify(syntheticReceipt, null, 2)}\n`;
	const syntheticOverrides = new Map([
		[statusPath, syntheticStatus],
		[receiptDeclarationPath, syntheticReceiptText],
		["docs/HANDOFF_MODE_C.md", syntheticHandoff],
	]);
	const syntheticBaseline = await checkPlan(repositoryRoot, syntheticOverrides, syntheticAuthority);
	if (syntheticBaseline.length > 0) {
		throw new Error(`P07B-C C0 plan checker synthetic C0B baseline failed:\n${syntheticBaseline.join("\n")}`);
	}

	let currentReceiptText;
	try {
		currentReceiptText = await readText(repositoryRoot, receiptDeclarationPath, new Map());
	} catch (error) {
		if (error.code !== "ENOENT") throw error;
	}
	const phaseReceiptCase = currentReceiptText === undefined ? {
		name: "provisional receipt-status inflation", path: statusPath,
		value: currentStatus.replace(
			`| \`${c0ClaimLabels[0]}\` | \`UNRECEIPTED\` |`,
			`| \`${c0ClaimLabels[0]}\` | \`RECEIPTED\` |`,
		),
		expect: "C0A provisional receipt map",
	} : {
		name: "reconciled receipt-grade mutation", path: receiptDeclarationPath,
		value: (() => {
			const candidate = JSON.parse(currentReceiptText);
			candidate.claims[0].grade = "UNRECEIPTED";
			return `${JSON.stringify(candidate, null, 2)}\n`;
		})(),
		expect: "invalid receipt declaration",
	};
	const cases = [
		{
			name: "required interlock authority edge", path: promptPath,
			value: prompt.replace("`ExecutionInterlock` is one store-private, store-wide operational CAS", "Interlock omitted"),
			expect: "missing required C0 ruling",
		},
		{
			name: "contradictory two-object ruling", path: "docs/SEMANTICS.md",
			value: `${await readText(repositoryRoot, "docs/SEMANTICS.md", new Map())}\npersist exactly two semantic truth objects\n`,
			expect: "contains superseded or overbroad ruling",
		},
		{
			name: "legacy fixture mutation", path: "spec/schema/v1/contract-execution.schema.json",
			value: `${await readText(repositoryRoot, "spec/schema/v1/contract-execution.schema.json", new Map())}\n`,
			expect: "C0 must not alter legacy planning fixture",
		},
		{
			name: "evidence content deletion", path: "research/deep-dive/p07b-c/08-different-model-red-team.md",
			value: await mutatedText("research/deep-dive/p07b-c/08-different-model-red-team.md", (body) => body.replace("## Post-write closure correction", "## Closure")),
			expect: "missing required C0 ruling",
		},
		{
			name: "prompt heading order", path: promptPath,
			value: prompt.replace(promptHeadings[2], "# TEMP-C2").replace(promptHeadings[3], promptHeadings[2]).replace("# TEMP-C2", promptHeadings[3]),
			expect: "heading order violation",
		},
		{
			name: "substantive evidence deletion", path: "research/deep-dive/p07b-c/05-dx-wake-fit.md",
			value: `${researchShape["research/deep-dive/p07b-c/05-dx-wake-fit.md"]}\ncontract, candidate, pinned commit, checked fields, result\nCountershape does not consume Wake sessions\n`,
			expect: "not substantive",
		},
		{
			name: "unit allowlist roster", path: configPath,
			value: `${JSON.stringify(configuration, null, 2)}\n`,
			expect: "invalid",
		},
		{
			name: "fourth semantic object", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => candidate.semantic_objects.push({
				name: "FourthObject", storage: "IMMUTABLE_NONHEAD", production_issuer: "C4",
			})),
			expect: "authority declaration mismatch",
		},
		{
			name: "second mutable operational object", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => candidate.operational_objects.push({
				name: "MutableStatus", storage: "STORE_PRIVATE_MUTABLE", authority: "RESULT_STATUS", production_acquirer: "C4",
			})),
			expect: "authority declaration mismatch",
		},
		{
			name: "premature C2 target issuer", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.semantic_objects[0].production_issuer = "C2"; }),
			expect: "authority declaration mismatch",
		},
		{
			name: "same-boot reset weakening", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.authority_edges.reset = "EXPLICIT_OPERATOR_ONLY"; }),
			expect: "authority declaration mismatch",
		},
		{
			name: "scope-domain loss", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.scope_domains.pop(); }),
			expect: "authority declaration mismatch",
		},
		{
			name: "P08 surface reservation loss", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.reserved_surface = "C6"; }),
			expect: "authority declaration mismatch",
		},
		{
			name: "Wake authority expansion", path: authorityDeclarationPath,
			value: mutateAuthority((candidate) => { candidate.wake_boundary = "SESSION_RESUME"; }),
			expect: "authority declaration mismatch",
		},
		phaseReceiptCase,
		{
			name: "synthetic C0B receipt/status disagreement",
			overrides: new Map([
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText.replace("TREE-EXACT", "UNRECEIPTED")],
				["docs/HANDOFF_MODE_C.md", syntheticHandoff],
			]),
			receiptAuthority: syntheticAuthority,
			expect: "invalid receipt declaration",
		},
		{
			name: "synthetic C0B wrong Git parent",
			overrides: new Map([
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText.replace(`"${syntheticReceipt.source_commit}"`, `"${"c".repeat(40)}"`)],
				["docs/HANDOFF_MODE_C.md", syntheticHandoff],
			]),
			receiptAuthority: syntheticAuthority,
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "synthetic C0B wrong Git tree",
			overrides: new Map([
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText.replace(`"${syntheticReceipt.source_tree}"`, `"${"d".repeat(40)}"`)],
				["docs/HANDOFF_MODE_C.md", syntheticHandoff],
			]),
			receiptAuthority: syntheticAuthority,
			expect: "Git/didrun authority mismatch",
		},
		{
			name: "synthetic C0B handoff mismatch",
			overrides: new Map([
				[statusPath, syntheticStatus],
				[receiptDeclarationPath, syntheticReceiptText],
				["docs/HANDOFF_MODE_C.md", syntheticHandoff.replace(
					`C0a source commit \`${syntheticReceipt.source_commit}\`, tree \`${syntheticReceipt.source_tree}\`.`,
					`C0a source commit \`${syntheticReceipt.source_commit}\`, tree \`${"e".repeat(40)}\`.`,
				)],
			]),
			receiptAuthority: syntheticAuthority,
			expect: "C0B source identity",
		},
	];

	for (const testCase of cases) {
		const overrides = testCase.overrides ?? new Map([[testCase.path, testCase.value]]);
		const errors = await checkPlan(repositoryRoot, overrides, testCase.receiptAuthority);
		if (!errors.some((error) => error.includes(testCase.expect))) {
			throw new Error(`P07B-C C0 plan checker self-test false negative: ${testCase.name}`);
		}
	}

	console.log(`P07B-C C0 plan checker self-test passed: ${cases.length} authority, structure, evidence, digest, and allowlist mutations rejected`);
}

async function main() {
	const mode = process.argv[2];
	if (mode === "--self-test") {
		if (process.argv.length !== 3) throw new Error("usage: check-p07b-c-plan.mjs [--self-test]");
		await runSelfTest();
		return;
	}
	if (mode !== undefined) throw new Error("usage: check-p07b-c-plan.mjs [--self-test]");

	const errors = await checkPlan();
	if (errors.length > 0) {
		console.error("P07B-C C0 plan check failed:");
		for (const error of errors) console.error(`- ${error}`);
		process.exitCode = 1;
		return;
	}

	console.log("P07B-C C0 plan check passed: documents contain the locked three-object/interlock/scope/classification rulings; runtime remains UNRECEIPTED");
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
	try {
		await main();
	} catch (error) {
		console.error(error.message);
		process.exitCode = 1;
	}
}
