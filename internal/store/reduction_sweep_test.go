package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/reduce"
)

type storeTestCompletedSweepEvaluation struct {
	EvaluationID                  string   `json:"evaluation_id"`
	ProposalDigest                string   `json:"proposal_digest"`
	NeighborStimulusDigest        string   `json:"neighbor_stimulus_digest"`
	NeighborMeasureDigest         string   `json:"neighbor_measure_digest"`
	ObservedOutcomeMapDigest      string   `json:"observed_outcome_map_digest"`
	ObservedPreservationMapDigest string   `json:"observed_preservation_map_digest"`
	Decision                      string   `json:"decision"`
	ReasonCode                    string   `json:"reason_code"`
	BatchDigests                  []string `json:"batch_digests"`
	AttemptDigests                []string `json:"attempt_digests"`
	WorldDigests                  []string `json:"world_digests"`
	ObservationDigests            []string `json:"observation_digests"`
}

type storeTestCompletedSweepIdentity struct {
	SchemaVersion                 string                              `json:"schema_version"`
	Kind                          string                              `json:"kind"`
	RunDigest                     string                              `json:"run_digest"`
	TranscriptDigest              string                              `json:"transcript_digest"`
	BaselineOutcomeMapDigest      string                              `json:"baseline_outcome_map_digest"`
	BaselinePreservationMapDigest string                              `json:"baseline_preservation_map_digest"`
	CurrentStimulusDigest         string                              `json:"current_stimulus_digest"`
	CurrentMeasureDefinition      string                              `json:"current_measure_definition_digest"`
	CurrentMeasureComponents      []int64                             `json:"current_measure_components"`
	CurrentMeasureDigest          string                              `json:"current_measure_digest"`
	ReducerSetDigest              string                              `json:"reducer_set_digest"`
	EnumerationCompleted          bool                                `json:"enumeration_completed"`
	Cancelled                     bool                                `json:"cancelled"`
	BudgetExhausted               bool                                `json:"budget_exhausted"`
	BaselineStale                 bool                                `json:"baseline_stale"`
	EnumeratedNeighborDigests     []string                            `json:"enumerated_neighbor_digests"`
	Evaluations                   []storeTestCompletedSweepEvaluation `json:"evaluations"`
}

func storeTestDigest(number int) domain.Digest {
	return domain.MustDigest(fmt.Sprintf("sha256:%064x", number))
}

func emptyCompletedSweepDraft(t *testing.T, seed int) reduce.CompletedSweepDraft {
	t.Helper()
	definition := storeTestDigest(seed*100 + 1)
	measure, err := reduce.NewMeasure(definition, []uint64{uint64(seed + 1), 0})
	if err != nil {
		t.Fatal(err)
	}
	identity := storeTestCompletedSweepIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "CompletedSweepDraft",
		RunDigest: storeTestDigest(seed*100 + 2).String(), TranscriptDigest: storeTestDigest(seed*100 + 3).String(),
		BaselineOutcomeMapDigest:      storeTestDigest(seed*100 + 4).String(),
		BaselinePreservationMapDigest: storeTestDigest(seed*100 + 5).String(),
		CurrentStimulusDigest:         storeTestDigest(seed*100 + 6).String(),
		CurrentMeasureDefinition:      definition.String(), CurrentMeasureComponents: []int64{int64(seed + 1), 0},
		CurrentMeasureDigest: measure.Digest().String(), ReducerSetDigest: storeTestDigest(seed*100 + 7).String(),
		EnumerationCompleted: true, Cancelled: false, BudgetExhausted: false, BaselineStale: false,
		EnumeratedNeighborDigests: []string{}, Evaluations: []storeTestCompletedSweepEvaluation{},
	}
	_, canonicalBytes, err := canon.DigestTyped("CompletedSweepDraft", identity)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := reduce.ParseCompletedSweepDraft(canonicalBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !draft.Valid() || len(draft.NeighborDigests()) != 0 {
		t.Fatal("empty-neighbor completed sweep draft did not reconstruct exactly")
	}
	return draft
}

func newStoreForTest(t *testing.T) (*ReductionSweepStore, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "reduction-store")
	store, err := OpenReductionSweepStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return store, root
}

