package app

import (
	"github.com/nelsonwerd/countershape/internal/canon"
	corespec "github.com/nelsonwerd/countershape/internal/spec"
)

type ToolView struct {
	Name              string `json:"name"`
	VersionConstraint string `json:"version_constraint"`
}

type ScheduleView struct {
	Authority           string  `json:"authority"`
	DiscoveryRepeats    *int    `json:"discovery_repeats"`
	ConfirmationRepeats *int    `json:"confirmation_repeats"`
	Concurrency         *string `json:"concurrency"`
	Rotation            *string `json:"rotation"`
}

type BudgetView struct {
	CandidateCount            int   `json:"candidate_count"`
	MaterializedEntryCount    int   `json:"materialized_entry_count"`
	MaterializedBytesPerWorld int64 `json:"materialized_bytes_per_world"`
	SingleBlobBytes           int64 `json:"single_blob_bytes"`
	ReadinessMS               int64 `json:"readiness_ms"`
	ProbeMS                   int64 `json:"probe_ms"`
	TeardownMS                int64 `json:"teardown_ms"`
	StdoutBytes               int64 `json:"stdout_bytes"`
	StderrBytes               int64 `json:"stderr_bytes"`
	HTTPBodyBytes             int64 `json:"http_body_bytes"`
	ProposedShrinkStimuli     int   `json:"proposed_shrink_stimuli"`
	TotalCandidateTrials      int   `json:"total_candidate_trials"`
	ShrinkWallMS              int64 `json:"shrink_wall_ms"`
}

type SpecSummary struct {
	Input            string       `json:"input"`
	InputBytes       int          `json:"input_bytes"`
	CanonicalBytes   int          `json:"canonical_bytes"`
	SchemaVersion    string       `json:"schema_version"`
	Kind             string       `json:"kind"`
	SourceDigest     string       `json:"source_digest"`
	AdapterDomain    string       `json:"adapter_domain"`
	AdapterVersion   string       `json:"adapter_version"`
	ExecutionShape   string       `json:"execution_shape"`
	StartArgv        []string     `json:"start_argv"`
	SetupArgv        []string     `json:"setup_argv"`
	EnvironmentNames []string     `json:"environment_names"`
	SecretSlotNames  []string     `json:"secret_slot_names"`
	RequiredTools    []ToolView   `json:"required_tools"`
	Schedule         ScheduleView `json:"schedule"`
	Budgets          BudgetView   `json:"budgets"`
}

type ValidatedSpec struct {
	parsed  corespec.ParsedSource
	summary SpecSummary
}

func (v ValidatedSpec) ParsedSource() corespec.ParsedSource { return v.parsed }
func (v ValidatedSpec) Summary() SpecSummary                { return cloneSpecSummary(v.summary) }

type presentationError struct{ detail string }

func (e *presentationError) Error() string { return "SPEC_PRESENTATION_INVARIANT: " + e.detail }

func parseValidatedSpec(input sourceInput) (ValidatedSpec, error) {
	parsed, err := corespec.ParseSource(input.exact)
	if err != nil {
		return ValidatedSpec{}, err
	}
	canonical := parsed.CanonicalBytes()
	value, err := canon.Parse(canonical)
	if err != nil {
		return ValidatedSpec{}, &presentationError{detail: "strict source could not be reopened"}
	}
	summary, err := projectSpecSummary(value)
	if err != nil {
		return ValidatedSpec{}, err
	}
	summary.Input = input.displayPath
	summary.InputBytes = len(input.exact)
	summary.CanonicalBytes = len(canonical)
	summary.SourceDigest = parsed.Digest().String()
	return ValidatedSpec{parsed: parsed, summary: summary}, nil
}

