//go:build darwin && arm64 && cgo

package noderuntime

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/nelsonwerd/countershape/internal/domain"
)

func executableIdentityFromStat(stat syscall.Stat_t) executableIdentity {
	return executableIdentity{
		device: uint64(stat.Dev), inode: stat.Ino, links: uint64(stat.Nlink), uid: uint64(stat.Uid), gid: uint64(stat.Gid),
		mode: uint32(stat.Mode), size: stat.Size,
		mtimeSec: stat.Mtimespec.Sec, mtimeNsec: stat.Mtimespec.Nsec,
		ctimeSec: stat.Ctimespec.Sec, ctimeNsec: stat.Ctimespec.Nsec,
		birthSec: stat.Birthtimespec.Sec, birthNsec: stat.Birthtimespec.Nsec,
	}
}

func descriptorPath(fd int) (string, error) {
	var buffer [maxExecutablePath + 1]byte
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(syscall.F_GETPATH), uintptr(unsafe.Pointer(&buffer[0])))
	if errno != 0 {
		return "", errno
	}
	end := 0
	for end < len(buffer) && buffer[end] != 0 {
		end++
	}
	if end == 0 || end == len(buffer) {
		return "", errors.New("descriptor path is empty or unterminated")
	}
	return string(buffer[:end]), nil
}

func statLeaf(path string) (syscall.Stat_t, error) {
	var stat syscall.Stat_t
	if err := syscall.Lstat(path, &stat); err != nil {
		return stat, err
	}
	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG {
		return stat, errors.New("executable leaf is not a regular file")
	}
	return stat, nil
}

func measureExecutable(path string) (runtimeSnapshot, error) {
	beforePath, err := statLeaf(path)
	if err != nil {
		return runtimeSnapshot{}, err
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return runtimeSnapshot{}, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return runtimeSnapshot{}, errors.New("executable descriptor could not be owned")
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	var before syscall.Stat_t
	if err := syscall.Fstat(fd, &before); err != nil {
		return runtimeSnapshot{}, err
	}
	if executableIdentityFromStat(before) != executableIdentityFromStat(beforePath) {
		return runtimeSnapshot{}, errors.New("executable path and descriptor identities differ")
	}
	identity := executableIdentityFromStat(before)
	permissions := os.FileMode(before.Mode & 0o7777)
	if !identity.valid() || before.Size > maxExecutableBytes || permissions&0o111 == 0 || permissions&0o022 != 0 || permissions&0o7000 != 0 {
		return runtimeSnapshot{}, errors.New("executable mode, size, or ownership profile is refused")
	}
	canonical, err := descriptorPath(fd)
	if err != nil || canonical != path || filepath.Clean(canonical) != canonical {
		return runtimeSnapshot{}, errors.Join(errors.New("descriptor-derived executable path differs"), err)
	}
	hasher := sha256.New()
	_, _ = io.WriteString(hasher, "countershape/v1/NodeExecutableBytes")
	_, _ = hasher.Write([]byte{0})
	written, err := io.Copy(hasher, io.LimitReader(file, maxExecutableBytes+1))
	if err != nil || written != before.Size || written < 1 || written > maxExecutableBytes {
		return runtimeSnapshot{}, errors.Join(errors.New("executable byte stream was incomplete or oversized"), err)
	}
	var after syscall.Stat_t
	if err := syscall.Fstat(fd, &after); err != nil {
		return runtimeSnapshot{}, err
	}
	afterPath, pathErr := statLeaf(path)
	if pathErr != nil || executableIdentityFromStat(after) != identity || executableIdentityFromStat(afterPath) != identity {
		return runtimeSnapshot{}, errors.Join(errors.New("executable identity changed during measurement"), pathErr)
	}
	if err := file.Close(); err != nil {
		return runtimeSnapshot{}, err
	}
	closed = true
	digest, err := domain.ParseDigest("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
	if err != nil {
		return runtimeSnapshot{}, err
	}
	return runtimeSnapshot{
		identity: identity, path: path, bytesDigest: digest,
		mode: fmt.Sprintf("100%03o", before.Mode&0o777), byteCount: written,
	}, nil
}
