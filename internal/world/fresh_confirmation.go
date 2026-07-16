package world

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const freshConfirmationAuthority = "U6_FINALIZED_PHYSICAL_WORLD_RECEIPT_V1"

// FreshExecutionFact is sealed evidence that one CONFIRMATION world crossed a
// concrete adapter edge and completed the clean lifecycle required for an
// eligible observation. It is deliberately not parseable or publicly
// constructible: durable semantic authority belongs to FreshConfirmation,
// which must additionally bind the complete map and the prior lineage.
//
// This is local-process evidence, not hostile-candidate attestation. The CLI
// and HTTP invocation files are written by the candidate fixtures and prove
// the declared reference protocol only.
type FreshExecutionFact struct {
	digest                  domain.Digest
	canonicalBytes          []byte
	adapter                 domain.AdapterDomain
	planDigest              domain.Digest
	candidate               domain.CandidateExecutionKey
	stimulusDigest          domain.Digest
	attemptDigest           domain.Digest
	worldDigest             domain.Digest
	processDigest           domain.Digest
	materializationDigest   domain.Digest
	materializationPolicy   domain.Digest
	portableTreeDigest      domain.Digest
	executionBindingDigest  domain.Digest
	overlayDigest           domain.Digest
	httpReadinessDigest     domain.Digest
	httpExchangeDigest      domain.Digest
	invocationReceiptDigest domain.Digest
	invocationFileDigest    domain.Digest
	rootLayoutDigest        domain.Digest
	purpose                 domain.AttemptPurpose
	instanceNonce           string
	scheduleOrdinal         int
}

type freshExecutionIdentity struct {
	SchemaVersion           string `json:"schema_version"`
	Kind                    string `json:"kind"`
	Authority               string `json:"authority"`
	Adapter                 string `json:"adapter"`
	WorldPlanDigest         string `json:"world_plan_digest"`
	CandidateExecutionKey   string `json:"candidate_execution_key"`
	StimulusDigest          string `json:"stimulus_digest"`
	AttemptArtifactDigest   string `json:"attempt_artifact_digest"`
	WorldInstanceDigest     string `json:"world_instance_digest"`
	ProcessLifecycleDigest  string `json:"process_lifecycle_digest"`
	MaterializationDigest   string `json:"materialization_manifest_digest"`
	MaterializationPolicy   string `json:"materialization_policy_digest"`
	PortableTreeDigest      string `json:"portable_tree_digest"`
	ExecutionBindingDigest  string `json:"execution_binding_digest"`
	AdapterOverlayDigest    string `json:"adapter_overlay_digest"`
	HTTPReadinessDigest     string `json:"http_readiness_digest"`
	HTTPExchangeDigest      string `json:"http_exchange_digest"`
	InvocationReceiptDigest string `json:"invocation_receipt_digest"`
	InvocationFileDigest    string `json:"invocation_file_digest"`
	RootLayoutDigest        string `json:"owned_root_layout_digest"`
	Purpose                 string `json:"attempt_purpose"`
	InstanceNonce           string `json:"instance_nonce"`
	ScheduleOrdinal         int    `json:"schedule_ordinal"`
	InvocationTrust         string `json:"invocation_trust"`
	FreshnessSemantics      string `json:"freshness_semantics"`
}

// FreshExecutionRecord is an inert strict inspection result. It deliberately
// does not expose or recreate FreshExecutionFact's live construction authority.
type FreshExecutionRecord struct{ fact FreshExecutionFact }

