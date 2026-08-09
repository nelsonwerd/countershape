#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { chmod, copyFile, cp, link, mkdir, mkdtemp, readFile, realpath, rm, symlink, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

import { collectEvidence, parseTimeReport, protocolAuthority, validateGlobalStudies, validateProductResult } from "./check-u7-study-harness.mjs";

const modulePath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(modulePath), "..");
const checkerRelative = "tools/check-u7-study-harness.mjs";
const checkerPath = resolve(repositoryRoot, checkerRelative);
const nodePath = "/opt/homebrew/bin/node";
const goPath = "/opt/homebrew/bin/go";
const gitPath = "/usr/bin/git";
const timePath = "/usr/bin/time";
const timeDigest = "d2210b72e8c978748a0f0ac7d0819dda4dd411bb7a2e9730476038b0d695e7a2";
const nodeDigest = "87989003817c5347d6bad48e46897e1b6509328bb7e8295d83cb6b40af836b9c";
const goDigest = "3f947495f00cb7f8088a5cfd694da8dc43869b33f5e7377b048fb18922ffb7e0";
const harnessProtocol = "countershape/u7-study-harness/v1";
const productResultSchema = "countershape/u7-study-domain-result/v1";
const trialSchema = "countershape/u7-study-trial/v1";
const artifactSchema = "countershape/u7-study-artifact/v1";
const observationAuthority = "U7P_FROZEN_HARNESS_DIRECT_PROCESS_GIT_AND_ARTIFACT_OBSERVATION";
const semanticCeiling = "ARTIFACT_BYTES_AND_SUBJECT_PROCESS_TOPOLOGY_OBSERVED_PRODUCT_SEMANTICS_AND_FULL_HARNESS_RESOURCES_NOT_INDEPENDENTLY_ESTABLISHED";
const driverProtocol = "FIXTURE_ONLY_NO_EVIDENCE_ROOT";
const phaseVerdicts = Object.freeze({ U7B: "LOCAL_HTTP_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED", U7C: "LOCAL_CLI_REFERENCE_FUNCTIONAL_GREEN_SUBJECT_RESOURCE_OBSERVED", U7D: "LOCAL_REFERENCE_MILESTONE_FUNCTIONAL_GREEN_FULL_STUDY_RESOURCE_UNRECEIPTED" });

export const requiredCaseIDs = Object.freeze([
	"live-protocol",
	"clean-u7b",
	"clean-u7c",
	"clean-u7d",
	"args-missing",
	"args-u7a",
	"args-trailing",
	"wrong-tmpdir",
	"repeat-root",
	"driver-stdout",
	"driver-stderr",
	"driver-nonzero",
	"driver-mode",
	"driver-evidence-env-scrub",
	"driver-prepopulation",
	"fixture-external-gitdir",
	"fixture-shallow",
	"fixture-graft",
	"fixture-history",
	"fixture-replace-ref",
	"fixture-assume-unchanged",
	"fixture-skip-worktree",
	"fixture-sparse",
	"fixture-ignored-input",
	"product-malformed",
	"product-stderr",
	"product-nonzero",
	"product-created-evidence-root",
	"cross-domain-fixture-alias",
	"cross-domain-fresh-alias",
	"runtime-node-digest",
	"runtime-go-digest",
	"runtime-git-digest",
	"time-positive",
	"time-cr",
	"time-bom",
	"time-no-newline",
	"time-missing-row",
	"time-extra-row",
	"time-reordered-row",
	"time-bad-real",
	"time-zero-rss",
	"time-duplicate-rss",
	"product-result-schema",
	"product-result-domain",
	"product-result-extra",
	"evidence-missing",
	"evidence-extra",
	"evidence-symlink",
	"evidence-hardlink",
	"evidence-mode",
	"evidence-noncanonical",
	"evidence-wrong-trial",
	"output-root-extra",
	"output-harness-sha",
	"output-phase",
	"output-verdict",
	"output-process-argv",
	"output-prepare-executable-digest",
	"output-compile-executable-digest",
	"output-execute-executable-digest",
	"output-driver-digest",
	"output-environment-digest",
	"output-time-digest",
	"output-manifest-path",
	"output-manifest-digest",
	"output-fresh-binding",
	"output-phase-trial",
	"output-fixture-gitdir",
	"output-aggregate-wall",
]);
const requiredCaseDigest = "b84c42cbe93af59387c4fe0395ff474e8581124b8e84ab14f95201b99ce3c10a";

function fail(code, detail) {
	throw new Error(`U7_STUDY_HARNESS_SELFTEST_${code}: ${detail}`);
}

