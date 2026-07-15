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
	digest                     domain.Digest
	attemptID                  string
	attemptArtifact            domain.Digest
	worldDigest                domain.Digest
	toolName                   string
	toolPath                   string
	toolVersion                string
	toolMajor                  int
	toolDigest                 domain.Digest
	logicalArgv                []string
	declaredLogicalArgv        []string
	environment                []string
	pid                        int
	processGroupID             int
	exitCode                   int
	exitSignal                 string
	stdout                     []byte
	stderr                     []byte
	stdoutObserved             int64
	stderrObserved             int64
	stdoutOverflow             bool
	stderrOverflow             bool
	primary                    domain.ControlReason
	preTermProbe               string
	termSent                   bool
	killSent                   bool
	directChildWaited          bool
	stdoutDrained              bool
	stderrDrained              bool
	finalProbeClean            bool
	teardownError              bool
	orphanRisk                 bool
	cleanupBoundary            string
	signalBoundary             string
	markerBeforeSpawn          bool
	spawnAttempted             bool
	started                    bool
	processGroupOwned          bool
	diagnosticCode             string
	states                     []domain.AttemptState
	executionAuthority         string
	physicalExecutionEntered   bool
	executionBindingDigest     domain.Digest
	stimulusDigest             domain.Digest
	executionPayloadDigest     domain.Digest
	fixtureRecipeDigest        domain.Digest
	fixtureOverlayDigest       domain.Digest
	cwdPolicy                  string
	workingDirectory           string
	stdinPresence              string
	stdinBytes                 int64
	stdinDigest                domain.Digest
	stdinDelivery              string
	stdinPipeAllocated         bool
	stdinWriterStarted         bool
	stdinHandoffAttempted      bool
	stdinWritten               int64
	stdinDeliveryComplete      bool
	stdinDeliveryError         string
	stdoutCaptureLimit         int64
	stderrCaptureLimit         int64
	scheduleOrdinal            int
	scheduleCandidateCount     int
	scheduleRepetition         int
	schedulePresent            bool
	invocationEvidenceDigest   domain.Digest
	invocationEvidenceStatus   CLIInvocationEvidenceStatus
	invocationEvidencePresence CLIInvocationEvidencePresence
	invocationEvidenceValid    bool
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
func (r ProcessReceipt) DeclaredLogicalArgv() []string {
	return append([]string(nil), r.declaredLogicalArgv...)
}
func (r ProcessReceipt) Environment() []string      { return append([]string(nil), r.environment...) }
func (r ProcessReceipt) PID() int                   { return r.pid }
func (r ProcessReceipt) ProcessGroupID() int        { return r.processGroupID }
func (r ProcessReceipt) ExitCode() int              { return r.exitCode }
func (r ProcessReceipt) ExitSignal() string         { return r.exitSignal }
func (r ProcessReceipt) Stdout() []byte             { return append([]byte(nil), r.stdout...) }
func (r ProcessReceipt) Stderr() []byte             { return append([]byte(nil), r.stderr...) }
func (r ProcessReceipt) StdoutObservedBytes() int64 { return r.stdoutObserved }
func (r ProcessReceipt) StderrObservedBytes() int64 { return r.stderrObserved }
func (r ProcessReceipt) StdoutOverflow() bool       { return r.stdoutOverflow }
func (r ProcessReceipt) StderrOverflow() bool       { return r.stderrOverflow }
func (r ProcessReceipt) PrimaryControl() (domain.ControlReason, bool) {
	return r.primary, r.primary != ""
}
func (r ProcessReceipt) PreTermGroupProbe() string             { return r.preTermProbe }
func (r ProcessReceipt) TermSent() bool                        { return r.termSent }
func (r ProcessReceipt) KillSent() bool                        { return r.killSent }
func (r ProcessReceipt) DirectChildWaited() bool               { return r.directChildWaited }
func (r ProcessReceipt) DrainsComplete() bool                  { return r.stdoutDrained && r.stderrDrained }
func (r ProcessReceipt) FinalGroupProbeClean() bool            { return r.finalProbeClean }
func (r ProcessReceipt) TeardownError() bool                   { return r.teardownError }
func (r ProcessReceipt) OrphanRisk() bool                      { return r.orphanRisk }
func (r ProcessReceipt) CleanupBoundary() string               { return r.cleanupBoundary }
func (r ProcessReceipt) ProcessGroupSignalBoundary() string    { return r.signalBoundary }
func (r ProcessReceipt) MarkerExistedBeforeSpawn() bool        { return r.markerBeforeSpawn }
func (r ProcessReceipt) SpawnAttempted() bool                  { return r.spawnAttempted }
func (r ProcessReceipt) Started() bool                         { return r.started }
func (r ProcessReceipt) ProcessGroupOwned() bool               { return r.processGroupOwned }
func (r ProcessReceipt) DiagnosticCode() string                { return r.diagnosticCode }
func (r ProcessReceipt) ExecutionAuthorityMarker() string      { return r.executionAuthority }
func (r ProcessReceipt) PhysicalExecutionEntered() bool        { return r.physicalExecutionEntered }
func (r ProcessReceipt) ExecutionBindingDigest() domain.Digest { return r.executionBindingDigest }
func (r ProcessReceipt) StimulusDigest() domain.Digest         { return r.stimulusDigest }
func (r ProcessReceipt) ExecutionPayloadDigest() domain.Digest { return r.executionPayloadDigest }
func (r ProcessReceipt) FixtureRecipeDigest() domain.Digest    { return r.fixtureRecipeDigest }
func (r ProcessReceipt) FixtureOverlayDigest() domain.Digest   { return r.fixtureOverlayDigest }
func (r ProcessReceipt) CWDPolicy() string                     { return r.cwdPolicy }
func (r ProcessReceipt) WorkingDirectory() string              { return r.workingDirectory }
func (r ProcessReceipt) StdinPresence() string                 { return r.stdinPresence }
func (r ProcessReceipt) StdinBytes() int64                     { return r.stdinBytes }
func (r ProcessReceipt) StdinDigest() domain.Digest            { return r.stdinDigest }
func (r ProcessReceipt) StdinDelivery() string                 { return r.stdinDelivery }
func (r ProcessReceipt) StdinPipeAllocated() bool              { return r.stdinPipeAllocated }
func (r ProcessReceipt) StdinWriterStarted() bool              { return r.stdinWriterStarted }
func (r ProcessReceipt) StdinHandoffAttempted() bool           { return r.stdinHandoffAttempted }
func (r ProcessReceipt) StdinWrittenBytes() int64              { return r.stdinWritten }
func (r ProcessReceipt) StdinDeliveryComplete() bool           { return r.stdinDeliveryComplete }
func (r ProcessReceipt) StdinDeliveryErrorCode() string        { return r.stdinDeliveryError }
func (r ProcessReceipt) StdoutCaptureLimit() int64             { return r.stdoutCaptureLimit }
func (r ProcessReceipt) StderrCaptureLimit() int64             { return r.stderrCaptureLimit }
func (r ProcessReceipt) ScheduleOrdinal() (int, bool) {
	return r.scheduleOrdinal, r.schedulePresent

}
func (r ProcessReceipt) ScheduleCandidateCount() (int, bool) {
	return r.scheduleCandidateCount, r.schedulePresent
}
func (r ProcessReceipt) ScheduleRepetition() (int, bool) {
	return r.scheduleRepetition, r.schedulePresent
}
func (r ProcessReceipt) InvocationEvidenceDigest() domain.Digest { return r.invocationEvidenceDigest }
func (r ProcessReceipt) InvocationEvidenceStatus() CLIInvocationEvidenceStatus {
	return r.invocationEvidenceStatus
}
func (r ProcessReceipt) InvocationEvidencePresence() CLIInvocationEvidencePresence {
	return r.invocationEvidencePresence
}
func (r ProcessReceipt) InvocationEvidencePresent() bool {
	return r.invocationEvidencePresence == CLIInvocationPresencePresent
}
func (r ProcessReceipt) InvocationEvidenceValidated() bool { return r.invocationEvidenceValid }

