#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { chmod, copyFile, cp, link, mkdir, mkdtemp, readFile, realpath, rm, symlink, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

const modulePath = fileURLToPath(import.meta.url);
const sourceRoot = resolve(dirname(modulePath), "..");
const checkerRelative = "tools/check-u7-architecture.mjs";
const checkerPath = resolve(sourceRoot, checkerRelative);
const phases = Object.freeze(["U7P", "U7M", "U7A", "U7B", "U7N", "U7C", "U7D", "U7Q", "U7S"]);
const modulePrefix = "github.com/nelsonwerd/countershape/";

const protectedCopyPaths = Object.freeze([
	"internal/canon",
	"internal/domain",
	"internal/observe",
	"internal/compare",
	"internal/reduce",
	"internal/reduction",
	"internal/choice",
	"tools/check-u5-architecture.mjs",
	"tools/check-u5-architecture-selftest.mjs",
	"tools/check-u6-architecture.mjs",
	"tools/check-u6-architecture-selftest.mjs",
	"spec/verification/u7-unit-paths.json",
	checkerRelative,
]);

const declaredSurfaceByPhase = Object.freeze({
	U7P: Object.freeze([]),
	U7M: Object.freeze([]),
	U7A: Object.freeze([
		"cmd/countershape/main.go",
		"internal/reference/app/command.go",
		"internal/reference/app/envelope.go",
		"internal/reference/app/render.go",
		"internal/reference/app/workspace.go",
		"internal/reference/app/spec.go",
		"internal/reference/app/validate.go",
		"internal/reference/app/preflight.go",
		"internal/reference/app/app_test.go",
		"internal/reference/app/cli_e2e_test.go",
	]),
	U7B: Object.freeze([
		"cmd/countershape/main.go",
		"internal/reference/app/command.go",
		"internal/reference/app/envelope.go",
		"internal/reference/app/render.go",
		"internal/reference/app/workspace.go",
		"internal/reference/app/run_darwin.go",
		"internal/reference/httpstudy/study.go",
		"internal/reference/httpstudy/reduction.go",
		"internal/reference/httpstudy/result.go",
		"internal/reference/httpstudy/study_test.go",
		"internal/reference/httpstudy/e2e_darwin_test.go",
		"testkit/reference/http.go",
		"testkit/reference/http_test.go",
		"tools/run-u7-http-study.mjs",
	]),
	U7N: Object.freeze([]),
	U7C: Object.freeze([
		"cmd/countershape/main.go",
		"internal/reference/app/command.go",
		"internal/reference/app/envelope.go",
		"internal/reference/app/render.go",
		"internal/reference/app/workspace.go",
		"internal/reference/clistudy/study.go",
		"internal/reference/clistudy/reduction.go",
		"internal/reference/clistudy/choice.go",
		"internal/reference/clistudy/contract.go",
		"internal/reference/clistudy/inspect.go",
		"internal/reference/clistudy/study_test.go",
		"internal/reference/clistudy/choice_test.go",
		"internal/reference/clistudy/contract_darwin_test.go",
		"testkit/reference/cli.go",
		"testkit/reference/cli_test.go",
		"tools/run-u7-cli-study.mjs",
	]),
	U7D: Object.freeze([
		"cmd/countershape/main.go",
		"internal/reference/app/command.go",
		"internal/reference/app/envelope.go",
		"internal/reference/app/render.go",
		"internal/reference/app/workspace.go",
		"internal/reference/reproduce/run_darwin.go",
		"internal/reference/reproduce/run_darwin_test.go",
		"internal/reference/app/architecture_test.go",
		"internal/reference/app/exitcodes_test.go",
	]),
	U7Q: Object.freeze([]),
	U7S: Object.freeze([]),
});

