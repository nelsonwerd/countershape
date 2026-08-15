#!/usr/bin/env node

import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { basename, join, resolve } from "node:path";

function fail(code, detail) {
	process.stderr.write(`${code}: ${detail}\n`);
	process.exit(1);
}

function option(name) {
	const index = process.argv.indexOf(name);
	if (index < 0 || !process.argv[index + 1]) {
		fail("S6_AUDIT_ARGUMENT", name);
	}
	return process.argv[index + 1];
}

function sha256(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

function readJSONLines(path) {
	const text = readFileSync(path, "utf8").trim();
	return text === "" ? [] : text.split("\n").map((line) => JSON.parse(line));
}

function commandRows(rawRoot, ordinal) {
	return readJSONLines(join(rawRoot, `session-${ordinal}-codex.jsonl`))
		.filter((entry) => entry.type === "item.completed" && entry.item?.type === "command_execution")
		.map((entry) => ({ command: entry.item.command, exit_code: entry.item.exit_code }));
}

function capturedRows(rawRoot, ordinal) {
	return readJSONLines(join(rawRoot, `session-${ordinal}-capture`, "session.log"))
		.map((entry) => ({
			argv0: basename(entry.event.argv[0]),
			exit_code: entry.event.exit_code,
			coverage: entry.event.coverage,
			observed_via: entry.event.observed_via,
		}));
}

function countFullTranscriptBare(rows) {
	let count = 0;
	for (const row of rows) {
		const command = row.command;
		if (command === "/bin/zsh -lc fixture-test" || command === "/bin/zsh -lc fixture-fail") {
			count += 1;
		} else if (command === "/bin/zsh -lc 'git diff -- message.txt count.txt'") {
			count += 1;
		} else if (command.includes("rg --files")) {
			count += 1;
		} else if (command.includes("ls -la && sed -n '1,5p' message.txt && sed -n '1,5p' count.txt")) {
			count += 3;
		}
	}
	return count;
}

const rawRoot = resolve(option("--raw"));
const fixtureRoot = resolve(option("--fixture-root"));
const expectedPromptSHA256 = option("--prompt-sha256");
const expectedFixtureCommit = option("--fixture-commit");
const credentialPattern = /-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----|(?:AKIA|ASIA)[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{30,}|glpat-[A-Za-z0-9_-]{20,}|xox[baprs]-[A-Za-z0-9-]{20,}|AIza[0-9A-Za-z_-]{30,}|sk_live_[0-9A-Za-z]{16,}|sk-(?:proj|svcacct)-[A-Za-z0-9_-]{20,}|npm_[A-Za-z0-9]{24,}|pypi-[A-Za-z0-9_-]{24,}|AccountKey=[A-Za-z0-9+/=]{24,}/;

const sessions = [];
let requiredBareTotal = 0;
let requiredBareCaptured = 0;
let fullBareTotal = 0;
let fullBareCaptured = 0;
let absoluteStructuralTotal = 0;
let absoluteStructuralAccounted = 0;

for (const ordinal of [1, 2]) {
	const sessionRoot = join(fixtureRoot, `session-${ordinal}`);
	const promptSHA256 = sha256(readFileSync(join(sessionRoot, "TASK.md")));
	if (promptSHA256 !== expectedPromptSHA256) {
		fail("S6_PROMPT_DIGEST", `session ${ordinal}`);
	}
	const fixtureCommit = execFileSync("/usr/bin/git", ["-C", sessionRoot, "rev-parse", "HEAD"], { encoding: "utf8" }).trim();
	if (fixtureCommit !== expectedFixtureCommit) {
		fail("S6_FIXTURE_COMMIT", `session ${ordinal}`);
	}
	const changedPaths = execFileSync("/usr/bin/git", ["-C", sessionRoot, "diff", "--name-only"], { encoding: "utf8" })
		.trim().split("\n").filter(Boolean).sort();
	if (JSON.stringify(changedPaths) !== JSON.stringify(["count.txt", "message.txt"])) {
		fail("S6_CHANGED_PATHS", `session ${ordinal}: ${changedPaths.join(",")}`);
	}
	if (readFileSync(join(sessionRoot, "message.txt"), "utf8") !== "countershape-ready\n" ||
		readFileSync(join(sessionRoot, "count.txt"), "utf8") !== "2\n") {
		fail("S6_FIXTURE_CONTENT", `session ${ordinal}`);
	}

	const commands = commandRows(rawRoot, ordinal);
	const captured = capturedRows(rawRoot, ordinal);
	const expectedRequired = [
		{ command: "/bin/zsh -lc fixture-test", exit_code: 0 },
		{ command: "/bin/zsh -lc fixture-fail", exit_code: 7 },
		{ command: "/bin/zsh -lc /bin/pwd", exit_code: 0 },
		{ command: "/bin/zsh -lc 'git diff -- message.txt count.txt'", exit_code: 0 },
		{ command: "/bin/zsh -lc fixture-test", exit_code: 0 },
	];
	let cursor = 0;
	for (const expected of expectedRequired) {
		cursor = commands.findIndex((row, index) => index >= cursor && row.command === expected.command && row.exit_code === expected.exit_code);
		if (cursor < 0) {
			fail("S6_REQUIRED_COMMAND", `session ${ordinal}: ${expected.command}`);
		}
		cursor += 1;
	}
	const expectedCaptured = [
		{ argv0: "fixture-test-real", exit_code: 0 },
		{ argv0: "fixture-fail-real", exit_code: 7 },
		{ argv0: "fixture-test-real", exit_code: 0 },
	];
	if (captured.length !== expectedCaptured.length) {
		fail("S6_CAPTURE_CARDINALITY", `session ${ordinal}: ${captured.length}`);
	}
	for (let index = 0; index < expectedCaptured.length; index += 1) {
		const actual = captured[index];
		const expected = expectedCaptured[index];
		if (actual.argv0 !== expected.argv0 || actual.exit_code !== expected.exit_code ||
			actual.coverage !== "complete" || actual.observed_via !== "wrapper") {
			fail("S6_CAPTURE_ROW", `session ${ordinal}: ${index}`);
		}
	}
	const rawFiles = [
		join(rawRoot, `session-${ordinal}-codex.jsonl`),
		join(rawRoot, `session-${ordinal}-codex.stderr`),
		join(rawRoot, `session-${ordinal}-final.txt`),
		join(rawRoot, `session-${ordinal}-capture`, "session.log"),
	];
	for (const path of rawFiles) {
		if (credentialPattern.test(readFileSync(path, "utf8"))) {
			fail("S6_NAMED_CREDENTIAL_PATTERN", `session ${ordinal}`);
		}
	}

	const fullBare = countFullTranscriptBare(commands);
	requiredBareTotal += 4;
	requiredBareCaptured += 3;
	fullBareTotal += fullBare;
	fullBareCaptured += 3;
	absoluteStructuralTotal += 1;
	absoluteStructuralAccounted += 1;
	sessions.push({
		ordinal,
		fixture_commit: fixtureCommit,
		prompt_sha256: promptSHA256,
		changed_paths: changedPaths,
		required_bare: { captured: 3, total: 4 },
		full_transcript_bare: { captured: 3, total: fullBare },
		absolute_path: { captured: 0, structural_miss_accounted: 1, total: 1 },
		intentional_failure: { transcript_exit: 7, captured_exit: 7 },
	});
}

const requiredRecall = requiredBareCaptured / requiredBareTotal;
const fullRecall = fullBareCaptured / fullBareTotal;
const verdict = requiredRecall >= 0.95 && fullRecall >= 0.95 ? "PASS" :
	(requiredRecall >= 0.80 && fullRecall >= 0.80 ? "TIER-DOWNGRADE" : "PARK");
const result = {
	schema_version: "countershape/s6-live-agent-matrix/v1",
	provider: "OpenAI",
	model: "gpt-5.6-terra",
	codex_cli: {
		path: "/Applications/ChatGPT.app/Contents/Resources/codex",
		version: "codex-cli 0.147.0-alpha.6.5",
		flags: ["--ephemeral", "--ignore-user-config", "--ignore-rules", "--json", "--color", "never", "--sandbox", "workspace-write", "--model", "gpt-5.6-terra", "-c", "model_reasoning_effort=medium"],
	},
	didrun: {
		path: "/opt/homebrew/bin/didrun",
		version: "didrun 0.1.0",
		capture: "Tier-1 PATH shim",
	},
	sessions,
	required_bare_recall: { captured: requiredBareCaptured, total: requiredBareTotal, ratio: requiredRecall },
	full_transcript_bare_recall: { captured: fullBareCaptured, total: fullBareTotal, ratio: fullRecall },
	absolute_path: { captured: 0, structural_miss_accounted: absoluteStructuralAccounted, total: absoluteStructuralTotal },
	verdict,
	limitations: [
		"STANDARD_GIT_SHIM_DID_NOT_SURVIVE_CODEX_SHELL_COMMAND_RESOLUTION",
		"UNSHIMMED_SPONTANEOUS_BARE_COMMANDS_WERE_SILENTLY_MISSED",
		"ABSOLUTE_PATH_EXECUTIONS_ARE_DOCUMENTED_STRUCTURAL_MISSES",
		"NAMED_CREDENTIAL_PATTERN_SCAN_IS_NOT_SECRET_ABSENCE_AUTHORITY",
	],
};
if (verdict !== "PARK") {
	fail("S6_EXPECTED_PARK", verdict);
}
process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
