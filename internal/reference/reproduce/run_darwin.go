package reproduce

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"sort"
	"sync"

	"github.com/nelsonwerd/countershape/internal/reference/app"
	"github.com/nelsonwerd/countershape/internal/reference/clistudy"
	"github.com/nelsonwerd/countershape/internal/reference/httpstudy"
)

const (
	// ClosureStatus names only a bounded artifact closure. It is not a core
	// Countershape classification, a didrun grade, or semantic reproduction.
	ClosureStatus = "BOUNDED_LOCAL_TWO_DOMAIN_TERMINAL_ARTIFACT_BYTE_CLOSURE"

	// These labels describe only facts established by the local Collector: two
	// matching byte projections per run after the retained Node runner closed.
	// They do not claim continuous immutability or inode identity. The U7
	// harness owns its separate six-process verdict and fallback.
	ClosureAuthority = "SIX_SUCCESSFUL_PACKAGE_HANDLER_VALIDATIONS_WITH_MATCHING_TERMINAL_ARTIFACT_BYTE_PROJECTIONS"
	AuthorityCeiling = "DOMAIN_SEMANTICS_DELEGATED_NO_INDEPENDENT_SEMANTIC_CLASSIFICATION_HARNESS_PROCESS_TOPOLOGY_EXECUTABLE_STDOUT_ENVIRONMENT_RESOURCE_OR_RECEIPT_AUTHORITY"
	HTTPAuthority    = "HTTP_SUCCESSFUL_PACKAGE_HANDLER_VALIDATIONS_AND_MATCHING_TERMINAL_ARTIFACT_BYTE_PROJECTIONS_ONLY"
	CLIAuthority     = "CLI_SUCCESSFUL_PACKAGE_HANDLER_VALIDATIONS_AND_MATCHING_TERMINAL_ARTIFACT_BYTE_PROJECTIONS_ONLY"

	studyRunCount            = 3
	evidenceFileCount        = 111
	maximumEvidenceFileBytes = int64(16 << 20)
	maximumEvidenceTreeBytes = int64(64 << 20)
	artifactSchema           = "countershape/u7-study-artifact/v1"
	trialSchema              = "countershape/u7-study-trial/v1"
	greenStatus              = "GREEN"
)

var exactUnreceipted = []string{
	"LINUX_UNRUN",
	"WINDOWS_UNRUN",
	"UNRUN_NODE_MAJORS",
	"BROAD_IMPORTED_REPOSITORY_BEHAVIOR_UNVALIDATED",
	"HOSTILE_CONTAINMENT_UNVALIDATED",
	"NETWORK_DENIAL_UNVALIDATED",
	"COMPREHENSION_UNVALIDATED",
	"REVIEW_COMPRESSION_UNVALIDATED",
	"ADOPTION_UNVALIDATED",
	"MAINTAINABILITY_UNVALIDATED",
	"PRODUCTION_READINESS_UNVALIDATED",
	"SECURITY_REVIEW_NOT_PERFORMED",
	"EXTERNAL_PLATFORM_BEHAVIOR_UNVALIDATED",
	"IMPORTED_REPOSITORY_TIMING_UNCLAIMED",
	"FULL_STUDY_RESOURCE_BOUND_UNVALIDATED",
	"HTTP_FINALIZED_CONTRACT_RUN_AUTHORITY_ABSENT",
	"HTTP_CONTRACT_EXECUTION_CLASSIFICATION_AUTHORITY_ABSENT",
	"TWO_DOMAIN_TARGET_RUN_CLASSIFICATION_REPRODUCTION_UNMET",
	"CLI_OFFICIAL_EXECUTION_GENERALIZATION_UNVALIDATED",
	"U7R_SELF_RECEIPT_ABSENT",
}

var deterministicArtifactPaths = map[string]string{
	"source_spec_sha256":     "deterministic/source-spec.json",
	"world_plan_sha256":      "deterministic/world-plan.json",
	"ruling_sha256":          "deterministic/ruling.json",
	"decision_record_sha256": "deterministic/decision-record.json",
	"contract_bundle_sha256": "deterministic/contract-bundle.json",
}