export const requiredCaseIDs = Object.freeze([
	"clean-u7p",
	"clean-u7m",
	"clean-u7a",
	"clean-u7b",
	"clean-u7n",
	"clean-u7c",
	"clean-u7d",
	"clean-u7q",
	"clean-u7s",
	"camouflage-control",
	"args-missing",
	"args-u7r",
	"args-trailing",
	"u7p-future-root",
	"u7p-extra-driver",
	"u7a-http-early",
	"u7b-cli-early",
	"u7n-cli-early",
	"u7c-reproduce-early",
	"missing-source",
	"extra-source",
	"package-mismatch",
	"source-symlink",
	"source-invalid-utf8",
	"source-mode",
	"build-tag",
	"required-main-edge",
	"required-domain-edge",
	"protected-byte",
	"protected-mode",
	"protected-extra",
	"protected-missing",
	"protected-symlink",
	"protected-hardlink",
	"protected-table-tamper",
	"spec-u7m-topology-drift",
	"spec-u7m-identity-drift",
	"spec-u7m-claim-drift",
	"spec-later-oracle-ownership",
	"spec-topology-drift",
	"spec-v2-schema",
	"spec-amendment-order-drift",
	"spec-u7n-amendment-drift",
	"spec-u7n-unit-order-drift",
	"spec-u7n-parent-drift",
	"spec-u7n-identity-drift",
	"spec-u7n-roster-drift",
	"spec-u7n-claim-drift",
	"spec-u7q-amendment-drift",
	"spec-u7q-unit-order-drift",
	"spec-u7q-parent-drift",
	"spec-u7q-identity-drift",
	"spec-u7q-roster-drift",
	"spec-u7q-claim-drift",
	"spec-u7q-receipt-source-drift",
	"spec-u7q-source-runtime-drift",
	"spec-u7q-projection-policy-drift",
	"spec-u7q-harness-ownership",
	"spec-u7q-runbook-ownership",
	"spec-u7s-amendment-drift",
	"spec-u7s-parent-drift",
	"spec-u7s-roster-drift",
	"spec-u7s-claim-drift",
	"spec-u7s-subject-path-drift",
	"spec-u7c-parent-drift",
	"spec-u7r-parent-drift",
	"spec-u7b-row-drift",
	"app-adapter-import",
	"http-cli-cross",
	"cli-http-cross",
	"reproduce-raw-kernel",
	"cmd-raw-kernel",
	"fixture-reference-edge",
	"fixture-http-cli-adapter",
	"fixture-cli-http-adapter",
	"fixture-http-test-cli-adapter",
	"fixture-cli-test-http-adapter",
	"third-party-import",
	"dot-import",
	"authority-redeclaration",
	"grouped-authority-redeclaration",
	"grouped-authority-constant",
	"grouped-authority-comma",
	"test-authority-redeclaration",
	"test-grouped-authority-constant",
	"test-local-authority-redeclaration",
	"render-recomputation",
	"render-canon-digest",
	"render-domain-fingerprint",
	"render-authority-reference",
	"local-digest-redeclaration",
	"process-owner",
	"process-alias-owner",
	"neutral-network-import",
	"neutral-network-subpackage",
	"node-network",
	"node-http2-network",
	"node-internal-http-network",
	"node-side-effect-network",
	"node-relative-import",
	"node-reexport",
	"node-computed-import",
	"node-runtime-resolver",
	"node-resolver-reference",
	"node-resolver-destructure",
	"node-resolver-computed",
	"node-process-alias",
	"node-global-alias",
	"node-process-import",
	"node-fetch-alias",
	"node-websocket-alias",
	"node-global-fetch",
	"node-global-websocket",
	"node-globalthis-eval",
	"node-global-function",
	"http-cli-coercion",
	"fixture-domain-coercion",
	"fixture-test-domain-coercion",
]);
const requiredCaseDigest = "6d038f94d4099de1469b281b1d3743fccfc70000564bfd512d2094caa60fa8cf";

function fail(code, detail) {
	throw new Error(`U7_ARCH_SELFTEST_${code}: ${detail}`);
}

function cumulativeSurface(phase) {
	const index = phases.indexOf(phase);
	if (index < 0) fail("PHASE", phase);
	const seen = new Set();
	const paths = [];
	for (const current of phases.slice(0, index + 1)) {
		for (const path of declaredSurfaceByPhase[current]) {
			if (!seen.has(path)) {
				seen.add(path);
				paths.push(path);
			}
		}
	}
	return paths;
}

function expectedPackage(path) {
	if (path.startsWith("cmd/countershape/")) return "main";
	if (path.startsWith("internal/reference/app/")) return "app";
	if (path.startsWith("internal/reference/httpstudy/")) return "httpstudy";
	if (path.startsWith("internal/reference/clistudy/")) return "clistudy";
	if (path.startsWith("internal/reference/reproduce/")) return "reproduce";
	if (path.startsWith("testkit/reference/")) return "reference";
	fail("PACKAGE", path);
}

function exactSuccess(phase) {
	return `U7_ARCHITECTURE_OK phase=${phase} governed=${cumulativeSurface(phase).length} protected=67\n`;
}

async function writeExact(root, path, bytes, mode = 0o644) {
	const absolute = resolve(root, path);
	if (!absolute.startsWith(`${root}/`)) fail("PATH_ESCAPE", path);
	await mkdir(dirname(absolute), { recursive: true, mode: 0o700 });
	await writeFile(absolute, bytes, { mode });
	await chmod(absolute, mode);
}

async function writeGo(root, path, extra = "") {
	await writeExact(root, path, `package ${expectedPackage(path)}\n${extra}`);
}

async function createSurface(root, phase) {
	for (const path of cumulativeSurface(phase)) {
		if (path.endsWith(".go")) await writeGo(root, path);
		else await writeExact(root, path, "#!/usr/bin/env node\n", 0o644);
	}
	const index = phases.indexOf(phase);
	if (index >= phases.indexOf("U7A")) {
		const imports = [`${modulePrefix}internal/reference/app`];
		if (index >= phases.indexOf("U7B")) imports.push(`${modulePrefix}internal/reference/httpstudy`);
		if (index >= phases.indexOf("U7C")) imports.push(`${modulePrefix}internal/reference/clistudy`);
		if (index >= phases.indexOf("U7D")) imports.push(`${modulePrefix}internal/reference/reproduce`);
		await writeGo(root, "cmd/countershape/main.go", `import (\n${imports.map((value) => `\t_ "${value}"`).join("\n")}\n)\n`);
	}
	if (index >= phases.indexOf("U7B")) {
		await writeGo(root, "internal/reference/httpstudy/study.go", `import (\n\t_ "${modulePrefix}internal/reference/app"\n\t_ "${modulePrefix}internal/adapters/http"\n\t_ "${modulePrefix}testkit/reference"\n)\n`);
	}
	if (index >= phases.indexOf("U7C")) {
		await writeGo(root, "internal/reference/clistudy/study.go", `import (\n\t_ "${modulePrefix}internal/reference/app"\n\t_ "${modulePrefix}internal/adapters/cli"\n\t_ "${modulePrefix}testkit/reference"\n)\n`);
	}
	if (index >= phases.indexOf("U7D")) {
		await writeGo(root, "internal/reference/reproduce/run_darwin.go", `import (\n\t_ "${modulePrefix}internal/reference/app"\n\t_ "${modulePrefix}internal/reference/httpstudy"\n\t_ "${modulePrefix}internal/reference/clistudy"\n)\n`);
	}
}

