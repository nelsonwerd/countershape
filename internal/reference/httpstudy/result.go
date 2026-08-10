package httpstudy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	evidenceArtifactSchema = "countershape/u7-study-artifact/v1"
	evidenceTrialSchema    = "countershape/u7-study-trial/v1"
	evidenceDomainHTTP     = "http"
	evidenceStatusGreen    = "GREEN"
	maximumEvidenceBytes   = 16 << 20
	maximumSafeJSONInteger = int64(1<<53 - 1)
)

// EvidencePayload is a presentation map supplied by the study after the
// corresponding core object has already been constructed and validated. The
// publisher canonicalizes the map but does not infer or change its semantics.
type EvidencePayload map[string]any

// DeterministicEvidencePayloads names the five semantic payloads whose exact
// wrapper bytes must repeat across all three harness runs.
type DeterministicEvidencePayloads struct {
	SourceSpec     EvidencePayload
	WorldPlan      EvidencePayload
	Ruling         EvidencePayload
	DecisionRecord EvidencePayload
	ContractBundle EvidencePayload
}

// FreshEvidencePayloads names eight current-run protocol slots. Six carry
// allocated physical or target facts; the two legacy run/classification slots
// carry explicit absence projections. Ordinal is added only by the fresh
// wrapper and is never added to a deterministic payload.
type FreshEvidencePayloads struct {
	WorldInstance      EvidencePayload
	Attempts           EvidencePayload
	Measurements       EvidencePayload
	Captures           EvidencePayload
	Confirmation       EvidencePayload
	TargetProjection   EvidencePayload
	ProcessObservation EvidencePayload
	StandaloneResult   EvidencePayload
}

// phaseTrialReceipt is derived only from the typed completed study. The
// authority is deliberately not copied into the frozen six-field trial
// artifact; it authorizes GREEN only after the exact roster is validated.
type phaseTrialReceipt struct {
	phase     string
	trial     int
	authority domain.Digest
}

type evidencePublicationInput struct {
	Root                 string
	Domain               string
	Ordinal              int
	PhysicalRunAuthority domain.Digest
	PhaseTrials          []phaseTrialReceipt
	Deterministic        DeterministicEvidencePayloads
	Fresh                FreshEvidencePayloads
}

// EvidenceManifestEntry is one terminally reopened file in relative-path
// order. SHA256 is the lowercase hexadecimal digest expected by the harness.
type EvidenceManifestEntry struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// EvidencePublication is a defensive snapshot of the exact tree that was
// reopened after publication. It is publication evidence, not a semantic
// verdict about the supplied core payloads.
type EvidencePublication struct {
	root           string
	ordinal        int
	manifest       []EvidenceManifestEntry
	manifestSHA256 string
	deterministic  map[string]string
	fresh          map[string]string
}

func (p EvidencePublication) Root() string { return p.root }

func (p EvidencePublication) Ordinal() int { return p.ordinal }

func (p EvidencePublication) Manifest() []EvidenceManifestEntry {
	return append([]EvidenceManifestEntry(nil), p.manifest...)
}

func (p EvidencePublication) ManifestSHA256() string { return p.manifestSHA256 }

func (p EvidencePublication) DeterministicSHA256() map[string]string {
	return cloneDigestMap(p.deterministic)
}

func (p EvidencePublication) FreshSHA256() map[string]string {
	return cloneDigestMap(p.fresh)
}

type evidencePublicationError struct {
	code   string
	detail string
}

func (e *evidencePublicationError) Error() string {
	return "U7_HTTP_EVIDENCE_" + e.code + ": " + e.detail
}

func evidenceFail(code, detail string) error {
	return &evidencePublicationError{code: code, detail: detail}
}

type namedEvidencePayload struct {
	field         string
	path          string
	payload       EvidencePayload
	deterministic bool
}

type directoryRecord struct {
	path string
	info os.FileInfo
}

type fileRecord struct {
	path   string
	exact  []byte
	info   os.FileInfo
	digest string
}

type deterministicArtifactLine struct {
	SchemaVersion string          `json:"schema_version"`
	Domain        string          `json:"domain"`
	Artifact      string          `json:"artifact"`
	Payload       json.RawMessage `json:"payload"`
}

