import { useMemo, useState } from "react";
import type {
  BenchResponse,
  BlindField,
  CustomValueInput,
  DraftInput,
  FieldAcknowledgement,
  MutationEnvelope,
} from "../../api/types";
import { evidenceDisplayText } from "./model";

const preRevealSurfaces = [
  "ORIGINAL_WITNESS",
  "MINIMIZED_WITNESS",
  "REDUCTION_DERIVATION",
  "PROJECTION_OPERATIONS",
  "NONASSERTED_FIELDS",
] as const;

type Mutation = (
  operation: "visit" | "propose" | "reveal" | "revise" | "finalize",
  input: Omit<MutationEnvelope, "expected_revision">,
) => Promise<void>;

interface Props {
  bench: BenchResponse;
  draft: DraftInput;
  setDraft: (draft: DraftInput) => void;
  pending: boolean;
  notice: { code: string; message: string } | null;
  mutate: Mutation;
}

export function DecisionBench({ bench, draft, setDraft, pending, notice, mutate }: Props) {
  const [rationale, setRationale] = useState("");
  const [actor, setActor] = useState("local-operator");
  const [annotation, setAnnotation] = useState("");
  const reviewed = useMemo(() => new Set(bench.reviewed_surfaces), [bench.reviewed_surfaces]);
  const allPreRevealReviewed = preRevealSurfaces.every((surface) => reviewed.has(surface));

  return (
    <>
      <section className="page-heading">
        <div>
          <p className="eyebrow">CLI precedence · exact-witness comparison</p>
          <h1>{titleFor(bench)}</h1>
          <p className="lede">{bench.summary}</p>
        </div>
        <div className="state-stack" aria-label="Package-issued states">
          <StatePill label="Study" value={bench.study_state} />
          <StatePill label="Candidate" value={bench.candidate_state} />
          <StatePill label="Decision" value={bench.choicepoint_state} />
        </div>
      </section>

      <div className="trust-banner">
        <span aria-hidden="true">⚠</span>
        <div><strong>Trusted code boundary</strong><p>{bench.trust_warning}</p></div>
      </div>
      {notice && <div className="mutation-refusal" role="alert"><strong>{humanToken(notice.code)}</strong><span>{notice.message}</span></div>}

      {bench.blind ? (
        <div className="decision-layout">
          <div className="decision-primary">
            <ScenarioSummary bench={bench} />
            <BlindCandidates bench={bench} draft={draft} setDraft={setDraft} revealed={bench.reveal !== undefined} />
            <EvidenceReview bench={bench} reviewed={reviewed} pending={pending} mutate={mutate} />
          </div>
          <aside className="decision-panel" aria-label="Ruling editor">
            {bench.result ? (
              <DecisionReceipt bench={bench} />
            ) : (
              <RulingEditor
                bench={bench}
                draft={draft}
                setDraft={setDraft}
                pending={pending}
                allPreRevealReviewed={allPreRevealReviewed}
                provenanceReviewed={reviewed.has("PROVENANCE_REVEAL")}
                rationale={rationale}
                setRationale={setRationale}
                actor={actor}
                setActor={setActor}
                annotation={annotation}
                setAnnotation={setAnnotation}
                mutate={mutate}
              />
            )}
          </aside>
        </div>
      ) : (
        <StaticState bench={bench} />
      )}

      <section id="scope" className="scope-panel">
        <div><p className="eyebrow">Authority ceiling</p><h2>What this screen does not prove</h2></div>
        <ul>{bench.nonclaims.map((nonclaim) => <li key={nonclaim}>{nonclaim}</li>)}</ul>
      </section>
    </>
  );
}

