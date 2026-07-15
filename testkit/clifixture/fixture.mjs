import {
  closeSync,
  existsSync,
  fsyncSync,
  openSync,
  readFileSync,
  writeSync,
} from "node:fs";
import { basename, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const fixtureVersion = "countershape-cli-precedence-fixture/v1";
const supportedPrecedence = new Set(["config-first", "env-first", "argv-first"]);
const supportedBehavior = new Set([
  "precedence",
  "alternating",
  "timeout",
  "output-limit",
  "signal",
  "malformed-projection",
  "missing-invocation",
  "malformed-invocation",
  "empty-output",
  "nonzero-exit",
]);

function die(message) {
  writeSync(2, Buffer.from(`fixture error: ${message}`, "utf8"));
  process.exit(64);
}

function requiredEnvironment(name) {
  if (!Object.prototype.hasOwnProperty.call(process.env, name) || process.env[name] === "") {
    die(`missing runner environment ${name}`);
  }
  return process.env[name];
}

function readClosedJSONObject(path, allowedKeys) {
  let parsed;
  try {
    parsed = JSON.parse(readFileSync(path, "utf8"));
  } catch {
    die(`cannot read strict JSON ${basename(path)}`);
  }
  if (parsed === null || Array.isArray(parsed) || typeof parsed !== "object") {
    die(`${basename(path)} must be a JSON object`);
  }
  const keys = Object.keys(parsed).sort();
  const allowed = [...allowedKeys].sort();
  if (keys.length !== allowed.length || keys.some((key, index) => key !== allowed[index])) {
    die(`${basename(path)} has an unsupported shape`);
  }
  return parsed;
}

function writeAll(fd, bytes) {
  let offset = 0;
  while (offset < bytes.length) {
    const count = writeSync(fd, bytes, offset, bytes.length - offset);
    if (count < 1) {
      die("invocation receipt write made no progress");
    }
    offset += count;
  }
}

function writeInvocationBytes(payload) {
  const evidenceRoot = requiredEnvironment("COUNTERSHAPE_EVIDENCE_ROOT");
  const receiptPath = join(evidenceRoot, "cli-invocation.json");
  let receiptFD;
  try {
    receiptFD = openSync(receiptPath, "wx", 0o600);
    writeAll(receiptFD, payload);
    fsyncSync(receiptFD);
  } catch {
    die("cannot create exclusive invocation receipt");
  } finally {
    if (receiptFD !== undefined) {
      closeSync(receiptFD);
    }
  }
  let directoryFD;
  try {
    directoryFD = openSync(evidenceRoot, "r");
    fsyncSync(directoryFD);
  } catch {
    die("cannot sync invocation receipt directory");
  } finally {
    if (directoryFD !== undefined) {
      closeSync(directoryFD);
    }
  }
}

function writeInvocationReceipt(logicalArgv) {
  const attemptID = requiredEnvironment("COUNTERSHAPE_ATTEMPT_ID");
  writeInvocationBytes(Buffer.from(JSON.stringify({
    attempt_id: attemptID,
    logical_argv: logicalArgv,
  }), "utf8"));
}

function parseArguments(argv) {
  const result = {
    behavior: "precedence",
    emitBytes: 2 * 1024 * 1024,
    mode: { present: false, value: "" },
    stderr: "",
  };
  for (let index = 0; index < argv.length; index += 1) {
    switch (argv[index]) {
      case "--mode":
        if (index + 1 >= argv.length || result.mode.present) {
          die("--mode requires exactly one value");
        }
        result.mode = { present: true, value: argv[index + 1] };
        index += 1;
        break;
      case "--clear-mode":
        if (result.mode.present) {
          die("mode was supplied more than once");
        }
        result.mode = { present: true, value: "" };
        break;
      case "--behavior":
        if (index + 1 >= argv.length || !supportedBehavior.has(argv[index + 1])) {
          die("--behavior has an unsupported value");
        }
        result.behavior = argv[index + 1];
        index += 1;
        break;
      case "--emit-bytes": {
        if (index + 1 >= argv.length || !/^[1-9][0-9]{0,7}$/.test(argv[index + 1])) {
          die("--emit-bytes requires a bounded positive decimal");
        }
        const amount = Number(argv[index + 1]);
        if (!Number.isSafeInteger(amount) || amount > 16 * 1024 * 1024) {
          die("--emit-bytes exceeds the fixture ceiling");
        }
        result.emitBytes = amount;
        index += 1;
        break;
      }
      case "--stderr":
        if (index + 1 >= argv.length) {
          die("--stderr requires a value");
        }
        result.stderr = argv[index + 1];
        index += 1;
        break;
      default:
        die(`unsupported argument ${argv[index]}`);
    }
  }
  return result;
}

function selectedValue(precedence, argvMode, configMode) {
  const envMode = Object.prototype.hasOwnProperty.call(process.env, "APP_MODE")
    ? { present: true, value: process.env.APP_MODE }
    : { present: false, value: "" };
  const sources = {
    argv: argvMode,
    config: { present: true, value: configMode },
    env: envMode,
  };
  const order = {
    "config-first": ["config", "env", "argv"],
    "env-first": ["env", "argv", "config"],
    "argv-first": ["argv", "env", "config"],
  }[precedence];
  for (const source of order) {
    if (sources[source].present) {
      return { mode: sources[source].value, source };
    }
  }
  die("precedence produced no value");
}

const candidateRoot = dirname(fileURLToPath(import.meta.url));
const role = readClosedJSONObject(join(candidateRoot, "candidate-role.json"), ["precedence"]);
if (typeof role.precedence !== "string" || !supportedPrecedence.has(role.precedence)) {
  die("candidate role has unsupported precedence");
}
const rawArgv = process.argv.slice(2);
const options = parseArguments(rawArgv);
// Drain fd 0 before emitting behavior. This makes the physical present-empty
// and present-bytes study path deterministic: the runner must finish its exact
// pipe handoff rather than racing a child that never reads stdin.
void readFileSync(0);
const logicalArgv = ["node", basename(process.argv[1]), ...rawArgv];
if (options.behavior === "malformed-invocation") {
  writeInvocationBytes(Buffer.from('{"attempt_id":', "utf8"));
} else if (options.behavior !== "missing-invocation") {
  writeInvocationReceipt(logicalArgv);
}

if (options.stderr !== "") {
  writeSync(2, Buffer.from(options.stderr, "utf8"));
}

switch (options.behavior) {
  case "timeout":
    setInterval(() => {}, 60_000);
    break;
  case "output-limit":
    writeAll(1, Buffer.alloc(options.emitBytes, 0x78));
    break;
  case "signal":
    process.kill(process.pid, "SIGTERM");
    setInterval(() => {}, 60_000);
    break;
  case "malformed-projection":
    writeSync(1, Buffer.from('{"mode":', "utf8"));
    break;
  case "missing-invocation":
  case "malformed-invocation":
    writeSync(1, Buffer.from('{"mode":"would-project","source":"argv"}', "utf8"));
    break;
  case "empty-output":
    break;
  case "alternating": {
    const repetition = requiredEnvironment("COUNTERSHAPE_SCHEDULE_REPETITION");
    if (!/^(0|[1-9][0-9]*)$/.test(repetition)) {
      die("schedule repetition is not canonical decimal");
    }
    const even = BigInt(repetition) % 2n === 0n;
    writeSync(1, Buffer.from(JSON.stringify({
      mode: even ? "alternating-a" : "alternating-b",
      source: "schedule-repetition",
    }), "utf8"));
    break;
  }
  case "precedence":
  case "nonzero-exit": {
    const fixtureRoot = requiredEnvironment("COUNTERSHAPE_FIXTURE_ROOT");
    const config = readClosedJSONObject(join(fixtureRoot, "config.json"), ["mode"]);
    if (typeof config.mode !== "string") {
      die("config mode must be a string");
    }
    let configMode = config.mode;
    const privateHomeSentinel = join(requiredEnvironment("HOME"), ".countershape-precedence.json");
    if (existsSync(privateHomeSentinel)) {
      const homeConfig = readClosedJSONObject(privateHomeSentinel, ["mode"]);
      if (typeof homeConfig.mode !== "string") {
        die("private HOME config mode must be a string");
      }
      configMode = homeConfig.mode;
    }
    const selected = selectedValue(role.precedence, options.mode, configMode);
    writeSync(1, Buffer.from(JSON.stringify(selected), "utf8"));
    if (options.behavior === "nonzero-exit") {
      process.exitCode = 7;
    }
    break;
  }
  default:
    die(`${fixtureVersion} reached an impossible behavior`);
}
