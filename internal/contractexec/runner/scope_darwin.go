//go:build darwin

package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nelsonwerd/countershape/internal/store"
)

// This digest enrolls one reviewed, dependency-free reference subject. It is
// deliberately a narrow fixture identity, not a generic Node completeness
// assertion. The matching source lives in testkit/contractexec/cli.
const (
	referenceRoleSHA256Hex    = "1ca8c9b4cbe776260b563dceb8a9cad36e3add9c1bcf6f577067c95a427a381e"
	referenceSubjectSHA256Hex = "44b88ba6108a277f459cc57d3cd8adb2a91121b2edf4ce8e669bdabb1ede145d"
)

const importCanaryModuleSource = `import { connect } from "node:net";
const socket = process.env.COUNTERSHAPE_C4_IMPORT_CANARY_SOCKET;
if (typeof socket !== "string" || socket.length === 0) throw new Error("missing import canary socket");
await new Promise((resolve, reject) => {
  const client = connect(socket, () => { client.end(); resolve(); });
  client.once("error", reject);
});
export const importedOutsideCandidate = true;
`

const (
	shortSocketParent            = "/private/tmp"
	maxDarwinUnixSocketPathBytes = 103
	candidateInventoryReadBatch  = 128

	maxCandidateInventoryEntries        = 20_000
	maxCandidateInventoryPathBytes      = 32 * 1024 * 1024
	maxCandidateInventoryDepth          = 128
	maxCandidateInventoryFileBytes      = 64 * 1024 * 1024
	maxCandidateInventoryAggregateBytes = 64 * 1024 * 1024

	// Darwin's O_NOFOLLOW only protects the final path component. The native
	// O_NOFOLLOW_ANY extension rejects a symlink in any component, which keeps
	// descriptor opens confined to the exact candidate tree even if a parent is
	// exchanged while the bounded inventory is running.
	darwinOpenNoFollowAny = 0x20000000
)

type inventoryLimits struct {
	entries        int
	pathBytes      int64
	depth          int
	fileBytes      int64
	aggregateBytes int64
	readBatch      int
}

type inventoryEntry struct {
	Path   string
	Mode   string
	Size   int64
	SHA256 string
}

type inventorySnapshot struct {
	entries  []inventoryEntry
	identity string
}

func (snapshot inventorySnapshot) valid() bool {
	return len(snapshot.entries) > 0 && len(snapshot.identity) == sha256.Size*2
}

func (snapshot inventorySnapshot) equal(other inventorySnapshot) bool {
	if !snapshot.valid() || !other.valid() || snapshot.identity != other.identity || len(snapshot.entries) != len(other.entries) {
		return false
	}
	for index := range snapshot.entries {
		if snapshot.entries[index] != other.entries[index] {
			return false
		}
	}
	return true
}

func (snapshot inventorySnapshot) referenceEnrolled() bool {
	if !snapshot.valid() || len(snapshot.entries) != 2 {
		return false
	}
	role := snapshot.entries[0]
	subject := snapshot.entries[1]
	return role.Path == "candidate-role.json" && role.Mode == "100644" && role.Size == 27 && role.SHA256 == referenceRoleSHA256Hex &&
		subject.Path == "fixture.mjs" && subject.Mode == "100755" && subject.Size == 8138 &&
		subject.SHA256 == referenceSubjectSHA256Hex
}

func snapshotCandidate(root string) (inventorySnapshot, error) {
	return snapshotCandidateWithLimits(root, inventoryLimits{
		entries: maxCandidateInventoryEntries, pathBytes: maxCandidateInventoryPathBytes,
		depth: maxCandidateInventoryDepth, fileBytes: maxCandidateInventoryFileBytes,
		aggregateBytes: maxCandidateInventoryAggregateBytes, readBatch: candidateInventoryReadBatch,
	})
}

type candidateInventoryDirectory struct {
	path     string
	relative string
	depth    int
}

type candidateInventoryState struct {
	entries        []inventoryEntry
	pathBytes      int64
	aggregateBytes int64
}

func (limits inventoryLimits) valid() bool {
	return limits.entries > 0 && limits.pathBytes > 0 && limits.depth > 0 &&
		limits.fileBytes > 0 && limits.aggregateBytes > 0 &&
		limits.fileBytes <= limits.aggregateBytes && limits.readBatch > 0 && limits.readBatch <= 4096
}

