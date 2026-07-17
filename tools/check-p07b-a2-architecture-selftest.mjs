#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { copyFile, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const checker = resolve(root, "tools/check-p07b-a2-architecture.mjs");
const files = Object.freeze([
	"internal/choice/choicepoint.go",
	"internal/choice/promotion/service.go",
	"internal/choice/session_roundtrip_test.go",
	"internal/confirmation/service_test.go",
	"internal/confirmation/wire.go",
	"internal/world/fresh_confirmation.go",
	"internal/domain/world.go",
	"internal/domain/domain_test.go",
	"internal/emit/node/model/predicate.go",
	"internal/emit/node/model/predicate_test.go",
	"internal/emit/node/model/source_profile.go",
	"internal/emit/node/internal/compilation/input.go",
	"internal/emit/node/service.go",
	"testkit/studies/cli_precedence/study_darwin_test.go",
	"testkit/studies/cli_precedence/reduction_darwin_test.go",
	"testkit/studies/http_invoices/reduction_darwin_test.go",
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
]);
const expectedRosterDigest = "sha256:459e3949ee720d3b7d7b2182445f5543c95f55f09c25ff7d16bdf70ba7036dbb";

function rosterDigest() {
	const hash = createHash("sha256");
	for (const [id, code] of roster) hash.update(id).update("\0").update(code).update("\0");
	return `sha256:${hash.digest("hex")}`;
}

async function copyFixture(label) {
	const directory = await mkdtemp(join(process.env.TMPDIR ?? "/tmp", `countershape-a2-arch-${label}-`));
	for (const path of files) {
		await mkdir(dirname(join(directory, path)), { recursive: true });
		await copyFile(join(root, path), join(directory, path));
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
		if (result.status !== 0 || !result.stdout.includes("P07B A2.1 architecture fixture OK") ||
			result.stdout.includes("P07B A2.1 architecture boundary OK")) {
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
	process.stdout.write(`P07B A2.1 architecture defensive self-test OK (${roster.length}/${roster.length}; roster ${expectedRosterDigest})\n`);
}

main().catch((error) => {
	process.stderr.write(`${error.stack ?? error}\n`);
	process.exitCode = 1;
});
