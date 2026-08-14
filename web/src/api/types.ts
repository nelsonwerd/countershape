export type PresentationState =
  | "empty"
  | "preparing"
  | "active"
  | "partial"
  | "error"
  | "completed"
  | "unstable"
  | "uncomparable"
  | "incomplete"
  | "discovered"
  | "decision-ready"
  | "predicate-editing"
  | "identity-reveal"
  | "resolved"
  | "reject-all-resolved"
  | "deferred"
  | "stale"
  | "invalidated";

export interface SessionResponse {
  schema_version: string;
  study_id: string;
  csrf: string;
  revision_digest: string;
  presentation_state: PresentationState;
}

export interface BlindField {
  field_id: string;
  tag: string;
  text: string;
  boolean: boolean;
  canonical_json_base64: string;
}

export interface BlindCard {
  alias: string;
  fields: BlindField[];
}

export interface BlindDTO {
  schema_version: string;
  kind: "BlindChoicepoint";
  choicepoint_digest: string;
  scenario: string;
  scope: string;
  original_stimulus_base64: string;
  minimized_stimulus_base64: string;
  reduction: {
    grade_status: string;
    proposal_limit: number;
    candidate_trial_limit: number;
    wall_limit_ms: number;
    evaluation_count: number;
    unresolved_count: number;
    limitations: string[];
    reducer_set_digest: string;
    final_sweep_state: string;
  };
  discovery_repeats_per_candidate: number;
  confirmation_repeats_per_candidate: number;
  projection_mode: string;
  projection_operations: Array<{ name: string; rule_digest: string }>;
  selectable_fields: string[];
  differing_fields: string[];
  cards: BlindCard[];
  trust_warnings: string[];
}

export interface RevealDTO {
  schema_version: string;
  kind: string;
  groups: Array<{
    alias: string;
    candidates: Array<{
      candidate_execution_key: string;
      tree_identity_digest: string;
      display_ref: string;
      producer_metadata: string;
    }>;
  }>;
  exclusions: unknown[];
}

export interface FieldAcknowledgement {
  field_id: string;
  disposition: "ASSERT" | "CONTEXT" | "";
}

export interface CustomValueInput {
  field_id: string;
  tag: string;
  text: string;
  boolean: boolean;
  bytes_base64: string;
  ordered_strings: string[];
  canonical_json: unknown;
}

export interface DraftInput {
  action: "ALLOW_OBSERVED" | "CUSTOM_EXPECTATION" | "REJECT_ALL" | "DEFER" | "REFINE";
  allowed_aliases: string[];
  field_acknowledgements: FieldAcknowledgement[];
  custom_values: CustomValueInput[];
  custom_reviewer: string;
}

export interface DecisionResult {
  decision_digest: string;
  action: string;
  selected_fields: string[];
  nonasserted_fields: string[];
  early_reveal: boolean;
  changed_after_reveal: boolean;
  change_rationale: string;
  compilation_status: string;
  emitted_files: string[];
  authority: string;
  authority_ceiling: string;
}

export interface BenchResponse {
  schema_version: string;
  kind: "StudioBench";
  study_id: string;
  presentation_state: PresentationState;
  study_state: string;
  candidate_state: string;
  choicepoint_state: string;
  summary: string;
  next_action: string;
  mutable: boolean;
  future_authority_unimplemented: boolean;
  trust_warning: string;
  network_mode: string;
  revision_digest: string;
  session_state: "BLIND_OPEN" | "PROVISIONAL_RECORDED" | "REVEALED" | "POST_REVEAL_RECORDED" | "FINALIZED";
  blind?: BlindDTO;
  reveal?: RevealDTO;
  field_acknowledgements: FieldAcknowledgement[];
  reviewed_surfaces: string[];
  draft?: DraftInput;
  result?: DecisionResult;
  nonclaims: string[];
}

export interface MutationEnvelope {
  expected_revision: string;
  surface?: string;
  draft?: DraftInput;
  rationale?: string;
  actor?: string;
  annotation?: string;
}

export interface APIErrorBody {
  code: string;
  detail: string;
}
