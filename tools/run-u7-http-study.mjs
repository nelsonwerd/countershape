#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { lstat, mkdir, readFile, realpath, writeFile, chmod } from "node:fs/promises";
import { basename, dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const modulePath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(modulePath), "..");
const gitPath = "/usr/bin/git";
const fixtureProgramPath = resolve(repositoryRoot, "testkit/httpfixture/server_child_bind.mjs");
const fixtureSchema = "countershape/u7-http-fixture/v1";
const fixtureDomain = "http";
const fixtureEntrypoint = "fixture/server_child_bind.mjs";
const roles = Object.freeze(["forbidden", "conceal-not-found", "metadata-disclosure", "alternating"]);
const manifestBytes = Buffer.from('{"schema_version":"countershape/u7-http-fixture/v1","domain":"http","entrypoint":"fixture/server_child_bind.mjs","roles":["forbidden","conceal-not-found","metadata-disclosure","alternating"]}', "utf8");
const gitignoreBytes = Buffer.from("/countershape\n", "utf8");
const commandTimeoutMilliseconds = 30_000;
const maximumGitOutputBytes = 4 * 1024 * 1024;

class DriverError extends Error {
	constructor(detail) {
		super(`U7_HTTP_STUDY_DRIVER: ${detail}`);
	}
}

function parseArguments(argv) {
	if (argv.length !== 7 || argv[0] !== "--prepare" || argv[1] !== "--domain" || argv[2] !== fixtureDomain ||
		argv[3] !== "--ordinal" || !/^[1-9][0-9]*$/u.test(argv[4]) || !Number.isSafeInteger(Number(argv[4])) ||
		argv[5] !== "--fixture-root" || !isAbsolute(argv[6]) || resolve(argv[6]) !== argv[6]) {
		throw new DriverError("expected --prepare --domain http --ordinal N --fixture-root ABSOLUTE_PATH");
	}
	return Object.freeze({ ordinal: Number(argv[4]), fixtureRoot: argv[6] });
}

function gitEnvironment(fixtureRoot) {
	return Object.freeze({
		HOME: fixtureRoot,
		TMPDIR: fixtureRoot,
		PATH: "/usr/bin:/bin",
		LANG: "C",
		LC_ALL: "C",
		TZ: "UTC",
		NO_COLOR: "1",
		GIT_CONFIG_NOSYSTEM: "1",
		GIT_CONFIG_GLOBAL: "/dev/null",
		GIT_TERMINAL_PROMPT: "0",
		GIT_ASKPASS: "/usr/bin/false",
		SSH_ASKPASS: "/usr/bin/false",
		GIT_PROTOCOL_FROM_USER: "0",
		GIT_OPTIONAL_LOCKS: "0",
		GIT_NO_REPLACE_OBJECTS: "1",
		GIT_NO_LAZY_FETCH: "1",
		GIT_AUTHOR_NAME: "Countershape U7 Fixture",
		GIT_AUTHOR_EMAIL: "u7-fixture@countershape.invalid",
		GIT_AUTHOR_DATE: "2000-01-01T00:00:00Z",
		GIT_COMMITTER_NAME: "Countershape U7 Fixture",
		GIT_COMMITTER_EMAIL: "u7-fixture@countershape.invalid",
		GIT_COMMITTER_DATE: "2000-01-01T00:00:00Z",
	});
}

function runGit(fixtureRoot, args, input = undefined, inRepository = true) {
	const closedArgs = [
		"--no-pager",
		"--no-replace-objects",
		"-c", "protocol.allow=never",
		"-c", "core.hooksPath=/dev/null",
		"-c", "credential.helper=",
		"-c", "commit.gpgSign=false",
		"-c", "tag.gpgSign=false",
	];
	if (inRepository) closedArgs.push("-C", fixtureRoot);
	closedArgs.push(...args);
	const result = spawnSync(gitPath, closedArgs, {
		cwd: inRepository ? fixtureRoot : dirname(fixtureRoot),
		env: gitEnvironment(fixtureRoot),
		input,
		encoding: null,
		timeout: commandTimeoutMilliseconds,
		maxBuffer: maximumGitOutputBytes,
		windowsHide: true,
	});
	if (result.error !== undefined || result.signal !== null || result.status !== 0) {
		throw new DriverError(`git ${args[0]} failed with ${result.status ?? result.signal ?? result.error?.code ?? "unknown"}`);
	}
	return Buffer.from(result.stdout ?? Buffer.alloc(0));
}

function oneLine(bytes, label) {
	const line = bytes.toString("utf8").replace(/\n$/u, "");
	if (!/^[0-9a-f]{40}$/u.test(line)) throw new DriverError(`${label} did not return one SHA-1 OID`);
	return line;
}

