#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { copyFile, cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const checker = resolve(root, "tools/check-p07b-a2-architecture.mjs");
const copyPaths = Object.freeze([
	"internal/adapters/cli/model",
	"internal/adapters/cli/projection.go",
	"internal/adapters/http/model",
	"internal/canon",
	"internal/choice/choicepoint.go",
	"internal/choice/promotion/residue.go",
	"internal/choice/promotion/service.go",
	"internal/choice/session_roundtrip_test.go",
	"internal/confirmation/service_test.go",
	"internal/confirmation/wire.go",
	"internal/contractsource",
	"internal/world/fresh_confirmation.go",
	"internal/domain",
	"internal/emit/node",
	"internal/observe/eligibility.go",
	"internal/observe/eligibilitycore",
	"internal/portablevalue",
	"internal/projectionprofile",
	"internal/runnerprofile",
	"testkit/contracts",
	"testkit/studies/cli_precedence/study_darwin_test.go",
	"testkit/studies/cli_precedence/reduction_darwin_test.go",
	"testkit/studies/http_invoices/reduction_darwin_test.go",
	"tools/generate-p07b-a2-runtime-example.mjs",
	"tools/verify-p07b-a2-recovery-process.mjs",
	"tools/capture-p07b-a2-human-surface.mjs",
]);

