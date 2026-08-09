#!/usr/bin/env node

import { realpath } from "node:fs/promises";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

import { hermeticPrefix, loadSpecification, repositoryRoot, unitByID } from "./check-u7-plan.mjs";

function fail(code, detail) {
	throw new Error(`U7_RUNBOOK_${code}: ${detail}`);
}

function shellQuote(value) {
	if (typeof value !== "string" || value.includes("\0") || value.includes("\n") || value.includes("\r")) fail("SHELL_VALUE", String(value));
	return `'${value.replaceAll("'", `'"'"'`)}'`;
}

function commandLine(argv) {
	return argv.map(shellQuote).join(" ");
}

function rootFenceLines(...commands) {
	if (commands.some((command) => typeof command !== "string" || /[\r\n]/u.test(command))) fail("FENCE", "one command per line");
	return [
		"(", "  set -eu", "  umask 077", `  cd -P -- ${shellQuote(repositoryRoot)}`,
		`  /bin/test \"$PWD\" = ${shellQuote(repositoryRoot)}`, ...commands.map((command) => `  ${command}`), ")",
	];
}

function fenced(commands) {
	return `\`\`\`sh\n${rootFenceLines(...commands).join("\n")}\n\`\`\``;
}

function isolatedProcessPrefix(specification, unit) {
	const runRoot = resolve(repositoryRoot, unit.final_root);
	return ["/usr/bin/env", "-i", `HOME=${resolve(runRoot, "home")}`, `TMPDIR=${resolve(runRoot, "tmp")}`,
		...specification.runtime_authority.didrun_outer_static_environment];
}

function isolatedLine(specification, unit, executable, ...args) {
	return commandLine([...isolatedProcessPrefix(specification, unit), executable, ...args]);
}

function recoveryProcessPrefix(specification) {
	const privateControlRoot = resolve(repositoryRoot, ".countershape");
	return ["/usr/bin/env", "-i", `HOME=${privateControlRoot}`, `TMPDIR=${privateControlRoot}`,
		...specification.runtime_authority.didrun_outer_static_environment];
}

function recoveryLine(specification, executable, ...args) {
	return commandLine([...recoveryProcessPrefix(specification), executable, ...args]);
}

function commitBindingLines(specification, unit) {
	return [
		`commit=$(${isolatedLine(specification, unit, "/usr/bin/git", "--no-replace-objects", "rev-parse", "--verify", "HEAD^{commit}")})`,
		`short=\${commit%"\${commit#????????????}"}`,
		`/bin/test "\${#short}" -eq 12`,
	];
}

function inheritedIdentityLines(specification, unit) {
	return [
		`author_name=$(${isolatedLine(specification, unit, "/usr/bin/git", "--no-replace-objects", "show", "-s", "--format=%an", "HEAD")})`,
		`author_email=$(${isolatedLine(specification, unit, "/usr/bin/git", "--no-replace-objects", "show", "-s", "--format=%ae", "HEAD")})`,
		`committer_name=$(${isolatedLine(specification, unit, "/usr/bin/git", "--no-replace-objects", "show", "-s", "--format=%cn", "HEAD")})`,
		`committer_email=$(${isolatedLine(specification, unit, "/usr/bin/git", "--no-replace-objects", "show", "-s", "--format=%ce", "HEAD")})`,
	];
}

function outerRecorderPrefix(specification, unit) {
	return [...isolatedProcessPrefix(specification, unit), specification.runtime_authority.didrun_path];
}

function recorderLine(specification, unit, ...args) {
	return commandLine([...outerRecorderPrefix(specification, unit), ...args]);
}

function commandFence(specification, unit, claim, index) {
	const claimArgs = ["claim", claim.type, "--label", claim.label, "--event", String(index), ...unit.allowed_paths.flatMap((path) => ["--path", path])];
	return fenced([
		recorderLine(specification, unit, "run", "--", ...hermeticPrefix(unit), ...claim.command),
		recorderLine(specification, unit, ...claimArgs),
	]);
}