// InspectFreshExecutionFactWire validates every closed member of one retained
// physical fact and returns only an inert view for the confirmation parser.
func InspectFreshExecutionFactWire(exact []byte) (FreshExecutionRecord, error) {
	value, err := canon.Parse(exact)
	if err != nil {
		return FreshExecutionRecord{}, refuse(CodeInvalidRequest, "fresh execution wire is outside the strict canonical profile", err)
	}
	canonical, err := value.CanonicalChecked()
	members, object := value.Members()
	if err != nil || !object || len(members) != 25 || !bytes.Equal(canonical, exact) {
		return FreshExecutionRecord{}, refuse(CodeInvalidRequest, "fresh execution wire member set or canonical bytes differ", err)
	}
	var identity freshExecutionIdentity
	if err := json.Unmarshal(exact, &identity); err != nil {
		return FreshExecutionRecord{}, refuse(CodeInvalidRequest, "fresh execution wire typed extraction failed", err)
	}
	if identity.SchemaVersion != domain.SchemaVersion || identity.Kind != "FreshExecutionFact" ||
		identity.Authority != freshConfirmationAuthority || identity.Purpose != string(domain.AttemptConfirmation) ||
		identity.InvocationTrust != "FIXTURE_OBSERVED_LOCAL_PROTOCOL_NOT_HOSTILE_PROCESS_ATTESTATION" ||
		identity.FreshnessSemantics != "RELATIVE_FRESHNESS_REQUIRES_CROSS_LINEAGE_DISJOINTNESS" ||
		identity.ScheduleOrdinal < 0 || !validInstanceNonce(identity.InstanceNonce) {
		return FreshExecutionRecord{}, refuse(CodeInvalidRequest, "fresh execution wire closed facts differ", nil)
	}
	adapter := domain.AdapterDomain(identity.Adapter)
	if adapter != domain.AdapterCLI && adapter != domain.AdapterHTTP {
		return FreshExecutionRecord{}, refuse(CodeInvalidRequest, "fresh execution wire adapter is unsupported", nil)
	}
	candidate, err := domain.ParseCandidateExecutionKey(identity.CandidateExecutionKey)
	if err != nil {
		return FreshExecutionRecord{}, refuse(CodeInvalidRequest, "fresh execution wire candidate is invalid", err)
	}
	parse := func(raw, label string) (domain.Digest, error) {
		digest, parseErr := domain.ParseDigest(raw)
		if parseErr != nil {
			return "", refuse(CodeInvalidRequest, "fresh execution wire "+label+" digest is invalid", parseErr)
		}
		return digest, nil
	}
	plan, planErr := parse(identity.WorldPlanDigest, "plan")
	stimulus, stimulusErr := parse(identity.StimulusDigest, "stimulus")
	attempt, attemptErr := parse(identity.AttemptArtifactDigest, "attempt")
	worldDigest, worldErr := parse(identity.WorldInstanceDigest, "world")
	process, processErr := parse(identity.ProcessLifecycleDigest, "process")
	materialization, materializationErr := parse(identity.MaterializationDigest, "materialization")
	materializationPolicy, policyErr := parse(identity.MaterializationPolicy, "materialization policy")
	portableTree, treeErr := parse(identity.PortableTreeDigest, "portable tree")
	binding, bindingErr := parse(identity.ExecutionBindingDigest, "execution binding")
	overlay, overlayErr := parse(identity.AdapterOverlayDigest, "adapter overlay")
	invocation, invocationErr := parse(identity.InvocationReceiptDigest, "invocation receipt")
	invocationFile, invocationFileErr := parse(identity.InvocationFileDigest, "invocation file")
	root, rootErr := parse(identity.RootLayoutDigest, "owned root")
	if err := errors.Join(
		planErr, stimulusErr, attemptErr, worldErr, processErr, materializationErr, policyErr, treeErr,
		bindingErr, overlayErr, invocationErr, invocationFileErr, rootErr,
	); err != nil {
		return FreshExecutionRecord{}, err
	}
	var readiness, exchange domain.Digest
	if adapter == domain.AdapterHTTP {
		readiness, planErr = parse(identity.HTTPReadinessDigest, "HTTP readiness")
		exchange, stimulusErr = parse(identity.HTTPExchangeDigest, "HTTP exchange")
		if planErr != nil || stimulusErr != nil {
			return FreshExecutionRecord{}, errors.Join(planErr, stimulusErr)
		}
	} else if identity.HTTPReadinessDigest != "" || identity.HTTPExchangeDigest != "" {
		return FreshExecutionRecord{}, refuse(CodeInvalidRequest, "CLI fresh execution wire carries HTTP authority", nil)
	}
	digestRaw, err := canon.DigestBytes("FreshExecutionFact", exact)
	if err != nil {
		return FreshExecutionRecord{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return FreshExecutionRecord{}, err
	}
	fact := FreshExecutionFact{
		digest: digest, canonicalBytes: append([]byte(nil), exact...), adapter: adapter,
		planDigest: plan, candidate: candidate, stimulusDigest: stimulus, attemptDigest: attempt,
		worldDigest: worldDigest, processDigest: process, materializationDigest: materialization,
		materializationPolicy: materializationPolicy, portableTreeDigest: portableTree,
		executionBindingDigest: binding, overlayDigest: overlay, httpReadinessDigest: readiness,
		httpExchangeDigest: exchange, invocationReceiptDigest: invocation, invocationFileDigest: invocationFile,
		rootLayoutDigest: root, purpose: domain.AttemptConfirmation, instanceNonce: identity.InstanceNonce,
		scheduleOrdinal: identity.ScheduleOrdinal,
	}
	if !fact.Valid() {
		return FreshExecutionRecord{}, refuse(CodeInvalidRequest, "fresh execution wire failed exact reconstruction", nil)
	}
	return FreshExecutionRecord{fact: fact}, nil
}

func (r FreshExecutionRecord) Digest() domain.Digest     { return r.fact.Digest() }
func (r FreshExecutionRecord) PlanDigest() domain.Digest { return r.fact.PlanDigest() }
func (r FreshExecutionRecord) CandidateKey() domain.CandidateExecutionKey {
	return r.fact.CandidateKey()
}
func (r FreshExecutionRecord) StimulusDigest() domain.Digest { return r.fact.StimulusDigest() }
func (r FreshExecutionRecord) AttemptArtifactDigest() domain.Digest {
	return r.fact.AttemptArtifactDigest()
}
func (r FreshExecutionRecord) WorldDigest() domain.Digest   { return r.fact.WorldDigest() }
func (r FreshExecutionRecord) ProcessDigest() domain.Digest { return r.fact.ProcessDigest() }
func (r FreshExecutionRecord) InvocationReceiptDigest() domain.Digest {
	return r.fact.InvocationReceiptDigest()
}
func (r FreshExecutionRecord) InvocationFileDigest() domain.Digest {
	return r.fact.InvocationFileDigest()
}
func (r FreshExecutionRecord) RootLayoutDigest() domain.Digest { return r.fact.RootLayoutDigest() }
func (r FreshExecutionRecord) Purpose() domain.AttemptPurpose  { return r.fact.Purpose() }
func (r FreshExecutionRecord) InstanceNonce() string           { return r.fact.InstanceNonce() }
func (r FreshExecutionRecord) ScheduleOrdinal() int            { return r.fact.ScheduleOrdinal() }

// FreshConfirmationEvidence revalidates a completed Result at the world
// boundary. A metadata-only map, PreparedTrial, nonce, or copied digest cannot
// call this method into existence without an actual private world.Result.
func (r Result) FreshConfirmationEvidence() (FreshExecutionFact, error) {
	world := r.world
	attempt := r.finalized
	process := r.process
	if world.Purpose() != domain.AttemptConfirmation || attempt.Purpose() != domain.AttemptConfirmation ||
		world.AttemptArtifactDigest() != attempt.ArtifactDigest() ||
		process.AttemptArtifactDigest() != attempt.ArtifactDigest() || process.WorldDigest() != world.Digest() ||
		attempt.HasControls() || !cleanConfirmationProcess(process) || !exactSuccessfulStateHistory(r.states) ||
		!r.materialization.Valid() {
		return FreshExecutionFact{}, refuse(CodeInvalidRequest, "result is not one clean finalized confirmation world", nil)
	}

	rootDigest, err := validateAndDigestConfirmationRoots(r)
	if err != nil {
		return FreshExecutionFact{}, err
	}

	invocation, err := confirmationInvocationAuthority(r)
	if err != nil {
		return FreshExecutionFact{}, err
	}
	identity := freshExecutionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "FreshExecutionFact", Authority: freshConfirmationAuthority,
		Adapter: string(invocation.adapter), WorldPlanDigest: world.PlanDigest().String(),
		CandidateExecutionKey: world.CandidateKey().String(), StimulusDigest: world.StimulusDigest().String(),
		AttemptArtifactDigest: attempt.ArtifactDigest().String(), WorldInstanceDigest: world.Digest().String(),
		ProcessLifecycleDigest: process.Digest().String(), MaterializationDigest: r.materialization.ManifestDigest.String(),
		MaterializationPolicy: r.materialization.PolicyDigest.String(), PortableTreeDigest: r.materialization.PortableTreeDigest.String(),
		ExecutionBindingDigest: invocation.binding.String(), AdapterOverlayDigest: invocation.overlay.String(),
		HTTPReadinessDigest: invocation.readiness.String(), HTTPExchangeDigest: invocation.exchange.String(),
		InvocationReceiptDigest: invocation.receipt.String(), InvocationFileDigest: invocation.file.String(),
		RootLayoutDigest: rootDigest.String(), Purpose: string(world.Purpose()), InstanceNonce: world.InstanceNonce(),
		ScheduleOrdinal:    world.ScheduleOrdinal(),
		InvocationTrust:    "FIXTURE_OBSERVED_LOCAL_PROTOCOL_NOT_HOSTILE_PROCESS_ATTESTATION",
		FreshnessSemantics: "RELATIVE_FRESHNESS_REQUIRES_CROSS_LINEAGE_DISJOINTNESS",
	}
	digestRaw, canonicalBytes, err := canon.DigestTyped("FreshExecutionFact", identity)
	if err != nil {
		return FreshExecutionFact{}, err
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		return FreshExecutionFact{}, err
	}
	fact := FreshExecutionFact{
		digest: digest, canonicalBytes: canonicalBytes, adapter: invocation.adapter, planDigest: world.PlanDigest(),
		candidate: world.CandidateKey(), stimulusDigest: world.StimulusDigest(),
		attemptDigest: attempt.ArtifactDigest(), worldDigest: world.Digest(), processDigest: process.Digest(),
		materializationDigest: r.materialization.ManifestDigest, materializationPolicy: r.materialization.PolicyDigest,
		portableTreeDigest: r.materialization.PortableTreeDigest, executionBindingDigest: invocation.binding,
		overlayDigest: invocation.overlay, httpReadinessDigest: invocation.readiness, httpExchangeDigest: invocation.exchange,
		invocationReceiptDigest: invocation.receipt, invocationFileDigest: invocation.file,
		rootLayoutDigest: rootDigest, purpose: world.Purpose(),
		instanceNonce: world.InstanceNonce(), scheduleOrdinal: world.ScheduleOrdinal(),
	}
	if !fact.Valid() {
		return FreshExecutionFact{}, refuse(CodeInvalidRequest, "constructed fresh execution fact is invalid", nil)
	}
	return fact, nil
}

