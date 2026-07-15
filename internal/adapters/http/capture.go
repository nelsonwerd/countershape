package http

import (
	"bytes"
	"sort"

	"github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
)

type HTTPTransportKind string

const (
	TransportNoResponse       HTTPTransportKind = "NO_RESPONSE"
	TransportCompleteResponse HTTPTransportKind = "COMPLETE_RESPONSE"
)

// HTTPCapturedObservation is the post-policy adapter fact. A transport
// control and a complete response are disjoint; no transport path can
// manufacture an application status.
type HTTPCapturedObservation struct {
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
	seedOverlayReceiptDigest          domain.Digest
	seedOverlayReceiptPresent         bool
	readinessReceiptDigest            domain.Digest
	readinessReceiptPresent           bool
	exchangeReceiptDigest             domain.Digest
	exchangeReceiptPresent            bool
	invocationReceiptDigest           domain.Digest
	invocationReceiptPresent          bool
	transportKind                     HTTPTransportKind
	response                          model.HTTPResponse
	controls                          []domain.ControlReason
	scratchRoot                       string
	readinessAccepted                 bool
	requestAttempted                  bool
	requestComplete                   bool
	responseParsed                    bool
}

type capturedObservationIdentity struct {
	SchemaVersion                     string   `json:"schema_version"`
	Kind                              string   `json:"kind"`
	WorldInstanceDigest               string   `json:"world_instance_digest"`
	WorldPlanDigest                   string   `json:"world_plan_digest"`
	CandidateExecutionKey             string   `json:"candidate_execution_key"`
	StimulusDigest                    string   `json:"stimulus_digest"`
	ExecutionPayloadDigest            string   `json:"execution_payload_digest"`
	AttemptArtifactDigest             string   `json:"attempt_artifact_digest"`
	TrialIndex                        int      `json:"trial_index"`
	CapturePolicyDigest               string   `json:"capture_policy_digest"`
	AdapterProjectionDefinitionDigest string   `json:"http_adapter_projection_definition_digest"`
	ProjectionDefinitionDigest        string   `json:"projection_definition_digest"`
	ProcessReceiptDigest              string   `json:"process_receipt_digest"`
	SeedOverlayReceiptDigest          string   `json:"seed_overlay_receipt_digest"`
	SeedOverlayReceiptPresence        string   `json:"seed_overlay_receipt_presence"`
	ReadinessReceiptDigest            string   `json:"readiness_receipt_digest"`
	ReadinessReceiptPresence          string   `json:"readiness_receipt_presence"`
	ExchangeReceiptDigest             string   `json:"exchange_receipt_digest"`
	ExchangeReceiptPresence           string   `json:"exchange_receipt_presence"`
	InvocationReceiptDigest           string   `json:"invocation_receipt_digest"`
	InvocationReceiptPresence         string   `json:"invocation_receipt_presence"`
	TransportKind                     string   `json:"transport_kind"`
	ResponseDigest                    string   `json:"response_digest"`
	ResponseWireDigest                string   `json:"response_wire_digest"`
	ResponseStatus                    int      `json:"response_status"`
	ResponseContentLength             int64    `json:"response_content_length"`
	Controls                          []string `json:"controls"`
	ScratchRoot                       string   `json:"scratch_root"`
	ReadinessAccepted                 bool     `json:"readiness_accepted"`
	RequestAttempted                  bool     `json:"request_attempted"`
	RequestComplete                   bool     `json:"request_complete"`
	ResponseParsed                    bool     `json:"response_parsed"`
	TransportFailureBecameStatus      bool     `json:"transport_failure_became_status"`
}

