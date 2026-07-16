#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { lstat, open, readdir } from "node:fs/promises";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const modulePrefix = "github.com/nelsonwerd/countershape/";
const exactFiles = Object.freeze([
	"internal/adapters/cli/model/projection_authority.go",
	"internal/adapters/cli/model/projection_schema.go",
	"internal/adapters/cli/model/projection_schema_test.go",
	"internal/adapters/http/model/binding.go",
	"internal/adapters/http/model/portable_start_test.go",
	"internal/adapters/http/model/projection_authority.go",
	"internal/adapters/http/model/projection_schema.go",
	"internal/adapters/http/model/projection_schema_test.go",
	"internal/adapters/http/model/start.go",
	"internal/adapters/http/stimulus.go",
	"internal/contractsource/errors.go",
	"internal/contractsource/profile_roster.go",
	"internal/contractsource/source.go",
	"internal/contractsource/source_test.go",
	"internal/projectionprofile/profile.go",
	"internal/projectionprofile/profile_test.go",
	"internal/runnerprofile/profile.go",
	"internal/world/http_api.go",
	"internal/world/http_portable_negative_darwin_test.go",
	"internal/world/http_receipt.go",
	"internal/world/http_service.go",
	"internal/world/http_service_darwin.go",
	"internal/world/http_service_darwin_test.go",
	"testkit/httpfixture/fixture.go",
	"testkit/httpfixture/fixture_test.go",
	"testkit/httpfixture/server.mjs",
	"testkit/httpfixture/server_child_bind.mjs",
	"testkit/studies/http_invoices/study.go",
	"testkit/studies/http_invoices/study_darwin_test.go",
	"tools/mutate-p07b.mjs",
]);

class ArchitectureError extends Error {
  constructor(code, detail) {
    super(`${code}: ${detail}`);
    this.code = code;
  }
}

function slash(value) { return value.split(sep).join("/"); }

async function readRegular(relativePath) {
  const absolute = resolve(root, relativePath);
  const fromRoot = relative(root, absolute);
  if (fromRoot === ".." || fromRoot.startsWith(`..${sep}`)) {
    throw new ArchitectureError("P07B_PATH_ESCAPE", relativePath);
  }
  const before = await lstat(absolute);
  if (!before.isFile() || before.isSymbolicLink()) {
    throw new ArchitectureError("P07B_NONREGULAR_AUTHORITY", relativePath);
  }
  let handle;
  try {
    handle = await open(absolute, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
    const opened = await handle.stat();
    if (!opened.isFile() || opened.dev !== before.dev || opened.ino !== before.ino || opened.size !== before.size) {
      throw new ArchitectureError("P07B_AUTHORITY_CHANGED", relativePath);
    }
    const bytes = await handle.readFile();
    const after = await handle.stat();
    if (after.dev !== opened.dev || after.ino !== opened.ino || after.size !== opened.size) {
      throw new ArchitectureError("P07B_AUTHORITY_CHANGED", relativePath);
    }
    return bytes;
  } finally {
    await handle?.close();
  }
}

function utf8(bytes, path) {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    throw new ArchitectureError("P07B_INVALID_UTF8", path);
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
        code[index] = " "; literal = ""; state = character === '"' ? "string" : character === "'" ? "rune" : "raw";
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
  if (state !== "code") throw new ArchitectureError("P07B_LEXICAL_INVALID", `${path}: ${state}`);
  return { code: code.join(""), commentless: commentless.join(""), literals };
}

function imports(commentless) {
  const result = [];
  const declarations = /(?:^|\n)\s*import\s*(?:\(([\s\S]*?)\)|(?:[._A-Za-z][._A-Za-z0-9]*\s+)?"([^"]+)")/gu;
  for (const declaration of commentless.matchAll(declarations)) {
    if (declaration[2]) result.push(declaration[2]);
    else for (const match of declaration[1].matchAll(/(?:^|\s)(?:[._A-Za-z][._A-Za-z0-9]*\s+)?"([^"]+)"/gu)) result.push(match[1]);
  }
  return result;
}