func (r ProcessReceipt) clone() ProcessReceipt {
	result := r
	result.logicalArgv = append([]string(nil), r.logicalArgv...)
	result.declaredLogicalArgv = append([]string(nil), r.declaredLogicalArgv...)
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
	cli             processCLILineage
}

type processCLILineage struct {
	authority                  string
	physicalExecutionEntered   bool
	declaredLogicalArgv        []string
	executionBindingDigest     domain.Digest
	stimulusDigest             domain.Digest
	executionPayloadDigest     domain.Digest
	fixtureRecipeDigest        domain.Digest
	fixtureOverlayDigest       domain.Digest
	cwdPolicy                  string
	workingDirectory           string
	stdinPresence              string
	stdinBytes                 int64
	stdinDigest                domain.Digest
	stdinDelivery              string
	stdinPipeAllocated         bool
	stdinWriterStarted         bool
	stdinHandoffAttempted      bool
	stdinWritten               int64
	stdinDeliveryComplete      bool
	stdinDeliveryError         string
	stdoutCaptureLimit         int64
	stderrCaptureLimit         int64
	scheduleOrdinal            int
	candidateCount             int
	scheduleRepetition         int
	invocationEvidenceDigest   domain.Digest
	invocationEvidenceStatus   CLIInvocationEvidenceStatus
	invocationEvidencePresence CLIInvocationEvidencePresence
	invocationEvidenceValid    bool
}

