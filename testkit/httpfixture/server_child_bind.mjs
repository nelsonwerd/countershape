import { createHash } from "node:crypto";
import {
  closeSync,
  existsSync,
  fsyncSync,
  openSync,
  readFileSync,
  writeSync,
} from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { createServer } from "node:net";

const fixtureVersion = "countershape-http-invoice-fixture/v1";
const portableReadinessFD = 3;
const maxRequestBytes = 64 * 1024;
const supportedRoles = new Set([
  "forbidden",
  "conceal-not-found",
  "metadata-disclosure",
  "alternating",
]);

function writeAll(fd, bytes) {
  let offset = 0;
  while (offset < bytes.length) {
    const count = writeSync(fd, bytes, offset, bytes.length - offset);
    if (count < 1) {
      throw new Error("write made no progress");
    }
    offset += count;
  }
}

function die(message) {
  try {
    writeAll(2, Buffer.from(`fixture error: ${message}`, "utf8"));
  } catch {
    // The fixture is already failing; preserve the original reason.
  }
  process.exit(64);
}

function requiredEnvironment(name) {
  if (!Object.prototype.hasOwnProperty.call(process.env, name) || process.env[name] === "") {
    die(`missing runner environment ${name}`);
  }
  return process.env[name];
}

function readJSONObject(path) {
  let parsed;
  try {
    parsed = JSON.parse(readFileSync(path, "utf8"));
  } catch {
    die(`cannot read strict JSON ${path}`);
  }
  if (parsed === null || Array.isArray(parsed) || typeof parsed !== "object") {
    die(`${path} must be a JSON object`);
  }
  return parsed;
}

function hasExactKeys(parsed, allowedKeys) {
  const keys = Object.keys(parsed).sort();
  const allowed = [...allowedKeys].sort();
  return keys.length === allowed.length && keys.every((key, index) => key === allowed[index]);
}

function readClosedJSONObject(path, allowedKeys) {
  const parsed = readJSONObject(path);
  if (!hasExactKeys(parsed, allowedKeys)) {
    die(`${path} has an unsupported shape`);
  }
  return parsed;
}

function syncExclusive(path, payload) {
  let fd;
  try {
    fd = openSync(path, "wx", 0o600);
    writeAll(fd, payload);
    fsyncSync(fd);
  } catch {
    die(`cannot create exclusive evidence ${path}`);
  } finally {
    if (fd !== undefined) {
      closeSync(fd);
    }
  }
  let directoryFD;
  try {
    directoryFD = openSync(dirname(path), "r");
    fsyncSync(directoryFD);
  } catch {
    die(`cannot sync evidence directory ${dirname(path)}`);
  } finally {
    if (directoryFD !== undefined) {
      closeSync(directoryFD);
    }
  }
}

function validateSeed(seed) {
  const fullSeed = hasExactKeys(seed, ["invoice_id", "metadata", "owner_tenant"]);
  const tenantlessSeed = hasExactKeys(seed, ["invoice_id", "metadata"]);
  if (
    (!fullSeed && !tenantlessSeed) ||
    seed.invoice_id !== "inv-204" ||
    seed.metadata === null || Array.isArray(seed.metadata) || typeof seed.metadata !== "object"
  ) {
    die("invoice seed values have an unsupported shape");
  }
  const metadataKeys = Object.keys(seed.metadata).sort();
  const expectedMetadataKeys = fullSeed
    ? "amount_cents\ncurrency\nowner_tenant"
    : "amount_cents\ncurrency";
  if (
    metadataKeys.join("\n") !== expectedMetadataKeys ||
    seed.metadata.amount_cents !== 4200 || seed.metadata.currency !== "USD"
  ) {
    die("invoice metadata has an unsupported shape");
  }
  if (fullSeed && (
    seed.owner_tenant !== "tenant-b" || seed.metadata.owner_tenant !== "tenant-b"
  )) {
    die("invoice tenant seed values disagree");
  }
  return fullSeed;
}

