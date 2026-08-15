import { expect, test } from "@playwright/test";
import { startStudio, type RunningStudio } from "../helpers/studio";

test.describe("loopback browser boundary", () => {
  let studio: RunningStudio;

  test.beforeAll(async () => {
    studio = await startStudio("decision-ready");
  });

  test.afterAll(async () => {
    await studio.stop();
  });

  test("strips fragment authority and never sends it in a request target", async ({ page, browser }) => {
    const traffic: Array<{ url: string; authorization: string | undefined }> = [];
    page.on("request", (request) => traffic.push({ url: request.url(), authorization: request.headers()["authorization"] }));
    const response = await page.goto(studio.url);
    expect(response?.headers()["content-security-policy"]).toContain("default-src 'none'");
    expect(response?.headers()["access-control-allow-origin"]).toBeUndefined();
    await expect(page).toHaveURL(studio.origin + "/");
    await expect(page.locator("body")).not.toContainText(studio.token);
    expect(traffic.every((request) => !request.url.includes(studio.token))).toBe(true);
    expect(traffic.filter((request) => request.url.includes("/api/")).every((request) => request.authorization === "Bearer " + studio.token)).toBe(true);
    expect(traffic.filter((request) => !request.url.includes("/api/")).every((request) => request.authorization === undefined)).toBe(true);

    const fresh = await browser.newContext();
    const unauthenticated = await fresh.newPage();
    try {
      await unauthenticated.goto(studio.origin);
      await expect(unauthenticated.getByText(/no studio launch credential/i)).toBeVisible();
    } finally {
      await fresh.close();
    }
  });

  test("rejects missing authority, foreign Origin, and stale CAS", async ({ request }) => {
    let response = await request.get(studio.origin + "/api/v1/session");
    expect(response.status()).toBe(401);

    response = await request.get(studio.origin + "/api/v1/session", { headers: { Authorization: "Bearer " + studio.token } });
    expect(response.status()).toBe(200);
    const session = await response.json() as { csrf: string; revision_digest: string };
    const body = { expected_revision: session.revision_digest, surface: "ORIGINAL_WITNESS" };

    response = await request.post(studio.origin + "/api/v1/bench/seed-cli-precedence/visit", {
      headers: {
        Authorization: "Bearer " + studio.token,
        "Content-Type": "application/json",
        "X-CSRF-Token": session.csrf,
        Origin: "http://127.0.0.1:1",
      },
      data: body,
    });
    expect(response.status()).toBe(403);

    const exactHeaders = {
      Authorization: "Bearer " + studio.token,
      "Content-Type": "application/json",
      "X-CSRF-Token": session.csrf,
      Origin: studio.origin,
    };
    response = await request.post(studio.origin + "/api/v1/bench/seed-cli-precedence/visit", { headers: exactHeaders, data: body });
    expect(response.status()).toBe(200);
    response = await request.post(studio.origin + "/api/v1/bench/seed-cli-precedence/visit", { headers: exactHeaders, data: body });
    expect(response.status()).toBe(409);
    await expect(response.json()).resolves.toMatchObject({ code: "STUDIO_STALE_REVISION" });
  });

  test("a stale browser tab reloads package state instead of overwriting it", async ({ browser }) => {
    const context = await browser.newContext();
    const first = await context.newPage();
    const second = await context.newPage();
    try {
      await Promise.all([first.goto(studio.url), second.goto(studio.url)]);
      const firstSurface = first.locator("details[data-surface-id='MINIMIZED_WITNESS']");
      await firstSurface.locator("summary").click();
      await expect(firstSurface.getByText("Reviewed", { exact: true })).toBeVisible();

      const staleSurface = second.locator("details[data-surface-id='PROJECTION_OPERATIONS']");
      await staleSurface.locator("summary").click();
      await expect(second.getByRole("alert")).toContainText("Stale Revision");
      await expect(second.getByRole("heading", { name: "Decision-ready evidence" })).toBeVisible();
    } finally {
      await context.close();
    }
  });
});