func (l processCLILineage) present() bool { return l.authority != "" }

func (l processCLILineage) valid() bool {
	if !l.present() {
		return true
	}
	if l.authority != CLIExecutionAuthorityV1 || !l.executionBindingDigest.Valid() ||
		len(l.declaredLogicalArgv) == 0 || !l.stimulusDigest.Valid() ||
		!l.executionPayloadDigest.Valid() || !l.fixtureRecipeDigest.Valid() ||
		(l.stdinPresence != "ABSENT" && l.stdinPresence != "PRESENT") ||
		l.stdinBytes < 0 || !l.stdinDigest.Valid() || l.scheduleOrdinal < 0 ||
		l.candidateCount < 2 || l.candidateCount > 4 || l.scheduleRepetition < 0 ||
		l.scheduleOrdinal/l.candidateCount != l.scheduleRepetition ||
		l.stdinWritten < 0 || l.stdinWritten > l.stdinBytes ||
		l.invocationEvidenceStatus == "" {
		return false
	}
	if !l.physicalExecutionEntered {
		return l.stdoutCaptureLimit == 0 && l.stderrCaptureLimit == 0 &&
			l.stdinDelivery == "NOT_APPLIED" && !l.stdinPipeAllocated &&
			!l.stdinWriterStarted && !l.stdinHandoffAttempted && l.stdinWritten == 0 &&
			!l.stdinDeliveryComplete && l.stdinDeliveryError == "" && l.invocationLineageValid()
	}
	if l.stdoutCaptureLimit < 1 || l.stderrCaptureLimit < 1 {
		return false
	}
	if l.stdinDelivery == "NOT_APPLIED" {
		return !l.stdinWriterStarted && !l.stdinHandoffAttempted && l.stdinWritten == 0 &&
			!l.stdinDeliveryComplete && l.stdinDeliveryError == "" &&
			(l.stdinPresence == "PRESENT" || !l.stdinPipeAllocated) && l.invocationLineageValid()
	}
	if l.stdinPresence == "ABSENT" {
		return l.stdinBytes == 0 && l.stdinDelivery == "ABSENT_NULL_DEVICE" &&
			!l.stdinPipeAllocated && !l.stdinWriterStarted && !l.stdinHandoffAttempted &&
			l.stdinWritten == 0 && !l.stdinDeliveryComplete && l.stdinDeliveryError == "" &&
			l.invocationLineageValid()
	}
	if l.stdinDelivery != "PRESENT_EXPLICIT_PIPE_WRITER" ||
		l.stdinWriterStarted != l.stdinHandoffAttempted || (l.stdinWriterStarted && !l.stdinPipeAllocated) ||
		(!l.stdinHandoffAttempted && (l.stdinWritten != 0 || l.stdinDeliveryComplete || l.stdinDeliveryError != "")) ||
		(l.stdinDeliveryComplete && (l.stdinWritten != l.stdinBytes || l.stdinDeliveryError != "")) ||
		(l.stdinDeliveryError != "" && l.stdinDeliveryComplete) {
		return false
	}
	return l.invocationLineageValid()
}

