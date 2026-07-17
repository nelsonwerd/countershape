//go:build darwin && cgo

package cli_precedence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"syscall"
	"testing"
	"time"

	countercli "github.com/nelsonwerd/countershape/internal/adapters/cli"
	climodel "github.com/nelsonwerd/countershape/internal/adapters/cli/model"
	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/compare"
	"github.com/nelsonwerd/countershape/internal/confirmation"
	"github.com/nelsonwerd/countershape/internal/contractsource"
	"github.com/nelsonwerd/countershape/internal/domain"
	nodeemit "github.com/nelsonwerd/countershape/internal/emit/node"
	"github.com/nelsonwerd/countershape/internal/projectiontranslate"
	reducer "github.com/nelsonwerd/countershape/internal/reduce"
	grade "github.com/nelsonwerd/countershape/internal/reduction"
	"github.com/nelsonwerd/countershape/internal/store"
	"github.com/nelsonwerd/countershape/testkit/clifixture"
)

const (
	cliA21StudyLabel    = "physical CLI choicepoint promotion"
	cliA21RestartEnv    = "COUNTERSHAPE_A21_CLI_RESTART_REQUEST"
	cliA21RestartSchema = "countershape/a2.1/restart/v1"
	cliA21RequestLimit  = 8 << 10
	cliA21ResultLimit   = 2 << 20
)

type cliA21RestartRequest struct {
	Schema     string `json:"schema"`
	StoreRoot  string `json:"store_root"`
	SourcePath string `json:"source_path"`
	ResultPath string `json:"result_path"`
}

type cliA21RestartResult struct {
	Schema                  string   `json:"schema"`
	CompilationDigest       string   `json:"compilation_digest"`
	DecisionRecordDigest    string   `json:"decision_record_digest"`
	ChoicepointDigest       string   `json:"choicepoint_digest"`
	SourceDigest            string   `json:"source_digest"`
	SourceProfileDigest     string   `json:"source_profile_digest"`
	Action                  string   `json:"action"`
	SelectedFields          []string `json:"selected_fields"`
	AllowedTupleCanonical64 []string `json:"allowed_tuple_canonical_base64"`
}

type cliA21PreparedDiagnostic struct {
	Valid               bool
	Digest              string
	DecisionRecord      string
	Choicepoint         string
	Source              string
	SourceProfile       string
	Action              string
	SelectedCount       int
	SelectedPreview     []string
	AllowedTupleCount   int
	AllowedTupleSHA256s []string
}

func cliA21ResolvedStoreTempDir(t testing.TB) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		t.Fatalf("resolved temporary directory is not clean and absolute: %q", root)
	}
	return root
}

func cliA21DescribePrepared(prepared nodeemit.PreparedCompilation) cliA21PreparedDiagnostic {
	selected := prepared.SelectedFields()
	selectedLimit := len(selected)
	if selectedLimit > 8 {
		selectedLimit = 8
	}
	tuples := prepared.AllowedTupleCanonicalBytes()
	tupleLimit := len(tuples)
	if tupleLimit > 8 {
		tupleLimit = 8
	}
	tupleDigests := make([]string, tupleLimit)
	for index, tuple := range tuples[:tupleLimit] {
		digest := sha256.Sum256(tuple)
		tupleDigests[index] = fmt.Sprintf("sha256:%x", digest)
	}
	return cliA21PreparedDiagnostic{
		Valid: prepared.Valid(), Digest: prepared.Digest().String(),
		DecisionRecord: prepared.DecisionRecordDigest().String(), Choicepoint: prepared.ChoicepointDigest().String(),
		Source: prepared.SourceDigest().String(), SourceProfile: prepared.SourceProfileDigest().String(), Action: prepared.Action(),
		SelectedCount: len(selected), SelectedPreview: append([]string(nil), selected[:selectedLimit]...),
		AllowedTupleCount: len(tuples), AllowedTupleSHA256s: tupleDigests,
	}
}

type cliA21BoundedOutput struct {
	data      []byte
	truncated bool
}

func (w *cliA21BoundedOutput) Write(input []byte) (int, error) {
	const limit = 64 << 10
	if remaining := limit - len(w.data); remaining > 0 {
		if len(input) < remaining {
			remaining = len(input)
		}
		w.data = append(w.data, input[:remaining]...)
	}
	if len(w.data) == limit {
		w.truncated = true
	}
	return len(input), nil
}

func (w cliA21BoundedOutput) String() string {
	if w.truncated {
		return string(w.data) + "\n[child output truncated]"
	}
	return string(w.data)
}

func cliA21WriteExclusive(path string, body []byte, limit int) error {
	if len(body) == 0 || len(body) > limit {
		return fmt.Errorf("body size %d is outside 1..%d", len(body), limit)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	n, writeErr := file.Write(body)
	if writeErr == nil && n != len(body) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func cliA21ReadPrivateRegular(path string, limit int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > int64(limit) {
		return nil, fmt.Errorf("%s is not a bounded private regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(body) == 0 || len(body) > limit || int64(len(body)) != info.Size() {
		return nil, fmt.Errorf("%s changed size while being read", path)
	}
	return body, nil
}

func cliA21DecodeStrict(body []byte, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing JSON value: %v", err)
	}
	return nil
}

func cliA21CleanAbsolute(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path
}

func cliA21StoreAuthorityRefused(err error) bool {
	var typed *store.Error
	return errors.As(err, &typed) && typed.Code == "OBJECT_AUTHORITY_REFUSED"
}

func cliA21StoreInventory(t *testing.T, root string) []string {
	t.Helper()
	entries := make([]string, 0, 32)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("store inventory has no Darwin stat identity for %s", relative)
		}
		identity := fmt.Sprintf("%d\x00%d\x00%d", stat.Dev, stat.Ino, info.ModTime().UnixNano())
		if entry.IsDir() {
			entries = append(entries, fmt.Sprintf("D\x00%s\x00%#o\x00%s", relative, info.Mode().Perm(), identity))
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("store inventory contains non-regular entry %s (%s)", relative, entry.Type())
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		entries = append(entries, fmt.Sprintf("F\x00%s\x00%#o\x00%d\x00%x\x00%s", relative, info.Mode().Perm(), len(body), digest, identity))
		return nil
	})
	if err != nil {
		t.Fatalf("inventory CLI ruling store: %v", err)
	}
	sort.Strings(entries)
	return entries
}