func cleanConfirmationProcess(process ProcessReceipt) bool {
	_, controlled := process.PrimaryControl()
	return process.Digest().Valid() && !controlled && process.MarkerExistedBeforeSpawn() &&
		process.SpawnAttempted() && process.Started() && process.ProcessGroupOwned() &&
		process.DirectChildWaited() && process.DrainsComplete() && process.FinalGroupProbeClean() &&
		!process.TeardownError() && !process.OrphanRisk() && process.DiagnosticCode() == "" &&
		process.CleanupBoundary() == processEscapeExclusion &&
		process.ProcessGroupSignalBoundary() == processGroupReuseExclusion &&
		exactSuccessfulStateHistory(process.states)
}

func exactSuccessfulStateHistory(states []domain.AttemptState) bool {
	want := []domain.AttemptState{
		domain.AttemptAllocated, domain.AttemptMaterializing, domain.AttemptStarting, domain.AttemptReady,
		domain.AttemptProbing, domain.AttemptCapturing, domain.AttemptTearingDown, domain.AttemptFinalized,
	}
	if len(states) != len(want) {
		return false
	}
	for index := range want {
		if states[index] != want[index] {
			return false
		}
	}
	return true
}

type confirmationInvocation struct {
	adapter   domain.AdapterDomain
	receipt   domain.Digest
	file      domain.Digest
	binding   domain.Digest
	overlay   domain.Digest
	readiness domain.Digest
	exchange  domain.Digest
}

