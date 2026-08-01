//go:build darwin

package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	httpmodel "github.com/nelsonwerd/countershape/internal/adapters/http/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	invocationValidated        = "VALIDATED"
	invocationAbsent           = "ABSENT"
	invocationUnexpected       = "UNEXPECTED"
	invocationFileFacts        = "FILE_FACTS_MISMATCH"
	invocationTooLarge         = "TOO_LARGE"
	invocationReadFailed       = "READ_FAILED"
	invocationMalformed        = "MALFORMED_STRICT_JSON"
	invocationSchemaMismatch   = "SCHEMA_MISMATCH"
	invocationAttemptMismatch  = "ATTEMPT_ID_MISMATCH"
	invocationStimulusMismatch = "STIMULUS_DIGEST_MISMATCH"
	invocationRequestMismatch  = "REQUEST_BINDING_MISMATCH"
	invocationCountMismatch    = "INVOCATION_COUNT_MISMATCH"
	darwinOpenNoFollowAny      = 0x20000000
)

type invocationFinding struct {
	Status                   string
	Presence                 string
	FileSHA256               string
	Bytes                    int64
	AttemptValidated         bool
	StimulusValidated        bool
	RequestBytesValidated    bool
	RequestDigestValidated   bool
	InvocationCountValidated bool
}

func (finding invocationFinding) validated() bool {
	return finding.Status == invocationValidated && finding.Presence == "PRESENT" &&
		len(finding.FileSHA256) == sha256.Size*2 &&
		finding.AttemptValidated && finding.StimulusValidated &&
		finding.RequestBytesValidated && finding.RequestDigestValidated &&
		finding.InvocationCountValidated
}

func (finding invocationFinding) contradictsBinding() bool {
	switch finding.Status {
	case invocationUnexpected, invocationAttemptMismatch, invocationStimulusMismatch,
		invocationRequestMismatch, invocationCountMismatch:
		return true
	default:
		return false
	}
}

func (finding invocationFinding) facts() map[string]any {
	return map[string]any{
		"status":                     finding.Status,
		"presence":                   finding.Presence,
		"file_sha256":                finding.FileSHA256,
		"bytes":                      finding.Bytes,
		"attempt_id_validated":       finding.AttemptValidated,
		"stimulus_digest_validated":  finding.StimulusValidated,
		"request_bytes_validated":    finding.RequestBytesValidated,
		"request_digest_validated":   finding.RequestDigestValidated,
		"invocation_count_validated": finding.InvocationCountValidated,
		"candidate_written_evidence_is_process_attestation": false,
	}
}

func inspectAndRetireInvocation(
	ctx context.Context,
	root, markerPath, attemptID string,
	stimulusDigest domain.Digest,
	request httpmodel.HTTPRequestWire,
	authority receiptAuthority,
) (invocationFinding, error) {
	if ctx == nil || !authority.valid() || !validRuntimeAttemptID(attemptID) || !stimulusDigest.Valid() {
		return invocationFinding{}, errors.New("HTTP invocation expectation is incomplete")
	}
	path := filepath.Join(root, invocationFilename)
	finding := invocationFinding{Status: invocationAbsent, Presence: "ABSENT"}
	info, statErr := os.Lstat(path)
	expectedNames := []string{filepath.Base(markerPath)}
	if statErr == nil {
		expectedNames = append(expectedNames, filepath.Base(path))
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return invocationFinding{}, statErr
	}
	if err := validateEvidenceRoster(ctx, root, authority.root, expectedNames); err != nil {
		return invocationFinding{}, err
	}
	marker, markerErr := os.Lstat(markerPath)
	if markerErr != nil || marker.Mode() != authority.marker.Mode() || !os.SameFile(authority.marker, marker) {
		return invocationFinding{}, errors.Join(markerErr, errors.New("attempt marker identity changed before receipt inspection"))
	}
	if statErr == nil {
		finding.Presence = "PRESENT"
		switch {
		case !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
			info.Mode().Perm() != 0o600 || info.Size() < 1:
			finding.Status = invocationFileFacts
			finding.Bytes = info.Size()
		case info.Size() > maxInvocationBytes:
			finding.Status = invocationTooLarge
			finding.Bytes = info.Size()
		case !request.Valid():
			finding.Status = invocationUnexpected
			finding.Bytes = info.Size()
		default:
			finding = readInvocation(path, info, attemptID, stimulusDigest, request)
		}
	}
	if err := retireInvocation(ctx, root, markerPath, path, authority, info); err != nil {
		return invocationFinding{}, err
	}
	return finding, nil
}

