//go:build darwin && cgo

package gitobj

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/testkit/gitrepo"
)

const mutationSystemGit = "/usr/bin/git"

func mutationRepository(t *testing.T) gitrepo.Repository {
	t.Helper()
	repository, err := gitrepo.Init(context.Background(), mutationSystemGit, t.TempDir(), gitrepo.SHA1)
	if err != nil {
		t.Fatal(err)
	}
	return repository
}

func mutationRefusalCode(t *testing.T, err error, want RefusalCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("wanted refusal %s, got success", want)
	}
	code, ok := RefusalCodeOf(err)
	if !ok || code != want {
		t.Fatalf("wanted refusal %s, got %v", want, err)
	}
}

func mutationFixedDigest(character string) domain.Digest {
	return domain.MustDigest("sha256:" + strings.Repeat(character, 64))
}

func mutationPinnedTree(
	t *testing.T,
	format ObjectFormat,
	commitCharacter string,
	treeCharacter string,
	repositoryReceiptCharacter string,
) PinnedTree {
	t.Helper()
	width := format.oidBytes() * 2
	commitOID := strings.Repeat(commitCharacter, width)
	treeOID := strings.Repeat(treeCharacter, width)
	identity, err := digestIdentity("PinnedTreeIdentity", struct {
		SchemaVersion string       `json:"schema_version"`
		Kind          string       `json:"kind"`
		ObjectFormat  ObjectFormat `json:"object_format"`
		CommitOID     string       `json:"commit_oid"`
		TreeOID       string       `json:"tree_oid"`
	}{domain.SchemaVersion, "PinnedTreeIdentity", format, commitOID, treeOID})
	if err != nil {
		t.Fatal(err)
	}
	state := &repositoryState{
		objectFormat: format,
		fingerprint:  mutationFixedDigest(repositoryReceiptCharacter),
	}
	return PinnedTree{
		repository: state,
		identity:   identity,
		commitOID:  commitOID,
		treeOID:    treeOID,
		displayRef: "refs/heads/mutation-contract",
	}
}

func mutationInspectedTree(
	pinned PinnedTree,
	policy Policy,
	portableTreeCharacter string,
	portableBlobCharacter string,
	device string,
	inode string,
) InspectedTree {
	oid := strings.Repeat("e", pinned.repository.objectFormat.oidBytes()*2)
	return InspectedTree{
		pinned: pinned,
		policy: policy,
		entries: []entryRecord{{
			path:           "candidate.txt",
			mode:           "100644",
			oid:            oid,
			size:           1,
			portableDigest: mutationFixedDigest(portableBlobCharacter),
		}},
		portableTreeDigest: mutationFixedDigest(portableTreeCharacter),
		targetVolume:       targetVolume{identity: filesystemIdentity{Device: device, Inode: inode}},
	}
}

func TestAlternateObjectDirectoryIsRejectedBeforeObjectRead(t *testing.T) {
	primary := mutationRepository(t)
	alternate := mutationRepository(t)
	if _, err := alternate.WriteBlob(context.Background(), []byte("alternate object\n")); err != nil {
		t.Fatal(err)
	}
	if err := primary.InstallAlternates(filepath.Join(alternate.Root, "objects")); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "git-commands.log")
	wrapper, err := gitrepo.WriteGitCommandLogWrapper(t.TempDir(), mutationSystemGit, logPath)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := OpenRepository(context.Background(), OpenConfig{
		GitExecutable: wrapper,
		Repository:    primary.Root,
		ScratchRoot:   t.TempDir(),
	})
	if err == nil {
		_ = opened.Close()
		t.Fatal("repository with a valid alternate object directory was admitted")
	}
	mutationRefusalCode(t, err, CodeAlternatesRejected)
	commands, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(commands), " cat-file ") {
		t.Fatalf("alternate refusal occurred after an object read:\n%s", commands)
	}
}