function functionBody(code, name) {
  const match = new RegExp(`\\bfunc\\s+(?:\\([^)]*\\)\\s+)?${name}\\s*\\(`, "u").exec(code);
  if (!match) return "";
  const open = code.indexOf("{", match.index);
  if (open < 0) return "";
  let depth = 0;
  for (let index = open; index < code.length; index += 1) {
    if (code[index] === "{") depth += 1;
    if (code[index] === "}") {
      depth -= 1;
      if (depth === 0) return code.slice(open + 1, index);
    }
  }
  return "";
}

function structBody(code, name) {
  const match = new RegExp(`\\btype\\s+${name}\\s+struct\\s*\\{`, "u").exec(code);
  if (!match) return "";
  const open = code.indexOf("{", match.index);
  let depth = 0;
  for (let index = open; index < code.length; index += 1) {
    if (code[index] === "{") depth += 1;
    if (code[index] === "}") {
      depth -= 1;
      if (depth === 0) return code.slice(open + 1, index);
    }
  }
  return "";
}

function requireIncludes(violations, source, anchor, code, detail) {
  if (!source.includes(anchor)) violations.push([code, detail]);
}

function requireExcludes(violations, source, anchor, code, detail) {
  if (source.includes(anchor)) violations.push([code, detail]);
}

