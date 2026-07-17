// Package node joins current ruling authority to an exact PortableSource and
// prepares the private pure input consumed by the later Node compiler.
package node

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/internal/compilation"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	"github.com/nelsonwerd/countershape/internal/store"
)

const (
	CodeInvalidPreparation          = "INVALID_COMPILATION_PREPARATION"
	CodeSourceRulingMismatch        = "SOURCE_RULING_MISMATCH"
	CodeConfirmationBindingMismatch = "CONFIRMATION_BINDING_MISMATCH"
	CodeProjectionMismatch          = "PROJECTION_RETRANSLATION_MISMATCH"
	CodeRulingPartitionMismatch     = "RULING_PARTITION_MISMATCH"
	CodeSanitizedInputInvalid       = "SANITIZED_COMPILATION_INPUT_INVALID"
)

type Error struct {
	Code   string
	Detail string
	Cause  error
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return e.Code + ": " + e.Detail
}

func (e *Error) Unwrap() error { return e.Cause }

func IsCode(err error, code string) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func refuse(code, detail string, cause error) error {
	return &Error{Code: code, Detail: detail, Cause: cause}
}

type preparedCompilationSeal struct{}

var preparedAuthority = &preparedCompilationSeal{}

// PreparedCompilation privately retains the original current-ruling
// preparation for P07B-B and a separate sanitized compiler input for A2.2.
// Its public views are inert summaries; Valid proves construction integrity,
// never continuing store currentness.
type PreparedCompilation struct {
	preparation promotion.PortableRulingPreparation
	input       compilation.Input
	seal        *preparedCompilationSeal
}

func PrepareCompilation(
	ctx context.Context,
	objectStore *store.ObjectStore,
	preparation promotion.PortableRulingPreparation,
	source contractsource.PortableSource,
) (PreparedCompilation, error) {
	if ctx == nil || objectStore == nil {
		return PreparedCompilation{}, refuse(CodeInvalidPreparation, "context and object store are required", nil)
	}
	snapshot, err := promotion.OpenPortableCompilationSnapshot(ctx, objectStore, preparation)
	if err != nil {
		return PreparedCompilation{}, err
	}
	if !snapshot.Valid() {
		return PreparedCompilation{}, refuse(CodeInvalidPreparation, "current ruling snapshot is invalid", nil)
	}

	exactSource, err := contractsource.Parse(source.CanonicalBytes())
	if err != nil || exactSource.Digest() != source.Digest() ||
		!bytes.Equal(exactSource.CanonicalBytes(), source.CanonicalBytes()) {
		return PreparedCompilation{}, refuse(CodeSourceRulingMismatch, "PortableSource did not reparse exactly", err)
	}

	decision := snapshot.DecisionRecord()
	choicepoint := snapshot.ChoicepointRecord()
	confirmationRecord := snapshot.ConfirmationRecord()
	if err := requireSourceChoicepointJoin(exactSource, choicepoint); err != nil {
		return PreparedCompilation{}, err
	}
	bindings := confirmationRecord.ExecutionBindingDigests()
	if len(bindings) == 0 {
		return PreparedCompilation{}, refuse(CodeConfirmationBindingMismatch, "confirmation retained no execution-binding roster", nil)
	}
	for index, binding := range bindings {
		if binding != exactSource.ExecutionBindingDigest() {
			return PreparedCompilation{}, refuse(
				CodeConfirmationBindingMismatch,
				fmt.Sprintf("confirmation execution binding %d differs from the exact source", index), nil,
			)
		}
	}

	confirmed, err := compare.RequireConfirmedOutcomeMap(confirmationRecord.ConfirmedMap())
	if err != nil {
		return PreparedCompilation{}, refuse(CodeProjectionMismatch, "confirmation map is not one strict confirmed roster", err)
	}
	proofRecords := confirmationRecord.ProjectionProofs()
	proofs := make([]projectiontranslate.ProjectionProof, len(proofRecords))
	for index, proof := range proofRecords {
		if !proof.Valid() {
			return PreparedCompilation{}, refuse(CodeProjectionMismatch, "confirmation projection proof is invalid", nil)
		}
		proofs[index] = projectiontranslate.ProjectionProof{
			CandidateExecutionKey: proof.CandidateExecutionKey(),
			CanonicalProjection:   proof.CanonicalProjection(),
		}
	}
	translations, err := projectiontranslate.TranslateConfirmed(
		exactSource.ProjectionBinding(), confirmed.ProjectionRoster(), proofs,
	) // P07B_A2_EXPLICIT_RETRANSLATION_ANCHOR
	if err != nil || !translations.Valid() {
		return PreparedCompilation{}, refuse(CodeProjectionMismatch, "proof-first projection retranslation failed", err)
	}
	translatedProfile := translations.Profile()
	sourceProfile := exactSource.Profile()
	preparedProfile := snapshot.PortableProfile()
	if !preparedProfile.Valid() || snapshot.ProfileDigest() != translatedProfile.Digest() ||
		preparedProfile.Digest() != translatedProfile.Digest() || sourceProfile.Digest() != translatedProfile.Digest() ||
		!bytes.Equal(preparedProfile.CanonicalBytes(), translatedProfile.CanonicalBytes()) ||
		!bytes.Equal(sourceProfile.CanonicalBytes(), translatedProfile.CanonicalBytes()) {
		return PreparedCompilation{}, refuse(CodeProjectionMismatch, "portable projection profile bytes or digest differ", nil)
	}

	predicate, err := revalidatePartition(decision, translations, exactSource)
	if err != nil {
		return PreparedCompilation{}, err
	}
	declaredProfile, err := model.NewSourceProfile(exactSource)
	if err != nil || !declaredProfile.ValidFor(exactSource) {
		return PreparedCompilation{}, refuse(CodeSanitizedInputInvalid, "declared emitter profile is invalid", err)
	}
	action, err := compilationAction(decision.Action())
	if err != nil {
		return PreparedCompilation{}, err
	}
	input, err := compilation.New(
		decision.Digest(), choicepoint.Digest(), action, exactSource, declaredProfile, predicate,
	)
	if err != nil || !input.Valid() {
		return PreparedCompilation{}, refuse(CodeSanitizedInputInvalid, "sanitized compiler input failed reconstruction", err)
	}
	prepared := PreparedCompilation{preparation: preparation, input: input, seal: preparedAuthority}
	if !prepared.Valid() {
		return PreparedCompilation{}, refuse(CodeSanitizedInputInvalid, "prepared compilation failed sealed validation", nil)
	}
	return prepared, nil
}