function ScenarioSummary({ bench }: { bench: BenchResponse }) {
  const blind = bench.blind!;
  return (
    <section className="section-card scenario-section" aria-labelledby="scenario-heading">
      <div className="section-heading">
        <div><p className="eyebrow">Human-authored question</p><h2 id="scenario-heading">Decision jurisdiction</h2></div>
        <span className="privacy-badge">Exact witness</span>
      </div>
      <blockquote>{evidenceDisplayText(blind.scenario)}</blockquote>
      <dl className="jurisdiction-facts">
        <div><dt>Scope</dt><dd>{evidenceDisplayText(blind.scope)}</dd></div>
        <div><dt>Discovery</dt><dd>{blind.discovery_repeats_per_candidate} trials / candidate</dd></div>
        <div><dt>Fresh confirmation</dt><dd>{blind.confirmation_repeats_per_candidate} trials / candidate</dd></div>
        <div><dt>Projection</dt><dd>{humanToken(blind.projection_mode)}</dd></div>
      </dl>
    </section>
  );
}

function StatePill({ label, value }: { label: string; value: string }) {
  return <span className="state-pill"><small>{label}</small><strong>{humanToken(value)}</strong></span>;
}

function BlindCandidates({ bench, draft, setDraft, revealed }: Pick<Props, "bench" | "draft" | "setDraft"> & { revealed: boolean }) {
  const revealByAlias = new Map((bench.reveal?.groups ?? []).map((group) => [group.alias, group.candidates]));
  const selected = new Set(draft.allowed_aliases);
  const maySelect = draft.action === "ALLOW_OBSERVED";
  return (
    <section className="section-card candidates-section" aria-labelledby="candidate-heading">
      <div className="section-heading">
        <div><p className="eyebrow">Blind comparison</p><h2 id="candidate-heading">Observed outcome cards</h2></div>
        <span className="privacy-badge">{revealed ? "Identity revealed" : "Provenance hidden"}</span>
      </div>
      <p className="section-intro">Cards are ordered by package-derived projection identity—not candidate order, popularity, or support count.</p>
      <div className="candidate-grid">
        {(bench.blind?.cards ?? []).map((card, index) => {
          const isSelected = selected.has(card.alias);
          return (
            <article className={`candidate-card ${isSelected ? "selected" : ""}`} key={card.alias}>
              <div className="candidate-topline">
                <span className="candidate-letter" aria-hidden="true">{String.fromCharCode(65 + index)}</span>
                <div><p>Outcome card</p><code>{shortAlias(card.alias)}</code></div>
                {maySelect && (
                  <button
                    type="button"
                    className="select-control"
                    aria-pressed={isSelected}
                    onClick={() => setDraft({ ...draft, allowed_aliases: toggle(draft.allowed_aliases, card.alias) })}
                  >{isSelected ? "Selected" : "Allow"}</button>
                )}
              </div>
              <dl>{card.fields.map((field) => <FieldFact field={field} key={field.field_id} />)}</dl>
              {revealed && (
                <div className="identity-reveal">
                  <p className="mini-label">Revealed provenance</p>
                  {(revealByAlias.get(card.alias) ?? []).map((candidate) => (
                    <div className="identity-row" key={candidate.candidate_execution_key}>
                      <strong>{evidenceDisplayText(candidate.display_ref)}</strong>
                      <span>{evidenceDisplayText(candidate.producer_metadata)}</span>
                      <code>{compactDigest(candidate.tree_identity_digest)}</code>
                    </div>
                  ))}
                </div>
              )}
            </article>
          );
        })}
      </div>
    </section>
  );
}

function FieldFact({ field }: { field: BlindField }) {
  return (
    <div className="fact-row">
      <dt>{humanToken(field.field_id)}</dt>
      <dd>{evidenceDisplayText(displayField(field))}</dd>
    </div>
  );
}