type freshArtifactLine struct {
	SchemaVersion string          `json:"schema_version"`
	Domain        string          `json:"domain"`
	Ordinal       int             `json:"ordinal"`
	Artifact      string          `json:"artifact"`
	Payload       json.RawMessage `json:"payload"`
}

type trialArtifactLine struct {
	SchemaVersion string `json:"schema_version"`
	Domain        string `json:"domain"`
	Ordinal       int    `json:"ordinal"`
	Phase         string `json:"phase"`
	Trial         int    `json:"trial"`
	Status        string `json:"status"`
}

var evidenceDirectories = []string{
	"deterministic",
	"fresh",
	"phases",
	"phases/search",
	"phases/confirm",
	"phases/contract",
}

var trialPhaseCounts = []struct {
	phase string
	count int
}{
	{phase: "search", count: 80},
	{phase: "confirm", count: 8},
	{phase: "contract", count: 10},
}

// publishCompletedEvidence is the only product publication seam. Callers
// cannot supply payload maps, completion status, or phase hashes directly.
func publishCompletedEvidence(completed CompletedHTTPStudy, root string) (EvidencePublication, error) {
	input, err := evidenceInput(completed, root)
	if err != nil {
		return EvidencePublication{}, err
	}
	return publishEvidenceInput(input)
}

// publishEvidenceInput populates one harness-owned empty evidence directory.
// It is private so only the completed-study adapter and focused package tests
// can reach the lower-level filesystem protocol.
func publishEvidenceInput(input evidencePublicationInput) (EvidencePublication, error) {
	if input.Domain != evidenceDomainHTTP || input.Ordinal < 1 || input.Ordinal > 3 {
		return EvidencePublication{}, evidenceFail("INPUT", "domain or ordinal is outside the frozen HTTP protocol")
	}
	planned, deterministicDigests, freshDigests, err := planEvidenceFiles(input)
	if err != nil {
		return EvidencePublication{}, err
	}
	rootHandle, rootInfo, err := openEmptyEvidenceRoot(input.Root)
	if err != nil {
		return EvidencePublication{}, err
	}
	defer rootHandle.Close()

	directories := make([]directoryRecord, 0, len(evidenceDirectories))
	for _, relative := range evidenceDirectories {
		record, makeErr := createEvidenceDirectory(input.Root, relative)
		if makeErr != nil {
			return EvidencePublication{}, makeErr
		}
		directories = append(directories, record)
	}

	files := make(map[string]fileRecord, 111)
	plannedPaths := make([]string, 0, len(planned))
	for relative := range planned {
		plannedPaths = append(plannedPaths, relative)
	}
	sort.Strings(plannedPaths)
	for _, relative := range plannedPaths {
		record, writeErr := publishEvidenceFile(input.Root, relative, planned[relative])
		if writeErr != nil {
			return EvidencePublication{}, writeErr
		}
		files[relative] = record
	}
	manifest, terminalErr := reopenEvidenceTree(input.Root, rootHandle, rootInfo, directories, files)
	if terminalErr != nil {
		return EvidencePublication{}, terminalErr
	}
	manifestLine, err := canonicalJSONLine(manifest)
	if err != nil {
		return EvidencePublication{}, evidenceFail("ENCODE", "manifest")
	}
	return EvidencePublication{
		root: input.Root, ordinal: input.Ordinal,
		manifest: append([]EvidenceManifestEntry(nil), manifest...), manifestSHA256: digestHex(manifestLine),
		deterministic: cloneDigestMap(deterministicDigests), fresh: cloneDigestMap(freshDigests),
	}, nil
}

