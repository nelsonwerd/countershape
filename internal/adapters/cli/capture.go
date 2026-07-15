package cli

import (
	"bytes"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/world"
)

const maxCaptureBytes = 16 << 20

type CLIChannel string

const (
	CLIChannelExit   CLIChannel = "exit"
	CLIChannelStdout CLIChannel = "stdout"
	CLIChannelStderr CLIChannel = "stderr"
)

type ChannelState string

const (
	ChannelPresent   ChannelState = "PRESENT"
	ChannelAbsent    ChannelState = "ABSENT"
	ChannelTruncated ChannelState = "TRUNCATED"
)

type CLICapturePolicyConfig = model.CLICapturePolicyConfig
type CLICapturePolicy = model.CLICapturePolicy

func NewCLICapturePolicy(config CLICapturePolicyConfig) (CLICapturePolicy, error) {
	return model.NewCLICapturePolicy(config)
}

type CLICompletionKind string

const (
	CompletionExited   CLICompletionKind = "EXITED"
	CompletionSignaled CLICompletionKind = "SIGNALED"
)

type CLICompletion struct {
	kind   CLICompletionKind
	code   int
	signal string
}

func (c CLICompletion) Kind() CLICompletionKind { return c.kind }
func (c CLICompletion) ExitCode() (int, bool)   { return c.code, c.kind == CompletionExited }
func (c CLICompletion) Signal() (string, bool)  { return c.signal, c.kind == CompletionSignaled }

func (c CLICompletion) valid() bool {
	if c.kind == CompletionExited {
		return c.code >= 0 && c.code <= 255 && c.signal == ""
	}
	return c.kind == CompletionSignaled && c.code == 0 && validProjectionText(c.signal)
}

type CLIChannelCapture struct {
	name           CLIChannel
	state          ChannelState
	bytes          []byte
	retainedDigest domain.Digest
	observedBytes  int64
	absentReason   string
}

func (c CLIChannelCapture) Name() CLIChannel              { return c.name }
func (c CLIChannelCapture) State() ChannelState           { return c.state }
func (c CLIChannelCapture) Bytes() []byte                 { return append([]byte(nil), c.bytes...) }
func (c CLIChannelCapture) RetainedDigest() domain.Digest { return c.retainedDigest }
func (c CLIChannelCapture) ObservedBytes() int64          { return c.observedBytes }
func (c CLIChannelCapture) AbsentReason() (string, bool) {
	return c.absentReason, c.state == ChannelAbsent
}

type CLIFixtureInvocationReceipt struct {
	presence     Presence
	digest       domain.Digest
	status       world.CLIInvocationEvidenceStatus
	filePresence world.CLIInvocationEvidencePresence
	validated    bool
	reason       string
}

func (r CLIFixtureInvocationReceipt) Present() bool                             { return r.presence == PresencePresent }
func (r CLIFixtureInvocationReceipt) Digest() (domain.Digest, bool)             { return r.digest, r.Present() }
func (r CLIFixtureInvocationReceipt) Status() world.CLIInvocationEvidenceStatus { return r.status }
func (r CLIFixtureInvocationReceipt) FilePresence() world.CLIInvocationEvidencePresence {
	return r.filePresence
}
func (r CLIFixtureInvocationReceipt) FilePresent() bool {
	return r.filePresence == world.CLIInvocationPresencePresent
}
func (r CLIFixtureInvocationReceipt) Validated() bool { return r.validated }
func (r CLIFixtureInvocationReceipt) Reason() (string, bool) {
	return r.reason, r.presence == PresenceAbsent
}

func (r CLIFixtureInvocationReceipt) valid() bool {
	if r.presence == PresencePresent {
		return r.digest.Valid() && r.status != "" && r.reason == "" &&
			(r.validated == (r.status == world.CLIInvocationValidated)) &&
			invocationStatusPresenceValid(r.status, r.filePresence)
	}
	return r.presence == PresenceAbsent && !r.digest.Valid() &&
		(r.status == "" || r.status == world.CLIInvocationNotInspected) &&
		r.filePresence == world.CLIInvocationPresenceUnknown &&
		!r.validated && validProjectionText(r.reason)
}