func TestUnsafePathRefusesBeforeCandidateBlobRead(t *testing.T) {
	fixture := mutationRepository(t)
	blobOID, err := fixture.WriteBlob(context.Background(), []byte("hostile candidate bytes\n"))
	if err != nil {
		t.Fatal(err)
	}
	treeOID, err := fixture.WriteRawTree(context.Background(), []gitrepo.RawTreeEntry{{
		Mode: "100644",
		Name: []byte(".git"),
		OID:  blobOID,
	}})
	if err != nil {
		t.Fatal(err)
	}
	commitOID, err := fixture.WriteCommitLiteral(context.Background(), treeOID)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.UpdateRef(context.Background(), "refs/heads/candidate", commitOID); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "git-commands.log")
	wrapper, err := gitrepo.WriteGitCommandLogWrapper(t.TempDir(), mutationSystemGit, logPath)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := OpenRepository(context.Background(), OpenConfig{
		GitExecutable: wrapper,
		Repository:    fixture.Root,
		ScratchRoot:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("close repository: %v", err)
		}
	})
	pinned, err := repository.Pin(context.Background(), "refs/heads/candidate")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	inspectionPolicy, err := NewPolicy(8, 4096, 4096)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Inspect(context.Background(), pinned, inspectionPolicy, t.TempDir())
	mutationRefusalCode(t, err, CodeUnsafePath)
	commands, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	transcript := string(commands)
	if !strings.Contains(transcript, " cat-file tree ") {
		t.Fatalf("fixture never consumed the verified tree object:\n%s", transcript)
	}
	if strings.Contains(transcript, " cat-file blob ") {
		t.Fatalf("unsafe path was refused only after candidate blob consumption:\n%s", transcript)
	}
}

func TestTargetFilesystemReservationCannotBeReplacedByLexicalCollisionCheck(t *testing.T) {
	probeFailure := errors.New("topology probe sentinel")
	probeCalled := false
	_, err := reserveTargetVolume(
		t.TempDir(),
		[]entryRecord{{path: "safe.txt"}},
		func(string, []entryRecord) (targetVolume, error) {
			probeCalled = true
			return targetVolume{}, probeFailure
		},
	)
	if !probeCalled {
		t.Fatal("target-volume reservation did not invoke the physical topology probe")
	}
	if !errors.Is(err, probeFailure) {
		t.Fatalf("target-volume reservation did not preserve the topology probe result: %v", err)
	}
}

func TestCandidateSetRejectsDuplicatePortableTreeAcrossObjectFormats(t *testing.T) {
	sha1Pin := mutationPinnedTree(t, ObjectSHA1, "a", "b", "1")
	sha256Pin := mutationPinnedTree(t, ObjectSHA256, "c", "d", "2")
	selected, err := SelectTrees(sha1Pin, sha256Pin)
	if err != nil {
		t.Fatal(err)
	}
	inspectionPolicy, err := NewPolicy(8, 4096, 4096)
	if err != nil {
		t.Fatal(err)
	}
	sharedPortableTree := "5"
	sha1Inspection := mutationInspectedTree(sha1Pin, inspectionPolicy, sharedPortableTree, "7", "11", "12")
	sha256Inspection := mutationInspectedTree(sha256Pin, inspectionPolicy, sharedPortableTree, "7", "21", "22")
	_, err = NewCandidateSet(selected, inspectionPolicy, sha1Inspection, sha256Inspection)
	mutationRefusalCode(t, err, CodeCandidateSetRejected)
}