func confirmationInvocationAuthority(r Result) (confirmationInvocation, error) {
	cliFixture, hasCLIFixture := r.CLIFixtureOverlay()
	cliInvocation, hasCLIInvocation := r.CLIInvocationEvidence()
	httpSeed, hasHTTPSeed := r.HTTPSeedOverlay()
	httpReadiness, hasHTTPReadiness := r.HTTPReadiness()
	httpExchange, hasHTTPExchange := r.HTTPExchange()
	httpInvocation, hasHTTPInvocation := r.HTTPInvocationEvidence()

	cliShape := hasCLIFixture && hasCLIInvocation && !hasHTTPSeed && !hasHTTPReadiness && !hasHTTPExchange && !hasHTTPInvocation
	httpShape := hasHTTPSeed && hasHTTPReadiness && hasHTTPExchange && hasHTTPInvocation && !hasCLIFixture && !hasCLIInvocation
	switch {
	case cliShape:
		ordinal, schedulePresent := r.process.ScheduleOrdinal()
		if !cliFixture.Valid() || !cliInvocation.Valid() || cliInvocation.Status() != CLIInvocationValidated ||
			!cliInvocation.AttemptIDValidated() || !cliInvocation.LogicalArgvValidated() ||
			cliInvocation.Path() != filepath.Join(r.roots.evidence, cliInvocationFilename) ||
			r.process.ExecutionAuthorityMarker() != CLIExecutionAuthorityV1 || !r.process.PhysicalExecutionEntered() ||
			r.process.InvocationEvidenceDigest() != cliInvocation.Digest() || !r.process.InvocationEvidenceValidated() ||
			r.process.StimulusDigest() != r.world.StimulusDigest() ||
			!schedulePresent || ordinal != r.world.ScheduleOrdinal() || cliFixture.Root() != r.roots.fixture ||
			cliFixture.ExecutionBindingDigest() != cliInvocation.ExecutionBindingDigest() ||
			cliFixture.ExecutionBindingDigest() != r.process.ExecutionBindingDigest() {
			return confirmationInvocation{}, refuse(CodeCLIExecutionRejected, "CLI confirmation invocation authority is incomplete", nil)
		}
		info, statErr := os.Lstat(cliInvocation.Path())
		if statErr != nil || !info.Mode().IsRegular() {
			return confirmationInvocation{}, refuse(CodeCLIExecutionRejected, "CLI invocation evidence cannot be reopened", statErr)
		}
		status, fileDigest, size, attemptOK, argvOK := readAndValidateCLIInvocation(
			cliInvocation.Path(), info.Size(), cliInvocation.ExpectedAttemptID(), cliInvocation.ExpectedLogicalArgv(),
		)
		if status != CLIInvocationValidated || fileDigest != cliInvocation.FileDigest() || size != cliInvocation.Bytes() || !attemptOK || !argvOK {
			return confirmationInvocation{}, refuse(CodeCLIExecutionRejected, "CLI invocation evidence changed after execution", nil)
		}
		return confirmationInvocation{
			adapter: domain.AdapterCLI, receipt: cliInvocation.Digest(), file: cliInvocation.FileDigest(),
			binding: cliInvocation.ExecutionBindingDigest(), overlay: cliFixture.Digest(),
		}, nil
	case httpShape:
		fileDigest, hasFileDigest := httpInvocation.FileDigest()
		if !httpSeed.Valid() || !httpReadiness.Valid() || !httpReadiness.Accepted() ||
			!httpExchange.Valid() || !httpExchange.RequestComplete() || !httpExchange.ResponseParsed() ||
			httpExchange.ResponseOverflow() || httpExchange.DiagnosticCode() != "" ||
			!httpInvocation.Valid() || !httpInvocation.Validated() || !hasFileDigest ||
			httpInvocation.Path() != filepath.Join(r.roots.evidence, httpInvocationFilename) ||
			httpReadiness.WorldDigest() != r.world.Digest() || httpExchange.WorldDigest() != r.world.Digest() ||
			httpInvocation.WorldDigest() != r.world.Digest() ||
			httpReadiness.AttemptArtifactDigest() != r.finalized.ArtifactDigest() ||
			httpExchange.AttemptArtifactDigest() != r.finalized.ArtifactDigest() ||
			httpInvocation.AttemptArtifactDigest() != r.finalized.ArtifactDigest() ||
			httpSeed.Root() != r.roots.fixture ||
			httpSeed.ExecutionBindingDigest() != httpReadiness.ExecutionBindingDigest() ||
			httpSeed.ExecutionBindingDigest() != httpExchange.ExecutionBindingDigest() ||
			httpSeed.ExecutionBindingDigest() != httpInvocation.ExecutionBindingDigest() {
			return confirmationInvocation{}, refuse(CodeHTTPExecutionRejected, "HTTP confirmation invocation authority is incomplete", nil)
		}
		info, statErr := os.Lstat(httpInvocation.Path())
		if statErr != nil || !info.Mode().IsRegular() {
			return confirmationInvocation{}, refuse(CodeHTTPExecutionRejected, "HTTP invocation evidence cannot be reopened", statErr)
		}
		status, reopenedDigest, size, attemptOK, stimulusOK, requestBytesOK, requestDigestOK, countOK :=
			readAndValidateHTTPInvocation(
				httpInvocation.Path(), info.Size(), r.process.AttemptID(), r.world.StimulusDigest(), httpExchange.RequestWire(),
			)
		if status != HTTPInvocationValidated || reopenedDigest != fileDigest || size != httpInvocation.FileBytes() ||
			!attemptOK || !stimulusOK || !requestBytesOK || !requestDigestOK || !countOK {
			return confirmationInvocation{}, refuse(CodeHTTPExecutionRejected, "HTTP invocation evidence changed after execution", nil)
		}
		return confirmationInvocation{
			adapter: domain.AdapterHTTP, receipt: httpInvocation.Digest(), file: fileDigest,
			binding: httpInvocation.ExecutionBindingDigest(), overlay: httpSeed.Digest(),
			readiness: httpReadiness.Digest(), exchange: httpExchange.Digest(),
		}, nil
	default:
		return confirmationInvocation{}, refuse(CodeInvalidRequest, "confirmation result has no single concrete adapter authority", nil)
	}
}