function sha256(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

const checkerDigest = sha256(await readFile(checkerPath));

function canonicalJSONLine(value) {
	return Buffer.from(`${JSON.stringify(value)}\n`, "utf8");
}

function exactKeys(value, keys) {
	return value !== null && typeof value === "object" && !Array.isArray(value) &&
		isDeepStrictEqual(Object.keys(value).sort(), [...keys].sort());
}

function validDigest(value) {
	return typeof value === "string" && /^[0-9a-f]{64}$/u.test(value) && !/^0+$/u.test(value);
}

function positiveInteger(value, maximum = Number.MAX_SAFE_INTEGER) {
	return Number.isSafeInteger(value) && value > 0 && value <= maximum;
}

function byteCompare(left, right) {
	return Buffer.compare(Buffer.from(left, "utf8"), Buffer.from(right, "utf8"));
}

function expectedManifestPaths() {
	const paths = [...Object.values(protocolAuthority.deterministic_artifact_paths), ...Object.values(protocolAuthority.fresh_artifact_paths)];
	for (const phase of protocolAuthority.phase_budgets.slice(2)) {
		for (let ordinal = 1; ordinal <= phase.per_run_trials; ordinal += 1) paths.push(`phases/${phase.id}/trial-${String(ordinal).padStart(3, "0")}.json`);
	}
	return paths.sort(byteCompare);
}

const manifestPaths = Object.freeze(expectedManifestPaths());
const emptyDigest = sha256(Buffer.alloc(0));

function expectedChildEnvironment(repository, phase, run, study, name) {
	const finalTmp = resolve(repository, `.countershape/${phase.toLowerCase()}-final/tmp`);
	const runRoot = dirname(run.execution.cwd);
	const environment = {
		...cleanEnvironment(finalTmp),
		HOME: resolve(runRoot, "home"),
		TMPDIR: resolve(runRoot, "tmp"),
		GOTMPDIR: resolve(repository, ".cache-gotmp"),
		GOCACHE: resolve(repository, ".cache-gocache"),
		GOPATH: resolve(repository, ".cache-gopath"),
		GOMODCACHE: resolve(repository, ".cache-gomodcache"),
		COUNTERSHAPE_STUDY_DOMAIN: study.id,
		COUNTERSHAPE_STUDY_ORDINAL: String(run.ordinal),
		GIT_CONFIG_NOSYSTEM: "1",
		GIT_CONFIG_GLOBAL: "/dev/null",
		GIT_NO_LAZY_FETCH: "1",
		GIT_OPTIONAL_LOCKS: "0",
		GIT_TERMINAL_PROMPT: "0",
		GIT_NO_REPLACE_OBJECTS: "1",
	};
	if (name === "execute") environment.COUNTERSHAPE_EVIDENCE_ROOT = resolve(runRoot, "evidence");
	const rows = Object.keys(environment).sort(byteCompare).map((key) => `${key}=${environment[key]}`);
	return sha256(Buffer.from(`${rows.join("\n")}\n`, "utf8"));
}

function validateProcessObservation(observation, name, run, study, phase, repository) {
	const keys = ["name", "observer_argv", "cwd", "executable_path", "executable_sha256", "observer_path", "observer_sha256", "environment_sha256", "wall_time_ms", "peak_rss_bytes", "time_report_path", "time_report_bytes", "time_report_sha256", "time_report", "stdout_bytes", "stdout_sha256"];
	if (!exactKeys(observation, keys) || observation.name !== name || observation.observer_path !== timePath || observation.observer_sha256 !== timeDigest ||
		!validDigest(observation.environment_sha256) || !positiveInteger(observation.wall_time_ms, 1_200_000) || !positiveInteger(observation.peak_rss_bytes, 16 * 1024 ** 3)) fail("OUTPUT", `${study.id}:${run.ordinal}:${name}:shape`);
	const reportBytes = Buffer.from(observation.time_report, "utf8");
	const parsed = parseTimeReport(reportBytes);
	if (observation.time_report_bytes !== reportBytes.length || observation.time_report_sha256 !== sha256(reportBytes) ||
		observation.wall_time_ms !== parsed.wall_time_ms || observation.peak_rss_bytes !== parsed.peak_rss_bytes) fail("OUTPUT", `${study.id}:${run.ordinal}:${name}:time`);
	const runRoot = dirname(run.execution.cwd);
	const reportPath = resolve(runRoot, "observations", `${name}.time`);
	if (resolve(repository, observation.time_report_path) !== reportPath || !validDigest(observation.executable_sha256)) fail("OUTPUT", `${study.id}:${run.ordinal}:${name}:paths`);
	let cwd; let executable; let executableSha256; let args; let expectedStdout;
	if (name === "prepare") {
		cwd = repository;
		executable = nodePath;
		executableSha256 = nodeDigest;
		args = [resolve(repository, run.driver_authority.path), "--prepare", "--domain", study.id, "--ordinal", String(run.ordinal), "--fixture-root", run.execution.cwd];
		expectedStdout = Buffer.alloc(0);
	} else if (name === "compile") {
		cwd = repository;
		executable = goPath;
		executableSha256 = goDigest;
		args = ["build", "-trimpath", "-mod=readonly", "-buildvcs=false", "-o", run.execution.executable_path, "./cmd/countershape"];
		expectedStdout = Buffer.alloc(0);
	} else {
		cwd = run.execution.cwd;
		executable = run.execution.executable_path;
		executableSha256 = run.execution.executable_sha256;
		args = ["study", study.id, "--json"];
		expectedStdout = canonicalJSONLine(run.product_result);
	}
	const expectedArgv = [timePath, "-p", "-l", "-o", reportPath, executable, ...args];
	if (observation.cwd !== cwd || observation.executable_path !== executable || observation.executable_sha256 !== executableSha256 ||
		observation.environment_sha256 !== expectedChildEnvironment(repository, phase, run, study, name) || !isDeepStrictEqual(observation.observer_argv, expectedArgv) ||
		observation.stdout_bytes !== expectedStdout.length || observation.stdout_sha256 !== sha256(expectedStdout)) fail("OUTPUT", `${study.id}:${run.ordinal}:${name}:process`);
}

function validateHarnessEvidence(value, phase, rawLine, repository, driverDigests) {
	if (!canonicalJSONLine(value).equals(Buffer.from(rawLine, "utf8"))) fail("OUTPUT", `${phase}:canonical`);
	if (!exactKeys(value, ["schema_version", "phase", "harness_authority", "environment", "studies", "milestone_verdict", "honest_fallback", "unreceipted"]) ||
		value.schema_version !== "countershape/u7-study-evidence/v1" || value.phase !== phase || value.milestone_verdict !== phaseVerdicts[phase] ||
		value.honest_fallback !== "NOT_APPLICABLE_SOURCE_GREEN" || !isDeepStrictEqual(value.unreceipted, ["SYNTHETIC_SELFTEST_ONLY"])) fail("OUTPUT", `${phase}:root`);
	const authority = value.harness_authority;
	if (!exactKeys(authority, ["protocol", "path", "sha256", "protocol_sha256", "observation_authority", "semantic_ceiling"]) ||
		authority.protocol !== harnessProtocol || authority.path !== checkerRelative || authority.sha256 !== checkerDigest || authority.protocol_sha256 !== sha256(canonicalJSONLine(protocolAuthority)) ||
		authority.observation_authority !== observationAuthority || authority.semantic_ceiling !== semanticCeiling) fail("OUTPUT", `${phase}:harness`);
	const environment = value.environment;
	if (!exactKeys(environment, ["platform", "kernel_release", "architecture", "git_version", "go_version", "node_version", "cpu_model", "logical_cpu_count", "memory_bytes", "runtime_authority_sha256"]) ||
		environment.platform !== "darwin" || environment.architecture !== "arm64" || environment.git_version !== "git version 2.50.1 (Apple Git-155)" ||
		environment.go_version !== "go version go1.26.5 darwin/arm64" || environment.node_version !== "v25.2.1" || !validDigest(environment.runtime_authority_sha256) ||
		!positiveInteger(environment.logical_cpu_count, 512) || !positiveInteger(environment.memory_bytes, 2 ** 50)) fail("OUTPUT", `${phase}:environment`);
	const expectedDomains = phase === "U7B" ? ["http"] : phase === "U7C" ? ["cli"] : ["http", "cli"];
	if (!Array.isArray(value.studies) || !isDeepStrictEqual(value.studies.map((study) => study?.id), expectedDomains)) fail("OUTPUT", `${phase}:domains`);
	for (const study of value.studies) {
		const studyKeys = ["id", "fixture_commit", "fixture_authority", "logical_product_command", "run_count", "trial_count", "budget_trial_count", "observed_subject_process_wall_time_ms", "budget_subject_process_wall_time_ms", "observed_subject_process_peak_rss_bytes", "budget_subject_process_peak_rss_bytes", "observation_authority", "phases", "deterministic_artifacts", "runs"];
		if (!exactKeys(study, studyKeys) || !/^[0-9a-f]{40}$/u.test(study.fixture_commit) || study.run_count !== 3 || study.trial_count !== 300 || study.budget_trial_count !== 300 ||
			study.observation_authority !== observationAuthority || !isDeepStrictEqual(study.logical_product_command, ["countershape", "study", study.id, "--json"]) ||
			!positiveInteger(study.observed_subject_process_wall_time_ms, study.budget_subject_process_wall_time_ms) || !positiveInteger(study.observed_subject_process_peak_rss_bytes, study.budget_subject_process_peak_rss_bytes) ||
			!Array.isArray(study.runs) || study.runs.length !== 3) fail("OUTPUT", `${phase}:${study.id}:study`);
		const expectedStudyPhases = protocolAuthority.phase_budgets.map((row) => ({ id: row.id, trial_count: row.per_run_trials * 3, trial_budget: row.per_run_trials * 3 }));
		if (!isDeepStrictEqual(study.phases, expectedStudyPhases) || !exactKeys(study.deterministic_artifacts, [...Object.keys(protocolAuthority.deterministic_artifact_paths), "reference_binary_sha256"])) fail("OUTPUT", `${phase}:${study.id}:phases`);
		for (const digest of Object.values(study.deterministic_artifacts)) if (!validDigest(digest)) fail("OUTPUT", `${phase}:${study.id}:deterministic`);
		const fixtureAuthority = study.fixture_authority;
		if (!exactKeys(fixtureAuthority, ["head_commit", "tree", "clean_status_bytes", "clean_status_sha256", "repository_count", "fresh_repository_per_run", "repository_git_directories"]) ||
			fixtureAuthority.head_commit !== study.fixture_commit || !/^[0-9a-f]{40}$/u.test(fixtureAuthority.tree) || fixtureAuthority.clean_status_bytes !== 0 ||
			fixtureAuthority.clean_status_sha256 !== emptyDigest || fixtureAuthority.repository_count !== 3 || fixtureAuthority.fresh_repository_per_run !== true ||
			!isDeepStrictEqual(fixtureAuthority.repository_git_directories, study.runs.map((run) => run.fixture.git_directory))) fail("OUTPUT", `${phase}:${study.id}:fixture-authority`);
		let studyWall = 0; let studyPeak = 0;
		for (const [index, run] of study.runs.entries()) {
			const runKeys = ["ordinal", "trial_count", "trial_budget", "observed_subject_process_wall_time_ms", "budget_subject_process_wall_time_ms", "observed_subject_process_peak_rss_bytes", "budget_subject_process_peak_rss_bytes", "observation_authority", "fixture", "driver_authority", "execution", "processes", "phases", "product_result", "evidence_root", "evidence_manifest", "evidence_manifest_sha256", "deterministic_artifacts", ...Object.keys(protocolAuthority.fresh_artifact_paths), "fixture_invocation_sha256"];
			if (!exactKeys(run, runKeys) || run.ordinal !== index + 1 || run.trial_count !== 100 || run.trial_budget !== 100 || run.observation_authority !== observationAuthority ||
				!positiveInteger(run.observed_subject_process_wall_time_ms, run.budget_subject_process_wall_time_ms) || !positiveInteger(run.observed_subject_process_peak_rss_bytes, run.budget_subject_process_peak_rss_bytes) ||
				!isDeepStrictEqual(run.deterministic_artifacts, study.deterministic_artifacts)) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:run`);
			const expectedRunPhases = protocolAuthority.phase_budgets.map((row) => ({ id: row.id, trial_count: row.per_run_trials, trial_budget: row.per_run_trials }));
			if (!isDeepStrictEqual(run.phases, expectedRunPhases) || !exactKeys(run.product_result, ["schema_version", "domain", "ordinal", "status"]) ||
				run.product_result.schema_version !== productResultSchema || run.product_result.domain !== study.id || run.product_result.ordinal !== run.ordinal || run.product_result.status !== "GREEN") fail("OUTPUT", `${phase}:${study.id}:${index + 1}:result`);
			const runRoot = resolve(repository, `.countershape/${phase.toLowerCase()}-final/tmp/studies`, study.id, `run-${run.ordinal}`);
			const expectedFixture = resolve(runRoot, "fixture");
			const expectedBinary = resolve(run.execution.cwd, "countershape");
			if (!exactKeys(run.driver_authority, ["path", "sha256"]) || run.driver_authority.path !== `tools/run-u7-${study.id}-study.mjs` || run.driver_authority.sha256 !== driverDigests[study.id] ||
				run.execution.cwd !== expectedFixture || !isDeepStrictEqual(run.execution, { cwd: expectedFixture, argv: [expectedBinary, "study", study.id, "--json"], executable_path: expectedBinary, executable_sha256: study.deterministic_artifacts.reference_binary_sha256 }) ||
				run.fixture_invocation_sha256 !== sha256(canonicalJSONLine(run.execution)) || run.evidence_root !== relative(repository, resolve(runRoot, "evidence"))) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:execution`);
			if (!exactKeys(run.fixture, ["head_commit", "tree", "commit_count", "git_directory", "git_common_directory", "git_alternates", "clean_status_bytes", "clean_status_sha256"]) ||
				run.fixture.head_commit !== study.fixture_commit || run.fixture.tree !== fixtureAuthority.tree || run.fixture.commit_count !== 1 ||
				run.fixture.git_directory !== relative(repository, resolve(run.execution.cwd, ".git")) || run.fixture.git_common_directory !== run.fixture.git_directory ||
				run.fixture.git_alternates !== "ABSENT" || run.fixture.clean_status_bytes !== 0 || run.fixture.clean_status_sha256 !== emptyDigest) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:fixture`);
			if (!exactKeys(run.processes, ["prepare", "compile", "execute"])) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:processes`);
			for (const name of ["prepare", "compile", "execute"]) validateProcessObservation(run.processes[name], name, run, study, phase, repository);
			const processWall = run.processes.prepare.wall_time_ms + run.processes.compile.wall_time_ms + run.processes.execute.wall_time_ms;
			const processPeak = Math.max(run.processes.prepare.peak_rss_bytes, run.processes.compile.peak_rss_bytes, run.processes.execute.peak_rss_bytes);
			if (run.observed_subject_process_wall_time_ms !== processWall || run.observed_subject_process_peak_rss_bytes !== processPeak || !Array.isArray(run.evidence_manifest) ||
				!isDeepStrictEqual(run.evidence_manifest.map((row) => row?.path), manifestPaths)) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:totals`);
			for (const row of run.evidence_manifest) if (!exactKeys(row, ["path", "mode", "bytes", "sha256"]) || row.mode !== "0600" || !positiveInteger(row.bytes, 16 * 1024 * 1024) || !validDigest(row.sha256)) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:manifest`);
			if (run.evidence_manifest_sha256 !== sha256(canonicalJSONLine(run.evidence_manifest))) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:manifest-digest`);
			const manifest = new Map(run.evidence_manifest.map((row) => [row.path, row.sha256]));
			for (const [field, path] of Object.entries(protocolAuthority.deterministic_artifact_paths)) if (run.deterministic_artifacts[field] !== manifest.get(path)) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:deterministic-binding`);
			for (const [field, path] of Object.entries(protocolAuthority.fresh_artifact_paths)) if (run[field] !== manifest.get(path)) fail("OUTPUT", `${phase}:${study.id}:${index + 1}:fresh-binding`);
			studyWall += run.observed_subject_process_wall_time_ms; studyPeak = Math.max(studyPeak, run.observed_subject_process_peak_rss_bytes);
		}
		if (study.observed_subject_process_wall_time_ms !== studyWall || study.observed_subject_process_peak_rss_bytes !== studyPeak) fail("OUTPUT", `${phase}:${study.id}:aggregate`);
	}
	validateGlobalStudies(value.studies);
	return value;
}