func snapshotCandidateWithLimits(root string, limits inventoryLimits) (inventorySnapshot, error) {
	if !limits.valid() {
		return inventorySnapshot{}, errors.New("candidate inventory limits are invalid")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return inventorySnapshot{}, errors.New("candidate root is not clean and absolute")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 ||
		rootInfo.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return inventorySnapshot{}, errors.Join(err, errors.New("candidate root is not an exact directory"))
	}
	state := candidateInventoryState{entries: make([]inventoryEntry, 0, 16)}
	directories := []candidateInventoryDirectory{{path: root}}
	for len(directories) > 0 {
		directory := directories[0]
		directories = directories[1:]
		children, readErr := readCandidateInventoryDirectory(root, directory, limits, &state)
		if readErr != nil {
			return inventorySnapshot{}, readErr
		}
		directories = append(directories, children...)
	}
	afterRoot, rootErr := os.Lstat(root)
	if rootErr != nil || !afterRoot.IsDir() || afterRoot.Mode() != rootInfo.Mode() || !os.SameFile(rootInfo, afterRoot) {
		return inventorySnapshot{}, errors.Join(rootErr, errors.New("candidate root changed during inventory"))
	}
	if len(state.entries) == 0 {
		return inventorySnapshot{}, errors.New("candidate inventory is empty")
	}
	sort.Slice(state.entries, func(left, right int) bool { return state.entries[left].Path < state.entries[right].Path })
	digest := sha256.New()
	for _, entry := range state.entries {
		digest.Write([]byte(entry.Path))
		digest.Write([]byte{0})
		digest.Write([]byte(entry.Mode))
		digest.Write([]byte{0})
		digest.Write([]byte(entry.SHA256))
		digest.Write([]byte{0})
		var size [8]byte
		for index := 7; index >= 0; index-- {
			size[index] = byte(entry.Size)
			entry.Size >>= 8
		}
		digest.Write(size[:])
	}
	return inventorySnapshot{entries: state.entries, identity: hex.EncodeToString(digest.Sum(nil))}, nil
}

func readCandidateInventoryDirectory(
	root string,
	directory candidateInventoryDirectory,
	limits inventoryLimits,
	state *candidateInventoryState,
) ([]candidateInventoryDirectory, error) {
	before, err := os.Lstat(directory.path)
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 ||
		before.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return nil, errors.Join(err, errors.New("candidate inventory directory is not exact"))
	}
	descriptor, err := syscall.Open(
		directory.path,
		syscall.O_RDONLY|syscall.O_DIRECTORY|darwinOpenNoFollowAny|syscall.O_CLOEXEC|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return nil, err
	}
	handle := os.NewFile(uintptr(descriptor), directory.path)
	if handle == nil {
		_ = syscall.Close(descriptor)
		return nil, errors.New("candidate inventory directory descriptor could not be owned")
	}
	opened, openErr := handle.Stat()
	if openErr != nil || !opened.IsDir() || opened.Mode() != before.Mode() || !os.SameFile(before, opened) {
		return nil, errors.Join(openErr, handle.Close(), errors.New("candidate inventory directory descriptor differs"))
	}
	children := make([]candidateInventoryDirectory, 0)
	for {
		batch, readErr := handle.ReadDir(limits.readBatch)
		for _, entry := range batch {
			name := entry.Name()
			if name == "" || name == "." || name == ".." || strings.ContainsRune(name, filepath.Separator) {
				return nil, errors.Join(handle.Close(), errors.New("candidate inventory directory returned an invalid name"))
			}
			relative := name
			if directory.relative != "" {
				relative = filepath.Join(directory.relative, name)
			}
			relative = filepath.ToSlash(relative)
			depth := directory.depth + 1
			if depth > limits.depth {
				return nil, errors.Join(handle.Close(), errors.New("candidate inventory exceeds the depth limit"))
			}
			if len(state.entries) >= limits.entries {
				return nil, errors.Join(handle.Close(), errors.New("candidate inventory exceeds the entry limit"))
			}
			pathBytes := int64(len([]byte(relative)))
			if pathBytes > limits.pathBytes-state.pathBytes {
				return nil, errors.Join(handle.Close(), errors.New("candidate inventory exceeds the path-byte limit"))
			}
			path := filepath.Join(root, filepath.FromSlash(relative))
			listed, infoErr := entry.Info()
			info, statErr := os.Lstat(path)
			if infoErr != nil || statErr != nil || info.Mode()&os.ModeSymlink != 0 ||
				info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || !os.SameFile(listed, info) {
				return nil, errors.Join(infoErr, statErr, handle.Close(), errors.New("candidate inventory contains a changed or unreadable entry"))
			}
			item := inventoryEntry{Path: relative}
			switch {
			case info.IsDir():
				item.Mode = "040" + modeText(info.Mode().Perm())
				children = append(children, candidateInventoryDirectory{path: path, relative: relative, depth: depth})
			case info.Mode().IsRegular():
				if info.Size() < 0 || info.Size() > limits.fileBytes {
					return nil, errors.Join(handle.Close(), errors.New("candidate inventory exceeds the single-file byte limit"))
				}
				if info.Size() > limits.aggregateBytes-state.aggregateBytes {
					return nil, errors.Join(handle.Close(), errors.New("candidate inventory exceeds the aggregate byte limit"))
				}
				item.Mode = "100" + modeText(info.Mode().Perm())
				item.Size = info.Size()
				item.SHA256, err = digestCandidateInventoryFile(path, info, limits.fileBytes)
				if err != nil {
					return nil, errors.Join(err, handle.Close())
				}
				state.aggregateBytes += info.Size()
			default:
				return nil, errors.Join(handle.Close(), errors.New("candidate inventory contains a special entry"))
			}
			state.pathBytes += pathBytes
			state.entries = append(state.entries, item)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, errors.Join(readErr, handle.Close(), errors.New("candidate inventory directory read failed"))
		}
		if len(batch) == 0 {
			return nil, errors.Join(handle.Close(), io.ErrNoProgress)
		}
	}
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(directory.path)
	if descriptorErr != nil || closeErr != nil || pathErr != nil || !afterDescriptor.IsDir() ||
		afterDescriptor.Mode() != opened.Mode() || afterPath.Mode() != opened.Mode() ||
		!os.SameFile(opened, afterDescriptor) || !os.SameFile(opened, afterPath) {
		return nil, errors.Join(descriptorErr, closeErr, pathErr, errors.New("candidate inventory directory changed during traversal"))
	}
	return children, nil
}

