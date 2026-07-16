#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { cp, chmod, mkdtemp, mkdir, readFile, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const sourceRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const copyPaths = Object.freeze([
  "internal/domain",
  "internal/compare",
  "internal/observe",
  "internal/reduce",
  "internal/reduction",
  "internal/store",
  "internal/confirmation",
  "internal/choice",
  "internal/world",
  "spec/schema/v1/common.schema.json",
  "spec/schema/v1/choicepoint.schema.json",
  "spec/schema/v1/decision-record.schema.json",
  "spec/examples/v1/choicepoint.valid.json",
  "spec/examples/v1/decision-record.valid.json",
  "tools/check-u5-architecture.mjs",
  "tools/check-u6-architecture.mjs",
]);

// The case roster is intentionally independent from the switch below and
// checksum-pinned. Removing a hostile case therefore requires an explicit,
// review-visible update to both the roster and its frozen digest.
const requiredCaseIDs = Object.freeze([
  "clean",
  "arguments",
  "harmless-comment",
  "harmless-astral-comment",
  "comment-camouflage",
  "raw-head-export",
  "raw-head-method-value-export",
  "raw-head-wrapper-export",
  "refine-after-issue",
  "cas-current-token-forged",
  "cas-field-valid-substitution",
  "cas-dead-return-decoy",
  "illegal-transition",
  "transition-dead-return-decoy",
  "immutable-destination-overwrite",
  "confirmation-ordinal-swap",
  "confirmation-offset-zero",
  "confirmation-nonce-detached",
  "blind-alias-argument",
  "blind-support-order",
  "blind-comparator-dead-return-decoy",
  "reject-all-compilable",
  "forged-compile-eligibility-file",
  "embedded-compile-eligibility-file",
  "intermediate-compile-eligibility-interface-file",
  "receipt-grade-normalized",
  "receipt-grade-pattern",
  "schema-opened",
  "schema-runtime-drift",
  "duplicate-json-member",
  "public-issuer-variable",
  "public-authority-alias-rhs",
  "renamed-issuer-import",
  "dot-issuer-import",
  "alternate-issuer-file",
  "forged-draft-file",
  "forged-draft-new-file",
  "forged-promotion-capability-file",
  "wire-to-draft",
  "publication-predecessor-detached",
  "publication-dead-return-decoy",
  "ruling-action-detached",
  "unknown-source-member",
  "package-mismatch",
  "symlinked-go-authority",
  "symlinked-json-authority",
  "invalid-utf8-go",
  "manifest-barrier-tamper",
  "manifest-tool-failure",
]);
const requiredRosterDigest = "2eb647f2febf5d00918b8cc1ee2e22a006c0f6485fce4ce9ee2860c762ea8596";

async function copyFixture() {
  const fixture = await mkdtemp(join(tmpdir(), "countershape-u6-architecture-"));
  await chmod(fixture, 0o700);
  for (const path of copyPaths) {
    const destination = join(fixture, path);
    await mkdir(dirname(destination), { recursive: true, mode: 0o700 });
    await cp(join(sourceRoot, path), destination, { recursive: true, errorOnExist: true, dereference: false });
  }
  return fixture;
}

async function replaceExact(fixture, path, before, after) {
  const absolute = join(fixture, path);
  const source = await readFile(absolute, "utf8");
  const first = source.indexOf(before);
  if (first < 0 || source.indexOf(before, first + before.length) >= 0) {
    throw new Error(`self-test mutation anchor is not unique: ${path}: ${before}`);
  }
  await writeFile(absolute, source.replace(before, after), { mode: 0o600 });
}

async function appendSource(fixture, path, source) {
  const absolute = join(fixture, path);
  const current = await readFile(absolute, "utf8");
  await writeFile(absolute, `${current}${source}`, { mode: 0o600 });
}

async function writeSource(fixture, path, source) {
  const absolute = join(fixture, path);
  await mkdir(dirname(absolute), { recursive: true, mode: 0o700 });
  await writeFile(absolute, source, { mode: 0o600 });
}