func TestPrePlanCandidateSetDigestExcludesRepositoryAndPostPlanAuthority(t *testing.T) {
	leftPins := []PinnedTree{
		mutationPinnedTree(t, ObjectSHA1, "a", "b", "1"),
		mutationPinnedTree(t, ObjectSHA256, "c", "d", "2"),
	}
	rightPins := []PinnedTree{
		mutationPinnedTree(t, ObjectSHA1, "a", "b", "3"),
		mutationPinnedTree(t, ObjectSHA256, "c", "d", "4"),
	}
	leftSelected, err := SelectTrees(leftPins...)
	if err != nil {
		t.Fatal(err)
	}
	rightSelected, err := SelectTrees(rightPins...)
	if err != nil {
		t.Fatal(err)
	}
	if leftSelected.Digest() != rightSelected.Digest() {
		t.Fatal("repository receipt leaked into the pre-plan selected-set identity")
	}
	leftPolicy, err := NewPolicy(4, 4096, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rightPolicy, err := NewPolicy(8, 8192, 4096)
	if err != nil {
		t.Fatal(err)
	}
	leftDeclaration, err := NewCandidateSet(
		leftSelected,
		leftPolicy,
		mutationInspectedTree(leftPins[0], leftPolicy, "5", "9", "11", "12"),
		mutationInspectedTree(leftPins[1], leftPolicy, "6", "a", "21", "22"),
	)
	if err != nil {
		t.Fatal(err)
	}
	rightDeclaration, err := NewCandidateSet(
		rightSelected,
		rightPolicy,
		mutationInspectedTree(rightPins[0], rightPolicy, "7", "b", "31", "32"),
		mutationInspectedTree(rightPins[1], rightPolicy, "8", "c", "41", "42"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if leftDeclaration.PolicyDigest() == rightDeclaration.PolicyDigest() ||
		leftDeclaration.candidates[0].portableTreeDigest == rightDeclaration.candidates[0].portableTreeDigest {
		t.Fatal("fixture did not vary post-selection inspection authority")
	}
	if leftDeclaration.Digest() != leftSelected.Digest() ||
		rightDeclaration.Digest() != rightSelected.Digest() ||
		leftDeclaration.Digest() != rightDeclaration.Digest() {
		t.Fatalf(
			"post-selection authority polluted the pre-plan digest: left=%s right=%s selected=%s",
			leftDeclaration.Digest(), rightDeclaration.Digest(), leftSelected.Digest(),
		)
	}
}

var durableManifestFault = errors.New("injected durable manifest operation failure")

type durableManifestFaultFile struct {
	handle   *os.File
	phase    string
	injected bool
}

func (file *durableManifestFaultFile) Write(data []byte) (int, error) {
	if file.phase == "write" && !file.injected {
		file.injected = true
		return 0, durableManifestFault
	}
	return file.handle.Write(data)
}

func (file *durableManifestFaultFile) Chmod(mode os.FileMode) error {
	if file.phase == "chmod" && !file.injected {
		file.injected = true
		return durableManifestFault
	}
	return file.handle.Chmod(mode)
}

func (file *durableManifestFaultFile) Sync() error {
	if file.phase == "sync" && !file.injected {
		file.injected = true
		return durableManifestFault
	}
	return file.handle.Sync()
}

func (file *durableManifestFaultFile) Close() error {
	if file.phase == "close" && !file.injected {
		file.injected = true
		return durableManifestFault
	}
	return file.handle.Close()
}

func durableManifestFaultOperations(phase string, parentSync func(string) error) durableExclusiveFileOperations {
	operations := defaultDurableExclusiveFileOperations()
	operations.open = func(path string, flags int, mode os.FileMode) (durableExclusiveFile, error) {
		handle, err := os.OpenFile(path, flags, mode)
		if err != nil {
			return nil, err
		}
		return &durableManifestFaultFile{handle: handle, phase: phase}, nil
	}
	operations.syncParent = parentSync
	return operations
}

func TestDurableManifestOperationFailuresRemoveAndSyncCreatedFile(t *testing.T) {
	for _, phase := range []string{"write", "chmod", "sync", "close"} {
		t.Run(phase, func(t *testing.T) {
			parent := t.TempDir()
			path := filepath.Join(parent, materializationManifestFilename)
			parentSyncs := 0
			operations := durableManifestFaultOperations(phase, func(actual string) error {
				parentSyncs++
				if actual != parent {
					t.Fatalf("cleanup synced %q, want %q", actual, parent)
				}
				return syncDirectory(actual)
			})
			err := writeDurableExclusiveFileWithOperations(path, []byte("manifest\n"), 0o600, operations)
			mutationRefusalCode(t, err, CodePublicationFailed)
			if !errors.Is(err, durableManifestFault) {
				t.Fatalf("%s failure lost its operation cause: %v", phase, err)
			}
			if parentSyncs != 1 {
				t.Fatalf("%s failure synced its parent %d times, want 1", phase, parentSyncs)
			}
			if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("%s failure did not prove manifest absence: %v", phase, statErr)
			}
		})
	}
}

func TestDurableManifestParentSyncFailureRunsDurableCleanup(t *testing.T) {
	t.Run("cleanup proves durable absence", func(t *testing.T) {
		parent := t.TempDir()
		path := filepath.Join(parent, materializationManifestFilename)
		parentSyncs := 0
		operations := durableManifestFaultOperations("", func(actual string) error {
			parentSyncs++
			if parentSyncs == 1 {
				return durableManifestFault
			}
			return syncDirectory(actual)
		})
		err := writeDurableExclusiveFileWithOperations(path, []byte("manifest\n"), 0o600, operations)
		mutationRefusalCode(t, err, CodePublicationFailed)
		if parentSyncs != 2 {
			t.Fatalf("parent-sync failure made %d sync attempts, want create and cleanup attempts", parentSyncs)
		}
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("parent-sync failure did not clean the created manifest: %v", statErr)
		}
	})

	t.Run("cleanup sync cannot prove durable absence", func(t *testing.T) {
		parent := t.TempDir()
		path := filepath.Join(parent, materializationManifestFilename)
		parentSyncs := 0
		operations := durableManifestFaultOperations("", func(string) error {
			parentSyncs++
			return durableManifestFault
		})
		err := writeDurableExclusiveFileWithOperations(path, []byte("manifest\n"), 0o600, operations)
		mutationRefusalCode(t, err, CodePublicationAmbiguous)
		if parentSyncs != 2 {
			t.Fatalf("ambiguous parent-sync failure made %d sync attempts, want 2", parentSyncs)
		}
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("ambiguous parent-sync fixture did not remove the manifest: %v", statErr)
		}
	})
}

