package http

import (
	"github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type HTTPStimulus = model.HTTPStimulus
type HTTPStimulusConfig = model.HTTPStimulusConfig
type HTTPStimulusMeasure = model.HTTPStimulusMeasure
type HTTPMethod = model.HTTPMethod
type HTTPQueryEntry = model.HTTPQueryEntry
type HTTPRequestHeader = model.HTTPRequestHeader
type HTTPBody = model.HTTPBody
type HTTPSeedFile = model.HTTPSeedFile
type SeedMode = model.SeedMode
type Presence = model.Presence
type HTTPStartSpec = model.HTTPStartSpec
type HTTPFixtureRecipe = model.HTTPFixtureRecipe
type HTTPReadinessContract = model.HTTPReadinessContract
type HTTPCapturePolicy = model.HTTPCapturePolicy
type HTTPCapturePolicyConfig = model.HTTPCapturePolicyConfig
type HTTPExecutionBinding = model.HTTPExecutionBinding

const (
	MethodGET                = model.MethodGET
	MethodPOST               = model.MethodPOST
	MethodPUT                = model.MethodPUT
	MethodPATCH              = model.MethodPATCH
	MethodDELETE             = model.MethodDELETE
	PresenceAbsent           = model.PresenceAbsent
	PresencePresent          = model.PresencePresent
	SeedMode0644             = model.SeedMode0644
	ReadinessSuccessByte     = model.ReadinessSuccessByte
	ReadinessProtocolV1      = model.ReadinessProtocolV1
	ReadinessSignalNameV1    = model.ReadinessSignalNameV1
	HTTPExecutionAuthorityV1 = model.HTTPExecutionAuthorityV1
)

func QueryFlag(name string) (HTTPQueryEntry, error)         { return model.QueryFlag(name) }
func QueryValue(name, value string) (HTTPQueryEntry, error) { return model.QueryValue(name, value) }
func NewRequestHeader(name, value string) (HTTPRequestHeader, error) {
	return model.NewRequestHeader(name, value)
}
func AbsentBody() HTTPBody                       { return model.AbsentBody() }
func PresentBody(value []byte) (HTTPBody, error) { return model.PresentBody(value) }
func NewSeedFile(path string, contents []byte, mode SeedMode) (HTTPSeedFile, error) {
	return model.NewSeedFile(path, contents, mode)
}
func NewHTTPStimulus(config HTTPStimulusConfig) (HTTPStimulus, error) {
	return model.NewHTTPStimulus(config)
}
func NewHTTPStartSpec(entrypoint string) (HTTPStartSpec, error) {
	return model.NewHTTPStartSpec(entrypoint)
}
func NewHTTPFixtureRecipe() (HTTPFixtureRecipe, error) { return model.NewHTTPFixtureRecipe() }
func NewHTTPReadinessContract() (HTTPReadinessContract, error) {
	return model.NewHTTPReadinessContract()
}
func NewHTTPCapturePolicy(config HTTPCapturePolicyConfig) (HTTPCapturePolicy, error) {
	return model.NewHTTPCapturePolicy(config)
}
func BindExecution(
	plan domain.WorldPlan,
	stimulus HTTPStimulus,
	start HTTPStartSpec,
	capture HTTPCapturePolicy,
	readiness HTTPReadinessContract,
	projection HTTPProjectionDefinition,
) (HTTPExecutionBinding, error) {
	if !projection.Valid() {
		return HTTPExecutionBinding{}, refuse(CodeInvalidProjection, "binding requires one resolved HTTP projection definition")
	}
	authority, err := model.ResolveHTTPProjectionAuthority(projection.Digest(), projection.CanonicalBytes(), projection.Binding())
	if err != nil {
		return HTTPExecutionBinding{}, err
	}
	return model.BindExecution(plan, stimulus, start, capture, readiness, authority)
}