async function requireAbsentCanonicalRoot(fixtureRoot) {
	const parent = dirname(fixtureRoot);
	const resolvedParent = await realpath(parent);
	if (join(resolvedParent, basename(fixtureRoot)) !== fixtureRoot) {
		throw new DriverError("fixture root parent is not canonical and symlink-free");
	}
	try {
		await lstat(fixtureRoot);
		throw new DriverError("fixture root already exists");
	} catch (error) {
		if (error instanceof DriverError) throw error;
		if (error?.code !== "ENOENT") throw new DriverError(`fixture root cannot be checked: ${error?.code ?? error?.message}`);
	}
}

async function writePrivateFile(path, bytes, executable = false) {
	await mkdir(dirname(path), { recursive: true, mode: 0o700 });
	await writeFile(path, bytes, { flag: "wx", mode: executable ? 0o700 : 0o600 });
	await chmod(path, executable ? 0o700 : 0o600);
}

function roleBytes(role) {
	return Buffer.from(`{"role":"${role}"}`, "utf8");
}

function treeInput(entries) {
	const chunks = [];
	for (const entry of entries) {
		chunks.push(Buffer.from(`${entry.mode} ${entry.type} ${entry.oid}\t${entry.name}\0`, "utf8"));
	}
	return Buffer.concat(chunks);
}

function writeBlob(fixtureRoot, bytes) {
	return oneLine(runGit(fixtureRoot, ["hash-object", "-w", "--stdin"], bytes), "hash-object");
}

function writeTree(fixtureRoot, entries) {
	return oneLine(runGit(fixtureRoot, ["mktree", "-z"], treeInput(entries)), "mktree");
}

function writeCommit(fixtureRoot, tree, message) {
	return oneLine(runGit(fixtureRoot, ["commit-tree", tree, "-m", message]), "commit-tree");
}

async function prepareFixture(fixtureRoot) {
	await requireAbsentCanonicalRoot(fixtureRoot);
	await mkdir(fixtureRoot, { mode: 0o700 });
	await chmod(fixtureRoot, 0o700);
	runGit(fixtureRoot, ["init", "--quiet", "--initial-branch=main", fixtureRoot], undefined, false);

	const fixtureProgram = await readFile(fixtureProgramPath);
	if (fixtureProgram.length === 0) throw new DriverError("portable HTTP fixture program is empty");
	const programBlob = writeBlob(fixtureRoot, fixtureProgram);
	const fixtureTree = writeTree(fixtureRoot, [
		Object.freeze({ mode: "100755", type: "blob", oid: programBlob, name: "server_child_bind.mjs" }),
	]);
	for (const role of roles) {
		const roleBlob = writeBlob(fixtureRoot, roleBytes(role));
		const candidateTree = writeTree(fixtureRoot, [
			Object.freeze({ mode: "100644", type: "blob", oid: roleBlob, name: "candidate-role.json" }),
			Object.freeze({ mode: "040000", type: "tree", oid: fixtureTree, name: "fixture" }),
		]);
		const candidateCommit = writeCommit(fixtureRoot, candidateTree, `Countershape U7 HTTP candidate: ${role}`);
		runGit(fixtureRoot, ["update-ref", `refs/heads/${role}`, candidateCommit]);
	}

	await writePrivateFile(join(fixtureRoot, ".gitignore"), gitignoreBytes);
	await writePrivateFile(join(fixtureRoot, "authority/http/manifest.json"), manifestBytes);
	for (const role of roles) {
		const authorityRoot = join(fixtureRoot, "authority/http", role);
		await writePrivateFile(join(authorityRoot, "candidate-role.json"), roleBytes(role));
		await writePrivateFile(join(authorityRoot, fixtureEntrypoint), fixtureProgram, true);
	}
	runGit(fixtureRoot, ["add", "--all", "--", "."]);
	const headTree = oneLine(runGit(fixtureRoot, ["write-tree"]), "write-tree");
	const headCommit = writeCommit(fixtureRoot, headTree, "Countershape U7 HTTP fixture authority");
	runGit(fixtureRoot, ["update-ref", httpMainRef(), headCommit]);

	const status = runGit(fixtureRoot, ["status", "--porcelain=v2", "-z", "--untracked-files=all"]);
	const count = runGit(fixtureRoot, ["rev-list", "--count", "HEAD"]).toString("utf8");
	const parents = runGit(fixtureRoot, ["rev-list", "--parents", "-n", "1", "HEAD"]).toString("utf8").trim().split(" ");
	if (status.length !== 0 || count !== "1\n" || parents.length !== 1 || parents[0] !== headCommit) {
		throw new DriverError("prepared repository is not clean with one parentless HEAD commit");
	}
}

function httpMainRef() {
	return "refs/heads/main";
}

async function main() {
	process.umask(0o077);
	const request = parseArguments(process.argv.slice(2));
	void request.ordinal;
	void fixtureSchema;
	await prepareFixture(request.fixtureRoot);
}

main().catch((error) => {
	process.stdout.write("");
	process.stderr.write(`${error?.message ?? error}\n`);
	process.exitCode = 1;
});
