// Package report renders one inert, default-minimized local presentation from
// server-issued immutable artifacts. It never derives conformance authority.
package report

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/server"
)

const (
	Warning        = "CONFIDENTIALITY NOT ESTABLISHED"
	maxReportBytes = 2 << 20
	maxVisibleText = 4096
)

type blindField struct {
	FieldID, Tag, Text  string
	Boolean             bool
	CanonicalJSONBase64 string `json:"canonical_json_base64"`
}
type blindCard struct {
	Alias  string       `json:"alias"`
	Fields []blindField `json:"fields"`
}
type reductionFacts struct {
	GradeStatus         string   `json:"grade_status"`
	ProposalLimit       uint64   `json:"proposal_limit"`
	CandidateTrialLimit uint64   `json:"candidate_trial_limit"`
	WallLimitMS         int64    `json:"wall_limit_ms"`
	EvaluationCount     int      `json:"evaluation_count"`
	UnresolvedCount     int      `json:"unresolved_count"`
	Limitations         []string `json:"limitations"`
	ReducerSetDigest    string   `json:"reducer_set_digest"`
	FinalSweepState     string   `json:"final_sweep_state"`
}
type projectionOperation struct {
	Name       string `json:"name"`
	RuleDigest string `json:"rule_digest"`
}
type blindPayload struct {
	ChoicepointDigest       string                `json:"choicepoint_digest"`
	Scenario                string                `json:"scenario"`
	Scope                   string                `json:"scope"`
	OriginalStimulusBase64  string                `json:"original_stimulus_base64"`
	MinimizedStimulusBase64 string                `json:"minimized_stimulus_base64"`
	Reduction               reductionFacts        `json:"reduction"`
	DiscoveryRepeats        int                   `json:"discovery_repeats_per_candidate"`
	ConfirmationRepeats     int                   `json:"confirmation_repeats_per_candidate"`
	ProjectionMode          string                `json:"projection_mode"`
	ProjectionOperations    []projectionOperation `json:"projection_operations"`
	SelectableFields        []string              `json:"selectable_fields"`
	DifferingFields         []string              `json:"differing_fields"`
	Cards                   []blindCard           `json:"cards"`
	TrustWarnings           []string              `json:"trust_warnings"`
}
type decisionResult struct {
	DecisionDigest     string   `json:"decision_digest"`
	Action             string   `json:"action"`
	SelectedFields     []string `json:"selected_fields"`
	NonassertedFields  []string `json:"nonasserted_fields"`
	EarlyReveal        bool     `json:"early_reveal"`
	ChangedAfterReveal bool     `json:"changed_after_reveal"`
	ChangeRationale    string   `json:"change_rationale"`
	CompilationStatus  string   `json:"compilation_status"`
	Authority          string   `json:"authority"`
	AuthorityCeiling   string   `json:"authority_ceiling"`
}
type benchPayload struct {
	SchemaVersion    string          `json:"schema_version"`
	Kind             string          `json:"kind"`
	StudyID          string          `json:"study_id"`
	Presentation     string          `json:"presentation_state"`
	StudyState       string          `json:"study_state"`
	CandidateState   string          `json:"candidate_state"`
	ChoicepointState string          `json:"choicepoint_state"`
	Summary          string          `json:"summary"`
	NextAction       string          `json:"next_action"`
	TrustWarning     string          `json:"trust_warning"`
	NetworkMode      string          `json:"network_mode"`
	Blind            json.RawMessage `json:"blind"`
	Result           *decisionResult `json:"result"`
	Nonclaims        []string        `json:"nonclaims"`
}
type receipt struct {
	Authority string `json:"authority"`
	Grade     string `json:"grade_verbatim"`
	Commit    string `json:"commit_oid"`
	Command   string `json:"command_digest"`
}
type decisionPayload struct {
	ActorID               string    `json:"actor_id"`
	ActorAttribution      string    `json:"actor_attribution"`
	ActorAuthenticity     string    `json:"actor_authenticity"`
	HumanAnnotation       string    `json:"human_annotation"`
	DidrunReceipts        []receipt `json:"didrun_receipts"`
	ReceiptStatus         string    `json:"receipt_status_when_empty"`
	ReceiptInterpretation string    `json:"receipt_interpretation"`
}
type exclusion struct {
	Classification string `json:"classification"`
}
type revealPayload struct {
	Exclusions []exclusion `json:"exclusions"`
}