async function copyFixture(phase) {
	const created = await mkdtemp(join(tmpdir(), "countershape-u7-architecture-"));
	const root = await realpath(created);
	await chmod(root, 0o700);
	for (const path of protectedCopyPaths) {
		const destination = resolve(root, path);
		await mkdir(dirname(destination), { recursive: true, mode: 0o700 });
		const source = resolve(sourceRoot, path);
		if (["spec/verification/u7-unit-paths.json", checkerRelative].includes(path)) {
			await copyFile(source, destination);
			await chmod(destination, 0o644);
		} else {
			await cp(source, destination, { recursive: true, errorOnExist: true, dereference: false });
		}
	}
	await createSurface(root, phase);
	return root;
}

function childEnvironment() {
	return {
		PATH: "/usr/bin:/bin:/opt/homebrew/bin",
		LANG: "C",
		LC_ALL: "C",
		TZ: "UTC",
		NO_COLOR: "1",
		NODE_OPTIONS: "",
		NODE_PATH: "",
	};
}

function invoke(checker, argv) {
	const result = spawnSync(process.execPath, [checker, ...argv], {
		cwd: dirname(dirname(checker)),
		env: childEnvironment(),
		encoding: "utf8",
		timeout: 30_000,
		maxBuffer: 4 * 1024 * 1024,
	});
	if (result.error) fail("CHILD", `${result.error.code ?? result.error.message}`);
	if (result.signal !== null) fail("CHILD", `signal ${result.signal}`);
	return Object.freeze({ status: result.status, stdout: result.stdout, stderr: result.stderr });
}

async function replaceExact(root, path, before, after) {
	const absolute = resolve(root, path);
	const source = await readFile(absolute, "utf8");
	const first = source.indexOf(before);
	if (first < 0 || source.indexOf(before, first + before.length) >= 0) fail("ANCHOR", `${path}: ${before}`);
	await writeFile(absolute, source.replace(before, after), { mode: 0o644 });
}

async function removeExact(root, path) {
	await unlink(resolve(root, path));
}

async function mutateJSON(root, path, mutate) {
	const absolute = resolve(root, path);
	const value = JSON.parse(await readFile(absolute, "utf8"));
	mutate(value);
	await writeFile(absolute, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o644 });
}

const usage = "U7_ARCH_USAGE: check-u7-architecture.mjs --phase U7P|U7M|U7A|U7B|U7N|U7C|U7D|U7Q|U7S\n";

