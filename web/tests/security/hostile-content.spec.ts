import { expect, test } from "@playwright/test";
import type { BenchResponse } from "../../src/api/types";
import { evidenceDisplayText } from "../../src/features/decision/model";
import { startStudio } from "../helpers/studio";

const hostile = [
  `<script>globalThis.pwned = true</script>`,
  `</script><img src=x onerror="globalThis.pwned = true">`,
  `\u001b]8;;https://attacker.invalid\u0007FORGED LINK\u001b]8;;\u0007`,
  `\u202e../private/token\ufffd`,
  `# PASS verified software`,
  `secret=countershape-fixture-secret`,
  `../../../../Users/example/.ssh/id_ed25519`,
  `A`.repeat(16_384),
].join("\n");

test("hostile candidate output remains inert evidence instead of browser chrome", async ({ page, request }) => {
  const studio = await startStudio();
  const consoleMessages: string[] = [];
  const pageErrors: string[] = [];
  page.on("console", (message) => consoleMessages.push(message.text()));
  page.on("pageerror", (error) => pageErrors.push(error.message));
  try {
    const response = await request.get(studio.origin + "/api/v1/bench/seed-cli-precedence", {
      headers: { Authorization: "Bearer " + studio.token },
    });
    expect(response.status()).toBe(200);
    const bench = await response.json() as BenchResponse;
    const field = bench.blind?.cards[0]?.fields[0];
    if (!field) throw new Error("real blind payload omitted its first typed field");
    field.tag = "STRING";
    field.text = hostile;
    field.canonical_json_base64 = "";

    await page.route("**/api/v1/bench/seed-cli-precedence", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(bench),
        headers: { "Cache-Control": "no-store" },
      });
    });
    await page.goto(studio.url);

    const rendered = page.locator(".candidate-card").first().locator(".fact-row dd").first();
    const encoded = evidenceDisplayText(hostile);
    await expect(rendered).toHaveText(encoded);
    expect(await rendered.textContent()).toBe(encoded);
    expect(encoded).toContain("<U+001B>");
    expect(encoded).toContain("<U+202E>");
    expect(encoded).toContain("<U+FFFD>");
    expect(await rendered.textContent()).not.toContain("\u001b");
    expect(await rendered.textContent()).not.toContain("\u202e");
    expect(await page.evaluate(() => (globalThis as typeof globalThis & { pwned?: boolean }).pwned)).toBeUndefined();
    await expect(page.locator(`img[src="x"]`)).toHaveCount(0);
    await expect(page.locator("script:not([src])")).toHaveCount(0);
    await expect(page.locator(`a[href="https://attacker.invalid"]`)).toHaveCount(0);
    await expect(page.getByRole("heading", { name: /PASS verified software/i })).toHaveCount(0);
    await expect(page.getByRole("button", { name: /PASS verified software/i })).toHaveCount(0);
    expect(await rendered.evaluate((element) => Array.from(element.attributes).every((attribute) => !attribute.value.includes("PASS verified software")))).toBe(true);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    expect(pageErrors).toEqual([]);
    expect(consoleMessages.join("\n")).not.toContain(hostile);
    expect(consoleMessages.join("\n")).not.toContain(studio.token);
  } finally {
    await studio.stop();
  }
});
