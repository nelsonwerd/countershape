//go:build darwin && arm64 && cgo

package contractmaterialize

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
	"github.com/nelsonwerd/countershape/internal/emit/node/model"
)

func TestDescriptorRelativeSixFilePublicationIsExclusiveAndExact(t *testing.T) {
	bundle := materializerTestBundle(t)
	ops := defaultOperations()
	parentPath := materializerResolvedTempDir(t)
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()

	stage, err := allocateStage(parent.handle, ops)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeExactBundle(context.Background(), stage.handle, bundle, ops); err != nil {
		t.Fatal(err)
	}
	if err := verifyExactDirectoryHandle(stage.handle, bundle, ops); err != nil {
		t.Fatal(err)
	}
	if err := stage.handle.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := ops.renameExclusiveAt(parent.handle, stage.name, "contract"); err != nil {
		t.Fatal(err)
	}
	if err := parent.handle.Sync(); err != nil {
		t.Fatal(err)
	}
	final, err := requireNamedDirectory(parent.handle, "contract", ops)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyExactDirectoryHandle(final, bundle, ops); err != nil {
		t.Fatal(err)
	}
	finalInfo, err := final.Stat()
	if err != nil || !os.SameFile(stage.info, finalInfo) {
		t.Fatalf("published directory identity changed: %v", err)
	}
	if err := final.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stage.handle.Close(); err != nil {
		t.Fatal(err)
	}

	loser, err := allocateStage(parent.handle, ops)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeExactBundle(context.Background(), loser.handle, bundle, ops); err != nil {
		t.Fatal(err)
	}
	if err := ops.renameExclusiveAt(parent.handle, loser.name, "contract"); err == nil {
		t.Fatal("exclusive directory publication overwrote an existing destination")
	}
	if err := verifyExactDirectoryHandle(loser.handle, bundle, ops); err != nil {
		t.Fatalf("losing exclusive rename consumed or changed its source: %v", err)
	}
	if err := cleanupStage(parent.handle, loser, bundle, ops); err != nil {
		t.Fatalf("clean losing stage: %v", err)
	}
	if existing, err := requireNamedDirectory(parent.handle, "contract", ops); err != nil {
		t.Fatal(err)
	} else {
		defer existing.Close()
		if err := verifyExactDirectoryHandle(existing, bundle, ops); err != nil {
			t.Fatalf("exclusive collision changed the winner: %v", err)
		}
	}
}