function EvidenceReview({ bench, reviewed, pending, mutate }: { bench: BenchResponse; reviewed: Set<string>; pending: boolean; mutate: Mutation }) {
  const blind = bench.blind!;
  const sections = [
    { id: "ORIGINAL_WITNESS", title: "Original witness", detail: evidenceDisplayText(decodeBase64(blind.original_stimulus_base64)) },
    { id: "MINIMIZED_WITNESS", title: "Minimized witness", detail: evidenceDisplayText(decodeBase64(blind.minimized_stimulus_base64)) },
    { id: "REDUCTION_DERIVATION", title: "Reduction derivation", detail: `${blind.reduction.grade_status} · ${blind.reduction.evaluation_count} evaluations · ${blind.reduction.unresolved_count} unresolved · ${blind.reduction.proposal_limit} proposals · ${blind.reduction.candidate_trial_limit} candidate trials · ${blind.reduction.wall_limit_ms}ms wall · ${blind.reduction.final_sweep_state}` },
    { id: "PROJECTION_OPERATIONS", title: "Captured → Projection → Operations", detail: `Captured process result → ${blind.projection_mode} → ${blind.projection_operations.map((operation) => operation.name).join(" · ")}` },
    { id: "NONASSERTED_FIELDS", title: "Nonasserted fields", detail: "Every differing field must be explicitly marked Assert or Context before a ruling can be recorded." },
  ];
  if (bench.reveal) {
    sections.push({ id: "PROVENANCE_REVEAL", title: "Provenance reveal", detail: "Candidate refs and producer labels are now visible. Revisit the ruling without changing it silently." });
  }
  return (
    <section className="section-card evidence-section" aria-labelledby="evidence-heading">
      <div className="section-heading"><div><p className="eyebrow">Required review</p><h2 id="evidence-heading">Evidence surfaces</h2></div><span className="review-count">{sections.filter((section) => reviewed.has(section.id)).length}/{sections.length}</span></div>
      <div className="evidence-list">
        {sections.map((section) => (
          <details key={section.id} data-surface-id={section.id} onToggle={(event) => {
            if ((event.currentTarget as HTMLDetailsElement).open && !reviewed.has(section.id) && !pending) {
              void mutate("visit", { surface: section.id });
            }
          }}>
            <summary><span>{section.title}</span><span className={reviewed.has(section.id) ? "reviewed" : "unreviewed"}>{reviewed.has(section.id) ? "Reviewed" : "Open to review"}</span></summary>
            <pre tabIndex={0} aria-label={`${section.title} evidence content`}>{section.detail}</pre>
          </details>
        ))}
      </div>
    </section>
  );
}

