//go:build darwin

package store

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"
)

type studyLock struct{ file *os.File }

func openAndLockStudy(ctx context.Context, path string, create bool) (studyLock, error) {
	flags := syscall.O_RDWR | syscall.O_CLOEXEC | syscall.O_NOFOLLOW
	if create {
		flags |= syscall.O_CREAT
	}
	fd, err := syscall.Open(path, flags, 0o600)
	if err != nil {
		return studyLock{}, err
	}
	file := os.NewFile(uintptr(fd), path)
	fail := func(err error) (studyLock, error) {
		_ = file.Close()
		return studyLock{}, err
	}
	pathInfo, err := os.Lstat(path)
	openedInfo, statErr := file.Stat()
	if err != nil || statErr != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !openedInfo.Mode().IsRegular() ||
		openedInfo.Mode().Perm() != 0o600 || !os.SameFile(pathInfo, openedInfo) {
		return fail(errors.Join(err, statErr, errors.New("study lock path facts are invalid")))
	}
	for {
		err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return studyLock{file: file}, nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			return fail(err)
		}
		if ctx == nil {
			return fail(errors.New("nil lock context"))
		}
		select {
		case <-ctx.Done():
			return fail(ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func (lock studyLock) release() {
	if lock.file == nil {
		return
	}
	_ = syscall.Flock(int(lock.file.Fd()), syscall.LOCK_UN)
	_ = lock.file.Close()
}

// The per-study flock supplies compare-and-swap serialization. Darwin rename
// supplies atomic replacement; surrounding path/fstat checks reject symlinks.
// This is not a hostile same-UID race defense.
func renameHeadNoFollow(oldPath, newPath string) error {
	if info, err := os.Lstat(oldPath); err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.Join(err, errors.New("head temporary is not a real file"))
	}
	if info, err := os.Lstat(newPath); err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()) {
		return errors.New("head destination is not a real file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(oldPath, newPath)
}