func planEvidenceFiles(input evidencePublicationInput) (map[string][]byte, map[string]string, map[string]string, error) {
	payloads := evidencePayloadRoster(input)
	if err := validatePhysicalRunAuthorityPayloads(input.PhysicalRunAuthority, payloads); err != nil {
		return nil, nil, nil, err
	}
	planned := make(map[string][]byte, 111)
	payloadBytes := make(map[string][]byte, len(payloads))
	payloadDigests := make(map[string]string, len(payloads))
	deterministicDigests := make(map[string]string, 5)
	freshDigests := make(map[string]string, 8)
	for _, entry := range payloads {
		rawPayload, err := canonicalEvidencePayload(entry.field, entry.payload)
		if err != nil {
			return nil, nil, nil, err
		}
		payloadDigest := digestHex(rawPayload)
		for priorField, prior := range payloadBytes {
			if bytes.Equal(prior, rawPayload) || payloadDigests[priorField] == payloadDigest {
				return nil, nil, nil, evidenceFail("PAYLOAD_ALIAS", entry.field+" duplicates "+priorField)
			}
		}
		payloadBytes[entry.field] = append([]byte(nil), rawPayload...)
		payloadDigests[entry.field] = payloadDigest

		var exact []byte
		if entry.deterministic {
			exact, err = canonicalJSONLine(deterministicArtifactLine{
				SchemaVersion: evidenceArtifactSchema,
				Domain:        input.Domain,
				Artifact:      entry.field,
				Payload:       rawPayload,
			})
		} else {
			exact, err = canonicalJSONLine(freshArtifactLine{
				SchemaVersion: evidenceArtifactSchema,
				Domain:        input.Domain,
				Ordinal:       input.Ordinal,
				Artifact:      entry.field,
				Payload:       rawPayload,
			})
		}
		if err != nil {
			return nil, nil, nil, evidenceFail("ENCODE", entry.field)
		}
		if _, duplicate := planned[entry.path]; duplicate {
			return nil, nil, nil, evidenceFail("ROSTER", "duplicate "+entry.path)
		}
		planned[entry.path] = exact
		if entry.deterministic {
			deterministicDigests[entry.field] = digestHex(exact)
		} else {
			freshDigests[entry.field] = digestHex(exact)
		}
	}

	trialFiles, err := planPhaseTrialFiles(input.Domain, input.Ordinal, input.PhaseTrials)
	if err != nil {
		return nil, nil, nil, err
	}
	for relative, exact := range trialFiles {
		if _, duplicate := planned[relative]; duplicate {
			return nil, nil, nil, evidenceFail("ROSTER", "duplicate "+relative)
		}
		planned[relative] = exact
	}
	if len(planned) != 111 {
		return nil, nil, nil, evidenceFail("ROSTER", fmt.Sprintf("constructed %d files", len(planned)))
	}
	if err := validateDigestSeparation(deterministicDigests, freshDigests); err != nil {
		return nil, nil, nil, err
	}
	return planned, deterministicDigests, freshDigests, nil
}

func validatePhysicalRunAuthorityPayloads(authority domain.Digest, payloads []namedEvidencePayload) error {
	if _, err := rawSHA256(authority); err != nil {
		return evidenceFail("PHYSICAL_RUN_AUTHORITY", "the completed run authority is invalid")
	}
	want := authority.String()
	for _, entry := range payloads {
		observed, present := entry.payload["physical_run_authority"]
		if entry.deterministic {
			if present {
				return evidenceFail("PHYSICAL_RUN_AUTHORITY", entry.field+" must remain run-independent")
			}
			continue
		}
		value, ok := observed.(string)
		if !present || !ok || value != want {
			return evidenceFail("PHYSICAL_RUN_AUTHORITY", entry.field+" is not bound to the completed physical run")
		}
	}
	return nil
}

