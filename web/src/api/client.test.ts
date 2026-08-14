import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { StudioClient } from "./client";

const token = "A".repeat(43);

describe("StudioClient launch authority", () => {
  beforeEach(() => {
    window.sessionStorage.clear();
    window.history.replaceState(null, "", "/#access_token=" + token);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("moves the fragment credential into tab storage and strips the URL", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      schema_version: "countershape/studio/v1",
      study_id: "seed-cli-precedence",
      csrf: "csrf",
      revision_digest: "revision:test",
      presentation_state: "decision-ready",
    }), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("fetch", fetch);

    const client = StudioClient.fromLaunchLocation();
    expect(window.location.hash).toBe("");
    expect(window.location.href).not.toContain(token);
    expect(window.sessionStorage.length).toBe(1);
    await client.session();
    const init = fetch.mock.calls[0]?.[1] as RequestInit;
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer " + token);
    expect(fetch.mock.calls[0]?.[0]).toBe("/api/v1/session");
  });

  it("refuses a malformed fragment and removes it", () => {
    window.history.replaceState(null, "", "/#access_token=not-valid");
    expect(() => StudioClient.fromLaunchLocation()).toThrow(/malformed/);
    expect(window.location.hash).toBe("");
    expect(window.sessionStorage.length).toBe(0);
  });
});
