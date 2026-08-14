# Blind screenshot critic brief

Inspect only the seven PNG files in the current directory. Their filenames are neutral state and viewport labels. Do not inspect source code, repository files, implementation notes, prior findings, or any other local content. Treat the images as screenshots of one bounded local decision studio. Do not infer runtime behavior that the pixels do not show.

Evaluate these dimensions:

1. State and decision-jurisdiction clarity.
2. Equal outcome weight and absence of majority, popularity, recommendation, or winner cues.
3. Prominence and legibility of the trust boundary.
4. Understanding of reduced versus original evidence and Captured → Projection → Operations.
5. Asserted versus nonasserted-field clarity.
6. Safety and clarity of the mobile final-action region.
7. Visible keyboard/focus cues where a static screenshot permits judgment.
8. Language that overstates certainty, correctness, determinism, anonymity, safety, or authority.

Return Markdown with:

- the provider/model identity you can truthfully report;
- a one-sentence overall assessment;
- severity-ranked findings using only `BLOCKER`, `HIGH`, `MEDIUM`, `LOW`, or `NONE`;
- for each finding: screenshot filename, visible evidence, why it matters, and a concrete recommendation;
- an explicit `SHIP`, `SHIP WITH FIXES`, or `BLOCK` verdict.

Do not offer praise merely because the rubric asks for a verdict. If a dimension cannot be judged from these screenshots, say so instead of guessing.