var freshArtifactPaths = map[string]string{
	"world_instance_sha256":            "fresh/world-instance.json",
	"attempts_sha256":                  "fresh/attempts.json",
	"measurements_sha256":              "fresh/measurements.json",
	"captures_sha256":                  "fresh/captures.json",
	"confirmation_sha256":              "fresh/confirmation.json",
	"contract_execution_target_sha256": "fresh/contract-execution-target.json",
	"finalized_contract_run_sha256":    "fresh/finalized-contract-run.json",
	"contract_execution_sha256":        "fresh/contract-execution.json",
}

var exactEvidenceDirectories = []string{
	"deterministic",
	"fresh",
	"phases",
	"phases/confirm",
	"phases/contract",
	"phases/search",
}

// ClosureError carries a stable refusal code without producing a partial
// closure.
type ClosureError struct {
	Code   string
	Detail string
}

func (err *ClosureError) Error() string {
	return "U7_REPRODUCE_" + err.Code + ": " + err.Detail
}

// ErrorCode returns a reproduce refusal code, or the empty string for an
// error not owned by this package.
func ErrorCode(err error) string {
	var closureErr *ClosureError
	if !errors.As(err, &closureErr) {
		return ""
	}
	return closureErr.Code
}

type closureSeal struct{ marker byte }

var issuedClosure = &closureSeal{marker: 1}

// RunSummary is a defensive byte-identity projection of one terminal tree.
type RunSummary struct {
	ordinal       int
	manifest      []app.EvidenceFile
	manifestSHA   string
	deterministic map[string]string
	fresh         map[string]string
}

func (summary RunSummary) Ordinal() int           { return summary.ordinal }
func (summary RunSummary) ManifestSHA256() string { return summary.manifestSHA }

func (summary RunSummary) Manifest() []app.EvidenceFile {
	return append([]app.EvidenceFile(nil), summary.manifest...)
}

func (summary RunSummary) DeterministicSHA256() map[string]string {
	return maps.Clone(summary.deterministic)
}

func (summary RunSummary) FreshSHA256() map[string]string {
	return maps.Clone(summary.fresh)
}

// DomainSummary keeps each package-handler boundary visible while carrying
// only independently re-opened byte facts.
type DomainSummary struct {
	domain    string
	authority string
	runs      []RunSummary
}

func (summary DomainSummary) Domain() string     { return summary.domain }
func (summary DomainSummary) Authority() string  { return summary.authority }
func (summary DomainSummary) Runs() []RunSummary { return cloneRuns(summary.runs) }

// Closure is immutable outside this package. Valid replays its bounded shape;
// it does not claim the six separate subject processes shared one lifetime.
type Closure struct {
	status      string
	authority   string
	ceiling     string
	domains     []DomainSummary
	unreceipted []string
	seal        *closureSeal
}

func (closure Closure) Status() string    { return closure.status }
func (closure Closure) Authority() string { return closure.authority }
func (closure Closure) Ceiling() string   { return closure.ceiling }

func (closure Closure) Domains() []DomainSummary {
	result := make([]DomainSummary, len(closure.domains))
	for index, domain := range closure.domains {
		result[index] = DomainSummary{domain: domain.domain, authority: domain.authority, runs: cloneRuns(domain.runs)}
	}
	return result
}

func (closure Closure) Unreceipted() []string {
	return append([]string(nil), closure.unreceipted...)
}

func (closure Closure) Valid() bool {
	if closure.seal != issuedClosure || closure.status != ClosureStatus || closure.authority != ClosureAuthority ||
		closure.ceiling != AuthorityCeiling ||
		!slices.Equal(closure.unreceipted, exactUnreceipted) || len(closure.domains) != 2 ||
		closure.domains[0].domain != "http" || closure.domains[0].authority != HTTPAuthority ||
		closure.domains[1].domain != "cli" || closure.domains[1].authority != CLIAuthority {
		return false
	}
	for _, domain := range closure.domains {
		if len(domain.runs) != studyRunCount {
			return false
		}
		for index, run := range domain.runs {
			if run.ordinal != index+1 || !validRunSummary(run) {
				return false
			}
		}
	}
	return validateCrossRunRelations(closure.domains[0].runs, closure.domains[1].runs) == nil
}

// Studies installs the unchanged domain handlers behind one additional
// descriptor-relative terminal byte check. The wrapper validates one run; the
// external U7 harness, not this process, owns the actual 3+3 aggregation.
func Studies() map[string]app.StudyHandler {
	return NewCollector().Studies()
}

