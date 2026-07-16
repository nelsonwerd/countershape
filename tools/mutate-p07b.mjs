#!/usr/bin/env node

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { constants } from "node:fs";
import { chmod, mkdir, mkdtemp, open, readFile, realpath, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
  MutationGateError,
  NamedTestOutcome,
  copyRegularAllowlist,
  mutateRegularFileNoFollow,
  revalidateAdmittedGoExecutable,
} from "./mutate-u1.mjs";
import {
  admitU2DarwinToolchain,
  minimalU2GoEnvironment,
  revalidateAdmittedDarwinCCompiler,
  runNamedU2GoTest,
  runU2Mutant,
} from "./mutate-u2.mjs";
import {
  U5_REVIEWED_EMBED_BINDINGS,
  U5_REVIEWED_NON_GO_FILES,
  assertExactU5MutantDelta,
  assertU5ManifestMatchesTree,
  assertU5PrivateCopy,
  assertU5SourceAndSeedUnchanged,
  cleanupU5SeedSnapshot,
  createU5SeedSnapshot,
} from "./mutate-u5.mjs";
import {
  U6_RUNTIME_SUPPORT_FILES,
  U6_SANDBOX_FILE_ALLOWLIST,
  U6_SOURCE_SCOPE_ROOTS,
} from "./mutate-u6.mjs";

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = resolve(dirname(modulePath), "..");

export const P07B_ADDITIONAL_FILES = Object.freeze([
  "internal/adapters/cli/model/projection_schema.go",
  "internal/adapters/cli/model/projection_schema_test.go",
  "internal/adapters/http/model/portable_start_test.go",
  "internal/adapters/http/model/projection_schema.go",
  "internal/adapters/http/model/projection_schema_test.go",
  "internal/contractsource/errors.go",
  "internal/contractsource/profile_roster.go",
  "internal/contractsource/source.go",
  "internal/contractsource/source_test.go",
  "internal/runnerprofile/profile.go",
  "internal/world/http_portable_negative_darwin_test.go",
  "testkit/httpfixture/server_child_bind.mjs",
]);

export const P07B_RUNTIME_SUPPORT_FILES = Object.freeze([
  ...U6_RUNTIME_SUPPORT_FILES,
	"spec/schema/v1/common.schema.json",
	"spec/schema/v1/choicepoint.schema.json",
	"spec/schema/v1/decision-record.schema.json",
  "tools/check-u5-architecture.mjs",
  "tools/check-u6-architecture.mjs",
  "tools/check-p07b-architecture.mjs",
  "tools/check-p07b-architecture-selftest.mjs",
	"tools/mutate-p07b.mjs",
]);

export const P07B_SANDBOX_FILE_ALLOWLIST = Object.freeze([
  ...U6_SANDBOX_FILE_ALLOWLIST,
  ...P07B_ADDITIONAL_FILES,
  ...P07B_RUNTIME_SUPPORT_FILES.filter((file) => !U6_RUNTIME_SUPPORT_FILES.includes(file)),
]);

export const P07B_SOURCE_SCOPE_ROOTS = Object.freeze([
  ...U6_SOURCE_SCOPE_ROOTS,
  "internal/contractsource",
  "internal/runnerprofile",
]);

export const P07B_REVIEWED_NON_GO_FILES = Object.freeze([
  ...U5_REVIEWED_NON_GO_FILES,
  "testkit/httpfixture/server_child_bind.mjs",
]);

export const P07B_REVIEWED_EMBED_BINDINGS = Object.freeze([
  ...U5_REVIEWED_EMBED_BINDINGS,
  Object.freeze({
    owner: "testkit/httpfixture/fixture.go",
    asset: "testkit/httpfixture/server_child_bind.mjs",
    directive: "//go:embed server_child_bind.mjs",
  }),
]);

export const REQUIRED_P07B_MUTANT_IDS = Object.freeze([
  "cli-accept-invented-projection-schema",
  "http-accept-invented-projection-schema",
  "profile-accept-unknown-member",
  "source-accept-unregenerated-envelope",
  "source-drop-adapter-projection-payload",
  "source-http-view-drop-projection-authority",
  "collapse-portable-start-onto-legacy",
  "accept-wrong-port-frame-prefix",
  "accept-zero-port-frame",
  "encode-portable-request-before-frame",
  "drop-rejected-readiness-only-receipt",
  "ignore-readiness-authority-tamper",
  "unbind-portable-frame-digest",
  "reexpose-legacy-frame-bytes",
  "relabel-portable-measurement-as-legacy",
  "select-legacy-tree-for-portable-study",
  "select-legacy-entrypoint-for-portable-study",
  "corrupt-legacy-fixture-ready-byte",
	"accept-cli-profile-order-alias",
	"accept-plan-secret-slots",
	"accept-wrong-node-tool-constraint",
	"accept-wrong-runner-lineage",
	"raise-source-canonical-ceiling",
	"accept-port-frame-without-eof",
	"ignore-port-endpoint-join",
	"admit-accepted-readiness-without-request",
	"add-source-transitive-gitobj-dependency",
	"comment-spoof-closed-cli-schema",
	"collapse-portable-runner-lineage",
	"collapse-child-bind-onto-readiness-fd",
]);

const REQUIRED_P07B_MUTANT_ID_DIGEST = "sha256:79036a0a86685d70d62c3239ce1488d16ccfd9738277c199a7685f547f6f369a";

