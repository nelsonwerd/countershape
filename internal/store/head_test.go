//go:build darwin

package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
)

func TestStudyHeadLegalLineageAndStaleCallsCreateNoFiles(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	study, err := NewStudyID("legal lineage / raw title never enters paths")
	if err != nil {
		t.Fatal(err)
	}
	head, err := value.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan"))
	if err != nil {
		t.Fatal(err)
	}
	if head.Revision() != 1 || head.Stage() != StageSourcePlan || strings.Contains(root, study.String()) {
		t.Fatal("initial head or hashed study namespace is invalid")
	}
	stages := []struct {
		stage LineageStage
		kind  string
	}{
		{StageBaseline, "CandidateOutcomeMap"}, {StageDivergence, "CandidateOutcomeMap"},
		{StageReduction, "ReductionRun"}, {StageConfirmation, "FreshConfirmation"},
		{StageChoicepointReady, "Choicepoint"}, {StageRuling, "DecisionRecord"},
		{StageResidue, "ContractBundle"},
	}
	stale := head
	for index, step := range stages {
		next := semanticObjectForTest(t, step.kind, fmt.Sprintf("step-%d", index))
		head, err = value.advanceHead(context.Background(), head, step.stage, next)
		if err != nil {
			t.Fatalf("advance to %s: %v", step.stage, err)
		}
		if head.Revision() != int64(index+2) || head.Stage() != step.stage || head.CurrentDigest() != next.Digest() {
			t.Fatalf("advanced head = %#v", head)
		}
	}
	before := filesystemSnapshot(t, root)
	for _, attack := range []struct {
		name  string
		head  HeadToken
		stage LineageStage
		kind  string
	}{
		{"stale-replay", stale, StageBaseline, "CandidateOutcomeMap"},
		{"backward", head, StageChoicepointReady, "Choicepoint"},
		{"wrong-kind", head, StageResidue, "Choicepoint"},
	} {
		t.Run(attack.name, func(t *testing.T) {
			if _, err := value.advanceHead(context.Background(), attack.head, attack.stage,
				semanticObjectForTest(t, attack.kind, attack.name)); err == nil {
				t.Fatal("illegal or stale transition was accepted")
			}
			assertFilesystemSnapshot(t, root, before)
		})
	}
	opened, err := value.OpenHead(context.Background(), study)
	if err != nil || !sameHeadToken(opened, head) {
		t.Fatalf("exact current head did not reopen: %v", err)
	}
}

func TestStaleHeadRefusesBeforeSuccessorObjectPublication(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	study, _ := NewStudyID("stale compare precedes publication")
	stale, err := value.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := value.advanceHead(
		context.Background(), stale, StageBaseline,
		semanticObjectForTest(t, "CandidateOutcomeMap", "winning baseline"),
	); err != nil {
		t.Fatal(err)
	}
	loser := semanticObjectForTest(t, "CandidateOutcomeMap", "stale losing baseline")
	before := filesystemSnapshot(t, root)
	_, err = value.advanceHead(context.Background(), stale, StageBaseline, loser)
	var storeErr *Error
	if !errors.As(err, &storeErr) || storeErr.Code != codeCASConflict {
		t.Fatalf("stale head refusal = %v, want exact %s", err, codeCASConflict)
	}
	if _, _, err := value.Read(context.Background(), loser.Kind(), loser.Digest()); err == nil {
		t.Fatal("stale successor object was published before the head compare-and-swap")
	}
	assertFilesystemSnapshot(t, root, before)
}

func TestCancelledContextRefusesHeadCreationAndAdvanceBeforePublication(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	study, _ := NewStudyID("cancelled create has no publication")
	beforeCreate := filesystemSnapshot(t, root)
	if _, err := value.createStudyObject(
		cancelled, study, semanticObjectForTest(t, "WorldPlan", "cancelled plan"),
	); err == nil {
		t.Fatal("cancelled context created a study")
	}
	assertFilesystemSnapshot(t, root, beforeCreate)

	liveStudy, _ := NewStudyID("cancelled advance has no publication")
	head, err := value.createStudyObject(
		context.Background(), liveStudy, semanticObjectForTest(t, "WorldPlan", "live plan"),
	)
	if err != nil {
		t.Fatal(err)
	}
	next := semanticObjectForTest(t, "CandidateOutcomeMap", "cancelled successor")
	beforeAdvance := filesystemSnapshot(t, root)
	if _, err := value.advanceHead(cancelled, head, StageBaseline, next); err == nil {
		t.Fatal("cancelled context advanced a study")
	}
	if _, _, err := value.Read(context.Background(), next.Kind(), next.Digest()); err == nil {
		t.Fatal("cancelled advance published its successor object")
	}
	assertFilesystemSnapshot(t, root, beforeAdvance)
}