type selectedOutcome struct {
	Alias  string
	Fields []fieldValue
}
type fieldValue struct{ ID, Value string }
type receiptView struct{ Authority, Grade, Commit, Command string }
type preview struct{ Digest, Kind, Body string }
type rawArtifact struct{ Name, Digest, Body string }
type view struct {
	Warning, StudyID, Presentation, StudyState, CandidateState, ChoicepointState, Scenario, ChoicepointDigest, DecisionDigest, Action, CompilationStatus string
	Summary, NextAction, Scope, ProjectionMode, Actor, ActorAuthority, Annotation, ContractDigest, CurrentConformance                                    string
	DiscoveryRepeats, ConfirmationRepeats, OutcomeCount, ExclusionCount                                                                                  int
	Reduction                                                                                                                                            reductionFacts
	Operations                                                                                                                                           []projectionOperation
	Selected                                                                                                                                             []selectedOutcome
	Exclusions                                                                                                                                           []string
	SelectedFields, NonassertedFields, TrustWarnings, Nonclaims                                                                                          []string
	Original, Minimized                                                                                                                                  preview
	Receipts                                                                                                                                             []receiptView
	ReceiptEmpty                                                                                                                                         string
	IncludeRaw                                                                                                                                           bool
	Raw                                                                                                                                                  []rawArtifact
}

// Render builds deterministic HTML bytes from one exact server snapshot.
func Render(snapshot server.ExportSnapshot, includeRaw bool) ([]byte, error) {
	if !snapshot.Valid() {
		return nil, errors.New("REPORT_SOURCE_SNAPSHOT_REFUSED")
	}
	var bench benchPayload
	if err := strictJSON(snapshot.BenchBytes(), &bench); err != nil || bench.Result == nil {
		return nil, errors.New("REPORT_BENCH_REFUSED")
	}
	var blind blindPayload
	if err := strictJSON(bench.Blind, &blind); err != nil {
		return nil, errors.New("REPORT_BLIND_REFUSED")
	}
	var decision decisionPayload
	if err := strictJSON(snapshot.DecisionBytes(), &decision); err != nil {
		return nil, errors.New("REPORT_DECISION_REFUSED")
	}
	var reveal revealPayload
	if err := strictJSON(snapshot.RevealBytes(), &reveal); err != nil {
		return nil, errors.New("REPORT_REVEAL_REFUSED")
	}
	original, err := stimulusPreview(blind.OriginalStimulusBase64)
	if err != nil {
		return nil, err
	}
	minimized, err := stimulusPreview(blind.MinimizedStimulusBase64)
	if err != nil {
		return nil, err
	}
	w := view{
		Warning: Warning, StudyID: visible(bench.StudyID), Presentation: visible(bench.Presentation), StudyState: visible(bench.StudyState), CandidateState: visible(bench.CandidateState), ChoicepointState: visible(bench.ChoicepointState),
		Scenario: visible(blind.Scenario), ChoicepointDigest: visible(blind.ChoicepointDigest), DecisionDigest: visible(bench.Result.DecisionDigest), Action: visible(bench.Result.Action), CompilationStatus: visible(bench.Result.CompilationStatus),
		Summary: visible(bench.Summary), NextAction: visible(bench.NextAction), Scope: visible(blind.Scope), ProjectionMode: visible(blind.ProjectionMode), Actor: visible(decision.ActorID), ActorAuthority: visible(decision.ActorAuthenticity), Annotation: visible(decision.HumanAnnotation),
		ContractDigest: "UNRECEIPTED", CurrentConformance: "UNRECEIPTED — no emitted contract or execution result is present in this package-issued presentation fixture.",
		DiscoveryRepeats: blind.DiscoveryRepeats, ConfirmationRepeats: blind.ConfirmationRepeats, OutcomeCount: len(blind.Cards), ExclusionCount: len(reveal.Exclusions), Reduction: blind.Reduction, Operations: blind.ProjectionOperations,
		SelectedFields: visibleList(bench.Result.SelectedFields), NonassertedFields: visibleList(bench.Result.NonassertedFields), TrustWarnings: visibleList(blind.TrustWarnings), Nonclaims: visibleList(bench.Nonclaims), Original: original, Minimized: minimized, IncludeRaw: includeRaw,
	}
	selected := map[string]struct{}{}
	for _, id := range bench.Result.SelectedFields {
		selected[id] = struct{}{}
	}
	for _, card := range blind.Cards {
		out := selectedOutcome{Alias: visible(card.Alias), Fields: []fieldValue{}}
		for _, field := range card.Fields {
			if _, ok := selected[field.FieldID]; ok {
				out.Fields = append(out.Fields, fieldValue{visible(field.FieldID), redactSelected(fieldText(field))})
			}
		}
		w.Selected = append(w.Selected, out)
	}
	for _, item := range reveal.Exclusions {
		w.Exclusions = append(w.Exclusions, visible(item.Classification))
	}
	if len(w.Exclusions) == 0 {
		w.Exclusions = []string{"EMPTY — every candidate in this fixture remained eligible."}
	}
	if len(decision.DidrunReceipts) == 0 {
		w.ReceiptEmpty = visible(decision.ReceiptStatus)
		if w.ReceiptEmpty == "" {
			w.ReceiptEmpty = "UNRECEIPTED"
		}
	} else {
		for _, item := range decision.DidrunReceipts {
			w.Receipts = append(w.Receipts, receiptView{visible(item.Authority), visible(item.Grade), visible(item.Commit), visible(item.Command)})
		}
	}
	if includeRaw {
		w.Raw = []rawArtifact{{"Studio bench", digest(snapshot.BenchBytes()), visible(string(snapshot.BenchBytes()))}, {"Choicepoint", digest(snapshot.ChoicepointBytes()), visible(string(snapshot.ChoicepointBytes()))}, {"Reveal", digest(snapshot.RevealBytes()), visible(string(snapshot.RevealBytes()))}, {"DecisionRecord", digest(snapshot.DecisionBytes()), visible(string(snapshot.DecisionBytes()))}}
	}
	var output bytes.Buffer
	if err := reportTemplate.Execute(&output, w); err != nil || output.Len() > maxReportBytes {
		return nil, errors.New("REPORT_OUTPUT_LIMIT_REFUSED")
	}
	return output.Bytes(), nil
}

func strictJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}

func stimulusPreview(encoded string) (preview, error) {
	body, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(body) == 0 || len(body) > 1<<20 {
		return preview{}, errors.New("REPORT_STIMULUS_REFUSED")
	}
	var value struct {
		Kind        string                            `json:"kind"`
		Argv        []string                          `json:"argv"`
		Environment []struct{ Name, Presence string } `json:"environment"`
		Fixtures    []struct {
			Path, Mode, ContentsDigest string
			ContentsBytes              int `json:"contents_bytes"`
		} `json:"fixtures"`
		Stdin struct {
			Presence   string `json:"presence"`
			ByteLength int    `json:"byte_length"`
		} `json:"stdin"`
	}
	if err := json.Unmarshal(body, &value); err != nil {
		return preview{}, errors.New("REPORT_STIMULUS_REFUSED")
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("argv_items=%d", len(value.Argv)), fmt.Sprintf("environment_names=%d", len(value.Environment)), fmt.Sprintf("fixture_files=%d", len(value.Fixtures)), "redaction=countershape/report-preview/v1")
	for _, item := range value.Environment {
		lines = append(lines, "env "+visible(item.Name)+" presence="+visible(item.Presence)+" value=OMITTED")
	}
	for _, item := range value.Fixtures {
		lines = append(lines, fmt.Sprintf("fixture %s mode=%s bytes=%d digest=%s body=OMITTED", visible(item.Path), visible(item.Mode), item.ContentsBytes, visible(item.ContentsDigest)))
	}
	lines = append(lines, fmt.Sprintf("stdin presence=%s bytes=%d body=OMITTED", visible(value.Stdin.Presence), value.Stdin.ByteLength))
	return preview{Digest: digest(body), Kind: visible(value.Kind), Body: strings.Join(lines, "\n")}, nil
}

func fieldText(field blindField) string {
	if field.CanonicalJSONBase64 != "" {
		body, err := base64.StdEncoding.Strict().DecodeString(field.CanonicalJSONBase64)
		if err != nil {
			return "INVALID_CANONICAL_VALUE"
		}
		return string(body)
	}
	if field.Tag == "BOOLEAN" {
		return fmt.Sprintf("%t", field.Boolean)
	}
	return field.Text
}

var secretPattern = regexp.MustCompile(`(?i)(?:\b(?:sk|ghp|github_pat|xox[baprs]|token|bearer|password|secret)[-_:= ]+[a-z0-9+/=_-]{8,}|\bAKIA[A-Z0-9]{16}\b)`)

