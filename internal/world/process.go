package world

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/processmechanics"
)

const processEscapeExclusion = processmechanics.EscapeExclusion

// A zero-signal probe narrows accidental signaling, but Darwin exposes no
// atomic "probe this exact group incarnation and signal it" operation. The
// group can disappear and its numeric PGID can be reused between those calls.
// Countershape therefore receipts this residual boundary instead of promoting
// the pre-TERM probe into an identity/containment claim.
const processGroupReuseExclusion = processmechanics.ProcessGroupReuseExclusion

const (
	preTermProbeNotApplicable = "NOT_APPLICABLE"
	preTermProbePresent       = "PRESENT"
	preTermProbeAbsent        = "ABSENT"
	preTermProbeUncertain     = "UNCERTAIN"
)

const terminalArbitrationContract = "OWNER_OBSERVED_PRIORITY_OUTPUT_PROBE_TRANSPORT_CANCEL_DEADLINE_WAIT"

type processStdinPresence string

const (
	processStdinLegacy  processStdinPresence = ""
	processStdinAbsent  processStdinPresence = "ABSENT"
	processStdinPresent processStdinPresence = "PRESENT"
)

// processStdin preserves the CLI distinction between no stdin and a present
// zero-byte stream. The legacy zero value retains U2's nil-stdin behavior.
type processStdin struct {
	presence processStdinPresence
	bytes    []byte
}

func (s processStdin) valid() bool {
	return ((s.presence == processStdinLegacy || s.presence == processStdinAbsent) && len(s.bytes) == 0) ||
		s.presence == processStdinPresent
}

type processRequest struct {
	tool              resolvedTool
	logicalArgv       []string
	environment       []string
	stdin             processStdin
	cwd               string
	stdoutLimit       int64
	stderrLimit       int64
	executionBudgetMS int64
	teardownBudgetMS  int64
	markerBeforeSpawn bool
	onGroupOwned      func() error
}

type physicalProcessResult struct {
	physicalExecutionEntered bool
	spawnAttempted           bool
	markerBeforeSpawn        bool
	started                  bool
	pid                      int
	processGroupID           int
	processGroupOwned        bool
	exitCode                 int
	exitSignal               string
	waitError                string
	stdout                   []byte
	stderr                   []byte
	stdoutObserved           int64
	stderrObserved           int64
	stdoutOverflow           bool
	stderrOverflow           bool
	stdoutCaptureLimit       int64
	stderrCaptureLimit       int64
	primary                  domain.ControlReason
	preTermProbe             string
	termSent                 bool
	killSent                 bool
	directChildWaited        bool
	stdoutDrained            bool
	stderrDrained            bool
	finalProbeClean          bool
	finalProbeError          string
	teardownError            bool
	orphanRisk               bool
	diagnosticCode           string
	stdinPresence            processStdinPresence
	stdinDeclared            int64
	stdinDigest              domain.Digest
	stdinPipeAllocated       bool
	stdinWriterStarted       bool
	stdinHandoffAttempted    bool
	stdinWritten             int64
	stdinComplete            bool
	stdinErrorCode           string
}

func runProcess(ctx context.Context, request processRequest) physicalProcessResult {
	return runPlatformProcess(ctx, request)
}

func digestProcessStdin(stdin processStdin) (domain.Digest, error) {
	digest, err := canon.DigestBytes("CLIStdinBytes", stdin.bytes)
	if err != nil {
		return "", err
	}
	return domain.ParseDigest(digest.String())
}

func applyCompletedOutputControl(result *physicalProcessResult) {
	if outputOverflowIsPrimaryControl && result.primary == "" && (result.stdoutOverflow || result.stderrOverflow) {
		result.primary = domain.ControlOutputLimit
	}
}
