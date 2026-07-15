package world

import (
	"context"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	httpListenerChildFD         = 3
	httpReadinessChildFD        = 4
	httpResponseHeadLimit       = int64(64 << 10)
	httpListenerFDEnvironment   = "COUNTERSHAPE_HTTP_LISTEN_FD"
	httpReadinessFDEnvironment  = "COUNTERSHAPE_HTTP_READINESS_FD"
	httpListenerPortEnvironment = "COUNTERSHAPE_HTTP_PORT"
)

func isHTTPDescriptorEnvironmentName(name string) bool {
	switch name {
	case httpListenerFDEnvironment, httpReadinessFDEnvironment, httpListenerPortEnvironment:
		return true
	default:
		return false
	}
}

// httpResponseSummary is the only adapter-owned result the impure process
// owner needs in order to bind a successful parse to its physical exchange.
// The full typed response remains with the HTTP adapter.
type httpResponseSummary struct {
	digest     domain.Digest
	wireDigest domain.Digest
	status     int
}

type httpServiceRequest struct {
	tool                resolvedTool
	binding             httpmodel.HTTPExecutionBinding
	logicalArgv         []string
	environment         []string
	cwd                 string
	responseLimit       int64
	stdoutLimit         int64
	stderrLimit         int64
	readinessBudgetMS   int64
	probeBudgetMS       int64
	teardownBudgetMS    int64
	markerBeforeSpawn   bool
	onReadinessAccepted func() error
	onResponseCaptured  func() error
}

type httpReadinessPhysical struct {
	listenerFD     int
	readinessFD    int
	endpoint       string
	port           int
	bytesObserved  int64
	observedByte   byte
	eofObserved    bool
	accepted       bool
	diagnosticCode string
}

type httpExchangePhysical struct {
	connectionAttempts    int
	requestWire           []byte
	requestSemanticDigest domain.Digest
	requestRawSHA256      string
	requestWritten        int64
	requestComplete       bool
	responseWire          []byte
	responseObserved      int64
	responseOverflow      bool
	responseSummary       httpResponseSummary
	responseParsed        bool
	diagnosticCode        string
}

type liveHTTPServiceResult struct {
	process     physicalProcessResult
	readiness   httpReadinessPhysical
	exchange    httpExchangePhysical
	environment []string
}

func runLiveHTTPService(ctx context.Context, request httpServiceRequest) liveHTTPServiceResult {
	return runPlatformLiveHTTPService(ctx, request)
}