func invocationStatusPresenceValid(
	status world.CLIInvocationEvidenceStatus,
	presence world.CLIInvocationEvidencePresence,
) bool {
	switch status {
	case world.CLIInvocationAbsent:
		return presence == world.CLIInvocationPresenceAbsent
	case world.CLIInvocationReadFailed:
		return presence == world.CLIInvocationPresencePresent || presence == world.CLIInvocationPresenceUnknown
	case world.CLIInvocationValidated, world.CLIInvocationNotRegular, world.CLIInvocationTooLarge,
		world.CLIInvocationMalformed, world.CLIInvocationSchemaMismatch,
		world.CLIInvocationAttemptMismatch, world.CLIInvocationArgvMismatch:
		return presence == world.CLIInvocationPresencePresent
	default:
		return false
	}
}

// CLICapturedObservation is durable post-policy evidence. Its constructor is
// the world-result adapter below; no public digest/data pairing constructor
// exists. Controls and fixture invocation instrumentation stay outside the
// selected behavior projection.
type CLICapturedObservation struct {
	digest                            domain.Digest
	canonicalBytes                    []byte
	worldDigest                       domain.Digest
	planDigest                        domain.Digest
	candidateKey                      domain.CandidateExecutionKey
	stimulusDigest                    domain.Digest
	executionPayloadDigest            domain.Digest
	attemptDigest                     domain.Digest
	trialIndex                        int
	capturePolicyDigest               domain.Digest
	adapterProjectionDefinitionDigest domain.Digest
	projectionDefinitionDigest        domain.Digest
	processReceiptDigest              domain.Digest
	fixtureOverlayDigest              domain.Digest
	fixtureOverlayPresent             bool
	invocationReceipt                 CLIFixtureInvocationReceipt
	completion                        CLICompletion
	completionPresent                 bool
	stdout                            CLIChannelCapture
	stderr                            CLIChannelCapture
	controls                          []domain.ControlReason
	markerBeforeSpawn                 bool
	spawnAttempted                    bool
	started                           bool
}

type channelCaptureIdentity struct {
	Name           string `json:"name"`
	State          string `json:"state"`
	CapturedBytes  int64  `json:"captured_bytes"`
	RetainedDigest string `json:"retained_digest"`
	ObservedBytes  int64  `json:"observed_bytes"`
	AbsentReason   string `json:"absent_reason"`
}

type completionCaptureIdentity struct {
	Presence string `json:"presence"`
	Kind     string `json:"kind"`
	Code     int    `json:"code"`
	Signal   string `json:"signal"`
}

type invocationReceiptIdentity struct {
	Presence     string `json:"presence"`
	Digest       string `json:"digest"`
	Status       string `json:"status"`
	FilePresence string `json:"file_presence"`
	FilePresent  bool   `json:"file_present"`
	Validated    bool   `json:"validated"`
	Reason       string `json:"reason"`
}

type capturedObservationIdentity struct {
	SchemaVersion                     string                    `json:"schema_version"`
	Kind                              string                    `json:"kind"`
	WorldInstanceDigest               string                    `json:"world_instance_digest"`
	WorldPlanDigest                   string                    `json:"world_plan_digest"`
	CandidateExecutionKey             string                    `json:"candidate_execution_key"`
	StimulusDigest                    string                    `json:"stimulus_digest"`
	ExecutionPayloadDigest            string                    `json:"execution_payload_digest"`
	TrialIndex                        int                       `json:"trial_index"`
	AttemptArtifactDigest             string                    `json:"attempt_artifact_digest"`
	CapturePolicyDigest               string                    `json:"capture_policy_digest"`
	AdapterProjectionDefinitionDigest string                    `json:"cli_adapter_projection_definition_digest"`
	ProjectionDefinitionDigest        string                    `json:"projection_definition_digest"`
	ProcessReceiptDigest              string                    `json:"process_receipt_digest"`
	FixtureOverlayReceipt             invocationReceiptIdentity `json:"fixture_overlay_receipt"`
	FixtureInvocationReceipt          invocationReceiptIdentity `json:"fixture_invocation_receipt"`
	Completion                        completionCaptureIdentity `json:"completion"`
	Channels                          []channelCaptureIdentity  `json:"channels"`
	Controls                          []string                  `json:"controls"`
	MarkerBeforeSpawn                 bool                      `json:"marker_before_spawn"`
	SpawnAttempted                    bool                      `json:"spawn_attempted"`
	Started                           bool                      `json:"started"`
	UndetectedControlFailureExcluded  bool                      `json:"undetected_control_failure_excluded"`
}