func digestCandidateInventoryFile(path string, expected os.FileInfo, byteLimit int64) (string, error) {
	descriptor, err := syscall.Open(
		path,
		syscall.O_RDONLY|syscall.O_CLOEXEC|darwinOpenNoFollowAny|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return "", err
	}
	handle := os.NewFile(uintptr(descriptor), path)
	if handle == nil {
		_ = syscall.Close(descriptor)
		return "", errors.New("candidate inventory file descriptor could not be owned")
	}
	opened, openErr := handle.Stat()
	if openErr != nil || !opened.Mode().IsRegular() || opened.Mode() != expected.Mode() ||
		opened.Size() != expected.Size() || opened.Size() > byteLimit || !os.SameFile(expected, opened) {
		return "", errors.Join(openErr, handle.Close(), errors.New("candidate inventory file descriptor differs"))
	}
	digest := sha256.New()
	written, readErr := io.Copy(digest, io.LimitReader(handle, opened.Size()))
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(path)
	if readErr != nil || descriptorErr != nil || closeErr != nil || pathErr != nil ||
		written != opened.Size() || written > byteLimit || !afterPath.Mode().IsRegular() ||
		afterDescriptor.Mode() != opened.Mode() || afterPath.Mode() != opened.Mode() ||
		afterDescriptor.Size() != opened.Size() || afterPath.Size() != opened.Size() ||
		!os.SameFile(opened, afterDescriptor) || !os.SameFile(opened, afterPath) {
		return "", errors.Join(readErr, descriptorErr, closeErr, pathErr, errors.New("candidate inventory file changed during bounded read"))
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func modeText(mode os.FileMode) string {
	const digits = "01234567"
	return string([]byte{
		digits[(mode>>6)&7], digits[(mode>>3)&7], digits[mode&7],
	})
}

type unixCanary struct {
	path     string
	listener *net.UnixListener
	identity os.FileInfo
	accepted atomic.Int64
	done     chan error
	once     sync.Once
	closeErr error
}

func newUnixCanary(path string) (*unixCanary, error) {
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, err
	}
	listener.SetUnlinkOnClose(false)
	identity, identityErr := os.Lstat(path)
	if identityErr != nil || identity.Mode()&os.ModeSocket == 0 || identity.Mode()&os.ModeSymlink != 0 {
		closeErr := listener.Close()
		removeErr := os.Remove(path)
		if errors.Is(removeErr, os.ErrNotExist) {
			removeErr = nil
		}
		return nil, errors.Join(identityErr, closeErr, removeErr, errors.New("canary socket identity was not established"))
	}
	canary := &unixCanary{path: path, listener: listener, identity: identity, done: make(chan error, 1)}
	go func() {
		for {
			connection, err := listener.AcceptUnix()
			if err != nil {
				canary.done <- err
				return
			}
			canary.accepted.Add(1)
			_ = connection.Close()
		}
	}()
	return canary, nil
}

func (canary *unixCanary) close() error {
	if canary == nil {
		return nil
	}
	canary.once.Do(func() {
		currentIdentity, integrityErr := os.Lstat(canary.path)
		if integrityErr == nil && (currentIdentity.Mode()&os.ModeSocket == 0 || currentIdentity.Mode()&os.ModeSymlink != 0 ||
			!os.SameFile(canary.identity, currentIdentity)) {
			integrityErr = errors.New("canary socket identity changed before closure")
		}
		deadlineErr := canary.listener.SetDeadline(time.Now().Add(25 * time.Millisecond))
		acceptObserved := false
		var acceptErr error
		if deadlineErr == nil {
			select {
			case terminalErr := <-canary.done:
				acceptObserved = true
				acceptErr = classifyCanaryAcceptError(terminalErr, true, false)
			case <-time.After(100 * time.Millisecond):
				acceptErr = errors.New("canary accept loop did not stop at its deadline")
			}
		}
		closeErr := canary.listener.Close()
		if !acceptObserved {
			select {
			case terminalErr := <-canary.done:
				acceptErr = errors.Join(acceptErr, classifyCanaryAcceptError(terminalErr, false, true))
			case <-time.After(100 * time.Millisecond):
				acceptErr = errors.Join(acceptErr, errors.New("canary accept loop did not stop after listener close"))
			}
		}
		canary.closeErr = errors.Join(integrityErr, deadlineErr, closeErr, acceptErr)
	})
	return canary.closeErr
}

func classifyCanaryAcceptError(err error, deadlineArmed, listenerClosed bool) error {
	if err == nil {
		return errors.New("canary accept loop ended without a terminal error")
	}
	if deadlineArmed {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			return nil
		}
	}
	if listenerClosed && errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

type scopeProbe struct {
	root                 string
	rootIdentity         os.FileInfo
	socketRoot           string
	socketRootIdentity   os.FileInfo
	socketAlias          string
	socketAliasIdentity  os.FileInfo
	importModule         string
	importModuleIdentity os.FileInfo
	moduleSHA256         string
	importCanary         *unixCanary
	serviceCanary        *unixCanary
	once                 sync.Once
	measurements         scopeMeasurements
	err                  error
}

type scopeMeasurements struct {
	ImportAccepts  int64
	ServiceAccepts int64
	ModuleSHA256   string
	Ambiguous      bool
}

func newScopeProbe(roots store.ConformanceAttemptRoots) (*scopeProbe, error) {
	if roots.TemporaryRoot() == "" || roots.StateRoot() == "" {
		return nil, errors.New("attempt roots are unavailable")
	}
	parent := roots.TemporaryRoot()
	parentInfo, err := os.Lstat(parent)
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 ||
		parentInfo.Mode().Perm()&0o077 != 0 || parentInfo.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return nil, errors.Join(err, errors.New("attempt temporary root is not an owned private directory"))
	}
	root, err := os.MkdirTemp(parent, "scope-")
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		_ = os.Remove(root)
		return nil, err
	}
	rootIdentity, err := os.Lstat(root)
	if err != nil || !rootIdentity.IsDir() || rootIdentity.Mode().Perm() != 0o700 ||
		rootIdentity.Mode()&(os.ModeSymlink|os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		_ = os.Remove(root)
		return nil, errors.Join(err, errors.New("scope root identity was not established"))
	}
	socketRoot, socketAlias, err := newShortSocketRoot(root)
	if err != nil {
		_ = os.Remove(root)
		return nil, err
	}
	socketRootIdentity, socketRootErr := os.Lstat(socketRoot)
	socketAliasIdentity, socketAliasErr := os.Lstat(socketAlias)
	if socketRootErr != nil || socketAliasErr != nil || !socketRootIdentity.IsDir() ||
		socketRootIdentity.Mode().Perm() != 0o700 || socketAliasIdentity.Mode()&os.ModeSymlink == 0 {
		_ = os.Remove(socketAlias)
		_ = os.Remove(socketRoot)
		_ = os.Remove(root)
		return nil, errors.Join(socketRootErr, socketAliasErr, errors.New("short scope root identity was not established"))
	}
	probe := &scopeProbe{
		root: root, rootIdentity: rootIdentity,
		socketRoot: socketRoot, socketRootIdentity: socketRootIdentity,
		socketAlias: socketAlias, socketAliasIdentity: socketAliasIdentity,
	}
	cleanup := func(cause error) (*scopeProbe, error) {
		if probe.importCanary != nil {
			_ = probe.importCanary.close()
		}
		if probe.serviceCanary != nil {
			_ = probe.serviceCanary.close()
		}
		for _, path := range []string{
			filepath.Join(root, "i"), filepath.Join(root, "s"), probe.importModule, socketAlias,
		} {
			if path != "" {
				_ = os.Remove(path)
			}
		}
		_ = os.Remove(root)
		_ = os.Remove(socketRoot)
		return nil, cause
	}
	probe.importCanary, err = newUnixCanary(filepath.Join(socketAlias, "i"))
	if err != nil {
		return cleanup(err)
	}
	probe.serviceCanary, err = newUnixCanary(filepath.Join(socketAlias, "s"))
	if err != nil {
		return cleanup(err)
	}
	probe.importModule = filepath.Join(root, "outside-candidate.mjs")
	file, err := os.OpenFile(probe.importModule, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return cleanup(err)
	}
	body := []byte(importCanaryModuleSource)
	written, writeErr := file.Write(body)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil || written != len(body) {
		return cleanup(errors.Join(writeErr, syncErr, closeErr, errors.New("import canary module write was incomplete")))
	}
	probe.importModuleIdentity, err = os.Lstat(probe.importModule)
	if err != nil || !probe.importModuleIdentity.Mode().IsRegular() || probe.importModuleIdentity.Mode().Perm() != 0o600 {
		return cleanup(errors.Join(err, errors.New("import canary module identity was not established")))
	}
	digest := sha256.Sum256(body)
	probe.moduleSHA256 = hex.EncodeToString(digest[:])
	return probe, nil
}

