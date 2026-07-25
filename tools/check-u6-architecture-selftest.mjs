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
	"internal/portablevalue",
	"internal/projectionprofile",
	"internal/projectiontranslate",
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
	"c3-store-bridge",
	"c3-store-bridge-raw-import",
	"c3-store-bridge-renamed-identifiers",
	"c3-store-bridge-wrong-import-alias",
	"c3-store-model-import-without-bridge",
	"c3-store-bridge-signature-drift",
	"c3-store-bridge-field-type-drift",
	"c3-store-bridge-field-tag",
	"c3-store-bridge-swapped-field-order",
	"c3-store-bridge-hidden-field",
	"c3-store-bridge-exported-fifth-field",
	"c3-store-bridge-extra-parameter",
	"c3-store-bridge-result-drift",
	"c3-store-model-import-foreign-file",
	"c3-store-model-import-raw-foreign-file",
	"c3-store-model-import-escaped-foreign-file",
	"c3-store-import-camouflage-raw-string",
	"c3-store-bridge-local-shadow",
	"c3-store-bridge-type-alias-foreign-owner",
	"c3-store-bridge-duplicate-top-level-type",
	"c3-store-bridge-duplicate-method",
	"c4-store-bridge",
	"c4-store-bridge-renamed-identifiers",
	"c4-store-bridge-wrong-model-alias",
	"c4-store-bridge-wrong-hostepoch-alias",
	"c4-store-bridge-exported-embedded-field",
	"c4-store-bridge-foreign-type-owner",
	"c4-store-bridge-receiver-kind-drift",
	"c4-store-bridge-foreign-method-owner",
	"c4-store-hostepoch-import-foreign-file",
	"c4-world-mechanics-wrong-alias",
	"c4-world-mechanics-foreign-file",
	"c4-world-mechanics-missing-authorized-file",
	"eligibility-core-capability-import",
	"eligibility-core-api-opening",
	"eligibility-owner-bypass",
	"portablevalue-full-adapter-import",
	"world-full-adapter-import",
	"second-profile-derivation-reference",
	"profile-constructor-reexport",
	"portable-profile-roster-firewall-bypass",
	"portable-profile-roster-version-drift",
	"portable-expectation-roster-firewall-bypass",
	"portable-mode-semantics-firewall-bypass",
	"expectation-domain-exported-field",
	"portable-registry-expectation-binding-bypass",
	"expectation-domain-propagation-bypass",
	"selected-expectation-validation-bypass",
	"complete-expectation-validation-bypass",
	"universe-expectation-seal-bypass",
	"expectation-domain-validity-bypass",
	"expectation-domain-validity-dead-return",
	"expectation-domain-validation-dead-return",
	"confirmed-expectation-authority-detached",
	"choice-expectation-validator-bypass",
	"choice-expectation-validator-dead-return",
  "choice-adapter-domain-switch",
  "caller-projection-builder",
  "fresh-choicepoint-legacy",
  "legacy-parse-portable",
  "second-choice-translation",
  "choice-direct-strict-translate",
  "registry-from-tuple-authority",
  "portable-registry-order-disabled",
  "outcome-seal-before-sort",
  "selected-order-lexical",
  "selectable-order-lexical",
  "differing-all-fields",
  "selected-tuple-exported-field",
  "custom-expectation-complete-tuple",
  "selected-tuple-unselected-bypass",
  "portable-json-digest-identity",
  "bytes-blind-wrong-slot",
  "ordered-list-decision-wrong-slot",
	"portable-decision-propose-preflight-bypass",
	"portable-decision-revise-preflight-bypass",
	"portable-decision-final-budget-bypass",
	"decision-receipt-count-guard-bypass",
	"decision-receipt-payload-guard-bypass",
	"choicepoint-receipt-count-guard-bypass",
	"choicepoint-receipt-payload-guard-bypass",
	"alias-admission-before-copy-bypass",
	"draft-alias-admission-bypass",
	"proof-copy-before-admission",
	"confirmed-roster-max-bypass",
	"receipt-wire-grade-budget-bypass",
	"decision-receipt-rogue-precopy",
	"choicepoint-receipt-rogue-precopy",
	"alias-rogue-precopy",
	"projection-proof-rogue-precopy",
	"decision-receipt-json-precopy",
	"choicepoint-receipt-json-precopy",
	"alias-join-precopy",
	"projection-proof-bytes-clone-precopy",
	"alias-byte-guard-bypass",
	"projection-proof-byte-guard-bypass",
	"confirmed-translation-limit-drift",
	"blind-alias-limit-drift",
	"decision-receipt-limit-drift",
	"choicepoint-receipt-limit-drift",
	"choicepoint-nested-limit-drift",
  "legacy-preparation-guard-bypass",
  "portable-inspection-exported-field",
  "preparation-current-head-bypass",
  "forged-preparation-seal",
  "choice-mode-schema-drift",
  "exact-value-tags-schema-drift",
  "exact-value-schema-opened",
  "bytes-schema-slot-drift",
  "ordered-list-schema-slot-drift",
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
const requiredRosterDigest = "4f89f1624e683f1c84ed692658d0c1669d55de1cc05f77c579e60d07319661b9";

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

