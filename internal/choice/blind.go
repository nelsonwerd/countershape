package choice

import (
	"bytes"
	"encoding/base64"
	"sort"
	"strings"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/reduce"
)

const blindScope = "CANDIDATE_NEUTRAL_EXACT_WITNESS_CHOICE_V1"

const maxBlindAliasBytes = len("blind:") + 64

type BlindField struct {
	FieldID             string `json:"field_id"`
	Tag                 string `json:"tag"`
	Text                string `json:"text"`
	Boolean             bool   `json:"boolean"`
	CanonicalJSONBase64 string `json:"canonical_json_base64"`
}

type BlindCard struct {
	Alias  string       `json:"alias"`
	Fields []BlindField `json:"fields"`
}

type BlindReductionFacts struct {
	GradeStatus         string   `json:"grade_status"`
	ProposalLimit       uint64   `json:"proposal_limit"`
	CandidateTrialLimit uint64   `json:"candidate_trial_limit"`
	WallLimitMS         int64    `json:"wall_limit_ms"`
	EvaluationCount     int      `json:"evaluation_count"`
	UnresolvedCount     int      `json:"unresolved_count"`
	Limitations         []string `json:"limitations"`
	ReducerSetDigest    string   `json:"reducer_set_digest"`
	FinalSweepState     string   `json:"final_sweep_state"`
}

type BlindProjectionOperation struct {
	Name       string `json:"name"`
	RuleDigest string `json:"rule_digest"`
}

type blindDTOIdentity struct {
	SchemaVersion                   string                     `json:"schema_version"`
	Kind                            string                     `json:"kind"`
	ChoicepointDigest               string                     `json:"choicepoint_digest"`
	Scenario                        string                     `json:"scenario"`
	Scope                           string                     `json:"scope"`
	OriginalStimulusBase64          string                     `json:"original_stimulus_base64"`
	MinimizedStimulusBase64         string                     `json:"minimized_stimulus_base64"`
	Reduction                       BlindReductionFacts        `json:"reduction"`
	DiscoveryRepeatsPerCandidate    int                        `json:"discovery_repeats_per_candidate"`
	ConfirmationRepeatsPerCandidate int                        `json:"confirmation_repeats_per_candidate"`
	ProjectionMode                  string                     `json:"projection_mode"`
	ProjectionOperations            []BlindProjectionOperation `json:"projection_operations"`
	SelectableFields                []string                   `json:"selectable_fields"`
	DifferingFields                 []string                   `json:"differing_fields"`
	Cards                           []BlindCard                `json:"cards"`
	TrustWarnings                   []string                   `json:"trust_warnings"`
}

type BlindDTO struct {
	digest    domain.Digest
	identity  blindDTOIdentity
	canonical []byte
}

type BlindView struct {
	dto     BlindDTO
	aliases map[string][]ConfirmedOutcomeRef
}

// blindGroup is the candidate-neutral presentation unit. Support refs are
// retained only for the sealed alias resolver; neither their identities nor
// their count may influence the card alias or presentation order.
type blindGroup struct {
	fingerprint domain.ProjectionFingerprint
	tuple       CompleteTuple
	refs        []ConfirmedOutcomeRef
	alias       string
	orderKey    string
}