// Darwin limits sockaddr_un paths independently of filesystem path limits.
// The attempt root is intentionally allowed to be arbitrarily deep, so only
// the two transient canary sockets are addressed through a fresh, exact-mode
// alias directory under the platform's root-owned sticky temporary directory.
// The alias resolves back into the attempt-private probe root, so the socket
// objects and module remain owned by the attempt while the kernel sees a short
// sockaddr_un spelling.
func newShortSocketRoot(attemptPrivateRoot string) (string, string, error) {
	if !filepath.IsAbs(attemptPrivateRoot) || filepath.Clean(attemptPrivateRoot) != attemptPrivateRoot {
		return "", "", errors.New("attempt-private scope root is not clean and absolute")
	}
	parentInfo, err := os.Lstat(shortSocketParent)
	parentStat, statOK := parentInfoSyscall(parentInfo)
	if err != nil || !statOK || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 ||
		parentInfo.Mode().Perm() != 0o777 || parentInfo.Mode()&os.ModeSticky == 0 ||
		parentInfo.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 || parentStat.Uid != 0 {
		return "", "", errors.Join(err, errors.New("Darwin short-socket parent is not the root-owned sticky directory"))
	}
	root, err := os.MkdirTemp(shortSocketParent, "cs4-")
	if err != nil {
		return "", "", err
	}
	cleanup := true
	alias := ""
	defer func() {
		if cleanup {
			if alias != "" {
				_ = os.Remove(alias)
			}
			_ = os.Remove(root)
		}
	}()
	if err := os.Chmod(root, 0o700); err != nil {
		return "", "", err
	}
	info, err := os.Lstat(root)
	stat, statOK := parentInfoSyscall(info)
	if err != nil || !statOK || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != 0o700 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 ||
		stat.Uid != uint32(os.Getuid()) {
		return "", "", errors.Join(err, errors.New("Darwin short-socket root is not an exact caller-owned private directory"))
	}
	alias = filepath.Join(root, "r")
	if err := os.Symlink(attemptPrivateRoot, alias); err != nil {
		return "", "", err
	}
	linked, err := os.Readlink(alias)
	resolved, resolveErr := filepath.EvalSymlinks(alias)
	if err != nil || resolveErr != nil || linked != attemptPrivateRoot || resolved != attemptPrivateRoot {
		return "", "", errors.Join(err, resolveErr, errors.New("Darwin short-socket alias did not resolve exactly"))
	}
	for _, name := range []string{"i", "s"} {
		if len([]byte(filepath.Join(alias, name))) > maxDarwinUnixSocketPathBytes {
			return "", "", errors.New("Darwin short-socket alias exceeds sockaddr_un capacity")
		}
	}
	cleanup = false
	return root, alias, nil
}

