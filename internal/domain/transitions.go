package domain

import "fmt"

// U1 intentionally exposes no generic artifact stage/digest API. Future units
// introduce successor types only beside the semantic evidence that guards each
// edge. A decorative stage-specific digest chain would still permit false
// divergence, confirmation, ruling, and contract authority.

type AttemptPurpose string

const (
	AttemptDiscovery    AttemptPurpose = "DISCOVERY"
	AttemptReduction    AttemptPurpose = "REDUCTION"
	AttemptFinalSweep   AttemptPurpose = "FINAL_SWEEP"
	AttemptConfirmation AttemptPurpose = "CONFIRMATION"
	AttemptConformance  AttemptPurpose = "CONFORMANCE"
)

func (p AttemptPurpose) Valid() bool {
	switch p {
	case AttemptDiscovery, AttemptReduction, AttemptFinalSweep, AttemptConfirmation, AttemptConformance:
		return true
	default:
		return false
	}
}

type AttemptState string

const (
	AttemptAllocated     AttemptState = "ALLOCATED"
	AttemptMaterializing AttemptState = "MATERIALIZING"
	AttemptStarting      AttemptState = "STARTING"
	AttemptReady         AttemptState = "READY"
	AttemptProbing       AttemptState = "PROBING"
	AttemptCapturing     AttemptState = "CAPTURING"
	AttemptTearingDown   AttemptState = "TEARING_DOWN"
	AttemptFinalized     AttemptState = "FINALIZED"
)

type ControlReason string

const (
	ControlMaterializationError ControlReason = "MATERIALIZATION_ERROR"
	ControlSetupError           ControlReason = "SETUP_ERROR"
	ControlStartError           ControlReason = "START_ERROR"
	ControlReadinessError       ControlReason = "READINESS_ERROR"
	ControlProbeTransportError  ControlReason = "PROBE_TRANSPORT_ERROR"
	ControlTimeout              ControlReason = "TIMEOUT"
	ControlCancelled            ControlReason = "CANCELLED"
	ControlOutputLimit          ControlReason = "OUTPUT_LIMIT"
	ControlProjectionRejected   ControlReason = "PROJECTION_REJECTED"
	ControlOrphanRisk           ControlReason = "ORPHAN_RISK"
	ControlTeardownError        ControlReason = "TEARDOWN_ERROR"
	ControlEnvelopeRejected     ControlReason = "ENVELOPE_REJECTED"
	ControlUnsupportedGitMode   ControlReason = "UNSUPPORTED_GIT_MODE"
	ControlMissingObject        ControlReason = "MISSING_OBJECT"
	ControlBudgetExhausted      ControlReason = "BUDGET_EXHAUSTED"
)

func (r ControlReason) Valid() bool {
	switch r {
	case ControlMaterializationError, ControlSetupError, ControlStartError,
		ControlReadinessError, ControlProbeTransportError, ControlTimeout,
		ControlCancelled, ControlOutputLimit, ControlProjectionRejected,
		ControlOrphanRisk, ControlTeardownError, ControlEnvelopeRejected,
		ControlUnsupportedGitMode, ControlMissingObject, ControlBudgetExhausted:
		return true
	default:
		return false
	}
}

type Attempt struct {
	id               string
	artifactDigest   Digest
	purpose          AttemptPurpose
	state            AttemptState
	primaryControl   *ControlReason
	teardownControls []ControlReason
	stateHistory     []AttemptState
}

func NewAttempt(id string, artifactDigest Digest, purpose AttemptPurpose) (Attempt, error) {
	if id == "" || !artifactDigest.Valid() || !purpose.Valid() {
		return Attempt{}, refuse(ErrInvalidAttemptTransition, "attempt identity is incomplete")
	}
	return Attempt{
		id:             id,
		artifactDigest: artifactDigest,
		purpose:        purpose,
		state:          AttemptAllocated,
		stateHistory:   []AttemptState{AttemptAllocated},
	}, nil
}

func (a Attempt) Advance(next AttemptState) (Attempt, error) {
	legal := map[AttemptState]AttemptState{
		AttemptAllocated:     AttemptMaterializing,
		AttemptMaterializing: AttemptStarting,
		AttemptStarting:      AttemptReady,
		AttemptReady:         AttemptProbing,
		AttemptProbing:       AttemptCapturing,
		AttemptCapturing:     AttemptTearingDown,
		AttemptTearingDown:   AttemptFinalized,
	}
	if a.primaryControl != nil && next != AttemptTearingDown && next != AttemptFinalized {
		return Attempt{}, refuse(ErrInvalidAttemptTransition, "control failure must route through teardown")
	}
	if legal[a.state] != next {
		return Attempt{}, refuse(ErrInvalidAttemptTransition, fmt.Sprintf("%s -> %s", a.state, next))
	}
	result := a.clone()
	result.state = next
	result.stateHistory = append(result.stateHistory, next)
	return result, nil
}