func readInvocation(
	path string,
	expected os.FileInfo,
	attemptID string,
	stimulusDigest domain.Digest,
	request httpmodel.HTTPRequestWire,
) invocationFinding {
	failure := func(status string, count int64, digest string) invocationFinding {
		return invocationFinding{Status: status, Presence: "PRESENT", FileSHA256: digest, Bytes: count}
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return failure(invocationReadFailed, 0, "")
	}
	handle := os.NewFile(uintptr(fd), path)
	if handle == nil {
		_ = syscall.Close(fd)
		return failure(invocationReadFailed, 0, "")
	}
	opened, openErr := handle.Stat()
	if openErr != nil || !opened.Mode().IsRegular() || opened.Size() != expected.Size() ||
		opened.Size() > maxInvocationBytes || !os.SameFile(expected, opened) {
		_ = handle.Close()
		return failure(invocationReadFailed, 0, "")
	}
	body, readErr := io.ReadAll(io.LimitReader(handle, maxInvocationBytes+1))
	after, afterErr := handle.Stat()
	closeErr := handle.Close()
	pathAfter, pathErr := os.Lstat(path)
	if readErr != nil || afterErr != nil || closeErr != nil || pathErr != nil ||
		int64(len(body)) != opened.Size() || !os.SameFile(opened, after) ||
		!os.SameFile(opened, pathAfter) {
		return failure(invocationReadFailed, int64(len(body)), "")
	}
	sum := sha256.Sum256(body)
	digestText := hex.EncodeToString(sum[:])
	value, parseErr := canon.Parse(body)
	if parseErr != nil {
		return failure(invocationMalformed, int64(len(body)), digestText)
	}
	canonical, canonicalErr := value.CanonicalChecked()
	members, object := value.Members()
	if canonicalErr != nil || !bytes.Equal(canonical, body) || !object || len(members) != 7 {
		return failure(invocationSchemaMismatch, int64(len(body)), digestText)
	}
	attemptValue, hasAttempt := value.LookupMember("attempt_id")
	countValue, hasCount := value.LookupMember("invocation_count")
	kindValue, hasKind := value.LookupMember("kind")
	requestCountValue, hasRequestCount := value.LookupMember("request_byte_count")
	requestDigestValue, hasRequestDigest := value.LookupMember("request_byte_sha256")
	schemaValue, hasSchema := value.LookupMember("schema_version")
	stimulusValue, hasStimulus := value.LookupMember("stimulus_digest")
	attempt, attemptOK := attemptValue.Text()
	count, countOK := countValue.Int64()
	kind, kindOK := kindValue.Text()
	requestCount, requestCountOK := requestCountValue.Int64()
	requestDigest, requestDigestOK := requestDigestValue.Text()
	schema, schemaOK := schemaValue.Text()
	stimulus, stimulusOK := stimulusValue.Text()
	if !hasAttempt || !hasCount || !hasKind || !hasRequestCount || !hasRequestDigest ||
		!hasSchema || !hasStimulus || !attemptOK || !countOK || !kindOK ||
		!requestCountOK || !requestDigestOK || !schemaOK || !stimulusOK ||
		kind != "HTTPFixtureInvocationEvidence" || schema != domain.SchemaVersion {
		return failure(invocationSchemaMismatch, int64(len(body)), digestText)
	}
	finding := failure(invocationValidated, int64(len(body)), digestText)
	finding.AttemptValidated = attempt == attemptID
	finding.StimulusValidated = stimulus == stimulusDigest.String()
	finding.RequestBytesValidated = requestCount == int64(request.ByteLength())
	finding.RequestDigestValidated = requestDigest == "sha256:"+request.RawSHA256()
	finding.InvocationCountValidated = count == 1
	switch {
	case !finding.AttemptValidated:
		finding.Status = invocationAttemptMismatch
	case !finding.StimulusValidated:
		finding.Status = invocationStimulusMismatch
	case !finding.RequestBytesValidated || !finding.RequestDigestValidated:
		finding.Status = invocationRequestMismatch
	case !finding.InvocationCountValidated:
		finding.Status = invocationCountMismatch
	}
	return finding
}