func requireSourceChoicepointJoin(source contractsource.PortableSource, record choice.ChoicepointRecord) error {
	plan := record.WorldPlan()
	sourcePlan := source.Plan()
	if !plan.Digest().Valid() || plan.Digest() != sourcePlan.Digest() ||
		!bytes.Equal(plan.CanonicalBytes(), sourcePlan.CanonicalBytes()) ||
		plan.Adapter().Domain != source.Adapter() ||
		plan.ProjectionDefinitionBinding().Digest() != source.ProjectionBinding().Digest() ||
		!bytes.Equal(plan.ProjectionDefinitionBinding().CanonicalBytes(), source.ProjectionBinding().CanonicalBytes()) {
		return refuse(CodeSourceRulingMismatch, "source and Choicepoint WorldPlan authority differ", nil)
	}
	stimulus := record.MinimizedStimulus()
	if stimulus.Kind() != source.StimulusKind() || stimulus.Digest() != source.StimulusDigest() ||
		!bytes.Equal(stimulus.CanonicalBytes(), source.StimulusCanonicalBytes()) {
		return refuse(CodeSourceRulingMismatch, "source and Choicepoint minimized stimulus differ", nil)
	}
	return nil
}

type translatedTuple struct {
	key   string
	tuple model.ExactTuple
}

