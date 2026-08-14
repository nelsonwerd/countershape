import { expect, test, type Browser, type Page } from "@playwright/test";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";
import { startStudio } from "../tests/helpers/studio";

const captureRoot = process.env.COUNTERSHAPE_P09_CAPTURE_ROOT;
if (!captureRoot) throw new Error("COUNTERSHAPE_P09_CAPTURE_ROOT is required");

test("captures sanitized real-binary inputs for the P10 visual loop", async ({ browser }) => {
  await mkdir(captureRoot, { recursive: true, mode: 0o700 });
  await captureState(browser, "decision-ready", { width: 1440, height: 900 }, "decision-ready-desktop.png");
  await captureState(browser, "decision-ready", { width: 375, height: 812 }, "decision-ready-mobile.png");
  await captureIdentityReveal(browser);
  await captureState(browser, "resolved", { width: 1440, height: 900 }, "resolved-desktop.png");
  await captureState(browser, "error", { width: 375, height: 812 }, "error-mobile.png");
});

async function captureState(browser: Browser, state: string, viewport: { width: number; height: number }, name: string) {
  const studio = await startStudio(state);
  const context = await browser.newContext({ viewport, colorScheme: "dark", reducedMotion: "reduce" });
  const page = await context.newPage();
  try {
    await page.goto(studio.url);
    await expect(page).toHaveURL(studio.origin + "/");
    await expect(page.getByRole("heading", { level: 1, name: headingFor(state) })).toBeVisible();
    await expect(page.locator("body")).not.toContainText(studio.token);
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({ path: join(captureRoot!, name), fullPage: true, animations: "disabled" });
  } finally {
    await context.close();
    await studio.stop();
  }
}

async function captureIdentityReveal(browser: Browser) {
  const studio = await startStudio("decision-ready");
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: "dark", reducedMotion: "reduce" });
  const page = await context.newPage();
  try {
    await page.goto(studio.url);
    for (const surfaceID of ["ORIGINAL_WITNESS", "MINIMIZED_WITNESS", "REDUCTION_DERIVATION", "PROJECTION_OPERATIONS", "NONASSERTED_FIELDS"]) {
      await openEvidenceSurface(page, surfaceID);
    }
    await page.getByRole("button", { name: "Allow", exact: true }).first().click();
    for (const row of await page.locator(".ack-row").all()) {
      await row.getByRole("button", { name: "Assert" }).click();
    }
    await page.getByRole("button", { name: "Record provisional ruling" }).click();
    await page.getByRole("button", { name: "Reveal candidate provenance" }).click();
    await expect(page.getByRole("heading", { name: "Identity reveal & affirmation" })).toBeVisible();
    await expect(page.locator(".identity-reveal").first()).toBeVisible();
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({ path: join(captureRoot!, "identity-reveal-desktop.png"), fullPage: true, animations: "disabled" });
  } finally {
    await context.close();
    await studio.stop();
  }
}

function headingFor(state: string) {
  if (state === "decision-ready") return "Decision-ready evidence";
  if (state === "resolved") return "Local ruling recorded";
  if (state === "error") return "Error";
  throw new Error(`capture state ${state} has no exact heading contract`);
}

async function openEvidenceSurface(page: Page, surfaceID: string) {
  const surface = page.locator(`details[data-surface-id='${surfaceID}']`);
  await surface.locator("summary").click();
  await expect(surface.getByText("Reviewed", { exact: true })).toBeVisible();
}
