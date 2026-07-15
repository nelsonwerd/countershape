package gitobj

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func testHex(character string, count int) string { return strings.Repeat(character, count) }

func TestIndependentBlobRehashRejectsMismatchedBytes(t *testing.T) {
	err := requireMatchingObjectOID(testHex("a", 40), testHex("b", 40), "blob")
	if code, ok := RefusalCodeOf(err); !ok || code != CodeObjectHashMismatch {
		t.Fatalf("mismatched object bytes returned %v, code=%q", err, code)
	}
	if err := requireMatchingObjectOID(testHex("a", 40), testHex("a", 40), "blob"); err != nil {
		t.Fatalf("matching object hash refused: %v", err)
	}
}

func TestRefusalCodeTraversesJoinedErrors(t *testing.T) {
	joined := errors.Join(errors.New("outer"), refuse(CodeUnsafePath, "joined", nil))
	if code, ok := RefusalCodeOf(joined); !ok || code != CodeUnsafePath {
		t.Fatalf("joined refusal code = %q, %v", code, ok)
	}
}

func rawTreeRecord(t *testing.T, mode, name, oid string) []byte {
	t.Helper()
	oidBytes, err := hex.DecodeString(oid)
	if err != nil {
		t.Fatal(err)
	}
	result := append([]byte(mode+" "+name), 0)
	return append(result, oidBytes...)
}

func TestRawTreeParserUsesExactOIDWidthAndCanonicalOrdering(t *testing.T) {
	first := rawTreeRecord(t, "100644", "a.txt", testHex("a", 40))
	second := rawTreeRecord(t, "100755", "b.txt", testHex("b", 40))
	parsed, err := parseRawTreeObject(ObjectSHA1, append(first, second...), newTreeParseBudget(4))
	if err != nil || len(parsed) != 2 || parsed[0].name[0] != 'a' || parsed[1].mode != "100755" {
		t.Fatalf("canonical raw tree did not parse: %+v, %v", parsed, err)
	}
	if _, err := parseRawTreeObject(ObjectSHA1, append(second, first...), newTreeParseBudget(4)); err == nil {
		t.Fatal("noncanonical raw-tree ordering was accepted")
	}
	if _, err := parseRawTreeObject(ObjectSHA256, first, newTreeParseBudget(4)); err == nil {
		t.Fatal("SHA-1-width raw OID was accepted for SHA-256")
	}
}

func TestTreeMetadataBudgetSeparatelyCapsObjectReads(t *testing.T) {
	small := newTreeParseBudget(1)
	if small.treeObjectLimit != maxTreeDepth+2 || small.recordLimit <= small.treeObjectLimit || small.metadataLimit <= 0 {
		t.Fatalf("small tree budget is not depth-safe and finite: %+v", small)
	}
	large := newTreeParseBudget(100000)
	if large.treeObjectLimit != maxTreeObjectReads || large.recordLimit != 100000+maxTreeObjectReads+256 ||
		large.metadataLimit != 32<<20 {
		t.Fatalf("large tree budget lost its independent hard caps: %+v", large)
	}
	large.treeObjects = large.treeObjectLimit
	if err := large.admitTreeObject(); err == nil {
		t.Fatal("tree-object read beyond the independent cap was accepted")
	} else if code, ok := RefusalCodeOf(err); !ok || code != CodeBudgetExceeded {
		t.Fatalf("tree-object read cap returned %v, code=%q", err, code)
	}
}

func TestPathValidationOccursBeforeAnyMaterializationAuthority(t *testing.T) {
	for _, candidate := range []string{"../escape", "safe/../escape", ".git/config", "safe\\escape", "line\nbreak"} {
		err := validateRawPath([]byte(candidate))
		if code, ok := RefusalCodeOf(err); !ok || code != CodeUnsafePath {
			t.Fatalf("path %q returned %v, code=%q", candidate, err, code)
		}
	}
}

func TestExecutableModeRemainsDistinct(t *testing.T) {
	mode, err := materializedFileMode("100755")
	if err != nil || mode.Perm() != 0o755 {
		t.Fatalf("executable mode collapsed: mode=%#o err=%v", mode.Perm(), err)
	}
	regular, err := materializedFileMode("100644")
	if err != nil || regular.Perm() != 0o644 || regular == mode {
		t.Fatalf("regular/executable modes are not distinct: regular=%#o executable=%#o err=%v", regular.Perm(), mode.Perm(), err)
	}
}

func TestSparseGitPolicyDisablesReplacementAndLazyFetch(t *testing.T) {
	environment := strings.Join(sparseGitEnvironment("/private/home", "/usr/bin/git"), "\n")
	for _, required := range []string{"GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null"} {
		if !strings.Contains(environment, required) {
			t.Fatalf("sparse Git environment lacks %s: %s", required, environment)
		}
	}
	args := strings.Join((closedGit{repository: "/repo"}).baseArgs(true), " ")
	if !strings.Contains(args, "--no-replace-objects") || !strings.Contains(args, "protocol.allow=never") {
		t.Fatalf("closed Git argv lost policy guards: %s", args)
	}
}