const cases = Object.freeze([
	...phases.map((phase) => Object.freeze({
		id: `clean-${phase.toLowerCase()}`,
		phase,
		status: 0,
		stdout: exactSuccess(phase),
		stderr: "",
	})),
	Object.freeze({
		id: "camouflage-control", phase: "U7A", status: 0, stdout: exactSuccess("U7A"), stderr: "",
		mutate: async (root) => writeGo(root, "internal/reference/app/command.go", "// type CandidateOutcomeMap struct{} 🚀\nvar harmless = `import \\\"example.com/rogue\\\"; type Ruling struct{}`\n"),
	}),
	Object.freeze({ id: "args-missing", rawArgs: [], status: 2, stdout: "", stderr: usage }),
	Object.freeze({ id: "args-u7r", rawArgs: ["--phase", "U7R"], status: 2, stdout: "", stderr: usage }),
	Object.freeze({ id: "args-trailing", rawArgs: ["--phase", "U7P", "extra"], status: 2, stdout: "", stderr: usage }),
	Object.freeze({ id: "u7p-future-root", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_FUTURE_SURFACE: cmd/countershape\n", mutate: (root) => writeGo(root, "cmd/countershape/main.go") }),
	Object.freeze({ id: "u7p-extra-driver", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_EXTRA: tools/run-u7-rogue.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-rogue.mjs", "#!/usr/bin/env node\n") }),
	Object.freeze({ id: "u7a-http-early", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_EXTRA: internal/reference/httpstudy/study.go\n", mutate: (root) => writeGo(root, "internal/reference/httpstudy/study.go") }),
	Object.freeze({ id: "u7b-cli-early", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_EXTRA: internal/reference/clistudy/study.go\n", mutate: (root) => writeGo(root, "internal/reference/clistudy/study.go") }),
	Object.freeze({ id: "u7n-cli-early", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_EXTRA: internal/reference/clistudy/study.go\n", mutate: (root) => writeGo(root, "internal/reference/clistudy/study.go") }),
	Object.freeze({ id: "u7c-reproduce-early", phase: "U7C", status: 1, stdout: "", stderr: "U7_ARCH_EXTRA: internal/reference/reproduce/run_darwin.go\n", mutate: (root) => writeGo(root, "internal/reference/reproduce/run_darwin.go") }),
	Object.freeze({ id: "missing-source", phase: "U7D", status: 1, stdout: "", stderr: "U7_ARCH_MISSING: internal/reference/app/command.go\n", mutate: (root) => removeExact(root, "internal/reference/app/command.go") }),
	Object.freeze({ id: "extra-source", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_EXTRA: internal/reference/app/rogue.go\n", mutate: (root) => writeGo(root, "internal/reference/app/rogue.go") }),
	Object.freeze({ id: "package-mismatch", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_PACKAGE: cmd/countershape/main.go: rogue != main\n", mutate: (root) => writeExact(root, "cmd/countershape/main.go", "package rogue\n") }),
	Object.freeze({ id: "source-symlink", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_NONREGULAR: internal/reference/app/command.go\n", mutate: async (root) => { await removeExact(root, "internal/reference/app/command.go"); await symlink("envelope.go", resolve(root, "internal/reference/app/command.go")); } }),
	Object.freeze({ id: "source-invalid-utf8", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_INVALID_UTF8: internal/reference/app/command.go\n", mutate: (root) => writeExact(root, "internal/reference/app/command.go", Buffer.from([0xff])) }),
	Object.freeze({ id: "source-mode", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_MODE: tools/run-u7-http-study.mjs: 100755 != 100644\n", mutate: (root) => chmod(resolve(root, "tools/run-u7-http-study.mjs"), 0o755) }),
	Object.freeze({ id: "build-tag", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_BUILD_TAG: internal/reference/app/command.go\n", mutate: (root) => writeExact(root, "internal/reference/app/command.go", "//go:build darwin\n\npackage app\n") }),
	Object.freeze({ id: "required-main-edge", phase: "U7A", status: 1, stdout: "", stderr: `U7_ARCH_REQUIRED_EDGE: cmd/countershape/main.go: ${modulePrefix}internal/reference/app\n`, mutate: (root) => writeGo(root, "cmd/countershape/main.go") }),
	Object.freeze({ id: "required-domain-edge", phase: "U7B", status: 1, stdout: "", stderr: `U7_ARCH_REQUIRED_EDGE: internal/reference/httpstudy/study.go: ${modulePrefix}testkit/reference\n`, mutate: (root) => writeGo(root, "internal/reference/httpstudy/study.go", `import (\n\t_ "${modulePrefix}internal/reference/app"\n\t_ "${modulePrefix}internal/adapters/http"\n)\n`) }),
	Object.freeze({ id: "protected-byte", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_PROTECTED_BYTES: internal/canon/digest.go\n", mutate: (root) => writeFile(resolve(root, "internal/canon/digest.go"), "package canon\n", { mode: 0o644 }) }),
	Object.freeze({ id: "protected-mode", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_MODE: internal/canon/digest.go: 100755 != 100644\n", mutate: (root) => chmod(resolve(root, "internal/canon/digest.go"), 0o755) }),
	Object.freeze({ id: "protected-extra", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_PROTECTED_EXTRA: internal/domain/rogue.go\n", mutate: (root) => writeExact(root, "internal/domain/rogue.go", "package domain\n") }),
	Object.freeze({ id: "protected-missing", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_PROTECTED_MISSING: internal/canon/digest.go\n", mutate: (root) => removeExact(root, "internal/canon/digest.go") }),
	Object.freeze({ id: "protected-symlink", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_NONREGULAR: internal/canon/digest.go\n", mutate: async (root) => { await removeExact(root, "internal/canon/digest.go"); await symlink("json.go", resolve(root, "internal/canon/digest.go")); } }),
	Object.freeze({ id: "protected-hardlink", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_LINK_COUNT: internal/canon/digest.go: 2\n", mutate: (root) => link(resolve(root, "internal/canon/digest.go"), resolve(root, "digest-hardlink")) }),
	Object.freeze({ id: "protected-table-tamper", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_PROTECTED_TABLE: aggregate authority mismatch\n", mutate: (root) => replaceExact(root, checkerRelative, "583c9856f4516842fed9e451719f9155814fc6b95c286bedcac38f42f5fd7ee0", "0000000000000000000000000000000000000000000000000000000000000000") }),
	Object.freeze({ id: "spec-u7m-topology-drift", phase: "U7M", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7M\n", mutate: (root) => replaceExact(root, "spec/verification/u7-unit-paths.json", '"id": "U7M",\n      "parent": "U7P"', '"id": "U7M",\n      "parent": "C6B"') }),
	Object.freeze({ id: "spec-u7m-identity-drift", phase: "U7M", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7M\n", mutate: (root) => replaceExact(root, "spec/verification/u7-unit-paths.json", '"id": "U7M",\n      "parent": "U7P",\n      "verification_profile": "SOURCE_FULL",\n      "product_authority": "NONE"', '"id": "U7M",\n      "parent": "U7P",\n      "verification_profile": "SOURCE_FULL",\n      "product_authority": "U7_REFERENCE_APPLICATION"') }),
	Object.freeze({ id: "spec-u7m-claim-drift", phase: "U7M", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: claim U7M zero-product-surface architecture conformance\n", mutate: (root) => replaceExact(root, "spec/verification/u7-unit-paths.json", '{"type": "tests-pass", "label": "U7M zero-product-surface architecture conformance", "command": ["/opt/homebrew/bin/node", "tools/check-u7-architecture.mjs", "--phase", "U7M"]}', '{"type": "tests-pass", "label": "U7M zero-product-surface architecture conformance", "command": ["/opt/homebrew/bin/node", "tools/check-u7-architecture.mjs", "--phase", "U7P"]}') }),
	Object.freeze({ id: "spec-later-oracle-ownership", phase: "U7M", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: oracle ownership tools/check-u7-architecture.mjs\n", mutate: async (root) => {
		await replaceExact(root, "spec/verification/u7-unit-paths.json", '"subject": "feat: add U7 reference CLI foundation",\n      "final_root": ".countershape/u7a-final",\n      "allowed_paths": [\n        "cmd/countershape/main.go",', '"subject": "feat: add U7 reference CLI foundation",\n      "final_root": ".countershape/u7a-final",\n      "allowed_paths": [\n        "tools/check-u7-architecture.mjs",\n        "cmd/countershape/main.go",');
		await replaceExact(root, "spec/verification/u7-unit-paths.json", '"docs/status/U7A-CLI-FOUNDATION.md"\n      ],\n      "required_paths": [\n        "cmd/countershape/main.go",', '"docs/status/U7A-CLI-FOUNDATION.md"\n      ],\n      "required_paths": [\n        "tools/check-u7-architecture.mjs",\n        "cmd/countershape/main.go",');
	} }),
	Object.freeze({ id: "spec-topology-drift", phase: "U7P", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7B\n", mutate: (root) => replaceExact(root, "spec/verification/u7-unit-paths.json", '"id": "U7B",\n      "parent": "U7A"', '"id": "U7B",\n      "parent": "U7P"') }),
	Object.freeze({ id: "spec-v2-schema", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: schema\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.schema_version = "countershape/u7-unit-paths/v2"; }) }),
	Object.freeze({ id: "spec-amendment-order-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: schema\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.topology_amendment_authorities.reverse(); }) }),
	Object.freeze({ id: "spec-u7n-amendment-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: schema\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.topology_amendment_authorities[1].authentication = "ESTABLISHED"; }) }),
	Object.freeze({ id: "spec-u7n-unit-order-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: unit order\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { [value.units[4], value.units[5]] = [value.units[5], value.units[4]]; }) }),
	Object.freeze({ id: "spec-u7n-parent-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7N\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7N").parent = "U7A"; }) }),
	Object.freeze({ id: "spec-u7n-identity-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7N\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7N").product_authority = "U7_REFERENCE_APPLICATION"; }) }),
	Object.freeze({ id: "spec-u7n-roster-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7N exact control roster\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { const unit = value.units.find((candidate) => candidate.id === "U7N"); unit.allowed_paths[0] = "docs/U7N-WRONG.md"; unit.required_paths[0] = "docs/U7N-WRONG.md"; }) }),
	Object.freeze({ id: "spec-u7n-claim-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: claim U7N zero-product-surface architecture conformance\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7N").claims[6].command[3] = "U7P"; }) }),
	Object.freeze({ id: "spec-u7q-amendment-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: schema\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.topology_amendment_authorities[2].authentication = "ESTABLISHED"; }) }),
	Object.freeze({ id: "spec-u7q-unit-order-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: unit order\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { [value.units[7], value.units[8]] = [value.units[8], value.units[7]]; }) }),
	Object.freeze({ id: "spec-u7q-parent-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7Q\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7Q").parent = "U7C"; }) }),
	Object.freeze({ id: "spec-u7q-identity-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7Q\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7Q").product_authority = "U7_REFERENCE_APPLICATION"; }) }),
	Object.freeze({ id: "spec-u7q-roster-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7Q exact control roster\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { const unit = value.units.find((candidate) => candidate.id === "U7Q"); unit.allowed_paths[0] = "docs/U7Q-WRONG.md"; unit.required_paths[0] = "docs/U7Q-WRONG.md"; }) }),
	Object.freeze({ id: "spec-u7q-claim-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: claim U7Q zero-product-surface architecture conformance\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7Q").claims.find((claim) => claim.label === "U7Q zero-product-surface architecture conformance").command[3] = "U7D"; }) }),
	Object.freeze({ id: "spec-u7q-receipt-source-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7 receipt source and projection policy\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.receipt_contract.source_boundary = "U7Q"; }) }),
	Object.freeze({ id: "spec-u7q-source-runtime-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7 receipt source and projection policy\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.receipt_contract.source_runtime_authority_sha256 = "f".repeat(64); }) }),
	Object.freeze({ id: "spec-u7q-projection-policy-drift", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7 receipt source and projection policy\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.receipt_contract.projection_policy += "_DRIFT"; }) }),
	Object.freeze({ id: "spec-u7q-harness-ownership", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7Q exact control roster\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { const unit = value.units.find((candidate) => candidate.id === "U7Q"); unit.allowed_paths[0] = "tools/check-u7-study-harness.mjs"; unit.required_paths[0] = "tools/check-u7-study-harness.mjs"; }) }),
	Object.freeze({ id: "spec-u7q-runbook-ownership", phase: "U7Q", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7Q exact control roster\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { const unit = value.units.find((candidate) => candidate.id === "U7Q"); unit.allowed_paths[0] = "tools/print-u7-final-runbook.mjs"; unit.required_paths[0] = "tools/print-u7-final-runbook.mjs"; }) }),
	Object.freeze({ id: "spec-u7s-amendment-drift", phase: "U7S", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: schema\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.topology_amendment_authorities[3].authentication = "ESTABLISHED"; }) }),
	Object.freeze({ id: "spec-u7s-parent-drift", phase: "U7S", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7S\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7S").parent = "U7D"; }) }),
	Object.freeze({ id: "spec-u7s-roster-drift", phase: "U7S", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7S exact control roster\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { const unit = value.units.find((candidate) => candidate.id === "U7S"); unit.allowed_paths[0] = "docs/U7S-WRONG.md"; unit.required_paths[0] = "docs/U7S-WRONG.md"; }) }),
	Object.freeze({ id: "spec-u7s-claim-drift", phase: "U7S", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: claim U7S zero-product-surface architecture conformance\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7S").claims.find((claim) => claim.label === "U7S zero-product-surface architecture conformance").command[3] = "U7Q"; }) }),
	Object.freeze({ id: "spec-u7s-subject-path-drift", phase: "U7S", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7 receipt source and projection policy\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.receipt_contract.study_subject_path = "/opt/homebrew/bin:/usr/bin:/bin"; }) }),
	Object.freeze({ id: "spec-u7c-parent-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7C\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7C").parent = "U7B"; }) }),
	Object.freeze({ id: "spec-u7r-parent-drift", phase: "U7S", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: U7R\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7R").parent = "U7Q"; }) }),
	Object.freeze({ id: "spec-u7b-row-drift", phase: "U7N", status: 1, stdout: "", stderr: "U7_ARCH_SPECIFICATION: sealed U7B row\n", mutate: (root) => mutateJSON(root, "spec/verification/u7-unit-paths.json", (value) => { value.units.find((unit) => unit.id === "U7B").subject += " drift"; }) }),
	Object.freeze({ id: "app-adapter-import", phase: "U7A", status: 1, stdout: "", stderr: `U7_ARCH_APP_EDGE: internal/reference/app/command.go: ${modulePrefix}internal/adapters/cli\n`, mutate: (root) => writeGo(root, "internal/reference/app/command.go", `import "${modulePrefix}internal/adapters/cli"\n`) }),
	Object.freeze({ id: "http-cli-cross", phase: "U7B", status: 1, stdout: "", stderr: `U7_ARCH_HTTP_EDGE: internal/reference/httpstudy/study.go: ${modulePrefix}internal/reference/clistudy\n`, mutate: (root) => writeGo(root, "internal/reference/httpstudy/study.go", `import "${modulePrefix}internal/reference/clistudy"\n`) }),
	Object.freeze({ id: "cli-http-cross", phase: "U7C", status: 1, stdout: "", stderr: "U7_ARCH_CLI_EDGE: internal/reference/clistudy/study.go: net/http\n", mutate: (root) => writeGo(root, "internal/reference/clistudy/study.go", 'import "net/http"\n') }),
	Object.freeze({ id: "reproduce-raw-kernel", phase: "U7D", status: 1, stdout: "", stderr: `U7_ARCH_REPRODUCE_EDGE: internal/reference/reproduce/run_darwin.go: ${modulePrefix}internal/domain\n`, mutate: (root) => writeGo(root, "internal/reference/reproduce/run_darwin.go", `import "${modulePrefix}internal/domain"\n`) }),
	Object.freeze({ id: "cmd-raw-kernel", phase: "U7A", status: 1, stdout: "", stderr: `U7_ARCH_CMD_EDGE: cmd/countershape/main.go: ${modulePrefix}internal/domain\n`, mutate: (root) => writeGo(root, "cmd/countershape/main.go", `import "${modulePrefix}internal/domain"\n`) }),
	Object.freeze({ id: "fixture-reference-edge", phase: "U7B", status: 1, stdout: "", stderr: `U7_ARCH_FIXTURE_EDGE: testkit/reference/http.go: ${modulePrefix}internal/reference/app\n`, mutate: (root) => writeGo(root, "testkit/reference/http.go", `import "${modulePrefix}internal/reference/app"\n`) }),
	Object.freeze({ id: "fixture-http-cli-adapter", phase: "U7B", status: 1, stdout: "", stderr: `U7_ARCH_FIXTURE_DOMAIN: testkit/reference/http.go: ${modulePrefix}internal/adapters/cli\n`, mutate: (root) => writeGo(root, "testkit/reference/http.go", `import "${modulePrefix}internal/adapters/cli"\n`) }),
	Object.freeze({ id: "fixture-cli-http-adapter", phase: "U7C", status: 1, stdout: "", stderr: `U7_ARCH_FIXTURE_DOMAIN: testkit/reference/cli.go: ${modulePrefix}internal/adapters/http\n`, mutate: (root) => writeGo(root, "testkit/reference/cli.go", `import "${modulePrefix}internal/adapters/http"\n`) }),
	Object.freeze({ id: "fixture-http-test-cli-adapter", phase: "U7B", status: 1, stdout: "", stderr: `U7_ARCH_FIXTURE_DOMAIN: testkit/reference/http_test.go: ${modulePrefix}internal/adapters/cli\n`, mutate: (root) => writeGo(root, "testkit/reference/http_test.go", `import "${modulePrefix}internal/adapters/cli"\n`) }),
	Object.freeze({ id: "fixture-cli-test-http-adapter", phase: "U7C", status: 1, stdout: "", stderr: `U7_ARCH_FIXTURE_DOMAIN: testkit/reference/cli_test.go: ${modulePrefix}internal/adapters/http\n`, mutate: (root) => writeGo(root, "testkit/reference/cli_test.go", `import "${modulePrefix}internal/adapters/http"\n`) }),
	Object.freeze({ id: "third-party-import", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_THIRD_PARTY_IMPORT: internal/reference/app/command.go: example.com/rogue\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", 'import "example.com/rogue"\n') }),
	Object.freeze({ id: "dot-import", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_DOT_IMPORT: internal/reference/app/command.go\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", 'import . "fmt"\n') }),
	Object.freeze({ id: "authority-redeclaration", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_AUTHORITY_REDECLARATION: internal/reference/app/command.go: CandidateOutcomeMap\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", "type CandidateOutcomeMap struct{}\n") }),
	Object.freeze({ id: "grouped-authority-redeclaration", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_AUTHORITY_REDECLARATION: internal/reference/app/command.go: CandidateOutcomeMap\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", "type (\n\tHarmless struct{}\n\tCandidateOutcomeMap struct{}\n)\n") }),
	Object.freeze({ id: "grouped-authority-constant", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_AUTHORITY_REDECLARATION: internal/reference/app/command.go: PRESERVES\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", "const (\n\tHarmless = 1\n\tPRESERVES = 2\n)\n") }),
	Object.freeze({ id: "grouped-authority-comma", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_AUTHORITY_REDECLARATION: internal/reference/app/command.go: PRESERVES\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", "const (\n\tHarmless, PRESERVES = 1, 2\n)\n") }),
	Object.freeze({ id: "test-authority-redeclaration", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_AUTHORITY_REDECLARATION: internal/reference/app/app_test.go: CandidateOutcomeMap\n", mutate: (root) => writeGo(root, "internal/reference/app/app_test.go", "type CandidateOutcomeMap struct{}\n") }),
	Object.freeze({ id: "test-grouped-authority-constant", phase: "U7C", status: 1, stdout: "", stderr: "U7_ARCH_AUTHORITY_REDECLARATION: internal/reference/clistudy/study_test.go: PRESERVES\n", mutate: (root) => writeGo(root, "internal/reference/clistudy/study_test.go", "const (\n\tHarmless = 1\n\tPRESERVES = 2\n)\n") }),
	Object.freeze({ id: "test-local-authority-redeclaration", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_AUTHORITY_REDECLARATION: internal/reference/app/app_test.go: CandidateOutcomeMap\n", mutate: (root) => writeGo(root, "internal/reference/app/app_test.go", "func testLocal() { type CandidateOutcomeMap struct{}; _ = CandidateOutcomeMap{} }\n") }),
	Object.freeze({ id: "render-recomputation", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_RENDER_AUTHORITY: internal/reference/app/render.go\n", mutate: (root) => writeGo(root, "internal/reference/app/render.go", "func render() { Compare() }\n") }),
	Object.freeze({ id: "render-canon-digest", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_RENDER_AUTHORITY: internal/reference/app/render.go\n", mutate: (root) => writeGo(root, "internal/reference/app/render.go", `import "${modulePrefix}internal/canon"\nfunc render() { _, _ = canon.DigestBytes("render", nil) }\n`) }),
	Object.freeze({ id: "render-domain-fingerprint", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_RENDER_AUTHORITY: internal/reference/app/render.go\n", mutate: (root) => writeGo(root, "internal/reference/app/render.go", `import "${modulePrefix}internal/domain"\nfunc render() { _, _ = domain.NewProjectionFingerprint(nil) }\n`) }),
	Object.freeze({ id: "render-authority-reference", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_RENDER_AUTHORITY: internal/reference/app/render.go\n", mutate: (root) => writeGo(root, "internal/reference/app/render.go", `import "${modulePrefix}internal/domain"\nvar parse = domain.ParseDigest\n`) }),
	Object.freeze({ id: "local-digest-redeclaration", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_AUTHORITY_REDECLARATION: internal/reference/app/app_test.go: Digest\n", mutate: (root) => writeGo(root, "internal/reference/app/app_test.go", "func local() { type Digest string; _ = Digest(\"x\") }\n") }),
	Object.freeze({ id: "process-owner", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_PROCESS_OWNER: internal/reference/app/command.go\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", 'import "os/exec"\n') }),
	Object.freeze({ id: "process-alias-owner", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_PROCESS_OWNER: internal/reference/app/command.go\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", 'import system "os"\nfunc launch() { system.StartProcess("x", nil, nil) }\n') }),
	Object.freeze({ id: "neutral-network-import", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_APP_EDGE: internal/reference/app/command.go: net/http\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", 'import "net/http"\n') }),
	Object.freeze({ id: "neutral-network-subpackage", phase: "U7A", status: 1, stdout: "", stderr: "U7_ARCH_APP_EDGE: internal/reference/app/command.go: net/http/cookiejar\n", mutate: (root) => writeGo(root, "internal/reference/app/command.go", 'import "net/http/cookiejar"\n') }),
	Object.freeze({ id: "node-network", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_NETWORK: tools/run-u7-http-study.mjs: node:http\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nimport http from "node:http";\n') }),
	Object.freeze({ id: "node-http2-network", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_NETWORK: tools/run-u7-http-study.mjs: node:http2\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nimport http2 from "node:http2";\n') }),
	Object.freeze({ id: "node-internal-http-network", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_NETWORK: tools/run-u7-http-study.mjs: node:_http_client\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nimport client from "node:_http_client";\n') }),
	Object.freeze({ id: "node-side-effect-network", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_NETWORK: tools/run-u7-http-study.mjs: node:http\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nimport "node:http";\n') }),
	Object.freeze({ id: "node-relative-import", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DEPENDENCY: tools/run-u7-http-study.mjs: ./foreign.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nimport "./foreign.mjs";\n') }),
	Object.freeze({ id: "node-reexport", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nexport * from "./foreign.mjs";\n') }),
	Object.freeze({ id: "node-computed-import", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nconst name = "foreign.mjs"; await import("./" + name);\n') }),
	Object.freeze({ id: "node-runtime-resolver", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nprocess.getBuiltinModule("node:http");\n') }),
	Object.freeze({ id: "node-resolver-reference", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nconst get = process.getBuiltinModule; get("node:http");\n') }),
	Object.freeze({ id: "node-resolver-destructure", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nconst { getBuiltinModule } = process; getBuiltinModule("node:http");\n') }),
	Object.freeze({ id: "node-resolver-computed", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nconst key = "getBuiltinModule"; process[key]("node:http");\n') }),
	Object.freeze({ id: "node-process-alias", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nconst runtime = process; runtime.getBuiltinModule("node:http");\n') }),
	Object.freeze({ id: "node-global-alias", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nconst runtime = globalThis; runtime.process.getBuiltinModule("node:http");\n') }),
	Object.freeze({ id: "node-process-import", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nimport runtime from "node:process"; runtime.getBuiltinModule("node:http");\n') }),
	Object.freeze({ id: "node-fetch-alias", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_NETWORK: tools/run-u7-http-study.mjs: fetch\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nconst request = fetch; await request("http://127.0.0.1");\n') }),
	Object.freeze({ id: "node-websocket-alias", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_NETWORK: tools/run-u7-http-study.mjs: WebSocket\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nconst Socket = WebSocket; new Socket("ws://127.0.0.1");\n') }),
	Object.freeze({ id: "node-global-fetch", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_NETWORK: tools/run-u7-http-study.mjs: global.fetch\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nawait global.fetch("http://127.0.0.1");\n') }),
	Object.freeze({ id: "node-global-websocket", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_NETWORK: tools/run-u7-http-study.mjs: global.WebSocket\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nnew global.WebSocket("ws://127.0.0.1");\n') }),
	Object.freeze({ id: "node-globalthis-eval", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nglobalThis.eval("0");\n') }),
	Object.freeze({ id: "node-global-function", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_NODE_DYNAMIC: tools/run-u7-http-study.mjs\n", mutate: (root) => writeExact(root, "tools/run-u7-http-study.mjs", '#!/usr/bin/env node\nnew global.Function("return 0");\n') }),
	Object.freeze({ id: "http-cli-coercion", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_HTTP_CLI_COERCION: internal/reference/httpstudy/study.go\n", mutate: (root) => writeGo(root, "internal/reference/httpstudy/study.go", "func use(CLIFixture int) {}\n") }),
	Object.freeze({ id: "fixture-domain-coercion", phase: "U7B", status: 1, stdout: "", stderr: "U7_ARCH_FIXTURE_DOMAIN: testkit/reference/http.go\n", mutate: (root) => writeGo(root, "testkit/reference/http.go", "type CLIFixture struct{}\n") }),
	Object.freeze({ id: "fixture-test-domain-coercion", phase: "U7C", status: 1, stdout: "", stderr: "U7_ARCH_FIXTURE_DOMAIN: testkit/reference/cli_test.go\n", mutate: (root) => writeGo(root, "testkit/reference/cli_test.go", "type HTTPFixture struct{}\n") }),
]);

function validateCaseAuthority() {
	if (!isDeepStrictEqual(cases.map((entry) => entry.id), requiredCaseIDs)) fail("CASE_ROSTER", "ordered IDs");
	const digest = createHash("sha256").update(`${requiredCaseIDs.join("\n")}\n`).digest("hex");
	if (digest !== requiredCaseDigest) fail("CASE_DIGEST", digest);
	const duplicate = requiredCaseIDs.find((id, index) => requiredCaseIDs.indexOf(id) !== index);
	if (duplicate !== undefined) fail("CASE_DUPLICATE", duplicate);
}

function assertResult(testCase, result) {
	if (result.status !== testCase.status || result.stdout !== testCase.stdout || result.stderr !== testCase.stderr) {
		fail("RESULT", `${testCase.id}: got status=${result.status} stdout=${JSON.stringify(result.stdout)} stderr=${JSON.stringify(result.stderr)}`);
	}
}

async function runCase(testCase) {
	if (testCase.rawArgs !== undefined) {
		const first = invoke(checkerPath, testCase.rawArgs);
		const second = invoke(checkerPath, testCase.rawArgs);
		assertResult(testCase, first);
		if (!isDeepStrictEqual(first, second)) fail("NONDETERMINISTIC", testCase.id);
		return;
	}
	const attempts = testCase.status === 0 ? 1 : 2;
	let first;
	for (let attempt = 0; attempt < attempts; attempt += 1) {
		const root = await copyFixture(testCase.phase);
		try {
			await testCase.mutate?.(root);
			const result = invoke(resolve(root, checkerRelative), ["--phase", testCase.phase]);
			assertResult(testCase, result);
			if (first === undefined) first = result;
			else if (!isDeepStrictEqual(first, result)) fail("NONDETERMINISTIC", testCase.id);
		} finally {
			await rm(root, { recursive: true, force: true });
		}
	}
}

export async function selfTest(phase) {
	if (!phases.includes(phase)) fail("USAGE", `phase ${phase}`);
	validateCaseAuthority();
	const live = invoke(checkerPath, ["--phase", phase]);
	if (live.status !== 0 || live.stdout !== exactSuccess(phase) || live.stderr !== "") {
		fail("LIVE", `status=${live.status} stdout=${JSON.stringify(live.stdout)} stderr=${JSON.stringify(live.stderr)}`);
	}
	for (const testCase of cases) await runCase(testCase);
	process.stdout.write(`U7 architecture defensive self-test passed: active=${phase} cases=${cases.length} clean=10 hostile=${cases.length - 10} digest=${requiredCaseDigest}\n`);
}

async function main() {
	if (process.argv.length !== 4 || process.argv[2] !== "--phase" || !phases.includes(process.argv[3])) {
		fail("USAGE", "check-u7-architecture-selftest.mjs --phase U7P|U7M|U7A|U7B|U7N|U7C|U7D|U7Q|U7S");
	}
	await selfTest(process.argv[3]);
}

async function dispatch() {
	if (process.argv[1] === undefined) return;
	const requested = resolve(process.argv[1]);
	let actual;
	try {
		actual = await realpath(requested);
	} catch {
		return;
	}
	if (actual !== modulePath) return;
	if (requested !== modulePath) fail("NONCANONICAL", requested);
	await main();
}

dispatch().catch((error) => {
	process.stderr.write(`${error?.message ?? error}\n`);
	process.exitCode = 1;
});
