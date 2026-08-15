import { describe, expect, it } from "vitest";
import { evidenceDisplayText } from "./model";

describe("evidenceDisplayText", () => {
  it("retains ordinary text and makes terminal and directional controls visible", () => {
    const source = `<script>\u001b]8;;https://example.invalid\u0007PASS\u202e../token\ufffd\nnext line`;
    const displayed = evidenceDisplayText(source);
    expect(displayed).toBe(`<script><U+001B>]8;;https://example.invalid<U+0007>PASS<U+202E>../token<U+FFFD><U+000A>next line`);
    expect(displayed).not.toContain("\u001b");
    expect(displayed).not.toContain("\u202e");
  });
});