const roster = Object.freeze([
	["transitive-forbidden-dependency", "P07B_A2_PURE_TRANSITIVE_FORBIDDEN_DEPENDENCY"],
	["generic-constructor", "P07B_A2_GENERIC_INPUT_CONSTRUCTOR"],
	["comment-string-spoof", "P07B_A2_RETRANSLATION_ANCHOR_MISSING"],
	["candidate-field", "P07B_A2_INPUT_AUTHORITY_LEAK"],
	["store-write", "P07B_A2_STORE_WRITE"],
	["remove-retranslation", "P07B_A2_RETRANSLATION_ANCHOR_MISSING"],
	["prepared-authority-return", "P07B_A2_PREPARED_AUTHORITY_EXPOSED"],
	["snapshot-head-field", "P07B_A2_SNAPSHOT_AUTHORITY_DRIFT"],
	["confirmation-roster-bypass", "P07B_A2_CONFIRMATION_ROSTER_NOT_SELF_VALIDATING"],
	["live-foreign-literal", "P07B_A2_PREMATURE_OR_FOREIGN_AUTHORITY"],
	["dead-helper-retranslation", "P07B_A2_RETRANSLATION_ANCHOR_MISSING"],
	["remove-cli-fresh-process", "P07B_A2_FRESH_PROCESS_RESTART_MISSING"],
	["ambient-http-restart-environment", "P07B_A2_FRESH_PROCESS_PARENT_INCOMPLETE"],
	["input-wire-canonical-member", "P07B_A2_INPUT_WIRE_DRIFT"],
	["predicate-wire-canonical-member", "P07B_A2_PREDICATE_WIRE_DRIFT"],
	["input-constructor-extra-argument", "P07B_A2_INPUT_CONSTRUCTOR_SIGNATURE"],
	["node-authority-function", "P07B_A2_NODE_TOP_LEVEL_API_DRIFT"],
	["premature-compiler-function", "P07B_A2_NODE_TOP_LEVEL_API_DRIFT"],
	["snapshot-store-write", "P07B_A2_SNAPSHOT_STORE_WRITE"],
	["unused-retranslation-result", "P07B_A2_RETRANSLATION_DATAFLOW"],
	["source-byte-join-bypass", "P07B_A2_SOURCE_JOIN_DATAFLOW"],
	["raw-service-import", "P07B_A2_SERVICE_IMPORT_ROSTER"],
	["append-ambient-after-closed-env", "P07B_A2_FRESH_PROCESS_ENVIRONMENT_DRIFT"],
	["cli-tool-ignore-explicit", "P07B_A2_CLI_TOOL_AUTHORITY_MISSING"],
	["fresh-inspection-retained-reuse", "P07B_A2_SNAPSHOT_FRESH_INSPECTION_DATAFLOW"],
	["prepared-profile-byte-bypass", "P07B_A2_PREPARED_PROFILE_BYTE_JOIN"],
	["compilation-exported-var-alias", "P07B_A2_COMPILATION_EXPORTED_DECLARATION_DRIFT"],
	["node-exported-constructor-alias", "P07B_A2_NODE_EXPORTED_DECLARATION_DRIFT"],
	["node-exported-type-alias", "P07B_A2_NODE_EXPORTED_DECLARATION_DRIFT"],
	["node-exported-const-alias", "P07B_A2_NODE_EXPORTED_DECLARATION_DRIFT"],
	["promotion-sibling-source", "P07B_A2_PROMOTION_PRODUCTION_MAP_DRIFT"],
	["snapshot-indirect-store-write", "P07B_A2_SNAPSHOT_CALL_GRAPH_DRIFT"],
	["compilation-multiname-exported-var-alias", "P07B_A2_COMPILATION_EXPORTED_DECLARATION_DRIFT"],
	["promotion-multiname-snapshot-alias", "P07B_A2_PROMOTION_EXPORTED_DECLARATION_DRIFT"],
	["promotion-exported-function", "P07B_A2_PROMOTION_TOP_LEVEL_API_DRIFT"],
	["promotion-grouped-multiname-snapshot-alias", "P07B_A2_PROMOTION_EXPORTED_DECLARATION_DRIFT"],
	["promotion-unicode-exported-alias", "P07B_A2_PROMOTION_EXPORTED_DECLARATION_DRIFT"],
	["promotion-receiver-method", "P07B_A2_PROMOTION_EXPORTED_METHOD_DRIFT"],
	["prepared-profile-short-circuit", "P07B_A2_PREPARED_PROFILE_BYTE_JOIN"],
	["fresh-inspection-short-circuit", "P07B_A2_SNAPSHOT_FRESH_INSPECTION_DATAFLOW"],
	["fresh-inspection-overwrite", "P07B_A2_SNAPSHOT_FRESH_INSPECTION_DATAFLOW"],
	["prepared-profile-overwrite", "P07B_A2_PREPARED_PROFILE_BYTE_JOIN"],
	["snapshot-valid-short-circuit", "P07B_A2_SNAPSHOT_PROFILE_VALID_DATAFLOW"],
	["confirmation-roster-short-circuit", "P07B_A2_CONFIRMATION_ROSTER_NOT_SELF_VALIDATING"],
	["confirmation-binding-roster-truncation", "P07B_A2_CONFIRMATION_BINDING_ACCESSOR_MISSING"],
	["node-refusal-literal-drift", "P07B_A2_NODE_AST_DRIFT"],
	["promotion-refusal-literal-drift", "P07B_A2_PROMOTION_AST_DRIFT"],
	["compilation-digest-mislabel", "P07B_A2_COMPILATION_AST_DRIFT"],
	["prepared-digest-mislabel", "P07B_A2_NODE_AST_DRIFT"],
	["compilation-predicate-wire-omission", "P07B_A2_COMPILATION_AST_DRIFT"],
	["predicate-tuple-wire-omission", "P07B_A2_MODEL_AST_DRIFT"],
	["choice-minimized-getter-drift", "P07B_A2_CHOICE_ACCESSOR_DATAFLOW"],
	["world-binding-getter-drift", "P07B_A2_WORLD_BINDING_ACCESSOR_DATAFLOW"],
	["noop-required-authority-getter-test", "P07B_A2_TEST_AST_DRIFT"],
	["add-build-tag-to-required-test", "P07B_A2_TEST_AST_DRIFT"],
	["change-physical-build-tag", "P07B_A2_TEST_AST_DRIFT"],
	["add-build-tag-to-production", "P07B_A2_CHOICE_AST_DRIFT"],
	["compiler-os-import", "P07B_A2_PURE_TRANSITIVE_FORBIDDEN_DEPENDENCY"],
	["compiler-store-import", "P07B_A2_PURE_TRANSITIVE_INTERNAL_DEPENDENCY"],
	["prepared-bundle-authority-getter", "P07B_A2_PREPARED_BUNDLE_API_DRIFT"],
	["bundle-self-digest", "P07B_A2_BUNDLE_SELF_DIGEST"],
	["manifest-parse-includes-self", "P07B_A2_BUNDLE_RECOVERY_DATAFLOW"],
	["manifest-self-coverage-bypass", "P07B_A2_MANIFEST_SELF_COVERAGE"],
	["asset-byte-drift", "P07B_A2_FIXED_ASSET_DIGEST"],
	["asset-pin-drift", "P07B_A2_FIXED_ASSET_GO_PIN"],
	["asset-validation-bypass", "P07B_A2_FIXED_ASSET_VALIDATION_DATAFLOW"],
	["asset-self-derived-raw-digest", "P07B_A2_FIXED_ASSET_SELF_DERIVED"],
	["asset-extra-embed", "P07B_A2_FIXED_ASSET_EMBED_ROSTER"],
	["js-static-import-drift", "P07B_A2_JS_IMPORT_ROSTER"],
	["js-second-dynamic-edge", "P07B_A2_JS_DYNAMIC_EDGE"],
	["js-import-before-verification", "P07B_A2_ENTRYPOINT_VERIFY_BEFORE_IMPORT"],
	["js-high-level-http-import", "P07B_A2_JS_IMPORT_ROSTER"],
	["js-fetch-capability", "P07B_A2_JS_CAPABILITY_OPENING"],
	["js-shell-execution", "P07B_A2_JS_SPAWN_PROFILE"],
	["js-ambient-environment", "P07B_A2_JS_CAPABILITY_OPENING"],
	["js-direct-target-root", "P07B_A2_DIRECT_TARGET_EXECUTION"],
	["js-production-fault-hook", "P07B_A2_PRODUCTION_FAULT_HOOK"],
	["go-request-expected-field", "P07B_A2_PARITY_REQUEST_ROSTER"],
	["go-evaluator-corpus-authority-reference", "P07B_A2_PARITY_CORPUS_ACCESS"],
	["runner-expected-access", "P07B_A2_PARITY_RUNNER_CAPABILITY"],
	["node-evaluator-expected-read", "P07B_A2_PARITY_ORACLE_ACCESS"],
	["node-evaluator-case-id-switch", "P07B_A2_PARITY_ORACLE_ACCESS"],
	["node-evaluator-corpus-authority-reference", "P07B_A2_PARITY_CORPUS_ACCESS"],
	["js-extra-export", "P07B_A2_JS_EXPORT_ROSTER"],
	["shared-cli-owner-bypass", "P07B_A2_SHARED_CLI_PROJECTION_OWNER"],
	["shared-eligibility-owner-bypass", "P07B_A2_SHARED_ELIGIBILITY_OWNER"],
	["direct-cancellation-insertion", "P07B_A2_CANCELLATION_JURISDICTION"],
	["runner-ambient-environment", "P07B_A2_PARITY_RUNNER_CAPABILITY"],
	["package-map-sibling", "P07B_A2_PACKAGE_MAP_DRIFT"],
	["pure-unlisted-stdlib-import", "P07B_A2_PURE_IMPORT_EDGE_ROSTER"],
	["manifest-remove-self-exclusion", "P07B_A2_MANIFEST_SELF_COVERAGE"],
	["asset-early-reviewed-return", "P07B_A2_FIXED_ASSET_VALIDATION_DATAFLOW"],
	["asset-wrong-reviewed-return", "P07B_A2_FIXED_ASSET_SELF_DERIVED"],
	["asset-embed-fs", "P07B_A2_FIXED_ASSET_EMBED_ROSTER"],
	["js-existing-module-capability-import", "P07B_A2_JS_IMPORT_ROSTER"],
	["runner-powerful-harness-import", "P07B_A2_PARITY_RUNNER_CAPABILITY"],
	["node-evaluator-arrow-oracle-read", "P07B_A2_PARITY_ORACLE_ACCESS"],
	["node-evaluator-destructured-oracle-read", "P07B_A2_PARITY_ORACLE_ACCESS"],
	["js-indirect-import-before-verification", "P07B_A2_ENTRYPOINT_VERIFY_BEFORE_IMPORT"],
	["js-spawn-options-spread", "P07B_A2_JS_SPAWN_PROFILE"],
	["js-spawn-alias", "P07B_A2_JS_SPAWN_PROFILE"],
	["js-global-fetch-capability", "P07B_A2_JS_CAPABILITY_OPENING"],
	["probe-package-map-sibling", "P07B_A2_PACKAGE_MAP_DRIFT"],
	["compiler-probe-import-drift", "P07B_A2_PROBE_IMPORT_ROSTER"],
	["compiler-probe-dataflow-drift", "P07B_A2_COMPILER_PROBE_AST_DRIFT"],
	["parser-probe-compiler-import", "P07B_A2_PROBE_IMPORT_ROSTER"],
	["parser-probe-schema-drift", "P07B_A2_PARSER_PROBE_AST_DRIFT"],
	["readme-warning-drift", "P07B_A2_BUNDLE_AST_DRIFT"],
	["readme-test-noop", "P07B_A2_TEST_AST_DRIFT"],
	["example-generator-byte-drift", "P07B_A2_EXAMPLE_GENERATOR_DIGEST"],
	["recovery-verifier-byte-drift", "P07B_A2_RECOVERY_VERIFIER_DIGEST"],
	["human-capture-byte-drift", "P07B_A2_HUMAN_CAPTURE_DIGEST"],
	["human-capture-watchdog-removal", "P07B_A2_CAPTURE_WATCHDOG"],
	["human-capture-bundle-binding-removal", "P07B_A2_CAPTURE_BUNDLE_BINDING"],
	["human-capture-open-tap", "P07B_A2_CAPTURE_CLOSED_TAP"],
	["human-capture-control-roster-weakening", "P07B_A2_CAPTURE_CONTROL_REJECTION"],
	["human-capture-scenario-metadata-removal", "P07B_A2_CAPTURE_SCENARIO_METADATA"],
]);
const expectedRosterDigest = "sha256:832d98ce1e73056e3133a9b6cc31d1ef3da4c88a451f85f122d35ab38b05c80e";