func redactSelected(value string) string {
	if secretPattern.MatchString(value) {
		return "REDACTED_KNOWN_PATTERN digest=" + digest([]byte(value))
	}
	return visible(value)
}
func digest(body []byte) string {
	value := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(value[:])
}
func visibleList(items []string) []string {
	result := make([]string, len(items))
	for i := range items {
		result[i] = visible(items[i])
	}
	return result
}
func visible(value string) string {
	original := value
	var result strings.Builder
	for len(value) > 0 && result.Len() < maxVisibleText {
		r, size := utf8.DecodeRuneInString(value)
		value = value[size:]
		if r == utf8.RuneError && size == 1 {
			result.WriteString("⟦INVALID-UTF8⟧")
			continue
		}
		if unicode.IsControl(r) || r == '\u202a' || r == '\u202b' || r == '\u202d' || r == '\u202e' || r == '\u2066' || r == '\u2067' || r == '\u2068' || r == '\u2069' {
			result.WriteString(fmt.Sprintf("⟦U+%04X⟧", r))
			continue
		}
		result.WriteRune(r)
	}
	if value != "" {
		result.WriteString("⟦TRUNCATED sha256=")
		result.WriteString(strings.TrimPrefix(digest([]byte(original)), "sha256:"))
		result.WriteString("⟧")
	}
	return result.String()
}

// WriteExclusive publishes bytes through an exclusive same-directory hard-link
// commit. Existing paths, symlinks, noncanonical parents, and overwrite races
// are refused. The resulting regular file is mode 0600 with link count one.
func WriteExclusive(name string, body []byte) error {
	if name == "" || len(body) == 0 || len(body) > maxReportBytes {
		return errors.New("REPORT_OUTPUT_REFUSED")
	}
	abs, err := filepath.Abs(name)
	if err != nil || filepath.Clean(abs) != abs {
		return errors.New("REPORT_PATH_REFUSED")
	}
	requestedParent := filepath.Dir(abs)
	requestedInfo, requestedErr := os.Lstat(requestedParent)
	if requestedErr != nil || !requestedInfo.IsDir() || requestedInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("REPORT_PATH_REFUSED")
	}
	parent, err := filepath.EvalSymlinks(requestedParent)
	if err != nil {
		return errors.New("REPORT_PATH_REFUSED")
	}
	abs = filepath.Join(parent, filepath.Base(abs))
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("REPORT_PATH_REFUSED")
	}
	if _, err := os.Lstat(abs); !errors.Is(err, os.ErrNotExist) {
		return errors.New("REPORT_OVERWRITE_REFUSED")
	}
	temp, err := os.CreateTemp(parent, ".countershape-report-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	cleanup := func() { _ = temp.Close(); _ = os.Remove(tempName) }
	defer cleanup()
	if err := temp.Chmod(0o600); err != nil {
		return err
	}
	n, writeErr := temp.Write(body)
	syncErr := temp.Sync()
	closeErr := temp.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil || n != len(body) {
		return errors.Join(writeErr, syncErr, closeErr)
	}
	if err := os.Link(tempName, abs); err != nil {
		return fmt.Errorf("REPORT_ATOMIC_PUBLICATION_REFUSED: %w", err)
	}
	if err := os.Remove(tempName); err != nil {
		_ = os.Remove(abs)
		return err
	}
	directory, err := os.Open(parent)
	if err != nil {
		_ = os.Remove(abs)
		return err
	}
	directorySyncErr := directory.Sync()
	directoryCloseErr := directory.Close()
	if directorySyncErr != nil || directoryCloseErr != nil {
		_ = os.Remove(abs)
		return errors.Join(directorySyncErr, directoryCloseErr)
	}
	result, err := os.Lstat(abs)
	if err != nil {
		_ = os.Remove(abs)
		return errors.New("REPORT_TERMINAL_REOPEN_REFUSED")
	}
	stat, statOK := result.Sys().(*syscall.Stat_t)
	if !result.Mode().IsRegular() || result.Mode().Perm() != 0o600 || !statOK || stat.Nlink != 1 {
		_ = os.Remove(abs)
		return errors.New("REPORT_TERMINAL_REOPEN_REFUSED")
	}
	return nil
}

