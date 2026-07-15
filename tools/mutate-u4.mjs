#!/usr/bin/env node

import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import {
  chmod,
  lstat,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  realpath,
  rm,
} from "node:fs/promises";
import { tmpdir } from "node:os";
import {
  dirname,
  join,
  relative as relativePath,
  resolve,
  sep,
} from "node:path";
import { fileURLToPath } from "node:url";

import {
  MutationGateError,
  NamedTestOutcome,
  applyExactMutation,
  copyRegularAllowlist,
  mutateRegularFileNoFollow,
  readAllowlistManifest,
  revalidateAdmittedGoExecutable,
  resolveAdmittedGoExecutable,
} from "./mutate-u1.mjs";
import {
  admitU2DarwinToolchain,
  minimalU2GoEnvironment,
  revalidateAdmittedDarwinCCompiler,
  runNamedU2GoTest,
  runU2Mutant,
} from "./mutate-u2.mjs";
import { U3_SANDBOX_FILE_ALLOWLIST, U3_SOURCE_SCOPE_ROOTS } from "./mutate-u3.mjs";

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = resolve(dirname(modulePath), "..");

export const U4_REVIEWED_NON_GO_FILES = Object.freeze([
  "testkit/httpfixture/server.mjs",
]);

const U4_REVIEWED_EMBED_BINDINGS = Object.freeze([
  Object.freeze({
    owner: "testkit/httpfixture/fixture.go",
    asset: "testkit/httpfixture/server.mjs",
    directive: "//go:embed server.mjs",
  }),
]);

const U4_ADDITIONAL_FILES = Object.freeze([
  "internal/adapters/http/capture.go",
  "internal/adapters/http/capture_test.go",
  "internal/adapters/http/digest.go",
  "internal/adapters/http/errors.go",
  "internal/adapters/http/observe_bridge.go",
  "internal/adapters/http/projection.go",
  "internal/adapters/http/projection_input.go",
  "internal/adapters/http/projection_test.go",
  "internal/adapters/http/projection_valid.go",
  "internal/adapters/http/stimulus.go",
  "internal/adapters/http/wire.go",
  "internal/adapters/http/world_bridge.go",
  "internal/spec/compiler.go",
  "internal/spec/compiler_test.go",
  "internal/spec/errors.go",
  "internal/spec/parse.go",
  "testkit/httpfixture/fixture.go",
  "testkit/httpfixture/fixture_test.go",
  "testkit/httpfixture/server.mjs",
  "testkit/studies/http_invoices/negative_shared_root_darwin_test.go",
  "testkit/studies/http_invoices/study.go",
  "testkit/studies/http_invoices/study_darwin_test.go",
]);

// This positive manifest is the complete transitive source closure for every
// U4 named mutant test. The sole reviewed non-Go source is server.mjs, required
// by fixture.go's go:embed declaration and by the physical invoice-study
// killers. It deliberately excludes docs, VCS state, didrun data, credentials,
// study output, and every unreviewed artifact.
export const U4_SANDBOX_FILE_ALLOWLIST = Object.freeze([
  ...U3_SANDBOX_FILE_ALLOWLIST,
  ...U4_ADDITIONAL_FILES,
]);

export const U4_SOURCE_SCOPE_ROOTS = Object.freeze([
  ...U3_SOURCE_SCOPE_ROOTS.filter((root) => root !== "internal/adapters/http/model"),
  "internal/adapters/http",
  "internal/spec",
  "testkit/httpfixture",
  "testkit/studies/http_invoices",
]);

export const REQUIRED_U4_MUTANT_IDS = Object.freeze([
  "readiness-by-http-request",
  "shared-root-product-option",
  "proxy-environment-use",
  "redirect-following-second-connection",
  "status-500-as-infrastructure-failure",
  "transport-failure-as-status",
  "teardown-failure-admitted",
  "volatile-removal-hidden-from-transcript",
  "status-removal-accepted",
  "disclosure-metadata-removal-accepted",
  "alternating-d-majority-classified",
  "outcome-map-reduced-to-group-shape",
  "collapse-absent-body-into-present-empty",
  "accept-content-length-mismatch",
  "accept-wrong-readiness-byte",
  "ignore-invocation-request-digest",
]);

