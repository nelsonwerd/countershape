package store

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