// Collector is an opaque per-composition owner for a complete 3+3 closure. It
// has no caller-settable evidence fields: only its wrapped handlers can admit
// app-minted terminal snapshots. A caller that keeps one Collector across six
// app.Run invocations can request the combined Closure; the external U7
// harness instead runs six processes and owns its own aggregate evidence.
type Collector struct {
	state *collectorState
}

type collectorState struct {
	mu       sync.Mutex
	http     [studyRunCount]RunSummary
	cli      [studyRunCount]RunSummary
	httpSeen [studyRunCount]bool
	cliSeen  [studyRunCount]bool
	closed   bool
}

// NewCollector returns one empty, independent collector.
func NewCollector() *Collector { return &Collector{state: &collectorState{}} }

// Studies returns the exact two package-owned handlers wrapped by this
// collector. Every call returns a defensive map; the handlers intentionally
// share only this Collector's bounded state.
func (collector *Collector) Studies() map[string]app.StudyHandler {
	return collector.studiesWithHandlers(clistudy.Handler(), httpstudy.Handler())
}

func (collector *Collector) studiesWithHandlers(cli, http app.StudyHandler) map[string]app.StudyHandler {
	return map[string]app.StudyHandler{
		"cli":  closeAfterHandler(collector, "cli", cli),
		"http": closeAfterHandler(collector, "http", http),
	}
}

func closeAfterHandler(collector *Collector, domain string, handler app.StudyHandler) app.StudyHandler {
	return app.CloseStudy(handler, func(request app.StudyRequest, snapshot app.StudyEvidenceSnapshot) (app.StudyTerminalCommit, error) {
		if request.Domain != domain || request.Ordinal < 1 || request.Ordinal > studyRunCount {
			return nil, handlerRefusal(refuse("HANDLER_INPUT", "installed study request differs from its frozen domain boundary"))
		}
		run, err := closeEvidenceRun(domain, request.Ordinal, snapshot)
		if err != nil {
			return nil, handlerRefusal(err)
		}
		return func() error {
			return collector.publish(domain, request.Ordinal, run)
		}, nil
	})
}

func (collector *Collector) capture(domain string, ordinal int, snapshot app.StudyEvidenceSnapshot) error {
	if collector == nil {
		return refuse("COLLECTOR", "study collector is absent")
	}
	run, err := closeEvidenceRun(domain, ordinal, snapshot)
	if err != nil {
		return err
	}
	return collector.publish(domain, ordinal, run)
}

func (collector *Collector) publish(domain string, ordinal int, run RunSummary) error {
	if collector == nil || collector.state == nil {
		return refuse("COLLECTOR", "study collector is absent")
	}
	if (domain != "http" && domain != "cli") || ordinal < 1 || ordinal > studyRunCount ||
		run.ordinal != ordinal || !validRunSummary(run) {
		return refuse("INPUT", "prepared study run is invalid")
	}
	state := collector.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.closed {
		return refuse("COLLECTOR_CLOSED", "study collector already produced its terminal closure")
	}
	index := ordinal - 1
	if domain == "http" {
		if state.httpSeen[index] {
			return refuse("DUPLICATE", fmt.Sprintf("http ordinal %d was already collected", ordinal))
		}
		state.http[index] = cloneRunSummary(run)
		state.httpSeen[index] = true
		return nil
	}
	if domain != "cli" {
		return refuse("INPUT", "study domain is invalid")
	}
	if state.cliSeen[index] {
		return refuse("DUPLICATE", fmt.Sprintf("cli ordinal %d was already collected", ordinal))
	}
	state.cli[index] = cloneRunSummary(run)
	state.cliSeen[index] = true
	return nil
}