func revalidatePartition(
	decision choice.DecisionRecord,
	translations projectiontranslate.ConfirmedTranslations,
	source contractsource.PortableSource,
) (model.Predicate, error) {
	compilable, ok := decision.CompilableRuling()
	if !ok || (decision.Action() != choice.ActionAllowObserved && decision.Action() != choice.ActionCustomExpectation) ||
		compilable.Action() != decision.Action() {
		return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "decision action is not one compilable v1 action", nil)
	}
	selectedIDs := compilable.SelectedFields()
	selected := make([]string, len(selectedIDs))
	for index, field := range selectedIDs {
		selected[index] = field.String()
	}
	if len(selected) == 0 || !equalStrings(selected, decision.SelectedFields()) {
		return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "decision selected fields differ from its compilable ruling", nil)
	}

	translated := make(map[string]model.ExactTuple, len(translations.Outcomes()))
	for _, outcome := range translations.Outcomes() {
		tuple, err := translatedSelectedTuple(outcome.Tuple(), selected)
		if err != nil {
			return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "translated tuple could not be projected exactly", err)
		}
		key := outcomeIdentity(outcome.CandidateExecutionKey(), outcome.ProjectionFingerprint())
		if _, duplicate := translated[key]; duplicate {
			return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "translated outcome identity repeats", nil)
		}
		translated[key] = tuple
	}

	allowedRefs, err := referenceTupleSet(compilable.AllowedOutcomes(), translated)
	if err != nil {
		return model.Predicate{}, err
	}
	disallowedRefs, err := referenceTupleSet(compilable.DisallowedOutcomes(), translated)
	if err != nil {
		return model.Predicate{}, err
	}
	if !exactReferencePartition(compilable.AllowedOutcomes(), compilable.DisallowedOutcomes(), translated) {
		return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "allowed/disallowed references do not partition translated outcomes", nil)
	}
	allowedRuling, err := choiceTupleSet(compilable.AllowedTuples(), selected)
	if err != nil {
		return model.Predicate{}, err
	}
	disallowedRuling, err := choiceTupleSet(compilable.DisallowedTuples(), selected)
	if err != nil {
		return model.Predicate{}, err
	}
	if decision.Action() == choice.ActionAllowObserved {
		if len(compilable.AllowedOutcomes()) == 0 || !sameTupleSet(allowedRefs, allowedRuling) ||
			!sameTupleSet(disallowedRefs, disallowedRuling) {
			return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "observed tuple sets differ from their exact reference partition", nil)
		}
	} else {
		_, reviewed := compilable.CustomReview()
		if !reviewed || len(compilable.AllowedOutcomes()) != 0 || len(allowedRuling) != 1 ||
			len(compilable.DisallowedOutcomes()) != len(translated) ||
			!sameTupleSet(disallowedRefs, disallowedRuling) {
			return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "custom expectation partition or review differs", nil)
		}
		for _, observed := range translated {
			if bytes.Equal(observed.CanonicalBytes(), allowedRuling[0].CanonicalBytes()) {
				return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "custom expectation matches a confirmed tuple", nil)
			}
		}
	}
	for _, allowed := range allowedRuling {
		for _, disallowed := range disallowedRuling {
			if bytes.Equal(allowed.CanonicalBytes(), disallowed.CanonicalBytes()) {
				return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "allowed and disallowed tuples are not separated", nil)
			}
		}
	}
	predicate, err := model.NewPredicate(source.StimulusDigest(), translations.Profile(), selected, allowedRuling)
	if err != nil || !predicate.ValidFor(source.Profile(), source.StimulusDigest()) {
		return model.Predicate{}, refuse(CodeRulingPartitionMismatch, "canonical selected-field predicate is invalid", err)
	}
	return predicate, nil
}

func translatedSelectedTuple(tuple projectiontranslate.Tuple, selected []string) (model.ExactTuple, error) {
	fields := make([]model.ExactField, len(selected))
	for index, fieldID := range selected {
		value, present := tuple.Value(fieldID)
		if !present {
			return model.ExactTuple{}, refuse(CodeRulingPartitionMismatch, "translated tuple omits a selected field", nil)
		}
		exact, err := model.NewExactValue(value)
		if err != nil {
			return model.ExactTuple{}, err
		}
		fields[index], err = model.NewExactField(fieldID, exact)
		if err != nil {
			return model.ExactTuple{}, err
		}
	}
	return model.NewExactTuple(fields)
}

func choiceTupleSet(tuples []choice.CompleteTuple, selected []string) ([]model.ExactTuple, error) {
	result := make([]model.ExactTuple, len(tuples))
	for tupleIndex, tuple := range tuples {
		if len(tuple.Fields) != len(selected) {
			return nil, refuse(CodeRulingPartitionMismatch, "ruling tuple does not cover selected fields", nil)
		}
		fields := make([]model.ExactField, len(tuple.Fields))
		for fieldIndex, field := range tuple.Fields {
			if field.FieldID != selected[fieldIndex] {
				return nil, refuse(CodeRulingPartitionMismatch, "ruling tuple field order differs", nil)
			}
			value, err := portableFromChoice(field.Value)
			if err != nil {
				return nil, err
			}
			exact, err := model.NewExactValue(value)
			if err != nil {
				return nil, err
			}
			fields[fieldIndex], err = model.NewExactField(field.FieldID, exact)
			if err != nil {
				return nil, err
			}
		}
		var err error
		result[tupleIndex], err = model.NewExactTuple(fields)
		if err != nil {
			return nil, err
		}
	}
	return dedupeTuples(result), nil
}

func portableFromChoice(value choice.ExactValue) (portablevalue.Value, error) {
	switch value.Tag() {
	case choice.ValueMissing:
		return portablevalue.Missing(), nil
	case choice.ValueNull:
		return portablevalue.Null(), nil
	case choice.ValueBoolean:
		return portablevalue.Boolean(value.Boolean()), nil
	case choice.ValueInteger:
		return portablevalue.Integer(value.Text())
	case choice.ValueString:
		return portablevalue.String(value.Text())
	case choice.ValueBytes:
		return portablevalue.Bytes(value.Bytes())
	case choice.ValueOrderedStringList:
		members, ok := value.OrderedStrings()
		if !ok {
			break
		}
		return portablevalue.OrderedStringList(members)
	case choice.ValueCanonicalJSON:
		return portablevalue.CanonicalJSON(value.CanonicalBytes())
	}
	return portablevalue.Value{}, refuse(CodeRulingPartitionMismatch, "Choice exact value could not be reconstructed", nil)
}

