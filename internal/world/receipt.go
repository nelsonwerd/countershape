package world

import (
	"errors"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/gitobj"
)

const (
	diagnosticMaterializationFailed             = "MATERIALIZATION_FAILED"
	diagnosticMaterializationReceiptInvalid     = "MATERIALIZATION_RECEIPT_BINDING_INVALID"
	diagnosticMaterializationRootUnavailable    = "MATERIALIZATION_ROOT_UNAVAILABLE"
	diagnosticMaterializationGitMetadataPresent = "MATERIALIZATION_GIT_METADATA_PRESENT"
	diagnosticMaterializationRevalidationFailed = "MATERIALIZATION_REVALIDATION_FAILED"
	diagnosticMarkerContentInvalid              = "ATTEMPT_MARKER_CONTENT_INVALID_BEFORE_SPAWN"
	diagnosticMarkerModeInvalid                 = "ATTEMPT_MARKER_MODE_INVALID_BEFORE_SPAWN"
	diagnosticCancelledDuringMaterialization    = "CONTEXT_CANCELLED_DURING_MATERIALIZATION"
	diagnosticCancelledAfterMaterialization     = "CONTEXT_CANCELLED_AFTER_MATERIALIZATION"
	diagnosticCancelledBeforeSpawn              = "CONTEXT_CANCELLED_BEFORE_SPAWN"
	diagnosticGenericPreProcessCancellation     = "CONTEXT_CANCELLED_BEFORE_PROCESS"
	diagnosticGenericPreProcessStartFailure     = "PROCESS_START_FAILED_BEFORE_SPAWN"
)

// receiptedDiagnostic carries closed machine data through internal failure
// paths. Its cause remains available to errors.Is/As, but arbitrary error text
// never enters the lifecycle digest.
type receiptedDiagnostic struct {
	code  string
	cause error
}

func (d *receiptedDiagnostic) Error() string { return d.code }
func (d *receiptedDiagnostic) Unwrap() error { return d.cause }

func withReceiptDiagnostic(code string, cause error) error {
	return &receiptedDiagnostic{code: code, cause: cause}
}

type ProcessReceipt struct {
	digest            domain.Digest
	attemptID         string
	attemptArtifact   domain.Digest
	worldDigest       domain.Digest
	toolName          string
	toolPath          string
	toolVersion       string
	toolMajor         int
	toolDigest        domain.Digest
	logicalArgv       []string
	environment       []string
	pid               int
	processGroupID    int
	exitCode          int
	exitSignal        string
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
	teardownError     bool
	orphanRisk        bool
	cleanupBoundary   string
	signalBoundary    string
	markerBeforeSpawn bool
	spawnAttempted    bool
	started           bool
	processGroupOwned bool
	diagnosticCode    string
	states            []domain.AttemptState
}