function cleanEnvironment(tmp) {
	return {
		PATH: "/usr/bin:/bin:/opt/homebrew/bin",
		LANG: "C",
		LC_ALL: "C",
		TZ: "UTC",
		NO_COLOR: "1",
		NODE_OPTIONS: "",
		NODE_PATH: "",
		HOME: join(tmp, "home"),
		TMPDIR: tmp,
		GOTMPDIR: join(tmp, "gotmp"),
		GOCACHE: join(tmp, "gocache"),
		GOPATH: join(tmp, "gopath"),
		GOMODCACHE: join(tmp, "gomodcache"),
		GOFLAGS: "-mod=readonly -buildvcs=false -p=1",
		GOMAXPROCS: "2",
		CGO_ENABLED: "1",
		COUNTERSHAPE_NODE: nodePath,
		COUNTERSHAPE_GO: goPath,
		COUNTERSHAPE_GIT: gitPath,
		COUNTERSHAPE_SH: "/bin/sh",
		COUNTERSHAPE_GOFMT: "/opt/homebrew/bin/gofmt",
		COUNTERSHAPE_CC: "/usr/bin/clang",
		COUNTERSHAPE_CXX: "/usr/bin/clang++",
		CC: "/usr/bin/clang",
		CXX: "/usr/bin/clang++",
	};
}

