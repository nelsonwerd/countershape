//go:build darwin

package promotion_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/choice"
	"github.com/nelsonwerd/countershape/internal/choice/promotion"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/store"
)

type legacySnapshotFixtures struct {
	choicepointBytes []byte
	decisionBytes    []byte
	worldPlanBytes   []byte
	worldPlanDigest  domain.Digest
	choicepoint      choice.ChoicepointRecord
	decision         choice.DecisionRecord
}

type snapshotHeadWire struct {
	SchemaVersion        string `json:"schema_version"`
	Kind                 string `json:"kind"`
	StudyID              string `json:"study_id"`
	Revision             int64  `json:"revision"`
	Stage                string `json:"stage"`
	CurrentKind          string `json:"current_kind"`
	CurrentDigest        string `json:"current_digest"`
	LineageRootDigest    string `json:"lineage_root_digest"`
	PreviousHeadDigest   string `json:"previous_head_digest"`
	PreviousObjectDigest string `json:"previous_object_digest"`
}

type snapshotHeadStep struct {
	stage   store.LineageStage
	kind    string
	digest  domain.Digest
	content []byte
}

func TestPreUpgradeLegacyChoiceSnapshotsReopenWithoutPortableUpgrade(t *testing.T) {
	fixtures := loadLegacySnapshotFixtures(t)
	for _, target := range []struct {
		name     string
		revision int
	}{
		{name: "ready", revision: 6},
		{name: "ruling", revision: 7},
	} {
		root, study := materializeLegacySnapshot(t, fixtures, target.revision)
		reopened, err := store.OpenObjectStore(root)
		if err != nil {
			t.Fatalf("open %s legacy snapshot: %v", target.name, err)
		}
		if target.revision == 6 {
			ready, openErr := promotion.OpenReady(context.Background(), reopened, study)
			if openErr != nil || !bytes.Equal(ready.Record().CanonicalBytes(), fixtures.choicepointBytes) {
				t.Fatalf("legacy ready snapshot did not reopen byte-exactly: %v", openErr)
			}
			continue
		}

		ruling, openErr := promotion.OpenRuling(context.Background(), reopened, study)
		if openErr != nil || !bytes.Equal(ruling.Record().CanonicalBytes(), fixtures.decisionBytes) ||
			ruling.Record().ChoicepointDigest() != fixtures.choicepoint.Digest() {
			t.Fatalf("legacy ruling snapshot did not reopen byte-exactly: %v", openErr)
		}
		storedChoicepoint, authority, readErr := reopened.Read(context.Background(), "Choicepoint", fixtures.choicepoint.Digest())
		if readErr != nil || reopened.Validate(context.Background(), storedChoicepoint, authority) != nil ||
			!bytes.Equal(storedChoicepoint.CanonicalBytes(), fixtures.choicepointBytes) {
			t.Fatalf("legacy ruling lost its exact Choicepoint predecessor: %v", readErr)
		}
		before, err := reopened.OpenHead(context.Background(), study)
		if err != nil {
			t.Fatal(err)
		}
		preparation, prepareErr := promotion.PreparePortableRuling(context.Background(), reopened, ruling)
		if preparation.Valid() || !choice.IsRefusal(prepareErr, choice.CodeLegacyWholeProjectionNotPortable) {
			t.Fatalf("legacy portable preparation validity=%t error=%v", preparation.Valid(), prepareErr)
		}
		after, err := reopened.OpenHead(context.Background(), study)
		if err != nil || before.HeadDigest() != after.HeadDigest() || after.Stage() != store.StageRuling ||
			after.CurrentDigest() != fixtures.decision.Digest() {
			t.Fatalf("legacy refusal changed the durable head: %v", err)
		}
		storedDecision, decisionAuthority, readErr := reopened.Read(context.Background(), "DecisionRecord", fixtures.decision.Digest())
		if readErr != nil || reopened.Validate(context.Background(), storedDecision, decisionAuthority) != nil ||
			!bytes.Equal(storedDecision.CanonicalBytes(), fixtures.decisionBytes) {
			t.Fatalf("legacy refusal changed the durable DecisionRecord: %v", readErr)
		}
	}
}

func loadLegacySnapshotFixtures(t testing.TB) legacySnapshotFixtures {
	t.Helper()
	choicepointBytes := legacyExampleBytes(t, "choicepoint.valid.json")
	decisionBytes := legacyExampleBytes(t, "decision-record.valid.json")
	choicepoint, err := choice.ParseChoicepointRecord(choicepointBytes)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := choice.ParseDecisionRecord(decisionBytes, choicepoint)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		WorldPlanBase64 string `json:"world_plan_base64"`
		WorldPlanDigest string `json:"world_plan_digest"`
	}
	if err := json.Unmarshal(choicepointBytes, &envelope); err != nil {
		t.Fatal(err)
	}
	worldPlanBytes, err := base64.StdEncoding.Strict().DecodeString(envelope.WorldPlanBase64)
	if err != nil || base64.StdEncoding.EncodeToString(worldPlanBytes) != envelope.WorldPlanBase64 {
		t.Fatalf("legacy WorldPlan is not canonical padded base64: %v", err)
	}
	worldPlanDigest, err := domain.ParseDigest(envelope.WorldPlanDigest)
	if err != nil || choicepoint.ConfirmationRecord().PlanDigest() != worldPlanDigest {
		t.Fatalf("legacy WorldPlan lineage differs: %v", err)
	}
	return legacySnapshotFixtures{
		choicepointBytes: choicepointBytes, decisionBytes: decisionBytes,
		worldPlanBytes: worldPlanBytes, worldPlanDigest: worldPlanDigest,
		choicepoint: choicepoint, decision: decision,
	}
}

