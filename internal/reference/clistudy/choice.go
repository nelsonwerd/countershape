package clistudy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodeemit "github.com/nelsonwerd/countershape/internal/emit/node"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	"github.com/nelsonwerd/countershape/internal/store"
)

type choiceActionAudit struct {
	action                 choice.Action
	decision               choice.DecisionRecord
	preparationRefusalCode string
	preparationErr         error
	destinationPath        string
	destinationBefore      []string
	destinationAfter       []string
	customReviewer         string
	customReviewEvidence   domain.Digest
}

// choiceAudit retains decisions, session states, and exact tuple bytes. Its
// status facts are recomputed from the retained choice authorities.
type choiceAudit struct {
	decisionDigest    domain.Digest
	earlyReveal       bool
	blindState        choice.SessionState
	provisionalState  choice.SessionState
	revealedState     choice.SessionState
	finalState        choice.SessionState
	ambiguityCode     choice.RefusalCode
	allowedTupleBytes [][]byte
	noncompilableCode string
	destinationBefore []string
	destinationAfter  []string
	actions           []choiceActionAudit
	validationProof   domain.Digest
}

type choiceAuditProofAction struct {
	Action                 string   `json:"action"`
	DecisionDigest         string   `json:"decision_digest"`
	DecisionCanonicalSHA   string   `json:"decision_canonical_sha256"`
	PreparationRefusalCode string   `json:"preparation_refusal_code"`
	PreparationError       string   `json:"preparation_error"`
	DestinationPath        string   `json:"destination_path"`
	DestinationBefore      []string `json:"destination_before"`
	DestinationAfter       []string `json:"destination_after"`
	CustomReviewer         string   `json:"custom_reviewer"`
	CustomReviewEvidence   string   `json:"custom_review_evidence"`
}

const (
	cliChoiceActor      = "u7-reference-operator"
	cliChoiceAnnotation = "Accept exactly the default-exit correlated config and argv precedence outcomes."
	cliCustomMode       = "reviewed-custom"
	cliCustomReviewer   = "u7-cli-custom-reviewer"
	cliCustomAnnotation = "Separately reviewed exact custom CLI expectation."
)

func requiredChoiceSurfaces() []choice.ReviewSurface {
	return []choice.ReviewSurface{
		choice.SurfaceOriginalWitness,
		choice.SurfaceMinimizedWitness,
		choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations,
		choice.SurfaceNonassertedFields,
	}
}

func cliObservedSelectedFields() []string {
	return []string{
		string(countercli.CLIFieldExitCode),
		string(countercli.CLIFieldStdoutJSONMode),
		string(countercli.CLIFieldStdoutJSONSource),
	}
}

func visitChoiceSurfaces(session choice.Session, includeProvenance bool) (choice.Session, error) {
	surfaces := requiredChoiceSurfaces()
	if includeProvenance {
		surfaces = append(surfaces, choice.SurfaceProvenance)
	}
	var err error
	for _, surface := range surfaces {
		session, err = session.Visit(surface)
		if err != nil {
			return choice.Session{}, err
		}
	}
	return session, nil
}

