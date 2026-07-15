package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/world"
)

func cliTestDigest(character string) domain.Digest {
	return domain.MustDigest("sha256:" + strings.Repeat(character, 64))
}

func TestCLICapturePolicyHasIndependentExactChannelCaps(t *testing.T) {
	policy, err := NewCLICapturePolicy(CLICapturePolicyConfig{StdoutBytes: 4, StderrBytes: 7})
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Valid() || policy.StdoutBytes() != 4 || policy.StderrBytes() != 7 {
		t.Fatal("capture policy lost independent channel caps")
	}
	second, err := NewCLICapturePolicy(CLICapturePolicyConfig{StdoutBytes: 4, StderrBytes: 7})
	if err != nil || second.Digest() != policy.Digest() || !bytes.Equal(second.CanonicalBytes(), policy.CanonicalBytes()) {
		t.Fatal("capture policy identity is not canonical")
	}
	if _, err := NewCLICapturePolicy(CLICapturePolicyConfig{StdoutBytes: 0, StderrBytes: 1}); err == nil {
		t.Fatal("zero output cap was accepted")
	}
	if !appliedCaptureLimitsMatch(true, 4, 7, policy) || appliedCaptureLimitsMatch(true, 8, 7, policy) ||
		appliedCaptureLimitsMatch(true, 4, 8, policy) ||
		!appliedCaptureLimitsMatch(false, 0, 0, policy) || appliedCaptureLimitsMatch(false, 4, 7, policy) {
		t.Fatal("low-volume execution could escape exact applied capture-limit authority")
	}
}

