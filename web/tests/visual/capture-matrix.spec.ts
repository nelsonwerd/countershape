import { chromium, expect, test, type Browser } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";
import { startStudio } from "../helpers/studio";

const captureRoot = process.env.COUNTERSHAPE_P10_CAPTURE_ROOT;
const passName = process.env.COUNTERSHAPE_P10_PASS ?? "unlabeled";

const states = [
  "empty", "preparing", "active", "partial", "error", "unstable", "uncomparable", "discovered",
  "decision-ready", "predicate-editing", "identity-reveal", "resolved", "reject-all-resolved", "deferred",
  "stale", "invalidated", "incomplete", "completed",
] as const;

test("captures the complete sanitized presentation matrix at both required viewports", async () => {
  test.skip(!captureRoot, "COUNTERSHAPE_P10_CAPTURE_ROOT is required for an evidence capture");
  test.setTimeout(8 * 60_000);
  await mkdir(captureRoot!, { recursive: true, mode: 0o700 });
  const browser = await chromium.launch({
    headless: true,
    args: ["--disable-gpu", "--disable-lcd-text", "--disable-font-subpixel-positioning", "--force-color-profile=srgb"],
  });
  try {
    for (const state of states) {
      await capture(browser, state, "desktop", { width: 1440, height: 900 });
      await capture(browser, state, "mobile", { width: 375, height: 812 });
    }
  } finally {
    await browser.close();
  }
});

async function capture(browser: Browser, state: string, viewportName: string, viewport: { width: number; height: number }) {
  const studio = await startStudio(state);
  const context = await browser.newContext({ viewport, colorScheme: "dark", reducedMotion: "reduce" });
  const page = await context.newPage();
  try {
    await page.goto(studio.url);
    await expect(page).toHaveURL(studio.origin + "/");
    await expect(page.locator("main h1")).toBeVisible();
    await expect(page.locator("main h1")).not.toHaveText("Opening the local evidence boundary");
    await expect(page.locator("body")).not.toContainText(studio.token);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({
      path: join(captureRoot!, `${passName}-${state}-${viewportName}.png`),
      fullPage: false,
      animations: "disabled",
    });
  } finally {
    await context.close();
    await studio.stop();
  }
}
