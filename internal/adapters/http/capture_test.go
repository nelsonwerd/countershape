package http

import (
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func TestHTTPTransportFailureCannotSynthesizeStatusMutationGuard(t *testing.T) {
	input := validCapturedInput(t)
	input.TransportKind = TransportNoResponse
	input.Controls = []domain.ControlReason{domain.ControlTimeout}
	input.Response = mustParsedHTTPResponse(t, 500, "Internal Server Error", []byte(`{"request_id":"x","scratch_root":"/tmp/http-capture","kind":"failure","metadata":{}}`))
	input.ResponseParsed = true
	if _, err := newCapturedObservation(input); err == nil {
		t.Fatal("transport-control member accepted a parsed response and application status")
	}
}

func TestHTTPCaptureAllowsControlledOptionalReceiptStagesMutationGuard(t *testing.T) {
	t.Run("control before seed overlay", func(t *testing.T) {
		input := validCapturedInput(t)
		input.TransportKind = TransportNoResponse
		input.Controls = []domain.ControlReason{domain.ControlTimeout}
		observation, err := newCapturedObservation(input)
		if err != nil {
			t.Fatal(err)
		}
		if _, present := observation.Response(); present {
			t.Fatal("controlled no-response observation synthesized a response")
		}
	})

	t.Run("complete response retained across teardown control without invocation receipt", func(t *testing.T) {
		input := validCapturedInput(t)
		input.TransportKind = TransportCompleteResponse
		input.Controls = []domain.ControlReason{domain.ControlTeardownError}
		input.Response = mustParsedHTTPResponse(t, 500, "Internal Server Error", []byte(`{"request_id":"x","scratch_root":"/tmp/http-capture","kind":"failure","metadata":{}}`))
		input.SeedOverlayReceiptPresent = true
		input.SeedOverlayReceiptDigest = testHTTPDigest(t, "seed")
		input.ReadinessReceiptPresent = true
		input.ReadinessReceiptDigest = testHTTPDigest(t, "readiness")
		input.ExchangeReceiptPresent = true
		input.ExchangeReceiptDigest = testHTTPDigest(t, "exchange")
		input.ReadinessAccepted = true
		input.RequestAttempted = true
		input.RequestComplete = true
		input.ResponseParsed = true
		observation, err := newCapturedObservation(input)
		if err != nil {
			t.Fatal(err)
		}
		response, present := observation.Response()
		if !present || response.Status() != 500 || len(observation.Controls()) != 1 {
			t.Fatal("complete application response was lost instead of retained behind teardown control")
		}
	})

	t.Run("receipt stages remain monotonic", func(t *testing.T) {
		input := validCapturedInput(t)
		input.TransportKind = TransportNoResponse
		input.Controls = []domain.ControlReason{domain.ControlTimeout}
		input.ReadinessReceiptPresent = true
		input.ReadinessReceiptDigest = testHTTPDigest(t, "readiness-without-seed")
		if _, err := newCapturedObservation(input); err == nil {
			t.Fatal("readiness receipt without seed-overlay predecessor was accepted")
		}
	})

	t.Run("no-response lifecycle facts remain causally monotonic", func(t *testing.T) {
		for _, mutate := range []func(*capturedObservationInput){
			func(input *capturedObservationInput) { input.ReadinessAccepted = true },
			func(input *capturedObservationInput) { input.RequestAttempted = true },
			func(input *capturedObservationInput) { input.RequestComplete = true },
			func(input *capturedObservationInput) {
				input.InvocationReceiptPresent = true
				input.InvocationReceiptDigest = testHTTPDigest(t, "invocation-without-request")
			},
		} {
			input := validCapturedInput(t)
			input.TransportKind = TransportNoResponse
			input.Controls = []domain.ControlReason{domain.ControlProbeTransportError}
			mutate(&input)
			if _, err := newCapturedObservation(input); err == nil {
				t.Fatal("causally impossible no-response lifecycle was accepted")
			}
		}
	})
}

func TestHTTPStatus500IsEligibleWhenPhysicalLifecycleIsCleanMutationGuard(t *testing.T) {
	input := validCapturedInput(t)
	input.TransportKind = TransportCompleteResponse
	input.Response = mustParsedHTTPResponse(t, 500, "Internal Server Error", []byte(`{"request_id":"x","scratch_root":"/tmp/http-capture","kind":"failure","metadata":{}}`))
	input.SeedOverlayReceiptPresent = true
	input.SeedOverlayReceiptDigest = testHTTPDigest(t, "seed-clean")
	input.ReadinessReceiptPresent = true
	input.ReadinessReceiptDigest = testHTTPDigest(t, "readiness-clean")
	input.ExchangeReceiptPresent = true
	input.ExchangeReceiptDigest = testHTTPDigest(t, "exchange-clean")
	input.InvocationReceiptPresent = true
	input.InvocationReceiptDigest = testHTTPDigest(t, "invocation-clean")
	input.ReadinessAccepted = true
	input.RequestAttempted = true
	input.RequestComplete = true
	input.ResponseParsed = true
	observation, err := newCapturedObservation(input)
	if err != nil {
		t.Fatal(err)
	}
	response, present := observation.Response()
	if !present || response.Status() != 500 || len(observation.Controls()) != 0 {
		t.Fatal("clean 500 application response was classified as transport control")
	}
}

func validCapturedInput(t *testing.T) capturedObservationInput {
	t.Helper()
	digests := make([]domain.Digest, 6)
	for index := range digests {
		digests[index] = testHTTPDigest(t, "candidate-"+string(rune('a'+index)))
	}
	candidateKey, err := domain.NewCandidateExecutionKey(domain.CandidateExecutionIdentity{
		TreeIdentityDigest:          digests[0],
		MaterializationPolicyDigest: digests[1],
		WorldPlanDigest:             digests[2],
		AdapterDigest:               digests[3],
		RunnerDigest:                digests[4],
		ProjectionDefinitionDigest:  digests[5],
	})
	if err != nil {
		t.Fatal(err)
	}
	return capturedObservationInput{
		WorldDigest:                       testHTTPDigest(t, "world"),
		PlanDigest:                        testHTTPDigest(t, "plan"),
		CandidateKey:                      candidateKey,
		StimulusDigest:                    testHTTPDigest(t, "stimulus"),
		ExecutionPayloadDigest:            testHTTPDigest(t, "payload"),
		AttemptDigest:                     testHTTPDigest(t, "attempt"),
		TrialIndex:                        0,
		CapturePolicyDigest:               testHTTPDigest(t, "capture-policy"),
		AdapterProjectionDefinitionDigest: testHTTPDigest(t, "adapter-projection"),
		ProjectionDefinitionDigest:        testHTTPDigest(t, "projection-binding"),
		ProcessReceiptDigest:              testHTTPDigest(t, "process"),
		ScratchRoot:                       "/tmp/http-capture",
	}
}

func testHTTPDigest(t *testing.T, label string) domain.Digest {
	t.Helper()
	digest, err := digestBytes("HTTPTestDigest", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