func retireInvocation(
	ctx context.Context,
	root, markerPath, path string,
	authority receiptAuthority,
	observed os.FileInfo,
) error {
	if observed != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		current, err := os.Lstat(path)
		if err != nil || !os.SameFile(observed, current) || current.Mode() != observed.Mode() ||
			current.Size() != observed.Size() {
			return errors.Join(err, errors.New("invocation receipt changed before retirement"))
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		directory, err := os.Open(root)
		if err != nil {
			return err
		}
		syncErr := directory.Sync()
		closeErr := directory.Close()
		if syncErr != nil || closeErr != nil {
			return errors.Join(syncErr, closeErr)
		}
	}
	if err := validateEvidenceRoster(ctx, root, authority.root, []string{filepath.Base(markerPath)}); err != nil {
		return err
	}
	marker, markerErr := os.Lstat(markerPath)
	if markerErr != nil || marker.Mode() != authority.marker.Mode() || !os.SameFile(authority.marker, marker) {
		return errors.Join(markerErr, errors.New("attempt marker changed during receipt retirement"))
	}
	return nil
}

func validateEvidenceRoster(
	ctx context.Context,
	root string,
	expectedRoot os.FileInfo,
	expectedNames []string,
) error {
	if ctx == nil {
		return errors.New("evidence roster context is absent")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(expectedNames) < 1 || len(expectedNames) > 2 {
		return errors.New("evidence roster expectation is outside the closed profile")
	}
	before, err := os.Lstat(root)
	if err != nil || !before.IsDir() || before.Mode() != expectedRoot.Mode() ||
		!os.SameFile(expectedRoot, before) {
		return errors.Join(err, errors.New("evidence root identity changed"))
	}
	fd, err := syscall.Open(
		root,
		syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|syscall.O_NONBLOCK|darwinOpenNoFollowAny,
		0,
	)
	if err != nil {
		return errors.Join(err, errors.New("evidence root descriptor did not open"))
	}
	handle := os.NewFile(uintptr(fd), root)
	if handle == nil {
		_ = syscall.Close(fd)
		return errors.New("evidence root descriptor could not be owned")
	}
	opened, openErr := handle.Stat()
	if openErr != nil || opened.Mode() != before.Mode() || !os.SameFile(before, opened) {
		_ = handle.Close()
		return errors.Join(openErr, errors.New("evidence root descriptor identity differs"))
	}
	expected := append([]string(nil), expectedNames...)
	sort.Strings(expected)
	entries, readErr := handle.ReadDir(len(expected) + 1)
	tail, tailErr := handle.ReadDir(1)
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(root)
	if len(entries) != len(expected) ||
		(readErr != nil && !errors.Is(readErr, io.EOF)) ||
		len(tail) != 0 || !errors.Is(tailErr, io.EOF) ||
		descriptorErr != nil || closeErr != nil || pathErr != nil ||
		afterDescriptor.Mode() != opened.Mode() || afterPath.Mode() != opened.Mode() ||
		!os.SameFile(opened, afterDescriptor) || !os.SameFile(opened, afterPath) {
		return errors.Join(
			readErr, tailErr, descriptorErr, closeErr, pathErr,
			errors.New("evidence root bounded roster or identity differs"),
		)
	}
	observed := make([]string, len(entries))
	for index, entry := range entries {
		if entry.Name() == "" || entry.IsDir() {
			return errors.New("evidence root roster differs")
		}
		observed[index] = entry.Name()
	}
	sort.Strings(observed)
	for index := range expected {
		if observed[index] != expected[index] ||
			(index > 0 && expected[index] == expected[index-1]) {
			return errors.New("evidence root roster differs")
		}
	}
	return nil
}
