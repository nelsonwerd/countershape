import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { startStudio } from "../helpers/studio";

const blindSurfaces = ["ORIGINAL_WITNESS", "MINIMIZED_WITNESS", "REDUCTION_DERIVATION", "PROJECTION_OPERATIONS", "NONASSERTED_FIELDS"];

test("320px and 200-percent-equivalent layouts preserve evidence and safe actions", async ({ browser }) => {
  for (const shape of [
    { name: "320px", width: 320, height: 812, deviceScaleFactor: 1 },
    { name: "1280px intermediate", width: 1280, height: 720, deviceScaleFactor: 1 },
    { name: "200-percent-equivalent", width: 720, height: 450, deviceScaleFactor: 2 },
  ]) {
    const studio = await startStudio();
    const context = await browser.newContext({
      viewport: { width: shape.width, height: shape.height },
      deviceScaleFactor: shape.deviceScaleFactor,
      colorScheme: "dark",
      reducedMotion: "reduce",
    });
    const page = await context.newPage();
    try {
      await page.goto(studio.url);
      await expect(page.getByRole("heading", { name: "Decision-ready evidence" })).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), shape.name).toBe(true);
      await expect(page.getByText("Trusted code boundary", { exact: true })).toBeVisible();
      await expect(page.getByLabel("Exact tuple-set preview")).toContainText("Not asserted");
      if (shape.width <= 760) {
        const switcher = page.getByLabel("Outcome to inspect");
        await switcher.getByRole("button", { name: "Outcome B" }).click();
        await expect(page.locator(".candidate-card.mobile-active")).toContainText("Outcome card");
        await expect(page.locator(".candidate-card:visible")).toHaveCount(1);
        await expect(page.getByRole("link", { name: /Review ruling/ })).toBeVisible();
      }
      const undersized = await page.locator("button:visible").evaluateAll((buttons) => buttons
        .map((button) => ({ name: button.textContent?.trim() ?? "", height: button.getBoundingClientRect().height }))
        .filter((item) => item.height < 44));
      expect(undersized, `${shape.name} button targets`).toEqual([]);
    } finally {
      await context.close();
      await studio.stop();
    }
  }
});

test("forced colors and reduced motion preserve non-color state cues", async ({ browser }) => {
  const studio = await startStudio();
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, forcedColors: "active", reducedMotion: "reduce" });
  const page = await context.newPage();
  try {
    await page.goto(studio.url);
    expect(await page.evaluate(() => matchMedia("(forced-colors: active)").matches)).toBe(true);
    expect(await page.evaluate(() => matchMedia("(prefers-reduced-motion: reduce)").matches)).toBe(true);
    await expect(page.getByText("Provenance hidden", { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Allow", exact: true }).first().click();
    await expect(page.getByRole("button", { name: "Selected", exact: true })).toBeVisible();
    const axe = await new AxeBuilder({ page }).analyze();
    expect(axe.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical")).toEqual([]);
  } finally {
    await context.close();
    await studio.stop();
  }
});

test("package transitions restore focus to the state heading", async ({ page }) => {
  const studio = await startStudio();
  try {
    await page.goto(studio.url);
    await reviewBlind(page);
    await page.getByRole("button", { name: "Allow", exact: true }).first().click();
    for (const row of await page.locator(".ack-row").all()) await row.getByRole("button", { name: "Assert" }).click();
    await page.getByRole("button", { name: "Record provisional ruling" }).click();
    await expect(page.locator("main h1")).toBeFocused();
    await page.getByRole("button", { name: "Reveal candidate provenance" }).click();
    await expect(page.getByRole("heading", { name: "Identity reveal & affirmation" })).toBeFocused();
  } finally {
    await studio.stop();
  }
});

async function reviewBlind(page: Page) {
  for (const surfaceID of blindSurfaces) {
    const surface = page.locator(`details[data-surface-id='${surfaceID}']`);
    await surface.locator("summary").click();
    await expect(surface.getByText("Reviewed", { exact: true })).toBeVisible();
  }
}
