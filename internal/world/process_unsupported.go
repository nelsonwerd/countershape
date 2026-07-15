//go:build !darwin

package world

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/domain"
)

// The shared output-control helper remains compilable on unsupported hosts;
// runPlatformProcess still returns only the explicit platform refusal below.
const outputOverflowIsPrimaryControl = true

func runPlatformProcess(context.Context, processRequest) physicalProcessResult {
	return physicalProcessResult{primary: domain.ControlStartError, finalProbeError: string(CodeUnsupportedPlatform)}
}
