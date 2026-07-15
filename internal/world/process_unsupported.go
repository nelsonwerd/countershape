//go:build !darwin

package world

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/domain"
)

// The shared output-control helper remains compilable on unsupported hosts;
// runPlatformProcess still returns only the explicit platform refusal below.
const outputOverflowIsPrimaryControl = true

func runPlatformProcess(_ context.Context, request processRequest) physicalProcessResult {
	stdinDigest, _ := digestProcessStdin(request.stdin)
	return physicalProcessResult{
		physicalExecutionEntered: false,
		primary:                  domain.ControlStartError,
		finalProbeError:          string(CodeUnsupportedPlatform),
		stdinPresence:            request.stdin.presence, stdinDeclared: int64(len(request.stdin.bytes)),
		stdinDigest: stdinDigest, stdinComplete: request.stdin.presence != processStdinPresent,
	}
}
