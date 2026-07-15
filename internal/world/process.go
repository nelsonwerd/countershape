package world

import (
	"context"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const processEscapeExclusion = "PROCESS_GROUP_OR_SESSION_ESCAPE_EXCLUDED_FROM_CONTAINMENT_CLAIM" // MUTANT_U2_CLAIM_PROCESS_ESCAPE_CONTAINMENT

// A zero-signal probe narrows accidental signaling, but Darwin exposes no
// atomic "probe this exact group incarnation and signal it" operation. The
// group can disappear and its numeric PGID can be reused between those calls.
// Countershape therefore receipts this residual boundary instead of promoting
// the pre-TERM probe into an identity/containment claim.
const processGroupReuseExclusion = "PRE_TERM_PROBE_AND_SIGNAL_ARE_NON_ATOMIC_PGID_REUSE_EXCLUDED_FROM_CLEANUP_CLAIM"

const (
	preTermProbeNotApplicable = "NOT_APPLICABLE"
	preTermProbePresent       = "PRESENT"
	preTermProbeAbsent        = "ABSENT"
	preTermProbeUncertain     = "UNCERTAIN"
)

const terminalArbitrationContract = "OWNER_OBSERVED_PRIORITY_OUTPUT_CANCEL_DEADLINE_WAIT"

type processRequest struct {
	tool              resolvedTool
	logicalArgv       []string
	environment       []string
	cwd               string
	stdoutLimit       int64
	stderrLimit       int64
	executionBudgetMS int64
	teardownBudgetMS  int64
	markerBeforeSpawn bool
	onGroupOwned      func() error
}

type physicalProcessResult struct {
	spawnAttempted    bool
	markerBeforeSpawn bool
	started           bool
	pid               int
	processGroupID    int
	processGroupOwned bool
	exitCode          int
	exitSignal        string
	waitError         string
	stdout            []byte
	stderr            []byte
	stdoutObserved    int64
	stderrObserved    int64
	stdoutOverflow    bool
	stderrOverflow    bool
	primary           domain.ControlReason
	preTermProbe      string
	termSent          bool
	killSent          bool
	directChildWaited bool
	stdoutDrained     bool
	stderrDrained     bool
	finalProbeClean   bool
	finalProbeError   string
	teardownError     bool
	orphanRisk        bool
	diagnosticCode    string
}

func runProcess(ctx context.Context, request processRequest) physicalProcessResult {
	return runPlatformProcess(ctx, request)
}

func applyCompletedOutputControl(result *physicalProcessResult) {
	if outputOverflowIsPrimaryControl && result.primary == "" && (result.stdoutOverflow || result.stderrOverflow) {
		result.primary = domain.ControlOutputLimit
	}
}
