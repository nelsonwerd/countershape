#!/usr/bin/env node

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { cp, mkdir, mkdtemp, readFile, rm, symlink, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const sourceRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const copyPaths = Object.freeze([
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
	"tools/check-p07b-architecture.mjs",
	"tools/mutate-p07b.mjs",
]);

const requiredCaseIDs = Object.freeze([
  "clean",
  "harmless-comment",
  "source-os-import",
  "source-store-import",
  "exported-source-field",
  "generic-source-constructor",
  "cli-binding-bypass",
  "http-binding-bypass",
  "parser-rebuild-bypass",
  "inherited-http-admitted",
  "ambient-lookup-enabled",
  "secret-slot-guard-dropped",
  "legacy-start-rewritten",
  "portable-start-collapsed",
  "start-readiness-crosspair-dropped",
  "source-proof-authority-added",
  "required-test-deleted",
  "rogue-package-file",
  "symlinked-authority",
	"source-transitive-world-dependency",
	"runner-lineage-collapsed",
	"cli-closed-schema-bypass",
	"http-closed-schema-bypass",
	"cli-definition-byte-compare-dropped",
	"http-definition-byte-compare-dropped",
	"adapter-projection-envelope-dropped",
	"profile-parse-bypass",
	"profile-order-bypass",
	"portable-frame-prefix-dropped",
	"portable-frame-eof-dropped",
	"portable-listener-injected",
	"portable-request-before-frame",
	"portable-receipt-frame-digest-dropped",
	"portable-fixture-node-net-dropped",
	"exhaustive-parity-test-deleted",
	"comment-spoofed-closed-schema",
	"string-spoofed-closed-schema",
	"legacy-fixture-byte-drift",
	"portable-fixture-byte-drift",
	"portable-fixture-constructor-swapped",
	"portable-fixture-entrypoint-swapped",
	"portable-measurement-relabeled",
	"portable-listener-presence-forged",
	"legacy-frame-evidence-reexposed",
	"rejected-readiness-receipt-dropped",
	"rejected-readiness-invocation-inspected",
	"child-port-ownership-overclaimed",
	"fixture-history-test-deleted",
	"portable-negative-path-test-deleted",
	"mutation-gate-receipt-binding-dropped",
]);
const requiredCaseDigest = "9fcb89663a0709b2e5727488d7481d169a15a7f2d4d6cad70018273fc38c6437";
const expectedHostileCodes = Object.freeze({
	"source-os-import": "P07B_SOURCE_IMPORT_NOT_ALLOWED",
	"source-store-import": "P07B_SOURCE_IMPORT_NOT_ALLOWED",
	"exported-source-field": "P07B_SOURCE_CAPABILITY_OPEN",
	"generic-source-constructor": "P07B_GENERIC_SOURCE_CONSTRUCTOR",
	"cli-binding-bypass": "P07B_CLI_AUTHORITY_JOIN",
	"http-binding-bypass": "P07B_HTTP_AUTHORITY_JOIN",
	"parser-rebuild-bypass": "P07B_PARSE_AUTHORITY_JOIN",
	"inherited-http-admitted": "P07B_HTTP_AUTHORITY_JOIN",
	"ambient-lookup-enabled": "P07B_SOURCE_CLOSED_FACT_MISSING",
	"secret-slot-guard-dropped": "P07B_SOURCE_CLOSED_FACT_MISSING",
	"legacy-start-rewritten": "P07B_HTTP_START_HISTORY_OR_PORTABLE_PROFILE",
	"portable-start-collapsed": "P07B_HTTP_START_HISTORY_OR_PORTABLE_PROFILE",
	"start-readiness-crosspair-dropped": "P07B_HTTP_START_READINESS_CROSS_PAIR",
	"source-proof-authority-added": "P07B_SOURCE_FORBIDDEN_AUTHORITY",
	"required-test-deleted": "P07B_SOURCE_TEST_MISSING",
	"rogue-package-file": "P07B_PACKAGE_MAP_DRIFT",
	"symlinked-authority": "P07B_NONREGULAR_AUTHORITY",
	"source-transitive-world-dependency": "P07B_SOURCE_TRANSITIVE_INTERNAL_DEPENDENCY",
	"runner-lineage-collapsed": "P07B_RUNNER_PROFILE_MISSING",
	"cli-closed-schema-bypass": "P07B_CLI_PROJECTION_RESOLUTION_OPEN",
	"http-closed-schema-bypass": "P07B_HTTP_PROJECTION_RESOLUTION_OPEN",
	"cli-definition-byte-compare-dropped": "P07B_CLI_PROJECTION_RESOLUTION_OPEN",
	"http-definition-byte-compare-dropped": "P07B_HTTP_PROJECTION_RESOLUTION_OPEN",
	"adapter-projection-envelope-dropped": "P07B_SOURCE_PROJECTION_ENVELOPE_MISSING",
	"profile-parse-bypass": "P07B_PROFILE_PARSE_NOT_EXACT",
	"profile-order-bypass": "P07B_PROFILE_ROSTER_OPEN",
	"portable-frame-prefix-dropped": "P07B_HTTP_PORT_FRAME_OPEN",
	"portable-frame-eof-dropped": "P07B_HTTP_PHYSICAL_CHILD_BIND_MISSING",
	"portable-listener-injected": "P07B_HTTP_PHYSICAL_CHILD_BIND_MISSING",
	"portable-request-before-frame": "P07B_HTTP_REQUEST_ORDER_OPEN",
	"portable-receipt-frame-digest-dropped": "P07B_HTTP_PORTABLE_RECEIPT_INCOMPLETE",
	"portable-fixture-node-net-dropped": "P07B_HTTP_PORTABLE_FIXTURE_DRIFT",
	"exhaustive-parity-test-deleted": "P07B_SOURCE_TEST_MISSING",
	"comment-spoofed-closed-schema": "P07B_CLI_PROJECTION_RESOLUTION_OPEN",
	"string-spoofed-closed-schema": "P07B_CLI_PROJECTION_RESOLUTION_OPEN",
	"legacy-fixture-byte-drift": "P07B_HTTP_LEGACY_FIXTURE_DRIFT",
	"portable-fixture-byte-drift": "P07B_HTTP_PORTABLE_FIXTURE_DRIFT",
	"portable-fixture-constructor-swapped": "P07B_HTTP_STUDY_PORTABLE_SELECTION_OPEN",
	"portable-fixture-entrypoint-swapped": "P07B_HTTP_STUDY_PORTABLE_SELECTION_OPEN",
	"portable-measurement-relabeled": "P07B_HTTP_STUDY_PORTABLE_SELECTION_OPEN",
	"portable-listener-presence-forged": "P07B_HTTP_PORTABLE_RECEIPT_INCOMPLETE",
	"legacy-frame-evidence-reexposed": "P07B_HTTP_PHYSICAL_CHILD_BIND_MISSING",
	"rejected-readiness-receipt-dropped": "P07B_HTTP_REJECTED_READINESS_RECEIPT_DROPPED",
	"rejected-readiness-invocation-inspected": "P07B_HTTP_REJECTED_READINESS_RECEIPT_DROPPED",
	"child-port-ownership-overclaimed": "P07B_HTTP_PORT_OWNERSHIP_OVERCLAIM",
	"fixture-history-test-deleted": "P07B_HTTP_FIXTURE_HISTORY_TEST_MISSING",
	"portable-negative-path-test-deleted": "P07B_REQUIRED_TEST_MISSING",
	"mutation-gate-receipt-binding-dropped": "P07B_MUTATION_GATE_INCOMPLETE",
});
const expectedHostileCodeDigest = "7931373b4ebb2a1529861214b759ef80d88538dd9b273ec0780bb0c2f5b2260a";

async function copyFixture() {
  const fixture = await mkdtemp(join(tmpdir(), "countershape-p07b-architecture-"));
  for (const relativePath of copyPaths) {
    const target = join(fixture, relativePath);
    await mkdir(dirname(target), { recursive: true });
    await cp(join(sourceRoot, relativePath), target);
  }
  await mkdir(join(fixture, "tools"), { recursive: true });
  await writeFile(
    join(fixture, "tools/check-u6-architecture.mjs"),
    '#!/usr/bin/env node\nprocess.stdout.write("U6 architecture boundary OK (self-test inherited stub)\\n");\n',
    { mode: 0o755 },
  );
	await writeFile(
		join(fixture, "tools/go-stub.mjs"),
		`#!/usr/bin/env node
import { existsSync } from "node:fs";
const allowed = [
  "github.com/nelsonwerd/countershape/internal/canon",
  "github.com/nelsonwerd/countershape/internal/domain",
  "github.com/nelsonwerd/countershape/internal/adapters/cli/model",
  "github.com/nelsonwerd/countershape/internal/runnerprofile",
  "github.com/nelsonwerd/countershape/internal/adapters/http/model",
  "github.com/nelsonwerd/countershape/internal/portablevalue",
  "github.com/nelsonwerd/countershape/internal/projectionprofile",
  "github.com/nelsonwerd/countershape/internal/contractsource",
];
if (existsSync("p07b-rogue-dependency")) allowed.push("github.com/nelsonwerd/countershape/internal/world");
process.stdout.write(allowed.join("\\n") + "\\n");
`,
		{ mode: 0o755 },
	);
  return fixture;
}

async function replace(fixture, relativePath, from, to) {
  const path = join(fixture, relativePath);
  const source = await readFile(path, "utf8");
  if (!source.includes(from)) throw new Error(`self-test replacement anchor missing: ${relativePath}: ${from}`);
  await writeFile(path, source.replace(from, to));
}

function run(fixture) {
  return spawnSync(process.execPath, [join(fixture, "tools/check-p07b-architecture.mjs")], {
    cwd: fixture,
    encoding: "utf8",
    env: {
		PATH: dirname(process.execPath), HOME: fixture, TMPDIR: fixture, LANG: "C", TZ: "UTC", NO_COLOR: "1",
		COUNTERSHAPE_GO: join(fixture, "tools/go-stub.mjs"),
	},
  });
}

const cases = new Map([
  ["clean", async () => {}],
  ["harmless-comment", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "package contractsource", "package contractsource\n\n// harmless os/exec internal/store NewPortableSource candidate_execution_key camouflage");
  }],
  ["source-os-import", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", 'import (\n\t"bytes"', 'import (\n\t"bytes"\n\t"os"');
  }],
  ["source-store-import", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", '"github.com/nelsonwerd/countershape/internal/domain"', '"github.com/nelsonwerd/countershape/internal/domain"\n\t"github.com/nelsonwerd/countershape/internal/store"');
  }],
  ["exported-source-field", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "\tdigest           domain.Digest", "\tDigest           domain.Digest");
  }],
  ["generic-source-constructor", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "func NewCLISource", "func NewPortableSource() PortableSource { return PortableSource{} }\n\nfunc NewCLISource");
  }],
  ["cli-binding-bypass", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "execution, err := cli.BindExecution", "execution, err := cli_BindExecution");
  }],
  ["http-binding-bypass", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "execution, err := counterhttp.BindExecution", "execution, err := counterhttp_BindExecution");
  }],
  ["parser-rebuild-bypass", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "rebuilt, err := buildSource(identity)", "rebuilt, err := buildSourceBypass(identity)");
  }],
  ["inherited-http-admitted", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "input.Start.Authority() != counterhttp.HTTPPortableStartAuthorityV1", "false");
  }],
  ["ambient-lookup-enabled", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "AmbientExecutableLookup: false", "AmbientExecutableLookup: true");
  }],
  ["secret-slot-guard-dropped", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "if len(identity.PlanSecretSlots) != 0", "if false");
  }],
  ["legacy-start-rewritten", async (fixture) => {
    await replace(fixture, "internal/adapters/http/model/start.go", "return newHTTPStartSpec(entrypoint, HTTPStartAuthorityV1)", "return newHTTPStartSpec(entrypoint, HTTPPortableStartAuthorityV1)");
  }],
  ["portable-start-collapsed", async (fixture) => {
    await replace(fixture, "internal/adapters/http/model/start.go", "return newHTTPStartSpec(entrypoint, HTTPPortableStartAuthorityV1)", "return newHTTPStartSpec(entrypoint, HTTPStartAuthorityV1)");
  }],
  ["start-readiness-crosspair-dropped", async (fixture) => {
    await replace(fixture, "internal/adapters/http/model/binding.go", "start.Authority() == HTTPPortableStartAuthorityV1", "false");
  }],
  ["source-proof-authority-added", async (fixture) => {
    await replace(fixture, "internal/contractsource/source.go", "type sourceIdentity struct {", "type sourceIdentity struct {\n\tProjectionProof string `json:\"projection_proof\"`");
  }],
  ["required-test-deleted", async (fixture) => {
    await replace(fixture, "internal/contractsource/source_test.go", "func TestPortableSourceRequiresExactAuthorityJoin", "func deletedPortableSourceRequiresExactAuthorityJoin");
  }],
  ["rogue-package-file", async (fixture) => {
    await writeFile(join(fixture, "internal/contractsource/rogue.go"), "package contractsource\n");
  }],
  ["symlinked-authority", async (fixture) => {
    const target = join(fixture, "internal/contractsource/errors.go");
    await unlink(target);
    await symlink(join(sourceRoot, "internal/contractsource/errors.go"), target);
  }],
	["source-transitive-world-dependency", async (fixture) => {
		await writeFile(join(fixture, "p07b-rogue-dependency"), "world\n");
	}],
	["runner-lineage-collapsed", async (fixture) => {
		await replace(
			fixture,
			"internal/runnerprofile/profile.go",
			'HTTPPortableRunnerLineageV1 = "world.ExecuteHTTP/P07B/opaque-binding/child-bind-pipe-ready/v1"',
			'HTTPPortableRunnerLineageV1 = "world.ExecuteHTTP/U4/opaque-binding/inherited-listener/v1"',
		);
	}],
	["cli-closed-schema-bypass", async (fixture) => {
		await replace(
			fixture,
			"internal/adapters/cli/model/projection_authority.go",
			"expected, err := NewCLIProjectionAuthority(identity.Fields)",
			"expected := CLIProjectionAuthority{digest: digest, canonicalBytes: canonicalBytes, binding: binding, fieldIDs: identity.Fields}\n\t// NewCLIProjectionAuthority(identity.Fields)",
		);
	}],
	["http-closed-schema-bypass", async (fixture) => {
		await replace(
			fixture,
			"internal/adapters/http/model/projection_authority.go",
			"expected, err := NewHTTPProjectionAuthority()",
			"expected := HTTPProjectionAuthority{digest: digest, canonicalBytes: canonicalBytes, binding: binding, fieldIDs: identity.Fields}\n\t// NewHTTPProjectionAuthority()",
		);
	}],
	["cli-definition-byte-compare-dropped", async (fixture) => {
		await replace(
			fixture,
			"internal/adapters/cli/model/projection_authority.go",
			"!bytes.Equal(expected.canonicalBytes, canonicalBytes) ||",
			"false || /* !bytes.Equal(expected.canonicalBytes, canonicalBytes) || */",
		);
	}],
	["http-definition-byte-compare-dropped", async (fixture) => {
		await replace(
			fixture,
			"internal/adapters/http/model/projection_authority.go",
			"!bytes.Equal(expected.canonicalBytes, canonicalBytes) ||",
			"false || /* !bytes.Equal(expected.canonicalBytes, canonicalBytes) || */",
		);
	}],
	["adapter-projection-envelope-dropped", async (fixture) => {
		await replace(
			fixture,
			"internal/contractsource/source.go",
			"AdapterProjectionBase64: base64.StdEncoding.EncodeToString(adapterProjectionBytes),\n\t\tAdapterProjectionDigest: adapterProjectionDigest.String(),",
			"// AdapterProjectionBase64: base64.StdEncoding.EncodeToString(adapterProjectionBytes),\n\t\t// AdapterProjectionDigest: adapterProjectionDigest.String(),",
		);
	}],
	["profile-parse-bypass", async (fixture) => {
		await replace(
			fixture,
			"internal/projectionprofile/profile.go",
			"if err != nil || !bytes.Equal(rebuilt.CanonicalBytes(), exact) {",
			"if err != nil { // !bytes.Equal(rebuilt.CanonicalBytes(), exact)",
		);
	}],
	["profile-order-bypass", async (fixture) => {
		await replace(
			fixture,
			"internal/contractsource/profile_roster.go",
			"rosterIndex <= previousRosterIndex",
			"previousRosterIndex >= rosterIndex && false /* rosterIndex <= previousRosterIndex */",
		);
	}],
	["portable-frame-prefix-dropped", async (fixture) => {
		await replace(
			fixture,
			"internal/adapters/http/model/start.go",
			"!bytes.HasPrefix(exact, []byte(PortableReadinessFramePrefix))",
			"false /* !bytes.HasPrefix(exact, []byte(PortableReadinessFramePrefix)) */",
		);
	}],
	["portable-frame-eof-dropped", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_service_darwin.go",
			"valid := observed.err == nil && observed.eof",
			"valid := observed.err == nil // && observed.eof",
		);
		await replace(
			fixture,
			"internal/world/http_receipt.go",
			"frameValid := frameErr == nil && frame.Valid() && physical.eofObserved",
			"frameValid := frameErr == nil && frame.Valid() // && physical.eofObserved",
		);
	}],
	["portable-listener-injected", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_service_darwin.go",
			"extraFiles := []*os.File{readinessWriter}",
			"extraFiles := []*os.File{listenerFile, readinessWriter}",
		);
	}],
	["portable-request-before-frame", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_service_darwin.go",
			"if !portable {\n\t\trequestWire, err = httpmodel.EncodeRequest",
			"if !portable || portable {\n\t\trequestWire, err = httpmodel.EncodeRequest",
		);
	}],
	["portable-receipt-frame-digest-dropped", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_receipt.go",
			"FrameDigest: frameDigest.String(),",
			"FrameDigest: \"\", // FrameDigest: frameDigest.String(),",
		);
	}],
	["portable-fixture-node-net-dropped", async (fixture) => {
		await replace(
			fixture,
			"testkit/httpfixture/server_child_bind.mjs",
			'import { createServer } from "node:net";',
			'const createServer = globalThis.createServer; // import { createServer } from "node:net";',
		);
	}],
	["exhaustive-parity-test-deleted", async (fixture) => {
		await replace(
			fixture,
			"internal/contractsource/source_test.go",
			"func TestClosedProjectionModelsMatchAdaptersAndPortableProfilesExhaustively",
			"func deletedClosedProjectionModelsMatchAdaptersAndPortableProfilesExhaustively",
		);
	}],
	["comment-spoofed-closed-schema", async (fixture) => {
		await replace(
			fixture,
			"internal/adapters/cli/model/projection_authority.go",
			"expected, err := NewCLIProjectionAuthority(identity.Fields)",
			"// expected, err := NewCLIProjectionAuthority(identity.Fields)\n\texpected := CLIProjectionAuthority{}",
		);
	}],
	["string-spoofed-closed-schema", async (fixture) => {
		await replace(
			fixture,
			"internal/adapters/cli/model/projection_authority.go",
			"expected, err := NewCLIProjectionAuthority(identity.Fields)",
			'const spoofedCall = "NewCLIProjectionAuthority(identity.Fields)"\n\t_ = spoofedCall\n\tvar err error\n\texpected := CLIProjectionAuthority{}',
		);
	}],
	["legacy-fixture-byte-drift", async (fixture) => {
		await replace(
			fixture,
			"testkit/httpfixture/server.mjs",
			'const fixtureVersion = "countershape-http-invoice-fixture/v1";',
			'const fixtureVersion = "countershape-http-invoice-fixture/v1";\n// hostile byte drift',
		);
	}],
	["portable-fixture-byte-drift", async (fixture) => {
		await replace(
			fixture,
			"testkit/httpfixture/server_child_bind.mjs",
			'const fixtureVersion = "countershape-http-invoice-fixture/v1";',
			'const fixtureVersion = "countershape-http-invoice-fixture/v1";\n// hostile byte drift',
		);
	}],
	["portable-fixture-constructor-swapped", async (fixture) => {
		await replace(
			fixture,
			"testkit/studies/http_invoices/study.go",
			"fixtureFiles = httpfixture.PortableCandidateFiles",
			"fixtureFiles = httpfixture.CandidateFiles // httpfixture.PortableCandidateFiles",
		);
	}],
	["portable-fixture-entrypoint-swapped", async (fixture) => {
		await replace(
			fixture,
			"testkit/studies/http_invoices/study.go",
			"entrypoint = httpfixture.PortableEntrypoint",
			"entrypoint = httpfixture.Entrypoint // httpfixture.PortableEntrypoint",
		);
	}],
	["portable-measurement-relabeled", async (fixture) => {
		await replace(
			fixture,
			"testkit/studies/http_invoices/study.go",
			"{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, binding.Authority()},",
			"{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, counterhttp.HTTPExecutionAuthorityV1}, // binding.Authority()",
		);
	}],
	["portable-listener-presence-forged", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_receipt.go",
			"ListenerPresent: false, ReadinessFD: physical.readinessFD,",
			"ListenerPresent: true, ReadinessFD: physical.readinessFD, // ListenerPresent: false",
		);
	}],
	["legacy-frame-evidence-reexposed", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_service_darwin.go",
			"if portable {\n\t\t\tresult.readiness.frameBytes = append([]byte(nil), readiness.bytes...)\n\t\t}",
			"result.readiness.frameBytes = append([]byte(nil), readiness.bytes...) // if portable",
		);
	}],
	["rejected-readiness-receipt-dropped", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_api.go",
			"executionAuthority == httpmodel.HTTPPortableExecutionAuthorityV1 &&",
			"false && // executionAuthority == httpmodel.HTTPPortableExecutionAuthorityV1 &&",
		);
	}],
	["rejected-readiness-invocation-inspected", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_api.go",
			"if len(service.exchange.requestWire) > 0 {",
			"if true { // len(service.exchange.requestWire) > 0",
		);
	}],
	["child-port-ownership-overclaimed", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_api.go",
			"physical HTTP entry has no reported endpoint receipt",
			"physical HTTP entry has no owned endpoint receipt",
		);
	}],
	["fixture-history-test-deleted", async (fixture) => {
		await replace(
			fixture,
			"testkit/httpfixture/fixture_test.go",
			"func TestPortableCandidateFilesAreDistinctWithoutRewritingLegacyFixture",
			"func deletedPortableCandidateFilesAreDistinctWithoutRewritingLegacyFixture",
		);
	}],
	["portable-negative-path-test-deleted", async (fixture) => {
		await replace(
			fixture,
			"internal/world/http_portable_negative_darwin_test.go",
			"func TestHTTPPortableReadinessFailuresRemainFinalizedReceipts",
			"func deletedHTTPPortableReadinessFailuresRemainFinalizedReceipts",
		);
	}],
	["mutation-gate-receipt-binding-dropped", async (fixture) => {
		await replace(fixture, "tools/mutate-p07b.mjs", "source_manifest", "source_receipt_manifest");
	}],
]);