func buildBlindGroups(choicepoint domain.Digest, confirmed ConfirmedOutcomeSet) ([]blindGroup, error) {
	if !choicepoint.Valid() || !confirmed.Valid() {
		return nil, refusal(CodeInvalidConfirmedOutcomeSet, "blind grouping requires a valid Choicepoint and confirmed set")
	}
	groupsByFingerprint := map[string]*blindGroup{}
	for _, outcome := range confirmed.ordered {
		key := outcome.ref.fingerprint.String()
		current, present := groupsByFingerprint[key]
		if !present {
			alias, err := blindAliasForProjectionIdentity( // MUTANT_U6_BLIND_ALIAS_FIRST_MEMBER
				choicepoint, outcome.ref.fingerprint.String(),
			)
			if err != nil {
				return nil, err
			}
			order, err := canon.DigestBytes("BlindOutcomeOrder", []byte(key))
			if err != nil {
				return nil, err
			}
			current = &blindGroup{
				fingerprint: outcome.ref.fingerprint,
				tuple:       cloneTuple(outcome.tuple),
				alias:       alias,
				orderKey:    order.String(),
			}
			groupsByFingerprint[key] = current
		} else if tupleIdentityKey(confirmed.registry, current.tuple.Fields) != tupleIdentityKey(confirmed.registry, outcome.tuple.Fields) {
			return nil, refusal(CodeInvalidConfirmedOutcomeSet, "one exact fingerprint produced inconsistent blind facts")
		}
		current.refs = append(current.refs, outcome.ref)
	}
	groups := make([]blindGroup, 0, len(groupsByFingerprint))
	for _, value := range groupsByFingerprint {
		value.refs = append([]ConfirmedOutcomeRef(nil), value.refs...)
		groups = append(groups, *value)
	}
	// Ordering is derived only from the unexposed projection fingerprint. It
	// never reads support counts, candidate order, aliases, or reveal metadata.
	// The fingerprint tie-break keeps map iteration from becoming observable
	// even under a hypothetical order-digest collision.
	sort.Slice(groups, func(i, j int) bool { // MUTANT_U6_BLIND_ORDER_BY_SUPPORT
		if groups[i].orderKey != groups[j].orderKey {
			return groups[i].orderKey < groups[j].orderKey
		}
		return groups[i].fingerprint.String() < groups[j].fingerprint.String()
	})
	return groups, nil
}

func NewBlindView(record ChoicepointRecord) (BlindView, error) {
	if !record.Valid() || !record.confirmed.Valid() {
		return BlindView{}, refusal(CodeInvalidConfirmedOutcomeSet, "blind view requires a strict Choicepoint record")
	}
	groups, err := buildBlindGroups(record.digest, record.confirmed)
	if err != nil {
		return BlindView{}, err
	}

	transcript := record.confirmation.ReductionRunRecord().Transcript()
	unresolved := 0
	for _, entry := range transcript.Entries() {
		if entry.Decision() == reduce.Unresolved {
			unresolved++
		}
	}
	reduction := BlindReductionFacts{
		GradeStatus: record.confirmation.ReductionGradeStatus(), ProposalLimit: transcript.ProposalLimit(),
		CandidateTrialLimit: transcript.CandidateTrialLimit(), WallLimitMS: transcript.WallLimitMS(),
		EvaluationCount: len(transcript.Entries()), UnresolvedCount: unresolved,
		Limitations: transcript.Limitations(), ReducerSetDigest: transcript.ReducerSetDigest().String(),
		FinalSweepState: string(transcript.FinalSweepState()),
	}
	operations := record.plan.ProjectionDefinitionBinding().Operations()
	operationDTOs := make([]BlindProjectionOperation, len(operations))
	for index, operation := range operations {
		operationDTOs[index] = BlindProjectionOperation{Name: operation.Name, RuleDigest: operation.RuleDigest.String()}
	}
	cards, aliases, err := renderBlindGroups(groups)
	if err != nil {
		return BlindView{}, err
	}
	selectable := make([]string, 0, len(record.confirmed.registry.orderedIDs))
	for _, field := range record.confirmed.registry.Definitions() {
		selectable = append(selectable, field.ID)
	}
	differing, err := differingFields(record.confirmed.registry, groups)
	if err != nil {
		return BlindView{}, err
	}
	identity := blindDTOIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "BlindChoicepoint", ChoicepointDigest: record.digest.String(),
		Scenario: record.scenario, Scope: blindScope,
		OriginalStimulusBase64:  base64.StdEncoding.EncodeToString(record.original.canonical),
		MinimizedStimulusBase64: base64.StdEncoding.EncodeToString(record.minimized.canonical),
		Reduction:               reduction, DiscoveryRepeatsPerCandidate: record.plan.RepeatSchedule().DiscoveryRepeats,
		ConfirmationRepeatsPerCandidate: record.plan.RepeatSchedule().ConfirmationRepeats,
		ProjectionMode:                  record.mode.wire(), ProjectionOperations: operationDTOs,
		SelectableFields: selectable, DifferingFields: differing,
		Cards: cards,
		TrustWarnings: []string{
			"Exact witness evidence is not a universal behavioral-equivalence claim.",
			"Candidate-written fixture receipts are not hostile-process attestation.",
			"No majority, popularity, or support count is semantic authority.",
		},
	}
	view := BlindView{aliases: aliases}
	digestRaw, canonicalBytes, err := canon.DigestTyped("BlindChoicepoint", identity)
	if err != nil || len(canonicalBytes) > canon.MaxInputBytes {
		return BlindView{}, refusal(CodeInputLimitExceeded, "blind DTO exceeds the canonical resource profile")
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return BlindView{}, err
	}
	view.dto = BlindDTO{digest: digest, identity: identity, canonical: canonicalBytes}
	return view, nil
}