func assertCLICompilationFreshProcessRestart(
	t *testing.T,
	storeRoot string,
	source contractsource.PortableSource,
	prepared nodeemit.PreparedCompilation,
) {
	t.Helper()
	protocolRoot := t.TempDir()
	if err := os.Chmod(protocolRoot, 0o700); err != nil {
		t.Fatalf("make CLI restart protocol directory private: %v", err)
	}
	sourcePath := filepath.Join(protocolRoot, "portable-source.json")
	requestPath := filepath.Join(protocolRoot, "request.json")
	resultPath := filepath.Join(protocolRoot, "result.json")
	childCWD := filepath.Join(protocolRoot, "child-cwd")
	if err := os.Mkdir(childCWD, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := cliA21WriteExclusive(sourcePath, source.CanonicalBytes(), contractsource.MaxSourceCanonicalBytes); err != nil {
		t.Fatalf("write private CLI restart source: %v", err)
	}
	requestBody, err := json.Marshal(cliA21RestartRequest{
		Schema: cliA21RestartSchema, StoreRoot: storeRoot, SourcePath: sourcePath, ResultPath: resultPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cliA21WriteExclusive(requestPath, requestBody, cliA21RequestLimit); err != nil {
		t.Fatalf("write private CLI restart request: %v", err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		executable,
		"-test.run=^TestCLICompilationFreshProcessRestartHelper$",
		"-test.count=1",
		"-test.timeout=90s",
	)
	command.Dir = childCWD
	command.Env = []string{cliA21RestartEnv + "=" + requestPath}
	var output cliA21BoundedOutput
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			t.Fatalf("fresh-process CLI compilation restart timed out: %v; %s", ctx.Err(), output.String())
		}
		t.Fatalf("fresh-process CLI compilation restart failed: %v; %s", err, output.String())
	}
	resultBody, err := cliA21ReadPrivateRegular(resultPath, cliA21ResultLimit)
	if err != nil {
		t.Fatalf("read private CLI restart result: %v", err)
	}
	var result cliA21RestartResult
	if err := cliA21DecodeStrict(resultBody, &result); err != nil {
		t.Fatalf("decode strict CLI restart result: %v", err)
	}
	if result.Schema != cliA21RestartSchema ||
		result.CompilationDigest != prepared.Digest().String() ||
		result.DecisionRecordDigest != prepared.DecisionRecordDigest().String() ||
		result.ChoicepointDigest != prepared.ChoicepointDigest().String() ||
		result.SourceDigest != prepared.SourceDigest().String() ||
		result.SourceProfileDigest != prepared.SourceProfileDigest().String() ||
		result.Action != prepared.Action() || !slices.Equal(result.SelectedFields, prepared.SelectedFields()) {
		t.Fatalf("fresh-process CLI compilation summaries changed: %#v", result)
	}
	wantTuples := prepared.AllowedTupleCanonicalBytes()
	if len(result.AllowedTupleCanonical64) != len(wantTuples) {
		t.Fatalf("fresh-process CLI tuple count = %d, want %d", len(result.AllowedTupleCanonical64), len(wantTuples))
	}
	for index, encoded := range result.AllowedTupleCanonical64 {
		decoded, err := base64.StdEncoding.Strict().DecodeString(encoded)
		if err != nil || !bytes.Equal(decoded, wantTuples[index]) {
			t.Fatalf("fresh-process CLI tuple %d changed: %v", index, err)
		}
	}
}

func TestCLICompilationFreshProcessRestartHelper(t *testing.T) {
	requestPath := os.Getenv(cliA21RestartEnv)
	if requestPath == "" {
		return
	}
	if !cliA21CleanAbsolute(requestPath) {
		t.Fatal("CLI restart request path is not clean and absolute")
	}
	requestRoot := filepath.Dir(requestPath)
	rootInfo, err := os.Lstat(requestRoot)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode().Perm()&0o077 != 0 {
		t.Fatalf("CLI restart request directory is not private: %v", err)
	}
	requestBody, err := cliA21ReadPrivateRegular(requestPath, cliA21RequestLimit)
	if err != nil {
		t.Fatal(err)
	}
	var request cliA21RestartRequest
	if err := cliA21DecodeStrict(requestBody, &request); err != nil {
		t.Fatal(err)
	}
	if request.Schema != cliA21RestartSchema || !cliA21CleanAbsolute(request.StoreRoot) ||
		!cliA21CleanAbsolute(request.SourcePath) || !cliA21CleanAbsolute(request.ResultPath) ||
		filepath.Dir(request.SourcePath) != requestRoot || filepath.Dir(request.ResultPath) != requestRoot ||
		request.SourcePath == request.ResultPath || request.SourcePath == requestPath || request.ResultPath == requestPath {
		t.Fatal("CLI restart request is outside the closed path protocol")
	}
	if _, err := os.Lstat(request.ResultPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("CLI restart result path already exists or is inaccessible: %v", err)
	}
	sourceBody, err := cliA21ReadPrivateRegular(request.SourcePath, contractsource.MaxSourceCanonicalBytes)
	if err != nil {
		t.Fatal(err)
	}
	source, err := contractsource.Parse(sourceBody)
	if err != nil {
		t.Fatalf("strict CLI restart source parse: %v", err)
	}
	studyID, err := store.NewStudyID(cliA21StudyLabel)
	if err != nil {
		t.Fatal(err)
	}
	objectStore, err := store.OpenObjectStore(request.StoreRoot)
	if err != nil {
		t.Fatal(err)
	}
	ruling, err := promotion.OpenRuling(context.Background(), objectStore, studyID)
	if err != nil {
		t.Fatal(err)
	}
	preparation, err := promotion.PreparePortableRuling(context.Background(), objectStore, ruling)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := nodeemit.PrepareCompilation(context.Background(), objectStore, preparation, source)
	if err != nil || !prepared.Valid() {
		t.Fatalf("fresh-process CLI preparation is invalid: %#v, %v", cliA21DescribePrepared(prepared), err)
	}
	tuples := prepared.AllowedTupleCanonicalBytes()
	encodedTuples := make([]string, len(tuples))
	for index, tuple := range tuples {
		encodedTuples[index] = base64.StdEncoding.EncodeToString(tuple)
	}
	resultBody, err := json.Marshal(cliA21RestartResult{
		Schema: cliA21RestartSchema, CompilationDigest: prepared.Digest().String(),
		DecisionRecordDigest: prepared.DecisionRecordDigest().String(),
		ChoicepointDigest:    prepared.ChoicepointDigest().String(), SourceDigest: prepared.SourceDigest().String(),
		SourceProfileDigest: prepared.SourceProfileDigest().String(), Action: prepared.Action(),
		SelectedFields: prepared.SelectedFields(), AllowedTupleCanonical64: encodedTuples,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cliA21WriteExclusive(request.ResultPath, resultBody, cliA21ResultLimit); err != nil {
		t.Fatal(err)
	}
}

func TestCLIPhysicalReducerRemovesIrrelevantEnvironmentWithFreshEvidence(t *testing.T) {
	appMode, err := countercli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		t.Fatal(err)
	}
	noise, err := countercli.PresentEnvironment("Z_NOISE", "ignored")
	if err != nil {
		t.Fatal(err)
	}
	configFixture, err := countercli.NewFixtureFile(
		"config.json", clifixture.ConfigJSON("config"), countercli.FixtureMode0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	noisyStimulus, err := countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{clifixture.Entrypoint}, Argv: []string{"--mode", "argv"},
		Stdin: countercli.AbsentStdin(), Environment: []countercli.CLIEnvironmentBinding{appMode, noise},
		Fixtures: []countercli.CLIFixtureFile{configFixture}, CWDPolicy: countercli.CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	baselineConfig := cliPhysicalReductionConfig(t)
	baselineConfig.StimulusOverride = &noisyStimulus
	baselineCheckpoint, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	baselineResult, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !baselineResult.HasOutcomeMap || baselineResult.OutcomeMap.Phase() != domain.AttemptDiscovery {
		t.Fatal("physical CLI baseline did not produce a discovery outcome map")
	}
	if !baselineCheckpoint.HasOutcomeMap || baselineCheckpoint.Plan.Digest() != baselineResult.Plan.Digest() ||
		compare.AssessPreservation(baselineCheckpoint.OutcomeMap, baselineResult.OutcomeMap).Relation() != compare.PreservationEqual {
		t.Fatal("physical CLI pre-divergence baseline checkpoint changed semantic plan or labeled behavior")
	}
	baseline, err := compare.RequireDivergence(baselineResult.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: baselineResult.Stimulus, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIEnvironmentRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineResult.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if budget.ProposalLimit() != 5 || budget.CandidateTrialLimit() != 36 || budget.WallLimit() != 3*time.Minute {
		t.Fatalf("compiled CLI reduction budget = %d/%d/%s", budget.ProposalLimit(), budget.CandidateTrialLimit(), budget.WallLimit())
	}
	var evaluatorErr error
	var minimizedStudy StudyResult
	run, err := reducer.Run(context.Background(), reducer.RunInput[countercli.CLIStimulus]{
		Original: baselineResult.Stimulus,
		Reference: func(stimulus countercli.CLIStimulus) (domain.Digest, reducer.Measure, bool) {
			measure, measureErr := countercli.MeasureCLIStimulus(stimulus, policy)
			return stimulus.Digest(), measure, measureErr == nil
		},
		Enumerate: func(_ context.Context, stimulus countercli.CLIStimulus) ([]reducer.TypedProposal[countercli.CLIStimulus], error) {
			neighbors, enumerateErr := countercli.EnumerateCLINeighbors(stimulus, policy)
			if enumerateErr != nil {
				return nil, enumerateErr
			}
			proposals := make([]reducer.TypedProposal[countercli.CLIStimulus], len(neighbors))
			for index, neighbor := range neighbors {
				proposals[index] = reducer.TypedProposal[countercli.CLIStimulus]{
					Stimulus: neighbor.Stimulus(), Neighbor: neighbor.Neighbor(),
				}
			}
			return proposals, nil
		},
		Evaluate: func(ctx context.Context, stimulus countercli.CLIStimulus, _ reducer.Neighbor, purpose domain.AttemptPurpose, allowance reducer.EvaluationAllowance) (reducer.EvaluationObservation, error) {
			config := cliPhysicalReductionConfig(t)
			requiredTrials := uint64(config.MaxTotalTrials)
			if allowance.RemainingCandidateTrials < requiredTrials || !time.Now().Before(allowance.WallDeadline) {
				return reducer.EvaluationObservation{}, context.DeadlineExceeded
			}
			config.Purpose = purpose
			config.StimulusOverride = &stimulus
			evaluationContext, cancel := context.WithDeadline(ctx, allowance.WallDeadline)
			defer cancel()
			result, runErr := Run(evaluationContext, config)
			if runErr != nil {
				evaluatorErr = runErr
				return reducer.EvaluationObservation{}, runErr
			}
			if !result.HasOutcomeMap {
				evaluatorErr = context.Canceled
				return reducer.EvaluationObservation{}, evaluatorErr
			}
			outcome := result.OutcomeMap
			if compare.AssessPreservation(baseline.OutcomeMap(), outcome).Relation() == compare.PreservationEqual {
				minimizedStudy = result
			}
			return reducer.EvaluationObservation{OutcomeMap: &outcome, CandidateTrials: uint64(len(result.Trials))}, nil
		},
		Baseline: baseline, ReducerSet: policy.ReducerSet(), Budget: budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	if evaluatorErr != nil {
		t.Fatalf("physical CLI reducer evaluator failed: %v", evaluatorErr)
	}
	if !run.Valid() || run.DraftGrade() != reducer.GradeBestKnown || !run.HasAcceptedReduction() ||
		run.Transcript().FinalSweepState() != reducer.FinalSweepComplete || len(run.Transcript().Entries()) != 4 ||
		len(run.Transcript().AcceptedPath()) != 1 || run.MinimizedStimulusDigest() == baselineResult.Stimulus.Digest() {
		t.Fatalf("unexpected physical CLI reduction: grade=%s accepted=%t sweep=%s entries=%d limits=%v",
			run.DraftGrade(), run.HasAcceptedReduction(), run.Transcript().FinalSweepState(),
			len(run.Transcript().Entries()), run.Transcript().Limitations())
	}
	entries := run.Transcript().Entries()
	preserving := entries[1].Evaluation()
	finalSweep := entries[len(entries)-1].Evaluation()
	if preserving.Purpose() != domain.AttemptReduction || preserving.Decision() != reducer.Preserves ||
		!preserving.LogicalNonReuseWithBaseline() || len(preserving.ObservedAttemptDigests()) != 9 ||
		finalSweep.Purpose() != domain.AttemptFinalSweep || finalSweep.Decision() != reducer.Changes ||
		!finalSweep.LogicalNonReuseWithBaseline() || len(finalSweep.ObservedAttemptDigests()) != 9 {
		t.Fatalf("physical CLI evidence is incomplete: preserving=%s/%s/%d final=%s/%s/%d",
			preserving.Purpose(), preserving.Decision(), len(preserving.ObservedAttemptDigests()),
			finalSweep.Purpose(), finalSweep.Decision(), len(finalSweep.ObservedAttemptDigests()))
	}
	draft, present, err := run.CompletedSweepDraft()
	if err != nil || !present || !draft.Valid() || len(draft.NeighborDigests()) != 1 ||
		draft.CurrentStimulusDigest() != run.MinimizedStimulusDigest() {
		t.Fatalf("physical CLI empty-neighbor completed sweep was not retained: present=%t err=%v", present, err)
	}
	if parsed, parseErr := reducer.ParseCompletedSweepDraft(draft.CanonicalBytes()); parseErr != nil || parsed.Digest() != draft.Digest() {
		t.Fatalf("physical CLI completed sweep did not round trip exactly: %v", parseErr)
	}
	sweepStore, err := store.OpenReductionSweepStore(filepath.Join(cliA21ResolvedStoreTempDir(t), "sweep-store"))
	if err != nil {
		t.Fatal(err)
	}
	authority, err := sweepStore.Publish(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := grade.Finalize(context.Background(), run, &grade.SweepCompletion{
		Store: sweepStore, Draft: draft, Authority: authority,
	})
	if err != nil || !finalized.Valid() || finalized.Grade().Status() != grade.StatusOneMinimalUnder {
		t.Fatalf("physical CLI durable grade was not quantified: status=%s err=%v",
			finalized.Grade().Status(), err)
	}
	if !minimizedStudy.HasOutcomeMap || minimizedStudy.Stimulus.Digest() != run.MinimizedStimulusDigest() {
		t.Fatal("physical CLI reducer did not retain the exact minimized preserving study")
	}
	reducedBaseline, err := compare.RequireDivergence(minimizedStudy.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	confirmationConfig := cliPhysicalReductionConfig(t)
	confirmationConfig.Purpose = domain.AttemptConfirmation
	confirmationConfig.StimulusOverride = &minimizedStudy.Stimulus
	confirmationConfig.Confirmation = &ConfirmationInput{
		ReducedBaseline: reducedBaseline, ReductionRun: run, ReductionResult: finalized,
	}
	confirmedStudy, err := Run(context.Background(), confirmationConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !confirmedStudy.HasConfirmation || !confirmedStudy.Confirmation.Valid() || !confirmedStudy.HasOutcomeMap ||
		confirmedStudy.OutcomeMap.Phase() != domain.AttemptConfirmation ||
		confirmedStudy.OutcomeMap.ScheduleStartOffset() != 1 || len(confirmedStudy.Confirmation.Draft().PhysicalFacts()) != 9 {
		t.Fatal("physical CLI confirmation did not produce one complete fresh phase-bound matrix")
	}
	confirmationDraft := confirmedStudy.Confirmation.Draft()
	if parsed, parseErr := confirmation.ParseRecord(confirmationDraft.CanonicalBytes()); parseErr != nil || parsed.Digest() != confirmationDraft.Digest() {
		t.Fatalf("physical CLI confirmation did not round trip strictly: %v", parseErr)
	}
	confirmationWire := confirmationDraft.CanonicalBytes()
	unknownConfirmation := append([]byte(nil), confirmationWire[:len(confirmationWire)-1]...)
	unknownConfirmation = append(unknownConfirmation, []byte(`,"zz_unknown":true}`)...)
	if _, parseErr := confirmation.ParseRecord(unknownConfirmation); parseErr == nil {
		t.Fatal("FreshConfirmation parser accepted an unknown canonical member")
	}
	confirmationStore, err := store.OpenObjectStore(filepath.Join(cliA21ResolvedStoreTempDir(t), "confirmation-store"))
	if err != nil {
		t.Fatal(err)
	}
	object, err := store.NewSemanticObject("FreshConfirmation", confirmationDraft.Digest(), confirmationDraft.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	confirmationAuthority, err := confirmationStore.Publish(context.Background(), object)
	if err != nil {
		t.Fatal(err)
	}
	if err := confirmationStore.Validate(context.Background(), object, confirmationAuthority); err != nil {
		t.Fatal(err)
	}
	originalArtifact, err := choice.NewCanonicalArtifact(
		"CLIStimulus", baselineResult.Stimulus.Digest(), baselineResult.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	minimizedArtifact, err := choice.NewCanonicalArtifact(
		"CLIStimulus", confirmedStudy.Stimulus.Digest(), confirmedStudy.Stimulus.CanonicalBytes(),
	)
	if err != nil {
		t.Fatal(err)
	}
	reveals := make([]choice.CandidateReveal, len(confirmedStudy.CandidateBindings))
	for index, binding := range confirmedStudy.CandidateBindings {
		reveals[index] = choice.CandidateReveal{
			CandidateExecutionKey: binding.Key(), DisplayRef: string(confirmedStudy.CandidateRoles[binding.Key()]),
			ProducerMetadata: "local deterministic CLI fixture",
		}
	}
	choicepoint, err := choice.NewChoicepointRecord(choice.ChoicepointInput{
		Scenario: "Which exact CLI precedence behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, Confirmation: confirmationDraft.Record(), CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil || !choicepoint.Valid() {
		t.Fatalf("physical CLI Choicepoint construction failed: %v", err)
	}
	parsedChoicepoint, err := choice.ParseChoicepointRecord(choicepoint.CanonicalBytes())
	if err != nil || parsedChoicepoint.Digest() != choicepoint.Digest() {
		t.Fatalf("physical CLI Choicepoint did not round trip strictly: %v", err)
	}
	blind, err := choice.NewBlindView(choicepoint)
	if err != nil || len(blind.DTO().Cards()) != confirmedStudy.OutcomeMap.DistinctProjectionCount() {
		t.Fatalf("physical CLI blind DTO did not group exact outcomes: %v", err)
	}
	if blind.DTO().ProjectionMode() != "ADAPTER_BOUND_PORTABLE_FIELDS_V1" ||
		!slices.Equal(blind.DTO().SelectableFields(), []string{
			string(countercli.CLIFieldStdoutBytes), string(countercli.CLIFieldStdoutJSONMode),
			string(countercli.CLIFieldStdoutJSONSource),
		}) || !slices.Equal(blind.DTO().DifferingFields(), []string{
		string(countercli.CLIFieldStdoutBytes), string(countercli.CLIFieldStdoutJSONMode),
		string(countercli.CLIFieldStdoutJSONSource),
	}) {
		t.Fatalf("physical CLI Choicepoint lacks exact portable profile order or mode: selectable=%#v differing=%#v mode=%q",
			blind.DTO().SelectableFields(), blind.DTO().DifferingFields(), blind.DTO().ProjectionMode())
	}
	blindBytes := blind.DTO().CanonicalBytes()
	for _, binding := range confirmedStudy.CandidateBindings {
		for _, forbidden := range []string{
			binding.Key().String(), binding.Identity().TreeIdentityDigest.String(),
			string(confirmedStudy.CandidateRoles[binding.Key()]), "local deterministic CLI fixture",
		} {
			if forbidden != "" && bytes.Contains(blindBytes, []byte(forbidden)) {
				t.Fatalf("blind DTO leaked reveal or candidate identity %q", forbidden)
			}
		}
	}
	for _, entry := range confirmedStudy.OutcomeMap.Entries() {
		if bytes.Contains(blindBytes, []byte(entry.ProjectionFingerprint.String())) {
			t.Fatal("blind DTO leaked a raw projection fingerprint")
		}
	}
	choiceObject, err := store.NewSemanticObject("Choicepoint", choicepoint.Digest(), choicepoint.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	choiceAuthority, err := confirmationStore.Publish(context.Background(), choiceObject)
	if err != nil || confirmationStore.Validate(context.Background(), choiceObject, choiceAuthority) != nil {
		t.Fatalf("physical CLI Choicepoint did not persist immutably: %v", err)
	}
	promotionRoot := filepath.Join(cliA21ResolvedStoreTempDir(t), "promotion-store")
	promotionStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	studyID, err := store.NewStudyID(cliA21StudyLabel)
	if err != nil {
		t.Fatal(err)
	}
	head, err := promotionStore.CreateStudy(context.Background(), studyID, confirmedStudy.Plan)
	if err != nil {
		t.Fatal(err)
	}
	head, err = promotionStore.AdvanceBaseline(context.Background(), head, baselineCheckpoint.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	head, err = promotionStore.AdvanceDivergence(context.Background(), head, baseline)
	if err != nil {
		t.Fatal(err)
	}
	wrongDivergence, err := compare.RequireDivergence(baselineCheckpoint.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	wrongStudyID, err := store.NewStudyID("physical CLI mismatched reduction lineage")
	if err != nil {
		t.Fatal(err)
	}
	wrongHead, err := promotionStore.CreateStudy(context.Background(), wrongStudyID, confirmedStudy.Plan)
	if err != nil {
		t.Fatal(err)
	}
	wrongHead, err = promotionStore.AdvanceBaseline(context.Background(), wrongHead, baselineResult.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	wrongHead, err = promotionStore.AdvanceDivergence(context.Background(), wrongHead, wrongDivergence)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := promotionStore.AdvanceReduction(context.Background(), wrongHead, run); err == nil {
		t.Fatal("ReductionRun advanced beneath a different exact divergence predecessor")
	}
	unchangedWrongHead, err := promotionStore.OpenHead(context.Background(), wrongStudyID)
	if err != nil || unchangedWrongHead.HeadDigest() != wrongHead.HeadDigest() ||
		unchangedWrongHead.Stage() != store.StageDivergence {
		t.Fatalf("rejected reduction lineage changed its head: %v", err)
	}
	if _, _, err := promotionStore.Read(context.Background(), "ReductionRun", run.Digest()); err == nil {
		t.Fatal("rejected reduction lineage published its successor object")
	}
	head, err = promotionStore.AdvanceReduction(context.Background(), head, run)
	if err != nil {
		t.Fatal(err)
	}
	storedConfirmation, err := promotion.PersistConfirmation(context.Background(), promotionStore, head, confirmationDraft)
	if err != nil {
		t.Fatal(err)
	}
	ready, err := promotion.Promote(context.Background(), promotionStore, storedConfirmation, promotion.ChoicepointRequest{
		Scenario: "Which exact CLI precedence behavior should become the accepted contract?",
		Plan:     confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, CandidateReveals: reveals,
		EvidenceReceipts: []domain.ReceiptReference{},
	})
	if err != nil || ready.Record().Digest() != choicepoint.Digest() {
		t.Fatalf("physical CLI store-bound Choicepoint promotion failed: %v", err)
	}
	restartedStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	reopenedReady, err := promotion.OpenReady(context.Background(), restartedStore, studyID)
	if err != nil || reopenedReady.Record().Digest() != ready.Record().Digest() {
		t.Fatalf("physical CLI CHOICEPOINT_READY did not survive restart: %v", err)
	}
	if _, err := promotion.Promote(context.Background(), promotionStore, storedConfirmation, promotion.ChoicepointRequest{
		Scenario: "stale confirmation must not promote again", Plan: confirmedStudy.Plan, Envelope: confirmedStudy.Envelope,
		CandidateBindings: confirmedStudy.CandidateBindings, OriginalStimulus: originalArtifact,
		MinimizedStimulus: minimizedArtifact, CandidateReveals: reveals, EvidenceReceipts: []domain.ReceiptReference{},
	}); err == nil {
		t.Fatal("superseded confirmation authority promoted a second Choicepoint")
	}

	customBytes := []byte(`{ "mode" : "reviewed-custom" }`)
	customValue, err := choice.BytesValue(customBytes)
	if err != nil {
		t.Fatal(err)
	}
	customExpectation, err := choice.NewSelectedTuple(
		reopenedReady.Record(),
		[]string{string(countercli.CLIFieldStdoutBytes)},
		[]choice.FieldValue{{FieldID: string(countercli.CLIFieldStdoutBytes), Value: customValue}},
	)
	if err != nil {
		t.Fatal(err)
	}
	customBytesInput := choice.RulingDraftInput{
		Action: choice.ActionCustomExpectation, SelectedFields: []string{string(countercli.CLIFieldStdoutBytes)},
		AllowedAliases: []string{}, CustomExpectation: &customExpectation,
		CustomReviewer: "physical-cli-byte-reviewer", CustomReviewEvidence: confirmationDraft.Digest(),
	}
	customBytesSession, err := choice.NewSession(reopenedReady.Record())
	if err != nil {
		t.Fatal(err)
	}
	customBytesSession, _, err = customBytesSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields, choice.SurfaceProvenance,
	} {
		customBytesSession, err = customBytesSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	customBytesSession, err = customBytesSession.Revise(customBytesInput, "")
	if err != nil {
		t.Fatal(err)
	}
	_, customBytesDecision, err := customBytesSession.Finalize(
		"local-test-operator", "Exact CLI bytes are an adapter-realizable reviewed predicate.", []domain.ReceiptReference{},
	)
	if err != nil || !customBytesDecision.EarlyReveal() ||
		!slices.Equal(customBytesDecision.SelectedFields(), []string{string(countercli.CLIFieldStdoutBytes)}) ||
		!slices.Equal(customBytesDecision.NonassertedFields(), []string{
			string(countercli.CLIFieldStdoutJSONMode), string(countercli.CLIFieldStdoutJSONSource),
		}) {
		t.Fatalf("portable CLI byte DecisionRecord lost its selected-only partition: %v", err)
	}
	customBytesCompiled, ok := customBytesDecision.CompilableRuling()
	if !ok {
		t.Fatal("portable CLI byte DecisionRecord was not compilable")
	}
	customBytesAllowed := customBytesCompiled.AllowedTuples()
	if len(customBytesAllowed) != 1 || len(customBytesAllowed[0].Fields) != 1 ||
		customBytesAllowed[0].Fields[0].FieldID != string(countercli.CLIFieldStdoutBytes) ||
		!bytes.Equal(customBytesAllowed[0].Fields[0].Value.Bytes(), customBytes) {
		t.Fatalf("portable CLI byte DecisionRecord changed exact bytes: %#v", customBytesAllowed)
	}
	parsedCustomBytes, err := choice.ParseDecisionRecord(customBytesDecision.CanonicalBytes(), reopenedReady.Record())
	if err != nil || parsedCustomBytes.Digest() != customBytesDecision.Digest() {
		t.Fatalf("portable CLI byte DecisionRecord did not round trip strictly: %v", err)
	}

	decisionSession, err := choice.NewSession(reopenedReady.Record())
	if err != nil {
		t.Fatal(err)
	}
	cards := decisionSession.BlindDTO().Cards()
	if len(cards) < 3 {
		t.Fatal("divergent CLI Choicepoint did not expose its three correlated outcome cards")
	}
	allowedAliases := make([]string, 0, 2)
	for _, card := range cards {
		mode := ""
		source := ""
		for _, field := range card.Fields {
			switch field.FieldID {
			case string(countercli.CLIFieldStdoutJSONMode):
				mode = field.Text
			case string(countercli.CLIFieldStdoutJSONSource):
				source = field.Text
			}
		}
		if mode == source && (mode == "config" || mode == "argv") {
			allowedAliases = append(allowedAliases, card.Alias)
		}
	}
	if len(allowedAliases) != 2 {
		t.Fatalf("could not select exact correlated config/argv cards: %#v", cards)
	}
	standardInput := choice.RulingDraftInput{
		Action: choice.ActionAllowObserved, SelectedFields: []string{
			string(countercli.CLIFieldStdoutJSONMode), string(countercli.CLIFieldStdoutJSONSource),
		},
		AllowedAliases: allowedAliases,
	}
	if _, err := decisionSession.Propose(standardInput); !choice.IsRefusal(err, choice.CodeRequiredSurfaceNotVisited) {
		t.Fatalf("standard blind proposal bypassed evidence presentation: %v", err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields,
	} {
		decisionSession, err = decisionSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	decisionSession, err = decisionSession.Propose(standardInput)
	if err != nil {
		t.Fatal(err)
	}
	decisionSession, reveal, err := decisionSession.Reveal()
	if err != nil || len(reveal.Groups()) != len(cards) {
		t.Fatalf("standard decision reveal failed: %v", err)
	}
	groups := reveal.Groups()
	if len(groups[0].Candidates) == 0 {
		t.Fatal("reveal group omitted its exact supporting candidates")
	}
	originalDisplayRef := groups[0].Candidates[0].DisplayRef
	groups[0].Candidates[0].DisplayRef = "mutated caller copy"
	if reveal.Groups()[0].Candidates[0].DisplayRef != originalDisplayRef {
		t.Fatal("RevealDTO nested candidate slices were not defensively copied")
	}
	decisionSession, err = decisionSession.Visit(choice.SurfaceProvenance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decisionSession.Revise(standardInput, "unnecessary"); !choice.IsRefusal(err, choice.CodeUnnecessaryChangeRationale) {
		t.Fatalf("semantic no-op post-reveal ruling accepted a rationale: %v", err)
	}
	changedInput := standardInput
	changedInput.AllowedAliases = []string{cards[0].Alias}
	if _, err := decisionSession.Revise(changedInput, ""); !choice.IsRefusal(err, choice.CodeChangeRationaleRequired) {
		t.Fatalf("changed post-reveal ruling omitted its rationale: %v", err)
	}
	const changedRationale = "Provenance changed which exact outcome I accept."
	changedSession, changeErr := decisionSession.Revise(changedInput, changedRationale)
	if changeErr != nil || changedSession.State() != choice.SessionPostRevealRecorded {
		t.Fatalf("rationalized post-reveal change was refused: %v", changeErr)
	}
	_, changedDecision, changeErr := changedSession.Finalize(
		"local-test-operator", "Changed exact CLI witness after reveal.", []domain.ReceiptReference{},
	)
	if changeErr != nil || !changedDecision.ChangedAfterReveal() ||
		changedDecision.PostRevealChangeRationale() != changedRationale {
		t.Fatalf("rationalized post-reveal change was not retained in DecisionRecord: %v", changeErr)
	}
	parsedChanged, changeErr := choice.ParseDecisionRecord(changedDecision.CanonicalBytes(), reopenedReady.Record())
	if changeErr != nil || parsedChanged.Digest() != changedDecision.Digest() || !parsedChanged.ChangedAfterReveal() {
		t.Fatalf("rationalized DecisionRecord did not round trip strictly: %v", changeErr)
	}
	decisionSession, err = decisionSession.Revise(standardInput, "")
	if err != nil {
		t.Fatal(err)
	}
	finalizedSession, decision, err := decisionSession.Finalize("local-test-operator", "Exact CLI witness only.", []domain.ReceiptReference{})
	if err != nil || !decision.Valid() || decision.EarlyReveal() || decision.ChangedAfterReveal() ||
		decision.ReceiptStatusWhenEmpty() != domain.Unreceipted() || len(decision.Receipts()) != 0 {
		t.Fatalf("standard DecisionRecord did not preserve exact blind/reveal facts: %v", err)
	}
	if _, ok := decision.CompilableRuling(); !ok {
		t.Fatal("validated ALLOW_OBSERVED DecisionRecord lost sealed compile eligibility")
	}
	parsedDecision, err := choice.ParseDecisionRecord(decision.CanonicalBytes(), reopenedReady.Record())
	if err != nil || parsedDecision.Digest() != decision.Digest() {
		t.Fatalf("DecisionRecord did not round trip strictly: %v", err)
	}
	if !slices.Equal(parsedDecision.SelectedFields(), []string{
		string(countercli.CLIFieldStdoutJSONMode), string(countercli.CLIFieldStdoutJSONSource),
	}) || !slices.Equal(parsedDecision.NonassertedFields(), []string{string(countercli.CLIFieldStdoutBytes)}) {
		t.Fatalf("portable CLI DecisionRecord selected/nonasserted fields differ: selected=%#v context=%#v",
			parsedDecision.SelectedFields(), parsedDecision.NonassertedFields())
	}
	correlated, ok := parsedDecision.CompilableRuling()
	if !ok || len(correlated.AllowedTuples()) != 2 {
		t.Fatalf("portable CLI allow-many predicate lost its two complete tuples: %#v", correlated)
	}
	seenCorrelated := map[string]bool{}
	for _, tuple := range correlated.AllowedTuples() {
		if len(tuple.Fields) != 2 || tuple.Fields[0].Value.Text() != tuple.Fields[1].Value.Text() {
			t.Fatalf("allow-many predicate synthesized or retained a cross-product tuple: %#v", tuple)
		}
		seenCorrelated[tuple.Fields[0].Value.Text()] = true
	}
	if !seenCorrelated["config"] || !seenCorrelated["argv"] || seenCorrelated["env"] {
		t.Fatalf("allow-many predicate tuple roster = %#v", seenCorrelated)
	}
	tamperedCompilable := bytes.Replace(decision.CanonicalBytes(), []byte(`"compilable":true`), []byte(`"compilable":false`), 1)
	if bytes.Equal(tamperedCompilable, decision.CanonicalBytes()) {
		t.Fatal("DecisionRecord test did not locate the derived compilable projection")
	}
	if _, err := choice.ParseDecisionRecord(tamperedCompilable, reopenedReady.Record()); err == nil {
		t.Fatal("DecisionRecord parser trusted a tampered derived compilable projection")
	}
	if _, err := finalizedSession.Visit(choice.SurfaceOriginalWitness); !choice.IsRefusal(err, choice.CodeInvalidSessionState) {
		t.Fatalf("finalized decision session remained mutable: %v", err)
	}

	earlySession, err := choice.NewSession(reopenedReady.Record())
	if err != nil {
		t.Fatal(err)
	}
	earlySession, _, err = earlySession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields, choice.SurfaceProvenance,
	} {
		earlySession, err = earlySession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	earlyInput := choice.RulingDraftInput{
		Action: choice.ActionRejectAll, SelectedFields: []string{}, AllowedAliases: []string{},
	}
	earlySession, err = earlySession.Revise(earlyInput, "")
	if err != nil {
		t.Fatal(err)
	}
	firstReceipt, err := domain.NewDidrunReceipt(
		confirmationDraft.Digest().String()+"\x00opaque-grade", "test-commit", choicepoint.Digest(),
	)
	if err != nil {
		t.Fatal(err)
	}
	secondReceipt, err := domain.NewDidrunReceipt(
		"opaque-grade", "test-commit\x00"+choicepoint.Digest().String(), confirmationDraft.Digest(),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, rejectedDecision, err := earlySession.Finalize(
		"local-test-operator", "No positive oracle.", []domain.ReceiptReference{firstReceipt, secondReceipt},
	)
	if err != nil || !rejectedDecision.Valid() || !rejectedDecision.EarlyReveal() ||
		rejectedDecision.ChangedAfterReveal() || len(rejectedDecision.Receipts()) != 2 {
		t.Fatalf("early-reveal noncompilable DecisionRecord was invalid: %v", err)
	}
	if _, ok := rejectedDecision.CompilableRuling(); ok {
		t.Fatal("REJECT_ALL DecisionRecord was cast to a compilable ruling")
	}
	reparsedRejected, err := choice.ParseDecisionRecord(rejectedDecision.CanonicalBytes(), reopenedReady.Record())
	if err != nil || len(reparsedRejected.Receipts()) != 2 ||
		reparsedRejected.Receipts()[0].GradeVerbatim() == reparsedRejected.Receipts()[1].GradeVerbatim() {
		t.Fatalf("opaque DecisionRecord receipts did not remain distinct and verbatim: %v", err)
	}

	refineSession, err := choice.NewSession(reopenedReady.Record())
	if err != nil {
		t.Fatal(err)
	}
	refineSession, _, err = refineSession.Reveal()
	if err != nil {
		t.Fatal(err)
	}
	for _, surface := range []choice.ReviewSurface{
		choice.SurfaceOriginalWitness, choice.SurfaceMinimizedWitness, choice.SurfaceReductionDerivation,
		choice.SurfaceProjectionOperations, choice.SurfaceNonassertedFields, choice.SurfaceProvenance,
	} {
		refineSession, err = refineSession.Visit(surface)
		if err != nil {
			t.Fatal(err)
		}
	}
	refineSession, err = refineSession.Revise(choice.RulingDraftInput{
		Action: choice.ActionRefine, SelectedFields: []string{}, AllowedAliases: []string{},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	_, refineDecision, err := refineSession.Finalize(
		"local-test-operator", "A successor study is required.", []domain.ReceiptReference{},
	)
	if err != nil || refineDecision.Action() != choice.ActionRefine {
		t.Fatalf("semantic REFINE DecisionRecord was invalid: %v", err)
	}
	if _, ok := refineDecision.CompilableRuling(); ok {
		t.Fatal("REFINE DecisionRecord was cast to a compilable ruling")
	}
	_, err = promotion.Finalize(context.Background(), restartedStore, reopenedReady, refineDecision)
	var promotionErr *promotion.Error
	if !errors.As(err, &promotionErr) || promotionErr.Code != promotion.CodeRefineRequiresSuccessorStudy {
		t.Fatalf("durable REFINE did not fail with %s: %v", promotion.CodeRefineRequiresSuccessorStudy, err)
	}
	if _, _, err := restartedStore.Read(context.Background(), "DecisionRecord", refineDecision.Digest()); err == nil {
		t.Fatal("refused REFINE DecisionRecord was published without a successor study")
	}
	stillReady, err := promotion.OpenReady(context.Background(), restartedStore, studyID)
	if err != nil || stillReady.Record().Digest() != reopenedReady.Record().Digest() {
		t.Fatalf("refused REFINE mutated the current CHOICEPOINT_READY head: %v", err)
	}

	finalizedRuling, err := promotion.Finalize(context.Background(), restartedStore, reopenedReady, decision)
	if err != nil || finalizedRuling.Record().Digest() != decision.Digest() {
		t.Fatalf("current Choicepoint did not promote its exact DecisionRecord: %v", err)
	}
	rulingRestart, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	reopenedRuling, err := promotion.OpenRuling(context.Background(), rulingRestart, studyID)
	if err != nil || reopenedRuling.Record().Digest() != decision.Digest() {
		t.Fatalf("durable RULING did not survive strict restart reconstruction: %v", err)
	}
	preparation, err := promotion.PreparePortableRuling(context.Background(), rulingRestart, reopenedRuling)
	if err != nil || !preparation.Valid() || preparation.DecisionDigest() != parsedDecision.Digest() ||
		!slices.Equal(preparation.SelectedFields(), parsedDecision.SelectedFields()) ||
		promotion.ValidatePortableRulingPreparation(context.Background(), rulingRestart, preparation) != nil {
		t.Fatalf("current portable CLI ruling preparation failed: %#v, %v", preparation, err)
	}
	resolvedProfile, err := projectiontranslate.Resolve(confirmedStudy.ProjectionDefinition.Binding())
	if err != nil {
		t.Fatal(err)
	}
	projectionAuthority, err := climodel.ResolveCLIProjectionAuthority(
		confirmedStudy.ProjectionDefinition.Digest(), confirmedStudy.ProjectionDefinition.CanonicalBytes(),
		confirmedStudy.ProjectionDefinition.Binding(),
	)
	if err != nil {
		t.Fatal(err)
	}
	portableSource, err := contractsource.NewCLISource(contractsource.CLIInput{
		Plan: confirmedStudy.Plan, Stimulus: confirmedStudy.Stimulus, Capture: confirmedStudy.CapturePolicy,
		Profile: resolvedProfile.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		t.Fatalf("exact minimized CLI source construction failed: %v", err)
	}
	const cliSourceProfileCanonical = `{"adapter_domain":"CLI","launch_profile":"NODE_REPO_SCRIPT_V1","runtime_family":"NODE","scope":"DECLARED_SOURCE_PROFILE_NOT_EXECUTION_EVIDENCE","semantic_profile":"countershape-node-core-exact/v1","start_profile":"DIRECT_CHILD_V1","subject_entrypoint":"fixture.mjs"}`
	cliSourceProfileDigest, err := canon.DigestBytes("ContractSourceProfile", []byte(cliSourceProfileCanonical))
	if err != nil {
		t.Fatal(err)
	}
	wantPreparedTuples := [][]byte{
		[]byte(`{"fields":[{"field_id":"cli.stdout.json.mode","value":{"tag":"STRING","value":"argv"}},{"field_id":"cli.stdout.json.source","value":{"tag":"STRING","value":"argv"}}]}`),
		[]byte(`{"fields":[{"field_id":"cli.stdout.json.mode","value":{"tag":"STRING","value":"config"}},{"field_id":"cli.stdout.json.source","value":{"tag":"STRING","value":"config"}}]}`),
	}
	headBeforeCompilation, err := rulingRestart.OpenHead(context.Background(), studyID)
	if err != nil {
		t.Fatal(err)
	}
	inventoryBeforeCompilation := cliA21StoreInventory(t, promotionRoot)
	prepared, err := nodeemit.PrepareCompilation(context.Background(), rulingRestart, preparation, portableSource)
	if err != nil || !prepared.Valid() || prepared.DecisionRecordDigest() != decision.Digest() ||
		prepared.ChoicepointDigest() != choicepoint.Digest() || prepared.SourceDigest() != portableSource.Digest() ||
		prepared.SourceProfileDigest().String() != cliSourceProfileDigest.String() ||
		prepared.Action() != string(choice.ActionAllowObserved) ||
		!slices.Equal(prepared.SelectedFields(), []string{
			string(countercli.CLIFieldStdoutJSONMode), string(countercli.CLIFieldStdoutJSONSource),
		}) || !reflect.DeepEqual(prepared.AllowedTupleCanonicalBytes(), wantPreparedTuples) {
		t.Fatalf("current CLI compilation preparation = %#v, %v", cliA21DescribePrepared(prepared), err)
	}
	headAfterCompilation, err := rulingRestart.OpenHead(context.Background(), studyID)
	if err != nil || headAfterCompilation.HeadDigest() != headBeforeCompilation.HeadDigest() ||
		headAfterCompilation.Stage() != store.StageRuling || headAfterCompilation.CurrentDigest() != decision.Digest() {
		t.Fatalf("compilation preparation changed the durable head: %v", err)
	}
	if !reflect.DeepEqual(cliA21StoreInventory(t, promotionRoot), inventoryBeforeCompilation) {
		t.Fatal("successful CLI compilation preparation changed the durable store inventory")
	}
	defensiveSelected := prepared.SelectedFields()
	defensiveTuples := prepared.AllowedTupleCanonicalBytes()
	defensiveSelected[0] = "cli.stdout.bytes"
	defensiveTuples[0][0] ^= 0xff
	if !prepared.Valid() || prepared.SelectedFields()[0] != string(countercli.CLIFieldStdoutJSONMode) {
		t.Fatal("PreparedCompilation getters were not defensive")
	}
	parsedSource, err := contractsource.Parse(portableSource.CanonicalBytes())
	if err != nil {
		t.Fatal(err)
	}
	reconstructed, err := nodeemit.PrepareCompilation(context.Background(), rulingRestart, preparation, parsedSource)
	if err != nil || reconstructed.Digest() != prepared.Digest() {
		t.Fatalf("byte-identical independently reconstructed source changed preparation: %v", err)
	}
	noisySource, err := contractsource.NewCLISource(contractsource.CLIInput{
		Plan: baselineResult.Plan, Stimulus: baselineResult.Stimulus, Capture: baselineResult.CapturePolicy,
		Profile: resolvedProfile.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		t.Fatal(err)
	}
	parsedNoisySource, err := contractsource.Parse(noisySource.CanonicalBytes())
	if err != nil || !parsedNoisySource.Valid() || parsedNoisySource.Digest() != noisySource.Digest() ||
		!bytes.Equal(parsedNoisySource.CanonicalBytes(), noisySource.CanonicalBytes()) {
		t.Fatalf("noisy CLI source is not an independently valid exact authority: %v", err)
	}
	if noisySource.Adapter() != portableSource.Adapter() || noisySource.Plan().Digest() != portableSource.Plan().Digest() ||
		!bytes.Equal(noisySource.Plan().CanonicalBytes(), portableSource.Plan().CanonicalBytes()) ||
		noisySource.ProjectionBinding().Digest() != portableSource.ProjectionBinding().Digest() ||
		!bytes.Equal(noisySource.ProjectionBinding().CanonicalBytes(), portableSource.ProjectionBinding().CanonicalBytes()) ||
		noisySource.Profile().Digest() != portableSource.Profile().Digest() ||
		!bytes.Equal(noisySource.Profile().CanonicalBytes(), portableSource.Profile().CanonicalBytes()) ||
		noisySource.StimulusDigest() == portableSource.StimulusDigest() ||
		bytes.Equal(noisySource.StimulusCanonicalBytes(), portableSource.StimulusCanonicalBytes()) ||
		noisySource.ExecutionBindingDigest() == portableSource.ExecutionBindingDigest() ||
		bytes.Equal(noisySource.ExecutionBindingCanonicalBytes(), portableSource.ExecutionBindingCanonicalBytes()) {
		t.Fatal("CLI cross-study source matrix did not isolate stimulus/execution authority under one valid shape")
	}
	if refused, prepareErr := nodeemit.PrepareCompilation(
		context.Background(), rulingRestart, preparation, noisySource,
	); !nodeemit.IsCode(prepareErr, nodeemit.CodeSourceRulingMismatch) || refused.Valid() ||
		refused.Digest().Valid() || refused.DecisionRecordDigest().Valid() || refused.ChoicepointDigest().Valid() ||
		refused.SourceDigest().Valid() || refused.SourceProfileDigest().Valid() || refused.Action() != "" ||
		len(refused.SelectedFields()) != 0 || len(refused.AllowedTupleCanonicalBytes()) != 0 {
		t.Fatalf("noisy/original source cross-pair = valid %t, err %v", refused.Valid(), prepareErr)
	}
	planMismatchConfig := cliPhysicalReductionConfig(t)
	planMismatchConfig.Repetitions = 2
	planMismatchConfig.MaxTotalTrials = 6
	planMismatchConfig.StimulusOverride = &confirmedStudy.Stimulus
	planMismatchStudy, err := Run(context.Background(), planMismatchConfig)
	if err != nil || !planMismatchStudy.HasOutcomeMap {
		t.Fatalf("independently valid different-plan CLI study failed: %v", err)
	}
	planMismatchSource, err := contractsource.NewCLISource(contractsource.CLIInput{
		Plan: planMismatchStudy.Plan, Stimulus: confirmedStudy.Stimulus, Capture: planMismatchStudy.CapturePolicy,
		Profile: resolvedProfile.Profile(), Projection: projectionAuthority,
	})
	if err != nil {
		t.Fatalf("different-plan CLI source construction failed: %v", err)
	}
	parsedPlanMismatchSource, err := contractsource.Parse(planMismatchSource.CanonicalBytes())
	if err != nil || !parsedPlanMismatchSource.Valid() || parsedPlanMismatchSource.Digest() != planMismatchSource.Digest() ||
		!bytes.Equal(parsedPlanMismatchSource.CanonicalBytes(), planMismatchSource.CanonicalBytes()) {
		t.Fatalf("different-plan CLI source is not independently strict and valid: %v", err)
	}
	originalCLIView, originalViewOK := portableSource.CLIView()
	mismatchCLIView, mismatchViewOK := parsedPlanMismatchSource.CLIView()
	if !originalViewOK || !mismatchViewOK || parsedPlanMismatchSource.Adapter() != portableSource.Adapter() ||
		parsedPlanMismatchSource.Plan().Adapter().RunnerDigest != portableSource.Plan().Adapter().RunnerDigest ||
		parsedPlanMismatchSource.Entrypoint() != portableSource.Entrypoint() ||
		parsedPlanMismatchSource.StartProfile() != portableSource.StartProfile() ||
		parsedPlanMismatchSource.ProjectionBinding().Digest() != portableSource.ProjectionBinding().Digest() ||
		!bytes.Equal(parsedPlanMismatchSource.ProjectionBinding().CanonicalBytes(), portableSource.ProjectionBinding().CanonicalBytes()) ||
		parsedPlanMismatchSource.Profile().Digest() != portableSource.Profile().Digest() ||
		!bytes.Equal(parsedPlanMismatchSource.Profile().CanonicalBytes(), portableSource.Profile().CanonicalBytes()) ||
		parsedPlanMismatchSource.StimulusDigest() != portableSource.StimulusDigest() ||
		!bytes.Equal(parsedPlanMismatchSource.StimulusCanonicalBytes(), portableSource.StimulusCanonicalBytes()) ||
		mismatchCLIView.Capture().Digest() != originalCLIView.Capture().Digest() ||
		!bytes.Equal(mismatchCLIView.Capture().CanonicalBytes(), originalCLIView.Capture().CanonicalBytes()) ||
		parsedPlanMismatchSource.Plan().Digest() == portableSource.Plan().Digest() ||
		bytes.Equal(parsedPlanMismatchSource.Plan().CanonicalBytes(), portableSource.Plan().CanonicalBytes()) ||
		parsedPlanMismatchSource.ExecutionBindingDigest() == portableSource.ExecutionBindingDigest() ||
		bytes.Equal(parsedPlanMismatchSource.ExecutionBindingCanonicalBytes(), portableSource.ExecutionBindingCanonicalBytes()) {
		t.Fatal("CLI different-plan cross-study source did not isolate plan authority under the same valid source shape")
	}
	if refused, prepareErr := nodeemit.PrepareCompilation(
		context.Background(), rulingRestart, preparation, parsedPlanMismatchSource,
	); !nodeemit.IsCode(prepareErr, nodeemit.CodeSourceRulingMismatch) || refused.Valid() ||
		refused.Digest().Valid() || refused.DecisionRecordDigest().Valid() || refused.ChoicepointDigest().Valid() ||
		refused.SourceDigest().Valid() || refused.SourceProfileDigest().Valid() || refused.Action() != "" ||
		len(refused.SelectedFields()) != 0 || len(refused.AllowedTupleCanonicalBytes()) != 0 {
		t.Fatalf("different-plan/original source cross-pair = valid %t, err %v", refused.Valid(), prepareErr)
	}
	currentAfterPlanMismatch, err := rulingRestart.OpenHead(context.Background(), studyID)
	if err != nil || currentAfterPlanMismatch.HeadDigest() != headAfterCompilation.HeadDigest() ||
		currentAfterPlanMismatch.Stage() != store.StageRuling || currentAfterPlanMismatch.CurrentDigest() != decision.Digest() ||
		!reflect.DeepEqual(cliA21StoreInventory(t, promotionRoot), inventoryBeforeCompilation) {
		t.Fatalf("different-plan CLI source refusal changed durable store state: %v", err)
	}
	currentBeforeForeign, err := rulingRestart.OpenHead(context.Background(), studyID)
	if err != nil {
		t.Fatal(err)
	}
	foreignStore, err := store.OpenObjectStore(promotionRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !preparation.Valid() {
		t.Fatal("portable preparation seal integrity changed merely because another store instance opened")
	}
	foreignErr := promotion.ValidatePortableRulingPreparation(context.Background(), foreignStore, preparation)
	var foreignStoreErr *store.Error
	if !errors.As(foreignErr, &foreignStoreErr) || foreignStoreErr.Code != "OBJECT_AUTHORITY_REFUSED" {
		t.Fatalf("portable preparation crossed store authority: %v", foreignErr)
	}
	if refused, prepareErr := nodeemit.PrepareCompilation(
		context.Background(), foreignStore, preparation, portableSource,
	); !cliA21StoreAuthorityRefused(prepareErr) || refused.Valid() || refused.Digest().Valid() || refused.DecisionRecordDigest().Valid() ||
		refused.ChoicepointDigest().Valid() || refused.SourceDigest().Valid() || refused.SourceProfileDigest().Valid() ||
		refused.Action() != "" || len(refused.SelectedFields()) != 0 || len(refused.AllowedTupleCanonicalBytes()) != 0 {
		t.Fatalf("wrong-store compilation preparation = valid %t, err %v", refused.Valid(), prepareErr)
	}
	currentAfterForeign, err := foreignStore.OpenHead(context.Background(), studyID)
	if err != nil || currentAfterForeign.HeadDigest() != currentBeforeForeign.HeadDigest() ||
		currentAfterForeign.Stage() != currentBeforeForeign.Stage() ||
		currentAfterForeign.CurrentDigest() != currentBeforeForeign.CurrentDigest() {
		t.Fatalf("foreign preparation validation changed the durable head: %v", err)
	}
	foreignRuling, err := promotion.OpenRuling(context.Background(), foreignStore, studyID)
	if err != nil {
		t.Fatal(err)
	}
	foreignPreparation, err := promotion.PreparePortableRuling(context.Background(), foreignStore, foreignRuling)
	if err != nil || promotion.ValidatePortableRulingPreparation(context.Background(), foreignStore, foreignPreparation) != nil ||
		foreignPreparation.DecisionDigest() != preparation.DecisionDigest() ||
		foreignPreparation.ProfileDigest() != preparation.ProfileDigest() ||
		!slices.Equal(foreignPreparation.SelectedFields(), preparation.SelectedFields()) {
		t.Fatalf("reopened store could not reissue matching portable preparation: %#v, %v", foreignPreparation, err)
	}
	restartedPreparation, err := nodeemit.PrepareCompilation(
		context.Background(), foreignStore, foreignPreparation, parsedSource,
	)
	if err != nil || !restartedPreparation.Valid() || restartedPreparation.Digest() != prepared.Digest() ||
		!reflect.DeepEqual(restartedPreparation.AllowedTupleCanonicalBytes(), prepared.AllowedTupleCanonicalBytes()) {
		t.Fatalf("restart/reopen changed sanitized compilation input: %#v, %v", cliA21DescribePrepared(restartedPreparation), err)
	}
	assertCLICompilationFreshProcessRestart(t, promotionRoot, portableSource, prepared)
	if predecessorErr := promotion.ValidatePortableRulingPreparation(
		context.Background(), rulingRestart, foreignPreparation,
	); !cliA21StoreAuthorityRefused(predecessorErr) {
		t.Fatalf("foreign-store portable preparation crossed predecessor authority: %v", predecessorErr)
	}
	if refused, prepareErr := nodeemit.PrepareCompilation(
		context.Background(), rulingRestart, foreignPreparation, portableSource,
	); !cliA21StoreAuthorityRefused(prepareErr) || refused.Valid() || refused.Digest().Valid() || refused.DecisionRecordDigest().Valid() ||
		refused.ChoicepointDigest().Valid() || refused.SourceDigest().Valid() || refused.SourceProfileDigest().Valid() ||
		refused.Action() != "" || len(refused.SelectedFields()) != 0 || len(refused.AllowedTupleCanonicalBytes()) != 0 {
		t.Fatalf("reissued preparation crossed predecessor compilation authority: %#v, %v", cliA21DescribePrepared(refused), prepareErr)
	}
	currentAfterRestartMatrix, err := foreignStore.OpenHead(context.Background(), studyID)
	if err != nil || currentAfterRestartMatrix.HeadDigest() != currentBeforeForeign.HeadDigest() ||
		currentAfterRestartMatrix.Stage() != currentBeforeForeign.Stage() ||
		currentAfterRestartMatrix.CurrentDigest() != currentBeforeForeign.CurrentDigest() ||
		!reflect.DeepEqual(cliA21StoreInventory(t, promotionRoot), inventoryBeforeCompilation) {
		t.Fatalf("CLI A2.1 restart/refusal matrix changed durable store state: %v", err)
	}
	if _, err := promotion.Finalize(context.Background(), restartedStore, reopenedReady, rejectedDecision); err == nil {
		t.Fatal("stale CHOICEPOINT_READY authority published a second DecisionRecord")
	}
	if _, _, err := restartedStore.Read(context.Background(), "DecisionRecord", rejectedDecision.Digest()); err == nil {
		t.Fatal("losing stale DecisionRecord was published before the head compare-and-swap")
	}
}

func TestCLIPhysicalReducerBudgetFenceRetainsOnlyBestKnown(t *testing.T) {
	appMode, err := countercli.PresentEnvironment("APP_MODE", "env")
	if err != nil {
		t.Fatal(err)
	}
	noise, err := countercli.PresentEnvironment("Z_NOISE", "ignored")
	if err != nil {
		t.Fatal(err)
	}
	configFixture, err := countercli.NewFixtureFile(
		"config.json", clifixture.ConfigJSON("config"), countercli.FixtureMode0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	noisyStimulus, err := countercli.NewCLIStimulus(countercli.CLIStimulusConfig{
		Executable: "node", BaseArgv: []string{clifixture.Entrypoint}, Argv: []string{"--mode", "argv"},
		Stdin: countercli.AbsentStdin(), Environment: []countercli.CLIEnvironmentBinding{appMode, noise},
		Fixtures: []countercli.CLIFixtureFile{configFixture}, CWDPolicy: countercli.CWDMaterializedRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	baselineConfig := cliPhysicalBestKnownConfig(t)
	baselineConfig.StimulusOverride = &noisyStimulus
	baselineResult, err := Run(context.Background(), baselineConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !baselineResult.HasOutcomeMap || baselineResult.OutcomeMap.Phase() != domain.AttemptDiscovery {
		t.Fatal("physical CLI budget-fence baseline did not produce a discovery outcome map")
	}
	baseline, err := compare.RequireDivergence(baselineResult.OutcomeMap)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := countercli.NewCLIReductionPolicy(countercli.CLIReductionPolicyConfig{
		Anchor: baselineResult.Stimulus, PinnedFixturePaths: []string{"config.json"},
		EnabledRules: []countercli.CLIReducerID{countercli.CLIEnvironmentRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	budget, err := reducer.NewBudgetFromWorldPlan(baselineResult.Plan)
	if err != nil {
		t.Fatal(err)
	}
	planDigest, planBound := budget.WorldPlanDigest()
	if !planBound || planDigest != baselineResult.Plan.Digest() || budget.ProposalLimit() != 5 ||
		budget.CandidateTrialLimit() != 27 || budget.WallLimit() != 3*time.Minute {
		t.Fatalf("compiled CLI budget fence lost plan authority: bound=%t plan=%s budget=%d/%d/%s",
			planBound, planDigest, budget.ProposalLimit(), budget.CandidateTrialLimit(), budget.WallLimit())
	}
	physicalEvaluations := 0
	var evaluatorErr error
	run, err := reducer.Run(context.Background(), reducer.RunInput[countercli.CLIStimulus]{
		Original: baselineResult.Stimulus,
		Reference: func(stimulus countercli.CLIStimulus) (domain.Digest, reducer.Measure, bool) {
			measure, measureErr := countercli.MeasureCLIStimulus(stimulus, policy)
			return stimulus.Digest(), measure, measureErr == nil
		},
		Enumerate: func(_ context.Context, stimulus countercli.CLIStimulus) ([]reducer.TypedProposal[countercli.CLIStimulus], error) {
			neighbors, enumerateErr := countercli.EnumerateCLINeighbors(stimulus, policy)
			if enumerateErr != nil {
				return nil, enumerateErr
			}
			proposals := make([]reducer.TypedProposal[countercli.CLIStimulus], len(neighbors))
			for index, neighbor := range neighbors {
				proposals[index] = reducer.TypedProposal[countercli.CLIStimulus]{
					Stimulus: neighbor.Stimulus(), Neighbor: neighbor.Neighbor(),
				}
			}
			return proposals, nil
		},
		Evaluate: func(ctx context.Context, stimulus countercli.CLIStimulus, _ reducer.Neighbor, purpose domain.AttemptPurpose, allowance reducer.EvaluationAllowance) (reducer.EvaluationObservation, error) {
			config := cliPhysicalBestKnownConfig(t)
			requiredTrials := uint64(config.MaxTotalTrials)
			if allowance.RemainingCandidateTrials < requiredTrials || !time.Now().Before(allowance.WallDeadline) {
				return reducer.EvaluationObservation{}, context.DeadlineExceeded
			}
			physicalEvaluations++
			config.Purpose = purpose
			config.StimulusOverride = &stimulus
			evaluationContext, cancel := context.WithDeadline(ctx, allowance.WallDeadline)
			defer cancel()
			result, runErr := Run(evaluationContext, config)
			if runErr != nil {
				evaluatorErr = runErr
				return reducer.EvaluationObservation{}, runErr
			}
			if !result.HasOutcomeMap {
				evaluatorErr = context.Canceled
				return reducer.EvaluationObservation{}, evaluatorErr
			}
			outcome := result.OutcomeMap
			return reducer.EvaluationObservation{OutcomeMap: &outcome, CandidateTrials: uint64(len(result.Trials))}, nil
		},
		Baseline: baseline, ReducerSet: policy.ReducerSet(), Budget: budget,
	})
	if err != nil {
		t.Fatal(err)
	}
	if evaluatorErr != nil {
		t.Fatalf("physical CLI budget-fence evaluator failed: %v", evaluatorErr)
	}
	transcript := run.Transcript()
	if !run.Valid() || run.DraftGrade() != reducer.GradeBestKnown || !run.HasAcceptedReduction() ||
		transcript.FinalSweepState() != reducer.FinalSweepIncomplete || len(transcript.Entries()) != 3 ||
		len(transcript.AcceptedPath()) != 1 || run.MinimizedStimulusDigest() == baselineResult.Stimulus.Digest() ||
		physicalEvaluations != 3 || !slices.Contains(transcript.Limitations(), "CANDIDATE_TRIAL_BUDGET_EXHAUSTED") {
		t.Fatalf("physical CLI budget fence was promoted or lost its accepted reduction: grade=%s accepted=%t sweep=%s entries=%d evaluations=%d limits=%v",
			run.DraftGrade(), run.HasAcceptedReduction(), transcript.FinalSweepState(), len(transcript.Entries()),
			physicalEvaluations, transcript.Limitations())
	}
	entries := transcript.Entries()
	wantDecisions := []reducer.Decision{reducer.Changes, reducer.Preserves, reducer.Changes}
	for index, entry := range entries {
		evaluation := entry.Evaluation()
		if evaluation.Purpose() != domain.AttemptReduction || evaluation.Decision() != wantDecisions[index] ||
			entry.ProposalCount() != uint64(index+1) || entry.CandidateTrials() != 9 ||
			entry.TrialCount() != uint64((index+1)*9) || !evaluation.LogicalNonReuseWithBaseline() {
			t.Fatalf("physical CLI budget-fence entry %d = purpose=%s decision=%s proposals=%d trials=%d/%d nonreuse=%t",
				index, evaluation.Purpose(), evaluation.Decision(), entry.ProposalCount(), entry.CandidateTrials(),
				entry.TrialCount(), evaluation.LogicalNonReuseWithBaseline())
		}
	}
	if draft, present, draftErr := run.CompletedSweepDraft(); draftErr != nil || present || draft.Valid() {
		t.Fatalf("incomplete physical CLI sweep exposed a completion draft: present=%t valid=%t err=%v",
			present, draft.Valid(), draftErr)
	}
	finalized, err := grade.Finalize(context.Background(), run, nil)
	if err != nil {
		t.Fatal(err)
	}
	finalGrade := finalized.Grade()
	if !finalized.Valid() || !finalGrade.Valid() || finalGrade.Status() != grade.StatusBestKnown ||
		!slices.Contains(finalGrade.Limitations(), "CANDIDATE_TRIAL_BUDGET_EXHAUSTED") ||
		!slices.Contains(finalGrade.Limitations(), "DURABLE_SWEEP_AUTHORITY_ABSENT") {
		t.Fatalf("physical CLI budget fence did not retain an honest BEST_KNOWN grade: status=%s limits=%v",
			finalGrade.Status(), finalGrade.Limitations())
	}
	if _, quantified := finalGrade.QuantifiedReducerSetDigest(); quantified {
		t.Fatal("BEST_KNOWN physical CLI budget fence exposed a local-minimality quantifier")
	}
	if _, completed := finalGrade.CompletedSweepDigest(); completed {
		t.Fatal("BEST_KNOWN physical CLI budget fence exposed a completed-sweep digest")
	}
}

func cliPhysicalReductionConfig(t *testing.T) Config {
	t.Helper()
	config := referenceConfig(t)
	config.ProjectionFields = []countercli.CLIFieldID{
		countercli.CLIFieldStdoutBytes,
		countercli.CLIFieldStdoutJSONMode,
		countercli.CLIFieldStdoutJSONSource,
	}
	config.ReductionProposalLimit = 5
	// The plan reserves 18 trials for discovery+confirmation and 36 for U5.
	config.ReductionTotalCandidateTrials = 54
	config.ReductionWallMS = (3 * time.Minute).Milliseconds()
	return config
}

func cliPhysicalBestKnownConfig(t *testing.T) Config {
	t.Helper()
	config := referenceConfig(t)
	config.ReductionProposalLimit = 5
	// The plan reserves 18 trials for discovery+confirmation and only 27 for U5:
	// exactly three physical evaluations, with no authority left for the final sweep.
	config.ReductionTotalCandidateTrials = 45
	config.ReductionWallMS = (3 * time.Minute).Milliseconds()
	return config
}
