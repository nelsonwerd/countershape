package world

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	cliInvocationFilename  = "cli-invocation.json"
	maxCLIInvocationBytes  = 64 << 10
	cliInvocationAuthority = "U3_POST_TEARDOWN_REOPEN_BOUNDED_STRICT_INVOCATION_V1"
)

type CLIInvocationEvidenceStatus string

type CLIInvocationEvidencePresence string

const (
	CLIInvocationNotInspected    CLIInvocationEvidenceStatus = "NOT_INSPECTED_PRE_PROCESS"
	CLIInvocationValidated       CLIInvocationEvidenceStatus = "VALIDATED"
	CLIInvocationAbsent          CLIInvocationEvidenceStatus = "ABSENT"
	CLIInvocationNotRegular      CLIInvocationEvidenceStatus = "NOT_REGULAR"
	CLIInvocationTooLarge        CLIInvocationEvidenceStatus = "TOO_LARGE"
	CLIInvocationReadFailed      CLIInvocationEvidenceStatus = "READ_FAILED"
	CLIInvocationMalformed       CLIInvocationEvidenceStatus = "MALFORMED_STRICT_JSON"
	CLIInvocationSchemaMismatch  CLIInvocationEvidenceStatus = "SCHEMA_MISMATCH"
	CLIInvocationAttemptMismatch CLIInvocationEvidenceStatus = "ATTEMPT_ID_MISMATCH"
	CLIInvocationArgvMismatch    CLIInvocationEvidenceStatus = "LOGICAL_ARGV_MISMATCH"

	CLIInvocationPresenceAbsent  CLIInvocationEvidencePresence = "ABSENT"
	CLIInvocationPresencePresent CLIInvocationEvidencePresence = "PRESENT"
	CLIInvocationPresenceUnknown CLIInvocationEvidencePresence = "UNKNOWN"
)

// CLIInvocationEvidenceReceipt is a post-teardown observation of one fixed
// candidate-written evidence file. VALIDATED means the reopened bounded bytes
// were strict JSON containing exactly the allocated attempt ID and full logical
// argv. It is fixture evidence, not proof against a malicious candidate.
type CLIInvocationEvidenceReceipt struct {
	digest                 domain.Digest
	canonicalBytes         []byte
	executionBindingDigest domain.Digest
	path                   string
	status                 CLIInvocationEvidenceStatus
	presence               CLIInvocationEvidencePresence
	fileDigest             domain.Digest
	bytes                  int64
	attemptIDValidated     bool
	logicalArgvValidated   bool
	expectedAttemptID      string
	expectedLogicalArgv    []string
	authority              string
}

type cliInvocationEvidenceIdentity struct {
	SchemaVersion          string   `json:"schema_version"`
	Kind                   string   `json:"kind"`
	ExecutionBindingDigest string   `json:"execution_binding_digest"`
	Path                   string   `json:"path"`
	Status                 string   `json:"status"`
	Presence               string   `json:"presence"`
	FileDigest             string   `json:"file_digest"`
	Bytes                  int64    `json:"bytes"`
	ExpectedAttemptID      string   `json:"expected_attempt_id"`
	ExpectedLogicalArgv    []string `json:"expected_logical_argv"`
	AttemptIDValidated     bool     `json:"attempt_id_validated"`
	LogicalArgvValidated   bool     `json:"logical_argv_validated"`
	Authority              string   `json:"authority"`
	Nonclaim               string   `json:"nonclaim"`
}

func (r CLIInvocationEvidenceReceipt) Valid() bool {
	if !r.digest.Valid() || len(r.canonicalBytes) == 0 || !r.executionBindingDigest.Valid() ||
		!filepath.IsAbs(r.path) || r.status == "" || r.authority != cliInvocationAuthority ||
		r.expectedAttemptID == "" || len(r.expectedLogicalArgv) == 0 || !r.statusFactsValid() {
		return false
	}
	digest, canonicalBytes, err := canon.DigestTyped("CLIInvocationEvidenceReceipt", r.identity())
	return err == nil && digest.String() == r.digest.String() && bytes.Equal(canonicalBytes, r.canonicalBytes)
}

