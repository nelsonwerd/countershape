//go:build darwin

package scope

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
)

const (
	ImportModuleEnvironment  = "COUNTERSHAPE_C5_IMPORT_CANARY_MODULE"
	ImportSocketEnvironment  = "COUNTERSHAPE_C5_IMPORT_CANARY_SOCKET"
	ServiceSocketEnvironment = "COUNTERSHAPE_C5_SERVICE_CANARY_SOCKET"

	shortSocketParent            = "/private/tmp"
	maxDarwinUnixSocketPathBytes = 103
)

const importModuleSource = `import { connect } from "node:net";
const socket = process.env.COUNTERSHAPE_C5_IMPORT_CANARY_SOCKET;
if (typeof socket !== "string" || socket.length === 0) throw new Error("missing import canary socket");
await new Promise((resolve, reject) => {
  const client = connect(socket, () => { client.end(); resolve(); });
  client.once("error", reject);
});
export const importedOutsideCandidate = true;
`

type canary struct {
	path     string
	listener *net.UnixListener
	identity os.FileInfo
	accepted atomic.Int64
	done     chan error
	once     sync.Once
	err      error
}

type Probe struct {
	root                string
	rootIdentity        os.FileInfo
	socketRoot          string
	socketRootIdentity  os.FileInfo
	socketAlias         string
	socketAliasIdentity os.FileInfo
	module              string
	moduleInfo          os.FileInfo
	moduleSHA256        string
	importCanary        *canary
	serviceCanary       *canary
	once                sync.Once
	measurements        Measurements
	err                 error
}

func NewProbe(privateTemporaryRoot string) (*Probe, error) {
	if !filepath.IsAbs(privateTemporaryRoot) || filepath.Clean(privateTemporaryRoot) != privateTemporaryRoot {
		return nil, fail(CodeProbeIntegrity, nil)
	}
	parentInfo, err := os.Lstat(privateTemporaryRoot)
	parentStat, parentStatOK := statIdentity(parentInfo)
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 ||
		parentInfo.Mode().Perm()&0o077 != 0 ||
		parentInfo.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 ||
		!parentStatOK || parentStat.Uid != uint32(os.Getuid()) {
		return nil, fail(CodeProbeIntegrity, err)
	}
	root, err := os.MkdirTemp(privateTemporaryRoot, "scope-")
	if err != nil {
		return nil, fail(CodeProbeIntegrity, err)
	}
	cleanup := true
	var probe *Probe
	defer func() {
		if cleanup {
			if probe != nil {
				_ = probe.importCanary.close()
				_ = probe.serviceCanary.close()
			}
			_ = os.Remove(filepath.Join(root, "outside-candidate.mjs"))
			if probe != nil {
				_ = os.Remove(filepath.Join(probe.socketAlias, "i"))
				_ = os.Remove(filepath.Join(probe.socketAlias, "s"))
				_ = os.Remove(probe.socketAlias)
				_ = os.Remove(probe.socketRoot)
			}
			_ = os.Remove(root)
		}
	}()
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fail(CodeProbeIntegrity, err)
	}
	rootInfo, err := os.Lstat(root)
	rootStat, rootStatOK := statIdentity(rootInfo)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode().Perm() != 0o700 ||
		rootInfo.Mode()&os.ModeSymlink != 0 ||
		rootInfo.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 ||
		!rootStatOK || rootStat.Uid != uint32(os.Getuid()) {
		return nil, fail(CodeProbeIntegrity, err)
	}
	socketRoot, socketAlias, err := newShortSocketRoot(root)
	if err != nil {
		return nil, fail(CodeProbeIntegrity, err)
	}
	socketRootInfo, socketRootErr := os.Lstat(socketRoot)
	socketAliasInfo, socketAliasErr := os.Lstat(socketAlias)
	if socketRootErr != nil || socketAliasErr != nil || !socketRootInfo.IsDir() ||
		socketRootInfo.Mode().Perm() != 0o700 || socketRootInfo.Mode()&os.ModeSymlink != 0 ||
		socketAliasInfo.Mode()&os.ModeSymlink == 0 {
		_ = os.Remove(socketAlias)
		_ = os.Remove(socketRoot)
		return nil, fail(
			CodeProbeIntegrity,
			errors.Join(socketRootErr, socketAliasErr, errors.New("short scope root identity was not established")),
		)
	}
	probe = &Probe{
		root: root, rootIdentity: rootInfo,
		socketRoot: socketRoot, socketRootIdentity: socketRootInfo,
		socketAlias: socketAlias, socketAliasIdentity: socketAliasInfo,
	}
	probe.importCanary, err = newCanary(filepath.Join(socketAlias, "i"))
	if err != nil {
		return nil, err
	}
	probe.serviceCanary, err = newCanary(filepath.Join(socketAlias, "s"))
	if err != nil {
		return nil, err
	}
	probe.module = filepath.Join(root, "outside-candidate.mjs")
	body := []byte(importModuleSource)
	file, err := os.OpenFile(probe.module, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fail(CodeProbeIntegrity, err)
	}
	written, writeErr := file.Write(body)
	chmodErr := file.Chmod(0o600)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || chmodErr != nil || syncErr != nil || closeErr != nil || written != len(body) {
		return nil, fail(CodeProbeIntegrity, errors.Join(writeErr, chmodErr, syncErr, closeErr))
	}
	probe.moduleInfo, err = os.Lstat(probe.module)
	if err != nil || !probe.moduleInfo.Mode().IsRegular() || probe.moduleInfo.Mode().Perm() != 0o600 {
		return nil, fail(CodeProbeIntegrity, err)
	}
	digest := sha256.Sum256(body)
	probe.moduleSHA256 = hex.EncodeToString(digest[:])
	cleanup = false
	return probe, nil
}

