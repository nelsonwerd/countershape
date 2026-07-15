package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/world"
)

func testProjectionDefinition(t *testing.T, fields ...CLIFieldID) CLIProjectionDefinition {
	t.Helper()
	definition, err := NewCLIProjectionDefinition(CLIProjectionDefinitionConfig{Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func testObservation(
	t *testing.T,
	definition CLIProjectionDefinition,
	stdout, stderr []byte,
	completion CLICompletion,
	controls []domain.ControlReason,
	attemptCharacter string,
) CLICapturedObservation {
	t.Helper()
	stdoutChannel, err := adaptChannel(CLIChannelStdout, stdout, int64(len(stdout)), false, maxCaptureBytes, true)
	if err != nil {
		t.Fatal(err)
	}
	stderrChannel, err := adaptChannel(CLIChannelStderr, stderr, int64(len(stderr)), false, maxCaptureBytes, true)
	if err != nil {
		t.Fatal(err)
	}
	invocation := CLIFixtureInvocationReceipt{
		presence: PresencePresent, digest: cliTestDigest("d"), status: world.CLIInvocationValidated,
		filePresence: world.CLIInvocationPresencePresent, validated: true,
	}
	candidate, err := domain.ParseCandidateExecutionKey("candidate:" + stringsRepeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	observation := CLICapturedObservation{
		worldDigest: cliTestDigest("1"), planDigest: cliTestDigest("2"), candidateKey: candidate,
		stimulusDigest: cliTestDigest("3"), executionPayloadDigest: cliTestDigest("4"),
		attemptDigest: cliTestDigest(attemptCharacter), trialIndex: 0,
		capturePolicyDigest:               cliTestDigest("5"),
		adapterProjectionDefinitionDigest: definition.Digest(),
		projectionDefinitionDigest:        definition.Binding().Digest(),
		processReceiptDigest:              cliTestDigest("6"), fixtureOverlayDigest: cliTestDigest("e"),
		fixtureOverlayPresent: true, invocationReceipt: invocation,
		completion: completion, completionPresent: true, stdout: stdoutChannel, stderr: stderrChannel,
		controls: append([]domain.ControlReason{}, controls...), markerBeforeSpawn: true, spawnAttempted: true, started: true,
	}
	digest, canonicalBytes, err := cliDigestTyped("CLICapturedObservation", observation.identity())
	if err != nil {
		t.Fatal(err)
	}
	observation.digest = digest
	observation.canonicalBytes = canonicalBytes
	if !observation.Valid() {
		t.Fatal("test observation is invalid")
	}
	return observation
}

func stringsRepeat(value string, count int) string {
	result := ""
	for index := 0; index < count; index++ {
		result += value
	}
	return result
}

func TestCLIProjectionDefinitionIsClosedVisibleAndOrderNormalized(t *testing.T) {
	left := testProjectionDefinition(t, CLIFieldStdoutJSONSource, CLIFieldExitCode, CLIFieldStdoutJSONMode)
	right := testProjectionDefinition(t, CLIFieldStdoutJSONMode, CLIFieldStdoutJSONSource, CLIFieldExitCode)
	if !left.Valid() || left.Digest() != right.Digest() || left.Binding().Digest() != right.Binding().Digest() {
		t.Fatal("same closed field set produced different definition authority")
	}
	if len(left.Operations()) < len(left.Fields())+3 {
		t.Fatal("definition hides required channel/parse/encode operations")
	}
	if _, err := NewCLIProjectionDefinition(CLIProjectionDefinitionConfig{Fields: []CLIFieldID{"cli.stdout.json.arbitrary"}}); err == nil {
		t.Fatal("arbitrary field path escaped the closed registry")
	}
	if _, err := NewCLIProjectionDefinition(CLIProjectionDefinitionConfig{Fields: []CLIFieldID{CLIFieldExitCode, CLIFieldExitCode}}); err == nil {
		t.Fatal("duplicate projection field was accepted")
	}
}

func TestCLIProjectionNonzeroExitAndStrictJSONAreEligibleBehavior(t *testing.T) {
	definition := testProjectionDefinition(t, CLIFieldCompletionKind, CLIFieldExitCode, CLIFieldStdoutJSONMode, CLIFieldStdoutJSONSource)
	observation := testObservation(t, definition, []byte(`{"source":"argv","mode":""}`), []byte("diagnostic"), CLICompletion{kind: CompletionExited, code: 2}, nil, "7")
	result, err := definition.Project(observation)
	if err != nil {
		t.Fatal(err)
	}
	fields := result.Fields()
	if len(fields) != 4 {
		t.Fatalf("projected %d fields, want 4", len(fields))
	}
	code, ok := fields[1].Value().Integer()
	if !ok || code != 2 {
		t.Fatalf("exit code = %d,%v; want 2,true", code, ok)
	}
	mode, ok := fields[2].Value().String()
	if !ok || mode != "" || fields[2].Value().Tag() != ExactString {
		t.Fatal("present-empty JSON string was not preserved as present empty")
	}
	if !result.Derivation().Valid() || !result.DerivationDigest().Valid() {
		t.Fatal("projection did not produce opaque derivation authority")
	}
	wantOperations := definition.Operations()
	gotOperations := result.Operations()
	if len(gotOperations) != len(wantOperations) {
		t.Fatalf("visible transcript has %d operations, definition has %d", len(gotOperations), len(wantOperations))
	}
	for index := range wantOperations {
		if gotOperations[index].Name() != wantOperations[index].Name() || gotOperations[index].RuleDigest() != wantOperations[index].RuleDigest() {
			t.Fatalf("operation %d is hidden or reordered", index)
		}
	}
	if len(result.SourceLinks()) == 0 {
		t.Fatal("projection omitted captured-to-projected source links")
	}
}

func TestCLIProjectionEligibilityTranscriptNamesEveryReadFact(t *testing.T) {
	definition := testProjectionDefinition(t, CLIFieldStdoutBytes)
	observation := testObservation(
		t, definition, []byte("behavior"), []byte{}, CLICompletion{kind: CompletionExited, code: 0}, nil, "1",
	)
	result, err := definition.Project(observation)
	if err != nil {
		t.Fatal(err)
	}
	transcript := result.Transcript()
	if len(transcript) == 0 || transcript[0].Operation().Name() != "cli.require-eligible-capture/v1" ||
		transcript[0].Operation().Semantics() != "require no controls, present completion, complete stdout and stderr, present fixture overlay receipt, and validated fixture invocation receipt" {
		t.Fatalf("eligibility operation is not visibly complete: %#v", transcript)
	}
	wantLinks := []CLIProjectionSourceLink{
		sourceLink("", "controls"),
		sourceLink(CLIChannelExit, "completion"),
		sourceLink(CLIChannelStdout, "state"),
		sourceLink(CLIChannelStderr, "state"),
		sourceLink("", "fixture_overlay_receipt"),
		sourceLink("", "fixture_invocation_receipt"),
	}
	if !sourceLinksEqual(transcript[0].SourceLinks(), wantLinks) {
		t.Fatalf("eligibility source links = %#v, want %#v", transcript[0].SourceLinks(), wantLinks)
	}
}

func TestCLIProjectionBindingAcceptedChannelsCoverEveryPipelineRead(t *testing.T) {
	for _, definition := range []CLIProjectionDefinition{
		testProjectionDefinition(t, CLIFieldCompletionKind),
		testProjectionDefinition(t, CLIFieldStdoutJSONMode),
	} {
		want := []string{"exit", "stderr", "stdout"}
		got := definition.Binding().AcceptedChannels()
		if !equalStrings(got, want) {
			t.Fatalf("accepted channels %q do not cover eligibility pipeline reads %q", got, want)
		}
	}
}

func TestCLIProjectionResourceLimitsRejectWithoutPanicOrDataLoss(t *testing.T) {
	stdoutDefinition := testProjectionDefinition(t, CLIFieldStdoutBytes)
	stdout := testObservation(
		t, stdoutDefinition, bytes.Repeat([]byte("x"), 800<<10), []byte{},
		CLICompletion{kind: CompletionExited, code: 0}, nil, "2",
	)
	_, err := stdoutDefinition.Project(stdout)
	var rejection *ProjectionRejection
	if !errors.As(err, &rejection) || rejection.Code != CodeProjectionResourceLimit ||
		rejection.Field != CLIFieldStdoutBytes || !rejection.EvidenceDigest().Valid() {
		t.Fatalf("large stdout bytes rejection = %#v, err=%v", rejection, err)
	}

	stderrDefinition := testProjectionDefinition(t, CLIFieldStderrText)
	stderr := testObservation(
		t, stderrDefinition, []byte{}, bytes.Repeat([]byte("y"), 1<<20),
		CLICompletion{kind: CompletionExited, code: 0}, nil, "3",
	)
	_, err = stderrDefinition.Project(stderr)
	rejection = nil
	if !errors.As(err, &rejection) || rejection.Code != CodeProjectionResourceLimit ||
		rejection.Field != CLIFieldStderrText || !rejection.EvidenceDigest().Valid() {
		t.Fatalf("large stderr text rejection = %#v, err=%v", rejection, err)
	}
}

func TestCLIProjectionBytesAcceptsCompleteEmptyWhileStrictJSONRejectsIt(t *testing.T) {
	bytesDefinition := testProjectionDefinition(t, CLIFieldStdoutBytes)
	emptyBytes := testObservation(t, bytesDefinition, []byte{}, []byte{}, CLICompletion{kind: CompletionExited, code: 0}, nil, "8")
	result, err := bytesDefinition.Project(emptyBytes)
	if err != nil {
		t.Fatal(err)
	}
	projected, ok := result.Fields()[0].Value().Bytes()
	if !ok || len(projected) != 0 || result.Fields()[0].Value().Tag() != ExactBytes {
		t.Fatal("complete empty stdout was not an eligible exact byte value")
	}

	jsonDefinition := testProjectionDefinition(t, CLIFieldStdoutJSONMode)
	emptyJSON := testObservation(t, jsonDefinition, []byte{}, []byte{}, CLICompletion{kind: CompletionExited, code: 0}, nil, "9")
	_, err = jsonDefinition.Project(emptyJSON)
	var rejection *ProjectionRejection
	if !errors.As(err, &rejection) || rejection.Code != CodeProjectionJSON || !rejection.EvidenceDigest().Valid() || len(rejection.Transcript()) == 0 {
		t.Fatalf("empty strict JSON did not create typed rejection evidence: %#v", err)
	}
}

func TestCLIProjectionRetainsMissingPresentEmptyAndSignalDistinctions(t *testing.T) {
	jsonDefinition := testProjectionDefinition(t, CLIFieldStdoutJSONMode, CLIFieldStdoutJSONSource)
	missing := testObservation(t, jsonDefinition, []byte(`{}`), nil, CLICompletion{kind: CompletionExited, code: 0}, nil, "a")
	empty := testObservation(t, jsonDefinition, []byte(`{"mode":"","source":""}`), nil, CLICompletion{kind: CompletionExited, code: 0}, nil, "b")
	missingResult, err := jsonDefinition.Project(missing)
	if err != nil {
		t.Fatal(err)
	}
	emptyResult, err := jsonDefinition.Project(empty)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(missingResult.ProjectionBytes(), emptyResult.ProjectionBytes()) ||
		missingResult.Fields()[0].Value().Tag() != ExactMissing || emptyResult.Fields()[0].Value().Tag() != ExactString {
		t.Fatal("missing strict-JSON member collapsed into present empty")
	}

	exitDefinition := testProjectionDefinition(t, CLIFieldCompletionKind, CLIFieldExitCode, CLIFieldExitSignal)
	signaled := testObservation(t, exitDefinition, nil, nil, CLICompletion{kind: CompletionSignaled, signal: "SIGTERM"}, nil, "c")
	signalResult, err := exitDefinition.Project(signaled)
	if err != nil {
		t.Fatal(err)
	}
	if signalResult.Fields()[1].Value().Tag() != ExactMissing || signalResult.Fields()[2].Value().Tag() != ExactString {
		t.Fatal("signal termination collapsed into an exit code")
	}
}

func TestCLIProjectionStrictJSONRefusalsAreTypedAndEvidenceBearing(t *testing.T) {
	definition := testProjectionDefinition(t, CLIFieldStdoutJSONMode)
	for name, stdout := range map[string][]byte{
		"duplicate":    []byte(`{"mode":"a","mode":"b"}`),
		"non-object":   []byte(`[]`),
		"wrong-type":   []byte(`{"mode":1}`),
		"invalid-utf8": {0xff},
	} {
		t.Run(name, func(t *testing.T) {
			observation := testObservation(t, definition, stdout, nil, CLICompletion{kind: CompletionExited, code: 0}, nil, "d")
			_, err := definition.Project(observation)
			var rejection *ProjectionRejection
			if !errors.As(err, &rejection) || !rejection.EvidenceDigest().Valid() ||
				rejection.ObservationDigest() != observation.Digest() || rejection.DefinitionDigest() != definition.Binding().Digest() {
				t.Fatalf("rejection lacks exact evidence lineage: %#v", err)
			}
			evidence, bridgeErr := PrepareProjectionRejectionEvidence(observation, rejection)
			if bridgeErr != nil || !evidence.Valid() || evidence.ObservationDigest() != observation.Digest() ||
				evidence.DefinitionDigest() != definition.Binding().Digest() ||
				evidence.WorldDigest() != observation.WorldDigest() ||
				evidence.AttemptArtifactDigest() != observation.AttemptDigest() ||
				evidence.CandidateKey() != observation.CandidateKey() ||
				evidence.CapturePolicyDigest() != observation.CapturePolicyDigest() {
				t.Fatalf("strict rejection bridge failed: evidence=%#v err=%v", evidence, bridgeErr)
			}
		})
	}
}

func TestCLIProjectionKeepsStdoutAndStderrIndependent(t *testing.T) {
	definition := testProjectionDefinition(t, CLIFieldStdoutBytes, CLIFieldStderrText)
	left := testObservation(t, definition, []byte("stdout"), []byte("stderr"), CLICompletion{kind: CompletionExited, code: 0}, nil, "e")
	right := testObservation(t, definition, []byte("stderr"), []byte("stdout"), CLICompletion{kind: CompletionExited, code: 0}, nil, "f")
	leftResult, err := definition.Project(left)
	if err != nil {
		t.Fatal(err)
	}
	rightResult, err := definition.Project(right)
	if err != nil {
		t.Fatal(err)
	}
	leftFields := leftResult.Fields()
	if len(leftFields) != 2 {
		t.Fatalf("left projection has %d fields, want 2", len(leftFields))
	}
	leftStdout, leftStdoutOK := leftFields[0].Value().Bytes()
	leftStderr, leftStderrOK := leftFields[1].Value().String()
	if !leftStdoutOK || !bytes.Equal(leftStdout, []byte("stdout")) ||
		!leftStderrOK || leftStderr != "stderr" {
		t.Fatalf("left projection crossed physical channel sources: %#v", leftFields)
	}
	rightFields := rightResult.Fields()
	if len(rightFields) != 2 {
		t.Fatalf("right projection has %d fields, want 2", len(rightFields))
	}
	rightStdout, rightStdoutOK := rightFields[0].Value().Bytes()
	rightStderr, rightStderrOK := rightFields[1].Value().String()
	if !rightStdoutOK || !bytes.Equal(rightStdout, []byte("stderr")) ||
		!rightStderrOK || rightStderr != "stdout" {
		t.Fatalf("right projection crossed physical channel sources: %#v", rightFields)
	}
	if bytes.Equal(leftResult.ProjectionBytes(), rightResult.ProjectionBytes()) {
		t.Fatal("stdout/stderr swap did not alter typed projection")
	}
}

func TestCLIProjectionDerivationIsAttemptSpecificWhileProjectionBytesAreBehaviorOnly(t *testing.T) {
	definition := testProjectionDefinition(t, CLIFieldExitCode, CLIFieldStdoutJSONSource)
	left := testObservation(t, definition, []byte(`{"source":"config"}`), nil, CLICompletion{kind: CompletionExited, code: 0}, nil, "1")
	right := testObservation(t, definition, []byte(`{ "source" : "config" }`), nil, CLICompletion{kind: CompletionExited, code: 0}, nil, "2")
	leftResult, err := definition.Project(left)
	if err != nil {
		t.Fatal(err)
	}
	rightResult, err := definition.Project(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftResult.ProjectionBytes(), rightResult.ProjectionBytes()) {
		t.Fatal("format-only strict JSON difference escaped selected-field projection")
	}
	if leftResult.DerivationDigest() == rightResult.DerivationDigest() {
		t.Fatal("fresh attempt lineage collapsed into behavior fingerprint bytes")
	}
}

func TestCLIProjectionDerivationRejectsDeletedReorderedOrRelinkedTranscript(t *testing.T) {
	definition := testProjectionDefinition(
		t, CLIFieldCompletionKind, CLIFieldExitCode, CLIFieldStdoutJSONMode, CLIFieldStdoutJSONSource,
	)
	observation := testObservation(
		t, definition, []byte(`{"mode":"strict","source":"argv"}`), nil,
		CLICompletion{kind: CompletionExited, code: 0}, nil, "4",
	)
	result, err := definition.Project(observation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.derivation.transcript) < 2 || len(result.derivation.transcript[0].sourceLinks) == 0 {
		t.Fatal("test projection does not exercise a multi-operation linked transcript")
	}

	mutations := map[string]func(*CLIProjectionDerivation){
		"deleted-entry": func(derivation *CLIProjectionDerivation) {
			derivation.transcript = append(
				[]CLIProjectionTraceEntry(nil), derivation.transcript[:len(derivation.transcript)-1]...,
			)
		},
		"reordered-entries": func(derivation *CLIProjectionDerivation) {
			derivation.transcript[0], derivation.transcript[1] = derivation.transcript[1], derivation.transcript[0]
		},
		"changed-source-link": func(derivation *CLIProjectionDerivation) {
			derivation.transcript[0].sourceLinks[0].selector = []string{"different-authority"}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			derivation := cloneDerivation(result.derivation)
			mutate(&derivation)
			digest, canonicalBytes, digestErr := digestDerivation(
				derivation.observationDigest,
				derivation.adapterDefinitionDigest,
				derivation.definitionDigest,
				derivation.projectionByteDigest,
				derivation.transcript,
			)
			if digestErr != nil {
				t.Fatal(digestErr)
			}
			derivation.digest, derivation.canonicalBytes = digest, canonicalBytes
			if derivation.Valid() {
				t.Fatal("self-sealed transcript mutation remained a valid derivation")
			}
		})
	}
}

func TestCLIProjectionNeverRepairsControlIntoBehavior(t *testing.T) {
	definition := testProjectionDefinition(t, CLIFieldStdoutBytes)
	controlled := testObservation(t, definition, []byte("would-look-valid"), nil, CLICompletion{kind: CompletionExited, code: 0}, []domain.ControlReason{domain.ControlTimeout}, "3")
	_, err := definition.Project(controlled)
	var rejection *ProjectionRejection
	if !errors.As(err, &rejection) || rejection.Code != CodeProjectionControl || !rejection.EvidenceDigest().Valid() {
		t.Fatalf("control became behavior or lost typed rejection evidence: %#v", err)
	}
}

func TestCLIMissingInvocationMarkerIsPostCaptureProjectionRejection(t *testing.T) {
	definition := testProjectionDefinition(t, CLIFieldStdoutBytes)
	observation := testObservation(t, definition, []byte("valid-behavior-shape"), nil, CLICompletion{kind: CompletionExited, code: 0}, nil, "4")
	observation.invocationReceipt = CLIFixtureInvocationReceipt{
		presence: PresencePresent, digest: cliTestDigest("f"), status: world.CLIInvocationAbsent,
		filePresence: world.CLIInvocationPresenceAbsent, validated: false,
	}
	digest, canonicalBytes, err := cliDigestTyped("CLICapturedObservation", observation.identity())
	if err != nil {
		t.Fatal(err)
	}
	observation.digest, observation.canonicalBytes = digest, canonicalBytes
	if !observation.Valid() || len(observation.Controls()) != 0 || observation.ProjectionEligible() {
		t.Fatal("missing fixture marker was rewritten as lifecycle control or eligible behavior")
	}
	_, err = definition.Project(observation)
	var rejection *ProjectionRejection
	if !errors.As(err, &rejection) || rejection.Code != CodeProjectionControl ||
		rejection.ObservationDigest() != observation.Digest() || !rejection.EvidenceDigest().Valid() {
		t.Fatalf("missing fixture marker did not become evidence-bearing post-capture rejection: %#v", err)
	}
}