func projectSpecSummary(root canon.Value) (SpecSummary, error) {
	schema, err := requiredText(root, "schema_version")
	if err != nil {
		return SpecSummary{}, err
	}
	kind, err := requiredText(root, "kind")
	if err != nil {
		return SpecSummary{}, err
	}
	adapter, err := requiredObject(root, "adapter")
	if err != nil {
		return SpecSummary{}, err
	}
	domain, err := requiredText(adapter, "domain")
	if err != nil {
		return SpecSummary{}, err
	}
	version, err := requiredText(adapter, "adapter_version")
	if err != nil {
		return SpecSummary{}, err
	}
	shape, err := requiredText(root, "execution_shape")
	if err != nil {
		return SpecSummary{}, err
	}
	start, err := requiredStringArray(root, "start_argv")
	if err != nil {
		return SpecSummary{}, err
	}
	setup, err := optionalStringArray(root, "setup_argv")
	if err != nil {
		return SpecSummary{}, err
	}
	environmentNames, err := objectArrayTextField(root, "environment", "name")
	if err != nil {
		return SpecSummary{}, err
	}
	secretNames, err := objectArrayTextField(root, "secret_slots", "name")
	if err != nil {
		return SpecSummary{}, err
	}
	tools, err := toolViews(root)
	if err != nil {
		return SpecSummary{}, err
	}
	schedule, err := scheduleView(root)
	if err != nil {
		return SpecSummary{}, err
	}
	budgets, err := budgetView(root)
	if err != nil {
		return SpecSummary{}, err
	}
	return SpecSummary{
		SchemaVersion: schema, Kind: kind, AdapterDomain: domain, AdapterVersion: version,
		ExecutionShape: shape, StartArgv: start, SetupArgv: setup, EnvironmentNames: environmentNames,
		SecretSlotNames: secretNames, RequiredTools: tools, Schedule: schedule, Budgets: budgets,
	}, nil
}

func requiredObject(root canon.Value, name string) (canon.Value, error) {
	value, present := root.LookupMember(name)
	if !present || value.Kind() != canon.KindObject {
		return canon.Value{}, &presentationError{detail: "missing object " + name}
	}
	return value, nil
}

func requiredText(root canon.Value, name string) (string, error) {
	value, present := root.LookupMember(name)
	if !present {
		return "", &presentationError{detail: "missing text " + name}
	}
	text, ok := value.Text()
	if !ok {
		return "", &presentationError{detail: "non-text " + name}
	}
	return text, nil
}

func requiredStringArray(root canon.Value, name string) ([]string, error) {
	value, present := root.LookupMember(name)
	if !present {
		return nil, &presentationError{detail: "missing array " + name}
	}
	return stringArray(value, name)
}

func optionalStringArray(root canon.Value, name string) ([]string, error) {
	value, present := root.LookupMember(name)
	if !present {
		return []string{}, nil
	}
	return stringArray(value, name)
}

func stringArray(value canon.Value, label string) ([]string, error) {
	items, ok := value.Elements()
	if !ok {
		return nil, &presentationError{detail: "non-array " + label}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.Text()
		if !ok {
			return nil, &presentationError{detail: "non-text item " + label}
		}
		result = append(result, text)
	}
	return result, nil
}

func objectArrayTextField(root canon.Value, name, field string) ([]string, error) {
	value, present := root.LookupMember(name)
	if !present {
		return []string{}, nil
	}
	items, ok := value.Elements()
	if !ok {
		return nil, &presentationError{detail: "non-array " + name}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, err := requiredText(item, field)
		if err != nil {
			return nil, err
		}
		result = append(result, text)
	}
	return result, nil
}

func toolViews(root canon.Value) ([]ToolView, error) {
	value, present := root.LookupMember("required_tools")
	if !present {
		return []ToolView{}, nil
	}
	items, ok := value.Elements()
	if !ok {
		return nil, &presentationError{detail: "non-array required_tools"}
	}
	result := make([]ToolView, 0, len(items))
	for _, item := range items {
		name, err := requiredText(item, "name")
		if err != nil {
			return nil, err
		}
		constraint, err := requiredText(item, "version_constraint")
		if err != nil {
			return nil, err
		}
		result = append(result, ToolView{Name: name, VersionConstraint: constraint})
	}
	return result, nil
}