func (probe *Probe) Environment() map[string]string {
	if probe == nil || probe.importCanary == nil || probe.serviceCanary == nil {
		return nil
	}
	return map[string]string{
		ImportModuleEnvironment:  probe.module,
		ImportSocketEnvironment:  probe.importCanary.path,
		ServiceSocketEnvironment: probe.serviceCanary.path,
	}
}

func (probe *Probe) Finish(ctx context.Context) (Measurements, error) {
	if probe == nil {
		return Measurements{}, fail(CodeProbeAbsent, nil)
	}
	probe.once.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		rootRosterErr := boundedDirectRoster(
			ctx,
			probe.root,
			probe.rootIdentity,
			[]string{
				filepath.Base(probe.importCanary.path),
				filepath.Base(probe.serviceCanary.path),
				filepath.Base(probe.module),
			},
		)
		shortRosterErr := boundedDirectRoster(
			ctx,
			probe.socketRoot,
			probe.socketRootIdentity,
			[]string{filepath.Base(probe.socketAlias)},
		)
		moduleDigest, digestErr := digestFile(probe.module, probe.moduleInfo)
		if digestErr == nil && moduleDigest != probe.moduleSHA256 {
			digestErr = errors.New("scope import module bytes changed")
		}
		aliasTarget, aliasErr := os.Readlink(probe.socketAlias)
		aliasInfo, aliasInfoErr := os.Lstat(probe.socketAlias)
		if aliasErr == nil && aliasInfoErr == nil &&
			(aliasTarget != probe.root || aliasInfo.Mode() != probe.socketAliasIdentity.Mode() ||
				!os.SameFile(probe.socketAliasIdentity, aliasInfo)) {
			aliasErr = errors.New("scope socket alias identity or target changed")
		}
		importErr := probe.importCanary.close()
		serviceErr := probe.serviceCanary.close()
		probe.measurements = Measurements{
			ImportAccepts:  probe.importCanary.accepted.Load(),
			ServiceAccepts: probe.serviceCanary.accepted.Load(),
			ModuleSHA256:   probe.moduleSHA256,
		}
		integrityErr := errors.Join(
			ctx.Err(),
			rootRosterErr,
			shortRosterErr,
			digestErr,
			aliasErr,
			aliasInfoErr,
			importErr,
			serviceErr,
		)
		var rootCleanupErr error
		if integrityErr == nil {
			rootCleanupErr = errors.Join(
				removeRetainedLeaf(ctx, probe.importCanary.path, probe.importCanary.identity),
				removeRetainedLeaf(ctx, probe.serviceCanary.path, probe.serviceCanary.identity),
				removeRetainedLeaf(ctx, probe.module, probe.moduleInfo),
			)
			if rootCleanupErr == nil {
				rootCleanupErr = removeRetainedEmptyDirectory(ctx, probe.root, probe.rootIdentity)
			}
		}
		var shortCleanupErr error
		if ctx.Err() == nil && shortRosterErr == nil && aliasErr == nil &&
			aliasInfoErr == nil && importErr == nil && serviceErr == nil {
			shortCleanupErr = removeRetainedLeaf(
				ctx, probe.socketAlias, probe.socketAliasIdentity,
			)
			if shortCleanupErr == nil {
				shortCleanupErr = removeRetainedEmptyDirectory(
					ctx, probe.socketRoot, probe.socketRootIdentity,
				)
			}
		}
		probe.err = errors.Join(integrityErr, rootCleanupErr, shortCleanupErr)
		if probe.err != nil {
			probe.measurements.Ambiguous = true
			code := CodeProbeCleanup
			if integrityErr != nil {
				code = CodeProbeIntegrity
			}
			probe.measurements.Diagnostic = NewDiagnostic(code)
			probe.err = fail(code, probe.err)
		}
	})
	return probe.measurements, probe.err
}

