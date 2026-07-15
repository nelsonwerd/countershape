//go:build darwin && cgo

package gitobj_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nelsonwerd/countershape/internal/gitobj"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

func boundByRef(t *testing.T, candidates []gitobj.BoundCandidate, ref string) gitobj.BoundCandidate {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.Provenance().DisplayRef == ref {
			return candidate
		}
	}
	t.Fatalf("bound candidate %s is absent", ref)
	return gitobj.BoundCandidate{}
}

func TestMaterializedBytesModesAndGitAbsence(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	declaration, materializationPolicy := twoCandidateDeclaration(t, fixture,
		[]gitrepo.File{
			{Path: "README.txt", Mode: "100644", Content: []byte("first\n")},
			{Path: "bin/run", Mode: "100755", Content: []byte("#!/bin/sh\nexit 0\n")},
		},
		[]gitrepo.File{{Path: "README.txt", Mode: "100644", Content: []byte("second\n")}},
	)
	bound := boundByRef(t, bindCandidates(t, declaration, materializationPolicy), "refs/heads/first")
	parent := targetParent(t)
	receipt, err := (gitobj.DefaultMaterializer{}).Materialize(context.Background(), bound, parent)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Valid() || receipt.PublishedRoot != filepath.Join(parent, "candidate") || len(receipt.Entries) != 2 {
		t.Fatalf("invalid materialization receipt: %+v", receipt)
	}
	if receipt.ManifestPath != filepath.Join(parent, "materialization.manifest.json") {
		t.Fatalf("durable manifest path = %q", receipt.ManifestPath)
	}
	manifestInfo, err := os.Lstat(receipt.ManifestPath)
	if err != nil || !manifestInfo.Mode().IsRegular() || manifestInfo.Mode().Perm() != 0o600 || manifestInfo.Size() == 0 {
		t.Fatalf("durable manifest facts are incomplete: info=%v err=%v", manifestInfo, err)
	}
	if err := gitobj.ValidatePublishedMaterialization(context.Background(), bound, receipt); err != nil {
		t.Fatalf("freshly published candidate did not revalidate: %v", err)
	}
	for path, expected := range map[string]struct {
		content []byte
		mode    os.FileMode
	}{
		"README.txt": {content: []byte("first\n"), mode: 0o644},
		"bin/run":    {content: []byte("#!/bin/sh\nexit 0\n"), mode: 0o755},
	} {
		full := filepath.Join(receipt.PublishedRoot, filepath.FromSlash(path))
		content, readErr := os.ReadFile(full)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !bytes.Equal(content, expected.content) {
			t.Fatalf("materialized %s bytes differ: %q", path, content)
		}
		info, statErr := os.Lstat(full)
		if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != expected.mode {
			t.Fatalf("materialized %s mode: info=%v err=%v", path, info, statErr)
		}
	}
	if _, err := os.Lstat(filepath.Join(receipt.PublishedRoot, ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("materialized root contains .git or cannot prove absence: %v", err)
	}
}

func TestPublishedManifestAndOpaqueReceiptAreRevalidatedBeforeUse(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	declaration, materializationPolicy := twoCandidateDeclaration(t, fixture,
		[]gitrepo.File{{Path: "value.txt", Mode: "100644", Content: []byte("first\n")}},
		[]gitrepo.File{{Path: "value.txt", Mode: "100644", Content: []byte("second\n")}},
	)
	bound := boundByRef(t, bindCandidates(t, declaration, materializationPolicy), "refs/heads/first")
	receipt, err := (gitobj.DefaultMaterializer{}).Materialize(context.Background(), bound, targetParent(t))
	if err != nil {
		t.Fatal(err)
	}
	mutated := receipt
	mutated.Entries = append([]gitobj.ManifestEntry(nil), receipt.Entries...)
	mutated.Entries[0].Bytes++
	if mutated.Valid() {
		t.Fatal("caller-mutated receipt retained package authority")
	}
	if err := os.WriteFile(receipt.ManifestPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := gitobj.ValidatePublishedMaterialization(context.Background(), bound, receipt); err == nil {
		t.Fatal("changed durable manifest retained spawn authority")
	}
}

func TestAtomicPublicationNeverOverwritesExistingCandidate(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	declaration, materializationPolicy := twoCandidateDeclaration(t, fixture,
		[]gitrepo.File{{Path: "value.txt", Mode: "100644", Content: []byte("first\n")}},
		[]gitrepo.File{{Path: "value.txt", Mode: "100644", Content: []byte("second\n")}},
	)
	bound := boundByRef(t, bindCandidates(t, declaration, materializationPolicy), "refs/heads/first")
	parent := targetParent(t)
	materializer := gitobj.DefaultMaterializer{}
	first, err := materializer.Materialize(context.Background(), bound, parent)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(first.PublishedRoot, "value.txt"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = materializer.Materialize(context.Background(), bound, parent)
	refusalCode(t, err, gitobj.CodeInvalidConfig)
	after, err := os.ReadFile(filepath.Join(first.PublishedRoot, "value.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("second publication attempt changed the existing candidate")
	}
}

func TestMissingObjectLeavesNoPartialPublishedTree(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	commitRef(t, fixture, "refs/heads/first", []gitrepo.File{{Path: "first.txt", Mode: "100644", Content: []byte("first\n")}}, "", "first")
	commitRef(t, fixture, "refs/heads/second", []gitrepo.File{{Path: "second.txt", Mode: "100644", Content: []byte("second\n")}}, "", "second")
	repository := openRepository(t, fixture, systemGit)
	firstPin := pinRef(t, repository, "refs/heads/first")
	secondPin := pinRef(t, repository, "refs/heads/second")
	selected, err := gitobj.SelectTrees(firstPin, secondPin)
	if err != nil {
		t.Fatal(err)
	}
	materializationPolicy := policy(t, 64, 1<<20, 1<<18)
	target := targetParent(t)
	firstInspection := inspect(t, firstPin, materializationPolicy, target)
	secondInspection := inspect(t, secondPin, materializationPolicy, target)
	declaration, err := gitobj.NewCandidateSet(selected, materializationPolicy, firstInspection, secondInspection)
	if err != nil {
		t.Fatal(err)
	}
	bound := boundByRef(t, bindCandidates(t, declaration, materializationPolicy), "refs/heads/first")
	entries := firstInspection.Entries()
	if len(entries) != 1 {
		t.Fatalf("unexpected first manifest: %+v", entries)
	}
	if err := fixture.RemoveLooseObject(entries[0].GitOID); err != nil {
		t.Fatal(err)
	}
	parent := targetParent(t)
	_, err = (gitobj.DefaultMaterializer{}).Materialize(context.Background(), bound, parent)
	refusalCode(t, err, gitobj.CodeMissingObject)
	remaining, readErr := os.ReadDir(parent)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(remaining) != 0 {
		t.Fatalf("missing-object refusal left %d partial entries", len(remaining))
	}
}

func TestHooksFiltersAndNetworkSentinelsNeverRun(t *testing.T) {
	fixture := fixtureRepository(t, gitrepo.SHA1)
	commitRef(t, fixture, "refs/heads/first", []gitrepo.File{
		{Path: ".gitattributes", Mode: "100644", Content: []byte("*.txt filter=countershape\n")},
		{Path: "payload.txt", Mode: "100644", Content: []byte("raw candidate bytes\n")},
	}, "", "first")
	commitRef(t, fixture, "refs/heads/second", []gitrepo.File{{Path: "payload.txt", Mode: "100644", Content: []byte("other\n")}}, "", "second")
	sentinel := filepath.Join(t.TempDir(), "forbidden-policy-or-network")
	if err := fixture.InstallPolicySentinels(context.Background(), sentinel); err != nil {
		t.Fatal(err)
	}
	repository := openRepository(t, fixture, systemGit)
	first := pinRef(t, repository, "refs/heads/first")
	second := pinRef(t, repository, "refs/heads/second")
	selected, err := gitobj.SelectTrees(first, second)
	if err != nil {
		t.Fatal(err)
	}
	materializationPolicy := policy(t, 64, 1<<20, 1<<18)
	declaration, err := gitobj.InspectSelected(context.Background(), selected, materializationPolicy, targetParent(t))
	if err != nil {
		t.Fatal(err)
	}
	bound := boundByRef(t, bindCandidates(t, declaration, materializationPolicy), "refs/heads/first")
	receipt, err := (gitobj.DefaultMaterializer{}).Materialize(context.Background(), bound, targetParent(t))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(receipt.PublishedRoot, "payload.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, []byte("raw candidate bytes\n")) {
		t.Fatalf("filter transformed payload: %q", payload)
	}
	if _, err := os.Stat(sentinel); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Git hook, filter, or remote sentinel ran: %v", err)
	}
}