export const REVIEWED_U4_MUTANT_CONTRACT = Object.freeze([
  Object.freeze({
    id: "readiness-by-http-request",
    file: "internal/world/http_service_darwin.go",
    find: [
      "\tif result.process.primary == \"\" {",
      "\t\treadiness, earlyWait, control, diagnostic := awaitExactHTTPReadiness(",
      "\t\t\tctx, readinessReader, time.Duration(request.readinessBudgetMS)*time.Millisecond,",
    ].join("\n"),
    replace: [
      "\tif result.process.primary == \"\" {",
      "\t\t// Mutant: consume one real HTTP exchange as a warm-up before the",
      "\t\t// inherited readiness owner is allowed to classify the pipe signal.",
      "\t\t_ = performOneHTTPExchange(ctx, endpoint, requestWire.Bytes(), request.responseLimit, time.Duration(request.probeBudgetMS)*time.Millisecond)",
      "\t\treadiness, earlyWait, control, diagnostic := awaitExactHTTPReadiness(",
      "\t\t\tctx, readinessReader, time.Duration(request.readinessBudgetMS)*time.Millisecond,",
    ].join("\n"),
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoiceReferenceStudy",
  }),
  Object.freeze({
    id: "shared-root-product-option",
    file: "internal/world/allocate.go",
    find: "\tmarker.AllocationNonce = filepath.Base(attemptRoot)",
    replace: [
      "\t// Mutant: every product attempt aliases the same mutable state root,",
      "\t// while retaining otherwise fresh attempt/candidate/evidence roots.",
      "\tsharedState := filepath.Join(base, \"countershape-shared-http-state\")",
      "\tif err := os.MkdirAll(sharedState, 0o700); err != nil {",
      "\t\treturn Roots{}, \"\", nil, \"\", refuse(CodeInvalidAllocation, \"shared state root allocation failed\", err)",
      "\t}",
      "\troots.state = sharedState",
      "\tmarker.AllocationNonce = filepath.Base(roots.attempt)",
    ].join("\n"),
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoiceReferenceStudy",
  }),
  Object.freeze({
    id: "proxy-environment-use",
    file: "internal/world/http_service_darwin.go",
    find: "\tconnection, err := (&net.Dialer{}).DialContext(probeContext, \"tcp4\", endpoint)",
    replace: [
      "\tif proxy := os.Getenv(\"HTTP_PROXY\"); len(proxy) > len(\"http://\") && proxy[:len(\"http://\")] == \"http://\" {",
      "\t\tproxyConnection, proxyErr := (&net.Dialer{}).DialContext(probeContext, \"tcp4\", proxy[len(\"http://\"):])",
      "\t\tif proxyErr == nil {",
      "\t\t\t_, _ = proxyConnection.Write(requestWire)",
      "\t\t\t_ = proxyConnection.Close()",
      "\t\t}",
      "\t}",
      "\tdialer := &net.Dialer{}",
      "\tconnection, err := dialer.DialContext(probeContext, \"tcp4\", endpoint)",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestHTTPExchangeIgnoresAmbientProxyRoutingMutationGuard",
  }),
  Object.freeze({
    id: "redirect-following-second-connection",
    file: "internal/world/http_service_darwin.go",
    find: [
      "\tif readErr != nil {",
      "\t\treturn classifyHTTPExchangeError(probeContext, result, readErr, \"HTTP_RESPONSE_READ_FAILED\")",
      "\t}",
      "\treturn result",
      "}",
    ].join("\n"),
    replace: [
      "\tif readErr != nil {",
      "\t\treturn classifyHTTPExchangeError(probeContext, result, readErr, \"HTTP_RESPONSE_READ_FAILED\")",
      "\t}",
      "\tconst redirectPrefix = \"\\r\\nlocation: http://\"",
      "\tfor offset := 0; offset+len(redirectPrefix) <= len(result.wire); offset++ {",
      "\t\tif string(result.wire[offset:offset+len(redirectPrefix)]) != redirectPrefix {",
      "\t\t\tcontinue",
      "\t\t}",
      "\t\tstart := offset + len(redirectPrefix)",
      "\t\tend := start",
      "\t\tfor end < len(result.wire) && result.wire[end] != '/' && result.wire[end] != '\\r' {",
      "\t\t\tend++",
      "\t\t}",
      "\t\tif end > start {",
      "\t\t\treturn performOneHTTPExchange(ctx, string(result.wire[start:end]), requestWire, responseLimit, budget)",
      "\t\t}",
      "\t}",
      "\treturn result",
      "}",
    ].join("\n"),
    package: "./internal/world",
    testName: "TestHTTPExchangeNeverFollowsRedirectLocationMutationGuard",
  }),
  Object.freeze({
    id: "status-500-as-infrastructure-failure",
    file: "internal/adapters/http/model/wire.go",
    find: [
      "\t// MUTATION_ANCHOR: status-500-remains-complete-application-response",
      "\treason := string(line[13:])",
    ].join("\n"),
    replace: [
      "\t// MUTATION_ANCHOR: status-500-remains-complete-application-response",
      "\tif status == 500 {",
      "\t\treturn 0, \"\", refuse(CodeResponseStatus, \"500 treated as infrastructure failure\")",
      "\t}",
      "\treason := string(line[13:])",
    ].join("\n"),
    package: "./internal/adapters/http/model",
    testName: "TestHTTPStatus500RemainsCompleteApplicationResponseMutationGuard",
  }),
  Object.freeze({
    id: "transport-failure-as-status",
    file: "internal/adapters/http/capture.go",
    find: "\t\tif input.Response.Valid() || input.ResponseParsed || len(controls) == 0 {",
    replace: "\t\tif false && (input.Response.Valid() || input.ResponseParsed || len(controls) == 0) {",
    package: "./internal/adapters/http",
    testName: "TestHTTPTransportFailureCannotSynthesizeStatusMutationGuard",
  }),
  Object.freeze({
    id: "teardown-failure-admitted",
    file: "internal/adapters/http/projection.go",
    find: "\tif !present || len(observation.controls) != 0 {",
    replace: "\tif !present || false && len(observation.controls) != 0 {",
    package: "./internal/adapters/http",
    testName: "TestHTTPProjectionRefusesTeardownControlledResponseMutationGuard",
  }),
  Object.freeze({
    id: "volatile-removal-hidden-from-transcript",
    file: "internal/adapters/http/projection.go",
    find: "\ttranscript := d.trace()",
    replace: "\ttranscript := append(d.trace()[:4], d.trace()[6:]...)",
    package: "./internal/adapters/http",
    testName: "TestHTTPProjectionRuntimeTranscriptUsesExactSourcesForEveryOperation",
  }),
  Object.freeze({
    id: "status-removal-accepted",
    file: "internal/adapters/http/projection.go",
    find: "\t\t{id: HTTPFieldStatus, tag: \"INTEGER\", integer: int64(response.Status())},",
    replace: "\t\t// status field incorrectly removed",
    package: "./internal/adapters/http",
    testName: "TestHTTPProjectionMandatesStatusAndDisclosureMetadataMutationGuard",
  }),
  Object.freeze({
    id: "disclosure-metadata-removal-accepted",
    file: "internal/adapters/http/projection.go",
    find: "\t\t{id: HTTPFieldBodyMetadata, tag: \"CANONICAL_JSON\", canonicalJSON: string(metadataBytes)},",
    replace: "\t\t{id: HTTPFieldBodyMetadata, tag: \"CANONICAL_JSON\", canonicalJSON: string(metadataBytes[:0])}, // disclosure metadata incorrectly erased",
    package: "./internal/adapters/http",
    testName: "TestHTTPProjectionMandatesStatusAndDisclosureMetadataMutationGuard",
  }),
  Object.freeze({
    id: "alternating-d-majority-classified",
    file: "internal/observe/eligibility.go",
    find: [
      "\t} else if len(counts) >= 2 {",
      "\t\t// Once two eligible fingerprints exist, instability is established",
      "\t\t// evidence. A later orchestration stop can prevent stability from being",
      "\t\t// established, but cannot erase an observed disagreement.",
      "\t\tclassification.status = Unstable",
      "\t\tclassification.histogram = sortedHistogram(counts)",
    ].join("\n"),
    replace: [
      "\t} else if len(counts) >= 2 {",
      "\t\tclassification.status = ObservedStable",
      "\t\tfor fingerprint := range counts {",
      "\t\t\tclassification.fingerprint = fingerprint",
      "\t\t\tbreak",
      "\t\t}",
    ].join("\n"),
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoiceAlternatingCandidateUsesAllTrialsNeverMajority",
  }),
  Object.freeze({
    id: "outcome-map-reduced-to-group-shape",
    file: "internal/compare/outcome_map.go",
    find: [
      "\t\t\tentries = append(entries, Entry{",
      "\t\t\t\tCandidateKey:          key,",
      "\t\t\t\tProjectionFingerprint: fingerprint,",
    ].join("\n"),
    replace: [
      "\t\t\tentries = append(entries, Entry{",
      "\t\t\t\tCandidateKey:          roster[0], // mutant collapses every eligible label into one group representative",
      "\t\t\t\tProjectionFingerprint: fingerprint,",
    ].join("\n"),
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoiceOutcomeMapUsesExactCandidateLabelsNotGroupShape",
  }),
  Object.freeze({
    id: "collapse-absent-body-into-present-empty",
    file: "internal/adapters/http/model/stimulus.go",
    find: "func AbsentBody() HTTPBody { return HTTPBody{presence: PresenceAbsent, bytes: []byte{}} }",
    replace: "func AbsentBody() HTTPBody { return HTTPBody{presence: PresencePresent, bytes: []byte{}} }",
    package: "./internal/adapters/http/model",
    testName: "TestHTTPWireDistinguishesAbsentAndPresentEmptyBodyMutationGuard",
  }),
  Object.freeze({
    id: "accept-content-length-mismatch",
    file: "internal/adapters/http/model/wire.go",
    find: "\tif int64(len(body)) != contentLength {",
    replace: "\tif false && int64(len(body)) != contentLength {",
    package: "./internal/adapters/http/model",
    testName: "TestHTTPResponseParserRejectsContentLengthMismatchMutationGuard",
  }),
  Object.freeze({
    id: "accept-wrong-readiness-byte",
    file: "internal/world/http_service_darwin.go",
    find: "\t\t\tif observed.err != nil || !observed.eof || len(observed.bytes) != 1 || observed.bytes[0] != httpmodel.ReadinessSuccessByte {",
    replace: "\t\t\tif observed.err != nil || !observed.eof || len(observed.bytes) != 1 || false && observed.bytes[0] != httpmodel.ReadinessSuccessByte {",
    package: "./internal/world",
    testName: "TestHTTPReadinessRejectsWrongByteMutationGuard",
  }),
  Object.freeze({
    id: "ignore-invocation-request-digest",
    file: "internal/world/http_receipt.go",
    find: "\tif !requestByteCountMatches || !requestSHA256Matches {",
    replace: "\tif !requestByteCountMatches {",
    package: "./internal/world",
    testName: "TestHTTPInvocationEvidenceBindsAttemptStimulusAndExactRequest",
  }),
]);

export const U4_MUTANTS = Object.freeze(
  REVIEWED_U4_MUTANT_CONTRACT.map((mutant) => Object.freeze({ ...mutant })),
);

const tupleFields = Object.freeze(["file", "find", "id", "package", "replace", "testName"]);

function sameArray(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function exactOccurrenceCount(source, anchor) {
  if (typeof anchor !== "string" || anchor.length === 0) return 0;
  return source.split(anchor).length - 1;
}

export function assertU4MutantDefinitionSet(
  mutants = U4_MUTANTS,
  requiredIDs = REQUIRED_U4_MUTANT_IDS,
  reviewedContract = REVIEWED_U4_MUTANT_CONTRACT,
) {
  const requiredCount = 16;
  if (requiredIDs.length !== requiredCount || new Set(requiredIDs).size !== requiredCount) {
    throw new MutationGateError("INVALID_REQUIRED_ID_SET", `U4 requires exactly ${requiredCount} unique frozen IDs`);
  }
  if (mutants.length !== requiredCount) {
    throw new MutationGateError("MUTANT_SET_MISMATCH", `defined ${mutants.length} U4 mutants, want ${requiredCount}`);
  }
  const ids = mutants.map((mutant) => mutant.id);
  if (new Set(ids).size !== ids.length || !sameArray(ids, requiredIDs)) {
    throw new MutationGateError("MUTANT_SET_MISMATCH", "U4 mutant IDs or reviewed order changed");
  }
  if (reviewedContract.length !== requiredCount || !sameArray(reviewedContract.map((tuple) => tuple.id), requiredIDs)) {
    throw new MutationGateError("INVALID_REVIEWED_MUTANT_CONTRACT", "the independent U4 tuple contract changed shape or order");
  }
  const allowedFiles = new Set(U4_SANDBOX_FILE_ALLOWLIST);
  const reviewedByID = new Map(reviewedContract.map((tuple) => [tuple.id, tuple]));
  for (const mutant of mutants) {
    const reviewed = reviewedByID.get(mutant.id);
    if (!reviewed || !sameArray(Object.keys(mutant).sort(), tupleFields)) {
      throw new MutationGateError("MUTANT_CONTRACT_MISMATCH", `${mutant.id} differs from the reviewed tuple shape`);
    }
    for (const field of tupleFields) {
      if (mutant[field] !== reviewed[field]) {
        throw new MutationGateError("MUTANT_CONTRACT_MISMATCH", `${mutant.id}.${field} differs from the reviewed tuple`);
      }
    }
    if (!allowedFiles.has(mutant.file)) {
      throw new MutationGateError("MUTANT_FILE_NOT_ALLOWLISTED", `${mutant.id} targets ${mutant.file}`);
    }
    if (mutant.find.length === 0 || mutant.find === mutant.replace) {
      throw new MutationGateError("INVALID_MUTATION", `${mutant.id} is not one exact effective replacement`);
    }
    if (!/^(?:\.\/internal\/(?:adapters\/http(?:\/model)?|compare|observe|world)|\.\/testkit\/studies\/http_invoices)$/u.test(mutant.package)) {
      throw new MutationGateError("INVALID_MUTATION_PACKAGE", `${mutant.id} has invalid package ${mutant.package}`);
    }
    if (!/^Test[A-Za-z0-9_]+$/u.test(mutant.testName)) {
      throw new MutationGateError("INVALID_MUTATION_TEST", `${mutant.id} has invalid named test ${mutant.testName}`);
    }
  }
}

async function enumerateU4SourceScope(sourceRoot, scope, reviewedNonGo) {
  const absoluteRoot = resolve(sourceRoot, scope);
  let rootMetadata;
  try {
    rootMetadata = await lstat(absoluteRoot);
  } catch (error) {
    throw new MutationGateError("U4_MANIFEST_SCOPE_MISSING", `${scope}: ${error.code ?? error.message}`);
  }
  if (rootMetadata.isSymbolicLink() || !rootMetadata.isDirectory()) {
    throw new MutationGateError("U4_MANIFEST_SCOPE_NONREGULAR", `${scope} is not one real directory`);
  }
  const files = [];
  async function visit(directory) {
    const entries = await readdir(directory, { withFileTypes: true });
    entries.sort((left, right) => left.name.localeCompare(right.name, "en"));
    for (const entry of entries) {
      const absolute = join(directory, entry.name);
      const relative = relativePath(sourceRoot, absolute).split(sep).join("/");
      if (entry.isSymbolicLink()) {
        throw new MutationGateError("U4_MANIFEST_TREE_SYMLINK", `${relative} is a symbolic link`);
      }
      if (entry.isDirectory()) {
        await visit(absolute);
      } else if (!entry.isFile()) {
        throw new MutationGateError("U4_MANIFEST_TREE_NONREGULAR", `${relative} is not a regular file`);
      } else if (!relative.endsWith(".go") && !reviewedNonGo.has(relative)) {
        throw new MutationGateError("U4_MANIFEST_TREE_UNREVIEWED_NON_GO", `${relative} is not an admitted U4 source asset`);
      } else {
        files.push(relative);
      }
    }
  }
  await visit(absoluteRoot);
  return files;
}

export async function assertU4ManifestMatchesTree(
  sourceRoot = repoRoot,
  allowlist = U4_SANDBOX_FILE_ALLOWLIST,
  scopeRoots = U4_SOURCE_SCOPE_ROOTS,
  reviewedNonGoFiles = U4_REVIEWED_NON_GO_FILES,
) {
  if (allowlist.length === 0 || new Set(allowlist).size !== allowlist.length) {
    throw new MutationGateError("U4_INVALID_FILE_ALLOWLIST", "U4 file allowlist must be nonempty and unique");
  }
  if (scopeRoots.length === 0 || new Set(scopeRoots).size !== scopeRoots.length) {
    throw new MutationGateError("U4_INVALID_SOURCE_SCOPES", "U4 source scopes must be nonempty and unique");
  }
  if (reviewedNonGoFiles.length === 0 || new Set(reviewedNonGoFiles).size !== reviewedNonGoFiles.length) {
    throw new MutationGateError("U4_INVALID_NON_GO_ALLOWLIST", "U4 reviewed non-Go allowlist must be nonempty and unique");
  }
  const reviewedNonGo = new Set(reviewedNonGoFiles);
  const declaredNonGo = allowlist.filter((file) => file !== "go.mod" && !file.endsWith(".go")).sort();
  if (!sameArray(declaredNonGo, [...reviewedNonGoFiles].sort())) {
    throw new MutationGateError(
      "U4_NON_GO_ALLOWLIST_MISMATCH",
      `declared non-Go files [${declaredNonGo.join(",")}] differ from the reviewed closed set`,
    );
  }
  const isScoped = (file) => scopeRoots.some((root) => file.startsWith(`${root}/`));
  const support = allowlist.filter((file) => !isScoped(file)).sort();
  if (!sameArray(support, ["go.mod"])) {
    throw new MutationGateError("U4_MANIFEST_SUPPORT_SET_MISMATCH", `support files [${support.join(",")}] differ from go.mod`);
  }
  for (const file of reviewedNonGo) {
    if (!isScoped(file)) {
      throw new MutationGateError("U4_NON_GO_OUTSIDE_SCOPE", `${file} is outside every reviewed U4 source scope`);
    }
  }

  const actual = [];
  for (const scope of scopeRoots) {
    actual.push(...await enumerateU4SourceScope(sourceRoot, scope, reviewedNonGo));
  }
  actual.sort();
  if (new Set(actual).size !== actual.length) {
    throw new MutationGateError("U4_OVERLAPPING_SOURCE_SCOPES", "U4 source scopes enumerate at least one file twice");
  }
  const actualSet = new Set(actual);
  const expected = allowlist.filter((file) => file !== "go.mod").sort();
  const expectedSet = new Set(expected);
  const unlisted = actual.filter((file) => !expectedSet.has(file));
  const missing = expected.filter((file) => !actualSet.has(file));
  if (unlisted.length > 0 || missing.length > 0) {
    throw new MutationGateError("U4_MANIFEST_TREE_MISMATCH", `unlisted=[${unlisted.join(",")}] missing=[${missing.join(",")}]`);
  }
  if (!sameArray(U4_REVIEWED_EMBED_BINDINGS.map((binding) => binding.asset).sort(), [...reviewedNonGoFiles].sort())) {
    throw new MutationGateError("U4_EMBED_BINDING_SET_MISMATCH", "reviewed non-Go files differ from exact go:embed bindings");
  }
  for (const binding of U4_REVIEWED_EMBED_BINDINGS) {
    const owner = await readFile(resolve(sourceRoot, binding.owner), "utf8");
    if (exactOccurrenceCount(owner, binding.directive) !== 1) {
      throw new MutationGateError("U4_EMBED_DIRECTIVE_COUNT", `${binding.owner} does not bind ${binding.asset} exactly once`);
    }
  }
  // This second read admits every exact listed byte with no-follow checks and
  // thereby hashes the reviewed server.mjs alongside all Go source.
  return readAllowlistManifest(sourceRoot, allowlist);
}

function requireU4ManifestEqual(actual, expected, code, detail) {
  if (actual.digest !== expected.digest) {
    throw new MutationGateError(code, `${detail}: ${actual.digest} != ${expected.digest}`);
  }
}

function u4ManifestEntryMap(manifest) {
  return new Map(manifest.entries.map((entry) => [entry.file, entry]));
}

async function validateU4TreeAndManifest(root, allowlist, scopeRoots, reviewedNonGoFiles) {
  await assertU4ManifestMatchesTree(root, allowlist, scopeRoots, reviewedNonGoFiles);
  return readAllowlistManifest(root, allowlist);
}

async function createU4SeedSnapshot({ sourceRoot = repoRoot } = {}) {
  const resolvedSource = resolve(sourceRoot);
  const allowlist = Object.freeze([...U4_SANDBOX_FILE_ALLOWLIST]);
  const scopeRoots = Object.freeze([...U4_SOURCE_SCOPE_ROOTS]);
  const reviewedNonGoFiles = Object.freeze([...U4_REVIEWED_NON_GO_FILES]);
  const sourceManifest = await validateU4TreeAndManifest(resolvedSource, allowlist, scopeRoots, reviewedNonGoFiles);
  const parent = await mkdtemp(join(tmpdir(), "countershape-u4-seed-"));
  await chmod(parent, 0o700);
  const root = join(parent, "repo");
  try {
    await copyRegularAllowlist(resolvedSource, root, allowlist);
    const manifest = await validateU4TreeAndManifest(root, allowlist, scopeRoots, reviewedNonGoFiles);
    requireU4ManifestEqual(manifest, sourceManifest, "U4_SEED_COPY_MISMATCH", "immutable U4 seed differs from admitted source");
    return Object.freeze({
      sourceRoot: resolvedSource,
      sourceManifest,
      parent,
      root,
      manifest,
      allowlist,
      scopeRoots,
      reviewedNonGoFiles,
    });
  } catch (error) {
    await rm(parent, { recursive: true, force: true });
    throw error;
  }
}

async function cleanupU4SeedSnapshot(seed) {
  await rm(seed.parent, { recursive: true, force: true });
}

async function assertU4SourceAndSeedUnchanged(seed) {
  const seedManifest = await validateU4TreeAndManifest(seed.root, seed.allowlist, seed.scopeRoots, seed.reviewedNonGoFiles);
  requireU4ManifestEqual(seedManifest, seed.manifest, "U4_SEED_SNAPSHOT_CHANGED", "immutable U4 seed changed");
  const sourceManifest = await validateU4TreeAndManifest(seed.sourceRoot, seed.allowlist, seed.scopeRoots, seed.reviewedNonGoFiles);
  requireU4ManifestEqual(sourceManifest, seed.sourceManifest, "U4_ADMITTED_SOURCE_CHANGED", "admitted U4 source changed");
}

async function assertExactU4MutantDelta(sandboxRoot, seed, mutant) {
  const actual = await validateU4TreeAndManifest(sandboxRoot, seed.allowlist, seed.scopeRoots, seed.reviewedNonGoFiles);
  const expectedEntries = u4ManifestEntryMap(seed.manifest);
  const actualEntries = u4ManifestEntryMap(actual);
  const changed = [];
  for (const [file, expected] of expectedEntries) {
    const candidate = actualEntries.get(file);
    if (!candidate || candidate.bytes !== expected.bytes || candidate.sha256 !== expected.sha256) changed.push(file);
  }
  for (const file of actualEntries.keys()) if (!expectedEntries.has(file)) changed.push(file);
  changed.sort();
  if (changed.length !== 1 || changed[0] !== mutant.file) {
    throw new MutationGateError("U4_MUTANT_DELTA_SCOPE", `${mutant.id}: changed [${changed.join(",")}], want exactly ${mutant.file}`);
  }
  const seedBytes = await readFile(resolve(seed.root, mutant.file));
  const expectedBytes = Buffer.from(applyExactMutation(seedBytes.toString("utf8"), mutant), "utf8");
  const actualBytes = await readFile(resolve(sandboxRoot, mutant.file));
  if (!actualBytes.equals(expectedBytes)) {
    throw new MutationGateError("U4_MUTANT_DELTA_BYTES", `${mutant.id}: target bytes differ from the reviewed exact replacement`);
  }
  return actual;
}

export async function assertReviewedU4Anchors(sourceRoot = repoRoot, mutants = U4_MUTANTS) {
  assertU4MutantDefinitionSet(mutants);
  for (const mutant of mutants) {
    const source = await readFile(resolve(sourceRoot, mutant.file), "utf8");
    const count = exactOccurrenceCount(source, mutant.find);
    if (count !== 1) {
      throw new MutationGateError("MUTATION_ANCHOR_COUNT", `${mutant.id}: exact anchor count was ${count}, want 1`);
    }
    const mutated = applyExactMutation(source, mutant);
    if (mutated === source || exactOccurrenceCount(mutated, mutant.find) !== 0) {
      throw new MutationGateError("MUTATION_REPLACEMENT_INVALID", `${mutant.id}: replacement is not exact and singular`);
    }
  }
}

export async function assertReviewedU4NamedTests(sourceRoot = repoRoot, mutants = U4_MUTANTS) {
  assertU4MutantDefinitionSet(mutants);
  for (const mutant of mutants) {
    const packageRoot = mutant.package.slice(2);
    const testFiles = U4_SANDBOX_FILE_ALLOWLIST.filter(
      (file) => file.startsWith(`${packageRoot}/`) && file.endsWith("_test.go"),
    );
    let definitions = 0;
    const patternSource = `\\bfunc\\s+${mutant.testName}\\s*\\(`;
    for (const file of testFiles) {
      const source = await readFile(resolve(sourceRoot, file), "utf8");
      definitions += source.match(new RegExp(patternSource, "gu"))?.length ?? 0;
    }
    if (definitions !== 1) {
      throw new MutationGateError(
        "U4_NAMED_TEST_DEFINITION_COUNT",
        `${mutant.id}: ${mutant.package} ${mutant.testName} has ${definitions} admitted definitions, want 1`,
      );
    }
  }
}

function fakeClassification(outcome) {
  return { classification: { outcome, detail: outcome }, output: "" };
}

function inspectU4StudyExecutable(label, admission, versionArgs, versionPattern) {
  revalidateAdmittedGoExecutable(admission);
  let result;
  try {
    result = spawnSync(admission.path, versionArgs, {
      cwd: repoRoot,
      encoding: "utf8",
      env: {
        HOME: repoRoot,
        LANG: "C",
        LC_ALL: "C",
        NO_COLOR: "1",
        PATH: dirname(admission.path),
        TMPDIR: "/tmp",
        TZ: "UTC",
        GIT_CONFIG_NOSYSTEM: "1",
        GIT_CONFIG_GLOBAL: "/dev/null",
        GIT_TERMINAL_PROMPT: "0",
      },
      timeout: 10_000,
      maxBuffer: 64 * 1024,
    });
  } finally {
    revalidateAdmittedGoExecutable(admission);
  }
  const stdout = typeof result?.stdout === "string" ? result.stdout.trim() : "";
  const stderr = typeof result?.stderr === "string" ? result.stderr.trim() : "";
  if (result?.error || result?.signal || result?.status !== 0 || stderr !== "" || !versionPattern.test(stdout)) {
    throw new MutationGateError(
      "U4_STUDY_TOOL_INSPECTION_FAILED",
      `${label} did not prove one closed executable version: status=${result?.status} signal=${result?.signal} stdout=${JSON.stringify(stdout)} stderr=${JSON.stringify(stderr)}`,
    );
  }
  return stdout;
}

async function admitU4StudyTools(sourceEnvironment = process.env) {
  const nodePath = sourceEnvironment.COUNTERSHAPE_NODE || process.execPath;
  const gitPath = sourceEnvironment.COUNTERSHAPE_GIT || "/usr/bin/git";
  const node = await resolveAdmittedGoExecutable({ COUNTERSHAPE_GO: nodePath });
  const git = await resolveAdmittedGoExecutable({ COUNTERSHAPE_GO: gitPath });
  const nodeVersion = inspectU4StudyExecutable("Node", node, ["--version"], /^v[0-9]+\.[0-9]+\.[0-9]+[^\r\n]{0,100}$/u);
  const gitVersion = inspectU4StudyExecutable("Git", git, ["--version"], /^git version [^\r\n]{1,200}$/u);
  return Object.freeze({
    node: Object.freeze({ admission: node, version: nodeVersion }),
    git: Object.freeze({ admission: git, version: gitVersion }),
  });
}

function revalidateU4StudyTools(studyTools) {
  revalidateAdmittedGoExecutable(studyTools.node.admission);
  revalidateAdmittedGoExecutable(studyTools.git.admission);
}

function proveU4StudyToolsStable(studyTools) {
  const nodeVersion = inspectU4StudyExecutable(
    "Node",
    studyTools.node.admission,
    ["--version"],
    /^v[0-9]+\.[0-9]+\.[0-9]+[^\r\n]{0,100}$/u,
  );
  const gitVersion = inspectU4StudyExecutable(
    "Git",
    studyTools.git.admission,
    ["--version"],
    /^git version [^\r\n]{1,200}$/u,
  );
  if (nodeVersion !== studyTools.node.version || gitVersion !== studyTools.git.version) {
    throw new MutationGateError("U4_STUDY_TOOL_VERSION_CHANGED", "admitted Node or Git version output changed");
  }
  revalidateU4StudyTools(studyTools);
}

async function prepareU4CacheRoot(cacheRoot) {
  await mkdir(cacheRoot, { mode: 0o700 });
  for (const name of ["home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"]) {
    const directory = join(cacheRoot, name);
    await mkdir(directory, { recursive: true, mode: 0o700 });
    await chmod(directory, 0o700);
  }
}

async function firstExecutableOnClosedPath(toolDirectories, name) {
  for (const directory of toolDirectories) {
    const candidate = join(directory, name);
    let resolved;
    let metadata;
    try {
      resolved = await realpath(candidate);
      metadata = await lstat(resolved);
    } catch (error) {
      if (error?.code === "ENOENT" || error?.code === "ENOTDIR") continue;
      throw new MutationGateError("U4_CLOSED_PATH_INSPECTION_FAILED", `${candidate}: ${error.code ?? error.message}`);
    }
    if (metadata.isFile() && (metadata.mode & 0o111) !== 0) return resolved;
  }
  return "";
}

async function assertU4ClosedToolPath(toolDirectories, toolchain) {
  for (const [name, expected] of [
    ["go", toolchain.executable.path],
    ["clang", toolchain.compiler.path],
    ["git", toolchain.studyTools.git.admission.path],
    ["node", toolchain.studyTools.node.admission.path],
  ]) {
    const resolved = await firstExecutableOnClosedPath(toolDirectories, name);
    if (resolved !== expected) {
      throw new MutationGateError("U4_CLOSED_PATH_SHADOWED", `${name} resolved to ${resolved || "nothing"}, want ${expected}`);
    }
  }
}

async function prepareU4Experiment({ mutant, toolchain, phase, seed, phaseRecords }) {
  await assertU4SourceAndSeedUnchanged(seed);
  const sandboxParent = await mkdtemp(join(tmpdir(), `countershape-u4-${phase}-`));
  await chmod(sandboxParent, 0o700);
  const sandbox = join(sandboxParent, "repo");
  const cacheRoot = join(sandboxParent, "cache");
  try {
    await copyRegularAllowlist(seed.root, sandbox, seed.allowlist);
    const sourceManifest = await validateU4TreeAndManifest(sandbox, seed.allowlist, seed.scopeRoots, seed.reviewedNonGoFiles);
    requireU4ManifestEqual(sourceManifest, seed.manifest, "U4_EXPERIMENT_SEED_MISMATCH", `${mutant.id} ${phase} copy differs from seed`);
    await prepareU4CacheRoot(cacheRoot);
    proveU4StudyToolsStable(toolchain.studyTools);
    const baseEnvironment = minimalU2GoEnvironment(cacheRoot, toolchain);
    const toolDirectories = [...new Set([
      dirname(toolchain.executable.path),
      dirname(toolchain.compiler.path),
      dirname(toolchain.studyTools.git.admission.path),
      dirname(toolchain.studyTools.node.admission.path),
    ])];
    await assertU4ClosedToolPath(toolDirectories, toolchain);
    return {
      sandboxParent,
      sandbox,
      cacheRoot,
      environment: Object.freeze({ ...baseEnvironment, PATH: toolDirectories.join(":") }),
      toolchain,
      phase,
      sourceManifest,
      u4Seed: seed,
      phaseRecords,
    };
  } catch (error) {
    await rm(sandboxParent, { recursive: true, force: true });
    throw error;
  }
}

async function cleanupU4Experiment(context) {
  await rm(context.sandboxParent, { recursive: true, force: true });
}

async function expectedU4ExperimentManifest(context, mutant) {
  if (context.phase === "mutant") {
    return assertExactU4MutantDelta(context.sandbox, context.u4Seed, mutant);
  }
  const manifest = await validateU4TreeAndManifest(
    context.sandbox,
    context.u4Seed.allowlist,
    context.u4Seed.scopeRoots,
    context.u4Seed.reviewedNonGoFiles,
  );
  requireU4ManifestEqual(
    manifest,
    context.u4Seed.manifest,
    "U4_CONTROL_SOURCE_CHANGED",
    `${mutant.id} ${context.phase} source differs from the immutable seed`,
  );
  return manifest;
}

async function runNamedU4GoTest(context, mutant, toolchain) {
  await assertU4SourceAndSeedUnchanged(context.u4Seed);
  const before = await expectedU4ExperimentManifest(context, mutant);
  proveU4StudyToolsStable(toolchain.studyTools);
  let result;
  try {
    result = runNamedU2GoTest({
      sandbox: context.sandbox,
      mutant,
      environment: context.environment,
      toolchain,
    });
  } finally {
    proveU4StudyToolsStable(toolchain.studyTools);
  }
  const after = await expectedU4ExperimentManifest(context, mutant);
  requireU4ManifestEqual(after, before, "U4_TEST_CHANGED_SOURCE", `${mutant.id} ${context.phase} named test changed admitted source`);
  await assertU4SourceAndSeedUnchanged(context.u4Seed);
  context.phaseRecords.push(Object.freeze({ phase: context.phase, manifest: before.digest, sandbox: context.sandbox }));
  return result;
}

function exactU4ABADigest(mutant, phaseRecords, seed) {
  if (phaseRecords.length !== 3 || !sameArray(phaseRecords.map((record) => record.phase), ["baseline", "mutant", "post-control"]) ||
    new Set(phaseRecords.map((record) => record.sandbox)).size !== 3 ||
    phaseRecords[0].manifest !== seed.manifest.digest || phaseRecords[2].manifest !== seed.manifest.digest ||
    phaseRecords[1].manifest === seed.manifest.digest) {
    throw new MutationGateError("U4_ABA_RECEIPT_INVALID", `${mutant.id} did not retain exact fresh A/B/A closure facts`);
  }
  const canonical = phaseRecords
    .map((record) => `${record.phase}\u0000${record.manifest}\u0000${record.sandbox}`)
    .join("\n");
  return `sha256:${createHash("sha256").update(canonical, "utf8").digest("hex")}`;
}

export async function selfTest() {
  assertU4MutantDefinitionSet();
  const selfManifest = await assertU4ManifestMatchesTree(repoRoot);
  for (const file of U4_REVIEWED_NON_GO_FILES) {
    const entry = selfManifest.entries.find((candidate) => candidate.file === file);
    assert.ok(entry && entry.bytes > 0 && /^[0-9a-f]{64}$/u.test(entry.sha256), `${file} lacks exact manifest bytes`);
  }
  await assertReviewedU4Anchors(repoRoot);
  await assertReviewedU4NamedTests(repoRoot);

  await assert.rejects(
    () => assertU4ManifestMatchesTree(
      repoRoot,
      U4_SANDBOX_FILE_ALLOWLIST.filter((file) => file !== U4_REVIEWED_NON_GO_FILES[0]),
      U4_SOURCE_SCOPE_ROOTS,
      U4_REVIEWED_NON_GO_FILES,
    ),
    (error) => error instanceof MutationGateError && error.code === "U4_NON_GO_ALLOWLIST_MISMATCH",
  );
  await assert.rejects(
    () => assertU4ManifestMatchesTree(
      repoRoot,
      U4_SANDBOX_FILE_ALLOWLIST,
      U4_SOURCE_SCOPE_ROOTS,
      [...U4_REVIEWED_NON_GO_FILES, "testkit/httpfixture/unreviewed.asset"],
    ),
    (error) => error instanceof MutationGateError && error.code === "U4_NON_GO_ALLOWLIST_MISMATCH",
  );

  assert.throws(
    () => assertU4MutantDefinitionSet(U4_MUTANTS.slice(0, -1)),
    (error) => error instanceof MutationGateError && error.code === "MUTANT_SET_MISMATCH",
  );
  const reordered = [...U4_MUTANTS];
  [reordered[0], reordered[1]] = [reordered[1], reordered[0]];
  assert.throws(
    () => assertU4MutantDefinitionSet(reordered),
    (error) => error instanceof MutationGateError && error.code === "MUTANT_SET_MISMATCH",
  );
  const tampered = U4_MUTANTS.map((mutant) => ({ ...mutant }));
  tampered[0].replace += "\n// unreviewed weakening";
  assert.throws(
    () => assertU4MutantDefinitionSet(tampered),
    (error) => error instanceof MutationGateError && error.code === "MUTANT_CONTRACT_MISMATCH",
  );

  const phases = [];
  let invocation = 0;
  const killed = await runU2Mutant(U4_MUTANTS[0], {}, {
    async prepareExperiment({ phase }) {
      phases.push(phase);
      return { sandbox: `/fresh/u4/${phase}`, sandboxParent: `/fresh/u4/${phase}` };
    },
    async cleanupExperiment() {},
    async applyMutation() {},
    async runNamedTest() {
      const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass];
      return fakeClassification(outcomes[invocation++]);
    },
  });
  assert.match(killed, /^KILLED readiness-by-http-request/u);
  assert.deepEqual(phases, ["baseline", "mutant", "post-control"]);
  assert.equal(invocation, 3);
  const syntheticSeed = { manifest: { digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } };
  const syntheticRecords = [
    { phase: "baseline", manifest: syntheticSeed.manifest.digest, sandbox: "/fresh/u4/a" },
    { phase: "mutant", manifest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", sandbox: "/fresh/u4/b" },
    { phase: "post-control", manifest: syntheticSeed.manifest.digest, sandbox: "/fresh/u4/c" },
  ];
  assert.match(exactU4ABADigest(U4_MUTANTS[0], syntheticRecords, syntheticSeed), /^sha256:[0-9a-f]{64}$/u);
  assert.throws(
    () => exactU4ABADigest(U4_MUTANTS[0], syntheticRecords.map((record) => ({ ...record, sandbox: "/reused" })), syntheticSeed),
    (error) => error instanceof MutationGateError && error.code === "U4_ABA_RECEIPT_INVALID",
  );
  return `U4 mutation self-test passed: ${U4_MUTANTS.length} frozen tuples, exact manifest, exact anchors, fresh A/B/A control.`;
}

export async function main(arguments_ = process.argv.slice(2)) {
  if (arguments_.length === 1 && arguments_[0] === "--self-test") {
    process.stdout.write(`${await selfTest()}\n`);
    return;
  }
  if (arguments_.length !== 0) {
    throw new MutationGateError("UNSUPPORTED_ARGUMENT", "usage: node tools/mutate-u4.mjs [--self-test]");
  }

  assertU4MutantDefinitionSet();
  await assertU4ManifestMatchesTree(repoRoot);
  await assertReviewedU4Anchors(repoRoot);
  await assertReviewedU4NamedTests(repoRoot);
  const baseToolchain = await admitU2DarwinToolchain();
  const studyTools = await admitU4StudyTools();
  const toolchain = Object.freeze({ ...baseToolchain, studyTools });
  const seed = await createU4SeedSnapshot({ sourceRoot: repoRoot });
  const results = [];
  try {
    for (const mutant of U4_MUTANTS) {
      const phaseRecords = [];
      const result = await runU2Mutant(mutant, toolchain, {
        prepareExperiment({ mutant: currentMutant, toolchain: currentToolchain, phase }) {
          return prepareU4Experiment({ mutant: currentMutant, toolchain: currentToolchain, phase, seed, phaseRecords });
        },
        cleanupExperiment: cleanupU4Experiment,
        applyMutation: mutateRegularFileNoFollow,
        runNamedTest(context, currentMutant) {
          return runNamedU4GoTest(context, currentMutant, toolchain);
        },
      });
      results.push(`${result}; exact fresh A/B/A closure digest ${exactU4ABADigest(mutant, phaseRecords, seed)}`);
    }
    await assertU4SourceAndSeedUnchanged(seed);
  } finally {
    await cleanupU4SeedSnapshot(seed);
  }
  revalidateAdmittedGoExecutable(toolchain.executable);
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
  proveU4StudyToolsStable(toolchain.studyTools);
  process.stdout.write(`Trusted Go realpath: ${toolchain.executable.path}\n`);
  process.stdout.write(`Trusted Go sha256: ${toolchain.executable.sha256}\n`);
  process.stdout.write(`Trusted Go version: ${toolchain.version.line}\n`);
  process.stdout.write(`Trusted C compiler realpath: ${toolchain.compiler.path}\n`);
  process.stdout.write(`Trusted C compiler sha256: ${toolchain.compiler.sha256}\n`);
  process.stdout.write(`Closed Darwin CGO facts digest: ${toolchain.cgo.digest}\n`);
  process.stdout.write(`Trusted U4 Git realpath: ${toolchain.studyTools.git.admission.path}\n`);
  process.stdout.write(`Trusted U4 Git sha256: ${toolchain.studyTools.git.admission.sha256}\n`);
  process.stdout.write(`Trusted U4 Git version: ${toolchain.studyTools.git.version}\n`);
  process.stdout.write(`Trusted U4 Node realpath: ${toolchain.studyTools.node.admission.path}\n`);
  process.stdout.write(`Trusted U4 Node sha256: ${toolchain.studyTools.node.admission.sha256}\n`);
  process.stdout.write(`Trusted U4 Node version: ${toolchain.studyTools.node.version}\n`);
  process.stdout.write(`Immutable U4 seed manifest: ${seed.manifest.digest}\n`);
  for (const file of U4_REVIEWED_NON_GO_FILES) {
    const entry = seed.manifest.entries.find((candidate) => candidate.file === file);
    if (!entry) throw new MutationGateError("U4_NON_GO_MANIFEST_ENTRY_MISSING", `${file} is absent from the final seed manifest`);
    process.stdout.write(`Reviewed U4 non-Go source: ${entry.file} bytes=${entry.bytes} sha256=${entry.sha256}\n`);
  }
  process.stdout.write("Containment notice: private mutation copies retain the invoking user's host filesystem and network authority.\n");
  for (const result of results) process.stdout.write(`${result}\n`);
  process.stdout.write(`U4 mutation gate killed ${results.length}/${REQUIRED_U4_MUTANT_IDS.length} exact required mutants.\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