// Close consumes one complete Collector and returns its immutable mixed-
// authority artifact closure. It is an inert local join, not the U7 harness
// event. Semantic-looking payloads stay opaque; the domain packages own
// meaning, and no process or receipt authority is manufactured here.
func (collector *Collector) Close() (Closure, error) {
	if collector == nil || collector.state == nil {
		return Closure{}, refuse("COLLECTOR", "study collector is absent")
	}
	state := collector.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.closed {
		return Closure{}, refuse("COLLECTOR_CLOSED", "study collector already produced its terminal closure")
	}
	if !allSeen(state.httpSeen) || !allSeen(state.cliSeen) {
		return Closure{}, refuse("ROSTER", "expected exactly three HTTP and three CLI snapshots")
	}
	httpRuns := cloneRuns(state.http[:])
	cliRuns := cloneRuns(state.cli[:])
	if err := validateCrossRunRelations(httpRuns, cliRuns); err != nil {
		return Closure{}, err
	}

	closure := Closure{
		status: ClosureStatus, authority: ClosureAuthority, ceiling: AuthorityCeiling,
		domains: []DomainSummary{
			{domain: "http", authority: HTTPAuthority, runs: cloneRuns(httpRuns)},
			{domain: "cli", authority: CLIAuthority, runs: cloneRuns(cliRuns)},
		},
		unreceipted: append([]string(nil), exactUnreceipted...), seal: issuedClosure,
	}
	if !closure.Valid() {
		return Closure{}, refuse("INTERNAL", "constructed closure failed its defensive replay")
	}
	state.closed = true
	return closure, nil
}

