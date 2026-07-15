// Package http_invoices is the physical U4 reference study. It executes four
// immutable Node-core services in fresh worlds and crosses into generic truth
// only through typed HTTP evidence.
package http_invoices

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/internal/observe"
	"github.com/nelsonwerd/countershape/internal/spec"
	"github.com/nelsonwerd/countershape/internal/world"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
	"github.com/nelsonwerd/countershape/testkit/httpfixture"
)

type Config struct {
	Root             string
	GitExecutable    string
	NodeExecutable   string
	Repetitions      int
	MaxTotalTrials   int
	WallBudget       time.Duration
	CandidateOrder   []httpfixture.CandidateRole
	DisplayLabels    map[httpfixture.CandidateRole]string
	ProducerMetadata string
	// StimulusOverride is a testkit-only seam for physically executing one
	// already-validated typed neighbor. Nil selects the locked reference case.
	StimulusOverride *counterhttp.HTTPStimulus
}

type TrialEvidence struct {
	Role                httpfixture.CandidateRole
	Slot                observe.ScheduledTrial
	Admitted            bool
	Result              world.Result
	Measurements        domain.InstanceMeasurements
	Observation         counterhttp.HTTPCapturedObservation
	Projection          counterhttp.HTTPProjectionResult
	Projected           bool
	ProjectionRejection *counterhttp.ProjectionRejection
}

type StudyResult struct {
	Root                 string
	Elapsed              time.Duration
	SourceSpecDigest     domain.Digest
	SourceSpecBytes      []byte
	Plan                 domain.WorldPlan
	Envelope             domain.ComparisonEnvelope
	Stimulus             counterhttp.HTTPStimulus
	Binding              counterhttp.HTTPExecutionBinding
	StartSpec            counterhttp.HTTPStartSpec
	CapturePolicy        counterhttp.HTTPCapturePolicy
	Readiness            counterhttp.HTTPReadinessContract
	ProjectionDefinition counterhttp.HTTPProjectionDefinition
	Observation          observe.ObservationRun
	OutcomeMap           compare.CandidateOutcomeMap
	HasOutcomeMap        bool
	CandidateRoles       map[domain.CandidateExecutionKey]httpfixture.CandidateRole
	Trials               []TrialEvidence
	DisplayLabels        map[httpfixture.CandidateRole]string
	ProducerMetadata     string
}

func DefaultConfig(root, gitExecutable, nodeExecutable string) Config {
	return Config{
		Root: root, GitExecutable: gitExecutable, NodeExecutable: nodeExecutable,
		Repetitions: 3, MaxTotalTrials: 12, WallBudget: 45 * time.Second,
		CandidateOrder: httpfixture.Roles(),
	}
}