func TestPartialAndCancelledEvidenceNeverAdvanceStudyHead(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	study, err := NewStudyID("nonadvancing terminal evidence")
	if err != nil {
		t.Fatal(err)
	}
	head, err := value.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan"))
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range []SemanticObject{
		semanticObjectForTest(t, "PartialAttempt", "partial evidence"),
		semanticObjectForTest(t, "CancelledAttempt", "cancelled evidence"),
	} {
		authority, err := value.Publish(context.Background(), object)
		if err != nil {
			t.Fatalf("publish %s: %v", object.Kind(), err)
		}
		if err := value.Validate(context.Background(), object, authority); err != nil {
			t.Fatalf("validate %s: %v", object.Kind(), err)
		}
	}

	before := filesystemSnapshot(t, root)
	for _, transition := range []struct {
		stage LineageStage
		kind  string
	}{
		{StagePartial, "PartialAttempt"},
		{StageCancelled, "CancelledAttempt"},
	} {
		if _, err := value.advanceHead(context.Background(), head, transition.stage,
			semanticObjectForTest(t, transition.kind, "must not become a head")); err == nil {
			t.Fatalf("%s evidence advanced the study head", transition.stage)
		}
		assertFilesystemSnapshot(t, root, before)
	}
	opened, err := value.OpenHead(context.Background(), study)
	if err != nil || !sameHeadToken(opened, head) {
		t.Fatalf("nonadvancing evidence changed the head: %v", err)
	}
}

func TestRestartedStoreRejectsReplacedStudiesNamespaceBeforeOpenHead(t *testing.T) {
	first, root := newObjectStoreForTest(t)
	study, err := NewStudyID("replaced studies namespace")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan")); err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	studyPath, err := first.existingStudyPath(study)
	if err != nil {
		t.Fatal(err)
	}
	oldStudies := filepath.Join(root, "studies-replaced")
	if err := os.Rename(restarted.studies, oldStudies); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(restarted.studies, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(oldStudies, filepath.Base(studyPath)), filepath.Join(restarted.studies, filepath.Base(studyPath))); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.OpenHead(context.Background(), study); err == nil {
		t.Fatal("restarted store accepted a replaced studies namespace")
	}
}