func completeChoiceAndBundle(
	ctx context.Context,
	authorityRoot string,
	baselineCheckpoint Result,
	baseline compare.DivergentBaseline,
	original Result,
	reductionRun reducer.ReductionRun,
	confirmed Result,
) (
	choice.ChoicepointRecord,
	choice.DecisionRecord,
	promotion.Ruling,
	contractsource.PortableSource,
	emitmodel.ContractBundle,
	nodeemit.Residue,
	choiceAudit,
	*store.ObjectStore,
	error,
) {
	zero := func(err error) (
		choice.ChoicepointRecord, choice.DecisionRecord, promotion.Ruling, contractsource.PortableSource,
		emitmodel.ContractBundle, nodeemit.Residue, choiceAudit, *store.ObjectStore, error,
	) {
		return choice.ChoicepointRecord{}, choice.DecisionRecord{}, promotion.Ruling{}, contractsource.PortableSource{},
			emitmodel.ContractBundle{}, nodeemit.Residue{}, choiceAudit{}, nil, err
	}
	objectStore, err := store.OpenObjectStore(authorityRoot + "/object-store")
	if err != nil {
		return zero(err)
	}
	studyID, err := store.NewStudyID("u7 physical cli precedence contract")
	if err != nil {
		return zero(err)
	}
	head, err := objectStore.CreateStudy(ctx, studyID, confirmed.Plan)
	if err == nil {
		head, err = objectStore.AdvanceBaseline(ctx, head, baselineCheckpoint.OutcomeMap)
	}
	if err == nil {
		head, err = objectStore.AdvanceDivergence(ctx, head, baseline)
	}
	if err == nil {
		head, err = objectStore.AdvanceReduction(ctx, head, reductionRun)
	}
	if err != nil {
		return zero(err)
	}
	stored, err := promotion.PersistConfirmation(ctx, objectStore, head, confirmed.Confirmation.Draft())
	if err != nil {
		return zero(err)
	}
	originalArtifact, err := choice.NewCanonicalArtifact("CLIStimulus", original.Stimulus.Digest(), original.Stimulus.CanonicalBytes())
	if err != nil {
		return zero(err)
	}
	minimizedArtifact, err := choice.NewCanonicalArtifact("CLIStimulus", confirmed.Stimulus.Digest(), confirmed.Stimulus.CanonicalBytes())
	if err != nil {
		return zero(err)
	}
	reveals := make([]choice.CandidateReveal, len(confirmed.CandidateBindings))
	for index, binding := range confirmed.CandidateBindings {
		role, ok := confirmed.CandidateRoles[binding.Key()]
		if !ok {
			return zero(fmt.Errorf("CLI_STUDY_CHOICE_ROLE_REFUSED"))
		}
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: binding.Key(), DisplayRef: string(role),
			ProducerMetadata: "outer deterministic CLI precedence fixture",
		}
	}
	ready, err := promotion.Promote(ctx, objectStore, stored, promotion.ChoicepointRequest{
		Scenario: "Which exact correlated CLI precedence outcomes should become the accepted contract?",
		Plan:     confirmed.Plan, Envelope: confirmed.Envelope, CandidateBindings: confirmed.CandidateBindings,
		OriginalStimulus: originalArtifact, MinimizedStimulus: minimizedArtifact,
		CandidateReveals: reveals, EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil {
		return zero(err)
	}
	record := ready.Record()
	parsed, err := choice.ParseChoicepointRecord(record.CanonicalBytes())
	if err != nil || parsed.Digest() != record.Digest() {
		return zero(fmt.Errorf("CLI_STUDY_CHOICEPOINT_PARSE_REFUSED: %w", err))
	}

	blindSession, err := choice.NewSession(parsed)
	if err != nil || blindSession.State() != choice.SessionBlindOpen {
		return zero(fmt.Errorf("CLI_STUDY_BLIND_SESSION_REFUSED: %w", err))
	}
	cards := blindSession.BlindDTO().Cards()
	allowedAliases, err := exactCLIBlindAliases(cards)
	if err != nil {
		return zero(err)
	}
	selectedFields := cliObservedSelectedFields()
	standard := choice.RulingDraftInput{
		Action: choice.ActionAllowObserved, SelectedFields: selectedFields, AllowedAliases: allowedAliases,
	}
	ambiguousSession, err := choice.NewSession(parsed)
	if err == nil {
		ambiguousSession, err = visitChoiceSurfaces(ambiguousSession, false)
	}
	if err != nil {
		return zero(err)
	}
	ambiguous := choice.RulingDraftInput{
		Action:         choice.ActionAllowObserved,
		SelectedFields: []string{string(countercli.CLIFieldExitCode)}, AllowedAliases: allowedAliases,
	}
	refusedAmbiguity, ambiguityErr := ambiguousSession.Propose(ambiguous)
	if !choice.IsRefusal(ambiguityErr, choice.CodeAmbiguousScope) || refusedAmbiguity.State() != "" {
		return zero(fmt.Errorf("CLI_STUDY_AMBIGUOUS_SCOPE_NOT_REFUSED: %v", ambiguityErr))
	}

	session, err := visitChoiceSurfaces(blindSession, false)
	if err != nil {
		return zero(err)
	}
	session, err = session.Propose(standard)
	if err != nil || session.State() != choice.SessionProvisionalRecorded {
		return zero(fmt.Errorf("CLI_STUDY_BLIND_PROPOSAL_REFUSED: %w", err))
	}
	provisionalState := session.State()
	session, _, err = session.Reveal()
	if err != nil || session.State() != choice.SessionRevealed {
		return zero(fmt.Errorf("CLI_STUDY_REVEAL_REFUSED: %w", err))
	}
	revealedState := session.State()
	session, err = session.Visit(choice.SurfaceProvenance)
	if err == nil {
		session, err = session.Revise(standard, "")
	}
	if err != nil || session.State() != choice.SessionPostRevealRecorded {
		return zero(fmt.Errorf("CLI_STUDY_POST_REVEAL_REFUSED: %w", err))
	}
	finalizedSession, decision, err := session.Finalize(
		cliChoiceActor, cliChoiceAnnotation,
		[]domain.ReceiptReference{},
	)
	if err != nil || finalizedSession.State() != choice.SessionFinalized || !decision.Valid() || decision.EarlyReveal() ||
		decision.Action() != choice.ActionAllowObserved || !slices.Equal(decision.SelectedFields(), selectedFields) {
		return zero(fmt.Errorf("CLI_STUDY_DECISION_REFUSED: %w", err))
	}
	parsedDecision, err := choice.ParseDecisionRecord(decision.CanonicalBytes(), parsed)
	if err != nil || parsedDecision.Digest() != decision.Digest() {
		return zero(fmt.Errorf("CLI_STUDY_DECISION_PARSE_REFUSED: %w", err))
	}
	compiled, present := parsedDecision.CompilableRuling()
	if !present || len(compiled.AllowedTuples()) != 2 {
		return zero(fmt.Errorf("CLI_STUDY_CORRELATED_RULING_REFUSED"))
	}

	actionAudits := []choiceActionAudit{{action: choice.ActionAllowObserved, decision: parsedDecision}}
	customValue, err := choice.StringValue(cliCustomMode)
	if err != nil {
		return zero(err)
	}
	customTuple, err := choice.NewSelectedTuple(parsed,
		[]string{string(countercli.CLIFieldStdoutJSONMode)},
		[]choice.FieldValue{{FieldID: string(countercli.CLIFieldStdoutJSONMode), Value: customValue}},
	)
	if err != nil {
		return zero(err)
	}
	customDraft := choice.RulingDraftInput{
		Action:         choice.ActionCustomExpectation,
		SelectedFields: []string{string(countercli.CLIFieldStdoutJSONMode)}, AllowedAliases: []string{},
		CustomExpectation: &customTuple, CustomReviewer: cliCustomReviewer,
		CustomReviewEvidence: confirmed.Confirmation.Draft().Digest(),
	}
	customDecision, err := finalizeEarlyChoice(parsed, customDraft, cliCustomAnnotation)
	if err != nil {
		return zero(err)
	}
	customCompiled, ok := customDecision.CompilableRuling()
	customReview, reviewed := customCompiled.CustomReview()
	if !ok || !reviewed || customReview.Reviewer() != cliCustomReviewer ||
		customReview.EvidenceDigest() != confirmed.Confirmation.Draft().Digest() {
		return zero(fmt.Errorf("CLI_STUDY_CUSTOM_REVIEW_REFUSED"))
	}
	actionAudits = append(actionAudits, choiceActionAudit{
		action: choice.ActionCustomExpectation, decision: customDecision,
		customReviewer: cliCustomReviewer, customReviewEvidence: confirmed.Confirmation.Draft().Digest(),
	})
	for _, action := range []choice.Action{choice.ActionRejectAll, choice.ActionDefer, choice.ActionRefine} {
		draft := choice.RulingDraftInput{Action: action, SelectedFields: []string{}, AllowedAliases: []string{}}
		actionDecision, actionErr := finalizeEarlyChoice(parsed, draft, "Noncompilable control action "+string(action)+".")
		if actionErr != nil {
			return zero(actionErr)
		}
		if _, compilable := actionDecision.CompilableRuling(); compilable {
			return zero(fmt.Errorf("CLI_STUDY_NONCOMPILABLE_ACTION_REFUSED: %s", action))
		}
		actionFact, actionErr := exerciseNoncompilableChoiceBoundary(authorityRoot, action, actionDecision)
		if actionErr != nil {
			return zero(actionErr)
		}
		actionAudits = append(actionAudits, actionFact)
	}

	durableRuling, err := promotion.Finalize(ctx, objectStore, ready, parsedDecision)
	if err != nil || durableRuling.Record().Digest() != parsedDecision.Digest() ||
		!bytes.Equal(durableRuling.Record().CanonicalBytes(), parsedDecision.CanonicalBytes()) {
		return zero(fmt.Errorf("CLI_STUDY_DURABLE_RULING_REFUSED: %w", err))
	}
	preparation, err := promotion.PreparePortableRuling(ctx, objectStore, durableRuling)
	if err != nil || !preparation.Valid() || !slices.Equal(preparation.SelectedFields(), selectedFields) {
		return zero(fmt.Errorf("CLI_STUDY_RULING_PREPARATION_REFUSED: %w", err))
	}
	resolved, err := projectiontranslate.Resolve(confirmed.ProjectionDefinition.Binding())
	if err != nil {
		return zero(err)
	}
	projectionAuthority, err := climodel.ResolveCLIProjectionAuthority(
		confirmed.ProjectionDefinition.Digest(), confirmed.ProjectionDefinition.CanonicalBytes(),
		confirmed.ProjectionDefinition.Binding(),
	)
	if err != nil {
		return zero(err)
	}
	source, err := contractsource.NewCLISource(contractsource.CLIInput{
		Plan: confirmed.Plan, Stimulus: confirmed.Stimulus, Capture: confirmed.CapturePolicy,
		Profile: resolved.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		return zero(err)
	}
	parsedSource, err := contractsource.Parse(source.CanonicalBytes())
	if err != nil || parsedSource.Digest() != source.Digest() ||
		!bytes.Equal(parsedSource.CanonicalBytes(), source.CanonicalBytes()) {
		return zero(fmt.Errorf("CLI_STUDY_SOURCE_PARSE_REFUSED: %w", err))
	}
	source = parsedSource
	prepared, err := nodeemit.PrepareCompilation(ctx, objectStore, preparation, source)
	if err != nil || !prepared.Valid() || prepared.DecisionRecordDigest() != parsedDecision.Digest() ||
		prepared.ChoicepointDigest() != parsed.Digest() || prepared.SourceDigest() != source.Digest() {
		return zero(fmt.Errorf("CLI_STUDY_COMPILATION_PREPARATION_REFUSED: %w", err))
	}
	preparedBundle, err := nodeemit.CompilePrepared(prepared)
	if err != nil || !preparedBundle.Valid() || len(preparedBundle.Bundle().Files()) != 6 {
		return zero(fmt.Errorf("CLI_STUDY_BUNDLE_COMPILATION_REFUSED: %w", err))
	}
	bundle := preparedBundle.Bundle()
	parsedBundle, err := emitmodel.ParseContractBundle(bundle.CanonicalBytes(), bundle.Digest())
	if err != nil || parsedBundle.Digest() != bundle.Digest() ||
		!slices.Equal(parsedBundle.Predicate().SelectedFields(), selectedFields) ||
		len(parsedBundle.Predicate().AllowedTuples()) != 2 || !exactCLIBundleFiles(parsedBundle) {
		return zero(fmt.Errorf("CLI_STUDY_BUNDLE_PARSE_REFUSED: %w", err))
	}
	published, err := nodeemit.PublishPrepared(ctx, objectStore, preparedBundle)
	if err != nil || published.Disposition != nodeemit.PublicationCreated || !published.Residue.Valid() ||
		published.Residue.BundleDigest() != parsedBundle.Digest() {
		return zero(fmt.Errorf("CLI_STUDY_BUNDLE_PUBLICATION_REFUSED: %w", err))
	}
	reopenedResidue, err := nodeemit.ReopenResidue(ctx, objectStore, published.Residue)
	if err != nil || !reopenedResidue.Valid() || reopenedResidue.BundleDigest() != parsedBundle.Digest() ||
		reopenedResidue.DecisionRecordDigest() != parsedDecision.Digest() ||
		reopenedResidue.ChoicepointDigest() != parsed.Digest() ||
		!bytes.Equal(reopenedResidue.Bundle().CanonicalBytes(), parsedBundle.CanonicalBytes()) {
		return zero(fmt.Errorf("CLI_STUDY_BUNDLE_REOPEN_REFUSED: %w", err))
	}
	allowedTupleBytes := make([][]byte, len(parsedBundle.Predicate().AllowedTuples()))
	for index, tuple := range parsedBundle.Predicate().AllowedTuples() {
		allowedTupleBytes[index] = tuple.CanonicalBytes()
	}
	audit := choiceAudit{
		decisionDigest: parsedDecision.Digest(), earlyReveal: parsedDecision.EarlyReveal(),
		blindState: choice.SessionBlindOpen, provisionalState: provisionalState,
		revealedState: revealedState, finalState: finalizedSession.State(),
		ambiguityCode: choice.CodeAmbiguousScope, allowedTupleBytes: allowedTupleBytes,
		noncompilableCode: string(choice.CodeNoncompilablePredicate),
		destinationBefore: []string{}, destinationAfter: []string{}, actions: actionAudits,
	}
	if err := audit.validate(parsed, parsedDecision, parsedBundle); err != nil {
		return zero(err)
	}
	audit.validationProof, err = choiceAuditProof(parsed, parsedDecision, parsedBundle, audit)
	if err != nil {
		return zero(err)
	}
	return parsed, parsedDecision, durableRuling, source, parsedBundle, reopenedResidue, audit, objectStore, nil
}

// choiceAuditProof binds every retained choice/audit byte immediately after
// the stronger construction-time replay, including its live empty-destination
// checks. Public validation still independently replays the whole protocol.
func choiceAuditProof(
	record choice.ChoicepointRecord,
	decision choice.DecisionRecord,
	bundle emitmodel.ContractBundle,
	audit choiceAudit,
) (domain.Digest, error) {
	actions := make([]choiceAuditProofAction, len(audit.actions))
	for index, fact := range audit.actions {
		switch fact.action {
		case choice.ActionAllowObserved, choice.ActionCustomExpectation:
			if fact.preparationErr != nil {
				return "", fmt.Errorf("CLI_STUDY_CHOICE_CONSTRUCTION_PROOF_ERROR_REFUSED: %d", index)
			}
		case choice.ActionRejectAll, choice.ActionDefer, choice.ActionRefine:
			if !choice.IsRefusal(fact.preparationErr, choice.CodeNoncompilablePredicate) {
				return "", fmt.Errorf("CLI_STUDY_CHOICE_CONSTRUCTION_PROOF_ERROR_REFUSED: %d", index)
			}
		default:
			return "", fmt.Errorf("CLI_STUDY_CHOICE_CONSTRUCTION_PROOF_ACTION_REFUSED: %d", index)
		}
		preparationError := ""
		if fact.preparationErr != nil {
			preparationError = fact.preparationErr.Error()
		}
		actions[index] = choiceAuditProofAction{
			Action: string(fact.action), DecisionDigest: fact.decision.Digest().String(),
			DecisionCanonicalSHA:   sha256Hex(fact.decision.CanonicalBytes()),
			PreparationRefusalCode: fact.preparationRefusalCode, PreparationError: preparationError,
			DestinationPath:   fact.destinationPath,
			DestinationBefore: append([]string(nil), fact.destinationBefore...),
			DestinationAfter:  append([]string(nil), fact.destinationAfter...),
			CustomReviewer:    fact.customReviewer, CustomReviewEvidence: fact.customReviewEvidence.String(),
		}
	}
	allowedTupleSHA := make([]string, len(audit.allowedTupleBytes))
	for index, exact := range audit.allowedTupleBytes {
		allowedTupleSHA[index] = sha256Hex(exact)
	}
	exact, err := canon.CanonicalizeTyped(struct {
		SchemaVersion        string                   `json:"schema_version"`
		Kind                 string                   `json:"kind"`
		ChoicepointDigest    string                   `json:"choicepoint_digest"`
		ChoicepointCanonical string                   `json:"choicepoint_canonical_sha256"`
		DecisionDigest       string                   `json:"decision_digest"`
		DecisionCanonical    string                   `json:"decision_canonical_sha256"`
		BundleDigest         string                   `json:"bundle_digest"`
		BundleCanonical      string                   `json:"bundle_canonical_sha256"`
		AuditDecisionDigest  string                   `json:"audit_decision_digest"`
		EarlyReveal          bool                     `json:"early_reveal"`
		BlindState           string                   `json:"blind_state"`
		ProvisionalState     string                   `json:"provisional_state"`
		RevealedState        string                   `json:"revealed_state"`
		FinalState           string                   `json:"final_state"`
		AmbiguityCode        string                   `json:"ambiguity_code"`
		AllowedTupleSHA      []string                 `json:"allowed_tuple_sha256"`
		NoncompilableCode    string                   `json:"noncompilable_code"`
		DestinationBefore    []string                 `json:"destination_before"`
		DestinationAfter     []string                 `json:"destination_after"`
		Actions              []choiceAuditProofAction `json:"actions"`
	}{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIChoiceAuditConstructionProof",
		ChoicepointDigest: record.Digest().String(), ChoicepointCanonical: sha256Hex(record.CanonicalBytes()),
		DecisionDigest: decision.Digest().String(), DecisionCanonical: sha256Hex(decision.CanonicalBytes()),
		BundleDigest: bundle.Digest().String(), BundleCanonical: sha256Hex(bundle.CanonicalBytes()),
		AuditDecisionDigest: audit.decisionDigest.String(), EarlyReveal: audit.earlyReveal,
		BlindState: string(audit.blindState), ProvisionalState: string(audit.provisionalState),
		RevealedState: string(audit.revealedState), FinalState: string(audit.finalState),
		AmbiguityCode: string(audit.ambiguityCode), AllowedTupleSHA: allowedTupleSHA,
		NoncompilableCode: audit.noncompilableCode,
		DestinationBefore: append([]string(nil), audit.destinationBefore...),
		DestinationAfter:  append([]string(nil), audit.destinationAfter...), Actions: actions,
	})
	if err != nil {
		return "", err
	}
	digest, err := canon.DigestBytes("CLIChoiceAuditConstructionProof", exact)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

func validateChoiceAuditProof(
	audit choiceAudit,
	record choice.ChoicepointRecord,
	decision choice.DecisionRecord,
	bundle emitmodel.ContractBundle,
) error {
	if !audit.validationProof.Valid() {
		return fmt.Errorf("CLI_STUDY_CHOICE_CONSTRUCTION_PROOF_REFUSED")
	}
	rebuilt, err := choiceAuditProof(record, decision, bundle, audit)
	if err != nil || rebuilt != audit.validationProof {
		return fmt.Errorf("CLI_STUDY_CHOICE_CONSTRUCTION_PROOF_REFUSED: %v", err)
	}
	return nil
}

func exerciseNoncompilableChoiceBoundary(
	authorityRoot string,
	action choice.Action,
	decision choice.DecisionRecord,
) (choiceActionAudit, error) {
	if !decision.Valid() || decision.Action() != action ||
		(action != choice.ActionRejectAll && action != choice.ActionDefer && action != choice.ActionRefine) {
		return choiceActionAudit{}, fmt.Errorf("CLI_STUDY_NONCOMPILABLE_CONTROL_REFUSED: %s", action)
	}
	destination := filepath.Join(authorityRoot, "noncompilable-"+strings.ToLower(string(action)))
	if err := os.Mkdir(destination, 0o700); err != nil {
		return choiceActionAudit{}, err
	}
	before, err := snapshotChoiceDestination(destination)
	if err != nil {
		return choiceActionAudit{}, err
	}
	_, preparationErr := choice.InspectPortableRuling(decision)
	if !choice.IsRefusal(preparationErr, choice.CodeNoncompilablePredicate) {
		return choiceActionAudit{}, fmt.Errorf(
			"CLI_STUDY_NONCOMPILABLE_PREPARATION_NOT_REFUSED: %s: %v", action, preparationErr,
		)
	}
	after, err := snapshotChoiceDestination(destination)
	if err != nil || len(before) != 0 || len(after) != 0 || !slices.Equal(before, after) {
		return choiceActionAudit{}, fmt.Errorf(
			"CLI_STUDY_NONCOMPILABLE_DESTINATION_CHANGED: %s: %v", action, err,
		)
	}
	return choiceActionAudit{
		action: action, decision: decision,
		preparationRefusalCode: string(choice.CodeNoncompilablePredicate),
		preparationErr:         preparationErr, destinationPath: destination,
		destinationBefore: before, destinationAfter: after,
	}, nil
}

func snapshotChoiceDestination(path string) ([]string, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("CLI_STUDY_CHOICE_DESTINATION_REFUSED")
	}
	metadata, err := os.Lstat(path)
	if err != nil || !metadata.IsDir() || metadata.Mode()&os.ModeSymlink != 0 ||
		metadata.Mode().Perm() != 0o700 || metadata.Mode()&^os.ModePerm != os.ModeDir {
		return nil, fmt.Errorf("CLI_STUDY_CHOICE_DESTINATION_REFUSED: %v", err)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(entries))
	for index, entry := range entries {
		result[index] = entry.Name()
	}
	slices.Sort(result)
	return result, nil
}

func exactCLIBundleFiles(bundle emitmodel.ContractBundle) bool {
	want := []string{
		"README.md", "contract.test.mjs", "decision.json",
		"fixture.json", "harness.mjs", "manifest.json",
	}
	files := bundle.Files()
	if len(files) != len(want) {
		return false
	}
	for index, file := range files {
		if file.Path() != want[index] || file.Mode() != "100644" || file.ByteCount() < 1 ||
			!file.ByteSHA256().Valid() || len(file.Content()) != file.ByteCount() {
			return false
		}
	}
	return true
}

func finalizeEarlyChoice(record choice.ChoicepointRecord, draft choice.RulingDraftInput, annotation string) (choice.DecisionRecord, error) {
	session, err := choice.NewSession(record)
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	session, _, err = session.Reveal()
	if err == nil {
		session, err = visitChoiceSurfaces(session, true)
	}
	if err == nil {
		session, err = session.Revise(draft, "")
	}
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	_, decision, err := session.Finalize("u7-reference-operator", annotation, []domain.ReceiptReference{})
	if err != nil || !decision.Valid() || !decision.EarlyReveal() || decision.Action() != draft.Action {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_ACTION_DECISION_REFUSED: %s: %w", draft.Action, err)
	}
	parsed, err := choice.ParseDecisionRecord(decision.CanonicalBytes(), record)
	if err != nil || parsed.Digest() != decision.Digest() {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_ACTION_PARSE_REFUSED: %s: %w", draft.Action, err)
	}
	return parsed, nil
}

func exactCLIBlindAliases(cards []choice.BlindCard) ([]string, error) {
	if len(cards) != 3 {
		return nil, fmt.Errorf("CLI_STUDY_CHOICE_CARD_COUNT_REFUSED")
	}
	allowed := make([]string, 0, 2)
	seenAliases := make(map[string]struct{}, len(cards))
	seenPairs := make(map[string]struct{}, len(cards))
	for _, card := range cards {
		if card.Alias == "" || len(card.Fields) != 3 {
			return nil, fmt.Errorf("CLI_STUDY_CHOICE_CARD_SHAPE_REFUSED")
		}
		if _, duplicate := seenAliases[card.Alias]; duplicate {
			return nil, fmt.Errorf("CLI_STUDY_CHOICE_CARD_ALIAS_REFUSED")
		}
		seenAliases[card.Alias] = struct{}{}
		exit, mode, source := card.Fields[0], card.Fields[1], card.Fields[2]
		if exit.FieldID != string(countercli.CLIFieldExitCode) || exit.Tag != string(choice.ValueInteger) ||
			exit.Text != "0" || exit.Boolean || exit.CanonicalJSONBase64 != "" ||
			mode.FieldID != string(countercli.CLIFieldStdoutJSONMode) || mode.Tag != string(choice.ValueString) ||
			mode.Boolean || mode.CanonicalJSONBase64 != "" ||
			source.FieldID != string(countercli.CLIFieldStdoutJSONSource) || source.Tag != string(choice.ValueString) ||
			source.Boolean || source.CanonicalJSONBase64 != "" || mode.Text != source.Text {
			return nil, fmt.Errorf("CLI_STUDY_CHOICE_CARD_FIELDS_REFUSED")
		}
		switch mode.Text {
		case "config", "argv", "env":
		default:
			return nil, fmt.Errorf("CLI_STUDY_CHOICE_CARD_VALUE_REFUSED")
		}
		if _, duplicate := seenPairs[mode.Text]; duplicate {
			return nil, fmt.Errorf("CLI_STUDY_CHOICE_CARD_DUPLICATE_REFUSED")
		}
		seenPairs[mode.Text] = struct{}{}
		if mode.Text != "env" {
			allowed = append(allowed, card.Alias)
		}
	}
	if len(allowed) != 2 || len(seenPairs) != 3 {
		return nil, fmt.Errorf("CLI_STUDY_CHOICE_CARD_ROSTER_REFUSED")
	}
	return allowed, nil
}

func replayCLIChoiceDecision(record choice.ChoicepointRecord) (choice.DecisionRecord, error) {
	reopened, err := choice.ParseChoicepointRecord(record.CanonicalBytes())
	if err != nil || reopened.Digest() != record.Digest() ||
		!bytes.Equal(reopened.CanonicalBytes(), record.CanonicalBytes()) {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_CHOICEPOINT_REOPEN_REFUSED: %w", err)
	}
	session, err := choice.NewSession(reopened)
	if err != nil || session.State() != choice.SessionBlindOpen {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_CHOICE_REPLAY_OPEN_REFUSED: %w", err)
	}
	if _, provenanceErr := session.Visit(choice.SurfaceProvenance); !choice.IsRefusal(provenanceErr, choice.CodeInvalidSessionState) {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_CHOICE_PRE_REVEAL_PROVENANCE_REFUSED: %v", provenanceErr)
	}
	allowedAliases, err := exactCLIBlindAliases(session.BlindDTO().Cards())
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	session, err = visitChoiceSurfaces(session, false)
	if err != nil {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_CHOICE_REPLAY_VISIT_REFUSED: %w", err)
	}
	ambiguous := choice.RulingDraftInput{
		Action:         choice.ActionAllowObserved,
		SelectedFields: []string{string(countercli.CLIFieldExitCode)}, AllowedAliases: allowedAliases,
	}
	if refused, ambiguityErr := session.Propose(ambiguous); !choice.IsRefusal(ambiguityErr, choice.CodeAmbiguousScope) || refused.State() != "" {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_CHOICE_REPLAY_AMBIGUITY_REFUSED: %v", ambiguityErr)
	}
	standard := choice.RulingDraftInput{
		Action:         choice.ActionAllowObserved,
		SelectedFields: cliObservedSelectedFields(),
		AllowedAliases: allowedAliases,
	}
	session, err = session.Propose(standard)
	if err != nil || session.State() != choice.SessionProvisionalRecorded {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_CHOICE_REPLAY_PROPOSAL_REFUSED: %w", err)
	}
	session, _, err = session.Reveal()
	if err == nil {
		session, err = session.Visit(choice.SurfaceProvenance)
	}
	if err == nil {
		session, err = session.Revise(standard, "")
	}
	if err != nil {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_CHOICE_REPLAY_REVEAL_REFUSED: %w", err)
	}
	finalized, decision, err := session.Finalize(cliChoiceActor, cliChoiceAnnotation, []domain.ReceiptReference{})
	if err != nil || finalized.State() != choice.SessionFinalized || !exactCLIAllowDecision(decision) {
		return choice.DecisionRecord{}, fmt.Errorf("CLI_STUDY_CHOICE_REPLAY_FINALIZE_REFUSED: %w", err)
	}
	return decision, nil
}

func exactCLIChoiceOutcome(tuple choice.CompleteTuple) (string, bool) {
	fields := tuple.Fields
	if len(fields) != 3 || fields[0].FieldID != string(countercli.CLIFieldExitCode) ||
		fields[1].FieldID != string(countercli.CLIFieldStdoutJSONMode) ||
		fields[2].FieldID != string(countercli.CLIFieldStdoutJSONSource) ||
		fields[0].Value.Tag() != choice.ValueInteger || fields[0].Value.Text() != "0" ||
		fields[1].Value.Tag() != choice.ValueString || fields[2].Value.Tag() != choice.ValueString ||
		fields[1].Value.Text() != fields[2].Value.Text() {
		return "", false
	}
	return fields[1].Value.Text(), true
}

func exactCLIAllowDecision(decision choice.DecisionRecord) bool {
	wantFields := cliObservedSelectedFields()
	if !decision.Valid() || decision.Action() != choice.ActionAllowObserved || decision.EarlyReveal() ||
		decision.ChangedAfterReveal() || !slices.Equal(decision.SelectedFields(), wantFields) {
		return false
	}
	compiled, ok := decision.CompilableRuling()
	if !ok || compiled.Action() != choice.ActionAllowObserved {
		return false
	}
	selected := compiled.SelectedFields()
	if len(selected) != 3 || selected[0].String() != wantFields[0] || selected[1].String() != wantFields[1] ||
		selected[2].String() != wantFields[2] ||
		len(compiled.AllowedOutcomes()) != 2 || len(compiled.DisallowedOutcomes()) != 1 {
		return false
	}
	allowed := compiled.AllowedTuples()
	seen := make(map[string]struct{}, len(allowed))
	for _, tuple := range allowed {
		value, exact := exactCLIChoiceOutcome(tuple)
		if !exact || (value != "config" && value != "argv") {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	disallowed := compiled.DisallowedTuples()
	environment, exact := "", false
	if len(disallowed) == 1 {
		environment, exact = exactCLIChoiceOutcome(disallowed[0])
	}
	return len(seen) == 2 && exact && environment == "env"
}

func replayCLICustomDecision(record choice.ChoicepointRecord) (choice.DecisionRecord, error) {
	customValue, err := choice.StringValue(cliCustomMode)
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	customTuple, err := choice.NewSelectedTuple(
		record,
		[]string{string(countercli.CLIFieldStdoutJSONMode)},
		[]choice.FieldValue{{FieldID: string(countercli.CLIFieldStdoutJSONMode), Value: customValue}},
	)
	if err != nil {
		return choice.DecisionRecord{}, err
	}
	return finalizeEarlyChoice(record, choice.RulingDraftInput{
		Action:         choice.ActionCustomExpectation,
		SelectedFields: []string{string(countercli.CLIFieldStdoutJSONMode)}, AllowedAliases: []string{},
		CustomExpectation: &customTuple, CustomReviewer: cliCustomReviewer,
		CustomReviewEvidence: record.ConfirmationDigest(),
	}, cliCustomAnnotation)
}

func exactCLICustomDecision(record choice.ChoicepointRecord, decision choice.DecisionRecord) bool {
	expected, err := replayCLICustomDecision(record)
	if err != nil || expected.Digest() != decision.Digest() ||
		!bytes.Equal(expected.CanonicalBytes(), decision.CanonicalBytes()) ||
		!decision.EarlyReveal() || decision.Action() != choice.ActionCustomExpectation ||
		!slices.Equal(decision.SelectedFields(), []string{string(countercli.CLIFieldStdoutJSONMode)}) {
		return false
	}
	compiled, ok := decision.CompilableRuling()
	if !ok || compiled.Action() != choice.ActionCustomExpectation {
		return false
	}
	allowed := compiled.AllowedTuples()
	if len(allowed) != 1 || len(allowed[0].Fields) != 1 ||
		allowed[0].Fields[0].FieldID != string(countercli.CLIFieldStdoutJSONMode) ||
		allowed[0].Fields[0].Value.Tag() != choice.ValueString ||
		allowed[0].Fields[0].Value.Text() != cliCustomMode {
		return false
	}
	review, reviewed := compiled.CustomReview()
	return reviewed && review.Reviewer() == cliCustomReviewer &&
		review.EvidenceDigest() == record.ConfirmationDigest()
}

func exactCLIEmitOutcome(tuple emitmodel.ExactTuple) (string, bool) {
	fields := tuple.Fields()
	if len(fields) != 3 || fields[0].FieldID() != string(countercli.CLIFieldExitCode) ||
		fields[1].FieldID() != string(countercli.CLIFieldStdoutJSONMode) ||
		fields[2].FieldID() != string(countercli.CLIFieldStdoutJSONSource) {
		return "", false
	}
	exit, integer := fields[0].Value().PortableValue().IntegerText()
	mode, modeString := fields[1].Value().PortableValue().StringText()
	source, sourceString := fields[2].Value().PortableValue().StringText()
	if !integer || exit != "0" || !modeString || !sourceString || mode != source {
		return "", false
	}
	return mode, true
}

func exactCLIBundlePredicate(bundle emitmodel.ContractBundle) bool {
	if !bundle.Valid() || !slices.Equal(bundle.Predicate().SelectedFields(), cliObservedSelectedFields()) {
		return false
	}
	predicate := bundle.Predicate()
	allowed := predicate.AllowedTuples()
	if len(allowed) != 2 {
		return false
	}
	byValue := make(map[string]emitmodel.ExactTuple, 2)
	for _, tuple := range allowed {
		value, exact := exactCLIEmitOutcome(tuple)
		if !exact || (value != "config" && value != "argv") {
			return false
		}
		if _, duplicate := byValue[value]; duplicate {
			return false
		}
		byValue[value] = tuple
	}
	config, configPresent := byValue["config"]
	argv, argvPresent := byValue["argv"]
	if !configPresent || !argvPresent {
		return false
	}
	configFields, argvFields := config.Fields(), argv.Fields()
	for _, fields := range [][]emitmodel.ExactField{
		{configFields[0], configFields[1], argvFields[2]},
		{argvFields[0], argvFields[1], configFields[2]},
	} {
		mixed, err := emitmodel.NewExactTuple(fields)
		if err != nil {
			return false
		}
		matched, err := predicate.Matches(mixed)
		if err != nil || matched {
			return false
		}
	}
	return true
}

func (audit choiceAudit) validate(
	record choice.ChoicepointRecord,
	decision choice.DecisionRecord,
	bundles ...emitmodel.ContractBundle,
) error {
	if len(bundles) != 1 {
		return fmt.Errorf("CLI_STUDY_CHOICE_BUNDLE_AUTHORITY_REFUSED")
	}
	bundle := bundles[0]
	reopened, err := choice.ParseChoicepointRecord(record.CanonicalBytes())
	if err != nil || reopened.Digest() != record.Digest() ||
		!bytes.Equal(reopened.CanonicalBytes(), record.CanonicalBytes()) {
		return fmt.Errorf("CLI_STUDY_CHOICEPOINT_AUDIT_REOPEN_REFUSED: %w", err)
	}
	reopenedDecision, err := choice.ParseDecisionRecord(decision.CanonicalBytes(), reopened)
	if err != nil || reopenedDecision.Digest() != decision.Digest() ||
		!bytes.Equal(reopenedDecision.CanonicalBytes(), decision.CanonicalBytes()) {
		return fmt.Errorf("CLI_STUDY_DECISION_AUDIT_REOPEN_REFUSED: %w", err)
	}
	replayedDecision, err := replayCLIChoiceDecision(reopened)
	if err != nil || replayedDecision.Digest() != reopenedDecision.Digest() ||
		!bytes.Equal(replayedDecision.CanonicalBytes(), reopenedDecision.CanonicalBytes()) {
		return fmt.Errorf("CLI_STUDY_CHOICE_SEMANTIC_REPLAY_REFUSED: %w", err)
	}
	if !bundle.Valid() || !exactCLIAllowDecision(reopenedDecision) || !exactCLIBundlePredicate(bundle) ||
		audit.decisionDigest != reopenedDecision.Digest() || audit.earlyReveal ||
		audit.blindState != choice.SessionBlindOpen || audit.provisionalState != choice.SessionProvisionalRecorded ||
		audit.revealedState != choice.SessionRevealed || audit.finalState != choice.SessionFinalized ||
		audit.ambiguityCode != choice.CodeAmbiguousScope ||
		audit.noncompilableCode != string(choice.CodeNoncompilablePredicate) ||
		len(audit.destinationBefore) != 0 || len(audit.destinationAfter) != 0 ||
		!slices.Equal(audit.destinationBefore, audit.destinationAfter) {
		return fmt.Errorf("CLI_STUDY_CHOICE_AUDIT_REFUSED")
	}
	allowed := bundle.Predicate().AllowedTuples()
	if len(allowed) != 2 || len(audit.allowedTupleBytes) != len(allowed) {
		return fmt.Errorf("CLI_STUDY_CHOICE_TUPLE_COUNT_REFUSED")
	}
	for index, tuple := range allowed {
		if !tuple.Valid() || !bytes.Equal(tuple.CanonicalBytes(), audit.allowedTupleBytes[index]) {
			return fmt.Errorf("CLI_STUDY_CHOICE_TUPLE_REFUSED")
		}
	}
	wantActions := []choice.Action{
		choice.ActionAllowObserved, choice.ActionCustomExpectation,
		choice.ActionRejectAll, choice.ActionDefer, choice.ActionRefine,
	}
	if len(audit.actions) != len(wantActions) {
		return fmt.Errorf("CLI_STUDY_CHOICE_ACTION_COUNT_REFUSED")
	}
	for index, fact := range audit.actions {
		if fact.action != wantActions[index] || !fact.decision.Valid() || fact.decision.Action() != fact.action {
			return fmt.Errorf("CLI_STUDY_CHOICE_ACTION_REFUSED: %d", index)
		}
		parsed, err := choice.ParseDecisionRecord(fact.decision.CanonicalBytes(), reopened)
		if err != nil || parsed.Digest() != fact.decision.Digest() ||
			!bytes.Equal(parsed.CanonicalBytes(), fact.decision.CanonicalBytes()) {
			return fmt.Errorf("CLI_STUDY_CHOICE_ACTION_PARSE_REFUSED: %d", index)
		}
		switch fact.action {
		case choice.ActionAllowObserved:
			if parsed.Digest() != reopenedDecision.Digest() || parsed.Digest() != replayedDecision.Digest() ||
				!bytes.Equal(parsed.CanonicalBytes(), reopenedDecision.CanonicalBytes()) ||
				!exactCLIAllowDecision(parsed) {
				return fmt.Errorf("CLI_STUDY_CHOICE_ALLOW_JOIN_REFUSED")
			}
			if _, err := choice.InspectPortableRuling(parsed); err != nil {
				return fmt.Errorf("CLI_STUDY_CHOICE_ALLOW_INSPECTION_REFUSED: %v", err)
			}
		case choice.ActionCustomExpectation:
			if !exactCLICustomDecision(reopened, parsed) || fact.customReviewer != cliCustomReviewer ||
				fact.customReviewEvidence != reopened.ConfirmationDigest() {
				return fmt.Errorf("CLI_STUDY_CHOICE_CUSTOM_AUDIT_REFUSED")
			}
			if _, err := choice.InspectPortableRuling(parsed); err != nil {
				return fmt.Errorf("CLI_STUDY_CHOICE_CUSTOM_INSPECTION_REFUSED: %v", err)
			}
		default:
			expected, expectedErr := finalizeEarlyChoice(
				reopened,
				choice.RulingDraftInput{Action: fact.action, SelectedFields: []string{}, AllowedAliases: []string{}},
				"Noncompilable control action "+string(fact.action)+".",
			)
			_, replayErr := choice.InspectPortableRuling(parsed)
			_, compilable := parsed.CompilableRuling()
			liveDestination, destinationErr := snapshotChoiceDestination(fact.destinationPath)
			if expectedErr != nil || expected.Digest() != parsed.Digest() ||
				!bytes.Equal(expected.CanonicalBytes(), parsed.CanonicalBytes()) ||
				compilable || !parsed.EarlyReveal() ||
				fact.preparationRefusalCode != string(choice.CodeNoncompilablePredicate) ||
				!choice.IsRefusal(fact.preparationErr, choice.CodeNoncompilablePredicate) ||
				!choice.IsRefusal(replayErr, choice.CodeNoncompilablePredicate) ||
				fact.preparationErr.Error() != replayErr.Error() || destinationErr != nil ||
				len(fact.destinationBefore) != 0 || len(fact.destinationAfter) != 0 ||
				len(liveDestination) != 0 || !slices.Equal(fact.destinationBefore, fact.destinationAfter) ||
				!slices.Equal(fact.destinationAfter, liveDestination) {
				return fmt.Errorf("CLI_STUDY_CHOICE_NONCOMPILABLE_AUDIT_REFUSED: %s", fact.action)
			}
		}
	}
	return nil
}