export const REVIEWED_P07B_MUTANT_CONTRACT = Object.freeze([
  Object.freeze({
    id: "cli-accept-invented-projection-schema",
    file: "internal/adapters/cli/model/projection_authority.go",
    find: "\texpected, err := NewCLIProjectionAuthority(identity.Fields)",
    replace: [
      "\texpected, err := NewCLIProjectionAuthority(identity.Fields)",
      "\tif err == nil {",
      "\t\texpected = CLIProjectionAuthority{digest: digest, canonicalBytes: append([]byte(nil), canonicalBytes...), binding: binding, fieldIDs: append([]string(nil), identity.Fields...)}",
      "\t}",
    ].join("\n"),
    package: "./internal/adapters/cli/model",
    testName: "TestCLIProjectionAuthorityRejectsSelfConsistentInventedSchemas",
  }),
  Object.freeze({
    id: "http-accept-invented-projection-schema",
    file: "internal/adapters/http/model/projection_authority.go",
    find: "\texpected, err := NewHTTPProjectionAuthority()",
    replace: [
      "\texpected, err := NewHTTPProjectionAuthority()",
      "\tif err == nil {",
      "\t\texpected = HTTPProjectionAuthority{digest: digest, canonicalBytes: append([]byte(nil), canonicalBytes...), binding: binding, fieldIDs: append([]string(nil), identity.Fields...)}",
      "\t}",
    ].join("\n"),
    package: "./internal/adapters/http/model",
    testName: "TestHTTPProjectionAuthorityRejectsSelfConsistentInventedSchemas",
  }),
  Object.freeze({
    id: "profile-accept-unknown-member",
    file: "internal/projectionprofile/profile.go",
    find: "\tif err != nil || !bytes.Equal(rebuilt.CanonicalBytes(), exact) {",
    replace: "\tif err != nil {",
    package: "./internal/projectionprofile",
    testName: "TestParseRequiresExactTypedRegeneration",
  }),
  Object.freeze({
    id: "source-accept-unregenerated-envelope",
    file: "internal/contractsource/source.go",
    find: "\tif !bytes.Equal(rebuilt.canonical, exact) {",
    replace: "\tif false && !bytes.Equal(rebuilt.canonical, exact) {",
    package: "./internal/contractsource",
    testName: "TestPortableSourceRequiresExactAuthorityJoin",
  }),
  Object.freeze({
    id: "source-drop-adapter-projection-payload",
    file: "internal/contractsource/source.go",
    find: "\t\tAdapterProjectionBase64: base64.StdEncoding.EncodeToString(adapterProjectionBytes),",
    replace: "\t\tAdapterProjectionBase64: base64.StdEncoding.EncodeToString(nil),",
    package: "./internal/contractsource",
    testName: "TestPortableSourceReconstructsExactCLIAndHTTPAuthorities",
  }),
  Object.freeze({
    id: "source-http-view-drop-projection-authority",
    file: "internal/contractsource/source.go",
    find: "\t\tprojection: s.authorities.httpProjection, seal: portableSourceAuthority,",
    replace: "\t\tprojection: counterhttp.HTTPProjectionAuthority{}, seal: portableSourceAuthority,",
    package: "./internal/contractsource",
    testName: "TestPortableSourceViewsRecoverExactRawPayloadsWithoutPrivateJSONParsing",
  }),
  Object.freeze({
    id: "collapse-portable-start-onto-legacy",
    file: "internal/adapters/http/model/start.go",
    find: "\treturn newHTTPStartSpec(entrypoint, HTTPPortableStartAuthorityV1)",
    replace: "\treturn newHTTPStartSpec(entrypoint, HTTPStartAuthorityV1)",
    package: "./internal/adapters/http/model",
    testName: "TestPortableHTTPStartIsDistinctWithoutRewritingHistoricalBytes",
  }),
  Object.freeze({
    id: "accept-wrong-port-frame-prefix",
    file: "internal/adapters/http/model/start.go",
	find: "\tPortableReadinessFramePrefix        = \"COUNTERSHAPE_READY_V1 \"",
	replace: "\tPortableReadinessFramePrefix        = \"COUNTERSHAPE_READY_V2 \"",
    package: "./internal/adapters/http/model",
    testName: "TestPortableHTTPReadyPortFrameUsesOneStrictGrammar",
  }),
  Object.freeze({
    id: "accept-zero-port-frame",
    file: "internal/adapters/http/model/start.go",
	find: "\tif port < 1 || port > 65535 {",
	replace: "\tif port < 0 || port > 65535 {",
    package: "./internal/adapters/http/model",
    testName: "TestPortableHTTPReadyPortFrameUsesOneStrictGrammar",
  }),
  Object.freeze({
    id: "encode-portable-request-before-frame",
    file: "internal/world/http_service_darwin.go",
    find: "\tif !portable {\n\t\trequestWire, err = httpmodel.EncodeRequest(request.binding.Stimulus(), port)",
    replace: "\tif true {\n\t\trequestWire, err = httpmodel.EncodeRequest(request.binding.Stimulus(), port)",
    package: "./internal/world",
    testName: "TestHTTPPortableReadinessFailuresRemainFinalizedReceipts",
  }),
  Object.freeze({
    id: "drop-rejected-readiness-only-receipt",
    file: "internal/world/http_api.go",
    find: "\treturn executionAuthority == httpmodel.HTTPPortableExecutionAuthorityV1 &&",
    replace: "\treturn false && executionAuthority == httpmodel.HTTPPortableExecutionAuthorityV1 &&",
    package: "./internal/world",
    testName: "TestHTTPPortableReadinessFailuresRemainFinalizedReceipts",
  }),
  Object.freeze({
    id: "ignore-readiness-authority-tamper",
    file: "internal/world/http_receipt.go",
    find: "\treturn err == nil && rebuilt.digest == r.digest && rebuilt.authority == r.authority &&",
    replace: "\treturn err == nil && rebuilt.digest == r.digest && true &&",
    package: "./internal/world",
    testName: "TestHTTPPortableReadinessBindsExactChildReportedPortFrame",
  }),
  Object.freeze({
    id: "unbind-portable-frame-digest",
    file: "internal/world/http_receipt.go",
    find: "\t\tFrameDigest: frameDigest.String(), BytesObserved: physical.bytesObserved, EOFObserved: physical.eofObserved,",
    replace: "\t\tFrameDigest: \"\", BytesObserved: physical.bytesObserved, EOFObserved: physical.eofObserved,",
    package: "./internal/world",
    testName: "TestHTTPPortableReadinessBindsExactChildReportedPortFrame",
  }),
  Object.freeze({
    id: "reexpose-legacy-frame-bytes",
    file: "internal/world/http_service_darwin.go",
    find: "\t\tif portable {\n\t\t\tresult.readiness.frameBytes = append([]byte(nil), readiness.bytes...)\n\t\t}",
    replace: "\t\tresult.readiness.frameBytes = append([]byte(nil), readiness.bytes...)",
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoiceReferenceStudy",
  }),
  Object.freeze({
    id: "relabel-portable-measurement-as-legacy",
    file: "testkit/studies/http_invoices/study.go",
    find: "\t\t{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, binding.Authority()},",
    replace: "\t\t{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, counterhttp.HTTPExecutionAuthorityV1},",
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoicePortableChildBindPhysicalLineage",
  }),
  Object.freeze({
    id: "select-legacy-tree-for-portable-study",
    file: "testkit/studies/http_invoices/study.go",
    find: "\t\tfixtureFiles = httpfixture.PortableCandidateFiles",
    replace: "\t\tfixtureFiles = httpfixture.CandidateFiles",
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoicePortableChildBindPhysicalLineage",
  }),
  Object.freeze({
    id: "select-legacy-entrypoint-for-portable-study",
    file: "testkit/studies/http_invoices/study.go",
    find: "\t\tentrypoint = httpfixture.PortableEntrypoint",
    replace: "\t\tentrypoint = httpfixture.Entrypoint",
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoicePortableChildBindPhysicalLineage",
  }),
  Object.freeze({
    id: "corrupt-legacy-fixture-ready-byte",
    file: "testkit/httpfixture/server.mjs",
    find: "Buffer.from([0x01])",
    replace: "Buffer.from([0x02])",
    package: "./testkit/studies/http_invoices",
    testName: "TestHTTPInvoiceReferenceStudy",
  }),
  Object.freeze({
	id: "accept-cli-profile-order-alias",
	file: "internal/contractsource/profile_roster.go",
	find: [
	  "\tpreviousRosterIndex := -1",
	  "\tfor index, field := range fields {",
	  "\t\texpected, ok := cliProfileRosterV1[projectionFields[index]]",
	  "\t\trosterIndex := slices.Index(cliProfileOrderV1, projectionFields[index])",
	  "\t\tif !ok || rosterIndex <= previousRosterIndex || field.FieldID != projectionFields[index] ||",
	  "\t\t\t!equalDescriptor(field, expected) {",
	  "\t\t\treturn refuse(CodeSourceAuthorityMismatch, \"CLI profile is outside the closed v1 descriptor roster\", nil)",
	  "\t\t}",
	  "\t\tpreviousRosterIndex = rosterIndex",
	  "\t}",
	].join("\n"),
	replace: [
	  "\tfor _, field := range fields {",
	  "\t\texpected, ok := cliProfileRosterV1[field.FieldID]",
	  "\t\tif !ok || !equalDescriptor(field, expected) {",
	  "\t\t\treturn refuse(CodeSourceAuthorityMismatch, \"CLI profile is outside the closed v1 descriptor roster\", nil)",
	  "\t\t}",
	  "\t}",
	].join("\n"),
	package: "./internal/contractsource",
	testName: "TestPortableSourceRejectsCLIProfileOrderAlias",
  }),
  Object.freeze({
	id: "accept-plan-secret-slots",
	file: "internal/contractsource/source.go",
	find: "\tif len(identity.PlanSecretSlots) != 0 {",
	replace: "\tif false && len(identity.PlanSecretSlots) != 0 {",
	package: "./internal/contractsource",
	testName: "TestPortableSourceRequiresExactRunnerToolSecretsAndEntrypoint",
  }),
  Object.freeze({
	id: "accept-wrong-node-tool-constraint",
	file: "internal/contractsource/source.go",
	find: "\tif len(tools) != 1 || tools[0].Name != \"node\" || tools[0].VersionConstraint != NodeToolConstraintV1 {",
	replace: "\tif len(tools) != 1 || tools[0].Name != \"node\" || false {",
	package: "./internal/contractsource",
	testName: "TestPortableSourceRequiresExactRunnerToolSecretsAndEntrypoint",
  }),
  Object.freeze({
	id: "accept-wrong-runner-lineage",
	file: "internal/contractsource/source.go",
	find: "\tif err != nil || plan.Adapter().RunnerDigest != expectedRunner {",
	replace: "\tif err != nil || plan.Adapter().RunnerDigest != expectedRunner && false {",
	package: "./internal/contractsource",
	testName: "TestPortableSourceRequiresExactRunnerToolSecretsAndEntrypoint",
  }),
  Object.freeze({
	id: "raise-source-canonical-ceiling",
	file: "internal/contractsource/source.go",
	find: "\tif err != nil || len(canonicalBytes) == 0 || len(canonicalBytes) > MaxSourceCanonicalBytes {",
	replace: "\tif err != nil || len(canonicalBytes) == 0 || len(canonicalBytes) > MaxSourceCanonicalBytes+1 {",
	package: "./internal/contractsource",
	testName: "TestPortableSourceCeilingIsExactAndDefensive",
  }),
  Object.freeze({
	 id: "accept-port-frame-without-eof",
	file: "internal/world/http_receipt.go",
	find: "\tframeValid := frameErr == nil && frame.Valid() && physical.eofObserved",
	replace: "\tframeValid := frameErr == nil && frame.Valid()",
	package: "./internal/world",
	testName: "TestHTTPPortableReadinessBindsExactChildReportedPortFrame",
  }),
  Object.freeze({
	id: "ignore-port-endpoint-join",
	file: "internal/world/http_receipt.go",
	find: "\t\tif physical.port != int(frame.Port()) || physical.endpoint != \"127.0.0.1:\"+strconv.Itoa(physical.port) {",
	replace: "\t\tif physical.port != int(frame.Port()) {",
	package: "./internal/world",
	testName: "TestHTTPPortableReadinessBindsExactChildReportedPortFrame",
  }),
  Object.freeze({
	id: "admit-accepted-readiness-without-request",
	file: "internal/world/http_api.go",
	find: "\t\treadiness.protocol == httpmodel.PortableReadinessProtocolV1 && !readiness.accepted",
	replace: "\t\treadiness.protocol == httpmodel.PortableReadinessProtocolV1",
	package: "./internal/world",
	testName: "TestAcceptedPortableReadinessWithoutRequestRemainsInvariantFailure",
  }),
  Object.freeze({
	id: "add-source-transitive-gitobj-dependency",
	file: "internal/runnerprofile/profile.go",
	find: "import (\n\t\"strings\"",
	replace: "import (\n\t\"strings\"\n\n\t_ \"github.com/nelsonwerd/countershape/internal/gitobj\"",
	runner: "architecture",
	expectedArchitectureCode: "P07B_SOURCE_TRANSITIVE_INTERNAL_DEPENDENCY",
  }),
  Object.freeze({
	id: "comment-spoof-closed-cli-schema",
	file: "internal/adapters/cli/model/projection_authority.go",
	find: "\texpected, err := NewCLIProjectionAuthority(identity.Fields)",
	replace: [
	  "\t// NewCLIProjectionAuthority(identity.Fields) is intentionally only a spoofed comment.",
	  "\tvar err error",
	  "\texpected := CLIProjectionAuthority{digest: digest, canonicalBytes: append([]byte(nil), canonicalBytes...), binding: binding, fieldIDs: append([]string(nil), identity.Fields...)}",
	].join("\n"),
	runner: "architecture",
	expectedArchitectureCode: "P07B_CLI_PROJECTION_RESOLUTION_OPEN",
  }),
  Object.freeze({
	id: "collapse-portable-runner-lineage",
	file: "internal/runnerprofile/profile.go",
	find: "\tHTTPPortableRunnerLineageV1 = \"world.ExecuteHTTP/P07B/opaque-binding/child-bind-pipe-ready/v1\"",
	replace: "\tHTTPPortableRunnerLineageV1 = \"world.ExecuteHTTP/U4/opaque-binding/inherited-listener/v1\"",
	runner: "architecture",
	expectedArchitectureCode: "P07B_RUNNER_PROFILE_MISSING",
  }),
  Object.freeze({
	id: "collapse-child-bind-onto-readiness-fd",
	file: "testkit/httpfixture/server_child_bind.mjs",
	find: "server.listen({ host: \"127.0.0.1\", port: 0, exclusive: true }, () => {",
	replace: "server.listen({ fd: readinessFD, exclusive: true }, () => {",
	package: "./testkit/studies/http_invoices",
	testName: "TestHTTPInvoicePortableChildBindPhysicalLineage",
  }),
].map((tuple) => Object.freeze({
  runner: "go-test", package: "", testName: "", expectedArchitectureCode: "", ...tuple,
})));