func TestReductionSweepStorePublishesEmptySweepAndReopensAfterRestart(t *testing.T) {
	store, root := newStoreForTest(t)
	draft := emptyCompletedSweepDraft(t, 1)
	authority, err := store.Publish(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Validate(context.Background(), draft, authority); err != nil {
		t.Fatalf("issued authority did not validate: %v", err)
	}

	for _, directory := range []string{root, filepath.Join(root, reductionSweepObjectsDirectory)} {
		info, err := os.Lstat(directory)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
			t.Fatalf("private directory facts for %s = %#v, %v", directory, info, err)
		}
	}
	objectPath := store.objectPath(draft)
	info, err := os.Lstat(objectPath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("published object facts = %#v, %v", info, err)
	}
	body, err := os.ReadFile(objectPath)
	if err != nil || !bytes.Equal(body, draft.CanonicalBytes()) {
		t.Fatalf("published object did not preserve exact canonical bytes: %v", err)
	}
	if !strings.Contains(filepath.Base(objectPath), strings.TrimPrefix(draft.Digest().String(), "sha256:")) {
		t.Fatal("object address does not bind the completed-sweep digest")
	}

	// Publishing the same address is idempotent only after an exact reopen.
	secondAuthority, err := store.Publish(context.Background(), draft)
	if err != nil || store.Validate(context.Background(), draft, secondAuthority) != nil {
		t.Fatalf("exact idempotent publication failed: %v", err)
	}

	restarted, err := OpenReductionSweepStore(root)
	if err != nil {
		t.Fatal(err)
	}
	restartedAuthority, err := restarted.Open(context.Background(), draft)
	if err != nil {
		t.Fatalf("restart reopen failed: %v", err)
	}
	if err := restarted.Validate(context.Background(), draft, restartedAuthority); err != nil {
		t.Fatalf("restart-issued authority did not validate: %v", err)
	}
	if err := restarted.Validate(context.Background(), draft, authority); err == nil {
		t.Fatal("authority from the pre-restart store instance validated after restart")
	}
	if err := store.Validate(context.Background(), draft, restartedAuthority); err == nil {
		t.Fatal("authority from the restarted store instance validated in the original store")
	}
}

func TestReductionSweepStoreConcurrentInstancesPublishWithoutReplacement(t *testing.T) {
	root := filepath.Join(t.TempDir(), "reduction-store")
	stores := make([]*ReductionSweepStore, 2)
	for index := range stores {
		var err error
		stores[index], err = OpenReductionSweepStore(root)
		if err != nil {
			t.Fatal(err)
		}
	}
	draft := emptyCompletedSweepDraft(t, 2)
	authorities := make([]SweepCompletionAuthority, len(stores))
	errorsByPublisher := make([]error, len(stores))
	start := make(chan struct{})
	var publishers sync.WaitGroup
	for index := range stores {
		publishers.Add(1)
		go func(index int) {
			defer publishers.Done()
			<-start
			authorities[index], errorsByPublisher[index] = stores[index].Publish(context.Background(), draft)
		}(index)
	}
	close(start)
	publishers.Wait()
	for index, err := range errorsByPublisher {
		if err != nil {
			t.Fatalf("publisher %d failed: %v", index, err)
		}
		if err := stores[index].Validate(context.Background(), draft, authorities[index]); err != nil {
			t.Fatalf("publisher %d authority failed exact reopen: %v", index, err)
		}
	}
	body, err := os.ReadFile(stores[0].objectPath(draft))
	if err != nil || !bytes.Equal(body, draft.CanonicalBytes()) {
		t.Fatalf("concurrent no-replace publication lost exact bytes: %v", err)
	}
	objectPath := stores[0].objectPath(draft)
	before, err := os.Lstat(objectPath)
	if err != nil {
		t.Fatal(err)
	}
	third, err := OpenReductionSweepStore(root)
	if err != nil {
		t.Fatal(err)
	}
	thirdAuthority, err := third.Publish(context.Background(), draft)
	if err != nil || third.Validate(context.Background(), draft, thirdAuthority) != nil {
		t.Fatalf("preexisting exact publication did not reopen: %v", err)
	}
	after, err := os.Lstat(objectPath)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("idempotent publication replaced the content-addressed inode: %v", err)
	}
}

