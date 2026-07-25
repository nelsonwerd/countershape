//go:build !darwin

package world

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/domain"
)

// The world compatibility surface and neutral processmechanics package both
// remain buildable on unsupported hosts; this adapter preserves the existing
// world-specific refusal code without fabricating a physical attempt.
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
