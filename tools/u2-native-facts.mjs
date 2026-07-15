#!/opt/homebrew/Cellar/node/25.2.1/bin/node

import {
  chmod,
  mkdtemp,
  open,
  realpath,
  rm,
  stat,
  unlink,
} from "node:fs/promises";
import { createReadStream } from "node:fs";
import { createHash } from "node:crypto";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";

const expectedNode = "/opt/homebrew/Cellar/node/25.2.1/bin/node";
const goExecutable = "/opt/homebrew/Cellar/go/1.26.5/libexec/bin/go";
const gitExecutable = "/usr/bin/git";

function fail(code, detail) {
  throw new Error(`${code}: ${detail}`);
}

function runExact(executable, args, cwd, environment, { allowFailure = false } = {}) {
  const result = spawnSync(executable, args, {
    cwd,
    env: environment,
    encoding: "utf8",
    timeout: 30_000,
    maxBuffer: 1 << 20,
  });
  if (result.error) fail("NATIVE_FACT_TOOL", `${executable}: ${result.error.message}`);
  if (!allowFailure && result.status !== 0) {
    fail("NATIVE_FACT_TOOL", `${executable} exited ${result.status}: ${(result.stderr ?? "").trim()}`);
  }
  if (result.signal !== null) fail("NATIVE_FACT_SIGNAL", `${executable}: ${result.signal}`);
  return {
    status: result.status,
    stdout: result.stdout ?? "",
    stderr: result.stderr ?? "",
  };
}

function oneLine(value, label) {
  const line = value.endsWith("\n") ? value.slice(0, -1) : value;
  if (line.length < 1 || line.includes("\n") || line.includes("\r") || line.includes("\0")) {
    fail("NATIVE_FACT_FORMAT", `${label} is not one line`);
  }
  return line;
}

async function executableFact(path, versionArgs, cwd, environment) {
  const resolved = await realpath(path);
  const info = await stat(resolved);
  if (!info.isFile() || (info.mode & 0o111) === 0) fail("NATIVE_FACT_EXECUTABLE", path);
  if (info.size < 1 || info.size > 512 << 20) fail("NATIVE_FACT_EXECUTABLE_SIZE", path);
  const hasher = createHash("sha256");
  for await (const chunk of createReadStream(resolved)) hasher.update(chunk);
  const result = runExact(resolved, versionArgs, cwd, environment);
  return {
    requested_path: path,
    resolved_path: resolved,
    byte_count: info.size,
    sha256: hasher.digest("hex"),
    version: oneLine(result.stdout, path),
  };
}

async function aliasProbe(root, firstName, secondName) {
  const first = join(root, firstName);
  const second = join(root, secondName);
  const firstHandle = await open(first, "wx", 0o600);
  await firstHandle.close();
  let aliases = false;
  try {
    const secondHandle = await open(second, "wx", 0o600);
    await secondHandle.close();
  } catch (error) {
    if (error?.code !== "EEXIST") throw error;
    aliases = true;
  }
  await unlink(first);
  if (!aliases) await unlink(second);
  return aliases;
}

function parseFilesystemFact(dfOutput, diskutilOutput) {
  const lines = dfOutput.trim().split("\n");
  if (lines.length !== 2) fail("NATIVE_FACT_DF", `unexpected df output: ${dfOutput}`);
  const fields = lines[1].trim().split(/\s+/u);
  if (fields.length < 6 || !fields[0].startsWith("/dev/")) fail("NATIVE_FACT_DF", lines[1]);
  const personality = /^\s*File System Personality:\s*(.+)$/mu.exec(diskutilOutput)?.[1]?.trim();
  const bundle = /^\s*Type \(Bundle\):\s*(.+)$/mu.exec(diskutilOutput)?.[1]?.trim();
  if (!personality || !bundle) fail("NATIVE_FACT_DISKUTIL", "filesystem type fields are absent");
  return {
    device: fields[0],
    mount_point: fields.slice(5).join(" "),
    personality,
    bundle,
  };
}

async function main() {
  if (process.argv.length !== 2) fail("NATIVE_FACT_ARGUMENTS", "arguments are forbidden");
  if (process.platform !== "darwin" || process.arch !== "arm64") {
    fail("NATIVE_FACT_PLATFORM", `running ${process.platform}/${process.arch}, expected darwin/arm64`);
  }
  if ((await realpath(process.execPath)) !== expectedNode) {
    fail("NATIVE_FACT_NODE", `running ${process.execPath}, expected ${expectedNode}`);
  }
  const root = await mkdtemp(join(tmpdir(), "countershape-u2-native-facts-"));
  await chmod(root, 0o700);
  const environment = {
    HOME: root,
    TMPDIR: root,
    LANG: "C",
    LC_ALL: "C",
    TZ: "UTC",
    NO_COLOR: "1",
    PATH: "/usr/bin:/bin",
    GIT_CONFIG_NOSYSTEM: "1",
    GIT_CONFIG_GLOBAL: "/dev/null",
    GIT_TERMINAL_PROMPT: "0",
    GIT_NO_LAZY_FETCH: "1",
    GIT_NO_REPLACE_OBJECTS: "1",
    GOENV: "off",
    GOWORK: "off",
    GOTOOLCHAIN: "local",
    GOPROXY: "off",
  };
  try {
    const uname = oneLine(runExact("/usr/bin/uname", ["-mrs"], root, environment).stdout, "uname");
    const osVersion = oneLine(
      runExact("/usr/bin/sw_vers", ["-productVersion"], root, environment).stdout,
      "sw_vers",
    );
    const git = await executableFact(gitExecutable, ["--version"], root, environment);
    const go = await executableFact(goExecutable, ["version"], root, environment);
    const node = await executableFact(expectedNode, ["--version"], root, environment);

    const objectFormats = {};
    for (const format of ["sha1", "sha256"]) {
      const target = join(root, `git-${format}`);
      const result = runExact(
        git.resolved_path,
        ["init", "--bare", "--quiet", `--object-format=${format}`, target],
        root,
        environment,
        { allowFailure: true },
      );
      objectFormats[format] = {
        supported: result.status === 0,
        failure: result.status === 0 ? null : result.stderr.trim().slice(0, 512),
      };
    }

    const df = runExact("/bin/df", ["-P", root], root, environment).stdout;
    const device = df.trim().split("\n")[1]?.trim().split(/\s+/u)[0];
    if (!device?.startsWith("/dev/")) fail("NATIVE_FACT_DF", df);
    const diskutil = runExact("/usr/sbin/diskutil", ["info", device], root, environment).stdout;
    const filesystem = parseFilesystemFact(df, diskutil);
    filesystem.case_aliases = await aliasProbe(root, "CountershapeCaseProbe", "countershapecaseprobe");
    filesystem.unicode_nfc_nfd_aliases = await aliasProbe(root, "caf\u00e9", "cafe\u0301");
    filesystem.device_number = String((await stat(root)).dev);

    const facts = {
      schema_version: "countershape/u2-native-facts/v1",
      kind: "U2NativeEvidenceProfile",
      platform: process.platform,
      architecture: process.arch,
      uname,
      os_version: osVersion,
      filesystem,
      git,
      git_object_formats: objectFormats,
      go,
      node,
      environment_contract: "ENV_I_CLOSED_PROFILE",
    };
    process.stdout.write(`${JSON.stringify(facts, null, 2)}\n`);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