type capturedObservationInput struct {
	WorldDigest                       domain.Digest
	PlanDigest                        domain.Digest
	CandidateKey                      domain.CandidateExecutionKey
	StimulusDigest                    domain.Digest
	ExecutionPayloadDigest            domain.Digest
	AttemptDigest                     domain.Digest
	TrialIndex                        int
	CapturePolicyDigest               domain.Digest
	AdapterProjectionDefinitionDigest domain.Digest
	ProjectionDefinitionDigest        domain.Digest
	ProcessReceiptDigest              domain.Digest
	SeedOverlayReceiptDigest          domain.Digest
	SeedOverlayReceiptPresent         bool
	ReadinessReceiptDigest            domain.Digest
	ReadinessReceiptPresent           bool
	ExchangeReceiptDigest             domain.Digest
	ExchangeReceiptPresent            bool
	InvocationReceiptDigest           domain.Digest
	InvocationReceiptPresent          bool
	TransportKind                     HTTPTransportKind
	Response                          model.HTTPResponse
	Controls                          []domain.ControlReason
	ScratchRoot                       string
	ReadinessAccepted                 bool
	RequestAttempted                  bool
	RequestComplete                   bool
	ResponseParsed                    bool
}

func newCapturedObservation(input capturedObservationInput) (HTTPCapturedObservation, error) {
	if !input.WorldDigest.Valid() || !input.PlanDigest.Valid() || !input.CandidateKey.Valid() ||
		!input.StimulusDigest.Valid() || !input.ExecutionPayloadDigest.Valid() || !input.AttemptDigest.Valid() ||
		input.TrialIndex < 0 || !input.CapturePolicyDigest.Valid() ||
		!input.AdapterProjectionDefinitionDigest.Valid() || !input.ProjectionDefinitionDigest.Valid() ||
		!input.ProcessReceiptDigest.Valid() || input.ScratchRoot == "" {
		return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "captured observation lineage is incomplete")
	}
	for _, receipt := range []struct {
		present bool
		digest  domain.Digest
	}{
		{input.SeedOverlayReceiptPresent, input.SeedOverlayReceiptDigest},
		{input.ReadinessReceiptPresent, input.ReadinessReceiptDigest},
		{input.ExchangeReceiptPresent, input.ExchangeReceiptDigest},
		{input.InvocationReceiptPresent, input.InvocationReceiptDigest},
	} {
		if receipt.present != receipt.digest.Valid() {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "optional HTTP receipt presence disagrees with its digest")
		}
	}
	if (input.ReadinessReceiptPresent && !input.SeedOverlayReceiptPresent) ||
		(input.ExchangeReceiptPresent && !input.ReadinessReceiptPresent) ||
		(input.InvocationReceiptPresent && !input.ExchangeReceiptPresent) {
		return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "optional HTTP receipt stages are not monotonic")
	}
	controls := append([]domain.ControlReason(nil), input.Controls...)
	for _, control := range controls {
		if !control.Valid() {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "captured observation carries an invalid control")
		}
	}
	sort.Slice(controls, func(i, j int) bool { return controls[i] < controls[j] })
	for index := 1; index < len(controls); index++ {
		if controls[index] == controls[index-1] {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "captured observation carries a duplicate control")
		}
	}
	// MUTATION_ANCHOR: http-transport-failure-cannot-synthesize-status
	if input.TransportKind == TransportNoResponse {
		if input.Response.Valid() || input.ResponseParsed || len(controls) == 0 {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "transport control cannot carry a parsed response or application status")
		}
		if (input.ReadinessAccepted && !input.ReadinessReceiptPresent) ||
			(input.RequestAttempted && !input.ReadinessAccepted) ||
			(input.RequestComplete && !input.RequestAttempted) ||
			(input.InvocationReceiptPresent && !input.RequestComplete) {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "no-response lifecycle facts are not causally monotonic")
		}
	} else if input.TransportKind == TransportCompleteResponse {
		if !input.Response.Valid() || !input.ReadinessAccepted || !input.RequestAttempted || !input.RequestComplete ||
			!input.ResponseParsed || !input.SeedOverlayReceiptPresent || !input.ReadinessReceiptPresent ||
			!input.ExchangeReceiptPresent || (len(controls) == 0 && !input.InvocationReceiptPresent) {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "complete response lacks the required physical lifecycle")
		}
	} else {
		return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "unknown transport/response sum member")
	}
	observation := HTTPCapturedObservation{
		worldDigest: input.WorldDigest, planDigest: input.PlanDigest, candidateKey: input.CandidateKey,
		stimulusDigest: input.StimulusDigest, executionPayloadDigest: input.ExecutionPayloadDigest,
		attemptDigest: input.AttemptDigest, trialIndex: input.TrialIndex, capturePolicyDigest: input.CapturePolicyDigest,
		adapterProjectionDefinitionDigest: input.AdapterProjectionDefinitionDigest,
		projectionDefinitionDigest:        input.ProjectionDefinitionDigest, processReceiptDigest: input.ProcessReceiptDigest,
		seedOverlayReceiptDigest: input.SeedOverlayReceiptDigest, seedOverlayReceiptPresent: input.SeedOverlayReceiptPresent,
		readinessReceiptDigest: input.ReadinessReceiptDigest, readinessReceiptPresent: input.ReadinessReceiptPresent,
		exchangeReceiptDigest: input.ExchangeReceiptDigest, exchangeReceiptPresent: input.ExchangeReceiptPresent,
		invocationReceiptDigest: input.InvocationReceiptDigest, invocationReceiptPresent: input.InvocationReceiptPresent,
		transportKind: input.TransportKind, response: input.Response, controls: controls, scratchRoot: input.ScratchRoot,
		readinessAccepted: input.ReadinessAccepted, requestAttempted: input.RequestAttempted,
		requestComplete: input.RequestComplete, responseParsed: input.ResponseParsed,
	}
	digest, canonicalBytes, err := digestTyped("HTTPCapturedObservation", observation.identity())
	if err != nil {
		return HTTPCapturedObservation{}, err
	}
	observation.digest = digest
	observation.canonicalBytes = canonicalBytes
	return observation, nil
}