func parentInfoSyscall(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
}

func boundedDirectRoster(
	ctx context.Context,
	root string,
	retained os.FileInfo,
	expected []string,
) error {
	if ctx == nil || !filepath.IsAbs(root) || filepath.Clean(root) != root || retained == nil ||
		!retained.IsDir() || retained.Mode()&os.ModeSymlink != 0 || len(expected) > 16 {
		return errors.New("bounded direct-roster inputs are invalid")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	want := append([]string(nil), expected...)
	for _, name := range want {
		if name == "" || name == "." || name == ".." || strings.ContainsRune(name, filepath.Separator) {
			return errors.New("bounded direct-roster expectation contains an invalid name")
		}
	}
	sort.Strings(want)
	for index := 1; index < len(want); index++ {
		if want[index-1] == want[index] {
			return errors.New("bounded direct-roster expectation repeats a name")
		}
	}
	before, err := os.Lstat(root)
	if err != nil || !before.IsDir() || before.Mode() != retained.Mode() || !os.SameFile(retained, before) {
		return errors.Join(err, errors.New("bounded direct-roster root identity differs"))
	}
	descriptor, err := syscall.Open(
		root,
		syscall.O_RDONLY|syscall.O_DIRECTORY|darwinOpenNoFollowAny|syscall.O_CLOEXEC|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return err
	}
	handle := os.NewFile(uintptr(descriptor), root)
	if handle == nil {
		_ = syscall.Close(descriptor)
		return errors.New("bounded direct-roster descriptor could not be owned")
	}
	opened, openErr := handle.Stat()
	if openErr != nil || !opened.IsDir() || opened.Mode() != retained.Mode() || !os.SameFile(retained, opened) {
		return errors.Join(openErr, handle.Close(), errors.New("bounded direct-roster descriptor identity differs"))
	}
	allowed := make(map[string]struct{}, len(want))
	for _, name := range want {
		allowed[name] = struct{}{}
	}
	observed := make([]string, 0, len(want)+1)
	var rosterErr error
	for {
		if err := ctx.Err(); err != nil {
			rosterErr = err
			break
		}
		remaining := len(want) + 1 - len(observed)
		if remaining <= 0 {
			rosterErr = errors.New("bounded direct-roster cardinality differs")
			break
		}
		batch, readErr := handle.ReadDir(remaining)
		for _, entry := range batch {
			name := entry.Name()
			observed = append(observed, name)
			if _, present := allowed[name]; !present {
				rosterErr = errors.New("bounded direct-roster names differ")
				break
			}
		}
		if rosterErr != nil || len(observed) > len(want) {
			if rosterErr == nil {
				rosterErr = errors.New("bounded direct-roster cardinality differs")
			}
			break
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				rosterErr = readErr
			}
			break
		}
		if len(batch) == 0 {
			rosterErr = errors.New("bounded direct-roster made no progress")
			break
		}
	}
	sort.Strings(observed)
	syncErr := handle.Sync()
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(root)
	if rosterErr != nil || syncErr != nil || descriptorErr != nil || closeErr != nil || pathErr != nil ||
		!opened.IsDir() || afterDescriptor.Mode() != opened.Mode() || afterPath.Mode() != opened.Mode() ||
		!os.SameFile(opened, afterDescriptor) || !os.SameFile(opened, afterPath) {
		return errors.Join(rosterErr, syncErr, descriptorErr, closeErr, pathErr, errors.New("bounded direct-roster read did not close exactly"))
	}
	if len(observed) != len(want) {
		return errors.New("bounded direct-roster cardinality differs")
	}
	for index := range want {
		if observed[index] != want[index] {
			return errors.New("bounded direct-roster names differ")
		}
	}
	return ctx.Err()
}