func Run(ctx context.Context, config Config) (study StudyResult, returnErr error) {
	startedAt := time.Now()
	if ctx == nil || config.Repetitions < 1 || config.Repetitions > 5 ||
		config.MaxTotalTrials < 1 || config.WallBudget <= 0 {
		return StudyResult{}, fmt.Errorf("invalid HTTP invoice study configuration")
	}
	roles, err := normalizeRoles(config.CandidateOrder)
	if err != nil {
		return StudyResult{}, err
	}
	runRoot, err := newPrivateDirectory(config.Root, "http-invoices-run-")
	if err != nil {
		return StudyResult{}, err
	}
	gitScratch, err := newPrivateDirectory(runRoot, "git-scratch-")
	if err != nil {
		return StudyResult{}, err
	}
	inspectionRoot, err := newPrivateDirectory(runRoot, "inspection-target-")
	if err != nil {
		return StudyResult{}, err
	}
	allocationRoot, err := newPrivateDirectory(runRoot, "attempts-")
	if err != nil {
		return StudyResult{}, err
	}
	toolScratch, err := newPrivateDirectory(runRoot, "tool-scratch-")
	if err != nil {
		return StudyResult{}, err
	}

	fixtureRepository, err := gitrepo.Init(ctx, config.GitExecutable, runRoot, gitrepo.SHA1)
	if err != nil {
		return StudyResult{}, err
	}
	for _, role := range roles {
		files, fileErr := httpfixture.CandidateFiles(role)
		if fileErr != nil {
			return StudyResult{}, fileErr
		}
		commit, commitErr := fixtureRepository.CommitFiles(ctx, files, "", "HTTP invoice candidate "+string(role))
		if commitErr != nil {
			return StudyResult{}, commitErr
		}
		if updateErr := fixtureRepository.UpdateRef(ctx, "refs/heads/"+string(role), commit); updateErr != nil {
			return StudyResult{}, updateErr
		}
	}

	repository, err := gitobj.OpenRepository(ctx, gitobj.OpenConfig{
		GitExecutable: config.GitExecutable, Repository: fixtureRepository.Root, ScratchRoot: gitScratch,
	})
	if err != nil {
		return StudyResult{}, err
	}
	defer func() {
		if closeErr := repository.Close(); returnErr == nil && closeErr != nil {
			returnErr = closeErr
		}
	}()
	pins := make([]gitobj.PinnedTree, 0, len(roles))
	roleByTreeIdentity := make(map[domain.Digest]httpfixture.CandidateRole, len(roles))
	for _, role := range roles {
		pinned, pinErr := repository.Pin(ctx, "refs/heads/"+string(role))
		if pinErr != nil {
			return StudyResult{}, pinErr
		}
		pins = append(pins, pinned)
		roleByTreeIdentity[pinned.IdentityDigest()] = role
	}
	selected, err := gitobj.SelectTrees(pins...)
	if err != nil {
		return StudyResult{}, err
	}
	materializationPolicy, err := gitobj.NewPolicy(16, 1<<20, 1<<19)
	if err != nil {
		return StudyResult{}, err
	}
	declaration, err := gitobj.InspectSelected(ctx, selected, materializationPolicy, inspectionRoot)
	if err != nil {
		return StudyResult{}, err
	}

	stimulus, err := newInvoiceStimulus()
	if err != nil {
		return StudyResult{}, err
	}
	if config.StimulusOverride != nil {
		if !config.StimulusOverride.Valid() {
			return StudyResult{}, fmt.Errorf("invalid HTTP invoice stimulus override")
		}
		stimulus = *config.StimulusOverride
	}
	startSpec, err := counterhttp.NewHTTPStartSpec(httpfixture.Entrypoint)
	if err != nil {
		return StudyResult{}, err
	}
	fixtureRecipe, err := counterhttp.NewHTTPFixtureRecipe()
	if err != nil {
		return StudyResult{}, err
	}
	readiness, err := counterhttp.NewHTTPReadinessContract()
	if err != nil {
		return StudyResult{}, err
	}
	capturePolicy, err := counterhttp.NewHTTPCapturePolicy(counterhttp.HTTPCapturePolicyConfig{
		StatusLineBytes: 1024, HeaderBytes: 32 << 10, HeaderCount: 64, BodyBytes: 64 << 10,
	})
	if err != nil {
		return StudyResult{}, err
	}
	projection, err := counterhttp.NewHTTPProjectionDefinition()
	if err != nil {
		return StudyResult{}, err
	}
	envelope, err := newStudyEnvelope()
	if err != nil {
		return StudyResult{}, err
	}
	runnerDigest, err := digestBytes("HTTPStudyRunner", []byte("world.ExecuteHTTP/U4/opaque-binding/inherited-listener/v1"))
	if err != nil {
		return StudyResult{}, err
	}
	plan, sourceDigest, sourceBytes, err := compileStudyPlan(studyPlanInput{
		CandidateSetDigest: declaration.Digest(), MaterializationPolicyDigest: materializationPolicy.Digest(),
		ComparisonEnvelopeDigest: envelope.Digest(), RunnerDigest: runnerDigest,
		StartArgv: startSpec.LogicalArgv(), FixtureRecipeDigest: fixtureRecipe.Digest(),
		CapturePolicy: capturePolicy, ProjectionDefinition: projection.Binding(),
		Repetitions: config.Repetitions, CandidateCount: len(roles),
	})
	if err != nil {
		return StudyResult{}, err
	}
	executionBinding, err := counterhttp.BindExecution(plan, stimulus, startSpec, capturePolicy, readiness, projection)
	if err != nil {
		return StudyResult{}, err
	}
	boundCandidates, err := declaration.Bind(plan)
	if err != nil {
		return StudyResult{}, err
	}
	candidateByKey := make(map[domain.CandidateExecutionKey]gitobj.BoundCandidate, len(boundCandidates))
	roleByKey := make(map[domain.CandidateExecutionKey]httpfixture.CandidateRole, len(boundCandidates))
	for _, candidate := range boundCandidates {
		key := candidate.Binding().Key()
		role, present := roleByTreeIdentity[candidate.TreeIdentityDigest()]
		if !present {
			return StudyResult{}, fmt.Errorf("candidate has unknown immutable tree identity %s", candidate.TreeIdentityDigest())
		}
		candidateByKey[key] = candidate
		roleByKey[key] = role
	}
	toolRegistry, err := world.NewToolRegistry(ctx, toolScratch, world.ToolSpec{
		Name: "node", AbsolutePath: config.NodeExecutable,
		VersionConstraint: "executed-major-only", VersionArgs: []string{"--version"},
	})
	if err != nil {
		return StudyResult{}, err
	}
	roster := make([]domain.CandidateExecutionKey, 0, len(boundCandidates))
	for _, candidate := range boundCandidates {
		roster = append(roster, candidate.Binding().Key())
	}
	trialEvidence := make([]TrialEvidence, 0, config.Repetitions*len(roster))
	observationRun, err := observe.RunObservation(ctx, observe.ObservationConfig{
		Plan: plan, Purpose: domain.AttemptDiscovery, Envelope: envelope, CandidateRoster: roster,
		Repetitions: config.Repetitions,
		Budget:      observe.TrialBudget{MaxTotalTrials: config.MaxTotalTrials, WallBudget: config.WallBudget},
	}, func(trialContext context.Context, slot observe.ScheduledTrial) (observe.PreparedTrial, error) {
		candidate, present := candidateByKey[slot.CandidateKey()]
		if !present {
			return observe.PreparedTrial{}, fmt.Errorf("scheduled candidate is outside the opaque roster")
		}
		worldResult, executeErr := world.ExecuteHTTP(trialContext, world.HTTPRequest{
			Binding: executionBinding, Candidate: candidate, Tools: toolRegistry, AllocationRoot: allocationRoot,
			Purpose:         domain.AttemptDiscovery,
			InstanceNonce:   fmt.Sprintf("u4-discovery-%d-%d", slot.Repetition(), slot.Ordinal()),
			ScheduleOrdinal: slot.Ordinal(),
		})
		if executeErr != nil {
			return observe.PreparedTrial{}, executeErr
		}
		measurements, measurementErr := studyMeasurements(envelope, executionBinding, worldResult)
		if measurementErr != nil {
			return observe.PreparedTrial{}, measurementErr
		}
		observation, captureErr := counterhttp.AdaptWorldResult(worldResult, executionBinding, slot.Ordinal())
		if captureErr != nil {
			return observe.PreparedTrial{}, captureErr
		}
		evidence := TrialEvidence{
			Role: roleByKey[slot.CandidateKey()], Slot: slot, Result: worldResult,
			Measurements: measurements, Observation: observation,
		}
		if worldResult.FinalizedAttempt().HasControls() {
			prepared, prepareErr := observe.NewPreparedControlledTrial(
				slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements,
			)
			if prepareErr == nil {
				trialEvidence = append(trialEvidence, evidence)
			}
			return prepared, prepareErr
		}
		projected, projectErr := projection.Project(observation)
		if projectErr != nil {
			var rejection *counterhttp.ProjectionRejection
			if !errors.As(projectErr, &rejection) {
				return observe.PreparedTrial{}, projectErr
			}
			rejectionEvidence, bridgeErr := counterhttp.PrepareProjectionRejectionEvidence(observation, rejection)
			if bridgeErr != nil {
				return observe.PreparedTrial{}, bridgeErr
			}
			prepared, prepareErr := observe.NewPreparedProjectionRejectedTrial(
				slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements, rejectionEvidence,
			)
			if prepareErr == nil {
				evidence.ProjectionRejection = rejection
				trialEvidence = append(trialEvidence, evidence)
			}
			return prepared, prepareErr
		}
		structural, bridgeErr := counterhttp.PrepareStructuralCapture(
			worldResult.World(), worldResult.FinalizedAttempt(), observation, projected,
		)
		if bridgeErr != nil {
			return observe.PreparedTrial{}, bridgeErr
		}
		prepared, prepareErr := observe.NewPreparedCapturedTrial(
			slot, worldResult.World(), worldResult.FinalizedAttempt(), measurements, structural,
		)
		if prepareErr == nil {
			evidence.Projection = projected
			evidence.Projected = true
			trialEvidence = append(trialEvidence, evidence)
		}
		return prepared, prepareErr
	})
	if err != nil {
		return StudyResult{}, err
	}
	admittedAttempts := make(map[domain.Digest]struct{})
	for _, batch := range observationRun.Batches() {
		for _, digest := range batch.AttemptDigests() {
			admittedAttempts[digest] = struct{}{}
		}
	}
	for index := range trialEvidence {
		_, trialEvidence[index].Admitted = admittedAttempts[trialEvidence[index].Result.FinalizedAttempt().ArtifactDigest()]
	}

	study = StudyResult{
		Root: runRoot, Elapsed: time.Since(startedAt), SourceSpecDigest: sourceDigest,
		SourceSpecBytes: append([]byte(nil), sourceBytes...), Plan: plan, Envelope: envelope,
		Stimulus: stimulus, Binding: executionBinding, StartSpec: startSpec, CapturePolicy: capturePolicy,
		Readiness: readiness, ProjectionDefinition: projection, Observation: observationRun,
		CandidateRoles: cloneRoleMap(roleByKey), Trials: append([]TrialEvidence(nil), trialEvidence...),
		DisplayLabels: cloneLabels(config.DisplayLabels), ProducerMetadata: config.ProducerMetadata,
	}
	if mapInput, present := observationRun.OutcomeMapInput(); present {
		outcome, outcomeErr := compare.NewCandidateOutcomeMap(
			mapInput.StimulusDigest(), mapInput.EnvelopeDigest(), mapInput.Roster(), mapInput.Batches(),
		)
		if outcomeErr != nil {
			return StudyResult{}, outcomeErr
		}
		study.OutcomeMap = outcome
		study.HasOutcomeMap = true
	}
	return study, nil
}