async function main() {
  if (process.argv.length !== 2) throw new Error("P07B architecture self-test accepts no arguments");
  const digest = createHash("sha256").update(requiredCaseIDs.join("\n")).digest("hex");
  if (digest !== requiredCaseDigest) throw new Error(`P07B architecture self-test roster digest mismatch: ${digest}`);
	const hostileIDs = requiredCaseIDs.filter((id) => id !== "clean" && id !== "harmless-comment").sort();
	const declaredHostileIDs = Object.keys(expectedHostileCodes).sort();
	const codeDigest = createHash("sha256").update(
		Object.entries(expectedHostileCodes).sort(([left], [right]) => left.localeCompare(right, "en"))
			.map(([id, code]) => `${id}\0${code}`).join("\n"),
	).digest("hex");
	if (JSON.stringify(hostileIDs) !== JSON.stringify(declaredHostileIDs) || codeDigest !== expectedHostileCodeDigest) {
		throw new Error(`P07B architecture self-test expected-code contract mismatch: ${codeDigest}`);
	}
  if (cases.size !== requiredCaseIDs.length || requiredCaseIDs.some((id) => !cases.has(id))) {
    throw new Error("P07B architecture self-test case map differs from the checksum-pinned roster");
  }
  let hostile = 0;
  for (const id of requiredCaseIDs) {
    const fixture = await copyFixture();
    try {
      await cases.get(id)(fixture);
      const result = run(fixture);
      const clean = id === "clean" || id === "harmless-comment";
      if (clean) {
        if (result.error || result.signal || result.status !== 0 || !result.stdout.includes("P07B source architecture boundary OK")) {
          throw new Error(`${id} control failed: ${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
        }
      } else {
        hostile += 1;
        const codes = [...new Set(result.stderr.match(/P07B_[A-Z0-9_]+/gu) ?? [])];
		const expectedCode = expectedHostileCodes[id];
		const directCodes = new Set([
			"P07B_PACKAGE_MAP_DRIFT", "P07B_NONREGULAR_AUTHORITY", "P07B_SOURCE_TRANSITIVE_INTERNAL_DEPENDENCY",
		]);
		const expectedTop = directCodes.has(expectedCode) ? expectedCode : "P07B_ARCHITECTURE_VIOLATION";
        if (result.error || result.signal || result.status === 0 || codes[0] !== expectedTop || !codes.includes(expectedCode)) {
          throw new Error(`${id} hostile case escaped or failed untyped: ${result.status ?? result.signal}: ${result.stderr || result.stdout}`);
        }
		if (process.env.COUNTERSHAPE_P07B_SELFTEST_TRACE === "1") {
			process.stdout.write(`TRACE ${id} ${codes.join(",")}\n`);
		}
      }
    } finally {
      await rm(fixture, { recursive: true, force: true });
    }
  }
  process.stdout.write(`P07B architecture self-test OK (2 clean/comment controls; ${hostile}/${hostile} hostile cases)\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
