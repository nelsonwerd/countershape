package cli

import (
	"github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

// Top-level aliases keep adapter consumers ergonomic while the cycle-free
// model package remains the only package imported by the world edge.
type CLIStimulus = model.CLIStimulus
type CLIStimulusConfig = model.CLIStimulusConfig
type CLIStimulusMeasure = model.CLIStimulusMeasure
type CLIStdin = model.CLIStdin
type CLIEnvironmentBinding = model.CLIEnvironmentBinding
type CLIFixtureFile = model.CLIFixtureFile
type FixtureMode = model.FixtureMode
type CWDPolicy = model.CWDPolicy
type Presence = model.Presence
type CLIExecutionBinding = model.CLIExecutionBinding
type CLIFixtureRecipe = model.CLIFixtureRecipe

const (
	PresenceAbsent      = model.PresenceAbsent
	PresencePresent     = model.PresencePresent
	FixtureMode0644     = model.FixtureMode0644
	CWDMaterializedRoot = model.CWDMaterializedRoot
)

func AbsentStdin() CLIStdin                       { return model.AbsentStdin() }
func PresentStdin(value []byte) (CLIStdin, error) { return model.PresentStdin(value) }
func AbsentEnvironment(name string) (CLIEnvironmentBinding, error) {
	return model.AbsentEnvironment(name)
}
func PresentEnvironment(name, value string) (CLIEnvironmentBinding, error) {
	return model.PresentEnvironment(name, value)
}
func NewFixtureFile(path string, contents []byte, mode FixtureMode) (CLIFixtureFile, error) {
	return model.NewFixtureFile(path, contents, mode)
}
func NewCLIStimulus(config CLIStimulusConfig) (CLIStimulus, error) {
	return model.NewCLIStimulus(config)
}
func BindExecution(
	plan domain.WorldPlan,
	stimulus CLIStimulus,
	capturePolicy CLICapturePolicy,
	projection CLIProjectionDefinition,
) (CLIExecutionBinding, error) {
	if !projection.Valid() {
		return CLIExecutionBinding{}, refuse(CodeInvalidProjection, "execution binding requires one resolved CLI projection definition")
	}
	authority, err := model.ResolveCLIProjectionAuthority(
		projection.Digest(), projection.CanonicalBytes(), projection.Binding(),
	)
	if err != nil {
		return CLIExecutionBinding{}, err
	}
	return model.BindExecution(plan, stimulus, capturePolicy, authority)
}
func NewCLIFixtureRecipe() (CLIFixtureRecipe, error) { return model.NewCLIFixtureRecipe() }
