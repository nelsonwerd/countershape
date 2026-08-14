import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { startStudio, type RunningStudio } from "../helpers/studio";

const blindSurfaces = ["Original witness", "Minimized witness", "Reduction derivation", "Captured → Projection → Operations", "Nonasserted fields"];

test.describe("blind-first local ruling", () => {
  let studio: RunningStudio;

  test.beforeAll(async () => {
    studio = await startStudio("decision-ready");
  });

  test.afterAll(async () => {
    await studio.stop();
  });

  test("reviews every blind surface before reveal and finalizes through package transitions", async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.goto(studio.url);
    await expect(page).toHaveURL(studio.origin + "/");
    await expect(page.getByRole("heading", { name: "Decision-ready evidence" })).toBeVisible();
    await expect(page.getByText("Provenance hidden")).toBeVisible();
    await expect(page.locator(".identity-reveal")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Selected", exact: true })).toHaveCount(0);
    await expect(page.locator(".ack-row button[aria-pressed=true]")).toHaveCount(0);

    const initialAxe = await new AxeBuilder({ page }).analyze();
    expect(initialAxe.violations).toEqual([]);

    await reviewBlind(page);
    await selectFirstOutcome(page);
    await assertEveryField(page);
    await expect(page.getByRole("button", { name: "Record provisional ruling" })).toBeEnabled();
    await page.getByRole("button", { name: "Record provisional ruling" }).click();
    await expect(page.getByRole("button", { name: "Reveal candidate provenance" })).toBeVisible();

    await page.getByRole("button", { name: "Reveal candidate provenance" }).click();
    await expect(page.getByRole("heading", { name: "Identity reveal & affirmation" })).toBeVisible();
    await expect(page.locator(".identity-reveal").first()).toBeVisible();
    const provenance = page.locator("details").filter({ has: page.getByText("Provenance reveal", { exact: true }) });
    await provenance.locator("summary").click();
    await expect(provenance.getByText("Reviewed", { exact: true })).toBeVisible();

    await page.getByRole("button", { name: "Affirm or revise ruling" }).click();
    await expect(page.getByRole("button", { name: "Finalize local ruling" })).toBeVisible();
    await page.getByLabel("Annotation optional, local attribution only").fill("Reviewed in the bounded local studio.");
    await page.getByRole("button", { name: "Finalize local ruling" }).click();

    await expect(page.getByRole("heading", { name: "Local ruling recorded" })).toBeVisible();
    await expect(page.getByText("Ruling finalized")).toBeVisible();
    await expect(page.getByText("Files emitted").locator("..")) .toContainText("0");
    const finalAxe = await new AxeBuilder({ page }).analyze();
    expect(finalAxe.violations).toEqual([]);
  });
});

test("allow-many preview preserves complete tuples without a cross-product", async ({ page }) => {
  const studio = await startStudio();
  try {
    await page.goto(studio.url);
    await reviewBlind(page);
    const outcomes = await page.getByRole("button", { name: "Allow", exact: true }).all();
    await outcomes[0]?.click();
    await outcomes[1]?.click();
    await assertEveryField(page);
    await expect(page.getByLabel("Exact tuple-set preview").locator(".preview-tuple")).toHaveCount(2);
    await page.getByRole("button", { name: "Record provisional ruling" }).click();
    await expect(page.getByRole("button", { name: "Reveal candidate provenance" })).toBeVisible();
  } finally {
    await studio.stop();
  }
});

test("custom none-conforms ruling and noncompilable dispositions remain distinct", async ({ page }) => {
  test.setTimeout(120_000);
  const customStudio = await startStudio();
  try {
    await page.goto(customStudio.url);
    await reviewBlind(page);
    await page.getByRole("radio", { name: /^Custom expectation/ }).check();
    await assertEveryField(page);
    await page.getByLabel("Reviewer label").fill("local-reviewer");
    await page.getByLabel("Cli Stdout Json Source", { exact: true }).fill("custom-none-conforms");
    await expect(page.getByRole("button", { name: "Record provisional ruling" })).toBeEnabled();
    await page.getByRole("button", { name: "Record provisional ruling" }).click();
    await expect(page.getByRole("button", { name: "Reveal candidate provenance" })).toBeVisible();
  } finally {
    await customStudio.stop();
  }

  for (const action of ["Reject all", "Defer"]) {
    const studio = await startStudio();
    try {
      await page.goto(studio.url);
      await reviewBlind(page);
      await page.getByRole("radio", { name: new RegExp(`^${action}`) }).check();
      await contextEveryField(page);
      await expect(page.getByLabel("Exact tuple-set preview")).toContainText("explicitly noncompilable");
      await page.getByRole("button", { name: "Record provisional ruling" }).click();
      await page.getByRole("button", { name: "Reveal candidate provenance" }).click();
      await reviewProvenance(page);
      await page.getByRole("button", { name: "Affirm or revise ruling" }).click();
      await page.getByRole("button", { name: "Finalize local ruling" }).click();
      await expect(page.getByText("Files emitted").locator("..")).toContainText("0");
      await expect(page.getByText("Noncompilable Action")).toBeVisible();
    } finally {
      await studio.stop();
    }
  }
});