func differingFields(registry FieldRegistry, groups []blindGroup) ([]string, error) {
	if len(groups) == 0 || len(registry.orderedIDs) == 0 {
		return nil, refusal(CodeInvalidConfirmedOutcomeSet, "differing-field analysis requires a nonempty profile and blind group set")
	}
	result := make([]string, 0, len(registry.orderedIDs))
	for fieldIndex, fieldID := range registry.orderedIDs { // MUTANT_P07B_ALL_FIELDS_DIFFERING
		if len(groups[0].tuple.Fields) != len(registry.orderedIDs) || groups[0].tuple.Fields[fieldIndex].FieldID != fieldID {
			return nil, refusal(CodeInvalidConfirmedOutcomeSet, "blind tuple order differs from the profile registry")
		}
		first := groups[0].tuple.Fields[fieldIndex].Value.identityKey(registry.mode)
		var differs bool
		for groupIndex := 1; groupIndex < len(groups); groupIndex++ {
			fields := groups[groupIndex].tuple.Fields
			if len(fields) != len(registry.orderedIDs) || fields[fieldIndex].FieldID != fieldID {
				return nil, refusal(CodeInvalidConfirmedOutcomeSet, "blind tuple order differs from the profile registry")
			}
			if fields[fieldIndex].Value.identityKey(registry.mode) != first {
				differs = true
				break
			}
		}
		if differs {
			result = append(result, fieldID)
		}
	}
	return result, nil
}

func blindAlias(choicepoint domain.Digest, fingerprint domain.ProjectionFingerprint) (string, error) {
	return blindAliasForProjectionIdentity(choicepoint, fingerprint.String())
}

func blindAliasForProjectionIdentity(choicepoint domain.Digest, projectionIdentity string) (string, error) {
	digest, _, err := canon.DigestTyped("BlindOutcomeAlias", struct {
		SchemaVersion      string `json:"schema_version"`
		Kind               string `json:"kind"`
		ChoicepointDigest  string `json:"choicepoint_digest"`
		ProjectionIdentity string `json:"projection_fingerprint"`
	}{domain.SchemaVersion, "BlindOutcomeAlias", choicepoint.String(), projectionIdentity})
	if err != nil {
		return "", err
	}
	return "blind:" + strings.TrimPrefix(digest.String(), "sha256:"), nil
}

func renderBlindGroups(groups []blindGroup) ([]BlindCard, map[string][]ConfirmedOutcomeRef, error) {
	cards := make([]BlindCard, len(groups))
	aliases := make(map[string][]ConfirmedOutcomeRef, len(groups))
	for index, current := range groups {
		if current.alias == "" || len(current.refs) == 0 {
			return nil, nil, refusal(CodeInvalidConfirmedOutcomeSet, "blind group lacks an alias or supporting outcome")
		}
		if _, duplicate := aliases[current.alias]; duplicate {
			return nil, nil, refusal(CodeInvalidConfirmedOutcomeSet, "distinct blind groups collided on one alias")
		}
		fields := make([]BlindField, len(current.tuple.Fields))
		for fieldIndex, field := range current.tuple.Fields {
			fields[fieldIndex] = blindField(field)
		}
		cards[index] = BlindCard{Alias: current.alias, Fields: fields} // MUTANT_U6_BLIND_PRESENT_CANDIDATE_IDENTITY
		aliases[current.alias] = append([]ConfirmedOutcomeRef(nil), current.refs...)
	}
	return cards, aliases, nil
}