// AdaptWorldResult performs only a pure evidence transformation. The impure
// world edge has already executed and receipted the exact opaque binding.
func AdaptWorldResult(
	result world.Result,
	binding model.CLIExecutionBinding,
	trialIndex int,
) (CLICapturedObservation, error) {
	policy := binding.CapturePolicy()
	if !binding.Valid() || !policy.Valid() || trialIndex < 0 {
		return CLICapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "capture input authority is incomplete")
	}
	instance := result.World()
	attempt := result.FinalizedAttempt()
	process := result.Process()
	planBudgets := binding.Plan().Budgets()
	controls, err := exactControls(attempt, process)
	if err != nil {
		return CLICapturedObservation{}, err
	}
	if !instance.Digest().Valid() || !attempt.ArtifactDigest().Valid() || !process.Digest().Valid() ||
		instance.PlanDigest() != binding.PlanDigest() || instance.StimulusDigest() != binding.StimulusDigest() ||
		instance.CapturePolicyDigest() != policy.Digest() || instance.ScheduleOrdinal() != trialIndex ||
		instance.AttemptArtifactDigest() != attempt.ArtifactDigest() || process.WorldDigest() != instance.Digest() ||
		process.AttemptArtifactDigest() != attempt.ArtifactDigest() ||
		process.ExecutionBindingDigest() != binding.Digest() ||
		process.StimulusDigest() != binding.StimulusDigest() ||
		process.ExecutionPayloadDigest() != binding.ExecutionPayloadDigest() ||
		policy.StdoutBytes() != planBudgets.StdoutBytes || policy.StderrBytes() != planBudgets.StderrBytes ||
		!equalStrings(process.DeclaredLogicalArgv(), binding.LogicalArgv()) {
		return CLICapturedObservation{}, refuse(CodeCaptureLineageMismatch, "world, process, binding, or capture policy lineage differs")
	}
	if !appliedCaptureLimitsMatch(
		process.PhysicalExecutionEntered(), process.StdoutCaptureLimit(), process.StderrCaptureLimit(), policy,
	) || !processStdinMatchesBinding(process, binding.Stdin()) {
		return CLICapturedObservation{}, refuse(CodeCaptureLineageMismatch, "declared and physically applied CLI evidence differ")
	}
	if process.PhysicalExecutionEntered() {
		if !equalStrings(process.LogicalArgv(), binding.LogicalArgv()) {
			return CLICapturedObservation{}, refuse(CodeCaptureLineageMismatch, "executed argv differs from the binding declaration")
		}
	} else if len(controls) == 0 || len(process.LogicalArgv()) != 0 || process.SpawnAttempted() || process.Started() {
		return CLICapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "NOT_ENTERED physical execution lacks a pre-process control")
	}
	if ordinal, present := process.ScheduleOrdinal(); !present || ordinal != trialIndex {
		return CLICapturedObservation{}, refuse(CodeCaptureLineageMismatch, "process receipt schedule ordinal differs")
	}
	stdout, err := adaptChannel(CLIChannelStdout, process.Stdout(), process.StdoutObservedBytes(), process.StdoutOverflow(), policy.StdoutBytes(), process.Started())
	if err != nil {
		return CLICapturedObservation{}, err
	}
	stderr, err := adaptChannel(CLIChannelStderr, process.Stderr(), process.StderrObservedBytes(), process.StderrOverflow(), policy.StderrBytes(), process.Started())
	if err != nil {
		return CLICapturedObservation{}, err
	}
	primary, primaryPresent := attempt.PrimaryControl()
	if !truncatedChannelsMatchPrimary(stdout, stderr, primary, primaryPresent) {
		return CLICapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "truncated channel lacks an admissible owner primary control")
	}
	completion, completionPresent, err := adaptCompletion(process)
	if err != nil {
		return CLICapturedObservation{}, err
	}
	invocationReceipt, err := adaptInvocationEvidence(result, binding, process)
	if err != nil {
		return CLICapturedObservation{}, err
	}
	fixtureOverlayDigest, fixtureOverlayPresent, err := adaptFixtureOverlay(result, binding, process)
	if err != nil {
		return CLICapturedObservation{}, err
	}
	// MUTATION_ANCHOR: nonzero-exit-is-behavior
	if len(controls) == 0 && (!process.MarkerExistedBeforeSpawn() || !process.SpawnAttempted() || !process.Started() ||
		!process.ProcessGroupOwned() || !process.DirectChildWaited() || !process.DrainsComplete() ||
		!process.FinalGroupProbeClean() || !completionPresent || !fixtureOverlayPresent) {
		return CLICapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "eligible capture lacks lifecycle or fixture invocation evidence")
	}

	observation := CLICapturedObservation{
		worldDigest: instance.Digest(), planDigest: instance.PlanDigest(), candidateKey: instance.CandidateKey(),
		stimulusDigest: instance.StimulusDigest(), executionPayloadDigest: binding.ExecutionPayloadDigest(),
		attemptDigest: attempt.ArtifactDigest(), trialIndex: trialIndex, capturePolicyDigest: policy.Digest(),
		adapterProjectionDefinitionDigest: binding.AdapterProjectionDefinitionDigest(),
		projectionDefinitionDigest:        instance.ProjectionDefinitionDigest(),
		processReceiptDigest:              process.Digest(), fixtureOverlayDigest: fixtureOverlayDigest,
		fixtureOverlayPresent: fixtureOverlayPresent, invocationReceipt: invocationReceipt,
		completion: completion, completionPresent: completionPresent,
		stdout: stdout, stderr: stderr, controls: controls,
		markerBeforeSpawn: process.MarkerExistedBeforeSpawn(), spawnAttempted: process.SpawnAttempted(), started: process.Started(),
	}
	identity := observation.identity()
	digest, canonicalBytes, err := cliDigestTyped("CLICapturedObservation", identity)
	if err != nil {
		return CLICapturedObservation{}, err
	}
	observation.digest = digest
	observation.canonicalBytes = canonicalBytes
	return observation, nil
}

