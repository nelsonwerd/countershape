package model

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
)

type ContractExecution struct {
	targetDigest domain.Digest
	runDigest    domain.Digest
	result       ExecutionResult
	digest       domain.Digest
	canonical    []byte
}

func ClassifierProfileDigest() domain.Digest {
	profile := map[string]any{
		"schema_version": domain.SchemaVersion,
		"kind":           "ContractExecutionClassifierProfile",
		"profile":        ClassifierProfileV1,
	}
	exact, err := canon.CanonicalizeTyped(profile)
	if err != nil {
		return ""
	}
	digest, err := canon.DigestBytes("ContractExecutionClassifierProfile", exact)
	if err != nil {
		return ""
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return ""
	}
	return parsed
}

// DeriveContractExecution is the only ContractExecution constructor. It
// accepts no requested result, tuple, or reason. The exact target/run/bundle
// graph owns every classification input.
func DeriveContractExecution(
	bundle emitmodel.ContractBundle,
	target ContractExecutionTarget,
	run FinalizedContractRun,
) (ContractExecution, error) {
	if !bundle.Valid() || !target.Valid() || !run.Valid() || !run.MatchesTarget(target) {
		return ContractExecution{}, refuse(CodeClassificationInput, "bundle, target, or run is invalid or mismatched", nil)
	}
	if bundle.Digest() != target.ContractBundleDigest() {
		return ContractExecution{}, refuse(CodeClassificationInput, "bundle does not match the target binding", nil)
	}
	result := ResultIneligible
	if run.Disposition() == DispositionEligibleClean {
		tuple, ok := run.witness.observation.Tuple()
		if !ok {
			return ContractExecution{}, refuse(CodeClassificationInput, "eligible run has no projected tuple", nil)
		}
		matches, err := bundle.Predicate().Matches(tuple)
		if err != nil {
			return ContractExecution{}, refuse(CodeClassificationInput, "projected tuple is incompatible with the bundle predicate", err)
		}
		if matches {
			result = ResultConforms
		} else {
			result = ResultContradicts
		}
	}
	wire := executionWire(target.Digest(), run.Digest(), result)
	exact, digest, err := canonicalObject(ExecutionKind, wire)
	if err != nil {
		return ContractExecution{}, refuse(CodeInvalidExecution, "execution body could not be built", err)
	}
	return ContractExecution{
		targetDigest: target.Digest(), runDigest: run.Digest(), result: result,
		digest: digest, canonical: exact,
	}, nil
}

func executionWire(targetDigest, runDigest domain.Digest, result ExecutionResult) map[string]any {
	return map[string]any{
		"schema_version":                   domain.SchemaVersion,
		"kind":                             ExecutionKind,
		"execution_version":                ExecutionVersionV1,
		"publication_scope":                ExecutionPublicationScopeV1,
		"classifier_profile":               ClassifierProfileV1,
		"contract_execution_target_digest": targetDigest.String(),
		"finalized_contract_run_digest":    runDigest.String(),
		"result":                           string(result),
	}
}

func ParseContractExecution(
	exact []byte,
	expected domain.Digest,
	bundle emitmodel.ContractBundle,
	target ContractExecutionTarget,
	run FinalizedContractRun,
) (ContractExecution, error) {
	root, err := parseExactObject(exact, ExecutionKind, expected, CodeInvalidExecution, CodeExecutionDigest)
	if err != nil {
		return ContractExecution{}, err
	}
	if err := requireRoster(root, "schema_version", "kind", "execution_version", "publication_scope",
		"classifier_profile", "contract_execution_target_digest", "finalized_contract_run_digest", "result"); err != nil {
		return ContractExecution{}, refuse(CodeInvalidExecution, "execution root roster differs", err)
	}
	schema, _ := textMember(root, "schema_version")
	kind, _ := textMember(root, "kind")
	version, _ := textMember(root, "execution_version")
	scope, _ := textMember(root, "publication_scope")
	profile, _ := textMember(root, "classifier_profile")
	if schema != domain.SchemaVersion || kind != ExecutionKind || version != ExecutionVersionV1 ||
		scope != ExecutionPublicationScopeV1 || profile != ClassifierProfileV1 {
		return ContractExecution{}, refuse(CodeInvalidExecution, "execution constants differ", nil)
	}
	targetDigest, targetErr := digestMember(root, "contract_execution_target_digest")
	runDigest, runErr := digestMember(root, "finalized_contract_run_digest")
	resultText, resultErr := textMember(root, "result")
	result := ExecutionResult(resultText)
	if targetErr != nil || runErr != nil || resultErr != nil || targetDigest != target.Digest() || runDigest != run.Digest() ||
		(result != ResultConforms && result != ResultContradicts && result != ResultIneligible) {
		return ContractExecution{}, refuse(CodeInvalidExecution, "execution references or result are invalid", nil)
	}
	rebuilt, err := DeriveContractExecution(bundle, target, run)
	if err != nil || rebuilt.result != result || rebuilt.digest != expected || !bytes.Equal(rebuilt.canonical, exact) {
		return ContractExecution{}, refuse(CodeInvalidExecution, "execution differs from the exact derived result", err)
	}
	return rebuilt, nil
}

func (execution ContractExecution) Valid() bool {
	if !execution.targetDigest.Valid() || !execution.runDigest.Valid() || !execution.digest.Valid() ||
		(execution.result != ResultConforms && execution.result != ResultContradicts && execution.result != ResultIneligible) {
		return false
	}
	exact, digest, err := canonicalObject(ExecutionKind, executionWire(execution.targetDigest, execution.runDigest, execution.result))
	return err == nil && digest == execution.digest && bytes.Equal(exact, execution.canonical)
}

func (execution ContractExecution) Digest() domain.Digest             { return execution.digest }
func (execution ContractExecution) CanonicalBytes() []byte            { return cloneBytes(execution.canonical) }
func (execution ContractExecution) TargetDigest() domain.Digest       { return execution.targetDigest }
func (execution ContractExecution) FinalizedRunDigest() domain.Digest { return execution.runDigest }
func (execution ContractExecution) Result() ExecutionResult           { return execution.result }

func (execution ContractExecution) Equal(other ContractExecution) bool {
	return execution.Valid() && other.Valid() && execution.digest == other.digest && bytes.Equal(execution.canonical, other.canonical)
}
