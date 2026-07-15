package http

import (
	"bytes"

	"github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/world"
)

// AdaptWorldResult converts physical HTTP receipts into the adapter's closed
// transport/response sum. It reparses retained bytes and re-encodes the
// request, so no world assertion becomes projected behavior merely because it
// was recorded in a receipt.
func AdaptWorldResult(result world.Result, binding HTTPExecutionBinding, trialIndex int) (HTTPCapturedObservation, error) {
	policy := binding.CapturePolicy()
	if !binding.Valid() || !policy.Valid() || trialIndex < 0 {
		return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "capture input authority is incomplete")
	}
	instance := result.World()
	attempt := result.FinalizedAttempt()
	process := result.Process()
	controls, err := exactHTTPControls(attempt, process)
	if err != nil {
		return HTTPCapturedObservation{}, err
	}
	if !instance.Digest().Valid() || !attempt.ArtifactDigest().Valid() || !process.Digest().Valid() ||
		instance.PlanDigest() != binding.PlanDigest() || instance.StimulusDigest() != binding.StimulusDigest() ||
		instance.CapturePolicyDigest() != policy.Digest() || instance.ScheduleOrdinal() != trialIndex ||
		instance.AttemptArtifactDigest() != attempt.ArtifactDigest() || process.WorldDigest() != instance.Digest() ||
		process.AttemptArtifactDigest() != attempt.ArtifactDigest() {
		return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "world, process, binding, or capture policy lineage differs")
	}
	actualArgv := process.LogicalArgv()
	if (process.PhysicalExecutionEntered() || process.SpawnAttempted()) && !equalHTTPStrings(actualArgv, binding.LogicalArgv()) {
		return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "physical HTTP argv differs from the sealed binding")
	}
	if !process.PhysicalExecutionEntered() && !process.SpawnAttempted() && len(actualArgv) != 0 {
		return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "pre-spawn HTTP control carries non-applied argv")
	}

	seed, seedPresent := result.HTTPSeedOverlay()
	if seedPresent && (!seed.Valid() || seed.ExecutionBindingDigest() != binding.Digest() || seed.FixtureRecipeDigest() != binding.FixtureRecipe().Digest() || seed.Root() != result.Roots().Fixture()) {
		return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "HTTP seed overlay receipt differs from binding or roots")
	}
	readiness, readinessPresent := result.HTTPReadiness()
	if readinessPresent && (!readiness.Valid() || readiness.WorldDigest() != instance.Digest() || readiness.AttemptArtifactDigest() != attempt.ArtifactDigest() || readiness.ExecutionBindingDigest() != binding.Digest()) {
		return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "HTTP readiness receipt lineage differs")
	}
	exchange, exchangePresent := result.HTTPExchange()
	if exchangePresent && (!exchange.Valid() || exchange.WorldDigest() != instance.Digest() || exchange.AttemptArtifactDigest() != attempt.ArtifactDigest() || exchange.ExecutionBindingDigest() != binding.Digest()) {
		return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "HTTP exchange receipt lineage differs")
	}
	invocation, invocationPresent := result.HTTPInvocationEvidence()
	if invocationPresent && (!invocation.Valid() || invocation.WorldDigest() != instance.Digest() || invocation.AttemptArtifactDigest() != attempt.ArtifactDigest() || invocation.ExecutionBindingDigest() != binding.Digest()) {
		return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "HTTP invocation evidence lineage differs")
	}
	if (readinessPresent && !seedPresent) || (exchangePresent && !readinessPresent) || (invocationPresent && !exchangePresent) {
		return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "HTTP world receipt stages are not monotonic")
	}

	var response model.HTTPResponse
	if exchangePresent {
		if exchange.Endpoint() != readiness.Endpoint() || exchange.TransportProfile() != "DIRECT_TCP4_LITERAL_127_0_0_1" || exchange.ConnectionAttempts() > 1 || exchange.RedirectsFollowed() != 0 || exchange.Retries() != 0 || exchange.Proxy() != "NONE" || exchange.DNS() != "NONE_LITERAL_IP" || exchange.CookieJar() != "NONE" || exchange.Compression() != "NONE" {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "HTTP exchange violated the direct single-request transport profile")
		}
		requestWire, err := model.EncodeRequest(binding.Stimulus(), readiness.Port())
		if err != nil {
			return HTTPCapturedObservation{}, err
		}
		requestByteDigest, err := digestBytes("HTTPRequestWireBytes", requestWire.Bytes())
		if err != nil {
			return HTTPCapturedObservation{}, err
		}
		if !bytes.Equal(exchange.RequestWire(), requestWire.Bytes()) || exchange.RequestWireDigest() != requestWire.Digest() || exchange.RequestWireBytesDigest() != requestByteDigest || exchange.RequestWireRawSHA256() != requestWire.RawSHA256() {
			return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "retained request bytes differ from the sealed HTTP stimulus")
		}
		writtenBytes := exchange.RequestWrittenBytes()
		if writtenBytes < 0 || writtenBytes > int64(requestWire.ByteLength()) || (exchange.RequestComplete() && writtenBytes != int64(requestWire.ByteLength())) {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "request write progress is outside the sealed request boundary")
		}
		responseWire := exchange.ResponseWire()
		responseByteDigest, err := digestBytes("HTTPResponseWireBytes", responseWire)
		if err != nil {
			return HTTPCapturedObservation{}, err
		}
		if exchange.ResponseWireDigest() != responseByteDigest || exchange.ResponseObservedBytes() != int64(len(responseWire)) {
			return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "retained response bytes differ from the exchange receipt")
		}
		if exchange.ResponseParsed() {
			if !exchange.RequestComplete() || exchange.ResponseOverflow() {
				return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "parsed response has incomplete or overflowing retained bytes")
			}
			response, err = model.ParseResponse(responseWire, policy)
			if err != nil {
				return HTTPCapturedObservation{}, err
			}
			parsedDigest, hasParsedDigest := exchange.ParsedResponseDigest()
			status, hasStatus := exchange.Status()
			if !hasParsedDigest || !hasStatus || parsedDigest != response.Digest() || status != response.Status() || exchange.ResponseWireDigest() != response.WireDigest() {
				return HTTPCapturedObservation{}, refuse(CodeCaptureLineageMismatch, "reparsed response differs from retained exchange evidence")
			}
		} else {
			_, hasParsedDigest := exchange.ParsedResponseDigest()
			_, hasStatus := exchange.Status()
			if hasParsedDigest || hasStatus {
				return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "unparsed transport evidence cannot synthesize application response facts")
			}
		}
	}

	transportKind := TransportNoResponse
	if response.Valid() {
		transportKind = TransportCompleteResponse
		if len(controls) == 0 && (!invocationPresent || !invocation.Validated()) {
			return HTTPCapturedObservation{}, refuse(CodeCaptureEvidenceInvalid, "clean complete response lacks validated invocation evidence")
		}
	}
	return newCapturedObservation(capturedObservationInput{
		WorldDigest: instance.Digest(), PlanDigest: binding.PlanDigest(), CandidateKey: instance.CandidateKey(),
		StimulusDigest: binding.StimulusDigest(), ExecutionPayloadDigest: binding.ExecutionPayloadDigest(), AttemptDigest: attempt.ArtifactDigest(),
		TrialIndex: trialIndex, CapturePolicyDigest: policy.Digest(), AdapterProjectionDefinitionDigest: binding.AdapterProjectionDefinitionDigest(),
		ProjectionDefinitionDigest: binding.ProjectionDefinitionBinding().Digest(), ProcessReceiptDigest: process.Digest(),
		SeedOverlayReceiptDigest: seed.Digest(), SeedOverlayReceiptPresent: seedPresent,
		ReadinessReceiptDigest: readiness.Digest(), ReadinessReceiptPresent: readinessPresent,
		ExchangeReceiptDigest: exchange.Digest(), ExchangeReceiptPresent: exchangePresent,
		InvocationReceiptDigest: invocation.Digest(), InvocationReceiptPresent: invocationPresent,
		TransportKind: transportKind, Response: response, Controls: controls, ScratchRoot: result.Roots().State(),
		ReadinessAccepted: readinessPresent && readiness.Accepted(), RequestAttempted: exchangePresent && exchange.ConnectionAttempts() == 1,
		RequestComplete: exchangePresent && exchange.RequestComplete(), ResponseParsed: exchangePresent && exchange.ResponseParsed(),
	})
}

func exactHTTPControls(attempt domain.FinalizedAttempt, process world.ProcessReceipt) ([]domain.ControlReason, error) {
	controls := make([]domain.ControlReason, 0, 3)
	attemptPrimary, hasAttemptPrimary := attempt.PrimaryControl()
	processPrimary, hasProcessPrimary := process.PrimaryControl()
	if hasAttemptPrimary != hasProcessPrimary || (hasAttemptPrimary && attemptPrimary != processPrimary) {
		return nil, refuse(CodeCaptureEvidenceInvalid, "process and finalized-attempt primary controls differ")
	}
	if hasAttemptPrimary {
		controls = append(controls, attemptPrimary)
	}
	controls = append(controls, attempt.TeardownControls()...)
	if process.OrphanRisk() != containsHTTPReason(controls, domain.ControlOrphanRisk) || process.TeardownError() != containsHTTPReason(controls, domain.ControlTeardownError) {
		return nil, refuse(CodeCaptureEvidenceInvalid, "process and finalized-attempt teardown controls differ")
	}
	return controls, nil
}

func containsHTTPReason(reasons []domain.ControlReason, target domain.ControlReason) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}

func equalHTTPStrings(left, right []string) bool {
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