// Fail records the primary control result. Teardown and orphan findings are
// separate controls so cleanup failure cannot overwrite the original cause.
func (a Attempt) Fail(reason ControlReason) (Attempt, error) {
	if !reason.Valid() || reason == ControlTeardownError || reason == ControlOrphanRisk ||
		a.primaryControl != nil || a.state == AttemptFinalized || a.state == AttemptTearingDown {
		return Attempt{}, refuse(ErrInvalidAttemptTransition, "primary failure cannot be attached in this state")
	}
	result := a.clone()
	value := reason
	result.primaryControl = &value
	if result.state == AttemptAllocated {
		result.state = AttemptFinalized
		result.stateHistory = append(result.stateHistory, AttemptFinalized)
		return result, nil
	}
	result.state = AttemptTearingDown
	result.stateHistory = append(result.stateHistory, AttemptTearingDown)
	return result, nil
}

func (a Attempt) RecordTeardownControl(reason ControlReason) (Attempt, error) {
	if a.state != AttemptTearingDown || (reason != ControlTeardownError && reason != ControlOrphanRisk) {
		return Attempt{}, refuse(ErrInvalidAttemptTransition, "invalid teardown control")
	}
	for _, existing := range a.teardownControls {
		if existing == reason {
			return Attempt{}, refuse(ErrInvalidAttemptTransition, "duplicate teardown control")
		}
	}
	result := a.clone()
	result.teardownControls = append(result.teardownControls, reason)
	return result, nil
}

func (a Attempt) clone() Attempt {
	result := a
	result.stateHistory = append([]AttemptState(nil), a.stateHistory...)
	result.teardownControls = append([]ControlReason(nil), a.teardownControls...)
	if a.primaryControl != nil {
		value := *a.primaryControl
		result.primaryControl = &value
	}
	return result
}

// FinalizedAttempt is sealed structural authority that a complete lifecycle
// reached FINALIZED. It retains both primary and cleanup controls.
type FinalizedAttempt struct {
	artifactDigest   Digest
	purpose          AttemptPurpose
	primaryControl   *ControlReason
	teardownControls []ControlReason
}

func (a Attempt) FinalizedEvidence() (FinalizedAttempt, error) {
	if a.state != AttemptFinalized || !a.artifactDigest.Valid() || !a.purpose.Valid() {
		return FinalizedAttempt{}, refuse(ErrInvalidAttemptTransition, "attempt is not finalized")
	}
	result := FinalizedAttempt{
		artifactDigest:   a.artifactDigest,
		purpose:          a.purpose,
		teardownControls: append([]ControlReason(nil), a.teardownControls...),
	}
	if a.primaryControl != nil {
		value := *a.primaryControl
		result.primaryControl = &value
	}
	return result, nil
}

func (a Attempt) ID() string              { return a.id }
func (a Attempt) ArtifactDigest() Digest  { return a.artifactDigest }
func (a Attempt) Purpose() AttemptPurpose { return a.purpose }
func (a Attempt) State() AttemptState     { return a.state }

func (a Attempt) Control() (ControlReason, bool) {
	if a.primaryControl != nil {
		return *a.primaryControl, true
	}
	if len(a.teardownControls) > 0 {
		return a.teardownControls[0], true
	}
	return "", false
}

func (a Attempt) PrimaryControl() (ControlReason, bool) {
	if a.primaryControl == nil {
		return "", false
	}
	return *a.primaryControl, true
}

func (a Attempt) TeardownControls() []ControlReason {
	return append([]ControlReason(nil), a.teardownControls...)
}

func (a Attempt) StateHistory() []AttemptState {
	return append([]AttemptState(nil), a.stateHistory...)
}

func (f FinalizedAttempt) ArtifactDigest() Digest  { return f.artifactDigest }
func (f FinalizedAttempt) Purpose() AttemptPurpose { return f.purpose }

func (f FinalizedAttempt) PrimaryControl() (ControlReason, bool) {
	if f.primaryControl == nil {
		return "", false
	}
	return *f.primaryControl, true
}

func (f FinalizedAttempt) TeardownControls() []ControlReason {
	return append([]ControlReason(nil), f.teardownControls...)
}

func (f FinalizedAttempt) HasControls() bool {
	return f.primaryControl != nil || len(f.teardownControls) > 0
}