func scheduleView(root canon.Value) (ScheduleView, error) {
	value, present := root.LookupMember("repeat_schedule")
	if !present {
		return ScheduleView{Authority: "CORE_DEFAULTS_APPLY_ON_COMPILE"}, nil
	}
	result := ScheduleView{Authority: "DECLARED_SOURCE"}
	if err := optionalInt(value, "discovery_repeats", &result.DiscoveryRepeats); err != nil {
		return ScheduleView{}, err
	}
	if err := optionalInt(value, "confirmation_repeats", &result.ConfirmationRepeats); err != nil {
		return ScheduleView{}, err
	}
	if err := optionalText(value, "concurrency", &result.Concurrency); err != nil {
		return ScheduleView{}, err
	}
	if err := optionalText(value, "rotation", &result.Rotation); err != nil {
		return ScheduleView{}, err
	}
	return result, nil
}

func budgetView(root canon.Value) (BudgetView, error) {
	defaults := corespec.DefaultBudgets()
	if readiness, present := root.LookupMember("readiness"); !present {
		defaults.ReadinessMS = 0
	} else if kind, textErr := requiredText(readiness, "kind"); textErr == nil && kind == "NONE" {
		defaults.ReadinessMS = 0
	}
	result := BudgetView{
		CandidateCount: defaults.CandidateCount, MaterializedEntryCount: defaults.MaterializedEntryCount,
		MaterializedBytesPerWorld: defaults.MaterializedBytesPerWorld, SingleBlobBytes: defaults.SingleBlobBytes,
		ReadinessMS: defaults.ReadinessMS, ProbeMS: defaults.ProbeMS, TeardownMS: defaults.TeardownMS,
		StdoutBytes: defaults.StdoutBytes, StderrBytes: defaults.StderrBytes, HTTPBodyBytes: defaults.HTTPBodyBytes,
		ProposedShrinkStimuli: defaults.ProposedShrinkStimuli, TotalCandidateTrials: defaults.TotalCandidateTrials,
		ShrinkWallMS: defaults.ShrinkWallMS,
	}
	value, present := root.LookupMember("budgets")
	if !present {
		return result, nil
	}
	integerTargets := map[string]*int{
		"candidate_count": &result.CandidateCount, "materialized_entry_count": &result.MaterializedEntryCount,
		"proposed_shrink_stimuli": &result.ProposedShrinkStimuli, "total_candidate_trials": &result.TotalCandidateTrials,
	}
	int64Targets := map[string]*int64{
		"materialized_bytes_per_world": &result.MaterializedBytesPerWorld, "single_blob_bytes": &result.SingleBlobBytes,
		"readiness_ms": &result.ReadinessMS, "probe_ms": &result.ProbeMS, "teardown_ms": &result.TeardownMS,
		"stdout_bytes": &result.StdoutBytes, "stderr_bytes": &result.StderrBytes, "http_body_bytes": &result.HTTPBodyBytes,
		"shrink_wall_ms": &result.ShrinkWallMS,
	}
	for name, target := range integerTargets {
		if member, ok := value.LookupMember(name); ok {
			number, valid := member.Int64()
			if !valid {
				return BudgetView{}, &presentationError{detail: "non-integer budget " + name}
			}
			*target = int(number)
		}
	}
	for name, target := range int64Targets {
		if member, ok := value.LookupMember(name); ok {
			number, valid := member.Int64()
			if !valid {
				return BudgetView{}, &presentationError{detail: "non-integer budget " + name}
			}
			*target = number
		}
	}
	return result, nil
}

func optionalInt(root canon.Value, name string, target **int) error {
	value, present := root.LookupMember(name)
	if !present {
		return nil
	}
	number, ok := value.Int64()
	if !ok {
		return &presentationError{detail: "non-integer " + name}
	}
	converted := int(number)
	*target = &converted
	return nil
}

func optionalText(root canon.Value, name string, target **string) error {
	value, present := root.LookupMember(name)
	if !present {
		return nil
	}
	text, ok := value.Text()
	if !ok {
		return &presentationError{detail: "non-text " + name}
	}
	*target = &text
	return nil
}

func cloneSpecSummary(input SpecSummary) SpecSummary {
	result := input
	result.StartArgv = append([]string(nil), input.StartArgv...)
	result.SetupArgv = append([]string(nil), input.SetupArgv...)
	result.EnvironmentNames = append([]string(nil), input.EnvironmentNames...)
	result.SecretSlotNames = append([]string(nil), input.SecretSlotNames...)
	result.RequiredTools = append([]ToolView(nil), input.RequiredTools...)
	return result
}