type manifestEntry struct {
	Path   string `json:"path"`
	Mode   string `json:"mode"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
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

func closeEvidenceRun(domain string, ordinal int, snapshot app.StudyEvidenceSnapshot) (RunSummary, error) {
	if (domain != "http" && domain != "cli") || ordinal < 1 || ordinal > studyRunCount {
		return RunSummary{}, refuse("INPUT", "domain or ordinal is invalid")
	}
	if !slices.Equal(snapshot.Directories(), exactEvidenceDirectories) || snapshot.TotalBytes() < 1 ||
		snapshot.TotalBytes() > maximumEvidenceTreeBytes {
		return RunSummary{}, refuse("ROSTER", fmt.Sprintf("%s run %d directory or byte roster differs from protocol", domain, ordinal))
	}
	files := snapshot.Files()
	expectedPaths := expectedEvidencePaths()
	if len(files) != evidenceFileCount {
		return RunSummary{}, refuse("ROSTER", fmt.Sprintf("%s run %d does not contain 111 files", domain, ordinal))
	}

	manifest := make([]manifestEntry, len(files))
	exactByPath := make(map[string][]byte, len(files))
	var observedTotal int64
	for index, file := range files {
		exact := file.ExactBytes()
		if file.Path() != expectedPaths[index] || file.Mode() != "0600" || file.Bytes() < 1 ||
			file.Bytes() > maximumEvidenceFileBytes || int64(len(exact)) != file.Bytes() || !validSHA256(file.SHA256()) ||
			digestBytes(exact) != file.SHA256() {
			return RunSummary{}, refuse("FILE", fmt.Sprintf("%s run %d file %d differs from snapshot identity", domain, ordinal, index))
		}
		observedTotal += file.Bytes()
		manifest[index] = manifestEntry{Path: file.Path(), Mode: file.Mode(), Bytes: file.Bytes(), SHA256: file.SHA256()}
		exactByPath[file.Path()] = exact
	}
	if observedTotal != snapshot.TotalBytes() || len(exactByPath) != evidenceFileCount {
		return RunSummary{}, refuse("TREE_BYTES", fmt.Sprintf("%s run %d byte sum or paths differ", domain, ordinal))
	}

	deterministic, fresh, err := validateArtifactWrappers(domain, ordinal, exactByPath)
	if err != nil {
		return RunSummary{}, err
	}
	manifestLine, err := encodeCanonicalLine(manifest)
	if err != nil {
		return RunSummary{}, refuse("MANIFEST", fmt.Sprintf("%s run %d cannot encode", domain, ordinal))
	}
	filesOut := make([]app.EvidenceFile, len(manifest))
	for index, entry := range manifest {
		filesOut[index] = app.EvidenceFile{Path: entry.Path, Bytes: entry.Bytes, SHA256: entry.SHA256}
	}
	return RunSummary{
		ordinal: ordinal, manifest: filesOut, manifestSHA: digestBytes(manifestLine),
		deterministic: deterministic, fresh: fresh,
	}, nil
}

func validateArtifactWrappers(domain string, ordinal int, exactByPath map[string][]byte) (map[string]string, map[string]string, error) {
	deterministic := make(map[string]string, len(deterministicArtifactPaths))
	for field, path := range deterministicArtifactPaths {
		var line deterministicArtifactLine
		if err := decodeCanonicalLine(exactByPath[path], &line); err != nil || line.SchemaVersion != artifactSchema ||
			line.Domain != domain || line.Artifact != field || !opaqueObject(line.Payload) {
			return nil, nil, refuse("ARTIFACT", fmt.Sprintf("%s run %d deterministic %s has no exact wrapper", domain, ordinal, field))
		}
		deterministic[field] = digestBytes(exactByPath[path])
	}
	fresh := make(map[string]string, len(freshArtifactPaths))
	for field, path := range freshArtifactPaths {
		var line freshArtifactLine
		if err := decodeCanonicalLine(exactByPath[path], &line); err != nil || line.SchemaVersion != artifactSchema ||
			line.Domain != domain || line.Ordinal != ordinal || line.Artifact != field || !opaqueObject(line.Payload) {
			return nil, nil, refuse("ARTIFACT", fmt.Sprintf("%s run %d fresh %s has no exact wrapper", domain, ordinal, field))
		}
		fresh[field] = digestBytes(exactByPath[path])
	}
	for _, phase := range []struct {
		name  string
		count int
	}{{"search", 80}, {"confirm", 8}, {"contract", 10}} {
		for trial := 1; trial <= phase.count; trial++ {
			path := fmt.Sprintf("phases/%s/trial-%03d.json", phase.name, trial)
			var line trialArtifactLine
			if err := decodeCanonicalLine(exactByPath[path], &line); err != nil || line.SchemaVersion != trialSchema ||
				line.Domain != domain || line.Ordinal != ordinal || line.Phase != phase.name || line.Trial != trial || line.Status != greenStatus {
				return nil, nil, refuse("TRIAL", fmt.Sprintf("%s run %d %s/%d is not exact GREEN", domain, ordinal, phase.name, trial))
			}
		}
	}
	return deterministic, fresh, nil
}

func decodeCanonicalLine(exact []byte, target any) error {
	if len(exact) < 3 || exact[len(exact)-1] != '\n' || bytes.Count(exact, []byte{'\n'}) != 1 {
		return errors.New("not one newline-terminated JSON value")
	}
	body := exact[:len(exact)-1]
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("additional JSON value")
	}
	reencoded, err := encodeCanonicalLine(target)
	if err != nil || !bytes.Equal(reencoded[:len(reencoded)-1], body) {
		return errors.New("JSON is not the exact canonical wrapper")
	}
	return nil
}

func encodeCanonicalLine(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	exact := output.Bytes()
	if len(exact) == 0 || exact[len(exact)-1] != '\n' || bytes.Count(exact, []byte{'\n'}) != 1 {
		return nil, errors.New("value is not one canonical JSON line")
	}
	return append([]byte(nil), exact...), nil
}

func opaqueObject(value json.RawMessage) bool {
	if len(value) < 2 || value[0] != '{' || value[len(value)-1] != '}' || !json.Valid(value) {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(value, &object) == nil && len(object) > 0
}

func validateCrossRunRelations(httpRuns, cliRuns []RunSummary) error {
	for _, runs := range [][]RunSummary{httpRuns, cliRuns} {
		baseline := runs[0].deterministic
		for _, run := range runs[1:] {
			if !maps.Equal(baseline, run.deterministic) {
				return refuse("DETERMINISTIC", "deterministic wrapper bytes changed within one domain")
			}
		}
	}
	deterministic := make(map[string]string, len(deterministicArtifactPaths)*2)
	for domainIndex, run := range []RunSummary{httpRuns[0], cliRuns[0]} {
		domain := []string{"http", "cli"}[domainIndex]
		for field, digest := range run.deterministic {
			if prior, exists := deterministic[digest]; exists {
				return refuse("DOMAIN_ALIAS", field+" aliases "+prior+" across domains")
			}
			deterministic[digest] = domain + ":" + field
		}
	}
	fresh := make(map[string]string, len(freshArtifactPaths)*studyRunCount*2)
	manifestDigests := make(map[string]string, studyRunCount*2)
	for _, domain := range []struct {
		name string
		runs []RunSummary
	}{{"http", httpRuns}, {"cli", cliRuns}} {
		for _, run := range domain.runs {
			if prior, exists := manifestDigests[run.manifestSHA]; exists {
				return refuse("FRESHNESS", fmt.Sprintf("%s run %d manifest aliases %s", domain.name, run.ordinal, prior))
			}
			manifestDigests[run.manifestSHA] = fmt.Sprintf("%s:%d", domain.name, run.ordinal)
			for field, digest := range run.fresh {
				if prior, exists := deterministic[digest]; exists {
					return refuse("FRESHNESS", field+" aliases deterministic "+prior)
				}
				if prior, exists := fresh[digest]; exists {
					return refuse("FRESHNESS", field+" aliases "+prior)
				}
				fresh[digest] = fmt.Sprintf("%s:%d:%s", domain.name, run.ordinal, field)
			}
		}
	}
	return nil
}

func validRunSummary(run RunSummary) bool {
	if run.ordinal < 1 || run.ordinal > studyRunCount || !validSHA256(run.manifestSHA) ||
		len(run.manifest) != evidenceFileCount || !exactDigestKeys(run.deterministic, deterministicArtifactPaths) ||
		!exactDigestKeys(run.fresh, freshArtifactPaths) {
		return false
	}
	expectedPaths := expectedEvidencePaths()
	manifest := make([]manifestEntry, len(run.manifest))
	digestByPath := make(map[string]string, len(run.manifest))
	var total int64
	for index, file := range run.manifest {
		if file.Path != expectedPaths[index] || file.Bytes < 1 || file.Bytes > maximumEvidenceFileBytes ||
			!validSHA256(file.SHA256) {
			return false
		}
		total += file.Bytes
		if total > maximumEvidenceTreeBytes {
			return false
		}
		manifest[index] = manifestEntry{Path: file.Path, Mode: "0600", Bytes: file.Bytes, SHA256: file.SHA256}
		digestByPath[file.Path] = file.SHA256
	}
	if len(digestByPath) != evidenceFileCount {
		return false
	}
	manifestLine, err := encodeCanonicalLine(manifest)
	if err != nil || digestBytes(manifestLine) != run.manifestSHA {
		return false
	}
	for field, path := range deterministicArtifactPaths {
		if run.deterministic[field] != digestByPath[path] {
			return false
		}
	}
	for field, path := range freshArtifactPaths {
		if run.fresh[field] != digestByPath[path] {
			return false
		}
	}
	return true
}

func allSeen(seen [studyRunCount]bool) bool {
	for _, present := range seen {
		if !present {
			return false
		}
	}
	return true
}

func expectedEvidencePaths() []string {
	paths := make([]string, 0, evidenceFileCount)
	for _, path := range deterministicArtifactPaths {
		paths = append(paths, path)
	}
	for _, path := range freshArtifactPaths {
		paths = append(paths, path)
	}
	for _, phase := range []struct {
		name  string
		count int
	}{{"search", 80}, {"confirm", 8}, {"contract", 10}} {
		for trial := 1; trial <= phase.count; trial++ {
			paths = append(paths, fmt.Sprintf("phases/%s/trial-%03d.json", phase.name, trial))
		}
	}
	sort.Strings(paths)
	return paths
}

func exactDigestKeys(values, paths map[string]string) bool {
	if len(values) != len(paths) {
		return false
	}
	for key := range paths {
		if !validSHA256(values[key]) {
			return false
		}
	}
	return true
}

func digestBytes(exact []byte) string {
	digest := sha256.Sum256(exact)
	return hex.EncodeToString(digest[:])
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

func cloneRunSummary(input RunSummary) RunSummary {
	return RunSummary{
		ordinal: input.ordinal, manifest: append([]app.EvidenceFile(nil), input.manifest...), manifestSHA: input.manifestSHA,
		deterministic: maps.Clone(input.deterministic), fresh: maps.Clone(input.fresh),
	}
}

func cloneRuns(input []RunSummary) []RunSummary {
	result := make([]RunSummary, len(input))
	for index, run := range input {
		result[index] = cloneRunSummary(run)
	}
	return result
}

func refuse(code, detail string) error {
	return &ClosureError{Code: code, Detail: detail}
}

func handlerRefusal(err error) error {
	if err == nil {
		return nil
	}
	return &app.InputError{Code: "STUDY_EVIDENCE_CLOSURE_REFUSED", Detail: err.Error()}
}
