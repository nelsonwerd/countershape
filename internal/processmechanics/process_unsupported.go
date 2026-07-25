//go:build !darwin

package processmechanics

import "context"

type runningState struct{}

type Running struct{ state *runningState }

func (prepared *Prepared) startPlatform(_ context.Context) (SpawnObservation, *Running, error) {
	result := Result{ExitCode: -1, PreTermProbe: PreTermProbeNotApplicable}
	if prepared != nil && prepared.state != nil {
		result = initialResult(prepared.state.invocation)
		if err := prepared.begin(); err != nil {
			return SpawnObservation{}, nil, newStartError("PREPARED_PROCESS_ALREADY_CONSUMED", err, result)
		}
	}
	return SpawnObservation{}, nil, newStartError("UNSUPPORTED_PLATFORM", nil, result)
}

func (running *Running) AbortReadinessTransition() {}

func (running *Running) AbortSpawnObservationPersistence() {}

func (running *Running) Close() Result {
	return Result{ExitCode: -1, Primary: ControlStartError, PreTermProbe: PreTermProbeNotApplicable, DiagnosticCode: "UNSUPPORTED_PLATFORM"}
}