function decodeCanonicalQueryComponent(raw) {
  const decoded = [];
  for (let index = 0; index < raw.length; index += 1) {
    const code = raw.charCodeAt(index);
    if (
      (code >= 0x41 && code <= 0x5a) ||
      (code >= 0x61 && code <= 0x7a) ||
      (code >= 0x30 && code <= 0x39) ||
      raw[index] === "-" || raw[index] === "." || raw[index] === "_" || raw[index] === "~"
    ) {
      decoded.push(code);
      continue;
    }
    if (raw[index] !== "%" || index + 2 >= raw.length || !/^[0-9A-F]{2}$/.test(raw.slice(index + 1, index + 3))) {
      die("query component is outside the canonical fixture profile");
    }
    const value = Number.parseInt(raw.slice(index + 1, index + 3), 16);
    if (value < 0x20 || value > 0x7e) {
      die("query component is outside the fixture ASCII profile");
    }
    if (
      (value >= 0x41 && value <= 0x5a) ||
      (value >= 0x61 && value <= 0x7a) ||
      (value >= 0x30 && value <= 0x39) ||
      value === 0x2d || value === 0x2e || value === 0x5f || value === 0x7e
    ) {
      die("query component uses a noncanonical escaped unreserved byte");
    }
    decoded.push(value);
    index += 2;
  }
  return Buffer.from(decoded).toString("ascii");
}

function parseOrderedQuery(raw) {
  if (raw === undefined) {
    return [];
  }
  if (raw === "") {
    die("empty query suffix is outside the fixture profile");
  }
  return raw.split("&").map((member) => {
    if (member === "") {
      die("empty ordered query member is outside the fixture profile");
    }
    const separator = member.indexOf("=");
    const rawName = separator < 0 ? member : member.slice(0, separator);
    const rawValue = separator < 0 ? "" : member.slice(separator + 1);
    if (rawName === "") {
      die("query name is empty");
    }
    return Object.freeze({
      name: decodeCanonicalQueryComponent(rawName),
      value_presence: separator < 0 ? "ABSENT" : "PRESENT",
      value: decodeCanonicalQueryComponent(rawValue),
    });
  });
}