async function requireExact(fixture, path, needle) {
	const source = await readFile(join(fixture, path), "utf8");
	const first = source.indexOf(needle);
	if (first < 0 || source.indexOf(needle, first + needle.length) >= 0) {
		throw new Error(`self-test baseline anchor is not unique: ${path}: ${needle}`);
	}
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

const c3StoreOwnerPath = "internal/store/nonhead_contract.go";
const c3StoreModelImport = 'contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"';
const c3StoreInput = String.raw`type ConformanceAttemptInput struct {
	ContractBundleDigest        domain.Digest
	ResidueHeadDigest           domain.Digest
	TreeIdentityDigest          domain.Digest
	MaterializationPolicyDigest domain.Digest
}`;
const c3StoreRoots = "type ConformanceAttemptRoots struct{ roots conformanceAttemptRoots }";
const c3StoreMethodSignatures = Object.freeze([
	"func (store *ObjectStore) AllocateConformanceAttempt(ctx context.Context, input ConformanceAttemptInput) (ConformanceAttemptRecord, error)",
	"func (store *ObjectStore) OpenConformanceAttempt(ctx context.Context, digest domain.Digest) (ConformanceAttemptRecord, error)",
	"func (store *ObjectStore) PersistContractTargetRecord(ctx context.Context, attempt ConformanceAttemptRecord, target contractmodel.ContractExecutionTarget) (ContractTargetRecord, error)",
	"func (store *ObjectStore) OpenContractTargetRecord(ctx context.Context, attempt ConformanceAttemptRecord) (ContractTargetRecord, error)",
]);

async function requireC3StoreBridge(fixture) {
	for (const needle of [
		c3StoreModelImport,
		c3StoreInput,
		c3StoreRoots,
		"type ConformanceAttemptRecord struct {",
		"type ContractTargetRecord struct {",
		...c3StoreMethodSignatures,
	]) {
		await requireExact(fixture, c3StoreOwnerPath, needle);
	}
}

async function ensureC3StoreBridge(fixture) {
	const source = await readFile(join(fixture, c3StoreOwnerPath), "utf8");
	if (!source.includes(c3StoreModelImport)) {
		for (const partial of [
			"ConformanceAttemptInput",
			"ConformanceAttemptRoots",
			"ConformanceAttemptRecord",
			"ContractTargetRecord",
		]) {
			if (source.includes(partial)) {
				throw new Error(`self-test historical baseline contains partial C3 store surface: ${partial}`);
			}
		}
		await replaceExact(
			fixture,
			c3StoreOwnerPath,
			'\t"github.com/nelsonwerd/countershape/internal/canon"\n\t"github.com/nelsonwerd/countershape/internal/domain"',
			`\t"github.com/nelsonwerd/countershape/internal/canon"\n\t${c3StoreModelImport}\n\t"github.com/nelsonwerd/countershape/internal/domain"`,
		);
		await appendSource(
			fixture,
			c3StoreOwnerPath,
			String.raw`
${c3StoreInput}
type conformanceAttemptRoots struct{}
${c3StoreRoots}
type ConformanceAttemptRecord struct {
	marker byte
}
type ContractTargetRecord struct {
	marker byte
}
${c3StoreMethodSignatures[0]} { return ConformanceAttemptRecord{}, nil }
${c3StoreMethodSignatures[1]} { return ConformanceAttemptRecord{}, nil }
${c3StoreMethodSignatures[2]} { return ContractTargetRecord{}, nil }
${c3StoreMethodSignatures[3]} { return ContractTargetRecord{}, nil }
`,
		);
	}
	await requireC3StoreBridge(fixture);
}

const c4StoreOwnerPath = "internal/store/contract_run_bridge.go";
const c4StoreModelImport = 'contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"';
const c4StoreHostEpochImport = '"github.com/nelsonwerd/countershape/internal/hostepoch"';
const c4ProcessMechanicsImport = '"github.com/nelsonwerd/countershape/internal/processmechanics"';
const c4ConsumeForStartSignature =
	"func (owner ContractRunOwner) ConsumeForStart(ctx context.Context, bindingDigest domain.Digest) error";

async function requireC4Boundaries(fixture) {
	for (const needle of [
		c4StoreModelImport,
		c4StoreHostEpochImport,
		"type ContractRunOwner struct {",
		"type PrivateRunManifest struct {",
		"type FinalizedRunRecord struct {",
		"type TerminalClosure struct {",
		"type ContractExecutionRecord struct {",
		"func AcquireContractRunOwner(",
		c4ConsumeForStartSignature,
		"func PersistContractExecutionRecord(",
	]) {
		await requireExact(fixture, c4StoreOwnerPath, needle);
	}
	for (const path of ["internal/world/process.go", "internal/world/process_darwin.go"]) {
		await requireExact(fixture, path, c4ProcessMechanicsImport);
	}
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
			"package choice\n\n// AdvanceHead WakeEvent DisplayGroups() GeneralizationClaimed: true; projectionprofile.NewDerived; domain.AdapterCLI; HTTPStimulus",
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
		case "c3-store-bridge":
			await ensureC3StoreBridge(fixture);
			await ensureC3StoreBridge(fixture);
			break;
		case "c3-store-bridge-raw-import":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				'contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"',
				"contractmodel `github.com/nelsonwerd/countershape/internal/contractexec/model`",
			);
			break;
		case "c3-store-bridge-renamed-identifiers":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"func (store *ObjectStore) AllocateConformanceAttempt(ctx context.Context, input ConformanceAttemptInput)",
				"func (repository *ObjectStore) AllocateConformanceAttempt(callContext context.Context, request ConformanceAttemptInput)",
			);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"func (store *ObjectStore) OpenConformanceAttempt(ctx context.Context, digest domain.Digest)",
				"func (repository *ObjectStore) OpenConformanceAttempt(callContext context.Context, attemptDigest domain.Digest)",
			);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"func (store *ObjectStore) PersistContractTargetRecord(ctx context.Context, attempt ConformanceAttemptRecord, target contractmodel.ContractExecutionTarget)",
				"func (repository *ObjectStore) PersistContractTargetRecord(callContext context.Context, attemptRecord ConformanceAttemptRecord, executionTarget contractmodel.ContractExecutionTarget)",
			);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"func (store *ObjectStore) OpenContractTargetRecord(ctx context.Context, attempt ConformanceAttemptRecord)",
				"func (repository *ObjectStore) OpenContractTargetRecord(callContext context.Context, attemptRecord ConformanceAttemptRecord)",
			);
			break;
		case "c3-store-bridge-wrong-import-alias":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				'contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"',
				'modelalias "github.com/nelsonwerd/countershape/internal/contractexec/model"',
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-model-import-without-bridge":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				c3StoreOwnerPath,
				c3StoreRoots,
				"type MissingConformanceAttemptRoots struct{ roots conformanceAttemptRoots }",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-signature-drift":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				c3StoreOwnerPath,
				c3StoreMethodSignatures[2],
				"func (store *ObjectStore) PersistContractTargetRecord(ctx context.Context, attempt ConformanceAttemptRecord, target SemanticObject) (ContractTargetRecord, error)",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-field-type-drift":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"ContractBundleDigest        domain.Digest",
				"ContractBundleDigest        string",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-field-tag":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"ContractBundleDigest        domain.Digest",
				'ContractBundleDigest        domain.Digest `json:"contract_bundle_digest"`',
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-swapped-field-order":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"ContractBundleDigest        domain.Digest\n\tResidueHeadDigest           domain.Digest",
				"ResidueHeadDigest           domain.Digest\n\tContractBundleDigest        domain.Digest",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-hidden-field":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"MaterializationPolicyDigest domain.Digest\n}",
				"MaterializationPolicyDigest domain.Digest\n\thidden                     string\n}",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-exported-fifth-field":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"MaterializationPolicyDigest domain.Digest\n}",
				"MaterializationPolicyDigest domain.Digest\n\tExtraDigest                 domain.Digest\n}",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-extra-parameter":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				c3StoreMethodSignatures[0],
				"func (store *ObjectStore) AllocateConformanceAttempt(ctx context.Context, extra string, input ConformanceAttemptInput) (ConformanceAttemptRecord, error)",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-result-drift":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/nonhead_contract.go",
				"func (store *ObjectStore) OpenConformanceAttempt(ctx context.Context, digest domain.Digest) (ConformanceAttemptRecord, error)",
				"func (store *ObjectStore) OpenConformanceAttempt(ctx context.Context, digest domain.Digest) (ContractTargetRecord, error)",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-model-import-foreign-file":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/object_store.go",
				'\t"github.com/nelsonwerd/countershape/internal/canon"\n\t"github.com/nelsonwerd/countershape/internal/domain"',
				'\t"github.com/nelsonwerd/countershape/internal/canon"\n\t_ "github.com/nelsonwerd/countershape/internal/contractexec/model"\n\t"github.com/nelsonwerd/countershape/internal/domain"',
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-model-import-raw-foreign-file":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/object_store.go",
				'\t"github.com/nelsonwerd/countershape/internal/canon"\n\t"github.com/nelsonwerd/countershape/internal/domain"',
				'\t"github.com/nelsonwerd/countershape/internal/canon"\n\t_ `github.com/nelsonwerd/countershape/internal/contractexec/model`\n\t"github.com/nelsonwerd/countershape/internal/domain"',
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-model-import-escaped-foreign-file":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				"internal/store/object_store.go",
				'\t"github.com/nelsonwerd/countershape/internal/canon"\n\t"github.com/nelsonwerd/countershape/internal/domain"',
				'\t"github.com/nelsonwerd/countershape/internal/canon"\n\t_ "github.com/nelsonwerd/countershap\\x65/internal/contractexec/model"\n\t"github.com/nelsonwerd/countershape/internal/domain"',
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-import-camouflage-raw-string":
			await appendSource(
				fixture,
				"internal/store/object_store.go",
				'\nvar c3ImportCamouflage = `import contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"`\n',
			);
			break;
		case "c3-store-bridge-local-shadow":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				c3StoreOwnerPath,
				c3StoreInput,
				String.raw`func c3LocalInputShadow() {
	type ConformanceAttemptInput struct {
		ContractBundleDigest        domain.Digest
		ResidueHeadDigest           domain.Digest
		TreeIdentityDigest          domain.Digest
		MaterializationPolicyDigest domain.Digest
	}
	_ = ConformanceAttemptInput{}
}`,
			);
			await appendSource(fixture, "internal/store/object_store.go", "\ntype ConformanceAttemptInput struct { Wrong string }\n");
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-type-alias-foreign-owner":
			await ensureC3StoreBridge(fixture);
			await replaceExact(
				fixture,
				c3StoreOwnerPath,
				`${c3StoreRoots}\n`,
				"",
			);
			await appendSource(fixture, "internal/store/object_store.go", "\ntype ConformanceAttemptRoots = SemanticObject\n");
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-duplicate-top-level-type":
			await ensureC3StoreBridge(fixture);
			await appendSource(fixture, "internal/store/object_store.go", "\ntype ConformanceAttemptRoots struct{}\n");
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c3-store-bridge-duplicate-method":
			await ensureC3StoreBridge(fixture);
			await appendSource(
				fixture,
				"internal/store/object_store.go",
				"\nfunc (repository *ObjectStore) OpenContractTargetRecord(callContext context.Context, attemptRecord ConformanceAttemptRecord) (ContractTargetRecord, error) { return ContractTargetRecord{}, nil }\n",
			);
			expected = "U6_STORE_C3_MODEL_IMPORT_NOT_ADMITTED";
			break;
		case "c4-store-bridge":
			await requireC4Boundaries(fixture);
			break;
		case "c4-store-bridge-renamed-identifiers":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				c4StoreOwnerPath,
				c4ConsumeForStartSignature,
				"func (permit ContractRunOwner) ConsumeForStart(callContext context.Context, preparedBinding domain.Digest) error",
			);
			break;
		case "c4-store-bridge-wrong-model-alias":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				c4StoreOwnerPath,
				c4StoreModelImport,
				'modelalias "github.com/nelsonwerd/countershape/internal/contractexec/model"',
			);
			expected = "U6_STORE_C4_BRIDGE_NOT_ADMITTED";
			break;
		case "c4-store-bridge-wrong-hostepoch-alias":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				c4StoreOwnerPath,
				c4StoreHostEpochImport,
				'epochalias "github.com/nelsonwerd/countershape/internal/hostepoch"',
			);
			expected = "U6_STORE_C4_BRIDGE_NOT_ADMITTED";
			break;
		case "c4-store-bridge-exported-embedded-field":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				c4StoreOwnerPath,
				"type ContractExecutionRecord struct {\n\tstore  *ObjectStore",
				"type ContractExecutionRecord struct {\n\tstore  *ObjectStore\n\t*ObjectStore",
			);
			expected = "U6_STORE_C4_BRIDGE_NOT_ADMITTED";
			break;
		case "c4-store-bridge-foreign-type-owner":
			await requireC4Boundaries(fixture);
			await appendSource(
				fixture,
				"internal/store/object_store.go",
				"\ntype ContractExecutionRecord struct { hidden byte }\n",
			);
			expected = "U6_STORE_C4_BRIDGE_NOT_ADMITTED";
			break;
		case "c4-store-bridge-receiver-kind-drift":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				c4StoreOwnerPath,
				"func (record ContractExecutionRecord) Model() contractmodel.ContractExecution {",
				"func (record *ContractExecutionRecord) Model() contractmodel.ContractExecution {",
			);
			expected = "U6_STORE_C4_BRIDGE_NOT_ADMITTED";
			break;
		case "c4-store-bridge-foreign-method-owner":
			await requireC4Boundaries(fixture);
			await appendSource(
				fixture,
				"internal/store/object_store.go",
				"\nfunc (record ContractExecutionRecord) ReopenAuthority() ContractExecutionRecord { return record }\n",
			);
			expected = "U6_STORE_C4_BRIDGE_NOT_ADMITTED";
			break;
		case "c4-store-hostepoch-import-foreign-file":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				"internal/store/object_store.go",
				'\t"github.com/nelsonwerd/countershape/internal/domain"',
				'\t"github.com/nelsonwerd/countershape/internal/domain"\n\t_ "github.com/nelsonwerd/countershape/internal/hostepoch"',
			);
			expected = "U6_STORE_C4_BRIDGE_NOT_ADMITTED";
			break;
		case "c4-world-mechanics-wrong-alias":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				"internal/world/process.go",
				c4ProcessMechanicsImport,
				'mechanics "github.com/nelsonwerd/countershape/internal/processmechanics"',
			);
			expected = "U6_WORLD_C4_MECHANICS_IMPORT_NOT_ADMITTED";
			break;
		case "c4-world-mechanics-foreign-file":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				"internal/world/capture.go",
				'\t"math"',
				'\t"math"\n\n\t_ "github.com/nelsonwerd/countershape/internal/processmechanics"',
			);
			expected = "U6_WORLD_C4_MECHANICS_IMPORT_NOT_ADMITTED";
			break;
		case "c4-world-mechanics-missing-authorized-file":
			await requireC4Boundaries(fixture);
			await replaceExact(
				fixture,
				"internal/world/process.go",
				`\n\t${c4ProcessMechanicsImport}`,
				"",
			);
			expected = "U6_INTERNAL_IMPORT_LATTICE";
			break;
		case "eligibility-core-capability-import":
			await replaceExact(
				fixture,
				"internal/observe/eligibilitycore/eligibility.go",
				'import "github.com/nelsonwerd/countershape/internal/domain"',
				'import (\n\t_ "os"\n\n\t"github.com/nelsonwerd/countershape/internal/domain"\n)',
			);
			expected = "U6_ELIGIBILITY_CORE_IMPORT_ROSTER";
			break;
		case "eligibility-core-api-opening":
			await appendSource(
				fixture,
				"internal/observe/eligibilitycore/eligibility.go",
				"\nfunc OpenCapability() {}\n",
			);
			expected = "U6_ELIGIBILITY_CORE_API_ROSTER";
			break;
		case "eligibility-owner-bypass":
			await replaceExact(
				fixture,
				"internal/observe/eligibility.go",
				"eligibilitycore.Select(eligibilitycore.ControlIneligible, fact.controls)",
				"eligibilitycore.Select(eligibilitycore.BehaviorCaptured, nil)",
			);
			expected = "U6_ELIGIBILITY_OWNER_DATAFLOW";
			break;
		case "portablevalue-full-adapter-import":
			await replaceExact(
				fixture,
				"internal/portablevalue/value.go",
				"package portablevalue",
				"package portablevalue\n\nimport _ \"github.com/nelsonwerd/countershape/internal/adapters/cli\"",
			);
			expected = "U6_ADAPTER_IMPORT_OUTSIDE_TRANSLATOR_OR_WORLD_MODEL";
			break;
		case "world-full-adapter-import":
			await replaceExact(
				fixture,
				"internal/world/cli_api.go",
				"github.com/nelsonwerd/countershape/internal/adapters/cli/model",
				"github.com/nelsonwerd/countershape/internal/adapters/cli",
			);
			expected = "U6_ADAPTER_IMPORT_OUTSIDE_TRANSLATOR_OR_WORLD_MODEL";
			break;
		case "second-profile-derivation-reference":
			await writeSource(
				fixture,
				"internal/projectiontranslate/forged_profile.go",
				"package projectiontranslate\n\nimport pp \"github.com/nelsonwerd/countershape/internal/projectionprofile\"\n\nvar forgedProfileConstructor = pp.NewDerived\n",
			);
			expected = "U6_PROFILE_DERIVATION_REFERENCE_SURFACE_NOT_EXACT";
			break;
		case "profile-constructor-reexport":
			await writeSource(
				fixture,
				"internal/projectionprofile/forged_profile.go",
				"package projectionprofile\n\nvar ForgedProfileConstructor = NewDerived\n",
			);
			expected = "U6_PROFILE_INTERNAL_REBUILD_SURFACE_NOT_EXACT";
			break;
		case "portable-profile-roster-firewall-bypass":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/translate.go",
				"if err := verifyPortableProfileRosterV1(); err != nil {",
				"if false {",
			);
			expected = "U6_PROFILE_ROSTER_FIREWALL_NOT_EXACT";
			break;
		case "portable-profile-roster-version-drift":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/cli.go",
				'cliTranslatorVersion = "v1"',
				'cliTranslatorVersion = "v1-drift"',
			);
			expected = "U6_PROFILE_ROSTER_TRANSLATOR_IDENTITY_NOT_EXACT";
			break;
		case "portable-expectation-roster-firewall-bypass":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/profile_roster.go",
				"expectationDigest, expectationCount, err := derivePortableExpectationDomainRosterV1()",
				' expectationDigest, expectationCount, err := domain.Digest(""), 0, error(nil)',
			);
			expected = "U6_PORTABLE_MODE_SEMANTICS_FIREWALL_NOT_EXACT";
			break;
		case "portable-mode-semantics-firewall-bypass":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/profile_roster.go",
				"semanticsDigest, err := derivePortableModeSemanticsV1()",
				' semanticsDigest, err := domain.Digest(""), error(nil)',
			);
			expected = "U6_PORTABLE_MODE_SEMANTICS_FIREWALL_NOT_EXACT";
			break;
		case "expectation-domain-exported-field":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/expectation.go",
				"type ExpectationDomain struct {\n\tprofile   projectionprofile.Profile",
				"type ExpectationDomain struct {\n\tProfile   projectionprofile.Profile",
			);
			expected = "U6_EXPECTATION_DOMAIN_SURFACE_NOT_CLOSED";
			break;
		case "portable-registry-expectation-binding-bypass":
			await replaceExact(
				fixture,
				"internal/choice/portable.go",
				"!bytes.Equal(expectation.ProfileBytes(), profile.CanonicalBytes())",
				"false",
			);
			expected = "U6_PORTABLE_REGISTRY_EXPECTATION_BINDING_NOT_EXACT";
			break;
		case "expectation-domain-propagation-bypass":
			await replaceExact(
				fixture,
				"internal/choice/portable.go",
				"newPortableFieldRegistry(translations.Profile(), translations.ExpectationDomain())",
				"newPortableFieldRegistry(translations.Profile(), projectiontranslate.ExpectationDomain{})",
			);
			expected = "U6_EXPECTATION_DOMAIN_NOT_PROPAGATED_TO_CHOICE";
			break;
		case "selected-expectation-validation-bypass":
			await replaceExact(
				fixture,
				"internal/choice/validation.go",
				"if err := r.validatePortableExpectation(selectedOrder, values, true); err != nil {",
				"if false {",
			);
			expected = "U6_SELECTED_TUPLE_VALIDATION_NOT_EXACT";
			break;
		case "complete-expectation-validation-bypass":
			await replaceExact(
				fixture,
				"internal/choice/validation.go",
				"if err := r.validatePortableExpectation(r.orderedIDs, fields, false); err != nil {",
				"if false {",
			);
			expected = "U6_COMPLETE_TUPLE_EXPECTATION_VALIDATION_NOT_EXACT";
			break;
		case "universe-expectation-seal-bypass":
			await replaceExact(
				fixture,
				"internal/choice/validation.go",
				"if registry.expectationDomain.Valid() {",
				"if false {",
			);
			expected = "U6_EXPECTATION_DOMAIN_NOT_BOUND_TO_UNIVERSE_SEAL";
			break;
		case "expectation-domain-validity-bypass":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/expectation.go",
				"return err == nil && rebuilt.digest == d.digest && bytes.Equal(rebuilt.canonical, d.canonical)",
				"return err == nil",
			);
			expected = "U6_EXPECTATION_DOMAIN_VALIDITY_NOT_EXACT";
			break;
		case "expectation-domain-validity-dead-return":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/expectation.go",
				"func (d ExpectationDomain) Valid() bool {",
				"func (d ExpectationDomain) Valid() bool {\n\treturn true",
			);
			expected = "U6_EXPECTATION_DOMAIN_VALIDITY_NOT_EXACT";
			break;
		case "expectation-domain-validation-dead-return":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/expectation.go",
				"func (d ExpectationDomain) ValidateSelected(selected []SelectedValue) error {",
				"func (d ExpectationDomain) ValidateSelected(selected []SelectedValue) error {\n\treturn nil",
			);
			expected = "U6_EXPECTATION_DOMAIN_VALIDATION_NOT_EXACT";
			break;
		case "confirmed-expectation-authority-detached":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/confirmed.go",
				"resolved: resolved, expectation: expectation, outcomes: verified",
				"resolved: resolved, expectation: ExpectationDomain{}, outcomes: verified",
			);
			expected = "U6_CONFIRMED_EXPECTATION_AUTHORITY_NOT_EXACT";
			break;
		case "choice-expectation-validator-bypass":
			await replaceExact(
				fixture,
				"internal/choice/validation.go",
				"if err := r.expectationDomain.ValidateSelected(selected); err != nil {",
				"if err := func() error { return nil }(); err != nil {",
			);
			expected = "U6_CHOICE_EXPECTATION_VALIDATION_NOT_EXACT";
			break;
		case "choice-expectation-validator-dead-return":
			await replaceExact(
				fixture,
				"internal/choice/validation.go",
				"func (r FieldRegistry) validatePortableExpectation(order []string, values map[string]ExactValue, custom bool) error {",
				"func (r FieldRegistry) validatePortableExpectation(order []string, values map[string]ExactValue, custom bool) error {\n\treturn nil",
			);
			expected = "U6_CHOICE_EXPECTATION_VALIDATION_NOT_EXACT";
			break;
      case "choice-adapter-domain-switch":
        await writeSource(
          fixture,
          "internal/choice/forged_adapter_branch.go",
          "package choice\n\nimport \"github.com/nelsonwerd/countershape/internal/domain\"\n\nvar forgedAdapterBranch = domain.AdapterCLI\n",
        );
        expected = "U6_CHOICE_SWITCHES_ON_ADAPTER_DOMAIN";
        break;
      case "caller-projection-builder":
        await writeSource(
          fixture,
          "internal/choice/forged_projection_builder.go",
          "package choice\n\nfunc NewFieldRegistry() {}\n",
        );
        expected = "U6_CALLER_AUTHORED_PROJECTION_AUTHORITY_RESIDUE";
        break;
      case "fresh-choicepoint-legacy":
        await replaceExact(
          fixture,
          "internal/choice/choicepoint.go",
          "return buildChoicepointRecord(input, choicepointPortable, nil)",
          "return buildChoicepointRecord(input, choicepointLegacyWhole, nil)",
        );
        expected = "U6_FRESH_CHOICEPOINT_MODE_NOT_EXACT";
        break;
      case "legacy-parse-portable":
        await replaceExact(
          fixture,
          "internal/choice/choicepoint.go",
          "mode = choicepointLegacyWhole // MUTANT_P07B_LEGACY_REBUILT_PORTABLE",
          "mode = choicepointPortable // MUTANT_P07B_LEGACY_REBUILT_PORTABLE",
        );
        expected = "U6_CHOICEPOINT_PARSE_MODE_DISPATCH_NOT_EXACT";
        break;
      case "second-choice-translation":
        await writeSource(
          fixture,
          "internal/choice/forged_translation.go",
          "package choice\n\nimport \"github.com/nelsonwerd/countershape/internal/projectiontranslate\"\n\nvar forgedTranslation = projectiontranslate.TranslateConfirmed\n",
        );
        expected = "U6_PROOF_FIRST_TRANSLATION_SURFACE_NOT_EXACT";
        break;
      case "choice-direct-strict-translate":
        await writeSource(
          fixture,
          "internal/choice/forged_strict_translate.go",
          "package choice\n\nimport \"github.com/nelsonwerd/countershape/internal/projectiontranslate\"\n\nvar forgedStrictTranslate = projectiontranslate.StrictTranslate\n",
        );
        expected = "U6_CHOICE_BYPASSES_PROOF_FIRST_TRANSLATION";
        break;
      case "registry-from-tuple-authority":
        await replaceExact(
          fixture,
          "internal/choice/portable.go",
          "func newPortableFieldRegistry(profile projectionprofile.Profile, expectation projectiontranslate.ExpectationDomain)",
          "func newPortableFieldRegistry(profile projectionprofile.Profile, expectation projectiontranslate.ExpectationDomain, tuple CompleteTuple)",
        );
        expected = "U6_PORTABLE_REGISTRY_ACCEPTS_NONPROFILE_AUTHORITY";
        break;
      case "portable-registry-order-disabled":
        await replaceExact(
          fixture,
          "internal/choice/portable.go",
          "newFieldRegistry(definitions, true)",
          "newFieldRegistry(definitions, false)",
        );
        expected = "U6_PORTABLE_REGISTRY_DERIVATION_NOT_EXACT";
        break;
      case "outcome-seal-before-sort":
        await replaceExact(
          fixture,
          "internal/choice/portable.go",
          "\tsort.Slice(set.ordered, func(i, j int) bool { return set.ordered[i].ref.id.text < set.ordered[j].ref.id.text })\n\tseal, err := makeConfirmedOutcomeSetSeal(set.registry, set.outcomeMapDigest, set.preservationDigest, set.ordered)",
          "\tseal, err := makeConfirmedOutcomeSetSeal(set.registry, set.outcomeMapDigest, set.preservationDigest, set.ordered)\n\tsort.Slice(set.ordered, func(i, j int) bool { return set.ordered[i].ref.id.text < set.ordered[j].ref.id.text })",
        );
        expected = "U6_PORTABLE_OUTCOME_ID_ORDER_BEFORE_SEAL";
        break;
      case "selected-order-lexical":
        await replaceExact(
          fixture,
          "internal/choice/validation.go",
          "\tfor _, fieldID := range r.orderedIDs { // MUTANT_P07B_SELECTED_LEXICAL_ORDER",
          "\tsort.Strings(raw)\n\tfor _, fieldID := range raw { // MUTANT_P07B_SELECTED_LEXICAL_ORDER",
        );
        expected = "U6_SELECTED_FIELD_ORDER_LEXICAL";
        break;
      case "selectable-order-lexical":
        await replaceExact(
          fixture,
          "internal/choice/blind.go",
          "\tfor _, field := range record.confirmed.registry.Definitions() {\n\t\tselectable = append(selectable, field.ID)\n\t}",
          "\tfor _, field := range record.confirmed.registry.Definitions() {\n\t\tselectable = append(selectable, field.ID)\n\t}\n\tsort.Strings(selectable)",
        );
        expected = "U6_SELECTABLE_FIELD_ORDER_LEXICAL";
        break;
      case "differing-all-fields":
        await replaceExact(fixture, "internal/choice/blind.go", "\t\tif differs {", "\t\tif true {");
        expected = "U6_DIFFERING_FIELD_FILTER_NOT_EXACT";
        break;
      case "selected-tuple-exported-field":
        await replaceExact(
          fixture,
          "internal/choice/validation.go",
          "type SelectedTuple struct {\n\tuniverse    canon.Digest",
          "type SelectedTuple struct {\n\tUniverse    canon.Digest",
        );
        expected = "U6_SELECTED_TUPLE_SURFACE_NOT_CLOSED";
        break;
      case "custom-expectation-complete-tuple":
        await replaceExact(fixture, "internal/choice/session.go", "\tCustomExpectation    *SelectedTuple", "\tCustomExpectation    *CompleteTuple");
        expected = "U6_CUSTOM_EXPECTATION_NOT_SELECTED_ONLY";
        break;
      case "selected-tuple-unselected-bypass":
        await replaceExact(
          fixture,
          "internal/choice/validation.go",
          "\t\tif _, selectedField := allowed[fieldID.text]; !selectedField { // MUTANT_P07B_CUSTOM_UNSELECTED_FIELD",
          "\t\tif false { // MUTANT_P07B_CUSTOM_UNSELECTED_FIELD",
        );
        expected = "U6_SELECTED_TUPLE_VALIDATION_NOT_EXACT";
        break;
      case "portable-json-digest-identity":
        await replaceExact(
          fixture,
          "internal/choice/validation.go",
          "return \"J\" + lengthPrefix(string(v.canonical))",
          "return \"J\" + lengthPrefix(v.text)",
        );
        expected = "U6_PORTABLE_VALUE_IDENTITY_NOT_EXACT";
        break;
      case "bytes-blind-wrong-slot":
        await replaceExact(
          fixture,
          "internal/choice/blind.go",
          "result.Text = base64.StdEncoding.EncodeToString(value.Bytes())",
          "result.CanonicalJSONBase64 = base64.StdEncoding.EncodeToString(value.Bytes())",
        );
        expected = "U6_PORTABLE_BLIND_WIRE_NOT_COMPATIBLE";
        break;
      case "ordered-list-decision-wrong-slot":
        await replaceExact(
          fixture,
          "internal/choice/session.go",
          "wire.CanonicalJSONBase64 = base64.StdEncoding.EncodeToString(value.CanonicalBytes())",
          "wire.Text = base64.StdEncoding.EncodeToString(value.CanonicalBytes())",
        );
        expected = "U6_PORTABLE_DECISION_WIRE_NOT_COMPATIBLE";
        break;
		case "portable-decision-propose-preflight-bypass":
			await replaceExact(
				fixture,
				"internal/choice/session.go",
				"// that would fit only after a smaller post-reveal change is refused here.\n\tif err := preflightPortableDecisionBudget(result); err != nil {",
				"// that would fit only after a smaller post-reveal change is refused here.\n\tif false {",
			);
			expected = "U6_PORTABLE_DECISION_DRAFT_PREFLIGHT_NOT_EXACT";
			break;
		case "portable-decision-revise-preflight-bypass":
			await replaceExact(
				fixture,
				"internal/choice/session.go",
				"result.state = SessionPostRevealRecorded\n\tif err := preflightPortableDecisionBudget(result); err != nil {",
				"result.state = SessionPostRevealRecorded\n\tif false {",
			);
			expected = "U6_PORTABLE_DECISION_DRAFT_PREFLIGHT_NOT_EXACT";
			break;
		case "portable-decision-final-budget-bypass":
			await replaceExact(
				fixture,
				"internal/choice/decision.go",
				"if enforcePortableBudget && session.record.mode == choicepointPortable {",
				"if false && session.record.mode == choicepointPortable {",
			);
			expected = "U6_PORTABLE_DECISION_FINAL_BUDGET_NOT_EXACT";
			break;
		case "decision-receipt-count-guard-bypass":
			await replaceExact(
				fixture,
				"internal/choice/decision.go",
				"if len(receipts) > canon.MaxContainerMembers {",
				"if false {",
			);
			expected = "U6_DECISION_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "decision-receipt-payload-guard-bypass":
			await replaceExact(
				fixture,
				"internal/choice/decision.go",
				"if !consumeReceiptWireBudget(wire, &remaining) {",
				"if false {",
			);
			expected = "U6_DECISION_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "choicepoint-receipt-payload-guard-bypass":
			await replaceExact(
				fixture,
				"internal/choice/choicepoint.go",
				"if !consumeReceiptWireBudget(wire, &remaining) {",
				"if false {",
			);
			expected = "U6_CHOICEPOINT_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "choicepoint-receipt-count-guard-bypass":
			await replaceExact(
				fixture,
				"internal/choice/choicepoint.go",
				"if len(receipts) > canon.MaxContainerMembers {",
				"if false {",
			);
			expected = "U6_CHOICEPOINT_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "alias-admission-before-copy-bypass":
			await replaceExact(
				fixture,
				"internal/choice/blind.go",
				"if err := v.admitAliases(raw); err != nil {\n\t\treturn nil, nil, err\n\t}",
				"if false {\n\t\treturn nil, nil, nil\n\t}",
			);
			expected = "U6_ALIAS_ADMISSION_NOT_BEFORE_COPY_SORT_RESOLVE";
			break;
		case "draft-alias-admission-bypass":
			await replaceExact(
				fixture,
				"internal/choice/session.go",
				"if err := view.admitAliases(input.AllowedAliases); err != nil {\n\t\treturn rulingDraft{}, err\n\t}",
				"if false {\n\t\treturn rulingDraft{}, nil\n\t}",
			);
			expected = "U6_DRAFT_ALIAS_ADMISSION_ORDER_NOT_EXACT";
			break;
		case "proof-copy-before-admission":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/confirmed.go",
				"\t\tif len(proof.CanonicalProjection) > maxConfirmedProjectionBytes-projectionBytes {\n\t\t\treturn ConfirmedTranslations{}, refuse(CodeTranslationLimit, \"\", \"confirmed projection proofs exceed the aggregate retained-byte ceiling\")\n\t\t}\n\t\tprojectionBytes += len(proof.CanonicalProjection)\n\t\tfrozenProjection := append([]byte(nil), proof.CanonicalProjection...)",
				"\t\tfrozenProjection := append([]byte(nil), proof.CanonicalProjection...)\n\t\tif len(proof.CanonicalProjection) > maxConfirmedProjectionBytes-projectionBytes {\n\t\t\treturn ConfirmedTranslations{}, refuse(CodeTranslationLimit, \"\", \"confirmed projection proofs exceed the aggregate retained-byte ceiling\")\n\t\t}\n\t\tprojectionBytes += len(proof.CanonicalProjection)",
			);
			expected = "U6_PROJECTION_PROOF_ADMISSION_NOT_BEFORE_COPY";
			break;
		case "confirmed-roster-max-bypass":
			await replaceExact(
				fixture,
				"internal/compare/outcome_map.go",
				"len(r.entries) < 2 || len(r.entries) > 4",
				"len(r.entries) < 2",
			);
			expected = "U6_CONFIRMED_ROSTER_RESOURCE_PROFILE_NOT_EXACT";
			break;
		case "receipt-wire-grade-budget-bypass":
			await replaceExact(
				fixture,
				"internal/choice/choicepoint.go",
				"[...]string{wire.Authority, wire.GradeVerbatim, wire.CommitOID, wire.CommandDigest}",
				"[...]string{wire.Authority, wire.CommitOID, wire.CommandDigest}",
			);
			expected = "U6_RECEIPT_WIRE_BUDGET_NOT_EXACT";
			break;
		case "decision-receipt-rogue-precopy":
			await replaceExact(
				fixture,
				"internal/choice/decision.go",
				"func normalizeDecisionReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {",
				"func normalizeDecisionReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {\n\t_ = append([]domain.ReceiptReference(nil), receipts...)",
			);
			expected = "U6_DECISION_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "choicepoint-receipt-rogue-precopy":
			await replaceExact(
				fixture,
				"internal/choice/choicepoint.go",
				"func normalizeChoicepointReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {",
				"func normalizeChoicepointReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {\n\t_ = append([]domain.ReceiptReference(nil), receipts...)",
			);
			expected = "U6_CHOICEPOINT_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "alias-rogue-precopy":
			await replaceExact(
				fixture,
				"internal/choice/blind.go",
				"func (v BlindView) normalizeAliases(raw []string) ([]string, []ConfirmedOutcomeRef, error) {",
				"func (v BlindView) normalizeAliases(raw []string) ([]string, []ConfirmedOutcomeRef, error) {\n\t_ = append([]string(nil), raw...)",
			);
			expected = "U6_ALIAS_ADMISSION_NOT_BEFORE_COPY_SORT_RESOLVE";
			break;
		case "projection-proof-rogue-precopy":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/confirmed.go",
				"for index, proof := range proofs {",
				"for index, proof := range proofs {\n\t\t_ = append([]byte(nil), proof.CanonicalProjection...)",
			);
			expected = "U6_PROJECTION_PROOF_ADMISSION_NOT_BEFORE_COPY";
			break;
		case "decision-receipt-json-precopy":
			await replaceExact(
				fixture,
				"internal/choice/decision.go",
				"func normalizeDecisionReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {",
				"func normalizeDecisionReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {\n\t_, _ = json.Marshal(receipts)",
			);
			expected = "U6_DECISION_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "choicepoint-receipt-json-precopy":
			await replaceExact(
				fixture,
				"internal/choice/choicepoint.go",
				"func normalizeChoicepointReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {",
				"func normalizeChoicepointReceipts(receipts []domain.ReceiptReference) ([]domain.ReceiptReference, error) {\n\t_, _ = json.Marshal(receipts)",
			);
			expected = "U6_CHOICEPOINT_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "alias-join-precopy":
			await replaceExact(
				fixture,
				"internal/choice/blind.go",
				"func (v BlindView) normalizeAliases(raw []string) ([]string, []ConfirmedOutcomeRef, error) {",
				"func (v BlindView) normalizeAliases(raw []string) ([]string, []ConfirmedOutcomeRef, error) {\n\t_ = strings.Clone(strings.Join(raw, \"\"))",
			);
			expected = "U6_ALIAS_ADMISSION_NOT_BEFORE_COPY_SORT_RESOLVE";
			break;
		case "projection-proof-bytes-clone-precopy":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/confirmed.go",
				"for index, proof := range proofs {",
				"for index, proof := range proofs {\n\t\t_ = bytes.Clone(proof.CanonicalProjection)",
			);
			expected = "U6_PROJECTION_PROOF_ADMISSION_NOT_BEFORE_COPY";
			break;
		case "alias-byte-guard-bypass":
			await replaceExact(
				fixture,
				"internal/choice/blind.go",
				"if len(alias) > maxBlindAliasBytes || len(alias) > remaining {",
				"if false {",
			);
			expected = "U6_ALIAS_ADMISSION_PROFILE_NOT_EXACT";
			break;
		case "projection-proof-byte-guard-bypass":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/confirmed.go",
				"if len(proof.CanonicalProjection) > maxConfirmedProjectionBytes-projectionBytes {",
				"if false {",
			);
			expected = "U6_PROJECTION_PROOF_ADMISSION_NOT_BEFORE_COPY";
			break;
		case "confirmed-translation-limit-drift":
			await replaceExact(
				fixture,
				"internal/projectiontranslate/confirmed.go",
				"maxConfirmedProjectionBytes = 600 * 1024",
				"maxConfirmedProjectionBytes = 601 * 1024",
			);
			expected = "U6_CONFIRMED_TRANSLATION_RESOURCE_PROFILE_NOT_EXACT";
			break;
		case "blind-alias-limit-drift":
			await replaceExact(
				fixture,
				"internal/choice/blind.go",
				"const maxBlindAliasBytes = len(\"blind:\") + 64",
				"const maxBlindAliasBytes = len(\"blind:\") + 65",
			);
			expected = "U6_ALIAS_ADMISSION_PROFILE_NOT_EXACT";
			break;
		case "decision-receipt-limit-drift":
			await replaceExact(
				fixture,
				"internal/choice/decision.go",
				"remaining := canon.MaxInputBytes",
				"remaining := canon.MaxInputBytes + 1",
			);
			expected = "U6_DECISION_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "choicepoint-receipt-limit-drift":
			await replaceExact(
				fixture,
				"internal/choice/choicepoint.go",
				"remaining := canon.MaxInputBytes",
				"remaining := canon.MaxInputBytes + 1",
			);
			expected = "U6_CHOICEPOINT_RECEIPT_ADMISSION_NOT_BOUNDED";
			break;
		case "choicepoint-nested-limit-drift":
			await replaceExact(
				fixture,
				"internal/choice/choicepoint.go",
				"maxChoicepointNestedRawBytes = 600 * 1024",
				"maxChoicepointNestedRawBytes = 601 * 1024",
			);
			expected = "U6_CHOICEPOINT_RESOURCE_PROFILE_NOT_EXACT";
			break;
      case "legacy-preparation-guard-bypass":
        await replaceExact(
          fixture,
          "internal/choice/portable.go",
          "if decision.choicepoint.mode == choicepointLegacyWhole {",
          "if false {",
        );
        expected = "U6_LEGACY_PORTABLE_PREPARATION_GUARD_NOT_EXACT";
        break;
      case "portable-inspection-exported-field":
        await replaceExact(
          fixture,
          "internal/choice/portable.go",
          "type PortableRulingInspection struct {\n\tdecisionDigest domain.Digest",
          "type PortableRulingInspection struct {\n\tDecisionDigest domain.Digest",
        );
        expected = "U6_PORTABLE_INSPECTION_SURFACE_NOT_CLOSED";
        break;
      case "preparation-current-head-bypass":
        await replaceExact(
          fixture,
          "internal/choice/promotion/service.go",
          "\tif err := validateRuling(ctx, objectStore, ruling); err != nil {\n\t\treturn PortableRulingPreparation{}, err\n\t}\n\tinspection, err := choice.InspectPortableRuling(ruling.record)",
          "\tif false {\n\t\treturn PortableRulingPreparation{}, nil\n\t}\n\tinspection, err := choice.InspectPortableRuling(ruling.record)",
        );
        expected = "U6_PORTABLE_PREPARATION_CURRENT_HEAD_GUARD_NOT_EXACT";
        break;
      case "forged-preparation-seal":
        await writeSource(
          fixture,
          "internal/choice/promotion/forged_portable_preparation.go",
          "package promotion\n\nvar forgedPortablePreparationSeal = &portableRulingPreparationSeal{marker: 1}\n",
        );
        expected = "U6_PORTABLE_PREPARATION_CONSTRUCTION_OUTSIDE_OWNER";
        break;
      case "choice-mode-schema-drift": {
        const path = join(fixture, "spec/schema/v1/choicepoint.schema.json");
        const value = JSON.parse(await readFile(path, "utf8"));
        value.properties.choice_projection_mode.enum = ["WHOLE_EXACT_CANONICAL_PROJECTION_V1"];
        await writeFile(path, `${JSON.stringify(value)}\n`, { mode: 0o600 });
        expected = "U6_CHOICEPOINT_MODE_SCHEMA_NOT_EXACT";
        break;
      }
      case "exact-value-tags-schema-drift": {
        const path = join(fixture, "spec/schema/v1/decision-record.schema.json");
        const value = JSON.parse(await readFile(path, "utf8"));
        value.$defs.ExactValue.properties.tag.enum = value.$defs.ExactValue.properties.tag.enum.filter((tag) => tag !== "ORDERED_STRING_LIST");
        await writeFile(path, `${JSON.stringify(value)}\n`, { mode: 0o600 });
        expected = "U6_EXACT_VALUE_TAG_SCHEMA_NOT_EXACT";
        break;
      }
      case "exact-value-schema-opened": {
        const path = join(fixture, "spec/schema/v1/decision-record.schema.json");
        const value = JSON.parse(await readFile(path, "utf8"));
        value.$defs.ExactValue.additionalProperties = true;
        await writeFile(path, `${JSON.stringify(value)}\n`, { mode: 0o600 });
        expected = "U6_EXACT_VALUE_SCHEMA_SURFACE_NOT_CLOSED";
        break;
      }
      case "bytes-schema-slot-drift": {
        const path = join(fixture, "spec/schema/v1/decision-record.schema.json");
        const value = JSON.parse(await readFile(path, "utf8"));
        const rule = value.$defs.ExactValue.allOf.find((entry) => entry.if.properties.tag.const === "BYTES");
        rule.then.properties.text = { $ref: "#/$defs/NonemptyBase64" };
        await writeFile(path, `${JSON.stringify(value)}\n`, { mode: 0o600 });
        expected = "U6_BYTES_COMPATIBILITY_SCHEMA_NOT_EXACT";
        break;
      }
      case "ordered-list-schema-slot-drift": {
        const path = join(fixture, "spec/schema/v1/decision-record.schema.json");
        const value = JSON.parse(await readFile(path, "utf8"));
        const rule = value.$defs.ExactValue.allOf.find((entry) => entry.if.properties.tag.const === "ORDERED_STRING_LIST");
        rule.then.properties.canonical_json_base64 = { $ref: "#/$defs/OptionalBase64" };
        await writeFile(path, `${JSON.stringify(value)}\n`, { mode: 0o600 });
        expected = "U6_ORDERED_LIST_COMPATIBILITY_SCHEMA_NOT_EXACT";
        break;
      }
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
	const cleanControls = new Set([
		"clean",
		"harmless-comment",
		"harmless-astral-comment",
		"c3-store-bridge",
		"c3-store-bridge-raw-import",
		"c3-store-bridge-renamed-identifiers",
		"c3-store-import-camouflage-raw-string",
		"c4-store-bridge",
		"c4-store-bridge-renamed-identifiers",
	]);
	const hostile = observed.filter((name) => !cleanControls.has(name)).length;
	process.stdout.write(`U6 architecture checker self-test OK (${cleanControls.size} clean/comment controls; ${hostile}/${hostile} hostile cases)\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