func truncatedChannelsMatchPrimary(
	stdout, stderr CLIChannelCapture,
	primary domain.ControlReason,
	primaryPresent bool,
) bool {
	if stdout.state != ChannelTruncated && stderr.state != ChannelTruncated {
		return true
	}
	// MUTATION_ANCHOR: late-overflow-must-retain-earlier-primary
	if !primaryPresent {
		return false
	}
	switch primary {
	case domain.ControlOutputLimit, domain.ControlTimeout, domain.ControlCancelled,
		domain.ControlProbeTransportError, domain.ControlStartError:
		return true
	default:
		return false
	}
}

func appliedCaptureLimitsMatch(
	physicalExecutionEntered bool,
	stdoutApplied, stderrApplied int64,
	policy CLICapturePolicy,
) bool {
	// MUTATION_ANCHOR: applied-capture-limits-must-match-policy
	if !policy.Valid() {
		return false
	}
	if !physicalExecutionEntered {
		return stdoutApplied == 0 && stderrApplied == 0
	}
	return stdoutApplied == policy.StdoutBytes() && stderrApplied == policy.StderrBytes()
}

type processStdinEvidence struct {
	presence         string
	declaredBytes    int64
	digest           domain.Digest
	delivery         string
	pipeAllocated    bool
	writerStarted    bool
	handoffAttempted bool
	writtenBytes     int64
	complete         bool
	errorCode        string
	started          bool
	physicalEntered  bool
}

func processStdinMatchesBinding(process world.ProcessReceipt, stdin model.CLIStdin) bool {
	return stdinEvidenceMatchesBinding(processStdinEvidence{
		presence: process.StdinPresence(), declaredBytes: process.StdinBytes(), digest: process.StdinDigest(),
		delivery: process.StdinDelivery(), pipeAllocated: process.StdinPipeAllocated(),
		writerStarted: process.StdinWriterStarted(), handoffAttempted: process.StdinHandoffAttempted(),
		writtenBytes: process.StdinWrittenBytes(), complete: process.StdinDeliveryComplete(),
		errorCode: process.StdinDeliveryErrorCode(), started: process.Started(),
		physicalEntered: process.PhysicalExecutionEntered(),
	}, stdin)
}