export function renderRunbook(specification, unitID) {
	const unit = unitByID(specification, unitID);
	const lower = unit.id.toLowerCase();
	const commands = unit.claims.map((claim, index) => `### ${index + 1}. ${claim.label}\n\n${commandFence(specification, unit, claim, index)}`).join("\n\n");
	const preflightLine = (...args) => isolatedLine(specification, unit, "/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--verify-runbook-preflight", ...args);
	const recoveryCommand = (...args) => recoveryLine(specification, "/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", ...args);
	const preparePreflightLine = recoveryCommand("--verify-runbook-preflight", "prepare", unit.id);
	const commitLine = `${commandLine(isolatedProcessPrefix(specification, unit))} GIT_AUTHOR_NAME="$author_name" GIT_AUTHOR_EMAIL="$author_email" GIT_COMMITTER_NAME="$committer_name" GIT_COMMITTER_EMAIL="$committer_email" ${commandLine(["/usr/bin/git", "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgSign=false", "-c", "tag.gpgSign=false", "-c", "core.fsmonitor=false", "--no-replace-objects", "commit", "--no-verify", "--no-gpg-sign", "--cleanup=verbatim", "-m", unit.subject])}`;
	const recorder = commandLine(outerRecorderPrefix(specification, unit));
	const noteBase = recorderLine(specification, unit, "run", "--", ...hermeticPrefix(unit), "/usr/bin/git", "notes", "--ref=didrun", "show");
	return `# ${unit.id} deterministic final runbook\n\n` +
			`Generated instructions are not a receipt. Execute only after the candidate has stopped changing and every development ledger has been preserved. Grades remain \`UNRECEIPTED\` until the exact commit is sealed and explicit-commit strict verification exits zero.\n\n` +
			`Exactly one root-owned writer owns all repository, didrun, claim, Git, seal, recovery, and archive mutation for this full attempt. Do not enter the next fence or mutate candidate, evidence, or recovery state until any yielded didrun wrapper and every relevant child process are confirmed process-terminal. Empty or yielded app output is not terminal evidence.\n\n` +
			`## Prepare one fresh evidence boundary\n\n` +
			`The no-follow preflight rejects any filesystem object, including a dangling symbolic link, at the live ledger, verifier lock, or unit final root. This renderer never stages, deletes, or overwrites evidence; its terminal step atomically moves the completed live ledger into a fresh commit-addressed archive.\n\n` +
			`${fenced([
				preparePreflightLine,
				`/usr/bin/install -d -m 0700 ${shellQuote(resolve(repositoryRoot, ".countershape"))} ${shellQuote(resolve(repositoryRoot, ".countershape/evidence"))} ${shellQuote(resolve(repositoryRoot, ".didrun-history"))}`,
				`/usr/bin/install -d -m 0700 ${[unit.final_root, ...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"].map((name) => `${unit.final_root}/${name}`)].map((path) => shellQuote(resolve(repositoryRoot, path))).join(" ")}`,
			])}\n\n` +
		`## Run each exact command and claim it immediately\n\n${commands}\n\n` +
		`## Commit and attempt the mandatory plain seal\n\n` +
		`No unreceipted command belongs between the final claim and this commit. A failed plain seal leaves the commit in place; do not rerun this fence.\n\n` +
			`${fenced([...inheritedIdentityLines(specification, unit), commitLine, ...commitBindingLines(specification, unit), `${preflightLine("commit", unit.id)} "$commit"`, `${recorder} 'seal' '--commit' "$commit"`])}\n\n` +
		`If the plain seal refuses solely on didrun's high-entropy heuristic, stop and review the exact staged inventory and claimed credential scan. The following is a manual checkpoint, not an automatic fallback; deliberately remove the comment only after that review:\n\n` +
			`${fenced([...commitBindingLines(specification, unit), `# ${recorder} 'seal' '--commit' "$commit" '--allow-secrets'`])}\n\n` +
		`## Inspect the readable note through didrun\n\n` +
		`This post-seal event supports no new claim.\n\n` +
			`${fenced([...commitBindingLines(specification, unit), `${noteBase} "$commit"`])}\n\n` +
		`## Verify the exact commit and emit its HTML witness\n\n` +
		`${fenced([
				...commitBindingLines(specification, unit),
			`${recorder} 'verify' '--commit' "$commit" '--strict'`,
			`report=${shellQuote(`.countershape/evidence/${lower}-final-`)}"$short".html`,
				`${preflightLine("report", unit.id)} "$report"`,
			`${recorder} 'verify' '--commit' "$commit" '--strict' '--html' "$report"`,
			`/usr/bin/shasum -a 256 "$report"`,
		])}\n\n` +
				`## Rotate the immutable ledger snapshot\n\n` +
				`The idempotent archive command classifies the live and commit-addressed slots, verifies same-device rename authority before mutation, resumes an admitted empty-root or completed state after interruption, rejects every ambiguous/no-overwrite state, then recomputes the archived ledger authority and requires the live slot to be absent.\n\n` +
				`${fenced([
					...commitBindingLines(specification, unit),
					`archive_root=${shellQuote(`.didrun-history/${lower}-final-`)}"$short"`,
					`${isolatedLine(specification, unit, "/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--rotate-final-ledger", unit.id)} "$archive_root"`,
				])}\n\n` +
				`## Recover a failed attempt without erasing evidence\n\n` +
				`Do not execute this section after a successful archive. Any nonzero command permanently ends its ledger. Choose exactly one command below. Each command is a fail-closed, same-device, no-overwrite state machine: it preclassifies the ledger and final-root topology before its first mutation, resumes admitted partial moves after interruption, and reaches the same terminal state when rerun. It never deletes, copies, reuses, or relabels evidence.\n\n` +
				`If process terminality cannot yet be established, stop normal progression and make no mutation. After wrapper and relevant-child terminality is established, any recorded overlap or indeterminate interval permanently disqualifies the ledger; execute only the matching recovery command below.\n\n` +
				`For a failure before the unit commit, replace the deliberately invalid slug with one unique UTC-and-reason value matching \`YYYYMMDDTHHMMSSZ-lowercase-reason\`, then rerun this same fence until it reports complete:\n\n` +
				`${fenced([
					`attempt_slug='REPLACE_WITH_YYYYMMDDTHHMMSSZ-lowercase-reason'`,
					`${recoveryCommand("--recover-failed-precommit", unit.id)} "$attempt_slug"`,
				])}\n\n` +
				`For a failure at or after commit—including seal, note inspection, strict verification, HTML generation, or normal archive rotation—replace the deliberately invalid value with the exact full 40-hex failed unit commit, then rerun this same fence until it reports complete:\n\n` +
				`${fenced([
					`failed_commit='REPLACE_WITH_EXACT_40_HEX_FAILED_COMMIT'`,
					`${recoveryCommand("--recover-failed-postcommit", unit.id)} "$failed_commit"`,
				])}\n\n` +
				`Postcommit recovery validates the failed commit independently of HEAD, normalizes either the live ledger or any partial/complete normal commit archive, moves the hermetic final root into its exact failed archive, creates or revalidates \`refs/countershape/failed-attempts/${lower}/<commit>\`, then compare-and-swaps HEAD from the failed commit to its declared parent while preserving the failed tree in the index. Any note and ledger stay tied to the failed ref and are never reused. The replacement unit commit therefore remains a direct child of its declared parent. Same-UID hostile races and filesystem or power-loss durability beyond the observed rename/ref transitions remain unreceipted.\n`;
}