func (r ProcessReceipt) Digest() domain.Digest                { return r.digest }
func (r ProcessReceipt) AttemptID() string                    { return r.attemptID }
func (r ProcessReceipt) AttemptArtifactDigest() domain.Digest { return r.attemptArtifact }
func (r ProcessReceipt) WorldDigest() domain.Digest           { return r.worldDigest }
func (r ProcessReceipt) ToolName() string                     { return r.toolName }
func (r ProcessReceipt) ToolPath() string                     { return r.toolPath }
func (r ProcessReceipt) ToolVersion() string                  { return r.toolVersion }
func (r ProcessReceipt) ToolMajor() int                       { return r.toolMajor }
func (r ProcessReceipt) ToolExecutableDigest() domain.Digest  { return r.toolDigest }
func (r ProcessReceipt) LogicalArgv() []string                { return append([]string(nil), r.logicalArgv...) }
func (r ProcessReceipt) Environment() []string                { return append([]string(nil), r.environment...) }
func (r ProcessReceipt) PID() int                             { return r.pid }
func (r ProcessReceipt) ProcessGroupID() int                  { return r.processGroupID }
func (r ProcessReceipt) ExitCode() int                        { return r.exitCode }
func (r ProcessReceipt) ExitSignal() string                   { return r.exitSignal }
func (r ProcessReceipt) Stdout() []byte                       { return append([]byte(nil), r.stdout...) }
func (r ProcessReceipt) Stderr() []byte                       { return append([]byte(nil), r.stderr...) }
func (r ProcessReceipt) StdoutObservedBytes() int64           { return r.stdoutObserved }
func (r ProcessReceipt) StderrObservedBytes() int64           { return r.stderrObserved }
func (r ProcessReceipt) StdoutOverflow() bool                 { return r.stdoutOverflow }
func (r ProcessReceipt) StderrOverflow() bool                 { return r.stderrOverflow }
func (r ProcessReceipt) PrimaryControl() (domain.ControlReason, bool) {
	return r.primary, r.primary != ""
}
func (r ProcessReceipt) PreTermGroupProbe() string          { return r.preTermProbe }
func (r ProcessReceipt) TermSent() bool                     { return r.termSent }
func (r ProcessReceipt) KillSent() bool                     { return r.killSent }
func (r ProcessReceipt) DirectChildWaited() bool            { return r.directChildWaited }
func (r ProcessReceipt) DrainsComplete() bool               { return r.stdoutDrained && r.stderrDrained }
func (r ProcessReceipt) FinalGroupProbeClean() bool         { return r.finalProbeClean }
func (r ProcessReceipt) TeardownError() bool                { return r.teardownError }
func (r ProcessReceipt) OrphanRisk() bool                   { return r.orphanRisk }
func (r ProcessReceipt) CleanupBoundary() string            { return r.cleanupBoundary }
func (r ProcessReceipt) ProcessGroupSignalBoundary() string { return r.signalBoundary }
func (r ProcessReceipt) MarkerExistedBeforeSpawn() bool     { return r.markerBeforeSpawn }
func (r ProcessReceipt) SpawnAttempted() bool               { return r.spawnAttempted }
func (r ProcessReceipt) Started() bool                      { return r.started }
func (r ProcessReceipt) ProcessGroupOwned() bool            { return r.processGroupOwned }
func (r ProcessReceipt) DiagnosticCode() string             { return r.diagnosticCode }

func (r ProcessReceipt) clone() ProcessReceipt {
	result := r
	result.logicalArgv = append([]string(nil), r.logicalArgv...)
	result.environment = append([]string(nil), r.environment...)
	result.stdout = append([]byte(nil), r.stdout...)
	result.stderr = append([]byte(nil), r.stderr...)
	result.states = append([]domain.AttemptState(nil), r.states...)
	return result
}

type receiptInput struct {
	allocated       allocatedAttempt
	materialization gitobj.MaterializationReceipt
	attempt         domain.Attempt
	tool            resolvedTool
	logicalArgv     []string
	environment     []string
	physical        physicalProcessResult
	primary         domain.ControlReason
	diagnostic      error
}