func planPhaseTrialFiles(domain string, ordinal int, receipts []phaseTrialReceipt) (map[string][]byte, error) {
	expectedCount := 0
	for _, phase := range trialPhaseCounts {
		expectedCount += phase.count
	}
	if len(receipts) != expectedCount {
		return nil, evidenceFail("PHASE_RECEIPTS", fmt.Sprintf("received %d receipts, want %d", len(receipts), expectedCount))
	}

	planned := make(map[string][]byte, expectedCount)
	seenAuthorities := make(map[string]string, expectedCount)
	receiptIndex := 0
	for _, phase := range trialPhaseCounts {
		for trial := 1; trial <= phase.count; trial++ {
			receipt := receipts[receiptIndex]
			receiptIndex++
			relative := fmt.Sprintf("phases/%s/trial-%03d.json", phase.phase, trial)
			if receipt.phase != phase.phase || receipt.trial != trial {
				return nil, evidenceFail("PHASE_RECEIPTS", fmt.Sprintf("receipt %d is %s/%d, want %s/%d", receiptIndex, receipt.phase, receipt.trial, phase.phase, trial))
			}
			rawAuthority, authorityErr := rawSHA256(receipt.authority)
			if authorityErr != nil || !validAuthoritySHA256(rawAuthority) {
				return nil, evidenceFail("PHASE_AUTHORITY", relative+" lacks one nonzero lowercase SHA-256 authority")
			}
			if prior, duplicate := seenAuthorities[rawAuthority]; duplicate {
				return nil, evidenceFail("PHASE_AUTHORITY_ALIAS", relative+" duplicates "+prior)
			}
			seenAuthorities[rawAuthority] = relative

			exact, err := canonicalJSONLine(trialArtifactLine{
				SchemaVersion: evidenceTrialSchema,
				Domain:        domain,
				Ordinal:       ordinal,
				Phase:         phase.phase,
				Trial:         trial,
				Status:        evidenceStatusGreen,
			})
			if err != nil {
				return nil, evidenceFail("ENCODE", relative)
			}
			planned[relative] = exact
		}
	}
	return planned, nil
}

func validAuthoritySHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	nonzero := false
	for index := 0; index < len(value); index++ {
		character := value[index]
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
		if character != '0' {
			nonzero = true
		}
	}
	return nonzero
}

func evidencePayloadRoster(input evidencePublicationInput) []namedEvidencePayload {
	return []namedEvidencePayload{
		{field: "source_spec_sha256", path: "deterministic/source-spec.json", payload: input.Deterministic.SourceSpec, deterministic: true},
		{field: "world_plan_sha256", path: "deterministic/world-plan.json", payload: input.Deterministic.WorldPlan, deterministic: true},
		{field: "ruling_sha256", path: "deterministic/ruling.json", payload: input.Deterministic.Ruling, deterministic: true},
		{field: "decision_record_sha256", path: "deterministic/decision-record.json", payload: input.Deterministic.DecisionRecord, deterministic: true},
		{field: "contract_bundle_sha256", path: "deterministic/contract-bundle.json", payload: input.Deterministic.ContractBundle, deterministic: true},
		{field: "world_instance_sha256", path: "fresh/world-instance.json", payload: input.Fresh.WorldInstance},
		{field: "attempts_sha256", path: "fresh/attempts.json", payload: input.Fresh.Attempts},
		{field: "measurements_sha256", path: "fresh/measurements.json", payload: input.Fresh.Measurements},
		{field: "captures_sha256", path: "fresh/captures.json", payload: input.Fresh.Captures},
		{field: "confirmation_sha256", path: "fresh/confirmation.json", payload: input.Fresh.Confirmation},
		{field: "contract_execution_target_sha256", path: "fresh/contract-execution-target.json", payload: input.Fresh.TargetProjection},
		{field: "finalized_contract_run_sha256", path: "fresh/finalized-contract-run.json", payload: input.Fresh.ProcessObservation},
		{field: "contract_execution_sha256", path: "fresh/contract-execution.json", payload: input.Fresh.StandaloneResult},
	}
}

func openEmptyEvidenceRoot(root string) (*os.File, os.FileInfo, error) {
	if root == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, nil, evidenceFail("ROOT", "evidence root must be clean and absolute")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return nil, nil, evidenceFail("ROOT", "evidence root must exist without symlink indirection")
	}
	before, err := os.Lstat(root)
	if err != nil || !exactPrivateDirectory(before) {
		return nil, nil, evidenceFail("ROOT", "evidence root must be an existing exact 0700 directory")
	}
	handle, err := os.Open(root)
	if err != nil {
		return nil, nil, evidenceFail("ROOT", "evidence root could not be opened")
	}
	opened, err := handle.Stat()
	if err != nil || !os.SameFile(before, opened) || !exactPrivateDirectory(opened) {
		handle.Close()
		return nil, nil, evidenceFail("ROOT", "evidence root identity changed while opening")
	}
	entries, err := handle.ReadDir(-1)
	if err != nil || len(entries) != 0 {
		handle.Close()
		return nil, nil, evidenceFail("ROOT_NOT_EMPTY", "evidence root must be exactly empty")
	}
	return handle, opened, nil
}