func TestSelectedTreeSetIdentityIsPortableAcrossRepositoryReceipts(t *testing.T) {
	format := ObjectSHA1
	commitA, treeA := testHex("a", 40), testHex("b", 40)
	commitB, treeB := testHex("c", 40), testHex("d", 40)
	identityA, err := digestIdentity("PinnedTreeIdentity", struct {
		SchemaVersion string       `json:"schema_version"`
		Kind          string       `json:"kind"`
		ObjectFormat  ObjectFormat `json:"object_format"`
		CommitOID     string       `json:"commit_oid"`
		TreeOID       string       `json:"tree_oid"`
	}{domain.SchemaVersion, "PinnedTreeIdentity", format, commitA, treeA})
	if err != nil {
		t.Fatal(err)
	}
	identityB, err := digestIdentity("PinnedTreeIdentity", struct {
		SchemaVersion string       `json:"schema_version"`
		Kind          string       `json:"kind"`
		ObjectFormat  ObjectFormat `json:"object_format"`
		CommitOID     string       `json:"commit_oid"`
		TreeOID       string       `json:"tree_oid"`
	}{domain.SchemaVersion, "PinnedTreeIdentity", format, commitB, treeB})
	if err != nil {
		t.Fatal(err)
	}
	makePins := func(receiptCharacter string) []PinnedTree {
		receipt := domain.MustDigest("sha256:" + testHex(receiptCharacter, 64))
		state := &repositoryState{objectFormat: format, fingerprint: receipt}
		return []PinnedTree{
			{repository: state, identity: identityA, commitOID: commitA, treeOID: treeA, displayRef: "refs/heads/a"},
			{repository: state, identity: identityB, commitOID: commitB, treeOID: treeB, displayRef: "refs/heads/b"},
		}
	}
	left, err := SelectTrees(makePins("1")...)
	if err != nil {
		t.Fatal(err)
	}
	right, err := SelectTrees(makePins("2")...)
	if err != nil {
		t.Fatal(err)
	}
	if left.Digest() != right.Digest() {
		t.Fatalf("repository receipt leaked into selected-set identity: %s != %s", left.Digest(), right.Digest())
	}
}

func TestSelectedTreeSetRejectsSameExecutableTreeUnderDifferentCommits(t *testing.T) {
	format := ObjectSHA1
	tree := testHex("d", 40)
	state := &repositoryState{objectFormat: format, fingerprint: domain.MustDigest("sha256:" + testHex("1", 64))}
	makePin := func(commitCharacter string) PinnedTree {
		commit := testHex(commitCharacter, 40)
		identity, err := digestIdentity("PinnedTreeIdentity", struct {
			SchemaVersion string       `json:"schema_version"`
			Kind          string       `json:"kind"`
			ObjectFormat  ObjectFormat `json:"object_format"`
			CommitOID     string       `json:"commit_oid"`
			TreeOID       string       `json:"tree_oid"`
		}{domain.SchemaVersion, "PinnedTreeIdentity", format, commit, tree})
		if err != nil {
			t.Fatal(err)
		}
		return PinnedTree{repository: state, identity: identity, commitOID: commit, treeOID: tree, displayRef: "refs/heads/" + commitCharacter}
	}
	if _, err := SelectTrees(makePin("a"), makePin("b")); err == nil {
		t.Fatal("same executable tree under different commits was accepted twice")
	}
}

func TestObjectFormatOIDWidthsAreClosed(t *testing.T) {
	if !validOID(ObjectSHA1, testHex("a", 40)) || validOID(ObjectSHA1, testHex("a", 64)) {
		t.Fatal("SHA-1 OID width was not exact")
	}
	if !validOID(ObjectSHA256, testHex("a", 64)) || validOID(ObjectSHA256, testHex("a", 40)) {
		t.Fatal("SHA-256 OID width was not exact")
	}
}

func TestDisplayRefProfileRejectsEveryGitAmbiguityClass(t *testing.T) {
	for _, valid := range []string{
		"refs/heads/candidate", "refs/tags/v1.2.3", testHex("a", 40), testHex("b", 64),
	} {
		if !validDisplayRef(valid) {
			t.Fatalf("valid display ref was rejected: %q", valid)
		}
	}
	for _, invalid := range []string{
		"@", "candidate", "refs/heads/.hidden", "refs/.hidden/name", "refs/heads/name.",
		"refs/heads/name.lock", "refs/heads/name.lock/child", "refs/heads/a..b",
		"refs/heads/a@{b", "refs/heads/a//b", "refs/heads/a b", "refs/heads/a\\b",
		"refs/heads/a~1", "refs/heads/a^1", "refs/heads/a?b", "refs/heads/a*b",
	} {
		if validDisplayRef(invalid) {
			t.Fatalf("ambiguous display ref was accepted: %q", invalid)
		}
	}
}
