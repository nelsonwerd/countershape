#!/usr/bin/env node

// Compatibility entrypoint. It accepts no repository/root override and never
// performs PATH lookup. The same fingerprinted, native, explicitly trusted Go
// toolchain admission used by the mutation gate is required before go run.
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
    throw new Error("BOUNDARY_ARGUMENTS: only the fixed --self-test mode is accepted");
  }

  const toolchain = await admitTrustedGoToolchain();
  const cacheParent = await mkdtemp(join(tmpdir(), "countershape-u1-boundary-cache-"));
  await chmod(cacheParent, 0o700);
  try {
    for (const directory of ["home", "tmp", "go-tmp", "go-path", "go-build", "go-mod"]) {
      await mkdir(join(cacheParent, directory), { recursive: true, mode: 0o700 });
      await chmod(join(cacheParent, directory), 0o700);
    }
    // Compile only the reviewed analyzer entrypoint. Package-form `go run
    // ./tools/u1boundary` would compile an attacker-added sibling first, giving
    // its init function authority to exit successfully before topology scanning.
    const args = ["run", "./tools/u1boundary/main.go", "--root", root];
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
      throw new Error(`BOUNDARY_TOOL_EXEC: ${result.error.message}`);
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