function invoke(checker, args, cwd, env, timeout = 120_000) {
	const result = spawnSync(nodePath, [checker, ...args], {
		cwd,
		env,
		encoding: "utf8",
		timeout,
		maxBuffer: 4 * 1024 * 1024,
	});
	if (result.error) fail("CHILD", `${result.error.code ?? result.error.message}`);
	if (result.signal !== null) fail("CHILD", `signal ${result.signal}`);
	return Object.freeze({ status: result.status, stdout: result.stdout, stderr: result.stderr });
}

function assertResult(id, result, status, stdout, stderr) {
	if (result.status !== status || result.stdout !== stdout || result.stderr !== stderr) {
		fail("RESULT", `${id}: status=${result.status} stdout=${JSON.stringify(result.stdout)} stderr=${JSON.stringify(result.stderr)}`);
	}
}

const fakeDriverBase = String.raw`#!/usr/bin/env node
import { spawnSync } from "node:child_process";
import { chmod, lstat, mkdir, rename, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";

function value(flag) {
  const at = process.argv.indexOf(flag);
  if (at < 0 || at + 1 >= process.argv.length) throw new Error(flag);
  return process.argv[at + 1];
}
if (process.argv.length !== 9 || process.argv[2] !== "--prepare") throw new Error("usage");
const domain = value("--domain");
const fixture = value("--fixture-root");
if (!['http','cli'].includes(domain) || value("--ordinal") === "") throw new Error("arguments");
if (process.env.COUNTERSHAPE_EVIDENCE_ROOT !== undefined || process.argv.some((value) => value.includes("evidence"))) throw new Error("evidence authority exposed to fixture driver");
const siblingEvidence = resolve(dirname(fixture), "evidence");
try { await lstat(siblingEvidence); throw new Error("evidence root existed during fixture preparation"); }
catch (error) { if (error.code !== "ENOENT") throw error; }
await mkdir(fixture, { mode: 0o700 });
await chmod(fixture, 0o700);
await writeFile(new URL("file:" + fixture + "/.gitignore"), "/countershape\n/.countershape/\n*.ignored\n", { mode: 0o600 });
await writeFile(new URL("file:" + fixture + "/README.md"), domain + " fixture\n", { mode: 0o600 });
const gitEnv = { ...process.env, GIT_AUTHOR_NAME: "U7 Fixture", GIT_AUTHOR_EMAIL: "u7@example.invalid", GIT_AUTHOR_DATE: "2001-01-01T00:00:00Z", GIT_COMMITTER_NAME: "U7 Fixture", GIT_COMMITTER_EMAIL: "u7@example.invalid", GIT_COMMITTER_DATE: "2001-01-01T00:00:00Z" };
for (const args of [["init","-q"],["add","."],["-c","commit.gpgsign=false","-c","core.hooksPath=/dev/null","commit","-q","-m","fixture"]]) {
  const result = spawnSync("/usr/bin/git", args, { cwd: fixture, env: gitEnv, encoding: "utf8" });
  if (result.status !== 0 || result.stderr !== "") throw new Error("git " + args[0] + ":" + result.stderr);
}
`;

function fakeDriverSource(mutation) {
	if (mutation === "driver-stdout") return `${fakeDriverBase}\nprocess.stdout.write("noise\\n");\n`;
	if (mutation === "driver-stderr") return `${fakeDriverBase}\nprocess.stderr.write("noise\\n");\n`;
	if (mutation === "driver-nonzero") return `${fakeDriverBase}\nprocess.exit(7);\n`;
	if (mutation === "driver-prepopulation") return `${fakeDriverBase}\nawait mkdir(siblingEvidence, { recursive: true, mode: 0o700 });\n`;
	if (mutation === "fixture-external-gitdir") return `${fakeDriverBase}\nawait rename(fixture + "/.git", fixture + "-external.git");\nawait writeFile(fixture + "/.git", "gitdir: " + fixture + "-external.git\\n", { mode: 0o600 });\n`;
	if (mutation === "fixture-shallow") return `${fakeDriverBase}\n{ const head = spawnSync("/usr/bin/git", ["rev-parse","HEAD"], { cwd: fixture, env: gitEnv, encoding: "utf8" }).stdout.trim(); await writeFile(fixture + "/.git/shallow", head + "\\n", { mode: 0o600 }); }\n`;
	if (mutation === "fixture-graft") return `${fakeDriverBase}\nawait mkdir(fixture + "/.git/info", { recursive: true, mode: 0o700 });\nawait writeFile(fixture + "/.git/info/grafts", "0000000000000000000000000000000000000000\\n", { mode: 0o600 });\n`;
	if (mutation === "fixture-history") return `${fakeDriverBase}\nawait writeFile(fixture + "/README.md", domain + " fixture second commit\\n", { mode: 0o600 });\nfor (const args of [["add","README.md"],["-c","commit.gpgsign=false","-c","core.hooksPath=/dev/null","commit","-q","-m","second"]]) { const result = spawnSync("/usr/bin/git", args, { cwd: fixture, env: gitEnv, encoding: "utf8" }); if (result.status !== 0) throw new Error("history mutation"); }\n`;
	if (mutation === "fixture-replace-ref") return `${fakeDriverBase}\n{ const head = spawnSync("/usr/bin/git", ["rev-parse","HEAD"], { cwd: fixture, env: gitEnv, encoding: "utf8" }).stdout.trim(); await mkdir(fixture + "/.git/refs/replace", { recursive: true, mode: 0o700 }); await writeFile(fixture + "/.git/refs/replace/" + head, head + "\\n", { mode: 0o600 }); }\n`;
	if (mutation === "fixture-assume-unchanged") return `${fakeDriverBase}\nspawnSync("/usr/bin/git", ["update-index","--assume-unchanged","README.md"], { cwd: fixture, env: gitEnv });\nawait writeFile(fixture + "/README.md", "concealed assume-unchanged\\n", { mode: 0o600 });\n`;
	if (mutation === "fixture-skip-worktree") return `${fakeDriverBase}\nspawnSync("/usr/bin/git", ["update-index","--skip-worktree","README.md"], { cwd: fixture, env: gitEnv });\nawait writeFile(fixture + "/README.md", "concealed skip-worktree\\n", { mode: 0o600 });\n`;
	if (mutation === "fixture-sparse") return `${fakeDriverBase}\nawait writeFile(fixture + "/.git/info/sparse-checkout", "README.md\\n", { mode: 0o600 });\n`;
	if (mutation === "fixture-ignored-input") return `${fakeDriverBase}\nawait writeFile(fixture + "/behavior.ignored", "hidden input\\n", { mode: 0o600 });\n`;
	if (mutation === "cross-domain-fixture-alias") return fakeDriverBase.replace('domain + " fixture\\n"', '"shared fixture\\n"');
	return fakeDriverBase;
}

