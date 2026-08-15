import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { startStudio } from "../helpers/studio";

const presentationStates = [
  "empty", "preparing", "active", "partial", "error", "completed", "unstable", "uncomparable", "incomplete", "discovered",
  "decision-ready", "predicate-editing", "identity-reveal", "resolved", "reject-all-resolved", "deferred", "stale", "invalidated",
];

test("every closed presentation state renders without horizontal overflow", async ({ page }) => {
  test.setTimeout(120_000);
  for (const state of presentationStates) {
    const studio = await startStudio(state);
    try {
      await page.goto(studio.url);
      await expect(page.locator("main h1")).toBeVisible();
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
      await expect(page.locator("body")).not.toContainText(studio.token);
    } finally {
      await studio.stop();
    }
  }
});

test("reject-all completes with keyboard activation only", async ({ page }) => {
  const studio = await startStudio();
  try {
    await page.goto(studio.url);
    for (const title of ["Original witness", "Minimized witness", "Reduction derivation", "Captured → Projection → Operations", "Nonasserted fields"]) {
      const summary = page.locator("details").filter({ has: page.getByText(title, { exact: true }) }).locator("summary");
      await summary.focus();
      await page.keyboard.press("Enter");
      await expect(summary.locator("..").getByText("Reviewed", { exact: true })).toBeVisible();
    }
    const reject = page.getByRole("radio", { name: /^Reject all/ });
    await reject.focus();
    await page.keyboard.press("Space");
    for (const row of await page.locator(".ack-row").all()) {
      const context = row.getByRole("button", { name: "Context" });
      await context.focus();
      await page.keyboard.press("Enter");
    }
    const propose = page.getByRole("button", { name: "Record provisional ruling" });
    await propose.focus();
    await page.keyboard.press("Enter");
    const reveal = page.getByRole("button", { name: "Reveal candidate provenance" });
    await expect(reveal).toBeVisible();
    await reveal.focus();
    await page.keyboard.press("Enter");
    const provenance = page.locator("details").filter({ has: page.getByText("Provenance reveal", { exact: true }) }).locator("summary");
    await provenance.focus();
    await page.keyboard.press("Enter");
    await expect(provenance.locator("..").getByText("Reviewed", { exact: true })).toBeVisible();
    const affirm = page.getByRole("button", { name: "Affirm or revise ruling" });
    await affirm.focus();
    await page.keyboard.press("Enter");
    const finalize = page.getByRole("button", { name: "Finalize local ruling" });
    await expect(finalize).toBeVisible();
    await finalize.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByText("Ruling finalized")).toBeVisible();
  } finally {
    await studio.stop();
  }
});

test("mobile decision inspection retains every trust and evidence boundary", async ({ browser }) => {
  const studio = await startStudio();
  const context = await browser.newContext({ viewport: { width: 375, height: 812 }, colorScheme: "dark" });
  const page = await context.newPage();
  try {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.goto(studio.url);
    await expect(page.getByRole("heading", { name: "Decision-ready evidence" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Decision jurisdiction" })).toBeVisible();
    await expect(page.getByText("3 trials / candidate", { exact: true })).toBeVisible();
    await expect(page.getByText("2 trials / candidate", { exact: true })).toBeVisible();
    await expect(page.getByText("Provenance hidden", { exact: true })).toBeVisible();
    await expect(page.getByText(/temporary directory is for repeatability, not security isolation/)).toBeVisible();

    for (const title of ["Original witness", "Minimized witness", "Reduction derivation", "Captured → Projection → Operations", "Nonasserted fields"]) {
      const summary = page.getByText(title, { exact: true });
      await summary.click();
      await expect(summary.locator("..").getByText("Reviewed", { exact: true })).toBeVisible();
    }

    await expect(page.getByLabel("Exact tuple-set preview")).toContainText("Not asserted");
    await expect(page.getByRole("button", { name: "Record provisional ruling" })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    const axe = await new AxeBuilder({ page }).analyze();
    expect(axe.violations).toEqual([]);
  } finally {
    await context.close();
    await studio.stop();
  }
});