// RunCLI implements the explicit local export command.
func RunCLI(args []string, stdout, stderr io.Writer) int {
	var output string
	var includeRaw, acknowledge, danger bool
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--output":
			i++
			if i >= len(args) || output != "" {
				fmt.Fprintln(stderr, "countershape export: --output requires one path")
				return 2
			}
			output = args[i]
		case "--include-raw":
			includeRaw = true
		case "--acknowledge-confidentiality-not-established":
			acknowledge = true
		case "--i-understand-raw-may-contain-secrets":
			danger = true
		default:
			fmt.Fprintf(stderr, "countershape export: unknown argument %q\n", args[i])
			return 2
		}
	}
	if output == "" || !acknowledge {
		fmt.Fprintf(stderr, "countershape export: %s; pass --output PATH --acknowledge-confidentiality-not-established\n", Warning)
		return 2
	}
	if includeRaw != danger {
		fmt.Fprintln(stderr, "countershape export: raw export requires both --include-raw and --i-understand-raw-may-contain-secrets")
		return 2
	}
	snapshot, err := server.BuildSeedExportSnapshot()
	if err != nil {
		fmt.Fprintln(stderr, "countershape export: package snapshot refused")
		return 1
	}
	body, err := Render(snapshot, includeRaw)
	if err != nil {
		fmt.Fprintln(stderr, "countershape export: report rendering refused")
		return 1
	}
	if err := WriteExclusive(output, body); err != nil {
		fmt.Fprintf(stderr, "countershape export: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Countershape report written (%s).\n", Warning)
	return 0
}