const fakeGoBase = String.raw`package main

import (
  "encoding/json"
  "fmt"
  "os"
  "path/filepath"
  "strconv"
)

type trial struct { Schema string ` + "`json:\"schema_version\"`" + `; Domain string ` + "`json:\"domain\"`" + `; Ordinal int ` + "`json:\"ordinal\"`" + `; Phase string ` + "`json:\"phase\"`" + `; Trial int ` + "`json:\"trial\"`" + `; Status string ` + "`json:\"status\"`" + ` }
type payload struct { Value string ` + "`json:\"value\"`" + ` }
type artifact struct { Schema string ` + "`json:\"schema_version\"`" + `; Domain string ` + "`json:\"domain\"`" + `; Ordinal *int ` + "`json:\"ordinal,omitempty\"`" + `; Artifact string ` + "`json:\"artifact\"`" + `; Payload payload ` + "`json:\"payload\"`" + ` }
type result struct { Schema string ` + "`json:\"schema_version\"`" + `; Domain string ` + "`json:\"domain\"`" + `; Ordinal int ` + "`json:\"ordinal\"`" + `; Status string ` + "`json:\"status\"`" + ` }

func write(path string, value any) {
  if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil { panic(err) }
  bytes, err := json.Marshal(value); if err != nil { panic(err) }
  bytes = append(bytes, byte(10))
  if err := os.WriteFile(path, bytes, 0600); err != nil { panic(err) }
}

func main() {
  if len(os.Args) != 4 || os.Args[1] != "study" || os.Args[3] != "--json" { os.Exit(2) }
  domain := os.Args[2]
  ordinal, err := strconv.Atoi(os.Getenv("COUNTERSHAPE_STUDY_ORDINAL")); if err != nil { panic(err) }
  root := os.Getenv("COUNTERSHAPE_EVIDENCE_ROOT")
  info, err := os.Lstat(root); if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 { panic("harness-owned evidence root required") }
  entries, err := os.ReadDir(root); if err != nil || len(entries) != 0 { panic("empty evidence root required") }
  phases := []struct{Name string; Count int}{{"search",80},{"confirm",8},{"contract",10}}
  for _, phase := range phases { for index := 1; index <= phase.Count; index++ { write(filepath.Join(root,"phases",phase.Name,fmt.Sprintf("trial-%03d.json",index)), trial{"countershape/u7-study-trial/v1",domain,ordinal,phase.Name,index,"GREEN"}) } }
  deterministic := map[string]string{"source_spec_sha256":"source-spec.json","world_plan_sha256":"world-plan.json","ruling_sha256":"ruling.json","decision_record_sha256":"decision-record.json","contract_bundle_sha256":"contract-bundle.json"}
  for field, path := range deterministic { write(filepath.Join(root,"deterministic",path), artifact{"countershape/u7-study-artifact/v1",domain,nil,field,payload{domain+":"+field}}) }
  fresh := map[string]string{"world_instance_sha256":"world-instance.json","attempts_sha256":"attempts.json","measurements_sha256":"measurements.json","captures_sha256":"captures.json","confirmation_sha256":"confirmation.json","contract_execution_target_sha256":"contract-execution-target.json","finalized_contract_run_sha256":"finalized-contract-run.json","contract_execution_sha256":"contract-execution.json"}
  for field, path := range fresh { copy := ordinal; write(filepath.Join(root,"fresh",path), artifact{"countershape/u7-study-artifact/v1",domain,&copy,field,payload{fmt.Sprintf("%s:%d:%s",domain,ordinal,field)}}) }
  bytes, _ := json.Marshal(result{"countershape/u7-study-domain-result/v1",domain,ordinal,"GREEN"})
  os.Stdout.Write(append(bytes,byte(10)))
}
`;

function fakeGoSource(mutation) {
	if (mutation === "product-malformed") return fakeGoBase.replace("bytes, _ := json.Marshal(result", "fmt.Print(\"{bad\\n\"); return; bytes, _ := json.Marshal(result");
	if (mutation === "product-stderr") return fakeGoBase.replace("bytes, _ := json.Marshal(result", "fmt.Fprint(os.Stderr, \"noise\\n\"); bytes, _ := json.Marshal(result");
	if (mutation === "product-nonzero") return fakeGoBase.replace("bytes, _ := json.Marshal(result", "os.Exit(7); bytes, _ := json.Marshal(result");
	if (mutation === "product-created-evidence-root") return fakeGoBase.replace('root := os.Getenv("COUNTERSHAPE_EVIDENCE_ROOT")', 'root := os.Getenv("COUNTERSHAPE_EVIDENCE_ROOT"); if err := os.Remove(root); err != nil { panic(err) }; if err := os.Mkdir(root, 0700); err != nil { panic(err) }');
	return fakeGoBase;
}

async function writeSyntheticSpecification(root, harnessDigest) {
	const studyHarness = {
		protocol: harnessProtocol,
		protocol_sha256: sha256(canonicalJSONLine(protocolAuthority)),
		path: checkerRelative,
		sha256: harnessDigest,
		product_result_schema: productResultSchema,
		trial_schema: trialSchema,
		artifact_schema: artifactSchema,
		observation_authority: observationAuthority,
		semantic_ceiling: semanticCeiling,
		driver_protocol: driverProtocol,
		phase_verdicts: phaseVerdicts,
		observer_prefix: [timePath, "-p", "-l", "-o"],
		driver_by_domain: { http: "tools/run-u7-http-study.mjs", cli: "tools/run-u7-cli-study.mjs" },
		phase_budgets: [
			{ id: "fixture", per_run_trials: 1 }, { id: "compile", per_run_trials: 1 }, { id: "search", per_run_trials: 80 },
			{ id: "confirm", per_run_trials: 8 }, { id: "contract", per_run_trials: 10 },
		],
		deterministic_artifact_paths: { source_spec_sha256: "deterministic/source-spec.json", world_plan_sha256: "deterministic/world-plan.json", ruling_sha256: "deterministic/ruling.json", decision_record_sha256: "deterministic/decision-record.json", contract_bundle_sha256: "deterministic/contract-bundle.json" },
		fresh_artifact_paths: { world_instance_sha256: "fresh/world-instance.json", attempts_sha256: "fresh/attempts.json", measurements_sha256: "fresh/measurements.json", captures_sha256: "fresh/captures.json", confirmation_sha256: "fresh/confirmation.json", contract_execution_target_sha256: "fresh/contract-execution-target.json", finalized_contract_run_sha256: "fresh/finalized-contract-run.json", contract_execution_sha256: "fresh/contract-execution.json" },
	};
	const specification = {
		runtime_authority: {
			admitted_tools: [
				{ name: "git", path: gitPath, realpath: gitPath, sha256: "179301dcb41ea78accc3fa0048a7e6f6710d891945a751a34addd622020c1818", version: "git version 2.50.1 (Apple Git-155)" },
				{ name: "go", path: goPath, realpath: "/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go", sha256: "3f947495f00cb7f8088a5cfd694da8dc43869b33f5e7377b048fb18922ffb7e0", version: "go version go1.26.5 darwin/arm64" },
				{ name: "node", path: nodePath, realpath: "/opt/homebrew/Cellar/node/25.2.1/bin/node", sha256: "87989003817c5347d6bad48e46897e1b6509328bb7e8295d83cb6b40af836b9c", version: "v25.2.1" },
				{ name: "time", path: timePath, realpath: timePath, sha256: timeDigest, version: "NO_STABLE_VERSION_INTERFACE" },
			],
		},
		receipt_contract: {
			study_harness: studyHarness,
			study_evidence_schema: "countershape/u7-study-evidence/v1",
			study_run_count_per_domain: 3,
			study_trial_budget_per_domain: 300,
			study_subject_process_wall_time_budget_ms_per_domain: 900000,
			study_subject_process_peak_rss_budget_bytes_per_domain: 4294967296,
			milestone_verdict: "LOCAL_REFERENCE_MILESTONE_FUNCTIONAL_GREEN_FULL_STUDY_RESOURCE_UNRECEIPTED",
			honest_fallback: "NOT_APPLICABLE_SOURCE_GREEN",
			unreceipted: ["SYNTHETIC_SELFTEST_ONLY"],
		},
	};
	const path = resolve(root, "spec/verification/u7-unit-paths.json");
	await mkdir(dirname(path), { recursive: true, mode: 0o700 });
	await writeFile(path, `${JSON.stringify(specification, null, 2)}\n`, { mode: 0o600 });
}

