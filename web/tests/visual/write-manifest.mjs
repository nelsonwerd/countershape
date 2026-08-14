import { createHash } from "node:crypto";
import { readdir, readFile, writeFile } from "node:fs/promises";
import { basename, resolve } from "node:path";

const [directoryArg, pass, mode = "write"] = process.argv.slice(2);
if (!directoryArg || !/^pass[123]$|^final$/.test(pass ?? "") || !/^(write|verify)$/.test(mode)) {
  throw new Error("usage: node write-manifest.mjs <capture-directory> <pass1|pass2|pass3|final> [write|verify]");
}

const directory = resolve(directoryArg);
const expectedStates = [
  "empty", "preparing", "active", "partial", "error", "unstable", "uncomparable", "discovered",
  "decision-ready", "predicate-editing", "identity-reveal", "resolved", "reject-all-resolved", "deferred",
  "stale", "invalidated", "incomplete", "completed",
];
const expected = expectedStates.flatMap((state) => [
  `${pass}-${state}-desktop.png`,
  `${pass}-${state}-mobile.png`,
]).sort();
const actual = (await readdir(directory)).filter((name) => name.endsWith(".png")).sort();
if (JSON.stringify(actual) !== JSON.stringify(expected)) {
  throw new Error(`capture roster mismatch for ${basename(directory)}: ${actual.length} PNGs`);
}

const images = [];
for (const name of actual) {
  const bytes = await readFile(resolve(directory, name));
  if (bytes.toString("ascii", 1, 4) !== "PNG" || bytes.toString("ascii", 12, 16) !== "IHDR") {
    throw new Error(`${name} is not a strict PNG with an IHDR header`);
  }
  const width = bytes.readUInt32BE(16);
  const height = bytes.readUInt32BE(20);
  const expectedWidth = name.endsWith("-desktop.png") ? 1440 : 375;
  const expectedHeight = name.endsWith("-desktop.png") ? 900 : 812;
  if (width !== expectedWidth || height !== expectedHeight) {
    throw new Error(`${name} dimensions ${width}x${height}`);
  }
  images.push({ name, width, height, bytes: bytes.length, sha256: createHash("sha256").update(bytes).digest("hex") });
}

const manifest = {
  schema: "countershape/u8-p10-sanitized-visual-matrix/v1",
  pass,
  states: expectedStates,
  viewports: [{ name: "desktop", width: 1440, height: 900 }, { name: "mobile", width: 375, height: 812 }],
  sanitized_fixture_only: true,
  authority: "PIXEL_CAPTURE_ROSTER_AND_BYTES_ONLY_NO_VISUAL_QUALITY_COMPREHENSION_OR_SECURITY_AUTHORITY",
  images,
};
const manifestPath = resolve(directory, "MANIFEST.json");
const serialized = JSON.stringify(manifest, null, 2) + "\n";
if (mode === "verify") {
  if (await readFile(manifestPath, "utf8") !== serialized) {
    throw new Error(`${basename(directory)} manifest does not match its exact PNG roster`);
  }
  console.log(`${pass}: ${images.length} PNGs manifest-verified`);
} else {
  await writeFile(manifestPath, serialized, { mode: 0o644, flag: "wx" });
  console.log(`${pass}: ${images.length} PNGs manifest-bound`);
}
