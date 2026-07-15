// Package spec strictly parses an inert source specification and compiles it
// into a fully materialized deterministic WorldPlan. JSON token authority lives
// in internal/canon; this package never accepts map[string]any or an exported
// struct that bypasses the closed source grammar.
package spec

import "github.com/nelsonwerd/countershape/internal/domain"

type budgetOverrides struct {
	CandidateCount            *int
	MaterializedEntryCount    *int
	MaterializedBytesPerWorld *int64
	SingleBlobBytes           *int64
	ReadinessMS               *int64
	ProbeMS                   *int64
	TeardownMS                *int64
	StdoutBytes               *int64
	StderrBytes               *int64
	HTTPBodyBytes             *int64
	ProposedShrinkStimuli     *int
	TotalCandidateTrials      *int
	ShrinkWallMS              *int64
}

type repeatOverrides struct {
	DiscoveryRepeats    *int
	ConfirmationRepeats *int
}

// ParsedSource is an immutable source admitted by ParseSource. Its fields are
// deliberately private so production callers cannot bypass strict unknown-key,
// duplicate-name, numeric, or UTF-8 validation with an ordinary Go struct.
type ParsedSource struct {
	valid                       bool
	digest                      domain.Digest
	canonicalBytes              []byte
	candidateSetDigest          domain.Digest
	materializationPolicyDigest domain.Digest
	comparisonEnvelopeDigest    domain.Digest
	adapter                     domain.Adapter
	executionShape              domain.ExecutionShape
	startArgv                   []string
	setupArgv                   []string
	environment                 []domain.EnvironmentEntry
	secretSlots                 []domain.SecretSlot
	fixtureRecipeDigest         domain.Digest
	readiness                   domain.Readiness
	capturePolicyDigest         domain.Digest
	projectionDefinitionDigest  domain.Digest
	repeats                     repeatOverrides
	requiredTools               []domain.RequiredTool
	budgets                     budgetOverrides
}

func (source ParsedSource) CanonicalBytes() []byte {
	return append([]byte(nil), source.canonicalBytes...)
}

func (source ParsedSource) Digest() domain.Digest { return source.digest }

func DefaultBudgets() domain.Budgets {
	return domain.Budgets{
		CandidateCount:            2,
		MaterializedEntryCount:    25000,
		MaterializedBytesPerWorld: 268435456,
		SingleBlobBytes:           33554432,
		ReadinessMS:               5000,
		ProbeMS:                   3000,
		TeardownMS:                3000,
		StdoutBytes:               1048576,
		StderrBytes:               1048576,
		HTTPBodyBytes:             1048576,
		ProposedShrinkStimuli:     40,
		TotalCandidateTrials:      300,
		ShrinkWallMS:              600000,
	}
}

func Compile(source ParsedSource, projection domain.ProjectionDefinitionBinding) (domain.WorldPlan, error) {
	if !source.valid {
		return domain.WorldPlan{}, refuse(CodeUnparsedSource, "$", "source must come from ParseSource")
	}
	if !projection.Valid() || projection.Digest() != source.projectionDefinitionDigest || projection.AdapterDomain() != source.adapter.Domain {
		return domain.WorldPlan{}, refuse(CodeUnresolvedProjectionDefinition, "$.projection_definition_digest", "declared digest and adapter domain must resolve to one full projection-definition binding")
	}
	declaration := declarationForSource(source)
	if err := domain.ValidateWorldPlanDeclaration(declaration); err != nil {
		return domain.WorldPlan{}, err
	}
	return domain.NewWorldPlan(domain.WorldPlanConfig{
		CandidateSetDigest:          declaration.CandidateSetDigest,
		MaterializationPolicyDigest: declaration.MaterializationPolicyDigest,
		ComparisonEnvelopeDigest:    declaration.ComparisonEnvelopeDigest,
		Adapter:                     declaration.Adapter,
		ExecutionShape:              declaration.ExecutionShape,
		StartArgv:                   declaration.StartArgv,
		SetupArgv:                   declaration.SetupArgv,
		Environment:                 declaration.Environment,
		SecretSlots:                 declaration.SecretSlots,
		FixtureRecipeDigest:         declaration.FixtureRecipeDigest,
		Readiness:                   declaration.Readiness,
		CapturePolicyDigest:         declaration.CapturePolicyDigest,
		ProjectionDefinition:        projection,
		RepeatSchedule:              declaration.RepeatSchedule,
		RequiredTools:               declaration.RequiredTools,
		Budgets:                     declaration.Budgets,
	})
}