function RulingEditor(props: {
  bench: BenchResponse;
  draft: DraftInput;
  setDraft: (draft: DraftInput) => void;
  pending: boolean;
  allPreRevealReviewed: boolean;
  provenanceReviewed: boolean;
  rationale: string;
  setRationale: (value: string) => void;
  actor: string;
  setActor: (value: string) => void;
  annotation: string;
  setAnnotation: (value: string) => void;
  mutate: Mutation;
}) {
  const { bench, draft, setDraft, pending, allPreRevealReviewed, provenanceReviewed, rationale, setRationale, actor, setActor, annotation, setAnnotation, mutate } = props;
  const actionOptions: Array<[DraftInput["action"], string, string]> = [
    ["ALLOW_OBSERVED", "Allow observed", "Compile an exact predicate from selected observed cards."],
    ["CUSTOM_EXPECTATION", "Custom expectation", "Author a reviewed exact tuple; no observed-card shortcut."],
    ["REJECT_ALL", "Reject all", "Record no acceptable outcome; emits no predicate."],
    ["DEFER", "Defer", "Preserve an explicit noncompilable disposition."],
    ["REFINE", "Refine study", "Request new evidence rather than deciding from this set."],
  ];
  const selectedFields = draft.field_acknowledgements.filter((field) => field.disposition === "ASSERT").map((field) => field.field_id);
  const contextFields = draft.field_acknowledgements.filter((field) => field.disposition === "CONTEXT").map((field) => field.field_id);
  const pendingFields = draft.field_acknowledgements.filter((field) => field.disposition === "").map((field) => field.field_id);
  const customFields = new Set(draft.custom_values.map((value) => value.field_id));
  const everyFieldAcknowledged = pendingFields.length === 0 && draft.field_acknowledgements.length === (bench.blind?.differing_fields.length ?? 0);
  const draftShapeReady = everyFieldAcknowledged && (isNoncompilable(draft.action) ? selectedFields.length === 0 : (
    selectedFields.length > 0 && (
      (draft.action === "ALLOW_OBSERVED" && draft.allowed_aliases.length > 0) ||
      (draft.action === "CUSTOM_EXPECTATION" && draft.custom_reviewer.trim() !== "" && selectedFields.every((field) => customFields.has(field)))
    )
  ));
  const canPropose = allPreRevealReviewed && draftShapeReady && bench.session_state === "BLIND_OPEN";
  return (
    <div className="sticky-decision">
      <p className="eyebrow">Local ruling</p>
      <h2>Record your interpretation</h2>
      <p className="panel-intro">The package enforces the transition. This interface does not decide which outcome is “correct.”</p>

      <fieldset className="action-options" disabled={pending || bench.session_state === "FINALIZED"}>
        <legend>Disposition</legend>
        {actionOptions.map(([value, label, description]) => (
          <label key={value} className={draft.action === value ? "active" : ""}>
            <input type="radio" name="action" value={value} checked={draft.action === value} onChange={() => setDraft(changeAction(bench, draft, value))} />
            <span><strong>{label}</strong><small>{description}</small></span>
          </label>
        ))}
      </fieldset>

      <div className="field-acknowledgements">
        <div className="subheading"><strong>Differing fields</strong><small>{selectedFields.length} asserted · {contextFields.length} context · {pendingFields.length} pending</small></div>
        {draft.field_acknowledgements.map((acknowledgement) => (
          <div className="ack-row" key={acknowledgement.field_id}>
            <span>{humanToken(acknowledgement.field_id)}</span>
            <div role="group" aria-label={`${acknowledgement.field_id} disposition`}>
              {(["ASSERT", "CONTEXT"] as const).map((value) => (
                <button
                  type="button"
                  key={value}
                  aria-pressed={acknowledgement.disposition === value}
                  disabled={pending || (isNoncompilable(draft.action) && value === "ASSERT")}
                  onClick={() => setDraft(setAcknowledgement(bench, draft, acknowledgement.field_id, value))}
                >{humanToken(value)}</button>
              ))}
            </div>
          </div>
        ))}
      </div>

      {draft.action === "CUSTOM_EXPECTATION" && <CustomEditor bench={bench} draft={draft} setDraft={setDraft} />}

      <PredicatePreview bench={bench} draft={draft} />

      <div className="decision-actions">
        {bench.session_state === "BLIND_OPEN" && (
          <>
            <button className="button primary" disabled={pending || !canPropose} onClick={() => void mutate("propose", { draft })}>{pending ? "Checking…" : "Record provisional ruling"}</button>
            <button className="button secondary" disabled={pending || !allPreRevealReviewed} onClick={() => void mutate("reveal", {})}>Reveal before proposing</button>
            {!allPreRevealReviewed ? <p className="action-hint">Review all five blind evidence surfaces first.</p> : !draftShapeReady && <p className="action-hint">Acknowledge every field, then complete one exact predicate or a noncompilable disposition.</p>}
          </>
        )}
        {bench.session_state === "PROVISIONAL_RECORDED" && <button className="button reveal-button" disabled={pending} onClick={() => void mutate("reveal", {})}>Reveal candidate provenance</button>}
        {bench.session_state === "REVEALED" && (
          <>
            <label className="text-field">Post-reveal change rationale <span>required only if the full ruling changed</span><textarea value={rationale} onChange={(event) => setRationale(event.target.value)} maxLength={32768} /></label>
            <button className="button primary" disabled={pending || !provenanceReviewed} onClick={() => void mutate("revise", { draft, rationale })}>Affirm or revise ruling</button>
            {!provenanceReviewed && <p className="action-hint">Open the provenance surface before affirming.</p>}
          </>
        )}
        {bench.session_state === "POST_REVEAL_RECORDED" && (
          <>
            <label className="text-field">Actor label<input value={actor} onChange={(event) => setActor(event.target.value)} maxLength={512} /></label>
            <label className="text-field">Annotation <span>optional, local attribution only</span><textarea value={annotation} onChange={(event) => setAnnotation(event.target.value)} maxLength={32768} /></label>
            <button className="button primary finalize" disabled={pending || actor.trim() === ""} onClick={() => void mutate("finalize", { actor, annotation })}>Finalize local ruling</button>
          </>
        )}
      </div>
      <p className="emission-note"><span aria-hidden="true">◇</span> U8 does not emit executable contract files.</p>
    </div>
  );
}

