import { once } from "node:events";
import {
  canonicalizeJSON,
  evaluateParityOperation,
  parseCanonicalJSON,
} from "../program/v1/harness.mjs";

const MAX_FRAME_BYTES = 64 << 10;
const MAX_FRAME_BODY_BYTES = MAX_FRAME_BYTES - 1;

async function readOneFrame() {
  const chunks = [];
  let observed = 0;
  process.stdin.on("data", (chunk) => {
    observed += chunk.length;
    if (observed <= MAX_FRAME_BYTES) chunks.push(Buffer.from(chunk));
  });
  const completion = await Promise.race([
    once(process.stdin, "end").then(() => "end"),
    once(process.stdin, "error").then(() => "error"),
  ]);
  if (completion !== "end" || observed < 2 || observed > MAX_FRAME_BYTES) throw new Error("invalid request frame");
  const frame = Buffer.concat(chunks, observed);
  if (frame[frame.length - 1] !== 0x0a) throw new Error("missing request terminator");
  const body = frame.subarray(0, -1);
  if (body.length === 0 || body.length > MAX_FRAME_BODY_BYTES) throw new Error("invalid request body size");
  if (body.includes(0x0a) || body.includes(0x0d)) throw new Error("multiple or non-LF request framing");
  return parseCanonicalJSON(body);
}

async function writeOneFrame(value) {
  const encoded = canonicalizeJSON(Buffer.from(JSON.stringify(value), "utf8"));
  if (encoded.length === 0 || encoded.length > MAX_FRAME_BODY_BYTES) throw new Error("response frame exceeds protocol cap");
  const frame = Buffer.concat([encoded, Buffer.from("\n")]);
  if (!process.stdout.write(frame)) await once(process.stdout, "drain");
}

try {
  const request = await readOneFrame();
  await writeOneFrame(evaluateParityOperation(request));
} catch {
  process.exitCode = 1;
}
