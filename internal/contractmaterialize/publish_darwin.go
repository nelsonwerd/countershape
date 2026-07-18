//go:build darwin && arm64 && cgo

package contractmaterialize

/*
#include <errno.h>
#include <fcntl.h>
#include <stdlib.h>
#include <sys/stat.h>
#include <sys/stdio.h>
#include <unistd.h>

static int countershape_contract_openat(int dirfd, const char *name, int flags, mode_t mode, int *error_number) {
	int fd = openat(dirfd, name, flags, mode);
	if (fd >= 0) return fd;
	*error_number = errno;
	return -1;
}

static int countershape_contract_mkdirat(int dirfd, const char *name, mode_t mode) {
	if (mkdirat(dirfd, name, mode) == 0) return 0;
	return errno;
}

static int countershape_contract_unlinkat(int dirfd, const char *name, int directory) {
	int flags = directory ? AT_REMOVEDIR : 0;
	if (unlinkat(dirfd, name, flags) == 0) return 0;
	return errno;
}

static int countershape_contract_renameat_exclusive(int dirfd, const char *from, const char *to) {
	if (renameatx_np(dirfd, from, dirfd, to, RENAME_EXCL | RENAME_NOFOLLOW_ANY) == 0) return 0;
	return errno;
}
*/
import "C"

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

func nativePublicationSupported() bool { return true }

func trustedPublicationParent(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid()) && info.Mode().Perm()&0o700 == 0o700 &&
		info.Mode().Perm()&0o022 == 0
}

func ownedByEffectiveUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

func openDirectoryPathNoFollow(path string) (durableDirectory, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, syscall.EINVAL
	}
	flags := syscall.O_RDONLY | syscall.O_DIRECTORY | syscall.O_NOFOLLOW | syscall.O_CLOEXEC
	fd, err := syscall.Open(string(filepath.Separator), flags, 0)
	if err != nil {
		return nil, err
	}
	current := durableDirectory(os.NewFile(uintptr(fd), string(filepath.Separator)))
	if path == string(filepath.Separator) {
		return current, nil
	}
	for _, component := range strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			return nil, errors.Join(syscall.EINVAL, current.Close())
		}
		next, openErr := openDirectoryAtNoFollow(current, component)
		if openErr != nil {
			return nil, errors.Join(openErr, current.Close())
		}
		if closeErr := current.Close(); closeErr != nil {
			_ = next.Close()
			return nil, closeErr
		}
		current = next
	}
	return current, nil
}

func openDirectoryAtNoFollow(parent durableDirectory, name string) (durableDirectory, error) {
	fd, err := openAt(parent, name, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func openFileAtNoFollow(
	parent durableDirectory,
	name string,
	flags int,
	mode os.FileMode,
) (durableFile, error) {
	fd, err := openAt(parent, name, flags|syscall.O_NOFOLLOW|syscall.O_NONBLOCK|syscall.O_CLOEXEC, mode)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func openAt(parent durableDirectory, name string, flags int, mode os.FileMode) (int, error) {
	nameCString := C.CString(name)
	defer C.free(unsafe.Pointer(nameCString))
	errorNumber := C.int(0)
	fd := C.countershape_contract_openat(
		C.int(parent.Fd()), nameCString, C.int(flags), C.mode_t(mode.Perm()), &errorNumber,
	)
	if fd < 0 {
		return -1, syscall.Errno(errorNumber)
	}
	return int(fd), nil
}

func mkdirDirectoryAt(parent durableDirectory, name string, mode os.FileMode) error {
	nameCString := C.CString(name)
	defer C.free(unsafe.Pointer(nameCString))
	if errorNumber := C.countershape_contract_mkdirat(
		C.int(parent.Fd()), nameCString, C.mode_t(mode.Perm()),
	); errorNumber != 0 {
		return syscall.Errno(errorNumber)
	}
	return nil
}

func unlinkEntryAt(parent durableDirectory, name string, directory bool) error {
	nameCString := C.CString(name)
	defer C.free(unsafe.Pointer(nameCString))
	directoryFlag := C.int(0)
	if directory {
		directoryFlag = 1
	}
	if errorNumber := C.countershape_contract_unlinkat(
		C.int(parent.Fd()), nameCString, directoryFlag,
	); errorNumber != 0 {
		return syscall.Errno(errorNumber)
	}
	return nil
}

func renameExclusiveDirectoryAt(parent durableDirectory, from, to string) error {
	fromCString := C.CString(from)
	toCString := C.CString(to)
	defer C.free(unsafe.Pointer(fromCString))
	defer C.free(unsafe.Pointer(toCString))
	if errorNumber := C.countershape_contract_renameat_exclusive(
		C.int(parent.Fd()), fromCString, toCString,
	); errorNumber != 0 {
		return syscall.Errno(errorNumber)
	}
	return nil
}

func physicalLinkCount(info os.FileInfo) (uint64, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return uint64(stat.Nlink), true
}