func validateAndDigestConfirmationRoots(r Result) (domain.Digest, error) {
	roots := []struct {
		name string
		path string
	}{
		{"attempt", r.roots.attempt}, {"candidate-parent", r.roots.candidateParent}, {"fixture", r.roots.fixture},
		{"home", r.roots.home}, {"tmp", r.roots.temporary}, {"xdg-config", r.roots.xdgConfig},
		{"xdg-cache", r.roots.xdgCache}, {"xdg-data", r.roots.xdgData}, {"xdg-state", r.roots.xdgState},
		{"state", r.roots.state}, {"evidence", r.roots.evidence},
	}
	seen := map[string]struct{}{}
	for index, root := range roots {
		canonical, err := privateCanonicalDirectory(root.path)
		if err != nil || canonical != root.path {
			return "", refuse(CodeInvalidAllocation, "confirmation owned root is no longer private and canonical", err)
		}
		if _, duplicate := seen[root.path]; duplicate {
			return "", refuse(CodeInvalidAllocation, "confirmation owned roots alias", nil)
		}
		seen[root.path] = struct{}{}
		if index > 0 && (filepath.Dir(root.path) != r.roots.attempt || filepath.Base(root.path) != root.name) {
			return "", refuse(CodeInvalidAllocation, "confirmation owned root escaped its attempt", nil)
		}
	}
	markerInfo, err := os.Lstat(r.roots.marker)
	if err != nil || !markerInfo.Mode().IsRegular() || markerInfo.Mode().Perm() != 0o600 ||
		filepath.Dir(r.roots.marker) != r.roots.evidence || filepath.Base(r.roots.marker) != markerFilename {
		return "", refuse(CodeInvalidAllocation, "confirmation marker is unavailable or changed", err)
	}
	markerBytes, err := os.ReadFile(r.roots.marker)
	if err != nil {
		return "", refuse(CodeInvalidAllocation, "confirmation marker cannot be reopened", err)
	}
	markerValue, err := canon.Parse(markerBytes)
	if err != nil || !bytes.Equal(markerValue.Canonical(), markerBytes) {
		return "", refuse(CodeInvalidAllocation, "confirmation marker is not exact canonical bytes", err)
	}
	markerMembers, markerObject := markerValue.Members()
	markerSchema, schemaOK := markerText(markerValue, "schema_version")
	markerKind, kindOK := markerText(markerValue, "kind")
	markerPlan, planOK := markerText(markerValue, "world_plan_digest")
	markerCandidate, candidateOK := markerText(markerValue, "candidate_execution_key")
	markerStimulus, stimulusOK := markerText(markerValue, "stimulus_digest")
	markerPurpose, purposeOK := markerText(markerValue, "attempt_purpose")
	markerNonce, nonceOK := markerText(markerValue, "instance_nonce")
	markerAllocation, allocationOK := markerText(markerValue, "allocation_nonce")
	markerOrdering, orderingOK := markerText(markerValue, "marker_ordering")
	markerOrdinal, ordinalOK := markerInteger(markerValue, "schedule_ordinal")
	markerDigestRaw, digestErr := canon.DigestBytes("AttemptMarker", markerBytes)
	if !markerObject || len(markerMembers) != 11 || !schemaOK || !kindOK || !planOK || !candidateOK ||
		!stimulusOK || !purposeOK || !nonceOK || !allocationOK || !orderingOK || !ordinalOK || digestErr != nil ||
		markerSchema != domain.SchemaVersion || markerKind != "AttemptMarker" || markerPlan != r.world.PlanDigest().String() ||
		markerCandidate != r.world.CandidateKey().String() || markerStimulus != r.world.StimulusDigest().String() ||
		markerPurpose != string(r.world.Purpose()) || markerNonce != r.world.InstanceNonce() ||
		markerOrdinal != int64(r.world.ScheduleOrdinal()) || markerAllocation != filepath.Base(r.roots.attempt) ||
		markerOrdering != "DURABLE_BEFORE_SPAWN" || markerDigestRaw.String() != r.finalized.ArtifactDigest().String() {
		return "", refuse(CodeInvalidAllocation, "confirmation marker lineage changed", digestErr)
	}
	if r.materialization.PublishedRoot != filepath.Join(r.roots.candidateParent, "candidate") {
		return "", refuse(CodeInvalidAllocation, "confirmation materialization escaped its owned root", nil)
	}
	identity := struct {
		SchemaVersion string   `json:"schema_version"`
		Kind          string   `json:"kind"`
		Roots         []string `json:"roots"`
		Marker        string   `json:"marker"`
		Candidate     string   `json:"candidate"`
	}{domain.SchemaVersion, "OwnedWorldRootLayout", make([]string, len(roots)), r.roots.marker, r.materialization.PublishedRoot}
	for index, root := range roots {
		identity.Roots[index] = root.name + "=" + root.path
	}
	digestRaw, _, err := canon.DigestTyped("OwnedWorldRootLayout", identity)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digestRaw.String())
}