func stdinEvidenceMatchesBinding(evidence processStdinEvidence, stdin model.CLIStdin) bool {
	digest, _, err := cliDigestBytes("CLIStdinBytes", stdin.Bytes())
	if err != nil || evidence.presence != string(stdin.Presence()) ||
		evidence.declaredBytes != int64(len(stdin.Bytes())) || evidence.digest != digest ||
		evidence.writtenBytes < 0 || evidence.writtenBytes > evidence.declaredBytes {
		return false
	}
	if !evidence.physicalEntered {
		return evidence.delivery == "NOT_APPLIED" && !evidence.pipeAllocated &&
			!evidence.writerStarted && !evidence.handoffAttempted && evidence.writtenBytes == 0 &&
			!evidence.complete && evidence.errorCode == "" && !evidence.started
	}
	if !evidence.started {
		return evidence.delivery == "NOT_APPLIED" &&
			!evidence.writerStarted && !evidence.handoffAttempted && evidence.writtenBytes == 0 &&
			!evidence.complete && evidence.errorCode == "" &&
			(stdin.Present() || !evidence.pipeAllocated)
	}
	if !stdin.Present() {
		return evidence.delivery == "ABSENT_NULL_DEVICE" && !evidence.pipeAllocated &&
			!evidence.writerStarted && !evidence.handoffAttempted && evidence.writtenBytes == 0 &&
			!evidence.complete && evidence.errorCode == ""
	}
	if evidence.delivery != "PRESENT_EXPLICIT_PIPE_WRITER" ||
		evidence.writerStarted != evidence.handoffAttempted || (evidence.writerStarted && !evidence.pipeAllocated) {
		return false
	}
	if !evidence.pipeAllocated || !evidence.writerStarted || !evidence.handoffAttempted {
		return false
	}
	if evidence.complete {
		return evidence.writtenBytes == evidence.declaredBytes && evidence.errorCode == ""
	}
	return evidence.errorCode != ""
}

func adaptInvocationEvidence(result world.Result, binding model.CLIExecutionBinding, process world.ProcessReceipt) (CLIFixtureInvocationReceipt, error) {
	receipt, present := result.CLIInvocationEvidence()
	if !present {
		if process.InvocationEvidenceDigest().Valid() ||
			(process.InvocationEvidenceStatus() != "" && process.InvocationEvidenceStatus() != world.CLIInvocationNotInspected) ||
			process.InvocationEvidencePresence() != world.CLIInvocationPresenceUnknown ||
			process.InvocationEvidencePresent() || process.InvocationEvidenceValidated() {
			return CLIFixtureInvocationReceipt{}, refuse(CodeCaptureEvidenceInvalid, "process refers to unavailable invocation evidence")
		}
		return CLIFixtureInvocationReceipt{
			presence: PresenceAbsent, status: process.InvocationEvidenceStatus(),
			filePresence: world.CLIInvocationPresenceUnknown,
			reason:       "INVOCATION_EVIDENCE_NOT_AVAILABLE",
		}, nil
	}
	if !receipt.Valid() || receipt.ExecutionBindingDigest() != binding.Digest() ||
		process.InvocationEvidenceDigest() != receipt.Digest() || process.InvocationEvidenceStatus() != receipt.Status() ||
		process.InvocationEvidencePresence() != receipt.Presence() ||
		process.InvocationEvidencePresent() != receipt.Present() ||
		process.InvocationEvidenceValidated() != (receipt.Status() == world.CLIInvocationValidated) ||
		receipt.ExpectedAttemptID() != process.AttemptID() ||
		!equalStrings(receipt.ExpectedLogicalArgv(), binding.LogicalArgv()) {
		return CLIFixtureInvocationReceipt{}, refuse(CodeCaptureEvidenceInvalid, "invocation evidence and process receipt lineage differ")
	}
	return CLIFixtureInvocationReceipt{
		presence: PresencePresent, digest: receipt.Digest(), status: receipt.Status(),
		filePresence: receipt.Presence(), validated: receipt.Status() == world.CLIInvocationValidated,
	}, nil
}

