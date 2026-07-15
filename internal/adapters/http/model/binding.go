package model

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const HTTPExecutionAuthorityV1 = "U4_OPAQUE_HTTP_EXECUTION_BINDING_V1"

type HTTPExecutionBinding struct {
	digest         domain.Digest
	canonicalBytes []byte
	plan           domain.WorldPlan
	stimulus       HTTPStimulus
	start          HTTPStartSpec
	fixtureRecipe  HTTPFixtureRecipe
	capturePolicy  HTTPCapturePolicy
	readiness      HTTPReadinessContract
	projection     HTTPProjectionAuthority
}

type executionBindingIdentity struct {
	SchemaVersion           string `json:"schema_version"`
	Kind                    string `json:"kind"`
	Authority               string `json:"authority"`
	WorldPlanDigest         string `json:"world_plan_digest"`
	StimulusDigest          string `json:"stimulus_digest"`
	ExecutionPayloadDigest  string `json:"execution_payload_digest"`
	StartSpecDigest         string `json:"start_spec_digest"`
	FixtureRecipeDigest     string `json:"fixture_recipe_digest"`
	CapturePolicyDigest     string `json:"capture_policy_digest"`
	HTTPBodyCaptureBytes    int64  `json:"http_body_capture_bytes"`
	ReadinessContractDigest string `json:"readiness_contract_digest"`
	ProjectionBindingDigest string `json:"projection_definition_digest"`
	AdapterProjectionDigest string `json:"http_adapter_projection_definition_digest"`
}

func BindExecution(
	plan domain.WorldPlan,
	stimulus HTTPStimulus,
	start HTTPStartSpec,
	capturePolicy HTTPCapturePolicy,
	readiness HTTPReadinessContract,
	projection HTTPProjectionAuthority,
) (HTTPExecutionBinding, error) {
	if !plan.Digest().Valid() || len(plan.CanonicalBytes()) == 0 || !stimulus.Valid() || !start.Valid() ||
		!capturePolicy.Valid() || !readiness.Valid() || !projection.Valid() {
		return HTTPExecutionBinding{}, refuse(CodeInvalidBinding, "plan or HTTP execution authority is incomplete")
	}
	if plan.Adapter().Domain != domain.AdapterHTTP || plan.ExecutionShape() != domain.OneLoopbackHTTPRequest || len(plan.SetupArgv()) != 0 {
		return HTTPExecutionBinding{}, refuse(CodeInvalidBinding, "binding requires one loopback HTTP request without setup")
	}
	if !equalStrings(plan.StartArgv(), start.LogicalArgv()) {
		return HTTPExecutionBinding{}, refuse(CodeInvalidBinding, "plan start argv differs from the closed Node start spec")
	}
	fixtureRecipe, err := NewHTTPFixtureRecipe()
	if err != nil {
		return HTTPExecutionBinding{}, err
	}
	if plan.FixtureRecipeDigest() != fixtureRecipe.Digest() {
		return HTTPExecutionBinding{}, refuse(CodeInvalidBinding, "plan fixture recipe does not name the U4 private seed overlay")
	}
	if plan.Readiness().Kind != domain.FixtureOwnedReadiness || plan.Readiness().SignalName != readiness.SignalName() {
		return HTTPExecutionBinding{}, refuse(CodeInvalidBinding, "plan readiness does not name the exact one-byte protocol")
	}
	budgets := plan.Budgets()
	if plan.CapturePolicyDigest() != capturePolicy.Digest() || budgets.HTTPBodyBytes != capturePolicy.BodyBytes() {
		return HTTPExecutionBinding{}, refuse(CodeInvalidBinding, "plan capture policy does not resolve to the supplied HTTP caps")
	}
	if plan.ProjectionDefinitionDigest() != projection.Binding().Digest() {
		return HTTPExecutionBinding{}, refuse(CodeInvalidBinding, "plan projection binding does not resolve to the supplied HTTP definition")
	}
	identity := executionBindingIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPExecutionBinding", Authority: HTTPExecutionAuthorityV1,
		WorldPlanDigest: plan.Digest().String(), StimulusDigest: stimulus.Digest().String(),
		ExecutionPayloadDigest: stimulus.ExecutionPayloadDigest().String(), StartSpecDigest: start.Digest().String(),
		FixtureRecipeDigest: fixtureRecipe.Digest().String(), CapturePolicyDigest: capturePolicy.Digest().String(),
		HTTPBodyCaptureBytes: capturePolicy.BodyBytes(), ReadinessContractDigest: readiness.Digest().String(),
		ProjectionBindingDigest: projection.Binding().Digest().String(), AdapterProjectionDigest: projection.Digest().String(),
	}
	digest, canonicalBytes, err := digestTyped("HTTPExecutionBinding", identity)
	if err != nil {
		return HTTPExecutionBinding{}, err
	}
	return HTTPExecutionBinding{
		digest: digest, canonicalBytes: canonicalBytes, plan: plan, stimulus: stimulus, start: start,
		fixtureRecipe: fixtureRecipe, capturePolicy: capturePolicy, readiness: readiness, projection: projection,
	}, nil
}

func (b HTTPExecutionBinding) Valid() bool {
	if !b.digest.Valid() || len(b.canonicalBytes) == 0 {
		return false
	}
	rebuilt, err := BindExecution(b.plan, b.stimulus, b.start, b.capturePolicy, b.readiness, b.projection)
	return err == nil && rebuilt.digest == b.digest && bytes.Equal(rebuilt.canonicalBytes, b.canonicalBytes)
}
func (b HTTPExecutionBinding) Digest() domain.Digest { return b.digest }
func (b HTTPExecutionBinding) CanonicalBytes() []byte {
	return append([]byte(nil), b.canonicalBytes...)
}
func (b HTTPExecutionBinding) Plan() domain.WorldPlan        { return b.plan }
func (b HTTPExecutionBinding) PlanDigest() domain.Digest     { return b.plan.Digest() }
func (b HTTPExecutionBinding) Stimulus() HTTPStimulus        { return b.stimulus }
func (b HTTPExecutionBinding) StimulusDigest() domain.Digest { return b.stimulus.Digest() }
func (b HTTPExecutionBinding) ExecutionPayloadDigest() domain.Digest {
	return b.stimulus.ExecutionPayloadDigest()
}
func (b HTTPExecutionBinding) StartSpec() HTTPStartSpec           { return b.start }
func (b HTTPExecutionBinding) LogicalArgv() []string              { return b.start.LogicalArgv() }
func (b HTTPExecutionBinding) Seeds() []HTTPSeedFile              { return b.stimulus.Seeds() }
func (b HTTPExecutionBinding) FixtureRecipe() HTTPFixtureRecipe   { return b.fixtureRecipe }
func (b HTTPExecutionBinding) FixtureRecipeDigest() domain.Digest { return b.fixtureRecipe.Digest() }
func (b HTTPExecutionBinding) CapturePolicy() HTTPCapturePolicy   { return b.capturePolicy }
func (b HTTPExecutionBinding) CapturePolicyDigest() domain.Digest { return b.capturePolicy.Digest() }
func (b HTTPExecutionBinding) Readiness() HTTPReadinessContract   { return b.readiness }
func (b HTTPExecutionBinding) ReadinessDigest() domain.Digest     { return b.readiness.Digest() }
func (b HTTPExecutionBinding) AdapterProjectionDefinitionDigest() domain.Digest {
	return b.projection.Digest()
}
func (b HTTPExecutionBinding) ProjectionDefinitionBinding() domain.ProjectionDefinitionBinding {
	return b.projection.Binding()
}