func TestRetainedParentDescriptorCannotBeRedirectedByPathReplacement(t *testing.T) {
	bundle := materializerTestBundle(t)
	ops := defaultOperations()
	root := materializerShortTempDir(t)
	parentPath := filepath.Join(root, "parent")
	if err := os.Mkdir(parentPath, 0o700); err != nil {
		t.Fatal(err)
	}
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	moved := filepath.Join(root, "moved-parent")
	if err := os.Rename(parentPath, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(parentPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := validateParentPathIdentity(parent, ops); err == nil {
		t.Fatal("replacement parent path retained the original authority")
	}
	stage, err := allocateStage(parent.handle, ops)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(moved, stage.name)); err != nil {
		t.Fatalf("descriptor-relative stage did not stay under the retained parent: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(parentPath, stage.name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stage was redirected into replacement parent: %v", err)
	}
	if err := cleanupStage(parent.handle, stage, bundle, ops); err != nil {
		t.Fatalf("descriptor-relative cleanup failed under renamed parent: %v", err)
	}
}

func TestRetainedParentRefusesOwnershipModeTrustDrift(t *testing.T) {
	ops := defaultOperations()
	parentPath := materializerResolvedTempDir(t)
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	if err := os.Chmod(parentPath, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := validateParentPathIdentity(parent, ops); err == nil {
		t.Fatal("group/other-writable parent drift retained publication authority")
	}
}

func TestParentDescriptorWalkRefusesIntermediateSymlinkReplacement(t *testing.T) {
	ops := defaultOperations()
	root := materializerResolvedTempDir(t)
	ancestor := filepath.Join(root, "ancestor")
	parentPath := filepath.Join(ancestor, "parent")
	if err := os.MkdirAll(parentPath, 0o700); err != nil {
		t.Fatal(err)
	}
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	moved := filepath.Join(root, "moved-ancestor")
	if err := os.Rename(ancestor, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, ancestor); err != nil {
		t.Fatal(err)
	}
	if err := validateParentPathIdentity(parent, ops); err == nil {
		t.Fatal("intermediate symlink replacement retained parent authority")
	}
	if reopened, err := openDirectoryPathNoFollow(parentPath); err == nil {
		_ = reopened.Close()
		t.Fatal("component-wise no-follow open accepted an intermediate symlink")
	}
}

func TestDestinationFilesystemNameAliasIsRefused(t *testing.T) {
	ops := defaultOperations()
	parentPath := materializerResolvedTempDir(t)
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	if err := ops.mkdirAt(parent.handle, "Contract", 0o700); err != nil {
		t.Fatal(err)
	}
	if alias, err := openExactNamedDirectory(parent.handle, "contract", ops); err == nil || alias != nil {
		t.Fatalf("alternate-case destination was accepted: %#v, %v", alias, err)
	}
	if err := ops.mkdirAt(parent.handle, "contract", 0o700); err == nil {
		if exact, openErr := openExactNamedDirectory(parent.handle, "contract", ops); openErr == nil || exact != nil {
			t.Fatalf("exact destination with an alternate-case sibling was accepted: %#v, %v", exact, openErr)
		}
	}
	precomposed := "caf\u00e9"
	decomposed := "cafe\u0301"
	if err := ops.mkdirAt(parent.handle, precomposed, 0o700); err != nil {
		t.Fatal(err)
	}
	direct, directErr := ops.openDirectoryAt(parent.handle, decomposed)
	if directErr == nil {
		_ = direct.Close()
		if alias, err := openExactNamedDirectory(parent.handle, decomposed, ops); err == nil || alias != nil {
			t.Fatalf("normalization-insensitive filesystem alias was accepted: %#v, %v", alias, err)
		}
	}
}

func TestFinalMemberReopenRereadsSameInodeBytes(t *testing.T) {
	bundle := materializerTestBundle(t)
	ops := defaultOperations()
	parentPath := materializerResolvedTempDir(t)
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	stage, err := allocateStage(parent.handle, ops)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeExactBundle(context.Background(), stage.handle, bundle, ops); err != nil {
		t.Fatal(err)
	}
	target := bundle.Files()[0]
	targetPath := filepath.Join(parentPath, stage.name, target.Path())
	wrong := target.Content()
	wrong[0] ^= 0xff
	openCount := 0
	faultOps := ops
	faultOps.openFileAt = func(parent durableDirectory, name string, flags int, mode os.FileMode) (durableFile, error) {
		if name == target.Path() {
			openCount++
			if openCount == 2 {
				if err := os.WriteFile(targetPath, wrong, 0o644); err != nil {
					return nil, err
				}
			}
		}
		return ops.openFileAt(parent, name, flags, mode)
	}
	if err := verifyExactFileAt(stage.handle, target, faultOps); err == nil {
		t.Fatal("same-inode same-size mutation between member opens was accepted")
	}
	if openCount != 2 {
		t.Fatalf("member open count = %d, want 2", openCount)
	}
	if err := cleanupStage(parent.handle, stage, bundle, ops); err != nil {
		t.Fatalf("clean mutated test stage: %v", err)
	}
}

func TestStageAllocationCleansRetainedFaultsOrReportsUnknownOwnership(t *testing.T) {
	ops := defaultOperations()
	fault := errors.New("injected stage allocation fault")
	for _, test := range []struct {
		name       string
		wrap       func(durableDirectory) durableDirectory
		wantAbsent bool
	}{
		{"chmod", func(handle durableDirectory) durableDirectory {
			return &materializerFaultDirectory{durableDirectory: handle, chmodErr: fault}
		}, true},
		{"second-stat", func(handle durableDirectory) durableDirectory {
			return &materializerFaultDirectory{durableDirectory: handle, statErr: fault, statErrAt: 2}
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			parentPath := materializerShortTempDir(t)
			parent, err := retainParent(parentPath, ops)
			if err != nil {
				t.Fatal(err)
			}
			defer parent.handle.Close()
			faultOps := ops
			faultOps.openDirectoryAt = func(parent durableDirectory, name string) (durableDirectory, error) {
				handle, err := ops.openDirectoryAt(parent, name)
				if err != nil || !strings.HasPrefix(name, stageNamePrefix) {
					return handle, err
				}
				return test.wrap(handle), nil
			}
			if stage, err := allocateStage(parent.handle, faultOps); err == nil || stage.handle != nil {
				t.Fatalf("faulted stage allocation succeeded: %#v, %v", stage, err)
			}
			if names := materializerStageNames(t, parentPath); test.wantAbsent && len(names) != 0 {
				t.Fatalf("faulted retained allocation left stages: %v", names)
			}
		})
	}

	t.Run("open-identity-unavailable", func(t *testing.T) {
		parentPath := materializerResolvedTempDir(t)
		parent, err := retainParent(parentPath, ops)
		if err != nil {
			t.Fatal(err)
		}
		defer parent.handle.Close()
		faultOps := ops
		faultOps.openDirectoryAt = func(parent durableDirectory, name string) (durableDirectory, error) {
			if strings.HasPrefix(name, stageNamePrefix) {
				return nil, fault
			}
			return ops.openDirectoryAt(parent, name)
		}
		stage, allocationErr := allocateStage(parent.handle, faultOps)
		if allocationErr == nil || stage.handle != nil || !strings.Contains(allocationErr.Error(), "safe cleanup") {
			t.Fatalf("unretained stage allocation did not report cleanup uncertainty: %#v, %v", stage, allocationErr)
		}
		names := materializerStageNames(t, parentPath)
		if len(names) != 1 {
			t.Fatalf("cleanup-uncertain stage roster = %v, want one retained leak", names)
		}
		if err := os.Remove(filepath.Join(parentPath, names[0])); err != nil {
			t.Fatal(err)
		}
	})
}

func TestDirectoryRosterCloseFailureRefusesExactness(t *testing.T) {
	bundle := materializerTestBundle(t)
	ops := defaultOperations()
	parentPath := materializerResolvedTempDir(t)
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	stage, err := allocateStage(parent.handle, ops)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeExactBundle(context.Background(), stage.handle, bundle, ops); err != nil {
		t.Fatal(err)
	}
	faultOps := ops
	faultOps.openDirectoryAt = func(parent durableDirectory, name string) (durableDirectory, error) {
		handle, err := ops.openDirectoryAt(parent, name)
		if err != nil || name != "." {
			return handle, err
		}
		return &materializerFaultDirectory{durableDirectory: handle, closeErr: errors.New("injected roster close fault")}, nil
	}
	if err := verifyExactDirectoryHandle(stage.handle, bundle, faultOps); err == nil {
		t.Fatal("directory roster close failure was accepted")
	}
	if err := cleanupStage(parent.handle, stage, bundle, ops); err != nil {
		t.Fatalf("clean close-fault test stage: %v", err)
	}
}

func TestPartialPrivateStageCleanupIsExactAndAnchored(t *testing.T) {
	bundle := materializerTestBundle(t)
	ops := defaultOperations()
	parentPath := materializerResolvedTempDir(t)
	parent, err := retainParent(parentPath, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	stage, err := allocateStage(parent.handle, ops)
	if err != nil {
		t.Fatal(err)
	}
	first := bundle.Files()[0]
	handle, err := ops.openFileAt(stage.handle, first.Path(), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handle.Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}
	if err := handle.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cleanupStage(parent.handle, stage, bundle, ops); err != nil {
		t.Fatalf("partial owned stage was not safely removed: %v", err)
	}
	if existing, err := openExactNamedDirectory(parent.handle, stage.name, ops); err != nil || existing != nil {
		t.Fatalf("partial stage remains after cleanup: %#v, %v", existing, err)
	}

	unknownStage, err := allocateStage(parent.handle, ops)
	if err != nil {
		t.Fatal(err)
	}
	unknown, err := ops.openFileAt(unknownStage.handle, "unknown", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := unknown.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cleanupStage(parent.handle, unknownStage, bundle, ops); err == nil {
		t.Fatal("cleanup deleted a stage containing an unknown entry")
	}
	reopenedUnknown, err := requireNamedDirectory(parent.handle, unknownStage.name, ops)
	if err != nil {
		t.Fatal(err)
	}
	if err := ops.unlinkAt(reopenedUnknown, "unknown", false); err != nil {
		t.Fatal(err)
	}
	if err := reopenedUnknown.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ops.unlinkAt(parent.handle, unknownStage.name, true); err != nil {
		t.Fatal(err)
	}
}

func TestExactWriterHandlesShortWritesAndRefusesEveryFailurePhase(t *testing.T) {
	exact := []byte("exact contract bytes")
	short := &materializerFaultFile{maximumWrite: 1}
	if err := writeAndCloseExact(short, exact); err != nil || !bytes.Equal(short.body.Bytes(), exact) ||
		short.chmodCalls != 1 || short.syncCalls != 1 || short.closeCalls != 1 {
		t.Fatalf("short-write loop did not finish exactly: %#v, %v", short, err)
	}
	fault := errors.New("injected materializer fault")
	tests := []struct {
		name string
		file *materializerFaultFile
	}{
		{"zero-progress", &materializerFaultFile{zeroProgress: true}},
		{"write", &materializerFaultFile{writeErr: fault}},
		{"chmod", &materializerFaultFile{chmodErr: fault}},
		{"sync", &materializerFaultFile{syncErr: fault}},
		{"close", &materializerFaultFile{closeErr: fault}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := writeAndCloseExact(test.file, exact)
			if err == nil {
				t.Fatal("injected writer failure was accepted")
			}
			if test.name == "zero-progress" && !errors.Is(err, io.ErrNoProgress) {
				t.Fatalf("zero progress error = %v", err)
			}
			if test.name != "zero-progress" && !errors.Is(err, fault) {
				t.Fatalf("%s error lost its injected cause: %v", test.name, err)
			}
			if test.file.closeCalls != 1 {
				t.Fatalf("%s close calls = %d, want 1", test.name, test.file.closeCalls)
			}
		})
	}
}

func TestExactPhysicalModesRejectSpecialBits(t *testing.T) {
	for _, special := range []os.FileMode{os.ModeSetuid, os.ModeSetgid, os.ModeSticky} {
		if exactPhysicalMode(0o644|special, 0o644) || exactPhysicalMode(0o700|special, 0o700) {
			t.Fatalf("special mode bit %#o was accepted", special)
		}
	}
}

func TestDestinationSyntaxParentTrustAndNamedKindRefusals(t *testing.T) {
	for name, destination := range map[string]string{
		"empty":    "",
		"relative": "contract",
		"root":     string(filepath.Separator),
		"unclean":  filepath.Join(string(filepath.Separator), "tmp", "x", "..", "contract") + "/.",
		"control":  filepath.Join(string(filepath.Separator), "tmp", "contract\nname"),
	} {
		t.Run("syntax-"+name, func(t *testing.T) {
			if err := validDestinationSyntax(destination); err == nil {
				t.Fatalf("invalid destination %q was accepted", destination)
			}
		})
	}

	ops := defaultOperations()
	root := materializerShortTempDir(t)
	missing := filepath.Join(root, "missing")
	if parent, err := retainParent(missing, ops); err == nil || parent.handle != nil {
		t.Fatalf("missing parent was retained: %#v, %v", parent, err)
	}
	regular := filepath.Join(root, "regular")
	if err := os.WriteFile(regular, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if parent, err := retainParent(regular, ops); err == nil || parent.handle != nil {
		t.Fatalf("regular-file parent was retained: %#v, %v", parent, err)
	}
	realParent := filepath.Join(root, "real-parent")
	if err := os.Mkdir(realParent, 0o700); err != nil {
		t.Fatal(err)
	}
	symlinkParent := filepath.Join(root, "symlink-parent")
	if err := os.Symlink(realParent, symlinkParent); err != nil {
		t.Fatal(err)
	}
	if parent, err := retainParent(symlinkParent, ops); err == nil || parent.handle != nil {
		t.Fatalf("symlinked parent was retained: %#v, %v", parent, err)
	}
	sharedParent := filepath.Join(root, "shared-parent")
	if err := os.Mkdir(sharedParent, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sharedParent, 0o777); err != nil {
		t.Fatal(err)
	}
	if parent, err := retainParent(sharedParent, ops); err == nil || parent.handle != nil {
		t.Fatalf("group/other-writable parent was retained: %#v, %v", parent, err)
	}

	parent, err := retainParent(realParent, ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	assertRefused := func(name string) {
		t.Helper()
		handle, openErr := openExactNamedDirectory(parent.handle, name, ops)
		if handle != nil {
			_ = handle.Close()
		}
		if openErr == nil || handle != nil {
			t.Fatalf("non-directory destination %q was accepted: %#v, %v", name, handle, openErr)
		}
	}
	if err := os.WriteFile(filepath.Join(realParent, "file"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertRefused("file")
	if err := os.Symlink("file", filepath.Join(realParent, "symlink")); err != nil {
		t.Fatal(err)
	}
	assertRefused("symlink")
	if err := syscall.Mkfifo(filepath.Join(realParent, "fifo"), 0o600); err != nil {
		t.Fatal(err)
	}
	assertRefused("fifo")
	listener, err := net.Listen("unix", filepath.Join(realParent, "socket"))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	assertRefused("socket")

	dev, err := openDirectoryPathNoFollow("/dev")
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()
	device, deviceErr := openExactNamedDirectory(dev, "null", ops)
	if device != nil {
		_ = device.Close()
	}
	if deviceErr == nil || device != nil {
		t.Fatalf("device destination was accepted: %#v, %v", device, deviceErr)
	}
}

func TestExactDirectoryRefusesEveryPhysicalMemberMismatch(t *testing.T) {
	bundle := materializerTestBundle(t)
	ops := defaultOperations()
	target := bundle.Files()[0]

	tests := []struct {
		name   string
		mutate func(t *testing.T, parentPath string, stage retainedStage)
	}{
		{"root-wrong-mode", func(t *testing.T, _ string, stage retainedStage) {
			if err := stage.handle.Chmod(0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"missing-member", func(t *testing.T, parentPath string, stage retainedStage) {
			if err := os.Remove(filepath.Join(parentPath, stage.name, target.Path())); err != nil {
				t.Fatal(err)
			}
		}},
		{"extra-member", func(t *testing.T, parentPath string, stage retainedStage) {
			if err := os.WriteFile(filepath.Join(parentPath, stage.name, "extra"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"alternate-case-member", func(t *testing.T, parentPath string, stage retainedStage) {
			exactPath := filepath.Join(parentPath, stage.name, target.Path())
			aliasPath := filepath.Join(parentPath, stage.name, strings.ToUpper(target.Path()))
			if err := os.Remove(exactPath); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(aliasPath, target.Content(), 0o644); err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(filepath.Join(parentPath, stage.name))
			if err != nil {
				t.Fatal(err)
			}
			storedAlias := false
			for _, entry := range entries {
				if entry.Name() == strings.ToUpper(target.Path()) {
					storedAlias = true
				}
			}
			if !storedAlias {
				t.Skip("filesystem did not preserve a distinct alternate-case member name")
			}
		}},
		{"member-symlink", func(t *testing.T, parentPath string, stage retainedStage) {
			path := filepath.Join(parentPath, stage.name, target.Path())
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(bundle.Files()[1].Path(), path); err != nil {
				t.Fatal(err)
			}
		}},
		{"member-hardlink", func(t *testing.T, parentPath string, stage retainedStage) {
			path := filepath.Join(parentPath, stage.name, target.Path())
			if err := os.Link(path, filepath.Join(parentPath, "outside-hardlink")); err != nil {
				t.Fatal(err)
			}
		}},
		{"member-directory", func(t *testing.T, parentPath string, stage retainedStage) {
			path := filepath.Join(parentPath, stage.name, target.Path())
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"member-fifo", func(t *testing.T, parentPath string, stage retainedStage) {
			path := filepath.Join(parentPath, stage.name, target.Path())
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"member-socket", func(t *testing.T, parentPath string, stage retainedStage) {
			path := filepath.Join(parentPath, stage.name, target.Path())
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("unix", path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = listener.Close() })
		}},
		{"member-wrong-mode", func(t *testing.T, parentPath string, stage retainedStage) {
			if err := os.Chmod(filepath.Join(parentPath, stage.name, target.Path()), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"member-wrong-count", func(t *testing.T, parentPath string, stage retainedStage) {
			if err := os.WriteFile(filepath.Join(parentPath, stage.name, target.Path()), []byte("wrong"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"member-same-count-wrong-digest", func(t *testing.T, parentPath string, stage retainedStage) {
			wrong := target.Content()
			wrong[0] ^= 0xff
			if err := os.WriteFile(filepath.Join(parentPath, stage.name, target.Path()), wrong, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parentPath := materializerShortTempDir(t)
			parent, err := retainParent(parentPath, ops)
			if err != nil {
				t.Fatal(err)
			}
			defer parent.handle.Close()
			stage, err := allocateStage(parent.handle, ops)
			if err != nil {
				t.Fatal(err)
			}
			defer stage.handle.Close()
			if err := writeExactBundle(context.Background(), stage.handle, bundle, ops); err != nil {
				t.Fatal(err)
			}
			test.mutate(t, parentPath, stage)
			if err := verifyExactDirectoryHandle(stage.handle, bundle, ops); err == nil {
				t.Fatal("physical mismatch was accepted as the exact contract")
			}
		})
	}
}

func TestPostRenameAmbiguityReturnsZeroReceiptAndRetriesExactly(t *testing.T) {
	bundle := materializerTestBundle(t)
	for _, phase := range []string{
		"renamed-stage-sync",
		"published-parent-sync",
		"first-final-open",
		"first-final-close",
		"second-final-open",
		"second-final-close",
		"published-stage-close",
		"published-parent-close",
	} {
		t.Run(phase, func(t *testing.T) {
			parentPath := materializerResolvedTempDir(t)
			destination := filepath.Join(parentPath, "contract")
			sentinel := errors.New("injected " + phase)
			controller := &materializerPipelineFault{
				phase: phase, destinationName: filepath.Base(destination), sentinel: sentinel,
			}
			receipt, err := materializeAuthorized(
				context.Background(), materializerTestSource(bundle), destination, controller.operations(defaultOperations()),
			)
			if !controller.renamed || !controller.faulted {
				t.Fatalf("fault trace did not cross successful rename: renamed=%t faulted=%t", controller.renamed, controller.faulted)
			}
			if !IsCode(err, CodeExportAmbiguous) || !errors.Is(err, sentinel) ||
				receipt != (MaterializedContract{}) || receipt.Valid() || receipt.Destination() != "" ||
				receipt.BundleDigest().Valid() || receipt.ResidueHeadDigest().Valid() || receipt.Disposition() != "" {
				t.Fatalf("post-rename fault result = %#v, %v", receipt, err)
			}
			assertMaterializerExactOutput(t, destination, bundle)
			if stages := materializerStageNames(t, parentPath); len(stages) != 0 {
				t.Fatalf("post-rename ambiguity left private stages: %v", stages)
			}
			before := materializerOutputSnapshot(t, destination)
			retry, retryErr := materializeAuthorized(
				context.Background(), materializerTestSource(bundle), destination, defaultOperations(),
			)
			if retryErr != nil || !retry.Valid() || retry.Disposition() != AlreadyExact ||
				retry.State() != StateAlreadyExact || retry.BundleDigest() != bundle.Digest() {
				t.Fatalf("exact ambiguity retry = %#v, %v", retry, retryErr)
			}
			after := materializerOutputSnapshot(t, destination)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("exact ambiguity retry changed output: before=%v after=%v", before, after)
			}
		})
	}
}

func TestCancellationRefusesBeforeRenameAndReconcilesAfterRename(t *testing.T) {
	bundle := materializerTestBundle(t)

	t.Run("cancel-after-stage-sync-refuses-and-cleans", func(t *testing.T) {
		parentPath := materializerResolvedTempDir(t)
		destination := filepath.Join(parentPath, "contract")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		base := defaultOperations()
		ops := base
		cancelled := false
		ops.openDirectoryAt = func(parent durableDirectory, name string) (durableDirectory, error) {
			handle, err := base.openDirectoryAt(parent, name)
			if err != nil || !strings.HasPrefix(name, stageNamePrefix) {
				return handle, err
			}
			return &materializerCancelAfterSyncDirectory{
				durableDirectory: handle,
				cancel: func() {
					cancelled = true
					cancel()
				},
			}, nil
		}

		receipt, err := materializeAuthorized(ctx, materializerTestSource(bundle), destination, ops)
		if !cancelled || receipt != (MaterializedContract{}) || !IsCode(err, CodeExportIncomplete) ||
			!errors.Is(err, context.Canceled) {
			t.Fatalf("pre-rename cancellation = %#v, %v; cancelled=%t", receipt, err, cancelled)
		}
		if _, statErr := os.Lstat(destination); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("pre-rename cancellation published a destination: %v", statErr)
		}
		if stages := materializerStageNames(t, parentPath); len(stages) != 0 {
			t.Fatalf("pre-rename cancellation left private stages: %v", stages)
		}
	})

	t.Run("cancel-after-exclusive-rename-still-reconciles", func(t *testing.T) {
		parentPath := materializerResolvedTempDir(t)
		destination := filepath.Join(parentPath, "contract")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		base := defaultOperations()
		ops := base
		cancelled := false
		ops.renameExclusiveAt = func(parent durableDirectory, from, to string) error {
			if err := base.renameExclusiveAt(parent, from, to); err != nil {
				return err
			}
			cancelled = true
			cancel()
			return nil
		}

		receipt, err := materializeAuthorized(ctx, materializerTestSource(bundle), destination, ops)
		if !cancelled || !errors.Is(ctx.Err(), context.Canceled) || err != nil || !receipt.Valid() ||
			receipt.Disposition() != Created || receipt.Destination() != destination ||
			receipt.BundleDigest() != bundle.Digest() {
			t.Fatalf("post-rename cancellation = %#v, %v; cancelled=%t", receipt, err, cancelled)
		}
		assertMaterializerExactOutput(t, destination, bundle)
		if stages := materializerStageNames(t, parentPath); len(stages) != 0 {
			t.Fatalf("post-rename cancellation left private stages: %v", stages)
		}
	})
}

func TestExclusiveRenameFailureDistinguishesProvenNoEffectFromUnknownEffect(t *testing.T) {
	bundle := materializerTestBundle(t)
	for _, test := range []struct {
		name      string
		afterMove bool
		wantCode  string
		wantDest  bool
	}{
		{"failure-before-effect", false, CodeExportIncomplete, false},
		{"error-after-effect-and-inspection-failure", true, CodeExportAmbiguous, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			parentPath := materializerResolvedTempDir(t)
			destination := filepath.Join(parentPath, "contract")
			sentinel := errors.New("injected rename outcome")
			controller := &materializerPipelineFault{
				phase: "rename-return-error", destinationName: filepath.Base(destination), sentinel: sentinel,
				renameAfterEffect: test.afterMove,
			}
			receipt, err := materializeAuthorized(
				context.Background(), materializerTestSource(bundle), destination, controller.operations(defaultOperations()),
			)
			if receipt != (MaterializedContract{}) || !IsCode(err, test.wantCode) || !errors.Is(err, sentinel) {
				t.Fatalf("rename fault result = %#v, %v; want %s", receipt, err, test.wantCode)
			}
			_, statErr := os.Lstat(destination)
			if test.wantDest {
				if statErr != nil {
					t.Fatalf("effectful rename did not leave destination: %v", statErr)
				}
				assertMaterializerExactOutput(t, destination, bundle)
			} else if !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("no-effect rename left destination: %v", statErr)
			}
			if stages := materializerStageNames(t, parentPath); len(stages) != 0 {
				t.Fatalf("rename fault left private stages: %v", stages)
			}
			retry, retryErr := materializeAuthorized(
				context.Background(), materializerTestSource(bundle), destination, defaultOperations(),
			)
			wantDisposition := Created
			if test.wantDest {
				wantDisposition = AlreadyExact
			}
			if retryErr != nil || !retry.Valid() || retry.Disposition() != wantDisposition {
				t.Fatalf("rename fault retry = %#v, %v; want %s", retry, retryErr, wantDisposition)
			}
		})
	}
}

func TestExistingOutputObservationFailureIsAmbiguousAndRetryable(t *testing.T) {
	bundle := materializerTestBundle(t)
	parentPath := materializerResolvedTempDir(t)
	destination := filepath.Join(parentPath, "contract")
	created, err := materializeAuthorized(
		context.Background(), materializerTestSource(bundle), destination, defaultOperations(),
	)
	if err != nil || !created.Valid() || created.Disposition() != Created {
		t.Fatalf("initial materialization = %#v, %v", created, err)
	}
	before := materializerOutputSnapshot(t, destination)

	sentinel := errors.New("injected existing-output sync observation failure")
	base := defaultOperations()
	ops := base
	intercepted := false
	ops.openFileAt = func(
		directory durableDirectory,
		name string,
		flags int,
		mode os.FileMode,
	) (durableFile, error) {
		handle, openErr := base.openFileAt(directory, name, flags, mode)
		if openErr != nil || intercepted || flags != os.O_RDONLY {
			return handle, openErr
		}
		intercepted = true
		return &materializerSyncFaultFile{durableFile: handle, sentinel: sentinel}, nil
	}
	receipt, err := materializeAuthorized(
		context.Background(), materializerTestSource(bundle), destination, ops,
	)
	if !intercepted || receipt != (MaterializedContract{}) || !IsCode(err, CodeExportAmbiguous) ||
		!errors.Is(err, sentinel) {
		t.Fatalf("existing-output observation fault = %#v, %v; intercepted=%t", receipt, err, intercepted)
	}
	afterFault := materializerOutputSnapshot(t, destination)
	if !reflect.DeepEqual(before, afterFault) {
		t.Fatalf("observation fault changed exact output: before=%v after=%v", before, afterFault)
	}

	retry, retryErr := materializeAuthorized(
		context.Background(), materializerTestSource(bundle), destination, defaultOperations(),
	)
	if retryErr != nil || !retry.Valid() || retry.Disposition() != AlreadyExact {
		t.Fatalf("existing-output observation retry = %#v, %v", retry, retryErr)
	}
	if afterRetry := materializerOutputSnapshot(t, destination); !reflect.DeepEqual(before, afterRetry) {
		t.Fatalf("existing-output retry changed exact output: before=%v after=%v", before, afterRetry)
	}
}

func TestDestinationAppearanceAtExclusiveBoundaryConvergesOrRefusesImmutably(t *testing.T) {
	bundle := materializerTestBundle(t)
	for _, mismatch := range []bool{false, true} {
		name := "exact"
		if mismatch {
			name = "mismatch"
		}
		t.Run(name, func(t *testing.T) {
			parentPath := materializerResolvedTempDir(t)
			destination := filepath.Join(parentPath, "contract")
			base := defaultOperations()
			parent, err := retainParent(parentPath, base)
			if err != nil {
				t.Fatal(err)
			}
			winner, err := allocateStage(parent.handle, base)
			if err != nil {
				t.Fatal(err)
			}
			if err := writeExactBundle(context.Background(), winner.handle, bundle, base); err != nil {
				t.Fatal(err)
			}
			if err := verifyExactDirectoryHandle(winner.handle, bundle, base); err != nil {
				t.Fatal(err)
			}
			if err := winner.handle.Sync(); err != nil {
				t.Fatal(err)
			}
			if err := winner.handle.Close(); err != nil {
				t.Fatal(err)
			}
			if err := parent.handle.Close(); err != nil {
				t.Fatal(err)
			}
			if mismatch {
				member := bundle.Files()[0]
				changed := member.Content()
				changed[0] ^= 0xff
				if err := os.WriteFile(filepath.Join(parentPath, winner.name, member.Path()), changed, contractFileMode); err != nil {
					t.Fatal(err)
				}
			}

			ops := base
			collisionInjected := false
			ops.renameExclusiveAt = func(parent durableDirectory, from, to string) error {
				if !collisionInjected {
					collisionInjected = true
					if moveErr := base.renameExclusiveAt(parent, winner.name, to); moveErr != nil {
						return moveErr
					}
				}
				return base.renameExclusiveAt(parent, from, to)
			}
			receipt, err := materializeAuthorized(
				context.Background(), materializerTestSource(bundle), destination, ops,
			)
			if !collisionInjected {
				t.Fatal("destination appearance was not injected at the exclusive boundary")
			}
			if mismatch {
				if receipt != (MaterializedContract{}) || !IsCode(err, CodeExportIncomplete) ||
					IsCode(err, CodeExportAmbiguous) {
					t.Fatalf("mismatched collision result = %#v, %v", receipt, err)
				}
			} else if err != nil || !receipt.Valid() || receipt.Disposition() != AlreadyExact {
				t.Fatalf("exact collision result = %#v, %v", receipt, err)
			}
			if stages := materializerStageNames(t, parentPath); len(stages) != 0 {
				t.Fatalf("destination collision left private stages: %v", stages)
			}
			beforeRetry := materializerOutputSnapshot(t, destination)
			retry, retryErr := materializeAuthorized(
				context.Background(), materializerTestSource(bundle), destination, defaultOperations(),
			)
			if mismatch {
				if retry != (MaterializedContract{}) || !IsCode(retryErr, CodeExportIncomplete) ||
					IsCode(retryErr, CodeExportAmbiguous) {
					t.Fatalf("mismatched collision retry = %#v, %v", retry, retryErr)
				}
			} else if retryErr != nil || !retry.Valid() || retry.Disposition() != AlreadyExact {
				t.Fatalf("exact collision retry = %#v, %v", retry, retryErr)
			}
			if afterRetry := materializerOutputSnapshot(t, destination); !reflect.DeepEqual(beforeRetry, afterRetry) {
				t.Fatalf("collision retry changed winner: before=%v after=%v", beforeRetry, afterRetry)
			}
		})
	}
}

type materializerPipelineFault struct {
	phase             string
	destinationName   string
	sentinel          error
	renamed           bool
	faulted           bool
	renameAfterEffect bool
	pathOpenCount     int
	destinationOpens  int
}

type materializerPipelineDirectory struct {
	durableDirectory
	controller *materializerPipelineFault
	role       string
}

func (d *materializerPipelineDirectory) Sync() error {
	if d.controller.renamed && !d.controller.faulted &&
		((d.role == "stage" && d.controller.phase == "renamed-stage-sync") ||
			(d.role == "parent" && d.controller.phase == "published-parent-sync")) {
		d.controller.faulted = true
		return d.controller.sentinel
	}
	return d.durableDirectory.Sync()
}

func (d *materializerPipelineDirectory) Close() error {
	closeErr := d.durableDirectory.Close()
	if d.controller.renamed && !d.controller.faulted &&
		((d.role == "stage" && d.controller.phase == "published-stage-close") ||
			(d.role == "parent" && d.controller.phase == "published-parent-close") ||
			(d.role == "final-1" && d.controller.phase == "first-final-close") ||
			(d.role == "final-2" && d.controller.phase == "second-final-close")) {
		d.controller.faulted = true
		return errors.Join(closeErr, d.controller.sentinel)
	}
	return closeErr
}

func (c *materializerPipelineFault) operations(base operations) operations {
	ops := base
	ops.openDirectoryPath = func(path string) (durableDirectory, error) {
		handle, err := base.openDirectoryPath(path)
		if err != nil {
			return nil, err
		}
		c.pathOpenCount++
		if c.pathOpenCount == 1 {
			return &materializerPipelineDirectory{durableDirectory: handle, controller: c, role: "parent"}, nil
		}
		return handle, nil
	}
	ops.openDirectoryAt = func(parent durableDirectory, name string) (durableDirectory, error) {
		if c.renamed && name == c.destinationName {
			c.destinationOpens++
			if c.phase == "rename-return-error" && c.renameAfterEffect && !c.faulted {
				c.faulted = true
				return nil, c.sentinel
			}
			if !c.faulted && ((c.destinationOpens == 1 && c.phase == "first-final-open") ||
				(c.destinationOpens == 2 && c.phase == "second-final-open")) {
				c.faulted = true
				return nil, c.sentinel
			}
		}
		handle, err := base.openDirectoryAt(parent, name)
		if err != nil {
			return handle, err
		}
		if strings.HasPrefix(name, stageNamePrefix) {
			return &materializerPipelineDirectory{durableDirectory: handle, controller: c, role: "stage"}, nil
		}
		if c.renamed && name == c.destinationName {
			return &materializerPipelineDirectory{
				durableDirectory: handle, controller: c, role: fmt.Sprintf("final-%d", c.destinationOpens),
			}, nil
		}
		return handle, nil
	}
	ops.renameExclusiveAt = func(parent durableDirectory, from, to string) error {
		if c.phase == "rename-return-error" {
			if c.renameAfterEffect {
				if err := base.renameExclusiveAt(parent, from, to); err != nil {
					return err
				}
				c.renamed = true
			}
			return c.sentinel
		}
		if err := base.renameExclusiveAt(parent, from, to); err != nil {
			return err
		}
		c.renamed = true
		return nil
	}
	return ops
}

type materializerFaultFile struct {
	body         bytes.Buffer
	maximumWrite int
	zeroProgress bool
	writeErr     error
	chmodErr     error
	syncErr      error
	closeErr     error
	chmodCalls   int
	syncCalls    int
	closeCalls   int
}

type materializerSyncFaultFile struct {
	durableFile
	sentinel error
	faulted  bool
}

type materializerCancelAfterSyncDirectory struct {
	durableDirectory
	cancel    context.CancelFunc
	cancelled bool
}

func (d *materializerCancelAfterSyncDirectory) Sync() error {
	if err := d.durableDirectory.Sync(); err != nil {
		return err
	}
	if !d.cancelled {
		d.cancelled = true
		d.cancel()
	}
	return nil
}

func (f *materializerSyncFaultFile) Sync() error {
	baseErr := f.durableFile.Sync()
	if f.faulted {
		return baseErr
	}
	f.faulted = true
	return errors.Join(baseErr, f.sentinel)
}

type materializerFaultDirectory struct {
	durableDirectory
	chmodErr  error
	statErr   error
	statErrAt int
	statCalls int
	closeErr  error
}

func (d *materializerFaultDirectory) Stat() (os.FileInfo, error) {
	d.statCalls++
	if d.statErr != nil && d.statCalls == d.statErrAt {
		return nil, d.statErr
	}
	return d.durableDirectory.Stat()
}

func (d *materializerFaultDirectory) Chmod(mode os.FileMode) error {
	if d.chmodErr != nil {
		return d.chmodErr
	}
	return d.durableDirectory.Chmod(mode)
}

func (d *materializerFaultDirectory) Close() error {
	return errors.Join(d.durableDirectory.Close(), d.closeErr)
}

func (f *materializerFaultFile) Read([]byte) (int, error) { return 0, io.EOF }
func (f *materializerFaultFile) Write(body []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	if f.zeroProgress {
		return 0, nil
	}
	limit := len(body)
	if f.maximumWrite > 0 && limit > f.maximumWrite {
		limit = f.maximumWrite
	}
	return f.body.Write(body[:limit])
}
func (f *materializerFaultFile) Stat() (os.FileInfo, error) { return nil, errors.New("unused") }
func (f *materializerFaultFile) Chmod(os.FileMode) error {
	f.chmodCalls++
	return f.chmodErr
}
func (f *materializerFaultFile) Sync() error {
	f.syncCalls++
	return f.syncErr
}
func (f *materializerFaultFile) Close() error {
	f.closeCalls++
	return f.closeErr
}

func materializerTestBundle(t testing.TB) model.ContractBundle {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("materializer test source path is unavailable")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(repositoryRoot, "spec", "examples", "v1", "contract-bundle.valid.json"))
	if err != nil {
		t.Fatalf("read checked-in materializer fixture: %v", err)
	}
	canonical, err := canon.Canonicalize(raw)
	if err != nil {
		t.Fatalf("canonicalize checked-in materializer fixture: %v", err)
	}
	digestRaw, err := canon.DigestBytes(model.BundleDigestDomain, canonical)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := domain.ParseDigest(digestRaw.String())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := model.ParseContractBundle(canonical, digest)
	if err != nil || !bundle.Valid() {
		t.Fatalf("parse deterministic materializer fixture: %v", err)
	}
	return bundle
}

func materializerTestSource(bundle model.ContractBundle) materializationSource {
	snapshot := materializationSnapshot{
		bundle: bundle, bundleDigest: bundle.Digest(), headDigest: bundle.Digest(),
	}
	return materializationSource{
		snapshot: snapshot,
		revalidate: func(ctx context.Context) (materializationSnapshot, error) {
			if err := contextRefusal(ctx); err != nil {
				return materializationSnapshot{}, err
			}
			return snapshot, nil
		},
		validatePath: func(ctx context.Context, _ string) error { return contextRefusal(ctx) },
	}
}

func assertMaterializerExactOutput(t testing.TB, destination string, bundle model.ContractBundle) {
	t.Helper()
	ops := defaultOperations()
	parent, err := retainParent(filepath.Dir(destination), ops)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.handle.Close()
	handle, err := requireNamedDirectory(parent.handle, filepath.Base(destination), ops)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if err := verifyExactDirectoryHandle(handle, bundle, ops); err != nil {
		t.Fatalf("materialized output is not exact: %v", err)
	}
}

func materializerOutputSnapshot(t testing.TB, destination string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(destination)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := make(map[string]string, len(entries)+1)
	paths := append([]string{destination}, func() []string {
		result := make([]string, len(entries))
		for index, entry := range entries {
			result[index] = filepath.Join(destination, entry.Name())
		}
		return result
	}()...)
	for _, path := range paths {
		info, statErr := os.Lstat(path)
		if statErr != nil {
			t.Fatal(statErr)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatalf("snapshot path has no Darwin stat: %s", path)
		}
		body := []byte(nil)
		if info.Mode().IsRegular() {
			body, statErr = os.ReadFile(path)
			if statErr != nil {
				t.Fatal(statErr)
			}
		}
		digest := sha256.Sum256(body)
		snapshot[filepath.Base(path)] = fmt.Sprintf(
			"%d:%d:%#o:%d:%x", stat.Dev, stat.Ino, info.Mode(), info.Size(), digest,
		)
	}
	return snapshot
}

func materializerResolvedTempDir(t testing.TB) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		t.Fatalf("resolved materializer temp directory is not clean and absolute: %q", root)
	}
	return root
}

func materializerShortTempDir(t testing.TB) string {
	t.Helper()
	root, err := os.MkdirTemp("/private/tmp", "c-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		_ = os.RemoveAll(root)
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}

func materializerStageNames(t testing.TB, parent string) []string {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), stageNamePrefix) {
			names = append(names, entry.Name())
		}
	}
	return names
}