func markerText(value canon.Value, name string) (string, bool) {
	nested, present := value.LookupMember(name)
	if !present {
		return "", false
	}
	return nested.Text()
}

func markerInteger(value canon.Value, name string) (int64, bool) {
	nested, present := value.LookupMember(name)
	if !present {
		return 0, false
	}
	return nested.Int64()
}

func (f FreshExecutionFact) Valid() bool {
	if !f.digest.Valid() || len(f.canonicalBytes) == 0 ||
		(f.adapter != domain.AdapterCLI && f.adapter != domain.AdapterHTTP) ||
		!f.planDigest.Valid() || !f.candidate.Valid() || !f.stimulusDigest.Valid() ||
		!f.attemptDigest.Valid() || !f.worldDigest.Valid() || !f.processDigest.Valid() ||
		!f.materializationDigest.Valid() || !f.invocationReceiptDigest.Valid() ||
		!f.materializationPolicy.Valid() || !f.portableTreeDigest.Valid() || !f.executionBindingDigest.Valid() ||
		!f.overlayDigest.Valid() ||
		!f.invocationFileDigest.Valid() || !f.rootLayoutDigest.Valid() ||
		f.purpose != domain.AttemptConfirmation || !validInstanceNonce(f.instanceNonce) || f.scheduleOrdinal < 0 {
		return false
	}
	identity := freshExecutionIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "FreshExecutionFact", Authority: freshConfirmationAuthority,
		Adapter: string(f.adapter), WorldPlanDigest: f.planDigest.String(), CandidateExecutionKey: f.candidate.String(),
		StimulusDigest: f.stimulusDigest.String(), AttemptArtifactDigest: f.attemptDigest.String(),
		WorldInstanceDigest: f.worldDigest.String(), ProcessLifecycleDigest: f.processDigest.String(),
		MaterializationDigest: f.materializationDigest.String(), InvocationReceiptDigest: f.invocationReceiptDigest.String(),
		MaterializationPolicy: f.materializationPolicy.String(), PortableTreeDigest: f.portableTreeDigest.String(),
		ExecutionBindingDigest: f.executionBindingDigest.String(), AdapterOverlayDigest: f.overlayDigest.String(),
		HTTPReadinessDigest: f.httpReadinessDigest.String(), HTTPExchangeDigest: f.httpExchangeDigest.String(),
		InvocationFileDigest: f.invocationFileDigest.String(), RootLayoutDigest: f.rootLayoutDigest.String(),
		Purpose: string(f.purpose), InstanceNonce: f.instanceNonce, ScheduleOrdinal: f.scheduleOrdinal,
		InvocationTrust:    "FIXTURE_OBSERVED_LOCAL_PROTOCOL_NOT_HOSTILE_PROCESS_ATTESTATION",
		FreshnessSemantics: "RELATIVE_FRESHNESS_REQUIRES_CROSS_LINEAGE_DISJOINTNESS",
	}
	if f.adapter == domain.AdapterHTTP {
		if !f.httpReadinessDigest.Valid() || !f.httpExchangeDigest.Valid() {
			return false
		}
	} else if f.httpReadinessDigest.Valid() || f.httpExchangeDigest.Valid() {
		return false
	}
	digest, canonicalBytes, err := canon.DigestTyped("FreshExecutionFact", identity)
	return err == nil && digest.String() == f.digest.String() && bytes.Equal(canonicalBytes, f.canonicalBytes)
}

