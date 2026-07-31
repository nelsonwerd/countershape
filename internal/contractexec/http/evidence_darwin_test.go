//go:build darwin

package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	counterhttp "github.com/nelsonwerd/countershape/internal/adapters/http"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	contractscope "github.com/nelsonwerd/countershape/internal/contractexec/scope"
	"github.com/nelsonwerd/countershape/internal/domain"
	contractfixtures "github.com/nelsonwerd/countershape/testkit/contracts"
)

func TestC5NegativeReadinessRetainsExactRawFrameWithoutCapture(t *testing.T) {
	readiness := []byte{0xff, 0x00, 'N', 'O', '\n'}
	result := serviceResult{
		process: processResult{
			stdout: []byte("stdout"), stderr: []byte("stderr"),
			stdoutObserved: 6, stderrObserved: 6,
			stdoutDrained: true, stderrDrained: true,
		},
		readiness: readinessResult{
			frameBytes: append([]byte(nil), readiness...), bytesObserved: int64(len(readiness)),
			eofObserved: true, diagnosticCode: "HTTP_READINESS_PROTOCOL_REJECTED",
		},
	}
	body, err := buildProcessDrainEvidence(result)
	if err != nil {
		t.Fatal(err)
	}
	segments := decodeEvidenceSegments(t, body)
	want := map[string][]byte{
		"stdout":          result.process.stdout,
		"stderr":          result.process.stderr,
		"readiness_frame": readiness,
	}
	if len(segments) != len(want) {
		t.Fatalf("process drain segment count=%d want=%d", len(segments), len(want))
	}
	for name, expected := range want {
		if !bytes.Equal(segments[name], expected) {
			t.Fatalf("%s bytes=%x want=%x", name, segments[name], expected)
		}
	}
	if len(result.exchange.responseWire) != 0 {
		t.Fatal("negative readiness fixture unexpectedly contained an HTTP capture")
	}
}