func createEvidenceDirectory(root, relative string) (directoryRecord, error) {
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.Mkdir(absolute, 0o700); err != nil {
		return directoryRecord{}, evidenceFail("DIRECTORY", relative+" could not be created exclusively")
	}
	if err := os.Chmod(absolute, 0o700); err != nil {
		return directoryRecord{}, evidenceFail("DIRECTORY", relative+" mode could not be fixed")
	}
	info, err := os.Lstat(absolute)
	resolved, resolveErr := filepath.EvalSymlinks(absolute)
	if err != nil || resolveErr != nil || resolved != absolute || !exactPrivateDirectory(info) {
		return directoryRecord{}, evidenceFail("DIRECTORY", relative+" is not an exact private directory")
	}
	return directoryRecord{path: relative, info: info}, nil
}

func publishEvidenceFile(root, relative string, exact []byte) (fileRecord, error) {
	if len(exact) < 1 || len(exact) > maximumEvidenceBytes || exact[len(exact)-1] != '\n' || bytes.Count(exact, []byte{'\n'}) != 1 {
		return fileRecord{}, evidenceFail("FILE_BYTES", relative+" is not one bounded JSON line")
	}
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	file, err := os.OpenFile(absolute, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fileRecord{}, evidenceFail("FILE_CREATE", relative+" could not be created exclusively")
	}
	closeWith := func(result error) (fileRecord, error) {
		if closeErr := file.Close(); result == nil && closeErr != nil {
			result = evidenceFail("FILE_CLOSE", relative)
		}
		return fileRecord{}, result
	}
	if err := file.Chmod(0o600); err != nil {
		return closeWith(evidenceFail("FILE_MODE", relative))
	}
	written, err := file.Write(exact)
	if err != nil || written != len(exact) {
		return closeWith(evidenceFail("FILE_WRITE", relative))
	}
	if err := file.Sync(); err != nil {
		return closeWith(evidenceFail("FILE_SYNC", relative))
	}
	opened, err := file.Stat()
	if err != nil || !exactPrivateFile(opened) || opened.Size() != int64(len(exact)) {
		return closeWith(evidenceFail("FILE_STATE", relative))
	}
	if _, err := closeWith(nil); err != nil {
		return fileRecord{}, err
	}
	pathInfo, err := os.Lstat(absolute)
	if err != nil || !sameFileRecord(opened, pathInfo) || !exactPrivateFile(pathInfo) {
		return fileRecord{}, evidenceFail("FILE_STATE", relative+" changed after close")
	}
	return fileRecord{path: relative, exact: append([]byte(nil), exact...), info: pathInfo, digest: digestHex(exact)}, nil
}

func canonicalEvidencePayload(field string, payload EvidencePayload) ([]byte, error) {
	if len(payload) == 0 {
		return nil, evidenceFail("PAYLOAD", field+" must be a nonempty semantic object")
	}
	encoded, err := canonicalJSONLine(map[string]any(payload))
	if err != nil {
		return nil, evidenceFail("PAYLOAD", field+" could not be encoded")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, evidenceFail("PAYLOAD", field+" could not be reopened")
	}
	object, ok := decoded.(map[string]any)
	if !ok || len(object) == 0 {
		return nil, evidenceFail("PAYLOAD", field+" is not a nonempty object")
	}
	normalized, err := normalizeEvidenceValue(object)
	if err != nil {
		return nil, evidenceFail("PAYLOAD", field+": "+err.Error())
	}
	exact, err := canonicalJSONLine(normalized)
	if err != nil {
		return nil, evidenceFail("PAYLOAD", field+" normalization failed")
	}
	return bytes.TrimSuffix(exact, []byte{'\n'}), nil
}