func (r CLIInvocationEvidenceReceipt) statusFactsValid() bool {
	noValidation := !r.attemptIDValidated && !r.logicalArgvValidated
	switch r.status {
	case CLIInvocationNotInspected:
		return r.presence == CLIInvocationPresenceUnknown && !r.fileDigest.Valid() && r.bytes == 0 && noValidation
	case CLIInvocationValidated:
		return r.presence == CLIInvocationPresencePresent && r.fileDigest.Valid() && r.bytes >= 0 &&
			r.attemptIDValidated && r.logicalArgvValidated
	case CLIInvocationAbsent:
		return r.presence == CLIInvocationPresenceAbsent && !r.fileDigest.Valid() && r.bytes == 0 && noValidation
	case CLIInvocationNotRegular:
		return r.presence == CLIInvocationPresencePresent && !r.fileDigest.Valid() && r.bytes >= 0 && noValidation
	case CLIInvocationTooLarge:
		return r.presence == CLIInvocationPresencePresent && !r.fileDigest.Valid() && r.bytes > maxCLIInvocationBytes && noValidation
	case CLIInvocationReadFailed:
		return (r.presence == CLIInvocationPresencePresent || r.presence == CLIInvocationPresenceUnknown) &&
			!r.fileDigest.Valid() && r.bytes >= 0 && noValidation
	case CLIInvocationMalformed, CLIInvocationSchemaMismatch:
		return r.presence == CLIInvocationPresencePresent && r.fileDigest.Valid() && r.bytes >= 0 && noValidation
	case CLIInvocationAttemptMismatch:
		return r.presence == CLIInvocationPresencePresent && r.fileDigest.Valid() && r.bytes >= 0 && !r.attemptIDValidated
	case CLIInvocationArgvMismatch:
		return r.presence == CLIInvocationPresencePresent && r.fileDigest.Valid() && r.bytes >= 0 &&
			r.attemptIDValidated && !r.logicalArgvValidated
	default:
		return false
	}
}

func (r CLIInvocationEvidenceReceipt) identity() cliInvocationEvidenceIdentity {
	return cliInvocationEvidenceIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CLIInvocationEvidenceReceipt",
		ExecutionBindingDigest: r.executionBindingDigest.String(), Path: r.path,
		Status: string(r.status), Presence: string(r.presence), FileDigest: r.fileDigest.String(), Bytes: r.bytes,
		ExpectedAttemptID: r.expectedAttemptID, ExpectedLogicalArgv: append([]string(nil), r.expectedLogicalArgv...),
		AttemptIDValidated: r.attemptIDValidated, LogicalArgvValidated: r.logicalArgvValidated,
		Authority: r.authority, Nonclaim: "CANDIDATE_WRITTEN_EVIDENCE_IS_NOT_MALICIOUS_PROCESS_ATTESTATION",
	}
}

func (r CLIInvocationEvidenceReceipt) Digest() domain.Digest { return r.digest }
func (r CLIInvocationEvidenceReceipt) CanonicalBytes() []byte {
	return append([]byte(nil), r.canonicalBytes...)
}
func (r CLIInvocationEvidenceReceipt) ExecutionBindingDigest() domain.Digest {
	return r.executionBindingDigest
}
func (r CLIInvocationEvidenceReceipt) Path() string                        { return r.path }
func (r CLIInvocationEvidenceReceipt) Status() CLIInvocationEvidenceStatus { return r.status }
func (r CLIInvocationEvidenceReceipt) Presence() CLIInvocationEvidencePresence {
	return r.presence
}
func (r CLIInvocationEvidenceReceipt) Present() bool {
	return r.presence == CLIInvocationPresencePresent
}
func (r CLIInvocationEvidenceReceipt) FileDigest() domain.Digest  { return r.fileDigest }
func (r CLIInvocationEvidenceReceipt) Bytes() int64               { return r.bytes }
func (r CLIInvocationEvidenceReceipt) AttemptIDValidated() bool   { return r.attemptIDValidated }
func (r CLIInvocationEvidenceReceipt) LogicalArgvValidated() bool { return r.logicalArgvValidated }
func (r CLIInvocationEvidenceReceipt) ExpectedAttemptID() string  { return r.expectedAttemptID }
func (r CLIInvocationEvidenceReceipt) ExpectedLogicalArgv() []string {
	return append([]string(nil), r.expectedLogicalArgv...)
}
func (r CLIInvocationEvidenceReceipt) Authority() string { return r.authority }