function PredicatePreview({ bench, draft }: Pick<Props, "bench" | "draft">) {
  const asserted = draft.field_acknowledgements.filter((value) => value.disposition === "ASSERT").map((value) => value.field_id);
  const context = draft.field_acknowledgements.filter((value) => value.disposition === "CONTEXT").map((value) => value.field_id);
  const pending = draft.field_acknowledgements.filter((value) => value.disposition === "").map((value) => value.field_id);
  const cards = (bench.blind?.cards ?? []).filter((card) => draft.allowed_aliases.includes(card.alias));
  return (
    <div className="predicate-preview" aria-label="Exact tuple-set preview">
      <div className="subheading"><strong>Exact tuple-set preview</strong><small>No cross-product</small></div>
      {isNoncompilable(draft.action) ? (
        <p>No executable predicate: this disposition is explicitly noncompilable.</p>
      ) : draft.action === "CUSTOM_EXPECTATION" ? (
        <div className="preview-tuple"><span>Custom exact tuple</span>{draft.custom_values.filter((value) => asserted.includes(value.field_id)).map((value) => <code key={value.field_id}>{humanToken(value.field_id)} = {value.text || humanToken(value.tag)}</code>)}</div>
      ) : cards.length === 0 || asserted.length === 0 ? (
        <p>Select at least one complete observed card and explicitly assert a field.</p>
      ) : cards.map((card, index) => (
        <div className="preview-tuple" key={card.alias}>
          <span>Tuple {index + 1}</span>
          {asserted.map((fieldID) => {
            const field = card.fields.find((candidate) => candidate.field_id === fieldID);
            return <code key={fieldID}>{humanToken(fieldID)} = {field ? displayField(field) : "[absent]"}</code>;
          })}
        </div>
      ))}
      <p className="not-asserted"><strong>Not asserted</strong> {context.length ? context.map(humanToken).join(" · ") : "None"}</p>
      <p className="not-asserted"><strong>Pending acknowledgement</strong> {pending.length ? pending.map(humanToken).join(" · ") : "None"}</p>
    </div>
  );
}

function CustomEditor({ bench, draft, setDraft }: Pick<Props, "bench" | "draft" | "setDraft">) {
  const asserted = new Set(draft.field_acknowledgements.filter((value) => value.disposition === "ASSERT").map((value) => value.field_id));
  return (
    <div className="custom-editor">
      <label className="text-field">Reviewer label<input value={draft.custom_reviewer} onChange={(event) => setDraft({ ...draft, custom_reviewer: event.target.value })} maxLength={512} /></label>
      {(bench.blind?.differing_fields ?? []).filter((field) => asserted.has(field)).map((field) => {
        const value = draft.custom_values.find((item) => item.field_id === field) ?? customValueForField(bench, field);
        return (
          <label className="text-field" key={field}>{humanToken(field)}
            {value.tag === "BOOLEAN" ? (
              <input type="checkbox" checked={value.boolean} onChange={(event) => setDraft({ ...draft, custom_values: replaceCustomValue(draft.custom_values, { ...value, boolean: event.target.checked }) })} />
            ) : value.tag === "ORDERED_STRING_LIST" ? (
              <textarea value={value.ordered_strings.join("\n")} onChange={(event) => setDraft({ ...draft, custom_values: replaceCustomValue(draft.custom_values, { ...value, ordered_strings: event.target.value.split("\n") }) })} />
            ) : value.tag === "BYTES" ? (
              <input value={value.bytes_base64} aria-label={`${humanToken(field)} canonical base64`} onChange={(event) => setDraft({ ...draft, custom_values: replaceCustomValue(draft.custom_values, { ...value, bytes_base64: event.target.value }) })} />
            ) : (
              <input value={value.text} onChange={(event) => setDraft({ ...draft, custom_values: replaceCustomValue(draft.custom_values, { ...value, text: event.target.value }) })} />
            )}
          </label>
        );
      })}
    </div>
  );
}