function rosterDigest() {
	const hash = createHash("sha256");
	for (const [id, code] of roster) hash.update(id).update("\0").update(code).update("\0");
	return `sha256:${hash.digest("hex")}`;
}

async function copyFixture(label) {
	const directory = await mkdtemp(join(process.env.TMPDIR ?? "/tmp", `countershape-a2-arch-${label}-`));
	for (const path of copyPaths) {
		await mkdir(dirname(join(directory, path)), { recursive: true });
		await cp(join(root, path), join(directory, path), { recursive: true, errorOnExist: true, dereference: false });
	}
	return directory;
}

async function replaceRequired(directory, path, before, after) {
	const absolute = join(directory, path);
	const source = await readFile(absolute, "utf8");
	if (!source.includes(before)) throw new Error(`self-test precondition missing: ${path}:${before}`);
	await writeFile(absolute, source.replace(before, after));
}

function run(directory) {
	const paths = {};
	for (const name of ["HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"]) {
		const value = process.env[name];
		if (!value) throw new Error(`P07B_A2_SELFTEST_ENVIRONMENT_REQUIRED: ${name}`);
		paths[name] = value;
	}
	return spawnSync(process.execPath, [checker], {
		cwd: root, encoding: "utf8", timeout: 90_000, maxBuffer: 8 * 1024 * 1024,
		env: {
			...paths, PATH: `${dirname(process.execPath)}:/usr/bin:/bin`, LANG: "C", LC_ALL: "C", TZ: "UTC",
			GOENV: "off", GOWORK: "off", GOTOOLCHAIN: "local", GOPROXY: "off", GOSUMDB: "off",
			GOFLAGS: "-mod=readonly -buildvcs=false -p=1", CGO_ENABLED: "0", GOMAXPROCS: "2",
			COUNTERSHAPE_A2_SOURCE_ROOT: directory, COUNTERSHAPE_A2_SELFTEST: "1",
			COUNTERSHAPE_GO: process.env.COUNTERSHAPE_GO, NO_COLOR: "1",
		},
	});
}

async function cleanControl() {
	const directory = await copyFixture("clean");
	try {
		const result = run(directory);
		if (result.status !== 0 || !result.stdout.includes("P07B A2.2 architecture fixture OK") ||
			result.stdout.includes("P07B A2.2 architecture boundary OK")) {
			throw new Error(`clean control failed: ${result.stderr || result.stdout}`);
		}
	} finally {
		await rm(directory, { recursive: true, force: true });
	}
}

async function commentLiteralControl() {
	const directory = await copyFixture("literal-control");
	try {
		const path = "internal/emit/node/internal/compilation/input.go";
		const source = await readFile(join(directory, path), "utf8");
		await writeFile(join(directory, path), `${source}\n\n`);
		const result = run(directory);
		if (result.status !== 0) throw new Error(`comment/string clean control failed: ${result.stderr || result.stdout}`);
	} finally {
		await rm(directory, { recursive: true, force: true });
	}
}