func (o HTTPCapturedObservation) identity() capturedObservationIdentity {
	controls := make([]string, len(o.controls))
	for index, control := range o.controls {
		controls[index] = string(control)
	}
	responseDigest, wireDigest := "", ""
	status := 0
	contentLength := int64(0)
	presence := func(present bool) string {
		if present {
			return "PRESENT"
		}
		return "ABSENT"
	}
	if o.transportKind == TransportCompleteResponse {
		responseDigest = o.response.Digest().String()
		wireDigest = o.response.WireDigest().String()
		status = o.response.Status()
		contentLength = o.response.ContentLength()
	}
	return capturedObservationIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPCapturedObservation",
		WorldInstanceDigest: o.worldDigest.String(), WorldPlanDigest: o.planDigest.String(),
		CandidateExecutionKey: o.candidateKey.String(), StimulusDigest: o.stimulusDigest.String(),
		ExecutionPayloadDigest: o.executionPayloadDigest.String(), AttemptArtifactDigest: o.attemptDigest.String(),
		TrialIndex: o.trialIndex, CapturePolicyDigest: o.capturePolicyDigest.String(),
		AdapterProjectionDefinitionDigest: o.adapterProjectionDefinitionDigest.String(),
		ProjectionDefinitionDigest:        o.projectionDefinitionDigest.String(), ProcessReceiptDigest: o.processReceiptDigest.String(),
		SeedOverlayReceiptDigest: o.seedOverlayReceiptDigest.String(), SeedOverlayReceiptPresence: presence(o.seedOverlayReceiptPresent),
		ReadinessReceiptDigest: o.readinessReceiptDigest.String(), ReadinessReceiptPresence: presence(o.readinessReceiptPresent),
		ExchangeReceiptDigest: o.exchangeReceiptDigest.String(), ExchangeReceiptPresence: presence(o.exchangeReceiptPresent),
		InvocationReceiptDigest: o.invocationReceiptDigest.String(), InvocationReceiptPresence: presence(o.invocationReceiptPresent),
		TransportKind: string(o.transportKind), ResponseDigest: responseDigest, ResponseWireDigest: wireDigest,
		ResponseStatus: status, ResponseContentLength: contentLength, Controls: controls, ScratchRoot: o.scratchRoot,
		ReadinessAccepted: o.readinessAccepted, RequestAttempted: o.requestAttempted, RequestComplete: o.requestComplete,
		ResponseParsed: o.responseParsed, TransportFailureBecameStatus: false,
	}
}

