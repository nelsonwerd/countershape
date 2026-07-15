//go:build !darwin

package world

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func runPlatformLiveHTTPService(_ context.Context, _ httpServiceRequest) liveHTTPServiceResult {
	return liveHTTPServiceResult{process: physicalProcessResult{
		exitCode: -1, primary: domain.ControlStartError,
		diagnosticCode: "HTTP_LIVE_SERVICE_UNSUPPORTED_PLATFORM",
	}}
}
