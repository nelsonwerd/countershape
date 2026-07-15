package model

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/domain"
)

// CLIExecutionBinding is the only authority passed to the impure world edge.
// It retains the exact WorldPlan and immutable stimulus; there is no public
// constructor from digests or caller-paired execution data.
type CLIExecutionBinding struct {
	digest         domain.Digest
	canonicalBytes []byte
	plan           domain.WorldPlan
	stimulus       CLIStimulus
	fixtureRecipe  CLIFixtureRecipe
	capturePolicy  CLICapturePolicy
	projection     CLIProjectionAuthority
}

type executionBindingIdentity struct {
	SchemaVersion           string `json:"schema_version"`
	Kind                    string `json:"kind"`
	WorldPlanDigest         string `json:"world_plan_digest"`
	StimulusDigest          string `json:"stimulus_digest"`
	ExecutionPayloadDigest  string `json:"execution_payload_digest"`
	FixtureRecipeDigest     string `json:"fixture_recipe_digest"`
	CapturePolicyDigest     string `json:"capture_policy_digest"`
	StdoutCaptureBytes      int64  `json:"stdout_capture_bytes"`
	StderrCaptureBytes      int64  `json:"stderr_capture_bytes"`
	ProjectionBindingDigest string `json:"projection_definition_digest"`
	AdapterProjectionDigest string `json:"cli_adapter_projection_definition_digest"`
}

// BindExecution validates that the plan's immutable start argv is exactly the
// fixed executable/base prefix. Per-stimulus argv exists only in the binding
// and is appended exactly once by CLIStimulus.LogicalArgv.
func BindExecution(
	plan domain.WorldPlan,
	stimulus CLIStimulus,
	capturePolicy CLICapturePolicy,
	projection CLIProjectionAuthority,
) (CLIExecutionBinding, error) {
	if !plan.Digest().Valid() || len(plan.CanonicalBytes()) == 0 || !stimulus.Valid() ||
		!capturePolicy.Valid() || !projection.Valid() {
		return CLIExecutionBinding{}, refuse(CodeInvalidBinding, "plan or stimulus authority is incomplete")
	}
	if plan.Adapter().Domain != domain.AdapterCLI || plan.ExecutionShape() != domain.OneCLIInvocation || len(plan.SetupArgv()) != 0 {
		return CLIExecutionBinding{}, refuse(CodeInvalidBinding, "execution binding requires a one-invocation CLI plan without setup")
	}
	if !equalStrings(plan.StartArgv(), stimulus.BaseLogicalArgv()) {
		return CLIExecutionBinding{}, refuse(CodeInvalidBinding, "plan start argv must exactly match the stimulus executable/base prefix")
	}
	fixtureRecipe, err := NewCLIFixtureRecipe()
	if err != nil {
		return CLIExecutionBinding{}, err
	}
	if plan.FixtureRecipeDigest() != fixtureRecipe.Digest() {
		return CLIExecutionBinding{}, refuse(CodeInvalidBinding, "plan fixture recipe does not name the exact U3 private-overlay capability")
	}
	budgets := plan.Budgets()
	// MUTATION_ANCHOR: execution-binding-must-resolve-capture-policy
	if plan.CapturePolicyDigest() != capturePolicy.Digest() ||
		budgets.StdoutBytes != capturePolicy.StdoutBytes() || budgets.StderrBytes != capturePolicy.StderrBytes() {
		return CLIExecutionBinding{}, refuse(CodeInvalidBinding, "plan capture policy does not resolve to the supplied exact channel caps")
	}
	// MUTATION_ANCHOR: execution-binding-must-resolve-adapter-projection
	if plan.ProjectionDefinitionDigest() != projection.Binding().Digest() {
		return CLIExecutionBinding{}, refuse(CodeInvalidBinding, "plan projection binding does not resolve to the supplied CLI definition")
	}
	identity := executionBindingIdentity{
		SchemaVersion:           domain.SchemaVersion,
		Kind:                    "CLIExecutionBinding",
		WorldPlanDigest:         plan.Digest().String(),
		StimulusDigest:          stimulus.Digest().String(),
		ExecutionPayloadDigest:  stimulus.ExecutionPayloadDigest().String(),
		FixtureRecipeDigest:     fixtureRecipe.Digest().String(),
		CapturePolicyDigest:     capturePolicy.Digest().String(),
		StdoutCaptureBytes:      capturePolicy.StdoutBytes(),
		StderrCaptureBytes:      capturePolicy.StderrBytes(),
		ProjectionBindingDigest: projection.Binding().Digest().String(),
		AdapterProjectionDigest: projection.Digest().String(),
	}
	digest, canonicalBytes, err := digestTyped("CLIExecutionBinding", identity)
	if err != nil {
		return CLIExecutionBinding{}, err
	}
	return CLIExecutionBinding{
		digest: digest, canonicalBytes: canonicalBytes, plan: plan, stimulus: stimulus,
		fixtureRecipe: fixtureRecipe, capturePolicy: capturePolicy, projection: projection,
	}, nil
}

func (b CLIExecutionBinding) Valid() bool {
	if !b.digest.Valid() || len(b.canonicalBytes) == 0 {
		return false
	}
	rebuilt, err := BindExecution(b.plan, b.stimulus, b.capturePolicy, b.projection)
	return err == nil && rebuilt.digest == b.digest && bytes.Equal(rebuilt.canonicalBytes, b.canonicalBytes)
}

func (b CLIExecutionBinding) Digest() domain.Digest         { return b.digest }
func (b CLIExecutionBinding) CanonicalBytes() []byte        { return append([]byte(nil), b.canonicalBytes...) }
func (b CLIExecutionBinding) PlanDigest() domain.Digest     { return b.plan.Digest() }
func (b CLIExecutionBinding) Plan() domain.WorldPlan        { return b.plan }
func (b CLIExecutionBinding) Stimulus() CLIStimulus         { return b.stimulus }
func (b CLIExecutionBinding) StimulusDigest() domain.Digest { return b.stimulus.Digest() }
func (b CLIExecutionBinding) ExecutionPayloadDigest() domain.Digest {
	return b.stimulus.ExecutionPayloadDigest()
}
func (b CLIExecutionBinding) FixtureRecipe() CLIFixtureRecipe    { return b.fixtureRecipe }
func (b CLIExecutionBinding) FixtureRecipeDigest() domain.Digest { return b.fixtureRecipe.Digest() }
func (b CLIExecutionBinding) CapturePolicy() CLICapturePolicy    { return b.capturePolicy }
func (b CLIExecutionBinding) CapturePolicyDigest() domain.Digest { return b.capturePolicy.Digest() }
func (b CLIExecutionBinding) AdapterProjectionDefinitionDigest() domain.Digest {
	return b.projection.Digest()
}
func (b CLIExecutionBinding) ProjectionDefinitionBinding() domain.ProjectionDefinitionBinding {
	return b.projection.Binding()
}
func (b CLIExecutionBinding) LogicalArgv() []string                { return b.stimulus.LogicalArgv() }
func (b CLIExecutionBinding) Stdin() CLIStdin                      { return b.stimulus.Stdin() }
func (b CLIExecutionBinding) Environment() []CLIEnvironmentBinding { return b.stimulus.Environment() }
func (b CLIExecutionBinding) Fixtures() []CLIFixtureFile           { return b.stimulus.Fixtures() }
func (b CLIExecutionBinding) CWDPolicy() CWDPolicy                 { return b.stimulus.CWDPolicy() }

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