func (o HTTPCapturedObservation) Valid() bool {
	if !o.digest.Valid() || len(o.canonicalBytes) == 0 {
		return false
	}
	rebuilt, err := newCapturedObservation(capturedObservationInput{
		WorldDigest: o.worldDigest, PlanDigest: o.planDigest, CandidateKey: o.candidateKey,
		StimulusDigest: o.stimulusDigest, ExecutionPayloadDigest: o.executionPayloadDigest,
		AttemptDigest: o.attemptDigest, TrialIndex: o.trialIndex, CapturePolicyDigest: o.capturePolicyDigest,
		AdapterProjectionDefinitionDigest: o.adapterProjectionDefinitionDigest,
		ProjectionDefinitionDigest:        o.projectionDefinitionDigest, ProcessReceiptDigest: o.processReceiptDigest,
		SeedOverlayReceiptDigest: o.seedOverlayReceiptDigest, SeedOverlayReceiptPresent: o.seedOverlayReceiptPresent,
		ReadinessReceiptDigest: o.readinessReceiptDigest, ReadinessReceiptPresent: o.readinessReceiptPresent,
		ExchangeReceiptDigest: o.exchangeReceiptDigest, ExchangeReceiptPresent: o.exchangeReceiptPresent,
		InvocationReceiptDigest: o.invocationReceiptDigest, InvocationReceiptPresent: o.invocationReceiptPresent,
		TransportKind: o.transportKind, Response: o.response, Controls: o.controls, ScratchRoot: o.scratchRoot,
		ReadinessAccepted: o.readinessAccepted, RequestAttempted: o.requestAttempted,
		RequestComplete: o.requestComplete, ResponseParsed: o.responseParsed,
	})
	return err == nil && rebuilt.digest == o.digest && bytes.Equal(rebuilt.canonicalBytes, o.canonicalBytes)
}

func (o HTTPCapturedObservation) Digest() domain.Digest { return o.digest }
func (o HTTPCapturedObservation) CanonicalBytes() []byte {
	return append([]byte(nil), o.canonicalBytes...)
}
func (o HTTPCapturedObservation) WorldDigest() domain.Digest                 { return o.worldDigest }
func (o HTTPCapturedObservation) PlanDigest() domain.Digest                  { return o.planDigest }
func (o HTTPCapturedObservation) CandidateKey() domain.CandidateExecutionKey { return o.candidateKey }
func (o HTTPCapturedObservation) StimulusDigest() domain.Digest              { return o.stimulusDigest }
func (o HTTPCapturedObservation) AttemptDigest() domain.Digest               { return o.attemptDigest }
func (o HTTPCapturedObservation) CapturePolicyDigest() domain.Digest         { return o.capturePolicyDigest }
func (o HTTPCapturedObservation) ProjectionDefinitionDigest() domain.Digest {
	return o.projectionDefinitionDigest
}
func (o HTTPCapturedObservation) TransportKind() HTTPTransportKind { return o.transportKind }
func (o HTTPCapturedObservation) Response() (model.HTTPResponse, bool) {
	return o.response, o.transportKind == TransportCompleteResponse
}
func (o HTTPCapturedObservation) Controls() []domain.ControlReason {
	return append([]domain.ControlReason(nil), o.controls...)
}