func TestDurableManifestCleanupWithoutDurableAbsenceIsAmbiguous(t *testing.T) {
	t.Run("remove cannot prove absence", func(t *testing.T) {
		parent := t.TempDir()
		path := filepath.Join(parent, materializationManifestFilename)
		parentSyncs := 0
		operations := durableManifestFaultOperations("write", func(actual string) error {
			parentSyncs++
			return syncDirectory(actual)
		})
		operations.remove = func(string) error { return durableManifestFault }
		err := writeDurableExclusiveFileWithOperations(path, []byte("manifest\n"), 0o600, operations)
		mutationRefusalCode(t, err, CodePublicationAmbiguous)
		if parentSyncs != 1 {
			t.Fatalf("ambiguous removal synced its parent %d times, want 1", parentSyncs)
		}
		if _, statErr := os.Lstat(path); statErr != nil {
			t.Fatalf("ambiguous removal fixture did not retain the manifest: %v", statErr)
		}
	})

	t.Run("parent sync fails", func(t *testing.T) {
		parent := t.TempDir()
		path := filepath.Join(parent, materializationManifestFilename)
		parentSyncs := 0
		operations := durableManifestFaultOperations("chmod", func(string) error {
			parentSyncs++
			return durableManifestFault
		})
		err := writeDurableExclusiveFileWithOperations(path, []byte("manifest\n"), 0o600, operations)
		mutationRefusalCode(t, err, CodePublicationAmbiguous)
		if parentSyncs != 1 {
			t.Fatalf("failed cleanup sync ran %d times, want 1", parentSyncs)
		}
		if _, statErr := os.Lstat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("parent-sync ambiguity fixture did not remove the manifest: %v", statErr)
		}
	})
}