function runChecker(fixture, args = [], environment = {}) {
  return spawnSync(process.execPath, [join(fixture, "tools/check-u6-architecture.mjs"), ...args], {
    cwd: fixture,
    encoding: "utf8",
    env: { NO_COLOR: "1", TZ: "UTC", ...environment },
    timeout: 30_000,
    maxBuffer: 8 * 1024 * 1024,
  });
}

function assertChildCompleted(name, result) {
  if (result.error || result.signal || result.status === null) {
    throw new Error(
      `${name} checker process failed: status=${result.status} signal=${result.signal} error=${result.error?.message ?? ""}\n` +
      `${result.stdout ?? ""}${result.stderr ?? ""}`,
    );
  }
}

async function exercise(name) {
  const fixture = await copyFixture();
  try {
    let args = [];
    let environment = {};
    let expected = null;
    switch (name) {
      case "clean":
        break;
      case "arguments":
        args = ["--root", fixture];
        expected = "U6_ARCHITECTURE_ARGUMENTS";
        break;
      case "harmless-comment":
        await replaceExact(
          fixture,
          "internal/choice/blind.go",
          "package choice",
          "package choice\n\n// AdvanceHead WakeEvent DisplayGroups() GeneralizationClaimed: true",
        );
        break;
      case "harmless-astral-comment":
        await replaceExact(
          fixture,
          "internal/choice/blind.go",
          "package choice",
          "package choice\n\n// Astral lexical alignment control: 🧪",
        );
        break;
      case "comment-camouflage":
        await replaceExact(
          fixture,
          "internal/store/head.go",
          "if !sameHeadToken(currentToken, expected) {",
          "if true {\n\t\t// if !sameHeadToken(currentToken, expected) {",
        );
        expected = "U6_FULL_CAS_COMPARE_REQUIRED";
        break;
      case "raw-head-export":
        await replaceExact(fixture, "internal/store/head.go", "func (s *ObjectStore) advanceHead(", "func (s *ObjectStore) AdvanceHead(");
        expected = "U6_RAW_HEAD_API_EXPORTED";
        break;
      case "raw-head-method-value-export":
        await appendSource(fixture, "internal/store/head.go", "\nvar AdvanceAny = (*ObjectStore).advanceHead\n");
        expected = "U6_RAW_HEAD_REFERENCE_SURFACE_NOT_EXACT";
        break;
      case "raw-head-wrapper-export":
        await appendSource(
          fixture,
          "internal/store/head.go",
          "\nfunc (s *ObjectStore) AdvanceAny(ctx context.Context, expected HeadToken, next LineageStage, object SemanticObject) (HeadToken, error) {\n\treturn s.advanceHead(ctx, expected, next, object)\n}\n",
        );
        expected = "U6_RAW_HEAD_REFERENCE_SURFACE_NOT_EXACT";
        break;
      case "refine-after-issue":
        await replaceExact(
          fixture,
          "internal/choice/promotion/service.go",
          "if decision.Action() == choice.ActionRefine {",
          "if false && decision.Action() == choice.ActionRefine {",
        );
        expected = "U6_REFINE_GUARD_ORDER";
        break;
      case "cas-current-token-forged":
        await replaceExact(fixture, "internal/store/head.go", "currentToken := s.token(expected.study, current)", "currentToken := expected");
        expected = "U6_CAS_CURRENT_TOKEN_DERIVATION";
        break;
      case "cas-field-valid-substitution":
        await replaceExact(
          fixture,
          "internal/store/head.go",
          "left.headDigest == right.headDigest",
          "left.headDigest.Valid() == right.headDigest.Valid()",
        );
        expected = "U6_CAS_TOKEN_EXACT_EQUALITY_MISSING";
        break;
      case "cas-dead-return-decoy":
        await replaceExact(
          fixture,
          "internal/store/head.go",
          "func sameHeadToken(left, right HeadToken) bool {\n\treturn",
          "func sameHeadToken(left, right HeadToken) bool {\n\treturn true\n\treturn",
        );
        expected = "U6_CAS_TOKEN_BODY_NOT_EXACT";
        break;
      case "illegal-transition":
        await replaceExact(
          fixture,
          "internal/store/head.go",
          "return next == StageBaseline",
          "return next == StageBaseline || next == StageDivergence",
        );
        expected = "U6_TRANSITION_GRAPH_NOT_EXACT";
        break;
      case "transition-dead-return-decoy":
        await replaceExact(
          fixture,
          "internal/store/head.go",
          "func permittedTransition(current, next LineageStage) bool {\n\tswitch current {",
          "func permittedTransition(current, next LineageStage) bool {\n\treturn true\n\tswitch current {",
        );
        expected = "U6_TRANSITION_BODY_NOT_EXACT";
        break;
      case "immutable-destination-overwrite":
        await replaceExact(
          fixture,
          "internal/store/object_store.go",
          "temporary, err := writeSyncedTemporary(shard, objectTempPrefix, object.canonical)",
          "_ = os.Remove(destination)\n\ttemporary, err := writeSyncedTemporary(shard, objectTempPrefix, object.canonical)",
        );
        expected = "U6_IMMUTABLE_DESTINATION_MUTATION";
        break;
      case "confirmation-ordinal-swap":
        await replaceExact(
          fixture,
          "internal/confirmation/wire.go",
          "candidate != scheduledTrials[index].CandidateKey()",
          "candidate == \"\"",
        );
        expected = "U6_CONFIRMATION_ORDINAL_CANDIDATE_BINDING";
        break;
      case "confirmation-offset-zero":
        await replaceExact(fixture, "internal/observe/schedule.go", "startOffset = 1 % len(canonicalRoster)", "startOffset = 0");
        expected = "U6_CONFIRMATION_OFFSET_NOT_EXACT";
        break;
      case "confirmation-nonce-detached":
        await replaceExact(fixture, "internal/confirmation/service.go", "challenge.String(),", "\"detached\",");
        expected = "U6_CONFIRMATION_NONCE_BINDING";
        break;
      case "blind-alias-argument":
        await replaceExact(
          fixture,
          "internal/choice/blind.go",
          "choicepoint, outcome.ref.fingerprint.String(),",
          "choicepoint, outcome.ref.candidate.String(),",
        );
        expected = "U6_BLIND_ALIAS_NOT_FINGERPRINT_BOUND";
        break;
      case "blind-support-order":
        await replaceExact(
          fixture,
          "internal/choice/blind.go",
          "if groups[i].orderKey != groups[j].orderKey {\n\t\t\treturn groups[i].orderKey < groups[j].orderKey\n\t\t}",
          "if len(groups[i].refs) != len(groups[j].refs) {\n\t\t\treturn len(groups[i].refs) < len(groups[j].refs)\n\t\t}",
        );
        expected = "U6_BLIND_COMPARATOR_READS_HIDDEN_AUTHORITY";
        break;
      case "blind-comparator-dead-return-decoy":
        await replaceExact(
          fixture,
          "internal/choice/blind.go",
          "sort.Slice(groups, func(i, j int) bool { // MUTANT_U6_BLIND_ORDER_BY_SUPPORT\n\t\tif groups[i].orderKey",
          "sort.Slice(groups, func(i, j int) bool { // MUTANT_U6_BLIND_ORDER_BY_SUPPORT\n\t\tif true {\n\t\t\treturn supportLess(groups[i], groups[j])\n\t\t}\n\t\tif groups[i].orderKey",
        );
        await appendSource(
          fixture,
          "internal/choice/blind.go",
          "\nfunc supportLess(left, right blindGroup) bool { return len(left.refs) < len(right.refs) }\n",
        );
        expected = "U6_BLIND_COMPARATOR_BODY_NOT_EXACT";
        break;
      case "reject-all-compilable":
        await replaceExact(
          fixture,
          "internal/choice/validation.go",
          "eligibility: noncompilable",
          "eligibility: compilableRuling{action: input.Action}",
        );
        expected = "U6_COMPILE_ELIGIBILITY_CONSTRUCTION_NOT_CLOSED";
        break;
      case "forged-compile-eligibility-file":
        await writeSource(
          fixture,
          "internal/choice/forged_eligibility.go",
          "package choice\n\ntype forgedEligibility struct{}\n\nfunc (forgedEligibility) isCompileEligibility() {}\nfunc (forgedEligibility) Action() Action { return ActionRejectAll }\nfunc ForgeCompileEligibility() CompileEligibility { return forgedEligibility{} }\n",
        );
        expected = "U6_COMPILE_ELIGIBILITY_AUTHORITY_OUTSIDE_OWNER";
        break;
      case "embedded-compile-eligibility-file":
        await writeSource(
          fixture,
          "internal/choice/embedded_eligibility.go",
          "package choice\n\ntype embeddedEligibility struct { CompileEligibility }\n\nfunc (embeddedEligibility) Action() Action { return ActionRejectAll }\nfunc ForgeEmbeddedEligibility(value CompileEligibility) CompileEligibility { return embeddedEligibility{CompileEligibility: value} }\n",
        );
        expected = "U6_COMPILE_ELIGIBILITY_EMBEDDING_OUTSIDE_OWNER";
        break;
      case "intermediate-compile-eligibility-interface-file":
        await writeSource(
          fixture,
          "internal/choice/intermediate_eligibility.go",
          "package choice\n\ntype embeddedAuthority interface { CompileEligibility }\ntype forgedEligibility struct { embeddedAuthority }\n\nfunc (forgedEligibility) Action() Action { return ActionRejectAll }\nfunc ForgeIntermediateEligibility(value CompileEligibility) CompileEligibility {\n\treturn forgedEligibility{embeddedAuthority: value}\n}\n",
        );
        expected = "U6_COMPILE_ELIGIBILITY_EMBEDDING_OUTSIDE_OWNER";
        break;
      case "receipt-grade-normalized":
        await replaceExact(
          fixture,
          "internal/domain/receipt.go",
          "NewDidrunReceipt(wire.GradeVerbatim, wire.CommitOID, digest)",
          "NewDidrunReceipt(\"PASS\", wire.CommitOID, digest)",
        );
        expected = "U6_RECEIPT_GRADE_NORMALIZED";
        break;
      case "receipt-grade-pattern": {
        const path = join(fixture, "spec/schema/v1/common.schema.json");
        const value = JSON.parse(await readFile(path, "utf8"));
        value.$defs.ReceiptReference.properties.grade_verbatim.pattern = ".+";
        await writeFile(path, `${JSON.stringify(value)}\n`, { mode: 0o600 });
        expected = "U6_RECEIPT_GRADE_NOT_VERBATIM";
        break;
      }
      case "schema-opened": {
        const path = join(fixture, "spec/schema/v1/choicepoint.schema.json");
        const value = JSON.parse(await readFile(path, "utf8"));
        value.additionalProperties = true;
        await writeFile(path, `${JSON.stringify(value)}\n`, { mode: 0o600 });
        expected = "U6_SCHEMA_OPEN_OBJECT";
        break;
      }
      case "schema-runtime-drift": {
        const path = join(fixture, "spec/examples/v1/decision-record.valid.json");
        const value = JSON.parse(await readFile(path, "utf8"));
        value.artifact_digest = `sha256:${"0".repeat(64)}`;
        await writeFile(path, `${JSON.stringify(value)}\n`, { mode: 0o600 });
        expected = "U6_SCHEMA_RUNTIME_PARITY";
        break;
      }
      case "duplicate-json-member": {
        const path = join(fixture, "spec/examples/v1/choicepoint.valid.json");
        const source = await readFile(path, "utf8");
        const key = /^\{"([^"]+)":/u.exec(source)?.[1];
        if (!key) throw new Error("duplicate JSON mutation could not find first key");
        await writeFile(path, source.replace("{", `{${JSON.stringify(key)}:null,`), { mode: 0o600 });
        expected = "U6_DUPLICATE_JSON_MEMBER";
        break;
      }
      case "public-issuer-variable":
        await replaceExact(
          fixture,
          "internal/confirmation/authority/authority.go",
          "type Publication = publication.Authority",
          "type Publication = publication.Authority\n\nvar Issue = publication.Issue",
        );
        expected = "U6_PUBLIC_AUTHORITY_ALIAS_SURFACE_NOT_EXACT";
        break;
      case "public-authority-alias-rhs":
        await writeSource(
          fixture,
          "internal/confirmation/authority/authority.go",
          "package authority\n\nimport (\n\t\"github.com/nelsonwerd/countershape/internal/domain\"\n\t\"github.com/nelsonwerd/countershape/internal/confirmation/internal/publication\"\n)\n\ntype (\n\tPublication = interface {\n\t\tValid() bool\n\t\tDigest() domain.Digest\n\t\tPredecessor() domain.Digest\n\t\tCanonicalBytes() []byte\n\t}\n)\n\nfunc decoy() {\n\ttype Publication = publication.Authority\n\tvar _ Publication\n}\n",
        );
        expected = "U6_PUBLIC_AUTHORITY_ALIAS_DECLARATION_NOT_EXACT";
        break;
      case "renamed-issuer-import":
        await replaceExact(
          fixture,
          "internal/confirmation/service.go",
          "confirmationpublication \"github.com/nelsonwerd/countershape/internal/confirmation/internal/publication\"",
          "pub \"github.com/nelsonwerd/countershape/internal/confirmation/internal/publication\"",
        );
        await replaceExact(fixture, "internal/confirmation/service.go", "confirmationpublication.Issue(", "pub.Issue(");
        expected = "U6_CONFIRMATION_ISSUANCE_SURFACE_NOT_EXACT";
        break;
      case "dot-issuer-import":
        await replaceExact(
          fixture,
          "internal/confirmation/service.go",
          "confirmationpublication \"github.com/nelsonwerd/countershape/internal/confirmation/internal/publication\"",
          ". \"github.com/nelsonwerd/countershape/internal/confirmation/internal/publication\"",
        );
        await replaceExact(fixture, "internal/confirmation/service.go", "confirmationpublication.Issue(", "Issue(");
        expected = "U6_DOT_IMPORT_FORBIDDEN";
        break;
      case "alternate-issuer-file":
        await writeSource(
          fixture,
          "internal/confirmation/alternate_issuer.go",
          "package confirmation\n\nimport p \"github.com/nelsonwerd/countershape/internal/confirmation/internal/publication\"\n\nvar hiddenIssuer = p.Issue\n",
        );
        expected = "U6_PUBLICATION_IMPORT_OUTSIDE_OWNER";
        break;
      case "forged-draft-file":
        await writeSource(fixture, "internal/confirmation/forged_draft.go", "package confirmation\n\nfunc forgedDraft() Draft { return Draft{} }\n");
        expected = "U6_CONFIRMATION_LIVE_AUTHORITY_CONSTRUCTION_OUTSIDE_OWNER";
        break;
      case "forged-draft-new-file":
        await writeSource(fixture, "internal/confirmation/forged_draft.go", "package confirmation\n\nfunc forgedDraft() *Draft { return new(Draft) }\n");
        expected = "U6_CONFIRMATION_LIVE_AUTHORITY_CONSTRUCTION_OUTSIDE_OWNER";
        break;
      case "forged-promotion-capability-file":
        await writeSource(fixture, "internal/choice/promotion/forged_ready.go", "package promotion\n\nfunc forgedReady() *Ready { return new(Ready) }\n");
        expected = "U6_PROMOTION_LIVE_CAPABILITY_CONSTRUCTION_OUTSIDE_OWNER";
        break;
      case "wire-to-draft":
        await appendSource(fixture, "internal/confirmation/wire.go", "\nfunc forgedWireDraft() Draft { return Draft{} }\n");
        expected = "U6_CONFIRMATION_WIRE_RECREATES_LIVE_AUTHORITY";
        break;
      case "publication-predecessor-detached":
        await replaceExact(
          fixture,
          "internal/confirmation/internal/publication/authority.go",
          "parsedPredecessor != predecessor",
          "false",
        );
        expected = "U6_CONFIRMATION_PUBLICATION_PREDECESSOR_BINDING";
        break;
      case "publication-dead-return-decoy":
        await replaceExact(
          fixture,
          "internal/confirmation/internal/publication/authority.go",
          "func validateBody(digest, predecessor domain.Digest, canonical []byte) error {\n\tvalue, err := canon.Parse(canonical)",
          "func validateBody(digest, predecessor domain.Digest, canonical []byte) error {\n\treturn nil\n\tvalue, err := canon.Parse(canonical)",
        );
        expected = "U6_PUBLICATION_VALIDATOR_SUCCESS_PATH_NOT_EXACT";
        break;
      case "ruling-action-detached":
        await replaceExact(
          fixture,
          "internal/choice/promotion/internal/publication/authority.go",
          "bodyAction != action",
          "false",
        );
        expected = "U6_RULING_PUBLICATION_ACTION_BINDING";
        break;
      case "unknown-source-member":
        await writeSource(fixture, "internal/choice/unreviewed.txt", "authority\n");
        expected = "U6_MANIFEST_UNKNOWN_EXTENSION";
        break;
      case "package-mismatch":
        await replaceExact(fixture, "internal/choice/blind.go", "package choice", "package store");
        expected = "U6_MANIFEST_PACKAGE_MISMATCH";
        break;
      case "symlinked-go-authority": {
        const path = join(fixture, "internal/choice/blind.go");
        const target = join(fixture, "blind-target.go");
        await cp(path, target);
        await rm(path);
        await symlink(target, path);
        expected = "U6_MANIFEST_SYMLINK";
        break;
      }
      case "symlinked-json-authority": {
        const path = join(fixture, "spec/examples/v1/choicepoint.valid.json");
        const target = join(fixture, "choicepoint-target.json");
        await cp(path, target);
        await rm(path);
        await symlink(target, path);
        expected = "U6_MANIFEST_JSON_NONREGULAR";
        break;
      }
      case "invalid-utf8-go":
        await writeFile(join(fixture, "internal/choice/blind.go"), Uint8Array.of(0xff), { mode: 0o600 });
        expected = "U6_MANIFEST_INVALID_UTF8";
        break;
      case "manifest-barrier-tamper": {
        const path = "tools/check-u6-architecture.mjs";
        const barrier = "  // U6_SELFTEST_MANIFEST_BARRIER";
        const injection = "  await (await import(\"node:fs/promises\")).appendFile(resolve(root, \"internal/domain/envelope.go\"), \"\\n// concurrent manifest tamper\\n\");";
        await replaceExact(fixture, path, barrier, `${injection}\n${barrier}`);
        expected = "U6_MANIFEST_CHANGED_DURING_SCAN";
        break;
      }
      case "manifest-tool-failure":
        environment = { COUNTERSHAPE_U6_ARCHITECTURE_FORCE_TOOL_FAILURE: "1" };
        expected = "U6_MANIFEST_TOOL_FAILED";
        break;
      default:
        throw new Error(`unknown required self-test case ${name}`);
    }

    const result = runChecker(fixture, args, environment);
    assertChildCompleted(name, result);
    if (expected === null) {
      if (result.status !== 0 || !result.stdout.includes("U6 architecture boundary OK")) {
        throw new Error(`${name} clean fixture failed\n${result.stdout}\n${result.stderr}`);
      }
    } else if (result.status === 0 || !result.stderr.includes(expected)) {
      throw new Error(`${name} did not fail closed with ${expected}\n${result.stdout}\n${result.stderr}`);
    }
    return name;
  } finally {
    await rm(fixture, { recursive: true, force: true });
  }
}

async function main() {
  if (process.argv.length !== 2) throw new Error("U6 architecture self-test accepts no arguments");
  const rosterDigest = createHash("sha256").update(requiredCaseIDs.join("\n")).digest("hex");
  if (rosterDigest !== requiredRosterDigest) {
    throw new Error(`U6 architecture self-test roster digest mismatch: ${rosterDigest}`);
  }
  const observed = [];
  for (const name of requiredCaseIDs) observed.push(await exercise(name));
  if (observed.join("\n") !== requiredCaseIDs.join("\n")) throw new Error("self-test case roster changed during execution");
  const hostile = observed.length - 2;
  process.stdout.write(`U6 architecture checker self-test OK (2 clean/comment controls; ${hostile}/${hostile} hostile cases)\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