func removeRetainedLeaf(ctx context.Context, path string, retained os.FileInfo) error {
	if ctx == nil || !filepath.IsAbs(path) || filepath.Clean(path) != path || retained == nil || retained.IsDir() {
		return errors.New("retained leaf removal inputs are invalid")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := os.Lstat(path)
	if err != nil || current.IsDir() || current.Mode() != retained.Mode() || !os.SameFile(retained, current) {
		return errors.Join(err, errors.New("retained leaf identity differs before removal"))
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.Join(err, errors.New("retained leaf remains after removal"))
	}
	return ctx.Err()
}

func removeRetainedEmptyDirectory(ctx context.Context, path string, retained os.FileInfo) error {
	if err := boundedDirectRoster(ctx, path, retained, nil); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.Join(err, errors.New("retained empty directory remains after removal"))
	}
	return ctx.Err()
}

func (probe *scopeProbe) finish(ctx context.Context) (scopeMeasurements, error) {
	if probe == nil {
		return scopeMeasurements{}, errors.New("scope probe is absent")
	}
	probe.once.Do(func() {
		rootRosterErr := boundedDirectRoster(
			ctx,
			probe.root,
			probe.rootIdentity,
			[]string{filepath.Base(probe.importModule), filepath.Base(probe.importCanary.path), filepath.Base(probe.serviceCanary.path)},
		)
		shortRosterErr := boundedDirectRoster(
			ctx, probe.socketRoot, probe.socketRootIdentity, []string{filepath.Base(probe.socketAlias)},
		)
		moduleDigest, moduleErr := digestCandidateInventoryFile(
			probe.importModule, probe.importModuleIdentity, int64(len(importCanaryModuleSource)),
		)
		aliasTarget, aliasErr := os.Readlink(probe.socketAlias)
		aliasInfo, aliasInfoErr := os.Lstat(probe.socketAlias)
		if aliasErr == nil && aliasInfoErr == nil &&
			(aliasTarget != probe.root || aliasInfo.Mode() != probe.socketAliasIdentity.Mode() ||
				!os.SameFile(probe.socketAliasIdentity, aliasInfo)) {
			aliasErr = errors.New("scope socket alias identity or target changed")
		}
		if moduleErr == nil && moduleDigest != probe.moduleSHA256 {
			moduleErr = errors.New("scope import module bytes changed")
		}
		importErr := probe.importCanary.close()
		serviceErr := probe.serviceCanary.close()
		probe.measurements = scopeMeasurements{
			ImportAccepts:  probe.importCanary.accepted.Load(),
			ServiceAccepts: probe.serviceCanary.accepted.Load(),
			ModuleSHA256:   probe.moduleSHA256,
		}
		integrityErr := errors.Join(rootRosterErr, shortRosterErr, moduleErr, aliasErr, aliasInfoErr, importErr, serviceErr)
		if integrityErr == nil {
			removeImportErr := removeRetainedLeaf(ctx, probe.importCanary.path, probe.importCanary.identity)
			removeServiceErr := removeRetainedLeaf(ctx, probe.serviceCanary.path, probe.serviceCanary.identity)
			removeModuleErr := removeRetainedLeaf(ctx, probe.importModule, probe.importModuleIdentity)
			removeRootErr := errors.Join(removeImportErr, removeServiceErr, removeModuleErr)
			if removeRootErr == nil {
				removeRootErr = removeRetainedEmptyDirectory(ctx, probe.root, probe.rootIdentity)
			}
			removeAliasErr := removeRetainedLeaf(ctx, probe.socketAlias, probe.socketAliasIdentity)
			removeSocketRootErr := removeAliasErr
			if removeSocketRootErr == nil {
				removeSocketRootErr = removeRetainedEmptyDirectory(ctx, probe.socketRoot, probe.socketRootIdentity)
			}
			integrityErr = errors.Join(removeRootErr, removeSocketRootErr)
		}
		probe.err = integrityErr
		probe.measurements.Ambiguous = probe.err != nil
	})
	return probe.measurements, probe.err
}

func (probe *scopeProbe) close() error {
	_, err := probe.finish(context.Background())
	return err
}