type repeatedDirectoryEntryReader struct {
	entry        os.DirEntry
	remaining    int
	read         int
	maxRequested int
}

type boundedParentReader struct {
	repeatedDirectoryEntryReader
	closeErr error
	closed   bool
}

func (reader *boundedParentReader) Close() error {
	reader.closed = true
	return reader.closeErr
}

func (reader *repeatedDirectoryEntryReader) ReadDir(count int) ([]os.DirEntry, error) {
	requested := count
	if count > reader.maxRequested {
		reader.maxRequested = count
	}
	if reader.remaining == 0 {
		return nil, io.EOF
	}
	if count > reader.remaining {
		count = reader.remaining
	}
	entries := make([]os.DirEntry, count)
	for index := range entries {
		entries[index] = reader.entry
	}
	reader.remaining -= count
	reader.read += count
	if reader.remaining == 0 && count < requested {
		return entries, io.EOF
	}
	return entries, nil
}

func TestPromisorPackDirectoryRejectsEntryCountAboveClosedBound(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "fixture.pack"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("cannot build bounded reader fixture: entries=%d err=%v", len(entries), err)
	}
	reader := &repeatedDirectoryEntryReader{
		entry: entries[0], remaining: maxObjectPackDirectoryEntries + 1,
	}
	err = classifyPromisorPackDirectory(reader)
	mutationRefusalCode(t, err, CodeRepositoryRejected)
	if reader.read != maxObjectPackDirectoryEntries+1 {
		t.Fatalf("classification consumed %d entries, want exactly the refusal boundary", reader.read)
	}
	if reader.maxRequested > objectPackDirectoryReadBatch {
		t.Fatalf("classification requested an unbounded page of %d entries", reader.maxRequested)
	}
}

func TestPhysicalPromisorPackDirectoryRejectsEntryCountAboveClosedBound(t *testing.T) {
	objectDirectory := t.TempDir()
	packDirectory := filepath.Join(objectDirectory, "pack")
	if err := os.Mkdir(packDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	for index := 0; index <= maxObjectPackDirectoryEntries; index++ {
		path := filepath.Join(packDirectory, fmt.Sprintf("entry-%05d.idx", index))
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatalf("create physical pack entry %d: %v", index, err)
		}
	}
	mutationRefusalCode(t, rejectPromisorMarkers(objectDirectory), CodeRepositoryRejected)
}

func TestMaterializationParentInspectionStopsAtFirstPhysicalEntry(t *testing.T) {
	fixture := t.TempDir()
	if err := os.WriteFile(filepath.Join(fixture, "entry"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(fixture)
	if err != nil || len(entries) != 1 {
		t.Fatalf("cannot build parent reader fixture: entries=%d err=%v", len(entries), err)
	}
	reader := &boundedParentReader{repeatedDirectoryEntryReader: repeatedDirectoryEntryReader{
		entry: entries[0], remaining: maxObjectPackDirectoryEntries + 1,
	}}
	mutationRefusalCode(t, inspectEmptyMaterializationParent(reader), CodeInvalidConfig)
	if !reader.closed || reader.read != 1 || reader.maxRequested != 1 {
		t.Fatalf("parent inspection was not one-page bounded: closed=%v read=%d requested=%d", reader.closed, reader.read, reader.maxRequested)
	}

	closeFailure := &boundedParentReader{closeErr: errors.New("injected parent close failure")}
	mutationRefusalCode(t, inspectEmptyMaterializationParent(closeFailure), CodeInvalidConfig)
	if !closeFailure.closed {
		t.Fatal("parent close-failure fixture was not closed")
	}
}

func TestLargePhysicalMaterializationParentRefusesWithoutWholeDirectoryAdmission(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 2048; index++ {
		path := filepath.Join(parent, fmt.Sprintf("preexisting-%05d", index))
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatalf("create physical parent entry %d: %v", index, err)
		}
	}
	_, err = requirePrivateEmptyParent(parent)
	mutationRefusalCode(t, err, CodeInvalidConfig)
}