func newInvoiceStimulus() (counterhttp.HTTPStimulus, error) {
	return newInvoiceStimulusWithSeed(httpfixture.SeedJSON())
}

func newInvoiceStimulusWithSeed(seedBytes []byte) (counterhttp.HTTPStimulus, error) {
	query := make([]counterhttp.HTTPQueryEntry, 0, 5)
	for _, input := range [][2]string{
		{"actor_tenant", "tenant-a"}, {"role", "support"}, {"tag", "first"}, {"tag", "second"},
	} {
		entry, err := counterhttp.QueryValue(input[0], input[1])
		if err != nil {
			return counterhttp.HTTPStimulus{}, err
		}
		query = append(query, entry)
	}
	flag, err := counterhttp.QueryFlag("audit")
	if err != nil {
		return counterhttp.HTTPStimulus{}, err
	}
	query = append(query, flag)
	headers := make([]counterhttp.HTTPRequestHeader, 0, 9)
	for _, input := range [][2]string{
		{"accept", "application/json"},
		{"x-countershape-tenant", "tenant-a"},
		{"x-countershape-role", "support"},
		{"x-countershape-audit", "required"},
		{"x-countershape-tag", "first"},
		{"x-countershape-tag", "second"},
		{"x-countershape-trace", "first"},
		{"x-countershape-trace", "second"},
		{"x-countershape-present-empty", ""},
	} {
		header, headerErr := counterhttp.NewRequestHeader(input[0], input[1])
		if headerErr != nil {
			return counterhttp.HTTPStimulus{}, headerErr
		}
		headers = append(headers, header)
	}
	seed, err := counterhttp.NewSeedFile(
		httpfixture.SeedFilename, seedBytes, counterhttp.SeedMode0644,
	)
	if err != nil {
		return counterhttp.HTTPStimulus{}, err
	}
	return counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: counterhttp.MethodGET, Path: "/v1/invoices/inv-204", Query: query,
		Headers: headers, Body: counterhttp.AbsentBody(), Seeds: []counterhttp.HTTPSeedFile{seed},
	})
}

