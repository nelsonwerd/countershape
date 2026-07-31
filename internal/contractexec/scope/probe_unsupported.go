//go:build !darwin

package scope

import "context"

const (
	ImportModuleEnvironment  = "COUNTERSHAPE_C5_IMPORT_CANARY_MODULE"
	ImportSocketEnvironment  = "COUNTERSHAPE_C5_IMPORT_CANARY_SOCKET"
	ServiceSocketEnvironment = "COUNTERSHAPE_C5_SERVICE_CANARY_SOCKET"
)

type Probe struct{}

func NewProbe(string) (*Probe, error) {
	return nil, fail(CodeProbeAbsent, nil)
}

func (*Probe) Environment() map[string]string { return nil }

func (*Probe) Finish(context.Context) (Measurements, error) {
	return Measurements{}, fail(CodeProbeAbsent, nil)
}
