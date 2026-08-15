import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process";
import { chmod, mkdtemp, readFile, rm, stat } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

export interface RunningStudio {
  origin: string;
  url: string;
  token: string;
  stop(): Promise<void>;
}

export async function startStudio(state = "decision-ready"): Promise<RunningStudio> {
  const binary = process.env.COUNTERSHAPE_STUDIO_BINARY;
  if (!binary) throw new Error("COUNTERSHAPE_STUDIO_BINARY is required");
  const root = await mkdtemp(join(tmpdir(), "countershape-studio-e2e-"));
  await chmod(root, 0o700);
  const launchFile = join(root, "launch.json");
  const child = spawn(binary, ["studio", "--no-open", "--seed-state", state, "--launch-file", launchFile], {
    env: { ...process.env, COUNTERSHAPE_STUDIO_TEST_LAUNCH: "1" },
    stdio: ["ignore", "pipe", "pipe"],
  });
  let stdout = "";
  let stderr = "";
  child.stdout.setEncoding("utf8");
  child.stderr.setEncoding("utf8");
  child.stdout.on("data", (value: string) => { stdout += value; });
  child.stderr.on("data", (value: string) => { stderr += value; });
  const launch = await waitForLaunch(child, launchFile, () => stderr);
  const parsed = new URL(launch.url);
  const token = new URLSearchParams(parsed.hash.slice(1)).get("access_token") ?? "";
  if (!/^[A-Za-z0-9_-]{43}$/.test(token) || stdout.includes(token) || stderr.includes(token)) {
    child.kill("SIGKILL");
    throw new Error("studio launch credential escaped its private channel");
  }
  const mode = (await stat(launchFile)).mode & 0o777;
  if (mode !== 0o600) {
    child.kill("SIGKILL");
    throw new Error(`launch file mode = ${mode.toString(8)}`);
  }
  return {
    origin: launch.origin,
    url: launch.url,
    token,
    async stop() {
      await stopChild(child);
      await rm(root, { recursive: true, force: false });
    },
  };
}

async function waitForLaunch(child: ChildProcessWithoutNullStreams, launchFile: string, stderr: () => string): Promise<{ origin: string; url: string }> {
  const deadline = Date.now() + 10_000;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error(`studio exited ${child.exitCode}: ${stderr()}`);
    try {
      const value = JSON.parse(await readFile(launchFile, "utf8")) as { origin?: unknown; url?: unknown };
      if (typeof value.origin === "string" && typeof value.url === "string") return { origin: value.origin, url: value.url };
    } catch {
      // The exclusive private launch file may not exist yet.
    }
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  child.kill("SIGKILL");
  throw new Error("studio launch file did not become ready");
}

async function stopChild(child: ChildProcessWithoutNullStreams): Promise<void> {
  if (child.exitCode === null) child.kill("SIGTERM");
  const result = await Promise.race([
    new Promise<{ code: number | null; signal: NodeJS.Signals | null }>((resolve) => child.once("exit", (code, signal) => resolve({ code, signal }))),
    new Promise<never>((_, reject) => setTimeout(() => reject(new Error("studio did not terminate after SIGTERM")), 8_000)),
  ]);
  if (result.code !== 0 || result.signal !== null) {
    throw new Error(`studio terminal status code=${result.code} signal=${result.signal}`);
  }
}