type studyPlanInput struct {
	CandidateSetDigest          domain.Digest
	MaterializationPolicyDigest domain.Digest
	ComparisonEnvelopeDigest    domain.Digest
	RunnerDigest                domain.Digest
	StartArgv                   []string
	FixtureRecipeDigest         domain.Digest
	CapturePolicy               counterhttp.HTTPCapturePolicy
	ProjectionDefinition        domain.ProjectionDefinitionBinding
	Repetitions                 int
	CandidateCount              int
}

type studySourceAdapter struct {
	Domain         string `json:"domain"`
	AdapterVersion string `json:"adapter_version"`
	RunnerDigest   string `json:"runner_digest"`
}

type studySourceReadiness struct {
	Kind       string `json:"kind"`
	SignalName string `json:"signal_name"`
}

type studySourceSpec struct {
	SchemaVersion               string                    `json:"schema_version"`
	Kind                        string                    `json:"kind"`
	CandidateSetDigest          string                    `json:"candidate_set_digest"`
	MaterializationPolicyDigest string                    `json:"materialization_policy_digest"`
	ComparisonEnvelopeDigest    string                    `json:"comparison_envelope_digest"`
	Adapter                     studySourceAdapter        `json:"adapter"`
	ExecutionShape              string                    `json:"execution_shape"`
	StartArgv                   []string                  `json:"start_argv"`
	SetupArgv                   []string                  `json:"setup_argv"`
	Environment                 []domain.EnvironmentEntry `json:"environment"`
	SecretSlots                 []domain.SecretSlot       `json:"secret_slots"`
	FixtureRecipeDigest         string                    `json:"fixture_recipe_digest"`
	Readiness                   studySourceReadiness      `json:"readiness"`
	CapturePolicyDigest         string                    `json:"capture_policy_digest"`
	ProjectionDefinitionDigest  string                    `json:"projection_definition_digest"`
	RepeatSchedule              domain.RepeatSchedule     `json:"repeat_schedule"`
	RequiredTools               []domain.RequiredTool     `json:"required_tools"`
	Budgets                     domain.Budgets            `json:"budgets"`
}