func removeRetainedLeaf(ctx context.Context, path string, retained os.FileInfo) error {
	if ctx == nil || !filepath.IsAbs(path) || filepath.Clean(path) != path ||
		retained == nil || retained.IsDir() {
		return errors.New("retained scope leaf identity is absent")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := os.Lstat(path)
	if err != nil || current.Mode() != retained.Mode() || current.Size() != retained.Size() ||
		!os.SameFile(retained, current) {
		return errors.Join(err, errors.New("retained scope leaf identity changed before cleanup"))
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.Join(err, errors.New("retained scope leaf survived cleanup"))
	}
	return nil
}

func removeRetainedEmptyDirectory(
	ctx context.Context,
	path string,
	retained os.FileInfo,
) error {
	if err := boundedDirectRoster(ctx, path, retained, nil); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.Join(err, errors.New("retained scope directory survived cleanup"))
	}
	return nil
}

func boundedDirectRoster(
	ctx context.Context,
	root string,
	retained os.FileInfo,
	expected []string,
) error {
	if ctx == nil || !filepath.IsAbs(root) || filepath.Clean(root) != root ||
		retained == nil || !retained.IsDir() || retained.Mode()&os.ModeSymlink != 0 ||
		len(expected) > 16 {
		return errors.New("bounded scope roster inputs are invalid")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	want := append([]string(nil), expected...)
	for _, name := range want {
		if name == "" || name == "." || name == ".." ||
			strings.ContainsRune(name, filepath.Separator) {
			return errors.New("bounded scope roster expectation contains an invalid name")
		}
	}
	sort.Strings(want)
	for index := 1; index < len(want); index++ {
		if want[index-1] == want[index] {
			return errors.New("bounded scope roster expectation repeats a name")
		}
	}
	before, err := os.Lstat(root)
	if err != nil || !before.IsDir() || before.Mode() != retained.Mode() ||
		!os.SameFile(retained, before) {
		return errors.Join(err, errors.New("bounded scope roster root identity differs"))
	}
	fd, err := syscall.Open(
		root,
		syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC|
			syscall.O_NONBLOCK|darwinNoFollowAny,
		0,
	)
	if err != nil {
		return err
	}
	handle := os.NewFile(uintptr(fd), root)
	if handle == nil {
		_ = syscall.Close(fd)
		return errors.New("bounded scope roster descriptor could not be owned")
	}
	opened, openErr := handle.Stat()
	if openErr != nil || !opened.IsDir() || opened.Mode() != retained.Mode() ||
		!os.SameFile(retained, opened) {
		return errors.Join(
			openErr,
			handle.Close(),
			errors.New("bounded scope roster descriptor identity differs"),
		)
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
			rosterErr = errors.New("bounded scope roster cardinality differs")
			break
		}
		batch, readErr := handle.ReadDir(remaining)
		for _, entry := range batch {
			name := entry.Name()
			observed = append(observed, name)
			if name == "" || entry.IsDir() {
				rosterErr = errors.New("bounded scope roster contains an invalid leaf")
				break
			}
		}
		if rosterErr != nil || len(observed) > len(want) {
			if rosterErr == nil {
				rosterErr = errors.New("bounded scope roster cardinality differs")
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
			rosterErr = errors.New("bounded scope roster made no progress")
			break
		}
	}
	sort.Strings(observed)
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(root)
	if rosterErr != nil || descriptorErr != nil || closeErr != nil || pathErr != nil ||
		afterDescriptor.Mode() != opened.Mode() || afterPath.Mode() != opened.Mode() ||
		!os.SameFile(opened, afterDescriptor) || !os.SameFile(opened, afterPath) {
		return errors.Join(
			rosterErr,
			descriptorErr,
			closeErr,
			pathErr,
			errors.New("bounded scope roster read did not close exactly"),
		)
	}
	if len(observed) != len(want) {
		return errors.New("bounded scope roster cardinality differs")
	}
	for index := range want {
		if observed[index] != want[index] {
			return errors.New("bounded scope roster names differ")
		}
	}
	return ctx.Err()
}

func newShortSocketRoot(attemptPrivateRoot string) (string, string, error) {
	if !filepath.IsAbs(attemptPrivateRoot) ||
		filepath.Clean(attemptPrivateRoot) != attemptPrivateRoot {
		return "", "", errors.New("attempt-private scope root is not clean and absolute")
	}
	parentInfo, err := os.Lstat(shortSocketParent)
	parentStat, statOK := statIdentity(parentInfo)
	if err != nil || !statOK || !parentInfo.IsDir() ||
		parentInfo.Mode()&os.ModeSymlink != 0 ||
		parentInfo.Mode().Perm() != 0o777 ||
		parentInfo.Mode()&os.ModeSticky == 0 ||
		parentInfo.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 ||
		parentStat.Uid != 0 {
		return "", "", errors.Join(
			err,
			errors.New("Darwin short-socket parent is not the root-owned sticky directory"),
		)
	}
	root, err := os.MkdirTemp(shortSocketParent, "cs5-")
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
	stat, statOK := statIdentity(info)
	if err != nil || !statOK || !info.IsDir() ||
		info.Mode()&os.ModeSymlink != 0 ||
		info.Mode().Perm() != 0o700 ||
		info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 ||
		stat.Uid != uint32(os.Getuid()) {
		return "", "", errors.Join(
			err,
			errors.New("Darwin short-socket root is not an exact caller-owned private directory"),
		)
	}
	alias = filepath.Join(root, "r")
	if err := os.Symlink(attemptPrivateRoot, alias); err != nil {
		return "", "", err
	}
	linked, readErr := os.Readlink(alias)
	resolved, resolveErr := filepath.EvalSymlinks(alias)
	if readErr != nil || resolveErr != nil ||
		linked != attemptPrivateRoot || resolved != attemptPrivateRoot {
		return "", "", errors.Join(
			readErr,
			resolveErr,
			errors.New("Darwin short-socket alias did not resolve exactly"),
		)
	}
	for _, name := range []string{"i", "s"} {
		if len([]byte(filepath.Join(alias, name))) >
			maxDarwinUnixSocketPathBytes {
			return "", "", errors.New("Darwin short-socket alias exceeds sockaddr_un capacity")
		}
	}
	cleanup = false
	return root, alias, nil
}

func statIdentity(info os.FileInfo) (*syscall.Stat_t, bool) {
	if info == nil {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return stat, ok
}

func newCanary(path string) (*canary, error) {
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return nil, fail(CodeProbeIntegrity, err)
	}
	listener.SetUnlinkOnClose(false)
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode()&os.ModeSymlink != 0 {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, fail(CodeProbeIntegrity, err)
	}
	value := &canary{
		path: path, listener: listener, identity: info, done: make(chan error, 1),
	}
	go func() {
		for {
			connection, acceptErr := listener.AcceptUnix()
			if acceptErr != nil {
				value.done <- acceptErr
				return
			}
			value.accepted.Add(1)
			_ = connection.Close()
		}
	}()
	return value, nil
}

func (value *canary) close() error {
	if value == nil {
		return nil
	}
	value.once.Do(func() {
		current, statErr := os.Lstat(value.path)
		if statErr == nil && (current.Mode()&os.ModeSocket == 0 ||
			current.Mode()&os.ModeSymlink != 0 || !os.SameFile(value.identity, current)) {
			statErr = errors.New("canary identity changed")
		}
		deadlineErr := value.listener.SetDeadline(time.Now().Add(25 * time.Millisecond))
		var terminalErr error
		select {
		case terminalErr = <-value.done:
		case <-time.After(100 * time.Millisecond):
			terminalErr = errors.New("canary accept loop did not stop")
		}
		closeErr := value.listener.Close()
		if terminalErr != nil {
			var networkErr net.Error
			if errors.As(terminalErr, &networkErr) && networkErr.Timeout() {
				terminalErr = nil
			} else if errors.Is(terminalErr, net.ErrClosed) {
				terminalErr = nil
			}
		}
		value.err = errors.Join(statErr, deadlineErr, terminalErr, closeErr)
	})
	return value.err
}