func inspectCLIInvocationEvidence(
	binding climodel.CLIExecutionBinding,
	evidenceRoot, attemptID string,
	logicalArgv []string,
) (CLIInvocationEvidenceReceipt, error) {
	if !binding.Valid() || !filepath.IsAbs(evidenceRoot) || attemptID == "" || len(logicalArgv) == 0 {
		return CLIInvocationEvidenceReceipt{}, refuse(CodeCLIExecutionRejected, "invocation evidence expectation is incomplete", nil)
	}
	path := filepath.Join(evidenceRoot, cliInvocationFilename)
	status := CLIInvocationAbsent
	presence := CLIInvocationPresenceAbsent
	var fileDigest domain.Digest
	var fileBytes int64
	attemptMatches := false
	argvMatches := false

	info, statErr := os.Lstat(path)
	if statErr == nil {
		presence = CLIInvocationPresencePresent
		switch {
		case !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0:
			status = CLIInvocationNotRegular
			fileBytes = info.Size()
		case info.Size() > maxCLIInvocationBytes:
			status = CLIInvocationTooLarge
			fileBytes = info.Size()
		default:
			status, fileDigest, fileBytes, attemptMatches, argvMatches = readAndValidateCLIInvocation(
				path, info.Size(), attemptID, logicalArgv,
			)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		presence = CLIInvocationPresenceUnknown
		status = CLIInvocationReadFailed
	}

	receipt := CLIInvocationEvidenceReceipt{
		executionBindingDigest: binding.Digest(), path: path, status: status, presence: presence,
		fileDigest: fileDigest, bytes: fileBytes, attemptIDValidated: attemptMatches,
		logicalArgvValidated: argvMatches, expectedAttemptID: attemptID,
		expectedLogicalArgv: append([]string(nil), logicalArgv...), authority: cliInvocationAuthority,
	}
	identity := receipt.identity()
	digest, canonicalBytes, err := canon.DigestTyped("CLIInvocationEvidenceReceipt", identity)
	if err != nil {
		return CLIInvocationEvidenceReceipt{}, err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return CLIInvocationEvidenceReceipt{}, err
	}
	receipt.digest = parsed
	receipt.canonicalBytes = append([]byte(nil), canonicalBytes...)
	if !receipt.Valid() {
		return CLIInvocationEvidenceReceipt{}, refuse(CodeCLIExecutionRejected, "constructed invocation evidence receipt is invalid", nil)
	}
	return receipt, nil
}

func readAndValidateCLIInvocation(
	path string,
	expectedSize int64,
	expectedAttemptID string,
	expectedArgv []string,
) (CLIInvocationEvidenceStatus, domain.Digest, int64, bool, bool) {
	handle, err := os.Open(path)
	if err != nil {
		return CLIInvocationReadFailed, "", 0, false, false
	}
	opened, statErr := handle.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || opened.Size() != expectedSize || opened.Size() > maxCLIInvocationBytes {
		_ = handle.Close()
		if opened != nil && opened.Size() > maxCLIInvocationBytes {
			return CLIInvocationTooLarge, "", opened.Size(), false, false
		}
		return CLIInvocationReadFailed, "", 0, false, false
	}
	bytes, readErr := io.ReadAll(io.LimitReader(handle, maxCLIInvocationBytes+1))
	closeErr := handle.Close()
	if readErr != nil || closeErr != nil || len(bytes) > maxCLIInvocationBytes || int64(len(bytes)) != expectedSize {
		return CLIInvocationReadFailed, "", int64(len(bytes)), false, false
	}
	digest, digestErr := canon.DigestBytes("CLIInvocationEvidenceBytes", bytes)
	if digestErr != nil {
		return CLIInvocationReadFailed, "", int64(len(bytes)), false, false
	}
	parsedDigest, parseDigestErr := domain.ParseDigest(digest.String())
	if parseDigestErr != nil {
		return CLIInvocationReadFailed, "", int64(len(bytes)), false, false
	}
	value, parseErr := canon.Parse(bytes)
	if parseErr != nil {
		return CLIInvocationMalformed, parsedDigest, int64(len(bytes)), false, false
	}
	members, object := value.Members()
	if !object || len(members) != 2 {
		return CLIInvocationSchemaMismatch, parsedDigest, int64(len(bytes)), false, false
	}
	attemptValue, hasAttempt := value.LookupMember("attempt_id")
	argvValue, hasArgv := value.LookupMember("logical_argv")
	if !hasAttempt || !hasArgv {
		return CLIInvocationSchemaMismatch, parsedDigest, int64(len(bytes)), false, false
	}
	actualAttempt, attemptIsString := attemptValue.Text()
	argvElements, argvIsArray := argvValue.Elements()
	if !attemptIsString || !argvIsArray || len(argvElements) != len(expectedArgv) {
		return CLIInvocationSchemaMismatch, parsedDigest, int64(len(bytes)), false, false
	}
	attemptMatches := actualAttempt == expectedAttemptID
	argvMatches := true
	for index, element := range argvElements {
		text, stringValue := element.Text()
		if !stringValue || text != expectedArgv[index] {
			argvMatches = false
		}
	}
	if !attemptMatches {
		return CLIInvocationAttemptMismatch, parsedDigest, int64(len(bytes)), false, argvMatches
	}
	if !argvMatches {
		return CLIInvocationArgvMismatch, parsedDigest, int64(len(bytes)), true, false
	}
	// MUTATION_ANCHOR: invocation-evidence-must-match-attempt-and-full-logical-argv
	return CLIInvocationValidated, parsedDigest, int64(len(bytes)), true, true
}