async function createSynthetic(mutation = "clean") {
	const root = await realpath(await mkdtemp(join(tmpdir(), "countershape-u7-study-harness-")));
	await chmod(root, 0o700);
	const checker = resolve(root, checkerRelative);
	await mkdir(dirname(checker), { recursive: true, mode: 0o700 });
	await copyFile(checkerPath, checker);
	await chmod(checker, 0o600);
	const harnessDigest = sha256(await readFile(checker));
	await writeSyntheticSpecification(root, harnessDigest);
	for (const domain of ["http", "cli"]) {
		const path = resolve(root, `tools/run-u7-${domain}-study.mjs`);
		await writeFile(path, fakeDriverSource(mutation), { mode: 0o644 });
		await chmod(path, 0o644);
		if (mutation === "driver-mode") await chmod(path, 0o600);
	}
	await writeFile(resolve(root, "go.mod"), "module example.invalid/u7selftest\n\ngo 1.26.5\n", { mode: 0o600 });
	const mainPath = resolve(root, "cmd/countershape/main.go");
	await mkdir(dirname(mainPath), { recursive: true, mode: 0o700 });
	await writeFile(mainPath, fakeGoSource(mutation), { mode: 0o600 });
	return Object.freeze({ root, checker });
}

async function prepareEnvironment(root, phase) {
	const suffix = phase.toLowerCase();
	const finalTmp = resolve(root, `.countershape/${suffix}-final/tmp`);
	for (const path of [finalTmp, resolve(root, ".cache-home"), resolve(root, ".cache-gotmp"), resolve(root, ".cache-gocache"), resolve(root, ".cache-gopath"), resolve(root, ".cache-gomodcache")]) {
		await mkdir(path, { recursive: true, mode: 0o700 });
		await chmod(path, 0o700);
	}
	const environment = cleanEnvironment(finalTmp);
	environment.HOME = resolve(root, ".cache-home");
	environment.GOTMPDIR = resolve(root, ".cache-gotmp");
	environment.GOCACHE = resolve(root, ".cache-gocache");
	environment.GOPATH = resolve(root, ".cache-gopath");
	environment.GOMODCACHE = resolve(root, ".cache-gomodcache");
	return Object.freeze({ finalTmp, environment });
}

async function runSyntheticPhase(phase, mutation = "clean", environmentOverrides = {}) {
	const fixture = await createSynthetic(mutation);
	try {
		const prepared = await prepareEnvironment(fixture.root, phase);
		const sourceProbe = spawnSync(goPath, ["test", "-mod=readonly", "-buildvcs=false", "-run", "^$", "./cmd/countershape"], {
			cwd: fixture.root,
			env: prepared.environment,
			encoding: "utf8",
			timeout: 120_000,
			maxBuffer: 1024 * 1024,
			shell: false,
		});
		if (sourceProbe.error || sourceProbe.signal !== null || sourceProbe.status !== 0) {
			fail("FIXTURE_BUILD", `${sourceProbe.error?.code ?? sourceProbe.signal ?? sourceProbe.status}: ${sourceProbe.stderr}`);
		}
		const result = invoke(fixture.checker, ["--phase", phase], fixture.root, { ...prepared.environment, ...environmentOverrides }, 180_000);
		return Object.freeze({ fixture, prepared, result, preserve: false });
	} catch (error) {
		await rm(fixture.root, { recursive: true, force: true });
		throw error;
	}
}

async function syntheticDriverDigests(root) {
	return Object.freeze({
		http: sha256(await readFile(resolve(root, "tools/run-u7-http-study.mjs"))),
		cli: sha256(await readFile(resolve(root, "tools/run-u7-cli-study.mjs"))),
	});
}

const timeReport = "real 0.02\nuser 0.01\nsys 0.00\n             1234567  maximum resident set size\n                   0  average shared memory size\n                   0  average unshared data size\n                   0  average unshared stack size\n                 100  page reclaims\n                   0  page faults\n                   0  swaps\n                   0  block input operations\n                   0  block output operations\n                   0  messages sent\n                   0  messages received\n                   0  signals received\n                   1  voluntary context switches\n                   2  involuntary context switches\n                 300  instructions retired\n                 400  cycles elapsed\n                 500  peak memory footprint\n";

function expectThrow(id, code, action) {
	let observed;
	try { action(); }
	catch (error) { observed = String(error?.message ?? error); }
	if (observed === undefined || !observed.startsWith(`U7_STUDY_HARNESS_${code}:`)) fail("HOSTILE", `${id}: ${observed ?? "accepted"}`);
}

async function evidenceHostiles(sourceRoot) {
	const cases = [
		["evidence-missing", async (root) => unlink(resolve(root, "fresh/attempts.json")), "EVIDENCE_ROSTER"],
		["evidence-extra", async (root) => writeFile(resolve(root, "extra.json"), "{}\n", { mode: 0o600 }), "EVIDENCE_ROSTER"],
		["evidence-symlink", async (root) => { await unlink(resolve(root, "fresh/attempts.json")); await symlink("measurements.json", resolve(root, "fresh/attempts.json")); }, "EVIDENCE_NONREGULAR"],
		["evidence-hardlink", async (root) => { await unlink(resolve(root, "fresh/attempts.json")); await link(resolve(root, "fresh/measurements.json"), resolve(root, "fresh/attempts.json")); }, "REGULAR_FILE"],
		["evidence-mode", async (root) => chmod(resolve(root, "fresh/attempts.json"), 0o644), "EVIDENCE_MODE"],
		["evidence-noncanonical", async (root) => writeFile(resolve(root, "fresh/attempts.json"), "{ \"schema_version\": \"x\" }\n", { mode: 0o600 }), "CANONICAL_JSON"],
		["evidence-wrong-trial", async (root) => { const path = resolve(root, "phases/search/trial-001.json"); const value = JSON.parse(await readFile(path, "utf8")); value.trial = 2; await writeFile(path, canonicalJSONLine(value), { mode: 0o600 }); }, "TRIAL_ARTIFACT"],
	];
	for (const [id, mutate, code] of cases) {
		const root = await realpath(await mkdtemp(join(tmpdir(), `countershape-${id}-`)));
		await chmod(root, 0o700);
		try {
			await cp(sourceRoot, root, { recursive: true, force: false, dereference: false });
			await mutate(root);
			let observed;
			try { await collectEvidence(root, "http", 1); }
			catch (error) { observed = String(error?.message ?? error); }
			if (observed === undefined || !observed.startsWith(`U7_STUDY_HARNESS_${code}:`)) fail("HOSTILE", `${id}: ${observed ?? "accepted"}`);
		} finally { await rm(root, { recursive: true, force: true }); }
	}
}