func normalizeEvidenceValue(value any) (any, error) {
	switch typed := value.(type) {
	case nil, bool:
		return typed, nil
	case string:
		if !utf8.ValidString(typed) || bytes.ContainsRune([]byte(typed), '\u2028') || bytes.ContainsRune([]byte(typed), '\u2029') {
			return nil, fmt.Errorf("string is outside the exact cross-runtime JSON profile")
		}
		return typed, nil
	case json.Number:
		parsed, err := strconv.ParseFloat(typed.String(), 64)
		if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) || math.Trunc(parsed) != parsed || math.Abs(parsed) > float64(maximumSafeJSONInteger) {
			return nil, fmt.Errorf("number is not a safe JSON integer")
		}
		return int64(parsed), nil
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			normalized, err := normalizeEvidenceValue(item)
			if err != nil {
				return nil, err
			}
			result[index] = normalized
		}
		return result, nil
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if !utf8.ValidString(key) || bytes.ContainsRune([]byte(key), '\u2028') || bytes.ContainsRune([]byte(key), '\u2029') || isJSONArrayIndex(key) {
				return nil, fmt.Errorf("object key %q is outside the exact cross-runtime JSON profile", key)
			}
			normalized, err := normalizeEvidenceValue(item)
			if err != nil {
				return nil, err
			}
			result[key] = normalized
		}
		return result, nil
	default:
		return nil, fmt.Errorf("value has unsupported JSON type %T", value)
	}
}

func isJSONArrayIndex(value string) bool {
	if value == "0" {
		return true
	}
	if value == "" || value[0] == '0' {
		return false
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	return err == nil && parsed < 1<<32-1 && strconv.FormatUint(parsed, 10) == value
}

func canonicalJSONLine(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	exact := output.Bytes()
	if len(exact) < 2 || exact[len(exact)-1] != '\n' || bytes.Count(exact, []byte{'\n'}) != 1 {
		return nil, fmt.Errorf("encoded value is not one JSON line")
	}
	return append([]byte(nil), exact...), nil
}

func reopenEvidenceTree(
	root string,
	rootHandle *os.File,
	rootInfo os.FileInfo,
	directories []directoryRecord,
	files map[string]fileRecord,
) ([]EvidenceManifestEntry, error) {
	terminalRoot, err := rootHandle.Stat()
	pathRoot, pathErr := os.Lstat(root)
	if err != nil || pathErr != nil || !sameNode(rootInfo, terminalRoot) || !sameNode(terminalRoot, pathRoot) || !exactPrivateDirectory(pathRoot) {
		return nil, evidenceFail("TERMINAL_ROOT", "evidence root identity or mode changed")
	}
	observedDirectories, observedFiles, err := walkEvidenceTree(root)
	if err != nil {
		return nil, err
	}
	expectedDirectories := append([]string(nil), evidenceDirectories...)
	sort.Strings(expectedDirectories)
	expectedFiles := make([]string, 0, len(files))
	for path := range files {
		expectedFiles = append(expectedFiles, path)
	}
	sort.Strings(expectedFiles)
	if !reflect.DeepEqual(observedDirectories, expectedDirectories) || !reflect.DeepEqual(observedFiles, expectedFiles) {
		return nil, evidenceFail("TERMINAL_ROSTER", fmt.Sprintf("directories=%d files=%d", len(observedDirectories), len(observedFiles)))
	}
	for _, record := range directories {
		current, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(record.path)))
		if statErr != nil || !sameNode(record.info, current) || !exactPrivateDirectory(current) {
			return nil, evidenceFail("TERMINAL_DIRECTORY", record.path)
		}
	}
	manifest := make([]EvidenceManifestEntry, 0, len(expectedFiles))
	for _, relative := range expectedFiles {
		record := files[relative]
		exact, info, readErr := reopenEvidenceFile(filepath.Join(root, filepath.FromSlash(relative)))
		if readErr != nil || !sameFileRecord(record.info, info) || !bytes.Equal(exact, record.exact) || digestHex(exact) != record.digest {
			return nil, evidenceFail("TERMINAL_FILE", relative)
		}
		manifest = append(manifest, EvidenceManifestEntry{
			Path: relative, Mode: "0600", Bytes: len(exact), SHA256: record.digest,
		})
	}
	return manifest, nil
}