test("early reveal is recorded and an empty field scope cannot leave the browser", async ({ page }) => {
  test.setTimeout(90_000);
  const earlyStudio = await startStudio();
  try {
    await page.goto(earlyStudio.url);
    await reviewBlind(page);
    await selectFirstOutcome(page);
    await assertEveryField(page);
    await page.getByRole("button", { name: "Reveal before proposing" }).click();
    await expect(page.getByRole("heading", { name: "Identity reveal & affirmation" })).toBeVisible();
    await reviewProvenance(page);
    await page.getByRole("button", { name: "Affirm or revise ruling" }).click();
    await page.getByRole("button", { name: "Finalize local ruling" }).click();
    await expect(page.getByText("Ruling finalized")).toBeVisible();
    await expect(page.getByText("Early reveal").locator("..")).toContainText("Yes");
  } finally {
    await earlyStudio.stop();
  }

  const weakStudio = await startStudio();
  try {
    await page.goto(weakStudio.url);
    await reviewBlind(page);
    await expect(page.getByRole("button", { name: "Record provisional ruling" })).toBeDisabled();
    await expect(page.getByLabel("Exact tuple-set preview")).toContainText("explicitly assert a field");
    await expect(page.getByRole("heading", { name: "Decision-ready evidence" })).toBeVisible();
  } finally {
    await weakStudio.stop();
  }
});

test("a post-reveal ruling change requires and retains an explicit rationale", async ({ page }) => {
  const studio = await startStudio();
  try {
    await page.goto(studio.url);
    await reviewBlind(page);
    await selectFirstOutcome(page);
    await assertEveryField(page);
    await page.getByRole("button", { name: "Record provisional ruling" }).click();
    await page.getByRole("button", { name: "Reveal candidate provenance" }).click();
    await reviewProvenance(page);
    await page.getByRole("radio", { name: /^Defer/ }).check();
    await contextEveryField(page);
    await page.getByLabel(/Post-reveal change rationale/).fill("Provenance changed the appropriate local disposition.");
    await page.getByRole("button", { name: "Affirm or revise ruling" }).click();
    await page.getByRole("button", { name: "Finalize local ruling" }).click();
    await expect(page.getByText("Post-reveal change").locator("..")).toContainText("Yes — rationale retained");
  } finally {
    await studio.stop();
  }
});

test("mobile error fixture is explicit, readable, and noninteractive", async ({ browser }) => {
  const studio = await startStudio("error");
  const context = await browser.newContext({ viewport: { width: 375, height: 812 }, colorScheme: "dark" });
  const page = await context.newPage();
  try {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.goto(studio.url);
    await expect(page.getByRole("heading", { name: "Uncomparable" })).toBeVisible();
    await expect(page.getByText(/control failure prevented/).first()).toBeVisible();
    await expect(page.getByRole("button", { name: /ruling/i })).toHaveCount(0);
    const axe = await new AxeBuilder({ page }).analyze();
    expect(axe.violations).toEqual([]);
  } finally {
    await context.close();
    await studio.stop();
  }
});

async function reviewBlind(page: import("@playwright/test").Page) {
  for (const title of blindSurfaces) {
    const surface = page.locator("details").filter({ has: page.getByText(title, { exact: true }) });
    await surface.locator("summary").click();
    await expect(surface.getByText("Reviewed", { exact: true })).toBeVisible();
  }
}

async function reviewProvenance(page: import("@playwright/test").Page) {
  const provenance = page.locator("details").filter({ has: page.getByText("Provenance reveal", { exact: true }) });
  await provenance.locator("summary").click();
  await expect(provenance.getByText("Reviewed", { exact: true })).toBeVisible();
}

async function assertEveryField(page: import("@playwright/test").Page) {
  for (const row of await page.locator(".ack-row").all()) {
    await row.getByRole("button", { name: "Assert" }).click();
  }
}

async function contextEveryField(page: import("@playwright/test").Page) {
  for (const row of await page.locator(".ack-row").all()) {
    await row.getByRole("button", { name: "Context" }).click();
  }
}

async function selectFirstOutcome(page: import("@playwright/test").Page) {
  await page.getByRole("button", { name: "Allow", exact: true }).first().click();
}
