#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { lstat, mkdtemp, open, readdir, realpath, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const fixtureOverride = process.env.COUNTERSHAPE_A2_SOURCE_ROOT;
if (fixtureOverride && process.env.COUNTERSHAPE_A2_SELFTEST !== "1") {
	throw new Error("P07B_A2_UNTRUSTED_SOURCE_OVERRIDE: fixture override is self-test-only");
}
const sourceRoot = fixtureOverride ? resolve(fixtureOverride) : repositoryRoot;
const modulePrefix = "github.com/nelsonwerd/countershape/";
const pureAllowedInternal = new Set([
	"internal/adapters/cli/model", "internal/adapters/http/model", "internal/canon", "internal/contractsource",
	"internal/domain", "internal/emit/node/internal/compilation", "internal/emit/node/model",
	"internal/portablevalue", "internal/projectionprofile", "internal/runnerprofile",
]);
const forbiddenDependencyImports = new Set([
	"os", "io/fs", "path/filepath", "time", "runtime", "crypto/rand", "math/rand", "os/exec", "net", "net/http", "net/url",
	"plugin", "syscall", "unsafe",
]);

const goPublicAPIProgram = String.raw`package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"sort"
)

type Input struct {
	Path   string
	Source string
}

type Output struct {
	Path            string
	Consts          []string
	Types           []string
	Vars            []string
	Funcs           []string
	ExportedFuncs   []string
	ExportedMethods []string
	BodyDigests     map[string]string
	FileDigest      string
}

func receiverName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverName(value.X)
	case *ast.IndexExpr:
		return receiverName(value.X)
	case *ast.IndexListExpr:
		return receiverName(value.X)
	default:
		return ""
	}
}

func main() {
	decoder := json.NewDecoder(os.Stdin)
	decoder.DisallowUnknownFields()
	var inputs []Input
	if err := decoder.Decode(&inputs); err != nil {
		panic(err)
	}
	outputs := make([]Output, 0, len(inputs))
	for _, input := range inputs {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, input.Path, input.Source, parser.AllErrors|parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			panic(err)
		}
		output := Output{Path: input.Path, Consts: []string{}, Types: []string{}, Vars: []string{}, Funcs: []string{}, ExportedFuncs: []string{}, ExportedMethods: []string{}, BodyDigests: map[string]string{}}
		for _, declaration := range file.Decls {
			switch value := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range value.Specs {
					switch typed := specification.(type) {
					case *ast.ValueSpec:
						for _, name := range typed.Names {
							if !ast.IsExported(name.Name) {
								continue
							}
							if value.Tok == token.CONST {
								output.Consts = append(output.Consts, name.Name)
							} else if value.Tok == token.VAR {
								output.Vars = append(output.Vars, name.Name)
							}
						}
					case *ast.TypeSpec:
						if ast.IsExported(typed.Name.Name) {
							output.Types = append(output.Types, typed.Name.Name)
						}
					}
				}
			case *ast.FuncDecl:
				key := value.Name.Name
				if value.Recv == nil {
					output.Funcs = append(output.Funcs, value.Name.Name)
					if ast.IsExported(value.Name.Name) {
						output.ExportedFuncs = append(output.ExportedFuncs, value.Name.Name)
					}
				} else if len(value.Recv.List) == 1 {
					receiver := receiverName(value.Recv.List[0].Type)
					if receiver != "" {
						key = receiver+"."+value.Name.Name
						if ast.IsExported(value.Name.Name) {
							output.ExportedMethods = append(output.ExportedMethods, key)
						}
					}
				}
				var body bytes.Buffer
				if value.Body == nil || format.Node(&body, fileSet, value.Body) != nil {
					panic("function body could not be normalized: "+input.Path+":"+key)
				}
				digest := sha256.Sum256(body.Bytes())
				output.BodyDigests[key] = fmt.Sprintf("sha256:%x", digest)
			}
		}
		sort.Strings(output.Consts)
		sort.Strings(output.Types)
		sort.Strings(output.Vars)
		sort.Strings(output.Funcs)
		sort.Strings(output.ExportedFuncs)
		sort.Strings(output.ExportedMethods)
		var normalized bytes.Buffer
		if err := format.Node(&normalized, fileSet, file); err != nil {
			panic(err)
		}
		fileDigest := sha256.Sum256(normalized.Bytes())
		output.FileDigest = fmt.Sprintf("sha256:%x", fileDigest)
		outputs = append(outputs, output)
	}
	if err := json.NewEncoder(os.Stdout).Encode(outputs); err != nil {
		panic(err)
	}
}
`;

const productionFiles = Object.freeze([
	"internal/choice/choicepoint.go",
	"internal/choice/promotion/service.go",
	"internal/confirmation/wire.go",
	"internal/world/fresh_confirmation.go",
	"internal/domain/world.go",
	"internal/emit/node/model/predicate.go",
	"internal/emit/node/model/source_profile.go",
	"internal/emit/node/internal/compilation/input.go",
	"internal/emit/node/service.go",
]);

const testFiles = Object.freeze([
	"internal/choice/session_roundtrip_test.go",
	"internal/confirmation/service_test.go",
	"internal/domain/domain_test.go",
	"internal/emit/node/model/predicate_test.go",
	"testkit/studies/cli_precedence/study_darwin_test.go",
	"testkit/studies/cli_precedence/reduction_darwin_test.go",
	"testkit/studies/http_invoices/reduction_darwin_test.go",
]);

const exactFiles = Object.freeze([...productionFiles, ...testFiles]);

class ArchitectureError extends Error {
	constructor(code, detail) {
		super(`${code}: ${detail}`);
		this.code = code;
	}
}

function slash(value) { return value.split(sep).join("/"); }

function closedGoEnvironment(goPath) {
	const inheritedPaths = {};
	for (const name of ["HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"]) {
		const value = process.env[name];
		if (!value || !isAbsolute(value)) {
			throw new ArchitectureError("P07B_A2_GO_ENVIRONMENT_REQUIRED", `${name} must be one explicit absolute path`);
		}
		inheritedPaths[name] = value;
	}
	return Object.freeze({
		...inheritedPaths,
		PATH: `${dirname(goPath)}:/usr/bin:/bin`,
		GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local", GOPROXY: "off", GOSUMDB: "off",
		GOVCS: "*:off", GOFLAGS: "-mod=readonly -buildvcs=false -p=1", CGO_ENABLED: "0", GOMAXPROCS: "2",
		LANG: "C", LC_ALL: "C", TZ: "UTC", NO_COLOR: "1",
	});
}