func adaptFixtureOverlay(result world.Result, binding model.CLIExecutionBinding, process world.ProcessReceipt) (domain.Digest, bool, error) {
	receipt, present := result.CLIFixtureOverlay()
	if !present {
		if process.FixtureOverlayDigest().Valid() {
			return "", false, refuse(CodeCaptureEvidenceInvalid, "process refers to unavailable fixture overlay receipt")
		}
		return "", false, nil
	}
	if !receipt.Valid() || receipt.ExecutionBindingDigest() != binding.Digest() ||
		process.FixtureOverlayDigest() != receipt.Digest() {
		return "", false, refuse(CodeCaptureEvidenceInvalid, "fixture overlay and process receipt lineage differ")
	}
	return receipt.Digest(), true, nil
}

func adaptChannel(name CLIChannel, captured []byte, observed int64, overflow bool, limit int64, started bool) (CLIChannelCapture, error) {
	digest, _, err := cliDigestBytes("CLI"+stringsTitle(name)+"CapturedBytes", captured)
	if err != nil {
		return CLIChannelCapture{}, err
	}
	if !started {
		if len(captured) != 0 || observed != 0 || overflow {
			return CLIChannelCapture{}, refuse(CodeCaptureEvidenceInvalid, "unstarted process carries channel bytes")
		}
		return CLIChannelCapture{name: name, state: ChannelAbsent, bytes: []byte{}, retainedDigest: digest, absentReason: "PROCESS_NOT_STARTED"}, nil
	}
	if observed < int64(len(captured)) || int64(len(captured)) > limit {
		return CLIChannelCapture{}, refuse(CodeCaptureEvidenceInvalid, "channel counts violate the capture policy")
	}
	if overflow {
		if int64(len(captured)) != limit || observed <= limit {
			return CLIChannelCapture{}, refuse(CodeCaptureEvidenceInvalid, "truncation does not cross the exact channel cap")
		}
		return CLIChannelCapture{name: name, state: ChannelTruncated, bytes: append([]byte(nil), captured...), retainedDigest: digest, observedBytes: observed}, nil
	}
	if observed != int64(len(captured)) {
		return CLIChannelCapture{}, refuse(CodeCaptureEvidenceInvalid, "complete channel count differs from retained bytes")
	}
	return CLIChannelCapture{name: name, state: ChannelPresent, bytes: append([]byte(nil), captured...), retainedDigest: digest, observedBytes: observed}, nil
}

func adaptCompletion(process world.ProcessReceipt) (CLICompletion, bool, error) {
	if !process.Started() || !process.DirectChildWaited() {
		return CLICompletion{}, false, nil
	}
	if signal := process.ExitSignal(); signal != "" {
		completion := CLICompletion{kind: CompletionSignaled, signal: signal}
		if !completion.valid() {
			return CLICompletion{}, false, refuse(CodeCaptureEvidenceInvalid, "process signal is invalid")
		}
		return completion, true, nil
	}
	completion := CLICompletion{kind: CompletionExited, code: process.ExitCode()}
	if !completion.valid() {
		return CLICompletion{}, false, refuse(CodeCaptureEvidenceInvalid, "process exit code is invalid")
	}
	return completion, true, nil
}

func exactControls(attempt domain.FinalizedAttempt, process world.ProcessReceipt) ([]domain.ControlReason, error) {
	result := make([]domain.ControlReason, 0, 3)
	attemptPrimary, hasAttemptPrimary := attempt.PrimaryControl()
	processPrimary, hasProcessPrimary := process.PrimaryControl()
	if hasAttemptPrimary != hasProcessPrimary || (hasAttemptPrimary && attemptPrimary != processPrimary) {
		return nil, refuse(CodeCaptureEvidenceInvalid, "process and finalized-attempt primary controls differ")
	}
	if hasAttemptPrimary {
		result = append(result, attemptPrimary)
	}
	result = append(result, attempt.TeardownControls()...)
	if process.OrphanRisk() != containsReason(result, domain.ControlOrphanRisk) ||
		process.TeardownError() != containsReason(result, domain.ControlTeardownError) {
		return nil, refuse(CodeCaptureEvidenceInvalid, "process and finalized-attempt teardown controls differ")
	}
	return result, nil
}