function DecisionReceipt({ bench }: { bench: BenchResponse }) {
  const result = bench.result!;
  return (
    <div className="sticky-decision receipt-panel">
      <span className="receipt-check" aria-hidden="true">✓</span>
      <p className="eyebrow">Ruling finalized</p>
      <h2>{humanToken(result.action)}</h2>
      <p className="panel-intro">One package-validated, caller-attributed local decision record now exists.</p>
      <dl className="receipt-facts">
        <div><dt>Compilation</dt><dd>{humanToken(result.compilation_status)}</dd></div>
        <div><dt>Asserted</dt><dd>{result.selected_fields.length ? result.selected_fields.map(humanToken).join(", ") : "None"}</dd></div>
        <div><dt>Context</dt><dd>{result.nonasserted_fields.length ? result.nonasserted_fields.map(humanToken).join(", ") : "None"}</dd></div>
        <div><dt>Files emitted</dt><dd>{result.emitted_files.length}</dd></div>
        <div><dt>Early reveal</dt><dd>{result.early_reveal ? "Yes" : "No"}</dd></div>
        <div><dt>Post-reveal change</dt><dd>{result.changed_after_reveal ? "Yes — rationale retained" : "No"}</dd></div>
      </dl>
      <details className="digest-detail"><summary>Decision identity</summary><code>{result.decision_digest}</code></details>
      <div className="authority-note"><strong>Authority</strong><p>{humanToken(result.authority)}</p><strong>Ceiling</strong><p>{humanToken(result.authority_ceiling)}</p></div>
    </div>
  );
}

function StaticState({ bench }: { bench: BenchResponse }) {
  const symbols: Record<string, string> = { empty: "○", preparing: "◌", active: "↻", partial: "◐", error: "!", completed: "✓", unstable: "≈", uncomparable: "≠", incomplete: "…", discovered: "◇", stale: "↺", invalidated: "×" };
  return (
    <section className="static-state section-card" data-state={bench.presentation_state}>
      <span className="state-symbol" aria-hidden="true">{symbols[bench.presentation_state] ?? "◇"}</span>
      <p className="eyebrow">Presentation fixture · {bench.presentation_state}</p>
      <h2>{humanToken(bench.candidate_state)}</h2>
      <p>{bench.summary}</p>
      <div className="next-action"><strong>Next honest action</strong><p>{bench.next_action}</p></div>
      {bench.future_authority_unimplemented && <p className="fixture-caveat">This state is a deterministic presentation fixture. U8 does not claim a durable workflow behind it.</p>}
    </section>
  );
}

function titleFor(bench: BenchResponse): string {
  if (bench.result) return "Local ruling recorded";
  if (bench.reveal) return "Identity reveal & affirmation";
  if (bench.blind) return "Decision-ready evidence";
  return humanToken(bench.presentation_state);
}