func declarationForSource(source ParsedSource) domain.WorldPlanDeclarationConfig {
	budgets := DefaultBudgets()
	if source.readiness.Kind == domain.ReadinessNone {
		budgets.ReadinessMS = 0
	}
	applyBudgetOverrides(&budgets, source.budgets)
	discoveryRepeats := 3
	confirmationRepeats := 3
	if source.repeats.DiscoveryRepeats != nil {
		discoveryRepeats = *source.repeats.DiscoveryRepeats
	}
	if source.repeats.ConfirmationRepeats != nil {
		confirmationRepeats = *source.repeats.ConfirmationRepeats
	}

	return domain.WorldPlanDeclarationConfig{
		CandidateSetDigest:          source.candidateSetDigest,
		MaterializationPolicyDigest: source.materializationPolicyDigest,
		ComparisonEnvelopeDigest:    source.comparisonEnvelopeDigest,
		Adapter:                     source.adapter,
		ExecutionShape:              source.executionShape,
		StartArgv:                   append([]string(nil), source.startArgv...),
		SetupArgv:                   append([]string(nil), source.setupArgv...),
		Environment:                 append([]domain.EnvironmentEntry(nil), source.environment...),
		SecretSlots:                 append([]domain.SecretSlot(nil), source.secretSlots...),
		FixtureRecipeDigest:         source.fixtureRecipeDigest,
		Readiness:                   source.readiness,
		CapturePolicyDigest:         source.capturePolicyDigest,
		ProjectionDefinitionDigest:  source.projectionDefinitionDigest,
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats:    discoveryRepeats,
			ConfirmationRepeats: confirmationRepeats,
		},
		RequiredTools: append([]domain.RequiredTool(nil), source.requiredTools...),
		Budgets:       budgets,
	}
}

func applyBudgetOverrides(target *domain.Budgets, overrides budgetOverrides) {
	if overrides.CandidateCount != nil {
		target.CandidateCount = *overrides.CandidateCount
	}
	if overrides.MaterializedEntryCount != nil {
		target.MaterializedEntryCount = *overrides.MaterializedEntryCount
	}
	if overrides.MaterializedBytesPerWorld != nil {
		target.MaterializedBytesPerWorld = *overrides.MaterializedBytesPerWorld
	}
	if overrides.SingleBlobBytes != nil {
		target.SingleBlobBytes = *overrides.SingleBlobBytes
	}
	if overrides.ReadinessMS != nil {
		target.ReadinessMS = *overrides.ReadinessMS
	}
	if overrides.ProbeMS != nil {
		target.ProbeMS = *overrides.ProbeMS
	}
	if overrides.TeardownMS != nil {
		target.TeardownMS = *overrides.TeardownMS
	}
	if overrides.StdoutBytes != nil {
		target.StdoutBytes = *overrides.StdoutBytes
	}
	if overrides.StderrBytes != nil {
		target.StderrBytes = *overrides.StderrBytes
	}
	if overrides.HTTPBodyBytes != nil {
		target.HTTPBodyBytes = *overrides.HTTPBodyBytes
	}
	if overrides.ProposedShrinkStimuli != nil {
		target.ProposedShrinkStimuli = *overrides.ProposedShrinkStimuli
	}
	if overrides.TotalCandidateTrials != nil {
		target.TotalCandidateTrials = *overrides.TotalCandidateTrials
	}
	if overrides.ShrinkWallMS != nil {
		target.ShrinkWallMS = *overrides.ShrinkWallMS
	}
}