func referenceTupleSet(
	refs []choice.ConfirmedOutcomeRef,
	translated map[string]model.ExactTuple,
) ([]model.ExactTuple, error) {
	result := make([]model.ExactTuple, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		key := outcomeIdentity(ref.CandidateExecutionKey(), ref.ProjectionFingerprint())
		tuple, present := translated[key]
		if !present {
			return nil, refuse(CodeRulingPartitionMismatch, "ruling reference is outside the translated roster", nil)
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, refuse(CodeRulingPartitionMismatch, "ruling reference repeats", nil)
		}
		seen[key] = struct{}{}
		result = append(result, tuple)
	}
	return dedupeTuples(result), nil
}

func exactReferencePartition(
	allowed, disallowed []choice.ConfirmedOutcomeRef,
	translated map[string]model.ExactTuple,
) bool {
	seen := make(map[string]struct{}, len(allowed)+len(disallowed))
	for _, refs := range [][]choice.ConfirmedOutcomeRef{allowed, disallowed} {
		for _, ref := range refs {
			key := outcomeIdentity(ref.CandidateExecutionKey(), ref.ProjectionFingerprint())
			if _, present := translated[key]; !present {
				return false
			}
			if _, duplicate := seen[key]; duplicate {
				return false
			}
			seen[key] = struct{}{}
		}
	}
	return len(seen) == len(translated)
}

func outcomeIdentity(candidate domain.CandidateExecutionKey, fingerprint domain.ProjectionFingerprint) string {
	return candidate.String() + "\x00" + fingerprint.String()
}

func dedupeTuples(input []model.ExactTuple) []model.ExactTuple {
	seen := make(map[string]struct{}, len(input))
	result := make([]model.ExactTuple, 0, len(input))
	for _, tuple := range input {
		key := string(tuple.CanonicalBytes())
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tuple)
	}
	return result
}

func sameTupleSet(left, right []model.ExactTuple) bool {
	leftSet := make(map[string]struct{}, len(left))
	rightSet := make(map[string]struct{}, len(right))
	for _, tuple := range left {
		leftSet[string(tuple.CanonicalBytes())] = struct{}{}
	}
	for _, tuple := range right {
		rightSet[string(tuple.CanonicalBytes())] = struct{}{}
	}
	if len(leftSet) != len(rightSet) {
		return false
	}
	for key := range leftSet {
		if _, present := rightSet[key]; !present {
			return false
		}
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func compilationAction(action choice.Action) (compilation.DecisionAction, error) {
	switch action {
	case choice.ActionAllowObserved:
		return compilation.ActionAllowObserved, nil
	case choice.ActionCustomExpectation:
		return compilation.ActionCustomExpectation, nil
	default:
		return "", refuse(CodeRulingPartitionMismatch, "decision action is not compilable", nil)
	}
}

func (p PreparedCompilation) Valid() bool {
	return p.seal == preparedAuthority && p.preparation.Valid() && p.input.Valid() &&
		p.preparation.DecisionDigest() == p.input.DecisionRecordDigest() &&
		p.preparation.ProfileDigest() == p.input.Predicate().PortableProfileDigest()
}

func (p PreparedCompilation) Digest() domain.Digest { return p.input.Digest() }
func (p PreparedCompilation) DecisionRecordDigest() domain.Digest {
	return p.input.DecisionRecordDigest()
}
func (p PreparedCompilation) ChoicepointDigest() domain.Digest { return p.input.ChoicepointDigest() }
func (p PreparedCompilation) SourceDigest() domain.Digest      { return p.input.Source().Digest() }
func (p PreparedCompilation) SourceProfileDigest() domain.Digest {
	return p.input.SourceProfile().Digest()
}
func (p PreparedCompilation) Action() string           { return string(p.input.Action()) }
func (p PreparedCompilation) SelectedFields() []string { return p.input.Predicate().SelectedFields() }
func (p PreparedCompilation) AllowedTupleCanonicalBytes() [][]byte {
	tuples := p.input.Predicate().AllowedTuples()
	result := make([][]byte, len(tuples))
	for index, tuple := range tuples {
		result[index] = tuple.CanonicalBytes()
	}
	return result
}