func compileStudyPlan(input studyPlanInput) (domain.WorldPlan, domain.Digest, []byte, error) {
	budgets := domain.Budgets{
		CandidateCount: input.CandidateCount, MaterializedEntryCount: 16,
		MaterializedBytesPerWorld: 1 << 20, SingleBlobBytes: 1 << 19,
		ReadinessMS: 1500, ProbeMS: 2000, TeardownMS: 1000,
		StdoutBytes: 64 << 10, StderrBytes: 64 << 10, HTTPBodyBytes: input.CapturePolicy.BodyBytes(),
		ProposedShrinkStimuli: 0, TotalCandidateTrials: input.CandidateCount * input.Repetitions * 2,
		ShrinkWallMS: 1000,
	}
	sourceIdentity := studySourceSpec{
		SchemaVersion: "countershape-source/v1", Kind: "SourceSpec",
		CandidateSetDigest:          input.CandidateSetDigest.String(),
		MaterializationPolicyDigest: input.MaterializationPolicyDigest.String(),
		ComparisonEnvelopeDigest:    input.ComparisonEnvelopeDigest.String(),
		Adapter: studySourceAdapter{
			Domain: string(domain.AdapterHTTP), AdapterVersion: "http/v1", RunnerDigest: input.RunnerDigest.String(),
		},
		ExecutionShape: string(domain.OneLoopbackHTTPRequest), StartArgv: append([]string(nil), input.StartArgv...),
		SetupArgv: []string{},
		Environment: []domain.EnvironmentEntry{
			{Name: "LANG", Value: "C"}, {Name: "LC_ALL", Value: "C"},
			{Name: "NODE_NO_WARNINGS", Value: "1"}, {Name: "NO_COLOR", Value: "1"}, {Name: "TZ", Value: "UTC"},
		},
		SecretSlots: []domain.SecretSlot{}, FixtureRecipeDigest: input.FixtureRecipeDigest.String(),
		Readiness:                  studySourceReadiness{Kind: string(domain.FixtureOwnedReadiness), SignalName: counterhttp.ReadinessSignalNameV1},
		CapturePolicyDigest:        input.CapturePolicy.Digest().String(),
		ProjectionDefinitionDigest: input.ProjectionDefinition.Digest().String(),
		RepeatSchedule: domain.RepeatSchedule{
			DiscoveryRepeats: input.Repetitions, ConfirmationRepeats: input.Repetitions,
			Concurrency: domain.ScheduleSequential, Rotation: domain.ScheduleRotationStartByRepetitionV1,
		},
		RequiredTools: []domain.RequiredTool{{Name: "node", VersionConstraint: "executed-major-only"}},
		Budgets:       budgets,
	}
	sourceBytes, err := canon.CanonicalizeTyped(sourceIdentity)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	// The study must retain the exact inert source that compiled its plan.
	parsedSource, err := spec.ParseSource(sourceBytes)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	plan, err := spec.Compile(parsedSource, input.ProjectionDefinition)
	if err != nil {
		return domain.WorldPlan{}, "", nil, err
	}
	return plan, parsedSource.Digest(), parsedSource.CanonicalBytes(), nil
}

func normalizeRoles(input []httpfixture.CandidateRole) ([]httpfixture.CandidateRole, error) {
	if len(input) != 4 {
		return nil, fmt.Errorf("HTTP invoice study requires exactly four candidate roles")
	}
	seen := make(map[httpfixture.CandidateRole]struct{}, 4)
	for _, role := range input {
		if !role.Valid() {
			return nil, fmt.Errorf("invalid HTTP invoice role %q", role)
		}
		if _, duplicate := seen[role]; duplicate {
			return nil, fmt.Errorf("duplicate HTTP invoice role %q", role)
		}
		seen[role] = struct{}{}
	}
	return append([]httpfixture.CandidateRole(nil), input...), nil
}