const tupleFields = Object.freeze(["expectedArchitectureCode", "file", "find", "id", "package", "replace", "runner", "testName"]);
export const P07B_MUTANTS = Object.freeze(REVIEWED_P07B_MUTANT_CONTRACT.map((tuple) => Object.freeze({ ...tuple })));

function sha256(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

function exactOccurrenceCount(source, target) {
  if (typeof target !== "string" || target.length === 0) return 0;
  return source.split(target).length - 1;
}

function sameArray(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

export function assertP07BMutantDefinitionSet(mutants = P07B_MUTANTS) {
  if (!Object.isFrozen(mutants) || mutants.some((mutant) => !Object.isFrozen(mutant))) {
    throw new MutationGateError("P07B_MUTANT_SET_NOT_IMMUTABLE", "mutant roster and tuples must be recursively frozen");
  }
  const ids = mutants.map((mutant) => mutant.id);
  const digest = `sha256:${sha256(Buffer.from(REQUIRED_P07B_MUTANT_IDS.join("\n"), "utf8"))}`;
  if (digest !== REQUIRED_P07B_MUTANT_ID_DIGEST || !sameArray(ids, REQUIRED_P07B_MUTANT_IDS) || new Set(ids).size !== ids.length) {
    throw new MutationGateError("P07B_MUTANT_SET_MISMATCH", `${digest}: [${ids.join(",")}]`);
  }
  if (mutants.length !== REVIEWED_P07B_MUTANT_CONTRACT.length) {
    throw new MutationGateError("P07B_MUTANT_CONTRACT_MISMATCH", "working roster differs from the reviewed tuple count");
  }
  for (let index = 0; index < mutants.length; index += 1) {
    const mutant = mutants[index];
    const reviewed = REVIEWED_P07B_MUTANT_CONTRACT[index];
    if (!sameArray(Object.keys(mutant).sort(), [...tupleFields].sort()) ||
        tupleFields.some((field) => mutant[field] !== reviewed[field])) {
      throw new MutationGateError("P07B_MUTANT_CONTRACT_MISMATCH", `${mutant.id} differs from reviewed tuple ${index}`);
    }
	if (mutant.find.length === 0 || mutant.find === mutant.replace || !P07B_SANDBOX_FILE_ALLOWLIST.includes(mutant.file)) {
	  throw new MutationGateError("P07B_MUTANT_CONTRACT_MISMATCH", `${mutant.id} is not one allowlisted effective replacement`);
	}
	if (mutant.runner === "go-test") {
	  if (!/^\.\/[^\s]+$/u.test(mutant.package) || !/^Test[A-Za-z0-9_]+$/u.test(mutant.testName) || mutant.expectedArchitectureCode !== "") {
		throw new MutationGateError("P07B_MUTANT_CONTRACT_MISMATCH", `${mutant.id} has invalid Go-test authority`);
	  }
	} else if (mutant.runner === "architecture") {
	  if (mutant.package !== "" || mutant.testName !== "" || !/^P07B_[A-Z0-9_]+$/u.test(mutant.expectedArchitectureCode)) {
		throw new MutationGateError("P07B_MUTANT_CONTRACT_MISMATCH", `${mutant.id} has invalid architecture authority`);
	  }
	} else {
	  throw new MutationGateError("P07B_MUTANT_CONTRACT_MISMATCH", `${mutant.id} has unknown runner ${mutant.runner}`);
	}
  }
}

export function assertP07BManifestMatchesTree(sourceRoot = repoRoot) {
  return assertU5ManifestMatchesTree(
    sourceRoot,
    P07B_SANDBOX_FILE_ALLOWLIST,
    P07B_SOURCE_SCOPE_ROOTS,
    P07B_REVIEWED_NON_GO_FILES,
    P07B_REVIEWED_EMBED_BINDINGS,
    P07B_RUNTIME_SUPPORT_FILES,
  );
}

export async function assertReviewedP07BAnchors(sourceRoot = repoRoot, mutants = P07B_MUTANTS) {
  assertP07BMutantDefinitionSet(mutants);
  await assertP07BManifestMatchesTree(sourceRoot);
  for (const mutant of mutants) {
    const source = await readFile(resolve(sourceRoot, mutant.file), "utf8");
    const count = exactOccurrenceCount(source, mutant.find);
    if (count !== 1) {
      throw new MutationGateError("P07B_MUTATION_ANCHOR_INVALID", `${mutant.id}: find=${count}`);
    }
  }
}

export async function assertReviewedP07BNamedTests(sourceRoot = repoRoot, mutants = P07B_MUTANTS) {
  for (const mutant of mutants) {
	if (mutant.runner !== "go-test") continue;
    const prefix = `${mutant.package.slice(2)}/`;
    let count = 0;
    for (const file of P07B_SANDBOX_FILE_ALLOWLIST) {
      if (!file.startsWith(prefix) || !file.endsWith("_test.go")) continue;
      const source = await readFile(resolve(sourceRoot, file), "utf8");
      count += exactOccurrenceCount(source, `func ${mutant.testName}(`);
    }
    if (count !== 1) {
      throw new MutationGateError("P07B_NAMED_TEST_INVALID", `${mutant.id}: ${mutant.package}.${mutant.testName} count=${count}`);
    }
  }
}

function manifestEqual(left, right) {
  return left.digest === right.digest && left.entries.length === right.entries.length &&
    left.entries.every((entry, index) => JSON.stringify(entry) === JSON.stringify(right.entries[index]));
}

async function admitExternalExecutable(name, candidate, versionArgs) {
  if (!isAbsolute(candidate)) throw new MutationGateError("P07B_EXTERNAL_PATH_NOT_ABSOLUTE", `${name}: ${candidate}`);
  const path = await realpath(candidate);
  const inspectionHome = await mkdtemp(join(await realpath(tmpdir()), `countershape-p07b-${name}-inspection-`));
  let before;
  try {
    await chmod(inspectionHome, 0o700);
    before = await open(path, constants.O_RDONLY | (constants.O_NOFOLLOW ?? 0));
    const metadata = await before.stat();
    if (!metadata.isFile() || (metadata.mode & 0o111) === 0) {
      throw new MutationGateError("P07B_EXTERNAL_NOT_EXECUTABLE", `${name}: ${path}`);
    }
    const bytes = await before.readFile();
    const result = spawnSync(path, versionArgs, {
	  encoding: "utf8", timeout: 10_000, maxBuffer: 256 * 1024,
	  env: { HOME: inspectionHome, TMPDIR: inspectionHome, LANG: "C", LC_ALL: "C", NO_COLOR: "1", PATH: "/usr/bin:/bin", TZ: "UTC" },
    });
    if (result.error || result.signal || result.status !== 0) {
      throw new MutationGateError("P07B_EXTERNAL_VERSION_FAILED", `${name}: ${result.status ?? result.signal}: ${result.stderr}`);
    }
    return Object.freeze({
      name, path, dev: metadata.dev, ino: metadata.ino, size: metadata.size, mode: metadata.mode,
      sha256: sha256(bytes), version: (result.stdout || result.stderr).trim().split(/\r?\n/u)[0],
    });
  } finally {
    try {
      await before?.close();
    } finally {
      await rm(inspectionHome, { recursive: true, force: true });
    }
  }
}

async function revalidateExternalExecutable(admission) {
  const current = await admitExternalExecutable(admission.name, admission.path, ["--version"]);
  for (const field of ["path", "dev", "ino", "size", "mode", "sha256", "version"]) {
    if (current[field] !== admission[field]) throw new MutationGateError("P07B_EXTERNAL_CHANGED", `${admission.name}: ${field}`);
  }
}

async function ensurePrivateDirectory(path) {
  await mkdir(path, { recursive: true, mode: 0o700 });
  await chmod(path, 0o700);
}

async function prepareExperiment({ mutant, toolchain, phase, seed, phaseRecords, external }) {
  await assertU5SourceAndSeedUnchanged(seed);
  const parent = await mkdtemp(join(await realpath(tmpdir()), `countershape-p07b-${phase}-`));
  await chmod(parent, 0o700);
  const sandbox = join(parent, "repo");
  const cache = join(parent, "cache");
  try {
    await copyRegularAllowlist(seed.root, sandbox, seed.allowlist);
    await assertU5PrivateCopy(sandbox, seed.allowlist);
    const manifest = await assertP07BManifestMatchesTree(sandbox);
    if (!manifestEqual(manifest, seed.manifest)) throw new MutationGateError("P07B_EXPERIMENT_SEED_MISMATCH", `${mutant.id}:${phase}`);
    for (const directory of [cache, "home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"].map((value) => value === cache ? cache : join(cache, value))) {
      await ensurePrivateDirectory(directory);
    }
    await revalidateExternalExecutable(external.node);
    await revalidateExternalExecutable(external.git);
    const environment = Object.freeze({
      ...minimalU2GoEnvironment(cache, toolchain),
	  PATH: [...new Set([
		...minimalU2GoEnvironment(cache, toolchain).PATH.split(":"),
		dirname(external.node.path), dirname(external.git.path),
	  ])].join(":"),
	  COUNTERSHAPE_GO: toolchain.executable.path,
      COUNTERSHAPE_NODE: external.node.path,
      COUNTERSHAPE_GIT: external.git.path,
    });
    return { parent, sandbox, cache, environment, phase, seed, phaseRecords, external };
  } catch (error) {
    await rm(parent, { recursive: true, force: true });
    throw error;
  }
}

async function cleanupExperiment(context) {
  await rm(context.parent, { recursive: true, force: true });
}

async function expectedManifest(context, mutant) {
  if (context.phase === "mutant") return assertExactU5MutantDelta(context.sandbox, context.seed, mutant);
  const manifest = await assertP07BManifestMatchesTree(context.sandbox);
  if (!manifestEqual(manifest, context.seed.manifest)) throw new MutationGateError("P07B_CONTROL_SOURCE_CHANGED", `${mutant.id}:${context.phase}`);
  return manifest;
}

function phaseCommand(context, mutant, toolchain) {
	if (mutant.runner === "architecture") {
		return [context.external.node.path, resolve(context.sandbox, "tools/check-p07b-architecture.mjs")];
	}
	return [
		toolchain.executable.path, "test", "-json", mutant.package,
		"-run", `^${mutant.testName}$`, "-count=1",
	];
}

function classifyArchitectureResult(context, mutant, result) {
	const output = `${result.stdout ?? ""}\n${result.stderr ?? ""}`;
	if (!result.error && !result.signal && result.status === 0 &&
		(result.stdout ?? "").includes("P07B source architecture boundary OK")) {
		return { classification: { outcome: NamedTestOutcome.Pass, detail: "exact P07B architecture checker passed" }, output };
	}
	if (context.phase === "mutant" && !result.error && !result.signal && result.status !== 0 &&
		output.includes(mutant.expectedArchitectureCode)) {
		return { classification: { outcome: NamedTestOutcome.Failure, detail: mutant.expectedArchitectureCode }, output };
	}
	return {
		classification: {
			outcome: NamedTestOutcome.Infrastructure,
			detail: `architecture outcome ${result.status ?? result.signal ?? result.error?.message ?? "unknown"} did not prove ${mutant.expectedArchitectureCode || "clean success"}`,
		},
		output,
	};
}

function runArchitectureTest(context, mutant) {
	const result = spawnSync(
		context.external.node.path,
		[resolve(context.sandbox, "tools/check-p07b-architecture.mjs")],
		{
			cwd: context.sandbox, encoding: "utf8", env: context.environment,
			timeout: 120_000, maxBuffer: 2 * 1024 * 1024,
		},
	);
	return classifyArchitectureResult(context, mutant, result);
}

async function runNamedTest(context, mutant, toolchain) {
  await assertU5SourceAndSeedUnchanged(context.seed);
  const before = await expectedManifest(context, mutant);
	revalidateAdmittedGoExecutable(toolchain.executable);
  await revalidateExternalExecutable(context.external.node);
  await revalidateExternalExecutable(context.external.git);
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
  const result = mutant.runner === "architecture" ? runArchitectureTest(context, mutant) : runNamedU2GoTest({
		sandbox: context.sandbox, mutant, environment: context.environment, toolchain,
	});
  revalidateAdmittedDarwinCCompiler(toolchain.compiler);
	revalidateAdmittedGoExecutable(toolchain.executable);
  await revalidateExternalExecutable(context.external.node);
  await revalidateExternalExecutable(context.external.git);
  const after = await expectedManifest(context, mutant);
  if (!manifestEqual(after, before)) throw new MutationGateError("P07B_TEST_CHANGED_SOURCE", `${mutant.id}:${context.phase}`);
	const command = Object.freeze(phaseCommand(context, mutant, toolchain));
	context.phaseRecords.push(Object.freeze({
		phase: context.phase, manifest: before.digest, sandbox: context.sandbox,
		outcome: result.classification.outcome,
		outcome_detail: result.classification.detail,
		outcome_detail_digest: `sha256:${sha256(Buffer.from(result.classification.detail, "utf8"))}`,
		output: result.output ?? "",
		output_digest: `sha256:${sha256(Buffer.from(result.output ?? "", "utf8"))}`,
		command,
		command_digest: `sha256:${sha256(Buffer.from(JSON.stringify(command), "utf8"))}`,
	}));
  return result;
}

function exactToolFingerprints(toolchain, external) {
	return Object.freeze({
		go: Object.freeze({ path: toolchain.executable.path, sha256: toolchain.executable.sha256, version: toolchain.version.line }),
		cc: Object.freeze({ path: toolchain.compiler.path, sha256: toolchain.compiler.sha256, facts: toolchain.cgo.digest }),
		node: Object.freeze({ path: external.node.path, sha256: external.node.sha256, version: external.node.version }),
		git: Object.freeze({ path: external.git.path, sha256: external.git.sha256, version: external.git.version }),
	});
}

function sameObjectKeys(value, expected) {
	return value !== null && typeof value === "object" && sameArray(Object.keys(value).sort(), [...expected].sort());
}

function validateToolFingerprints(toolFingerprints) {
	if (!Object.isFrozen(toolFingerprints) || !sameObjectKeys(toolFingerprints, ["cc", "git", "go", "node"])) return false;
	for (const name of ["go", "node", "git"]) {
		const entry = toolFingerprints[name];
		if (!Object.isFrozen(entry) || !sameObjectKeys(entry, ["path", "sha256", "version"]) ||
			!isAbsolute(entry.path) || !/^[0-9a-f]{64}$/u.test(entry.sha256) ||
			typeof entry.version !== "string" || entry.version.length === 0 || entry.version.length > 4096) return false;
	}
	const compiler = toolFingerprints.cc;
	return Object.isFrozen(compiler) && sameObjectKeys(compiler, ["facts", "path", "sha256"]) &&
		isAbsolute(compiler.path) && /^[0-9a-f]{64}$/u.test(compiler.sha256) && /^sha256:[0-9a-f]{64}$/u.test(compiler.facts);
}

function expectedPhaseCommand(mutant, record, toolFingerprints) {
	if (mutant.runner === "architecture") {
		return [toolFingerprints.node.path, resolve(record.sandbox, "tools/check-p07b-architecture.mjs")];
	}
	return [
		toolFingerprints.go.path, "test", "-json", mutant.package,
		"-run", `^${mutant.testName}$`, "-count=1",
	];
}

export function exactP07BABADigest(mutant, phaseRecords, seed, toolFingerprints) {
  const reviewed = REVIEWED_P07B_MUTANT_CONTRACT.find((tuple) => tuple.id === mutant?.id);
  if (!Object.isFrozen(mutant) || !reviewed || tupleFields.some((field) => mutant[field] !== reviewed[field])) {
    throw new MutationGateError("P07B_ABA_RECEIPT_INVALID", `${mutant?.id ?? "unknown"} is not one frozen reviewed tuple`);
  }
  const phases = phaseRecords.map((record) => record.phase);
  const manifests = phaseRecords.map((record) => record.manifest);
  const sandboxes = phaseRecords.map((record) => record.sandbox);
  const outcomes = phaseRecords.map((record) => record.outcome);
	if (!seed?.manifest || !seed?.sourceManifest || !/^sha256:[0-9a-f]{64}$/u.test(seed.manifest.digest) ||
		!/^sha256:[0-9a-f]{64}$/u.test(seed.sourceManifest.digest) || !validateToolFingerprints(toolFingerprints) ||
		phaseRecords.length !== 3 || !sameArray(phases, ["baseline", "mutant", "post-control"]) ||
      sandboxes.some((value) => !isAbsolute(value)) || new Set(sandboxes).size !== 3 ||
      manifests[0] !== seed.manifest.digest || manifests[2] !== seed.manifest.digest || manifests[1] === seed.manifest.digest ||
	  !sameArray(outcomes, [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass]) ||
	  phaseRecords.some((record) => {
		const expectedCommand = expectedPhaseCommand(mutant, record, toolFingerprints);
		return !Object.isFrozen(record) || !sameObjectKeys(record, [
			"command", "command_digest", "manifest", "outcome", "outcome_detail", "outcome_detail_digest",
			"output", "output_digest", "phase", "sandbox",
		]) || !Object.isFrozen(record.command) || !sameArray(record.command, expectedCommand) ||
			typeof record.outcome_detail !== "string" || record.outcome_detail.length > 4096 ||
			typeof record.output !== "string" || record.output.length > 2 * 1024 * 1024 ||
			record.command_digest !== `sha256:${sha256(Buffer.from(JSON.stringify(record.command), "utf8"))}` ||
			record.outcome_detail_digest !== `sha256:${sha256(Buffer.from(record.outcome_detail, "utf8"))}` ||
			record.output_digest !== `sha256:${sha256(Buffer.from(record.output, "utf8"))}`;
	  })) {
    throw new MutationGateError("P07B_ABA_RECEIPT_INVALID", `${mutant.id} lacks fresh exact A/B/A closure`);
  }
  const canonical = JSON.stringify({
    schema_version: "countershape.p07b.mutation-receipt/v1",
    mutant_tuple: Object.fromEntries(tupleFields.map((field) => [field, mutant[field]])),
    phases: phaseRecords,
    seed_manifest: seed.manifest.digest,
	source_manifest: seed.sourceManifest.digest,
	tool_fingerprints: toolFingerprints,
  });
  return `sha256:${sha256(Buffer.from(canonical, "utf8"))}`;
}

function fakeClassification(outcome) {
  return { classification: { outcome, detail: outcome }, output: "" };
}

async function runRequiredArchitectureCheck(relativeScript, successMarker, toolchain, external) {
	revalidateAdmittedGoExecutable(toolchain.executable);
	revalidateAdmittedDarwinCCompiler(toolchain.compiler);
	await revalidateExternalExecutable(external.node);
	await revalidateExternalExecutable(external.git);
	const result = spawnSync(external.node.path, [resolve(repoRoot, relativeScript)], {
		cwd: repoRoot, encoding: "utf8", timeout: 120_000, maxBuffer: 2 * 1024 * 1024,
		env: {
			...process.env, NO_COLOR: "1", LANG: "C", LC_ALL: "C", TZ: "UTC",
			COUNTERSHAPE_GO: toolchain.executable.path,
			COUNTERSHAPE_NODE: external.node.path, COUNTERSHAPE_GIT: external.git.path,
			PATH: [...new Set([
				dirname(toolchain.executable.path), dirname(toolchain.compiler.path),
				dirname(external.node.path), dirname(external.git.path),
			])].join(":"),
		},
	});
	revalidateAdmittedGoExecutable(toolchain.executable);
	revalidateAdmittedDarwinCCompiler(toolchain.compiler);
	await revalidateExternalExecutable(external.node);
	await revalidateExternalExecutable(external.git);
	if (result.error || result.signal || result.status !== 0 || !(result.stdout ?? "").includes(successMarker)) {
		throw new MutationGateError(
			"P07B_ARCHITECTURE_PREFLIGHT_FAILED",
			`${relativeScript}: ${result.status ?? result.signal ?? result.error?.message}: ${result.stderr || result.stdout}`,
		);
	}
}

export async function selfTest() {
  assertP07BMutantDefinitionSet();
  await assertP07BManifestMatchesTree(repoRoot);
  await assertReviewedP07BAnchors();
  await assertReviewedP07BNamedTests();
  assert.throws(
    () => assertP07BMutantDefinitionSet(Object.freeze(P07B_MUTANTS.slice(0, -1))),
    (error) => error instanceof MutationGateError,
  );
	assert.throws(
		() => assertP07BMutantDefinitionSet(Object.freeze([P07B_MUTANTS[1], P07B_MUTANTS[0], ...P07B_MUTANTS.slice(2)])),
		(error) => error instanceof MutationGateError,
	);
	assert.throws(
		() => assertP07BMutantDefinitionSet(Object.freeze([...P07B_MUTANTS.slice(0, -1), P07B_MUTANTS.at(-2)])),
		(error) => error instanceof MutationGateError,
	);
	assert.throws(
		() => assertP07BMutantDefinitionSet(P07B_MUTANTS.map((mutant) => ({ ...mutant }))),
		(error) => error instanceof MutationGateError && error.code === "P07B_MUTANT_SET_NOT_IMMUTABLE",
	);
  const tampered = P07B_MUTANTS.map((mutant) => Object.freeze({ ...mutant }));
  tampered[0] = Object.freeze({ ...tampered[0], replace: `${tampered[0].replace}\n// weakened` });
  assert.throws(
    () => assertP07BMutantDefinitionSet(Object.freeze(tampered)),
    (error) => error instanceof MutationGateError && error.code === "P07B_MUTANT_CONTRACT_MISMATCH",
  );
  let invocation = 0;
  const phases = [];
  const killed = await runU2Mutant(P07B_MUTANTS[0], {}, {
    async prepareExperiment({ phase }) {
      phases.push(phase);
      return { sandbox: `/fresh/p07b/${phase}`, parent: `/fresh/p07b/${phase}` };
    },
    async cleanupExperiment() {},
    async applyMutation() {},
    async runNamedTest() {
      return fakeClassification([NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass][invocation++]);
    },
  });
  assert.match(killed, /^KILLED cli-accept-invented-projection-schema/u);
  assert.deepEqual(phases, ["baseline", "mutant", "post-control"]);
  const seed = {
	manifest: { digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" },
	sourceManifest: { digest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" },
  };
  const outcomes = [NamedTestOutcome.Pass, NamedTestOutcome.Failure, NamedTestOutcome.Pass];
	const toolFingerprints = Object.freeze({
		go: Object.freeze({ path: "/trusted/go", sha256: "a".repeat(64), version: "go1.test" }),
		cc: Object.freeze({ path: "/trusted/cc", sha256: "b".repeat(64), facts: `sha256:${"c".repeat(64)}` }),
		node: Object.freeze({ path: "/trusted/node", sha256: "d".repeat(64), version: "v1.test" }),
		git: Object.freeze({ path: "/trusted/git", sha256: "e".repeat(64), version: "git version test" }),
	});
	const syntheticRecords = (mutant) => ["baseline", "mutant", "post-control"].map((phase, index) => {
		const sandbox = `/fresh/p07b/${String.fromCharCode(97 + index)}`;
		const command = Object.freeze(mutant.runner === "architecture" ?
			[toolFingerprints.node.path, resolve(sandbox, "tools/check-p07b-architecture.mjs")] :
			[toolFingerprints.go.path, "test", "-json", mutant.package, "-run", `^${mutant.testName}$`, "-count=1"]);
		const outcomeDetail = outcomes[index];
		const output = `synthetic-${phase}`;
		return Object.freeze({
			phase, manifest: index === 1 ? "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" : seed.manifest.digest,
			sandbox, outcome: outcomes[index], outcome_detail: outcomeDetail,
			outcome_detail_digest: `sha256:${sha256(Buffer.from(outcomeDetail, "utf8"))}`,
			output, output_digest: `sha256:${sha256(Buffer.from(output, "utf8"))}`,
			command, command_digest: `sha256:${sha256(Buffer.from(JSON.stringify(command), "utf8"))}`,
		});
	});
	const records = syntheticRecords(P07B_MUTANTS[0]);
  assert.match(exactP07BABADigest(P07B_MUTANTS[0], records, seed, toolFingerprints), /^sha256:[0-9a-f]{64}$/u);
	const architectureMutant = P07B_MUTANTS.find((mutant) => mutant.runner === "architecture");
	assert.match(exactP07BABADigest(architectureMutant, syntheticRecords(architectureMutant), seed, toolFingerprints), /^sha256:[0-9a-f]{64}$/u);
	const badCommandDigest = Object.freeze([
		Object.freeze({ ...records[0], command_digest: `sha256:${"0".repeat(64)}` }), records[1], records[2],
	]);
	assert.throws(
		() => exactP07BABADigest(P07B_MUTANTS[0], badCommandDigest, seed, toolFingerprints),
		(error) => error instanceof MutationGateError && error.code === "P07B_ABA_RECEIPT_INVALID",
	);
	const badTools = Object.freeze({ ...toolFingerprints, node: Object.freeze({ ...toolFingerprints.node, sha256: "not-a-digest" }) });
	assert.throws(
		() => exactP07BABADigest(P07B_MUTANTS[0], records, seed, badTools),
		(error) => error instanceof MutationGateError && error.code === "P07B_ABA_RECEIPT_INVALID",
	);
  return `P07B mutation self-test passed: ${P07B_MUTANTS.length}/${P07B_MUTANTS.length} frozen tuples, exact anchors/tests/manifests, and synthetic fresh A/B/A closure.`;
}

export async function main(arguments_ = process.argv.slice(2)) {
  if (arguments_.length === 1 && arguments_[0] === "--self-test") {
    process.stdout.write(`${await selfTest()}\n`);
    return;
  }
  if (arguments_.length !== 0) throw new MutationGateError("P07B_UNSUPPORTED_ARGUMENT", "usage: node tools/mutate-p07b.mjs [--self-test]");

  await selfTest();
  assertP07BMutantDefinitionSet();
  await assertP07BManifestMatchesTree(repoRoot);
  await assertReviewedP07BAnchors();
  await assertReviewedP07BNamedTests();
  const toolchain = await admitU2DarwinToolchain();
  const external = Object.freeze({
    node: await admitExternalExecutable("node", process.env.COUNTERSHAPE_NODE ?? "/opt/homebrew/bin/node", ["--version"]),
    git: await admitExternalExecutable("git", process.env.COUNTERSHAPE_GIT ?? "/usr/bin/git", ["--version"]),
  });
	const toolFingerprints = exactToolFingerprints(toolchain, external);
	await runRequiredArchitectureCheck("tools/check-p07b-architecture.mjs", "P07B source architecture boundary OK", toolchain, external);
	await runRequiredArchitectureCheck("tools/check-p07b-architecture-selftest.mjs", "P07B architecture self-test OK", toolchain, external);
  const seed = await createU5SeedSnapshot({
    sourceRoot: repoRoot,
    allowlist: P07B_SANDBOX_FILE_ALLOWLIST,
    scopeRoots: P07B_SOURCE_SCOPE_ROOTS,
    reviewedNonGoFiles: P07B_REVIEWED_NON_GO_FILES,
    embedBindings: P07B_REVIEWED_EMBED_BINDINGS,
      runtimeSupportFiles: P07B_RUNTIME_SUPPORT_FILES,
  });
  const results = [];
  try {
    for (const mutant of P07B_MUTANTS) {
      const phaseRecords = [];
      const result = await runU2Mutant(mutant, toolchain, {
        prepareExperiment({ mutant: current, toolchain: admitted, phase }) {
          return prepareExperiment({ mutant: current, toolchain: admitted, phase, seed, phaseRecords, external });
        },
        cleanupExperiment,
        applyMutation: mutateRegularFileNoFollow,
        runNamedTest(context, current) {
          return runNamedTest(context, current, toolchain);
        },
      });
      results.push(Object.freeze({ result, receipt: exactP07BABADigest(mutant, phaseRecords, seed, toolFingerprints) }));
    }
    await assertU5SourceAndSeedUnchanged(seed);
  } finally {
    await cleanupU5SeedSnapshot(seed);
  }
  await revalidateExternalExecutable(external.node);
  await revalidateExternalExecutable(external.git);
	await runRequiredArchitectureCheck("tools/check-p07b-architecture.mjs", "P07B source architecture boundary OK", toolchain, external);
	await runRequiredArchitectureCheck("tools/check-p07b-architecture-selftest.mjs", "P07B architecture self-test OK", toolchain, external);

  process.stdout.write(`Admitted Node path fingerprint: ${external.node.path} sha256:${external.node.sha256} ${external.node.version}\n`);
  process.stdout.write(`Admitted Git path fingerprint: ${external.git.path} sha256:${external.git.sha256} ${external.git.version}\n`);
  process.stdout.write(`Immutable P07B seed manifest: ${seed.manifest.digest}\n`);
  process.stdout.write("Containment notice: private mutation copies retain the invoking user's host filesystem and network authority.\n");
  for (const item of results) process.stdout.write(`${item.result}; exact fresh A/B/A closure digest ${item.receipt}\n`);
  process.stdout.write(`P07B mutation gate: ${results.length}/${P07B_MUTANTS.length} required mutants killed; ${results.length}/${P07B_MUTANTS.length} fresh A/B/A receipts.\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(modulePath)) {
  main().catch((error) => {
    process.stderr.write(`${error.stack ?? error}\n`);
    process.exitCode = 1;
  });
}
