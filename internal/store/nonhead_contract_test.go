package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	contractmodel "github.com/nelsonwerd/countershape/internal/contractexec/model"
	"github.com/nelsonwerd/countershape/internal/domain"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
)

type c2StorageFixture struct {
	store     *ObjectStore
	root      string
	bundle    emitmodel.ContractBundle
	target    contractmodel.ContractExecutionTarget
	targetRec targetStorageRecord
	claim     startClaimRecord
	winner    startClaimWinner
	manifest  privateManifestRecord
	run       contractmodel.FinalizedContractRun
	runRec    finalizedRunStorageRecord
	execution contractmodel.ContractExecution
	execRec   executionStorageRecord
}

func c2Digest(character byte) domain.Digest {
	return domain.MustDigest("sha256:" + string(bytes.Repeat([]byte{character}, 64)))
}

func c2Bundle(t testing.TB) emitmodel.ContractBundle {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	pretty, err := os.ReadFile(filepath.Join(root, "spec", "examples", "v1", "contract-bundle.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	exact, err := canon.Canonicalize(pretty)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := canon.DigestBytes("ContractBundle", exact)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := domain.ParseDigest(digest.String())
	bundle, err := emitmodel.ParseContractBundle(exact, parsed)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func c2Target(t testing.TB, bundle emitmodel.ContractBundle, attempt domain.Digest, nonceByte byte) contractmodel.ContractExecutionTarget {
	t.Helper()
	tree := contractmodel.TreeBinding{
		ObjectFormat: "sha1", CommitOID: string(bytes.Repeat([]byte{'a'}, 40)),
		TreeOID: string(bytes.Repeat([]byte{'b'}, 40)), PortableTreeDigest: c2Digest('c'),
		MaterializationPolicyDigest: c2Digest('d'), MaterializationManifestDigest: c2Digest('e'),
	}
	treeIdentity, _, err := canon.DigestTyped("PinnedTreeIdentity", map[string]any{
		"schema_version": domain.SchemaVersion, "kind": "PinnedTreeIdentity",
		"object_format": tree.ObjectFormat, "commit_oid": tree.CommitOID, "tree_oid": tree.TreeOID,
	})
	if err != nil {
		t.Fatal(err)
	}
	tree.TreeIdentityDigest, _ = domain.ParseDigest(treeIdentity.String())
	target, err := contractmodel.NewContractExecutionTarget(contractmodel.TargetInput{
		ContractBundleDigest: bundle.Digest(),
		TerminalResidue: contractmodel.TerminalResidueBinding{
			StudyID: "study:" + string(bytes.Repeat([]byte{'1'}, 64)), HeadRevision: 8,
			HeadDigest: c2Digest('f'), LineageRootDigest: c2Digest('2'),
		},
		Tree: tree,
		Attempt: contractmodel.AttemptBinding{
			ArtifactDigest: attempt, InstanceNonce: string(bytes.Repeat([]byte{nonceByte}, 64)),
		},
		BootSession: contractmodel.BootSessionBinding{IdentityDigest: c2Digest('5')},
		Runtime: contractmodel.RuntimeBinding{
			AdmittedExecutablePath: "/opt/homebrew/bin/node", MeasuredProcessExecPath: "/opt/homebrew/bin/node",
			Version: "25.2.1", Major: 25, Platform: "darwin", Architecture: "arm64",
			ExecutableBytesDigest: c2Digest('6'), ExecutableMode: "100755", ExecutableByteCount: 1024,
			ProbeProgramDigest: c2Digest('7'),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func issueC2AttemptFixture(store *ObjectStore, digest domain.Digest) attemptStorageRecord {
	return attemptStorageRecord{storeInstance: store.instance, digest: digest, seal: &attemptRecordSeal{marker: 1}}
}

func c3AttemptAndTarget(
	t testing.TB,
	store *ObjectStore,
) (ConformanceAttemptRecord, contractmodel.ContractExecutionTarget) {
	t.Helper()
	bundle := c2Bundle(t)
	template := c2Target(t, bundle, c2Digest('8'), '9')
	templateInput := template.Input()
	attempt, err := store.AllocateConformanceAttempt(context.Background(), ConformanceAttemptInput{
		ContractBundleDigest:        templateInput.ContractBundleDigest,
		ResidueHeadDigest:           templateInput.TerminalResidue.HeadDigest,
		TreeIdentityDigest:          templateInput.Tree.TreeIdentityDigest,
		MaterializationPolicyDigest: templateInput.Tree.MaterializationPolicyDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	templateInput.Attempt = contractmodel.AttemptBinding{
		ArtifactDigest: attempt.Digest(), InstanceNonce: attempt.InstanceNonce(),
	}
	target, err := contractmodel.NewContractExecutionTarget(templateInput)
	if err != nil {
		t.Fatal(err)
	}
	return attempt, target
}

type c3AttemptHandle struct {
	store   *ObjectStore
	attempt ConformanceAttemptRecord
}

func c3OpenAttemptHandles(t testing.TB, root string, attemptDigest domain.Digest, count int) []c3AttemptHandle {
	t.Helper()
	handles := make([]c3AttemptHandle, count)
	for index := range handles {
		objectStore, err := OpenObjectStore(root)
		if err != nil {
			t.Fatal(err)
		}
		attempt, err := objectStore.OpenConformanceAttempt(context.Background(), attemptDigest)
		if err != nil || !attempt.Valid() {
			t.Fatalf("open independent attempt handle %d: %v", index, err)
		}
		handles[index] = c3AttemptHandle{store: objectStore, attempt: attempt}
	}
	return handles
}

func TestC3ConformanceAttemptIsFreshDurableAndRestartReopenable(t *testing.T) {
	objectStore, root := newObjectStoreForTest(t)
	first, firstTarget := c3AttemptAndTarget(t, objectStore)
	second, _ := c3AttemptAndTarget(t, objectStore)
	if !first.Valid() || !second.Valid() || first.Digest() == second.Digest() ||
		first.InstanceNonce() == second.InstanceNonce() || first.Digest() != firstTarget.AttemptArtifactDigest() {
		t.Fatal("fresh attempts did not produce distinct exact authority")
	}
	roots := first.Roots()
	for name, path := range map[string]string{
		"attempt": roots.AttemptRoot(), "candidate": roots.CandidateParent(), "fixture": roots.FixtureRoot(),
		"home": roots.HomeRoot(), "temporary": roots.TemporaryRoot(), "xdg-config": roots.XDGConfigRoot(),
		"xdg-cache": roots.XDGCacheRoot(), "xdg-data": roots.XDGDataRoot(), "xdg-state": roots.XDGStateRoot(),
		"state": roots.StateRoot(), "evidence": roots.EvidenceRoot(),
	} {
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("%s root facts differ: %#v %v", name, info, err)
		}
	}
	marker, err := os.Lstat(roots.MarkerPath())
	if err != nil || !marker.Mode().IsRegular() || marker.Mode().Perm() != 0o600 || marker.Size() < 1 {
		t.Fatalf("attempt marker facts differ: %#v %v", marker, err)
	}
	markerBytes, err := os.ReadFile(roots.MarkerPath())
	if err != nil {
		t.Fatal(err)
	}
	markerValue, err := canon.Parse(markerBytes)
	if err != nil {
		t.Fatal(err)
	}
	purpose, present := markerValue.LookupMember("attempt_purpose")
	purposeText, isText := purpose.Text()
	if !present || !isText || purposeText != "CONFORMANCE" {
		t.Fatal("attempt marker does not freeze the exact CONFORMANCE purpose member")
	}
	if _, legacy := markerValue.LookupMember("purpose"); legacy {
		t.Fatal("attempt marker retained the ambiguous legacy purpose member")
	}
	restarted, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := restarted.OpenConformanceAttempt(context.Background(), first.Digest())
	if err != nil || !reopened.Valid() || reopened.Digest() != first.Digest() ||
		reopened.InstanceNonce() != first.InstanceNonce() || reopened.Roots().CandidateParent() != roots.CandidateParent() {
		t.Fatalf("attempt restart reopen failed: %v", err)
	}
	for _, hostile := range []struct {
		name         string
		evidenceRoot bool
	}{
		{name: "attempt-root-extra", evidenceRoot: false},
		{name: "evidence-root-extra", evidenceRoot: true},
	} {
		t.Run(hostile.name, func(t *testing.T) {
			hostileStore, hostileStoreRoot := newObjectStoreForTest(t)
			hostileAttempt, _ := c3AttemptAndTarget(t, hostileStore)
			foreignParent := hostileAttempt.Roots().AttemptRoot()
			if hostile.evidenceRoot {
				foreignParent = hostileAttempt.Roots().EvidenceRoot()
			}
			sentinel := filepath.Join(foreignParent, "foreign", "deep", "sentinel")
			if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(sentinel, []byte("retain"), 0o600); err != nil {
				t.Fatal(err)
			}
			if hostileAttempt.Valid() {
				t.Fatal("foreign direct roster entry retained live attempt authority")
			}
			restartedStore, err := OpenObjectStore(hostileStoreRoot)
			if err != nil {
				t.Fatal(err)
			}
			if opened, err := restartedStore.OpenConformanceAttempt(context.Background(), hostileAttempt.Digest()); err == nil || opened.Valid() {
				t.Fatalf("foreign direct roster entry reopened attempt authority: valid=%t err=%v", opened.Valid(), err)
			}
			if body, err := os.ReadFile(sentinel); err != nil || string(body) != "retain" {
				t.Fatalf("bounded roster validation traversed or removed foreign residue: body=%q err=%v", body, err)
			}
		})
	}
	restore := c2CaseAliasPath(t, roots.AttemptRoot())
	if first.Valid() || reopened.Valid() {
		t.Fatal("case-aliased attempt root retained live authority")
	}
	if aliased, err := restarted.OpenConformanceAttempt(context.Background(), first.Digest()); err == nil || aliased.Valid() {
		t.Fatalf("case-aliased attempt root reopened authority: %v", err)
	}
	restore()
}

func TestC3ConformanceAttemptRejectsCrossStoreAndRootReplacement(t *testing.T) {
	objectStore, _ := newObjectStoreForTest(t)
	attempt, target := c3AttemptAndTarget(t, objectStore)
	foreign, _ := newObjectStoreForTest(t)
	if record, err := foreign.PersistContractTargetRecord(context.Background(), attempt, target); err == nil || record.Valid() {
		t.Fatalf("cross-store attempt published a target: %v", err)
	}
	candidate := attempt.Roots().CandidateParent()
	if err := os.Remove(candidate); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(candidate, 0o700); err != nil {
		t.Fatal(err)
	}
	if attempt.Valid() {
		t.Fatal("replaced attempt child retained authority")
	}
	if record, err := objectStore.PersistContractTargetRecord(context.Background(), attempt, target); err == nil || record.Valid() {
		t.Fatalf("replaced attempt root published a target: %v", err)
	}
}

func TestC3ConformanceAttemptConcurrentValidationAndReopenAreRaceFree(t *testing.T) {
	objectStore, root := newObjectStoreForTest(t)
	attempt, target := c3AttemptAndTarget(t, objectStore)
	var wait sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := 0; iteration < 20; iteration++ {
				if !attempt.Valid() || !attempt.Digest().Valid() || !attempt.ContractBundleDigest().Valid() ||
					!attempt.ResidueHeadDigest().Valid() || !attempt.TreeIdentityDigest().Valid() ||
					!attempt.MaterializationPolicyDigest().Valid() || attempt.InstanceNonce() == "" ||
					attempt.Roots().AttemptRoot() == "" {
					t.Error("concurrent attempt validation lost exact authority")
					return
				}
				reopened, err := objectStore.OpenConformanceAttempt(context.Background(), attempt.Digest())
				if err != nil || !reopened.Valid() || reopened.Digest() != attempt.Digest() {
					t.Errorf("concurrent attempt reopen failed: %v", err)
					return
				}
				published, err := objectStore.PersistContractTargetRecord(context.Background(), reopened, target)
				if err != nil || !published.Valid() || published.Digest() != target.Digest() {
					t.Errorf("concurrent exact target convergence failed: %v", err)
					return
				}
			}
		}()
	}
	wait.Wait()

	t.Run("independent handles converge one exact target", func(t *testing.T) {
		independentAttempt, independentTarget := c3AttemptAndTarget(t, objectStore)
		handles := c3OpenAttemptHandles(t, root, independentAttempt.Digest(), 8)
		type result struct {
			record ContractTargetRecord
			err    error
		}
		results := make(chan result, len(handles))
		launch := make(chan struct{})
		for _, handle := range handles {
			go func(handle c3AttemptHandle) {
				<-launch
				record, err := handle.store.PersistContractTargetRecord(context.Background(), handle.attempt, independentTarget)
				results <- result{record: record, err: err}
			}(handle)
		}
		close(launch)
		for range handles {
			result := <-results
			if result.err != nil || !result.record.Valid() || result.record.Digest() != independentTarget.Digest() ||
				result.record.AttemptDigest() != independentAttempt.Digest() {
				t.Fatalf("independent exact convergence failed: record=%#v err=%v", result.record, result.err)
			}
		}
	})

	t.Run("independent handles preserve one conflicting winner", func(t *testing.T) {
		conflictAttempt, firstTarget := c3AttemptAndTarget(t, objectStore)
		alternateInput := firstTarget.Input()
		alternateInput.BootSession.IdentityDigest = c2Digest('a')
		secondTarget, err := contractmodel.NewContractExecutionTarget(alternateInput)
		if err != nil || secondTarget.Digest() == firstTarget.Digest() {
			t.Fatalf("conflicting target fixture failed: %v", err)
		}
		handles := c3OpenAttemptHandles(t, root, conflictAttempt.Digest(), 8)
		type result struct {
			requested domain.Digest
			record    ContractTargetRecord
			err       error
		}
		results := make(chan result, len(handles))
		launch := make(chan struct{})
		for index, handle := range handles {
			candidate := firstTarget
			if index >= len(handles)/2 {
				candidate = secondTarget
			}
			go func(handle c3AttemptHandle, candidate contractmodel.ContractExecutionTarget) {
				<-launch
				record, err := handle.store.PersistContractTargetRecord(context.Background(), handle.attempt, candidate)
				results <- result{requested: candidate.Digest(), record: record, err: err}
			}(handle, candidate)
		}
		close(launch)
		successes := map[domain.Digest]int{}
		failures := map[domain.Digest]int{}
		for range handles {
			result := <-results
			if result.err == nil && result.record.Valid() {
				if result.record.Digest() != result.requested {
					t.Fatalf("successful conflicting publication returned another digest: got %s want %s", result.record.Digest(), result.requested)
				}
				successes[result.requested]++
				continue
			}
			if result.err == nil || result.record.Valid() {
				t.Fatalf("conflicting publication returned incoherent refusal: %#v %v", result.record, result.err)
			}
			failures[result.requested]++
		}
		if len(successes) != 1 {
			t.Fatalf("conflicting publications produced %d winners: %v", len(successes), successes)
		}
		var winnerDigest domain.Digest
		for digest, count := range successes {
			winnerDigest = digest
			if count != 4 {
				t.Fatalf("winning candidate did not converge for every matching caller: %s count=%d", digest, count)
			}
		}
		loserDigest := firstTarget.Digest()
		winnerTarget := firstTarget
		if winnerDigest == firstTarget.Digest() {
			loserDigest = secondTarget.Digest()
		} else if winnerDigest == secondTarget.Digest() {
			winnerTarget = secondTarget
		} else {
			t.Fatalf("winner digest was not one of the exact candidates: %s", winnerDigest)
		}
		if failures[loserDigest] != 4 || failures[winnerDigest] != 0 {
			t.Fatalf("conflicting result partition differs: successes=%v failures=%v", successes, failures)
		}
		freshHandle := c3OpenAttemptHandles(t, root, conflictAttempt.Digest(), 1)[0]
		reopened, err := freshHandle.store.OpenContractTargetRecord(context.Background(), freshHandle.attempt)
		if err != nil || !reopened.Valid() || reopened.Digest() != winnerDigest ||
			!bytes.Equal(reopened.record.record.object.CanonicalBytes(), winnerTarget.CanonicalBytes()) {
			t.Fatalf("fresh handle did not reopen the exact conflict winner: %v", err)
		}
		expectedRelation, expectedPath, err := contractRelationMaterial(freshHandle.store, reopened.record.record.relation)
		if err != nil || expectedPath != reopened.record.record.relationPath ||
			!bytes.Equal(expectedRelation, reopened.record.record.relationBytes) {
			t.Fatalf("conflict winner relation material differs: %v", err)
		}
		physical, err := readExactPrivateFile(expectedPath, int64(len(expectedRelation)))
		if err != nil || !bytes.Equal(physical, expectedRelation) {
			t.Fatalf("conflict winner relation did not reopen as exact durable bytes: %v", err)
		}
	})
}

func TestC3ContractTargetBridgeConvergesExactAndRejectsReuse(t *testing.T) {
	objectStore, root := newObjectStoreForTest(t)
	attempt, target := c3AttemptAndTarget(t, objectStore)
	record, err := objectStore.PersistContractTargetRecord(context.Background(), attempt, target)
	if err != nil || !record.Valid() || record.Digest() != target.Digest() || record.AttemptDigest() != attempt.Digest() {
		t.Fatalf("exact target bridge failed: %v", err)
	}
	reopened, err := objectStore.OpenContractTargetRecord(context.Background(), attempt)
	if err != nil || !reopened.Valid() || reopened.Digest() != record.Digest() {
		t.Fatalf("exact target relation did not reopen: %v", err)
	}
	if converged, err := objectStore.PersistContractTargetRecord(context.Background(), attempt, target); err != nil || !converged.Valid() || converged.Digest() != record.Digest() {
		t.Fatalf("exact retry did not reconcile: %v", err)
	}
	relationPath := record.record.record.relationPath
	relationBytes := append([]byte(nil), record.record.record.relationBytes...)
	relationDigest, err := canon.DigestBytes("ContractStorageRelation", relationBytes)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*contractmodel.TargetInput){
		"contract bundle": func(input *contractmodel.TargetInput) { input.ContractBundleDigest = c2Digest('0') },
		"residue study": func(input *contractmodel.TargetInput) {
			input.TerminalResidue.StudyID = "study:" + strings.Repeat("3", 64)
		},
		"residue revision":         func(input *contractmodel.TargetInput) { input.TerminalResidue.HeadRevision++ },
		"residue head":             func(input *contractmodel.TargetInput) { input.TerminalResidue.HeadDigest = c2Digest('0') },
		"residue lineage":          func(input *contractmodel.TargetInput) { input.TerminalResidue.LineageRootDigest = c2Digest('3') },
		"portable tree":            func(input *contractmodel.TargetInput) { input.Tree.PortableTreeDigest = c2Digest('0') },
		"materialization policy":   func(input *contractmodel.TargetInput) { input.Tree.MaterializationPolicyDigest = c2Digest('0') },
		"materialization manifest": func(input *contractmodel.TargetInput) { input.Tree.MaterializationManifestDigest = c2Digest('0') },
		"attempt artifact":         func(input *contractmodel.TargetInput) { input.Attempt.ArtifactDigest = c2Digest('0') },
		"attempt nonce":            func(input *contractmodel.TargetInput) { input.Attempt.InstanceNonce = strings.Repeat("a", 64) },
		"boot session":             func(input *contractmodel.TargetInput) { input.BootSession.IdentityDigest = c2Digest('a') },
		"runtime version":          func(input *contractmodel.TargetInput) { input.Runtime.Version = "25.2.2" },
		"runtime bytes":            func(input *contractmodel.TargetInput) { input.Runtime.ExecutableBytesDigest = c2Digest('a') },
		"runtime mode":             func(input *contractmodel.TargetInput) { input.Runtime.ExecutableMode = "100555" },
		"runtime byte count":       func(input *contractmodel.TargetInput) { input.Runtime.ExecutableByteCount++ },
		"runtime probe":            func(input *contractmodel.TargetInput) { input.Runtime.ProbeProgramDigest = c2Digest('a') },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			alternateInput := target.Input()
			mutate(&alternateInput)
			alternate, err := contractmodel.NewContractExecutionTarget(alternateInput)
			if err != nil || alternate.Digest() == target.Digest() {
				t.Fatalf("valid alternate target fixture failed: %v", err)
			}
			if conflicting, err := objectStore.PersistContractTargetRecord(context.Background(), attempt, alternate); err == nil || conflicting.Valid() {
				t.Fatalf("attempt reuse published alternate target: %v", err)
			}
			if !record.Valid() || record.Digest() != target.Digest() {
				t.Fatal("refused alternate invalidated the exact target record")
			}
			physical, err := readExactPrivateFile(relationPath, int64(len(relationBytes)))
			if err != nil || !bytes.Equal(physical, relationBytes) {
				t.Fatalf("refused alternate changed relation bytes: %v", err)
			}
			physicalDigest, err := canon.DigestBytes("ContractStorageRelation", physical)
			if err != nil || physicalDigest != relationDigest {
				t.Fatalf("refused alternate changed relation digest: got %s want %s err=%v", physicalDigest, relationDigest, err)
			}
			reopened, err := objectStore.OpenContractTargetRecord(context.Background(), attempt)
			if err != nil || !reopened.Valid() || reopened.Digest() != target.Digest() ||
				!bytes.Equal(reopened.record.record.object.CanonicalBytes(), target.CanonicalBytes()) {
				t.Fatalf("refused alternate displaced exact target authority: %v", err)
			}
		})
	}
	restarted, err := OpenObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	restartedAttempt, err := restarted.OpenConformanceAttempt(context.Background(), attempt.Digest())
	if err != nil {
		t.Fatal(err)
	}
	restartedTarget, err := restarted.OpenContractTargetRecord(context.Background(), restartedAttempt)
	if err != nil || !restartedTarget.Valid() || restartedTarget.Digest() != target.Digest() {
		t.Fatalf("restart target relation failed: %v", err)
	}
}

func TestC3ConformanceAttemptMarkerMutationRefusesReopen(t *testing.T) {
	tests := map[string]func(*testing.T, string){
		"bytes": func(t *testing.T, marker string) {
			if err := os.WriteFile(marker, []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"mode": func(t *testing.T, marker string) {
			if err := os.Chmod(marker, 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"symlink": func(t *testing.T, marker string) {
			if err := os.Remove(marker); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("missing", marker); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			objectStore, root := newObjectStoreForTest(t)
			attempt, _ := c3AttemptAndTarget(t, objectStore)
			mutate(t, attempt.Roots().MarkerPath())
			if attempt.Valid() {
				t.Fatal("mutated marker retained live authority")
			}
			restarted, err := OpenObjectStore(root)
			if err != nil {
				t.Fatal(err)
			}
			if reopened, err := restarted.OpenConformanceAttempt(context.Background(), attempt.Digest()); err == nil || reopened.Valid() {
				t.Fatalf("mutated marker reopened authority: %v", err)
			}
		})
	}
}

func c2SemanticObject(t testing.TB, kind string, digest domain.Digest, body []byte) SemanticObject {
	t.Helper()
	object, err := NewSemanticObject(kind, digest, body)
	if err != nil {
		t.Fatal(err)
	}
	return object
}

func c2SubstituteDirectory(t *testing.T, path string) {
	t.Helper()
	retained := path + ".retained"
	if err := os.Rename(path, retained); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(retained, path); err != nil {
		t.Fatal(err)
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
}

func c2CaseAliasPath(t *testing.T, path string) func() {
	t.Helper()
	directory, name := filepath.Dir(path), filepath.Base(path)
	aliasBytes := []byte(name)
	changed := false
	for index, character := range aliasBytes {
		switch {
		case character >= 'a' && character <= 'z':
			aliasBytes[index] = character - ('a' - 'A')
			changed = true
		case character >= 'A' && character <= 'Z':
			aliasBytes[index] = character + ('a' - 'A')
			changed = true
		}
		if changed {
			break
		}
	}
	if !changed {
		t.Fatalf("case-alias fixture %q has no ASCII letter", name)
	}
	alias := filepath.Join(directory, string(aliasBytes))
	hop := filepath.Join(directory, name+".case-hop")
	before, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, hop); err != nil {
		t.Fatal(err)
	}
	location := hop
	restore := func() {
		switch location {
		case path:
			return
		case alias:
			if err := os.Rename(alias, hop); err != nil {
				t.Error(err)
				return
			}
			location = hop
		}
		if err := os.Rename(hop, path); err != nil {
			t.Error(err)
			return
		}
		location = path
		if err := syncDirectory(directory); err != nil {
			t.Error(err)
			return
		}
	}
	t.Cleanup(restore)
	if err := os.Rename(hop, alias); err != nil {
		t.Fatal(err)
	}
	location = alias
	if err := syncDirectory(directory); err != nil {
		t.Fatal(err)
	}
	after, err := os.Lstat(alias)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("case-alias rename did not preserve identity: %v", err)
	}
	return restore
}

func c2ReplaceDirectoryWithExactClone(t *testing.T, path string) func() {
	t.Helper()
	retained := path + ".retained-real"
	if err := os.Rename(path, retained); err != nil {
		t.Fatal(err)
	}
	restored := false
	restore := func() {
		if restored {
			return
		}
		if err := os.RemoveAll(path); err != nil {
			t.Error(err)
			return
		}
		if err := os.Rename(retained, path); err != nil {
			t.Error(err)
			return
		}
		restored = true
		if err := syncDirectory(filepath.Dir(path)); err != nil {
			t.Error(err)
			return
		}
	}
	t.Cleanup(restore)
	var clone func(string, string)
	clone = func(source, destination string) {
		if err := os.Mkdir(destination, 0o700); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			from := filepath.Join(source, entry.Name())
			to := filepath.Join(destination, entry.Name())
			info, err := entry.Info()
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case info.Mode()&os.ModeSymlink != 0:
				t.Fatalf("exact clone source %q contains a symlink", from)
			case info.IsDir() && info.Mode().Perm() == 0o700:
				clone(from, to)
			case info.Mode().IsRegular() && info.Mode().Perm() == 0o600:
				if err := os.Link(from, to); err != nil {
					t.Fatal(err)
				}
			default:
				t.Fatalf("exact clone source %q has unsupported mode %s", from, info.Mode())
			}
		}
		if err := syncDirectory(destination); err != nil {
			t.Fatal(err)
		}
	}
	clone(retained, path)
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
	return restore
}

func c2EvidenceReference(t testing.TB, kind string, digest domain.Digest) contractmodel.EvidenceRef {
	t.Helper()
	var (
		ref contractmodel.EvidenceRef
		err error
	)
	switch kind {
	case "MATERIALIZATION_REVALIDATION":
		ref, err = contractmodel.NewMaterializationRevalidationRef(digest)
	case "RUNTIME_REVALIDATION":
		ref, err = contractmodel.NewRuntimeRevalidationRef(digest)
	case "PROCESS_RESULT":
		ref, err = contractmodel.NewProcessResultRef(digest)
	case "WAIT_RESULT":
		ref, err = contractmodel.NewWaitResultRef(digest)
	case "DRAIN_RESULT":
		ref, err = contractmodel.NewDrainResultRef(digest)
	case "TEARDOWN_RESULT":
		ref, err = contractmodel.NewTeardownResultRef(digest)
	case "ORPHAN_CHECK":
		ref, err = contractmodel.NewOrphanCheckRef(digest)
	case "FINALIZATION_MARKER":
		ref, err = contractmodel.NewFinalizationMarkerRef(digest)
	case "CAPTURED_OBSERVATION":
		ref, err = contractmodel.NewCapturedObservationRef(digest)
	case "PROJECTION_RESULT":
		ref, err = contractmodel.NewProjectionResultRef(digest)
	case "TARGET_INVENTORY":
		ref, err = contractmodel.NewTargetInventoryRef(digest)
	case "CHILD_BINDINGS":
		ref, err = contractmodel.NewChildBindingsRef(digest)
	case "IMPORT_RESOLUTION":
		ref, err = contractmodel.NewImportResolutionRef(digest)
	case "SERVICE_BINDINGS":
		ref, err = contractmodel.NewServiceBindingsRef(digest)
	case "NAMED_PARENT_SECRET_SENTINEL_INHERITANCE":
		ref, err = contractmodel.NewSentinelInheritanceRef(digest)
	default:
		t.Fatalf("unknown evidence kind %q", kind)
	}
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func c2CompleteFixture(t testing.TB) c2StorageFixture {
	t.Helper()
	store, root := newObjectStoreForTest(t.(*testing.T))
	bundle := c2Bundle(t)
	target := c2Target(t, bundle, c2Digest('3'), '4')
	attempt := issueC2AttemptFixture(store, target.AttemptArtifactDigest())
	targetObject := c2SemanticObject(t, contractmodel.TargetKind, target.Digest(), target.CanonicalBytes())
	targetRecord, _, err := persistTargetRecord(context.Background(), store, targetStorageInput{attempt: attempt, object: targetObject}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, claim, winner, err := acquireInterlockAndStartClaim(
		context.Background(), store, targetRecord, target.Input().BootSession.IdentityDigest, nil,
	)
	if err != nil || !winner.validFor(store) {
		t.Fatalf("start winner invalid: %v", err)
	}

	modelRefs := make(map[string]contractmodel.EvidenceRef, len(privateEvidenceKindOrder))
	blobs := make([]privateBlobInput, len(privateEvidenceKindOrder))
	for index, kind := range privateEvidenceKindOrder {
		digest := c2Digest("89abcdefabcdefa"[index])
		modelRefs[kind] = c2EvidenceReference(t, kind, digest)
		blobs[index] = privateBlobInput{
			body:       []byte("private-evidence-" + kind),
			references: []privateEvidenceReference{{kind: kind, digest: digest}},
		}
	}
	manifest, _, err := createPrivateManifest(context.Background(), store, targetRecord, claim, blobs, nil)
	if err != nil {
		t.Fatal(err)
	}
	processKinds := []string{
		"MATERIALIZATION_REVALIDATION", "RUNTIME_REVALIDATION", "PROCESS_RESULT", "WAIT_RESULT",
		"DRAIN_RESULT", "TEARDOWN_RESULT", "ORPHAN_CHECK", "FINALIZATION_MARKER",
	}
	processRefs := make([]contractmodel.EvidenceRef, len(processKinds))
	for index, kind := range processKinds {
		processRefs[index] = modelRefs[kind]
	}
	process, err := contractmodel.NewCleanProcessClosure(processRefs)
	if err != nil {
		t.Fatal(err)
	}
	spawn, _ := contractmodel.NewChildPIDObservation(4242)
	allowed := bundle.Predicate().AllowedTuples()
	observation, err := contractmodel.NewProjectedObservation(
		modelRefs["CAPTURED_OBSERVATION"], modelRefs["PROJECTION_RESULT"], allowed[0],
	)
	if err != nil {
		t.Fatal(err)
	}
	scopeConstructors := []struct {
		domain contractmodel.ScopeDomain
		kind   string
	}{
		{contractmodel.ScopeTargetInventory, "TARGET_INVENTORY"},
		{contractmodel.ScopeChildBindings, "CHILD_BINDINGS"},
		{contractmodel.ScopeImportResolution, "IMPORT_RESOLUTION"},
		{contractmodel.ScopeServiceBindings, "SERVICE_BINDINGS"},
		{contractmodel.ScopeSentinelInheritance, "NAMED_PARENT_SECRET_SENTINEL_INHERITANCE"},
	}
	checks := make([]contractmodel.ScopeCheck, len(scopeConstructors))
	for index, item := range scopeConstructors {
		checks[index], err = contractmodel.NewCleanScopeCheck(item.domain, modelRefs[item.kind])
		if err != nil {
			t.Fatal(err)
		}
	}
	scope, err := contractmodel.NewStandaloneScope(checks)
	if err != nil {
		t.Fatal(err)
	}
	manifestRef, err := contractmodel.NewPrivateEvidenceManifestRef(manifest.digest)
	if err != nil {
		t.Fatal(err)
	}
	manifestSummary, err := contractmodel.NewPrivateManifestSummary(manifestRef, manifest.blobCount, manifest.aggregateBytes)
	if err != nil {
		t.Fatal(err)
	}
	witness, err := contractmodel.NewClosedRunWitness(spawn, process, observation, scope, manifestSummary)
	if err != nil {
		t.Fatal(err)
	}
	run, err := contractmodel.NewFinalizedContractRun(target, claim.digest, witness)
	if err != nil {
		t.Fatal(err)
	}
	runObject := c2SemanticObject(t, contractmodel.RunKind, run.Digest(), run.CanonicalBytes())
	runRecord, _, err := persistFinalizedRunRecord(context.Background(), store, runStorageInput{
		target: targetRecord, claim: claim, object: runObject, manifest: manifest,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	execution, err := contractmodel.DeriveContractExecution(bundle, target, run)
	if err != nil {
		t.Fatal(err)
	}
	executionObject := c2SemanticObject(t, contractmodel.ExecutionKind, execution.Digest(), execution.CanonicalBytes())
	executionRecord, _, err := persistExecutionRecord(
		context.Background(), store, executionStorageInput{run: runRecord, object: executionObject}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return c2StorageFixture{
		store: store, root: root, bundle: bundle, target: target, targetRec: targetRecord,
		claim: claim, winner: winner, manifest: manifest, run: run, runRec: runRecord,
		execution: execution, execRec: executionRecord,
	}
}

func TestC2FixturePublishesAndReopensExactNonheadObjects(t *testing.T) {
	fixture := c2CompleteFixture(t)
	restarted, err := OpenObjectStore(fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	attempt := issueC2AttemptFixture(restarted, fixture.target.AttemptArtifactDigest())
	target, err := openTargetByAttempt(context.Background(), restarted, attempt)
	if err != nil {
		t.Fatal(err)
	}
	run, err := openFinalizedRunByTarget(context.Background(), restarted, target)
	if err != nil {
		t.Fatal(err)
	}
	execution, err := openExecutionByRunProfile(context.Background(), restarted, run)
	if err != nil {
		t.Fatal(err)
	}
	if target.record.object.Digest() != fixture.target.Digest() || run.record.object.Digest() != fixture.run.Digest() ||
		execution.record.object.Digest() != fixture.execution.Digest() {
		t.Fatal("restart reopen changed exact object identity")
	}
}

func TestC2ExactKeyMappingsConvergeOnlyForTypedParents(t *testing.T) {
	store, _ := newObjectStoreForTest(t)
	bundle := c2Bundle(t)
	target := c2Target(t, bundle, c2Digest('3'), '4')
	attempt := issueC2AttemptFixture(store, target.AttemptArtifactDigest())
	object := c2SemanticObject(t, contractmodel.TargetKind, target.Digest(), target.CanonicalBytes())
	first, _, err := persistTargetRecord(context.Background(), store, targetStorageInput{attempt: attempt, object: object}, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, effect, err := persistTargetRecord(context.Background(), store, targetStorageInput{attempt: attempt, object: object}, nil)
	if err != nil || effect != contractExactConverged || !second.validFor(store) || first.record.relationPath != second.record.relationPath {
		t.Fatalf("exact convergence failed: effect=%s err=%v", effect, err)
	}
	alternateInput := target.Input()
	alternateInput.Runtime.Version = "25.2.2"
	alternate, err := contractmodel.NewContractExecutionTarget(alternateInput)
	if err != nil {
		t.Fatal(err)
	}
	alternateObject := c2SemanticObject(t, contractmodel.TargetKind, alternate.Digest(), alternate.CanonicalBytes())
	if _, _, err := persistTargetRecord(context.Background(), store, targetStorageInput{attempt: attempt, object: alternateObject}, nil); err == nil {
		t.Fatal("alternate child occupied an exact attempt relationship key")
	}
	complete := c2CompleteFixture(t)
	executionObject := c2SemanticObject(
		t, contractmodel.ExecutionKind, complete.execution.Digest(), complete.execution.CanonicalBytes(),
	)
	retried, effect, err := persistExecutionRecord(
		context.Background(), complete.store,
		executionStorageInput{run: complete.runRec, object: executionObject}, nil,
	)
	relation := retried.record.relation
	if err != nil || effect != contractExactConverged || !retried.validFor(complete.store) ||
		retried.record.relationPath != complete.execRec.record.relationPath ||
		relation.relation != relationRunExecution || relation.parentKind != contractRunKind ||
		relation.parentDigest != complete.runRec.record.object.Digest() ||
		relation.secondaryParentKind != contractProfileKind ||
		relation.secondaryParentDigest != contractClassifierProfileDigest() ||
		relation.childKind != contractExecutionKind || relation.childDigest != complete.execution.Digest() {
		t.Fatalf("exact run/profile execution convergence failed: effect=%s relation=%+v err=%v", effect, relation, err)
	}
}

func TestC2MappingsRejectWrongKindCrossStoreAndAlternateLinks(t *testing.T) {
	fixture := c2CompleteFixture(t)
	foreign, _ := newObjectStoreForTest(t)
	if fixture.targetRec.validFor(foreign) || fixture.runRec.validFor(foreign) || fixture.execRec.validFor(foreign) {
		t.Fatal("live storage record crossed store instances")
	}
	if _, err := openFinalizedRunByTarget(context.Background(), foreign, fixture.targetRec); err == nil {
		t.Fatal("foreign target opened a run relationship")
	}
	for _, test := range []struct {
		name     string
		relation contractRelation
		object   SemanticObject
	}{
		func() struct {
			name     string
			relation contractRelation
			object   SemanticObject
		} {
			relation := fixture.targetRec.record.relation
			relation.relation = relationTargetRun
			return struct {
				name     string
				relation contractRelation
				object   SemanticObject
			}{"wrong-relation", relation, fixture.targetRec.record.object}
		}(),
		func() struct {
			name     string
			relation contractRelation
			object   SemanticObject
		} {
			relation := fixture.targetRec.record.relation
			relation.parentKind = contractTargetKind
			return struct {
				name     string
				relation contractRelation
				object   SemanticObject
			}{"wrong-parent-kind", relation, fixture.targetRec.record.object}
		}(),
		func() struct {
			name     string
			relation contractRelation
			object   SemanticObject
		} {
			relation := fixture.targetRec.record.relation
			relation.parentDigest = c2Digest('a')
			return struct {
				name     string
				relation contractRelation
				object   SemanticObject
			}{"wrong-parent-digest", relation, fixture.targetRec.record.object}
		}(),
		func() struct {
			name     string
			relation contractRelation
			object   SemanticObject
		} {
			relation := fixture.targetRec.record.relation
			relation.childKind = contractRunKind
			return struct {
				name     string
				relation contractRelation
				object   SemanticObject
			}{"wrong-child-kind", relation, fixture.targetRec.record.object}
		}(),
		func() struct {
			name     string
			relation contractRelation
			object   SemanticObject
		} {
			relation := fixture.targetRec.record.relation
			relation.targetDigest = c2Digest('b')
			return struct {
				name     string
				relation contractRelation
				object   SemanticObject
			}{"wrong-target-join", relation, fixture.targetRec.record.object}
		}(),
		func() struct {
			name     string
			relation contractRelation
			object   SemanticObject
		} {
			relation := fixture.runRec.record.relation
			relation.startClaimDigest = c2Digest('c')
			return struct {
				name     string
				relation contractRelation
				object   SemanticObject
			}{"wrong-start-claim-join", relation, fixture.runRec.record.object}
		}(),
		func() struct {
			name     string
			relation contractRelation
			object   SemanticObject
		} {
			relation := fixture.execRec.record.relation
			relation.classifierProfileDigest = c2Digest('d')
			return struct {
				name     string
				relation contractRelation
				object   SemanticObject
			}{"wrong-classifier-profile", relation, fixture.execRec.record.object}
		}(),
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateContractRelation(test.relation, test.object); err == nil {
				t.Fatal("invalid typed relationship joined its child")
			}
		})
	}

	t.Run("alternate-and-raw-links-are-not-discovery", func(t *testing.T) {
		isolated := c2CompleteFixture(t)
		isolated.store.instance.mu.Lock()
		alternateDirectory, err := isolated.store.relationDirectoryLocked(relationTargetRun)
		isolated.store.instance.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		alternatePath := filepath.Join(alternateDirectory, filepath.Base(isolated.targetRec.record.relationPath))
		if err := os.WriteFile(alternatePath, isolated.targetRec.record.relationBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		hex, _ := strictDigestHex(isolated.targetRec.record.relation.parentDigest)
		rawPath := filepath.Join(isolated.store.contractLinks, hex)
		if err := os.WriteFile(rawPath, isolated.targetRec.record.relationBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		attempt := issueC2AttemptFixture(isolated.store, isolated.target.AttemptArtifactDigest())
		if _, err := openTargetByAttempt(context.Background(), isolated.store, attempt); err != nil {
			t.Fatalf("alternate links displaced the exact typed mapping: %v", err)
		}
		if err := os.Remove(isolated.targetRec.record.relationPath); err != nil {
			t.Fatal(err)
		}
		if err := syncDirectory(filepath.Dir(isolated.targetRec.record.relationPath)); err != nil {
			t.Fatal(err)
		}
		if _, err := openTargetByAttempt(context.Background(), isolated.store, attempt); err == nil {
			t.Fatal("alternate typed or raw-key link substituted for the exact mapping")
		}
	})

	body, err := os.ReadFile(fixture.targetRec.record.relationPath)
	if err != nil {
		t.Fatal(err)
	}
	body[len(body)/2] ^= 1
	if err := os.WriteFile(fixture.targetRec.record.relationPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	attempt := issueC2AttemptFixture(fixture.store, fixture.target.AttemptArtifactDigest())
	if _, err := openTargetByAttempt(context.Background(), fixture.store, attempt); err == nil {
		t.Fatal("corrupt relationship reopened")
	}

	t.Run("typed-relation-parent-substitution-refuses-restart", func(t *testing.T) {
		isolated := c2CompleteFixture(t)
		c2SubstituteDirectory(t, filepath.Dir(isolated.targetRec.record.relationPath))
		if isolated.targetRec.validFor(isolated.store) {
			t.Fatal("same-instance record survived typed relation directory substitution")
		}
		restarted, err := OpenObjectStore(isolated.root)
		if err != nil {
			t.Fatal(err)
		}
		attempt := issueC2AttemptFixture(restarted, isolated.target.AttemptArtifactDigest())
		if _, err := openTargetByAttempt(context.Background(), restarted, attempt); err == nil {
			t.Fatal("restart promoted a relation through a substituted typed parent")
		}
	})

	t.Run("typed-publication-parent-substitution-invalidates-record", func(t *testing.T) {
		isolated := c2CompleteFixture(t)
		c2SubstituteDirectory(t, filepath.Dir(isolated.targetRec.record.witnessPath))
		if isolated.targetRec.validFor(isolated.store) {
			t.Fatal("record survived typed publication directory substitution")
		}
	})

	for _, test := range []struct {
		name string
		path func(c2StorageFixture) string
	}{
		{"typed-relation-real-parent-replacement", func(fixture c2StorageFixture) string {
			return filepath.Dir(fixture.targetRec.record.relationPath)
		}},
		{"typed-publication-real-parent-replacement", func(fixture c2StorageFixture) string {
			return filepath.Dir(fixture.targetRec.record.witnessPath)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			isolated := c2CompleteFixture(t)
			restore := c2ReplaceDirectoryWithExactClone(t, test.path(isolated))
			if isolated.targetRec.validFor(isolated.store) {
				t.Fatal("record survived a real same-mode parent with exact hard-linked contents")
			}
			restore()
			if !isolated.targetRec.validFor(isolated.store) {
				t.Fatal("record did not recover when its retained parent identity was restored")
			}
		})
	}

	for _, test := range []struct {
		name string
		path func(c2StorageFixture) string
	}{
		{"typed-relation-directory-case-alias", func(fixture c2StorageFixture) string {
			return filepath.Dir(fixture.targetRec.record.relationPath)
		}},
		{"typed-relation-leaf-case-alias", func(fixture c2StorageFixture) string {
			return fixture.targetRec.record.relationPath
		}},
		{"typed-publication-leaf-case-alias", func(fixture c2StorageFixture) string {
			return fixture.targetRec.record.witnessPath
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			isolated := c2CompleteFixture(t)
			restore := c2CaseAliasPath(t, test.path(isolated))
			if isolated.targetRec.validFor(isolated.store) {
				t.Fatal("record accepted a case-only path alias")
			}
			restore()
			if !isolated.targetRec.validFor(isolated.store) {
				t.Fatal("record did not recover after canonical spelling was restored")
			}
		})
	}
}

func TestC2CopiedOrParsedBodiesCannotEnterProductionMechanics(t *testing.T) {
	store, _ := newObjectStoreForTest(t)
	bundle := c2Bundle(t)
	target := c2Target(t, bundle, c2Digest('3'), '4')
	object := c2SemanticObject(t, contractmodel.TargetKind, target.Digest(), target.CanonicalBytes())
	if _, _, err := persistTargetRecord(context.Background(), store, targetStorageInput{object: object}, nil); err == nil {
		t.Fatal("copied model body entered mechanics without the test-only attempt fixture")
	}
	if _, _, err := persistExecutionRecord(context.Background(), store, executionStorageInput{object: object}, nil); err == nil {
		t.Fatal("generic object entered execution persistence without a typed run parent")
	}
}

func TestC2NonheadFaultMatrixSeparatesNoEffectAndUnknownEffect(t *testing.T) {
	for _, test := range []struct {
		phase           contractFaultPhase
		witnessPresent  bool
		relationPresent bool
	}{
		{faultBeforeLink, false, false},
		{faultAfterLink, true, false},
		{faultAfterParentSync, true, false},
		{faultBeforeTemporary, true, false},
		{faultAfterTempSync, true, false},
		{faultBeforeReopen, true, true},
	} {
		t.Run(string(test.phase), func(t *testing.T) {
			store, _ := newObjectStoreForTest(t)
			bundle := c2Bundle(t)
			target := c2Target(t, bundle, c2Digest('3'), '4')
			attempt := issueC2AttemptFixture(store, target.AttemptArtifactDigest())
			object := c2SemanticObject(t, contractmodel.TargetKind, target.Digest(), target.CanonicalBytes())
			relation := contractRelation{
				relation: relationAttemptTarget, parentKind: contractAttemptKind, parentDigest: attempt.digest,
				childKind: contractTargetKind, childDigest: object.Digest(), targetDigest: object.Digest(),
				attemptDigest: attempt.digest,
			}
			_, relationPath, err := contractRelationMaterial(store, relation)
			if err != nil {
				t.Fatal(err)
			}
			hex, _ := strictDigestHex(object.Digest())
			witnessPath := filepath.Join(store.contractRoot, contractPublicationDirectory, object.Kind(), hex)
			fault := func(actual contractFaultPhase) error {
				if actual == test.phase {
					return errors.New("injected fault")
				}
				return nil
			}
			record, effect, err := persistTargetRecord(
				context.Background(), store, targetStorageInput{attempt: attempt, object: object}, fault,
			)
			if err == nil || record.validFor(store) || effect != contractAmbiguous || mutationEffect(err) != contractAmbiguous {
				t.Fatal("faulted publication returned authority")
			}
			if _, _, err := store.Read(context.Background(), object.Kind(), object.Digest()); err != nil {
				t.Fatal("faulted typed publication did not retain its exact generic object")
			}
			for path, want := range map[string]bool{witnessPath: test.witnessPresent, relationPath: test.relationPresent} {
				_, statErr := os.Lstat(path)
				if (statErr == nil) != want || (statErr != nil && !errors.Is(statErr, os.ErrNotExist)) {
					t.Fatalf("fault phase %s path %s presence=%t err=%v", test.phase, path, want, statErr)
				}
			}
			record, retryEffect, err := persistTargetRecord(
				context.Background(), store, targetStorageInput{attempt: attempt, object: object}, nil,
			)
			if err != nil || !record.validFor(store) {
				t.Fatalf("exact retry did not converge: %v", err)
			}
			if retryEffect != contractExactConverged {
				t.Fatalf("exact retry effect=%s", retryEffect)
			}
		})
	}
	joined := errors.Join(
		mutationFailure(codeContractRecordRefused, contractKnownNoEffect, faultBeforeLink, errors.New("first")),
		mutationFailure(codeContractRecordAmbiguous, contractAmbiguous, faultBeforeReopen, errors.New("second")),
	)
	if mutationEffect(joined) != contractAmbiguous {
		t.Fatal("joined mutation effects did not preserve ambiguity dominance")
	}

	t.Run("relation-after-link-reopen-converges", func(t *testing.T) {
		store, root := newObjectStoreForTest(t)
		bundle := c2Bundle(t)
		target := c2Target(t, bundle, c2Digest('3'), '4')
		attempt := issueC2AttemptFixture(store, target.AttemptArtifactDigest())
		object := c2SemanticObject(t, contractTargetKind, target.Digest(), target.CanonicalBytes())
		afterLinks := 0
		fault := func(actual contractFaultPhase) error {
			if actual == faultAfterLink {
				afterLinks++
				if afterLinks == 2 {
					return errors.New("lost relation result before parent sync")
				}
			}
			return nil
		}
		if record, effect, err := persistTargetRecord(
			context.Background(), store, targetStorageInput{attempt: attempt, object: object}, fault,
		); err == nil || record.validFor(store) || effect != contractAmbiguous {
			t.Fatal("relation after-link ambiguity returned authority")
		}
		restarted, err := OpenObjectStore(root)
		if err != nil {
			t.Fatal(err)
		}
		reopenedAttempt := issueC2AttemptFixture(restarted, attempt.digest)
		record, err := openTargetByAttempt(context.Background(), restarted, reopenedAttempt)
		if err != nil || !record.validFor(restarted) {
			t.Fatalf("visible relation did not converge on reopen: %v", err)
		}
		restartedAgain, err := OpenObjectStore(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := openTargetByAttempt(
			context.Background(), restartedAgain, issueC2AttemptFixture(restartedAgain, attempt.digest),
		); err != nil {
			t.Fatalf("converged relation did not reopen again: %v", err)
		}
	})
}

func TestC2NonheadPublicationNeverMutatesStudyHead(t *testing.T) {
	store, _ := newObjectStoreForTest(t)
	before, err := os.Lstat(store.studies)
	if err != nil {
		t.Fatal(err)
	}
	entriesBefore, _ := os.ReadDir(store.studies)
	bundle := c2Bundle(t)
	target := c2Target(t, bundle, c2Digest('3'), '4')
	attempt := issueC2AttemptFixture(store, target.AttemptArtifactDigest())
	object := c2SemanticObject(t, contractmodel.TargetKind, target.Digest(), target.CanonicalBytes())
	if _, _, err := persistTargetRecord(context.Background(), store, targetStorageInput{attempt: attempt, object: object}, nil); err != nil {
		t.Fatal(err)
	}
	after, err := os.Lstat(store.studies)
	entriesAfter, _ := os.ReadDir(store.studies)
	if err != nil || !os.SameFile(before, after) || len(entriesBefore) != len(entriesAfter) {
		t.Fatal("nonhead publication changed the study-head namespace")
	}
}
