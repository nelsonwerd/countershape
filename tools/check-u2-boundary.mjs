#!/usr/bin/env node

// Fixed-root compatibility entrypoint. It accepts no repository override and
// never performs PATH lookup. The boundary analyzer is compiled as one exact
// reviewed file after the U1 toolchain admission has pinned executable bytes.
import { chmod, mkdir, mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

import {
  admitTrustedGoToolchain,
  minimalGoEnvironment,
  revalidateAdmittedGoExecutable,
} from "./mutate-u1.mjs";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");

async function main() {
  const callerArguments = process.argv.slice(2);
  if (
    callerArguments.length > 1 ||
    (callerArguments.length === 1 && callerArguments[0] !== "--self-test")
  ) {
    throw new Error("U2_BOUNDARY_ARGUMENTS: only the fixed --self-test mode is accepted");
  }

  const toolchain = await admitTrustedGoToolchain();
  const cacheParent = await mkdtemp(join(tmpdir(), "countershape-u2-boundary-cache-"));
  await chmod(cacheParent, 0o700);
  try {
    for (const directory of ["home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"]) {
      await mkdir(join(cacheParent, directory), { recursive: true, mode: 0o700 });
      await chmod(join(cacheParent, directory), 0o700);
    }
    const args = ["run", "./tools/u2boundary/main.go", "--root", root];
    if (callerArguments[0] === "--self-test") args.push("--self-test");

    revalidateAdmittedGoExecutable(toolchain.executable);
    let result;
    try {
      result = spawnSync(toolchain.executable.path, args, {
        cwd: root,
        env: minimalGoEnvironment(cacheParent, toolchain),
        stdio: "inherit",
        timeout: 120_000,
      });
    } finally {
      revalidateAdmittedGoExecutable(toolchain.executable);
    }
    if (result.error) {
      throw new Error(`U2_BOUNDARY_TOOL_EXEC: ${result.error.message}`);
    }
    process.exitCode = result.status ?? 1;
  } finally {
    await rm(cacheParent, { recursive: true, force: true });
  }
}

main().catch((error) => {
  process.stderr.write(`${error.stack ?? error}\n`);
  process.exitCode = 1;
});