function humanToken(value: string): string {
  return value.replace(/[_.-]+/g, " ").replace(/_/g, " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function compactDigest(value: string): string {
  return value.length > 24 ? `${value.slice(0, 16)}…${value.slice(-8)}` : value;
}

function shortAlias(alias: string): string {
  return alias.startsWith("blind:") ? `blind:${alias.slice(6, 14)}…${alias.slice(-6)}` : compactDigest(alias);
}

function displayField(field: BlindField): string {
  if (field.tag === "BOOLEAN") return field.boolean ? "true" : "false";
  if (field.text !== "") return field.text;
  if (field.canonical_json_base64 !== "") return decodeBase64(field.canonical_json_base64);
  return humanToken(field.tag);
}

function decodeBase64(value: string): string {
  try {
    const bytes = Uint8Array.from(atob(value), (character) => character.charCodeAt(0));
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    return "[exact bytes are not displayable as UTF-8]";
  }
}

function toggle(values: string[], value: string): string[] {
  return values.includes(value) ? values.filter((current) => current !== value) : [...values, value];
}

function isNoncompilable(action: DraftInput["action"]): boolean {
  return action === "REJECT_ALL" || action === "DEFER" || action === "REFINE";
}

function changeAction(bench: BenchResponse, draft: DraftInput, action: DraftInput["action"]): DraftInput {
  const customValues = action === "CUSTOM_EXPECTATION"
    ? draft.field_acknowledgements.filter((value) => value.disposition === "ASSERT").map((value) => customValueForField(bench, value.field_id))
    : [];
  return {
    ...draft,
    action,
    allowed_aliases: action === "ALLOW_OBSERVED" ? draft.allowed_aliases : [],
    field_acknowledgements: draft.field_acknowledgements.map((value) => ({ ...value, disposition: "" })),
    custom_values: customValues,
    custom_reviewer: action === "CUSTOM_EXPECTATION" ? draft.custom_reviewer : "",
  };
}

function setAcknowledgement(bench: BenchResponse, draft: DraftInput, field: string, disposition: Exclude<FieldAcknowledgement["disposition"], "">): DraftInput {
  const customValues = draft.action === "CUSTOM_EXPECTATION" && disposition === "ASSERT"
    ? replaceCustomValue(draft.custom_values, customValueForField(bench, field))
    : disposition === "CONTEXT" ? draft.custom_values.filter((value) => value.field_id !== field) : draft.custom_values;
  return {
    ...draft,
    field_acknowledgements: draft.field_acknowledgements.map((value) => value.field_id === field ? { ...value, disposition } : value),
    custom_values: customValues,
  };
}

function emptyCustomValue(field: string): CustomValueInput {
  return { field_id: field, tag: "STRING", text: "", boolean: false, bytes_base64: "", ordered_strings: [], canonical_json: null };
}

function replaceCustomValue(values: DraftInput["custom_values"], next: DraftInput["custom_values"][number]) {
  return [...values.filter((value) => value.field_id !== next.field_id), next];
}

function customValueForField(bench: BenchResponse, fieldID: string): CustomValueInput {
  const field = bench.blind?.cards[0]?.fields.find((candidate) => candidate.field_id === fieldID);
  if (!field) return emptyCustomValue(fieldID);
  const result = { ...emptyCustomValue(fieldID), tag: field.tag, text: field.text, boolean: field.boolean };
  if (field.tag === "BYTES") result.bytes_base64 = field.text;
  if (field.tag === "ORDERED_STRING_LIST" && field.canonical_json_base64) {
    try {
      const parsed = JSON.parse(decodeBase64(field.canonical_json_base64)) as unknown;
      if (Array.isArray(parsed) && parsed.every((value) => typeof value === "string")) result.ordered_strings = parsed;
    } catch { /* package validation remains authoritative */ }
  }
  if (field.tag === "CANONICAL_JSON" && field.canonical_json_base64) {
    try { result.canonical_json = JSON.parse(decodeBase64(field.canonical_json_base64)) as unknown; } catch { /* package validation remains authoritative */ }
  }
  return result;
}
