import type { BenchResponse, DraftInput } from "../../api/types";

const dangerousDirectionalControls = new Set([0x061c, 0x200e, 0x200f, 0x202a, 0x202b, 0x202c, 0x202d, 0x202e, 0x2066, 0x2067, 0x2068, 0x2069]);

export function initialDraft(bench: BenchResponse): DraftInput {
  return {
    action: "ALLOW_OBSERVED",
    allowed_aliases: [],
    field_acknowledgements: (bench.blind?.differing_fields ?? []).map((field) => ({ field_id: field, disposition: "" })),
    custom_values: [],
    custom_reviewer: "",
  };
}

// This is a display-only projection. The API retains exact package bytes;
// browser evidence makes terminal and directional controls visible so they
// cannot impersonate Studio chrome or reorder adjacent labels.
export function evidenceDisplayText(value: string): string {
  let result = "";
  for (const character of value) {
    const codepoint = character.codePointAt(0)!;
    if ((codepoint < 0x20 && codepoint !== 0x09) || codepoint === 0x7f || dangerousDirectionalControls.has(codepoint) || codepoint === 0xfffd) {
      result += `<U+${codepoint.toString(16).toUpperCase().padStart(4, "0")}>`;
    } else {
      result += character;
    }
  }
  return result;
}