func buildProcessReceipt(input receiptInput) (ProcessReceipt, error) {
	stdoutDigest, err := canon.DigestBytes("ProcessStdoutCapture", input.physical.stdout)
	if err != nil {
		return ProcessReceipt{}, err
	}
	stderrDigest, err := canon.DigestBytes("ProcessStderrCapture", input.physical.stderr)
	if err != nil {
		return ProcessReceipt{}, err
	}
	primary := input.primary
	if primary == "" {
		primary = input.physical.primary
	}
	diagnosticCode := receiptDiagnosticCode(input, primary)
	preTermProbe := input.physical.preTermProbe
	if preTermProbe == "" {
		preTermProbe = preTermProbeNotApplicable
	}
	states := input.attempt.StateHistory()
	identity := struct {
		SchemaVersion     string   `json:"schema_version"`
		Kind              string   `json:"kind"`
		AttemptID         string   `json:"attempt_id"`
		AttemptArtifact   string   `json:"attempt_artifact_digest"`
		WorldDigest       string   `json:"world_instance_digest"`
		ManifestDigest    string   `json:"materialization_manifest_digest"`
		AttemptRoot       string   `json:"attempt_root"`
		CandidateRoot     string   `json:"candidate_root"`
		HomeRoot          string   `json:"home_root"`
		TemporaryRoot     string   `json:"temporary_root"`
		XDGConfigRoot     string   `json:"xdg_config_root"`
		XDGCacheRoot      string   `json:"xdg_cache_root"`
		XDGDataRoot       string   `json:"xdg_data_root"`
		XDGStateRoot      string   `json:"xdg_state_root"`
		StateRoot         string   `json:"state_root"`
		EvidenceRoot      string   `json:"evidence_root"`
		MarkerPath        string   `json:"attempt_marker_path"`
		MarkerBeforeSpawn bool     `json:"marker_before_spawn"`
		ToolName          string   `json:"tool_name"`
		ToolPath          string   `json:"tool_path"`
		ToolVersion       string   `json:"tool_version"`
		ToolMajor         int      `json:"tool_major"`
		ToolDigest        string   `json:"tool_executable_digest"`
		LogicalArgv       []string `json:"logical_argv"`
		Environment       []string `json:"sparse_environment"`
		PID               int      `json:"pid"`
		ProcessGroupID    int      `json:"process_group_id"`
		ExitCode          int      `json:"exit_code"`
		ExitSignal        string   `json:"exit_signal"`
		StdoutDigest      string   `json:"stdout_digest"`
		StderrDigest      string   `json:"stderr_digest"`
		StdoutObserved    int64    `json:"stdout_observed_bytes"`
		StderrObserved    int64    `json:"stderr_observed_bytes"`
		StdoutOverflow    bool     `json:"stdout_overflow"`
		StderrOverflow    bool     `json:"stderr_overflow"`
		PrimaryControl    string   `json:"primary_control"`
		PreTermProbe      string   `json:"pre_term_group_probe"`
		TermSent          bool     `json:"term_sent"`
		KillSent          bool     `json:"kill_sent"`
		DirectChildWaited bool     `json:"direct_child_waited"`
		StdoutDrained     bool     `json:"stdout_drained"`
		StderrDrained     bool     `json:"stderr_drained"`
		FinalProbeClean   bool     `json:"final_group_probe_clean"`
		TeardownError     bool     `json:"teardown_error"`
		OrphanRisk        bool     `json:"orphan_risk"`
		CleanupBoundary   string   `json:"cleanup_boundary"`
		SignalBoundary    string   `json:"process_group_signal_boundary"`
		Arbitration       string   `json:"terminal_arbitration"`
		SpawnAttempted    bool     `json:"spawn_attempted"`
		Started           bool     `json:"started"`
		ProcessGroupOwned bool     `json:"process_group_owned"`
		DiagnosticCode    string   `json:"diagnostic_code"`
		StateHistory      []string `json:"state_history"`
	}{
		SchemaVersion: domain.SchemaVersion, Kind: "ProcessLifecycleReceipt",
		AttemptID: input.allocated.attemptID, AttemptArtifact: input.allocated.markerDigest.String(),
		WorldDigest: input.allocated.world.Digest().String(), ManifestDigest: input.materialization.ManifestDigest.String(),
		AttemptRoot: input.allocated.roots.attempt, CandidateRoot: input.materialization.PublishedRoot,
		HomeRoot: input.allocated.roots.home, TemporaryRoot: input.allocated.roots.temporary,
		XDGConfigRoot: input.allocated.roots.xdgConfig, XDGCacheRoot: input.allocated.roots.xdgCache,
		XDGDataRoot:  input.allocated.roots.xdgData,
		XDGStateRoot: input.allocated.roots.xdgState,
		StateRoot:    input.allocated.roots.state, EvidenceRoot: input.allocated.roots.evidence,
		MarkerPath: input.allocated.roots.marker, MarkerBeforeSpawn: input.physical.markerBeforeSpawn,
		ToolName: input.tool.name, ToolPath: input.tool.absolutePath, ToolVersion: input.tool.version,
		ToolMajor: input.tool.major, ToolDigest: input.tool.executableDigest.String(),
		LogicalArgv: append([]string(nil), input.logicalArgv...), Environment: append([]string(nil), input.environment...),
		PID: input.physical.pid, ProcessGroupID: input.physical.processGroupID,
		ExitCode: input.physical.exitCode, ExitSignal: input.physical.exitSignal,
		StdoutDigest: stdoutDigest.String(), StderrDigest: stderrDigest.String(),
		StdoutObserved: input.physical.stdoutObserved, StderrObserved: input.physical.stderrObserved,
		StdoutOverflow: input.physical.stdoutOverflow, StderrOverflow: input.physical.stderrOverflow,
		PrimaryControl: string(primary), PreTermProbe: preTermProbe,
		TermSent: input.physical.termSent, KillSent: input.physical.killSent,
		DirectChildWaited: input.physical.directChildWaited, StdoutDrained: input.physical.stdoutDrained,
		StderrDrained: input.physical.stderrDrained, FinalProbeClean: input.physical.finalProbeClean,
		TeardownError: input.physical.teardownError, OrphanRisk: input.physical.orphanRisk,
		CleanupBoundary: processEscapeExclusion, SignalBoundary: processGroupReuseExclusion,
		Arbitration:    terminalArbitrationContract,
		SpawnAttempted: input.physical.spawnAttempted, Started: input.physical.started,
		ProcessGroupOwned: input.physical.processGroupOwned, DiagnosticCode: diagnosticCode,
		StateHistory: stateStrings(states),
	}
	digest, _, err := canon.DigestTyped("ProcessLifecycleReceipt", identity)
	if err != nil {
		return ProcessReceipt{}, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return ProcessReceipt{}, err
	}
	return ProcessReceipt{
		digest: parsed, attemptID: input.allocated.attemptID, attemptArtifact: input.allocated.markerDigest,
		worldDigest: input.allocated.world.Digest(), toolName: input.tool.name, toolPath: input.tool.absolutePath,
		toolVersion: input.tool.version, toolMajor: input.tool.major, toolDigest: input.tool.executableDigest,
		logicalArgv: append([]string(nil), input.logicalArgv...), environment: append([]string(nil), input.environment...),
		pid: input.physical.pid, processGroupID: input.physical.processGroupID, exitCode: input.physical.exitCode,
		exitSignal: input.physical.exitSignal, stdout: append([]byte(nil), input.physical.stdout...),
		stderr: append([]byte(nil), input.physical.stderr...), stdoutObserved: input.physical.stdoutObserved,
		stderrObserved: input.physical.stderrObserved, stdoutOverflow: input.physical.stdoutOverflow,
		stderrOverflow: input.physical.stderrOverflow, primary: primary, preTermProbe: preTermProbe,
		termSent: input.physical.termSent,
		killSent: input.physical.killSent, directChildWaited: input.physical.directChildWaited,
		stdoutDrained: input.physical.stdoutDrained, stderrDrained: input.physical.stderrDrained,
		finalProbeClean: input.physical.finalProbeClean, teardownError: input.physical.teardownError,
		orphanRisk: input.physical.orphanRisk, cleanupBoundary: processEscapeExclusion,
		signalBoundary:    processGroupReuseExclusion,
		markerBeforeSpawn: input.physical.markerBeforeSpawn, spawnAttempted: input.physical.spawnAttempted,
		started: input.physical.started, processGroupOwned: input.physical.processGroupOwned,
		diagnosticCode: diagnosticCode, states: append([]domain.AttemptState(nil), states...),
	}, nil
}

func receiptDiagnosticCode(input receiptInput, primary domain.ControlReason) string {
	if input.physical.diagnosticCode != "" {
		return input.physical.diagnosticCode
	}
	var diagnostic *receiptedDiagnostic
	if errors.As(input.diagnostic, &diagnostic) {
		return diagnostic.code
	}
	if code, ok := RefusalCodeOf(input.diagnostic); ok {
		return string(code)
	}
	if code, ok := gitobj.RefusalCodeOf(input.diagnostic); ok {
		return string(code)
	}
	if input.diagnostic == nil {
		return ""
	}
	switch primary {
	case domain.ControlCancelled:
		return diagnosticGenericPreProcessCancellation
	case domain.ControlStartError:
		return diagnosticGenericPreProcessStartFailure
	default:
		return diagnosticMaterializationFailed
	}
}

func stateStrings(states []domain.AttemptState) []string {
	result := make([]string, len(states))
	for index, state := range states {
		result[index] = string(state)
	}
	return result
}