func containsReason(values []domain.ControlReason, target domain.ControlReason) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (o CLICapturedObservation) identity() capturedObservationIdentity {
	controls := make([]string, len(o.controls))
	for index, control := range o.controls {
		controls[index] = string(control)
	}
	completion := completionCaptureIdentity{Presence: string(PresenceAbsent)}
	if o.completionPresent {
		completion = completionCaptureIdentity{
			Presence: string(PresencePresent), Kind: string(o.completion.kind),
			Code: o.completion.code, Signal: o.completion.signal,
		}
	}
	invocation := invocationReceiptIdentity{
		Presence: string(o.invocationReceipt.presence), Status: string(o.invocationReceipt.status),
		FilePresence: string(o.invocationReceipt.filePresence), FilePresent: o.invocationReceipt.FilePresent(),
		Validated: o.invocationReceipt.validated, Reason: o.invocationReceipt.reason,
	}
	if o.invocationReceipt.Present() {
		invocation.Digest = o.invocationReceipt.digest.String()
	}
	fixtureOverlay := invocationReceiptIdentity{Presence: string(PresenceAbsent), Reason: "NOT_AVAILABLE_BEFORE_FIXTURE_MATERIALIZATION"}
	if o.fixtureOverlayPresent {
		fixtureOverlay = invocationReceiptIdentity{Presence: string(PresencePresent), Digest: o.fixtureOverlayDigest.String(), Validated: true}
	}
	return capturedObservationIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLICapturedObservation",
		WorldInstanceDigest: o.worldDigest.String(), WorldPlanDigest: o.planDigest.String(),
		CandidateExecutionKey: o.candidateKey.String(), StimulusDigest: o.stimulusDigest.String(),
		ExecutionPayloadDigest: o.executionPayloadDigest.String(), TrialIndex: o.trialIndex,
		AttemptArtifactDigest: o.attemptDigest.String(), CapturePolicyDigest: o.capturePolicyDigest.String(),
		AdapterProjectionDefinitionDigest: o.adapterProjectionDefinitionDigest.String(),
		ProjectionDefinitionDigest:        o.projectionDefinitionDigest.String(),
		ProcessReceiptDigest:              o.processReceiptDigest.String(), FixtureOverlayReceipt: fixtureOverlay,
		FixtureInvocationReceipt: invocation,
		Completion:               completion,
		// MUTATION_ANCHOR: stdout-stderr-distinct
		Channels: []channelCaptureIdentity{channelIdentity(o.stdout), channelIdentity(o.stderr)},
		Controls: controls, MarkerBeforeSpawn: o.markerBeforeSpawn, SpawnAttempted: o.spawnAttempted,
		Started: o.started, UndetectedControlFailureExcluded: false,
	}
}

func channelIdentity(channel CLIChannelCapture) channelCaptureIdentity {
	return channelCaptureIdentity{
		Name: string(channel.name), State: string(channel.state),
		CapturedBytes:  int64(len(channel.bytes)),
		RetainedDigest: channel.retainedDigest.String(), ObservedBytes: channel.observedBytes,
		AbsentReason: channel.absentReason,
	}
}

func (o CLICapturedObservation) Valid() bool {
	if !o.digest.Valid() || !o.worldDigest.Valid() || !o.planDigest.Valid() || !o.candidateKey.Valid() ||
		!o.stimulusDigest.Valid() || !o.executionPayloadDigest.Valid() || !o.attemptDigest.Valid() ||
		!o.capturePolicyDigest.Valid() || !o.adapterProjectionDefinitionDigest.Valid() ||
		!o.projectionDefinitionDigest.Valid() ||
		!o.processReceiptDigest.Valid() || (o.fixtureOverlayPresent != o.fixtureOverlayDigest.Valid()) ||
		o.trialIndex < 0 || !o.invocationReceipt.valid() || !o.stdout.valid() || !o.stderr.valid() ||
		(o.completionPresent && !o.completion.valid()) {
		return false
	}
	digest, canonicalBytes, err := cliDigestTyped("CLICapturedObservation", o.identity())
	return err == nil && digest == o.digest && bytes.Equal(canonicalBytes, o.canonicalBytes)
}