func newPrivateDirectory(parent, pattern string) (string, error) {
	if !filepath.IsAbs(parent) {
		return "", fmt.Errorf("study root must be absolute")
	}
	canonicalParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	directory, err := os.MkdirTemp(canonicalParent, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", err
	}
	if resolved != directory {
		return "", fmt.Errorf("study directory is not canonical")
	}
	return directory, nil
}

func digestBytes(kind string, bytes []byte) (domain.Digest, error) {
	digest, err := canon.DigestBytes(kind, bytes)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

func cloneRoleMap(input map[domain.CandidateExecutionKey]httpfixture.CandidateRole) map[domain.CandidateExecutionKey]httpfixture.CandidateRole {
	result := make(map[domain.CandidateExecutionKey]httpfixture.CandidateRole, len(input))
	for key, role := range input {
		result[key] = role
	}
	return result
}

func cloneLabels(input map[httpfixture.CandidateRole]string) map[httpfixture.CandidateRole]string {
	result := make(map[httpfixture.CandidateRole]string, len(input))
	for role, label := range input {
		result[role] = label
	}
	return result
}

const (
	httpStudyCWDPolicy          = "MATERIALIZED_ROOT"
	dimensionToolDigest         = "tool executable digest"
	dimensionToolMajor          = "tool major version"
	dimensionExecutionAuthority = "HTTP execution authority"
	dimensionLogicalArgv        = "logical argv"
	dimensionCWDPolicy          = "working-directory policy"
	dimensionReadinessProtocol  = "readiness protocol"
	dimensionWorldDigest        = "world instance digest"
	dimensionAttemptDigest      = "attempt artifact digest"
	dimensionScheduleOrdinal    = "schedule ordinal"
	dimensionPID                = "process id"
	dimensionPhysicalExecution  = "physical execution entered"
	dimensionWorkingDirectory   = "working directory"
	dimensionFixtureRoot        = "fixture root"
	dimensionStateRoot          = "scratch state root"
	dimensionSeedOverlay        = "HTTP seed overlay receipt"
	dimensionEndpoint           = "allocated loopback endpoint"
	dimensionPort               = "allocated loopback port"
	dimensionReadinessReceipt   = "readiness receipt"
	dimensionExchangeReceipt    = "HTTP exchange receipt"
	dimensionInvocationEvidence = "HTTP invocation evidence receipt"
)

func newStudyEnvelope() (domain.ComparisonEnvelope, error) {
	required := []struct {
		name   string
		source domain.MeasuredSource
	}{
		{dimensionToolDigest, domain.MeasuredToolReceipt},
		{dimensionToolMajor, domain.MeasuredToolReceipt},
		{dimensionExecutionAuthority, domain.MeasuredProcessReceipt},
		{dimensionLogicalArgv, domain.MeasuredProcessReceipt},
		{dimensionCWDPolicy, domain.MeasuredProcessReceipt},
		{dimensionReadinessProtocol, domain.MeasuredProcessReceipt},
	}
	recorded := []struct {
		name   string
		source domain.MeasuredSource
	}{
		{dimensionWorldDigest, domain.MeasuredWorldInstance},
		{dimensionAttemptDigest, domain.MeasuredWorldInstance},
		{dimensionScheduleOrdinal, domain.MeasuredWorldInstance},
		{dimensionPID, domain.MeasuredProcessReceipt},
		{dimensionPhysicalExecution, domain.MeasuredProcessReceipt},
		{dimensionWorkingDirectory, domain.MeasuredProcessReceipt},
		{dimensionFixtureRoot, domain.MeasuredFilesystemProbe},
		{dimensionStateRoot, domain.MeasuredFilesystemProbe},
		{dimensionSeedOverlay, domain.MeasuredFilesystemProbe},
		{dimensionEndpoint, domain.MeasuredProcessReceipt},
		{dimensionPort, domain.MeasuredProcessReceipt},
		{dimensionReadinessReceipt, domain.MeasuredProcessReceipt},
		{dimensionExchangeReceipt, domain.MeasuredProcessReceipt},
		{dimensionInvocationEvidence, domain.MeasuredProcessReceipt},
	}
	measured := make([]domain.MeasuredDimension, 0, len(required)+len(recorded))
	requiredEqual := make([]domain.RequiredEqualDimension, 0, len(required))
	tolerated := make([]domain.ToleratedDimension, 0, len(recorded))
	for _, dimension := range required {
		measured = append(measured, domain.MeasuredDimension{
			Name: dimension.name, Source: dimension.source, Comparison: domain.CompareExact,
		})
		requiredEqual = append(requiredEqual, domain.RequiredEqualDimension{Name: dimension.name})
	}
	for _, dimension := range recorded {
		measured = append(measured, domain.MeasuredDimension{
			Name: dimension.name, Source: dimension.source, Comparison: domain.CompareRecordedOnly,
		})
		tolerance := domain.MayDifferRecorded
		if dimension.name == dimensionStateRoot {
			tolerance = domain.ProjectedCapturePlaceholder
		}
		tolerated = append(tolerated, domain.ToleratedDimension{Name: dimension.name, Tolerance: tolerance})
	}
	return domain.NewComparisonEnvelope(domain.ComparisonEnvelopeConfig{
		Version: "http-invoices-study/darwin-v1", Measured: measured, RequiredEqual: requiredEqual,
		Tolerated: tolerated, Rejected: []domain.RejectedDimension{},
		Uncontrolled: []string{"host network availability", "kernel scheduling and wall-clock timing", "trusted candidate host file reads"},
	})
}

func studyMeasurements(
	envelope domain.ComparisonEnvelope,
	binding counterhttp.HTTPExecutionBinding,
	result world.Result,
) (domain.InstanceMeasurements, error) {
	if !binding.Valid() || binding.PlanDigest() != result.World().PlanDigest() {
		return domain.InstanceMeasurements{}, fmt.Errorf("HTTP measurement authority does not match the executed world")
	}
	process := result.Process()
	materialization := result.Materialization()
	if !materialization.Valid() || materialization.PublishedRoot == "" {
		return domain.InstanceMeasurements{}, fmt.Errorf("HTTP measurement is missing its materialized working directory")
	}
	seed, hasSeed := result.HTTPSeedOverlay()
	readiness, hasReadiness := result.HTTPReadiness()
	exchange, hasExchange := result.HTTPExchange()
	invocation, hasInvocation := result.HTTPInvocationEvidence()
	readinessProtocol := ""
	endpoint := ""
	port := int64(0)
	readinessStatus := "NOT_APPLIED"
	if hasReadiness {
		readinessProtocol = readiness.Protocol()
		endpoint = readiness.Endpoint()
		port = int64(readiness.Port())
		readinessStatus = fmt.Sprintf("accepted=%t;diagnostic=%s", readiness.Accepted(), readiness.DiagnosticCode())
	}
	exchangeStatus := "NOT_APPLIED"
	if hasExchange {
		exchangeStatus = fmt.Sprintf("parsed=%t;diagnostic=%s", exchange.ResponseParsed(), exchange.DiagnosticCode())
	}
	invocationStatus := "NOT_INSPECTED"
	if hasInvocation {
		invocationStatus = string(invocation.Status())
	}
	seedMeasurement, err := canonicalOptionalReceipt(hasSeed, seed.Digest(), "PRIVATE_SEED_OVERLAY")
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	readinessMeasurement, err := canonicalOptionalReceipt(hasReadiness, readiness.Digest(), readinessStatus)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	exchangeMeasurement, err := canonicalOptionalReceipt(hasExchange, exchange.Digest(), exchangeStatus)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	invocationMeasurement, err := canonicalOptionalReceipt(hasInvocation, invocation.Digest(), invocationStatus)
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	logicalArgv, err := canonicalStrings(process.LogicalArgv())
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	physicalExecution, err := canonicalPhysicalExecution(process.SpawnAttempted(), process.Started())
	if err != nil {
		return domain.InstanceMeasurements{}, err
	}
	values := []struct {
		name   string
		source domain.MeasuredSource
		value  any
	}{
		{dimensionToolDigest, domain.MeasuredToolReceipt, process.ToolExecutableDigest().String()},
		{dimensionToolMajor, domain.MeasuredToolReceipt, int64(process.ToolMajor())},
		{dimensionExecutionAuthority, domain.MeasuredProcessReceipt, counterhttp.HTTPExecutionAuthorityV1},
		{dimensionLogicalArgv, domain.MeasuredProcessReceipt, logicalArgv},
		{dimensionCWDPolicy, domain.MeasuredProcessReceipt, httpStudyCWDPolicy},
		{dimensionReadinessProtocol, domain.MeasuredProcessReceipt, readinessProtocol},
		{dimensionWorldDigest, domain.MeasuredWorldInstance, result.World().Digest().String()},
		{dimensionAttemptDigest, domain.MeasuredWorldInstance, result.FinalizedAttempt().ArtifactDigest().String()},
		{dimensionScheduleOrdinal, domain.MeasuredWorldInstance, int64(result.World().ScheduleOrdinal())},
		{dimensionPID, domain.MeasuredProcessReceipt, int64(process.PID())},
		{dimensionPhysicalExecution, domain.MeasuredProcessReceipt, physicalExecution},
		{dimensionWorkingDirectory, domain.MeasuredProcessReceipt, materialization.PublishedRoot},
		{dimensionFixtureRoot, domain.MeasuredFilesystemProbe, result.Roots().Fixture()},
		{dimensionStateRoot, domain.MeasuredFilesystemProbe, result.Roots().State()},
		{dimensionSeedOverlay, domain.MeasuredFilesystemProbe, seedMeasurement},
		{dimensionEndpoint, domain.MeasuredProcessReceipt, endpoint},
		{dimensionPort, domain.MeasuredProcessReceipt, port},
		{dimensionReadinessReceipt, domain.MeasuredProcessReceipt, readinessMeasurement},
		{dimensionExchangeReceipt, domain.MeasuredProcessReceipt, exchangeMeasurement},
		{dimensionInvocationEvidence, domain.MeasuredProcessReceipt, invocationMeasurement},
	}
	measurements := make([]domain.MeasurementValue, 0, len(values))
	for _, input := range values {
		var value canon.Value
		switch typed := input.value.(type) {
		case string:
			value, err = canon.String(typed)
		case int64:
			value, err = canon.Integer(typed)
		case bool:
			value = canon.Bool(typed)
		case canon.Value:
			value = typed
		default:
			err = fmt.Errorf("unsupported measurement value for %s", input.name)
		}
		if err != nil {
			return domain.InstanceMeasurements{}, err
		}
		measurements = append(measurements, domain.MeasurementValue{Name: input.name, Source: input.source, Value: value})
	}
	return domain.NewInstanceMeasurements(envelope, result.World(), measurements)
}

func canonicalPhysicalExecution(spawnAttempted, started bool) (canon.Value, error) {
	spawnValue := canon.Bool(spawnAttempted)
	startedValue := canon.Bool(started)
	return canon.Object([]canon.Member{
		{Name: "spawn_attempted", Value: spawnValue},
		{Name: "started", Value: startedValue},
	}...)
}

func canonicalOptionalReceipt(present bool, digest domain.Digest, status string) (canon.Value, error) {
	presence := "ABSENT"
	digestValue := ""
	if present {
		if !digest.Valid() {
			return canon.Value{}, fmt.Errorf("present optional HTTP receipt has no digest")
		}
		presence = "PRESENT"
		digestValue = digest.String()
	} else if digest.Valid() {
		return canon.Value{}, fmt.Errorf("absent optional HTTP receipt carries a digest")
	}
	presenceValue, err := canon.String(presence)
	if err != nil {
		return canon.Value{}, err
	}
	digestCanonical, err := canon.String(digestValue)
	if err != nil {
		return canon.Value{}, err
	}
	statusValue, err := canon.String(status)
	if err != nil {
		return canon.Value{}, err
	}
	return canon.Object(
		canon.Member{Name: "digest", Value: digestCanonical},
		canon.Member{Name: "presence", Value: presenceValue},
		canon.Member{Name: "status", Value: statusValue},
	)
}

func canonicalStrings(input []string) (canon.Value, error) {
	values := make([]canon.Value, len(input))
	for index, item := range input {
		value, err := canon.String(item)
		if err != nil {
			return canon.Value{}, err
		}
		values[index] = value
	}
	return canon.Array(values...)
}

func (r StudyResult) RolesByFingerprint() map[httpfixture.CandidateRole]domain.ProjectionFingerprint {
	result := make(map[httpfixture.CandidateRole]domain.ProjectionFingerprint)
	if !r.HasOutcomeMap {
		return result
	}
	for _, entry := range r.OutcomeMap.Entries() {
		if role, present := r.CandidateRoles[entry.CandidateKey]; present {
			result[role] = entry.ProjectionFingerprint
		}
	}
	return result
}

func (r StudyResult) CanonicalCandidateRoster() []domain.CandidateExecutionKey {
	result := make([]domain.CandidateExecutionKey, 0, len(r.CandidateRoles))
	for key := range r.CandidateRoles {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}