func TestReductionSweepStoreRejectsRootsAndObjectsThatAreNotPrivateDirectories(t *testing.T) {
	t.Run("root-symlink", func(t *testing.T) {
		parent := t.TempDir()
		target := filepath.Join(parent, "target")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(parent, "root")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenReductionSweepStore(link); err == nil {
			t.Fatal("symlink root was accepted")
		}
	})

	t.Run("root-file", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "root")
		if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenReductionSweepStore(root); err == nil {
			t.Fatal("regular-file root was accepted")
		}
	})

	t.Run("broad-root-mode", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "root")
		if err := os.Mkdir(root, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenReductionSweepStore(root); err == nil {
			t.Fatal("group/world-accessible root was accepted")
		}
	})

	for _, variant := range []string{"symlink", "file"} {
		t.Run("objects-"+variant, func(t *testing.T) {
			parent := t.TempDir()
			root := filepath.Join(parent, "root")
			if err := os.Mkdir(root, 0o700); err != nil {
				t.Fatal(err)
			}
			objects := filepath.Join(root, reductionSweepObjectsDirectory)
			if variant == "symlink" {
				target := filepath.Join(parent, "target")
				if err := os.Mkdir(target, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, objects); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(objects, []byte("not a directory"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenReductionSweepStore(root); err == nil {
				t.Fatalf("%s objects path was accepted", variant)
			}
		})
	}
}

func TestReductionSweepStoreRefusesPartialCorruptSymlinkAndOversizedObjects(t *testing.T) {
	for _, test := range []struct {
		name    string
		prepare func(t *testing.T, path string, draft reduce.CompletedSweepDraft)
	}{
		{
			name: "partial",
			prepare: func(t *testing.T, path string, _ reduce.CompletedSweepDraft) {
				t.Helper()
				if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "corrupt",
			prepare: func(t *testing.T, path string, draft reduce.CompletedSweepDraft) {
				t.Helper()
				body := draft.CanonicalBytes()
				body[len(body)/2] ^= 1
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlink",
			prepare: func(t *testing.T, path string, draft reduce.CompletedSweepDraft) {
				t.Helper()
				target := filepath.Join(filepath.Dir(filepath.Dir(path)), "copied-sweep.json")
				if err := os.WriteFile(target, draft.CanonicalBytes(), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "oversized",
			prepare: func(t *testing.T, path string, _ reduce.CompletedSweepDraft) {
				t.Helper()
				if err := os.WriteFile(path, make([]byte, canon.MaxInputBytes+1), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _ := newStoreForTest(t)
			draft := emptyCompletedSweepDraft(t, 10)
			path := store.objectPath(draft)
			test.prepare(t, path, draft)
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.Open(context.Background(), draft); err == nil {
				t.Fatal("invalid object reopened into an authority")
			}
			if _, err := store.Publish(context.Background(), draft); err == nil {
				t.Fatal("publication overwrote or accepted an invalid object")
			}
			after, err := os.Lstat(path)
			if err != nil || before.Mode().Type() != after.Mode().Type() || before.Size() != after.Size() {
				t.Fatalf("invalid object was replaced: before=%#v after=%#v err=%v", before, after, err)
			}
		})
	}
}

func TestSweepCompletionAuthorityFailsClosedAcrossDraftStoreJSONAndCorruption(t *testing.T) {
	store, root := newStoreForTest(t)
	draft := emptyCompletedSweepDraft(t, 20)
	authority, err := store.Publish(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}

	wrongDraft := emptyCompletedSweepDraft(t, 21)
	if err := store.Validate(context.Background(), wrongDraft, authority); err == nil {
		t.Fatal("authority validated against a different exact draft")
	}
	var zeroAuthority SweepCompletionAuthority
	if err := store.Validate(context.Background(), draft, zeroAuthority); err == nil {
		t.Fatal("zero authority validated")
	}

	wire, err := json.Marshal(authority)
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != "{}" || bytes.Contains(wire, []byte(strings.TrimPrefix(draft.Digest().String(), "sha256:"))) {
		t.Fatalf("opaque authority leaked a wire constructor: %s", wire)
	}
	var copiedFromJSON SweepCompletionAuthority
	if err := json.Unmarshal(wire, &copiedFromJSON); err != nil {
		t.Fatal(err)
	}
	if err := store.Validate(context.Background(), draft, copiedFromJSON); err == nil {
		t.Fatal("copied JSON reconstructed completion authority")
	}

	restarted, err := OpenReductionSweepStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Validate(context.Background(), draft, authority); err == nil {
		t.Fatal("wrong store instance validated authority")
	}

	// Validate must reopen; issuance is not a cached Boolean.
	path := store.objectPath(draft)
	corrupt := draft.CanonicalBytes()
	corrupt[0] = '['
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Validate(context.Background(), draft, authority); err == nil {
		t.Fatal("post-issuance corruption survived validation")
	}
}

func TestReductionSweepStoreRefusesZeroCancelledAndReplacedState(t *testing.T) {
	store, _ := newStoreForTest(t)
	draft := emptyCompletedSweepDraft(t, 30)

	var zeroStore ReductionSweepStore
	if _, err := zeroStore.Publish(context.Background(), draft); err == nil {
		t.Fatal("zero store published a draft")
	}
	if _, err := store.Publish(context.Background(), reduce.CompletedSweepDraft{}); err == nil {
		t.Fatal("zero draft was published")
	}
	if _, err := store.Open(context.Background(), reduce.CompletedSweepDraft{}); err == nil {
		t.Fatal("zero draft was opened")
	}
	if _, err := store.Publish(nil, draft); err == nil {
		t.Fatal("nil context published a draft")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Publish(cancelled, draft); err == nil {
		t.Fatal("cancelled publication returned authority")
	}
	if _, err := os.Lstat(store.objectPath(draft)); !os.IsNotExist(err) {
		t.Fatalf("cancelled pre-publication exposed a final object: %v", err)
	}

	if err := os.Remove(store.objects); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(store.objects, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), draft); err == nil {
		t.Fatal("replaced objects directory retained store authority")
	}
}

func TestOrphanTemporaryFileNeverActsAsCompletionAuthority(t *testing.T) {
	store, _ := newStoreForTest(t)
	draft := emptyCompletedSweepDraft(t, 40)
	orphan := filepath.Join(store.objects, reductionSweepTempPrefix+"crash")
	if err := os.WriteFile(orphan, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(context.Background(), draft); err == nil {
		t.Fatal("orphan temporary bytes opened as a completed sweep")
	}
	authority, err := store.Publish(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Validate(context.Background(), draft, authority); err != nil {
		t.Fatalf("orphan temp blocked exact final publication: %v", err)
	}
	if _, err := os.Lstat(orphan); err != nil {
		t.Fatalf("store treated an unrelated orphan as the final object: %v", err)
	}
}