async function mutate(id, expectedCode) {
	const directory = await copyFixture(id);
	try {
		switch (id) {
		case "transitive-forbidden-dependency":
			await mkdir(join(directory, "internal/runnerprofile"), { recursive: true });
			await copyFile(join(root, "internal/runnerprofile/profile.go"), join(directory, "internal/runnerprofile/profile.go"));
			await replaceRequired(directory, "internal/runnerprofile/profile.go", '"strings"', '"strings"\n\t_ "os"');
			break;
		case "generic-constructor":
			await replaceRequired(directory, "internal/emit/node/internal/compilation/input.go", "// New is the only sanitized-input constructor.", "func (Input) NewFromWire(raw []byte) Input { return Input{} }\n\n// New is the only sanitized-input constructor.");
			break;
		case "comment-string-spoof":
			await replaceRequired(directory, "internal/emit/node/service.go", "translations, err := projectiontranslate.TranslateConfirmed(", "// projectiontranslate.TranslateConfirmed P07B_A2_EXPLICIT_RETRANSLATION_ANCHOR\n\tmarker := \"projectiontranslate.TranslateConfirmed P07B_A2_EXPLICIT_RETRANSLATION_ANCHOR\"\n\t_ = marker\n\ttranslations, err := projectiontranslate.MissingTranslateConfirmed(");
			await replaceRequired(directory, "internal/emit/node/service.go", ") // P07B_A2_EXPLICIT_RETRANSLATION_ANCHOR", ")");
			break;
		case "candidate-field":
			await replaceRequired(directory, "internal/emit/node/internal/compilation/input.go", "type Input struct {", "type Input struct {\n\t*candidateAuthority");
			break;
		case "store-write":
			await replaceRequired(directory, "internal/emit/node/service.go", "snapshot, err := promotion.OpenPortableCompilationSnapshot", "persistA2(ctx, objectStore)\n\tsnapshot, err := promotion.OpenPortableCompilationSnapshot");
			await replaceRequired(directory, "internal/emit/node/service.go", "func requireSourceChoicepointJoin", "func persistA2(ctx context.Context, objectStore *store.ObjectStore) { _, _ = objectStore.Publish(ctx, store.SemanticObject{}) }\n\nfunc requireSourceChoicepointJoin");
			break;
		case "remove-retranslation":
			await replaceRequired(directory, "internal/emit/node/service.go", "projectiontranslate.TranslateConfirmed", "projectiontranslate.MissingTranslateConfirmed");
			await replaceRequired(directory, "internal/emit/node/service.go", " // P07B_A2_EXPLICIT_RETRANSLATION_ANCHOR", "");
			break;
		case "prepared-authority-return":
			await replaceRequired(directory, "internal/emit/node/service.go", "func (p PreparedCompilation) Valid() bool {", "func (p PreparedCompilation) Authority() promotion.PortableRulingPreparation { return p.preparation }\n\nfunc (p PreparedCompilation) Valid() bool {");
			break;
		case "snapshot-head-field":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "type PortableCompilationSnapshot struct {", "type PortableCompilationSnapshot struct {\n\thead store.HeadToken");
			break;
		case "confirmation-roster-bypass":
			await replaceRequired(directory, "internal/confirmation/wire.go", "sameDigestsInOrder(parsed.executionBindings, r.executionBindings)", "true");
			break;
		case "live-foreign-literal": {
			const path = "internal/emit/node/internal/compilation/input.go";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `${source}\nvar foreignArtifact = "ContractBundle"\n`);
			break;
		}
		case "dead-helper-retranslation":
			await replaceRequired(directory, "internal/emit/node/service.go", "projectiontranslate.TranslateConfirmed(", "projectiontranslate.MissingTranslateConfirmed(");
			await replaceRequired(directory, "internal/emit/node/service.go", "func requireSourceChoicepointJoin", "func deadRetranslationAnchor() { projectiontranslate.TranslateConfirmed() }\n\nfunc requireSourceChoicepointJoin");
			break;
		case "remove-cli-fresh-process":
			await replaceRequired(directory, "testkit/studies/cli_precedence/reduction_darwin_test.go", "func TestCLICompilationFreshProcessRestartHelper", "func hiddenCLICompilationRestartHelper");
			break;
		case "ambient-http-restart-environment":
			await replaceRequired(directory, "testkit/studies/http_invoices/reduction_darwin_test.go", 'command.Env = []string{httpA21RestartEnv + "=" + requestPath}', 'command.Env = append(os.Environ(), httpA21RestartEnv+"="+requestPath)');
			break;
		case "input-wire-canonical-member":
			await replaceRequired(directory, "internal/emit/node/internal/compilation/input.go", "type inputWire struct {", 'type inputWire struct {\n\tCandidateProofBase64 string `json:"candidate_proof_base64"`');
			break;
		case "predicate-wire-canonical-member":
			await replaceRequired(directory, "internal/emit/node/model/predicate.go", "type predicateWire struct {", 'type predicateWire struct {\n\tCandidateAlias string `json:"candidate_alias"`');
			break;
		case "input-constructor-extra-argument":
			await replaceRequired(directory, "internal/emit/node/internal/compilation/input.go", "\tpredicate model.Predicate,\n) (Input, error) {", "\tpredicate model.Predicate,\n\tcandidateProof []byte,\n) (Input, error) {");
			break;
		case "node-authority-function":
			await replaceRequired(directory, "internal/emit/node/service.go", "func requireSourceChoicepointJoin", "func Authority(p PreparedCompilation) promotion.PortableRulingPreparation { return p.preparation }\n\nfunc requireSourceChoicepointJoin");
			break;
		case "premature-compiler-function":
			await replaceRequired(directory, "internal/emit/node/service.go", "func requireSourceChoicepointJoin", 'func Compile(p PreparedCompilation) []byte { return []byte("premature") }\n\nfunc requireSourceChoicepointJoin');
			break;
		case "snapshot-store-write":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "\tif ctx == nil || objectStore == nil || !preparation.Valid() {", "\t_, _ = objectStore.Publish(ctx, store.SemanticObject{})\n\tif ctx == nil || objectStore == nil || !preparation.Valid() {");
			break;
		case "unused-retranslation-result":
			await replaceRequired(directory, "internal/emit/node/service.go", "revalidatePartition(decision, translations, exactSource)", "revalidatePartition(decision, projectiontranslate.ConfirmedTranslations{}, exactSource)");
			break;
		case "source-byte-join-bypass":
			await replaceRequired(directory, "internal/emit/node/service.go", "!bytes.Equal(plan.CanonicalBytes(), sourcePlan.CanonicalBytes())", "false");
			break;
		case "raw-service-import":
			await replaceRequired(directory, "internal/emit/node/service.go", '\t"fmt"', '\t"fmt"\n\t_ `syscall`');
			break;
		case "append-ambient-after-closed-env":
			await replaceRequired(directory, "testkit/studies/cli_precedence/reduction_darwin_test.go", 'command.Env = []string{cliA21RestartEnv + "=" + requestPath}', 'command.Env = []string{cliA21RestartEnv + "=" + requestPath}\n\tcommand.Env = append(command.Env, os.Environ()...)');
			break;
		case "cli-tool-ignore-explicit":
			await replaceRequired(directory, "testkit/studies/cli_precedence/study_darwin_test.go", 'path := os.Getenv("COUNTERSHAPE_" + strings.ToUpper(name))', 'path := ""');
			break;
		case "fresh-inspection-retained-reuse":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "freshInspection, err := choice.InspectPortableRuling(decision)", "freshInspection := preparation.inspection");
			break;
		case "prepared-profile-byte-bypass":
			await replaceRequired(directory, "internal/emit/node/service.go", "!bytes.Equal(preparedProfile.CanonicalBytes(), translatedProfile.CanonicalBytes())", "false");
			break;
		case "compilation-exported-var-alias":
			await replaceRequired(directory, "internal/emit/node/internal/compilation/input.go", "// New is the only sanitized-input constructor.", "var InputConstructor = New\n\n// New is the only sanitized-input constructor.");
			break;
		case "node-exported-constructor-alias":
			await replaceRequired(directory, "internal/emit/node/service.go", "func PrepareCompilation(", "var InputConstructor = compilation.New\n\nfunc PrepareCompilation(");
			break;
		case "node-exported-type-alias":
			await replaceRequired(directory, "internal/emit/node/service.go", "type PreparedCompilation struct {", "type AuthorityAlias = promotion.PortableRulingPreparation\n\ntype PreparedCompilation struct {");
			break;
		case "node-exported-const-alias":
			await replaceRequired(directory, "internal/emit/node/service.go", "type Error struct {", 'const AuthorityMode = "UNSAFE"\n\ntype Error struct {');
			break;
		case "promotion-sibling-source":
			await writeFile(join(directory, "internal/choice/promotion/sibling.go"), "package promotion\n\nfunc SnapshotAlias() PortableCompilationSnapshot { return PortableCompilationSnapshot{} }\n");
			break;
		case "snapshot-indirect-store-write":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "\tif ctx == nil || objectStore == nil || !preparation.Valid() {", "\tsnapshotSideEffect(ctx, objectStore)\n\tif ctx == nil || objectStore == nil || !preparation.Valid() {");
			await replaceRequired(directory, "internal/choice/promotion/service.go", "func mapSnapshotCurrentness", "func snapshotSideEffect(ctx context.Context, objectStore *store.ObjectStore) { _, _ = objectStore.Publish(ctx, store.SemanticObject{}) }\n\nfunc mapSnapshotCurrentness");
			break;
		case "compilation-multiname-exported-var-alias":
			await replaceRequired(directory, "internal/emit/node/internal/compilation/input.go", "// New is the only sanitized-input constructor.", "var _, InputConstructor = 0, New\n\n// New is the only sanitized-input constructor.");
			break;
		case "promotion-multiname-snapshot-alias":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "func OpenPortableCompilationSnapshot(", "var _, SnapshotOpener = 0, OpenPortableCompilationSnapshot\n\nfunc OpenPortableCompilationSnapshot(");
			break;
		case "promotion-exported-function":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "func mapSnapshotCurrentness", "func ExtraPromotionAPI() {}\n\nfunc mapSnapshotCurrentness");
			break;
		case "promotion-grouped-multiname-snapshot-alias":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "func OpenPortableCompilationSnapshot(", "var (\n\tsnapshotOpenerSentinel, SnapshotOpener = 0, OpenPortableCompilationSnapshot\n)\n\nfunc OpenPortableCompilationSnapshot(");
			break;
		case "promotion-unicode-exported-alias":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "func OpenPortableCompilationSnapshot(", "var σnapshotOpenerSentinel, ΣnapshotOpener = 0, OpenPortableCompilationSnapshot\n\nfunc OpenPortableCompilationSnapshot(");
			break;
		case "promotion-receiver-method":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "func mapSnapshotCurrentness", "func (Ruling) SnapshotOpener() any { return OpenPortableCompilationSnapshot }\n\nfunc mapSnapshotCurrentness");
			break;
		case "prepared-profile-short-circuit":
			await replaceRequired(directory, "internal/emit/node/service.go", "!bytes.Equal(preparedProfile.CanonicalBytes(), translatedProfile.CanonicalBytes()) ||", "false && !bytes.Equal(preparedProfile.CanonicalBytes(), translatedProfile.CanonicalBytes()) ||");
			break;
		case "fresh-inspection-short-circuit":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "freshInspection.ProfileDigest() != preparation.inspection.ProfileDigest() ||", "false && freshInspection.ProfileDigest() != preparation.inspection.ProfileDigest() ||");
			break;
		case "fresh-inspection-overwrite":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "freshInspection, err := choice.InspectPortableRuling(decision)", "freshInspection, err := choice.InspectPortableRuling(decision)\n\tfreshInspection = preparation.inspection");
			break;
		case "prepared-profile-overwrite":
			await replaceRequired(directory, "internal/emit/node/service.go", "preparedProfile := snapshot.PortableProfile()", "preparedProfile := snapshot.PortableProfile()\n\tpreparedProfile = translatedProfile");
			break;
		case "snapshot-valid-short-circuit":
			await replaceRequired(directory, "internal/choice/promotion/service.go", "!bytes.Equal(reparsedProfile.CanonicalBytes(), s.profile.CanonicalBytes()) {", "false && !bytes.Equal(reparsedProfile.CanonicalBytes(), s.profile.CanonicalBytes()) {");
			break;
		case "confirmation-roster-short-circuit":
			await replaceRequired(directory, "internal/confirmation/wire.go", "sameDigestsInOrder(parsed.executionBindings, r.executionBindings)", "(true || sameDigestsInOrder(parsed.executionBindings, r.executionBindings))");
			break;
		case "confirmation-binding-roster-truncation":
			await replaceRequired(directory, "internal/confirmation/wire.go", "return append([]domain.Digest(nil), r.executionBindings...)", "return append([]domain.Digest(nil), r.executionBindings[:1]...)");
			break;
		case "node-refusal-literal-drift":
			await replaceRequired(directory, "internal/emit/node/service.go", 'CodeSourceRulingMismatch        = "SOURCE_RULING_MISMATCH"', 'CodeSourceRulingMismatch        = "OTHER"');
			break;
		case "promotion-refusal-literal-drift":
			await replaceRequired(directory, "internal/choice/promotion/service.go", 'const CodeStaleChoicepoint = "STALE_CHOICEPOINT"', 'const CodeStaleChoicepoint = "OTHER"');
			break;
		case "compilation-digest-mislabel":
			await replaceRequired(directory, "internal/emit/node/internal/compilation/input.go", "func (i Input) Digest() domain.Digest               { return i.digest }", "func (i Input) Digest() domain.Digest               { return i.decision }");
			break;
		case "prepared-digest-mislabel":
			await replaceRequired(directory, "internal/emit/node/service.go", "func (p PreparedCompilation) Digest() domain.Digest { return p.input.Digest() }", "func (p PreparedCompilation) Digest() domain.Digest { return p.input.DecisionRecordDigest() }");
			break;
		case "compilation-predicate-wire-omission":
			await replaceRequired(directory, "internal/emit/node/internal/compilation/input.go", "PredicateBase64:      base64.StdEncoding.EncodeToString(predicate.CanonicalBytes()),", 'PredicateBase64:      "",');
			break;
		case "predicate-tuple-wire-omission":
			await replaceRequired(directory, "internal/emit/node/model/predicate.go", "wire.AllowedTuples[index] = tupleIdentity", "_ = tupleIdentity");
			break;
		case "choice-minimized-getter-drift":
			await replaceRequired(directory, "internal/choice/choicepoint.go", "r.minimized.Kind(), r.minimized.Digest(), r.minimized.CanonicalBytes()", "r.original.Kind(), r.original.Digest(), r.original.CanonicalBytes()");
			break;
		case "world-binding-getter-drift":
			await replaceRequired(directory, "internal/world/fresh_confirmation.go", "return r.fact.executionBindingDigest", "return r.fact.invocationFileDigest");
			break;
		case "noop-required-authority-getter-test":
			await replaceRequired(directory, "internal/choice/session_roundtrip_test.go", "func TestChoicepointAuthorityGettersAreExactAndDefensiveForLegacyAndPortableRecords(t *testing.T) {", "func TestChoicepointAuthorityGettersAreExactAndDefensiveForLegacyAndPortableRecords(t *testing.T) {\n\treturn");
			break;
		case "add-build-tag-to-required-test": {
			const path = "internal/choice/session_roundtrip_test.go";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `//go:build ignore\n\n${source}`);
			break;
		}
		case "change-physical-build-tag":
			await replaceRequired(directory, "testkit/studies/cli_precedence/reduction_darwin_test.go", "//go:build darwin && cgo", "//go:build ignore");
			break;
		case "add-build-tag-to-production": {
			const path = "internal/choice/choicepoint.go";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `//go:build ignore\n\n${source}`);
			break;
		}
		case "compiler-os-import":
			await replaceRequired(directory, "internal/emit/node/compiler/compiler.go", '"fmt"', '"fmt"\n\t_ "os"');
			break;
		case "compiler-store-import":
			await replaceRequired(directory, "internal/emit/node/compiler/compiler.go", '"fmt"', '"fmt"\n\t_ "github.com/nelsonwerd/countershape/internal/store"');
			break;
		case "prepared-bundle-authority-getter":
			await replaceRequired(directory, "internal/emit/node/service.go", "func (p PreparedBundle) Valid() bool {", "func (p PreparedBundle) Authority() PreparedCompilation { return p.prepared }\n\nfunc (p PreparedBundle) Valid() bool {");
			break;
		case "bundle-self-digest":
			await replaceRequired(directory, "internal/emit/node/model/bundle.go", '"external_service_binding", "determinism_profile",', '"external_service_binding", "bundle_digest", "determinism_profile",');
			break;
		case "manifest-parse-includes-self":
			await replaceRequired(directory, "internal/emit/node/model/bundle.go", 'parseIntegrityManifest(fileByPath["manifest.json"].content, files[:5])', 'parseIntegrityManifest(fileByPath["manifest.json"].content, files)');
			break;
		case "manifest-self-coverage-bypass":
			await replaceRequired(directory, "internal/emit/node/model/bundle.go", "len(protected) != 5", "len(protected) != 6");
			break;
		case "asset-byte-drift": {
			const path = "internal/emit/node/program/v1/contract.test.mjs";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `${source}// hostile reviewed-byte drift\n`);
			break;
		}
		case "asset-pin-drift":
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", "sha256:0ded06ad24d9b218fef7776835b26124483d788f2eb49f69d0ffb9806c85e021", "sha256:1ded06ad24d9b218fef7776835b26124483d788f2eb49f69d0ffb9806c85e021");
			break;
		case "asset-validation-bypass":
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", "if err := verifyReviewedDigest(source, reviewed); err != nil {", "if err := error(nil); err != nil {");
			break;
		case "asset-self-derived-raw-digest":
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", "_, reviewed, err := validatedAsset(path)", "source, _, err := validatedAsset(path)");
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", "\treturn reviewed, nil\n}\n\nfunc validatedAsset", "\tdigest := sha256.Sum256(source)\n\treturn \"sha256:\" + hex.EncodeToString(digest[:]), nil\n}\n\nfunc validatedAsset");
			break;
		case "asset-extra-embed":
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", "\t//go:embed harness.mjs\n\tharnessSource []byte", "\t//go:embed harness.mjs\n\tharnessSource []byte\n\n\t//go:embed harness.mjs\n\tduplicateHarnessSource []byte");
			break;
		case "js-static-import-drift":
			await replaceRequired(directory, "internal/emit/node/program/v1/contract.test.mjs", 'from "node:url"', 'from "node:util"');
			break;
		case "js-second-dynamic-edge":
			await replaceRequired(directory, "internal/emit/node/program/v1/contract.test.mjs", 'const harness = await import("./harness.mjs");', 'const harness = await import("./harness.mjs");\n    await import("./harness.mjs");');
			break;
		case "js-import-before-verification":
			await replaceRequired(directory, "internal/emit/node/program/v1/contract.test.mjs", "const verified = new Map();", 'const harness = await import("./harness.mjs");\n    const verified = new Map();');
			await replaceRequired(directory, "internal/emit/node/program/v1/contract.test.mjs", '    const harness = await import("./harness.mjs");\n    result = await harness.runContract({', "    result = await harness.runContract({");
			break;
		case "js-high-level-http-import":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", 'import { spawn } from "node:child_process";', 'import { spawn } from "node:child_process";\nimport { request } from "node:http";');
			break;
		case "js-fetch-capability":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "export async function runContract({ bundleRoot, targetRoot, decisionBytes, fixtureBytes }) {", 'export async function runContract({ bundleRoot, targetRoot, decisionBytes, fixtureBytes }) {\n  fetch("https://example.invalid");');
			break;
		case "js-shell-execution":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "env: environment, shell: false, detached: true,", "env: environment, shell: true, detached: true,");
			break;
		case "js-ambient-environment":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "cwd: roots.candidate, env: environment, shell: false,", "cwd: roots.candidate, env: process.env, shell: false,");
			break;
		case "js-direct-target-root":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "async function runCLI(contract, roots, environment, facts) {", "async function runCLI(contract, roots, environment, facts, targetRoot) {");
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "cwd: roots.candidate, env: environment, shell: false,", "cwd: targetRoot, env: environment, shell: false,");
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "await runCLI(contract, roots, environment, facts);", "await runCLI(contract, roots, environment, facts, targetRoot);");
			break;
		case "js-production-fault-hook":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", 'import { spawn } from "node:child_process";', 'import { spawn } from "node:child_process";\nconst faultHook = null;');
			break;
		case "go-request-expected-field":
			await replaceRequired(directory, "internal/emit/node/parity/types.go", "type Request struct {", "type Request struct {\n\texpected canon.Value");
			break;
		case "go-evaluator-corpus-authority-reference": {
			const path = "internal/emit/node/parity/operations.go";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `${source}\nvar corpus = "spec/vectors/v1/contract-parity.jsonl"\n`);
			break;
		}
		case "runner-expected-access":
			await replaceRequired(directory, "internal/emit/node/parity/runner.mjs", "await writeOneFrame(evaluateParityOperation(request));", "request.expected;\n  await writeOneFrame(evaluateParityOperation(request));");
			break;
		case "node-evaluator-expected-read":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "export function evaluateParityOperation(request) {\n  try {", "export function evaluateParityOperation(request) {\n  request.expected;\n  try {");
			break;
		case "node-evaluator-case-id-switch":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "export function evaluateParityOperation(request) {\n  try {", 'export function evaluateParityOperation(request) {\n  if (request.id === "hostile") return null;\n  try {');
			break;
		case "node-evaluator-corpus-authority-reference":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "export function evaluateParityOperation(request) {\n  try {", 'export function evaluateParityOperation(request) {\n  "spec/vectors/v1/contract-parity.jsonl";\n  try {');
			break;
		case "js-extra-export": {
			const path = "internal/emit/node/program/v1/contract.test.mjs";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `${source}export const hostileDiagnostic = true;\n`);
			break;
		}
		case "shared-cli-owner-bypass":
			await replaceRequired(directory, "internal/adapters/cli/projection.go", "behavior, rejection := d.projectEligibleBehavior(\n\t\tinput.Completion.value,\n\t\tappend([]byte(nil), input.Stdout...),\n\t\tappend([]byte(nil), input.Stderr...),\n\t)", "behavior := cliBehaviorProjection{}\n\tvar rejection *ProjectionRejection");
			break;
		case "shared-eligibility-owner-bypass": {
			await replaceRequired(directory, "internal/emit/node/parity/operations.go", "decision, err := eligibilitycore.Select(eligibilitycore.FactKind(kindText), reasons)", "decision, err := selectOwnerEligibilityBypass(eligibilitycore.FactKind(kindText), reasons)");
			const path = "internal/emit/node/parity/operations.go";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `${source}\nfunc selectOwnerEligibilityBypass(kind eligibilitycore.FactKind, reasons []domain.ControlReason) (eligibilitycore.Decision, error) {\n\treturn eligibilitycore.Decision{}, nil\n}\n`);
			break;
		}
		case "direct-cancellation-insertion":
			await replaceRequired(directory, "internal/emit/node/model/result.go", 'DirectOrphanRisk             DirectReason = "ORPHAN_RISK"', 'DirectCancelled              DirectReason = "CANCELLED"\n\tDirectOrphanRisk             DirectReason = "ORPHAN_RISK"');
			break;
		case "runner-ambient-environment":
			await replaceRequired(directory, "internal/emit/node/parity/runner.mjs", "try {\n  const request", "try {\n  process.env;\n  const request");
			break;
		case "package-map-sibling":
			await writeFile(join(directory, "internal/emit/node/parity/hostile.go"), "package parity\n");
			break;
		case "pure-unlisted-stdlib-import":
			await replaceRequired(directory, "internal/emit/node/compiler/compiler.go", '"fmt"', '"fmt"\n\t_ "database/sql"');
			break;
		case "manifest-remove-self-exclusion":
			await replaceRequired(directory, "internal/emit/node/model/bundle.go", 'path != protected[index].path || path == "manifest.json"', "path != protected[index].path");
			break;
		case "asset-early-reviewed-return":
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", "func validatedAsset(path string) ([]byte, string, error) {", "func validatedAsset(path string) ([]byte, string, error) {\n\tif path == ContractTestPath { return contractTestSource, contractTestReviewedRawSHA256, nil }");
			break;
		case "asset-wrong-reviewed-return":
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", "\treturn reviewed, nil\n}\n\nfunc validatedAsset", "\treturn contractTestReviewedRawSHA256, nil\n}\n\nfunc validatedAsset");
			break;
		case "asset-embed-fs":
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", '_ "embed"', '"embed"');
			await replaceRequired(directory, "internal/emit/node/program/v1/assets.go", "\t//go:embed harness.mjs\n\tharnessSource []byte", "\t//go:embed harness.mjs\n\tharnessSource []byte\n\n\t//go:embed harness.mjs\n\textraAssets embed.FS");
			break;
		case "js-existing-module-capability-import":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", 'import { spawn } from "node:child_process";', 'import { execSync, spawn } from "node:child_process";');
			break;
		case "runner-powerful-harness-import":
			await replaceRequired(directory, "internal/emit/node/parity/runner.mjs", "  parseCanonicalJSON,\n} from", "  parseCanonicalJSON,\n  runContract,\n} from");
			break;
		case "node-evaluator-arrow-oracle-read":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "export function evaluateParityOperation(request) {", "const readOracle = (request) => request.expected;\n\nexport function evaluateParityOperation(request) {\n  readOracle(request);");
			break;
		case "node-evaluator-destructured-oracle-read":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "export function evaluateParityOperation(request) {\n  try {", "export function evaluateParityOperation(request) {\n  const { expected } = request;\n  void expected;\n  try {");
			break;
		case "js-indirect-import-before-verification":
			await replaceRequired(directory, "internal/emit/node/program/v1/contract.test.mjs", 'test("Countershape selected-field contract"', 'const loadHarness = async () => import("./harness.mjs");\n\ntest("Countershape selected-field contract"');
			await replaceRequired(directory, "internal/emit/node/program/v1/contract.test.mjs", 'const harness = await import("./harness.mjs");', "const harness = await loadHarness();");
			break;
		case "js-spawn-options-spread":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "child = spawn(process.execPath, contract.runtime.argv, {\n      argv0:", "child = spawn(process.execPath, contract.runtime.argv, {\n      ...{},\n      argv0:");
			break;
		case "js-spawn-alias":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", 'import { spawn } from "node:child_process";', 'import { spawn } from "node:child_process";\nconst launch = spawn;');
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "child = spawn(process.execPath, contract.runtime.argv,", "child = launch(process.execPath, contract.runtime.argv,");
			break;
		case "js-global-fetch-capability":
			await replaceRequired(directory, "internal/emit/node/program/v1/harness.mjs", "export async function runContract({ bundleRoot, targetRoot, decisionBytes, fixtureBytes }) {", 'export async function runContract({ bundleRoot, targetRoot, decisionBytes, fixtureBytes }) {\n  globalThis.fetch("https://example.invalid");');
			break;
		case "probe-package-map-sibling":
			await writeFile(join(directory, "internal/emit/node/cmd/p07b-a2-compiler-probe/hostile.go"), "package main\n");
			break;
		case "compiler-probe-import-drift":
			await replaceRequired(directory, "internal/emit/node/cmd/p07b-a2-compiler-probe/main.go", '"strings"', '"strings"\n\t_ "time"');
			break;
		case "compiler-probe-dataflow-drift":
			await replaceRequired(directory, "internal/emit/node/cmd/p07b-a2-compiler-probe/main.go", "bundle.CanonicalBytes()", "nil");
			break;
		case "parser-probe-compiler-import":
			await replaceRequired(directory, "internal/emit/node/cmd/p07b-a2-parser-probe/main.go", '"strings"', '"strings"\n\t_ "github.com/nelsonwerd/countershape/internal/emit/node/compiler"');
			break;
		case "parser-probe-schema-drift":
			await replaceRequired(directory, "internal/emit/node/cmd/p07b-a2-parser-probe/main.go", '"countershape.p07b-a2.parser-probe/v1"', '"countershape.p07b-a2.parser-probe/v2"');
			break;
		case "readme-warning-drift":
			await replaceRequired(directory, "internal/emit/node/model/bundle.go", "WARNING: FULL USER AUTHORITY AND NETWORK ACCESS.", "WARNING: CONTAINED AND HARMLESS.");
			break;
		case "readme-test-noop":
			await replaceRequired(directory, "internal/emit/node/compiler/compiler_test.go", "func TestRenderedREADMEStatesInvocationAndTrustCeilings(t *testing.T) {", "func TestRenderedREADMEStatesInvocationAndTrustCeilings(t *testing.T) {\n\treturn");
			break;
		case "example-generator-byte-drift": {
			const path = "tools/generate-p07b-a2-runtime-example.mjs";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `${source}// hostile generator drift\n`);
			break;
		}
		case "recovery-verifier-byte-drift": {
			const path = "tools/verify-p07b-a2-recovery-process.mjs";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `${source}// hostile verifier drift\n`);
			break;
		}
		case "human-capture-byte-drift": {
			const path = "tools/capture-p07b-a2-human-surface.mjs";
			const source = await readFile(join(directory, path), "utf8");
			await writeFile(join(directory, path), `${source}// hostile capture drift\n`);
			break;
		}
		case "human-capture-watchdog-removal":
			await replaceRequired(directory, "tools/capture-p07b-a2-human-surface.mjs",
				"setTimeout(() => process.exit(70), 15_000).unref();",
				"setTimeout(() => process.exit(70), 14_999).unref();");
			break;
		case "human-capture-bundle-binding-removal":
			await replaceRequired(directory, "tools/capture-p07b-a2-human-surface.mjs",
				'const digest = typedDigest("ContractBundle", canonicalBody);',
				"const digest = rawSHA256(canonicalBody);");
			break;
		case "human-capture-open-tap":
			await replaceRequired(directory, "tools/capture-p07b-a2-human-surface.mjs",
				"TAP has an unrecognized trailing line", "TAP trailing lines ignored");
			break;
		case "human-capture-control-roster-weakening":
			await replaceRequired(directory, "tools/capture-p07b-a2-human-surface.mjs",
				String.raw`\u007f-\u009f`, String.raw`\u007f`);
			break;
		case "human-capture-scenario-metadata-removal":
			await replaceRequired(directory, "tools/capture-p07b-a2-human-surface.mjs",
				"setup_profile: scenario.setupProfile,", 'setup_profile: "REMOVED",');
			break;
		default:
			throw new Error(`unknown mutant ${id}`);
		}
		const result = run(directory);
		const output = `${result.stdout}\n${result.stderr}`;
		if (result.error || result.signal || !Number.isInteger(result.status)) {
			throw new Error(`${id} infrastructure failure: ${result.signal ?? result.error?.message ?? result.status}: ${output}`);
		}
		if (result.status === 0 || !output.includes(expectedCode)) {
			throw new Error(`${id} survived or wrong code ${expectedCode}: ${output}`);
		}
	} finally {
		await rm(directory, { recursive: true, force: true });
	}
}

async function main() {
	if (process.argv.length !== 2) throw new Error("P07B_A2_SELFTEST_ARGUMENTS: no arguments accepted");
	if (!process.env.COUNTERSHAPE_GO) throw new Error("P07B_A2_GO_AUTHORITY_REQUIRED");
	if (rosterDigest() !== expectedRosterDigest) {
		throw new Error(`P07B_A2_SELFTEST_ROSTER_DIGEST: ${rosterDigest()} != ${expectedRosterDigest}`);
	}
	await cleanControl();
	await commentLiteralControl();
	for (const [id, code] of roster) await mutate(id, code);
	process.stdout.write(`P07B A2.2 architecture defensive self-test OK (${roster.length}/${roster.length}; roster ${expectedRosterDigest})\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