func (l processCLILineage) invocationLineageValid() bool {
	if !l.physicalExecutionEntered {
		return l.invocationEvidenceStatus == CLIInvocationNotInspected &&
			l.invocationEvidencePresence == CLIInvocationPresenceUnknown &&
			!l.invocationEvidenceDigest.Valid() && !l.invocationEvidenceValid
	}
	if l.invocationEvidenceStatus == CLIInvocationNotInspected {
		return false
	}
	if !l.invocationEvidenceDigest.Valid() || !l.fixtureOverlayDigest.Valid() ||
		l.workingDirectory == "" || l.cwdPolicy == "" {
		return false
	}
	switch l.invocationEvidenceStatus {
	case CLIInvocationValidated:
		return l.invocationEvidencePresence == CLIInvocationPresencePresent && l.invocationEvidenceValid
	case CLIInvocationAbsent:
		return l.invocationEvidencePresence == CLIInvocationPresenceAbsent && !l.invocationEvidenceValid
	case CLIInvocationReadFailed:
		return (l.invocationEvidencePresence == CLIInvocationPresencePresent ||
			l.invocationEvidencePresence == CLIInvocationPresenceUnknown) && !l.invocationEvidenceValid
	case CLIInvocationNotRegular, CLIInvocationTooLarge, CLIInvocationMalformed,
		CLIInvocationSchemaMismatch, CLIInvocationAttemptMismatch, CLIInvocationArgvMismatch:
		return l.invocationEvidencePresence == CLIInvocationPresencePresent && !l.invocationEvidenceValid
	default:
		return false
	}
}