var reportTemplate = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"><title>Countershape local evidence report</title><style>
:root{color-scheme:light dark;--bg:#f4f1e8;--ink:#171c22;--muted:#58616d;--panel:#fffdf7;--line:#c8c1b2;--accent:#174f45;--warn:#7a241c}*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:16px/1.55 ui-monospace,SFMono-Regular,Menlo,monospace}main{max-width:1120px;margin:auto;padding:32px}.mast{border:3px solid var(--warn);background:var(--panel);padding:24px}.warning{font-weight:900;color:var(--warn);letter-spacing:.04em;font-size:clamp(1.3rem,4vw,2.5rem)}h1,h2{line-height:1.12}section{margin:22px 0;border:1px solid var(--line);background:var(--panel);padding:20px}.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.fact{border-left:4px solid var(--accent);padding-left:12px}.label{color:var(--muted);font-size:.78rem;text-transform:uppercase;letter-spacing:.08em}.value,pre{overflow-wrap:anywhere;word-break:break-word;white-space:pre-wrap}table{width:100%;border-collapse:collapse}th,td{text-align:left;vertical-align:top;border-bottom:1px solid var(--line);padding:8px}code{overflow-wrap:anywhere}details{margin:12px 0}@media(max-width:600px){main{padding:12px}.grid{grid-template-columns:1fr}section,.mast{padding:14px}}@media print{body{background:#fff;color:#000}main{max-width:none;padding:0}.mast,section{break-inside:avoid;background:#fff}details{display:block}details>summary{display:none}}@media(forced-colors:active){.mast,section,.fact{border-color:CanvasText}}
</style></head><body><main><header class="mast"><div class="warning">{{.Warning}}</div><h1>Local exact-witness evidence report</h1><p>This inert file minimizes a known fixture profile. It is not safe, sanitized, anonymous, shareable, or a security receipt.</p></header>
<section><h2>Study and choice authority</h2><div class="grid"><div class="fact"><div class="label">Study</div><div class="value">{{.StudyID}} · {{.Presentation}} · {{.StudyState}}</div></div><div class="fact"><div class="label">Choicepoint</div><div class="value">{{.ChoicepointDigest}}</div></div><div class="fact"><div class="label">Scenario</div><div class="value">{{.Scenario}}</div></div><div class="fact"><div class="label">Scope</div><div class="value">{{.Scope}}</div></div></div><p>{{.Summary}} Next: {{.NextAction}}</p></section>
<section><h2>Measured comparison envelope</h2><div class="grid"><div class="fact"><div class="label">Eligible outcome groups</div>{{.OutcomeCount}}</div><div class="fact"><div class="label">Typed exclusions</div>{{.ExclusionCount}}</div><div class="fact"><div class="label">Discovery repeats / candidate</div>{{.DiscoveryRepeats}}</div><div class="fact"><div class="label">Fresh confirmation repeats / candidate</div>{{.ConfirmationRepeats}}</div></div><p>Projection mode: <code>{{.ProjectionMode}}</code></p></section>
<section><h2>Stimulus previews</h2><p>Captured bodies, environment values, private roots, process streams, and secret slots are omitted. Preview operations: <code>countershape/report-preview/v1</code>.</p><div class="grid"><div><h3>Original</h3><p><code>{{.Original.Kind}}</code> · <code>{{.Original.Digest}}</code></p><pre>{{.Original.Body}}</pre></div><div><h3>Minimized</h3><p><code>{{.Minimized.Kind}}</code> · <code>{{.Minimized.Digest}}</code></p><pre>{{.Minimized.Body}}</pre></div></div></section>
<section><h2>Reduction and projection</h2><div class="grid"><div class="fact"><div class="label">Grade</div>{{.Reduction.GradeStatus}}</div><div class="fact"><div class="label">Budgets</div>proposals={{.Reduction.ProposalLimit}}, candidate trials={{.Reduction.CandidateTrialLimit}}, wall_ms={{.Reduction.WallLimitMS}}</div><div class="fact"><div class="label">Evaluations / unresolved</div>{{.Reduction.EvaluationCount}} / {{.Reduction.UnresolvedCount}}</div><div class="fact"><div class="label">Final sweep</div>{{.Reduction.FinalSweepState}}</div></div><h3>Captured → Projection operations</h3><ol>{{range .Operations}}<li><code>{{.Name}}</code> — <code>{{.RuleDigest}}</code></li>{{end}}</ol></section>
<section><h2>Eligible selected outcome facts</h2>{{range .Selected}}<h3>{{.Alias}}</h3>{{if .Fields}}<table><thead><tr><th>Selected field</th><th>Value</th></tr></thead><tbody>{{range .Fields}}<tr><td><code>{{.ID}}</code></td><td>{{.Value}}</td></tr>{{end}}</tbody></table>{{else}}<p>EMPTY — no asserted values.</p>{{end}}{{end}}<h3>Exclusions</h3><ul>{{range .Exclusions}}<li>{{.}}</li>{{end}}</ul></section>
<section><h2>Human ruling</h2><div class="grid"><div class="fact"><div class="label">DecisionRecord</div>{{.DecisionDigest}}</div><div class="fact"><div class="label">Action</div>{{.Action}}</div><div class="fact"><div class="label">Compile status</div>{{.CompilationStatus}}</div><div class="fact"><div class="label">Actor / authenticity</div>{{.Actor}} / {{.ActorAuthority}}</div></div><p>{{.Annotation}}</p><h3>Selected fields</h3><ul>{{range .SelectedFields}}<li><code>{{.}}</code></li>{{end}}</ul><h3>Explicitly nonasserted fields</h3><ul>{{range .NonassertedFields}}<li><code>{{.}}</code></li>{{end}}</ul></section>
<section><h2>Contract and current conformance</h2><p><b>Contract digest:</b> {{.ContractDigest}}</p><p><b>Current conformance:</b> {{.CurrentConformance}}</p></section>
<section><h2>Receipt references</h2>{{if .Receipts}}<table><thead><tr><th>Authority</th><th>Grade verbatim</th><th>Commit</th><th>Command</th></tr></thead><tbody>{{range .Receipts}}<tr><td>{{.Authority}}</td><td>{{.Grade}}</td><td>{{.Commit}}</td><td>{{.Command}}</td></tr>{{end}}</tbody></table>{{else}}<p>{{.ReceiptEmpty}}</p>{{end}}<p>Unknown grades remain verbatim and are never translated into conformance.</p></section>
<section><h2>Trust warnings and nonclaims</h2><ul>{{range .TrustWarnings}}<li>{{.}}</li>{{end}}{{range .Nonclaims}}<li>{{.}}</li>{{end}}<li>Fixture secrecy tests prove fixture coverage only; they do not establish arbitrary-secret absence.</li><li>Candidate producer/model provenance is omitted by the selected default reveal policy.</li></ul></section>
{{if .IncludeRaw}}<section><div class="warning">DANGEROUS RAW OPT-IN</div><p>These inert text blocks may contain secrets, paths, process evidence, or candidate provenance.</p>{{range .Raw}}<details><summary>{{.Name}} · {{.Digest}}</summary><pre>{{.Body}}</pre></details>{{end}}</section>{{end}}
</main></body></html>`))
