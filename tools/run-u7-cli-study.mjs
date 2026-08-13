#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { chmod, lstat, mkdir, readFile, realpath, writeFile } from "node:fs/promises";
import { basename, dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const modulePath = fileURLToPath(import.meta.url);
const repositoryRoot = resolve(dirname(modulePath), "..");
const gitPath = "/usr/bin/git";
const fixtureProgramPath = resolve(repositoryRoot, "testkit/clifixture/fixture.mjs");
const fixtureSchema = "countershape/u7-cli-fixture/v1";
const fixtureDomain = "cli";
const fixtureEntrypoint = "fixture.mjs";
const roles = Object.freeze(["config-first", "env-first", "argv-first"]);
const manifestBytes = Buffer.from('{"schema_version":"countershape/u7-cli-fixture/v1","domain":"cli","entrypoint":"fixture.mjs","roles":["config-first","env-first","argv-first"]}', "utf8");
const gitignoreBytes = Buffer.from("/countershape\n", "utf8");
const gitConfigBytes = Buffer.from("[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n\tlogallrefupdates = true\n", "utf8");
const commandTimeoutMilliseconds = 30_000;
const maximumGitOutputBytes = 4 * 1024 * 1024;

class DriverError extends Error {
	constructor(detail) {
		super(`U7_CLI_STUDY_DRIVER: ${detail}`);
	}
}

function parseArguments(argv) {
	if (argv.length !== 7 || argv[0] !== "--prepare" || argv[1] !== "--domain" || argv[2] !== fixtureDomain ||
		argv[3] !== "--ordinal" || !/^(?:1|2|3)$/u.test(argv[4]) ||
		argv[5] !== "--fixture-root" || !isAbsolute(argv[6]) || resolve(argv[6]) !== argv[6]) {
		throw new DriverError("expected --prepare --domain cli --ordinal 1|2|3 --fixture-root ABSOLUTE_PATH");
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
		GIT_ATTR_NOSYSTEM: "1",
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
	if (result.error !== undefined || result.signal !== null || result.status !== 0 || (result.stderr?.length ?? 0) !== 0) {
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
	const parentMetadata = await lstat(parent);
	if (join(resolvedParent, basename(fixtureRoot)) !== fixtureRoot || parentMetadata.isSymbolicLink() ||
		!parentMetadata.isDirectory() || (parentMetadata.mode & 0o7777) !== 0o700) {
		throw new DriverError("fixture root parent is not canonical, symlink-free, and private");
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

async function replacePrivateFile(path, bytes) {
	const before = await lstat(path);
	if (before.isSymbolicLink() || !before.isFile()) throw new DriverError("repository config is not a regular file");
	await writeFile(path, bytes, { flag: "w", mode: 0o600 });
	await chmod(path, 0o600);
	const after = await lstat(path);
	if (after.isSymbolicLink() || !after.isFile() || (after.mode & 0o7777) !== 0o600) {
		throw new DriverError("repository config mode is not exact");
	}
}

function roleBytes(role) {
	return Buffer.from(`{"precedence":"${role}"}`, "utf8");
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
	await replacePrivateFile(join(fixtureRoot, ".git/config"), gitConfigBytes);

	const fixtureProgram = await readFile(fixtureProgramPath);
	if (fixtureProgram.length === 0) throw new DriverError("portable CLI fixture program is empty");
	const programBlob = writeBlob(fixtureRoot, fixtureProgram);
	const candidateCommits = new Map();
	const candidateTrees = new Set();
	for (const role of roles) {
		const roleBlob = writeBlob(fixtureRoot, roleBytes(role));
		const candidateTree = writeTree(fixtureRoot, [
			Object.freeze({ mode: "100644", type: "blob", oid: roleBlob, name: "candidate-role.json" }),
			Object.freeze({ mode: "100755", type: "blob", oid: programBlob, name: fixtureEntrypoint }),
		]);
		if (candidateTrees.has(candidateTree)) throw new DriverError("candidate trees are not distinct");
		candidateTrees.add(candidateTree);
		const candidateCommit = writeCommit(fixtureRoot, candidateTree, `Countershape U7 CLI candidate: ${role}`);
		candidateCommits.set(role, candidateCommit);
		runGit(fixtureRoot, ["update-ref", `refs/heads/${role}`, candidateCommit]);
	}

	await writePrivateFile(join(fixtureRoot, ".gitignore"), gitignoreBytes);
	await writePrivateFile(join(fixtureRoot, "authority/cli/manifest.json"), manifestBytes);
	for (const role of roles) {
		const authorityRoot = join(fixtureRoot, "authority/cli", role);
		await writePrivateFile(join(authorityRoot, "candidate-role.json"), roleBytes(role));
		await writePrivateFile(join(authorityRoot, fixtureEntrypoint), fixtureProgram, true);
	}
	runGit(fixtureRoot, ["add", "--all", "--", "."]);
	const headTree = oneLine(runGit(fixtureRoot, ["write-tree"]), "write-tree");
	const headCommit = writeCommit(fixtureRoot, headTree, "Countershape U7 CLI fixture authority");
	runGit(fixtureRoot, ["update-ref", "refs/heads/main", headCommit]);

	const status = runGit(fixtureRoot, ["status", "--porcelain=v2", "-z", "--untracked-files=all"]);
	const count = runGit(fixtureRoot, ["rev-list", "--count", "HEAD"]).toString("utf8");
	const headParents = runGit(fixtureRoot, ["rev-list", "--parents", "-n", "1", "HEAD"]).toString("utf8").trim().split(" ");
	if (status.length !== 0 || count !== "1\n" || headParents.length !== 1 || headParents[0] !== headCommit) {
		throw new DriverError("prepared repository is not clean with one parentless HEAD commit");
	}
	for (const [role, commit] of candidateCommits) {
		const parents = runGit(fixtureRoot, ["rev-list", "--parents", "-n", "1", `refs/heads/${role}`]).toString("utf8").trim().split(" ");
		if (parents.length !== 1 || parents[0] !== commit) throw new DriverError(`candidate ${role} is not parentless`);
	}
	const refRoster = runGit(fixtureRoot, ["for-each-ref", "--format=%(refname)", "refs/heads/"]).toString("utf8").trim().split("\n").sort();
	const expectedRefs = ["refs/heads/main", ...roles.map((role) => `refs/heads/${role}`)].sort();
	if (JSON.stringify(refRoster) !== JSON.stringify(expectedRefs)) throw new DriverError("prepared ref roster is not exact");
	void fixtureSchema;
}

async function main() {
	process.umask(0o077);
	const request = parseArguments(process.argv.slice(2));
	void request.ordinal;
	await prepareFixture(request.fixtureRoot);
}

main().catch((error) => {
	process.stdout.write("");
	const detail = error instanceof DriverError
		? error.message
		: `U7_CLI_STUDY_DRIVER: internal failure ${error?.code ?? "unknown"}`;
	process.stderr.write(`${detail}\n`);
	process.exitCode = 1;
});