func buildProcessReceipt(input receiptInput) (ProcessReceipt, error) {
	if !input.cli.valid() {
		return ProcessReceipt{}, &domain.Error{Code: "INVALID_CLI_PROCESS_RECEIPT_LINEAGE"}
	}
	if input.cli.present() &&
		(input.cli.physicalExecutionEntered != input.physical.physicalExecutionEntered ||
			(input.cli.physicalExecutionEntered && !sameProcessStrings(input.logicalArgv, input.cli.declaredLogicalArgv)) ||
			(!input.cli.physicalExecutionEntered && len(input.logicalArgv) != 0) ||
			(input.cli.physicalExecutionEntered &&
				((input.cli.stdinDelivery == "NOT_APPLIED") != !input.physical.started))) {
		return ProcessReceipt{}, &domain.Error{Code: "INVALID_CLI_DECLARED_APPLIED_EXECUTION_SPLIT"}
	}
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
		SchemaVersion                 string   `json:"schema_version"`
		Kind                          string   `json:"kind"`
		AttemptID                     string   `json:"attempt_id"`
		AttemptArtifact               string   `json:"attempt_artifact_digest"`
		WorldDigest                   string   `json:"world_instance_digest"`
		ManifestDigest                string   `json:"materialization_manifest_digest"`
		AttemptRoot                   string   `json:"attempt_root"`
		CandidateRoot                 string   `json:"candidate_root"`
		HomeRoot                      string   `json:"home_root"`
		TemporaryRoot                 string   `json:"temporary_root"`
		XDGConfigRoot                 string   `json:"xdg_config_root"`
		XDGCacheRoot                  string   `json:"xdg_cache_root"`
		XDGDataRoot                   string   `json:"xdg_data_root"`
		XDGStateRoot                  string   `json:"xdg_state_root"`
		StateRoot                     string   `json:"state_root"`
		EvidenceRoot                  string   `json:"evidence_root"`
		MarkerPath                    string   `json:"attempt_marker_path"`
		MarkerBeforeSpawn             bool     `json:"marker_before_spawn"`
		ToolName                      string   `json:"tool_name"`
		ToolPath                      string   `json:"tool_path"`
		ToolVersion                   string   `json:"tool_version"`
		ToolMajor                     int      `json:"tool_major"`
		ToolDigest                    string   `json:"tool_executable_digest"`
		LogicalArgv                   []string `json:"logical_argv"`
		Environment                   []string `json:"sparse_environment"`
		PID                           int      `json:"pid"`
		ProcessGroupID                int      `json:"process_group_id"`
		ExitCode                      int      `json:"exit_code"`
		ExitSignal                    string   `json:"exit_signal"`
		StdoutDigest                  string   `json:"stdout_digest"`
		StderrDigest                  string   `json:"stderr_digest"`
		StdoutObserved                int64    `json:"stdout_observed_bytes"`
		StderrObserved                int64    `json:"stderr_observed_bytes"`
		StdoutOverflow                bool     `json:"stdout_overflow"`
		StderrOverflow                bool     `json:"stderr_overflow"`
		PrimaryControl                string   `json:"primary_control"`
		PreTermProbe                  string   `json:"pre_term_group_probe"`
		TermSent                      bool     `json:"term_sent"`
		KillSent                      bool     `json:"kill_sent"`
		DirectChildWaited             bool     `json:"direct_child_waited"`
		StdoutDrained                 bool     `json:"stdout_drained"`
		StderrDrained                 bool     `json:"stderr_drained"`
		FinalProbeClean               bool     `json:"final_group_probe_clean"`
		TeardownError                 bool     `json:"teardown_error"`
		OrphanRisk                    bool     `json:"orphan_risk"`
		CleanupBoundary               string   `json:"cleanup_boundary"`
		SignalBoundary                string   `json:"process_group_signal_boundary"`
		Arbitration                   string   `json:"terminal_arbitration"`
		SpawnAttempted                bool     `json:"spawn_attempted"`
		Started                       bool     `json:"started"`
		ProcessGroupOwned             bool     `json:"process_group_owned"`
		DiagnosticCode                string   `json:"diagnostic_code"`
		StateHistory                  []string `json:"state_history"`
		CLIExecutionAuthority         string   `json:"cli_execution_authority"`
		CLIPhysicalExecutionEntered   bool     `json:"cli_physical_execution_entered"`
		CLIDeclaredLogicalArgv        []string `json:"cli_declared_logical_argv"`
		CLIExecutionBindingDigest     string   `json:"cli_execution_binding_digest"`
		CLIStimulusDigest             string   `json:"cli_stimulus_digest"`
		CLIExecutionPayloadDigest     string   `json:"cli_execution_payload_digest"`
		CLIFixtureRecipeDigest        string   `json:"cli_fixture_recipe_digest"`
		CLIFixtureOverlayDigest       string   `json:"cli_fixture_overlay_digest"`
		CLICWDPolicy                  string   `json:"cli_cwd_policy"`
		CLIWorkingDirectory           string   `json:"cli_working_directory"`
		CLIStdinPresence              string   `json:"cli_declared_stdin_presence"`
		CLIStdinBytes                 int64    `json:"cli_declared_stdin_bytes"`
		CLIStdinDigest                string   `json:"cli_declared_stdin_digest"`
		CLIStdinDelivery              string   `json:"cli_applied_stdin_delivery"`
		CLIStdinPipeAllocated         bool     `json:"cli_applied_stdin_pipe_allocated"`
		CLIStdinWriterStarted         bool     `json:"cli_applied_stdin_writer_started"`
		CLIStdinHandoffAttempted      bool     `json:"cli_applied_stdin_handoff_attempted"`
		CLIStdinWrittenBytes          int64    `json:"cli_applied_stdin_written_bytes"`
		CLIStdinDeliveryComplete      bool     `json:"cli_applied_stdin_delivery_complete"`
		CLIStdinDeliveryError         string   `json:"cli_applied_stdin_delivery_error_code"`
		CLIStdoutCaptureLimit         int64    `json:"cli_applied_stdout_capture_limit_bytes"`
		CLIStderrCaptureLimit         int64    `json:"cli_applied_stderr_capture_limit_bytes"`
		CLIScheduleOrdinal            int      `json:"cli_schedule_ordinal"`
		CLIScheduleCandidateCount     int      `json:"cli_schedule_candidate_count"`
		CLIScheduleRepetition         int      `json:"cli_schedule_repetition"`
		CLISchedulePresent            bool     `json:"cli_schedule_present"`
		CLIInvocationEvidenceDigest   string   `json:"cli_invocation_evidence_digest"`
		CLIInvocationEvidenceStatus   string   `json:"cli_invocation_evidence_status"`
		CLIInvocationEvidencePresence string   `json:"cli_invocation_evidence_presence"`
		CLIInvocationEvidenceValid    bool     `json:"cli_invocation_evidence_validated"`
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
		StateHistory:                stateStrings(states),
		CLIExecutionAuthority:       input.cli.authority,
		CLIPhysicalExecutionEntered: input.cli.physicalExecutionEntered,
		CLIDeclaredLogicalArgv:      append([]string(nil), input.cli.declaredLogicalArgv...),
		CLIExecutionBindingDigest:   input.cli.executionBindingDigest.String(),
		CLIStimulusDigest:           input.cli.stimulusDigest.String(),
		CLIExecutionPayloadDigest:   input.cli.executionPayloadDigest.String(),
		CLIFixtureRecipeDigest:      input.cli.fixtureRecipeDigest.String(),
		CLIFixtureOverlayDigest:     input.cli.fixtureOverlayDigest.String(),
		CLICWDPolicy:                input.cli.cwdPolicy, CLIWorkingDirectory: input.cli.workingDirectory,
		CLIStdinPresence: input.cli.stdinPresence, CLIStdinBytes: input.cli.stdinBytes,
		CLIStdinDigest: input.cli.stdinDigest.String(), CLIStdinDelivery: input.cli.stdinDelivery,
		CLIStdinPipeAllocated:         input.cli.stdinPipeAllocated,
		CLIStdinWriterStarted:         input.cli.stdinWriterStarted,
		CLIStdinHandoffAttempted:      input.cli.stdinHandoffAttempted,
		CLIStdinWrittenBytes:          input.cli.stdinWritten,
		CLIStdinDeliveryComplete:      input.cli.stdinDeliveryComplete,
		CLIStdinDeliveryError:         input.cli.stdinDeliveryError,
		CLIStdoutCaptureLimit:         input.cli.stdoutCaptureLimit,
		CLIStderrCaptureLimit:         input.cli.stderrCaptureLimit,
		CLIScheduleOrdinal:            input.cli.scheduleOrdinal,
		CLIScheduleCandidateCount:     input.cli.candidateCount,
		CLIScheduleRepetition:         input.cli.scheduleRepetition,
		CLISchedulePresent:            input.cli.present(),
		CLIInvocationEvidenceDigest:   input.cli.invocationEvidenceDigest.String(),
		CLIInvocationEvidenceStatus:   string(input.cli.invocationEvidenceStatus),
		CLIInvocationEvidencePresence: string(input.cli.invocationEvidencePresence),
		CLIInvocationEvidenceValid:    input.cli.invocationEvidenceValid,
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
		executionAuthority: input.cli.authority,
		// MUTATION_ANCHOR: preprocess-control-must-not-forge-physical-entry
		physicalExecutionEntered: input.cli.physicalExecutionEntered,
		declaredLogicalArgv:      append([]string(nil), input.cli.declaredLogicalArgv...),
		executionBindingDigest:   input.cli.executionBindingDigest,
		stimulusDigest:           input.cli.stimulusDigest, executionPayloadDigest: input.cli.executionPayloadDigest,
		fixtureRecipeDigest: input.cli.fixtureRecipeDigest, fixtureOverlayDigest: input.cli.fixtureOverlayDigest,
		cwdPolicy:        input.cli.cwdPolicy,
		workingDirectory: input.cli.workingDirectory, stdinPresence: input.cli.stdinPresence,
		stdinBytes: input.cli.stdinBytes, stdinDigest: input.cli.stdinDigest, stdinDelivery: input.cli.stdinDelivery,
		stdinPipeAllocated:    input.cli.stdinPipeAllocated,
		stdinWriterStarted:    input.cli.stdinWriterStarted,
		stdinHandoffAttempted: input.cli.stdinHandoffAttempted,
		stdinWritten:          input.cli.stdinWritten, stdinDeliveryComplete: input.cli.stdinDeliveryComplete,
		stdinDeliveryError: input.cli.stdinDeliveryError,
		stdoutCaptureLimit: input.cli.stdoutCaptureLimit, stderrCaptureLimit: input.cli.stderrCaptureLimit,
		scheduleOrdinal: input.cli.scheduleOrdinal, scheduleCandidateCount: input.cli.candidateCount,
		scheduleRepetition: input.cli.scheduleRepetition, schedulePresent: input.cli.present(),
		invocationEvidenceDigest:   input.cli.invocationEvidenceDigest,
		invocationEvidenceStatus:   input.cli.invocationEvidenceStatus,
		invocationEvidencePresence: input.cli.invocationEvidencePresence,
		invocationEvidenceValid:    input.cli.invocationEvidenceValid,
	}, nil
}

func sameProcessStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
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