async function inspectGoExecutable(candidate) {
	if (!candidate || !isAbsolute(candidate)) {
		throw new ArchitectureError("P07B_A2_GO_AUTHORITY_REQUIRED", "COUNTERSHAPE_GO must be one absolute reviewed Go executable");
	}
	const path = await realpath(candidate);
	let handle;
	try {
		handle = await open(path, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const metadata = await handle.stat();
		if (!metadata.isFile() || (metadata.mode & 0o111) === 0) {
			throw new ArchitectureError("P07B_A2_GO_AUTHORITY_INVALID", path);
		}
		const bytes = await handle.readFile();
		return Object.freeze({
			path, dev: metadata.dev, ino: metadata.ino, size: metadata.size, mode: metadata.mode,
			sha256: createHash("sha256").update(bytes).digest("hex"),
		});
	} finally {
		await handle?.close();
	}
}

async function revalidateGoExecutable(admission) {
	const current = await inspectGoExecutable(admission.path);
	for (const field of ["path", "dev", "ino", "size", "mode", "sha256"]) {
		if (current[field] !== admission[field]) throw new ArchitectureError("P07B_A2_GO_AUTHORITY_CHANGED", field);
	}
}

async function withGoExecutable(admission, operation) {
	await revalidateGoExecutable(admission);
	try {
		return await operation();
	} finally {
		await revalidateGoExecutable(admission);
	}
}

async function readRegularAt(root, relativePath) {
	const absolute = resolve(root, relativePath);
	const fromRoot = relative(root, absolute);
	if (fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
		throw new ArchitectureError("P07B_A2_PATH_ESCAPE", relativePath);
	}
	const before = await lstat(absolute);
	if (!before.isFile() || before.isSymbolicLink()) {
		throw new ArchitectureError("P07B_A2_NONREGULAR_AUTHORITY", relativePath);
	}
	let handle;
	try {
		handle = await open(absolute, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
		const opened = await handle.stat();
		if (!opened.isFile() || opened.dev !== before.dev || opened.ino !== before.ino || opened.size !== before.size) {
			throw new ArchitectureError("P07B_A2_AUTHORITY_CHANGED", relativePath);
		}
		const bytes = await handle.readFile();
		const after = await handle.stat();
		if (after.dev !== opened.dev || after.ino !== opened.ino || after.size !== opened.size) {
			throw new ArchitectureError("P07B_A2_AUTHORITY_CHANGED", relativePath);
		}
		return bytes;
	} finally {
		await handle?.close();
	}
}

function readRegular(relativePath) { return readRegularAt(sourceRoot, relativePath); }

function utf8(bytes, path) {
	try {
		return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
	} catch {
		throw new ArchitectureError("P07B_A2_INVALID_UTF8", path);
	}
}

function lexical(source, path) {
	const code = source.split("");
	const commentless = source.split("");
	const literals = [];
	let literal = "";
	let state = "code";
	for (let index = 0; index < source.length; index += 1) {
		const character = source[index];
		const next = source[index + 1] ?? "";
		if (state === "code") {
			if (character === "/" && next === "/") {
				code[index] = code[index + 1] = commentless[index] = commentless[index + 1] = " ";
				index += 1; state = "line";
			} else if (character === "/" && next === "*") {
				code[index] = code[index + 1] = commentless[index] = commentless[index + 1] = " ";
				index += 1; state = "block";
			} else if (character === '"' || character === "'" || character === "`") {
				code[index] = " "; literal = "";
				state = character === '"' ? "string" : character === "'" ? "rune" : "raw";
			}
		} else if (state === "line") {
			code[index] = commentless[index] = character === "\n" ? "\n" : " ";
			if (character === "\n") state = "code";
		} else if (state === "block") {
			code[index] = commentless[index] = character === "\n" ? "\n" : " ";
			if (character === "*" && next === "/") {
				code[index + 1] = commentless[index + 1] = " "; index += 1; state = "code";
			}
		} else if (state === "raw") {
			code[index] = character === "\n" ? "\n" : " ";
			if (character === "`") { literals.push(literal); state = "code"; } else literal += character;
		} else {
			code[index] = character === "\n" ? "\n" : " ";
			if (character === "\\") {
				literal += character + next; index += 1;
				if (index < code.length) code[index] = source[index] === "\n" ? "\n" : " ";
			} else if ((state === "string" && character === '"') || (state === "rune" && character === "'")) {
				literals.push(literal); state = "code";
			} else literal += character;
		}
	}
	if (state === "line") state = "code";
	if (state !== "code") throw new ArchitectureError("P07B_A2_LEXICAL_INVALID", `${path}: ${state}`);
	return { code: code.join(""), commentless: commentless.join(""), literals };
}

function imports(entry) {
	const result = [];
	const declarations = /(?:^|\n)\s*import\s*(?:\(([\s\S]*?)\)|(?:[._A-Za-z][._A-Za-z0-9]*\s+)?(?:"([^"]+)"|`([^`]+)`))/gu;
	for (const declaration of entry.source.matchAll(declarations)) {
		const importIndex = entry.source.indexOf("import", declaration.index);
		if (importIndex < 0 || entry.lexical.code.slice(importIndex, importIndex + "import".length) !== "import") continue;
		if (declaration[2] || declaration[3]) result.push(declaration[2] ?? declaration[3]);
		else for (const match of declaration[1].matchAll(/(?:^|\s)(?:[._A-Za-z][._A-Za-z0-9]*\s+)?(?:"([^"]+)"|`([^`]+)`)/gu)) {
			result.push(match[1] ?? match[2]);
		}
	}
	return result;
}

function compact(source) { return source.replace(/\s+/gu, ""); }

function functionHeader(code, name) {
	return functionDeclarations(code).find((declaration) => declaration.receiverType === "" && declaration.name === name)?.header ?? "";
}

function braceDepthBefore(code, end) {
	let depth = 0;
	for (let index = 0; index < end; index += 1) {
		if (code[index] === "{") depth += 1;
		if (code[index] === "}") depth -= 1;
	}
	return depth;
}

function matchingParenthesis(code, open) {
	let depth = 0;
	for (let index = open; index < code.length; index += 1) {
		if (code[index] === "(") depth += 1;
		if (code[index] === ")") {
			depth -= 1;
			if (depth === 0) return index;
		}
	}
	return -1;
}

function skipWhitespace(code, cursor) {
	while (/\s/u.test(code[cursor] ?? "")) cursor += 1;
	return cursor;
}

function identifierAt(code, cursor) {
	return /^[A-Za-z_][A-Za-z0-9_]*/u.exec(code.slice(cursor))?.[0] ?? "";
}

function matchingBracket(code, open) {
	let depth = 0;
	for (let index = open; index < code.length; index += 1) {
		if (code[index] === "[") depth += 1;
		if (code[index] === "]") {
			depth -= 1;
			if (depth === 0) return index;
		}
	}
	return -1;
}

function functionDeclarations(code) {
	const declarations = [];
	for (const match of code.matchAll(/\bfunc\b/gu)) {
		if (braceDepthBefore(code, match.index) !== 0) continue;
		let cursor = skipWhitespace(code, match.index + match[0].length);
		let receiverType = "";
		if (code[cursor] === "(") {
			const close = matchingParenthesis(code, cursor);
			if (close < 0) continue;
			const receiver = code.slice(cursor + 1, close).replace(/\[[\s\S]*\]/gu, "");
			const identifiers = [...receiver.matchAll(/[A-Za-z_][A-Za-z0-9_]*/gu)].map((item) => item[0]);
			receiverType = identifiers.at(-1) ?? "";
			cursor = skipWhitespace(code, close + 1);
		}
		const name = identifierAt(code, cursor);
		if (!name) continue;
		cursor = skipWhitespace(code, cursor + name.length);
		if (code[cursor] === "[") {
			const close = matchingBracket(code, cursor);
			if (close < 0) continue;
			cursor = skipWhitespace(code, close + 1);
		}
		if (code[cursor] !== "(") continue;
		const parametersClose = matchingParenthesis(code, cursor);
		if (parametersClose < 0) continue;
		const parameters = code.slice(cursor + 1, parametersClose).trim().replace(/\s+/gu, " ");
		let parentheses = 0;
		let brackets = 0;
		let openBrace = -1;
		for (let index = parametersClose + 1; index < code.length; index += 1) {
			if (code[index] === "(") parentheses += 1;
			if (code[index] === ")") parentheses -= 1;
			if (code[index] === "[") brackets += 1;
			if (code[index] === "]") brackets -= 1;
			if (code[index] === "{" && parentheses === 0 && brackets === 0) { openBrace = index; break; }
		}
		if (openBrace < 0) continue;
		const returns = code.slice(parametersClose + 1, openBrace).trim().replace(/\s+/gu, " ");
		declarations.push({
			name, receiverType, parameters, returns, openBrace,
			header: code.slice(match.index, openBrace),
		});
	}
	return declarations;
}

function unqualifiedCallNames(code) {
	const excluded = new Set(["for", "if", "select", "switch"]);
	const names = [];
	for (const match of code.matchAll(/\b([A-Za-z_][A-Za-z0-9_]*)\s*\(/gu)) {
		if (excluded.has(match[1])) continue;
		const before = code.slice(0, match.index).match(/\S\s*$/u)?.[0]?.trim() ?? "";
		if (before === ".") continue;
		names.push(match[1]);
	}
	return [...new Set(names)].sort();
}

function normalizedStructFields(code, name) {
	return structBody(code, name).split(/\r?\n/u).map((line) => line.trim()).filter(Boolean)
		.map((line) => line.replace(/\s+/gu, " "));
}

function exportedReceiverSignatures(code, typeName) {
	return functionDeclarations(code).filter((declaration) => declaration.receiverType === typeName &&
		/^[A-Z]/u.test(declaration.name)).map((declaration) =>
		`${declaration.name}(${declaration.parameters})${declaration.returns ? ` ${declaration.returns}` : ""}`).sort();
}

function functionBody(code, name, receiverType = undefined) {
	const declaration = functionDeclarations(code).find((candidate) => candidate.name === name &&
		(receiverType === undefined || candidate.receiverType === receiverType));
	if (!declaration) return "";
	const openBrace = declaration.openBrace;
	let depth = 0;
	for (let index = openBrace; index < code.length; index += 1) {
		if (code[index] === "{") depth += 1;
		if (code[index] === "}") {
			depth -= 1;
			if (depth === 0) return code.slice(openBrace + 1, index);
		}
	}
	return "";
}

function structBody(code, name) {
	const match = new RegExp(`\\btype\\s+${name}\\s+struct\\s*\\{`, "u").exec(code);
	if (!match) return "";
	const openBrace = code.indexOf("{", match.index);
	let depth = 0;
	for (let index = openBrace; index < code.length; index += 1) {
		if (code[index] === "{") depth += 1;
		if (code[index] === "}") {
			depth -= 1;
			if (depth === 0) return code.slice(openBrace + 1, index);
		}
	}
	return "";
}

async function manifest() {
	const promotionProduction = (await readdir(resolve(sourceRoot, "internal/choice/promotion"), { withFileTypes: true }))
		.filter((entry) => entry.name.endsWith(".go") && !entry.name.endsWith("_test.go"));
	if (promotionProduction.some((entry) => !entry.isFile() || entry.isSymbolicLink()) ||
		JSON.stringify(promotionProduction.map((entry) => entry.name).sort()) !== JSON.stringify(["service.go"])) {
		throw new ArchitectureError(
			"P07B_A2_PROMOTION_PRODUCTION_MAP_DRIFT",
			promotionProduction.map((entry) => entry.name).sort().join(","),
		);
	}
	if (!fixtureOverride) {
		const exactPackageMaps = new Map([
			["internal/emit/node/model", ["predicate.go", "predicate_test.go", "source_profile.go"]],
			["internal/emit/node/internal/compilation", ["input.go"]],
			["internal/emit/node", ["internal", "model", "service.go"]],
		]);
		for (const [directory, expected] of exactPackageMaps) {
			const actual = (await readdir(resolve(sourceRoot, directory))).sort();
			if (JSON.stringify(actual) !== JSON.stringify([...expected].sort())) {
				throw new ArchitectureError("P07B_A2_PACKAGE_MAP_DRIFT", `${directory}: ${actual.join(",")}`);
			}
		}
	}
	const aggregate = createHash("sha256");
	const entries = [];
	for (const path of exactFiles) {
		const bytes = await readRegular(path);
		aggregate.update(slash(path)); aggregate.update("\0"); aggregate.update(bytes); aggregate.update("\0");
		const source = utf8(bytes, path);
		entries.push({ path, bytes, source, lexical: lexical(source, path) });
	}
	return { entries, digest: aggregate.digest("hex") };
}

async function goPublicAPIRosters(entries, go, environment) {
	const directory = await mkdtemp(join(await realpathDirectory(tmpdir()), "countershape-a2-go-ast-"));
	const helper = join(directory, "main.go");
	try {
		await writeFile(helper, goPublicAPIProgram, { mode: 0o600 });
		const result = spawnSync(go.path, ["run", "-mod=readonly", helper], {
			cwd: repositoryRoot, input: JSON.stringify(entries.map((entry) => ({ Path: entry.path, Source: entry.source }))),
			encoding: "utf8", timeout: 60_000, maxBuffer: 8 * 1024 * 1024,
			env: environment,
		});
		if (result.error || result.signal || result.status !== 0) {
			throw new ArchitectureError("P07B_A2_GO_AST_QUERY_FAILED", `${result.status ?? result.signal ?? result.error?.message}: ${result.stderr || result.stdout}`);
		}
		let rows;
		try {
			rows = JSON.parse(result.stdout);
		} catch (error) {
			throw new ArchitectureError("P07B_A2_GO_AST_QUERY_FAILED", `malformed JSON: ${error.message}`);
		}
		if (!Array.isArray(rows) || rows.length !== entries.length) {
			throw new ArchitectureError("P07B_A2_GO_AST_QUERY_FAILED", "row count differs from exact manifest");
		}
		const rosters = new Map();
		for (let index = 0; index < rows.length; index += 1) {
			const row = rows[index];
			const entry = entries[index];
			const fields = ["Consts", "Types", "Vars", "Funcs", "ExportedFuncs", "ExportedMethods"];
			if (!row || row.Path !== entry.path || fields.some((field) => !Array.isArray(row[field]) ||
				row[field].some((name) => typeof name !== "string")) || !row.BodyDigests || typeof row.BodyDigests !== "object" ||
				Array.isArray(row.BodyDigests) || Object.entries(row.BodyDigests).some(([name, digest]) =>
					typeof name !== "string" || !/^sha256:[0-9a-f]{64}$/u.test(digest)) ||
				!/^sha256:[0-9a-f]{64}$/u.test(row.FileDigest ?? "")) {
				throw new ArchitectureError("P07B_A2_GO_AST_QUERY_FAILED", `malformed row ${index}`);
			}
			rosters.set(entry.path, Object.freeze({
				const: Object.freeze([...row.Consts]), type: Object.freeze([...row.Types]), var: Object.freeze([...row.Vars]),
				funcs: Object.freeze([...row.Funcs]), exportedFuncs: Object.freeze([...row.ExportedFuncs]),
				exportedMethods: Object.freeze([...row.ExportedMethods]),
				bodyDigests: Object.freeze({ ...row.BodyDigests }), fileDigest: row.FileDigest,
			}));
		}
		return rosters;
	} finally {
		await rm(directory, { recursive: true, force: true });
	}
}

async function realpathDirectory(path) {
	const metadata = await lstat(path);
	if (!metadata.isDirectory() || metadata.isSymbolicLink()) {
		throw new ArchitectureError("P07B_A2_TEMP_DIRECTORY_INVALID", path);
	}
	return path;
}

function runInheritedA1(go, goEnvironment) {
	const environment = { ...goEnvironment, COUNTERSHAPE_GO: go.path };
	const result = spawnSync(process.execPath, [resolve(repositoryRoot, "tools/check-p07b-architecture.mjs")], {
		cwd: repositoryRoot, encoding: "utf8", timeout: 60_000, maxBuffer: 4 * 1024 * 1024,
		env: environment,
	});
	if (result.error || result.signal || result.status !== 0 || !result.stdout.includes("P07B source architecture boundary OK")) {
		throw new ArchitectureError("P07B_A2_INHERITED_A1_FAILED", `${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
	}
}

function runPureDependencyClosure(go, environment) {
	const result = spawnSync(go.path, [
		"list", "-deps", "-f", "{{if .Module}}{{.ImportPath}}|{{join .Imports \",\"}}{{end}}",
		"./internal/emit/node/model", "./internal/emit/node/internal/compilation",
	], {
		cwd: repositoryRoot, encoding: "utf8", timeout: 45_000, maxBuffer: 4 * 1024 * 1024,
		env: environment,
	});
	if (result.error || result.signal || result.status !== 0) {
		throw new ArchitectureError("P07B_A2_DEPENDENCY_QUERY_FAILED", `${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
	}
	for (const line of result.stdout.split(/\r?\n/u)) {
		if (!line) continue;
		const separator = line.indexOf("|");
		const packagePath = separator < 0 ? line : line.slice(0, separator);
		const rawImports = separator < 0 ? "" : line.slice(separator + 1);
		if (!packagePath.startsWith(modulePrefix)) {
			throw new ArchitectureError("P07B_A2_PURE_EXTERNAL_DEPENDENCY", packagePath);
		}
		const internal = packagePath.slice(modulePrefix.length);
		if (!pureAllowedInternal.has(internal)) {
			throw new ArchitectureError("P07B_A2_PURE_TRANSITIVE_INTERNAL_DEPENDENCY", internal);
		}
		for (const imported of rawImports.split(",").filter(Boolean)) {
			if (forbiddenDependencyImports.has(imported) || imported.startsWith("net/")) {
				throw new ArchitectureError("P07B_A2_PURE_TRANSITIVE_FORBIDDEN_DEPENDENCY", `${internal}->${imported}`);
			}
			if (imported.startsWith(modulePrefix) && !pureAllowedInternal.has(imported.slice(modulePrefix.length))) {
				throw new ArchitectureError("P07B_A2_PURE_TRANSITIVE_INTERNAL_DEPENDENCY", `${internal}->${imported.slice(modulePrefix.length)}`);
			}
			if (!imported.startsWith(modulePrefix) && imported.includes(".")) {
				throw new ArchitectureError("P07B_A2_PURE_EXTERNAL_DEPENDENCY", `${internal}->${imported}`);
			}
		}
	}
}

function runGoDirectImportRosters(go, environment) {
	const packages = [
		"./internal/emit/node", "./internal/emit/node/model", "./internal/emit/node/internal/compilation",
	];
	const result = spawnSync(go.path, ["list", "-f", "{{.ImportPath}}|{{join .Imports \",\"}}", ...packages], {
		cwd: repositoryRoot, encoding: "utf8", timeout: 45_000, maxBuffer: 4 * 1024 * 1024,
		env: environment,
	});
	if (result.error || result.signal || result.status !== 0) {
		throw new ArchitectureError("P07B_A2_DIRECT_IMPORT_QUERY_FAILED", `${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
	}
	const expected = new Map([
		[`${modulePrefix}internal/emit/node`, [
			"bytes", "context", "errors", "fmt", `${modulePrefix}internal/choice`, `${modulePrefix}internal/choice/promotion`,
			`${modulePrefix}internal/compare`, `${modulePrefix}internal/contractsource`, `${modulePrefix}internal/domain`,
			`${modulePrefix}internal/emit/node/internal/compilation`, `${modulePrefix}internal/emit/node/model`,
			`${modulePrefix}internal/portablevalue`, `${modulePrefix}internal/projectiontranslate`, `${modulePrefix}internal/store`,
		].sort()],
		[`${modulePrefix}internal/emit/node/model`, [
			"bytes", "encoding/base64", "errors", "fmt", "slices", "sort", `${modulePrefix}internal/canon`,
			`${modulePrefix}internal/contractsource`, `${modulePrefix}internal/domain`, `${modulePrefix}internal/portablevalue`,
			`${modulePrefix}internal/projectionprofile`, `${modulePrefix}internal/runnerprofile`,
		].sort()],
		[`${modulePrefix}internal/emit/node/internal/compilation`, [
			"bytes", "encoding/base64", `${modulePrefix}internal/canon`, `${modulePrefix}internal/contractsource`,
			`${modulePrefix}internal/domain`, `${modulePrefix}internal/emit/node/model`,
		].sort()],
	]);
	const actual = new Map();
	for (const line of result.stdout.trim().split(/\r?\n/u)) {
		const separator = line.indexOf("|");
		if (separator < 0) throw new ArchitectureError("P07B_A2_DIRECT_IMPORT_QUERY_FAILED", line);
		actual.set(line.slice(0, separator), line.slice(separator + 1).split(",").filter(Boolean).sort());
	}
	if (actual.size !== expected.size) {
		throw new ArchitectureError("P07B_A2_GO_DIRECT_IMPORT_ROSTER", `package count ${actual.size}`);
	}
	for (const [packagePath, wanted] of expected) {
		const present = actual.get(packagePath) ?? [];
		if (JSON.stringify(present) !== JSON.stringify(wanted)) {
			throw new ArchitectureError("P07B_A2_GO_DIRECT_IMPORT_ROSTER", `${packagePath}:${present.join(",")}`);
		}
	}
}

async function packageEntries(internalPackage) {
	const directory = internalPackage;
	let root = sourceRoot;
	try {
		const candidate = await lstat(resolve(sourceRoot, directory));
		if (!candidate.isDirectory() || candidate.isSymbolicLink()) {
			throw new ArchitectureError("P07B_A2_DEPENDENCY_DIRECTORY_INVALID", directory);
		}
	} catch (error) {
		if (error instanceof ArchitectureError) throw error;
		if (error?.code !== "ENOENT") throw error;
		root = repositoryRoot;
	}
	const directoryEntries = await readdir(resolve(root, directory), { withFileTypes: true });
	const result = [];
	for (const item of directoryEntries) {
		if (!item.name.endsWith(".go") || item.name.endsWith("_test.go")) continue;
		if (!item.isFile() || item.isSymbolicLink()) {
			throw new ArchitectureError("P07B_A2_DEPENDENCY_FILE_INVALID", `${directory}/${item.name}`);
		}
		const path = `${directory}/${item.name}`;
		const bytes = await readRegularAt(root, path);
		const source = utf8(bytes, path);
		result.push({ path, source, lexical: lexical(source, path) });
	}
	if (result.length === 0) throw new ArchitectureError("P07B_A2_DEPENDENCY_PACKAGE_EMPTY", directory);
	return result;
}

async function inspectStaticPureDependencyClosure() {
	const pending = ["internal/emit/node/model", "internal/emit/node/internal/compilation"];
	const visited = new Set();
	while (pending.length > 0) {
		const current = pending.shift();
		if (visited.has(current)) continue;
		visited.add(current);
		if (!pureAllowedInternal.has(current)) {
			throw new ArchitectureError("P07B_A2_PURE_TRANSITIVE_INTERNAL_DEPENDENCY", current);
		}
		for (const entry of await packageEntries(current)) {
			for (const imported of imports(entry)) {
				if (forbiddenDependencyImports.has(imported) || imported.startsWith("net/")) {
					throw new ArchitectureError("P07B_A2_PURE_TRANSITIVE_FORBIDDEN_DEPENDENCY", `${entry.path}->${imported}`);
				}
				if (imported.startsWith(modulePrefix)) {
					const internal = imported.slice(modulePrefix.length);
					if (!pureAllowedInternal.has(internal)) {
						throw new ArchitectureError("P07B_A2_PURE_TRANSITIVE_INTERNAL_DEPENDENCY", `${entry.path}->${internal}`);
					}
					pending.push(internal);
				} else if (imported.includes(".")) {
					throw new ArchitectureError("P07B_A2_PURE_EXTERNAL_DEPENDENCY", `${entry.path}->${imported}`);
				}
			}
		}
	}
}

function countMatches(source, expression) {
	return [...source.matchAll(expression)].length;
}

function inspect(input, publicAPI) {
	const violations = [];
	const at = (path) => input.entries.find((entry) => entry.path === path);
	const api = (path) => publicAPI.get(path);
	const add = (condition, code, detail) => { if (condition) violations.push([code, detail]); };
	const choicepointEntry = at("internal/choice/choicepoint.go");
	const promotionEntry = at("internal/choice/promotion/service.go");
	const confirmationEntry = at("internal/confirmation/wire.go");
	const worldEntry = at("internal/world/fresh_confirmation.go");
	const choicepoint = choicepointEntry.lexical.code;
	const promotion = promotionEntry.lexical.code;
	const confirmation = confirmationEntry.lexical.code;
	const world = worldEntry.lexical.code;
	const predicate = at("internal/emit/node/model/predicate.go");
	const sourceProfile = at("internal/emit/node/model/source_profile.go");
	const compilation = at("internal/emit/node/internal/compilation/input.go");
	const service = at("internal/emit/node/service.go");
	for (const [entry, expected, code] of [
		[choicepointEntry, "sha256:ff63d0c86eaa421d3a53a977a1684a00b7b9803001b8a850904328698d6b6791", "P07B_A2_CHOICE_AST_DRIFT"],
		[confirmationEntry, "sha256:afa19ed2e9cbccb4e38f6164aef7342435bc3aa299cc3f10b1614089bb1bbc17", "P07B_A2_CONFIRMATION_AST_DRIFT"],
		[promotionEntry, "sha256:8d838c8c6da6aa7a0cd9241d46c26a7b92acfc76e352fe6907ed5a5aaac03ce6", "P07B_A2_PROMOTION_AST_DRIFT"],
		[worldEntry, "sha256:93295e3eea7706fc7efb558db513618b0797394e58b5f9f18e0256be49ddf943", "P07B_A2_WORLD_AST_DRIFT"],
		[at("internal/domain/world.go"), "sha256:5c83df7c28163fb78ba7c01fe7099a0b44e93dae1cc50dc760b332fec17ee735", "P07B_A2_DOMAIN_AST_DRIFT"],
		[service, "sha256:a1349bfa80e4202db7a67ebe880ca33805009515f9d1fcede247e9c85b02e2b5", "P07B_A2_NODE_AST_DRIFT"],
		[compilation, "sha256:a5148b1ce018ff8a100fff20fe5a8a2e78cfa06aa6eada6beef9fc115607b8fd", "P07B_A2_COMPILATION_AST_DRIFT"],
		[predicate, "sha256:eef12044ff0aa29cf116d31115754962207b253a49de6923fba6d618d83a8359", "P07B_A2_MODEL_AST_DRIFT"],
		[sourceProfile, "sha256:449634bae75d1c1d55af53bd7c926e5309c85dbb240b25dbbacad89ed3c3038d", "P07B_A2_MODEL_AST_DRIFT"],
	]) add(api(entry.path).fileDigest !== expected, code, `${entry.path}:${api(entry.path).fileDigest}`);
	for (const [path, expected] of [
		["internal/choice/session_roundtrip_test.go", "sha256:e7abce945ce9f2bdfc2451d3478bdbb54369b03fe66470ce56cf96dddde7088e"],
		["internal/confirmation/service_test.go", "sha256:a51cfcdc1095ab742cfd5940bfccd06895122b308d2e437ca8dea6251c73e193"],
		["internal/domain/domain_test.go", "sha256:27ff69cc148d3ece3111b745be72231baa7a315e55bb71d36535a56ee686f4b9"],
		["internal/emit/node/model/predicate_test.go", "sha256:b42fb2979b259868a1e55eb40bbbdb62f8e4ce2cf120ff495967a59cc3299b29"],
		["testkit/studies/cli_precedence/study_darwin_test.go", "sha256:86ef88170329de110283b1f779713e56978225ffa3c365942a6db1afc62f879e"],
		["testkit/studies/cli_precedence/reduction_darwin_test.go", "sha256:fa92bafd20fdfad932c6d82a16e8807280bbaa8230771e731c636f9e94339e61"],
		["testkit/studies/http_invoices/reduction_darwin_test.go", "sha256:547d9be58290bb1291a4d5487f3288544495d5d51254eecf31f57c5239dcc05a"],
	]) add(api(path).fileDigest !== expected, "P07B_A2_TEST_AST_DRIFT", `${path}:${api(path).fileDigest}`);
	for (const [path, name, expected, code] of [
		[choicepointEntry.path, "ChoicepointRecord.WorldPlan", "sha256:3fab6ef185557551d26d665d28a243ea1da98b5552d544c576333ea7d70135db", "P07B_A2_CHOICE_ACCESSOR_DATAFLOW"],
		[choicepointEntry.path, "ChoicepointRecord.MinimizedStimulus", "sha256:1e067347e46fcb79b9fe1ef8dfd7184d6c2bb7787efd5cafbfd7220e7f852697", "P07B_A2_CHOICE_ACCESSOR_DATAFLOW"],
		[choicepointEntry.path, "ChoicepointRecord.PortableProfile", "sha256:5424633753a59126b723c8be37867a1d010ffe20d8391bf2bc22ecd5e448d0d5", "P07B_A2_CHOICE_ACCESSOR_DATAFLOW"],
		[worldEntry.path, "FreshExecutionRecord.ExecutionBindingDigest", "sha256:f1dc486d8c6c14ee02985a9b13b51188f3114d452a2b944f70ee5bd8222d21d6", "P07B_A2_WORLD_BINDING_ACCESSOR_DATAFLOW"],
		[confirmationEntry.path, "Record.ExecutionBindingDigests", "sha256:4ea008dc884fb5213c02e9e7c64019f61dca5ac44f2e6b1747b5cbf8fa74f4e3", "P07B_A2_CONFIRMATION_BINDING_ACCESSOR_MISSING"],
		[confirmationEntry.path, "Record.Valid", "sha256:55953eec2a7ff5b178c67d56605f1ef38d6376ae7e1e4829bf19fecdf5075a78", "P07B_A2_CONFIRMATION_ROSTER_NOT_SELF_VALIDATING"],
		[promotionEntry.path, "OpenPortableCompilationSnapshot", "sha256:6b794bbeb04617c99fb9f7eb0296b09a5c5444eb324d2539e290b1b80a1d717c", "P07B_A2_SNAPSHOT_FRESH_INSPECTION_DATAFLOW"],
		[promotionEntry.path, "PortableCompilationSnapshot.Valid", "sha256:3e4cff6521d54c2eb3ccf2e2b202662cee6940ea4bdfe414839f26b4f45695cf", "P07B_A2_SNAPSHOT_PROFILE_VALID_DATAFLOW"],
		[service.path, "PrepareCompilation", "sha256:65e46644ba7623fb0629e9a0791042f0ca86460eeaf43d0533215fec4fc72e78", "P07B_A2_PREPARED_PROFILE_BYTE_JOIN"],
		[service.path, "requireSourceChoicepointJoin", "sha256:023818d8c7bc17068386964ebd113a0e4abc5632c734880db69e1ef41497dd9e", "P07B_A2_SOURCE_JOIN_DATAFLOW"],
		[service.path, "revalidatePartition", "sha256:947d7ca5088024e7ffcb22bb292df6f60eedd284d64ff269aa6d0162f6d44790", "P07B_A2_RETRANSLATION_DATAFLOW"],
		[service.path, "PreparedCompilation.Valid", "sha256:1b0a72fb0e85378be23fcb7ce3df0d1b26114eab27616dd8c10b17c66574a338", "P07B_A2_PREPARED_AUTHORITY_EXPOSED"],
	]) {
		const actual = api(path).bodyDigests[name];
		add(actual !== expected, code, `${path}:${name}:${actual ?? "missing"}`);
	}

	for (const [source, anchor, code] of [
		[choicepoint, "func (r ChoicepointRecord) WorldPlan() domain.WorldPlan", "P07B_A2_CHOICE_PLAN_ACCESSOR_MISSING"],
		[choicepoint, "func (r ChoicepointRecord) MinimizedStimulus() CanonicalArtifact", "P07B_A2_CHOICE_STIMULUS_ACCESSOR_MISSING"],
		[choicepoint, "func (r ChoicepointRecord) PortableProfile() projectionprofile.Profile", "P07B_A2_CHOICE_PROFILE_ACCESSOR_MISSING"],
		[world, "func (r FreshExecutionRecord) ExecutionBindingDigest() domain.Digest", "P07B_A2_WORLD_BINDING_ACCESSOR_MISSING"],
		[confirmation, "func (r Record) ExecutionBindingDigests() []domain.Digest", "P07B_A2_CONFIRMATION_BINDING_ACCESSOR_MISSING"],
		[promotion, "func OpenPortableCompilationSnapshot(", "P07B_A2_SNAPSHOT_MISSING"],
	]) add(!source.includes(anchor), code, anchor);

	add(!confirmation.includes("physicalWireSummary") || !confirmation.includes("executionBindings") ||
		!functionBody(confirmation, "Valid").includes("ParseRecord") ||
		!functionBody(confirmation, "Valid").includes("sameDigestsInOrder(parsed.executionBindings, r.executionBindings)"),
		"P07B_A2_CONFIRMATION_ROSTER_NOT_SELF_VALIDATING", "retained binding roster must be strict-reparse-bound");
	add(!promotion.includes("mapSnapshotCurrentness") || !promotion.includes("CodeStaleChoicepoint") ||
		!functionBody(promotion, "OpenPortableCompilationSnapshot").includes("openRulingAtHead"),
	"P07B_A2_SNAPSHOT_CURRENTNESS_INCOMPLETE", "snapshot currentness/reopen path");
	add(promotion.includes("HeadToken()") || structBody(promotion, "PortableCompilationSnapshot").includes("HeadToken"),
		"P07B_A2_SNAPSHOT_EXPOSES_HEAD", "snapshot must expose no HeadToken");
	const exactSnapshotFields = [
		"decision choice.DecisionRecord", "choicepoint choice.ChoicepointRecord", "confirmation confirmation.Record",
		"profile projectionprofile.Profile", "inspection choice.PortableRulingInspection", "seal *portableCompilationSnapshotSeal",
	];
	const snapshotFields = normalizedStructFields(promotion, "PortableCompilationSnapshot");
	add(JSON.stringify(snapshotFields) !== JSON.stringify(exactSnapshotFields),
		"P07B_A2_SNAPSHOT_AUTHORITY_DRIFT", snapshotFields.join(","));
	const exactSnapshotMethods = [
		"Valid() bool", "DecisionRecord() choice.DecisionRecord", "ChoicepointRecord() choice.ChoicepointRecord",
		"ConfirmationRecord() confirmation.Record", "PortableProfile() projectionprofile.Profile", "ProfileDigest() domain.Digest",
	].sort();
	const snapshotMethods = exportedReceiverSignatures(promotion, "PortableCompilationSnapshot");
	add(JSON.stringify(snapshotMethods) !== JSON.stringify(exactSnapshotMethods),
		"P07B_A2_SNAPSHOT_API_DRIFT", snapshotMethods.join(","));
	const snapshotBody = functionBody(promotion, "OpenPortableCompilationSnapshot");
	add(/\.\s*(?:Publish|Advance[A-Za-z0-9_]*|CreateStudy|Write|Materialize|Rename)\b/u.test(snapshotBody),
		"P07B_A2_SNAPSHOT_STORE_WRITE", "portable snapshot contains a store write or head transition");
	const compactSnapshot = compact(snapshotBody);
	for (const anchor of [
		"freshInspection,err:=choice.InspectPortableRuling(decision)",
		"freshInspection.DecisionDigest()!=preparation.inspection.DecisionDigest()",
		"freshInspection.ProfileDigest()!=preparation.inspection.ProfileDigest()",
		"profile:=choicepoint.PortableProfile()",
		"profile.Digest()!=freshInspection.ProfileDigest()",
		"profile:profile,inspection:freshInspection",
	]) add(!compactSnapshot.includes(anchor), "P07B_A2_SNAPSHOT_FRESH_INSPECTION_DATAFLOW", anchor);
	const compactSnapshotValid = compact(functionBody(promotion, "Valid", "PortableCompilationSnapshot"));
	for (const anchor of [
		"reparsedProfile:=reparsedChoicepoint.PortableProfile()",
		"reparsedProfile.Digest()!=s.profile.Digest()",
		"!bytes.Equal(reparsedProfile.CanonicalBytes(),s.profile.CanonicalBytes())",
	]) add(!compactSnapshotValid.includes(anchor), "P07B_A2_SNAPSHOT_PROFILE_VALID_DATAFLOW", anchor);
	const expectedSnapshotCalls = [
		"ValidatePortableRulingPreparation", "equalStrings", "mapSnapshotCurrentness", "openRulingAtHead", "refuse",
	].sort();
	const actualSnapshotCalls = unqualifiedCallNames(snapshotBody);
	add(JSON.stringify(actualSnapshotCalls) !== JSON.stringify(expectedSnapshotCalls),
		"P07B_A2_SNAPSHOT_CALL_GRAPH_DRIFT", actualSnapshotCalls.join(","));
	const snapshotReturningFunctions = api(promotionEntry.path).exportedFuncs.filter((name) =>
		functionHeader(promotion, name).includes("PortableCompilationSnapshot"));
	add(JSON.stringify(snapshotReturningFunctions) !== JSON.stringify(["OpenPortableCompilationSnapshot"]),
		"P07B_A2_SNAPSHOT_PUBLIC_API_DRIFT", snapshotReturningFunctions.join(","));
	add(/\b(?:type|var)\s+[A-Z][A-Za-z0-9_]*[^\n]*PortableCompilationSnapshot/u.test(promotion),
		"P07B_A2_SNAPSHOT_PUBLIC_API_DRIFT", "exported alias or variable exposes snapshot authority");
	add(countMatches(promotion, /\bPortableCompilationSnapshot\b/gu) !== 20,
		"P07B_A2_SNAPSHOT_PUBLIC_API_DRIFT", "snapshot type reference roster changed");
	const exactPromotionFunctions = [
		"Finalize", "OpenConfirmation", "OpenPortableCompilationSnapshot", "OpenReady", "OpenRuling",
		"PersistConfirmation", "PreparePortableRuling", "Promote", "ValidatePortableRulingPreparation",
	].sort();
	const promotionFunctions = api(promotionEntry.path).exportedFuncs;
	add(JSON.stringify(promotionFunctions) !== JSON.stringify(exactPromotionFunctions),
		"P07B_A2_PROMOTION_TOP_LEVEL_API_DRIFT", promotionFunctions.join(","));
	const exactPromotionMethods = [
		"Error.Error", "Error.Unwrap",
		"PortableCompilationSnapshot.ChoicepointRecord", "PortableCompilationSnapshot.ConfirmationRecord",
		"PortableCompilationSnapshot.DecisionRecord", "PortableCompilationSnapshot.PortableProfile",
		"PortableCompilationSnapshot.ProfileDigest", "PortableCompilationSnapshot.Valid",
		"PortableRulingPreparation.DecisionDigest", "PortableRulingPreparation.ProfileDigest",
		"PortableRulingPreparation.SelectedFields", "PortableRulingPreparation.Valid",
		"Ready.Record", "Ready.StudyID", "Ruling.Record", "Ruling.StudyID",
		"StoredConfirmation.Record", "StoredConfirmation.StudyID",
	].sort();
	const promotionMethods = api(promotionEntry.path).exportedMethods;
	add(JSON.stringify(promotionMethods) !== JSON.stringify(exactPromotionMethods),
		"P07B_A2_PROMOTION_EXPORTED_METHOD_DRIFT", promotionMethods.join(","));
	const promotionStoreSelectors = [...promotion.matchAll(/\.\s*((?:Advance[A-Za-z0-9_]*|Publish|CreateStudy|Write|Materialize|Rename))\b/gu)]
		.map((match) => match[1]).sort();
	add(JSON.stringify(promotionStoreSelectors) !== JSON.stringify(["AdvanceChoicepoint", "AdvanceConfirmation", "AdvanceRuling"]),
		"P07B_A2_PROMOTION_STORE_WRITE_ROSTER", promotionStoreSelectors.join(","));

	const forbiddenDirect = new Set([
		"os", "io/fs", "path/filepath", "time", "runtime", "crypto/rand", "math/rand", "os/exec", "net", "net/http", "net/url",
		"plugin", "syscall", "unsafe",
	]);
	for (const entry of [predicate, sourceProfile, compilation]) {
		for (const imported of imports(entry)) {
			add(forbiddenDirect.has(imported) || imported.startsWith("net/"), "P07B_A2_PURE_FORBIDDEN_IMPORT", `${entry.path}:${imported}`);
		}
	}
	for (const imported of imports(service)) {
		add(forbiddenDirect.has(imported) || imported.startsWith("net/"), "P07B_A2_SERVICE_FORBIDDEN_IMPORT", imported);
	}

	const allowedPureInternal = new Set([
		"internal/canon", "internal/contractsource", "internal/domain", "internal/portablevalue", "internal/projectionprofile",
		"internal/emit/node/model", "internal/runnerprofile",
	]);
	for (const entry of [predicate, sourceProfile, compilation]) {
		for (const imported of imports(entry)) {
			if (imported.startsWith(modulePrefix)) {
				const internal = imported.slice(modulePrefix.length);
				add(!allowedPureInternal.has(internal), "P07B_A2_PURE_TRANSITIVE_FORBIDDEN_DEPENDENCY", `${entry.path}:${internal}`);
			}
		}
	}
	const expectedServiceImports = [
		"bytes", "context", "errors", "fmt",
		`${modulePrefix}internal/choice`, `${modulePrefix}internal/choice/promotion`, `${modulePrefix}internal/compare`,
		`${modulePrefix}internal/contractsource`, `${modulePrefix}internal/domain`,
		`${modulePrefix}internal/emit/node/internal/compilation`, `${modulePrefix}internal/emit/node/model`,
		`${modulePrefix}internal/portablevalue`, `${modulePrefix}internal/projectiontranslate`, `${modulePrefix}internal/store`,
	].sort();
	const actualServiceImports = imports(service).sort();
	add(JSON.stringify(actualServiceImports) !== JSON.stringify(expectedServiceImports),
		"P07B_A2_SERVICE_IMPORT_ROSTER", actualServiceImports.join(","));
	for (const [entry, expected] of [
		[predicate, [
			"bytes", "encoding/base64", "errors", "fmt", "sort",
			`${modulePrefix}internal/canon`, `${modulePrefix}internal/domain`,
			`${modulePrefix}internal/portablevalue`, `${modulePrefix}internal/projectionprofile`,
		]],
		[sourceProfile, [
			"bytes", "slices", `${modulePrefix}internal/canon`, `${modulePrefix}internal/contractsource`,
			`${modulePrefix}internal/domain`, `${modulePrefix}internal/runnerprofile`,
		]],
		[compilation, [
			"bytes", "encoding/base64", `${modulePrefix}internal/canon`, `${modulePrefix}internal/contractsource`,
			`${modulePrefix}internal/domain`, `${modulePrefix}internal/emit/node/model`,
		]],
	]) {
		const actual = imports(entry).sort();
		add(JSON.stringify(actual) !== JSON.stringify([...expected].sort()),
			"P07B_A2_PURE_IMPORT_ROSTER", `${entry.path}:${actual.join(",")}`);
	}

	const inputBody = structBody(compilation.lexical.code, "Input");
	add(inputBody === "", "P07B_A2_INPUT_MISSING", "sealed Input struct");
	const exactInputWireFields = [
		'SchemaVersion string `json:"schema_version"`', 'Kind string `json:"kind"`',
		'DecisionRecordDigest string `json:"decision_record_digest"`',
		'ChoicepointDigest string `json:"choicepoint_digest"`', 'DecisionAction string `json:"decision_action"`',
		'PortableSourceDigest string `json:"portable_source_digest"`',
		'PortableSourceBase64 string `json:"portable_source_base64"`',
		'SourceProfileDigest string `json:"source_profile_digest"`',
		'SourceProfileBase64 string `json:"source_profile_base64"`',
		'PredicateBase64 string `json:"predicate_base64"`',
	];
	const inputWireFields = normalizedStructFields(compilation.source, "inputWire");
	add(JSON.stringify(inputWireFields) !== JSON.stringify(exactInputWireFields),
		"P07B_A2_INPUT_WIRE_DRIFT", inputWireFields.join(","));
	const exactInputFields = [
		"decision domain.Digest", "choicepoint domain.Digest", "action DecisionAction",
		"source contractsource.PortableSource", "profile model.SourceProfile", "predicate model.Predicate",
		"digest domain.Digest", "canonical []byte", "seal *inputSeal",
	];
	const inputFields = normalizedStructFields(compilation.lexical.code, "Input");
	add(JSON.stringify(inputFields) !== JSON.stringify(exactInputFields),
		"P07B_A2_INPUT_AUTHORITY_LEAK", inputFields.join(","));
	add(normalizedStructFields(compilation.lexical.code, "inputSeal").length !== 0,
		"P07B_A2_INPUT_SEAL_DRIFT", normalizedStructFields(compilation.lexical.code, "inputSeal").join(","));
	add(!inputBody.includes("seal") || !functionBody(compilation.lexical.code, "Valid").includes("New"),
	"P07B_A2_INPUT_NOT_SEALED", "Input.Valid must reconstruct through New");
	const exportedInputConstructors = [...compilation.lexical.code.matchAll(/\bfunc\s+([A-Z][A-Za-z0-9_]*)\s*\(/gu)].map((match) => match[1]);
	add(exportedInputConstructors.some((name) => name !== "New"), "P07B_A2_GENERIC_INPUT_CONSTRUCTOR", exportedInputConstructors.join(","));
	add([...compilation.lexical.code.matchAll(/\bfunc\s+[A-Z][A-Za-z0-9_]*\s*\([^)]*map\s*\[/gu)].length > 0,
	"P07B_A2_MAP_CONSTRUCTOR", "public constructor accepts generic map");
	const exactInputMethods = [
		"Valid() bool", "Digest() domain.Digest", "CanonicalBytes() []byte", "DecisionRecordDigest() domain.Digest",
		"ChoicepointDigest() domain.Digest", "Action() DecisionAction", "Source() contractsource.PortableSource",
		"SourceProfile() model.SourceProfile", "Predicate() model.Predicate",
	].sort();
	const inputMethods = exportedReceiverSignatures(compilation.lexical.code, "Input");
	add(inputMethods.some((signature) => /^(?:New|Parse|Open|Decode|From)/u.test(signature)),
		"P07B_A2_GENERIC_INPUT_CONSTRUCTOR", inputMethods.join(","));
	add(JSON.stringify(inputMethods) !== JSON.stringify(exactInputMethods),
		"P07B_A2_INPUT_EXPORTED_API_DRIFT", inputMethods.join(","));
	add(/\bvar\s+[A-Z][A-Za-z0-9_]*\s*=\s*func\s*\(/u.test(compilation.lexical.code),
		"P07B_A2_GENERIC_INPUT_CONSTRUCTOR", "exported function variable");
	add(compact(functionHeader(compilation.lexical.code, "New")) !==
		"funcNew(decision,choicepointdomain.Digest,actionDecisionAction,sourcecontractsource.PortableSource,profilemodel.SourceProfile,predicatemodel.Predicate,)(Input,error)",
	"P07B_A2_INPUT_CONSTRUCTOR_SIGNATURE", compact(functionHeader(compilation.lexical.code, "New")));
	add(countMatches(compact(functionBody(compilation.lexical.code, "Valid")),
		/New\(i\.decision,i\.choicepoint,i\.action,i\.source,i\.profile,i\.predicate\)/gu) !== 1,
	"P07B_A2_INPUT_VALID_DATAFLOW", "Input.Valid must reconstruct from exactly its sealed fields");
	add(JSON.stringify(api(compilation.path).funcs) !== JSON.stringify(["New"]),
		"P07B_A2_COMPILATION_TOP_LEVEL_API_DRIFT", api(compilation.path).funcs.join(","));
	for (const [entry, expected, code] of [
		[promotionEntry, {
			const: ["CodeRefineRequiresSuccessorStudy", "CodeStaleChoicepoint"],
			type: ["ChoicepointRequest", "Error", "PortableCompilationSnapshot", "PortableRulingPreparation", "Ready", "Ruling", "StoredConfirmation"],
			var: [],
		}, "P07B_A2_PROMOTION_EXPORTED_DECLARATION_DRIFT"],
		[service, {
			const: ["CodeConfirmationBindingMismatch", "CodeInvalidPreparation", "CodeProjectionMismatch", "CodeRulingPartitionMismatch", "CodeSanitizedInputInvalid", "CodeSourceRulingMismatch"],
			type: ["Error", "PreparedCompilation"], var: [],
		}, "P07B_A2_NODE_EXPORTED_DECLARATION_DRIFT"],
		[compilation, {
			const: ["ActionAllowObserved", "ActionCustomExpectation"], type: ["DecisionAction", "Input"], var: [],
		}, "P07B_A2_COMPILATION_EXPORTED_DECLARATION_DRIFT"],
		[predicate, {
			const: ["PredicateKindV1", "PredicateScopeV1"], type: ["Error", "ExactField", "ExactTuple", "ExactValue", "Predicate"], var: [],
		}, "P07B_A2_MODEL_EXPORTED_DECLARATION_DRIFT"],
		[sourceProfile, {
			const: ["ContractSourceProfileDomain", "DeclaredSourceScopeV1", "RuntimeFamilyNodeV1"], type: ["SourceProfile"], var: [],
		}, "P07B_A2_MODEL_EXPORTED_DECLARATION_DRIFT"],
	]) {
		for (const keyword of ["const", "type", "var"]) {
			const actual = api(entry.path)[keyword];
			add(JSON.stringify(actual) !== JSON.stringify(expected[keyword]), code, `${keyword}:${actual.join(",")}`);
		}
	}
	for (const [entry, expected, code] of [
		[service, [
			"Error.Error", "Error.Unwrap", "PreparedCompilation.Action", "PreparedCompilation.AllowedTupleCanonicalBytes",
			"PreparedCompilation.ChoicepointDigest", "PreparedCompilation.DecisionRecordDigest", "PreparedCompilation.Digest",
			"PreparedCompilation.SelectedFields", "PreparedCompilation.SourceDigest", "PreparedCompilation.SourceProfileDigest",
			"PreparedCompilation.Valid",
		], "P07B_A2_NODE_EXPORTED_DECLARATION_DRIFT"],
		[compilation, [
			"Input.Action", "Input.CanonicalBytes", "Input.ChoicepointDigest", "Input.DecisionRecordDigest", "Input.Digest",
			"Input.Predicate", "Input.Source", "Input.SourceProfile", "Input.Valid",
		], "P07B_A2_COMPILATION_EXPORTED_DECLARATION_DRIFT"],
		[predicate, [
			"Error.Error", "Error.Unwrap", "ExactField.FieldID", "ExactField.Value", "ExactTuple.CanonicalBytes",
			"ExactTuple.Fields", "ExactTuple.Valid", "ExactValue.CanonicalBytes", "ExactValue.PortableValue", "ExactValue.Tag",
			"ExactValue.Valid", "Predicate.AllowedTuples", "Predicate.CanonicalBytes", "Predicate.PortableProfileDigest",
			"Predicate.SelectedFields", "Predicate.StimulusDigest", "Predicate.Valid", "Predicate.ValidFor",
		], "P07B_A2_MODEL_EXPORTED_DECLARATION_DRIFT"],
		[sourceProfile, [
			"SourceProfile.AdapterDomain", "SourceProfile.CanonicalBytes", "SourceProfile.Digest", "SourceProfile.StartProfile",
			"SourceProfile.SubjectEntrypoint", "SourceProfile.Valid", "SourceProfile.ValidFor",
		], "P07B_A2_MODEL_EXPORTED_DECLARATION_DRIFT"],
	]) {
		const actual = api(entry.path).exportedMethods;
		add(JSON.stringify(actual) !== JSON.stringify([...expected].sort()), code, `methods:${actual.join(",")}`);
	}
	for (const [entry, typeName, code] of [
		[service, "Error", "P07B_A2_NODE_EXPORTED_DECLARATION_DRIFT"],
		[predicate, "Error", "P07B_A2_MODEL_EXPORTED_DECLARATION_DRIFT"],
	]) {
		const actual = exportedReceiverSignatures(entry.lexical.code, typeName);
		add(JSON.stringify(actual) !== JSON.stringify(["Error() string", "Unwrap() error"]), code, `${typeName}:${actual.join(",")}`);
	}
	for (const [typeName, expected] of [
		["ExactValue", ["value portablevalue.Value", "canonical []byte"]],
		["ExactField", ["id string", "value ExactValue"]],
		["ExactTuple", ["fields []ExactField", "canonical []byte"]],
		["Predicate", ["stimulusDigest domain.Digest", "profile projectionprofile.Profile", "selected []string", "allowed []ExactTuple", "canonical []byte"]],
		["SourceProfile", ["adapter domain.AdapterDomain", "entrypoint string", "start string", "digest domain.Digest", "canonical []byte"]],
	]) {
		const actual = normalizedStructFields(typeName === "SourceProfile" ? sourceProfile.lexical.code : predicate.lexical.code, typeName);
		add(JSON.stringify(actual) !== JSON.stringify(expected), "P07B_A2_PURE_MODEL_STORAGE_DRIFT", `${typeName}:${actual.join(",")}`);
	}
	const exactPredicateWireFields = [
		'Kind string `json:"kind"`', 'Scope string `json:"scope"`',
		'StimulusDigest string `json:"stimulus_digest"`',
		'PortableProfileDigest string `json:"portable_profile_digest"`',
		'SelectedFields []string `json:"selected_fields"`',
		'AllowedTuples []exactTupleWire `json:"allowed_tuples"`',
	];
	const predicateWireFields = normalizedStructFields(predicate.source, "predicateWire");
	add(JSON.stringify(predicateWireFields) !== JSON.stringify(exactPredicateWireFields),
		"P07B_A2_PREDICATE_WIRE_DRIFT", predicateWireFields.join(","));
	add(compact(functionHeader(predicate.lexical.code, "NewPredicate")) !==
		"funcNewPredicate(stimulusDigestdomain.Digest,profileprojectionprofile.Profile,selectedFields[]string,allowedTuples[]ExactTuple,)(Predicate,error)",
	"P07B_A2_PREDICATE_CONSTRUCTOR_SIGNATURE", compact(functionHeader(predicate.lexical.code, "NewPredicate")));
	add(countMatches(compact(predicate.lexical.code),
		/NewPredicate\(p\.stimulusDigest,p\.profile,p\.selected,p\.allowed\)/gu) !== 1,
		"P07B_A2_PREDICATE_VALID_DATAFLOW", "Predicate.Valid must reconstruct from exactly its sealed fields");
	add(JSON.stringify(api(predicate.path).exportedFuncs) !==
		JSON.stringify(["IsCode", "NewExactField", "NewExactTuple", "NewExactValue", "NewPredicate"].sort()),
	"P07B_A2_MODEL_TOP_LEVEL_API_DRIFT", api(predicate.path).exportedFuncs.join(","));
	add(JSON.stringify(api(sourceProfile.path).exportedFuncs) !== JSON.stringify(["NewSourceProfile"]),
		"P07B_A2_MODEL_TOP_LEVEL_API_DRIFT", api(sourceProfile.path).exportedFuncs.join(","));
	add(compact(functionHeader(sourceProfile.lexical.code, "NewSourceProfile")) !==
		"funcNewSourceProfile(sourcecontractsource.PortableSource)(SourceProfile,error)",
		"P07B_A2_SOURCE_PROFILE_CONSTRUCTOR_SIGNATURE", compact(functionHeader(sourceProfile.lexical.code, "NewSourceProfile")));
	const exactSourceProfileMethods = [
		"Valid() bool", "ValidFor(source contractsource.PortableSource) bool", "Digest() domain.Digest", "CanonicalBytes() []byte",
		"AdapterDomain() domain.AdapterDomain", "SubjectEntrypoint() string", "StartProfile() string",
	].sort();
	const sourceProfileMethods = exportedReceiverSignatures(sourceProfile.lexical.code, "SourceProfile");
	add(JSON.stringify(sourceProfileMethods) !== JSON.stringify(exactSourceProfileMethods),
		"P07B_A2_SOURCE_PROFILE_EXPORTED_API_DRIFT", sourceProfileMethods.join(","));

	const productionCode = [predicate, sourceProfile, compilation, service].map((entry) => entry.lexical.code).join("\n");
	add(countMatches(productionCode, /\bcompilation\s*\.\s*New\s*\(/gu) !== 1,
	"P07B_A2_INPUT_CALLSITE_DRIFT", "compilation.New production call count must equal one");
	const prepareBody = functionBody(service.lexical.code, "PrepareCompilation");
	const compactPrepare = compact(prepareBody);
	const translationCall = "projectiontranslate.TranslateConfirmed(";
	const translationIndex = prepareBody.indexOf(translationCall);
	const partitionIndex = prepareBody.indexOf("revalidatePartition(", translationIndex + translationCall.length);
	add(countMatches(prepareBody, /\bprojectiontranslate\s*\.\s*TranslateConfirmed\s*\(/gu) !== 1 ||
		translationIndex < 0 || partitionIndex < translationIndex,
		"P07B_A2_RETRANSLATION_ANCHOR_MISSING", "explicit proof-first application retranslation");
	add(!compactPrepare.includes("translations,err:=projectiontranslate.TranslateConfirmed(exactSource.ProjectionBinding(),confirmed.ProjectionRoster(),proofs,)") ||
		!compactPrepare.includes("predicate,err:=revalidatePartition(decision,translations,exactSource)"),
	"P07B_A2_RETRANSLATION_DATAFLOW", "translated proof result must feed exact ruling partition validation");
	add(prepareBody === "", "P07B_A2_PREPARE_MISSING", "PrepareCompilation");
	for (const anchor of [
		"exactSource,err:=contractsource.Parse(source.CanonicalBytes())",
		"exactSource.Digest()!=source.Digest()",
		"!bytes.Equal(exactSource.CanonicalBytes(),source.CanonicalBytes())",
		"input,err:=compilation.New(decision.Digest(),choicepoint.Digest(),action,exactSource,declaredProfile,predicate,)",
	]) add(!compactPrepare.includes(anchor), "P07B_A2_PREPARE_EXACT_DATAFLOW", anchor);
	for (const anchor of [
		"translatedProfile:=translations.Profile()", "sourceProfile:=exactSource.Profile()",
		"preparedProfile:=snapshot.PortableProfile()", "!preparedProfile.Valid()",
		"snapshot.ProfileDigest()!=translatedProfile.Digest()",
		"preparedProfile.Digest()!=translatedProfile.Digest()",
		"sourceProfile.Digest()!=translatedProfile.Digest()",
		"!bytes.Equal(preparedProfile.CanonicalBytes(),translatedProfile.CanonicalBytes())",
		"!bytes.Equal(sourceProfile.CanonicalBytes(),translatedProfile.CanonicalBytes())",
	]) add(!compactPrepare.includes(anchor), "P07B_A2_PREPARED_PROFILE_BYTE_JOIN", anchor);
	add(!prepareBody.includes("requireSourceChoicepointJoin(exactSource, choicepoint)") ||
		!prepareBody.includes("binding != exactSource.ExecutionBindingDigest()") ||
		!prepareBody.includes("confirmationRecord.ExecutionBindingDigests()"),
		"P07B_A2_AUTHORITY_JOIN_MISSING", "source/choicepoint/confirmation exact join");
	add(/\.\s*(?:Publish|Advance[A-Za-z0-9_]*|CreateStudy|Write|Materialize|Rename)\b/u.test(service.lexical.code),
	"P07B_A2_STORE_WRITE", "node service contains a store write or transition");
	add(/\b(?:os|exec|net|time|runtime|rand|filepath)\s*\./u.test(service.lexical.code),
		"P07B_A2_SIDE_EFFECT", "node service contains a forbidden side-effect edge");
	const compactJoin = compact(functionBody(service.lexical.code, "requireSourceChoicepointJoin"));
	for (const anchor of [
		"plan.Digest()!=sourcePlan.Digest()", "!bytes.Equal(plan.CanonicalBytes(),sourcePlan.CanonicalBytes())",
		"plan.Adapter().Domain!=source.Adapter()",
		"plan.ProjectionDefinitionBinding().Digest()!=source.ProjectionBinding().Digest()",
		"!bytes.Equal(plan.ProjectionDefinitionBinding().CanonicalBytes(),source.ProjectionBinding().CanonicalBytes())",
		"stimulus.Kind()!=source.StimulusKind()", "stimulus.Digest()!=source.StimulusDigest()",
		"!bytes.Equal(stimulus.CanonicalBytes(),source.StimulusCanonicalBytes())",
	]) add(!compactJoin.includes(anchor), "P07B_A2_SOURCE_JOIN_DATAFLOW", anchor);
	const exactServiceFunctions = [
		"IsCode", "PrepareCompilation", "choiceTupleSet", "compilationAction", "dedupeTuples", "equalStrings",
		"exactReferencePartition", "outcomeIdentity", "portableFromChoice", "referenceTupleSet", "refuse",
		"requireSourceChoicepointJoin", "revalidatePartition", "sameTupleSet", "translatedSelectedTuple",
	].sort();
	const actualServiceFunctions = api(service.path).funcs;
	add(JSON.stringify(actualServiceFunctions) !== JSON.stringify(exactServiceFunctions),
		"P07B_A2_NODE_TOP_LEVEL_API_DRIFT", actualServiceFunctions.join(","));

	const exactPreparedFields = [
		"preparation promotion.PortableRulingPreparation", "input compilation.Input", "seal *preparedCompilationSeal",
	];
	const preparedFields = normalizedStructFields(service.lexical.code, "PreparedCompilation");
	add(JSON.stringify(preparedFields) !== JSON.stringify(exactPreparedFields),
		"P07B_A2_PREPARED_WRAPPER_INCOMPLETE", preparedFields.join(","));
	const exactPreparedMethods = [
		"Valid() bool", "Digest() domain.Digest", "DecisionRecordDigest() domain.Digest", "ChoicepointDigest() domain.Digest",
		"SourceDigest() domain.Digest", "SourceProfileDigest() domain.Digest", "Action() string", "SelectedFields() []string",
		"AllowedTupleCanonicalBytes() [][]byte",
	].sort();
	const preparedMethods = exportedReceiverSignatures(service.lexical.code, "PreparedCompilation");
	add(JSON.stringify(preparedMethods) !== JSON.stringify(exactPreparedMethods),
		"P07B_A2_PREPARED_AUTHORITY_EXPOSED", preparedMethods.join(","));

	const sourceWire = structBody(sourceProfile.lexical.code, "sourceProfileWire");
	const sourceMembers = ["RuntimeFamily", "SemanticProfile", "AdapterDomain", "LaunchProfile", "SubjectEntrypoint", "StartProfile", "Scope"];
	add(sourceMembers.some((member) => !new RegExp(`\\b${member}\\b`, "u").test(sourceWire)) ||
		countMatches(sourceWire, /^\s*[A-Z][A-Za-z0-9_]*\s+/gmu) !== 7,
	"P07B_A2_SOURCE_PROFILE_ROSTER", sourceWire.trim());
	add(!sourceProfile.lexical.code.includes("canon.DigestTyped(ContractSourceProfileDomain"),
	"P07B_A2_SOURCE_PROFILE_DOMAIN", "ContractSourceProfile exact typed digest");
	add(!sourceProfile.lexical.literals.includes("ContractSourceProfile"),
		"P07B_A2_SOURCE_PROFILE_LITERAL", "ContractSourceProfile literal missing");
	const exactSourceWireFields = [
		'RuntimeFamily string `json:"runtime_family"`', 'SemanticProfile string `json:"semantic_profile"`',
		'AdapterDomain string `json:"adapter_domain"`', 'LaunchProfile string `json:"launch_profile"`',
		'SubjectEntrypoint string `json:"subject_entrypoint"`', 'StartProfile string `json:"start_profile"`',
		'Scope string `json:"scope"`',
	];
	const sourceWireFields = normalizedStructFields(sourceProfile.source, "sourceProfileWire");
	add(JSON.stringify(sourceWireFields) !== JSON.stringify(exactSourceWireFields),
		"P07B_A2_SOURCE_PROFILE_ROSTER", sourceWireFields.join(","));
	const normalizedSourceProfile = sourceProfile.source.replace(/\s+/gu, " ");
	for (const anchor of [
		'RuntimeFamilyNodeV1 = "NODE"',
		'DeclaredSourceScopeV1 = "DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE"',
		'ContractSourceProfileDomain = "ContractSourceProfile"',
		'httpPortableStartProfileV1 = "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1"',
	]) add(!normalizedSourceProfile.includes(anchor), "P07B_A2_SOURCE_PROFILE_CONSTANT_DRIFT", anchor);
	add(!at("internal/emit/node/model/predicate_test.go").source.includes("sourceProfileGoldenDigest"),
		"P07B_A2_SOURCE_PROFILE_GOLDEN_MISSING", "independent body/domain digest golden");
	add(!predicate.lexical.code.includes("bytes.Compare") || !predicate.lexical.code.includes("unique[string(tuple.canonical)]"),
		"P07B_A2_TUPLE_CANONICAL_SET", "byte dedupe plus unsigned canonical ordering");

	for (const forbidden of ["ContractBundle", "manifest.json", "RESIDUE", "Wake", "didrun", "GradeVerbatim"]) {
		add(productionCode.includes(forbidden) || [predicate, sourceProfile, compilation, service].some((entry) => entry.lexical.literals.some((literal) => literal.includes(forbidden))),
			"P07B_A2_PREMATURE_OR_FOREIGN_AUTHORITY", forbidden);
	}
	const inheritedAuthorityCode = [choicepointEntry, promotionEntry, confirmationEntry, worldEntry];
	for (const forbidden of ["ContractBundle", "manifest.json", "RESIDUE", "Wake"]) {
		add(inheritedAuthorityCode.some((entry) => entry.lexical.code.includes(forbidden) || entry.lexical.literals.some((literal) => literal.includes(forbidden))),
			"P07B_A2_PREMATURE_OR_FOREIGN_AUTHORITY", `authority:${forbidden}`);
	}

	for (const [path, test] of [
		["internal/confirmation/service_test.go", "TestFreshConfirmationRetainsExactScheduleOrderedExecutionBindingsWithoutWireChange"],
		["internal/choice/session_roundtrip_test.go", "TestChoicepointAuthorityGettersAreExactAndDefensiveForLegacyAndPortableRecords"],
		["internal/emit/node/model/predicate_test.go", "TestExactValueWireCoversClosedPortableAlgebra"],
		["internal/emit/node/model/predicate_test.go", "TestPredicateCanonicalizesCompleteTupleSetByUnsignedBytes"],
		["internal/emit/node/model/predicate_test.go", "TestSourceProfileUsesExactSevenMemberContractDomain"],
		["internal/emit/node/model/predicate_test.go", "FuzzExactOpaqueBytesRemainExact"],
	]) add(!at(path).lexical.code.includes(`func ${test}(`), "P07B_A2_REQUIRED_TEST_MISSING", `${path}:${test}`);

	for (const [path, anchor] of [
		["testkit/studies/cli_precedence/reduction_darwin_test.go", "nodeemit.PrepareCompilation"],
		["testkit/studies/cli_precedence/reduction_darwin_test.go", "seenCorrelated"],
		["testkit/studies/cli_precedence/reduction_darwin_test.go", "planMismatchConfig.Repetitions = 2"],
		["testkit/studies/cli_precedence/reduction_darwin_test.go", "parsedPlanMismatchSource"],
		["testkit/studies/http_invoices/reduction_darwin_test.go", "config.PortableStart = true"],
		["testkit/studies/http_invoices/reduction_darwin_test.go", "allowPrepared"],
	]) add(!at(path).lexical.code.includes(anchor), "P07B_A2_PHYSICAL_TEST_MISSING", `${path}:${anchor}`);

	for (const restart of [
		{
			path: "testkit/studies/cli_precedence/reduction_darwin_test.go", prefix: "CLI",
			parent: "assertCLICompilationFreshProcessRestart", helper: "TestCLICompilationFreshProcessRestartHelper",
			env: "cliA21RestartEnv", study: "cliA21StudyLabel",
		},
		{
			path: "testkit/studies/http_invoices/reduction_darwin_test.go", prefix: "HTTP",
			parent: "assertHTTPCompilationFreshProcessRestart", helper: "TestHTTPCompilationFreshProcessRestartHelper",
			env: "httpA21RestartEnv", study: "httpA21StudyLabel",
		},
	]) {
		const entry = at(restart.path);
		const parent = functionBody(entry.lexical.code, restart.parent);
		const helper = functionBody(entry.lexical.code, restart.helper);
		add(parent === "" || helper === "", "P07B_A2_FRESH_PROCESS_RESTART_MISSING", restart.prefix);
		for (const anchor of [
			"exec.CommandContext(", "os.Executable()", "command.Dir = childCWD", `command.Env = []string{${restart.env} +`, "+ requestPath}",
			"prepared.Digest().String()", "prepared.SourceProfileDigest().String()",
			"prepared.AllowedTupleCanonicalBytes()", "base64.StdEncoding.Strict().DecodeString(",
		]) add(!parent.includes(anchor), "P07B_A2_FRESH_PROCESS_PARENT_INCOMPLETE", `${restart.prefix}:${anchor}`);
		add(countMatches(parent, /\bcommand\s*\.\s*Env\s*=/gu) !== 1 || parent.includes("os.Environ"),
			"P07B_A2_FRESH_PROCESS_ENVIRONMENT_DRIFT", restart.prefix);
		for (const anchor of [
			"contractsource.Parse(", `store.NewStudyID(${restart.study})`, "store.OpenObjectStore(",
			"promotion.OpenRuling(", "promotion.PreparePortableRuling(", "nodeemit.PrepareCompilation(",
			"prepared.SourceProfileDigest().String()", "prepared.AllowedTupleCanonicalBytes()",
		]) add(!helper.includes(anchor), "P07B_A2_FRESH_PROCESS_CHILD_INCOMPLETE", `${restart.prefix}:${anchor}`);
		add(!entry.lexical.literals.includes("countershape/a2.1/restart/v1"),
			"P07B_A2_FRESH_PROCESS_SCHEMA_MISSING", restart.prefix);
	}
	const cliStudy = at("testkit/studies/cli_precedence/study_darwin_test.go");
	const cliExecutableHelper = functionBody(cliStudy.lexical.code, "cliStudyTestExecutable");
	add(cliExecutableHelper === "" || !cliExecutableHelper.includes("os.Getenv") ||
		!cliExecutableHelper.includes("filepath.IsAbs") || !cliExecutableHelper.includes("filepath.EvalSymlinks"),
	"P07B_A2_CLI_TOOL_AUTHORITY_MISSING", "CLI physical helper must honor one explicit absolute admitted tool path");

	return violations;
}

async function main() {
	if (process.argv.length !== 2) throw new ArchitectureError("P07B_A2_ARGUMENTS", "no arguments accepted");
	const go = await inspectGoExecutable(process.env.COUNTERSHAPE_GO);
	const goEnvironment = closedGoEnvironment(go.path);
	await withGoExecutable(go, () => runInheritedA1(go, goEnvironment));
	if (!fixtureOverride) {
		await withGoExecutable(go, () => runPureDependencyClosure(go, goEnvironment));
		await withGoExecutable(go, () => runGoDirectImportRosters(go, goEnvironment));
	}
	await inspectStaticPureDependencyClosure();
	const initial = await manifest();
	const publicAPI = await withGoExecutable(go, () => goPublicAPIRosters(initial.entries, go, goEnvironment));
	const violations = inspect(initial, publicAPI);
	const final = await manifest();
	if (initial.digest !== final.digest) {
		throw new ArchitectureError("P07B_A2_MANIFEST_CHANGED", `${initial.digest}->${final.digest}`);
	}
	if (violations.length > 0) {
		const detail = violations.map(([code, message]) => `${code}: ${message}`).sort().join("\n");
		throw new ArchitectureError("P07B_A2_ARCHITECTURE_VIOLATION", detail);
	}
	const marker = fixtureOverride ? "P07B A2.1 architecture fixture OK" : "P07B A2.1 architecture boundary OK";
	process.stdout.write(`${marker} (${exactFiles.length} exact files; manifest ${initial.digest})\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