func TestCLIStdinEvidenceRequiresExactPhysicalBindingAndDelivery(t *testing.T) {
	absent := AbsentStdin()
	absentDigest, _, err := cliDigestBytes("CLIStdinBytes", absent.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	absentEvidence := processStdinEvidence{
		presence: string(absent.Presence()), digest: absentDigest,
		delivery: "ABSENT_NULL_DEVICE", started: true, physicalEntered: true,
	}
	if !stdinEvidenceMatchesBinding(absentEvidence, absent) {
		t.Fatal("exact absent stdin evidence was rejected")
	}
	presentEmpty, err := PresentStdin(nil)
	if err != nil {
		t.Fatal(err)
	}
	presentEmptyDigest, _, err := cliDigestBytes("CLIStdinBytes", presentEmpty.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	presentEmptyEvidence := processStdinEvidence{
		presence: string(presentEmpty.Presence()), digest: presentEmptyDigest,
		delivery: "PRESENT_EXPLICIT_PIPE_WRITER", pipeAllocated: true, writerStarted: true,
		handoffAttempted: true, complete: true, started: true, physicalEntered: true,
	}
	if !stdinEvidenceMatchesBinding(presentEmptyEvidence, presentEmpty) {
		t.Fatal("exact present-empty stdin evidence was rejected")
	}
	if stdinEvidenceMatchesBinding(presentEmptyEvidence, absent) {
		t.Fatal("present-empty stdin collapsed to absent stdin")
	}
	expected, err := PresentStdin([]byte("ab"))
	if err != nil {
		t.Fatal(err)
	}
	substitute, err := PresentStdin([]byte("cd"))
	if err != nil {
		t.Fatal(err)
	}
	substituteDigest, _, err := cliDigestBytes("CLIStdinBytes", substitute.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	substitutedEvidence := processStdinEvidence{
		presence: string(expected.Presence()), declaredBytes: 2, digest: substituteDigest,
		delivery: "PRESENT_EXPLICIT_PIPE_WRITER", pipeAllocated: true, writerStarted: true,
		handoffAttempted: true, writtenBytes: 2, complete: true, started: true, physicalEntered: true,
	}
	if stdinEvidenceMatchesBinding(substitutedEvidence, expected) {
		t.Fatal("same-length stdin substitution escaped the physical digest check")
	}
	notEntered := processStdinEvidence{
		presence: string(absent.Presence()), digest: absentDigest, delivery: "NOT_APPLIED",
	}
	if !stdinEvidenceMatchesBinding(notEntered, absent) {
		t.Fatal("declared stdin without physical execution was rejected")
	}
	notEntered.delivery = "ABSENT_NULL_DEVICE"
	if stdinEvidenceMatchesBinding(notEntered, absent) {
		t.Fatal("NOT_ENTERED stdin receipt claimed a delivery mechanism")
	}
	enteredUnstarted := processStdinEvidence{
		presence: string(presentEmpty.Presence()), digest: presentEmptyDigest,
		delivery: "NOT_APPLIED", pipeAllocated: true, physicalEntered: true,
	}
	if !stdinEvidenceMatchesBinding(enteredUnstarted, presentEmpty) {
		t.Fatal("failed Start could not retain pipe allocation without claiming stdin delivery")
	}
	enteredUnstarted.delivery = "PRESENT_EXPLICIT_PIPE_WRITER"
	if stdinEvidenceMatchesBinding(enteredUnstarted, presentEmpty) {
		t.Fatal("failed Start forged an explicit stdin writer delivery")
	}
	expectedDigest, _, err := cliDigestBytes("CLIStdinBytes", expected.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	exactEvidence := substitutedEvidence
	exactEvidence.digest = expectedDigest
	for name, mutate := range map[string]func(*processStdinEvidence){
		"pipe-not-allocated":    func(evidence *processStdinEvidence) { evidence.pipeAllocated = false },
		"writer-not-started":    func(evidence *processStdinEvidence) { evidence.writerStarted = false },
		"handoff-not-attempted": func(evidence *processStdinEvidence) { evidence.handoffAttempted = false },
	} {
		t.Run(name, func(t *testing.T) {
			mutated := exactEvidence
			mutate(&mutated)
			if stdinEvidenceMatchesBinding(mutated, expected) {
				t.Fatal("missing physical stdin edge was accepted")
			}
		})
	}
}

func TestCLIChannelCaptureDistinguishesExactCapOverflowAndEmpty(t *testing.T) {
	exact, err := adaptChannel(CLIChannelStdout, []byte("1234"), 4, false, 4, true)
	if err != nil {
		t.Fatal(err)
	}
	overflow, err := adaptChannel(CLIChannelStdout, []byte("1234"), 5, true, 4, true)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := adaptChannel(CLIChannelStdout, []byte{}, 0, false, 4, true)
	if err != nil {
		t.Fatal(err)
	}
	absent, err := adaptChannel(CLIChannelStdout, nil, 0, false, 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if exact.State() != ChannelPresent || overflow.State() != ChannelTruncated ||
		empty.State() != ChannelPresent || absent.State() != ChannelAbsent {
		t.Fatalf("unexpected channel states: exact=%s overflow=%s empty=%s absent=%s", exact.State(), overflow.State(), empty.State(), absent.State())
	}
	if exact.RetainedDigest() != overflow.RetainedDigest() {
		t.Fatal("same retained prefix should retain the same byte digest while state/count remain distinct")
	}
	if _, err := adaptChannel(CLIChannelStdout, []byte("1234"), 4, true, 4, true); err == nil {
		t.Fatal("overflow at, rather than beyond, the exact cap was accepted")
	}
	if _, err := adaptChannel(CLIChannelStdout, []byte("123"), 4, false, 4, true); err == nil {
		t.Fatal("complete channel with mismatched count was accepted")
	}
}

func TestCLIChannelCaptureKeepsStdoutAndStderrDigestDomainsDistinct(t *testing.T) {
	stdout, err := adaptChannel(CLIChannelStdout, []byte("same"), 4, false, 8, true)
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := adaptChannel(CLIChannelStderr, []byte("same"), 4, false, 8, true)
	if err != nil {
		t.Fatal(err)
	}
	if stdout.RetainedDigest() == stderr.RetainedDigest() {
		t.Fatal("stdout and stderr used the same channel digest domain")
	}
}

func TestFixtureInvocationReceiptIsExplicitlyTagged(t *testing.T) {
	present := CLIFixtureInvocationReceipt{
		presence: PresencePresent, digest: cliTestDigest("a"), status: world.CLIInvocationValidated,
		filePresence: world.CLIInvocationPresencePresent, validated: true,
	}
	absent := CLIFixtureInvocationReceipt{
		presence: PresenceAbsent, filePresence: world.CLIInvocationPresenceUnknown, reason: "PROCESS_NOT_STARTED",
	}
	if !present.Present() || absent.Present() || !present.valid() || !absent.valid() || !present.Validated() {
		t.Fatal("fixture invocation receipt did not retain tagged presence")
	}
	if _, ok := absent.Reason(); !ok {
		t.Fatal("absent invocation receipt lost its reason")
	}
	present.filePresence = world.CLIInvocationPresenceAbsent
	if present.valid() {
		t.Fatal("validated invocation receipt accepted contradictory file absence")
	}
	missing := CLIFixtureInvocationReceipt{
		presence: PresencePresent, digest: cliTestDigest("b"), status: world.CLIInvocationAbsent,
		filePresence: world.CLIInvocationPresenceAbsent,
	}
	if !missing.valid() || missing.FilePresent() || missing.FilePresence() != world.CLIInvocationPresenceAbsent {
		t.Fatal("missing invocation receipt lost exact absent file presence")
	}
	readFailed := CLIFixtureInvocationReceipt{
		presence: PresencePresent, digest: cliTestDigest("c"), status: world.CLIInvocationReadFailed,
		filePresence: world.CLIInvocationPresenceUnknown,
	}
	if !readFailed.valid() || readFailed.FilePresent() {
		t.Fatal("read-failed invocation receipt lost unknown file presence")
	}
}

func TestCLICapturedObservationDigestBackedChannelsReachDeclaredBoundary(t *testing.T) {
	definition := testProjectionDefinition(t, CLIFieldCompletionKind)
	stdout := bytes.Repeat([]byte("o"), maxCaptureBytes)
	stderr := bytes.Repeat([]byte("e"), maxCaptureBytes)
	observation := testObservation(
		t, definition, stdout, stderr, CLICompletion{kind: CompletionExited, code: 0}, nil, "f",
	)
	if !observation.Valid() || len(observation.CanonicalBytes()) >= 1<<20 {
		t.Fatal("declared channel boundary was embedded in, or invalidated, captured-observation identity")
	}
	tampered := observation
	tampered.stdout.bytes = append([]byte(nil), observation.stdout.bytes...)
	tampered.stdout.bytes[len(tampered.stdout.bytes)-1] ^= 0xff
	if tampered.Valid() {
		t.Fatal("same-length captured-byte substitution retained observation authority")
	}
}

func TestLateOutputOverflowRetainsEarlierPrimaryControl(t *testing.T) {
	truncated := CLIChannelCapture{name: CLIChannelStdout, state: ChannelTruncated}
	present := CLIChannelCapture{name: CLIChannelStderr, state: ChannelPresent}
	for _, primary := range []domain.ControlReason{
		domain.ControlOutputLimit,
		domain.ControlTimeout,
		domain.ControlCancelled,
		domain.ControlProbeTransportError,
		domain.ControlStartError,
	} {
		if !truncatedChannelsMatchPrimary(truncated, present, primary, true) {
			t.Fatalf("late overflow discarded earlier owner primary %s", primary)
		}
	}
	if truncatedChannelsMatchPrimary(truncated, present, "", false) {
		t.Fatal("truncated channel without a primary control was admitted")
	}
}