function validateCaseAuthority() {
	const duplicate = requiredCaseIDs.find((id, index) => requiredCaseIDs.indexOf(id) !== index);
	if (duplicate !== undefined) fail("CASE_DUPLICATE", duplicate);
	const digest = sha256(Buffer.from(`${requiredCaseIDs.join("\n")}\n`, "utf8"));
	if (digest !== requiredCaseDigest) fail("CASE_DIGEST", digest);
}

export async function selfTest(phase) {
	if (phase !== "U7P") fail("USAGE", phase);
	validateCaseAuthority();
	const liveTmp = resolve(repositoryRoot, ".countershape/u7p-final/tmp");
	const liveEnvironment = cleanEnvironment(liveTmp);
	const live = invoke(checkerPath, ["--protocol-check"], repositoryRoot, liveEnvironment);
	if (live.status !== 0 || !/^U7_STUDY_HARNESS_PROTOCOL_OK protocol=countershape\/u7-study-harness\/v1 digest=[0-9a-f]{64}\n$/u.test(live.stdout) || live.stderr !== "") {
		fail("LIVE_PROTOCOL", JSON.stringify(live));
	}

	let cleanU7DEvidence;
	let cleanU7DRepository;
	let cleanU7DDriverDigests;
	for (const phaseID of ["U7B", "U7C", "U7D"]) {
		const test = await runSyntheticPhase(phaseID);
		try {
			if (test.result.status !== 0 || test.result.stderr !== "") fail("CLEAN", `${phaseID}:${JSON.stringify(test.result)}`);
			const evidence = JSON.parse(test.result.stdout);
			const driverDigests = await syntheticDriverDigests(test.fixture.root);
			validateHarnessEvidence(evidence, phaseID, test.result.stdout, test.fixture.root, driverDigests);
			const expectedDomains = phaseID === "U7B" ? ["http"] : phaseID === "U7C" ? ["cli"] : ["http", "cli"];
			if (!isDeepStrictEqual(evidence.studies.map((study) => study.id), expectedDomains) || evidence.phase !== phaseID || evidence.studies.some((study) => study.runs.length !== 3)) fail("CLEAN", phaseID);
			if (phaseID === "U7D") { cleanU7DEvidence = evidence; cleanU7DRepository = test.fixture.root; cleanU7DDriverDigests = driverDigests; }
			if (phaseID === "U7B") await evidenceHostiles(resolve(test.fixture.root, ".countershape/u7b-final/tmp/studies/http/run-1/evidence"));
		} finally { await rm(test.fixture.root, { recursive: true, force: true }); }
	}
	const outputHostiles = [
		["output-root-extra", (value) => { value.extra = true; }],
		["output-harness-sha", (value) => { value.harness_authority.sha256 = "1".repeat(64); }],
		["output-phase", (value) => { value.phase = "U7C"; }],
		["output-verdict", (value) => { value.milestone_verdict = "WRONG"; }],
		["output-process-argv", (value) => { value.studies[0].runs[0].processes.prepare.observer_argv.pop(); }],
		["output-prepare-executable-digest", (value) => { value.studies[0].runs[0].processes.prepare.executable_sha256 = "1".repeat(64); }],
		["output-compile-executable-digest", (value) => { value.studies[0].runs[0].processes.compile.executable_sha256 = "1".repeat(64); }],
		["output-execute-executable-digest", (value) => { value.studies[0].runs[0].processes.execute.executable_sha256 = "1".repeat(64); }],
		["output-driver-digest", (value) => { value.studies[0].runs[0].driver_authority.sha256 = "1".repeat(64); }],
		["output-environment-digest", (value) => { value.studies[0].runs[0].processes.execute.environment_sha256 = "1".repeat(64); }],
		["output-time-digest", (value) => { value.studies[0].runs[0].processes.execute.time_report_sha256 = "1".repeat(64); }],
		["output-manifest-path", (value) => { value.studies[0].runs[0].evidence_manifest[0].path = "unexpected.json"; }],
		["output-manifest-digest", (value) => { value.studies[0].runs[0].evidence_manifest_sha256 = "1".repeat(64); }],
		["output-fresh-binding", (value) => { value.studies[0].runs[0].world_instance_sha256 = "1".repeat(64); }],
		["output-phase-trial", (value) => { value.studies[0].runs[0].phases[0].trial_count += 1; }],
		["output-fixture-gitdir", (value) => { value.studies[0].runs[0].fixture.git_directory += "-wrong"; }],
		["output-aggregate-wall", (value) => { value.studies[0].observed_subject_process_wall_time_ms += 1; }],
	];
	for (const [id, mutate] of outputHostiles) {
		const value = structuredClone(cleanU7DEvidence);
		mutate(value);
		let observed;
		try { validateHarnessEvidence(value, "U7D", canonicalJSONLine(value).toString("utf8"), cleanU7DRepository, cleanU7DDriverDigests); }
		catch (error) { observed = String(error?.message ?? error); }
		if (observed === undefined || !observed.startsWith("U7_STUDY_HARNESS_SELFTEST_OUTPUT:")) fail("HOSTILE", `${id}:${observed ?? "accepted"}`);
	}

	const usage = "U7_STUDY_HARNESS_USAGE: check-u7-study-harness.mjs --protocol-check | --phase U7B|U7C|U7D\n";
	const argsFixture = await createSynthetic();
	try {
		const prepared = await prepareEnvironment(argsFixture.root, "U7B");
		assertResult("args-missing", invoke(argsFixture.checker, [], argsFixture.root, prepared.environment), 2, "", usage);
		assertResult("args-u7a", invoke(argsFixture.checker, ["--phase", "U7A"], argsFixture.root, prepared.environment), 2, "", usage);
		assertResult("args-trailing", invoke(argsFixture.checker, ["--phase", "U7B", "extra"], argsFixture.root, prepared.environment), 2, "", usage);
		const wrong = { ...prepared.environment, TMPDIR: resolve(argsFixture.root, "wrong") };
		assertResult("wrong-tmpdir", invoke(argsFixture.checker, ["--phase", "U7B"], argsFixture.root, wrong), 1, "", `U7_STUDY_HARNESS_TMPDIR: ${wrong.TMPDIR}\n`);
		const first = invoke(argsFixture.checker, ["--phase", "U7B"], argsFixture.root, prepared.environment, 180_000);
		if (first.status !== 0) fail("REPEAT_SETUP", first.stderr);
		const repeat = invoke(argsFixture.checker, ["--phase", "U7B"], argsFixture.root, prepared.environment);
		if (repeat.status !== 1 || repeat.stdout !== "" || !repeat.stderr.startsWith("U7_STUDY_HARNESS_PATH_PREEXISTS:")) fail("REPEAT_ROOT", JSON.stringify(repeat));
	} finally { await rm(argsFixture.root, { recursive: true, force: true }); }
	const scrubbedExposure = await runSyntheticPhase("U7B", "clean", { COUNTERSHAPE_EVIDENCE_ROOT: "/forbidden/ambient/evidence" });
	try {
		if (scrubbedExposure.result.status !== 0 || scrubbedExposure.result.stderr !== "") fail("MUTATION", `driver-evidence-env-scrub:${JSON.stringify(scrubbedExposure.result)}`);
	} finally { await rm(scrubbedExposure.fixture.root, { recursive: true, force: true }); }

	for (const mutation of ["driver-stdout", "driver-stderr", "driver-nonzero", "driver-mode", "driver-prepopulation", "fixture-external-gitdir", "fixture-shallow", "fixture-graft", "fixture-history", "fixture-replace-ref", "fixture-assume-unchanged", "fixture-skip-worktree", "fixture-sparse", "fixture-ignored-input", "product-malformed", "product-stderr", "product-nonzero", "product-created-evidence-root"]) {
		const test = await runSyntheticPhase("U7B", mutation);
		try {
			const code = mutation === "driver-stdout" ? "PREPARE_STDOUT" : mutation === "driver-mode" ? "DRIVER_MODE" : mutation === "driver-prepopulation" ? "PATH_PREEXISTS" :
				mutation === "fixture-external-gitdir" ? "PRIVATE_DIRECTORY" : ["fixture-shallow", "fixture-graft", "fixture-replace-ref", "fixture-sparse"].includes(mutation) ? "FIXTURE_GIT_TOPOLOGY" :
				mutation === "fixture-history" ? "FIXTURE_AUTHORITY" : ["fixture-assume-unchanged", "fixture-skip-worktree"].includes(mutation) ? "FIXTURE_INDEX_FLAGS" :
				mutation === "fixture-ignored-input" ? "FIXTURE_WORKTREE_ROSTER" : mutation === "product-created-evidence-root" ? "EVIDENCE_ROOT_IDENTITY" : mutation === "product-malformed" ? "CANONICAL_JSON" :
				mutation.endsWith("stderr") ? "SUBJECT_STDERR" : "SUBJECT_EXIT";
			if (test.result.status !== 1 || test.result.stdout !== "" || !test.result.stderr.startsWith(`U7_STUDY_HARNESS_${code}:`)) fail("MUTATION", `${mutation}:${JSON.stringify(test.result)}`);
		} finally { await rm(test.fixture.root, { recursive: true, force: true }); }
	}

	for (const [mutation, code] of [["cross-domain-fixture-alias", "CROSS_DOMAIN_FIXTURE"]]) {
		const test = await runSyntheticPhase("U7D", mutation);
		try {
			if (test.result.status !== 1 || test.result.stdout !== "" || !test.result.stderr.startsWith(`U7_STUDY_HARNESS_${code}:`)) fail("MUTATION", `${mutation}:${JSON.stringify(test.result)}`);
		} finally { await rm(test.fixture.root, { recursive: true, force: true }); }
	}
	const freshAlias = structuredClone(cleanU7DEvidence.studies);
	freshAlias[1].runs[0].world_instance_sha256 = freshAlias[0].runs[0].world_instance_sha256;
	expectThrow("cross-domain-fresh-alias", "CROSS_DOMAIN_FRESHNESS", () => validateGlobalStudies(freshAlias));

	for (const name of ["node", "go", "git"]) {
		const fixture = await createSynthetic();
		try {
			const prepared = await prepareEnvironment(fixture.root, "U7B");
			const specificationPath = resolve(fixture.root, "spec/verification/u7-unit-paths.json");
			const specification = JSON.parse(await readFile(specificationPath, "utf8"));
			specification.runtime_authority.admitted_tools.find((tool) => tool.name === name).sha256 = "1".repeat(64);
			await writeFile(specificationPath, `${JSON.stringify(specification, null, 2)}\n`, { mode: 0o600 });
			const result = invoke(fixture.checker, ["--protocol-check"], fixture.root, prepared.environment);
			if (result.status !== 1 || result.stdout !== "" || !result.stderr.startsWith(`U7_STUDY_HARNESS_RUNTIME_TOOL: ${name}`)) fail("MUTATION", `runtime-${name}-digest:${JSON.stringify(result)}`);
		} finally { await rm(fixture.root, { recursive: true, force: true }); }
	}

	const parsed = parseTimeReport(Buffer.from(timeReport));
	if (parsed.wall_time_ms !== 20 || parsed.peak_rss_bytes !== 1234567) fail("TIME_POSITIVE", JSON.stringify(parsed));
	const timeHostiles = [
		["time-cr", timeReport.replace("real 0.02\n", "real 0.02\r\n")],
		["time-bom", `\ufeff${timeReport}`],
		["time-no-newline", timeReport.slice(0, -1)],
		["time-missing-row", timeReport.replace(/^.*page faults\n/mu, "")],
		["time-extra-row", `${timeReport}0 extra\n`],
		["time-reordered-row", timeReport.replace(
			"                 100  page reclaims\n                   0  page faults\n",
			"                   0  page faults\n                 100  page reclaims\n",
		)],
		["time-bad-real", timeReport.replace("real 0.02", "real 0.002")],
		["time-zero-rss", timeReport.replace("1234567  maximum", "0  maximum")],
		["time-duplicate-rss", timeReport.replace("average shared memory size", "maximum resident set size")],
	];
	for (const [id, value] of timeHostiles) expectThrow(id, "TIME_REPORT", () => parseTimeReport(Buffer.from(value)));

	const baseResult = { schema_version: productResultSchema, domain: "http", ordinal: 1, status: "GREEN" };
	validateProductResult(baseResult, "http", 1);
	expectThrow("product-result-schema", "PRODUCT_RESULT", () => validateProductResult({ ...baseResult, schema_version: "wrong" }, "http", 1));
	expectThrow("product-result-domain", "PRODUCT_RESULT", () => validateProductResult({ ...baseResult, domain: "cli" }, "http", 1));
	expectThrow("product-result-extra", "PRODUCT_RESULT", () => validateProductResult({ ...baseResult, extra: true }, "http", 1));

	process.stdout.write(`U7 study harness defensive self-test passed: active=U7P cases=${requiredCaseIDs.length} digest=${requiredCaseDigest} synthetic_phases=U7B,U7C,U7D direct_binary_observer=/usr/bin/time\n`);
}

async function main() {
	if (process.argv.length !== 4 || process.argv[2] !== "--phase" || process.argv[3] !== "U7P") fail("USAGE", "check-u7-study-harness-selftest.mjs --phase U7P");
	await selfTest(process.argv[3]);
}

async function dispatch() {
	if (process.argv[1] === undefined || resolve(process.argv[1]) !== modulePath) return;
	await main();
}

dispatch().catch((error) => {
	process.stderr.write(`${error?.message ?? error}\n`);
	process.exitCode = 1;
});