func (c CLIChannelCapture) valid() bool {
	if c.name != CLIChannelStdout && c.name != CLIChannelStderr {
		return false
	}
	digest, _, err := cliDigestBytes("CLI"+stringsTitle(c.name)+"CapturedBytes", c.bytes)
	if err != nil || digest != c.retainedDigest {
		return false
	}
	switch c.state {
	case ChannelAbsent:
		return len(c.bytes) == 0 && c.observedBytes == 0 && validProjectionText(c.absentReason)
	case ChannelPresent:
		return c.observedBytes == int64(len(c.bytes)) && c.absentReason == ""
	case ChannelTruncated:
		return len(c.bytes) > 0 && c.observedBytes > int64(len(c.bytes)) && c.absentReason == ""
	default:
		return false
	}
}

func (o CLICapturedObservation) Digest() domain.Digest { return o.digest }
func (o CLICapturedObservation) CanonicalBytes() []byte {
	return append([]byte(nil), o.canonicalBytes...)
}
func (o CLICapturedObservation) WorldDigest() domain.Digest                 { return o.worldDigest }
func (o CLICapturedObservation) PlanDigest() domain.Digest                  { return o.planDigest }
func (o CLICapturedObservation) CandidateKey() domain.CandidateExecutionKey { return o.candidateKey }
func (o CLICapturedObservation) StimulusDigest() domain.Digest              { return o.stimulusDigest }
func (o CLICapturedObservation) ExecutionPayloadDigest() domain.Digest {
	return o.executionPayloadDigest
}
func (o CLICapturedObservation) AttemptDigest() domain.Digest       { return o.attemptDigest }
func (o CLICapturedObservation) TrialIndex() int                    { return o.trialIndex }
func (o CLICapturedObservation) CapturePolicyDigest() domain.Digest { return o.capturePolicyDigest }
func (o CLICapturedObservation) ProjectionDefinitionDigest() domain.Digest {
	return o.projectionDefinitionDigest
}
func (o CLICapturedObservation) ProcessReceiptDigest() domain.Digest { return o.processReceiptDigest }
func (o CLICapturedObservation) FixtureOverlayDigest() (domain.Digest, bool) {
	return o.fixtureOverlayDigest, o.fixtureOverlayPresent
}
func (o CLICapturedObservation) InvocationReceipt() CLIFixtureInvocationReceipt {
	return o.invocationReceipt
}
func (o CLICapturedObservation) Completion() (CLICompletion, bool) {
	return o.completion, o.completionPresent
}
func (o CLICapturedObservation) Stdout() CLIChannelCapture { return cloneChannel(o.stdout) }
func (o CLICapturedObservation) Stderr() CLIChannelCapture { return cloneChannel(o.stderr) }
func (o CLICapturedObservation) Controls() []domain.ControlReason {
	return append([]domain.ControlReason(nil), o.controls...)
}

func (o CLICapturedObservation) ProjectionEligible() bool {
	return o.Valid() && o.projectionEligibilityFactsPresent()
}

func (o CLICapturedObservation) projectionEligibilityFactsPresent() bool {
	return len(o.controls) == 0 && o.completionPresent &&
		o.stdout.state == ChannelPresent && o.stderr.state == ChannelPresent &&
		o.fixtureOverlayPresent && o.invocationReceipt.Validated()
}

func cloneChannel(value CLIChannelCapture) CLIChannelCapture {
	value.bytes = append([]byte(nil), value.bytes...)
	return value
}

func cliDigestTyped(kind string, value any) (domain.Digest, []byte, error) {
	digest, canonicalBytes, err := canon.DigestTyped(kind, value)
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return "", nil, err
	}
	return parsed, append([]byte(nil), canonicalBytes...), nil
}

func cliDigestBytes(kind string, value []byte) (domain.Digest, []byte, error) {
	digest, err := canon.DigestBytes(kind, value)
	if err != nil {
		return "", nil, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	return parsed, append([]byte(nil), value...), err
}

func equalStrings(left, right []string) bool {
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

func validProjectionText(value string) bool {
	return value != "" && len(value) <= 1024 && utf8.ValidString(value)
}

func stringsTitle(channel CLIChannel) string {
	if channel == CLIChannelStdout {
		return "Stdout"
	}
	return "Stderr"
}