func (f FreshExecutionFact) Digest() domain.Digest                      { return f.digest }
func (f FreshExecutionFact) CanonicalBytes() []byte                     { return append([]byte(nil), f.canonicalBytes...) }
func (f FreshExecutionFact) Adapter() domain.AdapterDomain              { return f.adapter }
func (f FreshExecutionFact) PlanDigest() domain.Digest                  { return f.planDigest }
func (f FreshExecutionFact) CandidateKey() domain.CandidateExecutionKey { return f.candidate }
func (f FreshExecutionFact) StimulusDigest() domain.Digest              { return f.stimulusDigest }
func (f FreshExecutionFact) AttemptArtifactDigest() domain.Digest       { return f.attemptDigest }
func (f FreshExecutionFact) WorldDigest() domain.Digest                 { return f.worldDigest }
func (f FreshExecutionFact) ProcessDigest() domain.Digest               { return f.processDigest }
func (f FreshExecutionFact) MaterializationDigest() domain.Digest       { return f.materializationDigest }
func (f FreshExecutionFact) InvocationReceiptDigest() domain.Digest     { return f.invocationReceiptDigest }
func (f FreshExecutionFact) InvocationFileDigest() domain.Digest        { return f.invocationFileDigest }
func (f FreshExecutionFact) RootLayoutDigest() domain.Digest            { return f.rootLayoutDigest }
func (f FreshExecutionFact) Purpose() domain.AttemptPurpose             { return f.purpose }
func (f FreshExecutionFact) InstanceNonce() string                      { return f.instanceNonce }
func (f FreshExecutionFact) ScheduleOrdinal() int                       { return f.scheduleOrdinal }
