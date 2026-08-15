#!/usr/bin/env node

import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { lstatSync, readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

function fail(code, detail) {
	process.stderr.write(`${code}: ${detail}\n`);
	process.exit(1);
}

function git(...args) {
	return execFileSync("/usr/bin/git", args, { encoding: "utf8" });
}

function sha256(bytes) {
	return createHash("sha256").update(bytes).digest("hex");
}

const sourceCommit = "2fc5e5ffcea37d6151d0f833ff16a84a975da658";
const allowed = [
	"docs/FINAL_HANDOFF.md",
	"docs/HANDOFF_MODE_C.md",
	"docs/RECEIPTS.md",
	"evidence/final/environment-matrix.json",
	"evidence/final/unit-receipt-inventory.json",
	"evidence/s6/S6-LIVE-AGENT-RESULT.md",
	"evidence/s6/s6-matrix.json",
	"tools/final-audit/final-audit.mjs",
	"tools/final-audit/s6-audit.mjs",
];

const changed = git("diff", "--cached", "--name-only").trim().split("\n").filter(Boolean).sort();
if (JSON.stringify(changed) !== JSON.stringify(allowed)) {
	fail("P12_STAGED_ROSTER", changed.join(","));
}
if (git("diff", "--name-only").trim() !== "") {
	fail("P12_UNSTAGED_DELTA", "working tree differs from index");
}
execFileSync("/usr/bin/git", ["diff", "--cached", "--check"], { stdio: "inherit" });

for (const path of allowed) {
	const stat = lstatSync(path);
	if (!stat.isFile() || stat.isSymbolicLink() || stat.nlink !== 1) {
		fail("P12_FILE_AUTHORITY", path);
	}
}

const productDiff = git("diff", "--cached", "--name-only", sourceCommit).trim().split("\n").filter(Boolean).sort();
if (JSON.stringify(productDiff) !== JSON.stringify(allowed)) {
	fail("P12_PRODUCT_SCOPE", productDiff.join(","));
}

const trackedDidrun = git("ls-files", ".didrun", ".didrun/**").trim();
if (trackedDidrun !== "") {
	fail("P12_TRACKED_DIDRUN", trackedDidrun);
}

const credentialPattern = /-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY-----|(?:AKIA|ASIA)[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{30,}|glpat-[A-Za-z0-9_-]{20,}|xox[baprs]-[A-Za-z0-9-]{20,}|AIza[0-9A-Za-z_-]{30,}|sk_live_[0-9A-Za-z]{16,}|sk-(?:proj|svcacct)-[A-Za-z0-9_-]{20,}|npm_[A-Za-z0-9]{24,}|pypi-[A-Za-z0-9_-]{24,}|AccountKey=[A-Za-z0-9+/=]{24,}/;
for (const path of allowed.filter((candidate) => !candidate.startsWith("tools/final-audit/"))) {
	const text = path === "docs/HANDOFF_MODE_C.md"
		? git("diff", "--cached", "--unified=0", "--", path).split("\n").filter((line) => line.startsWith("+") && !line.startsWith("+++ ")).join("\n")
		: readFileSync(path, "utf8");
	if (credentialPattern.test(text)) {
		fail("P12_NAMED_CREDENTIAL_PATTERN", path);
	}
	if (text.includes("/Users/drewnelson")) {
		fail("P12_PRIVATE_USER_PATH", path);
	}
	if (/Co-Authored-By:/i.test(text)) {
		fail("P12_AI_COAUTHOR_TRAILER", path);
	}
}

const inventory = JSON.parse(readFileSync("evidence/final/unit-receipt-inventory.json", "utf8"));
const expectedCommits = [
	"41413ad6571a4778b306d4d06954bb9c0d26ce81",
	"13282ed01aeb226c79a11c13d3db3286a59529d2",
	"2ab4b651183835117263bc2c90881e71b9b1d702",
	"4a4ce4d6bba678bff1cac3c8c7a6235d8d51dfcc",
	"8197fd837e7ea585144c6e646c71b44c9594a151",
	"497418ef3d9ca87658a9a524e8beedd0ca07319e",
	"64b65cbfa6099f9be88c2de3cc2cc4dd5ff9ff81",
	"9bf3b243d69f4253bfcdf706995e99e2edd63e67",
	"ed58cb6dd6f50fd1f2a889b2e3d6b728a43cd3b0",
	"2fc5e5ffcea37d6151d0f833ff16a84a975da658",
];
if (inventory.schema_version !== "countershape/final-unit-receipt-inventory/v1" || inventory.units.length !== expectedCommits.length) {
	fail("P12_UNIT_INVENTORY_SHAPE", "root");
}
for (let index = 0; index < expectedCommits.length; index += 1) {
	const expectedCommit = expectedCommits[index];
	const unit = inventory.units[index];
	if (unit.commit !== expectedCommit || unit.unit !== `U${index}`) {
		fail("P12_UNIT_IDENTITY", `U${index}`);
	}
	const tree = git("show", "-s", "--format=%T", expectedCommit).trim();
	const subject = git("show", "-s", "--format=%s", expectedCommit).trim();
	const noteList = git("notes", "--ref=didrun", "list", expectedCommit).trim().split(/\s+/);
	const noteBytes = Buffer.from(git("notes", "--ref=didrun", "show", expectedCommit), "utf8");
	const note = JSON.parse(noteBytes.toString("utf8"));
	if (unit.tree !== tree || unit.subject !== subject || unit.note_blob !== noteList[0] || unit.note_body_sha256 !== sha256(noteBytes)) {
		fail("P12_UNIT_NOTE_BINDING", unit.unit);
	}
	if (note.commit !== expectedCommit || note.tree !== tree || unit.claims !== note.claims.length || unit.events !== note.coverage.total_events || unit.strict_exit_recorded !== 0) {
		fail("P12_UNIT_NOTE_SHAPE", unit.unit);
	}
	const grades = {};
	for (const claim of note.claims) {
		if (claim.exit_code !== 0 || !["tree-exact", "scope-exact"].includes(claim.grade)) {
			fail("P12_UNIT_GRADE", `${unit.unit}:${claim.grade}:${claim.exit_code}`);
		}
		grades[claim.grade] = (grades[claim.grade] || 0) + 1;
	}
	if (JSON.stringify(unit.grades) !== JSON.stringify(grades) || unit.secrets_override !== note.secrets_override) {
		fail("P12_UNIT_SUMMARY", unit.unit);
	}
	if (unit.html) {
		const bytes = readFileSync(unit.html.path);
		if (bytes.length !== unit.html.bytes || sha256(bytes) !== unit.html.sha256) {
			fail("P12_UNIT_HTML", unit.unit);
		}
	}
}

const matrix = JSON.parse(readFileSync("evidence/s6/s6-matrix.json", "utf8"));
if (matrix.verdict !== "PARK" || matrix.required_bare_recall.captured !== 6 || matrix.required_bare_recall.total !== 8 ||
	matrix.required_bare_recall.ratio !== 0.75 || matrix.full_transcript_bare_recall.captured !== 6 ||
	matrix.full_transcript_bare_recall.total !== 12 || matrix.full_transcript_bare_recall.ratio !== 0.5 ||
	matrix.absolute_path.structural_miss_accounted !== 2) {
	fail("P12_S6_MATRIX", "recall or verdict drift");
}

const receipts = readFileSync("docs/RECEIPTS.md", "utf8");
const requiredHeader = "Capability | Scope/environment | Load-bearing command | Event/claim reference | Verbatim didrun grade | What the grade does not establish";
if (!receipts.includes(requiredHeader) || (receipts.match(/UNRECEIPTED/g) || []).length < 10 || !receipts.includes("This committed table cannot self-receipt")) {
	fail("P12_RECEIPT_CONTRACT", "table or nonclaim ceiling");
}
const handoff = readFileSync("docs/FINAL_HANDOFF.md", "utf8");
for (const required of [
	"Nothing here claims that Countershape is finished",
	"S6 live-agent capture result",
	"The result is `PARK`",
	"Validated versus bet",
	"Human tail and ongoing costs",
	"CONFIDENTIALITY NOT ESTABLISHED",
]) {
	if (!handoff.includes(required)) {
		fail("P12_HANDOFF_CONTRACT", required);
	}
}

const serverFiles = readdirSync("internal/server").filter((name) => name.endsWith(".go") && !name.endsWith("_test.go"));
const serverSource = serverFiles.map((name) => readFileSync(join("internal/server", name), "utf8")).join("\n");
if (serverSource.includes("Access-Control-Allow-Origin") || serverSource.includes("0.0.0.0") || !serverSource.includes("127.0.0.1:0")) {
	fail("P12_SERVER_STATIC_BOUNDARY", "bind or CORS drift");
}
const exportSources = ["internal/report/report.go", "internal/server/export.go", "cmd/countershape/export_darwin.go"]
	.filter((path) => {
		try { lstatSync(path); return true; } catch { return false; }
	})
	.map((path) => readFileSync(path, "utf8")).join("\n");
if (!exportSources.includes("CONFIDENTIALITY NOT ESTABLISHED")) {
	fail("P12_EXPORT_WARNING", "missing exact warning");
}

const releaseProfile = readFileSync("packaging/release-profile.json", "utf8");
if (releaseProfile.includes("/Users/") || releaseProfile.includes(".didrun") || releaseProfile.includes("node_modules")) {
	fail("P12_PACKAGE_MANIFEST_LEAK", "release profile");
}

process.stdout.write(`COUNTERSHAPE_P12_FINAL_AUDIT_OK units=${inventory.units.length} paths=${allowed.length} s6=${matrix.verdict}\n`);