func materializeLegacySnapshot(t testing.TB, fixtures legacySnapshotFixtures, targetRevision int) (string, store.StudyID) {
	t.Helper()
	temporaryRoot, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(temporaryRoot, "legacy-store")
	seed, err := store.OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := fixtures.choicepoint.ConfirmationRecord()
	steps := []snapshotHeadStep{
		{stage: store.StageSourcePlan, kind: "WorldPlan", digest: fixtures.worldPlanDigest, content: fixtures.worldPlanBytes},
		{stage: store.StageBaseline, kind: "CandidateOutcomeMap", digest: confirmation.OriginalBaselineMap().ArtifactDigest().DomainDigest(), content: confirmation.OriginalBaselineMap().CanonicalBytes()},
		{stage: store.StageDivergence, kind: "CandidateOutcomeMap", digest: confirmation.ReducedBaselineMap().ArtifactDigest().DomainDigest(), content: confirmation.ReducedBaselineMap().CanonicalBytes()},
		{stage: store.StageReduction, kind: "ReductionRun", digest: confirmation.ReductionRunRecord().Digest(), content: confirmation.ReductionRunRecord().CanonicalBytes()},
		{stage: store.StageConfirmation, kind: "FreshConfirmation", digest: confirmation.Digest(), content: confirmation.CanonicalBytes()},
		{stage: store.StageChoicepointReady, kind: "Choicepoint", digest: fixtures.choicepoint.Digest(), content: fixtures.choicepointBytes},
		{stage: store.StageRuling, kind: "DecisionRecord", digest: fixtures.decision.Digest(), content: fixtures.decisionBytes},
	}
	for _, step := range steps {
		object, objectErr := store.NewSemanticObject(step.kind, step.digest, step.content)
		if objectErr != nil {
			t.Fatalf("legacy %s object: %v", step.kind, objectErr)
		}
		authority, publishErr := seed.Publish(context.Background(), object)
		if publishErr != nil || seed.Validate(context.Background(), object, authority) != nil {
			t.Fatalf("publish legacy %s object: %v", step.kind, publishErr)
		}
	}
	study, err := store.NewStudyID("pre-upgrade legacy " + steps[targetRevision-1].kind)
	if err != nil {
		t.Fatal(err)
	}
	headBytes := buildSnapshotHead(t, study, fixtures.worldPlanDigest, steps, targetRevision)
	studyPath := filepath.Join(root, "studies", strings.TrimPrefix(study.String(), "study:"))
	if err := os.Mkdir(studyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(studyPath, 0o700); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(filepath.Join(studyPath, ".head.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	headPath := filepath.Join(studyPath, "head.json")
	if err := os.WriteFile(headPath, headBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(studyPath, ".head.lock"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(headPath, 0o600); err != nil {
		t.Fatal(err)
	}
	return root, study
}

func buildSnapshotHead(
	t testing.TB,
	study store.StudyID,
	root domain.Digest,
	steps []snapshotHeadStep,
	targetRevision int,
) []byte {
	t.Helper()
	if targetRevision < 1 || targetRevision > len(steps) {
		t.Fatal("legacy snapshot target revision is outside the closed lineage")
	}
	previousHead := ""
	for index, step := range steps[:targetRevision] {
		previousObject := ""
		if index > 0 {
			previousObject = steps[index-1].digest.String()
		}
		wire := snapshotHeadWire{
			SchemaVersion: domain.SchemaVersion, Kind: "StudyHead", StudyID: study.String(),
			Revision: int64(index + 1), Stage: string(step.stage), CurrentKind: step.kind,
			CurrentDigest: step.digest.String(), LineageRootDigest: root.String(),
			PreviousHeadDigest: previousHead, PreviousObjectDigest: previousObject,
		}
		digest, canonical, err := canon.DigestTyped("StudyHead", wire)
		if err != nil {
			t.Fatal(err)
		}
		previousHead = digest.String()
		if index+1 == targetRevision {
			return canonical
		}
	}
	t.Fatal("legacy snapshot head was not built")
	return nil
}

func legacyExampleBytes(t testing.TB, name string) []byte {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve legacy fixture path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "spec", "examples", "v1", name))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 2 || raw[len(raw)-1] != '\n' || bytes.Contains(raw[:len(raw)-1], []byte{'\n'}) {
		t.Fatalf("%s must be one canonical line plus one packaging newline", name)
	}
	exact := append([]byte(nil), raw[:len(raw)-1]...)
	value, err := canon.Parse(exact)
	canonical, canonicalErr := value.CanonicalChecked()
	if err != nil || canonicalErr != nil || !bytes.Equal(canonical, exact) {
		t.Fatalf("%s is not exact canonical JSON: %v", name, err)
	}
	return exact
}