function occurrenceCount(text, value) {
	return text.split(value).length - 1;
}

export async function selfTest() {
	const specification = await loadSpecification();
	for (const unit of specification.units) {
		const rendered = renderRunbook(specification, unit.id);
		const writerSentinel = "Exactly one root-owned writer owns all repository, didrun, claim, Git, seal, recovery, and archive mutation for this full attempt.";
		const terminalSentinel = "Do not enter the next fence or mutate candidate, evidence, or recovery state until any yielded didrun wrapper and every relevant child process are confirmed process-terminal.";
		const yieldedSentinel = "Empty or yielded app output is not terminal evidence.";
		const recoverySentinel = "If process terminality cannot yet be established, stop normal progression and make no mutation. After wrapper and relevant-child terminality is established, any recorded overlap or indeterminate interval permanently disqualifies the ledger; execute only the matching recovery command below.";
		const didrunRunToken = `${shellQuote("/opt/homebrew/bin/didrun")} ${shellQuote("run")}`;
		const didrunClaimToken = `${shellQuote("/opt/homebrew/bin/didrun")} ${shellQuote("claim")}`;
		const prepareCommand = recoveryLine(specification, "/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--verify-runbook-preflight", "prepare", unit.id);
		const forbiddenPrepareCommand = isolatedLine(specification, unit, "/opt/homebrew/bin/node", "tools/check-u7-plan.mjs", "--verify-runbook-preflight", "prepare", unit.id);
		const privateRootInstall = `/usr/bin/install -d -m 0700 ${[unit.final_root, ...["home", "tmp", "gotmp", "gocache", "gopath", "gomodcache"].map((name) => `${unit.final_root}/${name}`)].map((path) => shellQuote(resolve(repositoryRoot, path))).join(" ")}`;
		const recorderToken = commandLine(outerRecorderPrefix(specification, unit));
		const fallbackCommand = `${recorderToken} 'seal' '--commit' "$commit" '--allow-secrets'`;
		const fenceCount = unit.claims.length + 8;
		if (!rendered.startsWith(`# ${unit.id} deterministic final runbook\n`) || !rendered.includes(shellQuote(unit.subject)) ||
			!rendered.includes(unit.final_root) || rendered.includes("git add") || rendered.includes("rm -rf") ||
			occurrenceCount(rendered, didrunClaimToken) !== unit.claims.length || occurrenceCount(rendered, didrunRunToken) !== unit.claims.length + 1 ||
			occurrenceCount(rendered, recorderToken) !== unit.claims.length * 2 + 5 ||
			occurrenceCount(rendered, "'commit' '--no-verify' '--no-gpg-sign' '--cleanup=verbatim'") !== 1 || occurrenceCount(rendered, "--allow-secrets") !== 1 ||
			occurrenceCount(rendered, `# ${fallbackCommand}`) !== 1 || rendered.includes("MANUAL CHECKPOINT ONLY:") ||
			occurrenceCount(rendered, `${recorderToken} 'verify' '--commit' "$commit" '--strict'`) !== 2 ||
			occurrenceCount(rendered, "  cd -P --") !== fenceCount || occurrenceCount(rendered, "  umask 077") !== fenceCount ||
			occurrenceCount(rendered, "--verify-runbook-preflight' 'prepare'") !== 1 ||
			occurrenceCount(rendered, prepareCommand) !== 1 || rendered.includes(forbiddenPrepareCommand) ||
			occurrenceCount(rendered, privateRootInstall) !== 1 ||
			occurrenceCount(rendered, "--verify-runbook-preflight' 'commit'") !== 1 ||
			occurrenceCount(rendered, "--verify-runbook-preflight' 'report'") !== 1 ||
			occurrenceCount(rendered, "--verify-runbook-preflight' 'archive'") !== 0 ||
			occurrenceCount(rendered, "--verify-runbook-preflight' 'archive-complete'") !== 0 ||
			occurrenceCount(rendered, "--rotate-final-ledger") !== 1 ||
			occurrenceCount(rendered, "--recover-failed-precommit") !== 1 ||
			occurrenceCount(rendered, "--recover-failed-postcommit") !== 1 ||
			occurrenceCount(rendered, writerSentinel) !== 1 || occurrenceCount(rendered, terminalSentinel) !== 1 ||
			occurrenceCount(rendered, yieldedSentinel) !== 1 || occurrenceCount(rendered, recoverySentinel) !== 1 ||
			occurrenceCount(rendered, "attempt_slug='REPLACE_WITH_YYYYMMDDTHHMMSSZ-lowercase-reason'") !== 1 ||
			occurrenceCount(rendered, "failed_commit='REPLACE_WITH_EXACT_40_HEX_FAILED_COMMIT'") !== 1 ||
			rendered.includes("/bin/mv") || rendered.includes("/bin/cp") || rendered.includes("/bin/rm") || rendered.includes("test ! -e") ||
			!rendered.includes("NODE_OPTIONS=") || !rendered.includes("PYTHONNOUSERSITE=1") || !rendered.includes("GIT_CONFIG_GLOBAL=/dev/null") ||
			!rendered.includes("GIT_AUTHOR_NAME=\"$author_name\"") || rendered.includes("'reset' '--soft'") ||
			rendered.includes("'update-ref' \"$failed_ref\"") || rendered.includes("ledger_archive=") || rendered.includes("final_root_archive=")) fail("RENDER", unit.id);
		let cursor = 0;
		for (const [index, claim] of unit.claims.entries()) {
			const command = recorderLine(specification, unit, "run", "--", ...hermeticPrefix(unit), ...claim.command);
			const declaration = recorderLine(specification, unit, "claim", claim.type, "--label", claim.label, "--event", String(index), ...unit.allowed_paths.flatMap((path) => ["--path", path]));
			const commandIndex = rendered.indexOf(command, cursor);
			const claimIndex = rendered.indexOf(declaration, commandIndex + command.length);
			if (commandIndex < cursor || claimIndex < commandIndex) fail("ORDER", `${unit.id}:${claim.label}`);
			cursor = claimIndex + declaration.length;
		}
			if (rendered.indexOf("'commit' '--no-verify'", cursor) < cursor || rendered.indexOf("'seal' '--commit'", cursor) < cursor ||
			rendered.indexOf("'verify' '--commit'", cursor) < cursor) fail("SEAL_ORDER", unit.id);
		const recovery = rendered.indexOf("## Recover a failed attempt without erasing evidence", cursor);
		const precommit = rendered.indexOf("--recover-failed-precommit", recovery);
		const postcommit = rendered.indexOf("--recover-failed-postcommit", precommit);
		if (!(recovery > cursor && precommit > recovery && postcommit > precommit)) {
			fail("RECOVERY_ORDER", unit.id);
		}
	}
	if (shellQuote("a'b") !== `'a'"'"'b'`) fail("SHELL_QUOTE", shellQuote("a'b"));
	console.log(`U7 final runbook renderer defensive self-test passed: units=${specification.units.length} exact command/claim manifests, isolated canonical-root fences, no-follow preparation/archive gates, recoverable plain-seal checkpoint, idempotent same-device failed-attempt recovery, explicit-commit strict/HTML closure, and shell quoting`);
}

async function main() {
	if (isDeepStrictEqual(process.argv.slice(2), ["--self-test"])) return selfTest();
	if (process.argv.length === 4 && process.argv[2] === "--unit") {
		const specification = await loadSpecification();
		process.stdout.write(`${renderRunbook(specification, process.argv[3])}\n`);
		return;
	}
	fail("USAGE", "print-u7-final-runbook.mjs --unit <unit> | --self-test");
}

async function dispatch() {
	if (process.argv[1] === undefined) return;
	const requested = resolve(process.argv[1]);
	let actual;
	try { actual = await realpath(requested); } catch { return; }
	const canonical = fileURLToPath(import.meta.url);
	if (actual !== canonical) return;
	if (requested !== canonical) fail("NONCANONICAL_ENTRY", requested);
	await main();
}

dispatch().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