func TestLaterHeadRefusesMissingLineageRootWorldPlan(t *testing.T) {
	value, _ := newObjectStoreForTest(t)
	study, err := NewStudyID("lineage root remains durable")
	if err != nil {
		t.Fatal(err)
	}
	initial := semanticObjectForTest(t, "WorldPlan", "lineage root")
	head, err := value.createStudyObject(context.Background(), study, initial)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := value.advanceHead(context.Background(), head, StageBaseline,
		semanticObjectForTest(t, "CandidateOutcomeMap", "baseline")); err != nil {
		t.Fatal(err)
	}
	rootPath, _, err := value.existingObjectPath(initial.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(rootPath); err != nil {
		t.Fatal(err)
	}
	if _, err := value.OpenHead(context.Background(), study); err == nil {
		t.Fatal("later head reopened without its lineage-root WorldPlan")
	}
}

func TestStudyHeadConcurrentStoreInstancesHaveExactlyOneWinner(t *testing.T) {
	first, root := newObjectStoreForTest(t)
	study, _ := NewStudyID("concurrent in-process stores")
	initial, err := first.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	otherExpected, err := second.OpenHead(context.Background(), study)
	if err != nil {
		t.Fatal(err)
	}
	stores := []*ObjectStore{first, second}
	expected := []HeadToken{initial, otherExpected}
	errorsByWriter := make([]error, 2)
	start := make(chan struct{})
	var writers sync.WaitGroup
	for index := range stores {
		writers.Add(1)
		go func(index int) {
			defer writers.Done()
			<-start
			_, errorsByWriter[index] = stores[index].advanceHead(
				context.Background(), expected[index], StageBaseline,
				semanticObjectForTest(t, "CandidateOutcomeMap", fmt.Sprintf("writer-%d", index)),
			)
		}(index)
	}
	close(start)
	writers.Wait()
	winners, conflicts := 0, 0
	for _, err := range errorsByWriter {
		if err == nil {
			winners++
			continue
		}
		var storeErr *Error
		if errors.As(err, &storeErr) && storeErr.Code == codeCASConflict {
			conflicts++
			continue
		}
		t.Fatalf("unexpected writer refusal: %v", err)
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("writers: winners=%d conflicts=%d errors=%v", winners, conflicts, errorsByWriter)
	}
}

func TestStudyHeadCASBindsEveryExpectedTokenFieldBeforePublication(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	study, _ := NewStudyID("field exact CAS")
	head, err := value.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan"))
	if err != nil {
		t.Fatal(err)
	}
	priorHead := head
	head, err = value.advanceHead(
		context.Background(), priorHead, StageBaseline, semanticObjectForTest(t, "CandidateOutcomeMap", "baseline"),
	)
	if err != nil {
		t.Fatal(err)
	}
	previousHead, hasPreviousHead := head.PreviousHeadDigest()
	if !hasPreviousHead || previousHead != priorHead.HeadDigest() {
		t.Fatalf("baseline previous head = %v/%v, want %s", previousHead, hasPreviousHead, priorHead.HeadDigest())
	}
	otherStudy, _ := NewStudyID("field exact CAS other existing study")
	otherHead, err := value.createStudyObject(context.Background(), otherStudy, semanticObjectForTest(t, "WorldPlan", "other plan"))
	if err != nil {
		t.Fatal(err)
	}
	otherPriorHead := otherHead
	otherHead, err = value.advanceHead(
		context.Background(), otherPriorHead, StageBaseline, semanticObjectForTest(t, "CandidateOutcomeMap", "other baseline"),
	)
	if err != nil {
		t.Fatal(err)
	}
	otherPreviousHead, otherHasPreviousHead := otherHead.PreviousHeadDigest()
	if !otherHasPreviousHead || otherPreviousHead != otherPriorHead.HeadDigest() {
		t.Fatalf("other baseline previous head = %v/%v, want %s", otherPreviousHead, otherHasPreviousHead, otherPriorHead.HeadDigest())
	}
	mutations := []struct {
		name   string
		mutate func(*HeadToken)
	}{
		{"study", func(token *HeadToken) { token.study = otherStudy }},
		{"revision", func(token *HeadToken) { token.revision++ }},
		{"stage", func(token *HeadToken) { token.stage = StageDivergence }},
		{"current-kind", func(token *HeadToken) { token.currentKind = "FreshConfirmation" }},
		{"current-digest", func(token *HeadToken) { token.currentDigest = otherHead.currentDigest }},
		{"previous-head", func(token *HeadToken) { token.previousHead = otherHead.previousHead }},
		{"previous-object", func(token *HeadToken) { token.previousObject = otherHead.previousObject }},
		{"lineage-root", func(token *HeadToken) { token.lineageRoot = otherHead.lineageRoot }},
		{"head-digest", func(token *HeadToken) { token.headDigest = otherHead.headDigest }},
	}
	before := filesystemSnapshot(t, root)
	for index, mutation := range mutations {
		expected := head
		mutation.mutate(&expected)
		if !expected.validFor(value) {
			t.Fatalf("CAS %s mutant accidentally invalidated the opaque token before comparison", mutation.name)
		}
		next := semanticObjectForTest(t, "CandidateOutcomeMap", fmt.Sprintf("field-mutant-%d", index))
		_, err := value.advanceHead(context.Background(), expected, StageDivergence, next)
		var storeErr *Error
		if !errors.As(err, &storeErr) || storeErr.Code != codeCASConflict {
			t.Fatalf("mutated %s refusal = %v, want exact %s", mutation.name, err, codeCASConflict)
		}
		if _, _, readErr := value.Read(context.Background(), next.Kind(), next.Digest()); readErr == nil {
			t.Fatalf("mutated %s token published its proposed successor", mutation.name)
		}
		assertFilesystemSnapshot(t, root, before)
	}

	foreign, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	foreignToken, err := foreign.OpenHead(context.Background(), study)
	if err != nil {
		t.Fatal(err)
	}
	foreignNext := semanticObjectForTest(t, "CandidateOutcomeMap", "foreign store instance")
	if _, err := value.advanceHead(context.Background(), foreignToken, StageDivergence, foreignNext); err == nil {
		t.Fatal("foreign store-instance token was accepted")
	}
	if _, _, err := value.Read(context.Background(), foreignNext.Kind(), foreignNext.Digest()); err == nil {
		t.Fatal("foreign store-instance refusal published its proposed successor")
	}
	assertFilesystemSnapshot(t, root, before)
}

func TestEveryIllegalStudyHeadEdgeRefusesWithoutPublishing(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	stages := []LineageStage{
		StageSourcePlan, StageBaseline, StageDivergence, StageReduction,
		StageConfirmation, StageChoicepointReady, StageRuling, StageResidue,
	}
	for currentIndex, currentStage := range stages {
		study, _ := NewStudyID(fmt.Sprintf("illegal edge matrix current %s", currentStage))
		head, err := value.createStudyObject(
			context.Background(), study,
			semanticObjectForTest(t, kindForHeadStageTest(StageSourcePlan), fmt.Sprintf("matrix-%d-source", currentIndex)),
		)
		if err != nil {
			t.Fatal(err)
		}
		for stageIndex := 1; stageIndex <= currentIndex; stageIndex++ {
			stage := stages[stageIndex]
			head, err = value.advanceHead(
				context.Background(), head, stage,
				semanticObjectForTest(t, kindForHeadStageTest(stage), fmt.Sprintf("matrix-%d-%d", currentIndex, stageIndex)),
			)
			if err != nil {
				t.Fatalf("reach %s: %v", currentStage, err)
			}
		}
		before := filesystemSnapshot(t, root)
		for _, nextStage := range append(append([]LineageStage{}, stages...), StagePartial, StageCancelled) {
			if permittedTransition(currentStage, nextStage) {
				continue
			}
			next := semanticObjectForTest(
				t, kindForHeadStageTest(nextStage), fmt.Sprintf("illegal-%s-to-%s", currentStage, nextStage),
			)
			if _, err := value.advanceHead(context.Background(), head, nextStage, next); err == nil {
				t.Fatalf("illegal edge %s -> %s was accepted", currentStage, nextStage)
			}
			if _, _, err := value.Read(context.Background(), next.Kind(), next.Digest()); err == nil {
				t.Fatalf("illegal edge %s -> %s published its object", currentStage, nextStage)
			}
			assertFilesystemSnapshot(t, root, before)
		}
	}
}

func kindForHeadStageTest(stage LineageStage) string {
	switch stage {
	case StageSourcePlan:
		return "WorldPlan"
	case StageBaseline, StageDivergence:
		return "CandidateOutcomeMap"
	case StageReduction:
		return "ReductionRun"
	case StageConfirmation:
		return "FreshConfirmation"
	case StageChoicepointReady:
		return "Choicepoint"
	case StageRuling:
		return "DecisionRecord"
	case StageResidue:
		return "ContractBundle"
	case StagePartial:
		return "PartialAttempt"
	case StageCancelled:
		return "CancelledAttempt"
	default:
		return "Unknown"
	}
}

func TestStudyHeadCASHelper(t *testing.T) {
	root := os.Getenv("COUNTERSHAPE_U6_CAS_ROOT")
	if root == "" {
		return
	}
	id := os.Getenv("COUNTERSHAPE_U6_CAS_ID")
	value, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	study, _ := NewStudyID("cross-process head CAS")
	expected, err := value.OpenHead(context.Background(), study)
	if err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(root, "ready."+id)
	if err := os.WriteFile(ready, []byte("ready"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate := filepath.Join(root, "go")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Lstat(gate); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cross-process barrier timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
	_, err = value.advanceHead(context.Background(), expected, StageBaseline,
		semanticObjectForTest(t, "CandidateOutcomeMap", "process-"+id))
	if err == nil {
		fmt.Println("WIN")
		return
	}
	var storeErr *Error
	if errors.As(err, &storeErr) && storeErr.Code == codeCASConflict {
		fmt.Println("CONFLICT")
		return
	}
	t.Fatal(err)
}

func TestStudyHeadCrossProcessCASHasExactlyOneWinner(t *testing.T) {
	value, root := newObjectStoreForTest(t)
	study, _ := NewStudyID("cross-process head CAS")
	if _, err := value.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan")); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	type child struct {
		command *exec.Cmd
		output  bytes.Buffer
	}
	children := make([]child, 2)
	for index := range children {
		id := strconv.Itoa(index)
		children[index].command = exec.Command(executable, "-test.run=^TestStudyHeadCASHelper$")
		children[index].command.Env = append(os.Environ(),
			"COUNTERSHAPE_U6_CAS_ROOT="+root, "COUNTERSHAPE_U6_CAS_ID="+id)
		children[index].command.Stdout = &children[index].output
		children[index].command.Stderr = &children[index].output
		if err := children[index].command.Start(); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		ready := 0
		for index := range children {
			if _, err := os.Lstat(filepath.Join(root, "ready."+strconv.Itoa(index))); err == nil {
				ready++
			}
		}
		if ready == len(children) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("children did not reach CAS barrier")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(root, "go"), []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	results := make([]string, len(children))
	for index := range children {
		if err := children[index].command.Wait(); err != nil {
			t.Fatalf("child %d: %v: %s", index, err, children[index].output.String())
		}
		output := children[index].output.String()
		if strings.Contains(output, "WIN") {
			results[index] = "WIN"
		} else if strings.Contains(output, "CONFLICT") {
			results[index] = "CONFLICT"
		} else {
			t.Fatalf("child %d emitted no verdict: %s", index, output)
		}
	}
	sortedResults := append([]string(nil), results...)
	sort.Strings(sortedResults)
	if strings.Join(sortedResults, ",") != "CONFLICT,WIN" {
		t.Fatalf("cross-process verdicts = %v", results)
	}
	for index, verdict := range results {
		object := semanticObjectForTest(t, "CandidateOutcomeMap", "process-"+strconv.Itoa(index))
		_, _, err := value.existingObjectPath(object.Digest())
		if verdict == "WIN" {
			if err != nil {
				t.Fatalf("winner path: %v", err)
			}
			reopened, authority, err := value.Read(context.Background(), object.Kind(), object.Digest())
			if err != nil || !sameSemanticObject(reopened, object) || value.Validate(context.Background(), reopened, authority) != nil {
				t.Fatalf("winner object did not reopen exactly: %v", err)
			}
			continue
		}
		if _, _, readErr := value.Read(context.Background(), object.Kind(), object.Digest()); readErr == nil {
			t.Fatal("CAS loser published an immutable successor")
		}
	}
}

func TestHeadCorruptionAndUnknownMembersFailClosed(t *testing.T) {
	value, _ := newObjectStoreForTest(t)
	study, _ := NewStudyID("corrupt head")
	if _, err := value.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan")); err != nil {
		t.Fatal(err)
	}
	studyPath, err := value.existingStudyPath(study)
	if err != nil {
		t.Fatal(err)
	}
	headPath := filepath.Join(studyPath, studyHeadFilename)
	body, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatal(err)
	}
	var identity map[string]any
	if err := json.Unmarshal(body, &identity); err != nil {
		t.Fatal(err)
	}
	identity["unknown_member"] = "must-refuse"
	corrupt, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(headPath, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := value.OpenHead(context.Background(), study); err == nil {
		t.Fatal("head with an unknown member reopened")
	}
}

func TestHeadParserRejectsStageRevisionSkip(t *testing.T) {
	value, _ := newObjectStoreForTest(t)
	study, _ := NewStudyID("stage revision mismatch")
	head, err := value.createStudyObject(context.Background(), study, semanticObjectForTest(t, "WorldPlan", "plan"))
	if err != nil {
		t.Fatal(err)
	}
	head, err = value.advanceHead(context.Background(), head, StageBaseline,
		semanticObjectForTest(t, "CandidateOutcomeMap", "baseline"))
	if err != nil {
		t.Fatal(err)
	}
	residue := semanticObjectForTest(t, "ContractBundle", "forged skipped stage")
	if _, err := value.Publish(context.Background(), residue); err != nil {
		t.Fatal(err)
	}
	studyPath, err := value.existingStudyPath(study)
	if err != nil {
		t.Fatal(err)
	}
	headPath := filepath.Join(studyPath, studyHeadFilename)
	body, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatal(err)
	}
	var identity map[string]any
	if err := json.Unmarshal(body, &identity); err != nil {
		t.Fatal(err)
	}
	identity["stage"] = string(StageResidue)
	identity["current_kind"] = residue.Kind()
	identity["current_digest"] = residue.Digest().String()
	encoded, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	forged, err := canon.Canonicalize(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(headPath, forged, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := value.OpenHead(context.Background(), study); err == nil {
		t.Fatal("head parser accepted revision 2 at RESIDUE")
	}
}

type snapshotEntry struct {
	mode os.FileMode
	body []byte
}

func filesystemSnapshot(t *testing.T, root string) map[string]snapshotEntry {
	t.Helper()
	result := map[string]snapshotEntry{}
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
		value := snapshotEntry{mode: info.Mode()}
		if info.Mode().IsRegular() {
			value.body, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		result[relative] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertFilesystemSnapshot(t *testing.T, root string, expected map[string]snapshotEntry) {
	t.Helper()
	actual := filesystemSnapshot(t, root)
	if len(actual) != len(expected) {
		t.Fatalf("filesystem inventory changed: before=%d after=%d", len(expected), len(actual))
	}
	for path, before := range expected {
		after, present := actual[path]
		if !present || before.mode != after.mode || !bytes.Equal(before.body, after.body) {
			t.Fatalf("filesystem entry %q changed after refused transition", path)
		}
	}
}