function parseRequestFrameIfComplete(bytes) {
  const headerEnd = bytes.indexOf("\r\n\r\n");
  if (headerEnd < 0) {
    return null;
  }
  const head = bytes.subarray(0, headerEnd).toString("latin1");
  const lines = head.split("\r\n");
  if (lines.length < 2 || lines.length > 64 || !/^[A-Z]+ [^ ]+ HTTP\/1\.1$/.test(lines[0])) {
    die("request line is outside the fixture profile");
  }
  const [method, target, version] = lines[0].split(" ");
  if (version !== "HTTP/1.1" || !/^\/[\x21-\x7e]*$/.test(target) || target.includes("#")) {
    die("request target is outside the fixture profile");
  }
  const querySeparator = target.indexOf("?");
  const path = querySeparator < 0 ? target : target.slice(0, querySeparator);
  const rawQuery = querySeparator < 0 ? undefined : target.slice(querySeparator + 1);
  if (path === "" || path.includes("%")) {
    die("request path is outside the fixture profile");
  }

  let contentLength = null;
  const orderedHeaders = [];
  for (const line of lines.slice(1)) {
    const split = line.indexOf(":");
    if (
      split < 1 || line.slice(split, split + 2) !== ": " ||
      /[^\x20-\x7e]/.test(line) ||
      !/^[!#$%&'*+.^_`|~0-9a-z-]+$/.test(line.slice(0, split))
    ) {
      die("request header is malformed");
    }
    const name = line.slice(0, split);
    const value = line.slice(split + 2);
    if (name === "transfer-encoding") {
      die("transfer encoding is outside the fixture profile");
    }
    if (name === "content-length") {
      if (contentLength !== null || !/^(0|[1-9][0-9]{0,7})$/.test(value)) {
        die("content length is malformed or duplicated");
      }
      contentLength = Number(value);
    }
    orderedHeaders.push(Object.freeze({ name, value }));
  }
  const bodyBytes = contentLength === null ? 0 : contentLength;
  const total = headerEnd + 4 + bodyBytes;
  if (total > maxRequestBytes) {
    die("request exceeds the fixture bound");
  }
  if (bytes.length < total) {
    return null;
  }
  const body = Buffer.from(bytes.subarray(headerEnd + 4, total));
  return Object.freeze({
    length: total,
    request: Object.freeze({
      method,
      path,
      ordered_query: Object.freeze(parseOrderedQuery(rawQuery)),
      ordered_headers: Object.freeze(orderedHeaders),
      body: Object.freeze({
        presence: contentLength === null ? "ABSENT" : "PRESENT",
        bytes: body,
      }),
    }),
  });
}

function exactOrderedMembers(actual, expected) {
  return actual.length === expected.length && actual.every((member, index) => (
    member.name === expected[index].name &&
    (member.value_presence ?? "PRESENT") === (expected[index].value_presence ?? "PRESENT") &&
    member.value === expected[index].value
  ));
}

function valuesFor(members, name) {
  return members.filter((member) => member.name === name).map((member) => ({
    presence: member.value_presence ?? "PRESENT",
    value: member.value,
  }));
}

function exactValues(actual, expected) {
  return actual.length === expected.length && actual.every((entry, index) => (
    entry.presence === expected[index].presence && entry.value === expected[index].value
  ));
}

function requestPolicyBehavior(request, seed) {
  if (request.method !== "GET") {
    return { status: 405, phrase: "Method Not Allowed", kind: "method_not_allowed" };
  }
  if (request.path !== `/v1/invoices/${seed.invoice_id}`) {
    return { status: 404, phrase: "Not Found", kind: "route_not_found" };
  }
  if (request.body.presence !== "ABSENT" || request.body.bytes.length !== 0) {
    return { status: 415, phrase: "Unsupported Media Type", kind: "body_not_allowed" };
  }

  const query = request.ordered_query;
  const headers = request.ordered_headers;
  const hostHeaders = valuesFor(headers, "host");
  const hostValue = hostHeaders[0]?.value ?? "";
  const hostMatch = /^127\.0\.0\.1:([1-9][0-9]{0,4})$/.exec(hostValue);
  if (!exactValues(hostHeaders, [{ presence: "PRESENT", value: hostValue }]) ||
      hostMatch === null || Number(hostMatch[1]) > 65535 ||
      !exactValues(valuesFor(headers, "connection"), [{ presence: "PRESENT", value: "close" }])) {
    return { status: 400, phrase: "Bad Request", kind: "wire_contract_error" };
  }
  if (!exactValues(valuesFor(headers, "accept"), [{ presence: "PRESENT", value: "application/json" }])) {
    return { status: 406, phrase: "Not Acceptable", kind: "accept_contract_error" };
  }

  const actorQuery = valuesFor(query, "actor_tenant");
  const tenantHeader = valuesFor(headers, "x-countershape-tenant");
  if (!exactValues(actorQuery, [{ presence: "PRESENT", value: "tenant-a" }]) ||
      !exactValues(tenantHeader, [{ presence: "PRESENT", value: "tenant-a" }]) ||
      actorQuery[0].value !== tenantHeader[0].value) {
    return { status: 400, phrase: "Bad Request", kind: "tenant_context_mismatch" };
  }

  const roleQuery = valuesFor(query, "role");
  const roleHeader = valuesFor(headers, "x-countershape-role");
  if (!exactValues(roleQuery, [{ presence: "PRESENT", value: "support" }]) ||
      !exactValues(roleHeader, [{ presence: "PRESENT", value: "support" }]) ||
      roleQuery[0].value !== roleHeader[0].value) {
    return { status: 403, phrase: "Forbidden", kind: "role_not_authorized" };
  }

  const auditQuery = valuesFor(query, "audit");
  const auditHeader = valuesFor(headers, "x-countershape-audit");
  if (!exactValues(auditQuery, [{ presence: "ABSENT", value: "" }]) ||
      !exactValues(auditHeader, [{ presence: "PRESENT", value: "required" }])) {
    return { status: 428, phrase: "Precondition Required", kind: "audit_contract_error" };
  }

  const expectedTags = [
    { presence: "PRESENT", value: "first" },
    { presence: "PRESENT", value: "second" },
  ];
  if (!exactValues(valuesFor(query, "tag"), expectedTags) ||
      !exactValues(valuesFor(headers, "x-countershape-tag"), expectedTags)) {
    return { status: 422, phrase: "Unprocessable Content", kind: "tag_policy_mismatch" };
  }
  if (!exactValues(valuesFor(headers, "x-countershape-trace"), expectedTags) ||
      !exactValues(valuesFor(headers, "x-countershape-present-empty"), [{ presence: "PRESENT", value: "" }])) {
    return { status: 400, phrase: "Bad Request", kind: "header_contract_error" };
  }

  const expectedQuery = [
    { name: "actor_tenant", value_presence: "PRESENT", value: "tenant-a" },
    { name: "role", value_presence: "PRESENT", value: "support" },
    { name: "tag", value_presence: "PRESENT", value: "first" },
    { name: "tag", value_presence: "PRESENT", value: "second" },
    { name: "audit", value_presence: "ABSENT", value: "" },
  ];
  const expectedHeaders = [
    { name: "host", value: hostValue },
    { name: "connection", value: "close" },
    { name: "accept", value: "application/json" },
    { name: "x-countershape-tenant", value: "tenant-a" },
    { name: "x-countershape-role", value: "support" },
    { name: "x-countershape-audit", value: "required" },
    { name: "x-countershape-tag", value: "first" },
    { name: "x-countershape-tag", value: "second" },
    { name: "x-countershape-trace", value: "first" },
    { name: "x-countershape-trace", value: "second" },
    { name: "x-countershape-present-empty", value: "" },
  ];
  if (!exactOrderedMembers(query, expectedQuery) || !exactOrderedMembers(headers, expectedHeaders)) {
    return { status: 400, phrase: "Bad Request", kind: "request_order_or_shape_error" };
  }
  return null;
}

function contaminationMarker(stateRoot) {
  return join(stateRoot, "authz-cache.json");
}

function writeConcealCache(stateRoot, invoiceID) {
  const payload = Buffer.from(JSON.stringify({
    decision: "conceal",
    invoice_id: invoiceID,
  }), "utf8");
  syncExclusive(contaminationMarker(stateRoot), payload);
}

function concealCachePresent(stateRoot, invoiceID) {
  const marker = contaminationMarker(stateRoot);
  if (!existsSync(marker)) {
    return false;
  }
  const cached = readClosedJSONObject(marker, ["decision", "invoice_id"]);
  if (cached.decision !== "conceal" || cached.invoice_id !== invoiceID) {
    die("authorization cache marker is malformed");
  }
  return true;
}

function selectedBehavior(role, repetition, seed, tenantSeedPresent, stateRoot) {
  let selected = role;
  if (role === "alternating") {
    if (!/^(0|[1-9][0-9]*)$/.test(repetition)) {
      die("schedule repetition is not canonical decimal");
    }
    selected = BigInt(repetition) % 2n === 0n
      ? "conceal-not-found"
      : "metadata-disclosure";
  }
  if (!tenantSeedPresent) {
    if (selected === "forbidden") {
      return { status: 200, phrase: "OK", kind: "authorized_without_tenant_seed" };
    }
    return { status: 500, phrase: "Internal Server Error", kind: "tenant_seed_missing" };
  }
  if (selected === "conceal-not-found") {
    writeConcealCache(stateRoot, seed.invoice_id);
    return { status: 404, phrase: "Not Found", kind: "not_found" };
  }
  if (selected === "metadata-disclosure") {
    if (concealCachePresent(stateRoot, seed.invoice_id)) {
      return { status: 404, phrase: "Not Found", kind: "not_found" };
    }
    return {
      status: 200,
      phrase: "OK",
      kind: "authorized_metadata",
      metadata: seed.metadata,
    };
  }
  if (selected === "forbidden") {
    return { status: 403, phrase: "Forbidden", kind: "forbidden" };
  }
  die("candidate role reached an impossible behavior");
}

const candidateRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const roleRecord = readClosedJSONObject(join(candidateRoot, "candidate-role.json"), ["role"]);
if (typeof roleRecord.role !== "string" || !supportedRoles.has(roleRecord.role)) {
  die("candidate role is unsupported");
}

const fixtureRoot = requiredEnvironment("COUNTERSHAPE_FIXTURE_ROOT");
const evidenceRoot = requiredEnvironment("COUNTERSHAPE_EVIDENCE_ROOT");
const stateRoot = requiredEnvironment("COUNTERSHAPE_STATE_ROOT");
const attemptID = requiredEnvironment("COUNTERSHAPE_ATTEMPT_ID");
const stimulusDigest = requiredEnvironment("COUNTERSHAPE_HTTP_STIMULUS_DIGEST");
const repetition = requiredEnvironment("COUNTERSHAPE_SCHEDULE_REPETITION");
const seed = readJSONObject(join(fixtureRoot, "invoice-seed.json"));
const tenantSeedPresent = validateSeed(seed);

let acceptedConnections = 0;
const server = createServer({ allowHalfOpen: false }, (socket) => {
  acceptedConnections += 1;
  if (acceptedConnections !== 1) {
    socket.destroy();
    die("fixture accepted more than one connection");
  }

  let requestBytes = Buffer.alloc(0);
  let handled = false;
  socket.on("data", (chunk) => {
    if (handled) {
      die("fixture received bytes after the exact request");
    }
    requestBytes = Buffer.concat([requestBytes, chunk]);
    if (requestBytes.length > maxRequestBytes) {
      die("request exceeds the fixture bound");
    }
    const requestFrame = parseRequestFrameIfComplete(requestBytes);
    if (requestFrame === null) {
      return;
    }
    if (requestFrame.length !== requestBytes.length) {
      die("fixture received bytes after the exact request");
    }
    handled = true;

    const requestSHA256 = `sha256:${createHash("sha256").update(requestBytes).digest("hex")}`;
    syncExclusive(
      join(evidenceRoot, "http-invocation.json"),
      Buffer.from(JSON.stringify({
        attempt_id: attemptID,
        invocation_count: 1,
        kind: "HTTPFixtureInvocationEvidence",
        request_byte_count: requestBytes.length,
        request_byte_sha256: requestSHA256,
        schema_version: "countershape/v1",
        stimulus_digest: stimulusDigest,
      }), "utf8"),
    );

    const behavior = requestPolicyBehavior(requestFrame.request, seed) ??
      selectedBehavior(roleRecord.role, repetition, seed, tenantSeedPresent, stateRoot);
    const bodyRecord = {
      fixture_version: fixtureVersion,
      invoice_id: seed.invoice_id,
      kind: behavior.kind,
      metadata: behavior.metadata ?? {},
    };
    bodyRecord.request_id = attemptID;
    bodyRecord.scratch_root = stateRoot;
    const body = Buffer.from(JSON.stringify(bodyRecord), "utf8");
    const responseHead = Buffer.from(
      `HTTP/1.1 ${behavior.status} ${behavior.phrase}\r\n` +
      "Content-Type: application/json\r\n" +
      `Content-Length: ${body.length}\r\n` +
      "Connection: close\r\n" +
      `X-Countershape-Fixture: ${fixtureVersion}\r\n\r\n`,
      "latin1",
    );
    socket.end(Buffer.concat([responseHead, body]));
    server.close();
  });
  socket.on("end", () => {
    if (!handled) {
      die("connection ended before one complete request");
    }
  });
  socket.on("error", () => die("fixture socket failed"));
});

server.on("error", () => die("fixture listener failed"));
const descriptor = (name, expected) => {
  const value = requiredEnvironment(name);
  if (!/^[1-9][0-9]*$/.test(value) || Number(value) !== expected) {
    die(`runner descriptor ${name} is outside the exact profile`);
  }
  return expected;
};

if (
  Object.prototype.hasOwnProperty.call(process.env, "COUNTERSHAPE_HTTP_LISTEN_FD") ||
  Object.prototype.hasOwnProperty.call(process.env, "COUNTERSHAPE_HTTP_PORT")
) {
  die("portable child-bind profile cannot inherit a listener or port");
}
const readinessFD = descriptor("COUNTERSHAPE_HTTP_READINESS_FD", portableReadinessFD);

server.listen({ host: "127.0.0.1", port: 0, exclusive: true }, () => {
  try {
    const address = server.address();
    if (
      address === null || typeof address !== "object" ||
      address.address !== "127.0.0.1" || !Number.isInteger(address.port) ||
      address.port < 1 || address.port > 65535
    ) {
      die("fixture did not bind literal IPv4 loopback");
    }
    writeAll(readinessFD, Buffer.from(`COUNTERSHAPE_READY_V1 ${address.port}\n`, "ascii"));
    closeSync(readinessFD);
  } catch {
    die("fixture readiness write failed");
  }
});