func TestC5ProjectionRejectionAgreesWithProcessEvidence(t *testing.T) {
	result := serviceResult{exchange: exchangeResult{responseParsed: true}}
	tuple, frame, projected, err := resolveHTTPProjection(executionInput{}, &result)
	if err != nil || projected || tuple.Valid() || len(frame) != 0 ||
		result.process.primary != domain.ControlProjectionRejected ||
		result.process.diagnosticCode != "HTTP_PROJECTION_REJECTED" {
		t.Fatalf(
			"projection rejection tuple=%t frame=%d projected=%t primary=%s diagnostic=%s err=%v",
			tuple.Valid(), len(frame), projected, result.process.primary,
			result.process.diagnosticCode, err,
		)
	}
	body, err := canonicalEvidence(
		contractmodel.EvidenceProcessResult,
		processResultEvidenceFacts(result),
	)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Facts struct {
			Primary    string `json:"primary"`
			Diagnostic string `json:"diagnostic_code"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Facts.Primary != string(domain.ControlProjectionRejected) ||
		decoded.Facts.Diagnostic != "HTTP_PROJECTION_REJECTED" {
		t.Fatalf("process evidence disagrees with projection control: %#v", decoded.Facts)
	}
}

func TestC5StartErrorPreservesEveryViolationOverMissing(t *testing.T) {
	domains := []contractmodel.ScopeDomain{
		contractmodel.ScopeTargetInventory,
		contractmodel.ScopeChildBindings,
		contractmodel.ScopeImportResolution,
		contractmodel.ScopeServiceBindings,
		contractmodel.ScopeSentinelInheritance,
	}
	kinds := []contractmodel.EvidenceKind{
		contractmodel.EvidenceTargetInventory,
		contractmodel.EvidenceChildBindings,
		contractmodel.EvidenceImportResolution,
		contractmodel.EvidenceServiceBindings,
		contractmodel.EvidenceSentinelInheritance,
	}
	violations := []contractmodel.ScopeViolation{
		contractmodel.ViolationTargetSourcePresent,
		contractmodel.ViolationChildBinding,
		contractmodel.ViolationImportResolution,
		contractmodel.ViolationServiceBinding,
		contractmodel.ViolationSentinelInherited,
	}
	for hostile := range domains {
		findings := make([]contractscope.Finding, len(domains))
		for index := range findings {
			findings[index] = contractscope.Finding{
				Domain: domains[index], State: contractmodel.ScopeCheckClean,
				Kind: kinds[index], Facts: map[string]any{"observed": true},
			}
		}
		findings[hostile].State = contractmodel.ScopeCheckViolated
		findings[hostile].Violation = violations[hostile]

		constrained := conservativeStartErrorFindings(findings)
		bodies := make(map[contractmodel.EvidenceKind][]byte)
		drafts, diagnostics, err := materializeScopeFindings(
			constrained,
			func(kind contractmodel.EvidenceKind, facts any) error {
				body, bodyErr := canonicalEvidence(kind, facts)
				if bodyErr == nil {
					bodies[kind] = body
				}
				return bodyErr
			},
		)
		if err != nil || len(drafts) != 5 || len(diagnostics) != 5 ||
			drafts[hostile].state != contractmodel.ScopeCheckViolated ||
			drafts[hostile].violation != violations[hostile] ||
			len(bodies[kinds[hostile]]) == 0 {
			t.Fatalf(
				"hostile domain %s was erased: drafts=%#v bodies=%d err=%v",
				domains[hostile], drafts, len(bodies), err,
			)
		}
		for index := range drafts {
			switch {
			case index == hostile:
			case index == 0:
				if drafts[index].state != contractmodel.ScopeCheckClean {
					t.Fatalf("independent target check became %s", drafts[index].state)
				}
			case drafts[index].state != contractmodel.ScopeCheckMissing ||
				!drafts[index].diagnostic.Valid():
				t.Fatalf("nonobserved start-error domain %s = %#v", domains[index], drafts[index])
			}
		}
	}
}

func TestC5EvidenceCapacityCoversExactMaximalRosterAndWire(t *testing.T) {
	if evidenceFramedBodyCount != 3 || evidenceCanonicalBodyCount != 12 ||
		evidenceFramedBodyCount+evidenceCanonicalBodyCount != evidenceMaximumBodyCount {
		t.Fatalf(
			"evidence body derivation framed=%d canonical=%d maximum=%d",
			evidenceFramedBodyCount, evidenceCanonicalBodyCount, evidenceMaximumBodyCount,
		)
	}
	queryEntry, err := counterhttp.QueryValue(
		strings.Repeat("%", 4096),
		strings.Repeat("%", 4096),
	)
	if err != nil {
		t.Fatal(err)
	}
	query := make([]counterhttp.HTTPQueryEntry, 7)
	for index := range query {
		query[index] = queryEntry
	}
	stimulus, err := counterhttp.NewHTTPStimulus(counterhttp.HTTPStimulusConfig{
		Method: counterhttp.MethodGET,
		Path:   "/",
		Query:  query,
		Body:   counterhttp.AbsentBody(),
	})
	if err != nil {
		t.Fatal(err)
	}
	maximumRequest, err := maximumHTTPRequestBytes(stimulus)
	wire, wireErr := counterhttp.EncodeRequest(stimulus, 65535)
	narrow, narrowErr := counterhttp.EncodeRequest(stimulus, 1)
	if err != nil || wireErr != nil || narrowErr != nil ||
		maximumRequest != int64(wire.ByteLength()) ||
		maximumRequest <= int64(narrow.ByteLength()) ||
		maximumRequest <= int64(len(stimulus.CanonicalBytes())) {
		t.Fatalf(
			"request bound maximum=%d wire=%d narrow=%d canonical=%d err=%v/%v/%v",
			maximumRequest, wire.ByteLength(), narrow.ByteLength(),
			len(stimulus.CanonicalBytes()), err, wireErr, narrowErr,
		)
	}
	readiness, err := counterhttp.NewPortableHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	maximumReadiness, err := maximumReadinessObservationBytes(readiness)
	if err != nil || maximumReadiness != int64(counterhttp.PortableReadinessFrameMax+1) {
		t.Fatalf("readiness bound=%d err=%v", maximumReadiness, err)
	}

	source, err := contractfixtures.HTTPSource()
	if err != nil {
		t.Fatal(err)
	}
	inventoryRoot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(inventoryRoot, "source.mjs"),
		[]byte("export default true;\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	before, err := contractscope.Snapshot(inventoryRoot)
	if err != nil || !before.Valid() {
		t.Fatalf("snapshot real capacity inventory: valid=%t err=%v", before.Valid(), err)
	}
	capacity, err := preflightEvidenceCapacity(source, before)
	view, ok := source.HTTPView()
	if err != nil || !ok || !view.Valid() {
		t.Fatalf("preflight real HTTP evidence capacity: view=%t err=%v", view.Valid(), err)
	}
	requestWire, requestErr := counterhttp.EncodeRequest(view.Stimulus(), 65535)
	readinessMaximum, readinessErr := maximumReadinessObservationBytes(view.Readiness())
	budgets := source.Plan().Budgets()
	expectedMaximum := budgets.StdoutBytes +
		budgets.StderrBytes +
		view.Capture().OwnerResponseReadLimit() +
		int64(requestWire.ByteLength()) +
		readinessMaximum +
		evidenceProjectionMaxBytes +
		evidenceCanonicalBodyCount*evidenceSummaryMaxBytes +
		2*evidenceFrameOverhead
	if requestErr != nil || readinessErr != nil ||
		capacity.maximumRequestBytes != int64(requestWire.ByteLength()) ||
		capacity.maximumReadinessBytes != readinessMaximum ||
		capacity.maximumUniqueBytes != expectedMaximum {
		t.Fatalf(
			"real preflight envelope maximum=%d request=%d readiness=%d want=%d err=%v/%v",
			capacity.maximumUniqueBytes,
			capacity.maximumRequestBytes,
			capacity.maximumReadinessBytes,
			expectedMaximum,
			requestErr,
			readinessErr,
		)
	}
	drainFrame, err := frameEvidence(
		"PROCESS_DRAINS",
		map[string]any{"maximal_profile_payloads": true},
		budgets.StdoutBytes+budgets.StderrBytes+readinessMaximum+evidenceFrameOverhead,
		evidenceSegment{name: "stdout", body: bytes.Repeat([]byte{0x01}, int(budgets.StdoutBytes))},
		evidenceSegment{name: "stderr", body: bytes.Repeat([]byte{0x02}, int(budgets.StderrBytes))},
		evidenceSegment{name: "readiness_frame", body: bytes.Repeat([]byte{0x03}, int(readinessMaximum))},
	)
	if err != nil {
		t.Fatalf("frame maximal drain evidence: %v", err)
	}
	responseLimit := view.Capture().OwnerResponseReadLimit()
	captureFrame, err := frameEvidence(
		"RAW_HTTP_EXCHANGE",
		map[string]any{"maximal_profile_payloads": true},
		int64(requestWire.ByteLength())+responseLimit+evidenceFrameOverhead,
		evidenceSegment{name: "request_wire", body: requestWire.Bytes()},
		evidenceSegment{name: "response_wire", body: bytes.Repeat([]byte{0x04}, int(responseLimit))},
	)
	if err != nil {
		t.Fatalf("frame maximal capture evidence: %v", err)
	}
	projectionPayloadBytes := evidenceProjectionMaxBytes - evidenceFrameOverhead
	projectionFrame, err := frameEvidence(
		"HTTP_PROJECTION",
		map[string]any{"maximal_profile_payloads": true},
		evidenceProjectionMaxBytes,
		evidenceSegment{
			name: "projection",
			body: bytes.Repeat([]byte{0x05}, int(projectionPayloadBytes)),
		},
		evidenceSegment{name: "exact_tuple", body: []byte("{}")},
	)
	if err != nil {
		t.Fatalf("frame maximal projection evidence: %v", err)
	}

	bodies := map[contractmodel.EvidenceKind][]byte{
		contractmodel.EvidenceDrainResult:         drainFrame,
		contractmodel.EvidenceCapturedObservation: captureFrame,
		contractmodel.EvidenceProjectionResult:    projectionFrame,
	}
	canonicalKinds := []contractmodel.EvidenceKind{
		contractmodel.EvidenceMaterializationRevalidation,
		contractmodel.EvidenceRuntimeRevalidation,
		contractmodel.EvidenceProcessResult,
		contractmodel.EvidenceWaitResult,
		contractmodel.EvidenceTeardownResult,
		contractmodel.EvidenceOrphanCheck,
		contractmodel.EvidenceFinalizationMarker,
		contractmodel.EvidenceTargetInventory,
		contractmodel.EvidenceChildBindings,
		contractmodel.EvidenceImportResolution,
		contractmodel.EvidenceServiceBindings,
		contractmodel.EvidenceSentinelInheritance,
	}
	canonicalPaddingBytes := int(evidenceSummaryMaxBytes - (8 << 10))
	for index, kind := range canonicalKinds {
		body, bodyErr := canonicalEvidence(kind, map[string]any{
			"ordinal": index,
			"padding": strings.Repeat(string(rune('a'+index)), canonicalPaddingBytes),
		})
		if bodyErr != nil {
			t.Fatalf("build near-ceiling canonical body %s: %v", kind, bodyErr)
		}
		bodies[kind] = body
	}
	aggregate := int64(0)
	unique := make(map[[sha256.Size]byte]struct{}, len(bodies))
	for _, body := range bodies {
		aggregate += int64(len(body))
		unique[sha256.Sum256(body)] = struct{}{}
	}
	if len(bodies) != evidenceMaximumBodyCount ||
		len(unique) != evidenceMaximumBodyCount ||
		aggregate < int64(evidenceCanonicalBodyCount)*int64(canonicalPaddingBytes) ||
		aggregate >= capacity.maximumUniqueBytes {
		t.Fatalf(
			"real maximal roster bodies=%d unique=%d aggregate=%d capacity=%d",
			len(bodies), len(unique), aggregate, capacity.maximumUniqueBytes,
		)
	}
	if err := capacity.validate(bodies); err != nil {
		t.Fatalf("exact maximal evidence roster was refused: %v", err)
	}
	tooSmall := capacity
	tooSmall.maximumUniqueBytes = aggregate - 1
	if err := tooSmall.validate(bodies); err == nil {
		t.Fatal("under-admitted evidence envelope accepted the maximal roster")
	}
	bodies[contractmodel.EvidencePrivateManifest], err = canonicalEvidence(
		contractmodel.EvidencePrivateManifest,
		map[string]any{"sixteenth_body": true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := capacity.validate(bodies); err == nil {
		t.Fatal("sixteenth private body escaped the C5 roster")
	}
}

func TestC5MissingProbeIsAmbiguous(t *testing.T) {
	input := executionInput{}
	measurements, err := input.finishProbe(context.Background())
	diagnostic, diagnosed := contractscope.DiagnosticOf(err)
	if err == nil || !measurements.Ambiguous ||
		measurements.Diagnostic.Code() != contractscope.CodeProbeAbsent ||
		!diagnosed || diagnostic.Code() != contractscope.CodeProbeAbsent {
		t.Fatalf("missing probe measurements=%#v diagnostic=%s err=%v", measurements, diagnostic.Code(), err)
	}
}

func decodeEvidenceSegments(t testing.TB, body []byte) map[string][]byte {
	t.Helper()
	if !bytes.HasPrefix(body, []byte(evidenceFrameMagic)) {
		t.Fatal("private evidence frame magic is absent")
	}
	cursor := len(evidenceFrameMagic)
	if len(body)-cursor < 8 {
		t.Fatal("private evidence metadata length is truncated")
	}
	metadataBytes := int(binary.BigEndian.Uint64(body[cursor : cursor+8]))
	cursor += 8
	if metadataBytes < 1 || metadataBytes > len(body)-cursor {
		t.Fatal("private evidence metadata is outside the frame")
	}
	var metadata struct {
		Segments []struct {
			Name string `json:"name"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(body[cursor:cursor+metadataBytes], &metadata); err != nil {
		t.Fatal(err)
	}
	cursor += metadataBytes
	segments := make(map[string][]byte, len(metadata.Segments))
	for _, segment := range metadata.Segments {
		if len(body)-cursor < 8 {
			t.Fatal("private evidence segment length is truncated")
		}
		count := int(binary.BigEndian.Uint64(body[cursor : cursor+8]))
		cursor += 8
		if count < 0 || count > len(body)-cursor {
			t.Fatal("private evidence segment is outside the frame")
		}
		if _, duplicate := segments[segment.Name]; duplicate {
			t.Fatalf("duplicate private evidence segment %q", segment.Name)
		}
		segments[segment.Name] = append([]byte(nil), body[cursor:cursor+count]...)
		cursor += count
	}
	if cursor != len(body) {
		t.Fatal("private evidence frame has trailing bytes")
	}
	return segments
}