func blindField(field FieldValue) BlindField {
	value := field.Value
	result := BlindField{FieldID: field.FieldID, Tag: string(value.Tag()), Text: value.Text(), Boolean: value.Boolean()}
	if value.Tag() == ValueBytes {
		result.Text = base64.StdEncoding.EncodeToString(value.Bytes())
	}
	if value.Tag() == ValueCanonicalJSON || value.Tag() == ValueOrderedStringList {
		result.CanonicalJSONBase64 = base64.StdEncoding.EncodeToString(value.CanonicalBytes())
		result.Text = ""
	}
	return result
}

func (d BlindDTO) CanonicalBytes() []byte { return append([]byte(nil), d.canonical...) }
func (d BlindDTO) Digest() domain.Digest  { return d.digest }
func (d BlindDTO) Cards() []BlindCard {
	result := make([]BlindCard, len(d.identity.Cards))
	for index, card := range d.identity.Cards {
		result[index] = BlindCard{Alias: card.Alias, Fields: append([]BlindField(nil), card.Fields...)}
	}
	return result
}
func (d BlindDTO) Scenario() string       { return d.identity.Scenario }
func (d BlindDTO) ProjectionMode() string { return d.identity.ProjectionMode }
func (d BlindDTO) SelectableFields() []string {
	return append([]string(nil), d.identity.SelectableFields...)
}
func (d BlindDTO) DifferingFields() []string {
	return append([]string(nil), d.identity.DifferingFields...)
}

func (v BlindView) DTO() BlindDTO {
	return BlindDTO{digest: v.dto.digest, identity: v.dto.identity, canonical: append([]byte(nil), v.dto.canonical...)}
}

func (v BlindView) normalizeAliases(raw []string) ([]string, []ConfirmedOutcomeRef, error) {
	if err := v.admitAliases(raw); err != nil {
		return nil, nil, err
	}
	aliases := make([]string, len(raw))
	copy(aliases, raw)
	sort.Strings(aliases)
	resolved, err := v.resolveAliases(aliases)
	if err != nil {
		return nil, nil, err
	}
	return aliases, resolved, nil
}

func (v BlindView) admitAliases(raw []string) error {
	if raw == nil {
		return refusal(CodeOmittedObservedSelections, "blind aliases must be explicit")
	}
	if len(raw) > len(v.aliases) {
		return refusal(CodeInputLimitExceeded, "blind alias selection exceeds the rendered-card ceiling")
	}
	remaining := maxBlindAliasBytes * len(v.aliases)
	for _, alias := range raw {
		if len(alias) > maxBlindAliasBytes || len(alias) > remaining {
			return refusal(CodeInputLimitExceeded, "blind alias exceeds the closed identity byte ceiling")
		}
		remaining -= len(alias)
	}
	return nil
}

func (v BlindView) resolveAliases(aliases []string) ([]ConfirmedOutcomeRef, error) {
	if aliases == nil {
		return nil, refusal(CodeOmittedObservedSelections, "blind aliases must be explicit")
	}
	seenAliases := map[string]struct{}{}
	seenRefs := map[string]struct{}{}
	result := []ConfirmedOutcomeRef{}
	for _, alias := range aliases {
		refs, present := v.aliases[alias]
		if alias == "" || !present {
			return nil, refusal(CodeUnconfirmedOutcomeSelection, "blind alias is foreign or unknown")
		}
		if _, duplicate := seenAliases[alias]; duplicate {
			return nil, refusal(CodeDuplicateObservedSelection, "blind alias is duplicated")
		}
		seenAliases[alias] = struct{}{}
		// Every supporting ref is expanded. Selecting only the first member
		// would turn candidate multiplicity into hidden semantic authority.
		for _, ref := range refs {
			if _, duplicate := seenRefs[ref.id.text]; duplicate {
				continue
			}
			seenRefs[ref.id.text] = struct{}{}
			result = append(result, ref)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].id.text < result[j].id.text })
	return result, nil
}

func (v BlindView) same(other BlindView) bool {
	return bytes.Equal(v.dto.canonical, other.dto.canonical)
}