func walkEvidenceTree(root string) ([]string, []string, error) {
	directories := []string{}
	files := []string{}
	var walk func(string, string) error
	walk = func(absolute, prefix string) error {
		entries, err := os.ReadDir(absolute)
		if err != nil {
			return evidenceFail("TERMINAL_WALK", prefix)
		}
		for _, entry := range entries {
			relative := entry.Name()
			if prefix != "" {
				relative = prefix + "/" + entry.Name()
			}
			path := filepath.Join(absolute, entry.Name())
			info, err := os.Lstat(path)
			if err != nil || info.Mode()&os.ModeSymlink != 0 {
				return evidenceFail("TERMINAL_NONREGULAR", relative)
			}
			if info.IsDir() {
				if !exactPrivateDirectory(info) {
					return evidenceFail("TERMINAL_DIRECTORY", relative)
				}
				directories = append(directories, relative)
				if err := walk(path, relative); err != nil {
					return err
				}
				continue
			}
			if !exactPrivateFile(info) {
				return evidenceFail("TERMINAL_FILE", relative)
			}
			files = append(files, relative)
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return nil, nil, err
	}
	sort.Strings(directories)
	sort.Strings(files)
	return directories, files, nil
}

func reopenEvidenceFile(path string) ([]byte, os.FileInfo, error) {
	pathInfo, err := os.Lstat(path)
	if err != nil || !exactPrivateFile(pathInfo) || pathInfo.Size() < 1 || pathInfo.Size() > maximumEvidenceBytes {
		return nil, nil, evidenceFail("FILE_STATE", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, evidenceFail("FILE_OPEN", path)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !sameFileRecord(pathInfo, opened) {
		return nil, nil, evidenceFail("FILE_STATE", path)
	}
	exact, err := io.ReadAll(io.LimitReader(file, maximumEvidenceBytes+1))
	if err != nil || len(exact) < 1 || len(exact) > maximumEvidenceBytes || int64(len(exact)) != opened.Size() {
		return nil, nil, evidenceFail("FILE_READ", path)
	}
	after, err := file.Stat()
	terminalPath, pathErr := os.Lstat(path)
	if err != nil || pathErr != nil || !sameFileRecord(opened, after) || !sameFileRecord(after, terminalPath) {
		return nil, nil, evidenceFail("FILE_STATE", path)
	}
	return exact, terminalPath, nil
}

func validateDigestSeparation(deterministic, fresh map[string]string) error {
	seen := make(map[string]string, len(deterministic)+len(fresh))
	for field, digest := range deterministic {
		seen[digest] = field
	}
	for field, digest := range fresh {
		if prior, present := seen[digest]; present {
			return evidenceFail("DIGEST_ALIAS", field+" aliases "+prior)
		}
		seen[digest] = field
	}
	return nil
}

func exactPrivateDirectory(info os.FileInfo) bool {
	links, ok := linkCount(info)
	return info != nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o700 &&
		info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0 && ok && links >= 2
}

func exactPrivateFile(info os.FileInfo) bool {
	links, ok := linkCount(info)
	return info != nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm() == 0o600 &&
		info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0 && ok && links == 1
}

func linkCount(info os.FileInfo) (uint64, bool) {
	if info == nil || info.Sys() == nil {
		return 0, false
	}
	value := reflect.ValueOf(info.Sys())
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0, false
	}
	field := value.FieldByName("Nlink")
	if !field.IsValid() {
		return 0, false
	}
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() < 0 {
			return 0, false
		}
		return uint64(field.Int()), true
	default:
		return 0, false
	}
}

func sameNode(left, right os.FileInfo) bool {
	return left != nil && right != nil && os.SameFile(left, right) && left.Mode() == right.Mode()
}

func sameFileRecord(left, right os.FileInfo) bool {
	if !sameNode(left, right) || left.Size() != right.Size() || !left.ModTime().Equal(right.ModTime()) {
		return false
	}
	leftLinks, leftOK := linkCount(left)
	rightLinks, rightOK := linkCount(right)
	return leftOK && rightOK && leftLinks == rightLinks
}

func digestHex(exact []byte) string {
	digest := sha256.Sum256(exact)
	return hex.EncodeToString(digest[:])
}

func cloneDigestMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