function runInheritedU6() {
  const checker = resolve(root, "tools/check-u6-architecture.mjs");
  const result = spawnSync(process.execPath, [checker], {
		cwd: root, encoding: "utf8", env: { ...process.env, NO_COLOR: "1" },
		timeout: 30_000, maxBuffer: 2 * 1024 * 1024,
	});
  if (result.error || result.signal || result.status !== 0 || !result.stdout.includes("U6 architecture boundary OK")) {
    throw new ArchitectureError("P07B_INHERITED_U6_FAILED", `${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
  }
}

function runSourceDependencyClosure() {
	const go = process.env.COUNTERSHAPE_GO;
	if (!go || !isAbsolute(go)) {
		throw new ArchitectureError("P07B_GO_AUTHORITY_REQUIRED", "COUNTERSHAPE_GO must name one absolute reviewed Go executable");
	}
	const result = spawnSync(go, ["list", "-deps", "-f", "{{.ImportPath}}", "./internal/contractsource"], {
		cwd: root,
		encoding: "utf8",
		timeout: 30_000,
		maxBuffer: 2 * 1024 * 1024,
		env: {
			...process.env, GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local", GOPROXY: "off",
		},
	});
	if (result.error || result.signal || result.status !== 0) {
		throw new ArchitectureError("P07B_DEPENDENCY_QUERY_FAILED", `${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
	}
	const allowed = new Set([
		"internal/adapters/cli/model", "internal/adapters/http/model", "internal/canon", "internal/contractsource",
		"internal/domain", "internal/portablevalue", "internal/projectionprofile", "internal/runnerprofile",
	]);
	for (const line of result.stdout.split(/\r?\n/u)) {
		if (!line.startsWith(modulePrefix)) continue;
		const internal = line.slice(modulePrefix.length);
		if (!allowed.has(internal)) {
			throw new ArchitectureError("P07B_SOURCE_TRANSITIVE_INTERNAL_DEPENDENCY", internal);
		}
	}
}

async function manifest() {
	for (const directory of ["internal/contractsource", "internal/runnerprofile", "testkit/httpfixture"]) {
		const actual = (await readdir(resolve(root, directory))).sort().map((name) => `${directory}/${name}`);
		const expected = exactFiles.filter((path) => dirname(path) === directory).sort();
		if (JSON.stringify(actual) !== JSON.stringify(expected)) {
			throw new ArchitectureError("P07B_PACKAGE_MAP_DRIFT", `${directory}: actual=[${actual.join(",")}] expected=[${expected.join(",")}]`);
		}
	}
  const entries = [];
  const aggregate = createHash("sha256");
	for (const path of exactFiles) {
		const bytes = await readRegular(path);
		aggregate.update(slash(path)); aggregate.update("\0"); aggregate.update(bytes); aggregate.update("\0");
		const source = utf8(bytes, path);
		const parsed = path.endsWith(".go") ? lexical(source, path) : { code: source, commentless: source, literals: [] };
		entries.push({ path, bytes, source, lexical: parsed });
	}
  return { entries, digest: aggregate.digest("hex") };
}

function inspect(input) {
  const violations = [];
  const at = (path) => input.entries.find((entry) => entry.path === path);
	const source = at("internal/contractsource/source.go");
	const errors = at("internal/contractsource/errors.go");
	const profileRoster = at("internal/contractsource/profile_roster.go");
	const start = at("internal/adapters/http/model/start.go");
	const binding = at("internal/adapters/http/model/binding.go");
	const wrapper = at("internal/adapters/http/stimulus.go");
	const cliProjectionAuthority = at("internal/adapters/cli/model/projection_authority.go");
	const cliProjectionSchema = at("internal/adapters/cli/model/projection_schema.go");
	const httpProjectionAuthority = at("internal/adapters/http/model/projection_authority.go");
	const httpProjectionSchema = at("internal/adapters/http/model/projection_schema.go");
	const projectionProfile = at("internal/projectionprofile/profile.go");
	const runnerProfile = at("internal/runnerprofile/profile.go");
	const worldAPI = at("internal/world/http_api.go");
	const worldService = at("internal/world/http_service_darwin.go");
	const worldReceipt = at("internal/world/http_receipt.go");
	const fixtureGo = at("testkit/httpfixture/fixture.go");
	const fixtureTests = at("testkit/httpfixture/fixture_test.go");
	const legacyFixture = at("testkit/httpfixture/server.mjs");
	const portableFixture = at("testkit/httpfixture/server_child_bind.mjs");
	const study = at("testkit/studies/http_invoices/study.go");
	const mutationGate = at("tools/mutate-p07b.mjs");
	const sourceCode = source.lexical.code;
	const sourceTagged = source.lexical.commentless;
	const productionSemantic = source.lexical.code + "\n" + source.lexical.literals.join("\n") +
		"\n" + errors.lexical.code + "\n" + errors.lexical.literals.join("\n") +
		"\n" + profileRoster.lexical.code + "\n" + profileRoster.lexical.literals.join("\n");

	const allowedStandard = new Set(["bytes", "encoding/base64", "encoding/json", "errors", "slices"]);
	const allowedInternal = new Set([
		"internal/adapters/cli/model", "internal/adapters/http/model", "internal/canon", "internal/domain",
		"internal/portablevalue", "internal/projectionprofile", "internal/runnerprofile",
	]);
	for (const entry of [source, errors, profileRoster]) {
    for (const imported of imports(entry.lexical.commentless)) {
      if (imported.startsWith(modulePrefix)) {
        const internal = imported.slice(modulePrefix.length);
        if (!allowedInternal.has(internal)) violations.push(["P07B_SOURCE_IMPORT_NOT_ALLOWED", `${entry.path}: ${internal}`]);
      } else if (!allowedStandard.has(imported)) {
        violations.push(["P07B_SOURCE_IMPORT_NOT_ALLOWED", `${entry.path}: ${imported}`]);
      }
    }
  }

  for (const forbidden of ["os", "path/filepath", "time", "runtime", "os/exec", "net", "internal/store", "internal/gitobj", "internal/choice/promotion"]) {
    if (imports(sourceTagged).some((value) => value === forbidden || value === `${modulePrefix}${forbidden}`)) {
      violations.push(["P07B_SOURCE_IMPURE_IMPORT", forbidden]);
    }
  }

  const portableStruct = structBody(sourceCode, "PortableSource");
  if (!portableStruct || /(?:^|\n)\s*[A-Z][A-Za-z0-9_]*\s+/u.test(portableStruct) || !portableStruct.includes("seal")) {
    violations.push(["P07B_SOURCE_CAPABILITY_OPEN", "PortableSource"]);
  }
  for (const forbiddenConstructor of ["NewPortableSource", "FromDigest", "FromBytesAndDigest", "UnsafePortableSource"]) {
    requireExcludes(violations, sourceCode, `func ${forbiddenConstructor}(`, "P07B_GENERIC_SOURCE_CONSTRUCTOR", forbiddenConstructor);
  }
  for (const constructor of ["NewCLISource", "NewHTTPSource", "Parse"]) {
    const count = (sourceCode.match(new RegExp(`\\bfunc\\s+${constructor}\\s*\\(`, "gu")) ?? []).length;
    if (count !== 1) violations.push(["P07B_SOURCE_CONSTRUCTOR_SURFACE", `${constructor}:${count}`]);
  }

  const cliConstructor = functionBody(sourceTagged, "NewCLISource");
  const httpConstructor = functionBody(sourceTagged, "NewHTTPSource");
	const parser = functionBody(sourceTagged, "Parse");
	const reconstructCLI = functionBody(sourceTagged, "reconstructCLI");
	const reconstructHTTP = functionBody(sourceTagged, "reconstructHTTP");
	const commonIdentity = functionBody(sourceTagged, "commonIdentity");
	for (const [body, anchors, label] of [
		[cliConstructor, ["validateCLIProjection(", "validateNodePlan(", "cli.BindExecution(", "input.Profile", "input.Projection"], "CLI"],
		[httpConstructor, ["HTTPPortableStartAuthorityV1", "PortableReadinessProtocolV1", "validateHTTPProjection(", "counterhttp.BindExecution(", "input.Profile", "input.Projection"], "HTTP"],
		[parser, ["canon.Parse(exact)", "value.CanonicalChecked()", "json.Unmarshal(exact, &identity)", "buildSource(identity)", "bytes.Equal(rebuilt.canonical, exact)"], "PARSE"],
		[reconstructCLI, ["strictBase64(wire.Stdin.BytesBase64", "cli.NewCLIStimulus(", "bytes.Equal(stimulus.CanonicalBytes(), stimulusBytes)", "cli.BindExecution(", "projection"], "CLI_RECONSTRUCT"],
		[reconstructHTTP, ["strictBase64(wire.Body.BytesBase64", "counterhttp.NewHTTPStimulus(", "NewPortableHTTPStartSpec", "NewPortableHTTPReadinessContract", "counterhttp.BindExecution(", "projection"], "HTTP_RECONSTRUCT"],
	]) for (const anchor of anchors) requireIncludes(violations, body, anchor, `P07B_${label}_AUTHORITY_JOIN`, anchor);
	for (const anchor of [
		"AdapterProjectionBase64: base64.StdEncoding.EncodeToString(adapterProjectionBytes)",
		"AdapterProjectionDigest: adapterProjectionDigest.String()",
	]) requireIncludes(violations, commonIdentity, anchor, "P07B_SOURCE_PROJECTION_ENVELOPE_MISSING", anchor);

  for (const forbidden of ["projection_proof", "allowed_tuple", "disallowed_tuple", "candidate_execution_key", "producer_metadata"]) {
    if (productionSemantic.includes(forbidden)) violations.push(["P07B_SOURCE_FORBIDDEN_AUTHORITY", forbidden]);
  }
	for (const anchor of [
		'SourceProfileV1             = "countershape-node-core-exact/v1"',
		'LaunchProfileV1             = "NODE_REPO_SCRIPT_V1"',
		'CLIStartProfileV1           = "DIRECT_CHILD_V1"',
		'PrivateRootPolicyV1         = "NEW_PRIVATE_ROOT_COPY_VERIFIED_SOURCE_V1"',
		"AmbientExecutableLookup: false", "ShellExecution: false", "SetupCommand: false",
		"ExternalHost: false", "ConfidentialityEstablished: false",
		"if len(identity.PlanSecretSlots) != 0", "AdapterProjectionBase64", "AdapterProjectionDigest",
		"projectionprofile.Parse(profileBytes, binding)", "ResolveCLIProjectionAuthority", "ResolveHTTPProjectionAuthority",
		"source envelope did not regenerate byte-exactly",
	]) requireIncludes(violations, source.lexical.commentless, anchor, "P07B_SOURCE_CLOSED_FACT_MISSING", anchor);

  const startCode = start.lexical.commentless;
  for (const anchor of [
    'HTTPStartAuthorityV1         = "NODE_CORE_INHERITED_LOOPBACK_LISTENER_V1"',
    'HTTPPortableStartAuthorityV1 = "NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1"',
		"return newHTTPStartSpec(entrypoint, HTTPStartAuthorityV1)",
		"return newHTTPStartSpec(entrypoint, HTTPPortableStartAuthorityV1)",
		"if legacy {",
    '"ASCII_COUNTERSHAPE_READY_V1_SPACE_PORT_LF_THEN_EOF_V1"',
	]) requireIncludes(violations, start.lexical.commentless, anchor, "P07B_HTTP_START_HISTORY_OR_PORTABLE_PROFILE", anchor);
	const portFrameParser = functionBody(start.lexical.commentless, "ParseHTTPReadyPortFrame");
	for (const anchor of [
		"len(exact) > PortableReadinessFrameMax", "bytes.HasPrefix(exact, []byte(PortableReadinessFramePrefix))",
		"exact[len(exact)-1] != '\\n'", "len(digits) > 1 && digits[0] == '0'",
		"digit < '0' || digit > '9'", "NewHTTPReadyPortFrame(port)", "bytes.Equal(rebuilt.canonical, exact)",
	]) requireIncludes(violations, portFrameParser, anchor, "P07B_HTTP_PORT_FRAME_OPEN", anchor);
  for (const anchor of ["start.Authority() == HTTPStartAuthorityV1", "start.Authority() == HTTPPortableStartAuthorityV1"]) {
    requireIncludes(violations, binding.source, anchor, "P07B_HTTP_START_READINESS_CROSS_PAIR", anchor);
  }
	for (const anchor of ["NewPortableHTTPStartSpec", "NewPortableHTTPReadinessContract", "HTTPPortableStartAuthorityV1"]) {
		requireIncludes(violations, wrapper.lexical.commentless, anchor, "P07B_HTTP_PORTABLE_WRAPPER_MISSING", anchor);
	}

	for (const [sourceCode, anchors, code] of [
		[functionBody(cliProjectionAuthority.lexical.code, "ResolveCLIProjectionAuthority"), ["NewCLIProjectionAuthority(identity.Fields)", "bytes.Equal(expected.canonicalBytes, canonicalBytes)", "bytes.Equal(expected.binding.CanonicalBytes(), binding.CanonicalBytes())"], "P07B_CLI_PROJECTION_RESOLUTION_OPEN"],
		[functionBody(httpProjectionAuthority.lexical.code, "ResolveHTTPProjectionAuthority"), ["NewHTTPProjectionAuthority()", "bytes.Equal(expected.canonicalBytes, canonicalBytes)", "bytes.Equal(expected.binding.CanonicalBytes(), binding.CanonicalBytes())"], "P07B_HTTP_PROJECTION_RESOLUTION_OPEN"],
		[cliProjectionSchema.lexical.code, ["closedCLIFieldRegistryV1", "closedCLIOperations", "NewCLIProjectionAuthority", "ProjectionComparatorExact"], "P07B_CLI_CLOSED_SCHEMA_MISSING"],
		[httpProjectionSchema.lexical.code, ["closedHTTPFieldsV1", "closedHTTPOperationsV1", "NewHTTPProjectionAuthority", "ProjectionComparatorExact"], "P07B_HTTP_CLOSED_SCHEMA_MISSING"],
		[functionBody(projectionProfile.lexical.code, "Parse"), ["CanonicalChecked()", "bytes.Equal(rebuilt.CanonicalBytes(), exact)"], "P07B_PROFILE_PARSE_NOT_EXACT"],
		[profileRoster.lexical.code, ["cliProfileOrderV1", "rosterIndex <= previousRosterIndex", "equalDescriptor", "validateCLIProjection", "validateHTTPProjection"], "P07B_PROFILE_ROSTER_OPEN"],
		[runnerProfile.lexical.code, ["ValidRepositoryNodeEntrypoint", "HTTPPortableDigest"], "P07B_RUNNER_PROFILE_MISSING"],
	]) for (const anchor of anchors) requireIncludes(violations, sourceCode, anchor, code, anchor);
	requireIncludes(violations, cliProjectionSchema.lexical.commentless, "CLIProjectionOperationRule", "P07B_CLI_CLOSED_SCHEMA_MISSING", "CLI operation-rule digest domain");
	requireIncludes(violations, httpProjectionSchema.lexical.commentless, "HTTPProjectionOperationRule", "P07B_HTTP_CLOSED_SCHEMA_MISSING", "HTTP operation-rule digest domain");
	requireIncludes(
		violations, runnerProfile.lexical.commentless,
		"world.ExecuteHTTP/P07B/opaque-binding/child-bind-pipe-ready/v1",
		"P07B_RUNNER_PROFILE_MISSING", "portable runner lineage literal",
	);

	const serviceCode = worldService.lexical.commentless;
	for (const anchor of [
		"portable := request.binding.StartSpec().Authority() == httpmodel.HTTPPortableStartAuthorityV1",
		"result.readiness.listenerFD = 0", "extraFiles := []*os.File{readinessWriter}",
		"if !portable {", "allocateInheritedLoopbackListener()", "ParseHTTPReadyPortFrame(readiness.bytes)",
		"endpoint = \"127.0.0.1:\" + strconv.Itoa(port)", "else if portable {",
		"requestWire, err = httpmodel.EncodeRequest(request.binding.Stimulus(), port)",
		"valid := observed.err == nil && observed.eof",
		"if portable {\n\t\t\tresult.readiness.frameBytes = append([]byte(nil), readiness.bytes...)",
	]) requireIncludes(violations, serviceCode, anchor, "P07B_HTTP_PHYSICAL_CHILD_BIND_MISSING", anchor);
	const encodeAnchor = "requestWire, err = httpmodel.EncodeRequest(request.binding.Stimulus(), port)";
	const encodePositions = [];
	for (let offset = serviceCode.indexOf(encodeAnchor); offset >= 0; offset = serviceCode.indexOf(encodeAnchor, offset + 1)) encodePositions.push(offset);
	const legacyGuard = serviceCode.indexOf("if !portable {");
	const spawn = serviceCode.indexOf("command.Start()");
	const portableFrame = serviceCode.indexOf("frame, frameErr := httpmodel.ParseHTTPReadyPortFrame(readiness.bytes)");
	if (encodePositions.length !== 2 || !(legacyGuard >= 0 && legacyGuard < encodePositions[0] && encodePositions[0] < spawn && spawn < portableFrame && portableFrame < encodePositions[1])) {
		violations.push(["P07B_HTTP_REQUEST_ORDER_OPEN", `encode positions=${encodePositions.join(",")}; guard=${legacyGuard}; spawn=${spawn}; frame=${portableFrame}`]);
	}
	for (const anchor of [
		"P07B_CHILD_BIND_PIPE_FRAME_READINESS_RECEIPT_V1", "readiness_frame_base64", "readiness_frame_digest",
		"physical.listenerFD != 0", "physical.readinessFD != httpPortableReadinessChildFD",
		"ParseHTTPReadyPortFrame(physical.frameBytes)", "physical.endpoint != \"127.0.0.1:\"+strconv.Itoa(physical.port)",
		"inherited_listener_present", "ListenerFDPresent() bool", "len(physical.frameBytes) != 0",
		"len(physical.frameBytes) == 0 && physical.observedByte != 0", "rebuilt.authority == r.authority",
		"ListenerPresent: false", "listenerPresent: false", "canon.DigestBytes(\"HTTPPortableReadinessFrameBytes\"",
		"FrameDigest: frameDigest.String()",
	]) requireIncludes(violations, worldReceipt.lexical.commentless, anchor, "P07B_HTTP_PORTABLE_RECEIPT_INCOMPLETE", anchor);
	const readinessOnly = functionBody(worldAPI.lexical.commentless, "rejectedPortableReadinessOnly");
	for (const anchor of [
		"executionAuthority == httpmodel.HTTPPortableExecutionAuthorityV1",
		"readiness.protocol == httpmodel.PortableReadinessProtocolV1 && !readiness.accepted",
	]) requireIncludes(violations, readinessOnly, anchor, "P07B_HTTP_REJECTED_READINESS_RECEIPT_DROPPED", anchor);
	for (const anchor of [
		"rejectedPortableReadinessOnly(binding.Authority(), service.readiness)",
		"if len(service.exchange.requestWire) > 0",
	]) requireIncludes(violations, worldAPI.lexical.commentless, anchor, "P07B_HTTP_REJECTED_READINESS_RECEIPT_DROPPED", anchor);
	requireExcludes(violations, worldAPI.lexical.commentless, "owned endpoint receipt", "P07B_HTTP_PORT_OWNERSHIP_OVERCLAIM", "child-reported endpoint is not listener ownership evidence");
	if (legacyFixture.bytes.length !== 18343 || createHash("sha256").update(legacyFixture.bytes).digest("hex") !== "d476c70a9c8e1605fc65335044afddaa9f56f007d1fcbc70ea87c0b747c72caa") {
		violations.push(["P07B_HTTP_LEGACY_FIXTURE_DRIFT", "server.mjs must retain the sealed U4 18,343-byte d476c70a... identity"]);
	}
	if (portableFixture.bytes.length !== 19264 || createHash("sha256").update(portableFixture.bytes).digest("hex") !== "8cb5dfc1d5276e9f6157c127339e6e6c6404f525a3e8f9bec4f3eb1ad84a9932") {
		violations.push(["P07B_HTTP_PORTABLE_FIXTURE_DRIFT", "server_child_bind.mjs must retain the reviewed P07B 19,264-byte 8cb5dfc1... identity"]);
	}
	for (const anchor of [
		"const listenerFD = 3;", "const readinessFD = 4;", "server.listen({ fd: listenerFD, exclusive: true }",
		"Buffer.from([0x01])", "closeSync(readinessFD)",
	]) requireIncludes(violations, legacyFixture.lexical.commentless, anchor, "P07B_HTTP_LEGACY_FIXTURE_DRIFT", anchor);
	for (const anchor of [
		'import { createServer } from "node:net";', 'server.listen({ host: "127.0.0.1", port: 0, exclusive: true }',
		'Object.prototype.hasOwnProperty.call(process.env, "COUNTERSHAPE_HTTP_LISTEN_FD")',
		'Object.prototype.hasOwnProperty.call(process.env, "COUNTERSHAPE_HTTP_PORT")',
		'Buffer.from(`COUNTERSHAPE_READY_V1 ${address.port}\\n`, "ascii")', "closeSync(readinessFD)",
	]) requireIncludes(violations, portableFixture.lexical.commentless, anchor, "P07B_HTTP_FIXTURE_CHILD_BIND_MISSING", anchor);
	if (createHash("sha256").update(portableFixture.bytes).digest("hex") === createHash("sha256").update(legacyFixture.bytes).digest("hex")) {
		violations.push(["P07B_HTTP_FIXTURE_CHILD_BIND_MISSING", "portable fixture collapsed onto sealed legacy bytes"]);
	}
	for (const [body, anchors, label] of [
		[functionBody(fixtureGo.lexical.commentless, "CandidateFiles"), ["candidateFiles(role, Entrypoint, fixtureProgram)"], "legacy candidate roster"],
		[functionBody(fixtureGo.lexical.commentless, "PortableCandidateFiles"), ["candidateFiles(role, PortableEntrypoint, portableFixtureProgram)"], "portable candidate roster"],
	]) for (const anchor of anchors) requireIncludes(violations, body, anchor, "P07B_HTTP_FIXTURE_SELECTION_OPEN", `${label}: ${anchor}`);
	for (const anchor of [
		"fixtureFiles := httpfixture.CandidateFiles", "fixtureFiles = httpfixture.PortableCandidateFiles",
		"entrypoint := httpfixture.Entrypoint", "entrypoint = httpfixture.PortableEntrypoint",
		"NewPortableHTTPStartSpec(entrypoint)",
		"{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, binding.Authority()}",
	]) requireIncludes(violations, study.lexical.commentless, anchor, "P07B_HTTP_STUDY_PORTABLE_SELECTION_OPEN", anchor);
	for (const anchor of ["legacyProgramBytes  = 18343", "legacyProgramSHA256", "TestPortableCandidateFilesAreDistinctWithoutRewritingLegacyFixture"]) {
		requireIncludes(violations, fixtureTests.lexical.code, anchor, "P07B_HTTP_FIXTURE_HISTORY_TEST_MISSING", anchor);
	}
	for (const anchor of [
		"REQUIRED_P07B_MUTANT_ID_DIGEST", "P07B_MUTANT_SET_MISMATCH", "runRequiredArchitectureCheck",
		"exactP07BABADigest", "source_manifest", "tool_fingerprints", "P07B mutation gate:",
	]) requireIncludes(violations, mutationGate.source, anchor, "P07B_MUTATION_GATE_INCOMPLETE", anchor);

	const sourceTests = at("internal/contractsource/source_test.go").lexical.code;
  const startTests = at("internal/adapters/http/model/portable_start_test.go").lexical.code;
	for (const test of [
		"TestClosedProjectionModelsMatchAdaptersAndPortableProfilesExhaustively",
		"TestPortableSourceRejectsProjectionProfileCrossPairsAndDrift",
		"TestPortableSourceRejectsCLIProfileOrderAlias",
		"TestPortableSourceReconstructsExactCLIAndHTTPAuthorities",
    "TestPortableSourceRequiresEveryExactPayloadByte",
    "TestPortableSourceRequiresExactAuthorityJoin",
    "TestPortableSourceLaunchRosterIsExact",
		"TestPortableSourcePreservesAbsentVersusPresentEmpty", "TestPortableSourceViewsRecoverExactRawPayloadsWithoutPrivateJSONParsing",
		"FuzzPortableSourceParseNeverMintsUnreconstructedAuthority",
	]) requireIncludes(violations, sourceTests, `func ${test}(`, "P07B_SOURCE_TEST_MISSING", test);
	requireIncludes(violations, startTests, "TestPortableHTTPStartIsDistinctWithoutRewritingHistoricalBytes", "P07B_HTTP_HISTORY_TEST_MISSING", "portable start history");
	for (const [path, test] of [
		["internal/adapters/cli/model/projection_schema_test.go", "TestCLIProjectionAuthorityRejectsSelfConsistentInventedSchemas"],
		["internal/adapters/http/model/projection_schema_test.go", "TestHTTPProjectionAuthorityRejectsSelfConsistentInventedSchemas"],
		["internal/world/http_service_darwin_test.go", "TestHTTPPortableReadinessBindsExactChildReportedPortFrame"],
		["internal/world/http_service_darwin_test.go", "TestHTTPReadinessUsesInheritedPipeWithoutHTTPWarmup"],
		["internal/world/http_portable_negative_darwin_test.go", "TestHTTPPortableReadinessFailuresRemainFinalizedReceipts"],
		["internal/world/http_portable_negative_darwin_test.go", "TestAcceptedPortableReadinessWithoutRequestRemainsInvariantFailure"],
		["internal/world/http_portable_negative_darwin_test.go", "TestPortableReadinessPortIsAReportNotListenerOwnership"],
		["internal/projectionprofile/profile_test.go", "TestParseRequiresExactTypedRegeneration"],
		["testkit/studies/http_invoices/study_darwin_test.go", "TestHTTPInvoicePortableChildBindPhysicalLineage"],
	]) requireIncludes(violations, at(path).lexical.code, `func ${test}(`, "P07B_REQUIRED_TEST_MISSING", `${path}:${test}`);

  return violations;
}

async function main() {
  if (process.argv.length !== 2) throw new ArchitectureError("P07B_ARCHITECTURE_ARGUMENTS", "no arguments accepted");
	runInheritedU6();
	runSourceDependencyClosure();
  const initial = await manifest();
  const violations = inspect(initial);
  const final = await manifest();
  if (initial.digest !== final.digest) throw new ArchitectureError("P07B_MANIFEST_CHANGED", `${initial.digest}->${final.digest}`);
  if (violations.length > 0) {
    const detail = violations.map(([code, message]) => `${code}: ${message}`).sort().join("\n");
    throw new ArchitectureError("P07B_ARCHITECTURE_VIOLATION", detail);
  }
  process.stdout.write(`P07B source architecture boundary OK (${exactFiles.length} exact files; manifest ${initial.digest})\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
